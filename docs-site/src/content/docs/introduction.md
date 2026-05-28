---
title: Introduction
description: What scrapedoctl is, why it exists, and its key features.
---

`scrapedoctl` is a professional-grade CLI tool and Model Context Protocol (MCP) server designed to bridge the gap between AI agents and the web. By utilizing the [Scrape.do](https://scrape.do/) API, it provides a robust, anti-bot-bypassing, and JS-rendering capable scraping engine that is both human and machine friendly.

## Project Goal

The primary goal of `scrapedoctl` is to provide AI agents (like Claude Code, Gemini CLI, etc.) with high-fidelity, LLM-optimized Markdown representations of web pages while minimizing costs through persistent caching and efficient request management.

## Key Features

- **Interactive REPL**: A Cisco-style shell with prefix matching, context help (`?`), and tab-completion.
- **Multi-Provider Web Search**: Search the web through Scrape.do, ScraperAPI, SerpAPI (7 engines), or custom exec plugins.
- **Persistent Caching**: Built-in SQLite storage to save tokens and maintain history.
- **Machine Interface**: Full MCP support (`scrape_url` and `web_search` tools) and JSON metadata for dynamic tool discovery.
- **Anti-Bot Bypassing**: Leveraging Scrape.do's proxy rotation and browser rendering.
- **Version & Update**: Built-in GitHub release checking with `scrapedoctl version` and self-update support.
- **Shell Completion Install**: Automatic XDG-compliant installation for Bash, Zsh, Fish, and PowerShell.
- **Modern Architecture**: Written in Go 1.26, zero-dependency core, and strict linting (golangci-lint v2).
- **PowerShell Module**: PSScriptAnalyzer-clean module with native binary discovery for cross-platform use.
