package search_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/pkg/search"
)

// ── Scrapedo Account ──

const mockScrapedoAccountResponse = `{
	"IsActive": true,
	"ConcurrentRequest": 5,
	"MaxMonthlyRequest": 1000,
	"RemainingConcurrentRequest": 5,
	"RemainingMonthlyRequest": 964
}`

func TestScrapedoProvider_Account_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/info")
		assert.NotEmpty(t, r.URL.Query().Get("token"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockScrapedoAccountResponse))
	}))
	defer srv.Close()

	p := search.NewScrapedoProvider("test-token")
	p.SetBaseURL(srv.URL)

	info, err := p.Account(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "scrapedo", info.Provider)
	assert.True(t, info.Active)
	assert.Equal(t, 36, info.UsedRequests)
	assert.Equal(t, 1000, info.MaxRequests)
	assert.Equal(t, 964, info.RemainingRequests)
	assert.Equal(t, 5, info.Concurrency)
}

func TestScrapedoProvider_Account_EmptyToken(t *testing.T) {
	p := search.NewScrapedoProvider("")
	_, err := p.Account(context.Background())
	require.Error(t, err)
}

func TestScrapedoProvider_Account_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"invalid token"}`))
	}))
	defer srv.Close()

	p := search.NewScrapedoProvider("bad-tok")
	p.SetBaseURL(srv.URL)

	_, err := p.Account(context.Background())
	require.Error(t, err)
}

func TestScrapedoProvider_Account_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	p := search.NewScrapedoProvider("tok")
	p.SetBaseURL(srv.URL)

	_, err := p.Account(context.Background())
	require.Error(t, err)
}

// ── SerpAPI Account ──

const mockSerpAPIAccountResponse = `{
	"plan_name": "Free Plan",
	"searches_per_month": 250,
	"total_searches_left": 238,
	"this_month_usage": 12,
	"account_rate_limit_per_hour": 250
}`

func TestSerpAPIProvider_Account_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/account")
		assert.NotEmpty(t, r.URL.Query().Get("api_key"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockSerpAPIAccountResponse))
	}))
	defer srv.Close()

	p := search.NewSerpAPIProvider("test-token")
	p.SetBaseURL(srv.URL)

	info, err := p.Account(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "serpapi", info.Provider)
	assert.Equal(t, "Free Plan", info.Plan)
	assert.True(t, info.Active)
	assert.Equal(t, 12, info.UsedRequests)
	assert.Equal(t, 250, info.MaxRequests)
	assert.Equal(t, 238, info.RemainingRequests)
	assert.Equal(t, 250, info.RateLimit)
}

func TestSerpAPIProvider_Account_EmptyToken(t *testing.T) {
	p := search.NewSerpAPIProvider("")
	_, err := p.Account(context.Background())
	require.Error(t, err)
}

func TestSerpAPIProvider_Account_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid key"}`))
	}))
	defer srv.Close()

	p := search.NewSerpAPIProvider("bad-tok")
	p.SetBaseURL(srv.URL)

	_, err := p.Account(context.Background())
	require.Error(t, err)
}

func TestSerpAPIProvider_Account_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{broken`))
	}))
	defer srv.Close()

	p := search.NewSerpAPIProvider("tok")
	p.SetBaseURL(srv.URL)

	_, err := p.Account(context.Background())
	require.Error(t, err)
}

// ── ScraperAPI Account ──

const mockScraperAPIAccountResponse = `{
	"concurrencyLimit": 5,
	"requestCount": 125,
	"requestLimit": 5000
}`

func TestScraperAPIProvider_Account_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/account")
		assert.NotEmpty(t, r.URL.Query().Get("api_key"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockScraperAPIAccountResponse))
	}))
	defer srv.Close()

	p := search.NewScraperAPIProvider("test-token")
	p.SetBaseURL(srv.URL)

	info, err := p.Account(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "scraperapi", info.Provider)
	assert.True(t, info.Active)
	assert.Equal(t, 125, info.UsedRequests)
	assert.Equal(t, 5000, info.MaxRequests)
	assert.Equal(t, 4875, info.RemainingRequests)
	assert.Equal(t, 5, info.Concurrency)
}

func TestScraperAPIProvider_Account_EmptyToken(t *testing.T) {
	p := search.NewScraperAPIProvider("")
	_, err := p.Account(context.Background())
	require.Error(t, err)
}

func TestScraperAPIProvider_Account_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	p := search.NewScraperAPIProvider("bad-tok")
	p.SetBaseURL(srv.URL)

	_, err := p.Account(context.Background())
	require.Error(t, err)
}

func TestScraperAPIProvider_Account_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()

	p := search.NewScraperAPIProvider("tok")
	p.SetBaseURL(srv.URL)

	_, err := p.Account(context.Background())
	require.Error(t, err)
}

// ── AccountChecker interface check ──

func TestAccountChecker_InterfaceCompliance(_ *testing.T) {
	var _ search.AccountChecker = search.NewScrapedoProvider("tok")
	var _ search.AccountChecker = search.NewSerpAPIProvider("tok")
	var _ search.AccountChecker = search.NewScraperAPIProvider("tok")
}
