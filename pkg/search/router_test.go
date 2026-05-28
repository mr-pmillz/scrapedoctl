package search_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mr-pmillz/scrapedoctl/pkg/search"
)

const (
	providerAlpha = "alpha"
	providerBeta  = "beta"
	providerGamma = "gamma"
)

type mockProvider struct {
	name    string
	engines []string
	results []search.Result
	err     error
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Engines() []string {
	return m.engines
}

func (m *mockProvider) Search(_ context.Context, query string, opts search.Options) (*search.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &search.Response{
		Query:    query,
		Engine:   opts.Engine,
		Provider: m.name,
		Results:  m.results,
	}, nil
}

func TestRouter_Register(t *testing.T) {
	t.Parallel()

	r := search.NewRouter()
	if len(r.Providers()) != 0 {
		t.Fatalf("new router should have 0 providers, got %d", len(r.Providers()))
	}

	p1 := &mockProvider{name: providerAlpha, engines: []string{"google"}}
	p2 := &mockProvider{name: providerBeta, engines: []string{"bing"}}

	r.Register(p1)
	r.Register(p2)

	if got := len(r.Providers()); got != 2 {
		t.Fatalf("Providers() len = %d, want 2", got)
	}
	if r.Providers()[0].Name() != providerAlpha {
		t.Errorf("Providers()[0].Name() = %q, want %q", r.Providers()[0].Name(), providerAlpha)
	}
	if r.Providers()[1].Name() != providerBeta {
		t.Errorf("Providers()[1].Name() = %q, want %q", r.Providers()[1].Name(), providerBeta)
	}
}

func TestRouter_Resolve_ByEngine(t *testing.T) {
	t.Parallel()

	r := search.NewRouter()
	r.Register(&mockProvider{name: providerAlpha, engines: []string{"google", "bing"}})
	r.Register(&mockProvider{name: providerBeta, engines: []string{"duckduckgo"}})

	p, err := r.Resolve("duckduckgo", "")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if p.Name() != providerBeta {
		t.Errorf("resolved provider = %q, want %q", p.Name(), providerBeta)
	}

	p, err = r.Resolve("google", "")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if p.Name() != providerAlpha {
		t.Errorf("resolved provider = %q, want %q", p.Name(), providerAlpha)
	}
}

func TestRouter_Resolve_ExplicitProvider(t *testing.T) {
	t.Parallel()

	r := search.NewRouter()
	r.Register(&mockProvider{name: providerAlpha, engines: []string{"google"}})
	r.Register(&mockProvider{name: providerBeta, engines: []string{"google", "bing"}})

	// Both support "google", but we force "beta".
	p, err := r.Resolve("google", providerBeta)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if p.Name() != providerBeta {
		t.Errorf("resolved provider = %q, want %q", p.Name(), providerBeta)
	}
}

func TestRouter_Resolve_NoMatch(t *testing.T) {
	t.Parallel()

	r := search.NewRouter()
	r.Register(&mockProvider{name: providerAlpha, engines: []string{"google"}})

	_, err := r.Resolve("yandex", "")
	if err == nil {
		t.Fatal("expected error for unmatched engine")
	}
	if !errors.Is(err, search.ErrNoProvider) {
		t.Errorf("error should wrap ErrNoProvider, got: %v", err)
	}
	if !strings.Contains(err.Error(), "yandex") {
		t.Errorf("error should mention engine, got: %v", err)
	}
	if !strings.Contains(err.Error(), "google") {
		t.Errorf("error should list available engines, got: %v", err)
	}
}

func TestRouter_Resolve_ProviderNotSupportEngine(t *testing.T) {
	t.Parallel()

	r := search.NewRouter()
	r.Register(&mockProvider{name: providerAlpha, engines: []string{"google"}})
	r.Register(&mockProvider{name: providerBeta, engines: []string{"bing"}})

	_, err := r.Resolve("bing", providerAlpha)
	if err == nil {
		t.Fatal("expected error when named provider doesn't support engine")
	}
	if !errors.Is(err, search.ErrProviderNotSupported) {
		t.Errorf("error should wrap ErrProviderNotSupported, got: %v", err)
	}
	if !strings.Contains(err.Error(), providerAlpha) {
		t.Errorf("error should mention provider name, got: %v", err)
	}
	if !strings.Contains(err.Error(), "bing") {
		t.Errorf("error should mention engine, got: %v", err)
	}
}

func TestRouter_AllEngines(t *testing.T) {
	t.Parallel()

	r := search.NewRouter()
	r.Register(&mockProvider{name: providerAlpha, engines: []string{"google", "bing"}})
	r.Register(&mockProvider{name: providerBeta, engines: []string{"bing", "duckduckgo"}})

	engines := r.AllEngines()
	if len(engines) != 3 {
		t.Fatalf("AllEngines() len = %d, want 3; got %v", len(engines), engines)
	}

	want := map[string]bool{"google": true, "bing": true, "duckduckgo": true}
	for _, e := range engines {
		if !want[e] {
			t.Errorf("unexpected engine %q", e)
		}
	}
}

func TestRouter_AllEngines_Empty(t *testing.T) {
	t.Parallel()

	r := search.NewRouter()
	engines := r.AllEngines()
	if engines != nil {
		t.Errorf("AllEngines() on empty router = %v, want nil", engines)
	}
}

func TestRouter_ProviderNames(t *testing.T) {
	t.Parallel()

	r := search.NewRouter()
	r.Register(&mockProvider{name: providerAlpha, engines: []string{"google"}})
	r.Register(&mockProvider{name: providerBeta, engines: []string{"bing"}})
	r.Register(&mockProvider{name: providerGamma, engines: []string{"duckduckgo"}})

	names := r.ProviderNames()
	if len(names) != 3 {
		t.Fatalf("ProviderNames() len = %d, want 3", len(names))
	}
	expected := []string{providerAlpha, providerBeta, providerGamma}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("ProviderNames()[%d] = %q, want %q", i, names[i], want)
		}
	}
}

