package scrapedo_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

func TestExtractLinks_HTMLHref(t *testing.T) {
	t.Parallel()

	content := `<a href="/about">About</a><a href="/docs">Docs</a>`
	links := scrapedo.ExtractLinks(content, "https://example.com")

	require.Len(t, links, 2)
	assert.Equal(t, "https://example.com/about", links[0])
	assert.Equal(t, "https://example.com/docs", links[1])
}

func TestExtractLinks_MarkdownLinks(t *testing.T) {
	t.Parallel()

	content := `[About](/about) and [Docs](https://example.com/docs)`
	links := scrapedo.ExtractLinks(content, "https://example.com")

	require.Len(t, links, 2)
	assert.Equal(t, "https://example.com/about", links[0])
	assert.Equal(t, "https://example.com/docs", links[1])
}

func TestExtractLinks_RelativeURLResolution(t *testing.T) {
	t.Parallel()

	content := `<a href="../sibling">Sibling</a>`
	links := scrapedo.ExtractLinks(content, "https://example.com/docs/page")

	require.Len(t, links, 1)
	assert.Equal(t, "https://example.com/sibling", links[0])
}

func TestExtractLinks_FragmentStripping(t *testing.T) {
	t.Parallel()

	content := `<a href="/about#section1">About</a><a href="/about#section2">About Again</a>`
	links := scrapedo.ExtractLinks(content, "https://example.com")

	require.Len(t, links, 1)
	assert.Equal(t, "https://example.com/about", links[0])
}

func TestExtractLinks_CrossDomainFiltering(t *testing.T) {
	t.Parallel()

	content := `<a href="https://other.com/page">Other</a><a href="/local">Local</a>`
	links := scrapedo.ExtractLinks(content, "https://example.com")

	require.Len(t, links, 1)
	assert.Equal(t, "https://example.com/local", links[0])
}

func TestExtractLinks_Deduplication(t *testing.T) {
	t.Parallel()

	content := `<a href="/about">A</a><a href="/about">B</a>[About](/about)`
	links := scrapedo.ExtractLinks(content, "https://example.com")

	require.Len(t, links, 1)
	assert.Equal(t, "https://example.com/about", links[0])
}

func TestExtractLinks_SkipsJavascriptAndMailto(t *testing.T) {
	t.Parallel()

	content := `<a href="javascript:void(0)">JS</a><a href="mailto:a@b.com">Mail</a><a href="/ok">OK</a>`
	links := scrapedo.ExtractLinks(content, "https://example.com")

	require.Len(t, links, 1)
	assert.Equal(t, "https://example.com/ok", links[0])
}

func TestExtractLinks_InvalidBaseURL(t *testing.T) {
	t.Parallel()

	links := scrapedo.ExtractLinks(`<a href="/x">X</a>`, "://bad")
	assert.Empty(t, links)
}

func TestExtractLinks_EmptyContent(t *testing.T) {
	t.Parallel()

	links := scrapedo.ExtractLinks("", "https://example.com")
	assert.Empty(t, links)
}

func TestExtractLinks_SortedOutput(t *testing.T) {
	t.Parallel()

	content := `<a href="/z">Z</a><a href="/a">A</a><a href="/m">M</a>`
	links := scrapedo.ExtractLinks(content, "https://example.com")

	require.Len(t, links, 3)
	assert.Equal(t, "https://example.com/a", links[0])
	assert.Equal(t, "https://example.com/m", links[1])
	assert.Equal(t, "https://example.com/z", links[2])
}
