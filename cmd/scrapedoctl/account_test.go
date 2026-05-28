package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/internal/config"
	"github.com/mr-pmillz/scrapedoctl/pkg/search"
)

func TestAccountCmd_Table(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"IsActive": true,
			"ConcurrentRequest": 5,
			"MaxMonthlyRequest": 1000,
			"RemainingConcurrentRequest": 5,
			"RemainingMonthlyRequest": 964
		}`))
	}))
	defer srv.Close()

	cfg = &config.Config{}
	router := search.NewRouter()
	p := search.NewScrapedoProvider("tok")
	p.SetBaseURL(srv.URL)
	router.Register(p)
	searchRouter = router

	output := captureStdout(t, func() {
		cmd := newAccountCmd()
		err := cmd.RunE(cmd, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "scrapedo")
	assert.Contains(t, output, "964")
}

func TestAccountCmd_JSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"IsActive": true,
			"ConcurrentRequest": 5,
			"MaxMonthlyRequest": 1000,
			"RemainingConcurrentRequest": 5,
			"RemainingMonthlyRequest": 964
		}`))
	}))
	defer srv.Close()

	cfg = &config.Config{}
	router := search.NewRouter()
	p := search.NewScrapedoProvider("tok")
	p.SetBaseURL(srv.URL)
	router.Register(p)
	searchRouter = router

	output := captureStdout(t, func() {
		cmd := newAccountCmd()
		require.NoError(t, cmd.Flags().Set("json", "true"))
		err := cmd.RunE(cmd, nil)
		require.NoError(t, err)
	})

	var infos []search.AccountInfo
	require.NoError(t, json.Unmarshal([]byte(output), &infos))
	require.Len(t, infos, 1)
	assert.Equal(t, "scrapedo", infos[0].Provider)
	assert.Equal(t, 964, infos[0].RemainingRequests)
}

func TestAccountCmd_NoRouter(t *testing.T) {
	searchRouter = nil

	cmd := newAccountCmd()
	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, errNoSearchProviders)
}

func TestFetchAccountInfos_SkipsNonChecker(t *testing.T) {
	router := search.NewRouter()
	router.Register(&nonCheckerProvider{})

	infos := fetchAccountInfos(context.Background(), router)
	assert.Empty(t, infos)
}

// nonCheckerProvider is a Provider that does NOT implement AccountChecker.
type nonCheckerProvider struct{}

func (p *nonCheckerProvider) Name() string {
	return "noop"
}

func (p *nonCheckerProvider) Engines() []string {
	return []string{"test"}
}

var errNonChecker = errors.New("not implemented")

func (p *nonCheckerProvider) Search(
	_ context.Context, _ string, _ search.Options,
) (*search.Response, error) {
	return nil, errNonChecker
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	return buf.String()
}
