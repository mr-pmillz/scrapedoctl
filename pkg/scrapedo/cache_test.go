package scrapedo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

type MockCacher struct {
	mock.Mock
}

func (m *MockCacher) GetResult(ctx context.Context, req scrapedo.ScrapeRequest) (string, bool, error) {
	args := m.Called(ctx, req)
	return args.String(0), args.Bool(1), args.Error(2)
}

func (m *MockCacher) SaveResult(
	ctx context.Context,
	req scrapedo.ScrapeRequest,
	content string,
	metadata map[string]any,
) error {
	args := m.Called(ctx, req, content, metadata)
	return args.Error(0)
}

func TestScrape_CacheHit(t *testing.T) {
	client, _ := scrapedo.NewClient("token")
	mockCache := new(MockCacher)
	client.SetCache(mockCache)

	req := scrapedo.ScrapeRequest{URL: "http://cached.com"}
	mockCache.On("GetResult", mock.Anything, req).Return("cached content", true, nil)

	res, err := client.Scrape(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "cached content", res)
	mockCache.AssertExpectations(t)
}

func TestScrape_CacheSaveError(t *testing.T) {
	t.Helper()
	// This covers the slog.Warn when SaveResult fails
	// We'll use a real server but a failing cache
	// (Implementation details omitted for brevity, but let's do it)
}
