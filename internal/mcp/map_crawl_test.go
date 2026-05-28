package mcp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	mcp_sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/internal/mcp"
	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

func setupMCPSession(t *testing.T, ts *httptest.Server) *mcp_sdk.ClientSession {
	t.Helper()

	client, err := scrapedo.NewClient("test-token")
	require.NoError(t, err)
	client.SetBaseURL(ts.URL)

	rec := &mockRecorder{}

	server, err := mcp.NewServerWithClientAndRecorder(client, nil, rec)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	clientTransport, serverTransport := mcp_sdk.NewInMemoryTransports()
	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	mcpClient := mcp_sdk.NewClient(&mcp_sdk.Implementation{Name: "test", Version: "1.0.0"}, nil)
	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { session.Close() })

	return session
}

func TestMCPServer_MapTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<a href="/about">About</a><a href="/docs">Docs</a>`))
	}))
	defer ts.Close()

	session := setupMCPSession(t, ts)

	args := map[string]any{"url": "http://example.com"}
	res, err := session.CallTool(context.Background(), &mcp_sdk.CallToolParams{
		Name:      "map_urls",
		Arguments: args,
	})

	require.NoError(t, err)
	assert.False(t, res.IsError)
	require.NotEmpty(t, res.Content)

	textContent, ok := res.Content[0].(*mcp_sdk.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Discovered")
}

func TestMCPServer_MapTool_EmptyURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	session := setupMCPSession(t, ts)

	args := map[string]any{"url": ""}
	res, err := session.CallTool(context.Background(), &mcp_sdk.CallToolParams{
		Name:      "map_urls",
		Arguments: args,
	})

	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestMCPServer_MapTool_WithSearch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<a href="/about">About</a><a href="/docs">Docs</a>`))
	}))
	defer ts.Close()

	session := setupMCPSession(t, ts)

	args := map[string]any{"url": "http://example.com", "search": "about"}
	res, err := session.CallTool(context.Background(), &mcp_sdk.CallToolParams{
		Name:      "map_urls",
		Arguments: args,
	})

	require.NoError(t, err)
	assert.False(t, res.IsError)

	textContent, ok := res.Content[0].(*mcp_sdk.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "about")
	assert.NotContains(t, textContent.Text, "docs")
}

func TestMCPServer_CrawlTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`# Page content`))
	}))
	defer ts.Close()

	session := setupMCPSession(t, ts)

	args := map[string]any{"url": "http://example.com", "maxPages": 1}
	res, err := session.CallTool(context.Background(), &mcp_sdk.CallToolParams{
		Name:      "crawl_site",
		Arguments: args,
	})

	require.NoError(t, err)
	assert.False(t, res.IsError)
	require.NotEmpty(t, res.Content)

	textContent, ok := res.Content[0].(*mcp_sdk.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Page 1")
}

func TestMCPServer_CrawlTool_EmptyURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	session := setupMCPSession(t, ts)

	args := map[string]any{"url": ""}
	res, err := session.CallTool(context.Background(), &mcp_sdk.CallToolParams{
		Name:      "crawl_site",
		Arguments: args,
	})

	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestMCPServer_ToolsRegistered(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	session := setupMCPSession(t, ts)

	tools, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)

	toolNames := make(map[string]bool)
	for _, tool := range tools.Tools {
		toolNames[tool.Name] = true
	}

	assert.True(t, toolNames["scrape_url"], "scrape_url should be registered")
	assert.True(t, toolNames["map_urls"], "map_urls should be registered")
	assert.True(t, toolNames["crawl_site"], "crawl_site should be registered")
}
