// Package main provides the entry point for scrapedoctl.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/mr-pmillz/scrapedoctl/internal/cache"
)

func newHistoryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "history <url>",
		Short: "Show scrape history for a URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if cacheStore == nil {
				return errCacheNotInitialized
			}

			targetURL := args[0]
			history, err := cacheStore.GetHistory(context.Background(), targetURL)
			if err != nil {
				return fmt.Errorf("failed to fetch history: %w", err)
			}

			if len(history) == 0 {
				fmt.Printf("No history found for %s\n", targetURL)
				return nil
			}

			return printHistoryTable(history)
		},
	}
}

func printHistoryTable(history []cache.ScrapeRecord) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "ID\tDATE\tCOST\tCREDITS\tSTATUS"); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	for _, r := range history {
		var meta map[string]any
		if err := json.Unmarshal([]byte(r.Metadata), &meta); err != nil {
			// Fallback if metadata is invalid
			meta = make(map[string]any)
		}
		if _, err := fmt.Fprintf(w, "%d\t%s\t%v\t%v\t%v\n",
			r.ID,
			r.CreatedAt.Format("2006-01-02 15:04:05"),
			meta["cost"],
			meta["remaining_credits"],
			meta["status"],
		); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("failed to flush: %w", err)
	}
	return nil
}
