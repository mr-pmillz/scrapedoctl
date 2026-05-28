package repl_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/internal/cache"
	"github.com/mr-pmillz/scrapedoctl/internal/config"
	"github.com/mr-pmillz/scrapedoctl/internal/repl"
	"github.com/mr-pmillz/scrapedoctl/pkg/scrapedo"
	"github.com/mr-pmillz/scrapedoctl/pkg/search"
)

type MockReader struct {
	lines []string
	index int
}

func (m *MockReader) Readline() (string, error) {
	if m.index >= len(m.lines) {
		return "", io.EOF
	}
	line := m.lines[m.index]
	m.index++
	return line, nil
}

func TestREPL_Run(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)
	s := repl.NewShell(client)

	t.Run("successful run with exit", func(t *testing.T) {
		reader := &MockReader{lines: []string{"help", "exit"}}
		s.SetReader(reader)
		err := s.Run(context.Background())
		require.NoError(t, err)
	})

	t.Run("empty lines and unknown command", func(t *testing.T) {
		reader := &MockReader{lines: []string{"", "unknown", "quit"}}
		s.SetReader(reader)
		err := s.Run(context.Background())
		require.NoError(t, err)
	})

	t.Run("readline failure", func(t *testing.T) {
		reader := &MockReader{lines: []string{}} // index 0 >= len 0 -> EOF
		s.SetReader(reader)
		err := s.Run(context.Background())
		require.Error(t, err)
	})

	t.Run("exit command", func(t *testing.T) {
		reader := &MockReader{lines: []string{"exit"}}
		s.SetReader(reader)
		err := s.Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("quit command", func(t *testing.T) {
		reader := &MockReader{lines: []string{"quit"}}
		s.SetReader(reader)
		err := s.Run(context.Background())
		assert.NoError(t, err)
	})
}

func TestREPL_ExecuteCommand(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mocked result"))
	}))
	defer ts.Close()

	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)
	client.SetBaseURL(ts.URL)
	s := repl.NewShell(client)

	ctx := context.Background()

	tests := []struct {
		name    string
		line    string
		wantErr error
	}{
		{"exit", "exit", repl.ErrExit},
		{"quit", "quit", repl.ErrExit},
		{"help", "help", nil},
		{"scrape", "scrape http://example.com", nil},
		{"scrape with params", "scrape http://example.com render=true super=true", nil},
		{"scrape invalid", "scrape", repl.ErrInvalidUsage},
		{"unknown", "unknown", repl.ErrUnknownCmd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.ExecuteCommand(ctx, tt.line)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestREPL_ScrapeError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)
	client.SetBaseURL(ts.URL)
	s := repl.NewShell(client)

	err = s.ExecuteCommand(context.Background(), "scrape http://example.com")
	require.Error(t, err)
}

// --- Mock search provider ---

type mockProvider struct {
	name    string
	engines []string
	resp    *search.Response
	err     error
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Engines() []string { return m.engines }
func (m *mockProvider) Search(_ context.Context, _ string, _ search.Options) (*search.Response, error) {
	return m.resp, m.err
}

// --- Mock cache store ---

type mockCacheStore struct {
	history      []cache.ScrapeRecord
	stats        cache.Stats
	usageSummary []cache.UsageSummary
	err          error
}

func (m *mockCacheStore) GetHistory(_ context.Context, _ string) ([]cache.ScrapeRecord, error) {
	return m.history, m.err
}

func (m *mockCacheStore) GetStats(_ context.Context) (cache.Stats, error) {
	return m.stats, m.err
}

func (m *mockCacheStore) Clear(_ context.Context) error {
	return m.err
}

func (m *mockCacheStore) GetUsageSummary(_ context.Context, _ time.Time) ([]cache.UsageSummary, error) {
	return m.usageSummary, m.err
}

// --- New command tests ---

func TestREPL_SearchCommand(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)

	router := search.NewRouter()
	router.Register(&mockProvider{
		name:    "mock",
		engines: []string{"google"},
		resp: &search.Response{
			Query:    "test",
			Engine:   "google",
			Provider: "mock",
			Results: []search.Result{
				{Position: 1, Title: "Test", URL: "http://example.com", Snippet: "snippet"},
			},
		},
	})

	s := repl.NewShell(client, repl.WithSearchRouter(router))
	ctx := context.Background()

	t.Run("basic search", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "search test query")
		require.NoError(t, err)
	})

	t.Run("search with engine param", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "search test engine=google")
		require.NoError(t, err)
	})

	t.Run("search no query", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "search")
		require.ErrorIs(t, err, repl.ErrInvalidUsage)
	})

	t.Run("search no router", func(t *testing.T) {
		s2 := repl.NewShell(client)
		err := s2.ExecuteCommand(ctx, "search test")
		require.ErrorIs(t, err, repl.ErrNoRouter)
	})
}

