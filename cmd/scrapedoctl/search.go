package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mr-pmillz/scrapedoctl/pkg/search"
)

// errNoSearchProviders is returned when no search providers are configured.
var errNoSearchProviders = errors.New(
	"no search providers configured (set SCRAPEDO_TOKEN or configure providers in config)",
)

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search the web using multiple engines",
		Long: "Search using Scrape.do, SerpAPI, or custom providers. " +
			"Supports Google, Bing, Yandex, DuckDuckGo, and more.",
		Args: cobra.MinimumNArgs(1),
		RunE: runSearch,
	}

	cmd.Flags().String("engine", "", "Search engine (default from config)")
	cmd.Flags().String("provider", "", "Force specific provider")
	cmd.Flags().String("lang", "", "Language code (hl)")
	cmd.Flags().String("country", "", "Country code (gl)")
	cmd.Flags().Int("limit", 0, "Max results (default from config)")
	cmd.Flags().Int("page", 1, "Page number")
	cmd.Flags().Bool("raw", false, "Include raw provider response")
	cmd.Flags().Bool("json", false, "Output as JSON")
	cmd.Flags().Bool("markdown", false, "Output as markdown")

	return cmd
}

func runSearch(cmd *cobra.Command, args []string) error {
	if searchRouter == nil || len(searchRouter.Providers()) == 0 {
		return errNoSearchProviders
	}

	opts, err := parseSearchFlags(cmd)
	if err != nil {
		return err
	}

	p, err := searchRouter.Resolve(opts.Engine, cmd.Flag("provider").Value.String())
	if err != nil {
		return fmt.Errorf("resolve provider: %w", err)
	}

	query := strings.Join(args, " ")
	resp, err := p.Search(cmd.Context(), query, opts)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if cacheStore != nil {
		//nolint:gosec // best-effort usage tracking
		_ = cacheStore.RecordUsage(
			cmd.Context(), p.Name(), opts.Engine, "search", query, "", 1,
		)
	}

	return writeSearchOutput(cmd, resp)
}

func parseSearchFlags(cmd *cobra.Command) (search.Options, error) {
	f := cmd.Flags()

	engine, err := f.GetString("engine")
	if err != nil {
		return search.Options{}, fmt.Errorf("read engine flag: %w", err)
	}

	lang, err := f.GetString("lang")
	if err != nil {
		return search.Options{}, fmt.Errorf("read lang flag: %w", err)
	}

	country, err := f.GetString("country")
	if err != nil {
		return search.Options{}, fmt.Errorf("read country flag: %w", err)
	}

	limit, err := f.GetInt("limit")
	if err != nil {
		return search.Options{}, fmt.Errorf("read limit flag: %w", err)
	}

	page, err := f.GetInt("page")
	if err != nil {
		return search.Options{}, fmt.Errorf("read page flag: %w", err)
	}

	raw, err := f.GetBool("raw")
	if err != nil {
		return search.Options{}, fmt.Errorf("read raw flag: %w", err)
	}

	if engine == "" {
		engine = cfg.Search.DefaultEngine
	}
	if limit == 0 {
		limit = cfg.Search.DefaultLimit
	}

	return search.Options{
		Engine:  engine,
		Lang:    lang,
		Country: country,
		Limit:   limit,
		Page:    page,
		Raw:     raw,
	}, nil
}

func writeSearchOutput(cmd *cobra.Command, resp *search.Response) error {
	//nolint:gosec // flags always exist — registered in newSearchCmd
	jsonOut, _ := cmd.Flags().GetBool("json")
	//nolint:gosec // flags always exist — registered in newSearchCmd
	mdOut, _ := cmd.Flags().GetBool("markdown")

	w := cmd.OutOrStdout()

	var err error
	switch {
	case jsonOut:
		err = search.FormatJSON(w, resp)
	case mdOut:
		err = search.FormatMarkdown(w, resp)
	default:
		err = search.FormatTable(w, resp)
	}

	if err != nil {
		return fmt.Errorf("format output: %w", err)
	}

	return nil
}
