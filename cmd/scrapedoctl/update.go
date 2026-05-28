package main

import (
	"fmt"
	"io"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/mr-pmillz/scrapedoctl/internal/version"
)

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Check for updates and show install instructions",
		RunE:  runUpdate,
	}
}

func runUpdate(cmd *cobra.Command, _ []string) error {
	w := cmd.OutOrStdout()

	fmt.Fprintf(w, "Current version: %s\n", version.Version)
	fmt.Fprintln(w, "Checking for updates...")

	tag, url, newer, err := version.CheckLatest(cmd.Context())
	if err != nil {
		return fmt.Errorf("update check failed: %w", err)
	}

	if !newer {
		fmt.Fprintf(w, "\nYou are already on the latest version (%s).\n", tag)
		return nil
	}

	fmt.Fprintf(w, "\nNew version available: %s\n\n", tag)
	fmt.Fprintf(w, "Release page: %s\n\n", url)

	printInstallInstructions(w, tag)

	return nil
}

func printInstallInstructions(w io.Writer, tag string) {
	fmt.Fprintln(w, "Install options:")
	fmt.Fprintln(w)

	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Direct binary download.
	base := fmt.Sprintf(
		"%s/download/%s/scrapedoctl_%s_%s",
		version.ReleasesURL, tag,
		capitalize(goos), archName(goarch),
	)

	switch goos {
	case "linux":
		fmt.Fprintf(w, "  # Download binary\n")
		fmt.Fprintf(w, "  curl -fsSL %s.tar.gz -o scrapedoctl.tar.gz\n", base)
		fmt.Fprintf(w, "  tar xzf scrapedoctl.tar.gz scrapedoctl\n")
		fmt.Fprintf(w, "  sudo mv scrapedoctl /usr/local/bin/\n")
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  # Or install via package manager\n")
		fmt.Fprintf(w, "  # RPM:  sudo rpm -U scrapedoctl_*.rpm\n")
		fmt.Fprintf(w, "  # DEB:  sudo dpkg -i scrapedoctl_*.deb\n")
	case "darwin":
		fmt.Fprintf(w, "  curl -fsSL %s.tar.gz -o scrapedoctl.tar.gz\n", base)
		fmt.Fprintf(w, "  tar xzf scrapedoctl.tar.gz scrapedoctl\n")
		fmt.Fprintf(w, "  sudo mv scrapedoctl /usr/local/bin/\n")
	case "windows":
		fmt.Fprintf(w, "  # Download from:\n")
		fmt.Fprintf(w, "  # %s.zip\n", base)
	default:
		fmt.Fprintf(w, "  Visit: %s\n", version.ReleasesURL)
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  All releases: %s\n", version.ReleasesURL)
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}

func archName(goarch string) string {
	if goarch == "amd64" {
		return "x86_64"
	}
	return goarch
}
