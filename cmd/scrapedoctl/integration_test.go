package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/internal/cache"
	"github.com/mr-pmillz/scrapedoctl/internal/install"
	"github.com/mr-pmillz/scrapedoctl/internal/version"
	"github.com/mr-pmillz/scrapedoctl/pkg/search"
)

func TestCLI_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "conf.toml")

	// Create a default config
	require.NoError(t, os.WriteFile(configPath, []byte(`
[global]
token = "test-token"
[cache]
enabled = true
path = "`+filepath.Join(tmpDir, "cache.db")+`"
`), 0o644))

	// Helper to run command and capture REAL stdout
	runCmd := func(args ...string) (string, string, error) {
		oldStdout := os.Stdout
		oldStderr := os.Stderr
		rOut, wOut, _ := os.Pipe()
		rErr, wErr, _ := os.Pipe()
		os.Stdout = wOut
		os.Stderr = wErr

		outChan := make(chan string)
		errChan := make(chan string)
		go func() {
			var buf bytes.Buffer
			_, _ = io.Copy(&buf, rOut)
			outChan <- buf.String()
		}()
		go func() {
			var buf bytes.Buffer
			_, _ = io.Copy(&buf, rErr)
			errChan <- buf.String()
		}()

		root := newRootCmd()
		fullArgs := append([]string{"--config", configPath}, args...)
		root.SetArgs(fullArgs)

		err := root.Execute()

		_ = wOut.Close()
		_ = wErr.Close()
		os.Stdout = oldStdout
		os.Stderr = oldStderr

		return <-outChan, <-errChan, err
	}

	t.Run("metadata command", func(t *testing.T) {
		stdout, _, err := runCmd("metadata")
		require.NoError(t, err)
		assert.Contains(t, stdout, `"name": "scrapedoctl"`)
		assert.Contains(t, stdout, `"version": "0.2.0"`)
	})

	t.Run("config list command", func(t *testing.T) {
		stdout, _, err := runCmd("config", "list")
		require.NoError(t, err)
		assert.Contains(t, stdout, "Global Token: test-token")
	})

	t.Run("config set command", func(t *testing.T) {
		_, _, err := runCmd("config", "set", "global.base_url=https://new-api.com")
		require.NoError(t, err)

		// Verify change
		stdout, _, _ := runCmd("config", "list")
		assert.Contains(t, stdout, "Global BaseURL: https://new-api.com")
	})

	t.Run("cache stats command", func(t *testing.T) {
		stdout, _, err := runCmd("cache", "stats")
		require.NoError(t, err)
		assert.Contains(t, stdout, "Total entries:")
	})

	t.Run("cache clear command", func(t *testing.T) {
		stdout, _, err := runCmd("cache", "clear")
		require.NoError(t, err)
		assert.Contains(t, stdout, "Cache cleared successfully")
	})

	t.Run("history command empty", func(t *testing.T) {
		stdout, _, err := runCmd("history", "https://example.com")
		require.NoError(t, err)
		assert.Contains(t, stdout, "No history found")
	})
}

func TestCLI_Integration_Errors(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("missing token error", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "no_token.toml")
		_ = os.WriteFile(configPath, []byte(`[global]`), 0o644)

		root := newRootCmd()
		root.SetArgs([]string{"--config", configPath, "scrape", "http://example.com"})

		// We can't easily capture the error from RunE when it's wrapped by Cobra in this simple test,
		// but Execute should return it.
		err := root.Execute()
		require.Error(t, err)
	})

	t.Run("invalid config path", func(t *testing.T) {
		root := newRootCmd()
		root.SetArgs([]string{"--config", "/nonexistent/path/conf.toml", "metadata"})
		err := root.Execute()
		require.NoError(t, err)
	})
}

// newTestRootCmd creates a root command with a temporary config file that has
// no token set and cache disabled. It also ensures SCRAPEDO_TOKEN is unset.
// Returns the command and the config-path prefix args for composing SetArgs.
func newTestRootCmd(t *testing.T) (*cobra.Command, []string) {
	t.Helper()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test_conf.toml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("[global]\n"), 0o644))
	t.Setenv("SCRAPEDO_TOKEN", "")
	root := newRootCmd()
	root.SilenceUsage = true
	return root, []string{"--config", cfgPath}
}

