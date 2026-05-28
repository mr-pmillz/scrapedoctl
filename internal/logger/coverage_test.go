package logger_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mr-pmillz/scrapedoctl/internal/config"
	"github.com/mr-pmillz/scrapedoctl/internal/logger"
)

func TestInit_Branches(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("mkdir error", func(_ *testing.T) {
		// Create a file where a directory should be
		conflictFile := filepath.Join(tmpDir, "conflict")
		_ = os.WriteFile(conflictFile, []byte("not a dir"), 0o644)

		cfg := config.LoggingConfig{
			Path:   filepath.Join(conflictFile, "log.txt"),
			Format: "text",
			Level:  "warn",
		}
		logger.Init(cfg)
		// Should fall back to stderr and continue
	})

	t.Run("levels", func(_ *testing.T) {
		levels := []string{"debug", "warn", "warning", "error", "info", "unknown"}
		for _, l := range levels {
			cfg := config.LoggingConfig{
				Level: l,
			}
			logger.Init(cfg)
		}
	})

	t.Run("json format", func(_ *testing.T) {
		cfg := config.LoggingConfig{
			Format: "json",
		}
		logger.Init(cfg)
	})
}

func TestParseLevel_Branches(t *testing.T) {
	assert.Equal(t, slog.LevelDebug, logger.ParseLevel("debug"))
	assert.Equal(t, slog.LevelWarn, logger.ParseLevel("warn"))
	assert.Equal(t, slog.LevelWarn, logger.ParseLevel("warning"))
	assert.Equal(t, slog.LevelError, logger.ParseLevel("error"))
	assert.Equal(t, slog.LevelInfo, logger.ParseLevel("info"))
	assert.Equal(t, slog.LevelInfo, logger.ParseLevel("unknown"))
}
