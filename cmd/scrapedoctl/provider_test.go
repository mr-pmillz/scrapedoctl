package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/internal/config"
)

func setupTestConfig(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.toml"

	var err error
	cfg, err = config.Load(configPath, "")
	require.NotNil(t, cfg)
	// ErrConfigNotFound is expected for a fresh temp path.
	_ = err
}

func TestProviderList(t *testing.T) {
	setupTestConfig(t)

	cfg.Global.Token = "test-token"
	cfg.Providers = map[string]config.ProviderConfig{
		"serpapi": {
			Token:   "serp-key",
			Engines: []string{"google", "bing", "yandex", "duckduckgo", "baidu", "yahoo", "naver"},
		},
		"scraperapi": {
			Token:   "scraper-key",
			Engines: []string{"google"},
		},
	}

	output := captureStdout(t, func() {
		cmd := newProviderListCmd()
		err := cmd.RunE(cmd, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "scrapedo")
	assert.Contains(t, output, "serpapi")
	assert.Contains(t, output, "scraperapi")
	assert.Contains(t, output, "active (global token)")
	assert.Contains(t, output, "active")
}

func TestProviderList_JSON(t *testing.T) {
	setupTestConfig(t)

	cfg.Global.Token = "test-token"
	cfg.Providers = map[string]config.ProviderConfig{
		"serpapi": {
			Token:   "serp-key",
			Engines: []string{"google"},
		},
	}

	output := captureStdout(t, func() {
		cmd := newProviderListCmd()
		require.NoError(t, cmd.Flags().Set("json", "true"))
		err := cmd.RunE(cmd, nil)
		require.NoError(t, err)
	})

	var rows []providerRow
	require.NoError(t, json.Unmarshal([]byte(output), &rows))
	require.Len(t, rows, 2) //nolint:mnd // scrapedo + serpapi
	assert.Equal(t, "scrapedo", rows[0].Name)
	assert.Equal(t, "serpapi", rows[1].Name)
}

func TestProviderAdd(t *testing.T) {
	setupTestConfig(t)

	err := runProviderAdd("serpapi", "my-serp-key")
	require.NoError(t, err)

	p, ok := cfg.Providers["serpapi"]
	require.True(t, ok)
	assert.Equal(t, "my-serp-key", p.Token)
	assert.Contains(t, p.Engines, "google")
	assert.Contains(t, p.Engines, "bing")
}

func TestProviderAdd_Unknown(t *testing.T) {
	setupTestConfig(t)

	err := runProviderAdd("notreal", "key")
	require.Error(t, err)
	assert.ErrorIs(t, err, errUnknownProvider)
}

func TestProviderRemove(t *testing.T) {
	setupTestConfig(t)

	cfg.Providers = map[string]config.ProviderConfig{
		"serpapi": {Token: "key", Engines: []string{"google"}},
	}

	err := runProviderRemove("serpapi")
	require.NoError(t, err)

	_, ok := cfg.Providers["serpapi"]
	assert.False(t, ok)
}

func TestProviderRemove_NotFound(t *testing.T) {
	setupTestConfig(t)

	err := runProviderRemove("nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, errProviderNotConfigured)
}
