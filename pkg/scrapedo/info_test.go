package scrapedo_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

func TestInfo_Success(t *testing.T) {
	t.Parallel()

	const respBody = `{
  "IsActive": true,
  "ConcurrentRequest": 10,
  "MaxMonthlyRequest": 250000,
  "RemainingConcurrentRequest": 10,
  "RemainingMonthlyRequest": 247675
}`

	var gotPath, gotToken, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.URL.Query().Get("token")
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(respBody))
	}))
	defer server.Close()

	client, err := scrapedo.NewClient("test-token")
	require.NoError(t, err)
	client.SetBaseURL(server.URL)

	info, err := client.Info(context.Background())
	require.NoError(t, err)
	require.NotNil(t, info)

	// Request shape
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/info", gotPath)
	assert.Equal(t, "test-token", gotToken)

	// Parsed payload
	assert.True(t, info.IsActive)
	assert.Equal(t, 10, info.ConcurrentRequest)
	assert.Equal(t, 250000, info.MaxMonthlyRequest)
	assert.Equal(t, 10, info.RemainingConcurrentRequest)
	assert.Equal(t, 247675, info.RemainingMonthlyRequest)
}

func TestInfo_BaseURLWithTrailingSlash(t *testing.T) {
	t.Parallel()

	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"IsActive":true,"ConcurrentRequest":20,"MaxMonthlyRequest":500000,"RemainingConcurrentRequest":20,"RemainingMonthlyRequest":499999}`))
	}))
	defer server.Close()

	client, _ := scrapedo.NewClient("token")
	client.SetBaseURL(server.URL + "/")

	_, err := client.Info(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "/info", gotPath, "trailing slash on baseURL must not produce //info")
}

func TestInfo_PropagatesAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid token"}`))
	}))
	defer server.Close()

	client, _ := scrapedo.NewClient("bad-token")
	client.SetBaseURL(server.URL)

	info, err := client.Info(context.Background())
	assert.Nil(t, info)
	require.Error(t, err)
	assert.True(t, errors.Is(err, scrapedo.ErrAPI), "should wrap ErrAPI")
	assert.Contains(t, err.Error(), "401")
}

func TestInfo_InvalidJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	client, _ := scrapedo.NewClient("token")
	client.SetBaseURL(server.URL)

	info, err := client.Info(context.Background())
	assert.Nil(t, info)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode info")
}

func TestInfo_ContextCancelled(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ConcurrentRequest":10}`))
	}))
	defer server.Close()

	client, _ := scrapedo.NewClient("token")
	client.SetBaseURL(server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately so the request can never complete

	info, err := client.Info(ctx)
	assert.Nil(t, info)
	require.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "context canceled") ||
			errors.Is(err, context.Canceled),
		"expected context cancellation, got %v", err)
}
