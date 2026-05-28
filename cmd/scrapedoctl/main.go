package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/mr-pmillz/scrapedoctl/internal/cache"
	"github.com/mr-pmillz/scrapedoctl/internal/config"
	"github.com/mr-pmillz/scrapedoctl/internal/logger"
	"github.com/mr-pmillz/scrapedoctl/internal/mcp"
	"github.com/mr-pmillz/scrapedoctl/internal/repl"
	"github.com/mr-pmillz/scrapedoctl/internal/ui"
	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
	"github.com/mr-pmillz/scrapedoctl/pkg/search"
)

var (
	// cfg is the loaded application configuration.
	cfg *config.Config
	// cacheStore is the persistent caching layer.
	cacheStore *cache.Store
	// searchRouter is the multi-provider search router.
	searchRouter *search.Router
	// configPath is the path to the configuration file.
	configPath string
	// profileName is the name of the profile to use.
	profileName string
)

// errMissingToken is returned when the SCRAPEDO_TOKEN environment variable is missing.
var errMissingToken = errors.New("SCRAPEDO_TOKEN environment variable is required (or set it in config file)")

// errInstallCommand is returned when the install command cannot be found or executed.
var errInstallCommand = errors.New("failed to find or execute install command")

// errCacheNotInitialized is returned when the cache is accessed but not initialized.
var errCacheNotInitialized = errors.New("cache is disabled or not initialized")

// errInvalidConfigFormat is returned when a configuration setting is not in key=value format.
var errInvalidConfigFormat = errors.New("invalid format, use key=value")

// errUnsupportedConfigKey is returned when an unknown configuration key is provided.
var errUnsupportedConfigKey = errors.New("unknown or unsupported key")

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if err := newRootCmd().Execute(); err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}
	return nil
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "scrapedoctl",
		Short:             "scrapedoctl is a CLI and MCP server for Scrape.do",
		PersistentPreRunE: handlePersistentPreRun,
	}

	ui.SetCustomHelp(cmd)

	cmd.PersistentFlags().StringVarP(&configPath, "config", "c", config.DefaultConfigPath, "path to config file")
	cmd.PersistentFlags().StringVarP(&profileName, "profile", "p", "", "use a specific configuration profile")

	cmd.AddCommand(newMCPCmd())
	cmd.AddCommand(newREPLCmd())
	cmd.AddCommand(newScrapeCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newMetadataCmd())
	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newHistoryCmd())
	cmd.AddCommand(newCacheCmd())
	cmd.AddCommand(newAccountCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newUsageCmd())
	cmd.AddCommand(newMapCmd())
	cmd.AddCommand(newCrawlCmd())
	cmd.AddCommand(newProviderCmd())
	cmd.AddCommand(newCompletionCmd(cmd))

	// Disable Cobra's auto-generated completion command (we have our own).
	cmd.CompletionOptions.DisableDefaultCmd = true

	return cmd
}

func handlePersistentPreRun(cmd *cobra.Command, _ []string) error {
	var err error
	cfg, err = config.Load(configPath, profileName)

	if errors.Is(err, config.ErrConfigNotFound) {
		if isBypassCommand(cmd) {
			return nil
		}
		return triggerInitialSetup(cmd)
	}

	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg != nil {
		logger.Init(cfg.Logging)
		initCache()
		searchRouter = initSearchRouter(cfg)
	}
	return nil
}

func triggerInitialSetup(cmd *cobra.Command) error {
	fmt.Println("No configuration file found. Starting initial setup...")

	// Initialize logger with defaults from the populated cfg
	logger.Init(cfg.Logging)

	// Find the install command and execute it
	installCmd, _, findErr := cmd.Root().Find([]string{"install"})
	if findErr != nil || installCmd == nil || installCmd.RunE == nil {
		return errInstallCommand
	}

	if runErr := installCmd.RunE(installCmd, nil); runErr != nil {
		return fmt.Errorf("install failed: %w", runErr)
	}

	fmt.Println("\nSetup complete. Please run the command again.")
	os.Exit(0)
	return nil
}

func initSearchRouter(c *config.Config) *search.Router {
	router := search.NewRouter()

	// Always register scrapedo if global token exists.
	if token := c.Global.Token; token != "" {
		router.Register(search.NewScrapedoProvider(token))
	}

	// Register configured providers.
	for name, pcfg := range c.Providers {
		switch {
		case pcfg.Type == "exec" && pcfg.Command != "":
			p := search.NewExecProvider(name, pcfg.Command, pcfg.Engines)
			if len(pcfg.Args) > 0 {
				p.WithArgs(pcfg.Args...)
			}
			router.Register(p)
		case name == "serpapi" && pcfg.Token != "":
			router.Register(search.NewSerpAPIProvider(pcfg.Token))
		case name == "scraperapi" && pcfg.Token != "":
			router.Register(search.NewScraperAPIProvider(pcfg.Token))
		}
	}

	return router
}

