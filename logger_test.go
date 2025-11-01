// logger_test.go
package mizu

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func doReqLog(t *testing.T, h http.Handler, method, target string, hdr map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestLogger_Prod_JSONFields(t *testing.T) {
	var buf bytes.Buffer

	r := NewRouter()
	r.Use(Logger(LoggerOptions{
		Mode:            Prod,
		Output:          &buf,
		UserAgent:       true,
		RequestIDHeader: "X-Request-Id",
		TraceExtractor: func(ctx context.Context) (string, string, bool) {
			return "trace-123", "span-456", true
		},
	}))
	r.Get("/p", func(c *Ctx) error {
		// Use a Ctx helper so StatusCode() reflects 201 for the logger
		return c.Text(201, "ok")
	})

	hdr := map[string]string{
		"User-Agent":   "ua-test",
		"X-Request-Id": "abc-1",
	}
	rr := doReqLog(t, r, http.MethodGet, "/p?q=1", hdr)
	if rr.Code != 201 {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	// Parse the JSON log line
	line := firstNonEmptyLine(buf.String())
	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("failed to unmarshal log json: %v\nline: %s", err, line)
	}

	// Required fields
	assertJSONInt(t, m, "status", 201)
	assertJSONString(t, m, "method", "GET")
	assertJSONString(t, m, "path", "/p")
	assertJSONString(t, m, "host", "example.com")
	// assertPresent(t, m, "duration_ms")
	assertJSONString(t, m, "request_id", "abc-1")
	assertJSONString(t, m, "user_agent", "ua-test")
	assertJSONString(t, m, "query", "q=1")
	assertJSONString(t, m, "trace_id", "trace-123")
	assertJSONString(t, m, "span_id", "span-456")
	assertJSONBool(t, m, "trace_sampled", true)

	// latency_human only for Dev, should be absent
	if _, ok := m["latency_human"]; ok {
		t.Fatalf("latency_human should be absent in Prod")
	}
}

func TestLogger_Dev_AddsLatencyHuman(t *testing.T) {
	var buf bytes.Buffer
	r := NewRouter()
	r.Use(Logger(LoggerOptions{Mode: Dev, Output: &buf}))
	r.Get("/d", func(c *Ctx) error { return c.Text(200, "x") })

	rr := doReqLog(t, r, http.MethodGet, "/d", nil)
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	out := buf.String()
	if !strings.Contains(out, "latency_human=") {
		t.Fatalf("expected latency_human in Dev output, got: %s", out)
	}
}

func TestLogger_ErrorLevelAndAttr(t *testing.T) {
	var buf bytes.Buffer
	r := NewRouter()
	r.Use(Logger(LoggerOptions{Mode: Prod, Output: &buf}))
	r.Get("/e", func(c *Ctx) error {
		c.Response().WriteHeader(503)
		return errors.New("boom")
	})

	rr := doReqLog(t, r, http.MethodGet, "/e", nil)
	if rr.Code != 503 {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
	line := firstNonEmptyLine(buf.String())
	if !strings.Contains(line, `"error":"boom"`) {
		t.Fatalf("expected error attr in log, got: %s", line)
	}
}

func TestLogger_RequestIDGen_SetsHeaderAndLogs(t *testing.T) {
	var buf bytes.Buffer
	r := NewRouter()
	r.Use(Logger(LoggerOptions{
		Mode:         Prod,
		Output:       &buf,
		RequestIDGen: func() string { return "gen-1" },
	}))
	r.Get("/r", func(c *Ctx) error { return c.NoContent() })

	rr := doReqLog(t, r, http.MethodGet, "/r", nil)
	if rr.Header().Get("X-Request-Id") != "gen-1" {
		t.Fatalf("expected response header X-Request-Id=gen-1, got %q", rr.Header().Get("X-Request-Id"))
	}
	line := firstNonEmptyLine(buf.String())
	if !strings.Contains(line, `"request_id":"gen-1"`) {
		t.Fatalf("expected request_id in log, got: %s", line)
	}
}

func TestLogger_Precedence_ExplicitLoggerWins(t *testing.T) {
	var bufLogger bytes.Buffer
	var bufOutput bytes.Buffer // should remain unused

	explicit := slog.New(slog.NewJSONHandler(&bufLogger, &slog.HandlerOptions{Level: slog.LevelInfo}))

	r := NewRouter()
	r.Use(Logger(LoggerOptions{
		Logger: explicit,
		Mode:   Prod,
		Output: &bufOutput,
	}))
	r.Get("/x", func(c *Ctx) error { return c.NoContent() })

	_ = doReqLog(t, r, http.MethodGet, "/x", nil)

	if bufLogger.Len() == 0 {
		t.Fatalf("expected logs in explicit logger buffer")
	}
	if bufOutput.Len() != 0 {
		t.Fatalf("did not expect logs in Output when Logger is provided")
	}
}

func TestLogger_AutoWithNonTTYFallsToProdJSON(t *testing.T) {
	var buf bytes.Buffer
	r := NewRouter()
	r.Use(Logger(LoggerOptions{Mode: Auto, Output: &buf}))
	r.Get("/a", func(c *Ctx) error { return c.NoContent() })

	_ = doReqLog(t, r, http.MethodGet, "/a", nil)
	line := firstNonEmptyLine(buf.String())
	if len(line) == 0 || line[0] != '{' {
		t.Fatalf("expected JSON line in Auto with non-tty Output, got: %s", line)
	}
}

func TestLogger_ForceColorOnUsesColorTextHandler(t *testing.T) {
	t.Setenv("FORCE_COLOR", "1")
	defer t.Setenv("FORCE_COLOR", "")

	var buf bytes.Buffer
	r := NewRouter()
	r.Use(Logger(LoggerOptions{Mode: Dev, Output: &buf})) // Color flag not set, FORCE_COLOR should enable color handler
	r.Get("/c", func(c *Ctx) error { return c.NoContent() })

	_ = doReqLog(t, r, http.MethodGet, "/c", nil)
	out := buf.String()
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected ANSI sequences in Dev FORCE_COLOR output, got: %s", out)
	}
}

func TestHelpers_levelFor(t *testing.T) {
	if got := levelFor(200, nil); got != slog.LevelInfo {
		t.Fatalf("levelFor 200 nil -> %v", got)
	}
	if got := levelFor(404, nil); got != slog.LevelWarn {
		t.Fatalf("levelFor 404 nil -> %v", got)
	}
	if got := levelFor(200, errors.New("x")); got != slog.LevelError {
		t.Fatalf("levelFor err -> %v", got)
	}
	if got := levelFor(500, nil); got != slog.LevelError {
		t.Fatalf("levelFor 500 -> %v", got)
	}
}

func TestHelpers_humanDuration(t *testing.T) {
	if !strings.Contains(humanDuration(500*time.Nanosecond), "ns") {
		t.Fatalf("expected ns formatting")
	}
	if !strings.Contains(humanDuration(500*time.Microsecond), "µs") {
		t.Fatalf("expected µs formatting")
	}
	if !strings.Contains(humanDuration(5*time.Millisecond), "ms") {
		t.Fatalf("expected ms formatting")
	}
	if !strings.Contains(humanDuration(1500*time.Millisecond), "s") {
		t.Fatalf("expected s formatting")
	}
}

func TestHelpers_attrInt(t *testing.T) {
	a1 := slog.Int64("status", 201)
	if v, ok := attrInt(a1); !ok || v != 201 {
		t.Fatalf("attrInt int64 failed")
	}
	a2 := slog.Uint64("status", 202)
	if v, ok := attrInt(a2); !ok || v != 202 {
		t.Fatalf("attrInt uint64 failed")
	}
	a3 := slog.Float64("status", 203.0)
	if v, ok := attrInt(a3); !ok || v != 203 {
		t.Fatalf("attrInt float64 failed")
	}
}

func TestColorTextHandler_AllMethods(t *testing.T) {
	var buf bytes.Buffer
	h := newColorTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	if !h.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatalf("Enabled should be true for info")
	}
	h2 := h.WithAttrs([]slog.Attr{slog.String("a", "b")})
	_ = h.WithGroup("g") // no-op but should return a handler

	// Create a record and add status for color branch
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "request", 0)
	rec.AddAttrs(slog.Int("status", 200))
	if err := h2.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	out := stripANSI(buf.String())
	if !strings.Contains(out, "a=b") {
		t.Fatalf("expected WithAttrs attr present, got: %s", out)
	}
	if !strings.Contains(out, "status=200") {
		t.Fatalf("expected status colored entry")
	}
}