// newTestRootCmdWithToken creates a root command that has a valid token and cache.
func newTestRootCmdWithToken(t *testing.T) (*cobra.Command, []string) {
	t.Helper()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test_conf.toml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(`
[global]
token = "test-token"
[cache]
enabled = true
path = "`+filepath.Join(tmpDir, "cache.db")+`"
`), 0o644))
	root := newRootCmd()
	root.SilenceUsage = true
	return root, []string{"--config", cfgPath}
}

// newTestRootCmdNoCacheWithToken creates a root command that has a valid token
// but cache disabled, useful for testing cache-not-initialized error paths.
func newTestRootCmdNoCacheWithToken(t *testing.T) (*cobra.Command, []string) {
	t.Helper()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test_conf.toml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(`
[global]
token = "test-token"
[cache]
enabled = false
`), 0o644))
	root := newRootCmd()
	root.SilenceUsage = true
	return root, []string{"--config", cfgPath}
}

func TestScrapeCmd_MissingToken(t *testing.T) {
	t.Setenv("SCRAPEDO_TOKEN", "")
	root, base := newTestRootCmd(t)
	root.SetArgs(append(base, "scrape", "https://example.com"))
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SCRAPEDO_TOKEN")
}

func TestScrapeCmd_MissingURL(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)
	root.SetArgs(append(base, "scrape"))
	err := root.Execute()
	require.Error(t, err)
}

func TestREPLCmd_MissingToken(t *testing.T) {
	t.Setenv("SCRAPEDO_TOKEN", "")
	root, base := newTestRootCmd(t)
	root.SetArgs(append(base, "repl"))
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SCRAPEDO_TOKEN")
}

func TestMCPCmd_MissingToken(t *testing.T) {
	t.Setenv("SCRAPEDO_TOKEN", "")
	root, base := newTestRootCmd(t)
	root.SetArgs(append(base, "mcp"))
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SCRAPEDO_TOKEN")
}

func TestHistoryCmd_CacheNotInitialized(t *testing.T) {
	oldCache := cacheStore
	cacheStore = nil
	t.Cleanup(func() { cacheStore = oldCache })

	root, base := newTestRootCmdNoCacheWithToken(t)
	root.SetArgs(append(base, "history", "https://example.com"))
	err := root.Execute()
	require.Error(t, err)
	assert.ErrorIs(t, err, errCacheNotInitialized)
}

func TestCacheStatsCmd_CacheNotInitialized(t *testing.T) {
	oldCache := cacheStore
	cacheStore = nil
	t.Cleanup(func() { cacheStore = oldCache })

	root, base := newTestRootCmdNoCacheWithToken(t)
	root.SetArgs(append(base, "cache", "stats"))
	err := root.Execute()
	require.Error(t, err)
	assert.ErrorIs(t, err, errCacheNotInitialized)
}

func TestCacheClearCmd_CacheNotInitialized(t *testing.T) {
	oldCache := cacheStore
	cacheStore = nil
	t.Cleanup(func() { cacheStore = oldCache })

	root, base := newTestRootCmdNoCacheWithToken(t)
	root.SetArgs(append(base, "cache", "clear"))
	err := root.Execute()
	require.Error(t, err)
	assert.ErrorIs(t, err, errCacheNotInitialized)
}

func TestConfigSetCmd_UnsupportedKey(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)
	root.SetArgs(append(base, "config", "set", "invalid_key=value"))
	err := root.Execute()
	require.Error(t, err)
	assert.ErrorIs(t, err, errUnsupportedConfigKey)
}

func TestConfigSetCmd_InvalidFormat(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)
	root.SetArgs(append(base, "config", "set", "noequalssign"))
	err := root.Execute()
	require.Error(t, err)
	assert.ErrorIs(t, err, errInvalidConfigFormat)
}

func TestConfigSetCmd_SetToken(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)
	root.SetArgs(append(base, "config", "set", "global.token=new-token"))
	err := root.Execute()
	require.NoError(t, err)
}

func TestConfigSetCmd_SetHistoryFile(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)
	root.SetArgs(append(base, "config", "set", "repl.history_file=/tmp/test_history"))
	err := root.Execute()
	require.NoError(t, err)
}

