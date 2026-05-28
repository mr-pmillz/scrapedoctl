---
title: Руководство пользователя
description: Скрапинг, поиск, кэш, конфигурация провайдеров и интерактивный REPL.
---

## Основная команда Scrape

Команда `scrape` используется для разовых задач по скрапингу. Результат выводится прямо в терминал.

```bash
scrapedoctl scrape https://google.com --render --super
```

### Опции:
- `--render`: Включить рендеринг JavaScript (использует реальный браузер на серверах Scrape.do).
- `--super`: Использовать резидентные/мобильные прокси для обхода продвинутых защит.
- `--no-cache`: Игнорировать локальный кэш SQLite и выполнить новый запрос к API без сохранения.
- `--refresh`: Выполнить новый запрос к API и сохранить результат как новую версию в истории.

## Мультипровайдерный веб-поиск

Команда `search` позволяет выполнять запросы к нескольким поисковым системам через единый интерфейс. Результаты могут быть отформатированы в виде таблицы (по умолчанию), JSON или Markdown.

### Базовый поиск

```bash
scrapedoctl search "golang concurrency patterns"
```

### Поиск с параметрами

```bash
# Указать конкретный движок и язык
scrapedoctl search "kubernetes best practices" --engine bing --lang en --country us

# Ограничить количество результатов и получить JSON
scrapedoctl search "rust async" --limit 5 --json

# Получить результат в Markdown (удобно для передачи AI-агентам)
scrapedoctl search "python type hints" --markdown

# Явно указать провайдера
scrapedoctl search "web scraping" --provider serpapi --engine duckduckgo

# Постраничная навигация
scrapedoctl search "machine learning" --page 2

# Включить необработанный ответ провайдера для отладки
scrapedoctl search "test query" --raw --json
```

### Флаги поиска

| Флаг | Описание |
|------|----------|
| `--engine` | Поисковый движок (google, bing, yandex, duckduckgo, baidu, yahoo, naver) |
| `--provider` | Явно указать провайдера по имени |
| `--lang` | Код языка, например `en`, `de`, `ja` |
| `--country` | Код страны, например `us`, `gb`, `jp` |
| `--limit` | Максимальное количество результатов |
| `--page` | Номер страницы (по умолчанию: 1) |
| `--raw` | Включить необработанный ответ провайдера |
| `--json` | Вывод в формате JSON |
| `--markdown` | Вывод в формате Markdown |

### Доступные провайдеры

- **Scrape.do Google Search** — встроенный, использует существующий `global.token`. Дополнительная настройка не требуется.
- **ScraperAPI Google Search** — встроенный, требует токен ScraperAPI в `[providers.scraperapi]`.
- **SerpAPI** — поддерживает 7 движков (Google, Bing, Yandex, DuckDuckGo, Baidu, Yahoo, Naver). Требует токен SerpAPI в `[providers.serpapi]`.
- **Exec-плагины** — пользовательские провайдеры поиска, использующие JSON-протокол через stdin/stdout. Спецификация описана в разделе «Архитектура».

## Интерактивный REPL

Для сессий, включающих работу с несколькими URL, используйте встроенную оболочку:

```bash
scrapedoctl repl
scrapedoctl> scrape https://example.com render=true
```

### Дерево команд в стиле Cisco

REPL использует структуру команд, вдохновлённую Cisco IOS, с поддержкой сокращений команд. Полное имя команды вводить не нужно — достаточно любого однозначного префикса.

```
scrapedoctl> show config              # Полная конфигурация (токены замаскированы)
scrapedoctl> show config global.token # Конкретный ключ конфигурации
scrapedoctl> show cache               # Статистика кэша
scrapedoctl> show history <url>       # История скрапинга для URL
scrapedoctl> show version             # Информация о версии и проверка обновлений
scrapedoctl> set <key> <value>        # Установить значение конфигурации
scrapedoctl> clear cache              # Очистить персистентный кэш
scrapedoctl> search <query>           # Поиск в интернете
```

