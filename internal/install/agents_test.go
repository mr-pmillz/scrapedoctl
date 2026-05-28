package install_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/internal/install"
)

func TestConfigureAgents(t *testing.T) {
	tempHome := t.TempDir()

	// Mock HOME/USERPROFILE
	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)

	apiToken := "test-token"
	agentIDs := []string{"claude", "codex"} // One JSON, one TOML

	err := install.ConfigureAgents(agentIDs, apiToken)
	require.NoError(t, err)

	// Verify Claude (JSON)
	claudePath := filepath.Join(tempHome, ".claude.json")
	if _, err := os.Stat(claudePath); os.IsNotExist(err) {
		t.Errorf("Claude config not created at %s", claudePath)
	} else {
		data, err := os.ReadFile(claudePath)
		require.NoError(t, err)
		var config map[string]any
		err = json.Unmarshal(data, &config)
		require.NoError(t, err)
		mcpServers := config["mcpServers"].(map[string]any)
		scrapeDo := mcpServers["scrape-do"].(map[string]any)
		if scrapeDo["env"].(map[string]any)["SCRAPEDO_TOKEN"] != apiToken {
			t.Errorf("Expected token %s, got %v", apiToken, scrapeDo["env"])
		}
	}

	// Verify Codex (TOML)
	codexPath := filepath.Join(tempHome, ".codex", "config.toml")
	if _, err := os.Stat(codexPath); os.IsNotExist(err) {
		t.Errorf("Codex config not created at %s", codexPath)
	} else {
		k := koanf.New(".")
		err = k.Load(install.FileProvider(codexPath), toml.Parser())
		require.NoError(t, err)
		if k.String("mcpServers.scrape-do.env.SCRAPEDO_TOKEN") != apiToken {
			t.Errorf("Expected token %s, got %v", apiToken, k.Get("mcpServers.scrape-do.env"))
		}
	}
}

func TestInjectConfig_AllAgents(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	def := install.MCPServerConfig{
		Command: "scrapedoctl",
		Args:    []string{"mcp"},
		Env:     map[string]string{"SCRAPEDO_TOKEN": "token"},
	}

	for _, info := range install.SupportedAgents {
		t.Run(info.ID, func(t *testing.T) {
			err := install.InjectConfig(info, def)
			require.NoError(t, err)
			path := install.ExpandPath(info.ConfigPath)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("Config file not created for %s at %s", info.ID, path)
			}
		})
	}
}

func TestInjectJSON_Merge(t *testing.T) {
	tempDir := t.TempDir()

	path := filepath.Join(tempDir, "test.json")
	initialContent := `{"existing": "value", "mcpServers": {"other": {"command": "other"}}}`
	err := os.WriteFile(path, []byte(initialContent), 0o644)
	require.NoError(t, err)

	def := install.MCPServerConfig{Command: "scrapedoctl"}
	err = install.InjectJSON(path, def)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var config map[string]any
	err = json.Unmarshal(data, &config)
	require.NoError(t, err)

	if config["existing"] != "value" {
		t.Errorf("Existing value lost")
	}
	mcpServers := config["mcpServers"].(map[string]any)
	if _, ok := mcpServers["other"]; !ok {
		t.Errorf("Existing mcpServer lost")
	}
	if _, ok := mcpServers["scrape-do"]; !ok {
		t.Errorf("scrape-do mcpServer not added")
	}
}

func TestInjectJSON_Corrupted(t *testing.T) {
	tempDir := t.TempDir()

	path := filepath.Join(tempDir, "test.json")
	err := os.WriteFile(path, []byte("invalid json"), 0o644)
	require.NoError(t, err)

	def := install.MCPServerConfig{Command: "scrapedoctl"}
	err = install.InjectJSON(path, def)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var config map[string]any
	err = json.Unmarshal(data, &config)
	require.NoError(t, err)
	if _, ok := config["mcpServers"]; !ok {
		t.Errorf("mcpServers not created after corruption")
	}
}

