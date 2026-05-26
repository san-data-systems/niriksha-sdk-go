// Package logger provides a package-level structured logger for the NirikshaAI SDK.
package logger

import (
	"log/slog"
	"os"
	"sync/atomic"
)

var defaultLogger atomic.Pointer[slog.Logger]

func init() {
	l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})).WithGroup("niriksha")
	defaultLogger.Store(l)
}

// Set replaces the SDK logger. Pass nil to restore the default.
func Set(l *slog.Logger) {
	if l == nil {
		l = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})).WithGroup("niriksha")
	}
	defaultLogger.Store(l)
}

// Get returns the current SDK logger.
func Get() *slog.Logger { return defaultLogger.Load() }

// Debug logs at DEBUG level.
func Debug(msg string, args ...any) { Get().Debug(msg, args...) }

// Info logs at INFO level.
func Info(msg string, args ...any) { Get().Info(msg, args...) }

// Warn logs at WARN level.
func Warn(msg string, args ...any) { Get().Warn(msg, args...) }

// Error logs at ERROR level.
func Error(msg string, args ...any) { Get().Error(msg, args...) }
