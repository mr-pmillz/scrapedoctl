package scrapedo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// AccountInfo mirrors the JSON returned by Scrape.do's GET /info endpoint.
// Field names match the upstream response exactly so json.Unmarshal works
// without struct tags.
type AccountInfo struct {
	IsActive                   bool
	ConcurrentRequest          int
	MaxMonthlyRequest          int
	RemainingConcurrentRequest int
	RemainingMonthlyRequest    int
}

// Info fetches the account's current concurrency cap and remaining monthly
// quota from Scrape.do's /info endpoint. Use it to size per-run concurrency
// safely (e.g., leave headroom for other pipelines sharing the same token)
// or to short-circuit a batch when no monthly quota is left.
//
// On any non-200 response Info returns an error wrapping ErrAPI; callers
// can branch with errors.Is(err, scrapedo.ErrAPI).
func (c *Client) Info(ctx context.Context) (*AccountInfo, error) {
	reqURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}
	reqURL.Path = singleSlashJoin(reqURL.Path, "info")
	q := url.Values{}
	q.Set("token", c.token)
	reqURL.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("new info request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("info request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read info body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d: %s", ErrAPI, resp.StatusCode, string(bodyBytes))
	}

	var info AccountInfo
	if err := json.Unmarshal(bodyBytes, &info); err != nil {
		return nil, fmt.Errorf("decode info: %w", err)
	}
	return &info, nil
}

// singleSlashJoin appends segment to base ensuring there is exactly one
// "/" between them. Avoids "//info" when base is "http://api.scrape.do/"
// and "info" when base has an empty path.
func singleSlashJoin(base, segment string) string {
	if base == "" {
		return "/" + segment
	}
	if base[len(base)-1] == '/' {
		return base + segment
	}
	return base + "/" + segment
}
