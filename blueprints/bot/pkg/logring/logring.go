// Package logring provides a thread-safe ring buffer for structured log entries.
// It captures application logs and makes them available for the dashboard's
// real-time log viewer with level filtering and text search.
package logring

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Level represents a log severity level.
type Level int

const (
	LevelTrace Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String returns the level name.
func (l Level) String() string {
	switch l {
	case LevelTrace:
		return "trace"
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	case LevelFatal:
		return "fatal"
	default:
		return "unknown"
	}
}

// ParseLevel converts a string to a Level.
func ParseLevel(s string) Level {
	switch strings.ToLower(s) {
	case "trace":
		return LevelTrace
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	case "fatal":
		return LevelFatal
	default:
		return LevelInfo
	}
}

// Entry is a single log entry.
type Entry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Subsystem string `json:"subsystem"`
	Message   string `json:"message"`
	Raw       string `json:"raw,omitempty"`
}

// Ring is a fixed-size ring buffer for log entries.
type Ring struct {
	mu      sync.RWMutex
	entries []Entry
	head    int
	count   int
	cap     int
}

// New creates a ring buffer with the given capacity.
func New(capacity int) *Ring {
	if capacity <= 0 {
		capacity = 2000
	}
	return &Ring{
		entries: make([]Entry, capacity),
		cap:     capacity,
	}
}

// Add appends an entry to the ring buffer.
func (r *Ring) Add(entry Entry) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries[r.head] = entry
	r.head = (r.head + 1) % r.cap
	if r.count < r.cap {
		r.count++
	}
}

// Log adds a formatted log entry.
func (r *Ring) Log(level Level, subsystem, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	r.Add(Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level.String(),
		Subsystem: subsystem,
		Message:   msg,
	})
}

// Info logs at info level.
func (r *Ring) Info(subsystem, format string, args ...any) {
	r.Log(LevelInfo, subsystem, format, args...)
}

// Warn logs at warn level.
func (r *Ring) Warn(subsystem, format string, args ...any) {
	r.Log(LevelWarn, subsystem, format, args...)
}

// Error logs at error level.
func (r *Ring) Error(subsystem, format string, args ...any) {
	r.Log(LevelError, subsystem, format, args...)
}

// Debug logs at debug level.
func (r *Ring) Debug(subsystem, format string, args ...any) {
	r.Log(LevelDebug, subsystem, format, args...)
}

// Tail returns the last n entries, optionally filtered by minimum level.
func (r *Ring) Tail(n int, minLevel string) []Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if n <= 0 || n > r.count {
		n = r.count
	}

	var lvl Level
	if minLevel != "" {
		lvl = ParseLevel(minLevel)
	}

	// Collect from oldest to newest.
	start := (r.head - r.count + r.cap) % r.cap
	var result []Entry
	for i := range r.count {
		idx := (start + i) % r.cap
		e := r.entries[idx]
		if minLevel != "" && ParseLevel(e.Level) < lvl {
			continue
		}
		result = append(result, e)
	}

	// Return only the last n.
	if len(result) > n {
		result = result[len(result)-n:]
	}
	return result
}

// Search returns entries matching the query string in message, subsystem, or raw fields.
func (r *Ring) Search(query string, minLevel string) []Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	q := strings.ToLower(query)
	var lvl Level
	if minLevel != "" {
		lvl = ParseLevel(minLevel)
	}

	start := (r.head - r.count + r.cap) % r.cap
	var result []Entry
	for i := range r.count {
		idx := (start + i) % r.cap
		e := r.entries[idx]
		if minLevel != "" && ParseLevel(e.Level) < lvl {
			continue
		}
		if strings.Contains(strings.ToLower(e.Message), q) ||
			strings.Contains(strings.ToLower(e.Subsystem), q) ||
			strings.Contains(strings.ToLower(e.Raw), q) {
			result = append(result, e)
		}
	}
	return result
}

// Count returns the number of entries currently stored.
func (r *Ring) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.count
}
