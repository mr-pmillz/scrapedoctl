// Package mcp provides the Model Context Protocol server implementation for scrapedoctl.
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mr-pmillz/scrapedoctl/internal/version"
	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
	"github.com/mr-pmillz/scrapedoctl/pkg/search"
)

// UsageRecorder records API usage for local analytics.
type UsageRecorder interface {
	RecordUsage(ctx context.Context, provider, engine, action, query, url string, credits int) error
}

// toolArgs defines the expected arguments from Claude for the scrape_url tool.
type toolArgs struct {
	URL     string            `json:"url"               jsonschema:"The target URL to scrape"`
	Render  bool              `json:"render,omitempty"  jsonschema:"Set to true to execute JavaScript"`
	Super   bool              `json:"super,omitempty"   jsonschema:"Set to true to utilize residential proxy"`
	GeoCode string            `json:"geoCode,omitempty" jsonschema:"2-letter country code (e.g. us, gb, de) to route requests through a specific location"`
	Session string            `json:"session,omitempty" jsonschema:"Unique string to maintain a sticky session (same proxy IP)"`
	Device  string            `json:"device,omitempty"  jsonschema:"Emulate a specific device (desktop, mobile, or tablet)"`
	Method  string            `json:"method,omitempty"  jsonschema:"HTTP method to use (default: GET)"`
	Headers map[string]string `json:"headers,omitempty" jsonschema:"Custom HTTP headers to forward to the target"`
	Body    string            `json:"body,omitempty"    jsonschema:"Request body for POST/PUT requests"`
	Actions []any             `json:"actions,omitempty" jsonschema:"Browser actions like click, scroll, etc. (requires render=true)"`
}

// searchToolArgs defines the expected arguments for the search tool.
type searchToolArgs struct {
	Query    string `json:"query"              jsonschema:"The search query string"`
	Engine   string `json:"engine,omitempty"   jsonschema:"Search engine to use (google, bing, yandex, duckduckgo, etc.)"`
	Provider string `json:"provider,omitempty" jsonschema:"Force a specific search provider"`
	Lang     string `json:"lang,omitempty"     jsonschema:"Language code (e.g. en, de, fr)"`
	Country  string `json:"country,omitempty"  jsonschema:"Country code (e.g. us, gb, de)"`
	Limit    int    `json:"limit,omitempty"    jsonschema:"Maximum number of results to return"`
}

// RunServer initializes and runs the standard stdio MCP server for Scrape.do.
func RunServer(ctx context.Context, apiToken string) error {
	client, err := scrapedo.NewClient(apiToken)
	if err != nil {
		return fmt.Errorf("failed to init scrape.do client: %w", err)
	}
	server, err := NewServerWithClient(client)
	if err != nil {
		return fmt.Errorf("failed to create mcp server: %w", err)
	}

	if err := server.Run(ctx, &mcpsdk.StdioTransport{}); err != nil {
		return fmt.Errorf("mcp server failed: %w", err)
	}

	return nil
}

// ErrClientNil is returned when a nil client is passed to RunServerWithClient.
var ErrClientNil = errors.New("client cannot be nil")

// ServerOption configures the MCP server.
type ServerOption func(*serverConfig)

type serverConfig struct {
	router   *search.Router
	recorder UsageRecorder
}

// WithUsageRecorder sets the usage recorder on the MCP server.
func WithUsageRecorder(r UsageRecorder) ServerOption {
	return func(c *serverConfig) { c.recorder = r }
}

// WithRouter sets the search router on the MCP server.
func WithRouter(r *search.Router) ServerOption {
	return func(c *serverConfig) { c.router = r }
}

// RunServerWithClient runs the server with a pre-configured client (e.g. with cache).
// An optional search router can be provided to enable the search tool.
func RunServerWithClient(ctx context.Context, client *scrapedo.Client, routers ...*search.Router) error {
	if client == nil {
		return ErrClientNil
	}

	var router *search.Router
	if len(routers) > 0 {
		router = routers[0]
	}

	server, err := NewServerWithClientAndRouter(client, router)
	if err != nil {
		return err
	}
	if err := server.Run(ctx, &mcpsdk.StdioTransport{}); err != nil {
		return fmt.Errorf("mcp server failed: %w", err)
	}
	return nil
}

// RunServerWithOpts runs the server with functional options.
func RunServerWithOpts(
	ctx context.Context, client *scrapedo.Client, opts ...ServerOption,
) error {
	if client == nil {
		return ErrClientNil
	}

	var sc serverConfig
	for _, o := range opts {
		o(&sc)
	}

	server, err := newServerInternal(client, sc.router, sc.recorder)
	if err != nil {
		return err
	}
	if err := server.Run(ctx, &mcpsdk.StdioTransport{}); err != nil {
		return fmt.Errorf("mcp server failed: %w", err)
	}
	return nil
}

// NewServer creates a new MCP server for Scrape.do with a default client.
func NewServer(apiToken string) (*mcpsdk.Server, error) {
	client, err := scrapedo.NewClient(apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to init scrape.do client: %w", err)
	}
	return NewServerWithClient(client)
}

// NewServerWithClient creates a new MCP server with the provided client (no search router).
func NewServerWithClient(client *scrapedo.Client) (*mcpsdk.Server, error) {
	return NewServerWithClientAndRouter(client, nil)
}

