package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/mr-pmillz/scrapedoctl/pkg/search"
)

func newAccountCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "account",
		Short: "Show usage, limits, and credits for all configured providers",
		RunE: func(_ *cobra.Command, _ []string) error {
			if searchRouter == nil {
				return errNoSearchProviders
			}

			infos := fetchAccountInfos(context.Background(), searchRouter)

			if jsonOutput {
				return printAccountJSON(infos)
			}

			printAccountTable(infos)

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func fetchAccountInfos(
	ctx context.Context, router *search.Router,
) []*search.AccountInfo {
	var infos []*search.AccountInfo

	for _, p := range router.Providers() {
		checker, ok := p.(search.AccountChecker)
		if !ok {
			continue
		}

		info, err := checker.Account(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", p.Name(), err)

			continue
		}

		infos = append(infos, info)
	}

	return infos
}

func printAccountJSON(infos []*search.AccountInfo) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	if err := enc.Encode(infos); err != nil {
		return fmt.Errorf("encode account info: %w", err)
	}

	return nil
}

func printAccountTable(infos []*search.AccountInfo) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "Provider\tPlan\tUsed\tLimit\tRemaining\tConcurrency")

	for _, info := range infos {
		plan := info.Plan
		if plan == "" {
			plan = "-"
		}

		conc := "-"
		if info.Concurrency > 0 {
			conc = strconv.Itoa(info.Concurrency)
		}

		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%s\n",
			info.Provider, plan,
			info.UsedRequests, info.MaxRequests, info.RemainingRequests,
			conc,
		)
	}

	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "flush: %v\n", err)
	}
}
