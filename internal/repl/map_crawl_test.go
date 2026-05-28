package repl_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/internal/repl"
	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

func TestREPL_MapCommand(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := r.URL.Query().Get("url")
		if target == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`<a href="/about">About</a><a href="/docs">Docs</a>`))
	}))
	defer ts.Close()

	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)
	client.SetBaseURL(ts.URL)
	s := repl.NewShell(client)
	ctx := context.Background()

	t.Run("basic map", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "map http://example.com")
		require.NoError(t, err)
	})

	t.Run("map with search filter", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "map http://example.com search=about")
		require.NoError(t, err)
	})

	t.Run("map with limit", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "map http://example.com limit=1")
		require.NoError(t, err)
	})

	t.Run("map no url", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "map")
		require.ErrorIs(t, err, repl.ErrInvalidUsage)
	})
}

func TestREPL_CrawlCommand(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`# Page content`))
	}))
	defer ts.Close()

	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)
	client.SetBaseURL(ts.URL)
	s := repl.NewShell(client)
	ctx := context.Background()

	t.Run("basic crawl", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "crawl http://example.com")
		require.NoError(t, err)
	})

	t.Run("crawl with depth and limit", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "crawl http://example.com depth=2 limit=5")
		require.NoError(t, err)
	})

	t.Run("crawl no url", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "crawl")
		require.ErrorIs(t, err, repl.ErrInvalidUsage)
	})
}

func TestREPL_CrawlError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)
	client.SetBaseURL(ts.URL)
	s := repl.NewShell(client)

	err = s.ExecuteCommand(context.Background(), "crawl http://example.com")
	assert.NoError(t, err)
}
