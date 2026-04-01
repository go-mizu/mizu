package mizu

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	_ = os.Unsetenv(key)
	t.Cleanup(func() { _ = os.Unsetenv(key) })
}

func TestDefaultRequestID_HexLenAndCharset(t *testing.T) {
	id := defaultRequestID()
	if len(id) != 32 {
		t.Fatalf("want 32 hex chars, got %d: %q", len(id), id)
	}
	for _, c := range id {
		ok := ('0' <= c && c <= '9') || ('a' <= c && c <= 'f')
		if !ok {
			t.Fatalf("id must be lowercase hex, got %q in %q", c, id)
		}
	}
}

func TestHumanDuration_AllBranches(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{500 * time.Nanosecond, "ns"},
		{50 * time.Microsecond, "µs"},
		{50 * time.Millisecond, "ms"},
		{2 * time.Second, "s"},
	}
	for _, tt := range tests {
		got := humanDuration(tt.d)
		if !strings.Contains(got, tt.want) {
			t.Fatalf("humanDuration(%v) = %q, want contains %q", tt.d, got, tt.want)
		}
	}
}

func TestStatusColor(t *testing.T) {
	if got := statusColor(200); got != colorGreen {
		t.Fatalf("200 => %q, want %q", got, colorGreen)
	}
	if got := statusColor(302); got != colorYellow {
		t.Fatalf("302 => %q, want %q", got, colorYellow)
	}
	if got := statusColor(404); got != colorYellow {
		t.Fatalf("404 => %q, want %q", got, colorYellow)
	}
	if got := statusColor(500); got != colorRed {
		t.Fatalf("500 => %q, want %q", got, colorRed)
	}
}

func TestLevelFor(t *testing.T) {
	if got := levelFor(200, nil); got != slog.LevelInfo {
		t.Fatalf("200 nil => %v, want info", got)
	}
	if got := levelFor(404, nil); got != slog.LevelWarn {
		t.Fatalf("404 nil => %v, want warn", got)
	}
	if got := levelFor(500, nil); got != slog.LevelError {
		t.Fatalf("500 nil => %v, want error", got)
	}
	if got := levelFor(200, errSentinel{}); got != slog.LevelError {
		t.Fatalf("200 err => %v, want error", got)
	}
}

type errSentinel struct{}

func (errSentinel) Error() string { return "boom" }

func TestResolveMode_AutoToProdWhenNonTerminal(t *testing.T) {
	var buf bytes.Buffer
	if got := resolveMode(Auto, &buf); got != Prod {
		t.Fatalf("Auto non terminal => %v, want Prod", got)
	}
	if got := resolveMode(Dev, &buf); got != Dev {
		t.Fatalf("explicit Dev => %v, want Dev", got)
	}
	if got := resolveMode(Prod, &buf); got != Prod {
		t.Fatalf("explicit Prod => %v, want Prod", got)
	}
}

func TestSupportsColorEnv_NO_COLORDisables(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("NO_COLOR", "anything")
	if supportsColorEnv() {
		t.Fatalf("NO_COLOR set should disable color")
	}
}

func TestSupportsColorEnv_TERMAndWindows(t *testing.T) {
	// Important: supportsColorEnv checks LookupEnv("NO_COLOR"),
	// so NO_COLOR must be unset, not set to "".
	unsetEnv(t, "NO_COLOR")
	t.Setenv("TERM", "xterm-256color")

	if runtime.GOOS == "windows" {
		if supportsColorEnv() {
			t.Fatalf("windows should report false in supportsColorEnv")
		}
		return
	}

	if !supportsColorEnv() {
		t.Fatalf("non-windows with TERM set should report true")
	}

	t.Setenv("TERM", "dumb")
	if supportsColorEnv() {
		t.Fatalf("TERM=dumb should disable color")
	}
}

func TestShouldColor_NonTerminalNeverColors(t *testing.T) {
	var buf bytes.Buffer

	unsetEnv(t, "NO_COLOR")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("FORCE_COLOR", "1")

	if got := shouldColor(&buf, true); got {
		t.Fatalf("non terminal writer should never color")
	}
	if got := shouldColor(&buf, false); got {
		t.Fatalf("non terminal writer should never color even when prefer=false")
	}
}

