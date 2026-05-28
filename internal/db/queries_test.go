package db_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/mr-pmillz/scrapedoctl/internal/db"
)

func setupTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	dbConn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	goose.SetBaseFS(db.Migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}

	if err := goose.Up(dbConn, "migrations"); err != nil {
		t.Fatal(err)
	}

	return dbConn, db.New(dbConn)
}

func TestQueries(t *testing.T) {
	dbConn, q := setupTestDB(t)
	defer dbConn.Close()

	ctx := context.Background()

	// Test InsertScrape
	params := db.InsertScrapeParams{
		RequestHash: "hash1",
		Url:         "http://example.com",
		Method:      "GET",
		Content:     "content1",
		Metadata:    "{}",
	}
	scrape, err := q.InsertScrape(ctx, params)
	require.NoError(t, err)
	assert.Equal(t, params.Url, scrape.Url)
	assert.NotEmpty(t, scrape.ID)

	// Test GetLatestScrape
	latest, err := q.GetLatestScrape(ctx, "hash1")
	require.NoError(t, err)
	assert.Equal(t, scrape.ID, latest.ID)

	// Test GetHistoryByUrl
	history, err := q.GetHistoryByUrl(ctx, "http://example.com")
	require.NoError(t, err)
	assert.Len(t, history, 1)

	// Test GetStats
	stats, err := q.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.TotalCount)
	assert.True(t, stats.TotalSizeBytes.Valid)
	assert.InDelta(t, float64(len("content1")), stats.TotalSizeBytes.Float64, 0.001)

	// Test DeleteOldVersions
	// Insert another one with same hash
	_, err = q.InsertScrape(ctx, params)
	require.NoError(t, err)

	err = q.DeleteOldVersions(ctx, db.DeleteOldVersionsParams{
		RequestHash:   "hash1",
		RequestHash_2: "hash1",
		Limit:         1,
	})
	require.NoError(t, err)

	history, err = q.GetHistoryByUrl(ctx, "http://example.com")
	require.NoError(t, err)
	assert.Len(t, history, 1)

	// Test ClearCache
	err = q.ClearCache(ctx)
	require.NoError(t, err)
	stats, err = q.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalCount)
}

func TestGetHistoryByUrl_Error(t *testing.T) {
	dbConn, q := setupTestDB(t)
	dbConn.Close() // Close it to cause error

	_, err := q.GetHistoryByUrl(context.Background(), "http://example.com")
	assert.Error(t, err)
}

func TestWithTx(t *testing.T) {
	dbConn, q := setupTestDB(t)
	defer dbConn.Close()

	tx, err := dbConn.BeginTx(context.Background(), nil)
	require.NoError(t, err)

	qtx := q.WithTx(tx)
	assert.NotNil(t, qtx)

	err = tx.Rollback()
	assert.NoError(t, err)
}
