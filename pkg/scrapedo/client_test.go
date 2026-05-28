package scrapedo_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

// setBaseURL uses unsafe to set the unexported baseURL field for testing.
func setBaseURL(c *scrapedo.Client, url string) {
	// struct layout: token string, baseURL string, httpClient *http.Client
	ptr := unsafe.Pointer(c)
	baseURLPtr := (*string)(unsafe.Pointer(uintptr(ptr) + unsafe.Sizeof("")))
	*baseURLPtr = url
}

// setHTTPClient uses unsafe to set the unexported httpClient field for testing.
func setHTTPClient(c *scrapedo.Client, hc *http.Client) {
	ptr := unsafe.Pointer(c)
	hcPtr := (**http.Client)(unsafe.Pointer(uintptr(ptr) + unsafe.Sizeof("")*2))
	*hcPtr = hc
}

type errorReader struct{}

func (e *errorReader) Read(_ []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func (e *errorReader) Close() error {
	return nil
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestNewClient(t *testing.T) {
	t.Parallel()

	t.Run("empty token", func(t *testing.T) {
		t.Parallel()
		client, err := scrapedo.NewClient("")
		require.ErrorIs(t, err, scrapedo.ErrEmptyToken)
		assert.Nil(t, client)
	})

	t.Run("valid token", func(t *testing.T) {
		t.Parallel()
		client, err := scrapedo.NewClient("test-token")
		require.NoError(t, err)
		assert.NotNil(t, client)
	})
}

func TestClient_SetBaseURL(t *testing.T) {
	t.Parallel()
	client, _ := scrapedo.NewClient("token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client.SetBaseURL(server.URL)
	_, err := client.Scrape(context.Background(), scrapedo.ScrapeRequest{URL: "https://example.com"})
	require.NoError(t, err)
}

func TestScrape_Success(t *testing.T) {
	t.Parallel()

	expectedResponse := "# Markdown Result\nHello world!"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assert method
		assert.Equal(t, http.MethodPost, r.Method)

		// Assert query params
		q := r.URL.Query()
		assert.Equal(t, "test-token", q.Get("token"))
		assert.Equal(t, "https://example.com", q.Get("url"))
		assert.Equal(t, "markdown", q.Get("output"))
		assert.Equal(t, "true", q.Get("render"))
		assert.Equal(t, "true", q.Get("super"))
		assert.Equal(t, "us", q.Get("geoCode"))
		assert.Equal(t, "test-session", q.Get("session"))
		assert.Equal(t, "mobile", q.Get("device"))
		assert.Equal(t, "true", q.Get("customHeaders"))
		assert.JSONEq(t, `[{"action":"click","selector":"#btn"}]`, q.Get("playWithBrowser"))

		// Assert headers
		assert.Equal(t, "test-value", r.Header.Get("X-Test-Header"))

		// Assert body
		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, "test-body", string(body))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(expectedResponse))
	}))
	defer server.Close()

	client, err := scrapedo.NewClient("test-token")
	require.NoError(t, err)

	setBaseURL(client, server.URL)

	ctx := context.Background()
	req := scrapedo.ScrapeRequest{
		URL:     "https://example.com",
		Render:  true,
		Super:   true,
		GeoCode: "us",
		Session: "test-session",
		Device:  "mobile",
		Method:  "POST",
		Headers: map[string]string{"X-Test-Header": "test-value"},
		Body:    []byte("test-body"),
		Actions: []any{map[string]string{"action": "click", "selector": "#btn"}},
	}

	result, err := client.Scrape(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, expectedResponse, result)
}

func TestScrape_OutputOverride(t *testing.T) {
	t.Parallel()

	t.Run("default is markdown", func(t *testing.T) {
		t.Parallel()
		var gotOutput string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotOutput = r.URL.Query().Get("output")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))
		defer server.Close()

		client, _ := scrapedo.NewClient("token")
		setBaseURL(client, server.URL)

		_, err := client.Scrape(context.Background(), scrapedo.ScrapeRequest{URL: "https://example.com"})
		require.NoError(t, err)
		assert.Equal(t, "markdown", gotOutput)
	})

	t.Run("raw output forwards verbatim", func(t *testing.T) {
		t.Parallel()
		var gotOutput string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotOutput = r.URL.Query().Get("output")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html>raw</html>"))
		}))
		defer server.Close()

		client, _ := scrapedo.NewClient("token")
		setBaseURL(client, server.URL)

		body, err := client.Scrape(context.Background(), scrapedo.ScrapeRequest{
			URL:    "https://example.com",
			Output: "raw",
		})
		require.NoError(t, err)
		assert.Equal(t, "raw", gotOutput)
		assert.Equal(t, "<html>raw</html>", body)
	})
}

