package mcp_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/internal/mcp"
	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
)

func TestRunServer_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := mcp.RunServer(ctx, "test-token")
	// If initialization succeeds, it should return context.Canceled when trying to run the transport
	require.Error(t, err)
}

func TestRunServer_EmptyToken(t *testing.T) {
	ctx := context.Background()
	err := mcp.RunServer(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to init scrape.do client")
}

func TestRunServerWithClient_NilClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := mcp.RunServerWithClient(ctx, nil) // Should return error because client is nil
	require.Error(t, err)
	assert.ErrorIs(t, err, mcp.ErrClientNil)
}

func TestRunServerWithClient_ValidClient_Cancel(t *testing.T) {
	client, err := scrapedo.NewClient("test-token")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately so server.Run returns an error

	err = mcp.RunServerWithClient(ctx, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mcp server failed")
}

func TestNewServer_Success(t *testing.T) {
	server, err := mcp.NewServer("valid-token")
	require.NoError(t, err)
	assert.NotNil(t, server)
}
