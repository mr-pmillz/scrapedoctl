---
name: scrape-do-installation
description: |
  Install scrapedoctl and handle authentication.
---

# scrapedoctl Installation

## Quick Install

Download the latest release:

```bash
# Linux (amd64)
curl -fsSL https://github.com/mr-pmillz/scrapedoctl/releases/latest/download/scrapedoctl_Linux_x86_64.tar.gz | tar xz
sudo mv scrapedoctl /usr/local/bin/

# macOS (Apple Silicon)
curl -fsSL https://github.com/mr-pmillz/scrapedoctl/releases/latest/download/scrapedoctl_Darwin_arm64.tar.gz | tar xz
sudo mv scrapedoctl /usr/local/bin/
```

Or install from Go:

```bash
go install github.com/mr-pmillz/scrapedoctl/cmd/scrapedoctl@latest
```

## Setup

Run the interactive installer:

```bash
scrapedoctl install
```

This will:
1. Ask for your Scrape.do API token (get one at https://scrape.do/dashboard)
2. Configure AI agent integrations (Claude, Codex, Gemini, etc.)
3. Optionally generate project files (.mcp.json, CLAUDE.md, AGENTS.md, GEMINI.md)

## Verify

```bash
scrapedoctl version
scrapedoctl account
```

## Authentication Errors

If you see "SCRAPEDO_TOKEN required":

1. Set the environment variable: `export SCRAPEDO_TOKEN="your-token"`
2. Or run `scrapedoctl install` to configure

For search providers (SerpAPI, ScraperAPI), configure tokens in `~/.scrapedoctl/conf.toml`.

## Shell Completions

```bash
scrapedoctl completion install bash   # or zsh, fish
```