func TestScrape_Failures(t *testing.T) {
	t.Parallel()

	t.Run("empty url", func(t *testing.T) {
		t.Parallel()
		client, _ := scrapedo.NewClient("token")
		_, err := client.Scrape(context.Background(), scrapedo.ScrapeRequest{URL: ""})
		require.ErrorIs(t, err, scrapedo.ErrEmptyURL)
	})

	t.Run("api error", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"success":false,"message":"Invalid token."}`))
		}))
		defer server.Close()

		client, _ := scrapedo.NewClient("invalid")
		setBaseURL(client, server.URL)

		_, err := client.Scrape(context.Background(), scrapedo.ScrapeRequest{URL: "https://example.com"})
		require.ErrorIs(t, err, scrapedo.ErrAPI)
		require.ErrorContains(t, err, "scrape.do API error: status 403")
		require.ErrorContains(t, err, "Invalid token.")
	})

	t.Run("invalid base url", func(t *testing.T) {
		t.Parallel()
		client, _ := scrapedo.NewClient("token")
		setBaseURL(client, ":") // invalid url

		_, err := client.Scrape(context.Background(), scrapedo.ScrapeRequest{URL: "https://example.com"})
		require.ErrorContains(t, err, "failed to parse base URL")
	})

	t.Run("http client error", func(t *testing.T) {
		t.Parallel()
		client, _ := scrapedo.NewClient("token")
		setHTTPClient(client, &http.Client{
			Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				return nil, assert.AnError
			}),
		})

		_, err := client.Scrape(context.Background(), scrapedo.ScrapeRequest{URL: "https://example.com"})
		require.ErrorContains(t, err, "http request failed")
	})

	t.Run("read error", func(t *testing.T) {
		t.Parallel()
		client, _ := scrapedo.NewClient("token")
		setHTTPClient(client, &http.Client{
			Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       &errorReader{},
				}, nil
			}),
		})

		_, err := client.Scrape(context.Background(), scrapedo.ScrapeRequest{URL: "https://example.com"})
		require.ErrorContains(t, err, "failed to read response body")
	})

	t.Run("invalid action", func(t *testing.T) {
		t.Parallel()
		client, _ := scrapedo.NewClient("token")
		_, err := client.Scrape(context.Background(), scrapedo.ScrapeRequest{
			URL: "https://example.com",
			Actions: []any{
				func() {}, // functions cannot be marshaled to JSON
			},
		})
		require.ErrorContains(t, err, "failed to marshal browser actions")
	})

	t.Run("invalid method", func(t *testing.T) {
		t.Parallel()
		client, _ := scrapedo.NewClient("token")
		_, err := client.Scrape(context.Background(), scrapedo.ScrapeRequest{
			URL:    "https://example.com",
			Method: "\x7f", // invalid method
		})
		require.ErrorContains(t, err, "failed to create http request")
	})
}

func TestLogMetadata(t *testing.T) {
	t.Parallel()

	client, _ := scrapedo.NewClient("test-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Scrape.do-Remaining-Credits", "100")
		w.Header().Set("Scrape.do-Initial-Status-Code", "200")
		w.Header().Set("Scrape.do-Request-Cost", "1")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	setBaseURL(client, server.URL)
	_, err := client.Scrape(context.Background(), scrapedo.ScrapeRequest{URL: "https://example.com"})
	require.NoError(t, err)

	// No metadata headers
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server2.Close()

	setBaseURL(client, server2.URL)
	_, err = client.Scrape(context.Background(), scrapedo.ScrapeRequest{URL: "https://example.com"})
	require.NoError(t, err)
}