func TestREPL_HistoryCommand(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)

	mc := &mockCacheStore{
		history: []cache.ScrapeRecord{
			{ID: 1, URL: "http://example.com"},
		},
	}

	s := repl.NewShell(client, repl.WithCache(mc))
	ctx := context.Background()

	t.Run("history with results", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "show history http://example.com")
		require.NoError(t, err)
	})

	t.Run("history no url", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "show history")
		require.ErrorIs(t, err, repl.ErrInvalidUsage)
	})

	t.Run("history empty results", func(t *testing.T) {
		mc2 := &mockCacheStore{history: nil}
		s2 := repl.NewShell(client, repl.WithCache(mc2))
		err := s2.ExecuteCommand(ctx, "show history http://nothing.com")
		require.NoError(t, err)
	})

	t.Run("history no cache", func(t *testing.T) {
		s2 := repl.NewShell(client)
		err := s2.ExecuteCommand(ctx, "show history http://example.com")
		require.ErrorIs(t, err, repl.ErrNoCache)
	})
}

func TestREPL_CacheStatsCommand(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)

	mc := &mockCacheStore{
		stats: cache.Stats{TotalCount: 42, TotalSize: 1024},
	}

	s := repl.NewShell(client, repl.WithCache(mc))
	ctx := context.Background()

	err = s.ExecuteCommand(ctx, "show cache")
	require.NoError(t, err)
}

func TestREPL_CacheClearCommand(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)

	mc := &mockCacheStore{}

	s := repl.NewShell(client, repl.WithCache(mc))
	ctx := context.Background()

	err = s.ExecuteCommand(ctx, "clear cache")
	require.NoError(t, err)
}

func TestREPL_ConfigListCommand(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)

	cfg := &config.Config{
		Global: config.GlobalConfig{
			Token:   "test-token",
			BaseURL: "https://api.scrape.do",
			Timeout: 60000,
		},
		Repl: config.ReplConfig{
			HistoryFile: "/tmp/history",
		},
	}

	s := repl.NewShell(client, repl.WithConfig(cfg))
	ctx := context.Background()

	err = s.ExecuteCommand(ctx, "show config")
	require.NoError(t, err)
}

func TestREPL_ConfigGetCommand(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)

	cfg := &config.Config{
		Global: config.GlobalConfig{
			Token:   "test-token",
			BaseURL: "https://api.scrape.do",
		},
	}

	s := repl.NewShell(client, repl.WithConfig(cfg))
	ctx := context.Background()

	t.Run("valid key", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "show config global.token")
		require.NoError(t, err)
	})

	t.Run("invalid key", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "show config unknown.key")
		require.ErrorIs(t, err, repl.ErrUnsupportedKey)
	})
}

func TestREPL_HelpCommand(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)
	s := repl.NewShell(client)
	ctx := context.Background()

	t.Run("general help", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "help")
		require.NoError(t, err)
	})

	t.Run("help for specific command", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "help search")
		require.NoError(t, err)
	})
}

func TestREPL_HelpSpecificCommand(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)
	s := repl.NewShell(client)
	ctx := context.Background()

	err = s.ExecuteCommand(ctx, "help search")
	require.NoError(t, err)
}

func TestREPL_UnknownCommand(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)
	s := repl.NewShell(client)
	ctx := context.Background()

	err = s.ExecuteCommand(ctx, "foobar")
	require.ErrorIs(t, err, repl.ErrUnknownCmd)
}

func TestREPL_EmptyLine(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)
	s := repl.NewShell(client)
	ctx := context.Background()

	err = s.ExecuteCommand(ctx, "")
	require.NoError(t, err)
}

func TestREPL_CacheNoStore(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)
	s := repl.NewShell(client)
	ctx := context.Background()

	t.Run("stats no cache", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "show cache")
		require.ErrorIs(t, err, repl.ErrNoCache)
	})

	t.Run("clear no cache", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "clear cache")
		require.ErrorIs(t, err, repl.ErrNoCache)
	})
}

