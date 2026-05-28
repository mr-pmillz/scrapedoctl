// Package logger provides structured logging with rotation.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/mr-pmillz/scrapedoctl/internal/config"
)

// Init initializes the global logger based on the provided configuration.
func Init(cfg config.LoggingConfig) {
	var writer io.Writer = os.Stderr

	if cfg.Path != "" {
		// Attempt to ensure directory exists
		dir := filepath.Dir(cfg.Path)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			fmt.Fprintf(
				os.Stderr,
				"Warning: failed to create log directory %s: %v. Falling back to stderr.\n",
				dir,
				err,
			)
		} else {
			writer = &lumberjack.Logger{
				Filename:   cfg.Path,
				MaxSize:    cfg.MaxSize,
				MaxBackups: cfg.MaxBackups,
				MaxAge:     cfg.MaxAge,
				Compress:   cfg.Compress,
			}
		}
	}

	level := parseLevel(cfg.Level)

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level}

	if strings.ToLower(cfg.Format) == "text" {
		handler = slog.NewTextHandler(writer, opts)
	} else {
		handler = slog.NewJSONHandler(writer, opts)
	}

	slog.SetDefault(slog.New(handler))
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
