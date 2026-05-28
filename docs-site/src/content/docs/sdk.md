---
title: Go SDK
description: Embed scrapedoctl in your own Go program via the pkg/scrapedo client — GET with super=true, raw-HTML output, concurrency, retry, caching, and more.
---

The `github.com/mr-pmillz/scrapedoctl/pkg/scrapedo` package is the same client that powers the `scrapedoctl` CLI and MCP server. You can import it directly from any Go module to add Scrape.do-powered fetching, residential-proxy bypassing, and (optionally) the SQLite cache to your own application.

## Install

Inside your module:

```bash
go get github.com/mr-pmillz/scrapedoctl/pkg/scrapedo@latest
```

The package has a zero-dependency public surface — it only uses the standard library. Caching is opt-in via a small `Cacher` interface (see [Persistent cache](#persistent-cache-optional) below).

## GET a URL with `super=true`

The minimal program below scrapes a URL through Scrape.do's residential / mobile proxy pool. Setting `Super: true` on the request is the SDK equivalent of the `super=true` query parameter on the underlying API call. `Method` defaults to `GET`, so no extra flag is needed.

```go
// File: main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

func main() {
	token := os.Getenv("SCRAPEDO_TOKEN")
	if token == "" {
		log.Fatal("SCRAPEDO_TOKEN is required")
	}

	client, err := scrapedo.NewClient(token)
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	markdown, err := client.Scrape(ctx, scrapedo.ScrapeRequest{
		URL:   "https://example.com",
		Super: true, // route through residential/mobile proxies (super=true)
	})
	if err != nil {
		log.Fatalf("scrape: %v", err)
	}

	fmt.Println(markdown)
}
```

Run it:

```bash
export SCRAPEDO_TOKEN="your-scrape-do-token"
go run .
```

The client requests Markdown by default (it sets `output=markdown` on every call) which is ideal for piping into an LLM. See [Get the raw HTML](#get-the-raw-html-instead-of-markdown) for how to override that.

## How `Super` maps to the API

When the call is built, `Super: true` is translated into a `super=true` query parameter on the outbound HTTPS request to `http://api.scrape.do`. The full request that gets executed looks roughly like:

```
GET http://api.scrape.do/?token=***&url=https%3A%2F%2Fexample.com&output=markdown&super=true
```

Use `Super` for sites that block datacenter IPs or aggressively rate-limit. It costs more credits per request than the standard pool, so prefer it only when a normal request is being blocked.

## `ScrapeRequest` reference

```go
type ScrapeRequest struct {
    URL     string            // required
    Render  bool              // headless-browser rendering (render=true)
    Super   bool              // residential/mobile proxies (super=true)
    GeoCode string            // 2-letter country code (e.g. "us", "gb")
    Session string            // sticky session id (reuses the same proxy IP)
    Device  string            // "desktop" (default), "mobile", "tablet"
    Method  string            // HTTP method; defaults to "GET"
    Output  string            // "markdown" (default) or "raw" for raw HTML
    Headers map[string]string // forwarded as customHeaders=true
    Body    []byte            // for POST/PUT requests
    Actions []any             // playWithBrowser actions when Render=true

    NoCache bool              // bypass the local cache and do not save
    Refresh bool              // force a new call and store a new history version
}
```

The `URL` field is the only required value. Any zero/empty field is omitted from the outbound query string (except `Output`, which defaults to `"markdown"`).

## Examples

### Get the raw HTML instead of Markdown

Set `Output: "raw"` to disable Scrape.do's Markdown conversion and stream back the target page as-is. This is what you want when you need to feed the response into an HTML parser like `goquery` or `golang.org/x/net/html`, or when the page's structure matters more than its text.

```go
html, err := client.Scrape(ctx, scrapedo.ScrapeRequest{
    URL:    "https://example.com",
    Super:  true,
    Output: "raw", // returns unmodified HTML; outbound param: output=raw
})
if err != nil {
    log.Fatalf("scrape: %v", err)
}

fmt.Println(html[:200])
```

Pipe the response straight into `goquery` to walk the DOM:

```go
import "github.com/PuerkitoBio/goquery"

doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
if err != nil {
    log.Fatalf("parse html: %v", err)
}

doc.Find("article h2").Each(func(_ int, s *goquery.Selection) {
    fmt.Println(strings.TrimSpace(s.Text()))
})
```

### Geo-targeted GET

```go
markdown, err := client.Scrape(ctx, scrapedo.ScrapeRequest{
    URL:     "https://example.com/pricing",
    Super:   true,
    GeoCode: "gb",
})
```

### JS-rendered page with a sticky session

```go
markdown, err := client.Scrape(ctx, scrapedo.ScrapeRequest{
    URL:     "https://shop.example.com/cart",
    Render:  true,
    Super:   true,
    Session: "user-42",
})
```

### POST with a JSON body and custom headers

```go
body := []byte(`{"query":"laptops"}`)
markdown, err := client.Scrape(ctx, scrapedo.ScrapeRequest{
    URL:    "https://api.example.com/search",
    Method: "POST",
    Body:   body,
    Headers: map[string]string{
        "Content-Type": "application/json",
        "Accept":       "application/json",
    },
})
```

### Discover same-domain links

```go
links := scrapedo.ExtractLinks(markdown, "https://example.com")
for _, l := range links {
    fmt.Println(l)
}
```

### Recursive breadth-first crawl

```go
err := client.Crawl(ctx, "https://example.com",
    scrapedo.CrawlOptions{MaxDepth: 2, MaxPages: 25},
    func(r scrapedo.CrawlResult) {
        if r.Error != nil {
            log.Printf("%s: %v", r.URL, r.Error)
            return
        }
        fmt.Printf("[%d] %s (%d bytes, %d links)\n", r.Depth, r.URL, r.Size, len(r.Links))
    },
)
```

### Concurrent bulk scraping

Fan a slice of URLs across a small worker pool. `golang.org/x/sync/errgroup` gives you the typical "stop on first error" semantics with a built-in concurrency limit.

```go
import "golang.org/x/sync/errgroup"

func scrapeAll(ctx context.Context, client *scrapedo.Client, urls []string) (map[string]string, error) {
    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(5) // cap in-flight requests

    var (
        mu      sync.Mutex
        results = make(map[string]string, len(urls))
    )

    for _, u := range urls {
        u := u // capture before the goroutine
        g.Go(func() error {
            md, err := client.Scrape(ctx, scrapedo.ScrapeRequest{URL: u, Super: true})
            if err != nil {
                return fmt.Errorf("%s: %w", u, err)
            }
            mu.Lock()
            results[u] = md
            mu.Unlock()
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return results, err
    }
    return results, nil
}
```

The `errgroup`-derived `ctx` is canceled the moment any goroutine returns an error, so siblings stop pulling new pages instead of racing to completion.

### Retry transient failures with backoff

`scrapedo.ErrAPI` wraps non-200 responses. A simple exponential backoff covers proxy hiccups and upstream 5xx without pulling in extra dependencies:

```go
func scrapeWithRetry(
    ctx context.Context, client *scrapedo.Client, req scrapedo.ScrapeRequest, attempts int,
) (string, error) {
    var lastErr error
    delay := 500 * time.Millisecond
    for i := 0; i < attempts; i++ {
        body, err := client.Scrape(ctx, req)
        if err == nil {
            return body, nil
        }
        lastErr = err

        // Only retry transient API errors — bail out fast on input errors.
        if !errors.Is(err, scrapedo.ErrAPI) {
            return "", err
        }

        select {
        case <-ctx.Done():
            return "", ctx.Err()
        case <-time.After(delay):
        }
        delay *= 2
    }
    return "", fmt.Errorf("giving up after %d attempts: %w", attempts, lastErr)
}
```

Pair this with `Refresh: true` if you also want each retry to record a new history entry instead of overwriting a cached failure.

### Stream the response to a file

For large pages (PDFs, sitemaps, long articles) it's often cleaner to dump the body straight to disk rather than hold it in memory:

```go
body, err := client.Scrape(ctx, scrapedo.ScrapeRequest{
    URL:    "https://example.com/long-article",
    Output: "raw",
})
if err != nil {
    log.Fatalf("scrape: %v", err)
}

if err := os.WriteFile("article.html", []byte(body), 0o644); err != nil {
    log.Fatalf("write: %v", err)
}
```

## Persistent cache (optional)

The CLI ships an SQLite-backed cache. To get the same behaviour in the SDK, pass any type that satisfies the `Cacher` interface to `client.SetCache`:

```go
type Cacher interface {
    GetResult(ctx context.Context, req ScrapeRequest) (string, bool, error)
    SaveResult(ctx context.Context, req ScrapeRequest, content string, metadata map[string]any) error
}
```

When a cache is attached, `Scrape` returns the cached body on hits and saves successful responses on misses unless the request sets `NoCache: true`. Use `Refresh: true` to force a fresh fetch while still recording the result as a new history version.

### In-memory cache implementation

A throwaway in-memory cache is enough for tests, short-lived scripts, or scratching a deduplication itch inside a single process. Anything from `crypto/sha256` is fine as a key — here is a minimal `sync.Map`-backed implementation:

```go
type memCache struct{ m sync.Map }

func (c *memCache) key(req scrapedo.ScrapeRequest) string {
    sum := sha256.Sum256([]byte(req.Method + "|" + req.URL + "|" + req.Output))
    return hex.EncodeToString(sum[:])
}

func (c *memCache) GetResult(_ context.Context, req scrapedo.ScrapeRequest) (string, bool, error) {
    v, ok := c.m.Load(c.key(req))
    if !ok {
        return "", false, nil
    }
    return v.(string), true, nil
}

func (c *memCache) SaveResult(_ context.Context, req scrapedo.ScrapeRequest, content string, _ map[string]any) error {
    c.m.Store(c.key(req), content)
    return nil
}

// wire it up
client.SetCache(&memCache{})
```

For production you'd usually swap this for a process-wide LRU, Redis, or the SQLite cache exposed by the `internal/cache` package via the CLI.

## Error handling

Sentinel errors live on the package so you can branch with `errors.Is`:

| Error | Returned when |
|-------|---------------|
| `scrapedo.ErrEmptyToken` | `NewClient("")` was called |
| `scrapedo.ErrEmptyURL`   | `ScrapeRequest.URL` was empty |
| `scrapedo.ErrAPI`        | Scrape.do returned a non-200 status |

```go
if errors.Is(err, scrapedo.ErrAPI) {
    // upstream rejected the request — inspect err.Error() for the status and body
}
```

## Observability

The client uses the standard library's `log/slog`. To inspect every outbound URL (with the token masked), wire a debug-level logger before calling `Scrape`:

```go
slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
```

Successful responses also log the remaining-credit and per-request cost headers from Scrape.do at info level.