func TestAttrInt_AllKinds(t *testing.T) {
	if v, ok := attrInt(slog.Int64("x", -2)); !ok || v != -2 {
		t.Fatalf("int64 => %v %v, want -2 true", v, ok)
	}
	if v, ok := attrInt(slog.Uint64("x", 7)); !ok || v != 7 {
		t.Fatalf("uint64 => %v %v, want 7 true", v, ok)
	}
	if v, ok := attrInt(slog.Float64("x", 3.9)); !ok || v != 3 {
		t.Fatalf("float64 => %v %v, want 3 true (trunc)", v, ok)
	}
	if v, ok := attrInt(slog.String("x", "nope")); ok || v != 0 {
		t.Fatalf("string => %v %v, want 0 false", v, ok)
	}
}

func TestColorTextHandler_Handle_SortsAndColorsStatus(t *testing.T) {
	var out bytes.Buffer
	h := newColorTextHandler(&out, &slog.HandlerOptions{Level: slog.LevelInfo})

	rec := slog.NewRecord(time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC), slog.LevelInfo, "request", 0)
	// Deliberately out of order.
	rec.AddAttrs(
		slog.String("z", "last"),
		slog.Int("status", 404),
		slog.String("a", "first"),
	)

	if err := h.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle error: %v", err)
	}

	raw := out.String()
	plain := stripANSI(raw)

	// Has level and message.
	if !strings.Contains(plain, "INFO") || !strings.Contains(plain, "request") {
		t.Fatalf("output missing INFO/message: %q", plain)
	}

	// Status should be present and colored (yellow for 4xx).
	if !strings.Contains(plain, "status=404") {
		t.Fatalf("output missing status: %q", plain)
	}
	if !strings.Contains(raw, colorYellow) {
		t.Fatalf("output missing 4xx color: %q", raw)
	}

	// Attributes are sorted by key, so "a=" should appear before "z=".
	ai := strings.Index(plain, "a=")
	zi := strings.Index(plain, "z=")
	if ai == -1 || zi == -1 || ai > zi {
		t.Fatalf("attrs not sorted: %q", plain)
	}
}

func TestColorTextHandler_Enabled_RespectsMinLevel(t *testing.T) {
	var out bytes.Buffer
	h := newColorTextHandler(&out, &slog.HandlerOptions{Level: slog.LevelWarn})

	if h.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatalf("info should be disabled when min=warn")
	}
	if !h.Enabled(context.Background(), slog.LevelWarn) {
		t.Fatalf("warn should be enabled when min=warn")
	}
}

func TestColorTextHandler_WithAttrs_CopiesAndAppends(t *testing.T) {
	var out bytes.Buffer
	h := newColorTextHandler(&out, &slog.HandlerOptions{Level: slog.LevelInfo})

	h2 := h.WithAttrs([]slog.Attr{slog.String("k", "v")})
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "m", 0)

	if err := h2.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle error: %v", err)
	}

	plain := stripANSI(out.String())
	if !strings.Contains(plain, "k=v") {
		t.Fatalf("WithAttrs did not include base attrs, got %q", plain)
	}
}

func TestColorTextHandler_WithGroup_IgnoredButReturnsHandler(t *testing.T) {
	var out bytes.Buffer
	h := newColorTextHandler(&out, &slog.HandlerOptions{Level: slog.LevelInfo})

	h2 := h.WithGroup("ignored")
	if h2 == nil {
		t.Fatalf("WithGroup must return a handler")
	}
}

func TestWriteTimestamp_FormatIncludesSeconds(t *testing.T) {
	var out bytes.Buffer
	h := newColorTextHandler(&out, &slog.HandlerOptions{Level: slog.LevelInfo})

	rec := slog.NewRecord(time.Date(2025, 1, 1, 9, 8, 7, 0, time.UTC), slog.LevelInfo, "m", 0)
	_ = h.Handle(context.Background(), rec)

	plain := stripANSI(out.String())
	if !strings.Contains(plain, "09:08:07") {
		t.Fatalf("timestamp should include seconds, got %q", plain)
	}
}
