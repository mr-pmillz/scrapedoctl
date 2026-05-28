package scrapedo_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

func TestCrawl_BasicTwoPages(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	pages := map[string]string{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := r.URL.Query().Get("url")

		mu.Lock()
		defer mu.Unlock()

		switch target {
		case "http://site.local/", "":
			pages["root"] = target
			_, _ = w.Write([]byte(`# Home\n[About](/about)\n<a href="/contact">Contact</a>`))
		case "http://site.local/about":
			pages["about"] = target
			_, _ = w.Write([]byte(`# About page`))
		case "http://site.local/contact":
			pages["contact"] = target
			_, _ = w.Write([]byte(`# Contact page`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client, err := scrapedo.NewClient("test-token")
	require.NoError(t, err)
	client.SetBaseURL(ts.URL)

	var results []scrapedo.CrawlResult

	err = client.Crawl(
		context.Background(),
		"http://site.local/",
		scrapedo.CrawlOptions{MaxDepth: 1, MaxPages: 10},
		func(r scrapedo.CrawlResult) { results = append(results, r) },
	)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
	assert.Equal(t, "http://site.local/", results[0].URL)
	assert.NoError(t, results[0].Error)
}

func TestCrawl_RespectsMaxPages(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<a href="/a">A</a><a href="/b">B</a><a href="/c">C</a>`))
	}))
	defer ts.Close()

	client, err := scrapedo.NewClient("test-token")
	require.NoError(t, err)
	client.SetBaseURL(ts.URL)

	var results []scrapedo.CrawlResult

	err = client.Crawl(
		context.Background(),
		"http://site.local/",
		scrapedo.CrawlOptions{MaxDepth: 2, MaxPages: 2},
		func(r scrapedo.CrawlResult) { results = append(results, r) },
	)

	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestCrawl_RespectsMaxDepth(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<a href="/deep">Deep</a>`))
	}))
	defer ts.Close()

	client, err := scrapedo.NewClient("test-token")
	require.NoError(t, err)
	client.SetBaseURL(ts.URL)

	var results []scrapedo.CrawlResult

	err = client.Crawl(
		context.Background(),
		"http://site.local/",
		scrapedo.CrawlOptions{MaxDepth: 0, MaxPages: 100},
		func(r scrapedo.CrawlResult) { results = append(results, r) },
	)

	require.NoError(t, err)
	// Only the start page should be scraped (depth 0, no following).
	assert.Len(t, results, 1)
}

func TestCrawl_EmptyURL(t *testing.T) {
	t.Parallel()

	client, err := scrapedo.NewClient("test-token")
	require.NoError(t, err)

	err = client.Crawl(
		context.Background(), "", scrapedo.CrawlOptions{},
		func(_ scrapedo.CrawlResult) {},
	)

	require.ErrorIs(t, err, scrapedo.ErrEmptyURL)
}

func TestCrawl_CancelledContext(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`content`))
	}))
	defer ts.Close()

	client, err := scrapedo.NewClient("test-token")
	require.NoError(t, err)
	client.SetBaseURL(ts.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = client.Crawl(
		ctx, "http://site.local/", scrapedo.CrawlOptions{MaxPages: 5},
		func(_ scrapedo.CrawlResult) {},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "crawl cancelled")
}

func TestCrawl_HandlesScrapeErrors(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer ts.Close()

	client, err := scrapedo.NewClient("test-token")
	require.NoError(t, err)
	client.SetBaseURL(ts.URL)

	var results []scrapedo.CrawlResult

	err = client.Crawl(
		context.Background(),
		"http://site.local/",
		scrapedo.CrawlOptions{MaxPages: 1},
		func(r scrapedo.CrawlResult) { results = append(results, r) },
	)

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Error(t, results[0].Error)
}