func TestREPL_ConfigNoConfig(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)
	s := repl.NewShell(client)
	ctx := context.Background()

	t.Run("show config no config", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "show config")
		require.ErrorIs(t, err, repl.ErrNoConfig)
	})

	t.Run("show config key no config", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "show config global.token")
		require.ErrorIs(t, err, repl.ErrNoConfig)
	})

	t.Run("set no config", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "set global.token foo")
		require.ErrorIs(t, err, repl.ErrNoConfig)
	})
}

func TestREPL_ShowSubcommandHelp(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)
	s := repl.NewShell(client)
	ctx := context.Background()

	// "show" without subcommand shows help.
	err = s.ExecuteCommand(ctx, "show")
	require.NoError(t, err)
}

func TestREPL_ClearSubcommandHelp(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)
	s := repl.NewShell(client)
	ctx := context.Background()

	// "clear" without subcommand shows help.
	err = s.ExecuteCommand(ctx, "clear")
	require.NoError(t, err)
}

// --- Prefix matching tests ---

func TestREPL_PrefixMatching(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mocked result"))
	}))
	defer ts.Close()

	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)
	client.SetBaseURL(ts.URL)

	cfg := &config.Config{
		Global: config.GlobalConfig{
			Token:   "test-token",
			BaseURL: "https://api.scrape.do",
		},
	}

	mc := &mockCacheStore{}

	router := search.NewRouter()
	router.Register(&mockProvider{
		name:    "mock",
		engines: []string{"google"},
		resp: &search.Response{
			Query:    "query",
			Engine:   "google",
			Provider: "mock",
			Results: []search.Result{
				{Position: 1, Title: "Test", URL: "http://example.com", Snippet: "snippet"},
			},
		},
	})

	s := repl.NewShell(client,
		repl.WithConfig(cfg),
		repl.WithCache(mc),
		repl.WithSearchRouter(router),
	)
	ctx := context.Background()

	t.Run("sh con resolves to show config", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "sh con")
		require.NoError(t, err)
	})

	t.Run("sea query resolves to search", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "sea query")
		require.NoError(t, err)
	})

	t.Run("s is ambiguous", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "s")
		require.ErrorIs(t, err, repl.ErrAmbiguousCmd)
	})

	t.Run("cl ca resolves to clear cache", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "cl ca")
		require.NoError(t, err)
	})
}

// --- Context help tests ---

func TestREPL_ContextHelp(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)
	s := repl.NewShell(client)
	ctx := context.Background()

	t.Run("? shows all commands", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "?")
		require.NoError(t, err)
	})

	t.Run("show? shows subcommands", func(t *testing.T) {
		err := s.ExecuteCommand(ctx, "show?")
		require.NoError(t, err)
	})
}

// --- set command tests ---

func TestREPL_SetCommand(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)

	t.Run("set without args", func(t *testing.T) {
		cfg := &config.Config{}
		s := repl.NewShell(client, repl.WithConfig(cfg))
		err := s.ExecuteCommand(context.Background(), "set")
		require.ErrorIs(t, err, repl.ErrInvalidUsage)
	})

	t.Run("set global.token mytoken", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := tmpDir + "/config.toml"
		cfg, loadErr := config.Load(configPath, "")
		// Config not found is expected for a fresh path.
		require.NotNil(t, cfg)
		_ = loadErr
		s := repl.NewShell(client, repl.WithConfig(cfg))
		err := s.ExecuteCommand(context.Background(), "set global.token mytoken")
		require.NoError(t, err)
	})

	t.Run("set unsupported key", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := tmpDir + "/config.toml"
		cfg, _ := config.Load(configPath, "")
		require.NotNil(t, cfg)
		s := repl.NewShell(client, repl.WithConfig(cfg))
		err := s.ExecuteCommand(context.Background(), "set bad.key value")
		require.ErrorIs(t, err, repl.ErrUnsupportedKey)
	})
}

// --- show version test ---

// --- mock account provider ---

type mockAccountProvider struct {
	name    string
	engines []string
	info    *search.AccountInfo
	err     error
}

func (m *mockAccountProvider) Name() string {
	return m.name
}

func (m *mockAccountProvider) Engines() []string {
	return m.engines
}

var errMockAccount = errors.New("mock account")

func (m *mockAccountProvider) Search(
	_ context.Context, _ string, _ search.Options,
) (*search.Response, error) {
	return nil, errMockAccount
}

