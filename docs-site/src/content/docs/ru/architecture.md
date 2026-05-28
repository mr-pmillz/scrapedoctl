---
title: Архитектура и дизайн
description: Обзор системы, слой хранения, интеграция с MCP и роутер мультипровайдерного поиска.
---

## Обзор системы

`scrapedoctl` построен как модульная система, разделяющая API-клиент, слой хранения и интерфейсы взаимодействия (CLI/MCP).

### Схема работы

```mermaid
graph TD
    User[User / AI Agent] --> CLI[CLI Entrypoint]
    CLI --> Config[Config Loader]
    Config --> Scrape{scrape}
    Config --> Search{search}
    Scrape --> Cache{Cache Check}
    Cache -- Hit --> Result[Return Result]
    Cache -- Miss --> API[Scrape.do API]
    API --> Save[Save to SQLite]
    Save --> Result
    Search --> Router[Search Router]
    Router --> ScrapeDoSearch[Scrape.do Search]
    Router --> SerpAPI[SerpAPI]
    Router --> ScraperAPI[ScraperAPI]
    Router --> ExecPlugin[Exec Plugin]
    ScrapeDoSearch --> SearchResult[Format & Output]
    SerpAPI --> SearchResult
    ScraperAPI --> SearchResult
    ExecPlugin --> SearchResult
```

## Model Context Protocol (MCP)

Реализация MCP позволяет любому совместимому клиенту (например, Claude Desktop или VS Code) использовать `scrapedoctl` как удалённый инструмент.

### Последовательность взаимодействия

```mermaid
sequenceDiagram
    participant Agent as AI Agent
    participant Server as MCP Server
    participant DB as SQLite Cache
    participant API as Scrape.do API

    Agent->>Server: listTools()
    Server-->>Agent: [scrape_url, web_search]
    Agent->>Server: callTool(url, render=true)
    Server->>DB: GetLatest(url_hash)
    alt Cache Found
        DB-->>Server: content
    else Cache Miss
        Server->>API: GET /scrape?url=...
        API-->>Server: 200 OK (Markdown)
        Server->>DB: InsertScrape(url, content)
    end
    Server-->>Agent: ToolResult(content)
```

## Слой хранения (SQLite)

Персистентный слой использует pure-Go реализацию SQLite (`modernc.org/sqlite`) в сочетании с `sqlc` для типобезопасного доступа к данным и `goose` для управления версиями миграций.

- **Нормализация запросов**: Все запросы нормализуются (сортировка параметров/заголовков) перед хешированием для обеспечения точности попадания в кэш.
- **Авто-очистка**: База данных самостоятельно управляет дисковым пространством на основе параметров конфигурации `keep_versions` и `max_size_mb`.

### Отслеживание использования

Каждая операция поиска и скрапинга записывается в таблицу `usage_log` для локальной аналитики. Схема таблицы:

| Колонка | Тип | Описание |
|---------|-----|----------|
| `id` | INTEGER | Автоинкрементный первичный ключ |
| `timestamp` | DATETIME | Время выполнения запроса |
| `provider` | TEXT | Имя провайдера (scrapedo, serpapi, scraperapi) |
| `engine` | TEXT | Используемый поисковый движок (google, bing и т.д.) |
| `action` | TEXT | Тип операции (search, scrape) |
| `query` | TEXT | Поисковый запрос или URL |
| `credits` | INTEGER | Кредиты, потреблённые запросом |

Эти данные используются командой `usage` в CLI и подкомандой `show usage` в REPL для предоставления разбивки по провайдеру, действию и временному диапазону.

## Архитектура провайдеров поиска

Подсистема поиска использует паттерн **Router** для маршрутизации запросов к наиболее подходящему провайдеру на основе поддержки движков и явного выбора провайдера.

### Разрешение провайдера

```mermaid
graph TD
    Query[search query + options] --> Router[Search Router]
    Router --> Resolve{Resolve Provider}
    Resolve -- "explicit provider flag" --> Direct[Use Named Provider]
    Resolve -- "by engine support" --> Match[First Provider Supporting Engine]
    Direct --> Execute[Provider.Search]
    Match --> Execute
    Execute --> Response[Unified Response]
    Response --> Format{Output Format}
    Format -- "table default" --> Table[Table]
    Format -- "json flag" --> JSON[JSON]
    Format -- "markdown flag" --> Markdown[Markdown]
```

### Встроенные провайдеры

| Провайдер | Движки | API-эндпоинт | Аутентификация |
|-----------|--------|-------------|----------------|
| Scrape.do | Google | `/plugin/google/search` | `global.token` (существующий) |
| ScraperAPI | Google | `api.scraperapi.com/structured/google/search` | `[providers.scraperapi].token` |
| SerpAPI | Google, Bing, Yandex, DuckDuckGo, Baidu, Yahoo, Naver | `serpapi.com/search` | `[providers.serpapi].token` |

### Протокол Exec-плагинов

Пользовательские провайдеры поиска реализуются как внешние исполняемые файлы. Плагин взаимодействует через JSON-протокол по stdin/stdout:

1. `scrapedoctl` записывает JSON-запрос в stdin плагина, содержащий поисковый запрос, движок и опции.
2. Плагин записывает JSON-ответ в stdout с результатами поиска.
3. Плагин завершается с кодом 0 в случае успеха или ненулевым кодом при ошибке.

Настройка exec-плагина в `conf.toml`:

```toml
[providers.my-plugin]
type    = "exec"
command = "/path/to/my-search-plugin"
engines = ["google", "custom-engine"]
```

### MCP-инструмент поиска

MCP-инструмент `web_search` предоставляет подсистему поиска AI-агентам. Когда настроен роутер поиска (доступен хотя бы один провайдер), MCP-сервер регистрирует инструмент `web_search` наряду с `scrape_url`. Агенты могут вызвать его с запросом и необязательным параметром движка. Если провайдеры поиска не настроены, инструмент не регистрируется.

```
Agent -> callTool("web_search", {query: "golang testing", engine: "google"})
Server -> Router.Resolve("google") -> Provider.Search(...)
Server -> Agent: ToolResult (markdown-formatted search results)
```

## CI/CD Pipeline

Проект использует современную CI/CD-конфигурацию:

- **golangci-lint v2.11.3** — комплексный линтинг Go с выводом в формате SARIF для загрузки в GitHub Code Scanning.
- **govulncheck** — проверка по базе уязвимостей Go при каждом запуске CI.
- **CodeQL** — семантический анализ кода GitHub для обнаружения уязвимостей безопасности.
- **UPX-сжатие бинарников** — релизные бинарники сжимаются с помощью UPX для уменьшения размера загрузки.
- **SARIF-загрузки** — результаты линтинга и анализа безопасности загружаются в формате SARIF для единой интеграции с вкладкой GitHub Security.
