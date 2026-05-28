package search_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mr-pmillz/scrapedoctl/pkg/search"
)

const noResults = "No results found.\n"

func sampleResponse() *search.Response {
	return &search.Response{
		Query:    "golang mcp",
		Engine:   "google",
		Provider: "scrapedo",
		Results: []search.Result{
			{
				Position: 1,
				Title:    "Go SDK for MCP",
				URL:      "https://github.com/modelcontextprotocol/go-sdk",
				Snippet:  "The official Go SDK",
			},
			{
				Position: 2,
				Title:    "MCP Documentation",
				URL:      "https://modelcontextprotocol.io/docs",
				Snippet:  "Learn about MCP",
			},
			{
				Position: 3,
				Title:    "Tutorial: Build MCP in Go",
				URL:      "https://dev.to/article/mcp-go",
				Snippet:  "Step by step guide",
			},
		},
	}
}

func emptyResponse() *search.Response {
	return &search.Response{
		Query:    "nothing",
		Engine:   "google",
		Provider: "scrapedo",
	}
}

func TestFormatTable_Success(t *testing.T) {
	var buf bytes.Buffer
	resp := sampleResponse()

	if err := search.FormatTable(&buf, resp); err != nil {
		t.Fatalf("FormatTable returned error: %v", err)
	}

	out := buf.String()

	// Verify header line.
	if !strings.Contains(out, "#") ||
		!strings.Contains(out, "Title") ||
		!strings.Contains(out, "URL") {
		t.Errorf("missing header columns in output:\n%s", out)
	}

	// Verify all three results appear.
	for _, r := range resp.Results {
		if !strings.Contains(out, r.Title) {
			t.Errorf("missing title %q in output:\n%s", r.Title, out)
		}
	}

	// Verify hostnames are shown instead of full URLs.
	for _, host := range []string{
		"github.com",
		"modelcontextprotocol.io",
		"dev.to",
	} {
		if !strings.Contains(out, host) {
			t.Errorf("expected hostname %s in output:\n%s", host, out)
		}
	}
}

func TestFormatTable_LongTitle(t *testing.T) {
	var buf bytes.Buffer
	longTitle := "This is a very long title that should be " +
		"truncated because it exceeds forty-five characters"
	resp := &search.Response{
		Query:    "test",
		Engine:   "google",
		Provider: "scrapedo",
		Results: []search.Result{
			{
				Position: 1,
				Title:    longTitle,
				URL:      "https://example.com/page",
				Snippet:  "A snippet",
			},
		},
	}

	if err := search.FormatTable(&buf, resp); err != nil {
		t.Fatalf("FormatTable returned error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "...") {
		t.Errorf("expected truncated title with '...' in output:\n%s", out)
	}

	// The full original title must NOT appear.
	if strings.Contains(out, resp.Results[0].Title) {
		t.Errorf("full title should have been truncated:\n%s", out)
	}
}

func TestFormatTable_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := search.FormatTable(&buf, emptyResponse()); err != nil {
		t.Fatalf("FormatTable returned error: %v", err)
	}

	if got := buf.String(); got != noResults {
		t.Errorf("expected %q, got %q", noResults, got)
	}
}

func TestFormatJSON_Success(t *testing.T) {
	var buf bytes.Buffer
	resp := sampleResponse()

	if err := search.FormatJSON(&buf, resp); err != nil {
		t.Fatalf("FormatJSON returned error: %v", err)
	}

	// Verify output is valid JSON that round-trips.
	var decoded search.Response
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf(
			"output is not valid JSON: %v\n%s",
			err, buf.String(),
		)
	}

	if decoded.Query != resp.Query {
		t.Errorf(
			"query mismatch: got %q, want %q",
			decoded.Query, resp.Query,
		)
	}

	if len(decoded.Results) != len(resp.Results) {
		t.Errorf(
			"results count mismatch: got %d, want %d",
			len(decoded.Results), len(resp.Results),
		)
	}
}

