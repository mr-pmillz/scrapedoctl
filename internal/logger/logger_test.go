package logger_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mr-pmillz/scrapedoctl/internal/config"
	"github.com/mr-pmillz/scrapedoctl/internal/logger"
)

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	cfg := config.LoggingConfig{
		Level:      "debug",
		Format:     "json",
		Path:       logPath,
		MaxSize:    1,
		MaxAge:     1,
		MaxBackups: 1,
		Compress:   true,
	}

	t.Run("initialize json logger with file", func(t *testing.T) {
		logger.Init(cfg)
		slog.Debug("test debug message")

		// Verify file exists
		assert.FileExists(t, logPath)

		data, err := os.ReadFile(logPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "test debug message")
		assert.Contains(t, string(data), "\"level\":\"DEBUG\"")
	})

	t.Run("initialize text logger with stderr fallback", func(_ *testing.T) {
		// Use an invalid path to trigger fallback
		cfg.Path = filepath.Join(tmpDir, "nonexistent_dir", "test.log")
		// We can't easily capture stderr here without complex piping,
		// but we can verify Init doesn't panic.
		cfg.Format = "text"
		logger.Init(cfg)
		slog.Info("test info message")
	})
}

func TestParseLevel(t *testing.T) {
	assert.Equal(t, slog.LevelDebug, logger.ParseLevel("debug"))
	assert.Equal(t, slog.LevelInfo, logger.ParseLevel("info"))
	assert.Equal(t, slog.LevelWarn, logger.ParseLevel("warn"))
	assert.Equal(t, slog.LevelWarn, logger.ParseLevel("warning"))
	assert.Equal(t, slog.LevelError, logger.ParseLevel("error"))
	assert.Equal(t, slog.LevelInfo, logger.ParseLevel("unknown"))
}
