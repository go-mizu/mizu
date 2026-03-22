package cc_v2

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// Logger wraps slog and optionally writes to a Redis stream.
type Logger struct {
	slog   *slog.Logger
	store  Store
	source string // "pipeline", "watcher", "scheduler"
}

// NewLogger creates a Logger that writes to stderr (human-readable) and Redis.
func NewLogger(source string, store Store) *Logger {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return &Logger{
		slog:   slog.New(handler),
		store:  store,
		source: source,
	}
}

func (l *Logger) Info(msg string, args ...any) {
	l.slog.Info(msg, args...)
	if l.store != nil {
		l.store.Log(context.Background(), l.source, "info", l.format(msg, args...))
	}
}

func (l *Logger) Warn(msg string, args ...any) {
	l.slog.Warn(msg, args...)
	if l.store != nil {
		l.store.Log(context.Background(), l.source, "warn", l.format(msg, args...))
	}
}

func (l *Logger) Error(msg string, args ...any) {
	l.slog.Error(msg, args...)
	if l.store != nil {
		l.store.Log(context.Background(), l.source, "error", l.format(msg, args...))
	}
}

func (l *Logger) format(msg string, args ...any) string {
	if len(args) == 0 {
		return msg
	}
	s := msg
	for i := 0; i+1 < len(args); i += 2 {
		s += fmt.Sprintf(" %v=%v", args[i], args[i+1])
	}
	return s
}

// PrintBanner prints the startup banner for a component.
func (l *Logger) PrintBanner(component string, fields map[string]string) {
	fmt.Fprintf(os.Stderr, "\n  CC v2 %s\n", component)
	fmt.Fprintf(os.Stderr, "  %s\n\n", time.Now().Format(time.RFC3339))
	for k, v := range fields {
		fmt.Fprintf(os.Stderr, "  %-14s %s\n", k, v)
	}
	fmt.Fprintln(os.Stderr)
}
