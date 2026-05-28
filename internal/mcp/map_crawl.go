package mcp

import (
	"context"
	"fmt"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

// mapToolArgs defines arguments for the map_urls tool.
type mapToolArgs struct {
	URL    string `json:"url"              jsonschema:"The target URL"`
	Search string `json:"search,omitempty" jsonschema:"Filter URLs by keyword"`
	Limit  int    `json:"limit,omitempty"  jsonschema:"Max URLs (default 100)"`
}

// crawlToolArgs defines arguments for the crawl_site tool.
type crawlToolArgs struct {
	URL      string `json:"url"                jsonschema:"Start URL"`
	MaxDepth int    `json:"maxDepth,omitempty" jsonschema:"Max depth (default 1)"`
	MaxPages int    `json:"maxPages,omitempty" jsonschema:"Max pages (default 10)"`
}

func addMapTool(server *mcpsdk.Server, client *scrapedo.Client, recorder UsageRecorder) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "map_urls",
		Description: "Discover all same-domain URLs on a web page by scraping it and extracting links.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, args mapToolArgs) (*mcpsdk.CallToolResult, any, error) {
		return handleMapTool(ctx, client, args, recorder)
	})
}

func handleMapTool(
	ctx context.Context, client *scrapedo.Client, args mapToolArgs, recorder UsageRecorder,
) (*mcpsdk.CallToolResult, any, error) {
	if args.URL == "" {
		return toolErr("url is required"), nil, nil
	}

	content, err := client.Scrape(ctx, scrapedo.ScrapeRequest{URL: args.URL})
	if err != nil {
		return toolErr(fmt.Sprintf("scrape failed: %v", err)), nil, nil
	}

	recordUsage(ctx, recorder, "map", args.URL)

	links := scrapedo.ExtractLinks(content, args.URL)
	links = applyMapFilters(links, args)

	text := fmt.Sprintf("Discovered %d URLs:\n\n%s", len(links), strings.Join(links, "\n"))

	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: text}},
	}, nil, nil
}

func applyMapFilters(links []string, args mapToolArgs) []string {
	if args.Search != "" {
		needle := strings.ToLower(args.Search)
		filtered := links[:0]

		for _, l := range links {
			if strings.Contains(strings.ToLower(l), needle) {
				filtered = append(filtered, l)
			}
		}

		links = filtered
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 100 //nolint:mnd // default limit
	}

	if len(links) > limit {
		links = links[:limit]
	}

	return links
}

func addCrawlTool(server *mcpsdk.Server, client *scrapedo.Client, recorder UsageRecorder) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "crawl_site",
		Description: "Crawl a website breadth-first, scraping multiple pages and returning all content as markdown.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, args crawlToolArgs) (*mcpsdk.CallToolResult, any, error) {
		return handleCrawlTool(ctx, client, args, recorder)
	})
}

func handleCrawlTool(
	ctx context.Context, client *scrapedo.Client, args crawlToolArgs, recorder UsageRecorder,
) (*mcpsdk.CallToolResult, any, error) {
	if args.URL == "" {
		return toolErr("url is required"), nil, nil
	}

	opts := buildCrawlOpts(args)
	var buf strings.Builder
	pageNum := 0

	err := client.Crawl(ctx, args.URL, opts, func(r scrapedo.CrawlResult) {
		pageNum++
		appendCrawlResult(&buf, r, pageNum)
		recordUsage(ctx, recorder, "crawl", r.URL)
	})
	if err != nil {
		return toolErr(fmt.Sprintf("crawl failed: %v", err)), nil, nil
	}

	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: buf.String()}},
	}, nil, nil
}

func buildCrawlOpts(args crawlToolArgs) scrapedo.CrawlOptions {
	depth := args.MaxDepth
	if depth <= 0 {
		depth = 1
	}

	pages := args.MaxPages
	if pages <= 0 {
		pages = 10 //nolint:mnd // default
	}

	return scrapedo.CrawlOptions{MaxDepth: depth, MaxPages: pages}
}

func appendCrawlResult(buf *strings.Builder, r scrapedo.CrawlResult, pageNum int) {
	if r.Error != nil {
		fmt.Fprintf(buf, "## Page %d: %s\n\nError: %v\n\n", pageNum, r.URL, r.Error)
		return
	}

	fmt.Fprintf(buf, "## Page %d: %s\n\n%s\n\n---\n\n", pageNum, r.URL, r.Content)
}

func recordUsage(ctx context.Context, recorder UsageRecorder, action, targetURL string) {
	if recorder != nil {
		//nolint:gosec // best-effort usage tracking
		_ = recorder.RecordUsage(ctx, "scrapedo", "", action, "", targetURL, 1)
	}
}

func toolErr(msg string) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: msg}},
		IsError: true,
	}
}