// NewServerWithClientAndRouter creates a new MCP server with the provided client
// and an optional search router.
func NewServerWithClientAndRouter(client *scrapedo.Client, router *search.Router) (*mcpsdk.Server, error) {
	return newServerInternal(client, router, nil)
}

// NewServerWithClientAndRecorder creates a new MCP server with the provided client,
// an optional search router, and an optional usage recorder.
func NewServerWithClientAndRecorder(
	client *scrapedo.Client, router *search.Router, recorder UsageRecorder,
) (*mcpsdk.Server, error) {
	return newServerInternal(client, router, recorder)
}

func newServerInternal(
	client *scrapedo.Client, router *search.Router, recorder UsageRecorder,
) (*mcpsdk.Server, error) {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "scrapedoctl",
		Version: version.Version,
	}, nil)

	addCLIHelpResource(server)
	addScrapeTool(server, client, recorder)
	addMapTool(server, client, recorder)
	addCrawlTool(server, client, recorder)

	if router != nil && len(router.Providers()) > 0 {
		addSearchTool(server, router, recorder)
	}

	return server, nil
}

func addCLIHelpResource(server *mcpsdk.Server) {
	// Add Resource: CLI Documentation
	server.AddResource(&mcpsdk.Resource{
		URI:         "resource://cli/help",
		Name:        "CLI Documentation",
		Description: "Detailed documentation of the scrapedoctl CLI commands and flags in JSON format.",
		MIMEType:    "application/json",
	}, func(_ context.Context, _ *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
		doc := map[string]any{
			"cli":         "scrapedoctl",
			"description": "Scrape.do CLI & MCP Server",
			"usage":       "scrapedoctl [command] [flags]",
			"commands": []map[string]any{
				{
					"name":        "scrape",
					"description": "Scrape a single URL to markdown",
					"flags":       []string{"--render", "--super", "--geoCode", "--session", "--device"},
				},
				{
					"name":        "repl",
					"description": "Start an interactive shell",
				},
				{
					"name":        "mcp",
					"description": "Run as an MCP stdio server",
				},
			},
		}
		data, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal documentation: %w", err)
		}
		return &mcpsdk.ReadResourceResult{
			Contents: []*mcpsdk.ResourceContents{
				{
					URI:      "resource://cli/help",
					MIMEType: "application/json",
					Text:     string(data),
				},
			},
		}, nil
	})
}

func addScrapeTool(server *mcpsdk.Server, client *scrapedo.Client, recorder UsageRecorder) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "scrape_url",
		Description: "Scrape a web page and return optimized markdown. Supports JS rendering, proxy rotation, geo-targeting, sticky sessions, device emulation, and POST requests with custom headers and browser actions.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, args toolArgs) (*mcpsdk.CallToolResult, any, error) {
		scrapeReq := scrapedo.ScrapeRequest{
			URL:     args.URL,
			Render:  args.Render,
			Super:   args.Super,
			GeoCode: args.GeoCode,
			Session: args.Session,
			Device:  args.Device,
			Method:  args.Method,
			Headers: args.Headers,
			Body:    []byte(args.Body),
			Actions: args.Actions,
		}

		result, err := client.Scrape(ctx, scrapeReq)
		if err != nil {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: fmt.Sprintf("Scrape failed: %v", err)},
				},
				IsError: true,
			}, nil, nil
		}

		if recorder != nil {
			//nolint:gosec // best-effort usage tracking
			_ = recorder.RecordUsage(
				ctx, "scrapedo", "", "scrape", "", args.URL, 1,
			)
		}

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{Text: result},
			},
		}, nil, nil
	})
}

func addSearchTool(server *mcpsdk.Server, router *search.Router, recorder UsageRecorder) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "web_search",
		Description: "Search the web using multiple engines. Returns markdown.",
	}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, args searchToolArgs) (*mcpsdk.CallToolResult, any, error) {
		return handleSearchTool(ctx, router, args, recorder)
	})
}

func handleSearchTool(
	ctx context.Context, router *search.Router, args searchToolArgs, recorder UsageRecorder,
) (*mcpsdk.CallToolResult, any, error) {
	if args.Query == "" {
		return searchErr("query is required"), nil, nil
	}

	engine := args.Engine
	if engine == "" {
		engine = "google"
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 10
	}

	p, err := router.Resolve(engine, args.Provider)
	if err != nil {
		return searchErr(fmt.Sprintf("Search failed: %v", err)), nil, nil
	}

	opts := search.Options{
		Engine: engine, Lang: args.Lang, Country: args.Country,
		Limit: limit, Page: 1,
	}

	resp, err := p.Search(ctx, args.Query, opts)
	if err != nil {
		return searchErr(fmt.Sprintf("Search failed: %v", err)), nil, nil
	}

	if recorder != nil {
		//nolint:gosec // best-effort usage tracking
		_ = recorder.RecordUsage(
			ctx, p.Name(), engine, "search", args.Query, "", 1,
		)
	}

	var buf bytes.Buffer
	if err := search.FormatMarkdown(&buf, resp); err != nil {
		return searchErr(fmt.Sprintf("Format failed: %v", err)), nil, nil
	}

	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: buf.String()}},
	}, nil, nil
}

func searchErr(msg string) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: msg}},
		IsError: true,
	}
}
