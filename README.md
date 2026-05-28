# scrapedoctl

<p align="center">
  <img src="https://img.shields.io/badge/Version-0.2.0-blue?style=for-the-badge" alt="Version 0.2.0">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.26">
  <a href="https://github.com/mr-pmillz/scrapedoctl/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/mr-pmillz/scrapedoctl/ci.yml?branch=main&style=for-the-badge&label=CI" alt="CI Status"></a>
  <a href="https://goreportcard.com/report/github.com/mr-pmillz/scrapedoctl"><img src="https://goreportcard.com/badge/github.com/mr-pmillz/scrapedoctl?style=for-the-badge" alt="Go Report Card"></a>
  <a href="https://github.com/mr-pmillz/scrapedoctl/actions/workflows/security.yml"><img src="https://img.shields.io/github/actions/workflow/status/mr-pmillz/scrapedoctl/security.yml?branch=main&style=for-the-badge&label=Security" alt="Security Status"></a>
  <a href="https://codecov.io/gh/mr-pmillz/scrapedoctl"><img src="https://img.shields.io/codecov/c/github/mr-pmillz/scrapedoctl?style=for-the-badge" alt="Codecov"></a>
  <a href="https://github.com/modelcontextprotocol"><img src="https://img.shields.io/badge/MCP-Protocol-1a73e8?style=for-the-badge" alt="MCP Protocol"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License"></a>
  <a href="https://pkg.go.dev/github.com/mr-pmillz/scrapedoctl"><img src="https://pkg.go.dev/badge/github.com/mr-pmillz/scrapedoctl.svg" alt="Go Reference"></a>
</p>

```text
  ____                                   _             _   _ 
 / ___|  ___ _ __ __ _ _ __   ___   __| | ___   ___| |_| |   
 \___ \ / __| '` / _` | '_ \ / _ \ / _` |/ _ \ / __| __| |   
  ___) | (__| | | (_| | |_) |  __/| (_| | (_) | (__| |_| |   
 |____/ \___|_|  \__,_| .__/ \___| \__,_|\___/ \___|\__|_|   
                      |_|                                    
```

**scrapedoctl** is a high-performance CLI utility and **Model Context Protocol (MCP)** server for [Scrape.do](https://scrape.do/). It is specifically architected to provide AI agents with a reliable, cost-effective, and powerful gateway to the live web.

---

## 🚀 Key Features

*   **🔍 Multi-Provider Search**: Search across Google, Bing, Yandex, DuckDuckGo, Baidu via Scrape.do, ScraperAPI, SerpAPI, or custom plugins.
*   **⚡ AI-First Design**: Native support for **LLM-optimized Markdown** output and MCP `web_search` tool.
*   **💾 Persistent Caching**: Built-in SQLite layer using `sqlc` to save tokens and maintain request history.
*   **📈 Account & Usage**: `account` command for provider usage/limits/credits; `usage` command for local analytics from SQLite.
*   **🤖 Agent Integration**: Interactive setup for **Claude Code, Gemini, Junie, Codex, Kimi, and OpenCode**.
*   **🛠️ Advanced Scraping**: Support for JS-rendering, residential proxies, geo-targeting, and browser actions.
*   **💻 Cisco-Style REPL**: Interactive shell with prefix matching, context help (`?`), and tab-completion.
*   **⌨️ Shell Completion**: `completion install` for Bash, Zsh, Fish (XDG-compliant, no .bashrc pollution).
*   **📊 Machine Metadata**: Dynamic tool discovery via JSON metadata and MCP resources.

---

## 📖 Documentation

Detailed guides are available in multiple languages:

*   🇬🇧 **[English Documentation](./docs/en/00-index.md)**
*   🇷🇺 **[Русская документация](./docs/ru/00-index.md)**

---

## 🛠️ Quick Start

### 1. Install
Download the latest binary from the [releases page](https://github.com/mr-pmillz/scrapedoctl/releases) or build from source:
```bash
go build -o bin/scrapedoctl ./cmd/scrapedoctl
```

### 2. Configure
Run any command to trigger the **interactive installer**:
```bash
scrapedoctl install
```

### 3. Search
```bash
# Google search (default)
scrapedoctl search "golang mcp sdk"

# Yandex via SerpAPI
scrapedoctl search "golang mcp sdk" --engine yandex

# JSON output
scrapedoctl search "golang mcp sdk" --json
```

### 4. Scrape
```bash
scrapedoctl scrape https://example.com --render
```

### 5. Interactive REPL
```bash
scrapedoctl repl
scrapedoctl> sh con              # show config (prefix matching)
scrapedoctl> se golang mcp       # search
scrapedoctl> show?               # context help
scrapedoctl> exit
```

### 6. Account & Usage
```bash
scrapedoctl account                   # Provider usage, limits, credits (table)
scrapedoctl account --json            # JSON output
scrapedoctl usage --week              # Local usage analytics (last 7 days)
scrapedoctl usage --month --json      # Last 30 days, JSON output
```

### 7. Shell Completions
```bash
scrapedoctl completion install bash   # ~/.local/share/bash-completion/completions/
scrapedoctl completion install zsh    # ~/.local/share/zsh/site-functions/
scrapedoctl completion install fish   # ~/.config/fish/completions/
```

---

## 🤖 AI Agent Setup (MCP)

`scrapedoctl` implements the Model Context Protocol. You can add it to your agent's configuration:

```json
{
  "mcpServers": {
    "scrape-do": {
      "command": "scrapedoctl",
      "args": ["mcp"]
    }
  }
}
```

---

## ⚖️ License

Distributed under the **MIT License**. See `LICENSE` for more information.

---

<p align="center">
  Built with ❤️ for the AI community.
</p>