func TestFormatJSON_WithRaw(t *testing.T) {
	var buf bytes.Buffer
	resp := sampleResponse()
	resp.Raw = map[string]any{
		"organic_count": 3,
		"debug":         true,
	}

	if err := search.FormatJSON(&buf, resp); err != nil {
		t.Fatalf("FormatJSON returned error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, `"raw"`) {
		t.Errorf("expected raw field in JSON output:\n%s", out)
	}

	if !strings.Contains(out, `"organic_count"`) {
		t.Errorf("expected organic_count in raw field:\n%s", out)
	}
}

func TestFormatJSON_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := search.FormatJSON(&buf, emptyResponse()); err != nil {
		t.Fatalf("FormatJSON returned error: %v", err)
	}

	if got := buf.String(); got != noResults {
		t.Errorf("expected %q, got %q", noResults, got)
	}
}

func TestFormatMarkdown_Success(t *testing.T) {
	var buf bytes.Buffer
	resp := sampleResponse()

	if err := search.FormatMarkdown(&buf, resp); err != nil {
		t.Fatalf("FormatMarkdown returned error: %v", err)
	}

	out := buf.String()

	// Verify header.
	wantHeader := `## Search: "golang mcp" (google via scrapedo)`
	if !strings.Contains(out, wantHeader) {
		t.Errorf("missing or incorrect header in output:\n%s", out)
	}

	// Verify numbered list with bold linked titles.
	wantFirst := "1. **[Go SDK for MCP]" +
		"(https://github.com/modelcontextprotocol/go-sdk)**"
	if !strings.Contains(out, wantFirst) {
		t.Errorf("missing first result link in output:\n%s", out)
	}

	wantSecond := "2. **[MCP Documentation]" +
		"(https://modelcontextprotocol.io/docs)**"
	if !strings.Contains(out, wantSecond) {
		t.Errorf("missing second result link in output:\n%s", out)
	}

	// Verify snippets.
	if !strings.Contains(out, "The official Go SDK") {
		t.Errorf("missing snippet in output:\n%s", out)
	}
}

func TestFormatMarkdown_Empty(t *testing.T) {
	var buf bytes.Buffer
	err := search.FormatMarkdown(&buf, emptyResponse())
	if err != nil {
		t.Fatalf("FormatMarkdown returned error: %v", err)
	}

	if got := buf.String(); got != noResults {
		t.Errorf("expected %q, got %q", noResults, got)
	}
}

func TestFormatMarkdown_SpecialChars(t *testing.T) {
	var buf bytes.Buffer
	resp := &search.Response{
		Query:    "test",
		Engine:   "google",
		Provider: "scrapedo",
		Results: []search.Result{
			{
				Position: 1,
				Title:    "Go *templates* & [links]",
				URL:      "https://example.com",
				Snippet:  "Use `backticks` and _underscores_",
			},
		},
	}

	if err := search.FormatMarkdown(&buf, resp); err != nil {
		t.Fatalf("FormatMarkdown returned error: %v", err)
	}

	out := buf.String()

	// Asterisks in title should be escaped.
	if strings.Contains(out, "*templates*") {
		t.Errorf("markdown special chars should be escaped:\n%s", out)
	}

	if !strings.Contains(out, `\*templates\*`) {
		t.Errorf("expected escaped asterisks in output:\n%s", out)
	}

	// Brackets should be escaped.
	if !strings.Contains(out, `\[links\]`) {
		t.Errorf("expected escaped brackets in output:\n%s", out)
	}

	// Backticks in snippet should be escaped.
	if !strings.Contains(out, "\\`backticks\\`") {
		t.Errorf("expected escaped backticks in snippet:\n%s", out)
	}

	// Underscores in snippet should be escaped.
	if !strings.Contains(out, `\_underscores\_`) {
		t.Errorf("expected escaped underscores in snippet:\n%s", out)
	}
}