func TestPrintHistoryTable(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	records := []cache.ScrapeRecord{
		{
			ID:        1,
			URL:       "https://example.com",
			CreatedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
			Metadata:  `{"cost": 1, "remaining_credits": 100, "status": 200}`,
			Content:   "test content",
		},
		{
			ID:        2,
			URL:       "https://example.com",
			CreatedAt: time.Date(2025, 1, 16, 12, 0, 0, 0, time.UTC),
			Metadata:  `{"cost": 2, "remaining_credits": 98, "status": 200}`,
			Content:   "test content 2",
		},
	}

	err := printHistoryTable(records)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	require.NoError(t, err)
	assert.Contains(t, output, "ID")
	assert.Contains(t, output, "DATE")
	assert.Contains(t, output, "COST")
	assert.Contains(t, output, "2025-01-15")
	assert.Contains(t, output, "2025-01-16")
}

func TestPrintHistoryTable_InvalidMetadata(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	records := []cache.ScrapeRecord{
		{
			ID:        1,
			URL:       "https://example.com",
			CreatedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
			Metadata:  `not valid json`,
			Content:   "test content",
		},
	}

	err := printHistoryTable(records)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	require.NoError(t, err)
}

func TestRun_Success(t *testing.T) {
	// run() with help should succeed
	oldArgs := os.Args
	os.Args = []string{"scrapedoctl", "help"}
	t.Cleanup(func() { os.Args = oldArgs })

	err := run()
	require.NoError(t, err)
}

func TestHistoryCmd_MissingURLArg(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)
	root.SetArgs(append(base, "history"))
	err := root.Execute()
	require.Error(t, err)
}

func TestScrapeCmd_WithRenderFlag(t *testing.T) {
	// Test that the scrape command accepts the --render flag
	// It will fail at the HTTP level but should pass flag parsing and token check
	t.Setenv("SCRAPEDO_TOKEN", "")
	root, base := newTestRootCmd(t)
	root.SetArgs(append(base, "scrape", "--render", "https://example.com"))
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SCRAPEDO_TOKEN")
}

func TestScrapeCmd_WithSuperFlag(t *testing.T) {
	t.Setenv("SCRAPEDO_TOKEN", "")
	root, base := newTestRootCmd(t)
	root.SetArgs(append(base, "scrape", "--super", "https://example.com"))
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SCRAPEDO_TOKEN")
}

func TestRootCmd_WithInvalidProfile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "conf.toml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(`
[global]
token = "test-token"
`), 0o644))

	root := newRootCmd()
	root.SilenceUsage = true
	root.SetArgs([]string{"--config", cfgPath, "--profile", "nonexistent", "metadata"})

	err := root.Execute()
	// Profile not found should produce an error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "profile not found")
}

func TestRootCmd_InvalidConfigLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "bad_conf.toml")
	// Write invalid TOML that will cause a parse error
	require.NoError(t, os.WriteFile(cfgPath, []byte(`