### Сокращения команд

Введите кратчайший однозначный префикс для любой команды или подкоманды:

```
scrapedoctl> sh con          # эквивалент: show config
scrapedoctl> cl ca           # эквивалент: clear cache
scrapedoctl> se golang       # эквивалент: search golang
```

### Контекстная помощь

Введите `?` в любой момент, чтобы увидеть доступные команды или подкоманды:

```
scrapedoctl> ?               # Список всех команд верхнего уровня
scrapedoctl> show ?          # Список всех подкоманд show
```

### Автодополнение по Tab

REPL предоставляет автодополнение по Tab для команд, подкоманд и параметров поиска.

## Персистентное кэширование и история

`scrapedoctl` автоматически сохраняет успешные запросы во внутреннюю базу данных SQLite (`~/.scrapedoctl/cache.db`).

### Просмотр истории:
```bash
scrapedoctl history https://example.com
```

### Обслуживание кэша:
```bash
scrapedoctl cache stats   # Показать размер БД и экономию
scrapedoctl cache clear   # Полностью очистить сохранённые результаты
```

## Управление конфигурацией

Вы можете управлять настройками через CLI или редактируя `~/.scrapedoctl/conf.toml`.

```bash
scrapedoctl config list
scrapedoctl config set global.timeout=30000
```

### Конфигурация провайдеров

Для использования нескольких провайдеров поиска добавьте секции `[search]` и `[providers.*]` в ваш `conf.toml`:

```toml
[search]
default_provider = "scrapedo"   # или "serpapi", "scraperapi", или пользовательское имя
default_engine   = "google"
default_limit    = 10

# Провайдер SerpAPI (поддерживает google, bing, yandex, duckduckgo, baidu, yahoo, naver)
[providers.serpapi]
token = "your-serpapi-key"

# Провайдер ScraperAPI
[providers.scraperapi]
token = "your-scraperapi-key"

# Пользовательский exec-плагин
[providers.my-custom-search]
type    = "exec"
command = "/usr/local/bin/my-search-plugin"
engines = ["google", "bing"]
```

Встроенный провайдер Scrape.do регистрируется автоматически при установленном `global.token`. Дополнительная запись в `[providers]` для него не требуется.

## Информация об аккаунте

Команда `account` получает данные об использовании, лимитах и оставшихся кредитах со всех настроенных провайдеров (Scrape.do, SerpAPI, ScraperAPI). Полезна для мониторинга потребления API.

```bash
# Табличный вывод (по умолчанию)
scrapedoctl account

# Вывод в формате JSON
scrapedoctl account --json
```

Команда запрашивает API аккаунта каждого провайдера и представляет единое представление с колонками: имя провайдера, использованные запросы, лимит запросов и оставшиеся кредиты.

В REPL используйте `show account`:

```
scrapedoctl> show account
```

## Аналитика использования

Команда `usage` отображает локальную аналитику использования из базы данных SQLite. Каждая операция поиска и скрапинга автоматически записывается в таблицу `usage_log` с указанием провайдера, движка, действия (search/scrape), запроса и потреблённых кредитов.

### Просмотр аналитики

```bash
# За последние 7 дней (по умолчанию)
scrapedoctl usage

# За последние 7 дней
scrapedoctl usage --week

# За последние 30 дней
scrapedoctl usage --month

# За всё время
scrapedoctl usage --all

# Вывод в формате JSON
scrapedoctl usage --month --json
```

Вывод группирует запросы по провайдеру и действию, давая наглядную картину использования инструмента.

В REPL используйте `show usage`:

```
scrapedoctl> show usage
```

## Версия и обновление

Проверка текущей версии и наличия обновлений:

```bash
scrapedoctl version
```

Команда выводит версию, Git-коммит и дату сборки, затем проверяет API релизов GitHub на наличие новой версии. Если обновление доступно, отображается ссылка на страницу релиза.