func (m *mockAccountProvider) Account(_ context.Context) (*search.AccountInfo, error) {
	return m.info, m.err
}

func TestREPL_ShowAccount(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		router := search.NewRouter()
		router.Register(&mockAccountProvider{
			name:    "mock",
			engines: []string{"google"},
			info: &search.AccountInfo{
				Provider:          "mock",
				Active:            true,
				UsedRequests:      36,
				MaxRequests:       1000,
				RemainingRequests: 964,
				Concurrency:       5,
			},
		})

		s := repl.NewShell(client, repl.WithSearchRouter(router))
		err := s.ExecuteCommand(context.Background(), "show account")
		require.NoError(t, err)
	})

	t.Run("no router", func(t *testing.T) {
		s := repl.NewShell(client)
		err := s.ExecuteCommand(context.Background(), "show account")
		require.ErrorIs(t, err, repl.ErrNoRouter)
	})

	t.Run("provider without AccountChecker", func(t *testing.T) {
		router := search.NewRouter()
		router.Register(&mockProvider{
			name:    "basic",
			engines: []string{"google"},
		})

		s := repl.NewShell(client, repl.WithSearchRouter(router))
		err := s.ExecuteCommand(context.Background(), "show account")
		require.NoError(t, err)
	})

	t.Run("provider with error", func(t *testing.T) {
		router := search.NewRouter()
		router.Register(&mockAccountProvider{
			name:    "failing",
			engines: []string{"google"},
			err:     assert.AnError,
		})

		s := repl.NewShell(client, repl.WithSearchRouter(router))
		err := s.ExecuteCommand(context.Background(), "show account")
		require.NoError(t, err)
	})

	t.Run("with plan and rate limit", func(t *testing.T) {
		router := search.NewRouter()
		router.Register(&mockAccountProvider{
			name:    "serpapi",
			engines: []string{"google"},
			info: &search.AccountInfo{
				Provider:          "serpapi",
				Plan:              "Free Plan",
				Active:            true,
				UsedRequests:      12,
				MaxRequests:       250,
				RemainingRequests: 238,
				RateLimit:         250,
			},
		})

		s := repl.NewShell(client, repl.WithSearchRouter(router))
		err := s.ExecuteCommand(context.Background(), "show account")
		require.NoError(t, err)
	})
}

func TestREPL_ShowUsage(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)

	t.Run("with results", func(t *testing.T) {
		mc := &mockCacheStore{
			usageSummary: []cache.UsageSummary{
				{Provider: "scrapedo", Action: "scrape", Count: 5, TotalCredits: 5},
				{Provider: "scrapedo", Action: "search", Count: 3, TotalCredits: 3},
			},
		}
		s := repl.NewShell(client, repl.WithCache(mc))
		err := s.ExecuteCommand(context.Background(), "show usage")
		require.NoError(t, err)
	})

	t.Run("with --week flag", func(t *testing.T) {
		mc := &mockCacheStore{usageSummary: nil}
		s := repl.NewShell(client, repl.WithCache(mc))
		err := s.ExecuteCommand(context.Background(), "show usage --week")
		require.NoError(t, err)
	})

	t.Run("with --month flag", func(t *testing.T) {
		mc := &mockCacheStore{usageSummary: nil}
		s := repl.NewShell(client, repl.WithCache(mc))
		err := s.ExecuteCommand(context.Background(), "show usage --month")
		require.NoError(t, err)
	})

	t.Run("with --all flag", func(t *testing.T) {
		mc := &mockCacheStore{usageSummary: nil}
		s := repl.NewShell(client, repl.WithCache(mc))
		err := s.ExecuteCommand(context.Background(), "show usage --all")
		require.NoError(t, err)
	})

	t.Run("no cache", func(t *testing.T) {
		s := repl.NewShell(client)
		err := s.ExecuteCommand(context.Background(), "show usage")
		require.ErrorIs(t, err, repl.ErrNoCache)
	})
}

func TestREPL_ShowVersion(t *testing.T) {
	client, err := scrapedo.NewClient("token")
	require.NoError(t, err)
	s := repl.NewShell(client)
	ctx := context.Background()

	// show version calls version.CheckLatest which may fail in test env,
	// but should not panic. We just verify it doesn't return a sentinel error.
	err = s.ExecuteCommand(ctx, "show version")
	// It might error due to network, but should not be a usage/config error.
	if err != nil {
		require.NotErrorIs(t, err, repl.ErrInvalidUsage)
	}
}
