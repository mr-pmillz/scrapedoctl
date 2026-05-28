package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

type crawlFlags struct {
	depth  int
	limit  int
	output string
	format string
}

func newCrawlCmd() *cobra.Command {
	cf := &crawlFlags{}
	cmd := &cobra.Command{
		Use:   "crawl <url>",
		Short: "Recursively crawl a site and save content",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return runCrawl(cmd, args, cf) },
	}

	cmd.Flags().IntVar(&cf.depth, "depth", 1, "Max crawl depth")
	//nolint:mnd // default crawl limit.
	cmd.Flags().IntVar(&cf.limit, "limit", 10, "Max pages to crawl")
	cmd.Flags().StringVar(&cf.output, "output", "./crawl-output", "Output directory")
	cmd.Flags().StringVar(&cf.format, "format", "markdown", "Output format: markdown, json")

	return cmd
}

func runCrawl(cmd *cobra.Command, args []string, cf *crawlFlags) error {
	client, err := buildClient()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cf.output, 0o750); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	pageNum := 0
	opts := scrapedo.CrawlOptions{
		MaxDepth: cf.depth,
		MaxPages: cf.limit,
		Format:   cf.format,
	}

	if err := client.Crawl(
		context.Background(), args[0], opts,
		func(r scrapedo.CrawlResult) {
			pageNum++
			handleCrawlResult(cmd, r, cf, pageNum, opts.MaxPages)
		},
	); err != nil {
		return fmt.Errorf("crawl failed: %w", err)
	}

	return nil
}

func handleCrawlResult(
	cmd *cobra.Command, r scrapedo.CrawlResult, cf *crawlFlags, pageNum, maxPages int,
) {
	if r.Error != nil {
		fmt.Printf("[%d/%d] %s → ERROR: %v\n", pageNum, maxPages, r.URL, r.Error)
		return
	}

	fmt.Printf("[%d/%d] %s → %s\n", pageNum, maxPages, r.URL, formatSize(r.Size))
	saveCrawlPage(r, cf)
	recordCrawlUsage(cmd, r.URL)
}

func saveCrawlPage(r scrapedo.CrawlResult, cf *crawlFlags) {
	filename := sanitizePath(r.URL) + ".md"
	path := filepath.Join(cf.output, filename)

	//nolint:gosec // output directory is user-specified
	if err := os.WriteFile(path, []byte(r.Content), 0o644); err != nil {
		fmt.Printf("  Warning: failed to save %s: %v\n", path, err)
	}
}

func recordCrawlUsage(cmd *cobra.Command, targetURL string) {
	if cacheStore != nil {
		//nolint:gosec // best-effort usage tracking
		_ = cacheStore.RecordUsage(
			cmd.Context(), "scrapedo", "", "crawl", "", targetURL, 1,
		)
	}
}

func formatSize(bytes int) string {
	const kb = 1024

	if bytes < kb {
		return fmt.Sprintf("%dB", bytes)
	}

	return fmt.Sprintf("%dKB", bytes/kb)
}
