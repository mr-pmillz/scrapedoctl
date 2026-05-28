package version_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/internal/version"
)

func TestInfo(t *testing.T) {
	t.Parallel()
	info := version.Info()
	assert.Contains(t, info, "scrapedoctl")
	assert.Contains(t, info, version.Version)
	assert.Contains(t, info, version.GitCommit)
	assert.Contains(t, info, version.BuildDate)
}

func TestCheckLatest_NewerVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"tag_name": "v99.0.0",
			"html_url": "https://github.com/mr-pmillz/scrapedoctl/releases/tag/v99.0.0",
		})
	}))
	defer srv.Close()

	old := version.SetAPIBaseURL(srv.URL)
	defer version.SetAPIBaseURL(old)

	tag, url, newer, err := version.CheckLatest(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "v99.0.0", tag)
	assert.Contains(t, url, "v99.0.0")
	assert.True(t, newer)
}

func TestCheckLatest_SameVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"tag_name": "v" + version.Version,
			"html_url": "https://github.com/mr-pmillz/scrapedoctl/releases",
		})
	}))
	defer srv.Close()

	old := version.SetAPIBaseURL(srv.URL)
	defer version.SetAPIBaseURL(old)

	_, _, newer, err := version.CheckLatest(context.Background())
	require.NoError(t, err)
	assert.False(t, newer)
}

func TestCheckLatest_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	old := version.SetAPIBaseURL(srv.URL)
	defer version.SetAPIBaseURL(old)

	_, _, _, err := version.CheckLatest(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, version.ErrGitHubAPI)
	assert.Contains(t, err.Error(), "500")
}

func TestCheckLatest_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not valid json at all`))
	}))
	defer srv.Close()

	old := version.SetAPIBaseURL(srv.URL)
	defer version.SetAPIBaseURL(old)

	_, _, _, err := version.CheckLatest(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

func TestConstants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "mr-pmillz", version.RepoOwner)
	assert.Equal(t, "scrapedoctl", version.RepoName)
	assert.Contains(t, version.RepoURL, "github.com/mr-pmillz/scrapedoctl")
	assert.Contains(t, version.ReleasesURL, "/releases")
}

func TestSetAPIBaseURL_ReturnsOld(t *testing.T) {
	original := version.SetAPIBaseURL("http://test.local")
	restored := version.SetAPIBaseURL(original)
	assert.Equal(t, "http://test.local", restored)
}
