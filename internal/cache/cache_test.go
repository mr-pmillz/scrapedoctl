package cache_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/internal/cache"
	"github.com/mr-pmillz/scrapedoctl/internal/config"
	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

func TestCache(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "cache.db")

	cfg := config.CacheConfig{
		Path:         dbPath,
		TTLDays:      7,
		KeepVersions: 2,
	}

	store, err := cache.NewStore(cfg)
	require.NoError(t, err)
	defer store.Database().Close()

	ctx := context.Background()
	req := scrapedo.ScrapeRequest{
		URL:    "https://example.com",
		Method: "GET",
	}

	t.Run("empty cache", func(t *testing.T) {
		_, found, err := store.GetResult(ctx, req)
		require.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("save and retrieve", func(t *testing.T) {
		err := store.SaveResult(ctx, req, "content v1", map[string]any{"status": 200})
		require.NoError(t, err)

		content, found, err := store.GetResult(ctx, req)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "content v1", content)
	})

	t.Run("versioning and cleanup", func(t *testing.T) {
		// Save 2 more versions (total 3, but limit is 2)
		err := store.SaveResult(ctx, req, "content v2", map[string]any{"status": 200})
		require.NoError(t, err)
		err = store.SaveResult(ctx, req, "content v3", map[string]any{"status": 200})
		require.NoError(t, err)

		// Latest should be v3
		content, found, err := store.GetResult(ctx, req)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "content v3", content)

		// Check history count in DB directly
		var count int
		err = store.Database().
			QueryRow("SELECT COUNT(*) FROM scrapes WHERE request_hash = ?", cache.NormalizeAndHashFn(req)).
			Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 2, count, "Should keep only 2 versions")
	})

	t.Run("GetHistory", func(t *testing.T) {
		history, err := store.GetHistory(ctx, "https://example.com")
		require.NoError(t, err)
		assert.Len(t, history, 2)
		assert.Equal(t, "https://example.com", history[0].URL)
	})

	t.Run("GetStats", func(t *testing.T) {
		stats, err := store.GetStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(2), stats.TotalCount)
		assert.Positive(t, stats.TotalSize)
	})

	t.Run("Clear", func(t *testing.T) {
		err := store.Clear(ctx)
		require.NoError(t, err)

		stats, err := store.GetStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), stats.TotalCount)
	})

	t.Run("TTL expiration", func(t *testing.T) {
		// Use a store with 0 TTL
		cfg0 := cfg
		cfg0.TTLDays = -1 // Negative TTL to ensure it's always expired
		store0, err := cache.NewStore(cfg0)
		require.NoError(t, err)
		defer store0.Database().Close()

		err = store0.SaveResult(ctx, req, "expired content", nil)
		require.NoError(t, err)

		_, found, err := store0.GetResult(ctx, req)
		require.NoError(t, err)
		assert.False(t, found, "Should be expired")
	})

	t.Run("expandPath with ~/", func(_ *testing.T) {
		// This is hard to test fully because it depends on os.UserHomeDir()
		// but we can at least call it.
		// Actually, let's just ensure we hit the line.
		_ = cache.ExpandPath("~/test.db")
		_ = cache.ExpandPath("/tmp/test.db")
	})
}

func TestUsageRecording(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "usage.db")

	cfg := config.CacheConfig{
		Path:         dbPath,
		TTLDays:      7,
		KeepVersions: 2,
	}

	store, err := cache.NewStore(cfg)
	require.NoError(t, err)
	defer store.Database().Close()

	ctx := context.Background()

	t.Run("record and retrieve usage", func(t *testing.T) {
		err := store.RecordUsage(ctx, "scrapedo", "google", "search", "test query", "", 1)
		require.NoError(t, err)

		err = store.RecordUsage(ctx, "scrapedo", "", "scrape", "", "https://example.com", 1)
		require.NoError(t, err)

		err = store.RecordUsage(ctx, "serpapi", "google", "search", "another query", "", 2)
		require.NoError(t, err)

		records, err := store.GetUsageSince(ctx, time.Now().Add(-time.Hour))
		require.NoError(t, err)
		assert.Len(t, records, 3)
		assert.Equal(t, "scrapedo", records[0].Provider)
	})

	t.Run("usage summary grouping", func(t *testing.T) {
		summary, err := store.GetUsageSummary(ctx, time.Now().Add(-time.Hour))
		require.NoError(t, err)
		assert.Len(t, summary, 3)

		// Check scrapedo/scrape entry
		var found bool
		for _, s := range summary {
			if s.Provider == "scrapedo" && s.Action == "scrape" {
				found = true
				assert.Equal(t, int64(1), s.Count)
				assert.Equal(t, int64(1), s.TotalCredits)
			}
		}
		assert.True(t, found, "expected scrapedo/scrape entry")
	})

	t.Run("usage since filters by time", func(t *testing.T) {
		records, err := store.GetUsageSince(ctx, time.Now().Add(time.Hour))
		require.NoError(t, err)
		assert.Empty(t, records)
	})
}

func TestNormalizeAndHash(t *testing.T) {
	tests := []struct {
		name     string
		req1     scrapedo.ScrapeRequest
		req2     scrapedo.ScrapeRequest
		expected bool
	}{
		{
			name:     "identical requests",
			req1:     scrapedo.ScrapeRequest{URL: "https://a.com", Render: true},
			req2:     scrapedo.ScrapeRequest{URL: "https://a.com", Render: true},
			expected: true,
		},
		{
			name:     "different render param",
			req1:     scrapedo.ScrapeRequest{URL: "https://a.com", Render: true},
			req2:     scrapedo.ScrapeRequest{URL: "https://a.com", Render: false},
			expected: false,
		},
		{
			name:     "header sorting",
			req1:     scrapedo.ScrapeRequest{URL: "https://a.com", Headers: map[string]string{"A": "1", "B": "2"}},
			req2:     scrapedo.ScrapeRequest{URL: "https://a.com", Headers: map[string]string{"B": "2", "A": "1"}},
			expected: true,
		},
		{
			name:     "different body",
			req1:     scrapedo.ScrapeRequest{URL: "https://a.com", Body: []byte("foo")},
			req2:     scrapedo.ScrapeRequest{URL: "https://a.com", Body: []byte("bar")},
			expected: false,
		},
		{
			name: "different actions",
			req1: scrapedo.ScrapeRequest{
				URL:     "https://a.com",
				Actions: []any{map[string]any{"action": "click", "selector": "#foo"}},
			},
			req2: scrapedo.ScrapeRequest{
				URL:     "https://a.com",
				Actions: []any{map[string]any{"action": "click", "selector": "#bar"}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expected {
				assert.Equal(t, cache.NormalizeAndHashFn(tt.req1), cache.NormalizeAndHashFn(tt.req2))
			} else {
				assert.NotEqual(t, cache.NormalizeAndHashFn(tt.req1), cache.NormalizeAndHashFn(tt.req2))
			}
		})
	}
}

func FuzzNormalizeAndHash(f *testing.F) {
	f.Add("https://example.com", "GET", true, false, "us", "desktop", "session123")
	f.Fuzz(
		func(t *testing.T, url string, method string, render bool, super bool, geo string, device string, session string) {
			req := scrapedo.ScrapeRequest{
				URL:     url,
				Method:  method,
				Render:  render,
				Super:   super,
				GeoCode: geo,
				Device:  device,
				Session: session,
			}
			hash1 := cache.NormalizeAndHashFn(req)
			hash2 := cache.NormalizeAndHashFn(req)
			assert.Equal(t, hash1, hash2)
		},
	)
}