func TestInjectTOML_Merge(t *testing.T) {
	tempDir := t.TempDir()

	path := filepath.Join(tempDir, "test.toml")
	initialContent := `
[other]
key = "value"

[mcpServers.other]
command = "other"
`
	err := os.WriteFile(path, []byte(initialContent), 0o644)
	require.NoError(t, err)

	def := install.MCPServerConfig{
		Command: "scrapedoctl",
		Args:    []string{"mcp"},
		Env:     map[string]string{"SCRAPEDO_TOKEN": "token"},
	}
	err = install.InjectTOML(path, def)
	require.NoError(t, err)

	k := koanf.New(".")
	err = k.Load(install.FileProvider(path), toml.Parser())
	require.NoError(t, err)

	if k.String("other.key") != "value" {
		t.Errorf("Existing TOML value lost")
	}
	if k.String("mcpServers.other.command") != "other" {
		t.Errorf("Existing TOML mcpServer lost")
	}
	if k.String("mcpServers.scrape-do.command") != "scrapedoctl" {
		t.Errorf("scrape-do TOML mcpServer not added")
	}
}

func TestInjectConfig_Error(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping directory permission test on Windows")
	}

	tempDir := t.TempDir()

	// Create a file where a directory should be
	blockedDir := filepath.Join(tempDir, "blocked")
	err := os.WriteFile(blockedDir, []byte("i am a file"), 0o644)
	require.NoError(t, err)

	info := install.AgentConfigInfo{
		ID:         "test",
		ConfigPath: filepath.Join(blockedDir, "config.json"),
		Format:     "json",
	}
	def := install.MCPServerConfig{}

	err = install.InjectConfig(info, def)
	assert.Error(t, err)
}

func TestExpandPath_NoHome(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")

	path := "~/test"
	_ = install.ExpandPath(path)
}

func TestGenerateProjectFiles_CreatesAll(t *testing.T) {
	dir := t.TempDir()

	err := install.GenerateProjectFiles(dir)
	require.NoError(t, err)

	expected := []string{".mcp.json", "CLAUDE.md", "AGENTS.md", "GEMINI.md"}
	for _, name := range expected {
		path := filepath.Join(dir, name)
		info, statErr := os.Stat(path)
		require.NoError(t, statErr, "file %s should exist", name)
		assert.Positive(t, info.Size(), "file %s should not be empty", name)
	}
}

func TestGenerateProjectFiles_MCPJSONContent(t *testing.T) {
	dir := t.TempDir()

	err := install.GenerateProjectFiles(dir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	servers, ok := parsed["mcpServers"].(map[string]any)
	require.True(t, ok, "mcpServers key must exist")

	scrapeDo, ok := servers["scrape-do"].(map[string]any)
	require.True(t, ok, "scrape-do server must exist")
	assert.Equal(t, "scrapedoctl", scrapeDo["command"])
}

func TestGenerateProjectFiles_SkipsExisting(t *testing.T) {
	dir := t.TempDir()

	// Pre-create CLAUDE.md with custom content.
	customContent := "my custom content"
	err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(customContent), 0o644)
	require.NoError(t, err)

	err = install.GenerateProjectFiles(dir)
	require.NoError(t, err)

	// CLAUDE.md should still have the original content.
	data, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Equal(t, customContent, string(data))

	// Other files should be created.
	for _, name := range []string{".mcp.json", "AGENTS.md", "GEMINI.md"} {
		_, statErr := os.Stat(filepath.Join(dir, name))
		assert.NoError(t, statErr, "file %s should exist", name)
	}
}

func TestGenerateProjectFiles_ContentKeywords(t *testing.T) {
	dir := t.TempDir()

	err := install.GenerateProjectFiles(dir)
	require.NoError(t, err)

	tests := []struct {
		file    string
		keyword string
	}{
		{"CLAUDE.md", "scrape_url"},
		{"CLAUDE.md", "web_search"},
		{"CLAUDE.md", "golangci-lint"},
		{"AGENTS.md", "Agent Contract"},
		{"AGENTS.md", "scrapedoctl mcp"},
		{"GEMINI.md", "Gemini Notes"},
		{"GEMINI.md", "AGENTS.md"},
	}

	for _, tc := range tests {
		data, readErr := os.ReadFile(filepath.Join(dir, tc.file))
		require.NoError(t, readErr)
		assert.Contains(t, string(data), tc.keyword,
			"%s should contain %q", tc.file, tc.keyword)
	}
}

func TestGenerateProjectFiles_WriteError(t *testing.T) {
	// Pass a non-existent directory to trigger a write error.
	err := install.GenerateProjectFiles("/nonexistent/path/that/does/not/exist")
	assert.Error(t, err)
}

func TestProjectFiles_Count(t *testing.T) {
	files := install.ProjectFiles()
	assert.Len(t, files, 4)
}