func TestRouter_Search(t *testing.T) {
	t.Parallel()

	results := []search.Result{
		{Position: 1, Title: "Result 1", URL: "https://example.com/1", Snippet: "First result"},
	}
	r := search.NewRouter()
	r.Register(&mockProvider{name: providerAlpha, engines: []string{"google"}, results: results})

	resp, err := r.Search(context.Background(), "test query", search.Options{Engine: "google"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if resp.Query != "test query" {
		t.Errorf("Query = %q, want %q", resp.Query, "test query")
	}
	if resp.Engine != "google" {
		t.Errorf("Engine = %q, want %q", resp.Engine, "google")
	}
	if resp.Provider != providerAlpha {
		t.Errorf("Provider = %q, want %q", resp.Provider, providerAlpha)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("Results len = %d, want 1", len(resp.Results))
	}
	if resp.Results[0].Title != "Result 1" {
		t.Errorf("Results[0].Title = %q, want %q", resp.Results[0].Title, "Result 1")
	}
}

func TestRouter_Search_Error(t *testing.T) {
	t.Parallel()

	providerErr := errors.New("API rate limit exceeded")
	r := search.NewRouter()
	r.Register(&mockProvider{name: providerAlpha, engines: []string{"google"}, err: providerErr})

	_, err := r.Search(context.Background(), "test", search.Options{Engine: "google"})
	if err == nil {
		t.Fatal("expected error from provider")
	}
	if !errors.Is(err, providerErr) {
		t.Errorf("error = %v, want wrapping %v", err, providerErr)
	}
}

func TestRouter_Search_NoProvider(t *testing.T) {
	t.Parallel()

	r := search.NewRouter()
	_, err := r.Search(context.Background(), "test", search.Options{Engine: "nonexistent"})
	if err == nil {
		t.Fatal("expected error when no provider matches")
	}
	if !errors.Is(err, search.ErrNoProvider) {
		t.Errorf("error should wrap ErrNoProvider, got: %v", err)
	}
}
