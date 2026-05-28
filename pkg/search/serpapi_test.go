package search_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mr-pmillz/scrapedoctl/pkg/search"
)

const (
	serpAPIMockResponse = `{
	"search_information": {"total_results": "About 5000"},
	"organic_results": [
		{
			"position": 1,
			"title": "Test",
			"link": "https://example.com",
			"snippet": "snippet text",
			"displayed_link": "example.com"
		}
	]
}`
	serpAPITestToken  = "tok123"
	serpAPITestEngine = "google"
)

func newSerpAPITestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	return ts
}

func newSerpAPIDefaultServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newSerpAPITestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(serpAPIMockResponse))
	})
}

func TestSerpAPIProvider_Name(t *testing.T) {
	p := search.NewSerpAPIProvider("test-token")
	if got := p.Name(); got != "serpapi" {
		t.Errorf("Name() = %q, want %q", got, "serpapi")
	}
}

func TestSerpAPIProvider_Engines(t *testing.T) {
	p := search.NewSerpAPIProvider("test-token")
	engines := p.Engines()

	want := []string{
		"google", "bing", "yandex",
		"duckduckgo", "baidu", "yahoo", "naver",
	}
	if len(engines) != len(want) {
		t.Fatalf("Engines() returned %d engines, want %d", len(engines), len(want))
	}

	for i, e := range want {
		if engines[i] != e {
			t.Errorf("Engines()[%d] = %q, want %q", i, engines[i], e)
		}
	}
}

func TestSerpAPIProvider_Search_Google(t *testing.T) {
	var gotQuery string

	ts := newSerpAPITestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("q")

		if r.URL.Query().Get("engine") != serpAPITestEngine {
			t.Errorf("engine = %q, want %q", r.URL.Query().Get("engine"), serpAPITestEngine)
		}

		if r.URL.Query().Get("api_key") != serpAPITestToken {
			t.Errorf("api_key = %q, want %q", r.URL.Query().Get("api_key"), serpAPITestToken)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(serpAPIMockResponse))
	})

	p := search.NewSerpAPIProvider(serpAPITestToken)
	p.SetBaseURL(ts.URL)

	resp, err := p.Search(context.Background(), "golang testing", search.Options{Engine: serpAPITestEngine})
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if gotQuery != "golang testing" {
		t.Errorf("query param q = %q, want %q", gotQuery, "golang testing")
	}

	if len(resp.Results) != 1 {
		t.Fatalf("got %d results, want 1", len(resp.Results))
	}

	if resp.Results[0].Title != "Test" {
		t.Errorf("result title = %q, want %q", resp.Results[0].Title, "Test")
	}

	if resp.Results[0].URL != "https://example.com" {
		t.Errorf("result URL = %q, want %q", resp.Results[0].URL, "https://example.com")
	}

	if resp.Engine != serpAPITestEngine {
		t.Errorf("Engine = %q, want %q", resp.Engine, serpAPITestEngine)
	}

	if resp.Provider != "serpapi" {
		t.Errorf("Provider = %q, want %q", resp.Provider, "serpapi")
	}
}

func TestSerpAPIProvider_Search_Yandex(t *testing.T) {
	var gotText string

	ts := newSerpAPITestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotText = r.URL.Query().Get("text")

		if r.URL.Query().Get("engine") != "yandex" {
			t.Errorf("engine = %q, want %q", r.URL.Query().Get("engine"), "yandex")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(serpAPIMockResponse))
	})

	p := search.NewSerpAPIProvider(serpAPITestToken)
	p.SetBaseURL(ts.URL)

	_, err := p.Search(context.Background(), "yandex search", search.Options{Engine: "yandex"})
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if gotText != "yandex search" {
		t.Errorf("query param text = %q, want %q", gotText, "yandex search")
	}
}

func TestSerpAPIProvider_Search_Yahoo(t *testing.T) {
	var gotP string

	ts := newSerpAPITestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotP = r.URL.Query().Get("p")

		if r.URL.Query().Get("engine") != "yahoo" {
			t.Errorf("engine = %q, want %q", r.URL.Query().Get("engine"), "yahoo")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(serpAPIMockResponse))
	})

	p := search.NewSerpAPIProvider(serpAPITestToken)
	p.SetBaseURL(ts.URL)

	_, err := p.Search(context.Background(), "yahoo search", search.Options{Engine: "yahoo"})
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if gotP != "yahoo search" {
		t.Errorf("query param p = %q, want %q", gotP, "yahoo search")
	}
}

func TestSerpAPIProvider_Search_Naver(t *testing.T) {
	var gotQuery string

	ts := newSerpAPITestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("query")

		if r.URL.Query().Get("engine") != "naver" {
			t.Errorf("engine = %q, want %q", r.URL.Query().Get("engine"), "naver")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(serpAPIMockResponse))
	})

	p := search.NewSerpAPIProvider(serpAPITestToken)
	p.SetBaseURL(ts.URL)

	_, err := p.Search(context.Background(), "naver search", search.Options{Engine: "naver"})
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if gotQuery != "naver search" {
		t.Errorf("query param query = %q, want %q", gotQuery, "naver search")
	}
}