// helpers

func firstNonEmptyLine(s string) string {
	for _, ln := range strings.Split(s, "\n") {
		if strings.TrimSpace(ln) != "" {
			return ln
		}
	}
	return ""
}

func assertJSONString(t *testing.T, m map[string]any, key, want string) {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("missing key %q", key)
	}
	vs, ok := v.(string)
	if !ok || vs != want {
		t.Fatalf("key %q want %q got %#v", key, want, v)
	}
}

func assertJSONInt(t *testing.T, m map[string]any, key string, want int) {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("missing key %q", key)
	}
	switch n := v.(type) {
	case float64:
		if int(n) != want {
			t.Fatalf("key %q want %d got %v", key, want, v)
		}
	default:
		t.Fatalf("key %q type not number: %#v", key, v)
	}
}

func assertJSONBool(t *testing.T, m map[string]any, key string, want bool) {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("missing key %q", key)
	}
	b, ok := v.(bool)
	if !ok || b != want {
		t.Fatalf("key %q want %v got %#v", key, want, v)
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inEsc := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if !inEsc {
			if ch == 0x1b && i+1 < len(s) && s[i+1] == '[' {
				inEsc = true
				i++ // skip '['
				continue
			}
			b.WriteByte(ch)
			continue
		}
		// inside escape sequence, consume until a letter
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			inEsc = false
		}
	}
	return b.String()
}