func initCache() {
	if cfg.Cache.Enabled {
		var err error
		cacheStore, err = cache.NewStore(cfg.Cache)
		if err != nil {
			slog.Warn("Failed to initialize cache", slog.Any("error", err))
		}
	}
}

func isBypassCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "help", "metadata", "install", "completion":
			return true
		}
	}
	return false
}

type scrapeFlags struct {
	render, super, noCache, refresh bool
}

func newScrapeCmd() *cobra.Command {
	sf := &scrapeFlags{}
	cmd := &cobra.Command{
		Use:   "scrape <url>",
		Short: "Scrape a single URL and output markdown",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return runScrape(cmd, args, sf) },
	}

	cmd.Flags().BoolVar(&sf.render, "render", false, "Execute JavaScript")
	cmd.Flags().BoolVar(&sf.super, "super", false, "Use residential proxy")
	cmd.Flags().BoolVar(&sf.noCache, "no-cache", false, "Bypass persistent cache")
	cmd.Flags().BoolVar(&sf.refresh, "refresh", false, "Force API call and update cache")

	return cmd
}

func runScrape(cmd *cobra.Command, args []string, sf *scrapeFlags) error {
	token := cfg.Global.Token
	if token == "" {
		token = os.Getenv("SCRAPEDO_TOKEN")
	}
	if token == "" {
		return errMissingToken
	}

	client, err := scrapedo.NewClient(token)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	if cacheStore != nil {
		client.SetCache(cacheStore)
	}

	req := buildScrapeRequest(cmd, args[0], sf)

	result, err := client.Scrape(context.Background(), req)
	if err != nil {
		return fmt.Errorf("scrape failed: %w", err)
	}

	if cacheStore != nil {
		//nolint:gosec // best-effort usage tracking
		_ = cacheStore.RecordUsage(
			cmd.Context(), "scrapedo", "", "scrape", "", args[0], 1,
		)
	}

	fmt.Println(result)
	return nil
}

func buildScrapeRequest(cmd *cobra.Command, url string, sf *scrapeFlags) scrapedo.ScrapeRequest {
	req := scrapedo.ScrapeRequest{
		URL:     url,
		Render:  cfg.Resolved.Render,
		Super:   cfg.Resolved.Super,
		GeoCode: cfg.Resolved.GeoCode,
		Session: cfg.Resolved.Session,
		Device:  cfg.Resolved.Device,
		NoCache: sf.noCache,
		Refresh: sf.refresh,
	}

	if cmd.Flags().Changed("render") {
		req.Render = sf.render
	}
	if cmd.Flags().Changed("super") {
		req.Super = sf.super
	}

	return req
}

func newREPLCmd() *cobra.Command {
	var simple bool

	cmd := &cobra.Command{
		Use:   "repl",
		Short: "Start an interactive Scrape.do shell",
		RunE: func(_ *cobra.Command, _ []string) error {
			if simple {
				_ = os.Setenv("SCRAPEDOCTL_SIMPLE_REPL", "1") //nolint:gosec // best-effort.
			}

			return runREPL()
		},
	}

	cmd.Flags().BoolVar(&simple, "simple", false,
		"Use simple line reader (no tab-completion, fixes prompt issues)")

	return cmd
}

func runREPL() error {
	token := cfg.Global.Token
	if token == "" {
		token = os.Getenv("SCRAPEDO_TOKEN")
	}

	if token == "" {
		return errMissingToken
	}

	client, err := scrapedo.NewClient(token)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	if cacheStore != nil {
		client.SetCache(cacheStore)
	}

	opts := buildREPLOpts()
	shell := repl.NewShell(client, opts...)

	if err := shell.Run(context.Background()); err != nil {
		return fmt.Errorf("repl: %w", err)
	}

	return nil
}

func buildREPLOpts() []repl.ShellOption {
	var opts []repl.ShellOption
	if searchRouter != nil {
		opts = append(opts, repl.WithSearchRouter(searchRouter))
	}

	if cacheStore != nil {
		opts = append(opts, repl.WithCache(cacheStore))
	}

	if cfg != nil {
		opts = append(opts, repl.WithConfig(cfg))
	}

	return opts
}

func newMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start the MCP server over stdio",
		RunE: func(_ *cobra.Command, _ []string) error {
			token := cfg.Global.Token
			if token == "" {
				token = os.Getenv("SCRAPEDO_TOKEN")
			}
			if token == "" {
				return errMissingToken
			}

			client, err := scrapedo.NewClient(token)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}
			if cacheStore != nil {
				client.SetCache(cacheStore)
			}

			// We use context.Background() since the MCP server handles its own lifecycle
			// over stdio and will exit when the stream closes.
			var opts []mcp.ServerOption
			if searchRouter != nil {
				opts = append(opts, mcp.WithRouter(searchRouter))
			}
			if cacheStore != nil {
				opts = append(opts, mcp.WithUsageRecorder(cacheStore))
			}
			return mcp.RunServerWithOpts(context.Background(), client, opts...)
		},
	}
}