func TestSerpAPIProvider_Search_Pagination(t *testing.T) {
	tests := []struct {
		engine  string
		page    int
		wantKey string
		wantVal string
	}{
		{"google", 2, "start", "10"},
		{"bing", 2, "first", "11"},
		{"yandex", 3, "p", "2"},
		{"duckduckgo", 2, "start", "30"},
		{"baidu", 3, "pn", "20"},
		{"yahoo", 2, "b", "11"},
		{"naver", 2, "start", "16"},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			ts := newSerpAPITestServer(t, func(w http.ResponseWriter, r *http.Request) {
				gotVal := r.URL.Query().Get(tt.wantKey)
				if gotVal != tt.wantVal {
					t.Errorf(
						"%s page %d: %s = %q, want %q",
						tt.engine, tt.page, tt.wantKey, gotVal, tt.wantVal,
					)
				}

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(serpAPIMockResponse))
			})

			p := search.NewSerpAPIProvider(serpAPITestToken)
			p.SetBaseURL(ts.URL)

			_, err := p.Search(context.Background(), "test", search.Options{
				Engine: tt.engine,
				Page:   tt.page,
			})
			if err != nil {
				t.Fatalf("Search() error: %v", err)
			}
		})
	}
}

func TestSerpAPIProvider_Search_WithLangCountry(t *testing.T) {
	ts := newSerpAPITestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if hl := r.URL.Query().Get("hl"); hl != "en" {
			t.Errorf("hl = %q, want %q", hl, "en")
		}

		if gl := r.URL.Query().Get("gl"); gl != "us" {
			t.Errorf("gl = %q, want %q", gl, "us")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(serpAPIMockResponse))
	})

	p := search.NewSerpAPIProvider(serpAPITestToken)
	p.SetBaseURL(ts.URL)

	_, err := p.Search(context.Background(), "test", search.Options{
		Engine:  serpAPITestEngine,
		Lang:    "en",
		Country: "us",
	})
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
}

func TestSerpAPIProvider_Search_Raw(t *testing.T) {
	ts := newSerpAPIDefaultServer(t)

	p := search.NewSerpAPIProvider(serpAPITestToken)
	p.SetBaseURL(ts.URL)

	resp, err := p.Search(context.Background(), "test", search.Options{
		Engine: serpAPITestEngine,
		Raw:    true,
	})
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if resp.Raw == nil {
		t.Fatal("Raw is nil, expected populated")
	}

	// Verify it's valid JSON by marshaling it back.
	if _, err := json.Marshal(resp.Raw); err != nil {
		t.Errorf("Raw is not valid JSON: %v", err)
	}
}

func TestSerpAPIProvider_Search_EmptyToken(t *testing.T) {
	p := search.NewSerpAPIProvider("")

	_, err := p.Search(context.Background(), "test", search.Options{Engine: serpAPITestEngine})
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}

func TestSerpAPIProvider_Search_APIError(t *testing.T) {
	ts := newSerpAPITestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error": "invalid api key"}`))
	})

	p := search.NewSerpAPIProvider("bad-token")
	p.SetBaseURL(ts.URL)

	_, err := p.Search(context.Background(), "test", search.Options{Engine: serpAPITestEngine})
	if err == nil {
		t.Fatal("expected error for non-200 status, got nil")
	}
}

func TestSerpAPIProvider_Search_InvalidJSON(t *testing.T) {
	ts := newSerpAPITestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not valid json`))
	})

	p := search.NewSerpAPIProvider(serpAPITestToken)
	p.SetBaseURL(ts.URL)

	_, err := p.Search(context.Background(), "test", search.Options{Engine: serpAPITestEngine})
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestQueryParam(t *testing.T) {
	tests := []struct {
		engine string
		want   string
	}{
		{"google", "q"},
		{"bing", "q"},
		{"yandex", "text"},
		{"duckduckgo", "q"},
		{"baidu", "q"},
		{"yahoo", "p"},
		{"naver", "query"},
		{"unknown", "q"},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			got := search.QueryParam(tt.engine)
			if got != tt.want {
				t.Errorf("QueryParam(%q) = %q, want %q", tt.engine, got, tt.want)
			}
		})
	}
}

func TestPaginationParam(t *testing.T) {
	tests := []struct {
		engine  string
		page    int
		wantKey string
		wantVal string
	}{
		{"google", 1, "start", "0"},
		{"google", 2, "start", "10"},
		{"google", 3, "start", "20"},
		{"bing", 1, "first", "1"},
		{"bing", 2, "first", "11"},
		{"bing", 3, "first", "21"},
		{"yandex", 1, "p", "0"},
		{"yandex", 2, "p", "1"},
		{"yandex", 3, "p", "2"},
		{"duckduckgo", 1, "start", "0"},
		{"duckduckgo", 2, "start", "30"},
		{"duckduckgo", 3, "start", "60"},
		{"baidu", 1, "pn", "0"},
		{"baidu", 2, "pn", "10"},
		{"baidu", 3, "pn", "20"},
		{"yahoo", 1, "b", "1"},
		{"yahoo", 2, "b", "11"},
		{"yahoo", 3, "b", "21"},
		{"naver", 1, "start", "1"},
		{"naver", 2, "start", "16"},
		{"naver", 3, "start", "31"},
	}

	for _, tt := range tests {
		t.Run(tt.engine+"_page"+tt.wantVal, func(t *testing.T) {
			key, val := search.PaginationParam(tt.engine, tt.page)
			if key != tt.wantKey || val != tt.wantVal {
				t.Errorf(
					"PaginationParam(%q, %d) = (%q, %q), want (%q, %q)",
					tt.engine, tt.page, key, val, tt.wantKey, tt.wantVal,
				)
			}
		})
	}
}
