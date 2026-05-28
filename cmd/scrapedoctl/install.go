// Package main provides the entry point for scrapedoctl.
package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/mr-pmillz/scrapedoctl/internal/install"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install and configure scrapedoctl",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runInstall()
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Generate project integration files (.mcp.json, CLAUDE.md, AGENTS.md, GEMINI.md)",
		RunE: func(_ *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			return install.GenerateProjectFiles(cwd)
		},
	})

	return cmd
}

func runInstall() error {
	token, agents, err := promptInstallForm()
	if err != nil {
		return err
	}

	ensureLogDirectory()

	if err := install.ConfigureAgents(agents, token); err != nil {
		return fmt.Errorf("failed to configure agents: %w", err)
	}

	if err := promptProjectFiles(); err != nil {
		return err
	}

	cfg.Global.Token = token
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println("\nInstallation complete! You can now use Scrape.do tools in your selected AI agents.")
	return nil
}

func promptInstallForm() (string, []string, error) {
	var token string
	var agents []string

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Scrape.do API Token").
				Description("Enter your API token (find it at https://scrape.do/dashboard)").
				Value(&token).
				EchoMode(huh.EchoModePassword),
		),
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Configure AI Agents").
				Description("Select agents to configure for Scrape.do MCP server (Space to select)").
				Options(
					huh.NewOption("Claude Code", "claude"),
					huh.NewOption("JetBrains Junie", "junie"),
					huh.NewOption("Gemini CLI", "gemini"),
					huh.NewOption("Codex AI", "codex"),
					huh.NewOption("Kimi AI", "kimi"),
					huh.NewOption("OpenCode AI", "opencode"),
				).
				Value(&agents),
		),
	).Run()
	if err != nil {
		return "", nil, fmt.Errorf("install form failed: %w", err)
	}

	return token, agents, nil
}

func ensureLogDirectory() {
	logDir := "/var/log/scrapedoctl"
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		fmt.Printf("\nWarning: Could not create log directory %s: %v\n", logDir, err)
		fmt.Printf("To fix this, please run: sudo mkdir -p %s && sudo chown $USER %s\n", logDir, logDir)
	} else {
		fmt.Printf("Log directory created: %s\n", logDir)
	}
}

func promptProjectFiles() error {
	var generateFiles bool

	err := huh.NewConfirm().
		Title("Generate project integration files in current directory?").
		Description(".mcp.json, CLAUDE.md, AGENTS.md, GEMINI.md").
		Value(&generateFiles).
		Run()
	if err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}

	if !generateFiles {
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	if err := install.GenerateProjectFiles(cwd); err != nil {
		return fmt.Errorf("failed to generate project files: %w", err)
	}

	return nil
}
