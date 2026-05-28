package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/mr-pmillz/scrapedoctl/internal/cache"
)

func newUsageCmd() *cobra.Command {
	var week, month, all, jsonOut bool

	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Show API usage statistics",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cacheStore == nil {
				return errCacheNotInitialized
			}

			since := usageSince(week, month, all)
			return printUsage(cmd, cacheStore, since, jsonOut)
		},
	}

	cmd.Flags().BoolVar(&week, "week", false, "Last 7 days")
	cmd.Flags().BoolVar(&month, "month", false, "Last 30 days")
	cmd.Flags().BoolVar(&all, "all", false, "All time")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	return cmd
}

func usageSince(week, month, all bool) time.Time {
	now := time.Now()

	switch {
	case all:
		return time.Time{}
	case month:
		return now.AddDate(0, 0, -30) //nolint:mnd // 30 days
	case week:
		return now.AddDate(0, 0, -7) //nolint:mnd // 7 days
	default:
		y, m, d := now.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	}
}

func printUsage(
	cmd *cobra.Command, store *cache.Store, since time.Time, jsonOut bool,
) error {
	ctx := cmd.Context()

	summary, err := store.GetUsageSummary(ctx, since)
	if err != nil {
		return fmt.Errorf("usage query failed: %w", err)
	}

	if jsonOut {
		return printUsageJSON(cmd, summary, since)
	}

	return printUsageTable(cmd, summary, since)
}

type usageJSON struct {
	Since string              `json:"since"`
	Rows  []usageJSONRow      `json:"rows"`
	Total usageJSONTotalEntry `json:"total"`
}

type usageJSONRow struct {
	Provider string `json:"provider"`
	Action   string `json:"action"`
	Count    int64  `json:"count"`
	Credits  int64  `json:"credits"`
}

type usageJSONTotalEntry struct {
	Count   int64 `json:"count"`
	Credits int64 `json:"credits"`
}

func printUsageJSON(cmd *cobra.Command, summary []cache.UsageSummary, since time.Time) error {
	var totalCount, totalCredits int64
	rows := make([]usageJSONRow, 0, len(summary))

	for _, s := range summary {
		totalCount += s.Count
		totalCredits += s.TotalCredits
		rows = append(rows, usageJSONRow{
			Provider: s.Provider, Action: s.Action,
			Count: s.Count, Credits: s.TotalCredits,
		})
	}

	sinceStr := "all time"
	if !since.IsZero() {
		sinceStr = since.Format("2006-01-02")
	}

	out := usageJSON{
		Since: sinceStr, Rows: rows,
		Total: usageJSONTotalEntry{Count: totalCount, Credits: totalCredits},
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal usage: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

func printUsageTable(cmd *cobra.Command, summary []cache.UsageSummary, since time.Time) error {
	w := cmd.OutOrStdout()

	sinceStr := "all time"
	if !since.IsZero() {
		sinceStr = since.Format("2006-01-02")
	}

	fmt.Fprintf(w, "Usage since %s:\n\n", sinceStr)
	fmt.Fprintf(w, "%-13s%-9s%-8s%s\n", "Provider", "Action", "Count", "Credits")

	var totalCount, totalCredits int64
	for _, s := range summary {
		totalCount += s.Count
		totalCredits += s.TotalCredits
		fmt.Fprintf(w, "%-13s%-9s%-8d%d\n", s.Provider, s.Action, s.Count, s.TotalCredits)
	}

	fmt.Fprintf(w, "\nTotal: %d requests, %d credits\n", totalCount, totalCredits)
	return nil
}
