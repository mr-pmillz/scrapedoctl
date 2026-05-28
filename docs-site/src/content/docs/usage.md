---
title: Usage Guide
description: Scrape, search, manage cache, configure providers, and run the interactive REPL.
---

## Core Scrape Command

The `scrape` command is used for one-off scraping tasks. It outputs the result directly to your terminal.

```bash
scrapedoctl scrape https://google.com --render --super
```

### Options:
- `--render`: Enable JavaScript rendering (uses a real browser on Scrape.do servers).
- `--super`: Use residential/mobile proxies to bypass advanced bot protection.
- `--no-cache`: Bypass the local SQLite cache and force a new API call without saving.
- `--refresh`: Force a new API call and store the result as a new version in history.

## Multi-Provider Web Search

The `search` command lets you query multiple search engines through a unified interface. Results can be formatted as a table (default), JSON, or Markdown.

### Basic Search

```bash
scrapedoctl search "golang concurrency patterns"
```

### Search with Options

```bash
# Use a specific engine and language
scrapedoctl search "kubernetes best practices" --engine bing --lang en --country us

# Limit results and get JSON output
scrapedoctl search "rust async" --limit 5 --json

# Get Markdown output (useful for piping to AI agents)
scrapedoctl search "python type hints" --markdown

# Force a specific provider
scrapedoctl search "web scraping" --provider serpapi --engine duckduckgo

# Page through results
scrapedoctl search "machine learning" --page 2

# Include raw provider response for debugging
scrapedoctl search "test query" --raw --json
```

### Search Flags

| Flag | Description |
|------|-------------|
| `--engine` | Search engine to use (google, bing, yandex, duckduckgo, baidu, yahoo, naver) |
| `--provider` | Force a specific provider by name |
| `--lang` | Language code, e.g. `en`, `de`, `ja` |
| `--country` | Country code, e.g. `us`, `gb`, `jp` |
| `--limit` | Maximum number of results to return |
| `--page` | Page number (default: 1) |
| `--raw` | Include the raw provider response in output |
| `--json` | Output as JSON |
| `--markdown` | Output as Markdown |

### Available Providers

- **Scrape.do Google Search** -- built-in, uses your existing `global.token`. No extra configuration needed.
- **ScraperAPI Google Search** -- built-in, requires a ScraperAPI token in `[providers.scraperapi]`.
- **SerpAPI** -- supports 7 engines (Google, Bing, Yandex, DuckDuckGo, Baidu, Yahoo, Naver). Requires a SerpAPI token in `[providers.serpapi]`.
- **Exec plugins** -- custom search providers using a stdin/stdout JSON protocol. See the Architecture section for the exec plugin specification.

## Interactive REPL

For sessions involving multiple URLs, use the built-in shell:

```bash
scrapedoctl repl
scrapedoctl> scrape https://example.com render=true
```

### Cisco-style Command Tree

The REPL uses a Cisco IOS-inspired command structure with prefix matching. You do not need to type full command names -- any unambiguous prefix works.

```
scrapedoctl> show config              # Full configuration (tokens are masked)
scrapedoctl> show config global.token # A specific configuration key
scrapedoctl> show cache               # Cache statistics
scrapedoctl> show history <url>       # Scrape history for a URL
scrapedoctl> show version             # Version info and update check
scrapedoctl> set <key> <value>        # Set a configuration value
scrapedoctl> clear cache              # Clear the persistent cache
scrapedoctl> search <query>           # Search the web
```

### Prefix Matching

Type the shortest unambiguous prefix for any command or subcommand:

```
scrapedoctl> sh con          # equivalent to: show config
scrapedoctl> cl ca           # equivalent to: clear cache
scrapedoctl> se golang       # equivalent to: search golang
```

### Context Help

Type `?` at any point to see available commands or subcommands:

```
scrapedoctl> ?               # Lists all top-level commands
scrapedoctl> show ?          # Lists all show subcommands
```

### Tab Completion

The REPL provides tab-completion for commands, subcommands, and search parameters.

## Persistent Cache & History

`scrapedoctl` automatically saves successful scrapes to an internal SQLite database (`~/.scrapedoctl/cache.db`).

### View History:
```bash
scrapedoctl history https://example.com
```

### Cache Maintenance:
```bash
scrapedoctl cache stats   # See DB size and savings
scrapedoctl cache clear   # Wipe all stored results
```

## Configuration Management

You can manage your settings via the CLI or by editing `~/.scrapedoctl/conf.toml`.

```bash
scrapedoctl config list
scrapedoctl config set global.timeout=30000
```

### Provider Configuration

To use multiple search providers, add `[search]` and `[providers.*]` sections to your `conf.toml`:

```toml
[search]
default_provider = "scrapedo"   # or "serpapi", "scraperapi", or a custom name
default_engine   = "google"
default_limit    = 10

# SerpAPI provider (supports google, bing, yandex, duckduckgo, baidu, yahoo, naver)
[providers.serpapi]
token = "your-serpapi-key"

# ScraperAPI provider
[providers.scraperapi]
token = "your-scraperapi-key"

# Custom exec plugin provider
[providers.my-custom-search]
type    = "exec"
command = "/usr/local/bin/my-search-plugin"
engines = ["google", "bing"]
```

The built-in Scrape.do provider is automatically registered when `global.token` is set. No additional `[providers]` entry is needed for it.

## Account Information

The `account` command retrieves usage, limits, and remaining credits from all configured providers (Scrape.do, SerpAPI, ScraperAPI). This is useful for monitoring API consumption across providers.

```bash
# Table output (default)
scrapedoctl account

# JSON output
scrapedoctl account --json
```

The command queries each provider's account/usage API and presents a unified view with columns for provider name, requests used, request limit, and remaining credits.

In the REPL, use `show account`:

```
scrapedoctl> show account
```

## Usage Analytics

The `usage` command displays local usage analytics collected from the SQLite database. Every search and scrape operation is automatically tracked in a `usage_log` table, recording the provider, engine, action (search/scrape), query, and credits consumed.

### Viewing Usage

```bash
# Last 7 days (default)
scrapedoctl usage

# Last 7 days
scrapedoctl usage --week

# Last 30 days
scrapedoctl usage --month

# All time
scrapedoctl usage --all

# JSON output
scrapedoctl usage --month --json
```

The output shows requests grouped by provider and action, giving you a clear picture of how you are using the tool.

In the REPL, use `show usage`:

```
scrapedoctl> show usage
```

## Version & Update

Check the current version and whether an update is available:

```bash
scrapedoctl version
```

This prints the version, Git commit, and build date, then checks the GitHub releases API for a newer version. If one is available, it displays a link to the release page.
