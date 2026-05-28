package search_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/mr-pmillz/scrapedoctl/pkg/search"
)

const scrapedoEngineGoogle = "google"

const mockScrapedoResponse = `{
  "search_information": {"total_results": 12400, "time_taken_displayed": "0.42"},
  "organic_results": [
    {"position": 1, "title": "Test Result", "link": "https://example.com", "snippet": "A snippet", "displayed_link": "example.com"}
  ]
}`

func newScrapedoTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func TestScrapedoProvider_Name(t *testing.T) {
	p := search.NewScrapedoProvider("tok")
	if got := p.Name(); got != "scrapedo" {
		t.Fatalf("Name() = %q, want %q", got, "scrapedo")
	}
}

func TestScrapedoProvider_Engines(t *testing.T) {
	p := search.NewScrapedoProvider("tok")
	engines := p.Engines()
	if len(engines) != 1 || engines[0] != scrapedoEngineGoogle {
		t.Fatalf("Engines() = %v, want [%s]", engines, scrapedoEngineGoogle)
	}
}

func TestScrapedoProvider_Search_Success(t *testing.T) {
	srv := newScrapedoTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockScrapedoResponse))
	})
	defer srv.Close()

	p := search.NewScrapedoProvider("test-token")
	p.SetBaseURL(srv.URL)

	resp, err := p.Search(context.Background(), "golang", search.Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Query != "golang" {
		t.Errorf("Query = %q, want %q", resp.Query, "golang")
	}
	if resp.Engine != scrapedoEngineGoogle {
		t.Errorf("Engine = %q, want %q", resp.Engine, scrapedoEngineGoogle)
	}
	if resp.Provider != "scrapedo" {
		t.Errorf("Provider = %q, want %q", resp.Provider, "scrapedo")
	}
	if len(resp.Results) != 1 {
		t.Fatalf("got %d results, want 1", len(resp.Results))
	}

	r := resp.Results[0]
	if r.Position != 1 {
		t.Errorf("Position = %d, want 1", r.Position)
	}
	if r.Title != "Test Result" {
		t.Errorf("Title = %q, want %q", r.Title, "Test Result")
	}
	if r.URL != "https://example.com" {
		t.Errorf("URL = %q, want %q", r.URL, "https://example.com")
	}
	if r.Snippet != "A snippet" {
		t.Errorf("Snippet = %q, want %q", r.Snippet, "A snippet")
	}
	if r.DisplayedURL != "example.com" {
		t.Errorf("DisplayedURL = %q, want %q", r.DisplayedURL, "example.com")
	}

	if resp.Metadata["total_results"] != float64(12400) {
		t.Errorf("total_results = %v, want 12400", resp.Metadata["total_results"])
	}
	if resp.Metadata["time_taken"] != "0.42" {
		t.Errorf("time_taken = %v, want %q", resp.Metadata["time_taken"], "0.42")
	}
	if resp.Raw != nil {
		t.Error("Raw should be nil when opts.Raw is false")
	}
}

func TestScrapedoProvider_Search_WithOptions(t *testing.T) {
	var capturedQuery url.Values

	srv := newScrapedoTestServer(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockScrapedoResponse))
	})
	defer srv.Close()

	p := search.NewScrapedoProvider("tok123")
	p.SetBaseURL(srv.URL)

	_, err := p.Search(context.Background(), "test query", search.Options{
		Lang:    "en",
		Country: "us",
		Page:    3,
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := map[string]string{
		"token": "tok123",
		"q":     "test query",
		"hl":    "en",
		"gl":    "us",
		"start": "20",
		"num":   "5",
	}
	for key, want := range checks {
		if got := capturedQuery.Get(key); got != want {
			t.Errorf("param %q = %q, want %q", key, got, want)
		}
	}
}

func TestScrapedoProvider_Search_Raw(t *testing.T) {
	srv := newScrapedoTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockScrapedoResponse))
	})
	defer srv.Close()

	p := search.NewScrapedoProvider("tok")
	p.SetBaseURL(srv.URL)

	resp, err := p.Search(context.Background(), "raw test", search.Options{Raw: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Raw == nil {
		t.Fatal("Raw should be populated when opts.Raw is true")
	}
}

func TestScrapedoProvider_Search_EmptyResults(t *testing.T) {
	srv := newScrapedoTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`{"search_information":{"total_results":0,` +
				`"time_taken_displayed":"0.01"},"organic_results":[]}`,
		))
	})
	defer srv.Close()

	p := search.NewScrapedoProvider("tok")
	p.SetBaseURL(srv.URL)

	resp, err := p.Search(context.Background(), "nothing", search.Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Results) != 0 {
		t.Errorf("got %d results, want 0", len(resp.Results))
	}
}

func TestScrapedoProvider_Search_APIError(t *testing.T) {
	srv := newScrapedoTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"invalid token"}`))
	})
	defer srv.Close()

	p := search.NewScrapedoProvider("bad-tok")
	p.SetBaseURL(srv.URL)

	_, err := p.Search(context.Background(), "fail", search.Options{})
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

func TestScrapedoProvider_Search_EmptyToken(t *testing.T) {
	p := search.NewScrapedoProvider("")

	_, err := p.Search(context.Background(), "test", search.Options{})
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestScrapedoProvider_Search_InvalidJSON(t *testing.T) {
	srv := newScrapedoTestServer(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not json`))
	})
	defer srv.Close()

	p := search.NewScrapedoProvider("tok")
	p.SetBaseURL(srv.URL)

	_, err := p.Search(context.Background(), "bad json", search.Options{})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
