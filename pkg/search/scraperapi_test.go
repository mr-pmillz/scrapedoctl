package search_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/mr-pmillz/scrapedoctl/pkg/search"
)

const scraperAPIEngineGoogle = "google"

const mockScraperAPIResponse = `{
  "search_information": {"query_displayed": "go language"},
  "organic_results": [
    {"position": 0, "title": "Go Programming", "link": "https://go.dev", "snippet": "An open-source language", "displayed_link": "go.dev", "highlighs": ["Go"]}
  ],
  "related_searches": [{"query": "golang tutorial"}],
  "pagination": {"pages_count": 8, "current_page": 1, "next_page_url": "https://api.scraperapi.com/structured/google/search?api_key=TOKEN&query=golang&page=2"}
}`

func newScraperAPITestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func TestScraperAPIProvider_Name(t *testing.T) {
	p := search.NewScraperAPIProvider("tok")
	if got := p.Name(); got != "scraperapi" {
		t.Fatalf("Name() = %q, want %q", got, "scraperapi")
	}
}

func TestScraperAPIProvider_Engines(t *testing.T) {
	p := search.NewScraperAPIProvider("tok")
	engines := p.Engines()
	if len(engines) != 1 || engines[0] != scraperAPIEngineGoogle {
		t.Fatalf("Engines() = %v, want [%s]", engines, scraperAPIEngineGoogle)
	}
}

func TestScraperAPIProvider_Search_Success(t *testing.T) {
	srv := newScraperAPITestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockScraperAPIResponse))
	})
	defer srv.Close()

	p := search.NewScraperAPIProvider("test-token")
	p.SetBaseURL(srv.URL)

	resp, err := p.Search(context.Background(), "go language", search.Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Query != "go language" {
		t.Errorf("Query = %q, want %q", resp.Query, "go language")
	}
	if resp.Engine != scraperAPIEngineGoogle {
		t.Errorf("Engine = %q, want %q", resp.Engine, scraperAPIEngineGoogle)
	}
	if resp.Provider != "scraperapi" {
		t.Errorf("Provider = %q, want %q", resp.Provider, "scraperapi")
	}
	if len(resp.Results) != 1 {
		t.Fatalf("got %d results, want 1", len(resp.Results))
	}

	r := resp.Results[0]
	if r.Position != 1 {
		t.Errorf("Position = %d, want 1 (0-based API position mapped to 1-based)", r.Position)
	}
	if r.Title != "Go Programming" {
		t.Errorf("Title = %q, want %q", r.Title, "Go Programming")
	}
	if r.URL != "https://go.dev" {
		t.Errorf("URL = %q, want %q", r.URL, "https://go.dev")
	}
	if r.Snippet != "An open-source language" {
		t.Errorf("Snippet = %q, want %q", r.Snippet, "An open-source language")
	}
	if r.DisplayedURL != "go.dev" {
		t.Errorf("DisplayedURL = %q, want %q", r.DisplayedURL, "go.dev")
	}

	if resp.Metadata["query_displayed"] != "go language" {
		t.Errorf("query_displayed = %v, want %q", resp.Metadata["query_displayed"], "go language")
	}
	if resp.Metadata["pages_count"] != 8 {
		t.Errorf("pages_count = %v, want 8", resp.Metadata["pages_count"])
	}
	if resp.Metadata["current_page"] != 1 {
		t.Errorf("current_page = %v, want 1", resp.Metadata["current_page"])
	}
	if resp.Raw != nil {
		t.Error("Raw should be nil when opts.Raw is false")
	}
}

func TestScraperAPIProvider_Search_WithCountry(t *testing.T) {
	var capturedQuery url.Values

	srv := newScraperAPITestServer(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockScraperAPIResponse))
	})
	defer srv.Close()

	p := search.NewScraperAPIProvider("tok123")
	p.SetBaseURL(srv.URL)

	_, err := p.Search(context.Background(), "test query", search.Options{
		Country: "us",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := map[string]string{
		"api_key":      "tok123",
		"query":        "test query",
		"country_code": "us",
	}
	for key, want := range checks {
		if got := capturedQuery.Get(key); got != want {
			t.Errorf("param %q = %q, want %q", key, got, want)
		}
	}
}

func TestScraperAPIProvider_Search_Raw(t *testing.T) {
	srv := newScraperAPITestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockScraperAPIResponse))
	})
	defer srv.Close()

	p := search.NewScraperAPIProvider("tok")
	p.SetBaseURL(srv.URL)

	resp, err := p.Search(context.Background(), "raw test", search.Options{Raw: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Raw == nil {
		t.Fatal("Raw should be populated when opts.Raw is true")
	}
}

func TestScraperAPIProvider_Search_EmptyResults(t *testing.T) {
	srv := newScraperAPITestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`{"search_information":{"query_displayed":"nothing"},` +
				`"organic_results":[],"pagination":{"pages_count":0,"current_page":1}}`,
		))
	})
	defer srv.Close()

	p := search.NewScraperAPIProvider("tok")
	p.SetBaseURL(srv.URL)

	resp, err := p.Search(context.Background(), "nothing", search.Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Results) != 0 {
		t.Errorf("got %d results, want 0", len(resp.Results))
	}
}

func TestScraperAPIProvider_Search_APIError(t *testing.T) {
	srv := newScraperAPITestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"invalid token"}`))
	})
	defer srv.Close()

	p := search.NewScraperAPIProvider("bad-tok")
	p.SetBaseURL(srv.URL)

	_, err := p.Search(context.Background(), "fail", search.Options{})
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

func TestScraperAPIProvider_Search_EmptyToken(t *testing.T) {
	p := search.NewScraperAPIProvider("")

	_, err := p.Search(context.Background(), "test", search.Options{})
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestScraperAPIProvider_Search_InvalidJSON(t *testing.T) {
	srv := newScraperAPITestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not json`))
	})
	defer srv.Close()

	p := search.NewScraperAPIProvider("tok")
	p.SetBaseURL(srv.URL)

	_, err := p.Search(context.Background(), "bad json", search.Options{})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
