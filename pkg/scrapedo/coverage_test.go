package scrapedo_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

func TestMaskTokenInURL(t *testing.T) {
	c, _ := scrapedo.NewClient("test-token")
	u, _ := url.Parse("http://api.scrape.do?token=test-token&url=http://example.com")
	masked := c.MaskTokenInURL(u)
	assert.Contains(t, masked, "token=%2A%2A%2A")

	u2, _ := url.Parse("http://api.scrape.do?url=http://example.com")
	masked2 := c.MaskTokenInURL(u2)
	assert.NotContains(t, masked2, "token=")
}

func TestScrape_DebugLogging(t *testing.T) {
	// Setup a custom logger with Debug level
	logger := slog.New(slog.DiscardHandler)
	oldLogger := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(oldLogger)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client, _ := scrapedo.NewClient("test-token")
	client.SetBaseURL(server.URL)

	_, err := client.Scrape(context.Background(), scrapedo.ScrapeRequest{URL: "https://example.com"})
	require.NoError(t, err)
}

func TestPrepareQueryParams_Error(t *testing.T) {
	c, _ := scrapedo.NewClient("test-token")
	_, err := c.PrepareQueryParams(scrapedo.ScrapeRequest{
		URL: "http://example.com",
		Actions: []any{
			func() {}, // Cannot marshal func to JSON
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal browser actions")
}
