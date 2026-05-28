package main

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

// buildClient creates a scrapedo.Client from config/env and attaches the cache.
func buildClient() (*scrapedo.Client, error) {
	token := cfg.Global.Token
	if token == "" {
		token = os.Getenv("SCRAPEDO_TOKEN")
	}

	if token == "" {
		return nil, errMissingToken
	}

	client, err := scrapedo.NewClient(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	if cacheStore != nil {
		client.SetCache(cacheStore)
	}

	return client, nil
}

// extractHost returns the hostname from a URL string.
func extractHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	return u.Host
}

// sanitizePath converts a URL path to a safe filename.
var unsafePathChars = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

func sanitizePath(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "page"
	}

	p := strings.Trim(u.Path, "/")
	if p == "" {
		return "index"
	}

	return unsafePathChars.ReplaceAllString(p, "_")
}
