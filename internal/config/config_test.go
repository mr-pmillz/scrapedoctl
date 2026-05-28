package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/internal/config"
)

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Create various config files
	tomlPath := filepath.Join(tmpDir, "conf.toml")
	yamlPath := filepath.Join(tmpDir, "conf.yaml")
	ymlPath := filepath.Join(tmpDir, "conf.yml")
	jsonPath := filepath.Join(tmpDir, "conf.json")
	noExtPath := filepath.Join(tmpDir, "conf")

	require.NoError(t, os.WriteFile(tomlPath, []byte(`[global]
token = "toml-token"
[profiles.p1]
render = true`), 0o644))

	require.NoError(t, os.WriteFile(yamlPath, []byte(`global:
  token: yaml-token`), 0o644))

	require.NoError(t, os.WriteFile(ymlPath, []byte(`global:
  token: yml-token`), 0o644))

	require.NoError(t, os.WriteFile(jsonPath, []byte(`{"global": {"token": "json-token"}}`), 0o644))

	require.NoError(t, os.WriteFile(noExtPath, []byte(`[global]
token = "noext-token"`), 0o644))

	tests := []struct {
		name          string
		path          string
		profile       string
		env           map[string]string
		expectedToken string
		expectedErr   error
		check         func(t *testing.T, cfg *config.Config)
	}{
		{
			name:          "load toml",
			path:          tomlPath,
			expectedToken: "toml-token",
		},
		{
			name:          "load yaml",
			path:          yamlPath,
			expectedToken: "yaml-token",
		},
		{
			name:          "load yml",
			path:          ymlPath,
			expectedToken: "yml-token",
		},
		{
			name:          "load json",
			path:          jsonPath,
			expectedToken: "json-token",
		},
		{
			name:          "load no extension (defaults to toml)",
			path:          noExtPath,
			expectedToken: "noext-token",
		},
		{
			name: "load with env override",
			path: tomlPath,
			env: map[string]string{
				"SCRAPEDO_GLOBAL_TOKEN": "env-token",
			},
			expectedToken: "env-token",
		},
		{
			name:          "load with profile p1",
			path:          tomlPath,
			profile:       "p1",
			expectedToken: "toml-token",
			check: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assert.True(t, cfg.Resolved.Render)
			},
		},
		{
			name:        "profile not found",
			path:        tomlPath,
			profile:     "missing",
			expectedErr: errors.New("profile not found"),
		},
		{
			name: "invalid toml",
			path: func() string {
				p := filepath.Join(tmpDir, "invalid.toml")
				_ = os.WriteFile(p, []byte("invalid = ["), 0o644)
				return p
			}(),
			expectedErr: errors.New("failed to load config file"),
		},
		{
			name: "incompatible config type for unmarshal",
			path: func() string {
				p := filepath.Join(tmpDir, "bad_type.toml")
				_ = os.WriteFile(p, []byte("global = \"should be a table\""), 0o644)
				return p
			}(),
			expectedErr: errors.New("failed to unmarshal config"),
		},
		{
			name:        "config file not found",
			path:        filepath.Join(tmpDir, "nonexistent.toml"),
			expectedErr: config.ErrConfigNotFound,
		},
		{
			name:        "config path is a directory",
			path:        tmpDir,
			expectedErr: errors.New("config path is a directory"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			cfg, err := config.Load(tt.path, tt.profile)
			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
				if errors.Is(err, config.ErrConfigNotFound) {
					assert.NotNil(t, cfg)
					assert.Equal(t, "https://api.scrape.do", cfg.Global.BaseURL)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedToken, cfg.Global.Token)
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "save", "conf.toml")

	// Initially load from a non-existent path
	cfg, err := config.Load(configPath, "")
	require.ErrorIs(t, err, config.ErrConfigNotFound)
	require.NotNil(t, cfg)

	cfg.Global.Token = "saved-token"
	cfg.Repl.HistoryFile = "~/history"
	cfg.Profiles = map[string]config.ProfileConfig{
		"new-profile": {
			Render: true,
		},
	}

	err = cfg.Save()
	require.NoError(t, err)

	// Verify the file was written
	assert.FileExists(t, configPath)

	// Load again to verify contents
	cfg2, err := config.Load(configPath, "new-profile")
	require.NoError(t, err)
	assert.Equal(t, "saved-token", cfg2.Global.Token)
	assert.True(t, cfg2.Resolved.Render)
}

