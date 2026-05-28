# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **SDK `Client.Info()`**: New `pkg/scrapedo.Client.Info(ctx) (*AccountInfo, error)` method that calls Scrape.do's `GET /info` endpoint and returns the account's active concurrency cap (`ConcurrentRequest`), monthly request quota (`MaxMonthlyRequest`), and remaining counters. Enables consumers to size per-run fan-out dynamically against the live account limit and to short-circuit batches when the monthly quota is exhausted. Non-200 responses wrap `ErrAPI` so callers can branch with `errors.Is`.
- **SDK `AccountInfo`**: Exported struct mirroring the `/info` JSON shape exactly (no struct tags) — `IsActive`, `ConcurrentRequest`, `MaxMonthlyRequest`, `RemainingConcurrentRequest`, `RemainingMonthlyRequest`.

## [0.2.0] - 2026-03-21

### Added
- **Account Command**: `scrapedoctl account` shows usage, limits, and credits for all configured providers (Scrape.do, SerpAPI, ScraperAPI). Supports table and `--json` output. Also available as `show account` in REPL.
- **Usage Command**: `scrapedoctl usage` displays local usage analytics from the SQLite database. Shows requests grouped by provider and action with `--week`, `--month`, `--all`, and `--json` flags. Also available as `show usage` in REPL.
- **Usage Tracking**: Every search and scrape operation is automatically recorded in a `usage_log` SQLite table with provider, engine, action, query, and credits fields.
- **Install Init**: `scrapedoctl install init` non-interactive subcommand that generates `.mcp.json`, `CLAUDE.md`, `AGENTS.md`, and `GEMINI.md` project files for AI agent integration.
- **ScraperAPI Provider**: 4th built-in search provider using Google Search via `api.scraperapi.com/structured`.
- **Multi-Provider Web Search**: New `search` command with pluggable provider architecture.
  - Built-in providers: Scrape.do (Google), ScraperAPI (Google), SerpAPI (Google, Bing, Yandex, DuckDuckGo, Baidu, Yahoo, Naver).
  - Exec plugin system for custom search providers via stdin/stdout JSON protocol.
  - Output formats: table (default), JSON (`--json`), Markdown (`--markdown`).
  - Flags: `--engine`, `--provider`, `--lang`, `--country`, `--limit`, `--page`, `--raw`.
  - Provider routing: automatic engine-to-provider resolution via config.
- **MCP Search Tool**: `web_search` tool exposed to AI agents via Model Context Protocol.
- **Cisco-Style REPL**: Major interactive shell upgrade.
  - Command tree: `show config/cache/history/version`, `set <key> <value>`, `clear cache`.
  - Prefix matching: `sh con` resolves to `show config`, `se query` to `search query`.
  - Context help: type `?` after any partial command to see available options.
  - Tab-completion for commands, subcommands, and search parameters (engine, provider).
  - Token masking in `show config` output.
  - Cisco-style error format: `% error message`.
- **Version Command**: `scrapedoctl version` shows version, commit, build date, Go version, and checks for updates via GitHub API.
- **Update Command**: `scrapedoctl update` checks for new releases and prints platform-specific install instructions.
- **Completion Install**: `scrapedoctl completion install bash/zsh/fish` installs completions to XDG-compliant directories automatically (no .bashrc modification needed).
- **Provider Configuration**: New `[search]` and `[providers.*]` sections in `conf.toml` for multi-provider setup.

### Changed
- **CI Pipeline**: Migrated to `golang:1.26-trixie` container with golangci-lint v2.11.3.
- **CI Security**: gosec now outputs SARIF (non-blocking), added CodeQL analysis with security-and-quality queries.
- **CI Build**: Added UPX binary compression (`--best --lzma`), binary uploaded as artifact.
- **CI Checks**: Added govulncheck, osv-scanner, and Trivy SARIF upload to GitHub Security tab.
- **PowerShell Module**: Fixed `Test-Path` with `-LiteralPath`, `Get-Command` with `-CommandType Application`, PSScriptAnalyzer clean.
- **Containerfile**: Replaced `segmentio/golines` with `golangci/golines`, pinned golangci-lint v2.11.3.

### Fixed
- **Lint**: Resolved all 14 golangci-lint warnings (goimports, gosec G104, revive unused-parameter, testifylint).
- **Test Coverage**: Raised `cmd/scrapedoctl` from 58% to 76%, `internal/mcp` from 73% to 84%.
- **REPL**: Fixed `errInvalidUsage` showing scrape usage text for all commands.
- **Exec Tests**: Eliminated "text file busy" race condition in exec plugin tests.

## [0.1.0] - 2026-03-21

### Added
- **Core MCP Server**: Go 1.26 Model Context Protocol server for Scrape.do with `scrape` tool.
- **Persistent Caching**: Integrated SQLite with `sqlc` and `goose` for saving Scrape.do results.
- **History Tracking**: `history` command to view and compare different versions of scraped pages.
- **Cache Management**: `cache stats` and `cache clear` commands.
- **Interactive Installer**: `install` command using `charmbracelet/huh` for multi-agent setup (Claude, Junie, Gemini, Codex, Kimi, OpenCode).
- **First-Run Setup**: Automatic trigger of the installer if no configuration is found.
- **Config Management CLI**: `config set/list` commands to manage settings from the terminal.
- **Advanced Configuration**: Integrated `koanf` for multi-layer configuration (defaults, file, env, flags).
- **Profiles Support**: Named profiles in `conf.toml`.
- **Stylized UI**: Professional ASCII banner and colorized help output using `lipgloss`.
- **Interactive REPL**: `repl` command using `reeflective/readline` for manual scraping sessions.
- **Machine Interface**: `metadata` command (JSON) and MCP Resource `resource://cli/help` for automated discovery.
- **Enhanced Scrape.do Support**: Parameters for `geoCode`, `session`, `device`, `method`, `headers`, `body`, and `actions` (playWithBrowser).
- **Advanced Logging**: Structured JSON logging with `slog` and rotation via `lumberjack`.
- **Shell Completions**: Bash, Zsh, Fish, and PowerShell completion scripts.
- **PowerShell Module**: Native `Invoke-Scrapedoctl` wrapper with argument completer for PS 7.4+.
- **CI/CD**: GitHub Actions workflows for linting, testing, security scanning, and automated releases via GoReleaser.
- **Packaging**: `.deb`, `.rpm` packages via GoReleaser nfpms, UPX binary compression.
- **Multi-Language Docs**: English and Russian documentation with Mermaid diagrams.
- **AI Discovery**: `llms.txt` for LLM tool discovery.
