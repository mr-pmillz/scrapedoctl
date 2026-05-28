package ui_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/internal/ui"
)

func TestSetCustomHelp(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "testcmd",
		Short: "Test command description",
	}
	subCmd := &cobra.Command{
		Use:   "sub",
		Short: "Subcommand description",
	}
	cmd.AddCommand(subCmd)
	cmd.Flags().String("foo", "", "Foo flag")

	ui.SetCustomHelp(cmd)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	cmd.HelpFunc()(cmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	output := buf.String()

	if !strings.Contains(output, "Scrape.do CLI & MCP Server") {
		t.Errorf("Expected banner and title, got: %s", output)
	}
	if !strings.Contains(output, "USAGE") {
		t.Error("Expected USAGE section")
	}
	if !strings.Contains(output, "testcmd") {
		t.Error("Expected command usage")
	}
	if !strings.Contains(output, "COMMANDS") {
		t.Error("Expected COMMANDS section")
	}
	if !strings.Contains(output, "sub") {
		t.Error("Expected subcommand name")
	}
	if !strings.Contains(output, "FLAGS") {
		t.Error("Expected FLAGS section")
	}
	if !strings.Contains(output, "--foo") {
		t.Error("Expected flag name")
	}
}