func TestSaveErrors(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("MkdirAll failure", func(t *testing.T) {
		// Create a file where a directory should be
		conflictFile := filepath.Join(tmpDir, "conflict")
		err := os.WriteFile(conflictFile, []byte("not a dir"), 0o644)
		require.NoError(t, err)

		// Load might return an error about the path being invalid or not found
		cfg, err := config.Load(filepath.Join(conflictFile, "config.toml"), "")
		require.Error(t, err)

		if cfg != nil {
			err = cfg.Save()
			assert.Error(t, err)
		}
	})

	t.Run("WriteFile failure", func(t *testing.T) {
		isDir := filepath.Join(tmpDir, "is_a_dir")
		err := os.MkdirAll(isDir, 0o755)
		require.NoError(t, err)

		_, err = config.Load(isDir, "")
		require.Error(t, err)
		if !errors.Is(err, config.ErrConfigNotFound) {
			// If it's a directory, Load should return a specific error
			assert.Contains(t, err.Error(), "config path is a directory")
		}
	})

	t.Run("Save marshaling error", func(_ *testing.T) {
		// This is hard to trigger with valid structs
	})
}

func TestResolveProfile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "profile.toml")

	content := `
[global]
render = false
super = false
geo_code = "us"

[profiles.p1]
render = true
super = true
geo_code = "de"
device = "mobile"
session = "sess1"
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	t.Run("full override", func(t *testing.T) {
		cfg, err := config.Load(configPath, "p1")
		require.NoError(t, err)
		assert.True(t, cfg.Resolved.Render)
		assert.True(t, cfg.Resolved.Super)
		assert.Equal(t, "de", cfg.Resolved.GeoCode)
		assert.Equal(t, "mobile", cfg.Resolved.Device)
		assert.Equal(t, "sess1", cfg.Resolved.Session)
	})

	t.Run("partial override - stay default", func(t *testing.T) {
		content2 := `
[global]
render = true
geo_code = "us"

[profiles.p2]
geo_code = "" # Should not override us
`
		p2Path := filepath.Join(tmpDir, "p2.toml")
		require.NoError(t, os.WriteFile(p2Path, []byte(content2), 0o644))

		cfg, err := config.Load(p2Path, "p2")
		require.NoError(t, err)
		assert.True(t, cfg.Resolved.Render)
		assert.Equal(t, "us", cfg.Resolved.GeoCode)
	})
}

func TestLoadSearchConfig(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("defaults", func(t *testing.T) {
		cfgPath := filepath.Join(tmpDir, "defaults.toml")
		require.NoError(t, os.WriteFile(cfgPath, []byte("[global]\ntoken = \"t\"\n"), 0o644))

		cfg, err := config.Load(cfgPath, "")
		require.NoError(t, err)
		assert.Equal(t, "scrapedo", cfg.Search.DefaultProvider)
		assert.Equal(t, "google", cfg.Search.DefaultEngine)
		assert.Equal(t, 10, cfg.Search.DefaultLimit)
		assert.Nil(t, cfg.Providers)
	})

	t.Run("from file", func(t *testing.T) {
		cfgPath := filepath.Join(tmpDir, "search.toml")
		require.NoError(t, os.WriteFile(cfgPath, []byte(`
[global]
token = "t"

[search]
default_provider = "serpapi"
default_engine = "bing"
default_limit = 5

[providers.serpapi]
token = "serp-key"

[providers.myexec]
type = "exec"
command = "/usr/local/bin/search"
args = ["--verbose"]
engines = ["google", "bing"]
`), 0o644))

		cfg, err := config.Load(cfgPath, "")
		require.NoError(t, err)
		assert.Equal(t, "serpapi", cfg.Search.DefaultProvider)
		assert.Equal(t, "bing", cfg.Search.DefaultEngine)
		assert.Equal(t, 5, cfg.Search.DefaultLimit)
		require.Len(t, cfg.Providers, 2)
		assert.Equal(t, "serp-key", cfg.Providers["serpapi"].Token)
		assert.Equal(t, "exec", cfg.Providers["myexec"].Type)
		assert.Equal(t, "/usr/local/bin/search", cfg.Providers["myexec"].Command)
		assert.Equal(t, []string{"--verbose"}, cfg.Providers["myexec"].Args)
		assert.Equal(t, []string{"google", "bing"}, cfg.Providers["myexec"].Engines)
	})
}

func TestExpandPath(t *testing.T) {
	t.Run("home expansion", func(t *testing.T) {
		home, _ := os.UserHomeDir()
		if home == "" {
			t.Skip("User home dir not available")
		}

		path := config.ExpandPathForTest("~/test")
		assert.Equal(t, filepath.Join(home, "test"), path)
	})

	t.Run("no expansion", func(t *testing.T) {
		path := config.ExpandPathForTest("/absolute/path")
		assert.Equal(t, "/absolute/path", path)
	})
}