func TestSupportsColorEnv_NoColorDisables(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("FORCE_COLOR", "")
	t.Setenv("TERM", "xterm-256color")
	if supportsColorEnv() {
		t.Fatalf("expected supportsColorEnv=false when NO_COLOR=1")
	}
}

func TestSupportsColorEnv_ForceColorEnables(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("FORCE_COLOR", "1")
	t.Setenv("TERM", "")
	if !supportsColorEnv() {
		t.Fatalf("expected supportsColorEnv=true when FORCE_COLOR=1")
	}
}

func TestSupportsColorEnv_DumbTermDisables(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("FORCE_COLOR", "")
	t.Setenv("TERM", "dumb")
	if supportsColorEnv() {
		t.Fatalf("expected supportsColorEnv=false for TERM=dumb")
	}
}

func TestSupportsColorEnv_TermBehavior(t *testing.T) {
	// This case is OS dependent. On Windows it should be false, on others true.
	t.Setenv("NO_COLOR", "")
	t.Setenv("FORCE_COLOR", "")
	t.Setenv("TERM", "xterm-256color")
	got := supportsColorEnv()
	if runtime.GOOS == "windows" {
		if got {
			t.Fatalf("expected supportsColorEnv=false on Windows")
		}
	} else {
		if !got {
			t.Fatalf("expected supportsColorEnv=true on non-Windows with TERM set")
		}
	}
}

func TestIsTerminal_FalseForNonFile(t *testing.T) {
	var buf bytes.Buffer
	if isTerminal(&buf) {
		t.Fatalf("expected isTerminal=false for non *os.File writer")
	}
}

func TestIsTerminal_FalseForRegularFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "not-a-tty")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	// Regular files are not character devices, should return false.
	if isTerminal(f) {
		t.Fatalf("expected isTerminal=false for regular file")
	}
}
