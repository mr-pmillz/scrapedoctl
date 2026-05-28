package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mr-pmillz/scrapedoctl/internal/config"
)

func TestRootCmd_AutoSetupTrigger(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "nonexistent.yaml")

	// We want to test that PersistentPreRunE triggers "No configuration file found"
	// and tries to run "install". Since "install" is interactive (huh.Form),
	// it will fail in tests unless we mock it or just check the output before it fails.

	cmd := newRootCmd()
	cmd.SetArgs([]string{"scrape", "http://example.com", "--config", configPath})

	// Capture output
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	// Run command. It should fail because "install" will try to run huh.Form.
	_ = cmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "No configuration file found. Starting initial setup...") {
		t.Errorf("Expected auto-setup trigger message, got: %s", output)
	}
}

func TestRootCmd_Help(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Help command failed: %v", err)
	}
}

func TestRootCmd_Metadata(t *testing.T) {
	// Initialize cfg to avoid nil pointer
	cfg = &config.Config{}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"metadata"})

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Metadata command failed: %v", err)
	}

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "\"name\": \"scrapedoctl\"") {
		t.Errorf("Expected JSON metadata, got: %s", output)
	}
}
