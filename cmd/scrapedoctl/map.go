package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

type mapFlags struct {
	search string
	limit  int
	asJSON bool
}

func newMapCmd() *cobra.Command {
	mf := &mapFlags{}
	cmd := &cobra.Command{
		Use:   "map <url>",
		Short: "Discover all same-domain URLs on a page",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return runMap(cmd, args, mf) },
	}

	cmd.Flags().StringVar(&mf.search, "search", "", "Filter URLs by keyword")
	cmd.Flags().IntVar(&mf.limit, "limit", 100, "Max URLs to return") //nolint:mnd // default limit
	cmd.Flags().BoolVar(&mf.asJSON, "json", false, "Output as JSON array")

	return cmd
}

func runMap(cmd *cobra.Command, args []string, mf *mapFlags) error {
	client, err := buildClient()
	if err != nil {
		return err
	}

	content, err := client.Scrape(context.Background(), scrapedo.ScrapeRequest{URL: args[0]})
	if err != nil {
		return fmt.Errorf("scrape failed: %w", err)
	}

	recordMapUsage(cmd, args[0])

	links := scrapedo.ExtractLinks(content, args[0])
	links = filterAndLimitLinks(links, mf)

	return outputLinks(links, args[0], mf)
}

func filterAndLimitLinks(links []string, mf *mapFlags) []string {
	if mf.search != "" {
		filtered := links[:0]
		needle := strings.ToLower(mf.search)

		for _, l := range links {
			if strings.Contains(strings.ToLower(l), needle) {
				filtered = append(filtered, l)
			}
		}

		links = filtered
	}

	if mf.limit > 0 && len(links) > mf.limit {
		links = links[:mf.limit]
	}

	return links
}

func outputLinks(links []string, rawURL string, mf *mapFlags) error {
	if mf.asJSON {
		return outputLinksJSON(links)
	}

	host := extractHost(rawURL)
	fmt.Printf("Discovered %d URLs on %s:\n\n", len(links), host)

	for i, l := range links {
		fmt.Printf("%3d  %s\n", i+1, l)
	}

	return nil
}

func outputLinksJSON(links []string) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	if err := enc.Encode(links); err != nil {
		return fmt.Errorf("json encode failed: %w", err)
	}

	return nil
}

func recordMapUsage(cmd *cobra.Command, targetURL string) {
	if cacheStore != nil {
		//nolint:gosec // best-effort usage tracking
		_ = cacheStore.RecordUsage(
			cmd.Context(), "scrapedo", "", "map", "", targetURL, 1,
		)
	}
}
