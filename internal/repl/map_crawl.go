package repl

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

func (s *Shell) handleMap(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%w: map <url> [search=keyword] [limit=N]", errInvalidUsage)
	}

	targetURL := args[0]
	searchKeyword, limit := parseMapArgs(args[1:])

	content, err := s.client.Scrape(ctx, scrapedo.ScrapeRequest{URL: targetURL})
	if err != nil {
		return fmt.Errorf("scrape failed: %w", err)
	}

	links := scrapedo.ExtractLinks(content, targetURL)
	links = filterLinks(links, searchKeyword, limit)

	fmt.Fprintf(s.out, "Discovered %d URLs:\n\n", len(links))

	for i, l := range links {
		fmt.Fprintf(s.out, "%3d  %s\n", i+1, l)
	}

	return nil
}

func parseMapArgs(args []string) (string, int) {
	var keyword string

	limit := 100 //nolint:mnd // default limit

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "search="):
			keyword = strings.TrimPrefix(arg, "search=")
		case strings.HasPrefix(arg, "limit="):
			if n, err := strconv.Atoi(strings.TrimPrefix(arg, "limit=")); err == nil {
				limit = n
			}
		}
	}

	return keyword, limit
}

func (s *Shell) handleCrawl(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%w: crawl <url> [depth=N] [limit=N]", errInvalidUsage)
	}

	targetURL := args[0]
	depth, limit := parseCrawlArgs(args[1:])

	opts := scrapedo.CrawlOptions{
		MaxDepth: depth,
		MaxPages: limit,
	}

	pageNum := 0

	if err := s.client.Crawl(ctx, targetURL, opts, func(r scrapedo.CrawlResult) {
		pageNum++
		if r.Error != nil {
			fmt.Fprintf(s.out, "[%d/%d] %s → ERROR: %v\n", pageNum, limit, r.URL, r.Error)
			return
		}
		fmt.Fprintf(s.out, "[%d/%d] %s → %dB\n", pageNum, limit, r.URL, r.Size)
	}); err != nil {
		return fmt.Errorf("crawl failed: %w", err)
	}

	return nil
}

func parseCrawlArgs(args []string) (int, int) {
	depth := 1
	limit := 10 //nolint:mnd // default limit

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "depth="):
			if n, err := strconv.Atoi(strings.TrimPrefix(arg, "depth=")); err == nil {
				depth = n
			}
		case strings.HasPrefix(arg, "limit="):
			if n, err := strconv.Atoi(strings.TrimPrefix(arg, "limit=")); err == nil {
				limit = n
			}
		}
	}

	return depth, limit
}

func filterLinks(links []string, keyword string, limit int) []string {
	if keyword != "" {
		needle := strings.ToLower(keyword)
		filtered := links[:0]

		for _, l := range links {
			if strings.Contains(strings.ToLower(l), needle) {
				filtered = append(filtered, l)
			}
		}

		links = filtered
	}

	if limit > 0 && len(links) > limit {
		links = links[:limit]
	}

	return links
}
