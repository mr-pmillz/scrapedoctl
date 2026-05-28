---
title: Go SDK
description: Использование scrapedoctl как Go SDK — GET с super=true, сырой HTML, конкурентный скрапинг, повтор, кэширование и многое другое.
---

Пакет `github.com/mr-pmillz/scrapedoctl/pkg/scrapedo` — это тот же клиент, на котором построены CLI и MCP-сервер `scrapedoctl`. Его можно импортировать из любого Go-модуля, чтобы добавить в своё приложение скрапинг через Scrape.do, обход блокировок с помощью резидентных прокси и (опционально) SQLite-кэш.

## Установка

В вашем модуле:

```bash
go get github.com/mr-pmillz/scrapedoctl/pkg/scrapedo@latest
```

У публичного API нет внешних зависимостей — используется только стандартная библиотека. Кэширование подключается опционально через небольшой интерфейс `Cacher` (см. раздел [Персистентный кэш](#персистентный-кэш-опционально) ниже).

## GET-запрос с `super=true`

Минимальная программа ниже скрапит URL через пул резидентных и мобильных прокси Scrape.do. Установка `Super: true` в структуре запроса эквивалентна параметру `super=true` в HTTP-запросе к API. Поле `Method` по умолчанию равно `GET`, поэтому отдельно указывать его не нужно.

```go
// Файл: main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

func main() {
	token := os.Getenv("SCRAPEDO_TOKEN")
	if token == "" {
		log.Fatal("SCRAPEDO_TOKEN не задан")
	}

	client, err := scrapedo.NewClient(token)
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	markdown, err := client.Scrape(ctx, scrapedo.ScrapeRequest{
		URL:   "https://example.com",
		Super: true, // маршрутизация через резидентные/мобильные прокси (super=true)
	})
	if err != nil {
		log.Fatalf("scrape: %v", err)
	}

	fmt.Println(markdown)
}
```

Запуск:

```bash
export SCRAPEDO_TOKEN="ваш-токен-scrape-do"
go run .
```

По умолчанию клиент запрашивает Markdown (всегда добавляет `output=markdown`), что удобно для передачи в LLM. О том, как получить сырой HTML, см. раздел [Получить сырой HTML](#получить-сырой-html-вместо-markdown).

## Как `Super` отображается в API-запрос

При сборке запроса `Super: true` превращается в параметр `super=true` исходящего HTTPS-запроса к `http://api.scrape.do`. Итоговый запрос выглядит примерно так:

```
GET http://api.scrape.do/?token=***&url=https%3A%2F%2Fexample.com&output=markdown&super=true
```

`Super` стоит использовать на сайтах, которые блокируют дата-центровые IP или агрессивно лимитируют запросы. Он расходует больше кредитов на запрос, чем обычный пул, поэтому включайте его только при необходимости.

## Справочник `ScrapeRequest`

```go
type ScrapeRequest struct {
    URL     string            // обязательное поле
    Render  bool              // рендеринг через headless-браузер (render=true)
    Super   bool              // резидентные/мобильные прокси (super=true)
    GeoCode string            // двухбуквенный код страны (например, "us", "gb")
    Session string            // sticky-сессия (тот же IP прокси)
    Device  string            // "desktop" (по умолчанию), "mobile", "tablet"
    Method  string            // HTTP-метод; по умолчанию "GET"
    Output  string            // "markdown" (по умолчанию) или "raw" для сырого HTML
    Headers map[string]string // пересылаются с customHeaders=true
    Body    []byte            // тело для POST/PUT-запросов
    Actions []any             // действия playWithBrowser при Render=true

    NoCache bool              // обойти локальный кэш и не сохранять результат
    Refresh bool              // принудительно сделать новый запрос и сохранить как новую версию
}
```

Обязательно только поле `URL`. Любое пустое поле опускается в исходящем запросе (кроме `Output`, который по умолчанию равен `"markdown"`).

## Примеры

### Получить сырой HTML вместо Markdown

Установка `Output: "raw"` отключает конвертацию в Markdown и возвращает страницу как есть. Это то, что нужно, когда вы хотите передать ответ в HTML-парсер (`goquery`, `golang.org/x/net/html`) или когда важна структура страницы, а не её текст.

```go
html, err := client.Scrape(ctx, scrapedo.ScrapeRequest{
    URL:    "https://example.com",
    Super:  true,
    Output: "raw", // возвращает немодифицированный HTML; исходящий параметр: output=raw
})
if err != nil {
    log.Fatalf("scrape: %v", err)
}

fmt.Println(html[:200])
```

Передайте ответ напрямую в `goquery`, чтобы обойти DOM:

```go
import "github.com/PuerkitoBio/goquery"

doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
if err != nil {
    log.Fatalf("parse html: %v", err)
}

doc.Find("article h2").Each(func(_ int, s *goquery.Selection) {
    fmt.Println(strings.TrimSpace(s.Text()))
})
```

### GET с гео-таргетингом

```go
markdown, err := client.Scrape(ctx, scrapedo.ScrapeRequest{
    URL:     "https://example.com/pricing",
    Super:   true,
    GeoCode: "gb",
})
```

### JS-рендеринг со sticky-сессией

```go
markdown, err := client.Scrape(ctx, scrapedo.ScrapeRequest{
    URL:     "https://shop.example.com/cart",
    Render:  true,
    Super:   true,
    Session: "user-42",
})
```

### POST с JSON-телом и пользовательскими заголовками

```go
body := []byte(`{"query":"laptops"}`)
markdown, err := client.Scrape(ctx, scrapedo.ScrapeRequest{
    URL:    "https://api.example.com/search",
    Method: "POST",
    Body:   body,
    Headers: map[string]string{
        "Content-Type": "application/json",
        "Accept":       "application/json",
    },
})
```

### Извлечение ссылок того же домена

```go
links := scrapedo.ExtractLinks(markdown, "https://example.com")
for _, l := range links {
    fmt.Println(l)
}
```

### Рекурсивный BFS-обход

```go
err := client.Crawl(ctx, "https://example.com",
    scrapedo.CrawlOptions{MaxDepth: 2, MaxPages: 25},
    func(r scrapedo.CrawlResult) {
        if r.Error != nil {
            log.Printf("%s: %v", r.URL, r.Error)
            return
        }
        fmt.Printf("[%d] %s (%d байт, %d ссылок)\n", r.Depth, r.URL, r.Size, len(r.Links))
    },
)
```

### Конкурентный массовый скрапинг

Распараллельте обход списка URL через небольшой пул воркеров. `golang.org/x/sync/errgroup` даёт привычную семантику «остановка при первой ошибке» и встроенный лимит конкурентности.

```go
import "golang.org/x/sync/errgroup"

func scrapeAll(ctx context.Context, client *scrapedo.Client, urls []string) (map[string]string, error) {
    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(5) // ограничение одновременных запросов

    var (
        mu      sync.Mutex
        results = make(map[string]string, len(urls))
    )

    for _, u := range urls {
        u := u // захват переменной перед горутиной
        g.Go(func() error {
            md, err := client.Scrape(ctx, scrapedo.ScrapeRequest{URL: u, Super: true})
            if err != nil {
                return fmt.Errorf("%s: %w", u, err)
            }
            mu.Lock()
            results[u] = md
            mu.Unlock()
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return results, err
    }
    return results, nil
}
```

Производный от `errgroup` контекст `ctx` отменяется сразу же, как только любая горутина вернёт ошибку, поэтому остальные перестанут тянуть новые страницы вместо гонки до завершения.

### Повтор при временных ошибках с backoff

`scrapedo.ErrAPI` оборачивает ответы со статусом, отличным от 200. Простая экспоненциальная пауза покрывает временные сбои прокси и 5xx-ответы апстрима без дополнительных зависимостей:

```go
func scrapeWithRetry(
    ctx context.Context, client *scrapedo.Client, req scrapedo.ScrapeRequest, attempts int,
) (string, error) {
    var lastErr error
    delay := 500 * time.Millisecond
    for i := 0; i < attempts; i++ {
        body, err := client.Scrape(ctx, req)
        if err == nil {
            return body, nil
        }
        lastErr = err

        // Повторяем только временные API-ошибки — на ошибках валидации сразу выходим.
        if !errors.Is(err, scrapedo.ErrAPI) {
            return "", err
        }

        select {
        case <-ctx.Done():
            return "", ctx.Err()
        case <-time.After(delay):
        }
        delay *= 2
    }
    return "", fmt.Errorf("сдались после %d попыток: %w", attempts, lastErr)
}
```

Используйте вместе с `Refresh: true`, если хотите, чтобы каждая повторная попытка записывалась как новая версия в истории, а не перезаписывала закэшированную ошибку.

### Сохранение ответа в файл

Для больших страниц (PDF, sitemap, длинные статьи) часто удобнее писать тело сразу на диск, не держа его в памяти:

```go
body, err := client.Scrape(ctx, scrapedo.ScrapeRequest{
    URL:    "https://example.com/long-article",
    Output: "raw",
})
if err != nil {
    log.Fatalf("scrape: %v", err)
}

if err := os.WriteFile("article.html", []byte(body), 0o644); err != nil {
    log.Fatalf("write: %v", err)
}
```

## Персистентный кэш (опционально)

CLI поставляет кэш на базе SQLite. Чтобы получить такое же поведение в SDK, передайте в `client.SetCache` любой тип, удовлетворяющий интерфейсу `Cacher`:

```go
type Cacher interface {
    GetResult(ctx context.Context, req ScrapeRequest) (string, bool, error)
    SaveResult(ctx context.Context, req ScrapeRequest, content string, metadata map[string]any) error
}
```

Когда кэш подключён, `Scrape` возвращает закэшированное тело при попадании и сохраняет успешные ответы при промахе, если в запросе не указано `NoCache: true`. `Refresh: true` принудительно делает новый запрос, но всё равно сохраняет результат как новую версию в истории.

### Реализация in-memory кэша

Лёгкий in-memory кэш достаточен для тестов, короткоживущих скриптов или дедупликации запросов в рамках одного процесса. В качестве ключа подойдёт что-нибудь из `crypto/sha256`. Минимальная реализация на `sync.Map`:

```go
type memCache struct{ m sync.Map }

func (c *memCache) key(req scrapedo.ScrapeRequest) string {
    sum := sha256.Sum256([]byte(req.Method + "|" + req.URL + "|" + req.Output))
    return hex.EncodeToString(sum[:])
}

func (c *memCache) GetResult(_ context.Context, req scrapedo.ScrapeRequest) (string, bool, error) {
    v, ok := c.m.Load(c.key(req))
    if !ok {
        return "", false, nil
    }
    return v.(string), true, nil
}

func (c *memCache) SaveResult(_ context.Context, req scrapedo.ScrapeRequest, content string, _ map[string]any) error {
    c.m.Store(c.key(req), content)
    return nil
}

// подключаем
client.SetCache(&memCache{})
```

В продакшене обычно вместо этого подключают LRU-кэш на уровне процесса, Redis или SQLite-кэш из пакета `internal/cache`, который использует CLI.

## Обработка ошибок

В пакете объявлены sentinel-ошибки, которые можно проверять через `errors.Is`:

| Ошибка | Когда возвращается |
|--------|--------------------|
| `scrapedo.ErrEmptyToken` | вызван `NewClient("")` |
| `scrapedo.ErrEmptyURL`   | `ScrapeRequest.URL` пуст |
| `scrapedo.ErrAPI`        | Scrape.do вернул статус ≠ 200 |

```go
if errors.Is(err, scrapedo.ErrAPI) {
    // апстрим отклонил запрос — статус и тело доступны в err.Error()
}
```

## Наблюдаемость

Клиент использует `log/slog` из стандартной библиотеки. Чтобы увидеть каждый исходящий URL (токен маскируется), подключите debug-уровень до вызова `Scrape`:

```go
slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
```

Успешные ответы также логируют заголовки Scrape.do с остатком кредитов и стоимостью запроса на уровне info.
