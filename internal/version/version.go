// Package version provides the application version information
// and update checking via GitHub releases.
package version

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Build-time variables injected via -ldflags.
var (
	Version   = "0.2.0"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

const (
	// RepoOwner is the GitHub repository owner.
	RepoOwner = "mr-pmillz"
	// RepoName is the GitHub repository name.
	RepoName = "scrapedoctl"
	// RepoURL is the GitHub repository URL.
	RepoURL = "https://github.com/" + RepoOwner + "/" + RepoName
	// ReleasesURL is the GitHub releases page.
	ReleasesURL = RepoURL + "/releases"
)

// ErrGitHubAPI is returned when the GitHub API returns a non-200 status.
var ErrGitHubAPI = errors.New("github API error")

// apiBaseURL is the base URL for the GitHub API (overridable for testing).
var apiBaseURL = "https://api.github.com"

// SetAPIBaseURL sets the base URL for the GitHub API and returns the previous value.
func SetAPIBaseURL(url string) string {
	old := apiBaseURL
	apiBaseURL = url
	return old
}

// ghRelease is a minimal GitHub release response.
type ghRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// Info returns a formatted version string.
func Info() string {
	return fmt.Sprintf(
		"scrapedoctl %s (commit: %s, built: %s)",
		Version, GitCommit, BuildDate,
	)
}

// CheckLatest queries the GitHub API for the latest release and
// returns (latestTag, releaseURL, isNewer, err).
func CheckLatest(ctx context.Context) (string, string, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	url := fmt.Sprintf(
		"%s/repos/%s/%s/releases/latest",
		apiBaseURL, RepoOwner, RepoName,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", false, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", false, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", false, fmt.Errorf("%w: status %d", ErrGitHubAPI, resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", "", false, fmt.Errorf("decode response: %w", err)
	}

	latest := strings.TrimPrefix(rel.TagName, "v")
	current := strings.TrimPrefix(Version, "v")

	// Strip -dev / -rc suffixes for comparison.
	currentBase := strings.SplitN(current, "-", 2)[0]

	newer := latest != currentBase && latest > currentBase

	return rel.TagName, rel.HTMLURL, newer, nil
}
