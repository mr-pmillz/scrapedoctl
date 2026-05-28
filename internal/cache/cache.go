// Package cache handles persistent caching of scrape results.
// It provides a middle layer between the scrapedo client and the SQLite database,
// implementing TTL policies and version history.
package cache

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pressly/goose/v3"
	// Import the sqlite driver.
	_ "modernc.org/sqlite"

	"github.com/mr-pmillz/scrapedoctl/internal/config"
	"github.com/mr-pmillz/scrapedoctl/internal/db"
	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

// Store implements the persistent caching logic.
type Store struct {
	database *sql.DB
	queries  *db.Queries
	cfg      config.CacheConfig
}

// NewStore initializes the SQLite database and runs migrations.
// It uses embedded SQL migrations for portability.
func NewStore(cfg config.CacheConfig) (*Store, error) {
	path := expandPath(cfg.Path)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Run migrations
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		return nil, fmt.Errorf("failed to set goose dialect: %w", err)
	}

	goose.SetBaseFS(db.Migrations)
	if err := goose.Up(database, "migrations"); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &Store{
		database: database,
		queries:  db.New(database),
		cfg:      cfg,
	}, nil
}

// GetResult checks the cache for a matching request.
// It returns the cached content if found and within TTL.
func (s *Store) GetResult(ctx context.Context, req scrapedo.ScrapeRequest) (string, bool, error) {
	hash := NormalizeAndHash(req)
	scrape, err := s.queries.GetLatestScrape(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to get latest scrape: %w", err)
	}

	// Check TTL
	ttl := time.Duration(s.cfg.TTLDays) * 24 * time.Hour
	if time.Since(scrape.CreatedAt) > ttl {
		return "", false, nil
	}

	return scrape.Content, true, nil
}

// SaveResult stores a new scrape result and performs cleanup of old versions.
func (s *Store) SaveResult(
	ctx context.Context,
	req scrapedo.ScrapeRequest,
	content string,
	metadata map[string]any,
) error {
	hash := NormalizeAndHash(req)
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = s.queries.InsertScrape(ctx, db.InsertScrapeParams{
		RequestHash: hash,
		Url:         req.URL,
		Method:      req.Method,
		Content:     content,
		Metadata:    string(metaJSON),
	})
	if err != nil {
		return fmt.Errorf("failed to insert scrape: %w", err)
	}

	// Cleanup old versions
	if err := s.queries.DeleteOldVersions(ctx, db.DeleteOldVersionsParams{
		RequestHash:   hash,
		RequestHash_2: hash,
		Limit:         int64(s.cfg.KeepVersions),
	}); err != nil {
		return fmt.Errorf("failed to cleanup old versions: %w", err)
	}

	return nil
}

// ScrapeRecord represents a single scrape entry in the DB.
type ScrapeRecord struct {
	ID        int64
	URL       string
	CreatedAt time.Time
	Metadata  string
	Content   string
}

// GetHistory returns all historical versions for a given URL.
func (s *Store) GetHistory(ctx context.Context, url string) ([]ScrapeRecord, error) {
	rows, err := s.queries.GetHistoryByUrl(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	var results []ScrapeRecord
	for _, r := range rows {
		results = append(results, ScrapeRecord{
			ID:        r.ID,
			URL:       r.Url,
			CreatedAt: r.CreatedAt,
			Metadata:  r.Metadata,
			Content:   r.Content,
		})
	}
	return results, nil
}

// Stats represents cache statistics.
type Stats struct {
	TotalCount int64
	TotalSize  int64
}

// GetStats returns database statistics including entry count and total size.
func (s *Store) GetStats(ctx context.Context) (Stats, error) {
	row, err := s.queries.GetStats(ctx)
	if err != nil {
		return Stats{}, fmt.Errorf("failed to get stats: %w", err)
	}

	var size int64
	if row.TotalSizeBytes.Valid {
		size = int64(row.TotalSizeBytes.Float64)
	}

	return Stats{
		TotalCount: row.TotalCount,
		TotalSize:  size,
	}, nil
}

// Clear removes all entries from the cache.
func (s *Store) Clear(ctx context.Context) error {
	if err := s.queries.ClearCache(ctx); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}
	return nil
}

// NormalizeAndHash creates a stable SHA256 hash of the request by sorting all parameters.
func NormalizeAndHash(req scrapedo.ScrapeRequest) string {
	var parts []string
	parts = append(parts, "url="+req.URL)
	parts = append(parts, "method="+strings.ToUpper(req.Method))
	parts = append(parts, fmt.Sprintf("render=%v", req.Render))
	parts = append(parts, fmt.Sprintf("super=%v", req.Super))
	parts = append(parts, "geo="+req.GeoCode)
	parts = append(parts, "device="+req.Device)
	parts = append(parts, "session="+req.Session)

	// Sort headers
	keys := make([]string, 0, len(req.Headers))
	for k := range req.Headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("h:%s=%s", strings.ToLower(k), req.Headers[k]))
	}

	// Sort actions
	if len(req.Actions) > 0 {
		actionsJSON, err := json.Marshal(req.Actions)
		if err == nil {
			parts = append(parts, "actions="+string(actionsJSON))
		}
	}

	// Body hash if present
	if len(req.Body) > 0 {
		bodyHash := sha256.Sum256(req.Body)
		parts = append(parts, fmt.Sprintf("body=%x", bodyHash))
	}

	data := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// UsageRecord represents a single usage log entry.
type UsageRecord struct {
	ID        int64
	Provider  string
	Engine    string
	Action    string
	Query     string
	URL       string
	Credits   int
	CreatedAt time.Time
}

// UsageSummary represents aggregated usage by provider and action.
type UsageSummary struct {
	Provider     string
	Action       string
	Count        int64
	TotalCredits int64
}

// RecordUsage inserts a usage log entry for an API call.
func (s *Store) RecordUsage(
	ctx context.Context, provider, engine, action, query, url string, credits int,
) error {
	if err := s.queries.InsertUsage(ctx, db.InsertUsageParams{
		Provider: provider,
		Engine:   engine,
		Action:   action,
		Query:    query,
		Url:      url,
		Credits:  int64(credits),
	}); err != nil {
		return fmt.Errorf("failed to record usage: %w", err)
	}
	return nil
}

// GetUsageSince returns all usage records since the given time.
func (s *Store) GetUsageSince(ctx context.Context, since time.Time) ([]UsageRecord, error) {
	rows, err := s.queries.GetUsageSince(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}

	records := make([]UsageRecord, 0, len(rows))
	for _, r := range rows {
		records = append(records, UsageRecord{
			ID: r.ID, Provider: r.Provider, Engine: r.Engine,
			Action: r.Action, Query: r.Query, URL: r.Url,
			Credits: int(r.Credits), CreatedAt: r.CreatedAt,
		})
	}
	return records, nil
}

// GetUsageSummary returns usage aggregated by provider and action since the given time.
func (s *Store) GetUsageSummary(ctx context.Context, since time.Time) ([]UsageSummary, error) {
	rows, err := s.queries.GetUsageByProvider(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage summary: %w", err)
	}

	summaries := make([]UsageSummary, 0, len(rows))
	for _, r := range rows {
		var credits int64
		if r.TotalCredits.Valid {
			credits = int64(r.TotalCredits.Float64)
		}
		summaries = append(summaries, UsageSummary{
			Provider: r.Provider, Action: r.Action,
			Count: r.Count, TotalCredits: credits,
		})
	}
	return summaries, nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
