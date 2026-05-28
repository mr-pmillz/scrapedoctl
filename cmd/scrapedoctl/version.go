package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/mr-pmillz/scrapedoctl/internal/version"
)

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version and check for updates",
		RunE:  runVersion,
	}
	return cmd
}

func runVersion(cmd *cobra.Command, _ []string) error {
	w := cmd.OutOrStdout()

	fmt.Fprintf(w, "scrapedoctl %s\n", version.Version)
	fmt.Fprintf(w, "  commit:   %s\n", version.GitCommit)
	fmt.Fprintf(w, "  built:    %s\n", version.BuildDate)
	fmt.Fprintf(w, "  go:       %s\n", runtime.Version())
	fmt.Fprintf(w, "  os/arch:  %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(w, "  repo:     %s\n", version.RepoURL)
	fmt.Fprintln(w)

	tag, url, newer, err := version.CheckLatest(cmd.Context())
	if err != nil {
		fmt.Fprintf(w, "  update check: failed (%v)\n", err)
		return nil
	}

	if newer {
		fmt.Fprintf(w, "  A new version is available: %s\n", tag)
		fmt.Fprintf(w, "  Download: %s\n", url)
		fmt.Fprintf(w, "  Run: scrapedoctl update\n")
	} else {
		fmt.Fprintf(w, "  You are up to date (%s)\n", tag)
	}

	return nil
}