[global
broken toml
`), 0o644))

	root := newRootCmd()
	root.SilenceUsage = true
	root.SetArgs([]string{"--config", cfgPath, "scrape", "https://example.com"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config")
}

func TestScrapeCmd_HTTPError(t *testing.T) {
	// Test scrape with a valid token but an unreachable URL.
	// This covers client creation, cache attachment, request building,
	// and the error return from client.Scrape.
	root, base := newTestRootCmdWithToken(t)
	root.SetArgs(append(base, "scrape", "https://localhost:1/nonexistent"))
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scrape failed")
}

func TestScrapeCmd_HTTPErrorWithFlags(t *testing.T) {
	// Same as above but with render and super flags to cover those branches
	root, base := newTestRootCmdWithToken(t)
	root.SetArgs(append(
		base, "scrape", "--render", "--super",
		"--no-cache", "--refresh", "https://localhost:1/nonexistent",
	))
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scrape failed")
}

func TestScrapeCmd_NoCacheHTTPError(t *testing.T) {
	// Test scrape without cache (cache disabled)
	root, base := newTestRootCmdNoCacheWithToken(t)
	// Need to nil out cacheStore explicitly since test ordering is unpredictable
	oldCache := cacheStore
	cacheStore = nil
	t.Cleanup(func() { cacheStore = oldCache })
	root.SetArgs(append(base, "scrape", "https://localhost:1/nonexistent"))
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scrape failed")
}

func TestRunError(t *testing.T) {
	// Test run() with args that cause an error
	oldArgs := os.Args
	os.Args = []string{"scrapedoctl", "scrape"}
	t.Cleanup(func() { os.Args = oldArgs })

	err := run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "execution failed")
}

func TestCacheCmd_Subcommands(t *testing.T) {
	// Test cache parent command with no subcommand shows help
	root, base := newTestRootCmdWithToken(t)
	root.SetArgs(append(base, "cache"))
	err := root.Execute()
	require.NoError(t, err)
}

func TestConfigCmd_NoSubcommand(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)
	root.SetArgs(append(base, "config"))
	err := root.Execute()
	require.NoError(t, err)
}

func TestSearchCmd_MissingQuery(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)
	root.SetArgs(append(base, "search"))
	err := root.Execute()
	require.Error(t, err)
}

func TestSearchCmd_NoProviders(t *testing.T) {
	// Use a config with no token so no providers are registered.
	t.Setenv("SCRAPEDO_TOKEN", "")
	root, base := newTestRootCmd(t)

	// Ensure searchRouter has no providers.
	oldRouter := searchRouter
	searchRouter = search.NewRouter()
	t.Cleanup(func() { searchRouter = oldRouter })

	root.SetArgs(append(base, "search", "test query"))
	err := root.Execute()
	require.Error(t, err)
	assert.ErrorIs(t, err, errNoSearchProviders)
}

func TestSearchCmd_UnsupportedEngine(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)
	root.SetArgs(append(base, "search", "--engine", "nonexistent", "test query"))
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no provider found for engine")
}

func TestUsageCmd_CacheNotInitialized(t *testing.T) {
	oldCache := cacheStore
	cacheStore = nil
	t.Cleanup(func() { cacheStore = oldCache })

	root, base := newTestRootCmdNoCacheWithToken(t)
	root.SetArgs(append(base, "usage"))
	err := root.Execute()
	require.Error(t, err)
	assert.ErrorIs(t, err, errCacheNotInitialized)
}

func TestUsageCmd_Default(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	root.SetArgs(append(base, "usage"))
	err := root.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	require.NoError(t, err)
	assert.Contains(t, output, "Usage since")
	assert.Contains(t, output, "Total:")
}

func TestUsageCmd_WithFlags(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)
	root.SetArgs(append(base, "usage", "--week"))
	err := root.Execute()
	require.NoError(t, err)
}

func TestUsageCmd_AllFlag(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)
	root.SetArgs(append(base, "usage", "--all"))
	err := root.Execute()
	require.NoError(t, err)
}

func TestUsageCmd_MonthFlag(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)
	root.SetArgs(append(base, "usage", "--month"))
	err := root.Execute()
	require.NoError(t, err)
}

func TestUsageCmd_JSONOutput(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	root.SetArgs(append(base, "usage", "--json"))
	err := root.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	require.NoError(t, err)
	assert.Contains(t, output, `"since"`)
	assert.Contains(t, output, `"total"`)
}

func TestSearchCmd_JSONOutput(t *testing.T) {
	// The search will fail at the HTTP level since the token is fake,
	// but this tests the command structure and flag parsing.
	root, base := newTestRootCmdWithToken(t)
	root.SetArgs(append(base, "search", "--json", "--engine", "google", "test query"))
	err := root.Execute()
	// Will fail because the real API is not available with a fake token.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search failed")
}

func TestVersionCmd(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs(append(base, "version"))

	// version calls CheckLatest which hits the real API; that may fail
	// but the command itself should not return an error.
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), version.Version)
}

func TestCompletionCmd_Fish(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	root.SetArgs(append(base, "completion", "fish"))
	err := root.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestCompletionCmd_InvalidShell(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)
	root.SetArgs(append(base, "completion", "invalid"))
	err := root.Execute()
	require.Error(t, err)
}

func TestCompletionCmd_Bash(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	root.SetArgs(append(base, "completion", "bash"))
	err := root.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
	assert.Contains(t, buf.String(), "bash")
}

func TestCompletionCmd_Zsh(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	root.SetArgs(append(base, "completion", "zsh"))
	err := root.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestCompletionInstallCmd_Bash(t *testing.T) {
	tmpDir := t.TempDir()

	// Override completionPaths to use the temp directory.
	oldPaths := completionPaths
	completionPaths = map[string]struct {
		system string
		user   string
	}{
		shellBash: {
			system: filepath.Join(tmpDir, "system", "scrapedoctl"),
			user:   filepath.Join(tmpDir, "user", "scrapedoctl"),
		},
	}
	t.Cleanup(func() { completionPaths = oldPaths })

	root, base := newTestRootCmdWithToken(t)
	root.SetArgs(append(base, "completion", "install", "bash"))
	err := root.Execute()
	require.NoError(t, err)

	// Verify the file was created in the user path.
	_, statErr := os.Stat(filepath.Join(tmpDir, "user", "scrapedoctl"))
	assert.NoError(t, statErr)
}

func TestInstallInitCmd(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to temp dir so install init writes there.
	t.Chdir(tmpDir)

	err := install.GenerateProjectFiles(tmpDir)
	require.NoError(t, err)

	// Verify 4 files created: .mcp.json, CLAUDE.md, AGENTS.md, GEMINI.md
	expected := []string{".mcp.json", "CLAUDE.md", "AGENTS.md", "GEMINI.md"}
	for _, name := range expected {
		_, statErr := os.Stat(filepath.Join(tmpDir, name))
		assert.NoError(t, statErr, "expected file %s to exist", name)
	}
}

func TestUpdateCmd(t *testing.T) {
	root, base := newTestRootCmdWithToken(t)

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs(append(base, "update"))

	// update calls CheckLatest which may fail with network error;
	// that's an error return, but the command structure is tested.
	_ = root.Execute()
	// Just verify the command ran and produced output.
	assert.Contains(t, buf.String(), "Current version")
}

func TestVersionCmd_WithMockNewer(t *testing.T) {
	srv := newMockGitHubServer(t, "v99.0.0")
	defer srv.Close()
	old := version.SetAPIBaseURL(srv.URL)
	defer version.SetAPIBaseURL(old)

	root, base := newTestRootCmdWithToken(t)
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs(append(base, "version"))
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "new version is available")
}

func TestVersionCmd_WithMockSame(t *testing.T) {
	srv := newMockGitHubServer(t, "v"+version.Version)
	defer srv.Close()
	old := version.SetAPIBaseURL(srv.URL)
	defer version.SetAPIBaseURL(old)

	root, base := newTestRootCmdWithToken(t)
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs(append(base, "version"))
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "up to date")
}

func TestUpdateCmd_WithMockNewer(t *testing.T) {
	srv := newMockGitHubServer(t, "v99.0.0")
	defer srv.Close()
	old := version.SetAPIBaseURL(srv.URL)
	defer version.SetAPIBaseURL(old)

	root, base := newTestRootCmdWithToken(t)
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs(append(base, "update"))
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "New version available")
	assert.Contains(t, buf.String(), "Install options")
}

func TestUpdateCmd_WithMockSame(t *testing.T) {
	srv := newMockGitHubServer(t, "v"+version.Version)
	defer srv.Close()
	old := version.SetAPIBaseURL(srv.URL)
	defer version.SetAPIBaseURL(old)

	root, base := newTestRootCmdWithToken(t)
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs(append(base, "update"))
	err := root.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "already on the latest version")
}

func TestCapitalize(t *testing.T) {
	assert.Equal(t, "Linux", capitalize("linux"))
	assert.Equal(t, "Darwin", capitalize("darwin"))
	assert.Empty(t, capitalize(""))
}

func TestArchName(t *testing.T) {
	assert.Equal(t, "x86_64", archName("amd64"))
	assert.Equal(t, "arm64", archName("arm64"))
}

// newMockGitHubServer returns an httptest.Server that mimics the GitHub releases API.
func newMockGitHubServer(t *testing.T, tagName string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]string{
			"tag_name": tagName,
			"html_url": "https://github.com/mr-pmillz/scrapedoctl/releases/tag/" + tagName,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}
