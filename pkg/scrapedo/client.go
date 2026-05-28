package scrapedo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

// DefaultBaseURL is the standard API endpoint for Scrape.do.
const DefaultBaseURL = "http://api.scrape.do"

// ErrEmptyToken is returned when no token is provided.
var ErrEmptyToken = errors.New("scrape.do token is required")

// ErrEmptyURL is returned when the target URL is missing.
var ErrEmptyURL = errors.New("target URL is required")

// ErrAPI is a generic error wrapper for API-level failures.
var ErrAPI = errors.New("scrape.do API error")

// Cacher defines the interface for persistent caching.
type Cacher interface {
	// GetResult checks the cache for a matching request.
	GetResult(ctx context.Context, req ScrapeRequest) (string, bool, error)
	// SaveResult stores a new scrape result and performs cleanup.
	SaveResult(ctx context.Context, req ScrapeRequest, content string, metadata map[string]any) error
}

// ScrapeRequest holds parameters for the Scrape.do API call.
type ScrapeRequest struct {
	// URL is the target URL to scrape (Required).
	URL string
	// Render set to true for JavaScript-heavy websites that need browser rendering.
	Render bool
	// Super set to true to use residential and mobile proxies.
	Super bool
	// GeoCode is a 2-letter country code (e.g., "us", "gb", "de") to route requests through a specific location.
	GeoCode string
	// Session is a unique string to maintain a sticky session (same proxy IP).
	Session string
	// Device emulates a specific device: "desktop" (default), "mobile", or "tablet".
	Device string
	// Method is the HTTP method: "GET" (default), "POST", "PUT", etc.
	Method string
	// Output selects the response format. Empty defaults to "markdown".
	// Common values: "markdown", "raw" (the unmodified HTML from the target).
	Output string
	// Headers are custom HTTP headers to be forwarded.
	Headers map[string]string
	// Body is the data to be sent for POST/PUT requests.
	Body []byte
	// Actions are the browser actions to perform (for render=true).
	Actions []any

	// NoCache bypasses the local SQLite cache and forces a new API call without saving.
	NoCache bool
	// Refresh forces a new API call and stores the result as a new version in history.
	Refresh bool
}

// Client is a bare-bones HTTP client for the Scrape.do API.
type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
	cache      Cacher
}

// NewClient creates a new Scrape.do client with the provided token.
func NewClient(token string) (*Client, error) {
	if token == "" {
		return nil, ErrEmptyToken
	}
	return &Client{
		token:      token,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{},
	}, nil
}

// SetBaseURL overrides the default API endpoint (useful for testing).
func (c *Client) SetBaseURL(u string) {
	c.baseURL = u
}

// SetCache sets the optional caching layer for the client.
func (c *Client) SetCache(cache Cacher) {
	c.cache = cache
}

// Scrape performs a GET/POST request to Scrape.do API with the given parameters.
func (c *Client) Scrape(ctx context.Context, req ScrapeRequest) (string, error) {
	if req.URL == "" {
		return "", ErrEmptyURL
	}

	// 1. Check Cache
	if content, found := c.checkCache(ctx, req); found {
		return content, nil
	}

	// 2. Prepare Request
	httpReq, err := c.prepareHTTPRequest(ctx, req)
	if err != nil {
		return "", err
	}

	// 3. Execute Request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	c.logMetadata(resp.Header)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d: %s", ErrAPI, resp.StatusCode, string(bodyBytes))
	}

	// 4. Save to Cache
	c.saveToCache(ctx, req, resp, bodyBytes)

	return string(bodyBytes), nil
}

func (c *Client) checkCache(ctx context.Context, req ScrapeRequest) (string, bool) {
	if c.cache != nil && !req.NoCache && !req.Refresh {
		if content, found, err := c.cache.GetResult(ctx, req); err == nil && found {
			slog.Info("Cache hit", slog.String("url", req.URL))
			return content, true
		}
	}
	return "", false
}

func (c *Client) prepareHTTPRequest(ctx context.Context, req ScrapeRequest) (*http.Request, error) {
	reqURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	q, err := c.prepareQueryParams(req)
	if err != nil {
		return nil, err
	}
	reqURL.RawQuery = q.Encode()

	method := req.Method
	if method == "" {
		method = http.MethodGet
	}

	var bodyReader io.Reader
	if req.Body != nil {
		bodyReader = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, reqURL.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	if slog.Default().Enabled(ctx, slog.LevelDebug) {
		maskedURL := c.maskTokenInURL(reqURL)
		slog.Debug("Sending request to Scrape.do",
			slog.String("method", method),
			slog.String("url", maskedURL),
			slog.Any("headers", req.Headers),
		)
	}

	return httpReq, nil
}

func (c *Client) saveToCache(ctx context.Context, req ScrapeRequest, resp *http.Response, bodyBytes []byte) {
	if c.cache != nil && !req.NoCache {
		metadata := map[string]any{
			"status":            resp.StatusCode,
			"remaining_credits": resp.Header.Get("Scrape.do-Remaining-Credits"),
			"cost":              resp.Header.Get("Scrape.do-Request-Cost"),
		}
		if err := c.cache.SaveResult(ctx, req, string(bodyBytes), metadata); err != nil {
			slog.Warn("Failed to save result to cache", slog.Any("error", err))
		}
	}
}

func (c *Client) prepareQueryParams(req ScrapeRequest) (url.Values, error) {
	q := url.Values{}
	q.Set("token", c.token)
	q.Set("url", req.URL)
	output := req.Output
	if output == "" {
		output = "markdown"
	}
	q.Set("output", output)

	if req.Render {
		q.Set("render", "true")
	}
	if req.Super {
		q.Set("super", "true")
	}
	if req.GeoCode != "" {
		q.Set("geoCode", req.GeoCode)
	}
	if req.Session != "" {
		q.Set("session", req.Session)
	}
	if req.Device != "" {
		q.Set("device", req.Device)
	}

	if len(req.Actions) > 0 {
		actionsJSON, err := json.Marshal(req.Actions)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal browser actions: %w", err)
		}
		q.Set("playWithBrowser", string(actionsJSON))
	}

	if len(req.Headers) > 0 {
		q.Set("customHeaders", "true")
	}

	return q, nil
}

// logMetadata extracts Scrape.do custom headers and logs them to stderr.
func (c *Client) logMetadata(headers http.Header) {
	// Custom headers from Scrape.do:
	remaining := headers.Get("Scrape.do-Remaining-Credits")
	targetStatus := headers.Get("Scrape.do-Initial-Status-Code")
	cost := headers.Get("Scrape.do-Request-Cost")

	if remaining != "" || targetStatus != "" || cost != "" {
		slog.Info("Scrape.do metadata",
			slog.String("remaining_credits", remaining),
			slog.String("target_status", targetStatus),
			slog.String("cost", cost),
		)
	}
}

func (c *Client) maskTokenInURL(u *url.URL) string {
	q := u.Query()
	if q.Get("token") != "" {
		q.Set("token", "***")
	}
	u.RawQuery = q.Encode()
	return u.String()
}
