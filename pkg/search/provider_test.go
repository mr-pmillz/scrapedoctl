package search_test

import (
	"encoding/json"
	"testing"

	"github.com/mr-pmillz/scrapedoctl/pkg/search"
)

func TestResponse_JSON(t *testing.T) {
	t.Parallel()

	orig := &search.Response{
		Query:    "golang testing",
		Engine:   "google",
		Provider: "scrapedo",
		Results: []search.Result{
			{
				Position:     1,
				Title:        "Testing in Go",
				URL:          "https://example.com/testing",
				Snippet:      "A guide to testing in Go.",
				DisplayedURL: "example.com/testing",
			},
			{
				Position: 2,
				Title:    "Go Test Coverage",
				URL:      "https://example.com/coverage",
				Snippet:  "How to measure test coverage.",
			},
		},
		Raw: map[string]any{"request_id": "abc123"},
		Metadata: map[string]any{
			"total_results": float64(42),
			"search_time":   "0.5s",
		},
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded search.Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Query != orig.Query {
		t.Errorf("Query = %q, want %q", decoded.Query, orig.Query)
	}
	if decoded.Engine != orig.Engine {
		t.Errorf("Engine = %q, want %q", decoded.Engine, orig.Engine)
	}
	if decoded.Provider != orig.Provider {
		t.Errorf("Provider = %q, want %q", decoded.Provider, orig.Provider)
	}
	if len(decoded.Results) != len(orig.Results) {
		t.Fatalf("Results len = %d, want %d", len(decoded.Results), len(orig.Results))
	}

	for i, got := range decoded.Results {
		want := orig.Results[i]
		if got.Position != want.Position {
			t.Errorf("Results[%d].Position = %d, want %d", i, got.Position, want.Position)
		}
		if got.Title != want.Title {
			t.Errorf("Results[%d].Title = %q, want %q", i, got.Title, want.Title)
		}
		if got.URL != want.URL {
			t.Errorf("Results[%d].URL = %q, want %q", i, got.URL, want.URL)
		}
		if got.Snippet != want.Snippet {
			t.Errorf("Results[%d].Snippet = %q, want %q", i, got.Snippet, want.Snippet)
		}
		if got.DisplayedURL != want.DisplayedURL {
			t.Errorf("Results[%d].DisplayedURL = %q, want %q", i, got.DisplayedURL, want.DisplayedURL)
		}
	}

	if decoded.Raw == nil {
		t.Error("Raw should not be nil after roundtrip")
	}
	if decoded.Metadata == nil {
		t.Error("Metadata should not be nil after roundtrip")
	}
	if decoded.Metadata["total_results"] != float64(42) {
		t.Errorf("Metadata[total_results] = %v, want 42", decoded.Metadata["total_results"])
	}
}

func TestResponse_JSONOmitEmpty(t *testing.T) {
	t.Parallel()

	resp := &search.Response{
		Query:    "test",
		Engine:   "google",
		Provider: "mock",
		Results:  []search.Result{},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	raw := string(data)
	if containsKey(raw, "raw") {
		t.Error("JSON should omit 'raw' when nil")
	}
	if containsKey(raw, "metadata") {
		t.Error("JSON should omit 'metadata' when nil")
	}

	// Also verify DisplayedURL is omitted when empty.
	result := search.Result{
		Position: 1,
		Title:    "Test",
		URL:      "https://example.com",
		Snippet:  "A test.",
	}
	resultData, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if containsKey(string(resultData), "displayed_url") {
		t.Error("JSON should omit 'displayed_url' when empty")
	}
}

func TestOptions_Defaults(t *testing.T) {
	t.Parallel()

	var opts search.Options

	if opts.Engine != "" {
		t.Errorf("Engine zero value = %q, want empty", opts.Engine)
	}
	if opts.Lang != "" {
		t.Errorf("Lang zero value = %q, want empty", opts.Lang)
	}
	if opts.Country != "" {
		t.Errorf("Country zero value = %q, want empty", opts.Country)
	}
	if opts.Limit != 0 {
		t.Errorf("Limit zero value = %d, want 0", opts.Limit)
	}
	if opts.Page != 0 {
		t.Errorf("Page zero value = %d, want 0", opts.Page)
	}
	if opts.Raw {
		t.Error("Raw zero value should be false")
	}
}

// containsKey checks if a JSON string contains a given key.
func containsKey(jsonStr, key string) bool {
	var m map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return false
	}
	_, ok := m[key]
	return ok
}
