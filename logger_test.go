// logger_test.go
package mizu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

func newCtxForLoggerTest(method, url string, rw io.Writer) (*Ctx, *httptest.ResponseRecorder, *http.Request, *slog.Logger) {
	r := httptest.NewRequest(method, url, nil)
	w := httptest.NewRecorder()
	bufLogger := slog.New(slog.NewTextHandler(rw, &slog.HandlerOptions{Level: slog.LevelInfo}))
	return newCtx(w, r, bufLogger), w, r, bufLogger
}

func Test_normalizeLoggerOptions_Defaults(t *testing.T) {
	got := normalizeLoggerOptions(LoggerOptions{})
	if got.RequestIDHeader != "X-Request-Id" {
		t.Fatalf("RequestIDHeader default, got %q", got.RequestIDHeader)
	}
	if got.Output == nil {
		t.Fatal("Output should default to os.Stderr")
	}
}

func Test_statusOrOK(t *testing.T) {
	if statusOrOK(0) != http.StatusOK {
		t.Fatal("statusOrOK(0) != 200")
	}
	if statusOrOK(204) != 204 {
		t.Fatal("statusOrOK(204) != 204")
	}
}

func Test_requestPath_EscapedAndPlain(t *testing.T) {
	r1 := httptest.NewRequest(http.MethodGet, "http://x/a%20b", nil)
	if got := requestPath(r1); got != "/a%20b" {
		t.Fatalf("escaped path, got %q", got)
	}
	r2 := httptest.NewRequest(http.MethodGet, "http://x", nil)
	r2.URL.Opaque = ""
	r2.URL.Path = "/plain"
	if got := requestPath(r2); got != "/plain" {
		t.Fatalf("plain path, got %q", got)
	}
}

func Test_levelFor(t *testing.T) {
	if got := levelFor(200, nil); got != slog.LevelInfo {
		t.Fatalf("level 200 -> %v", got)
	}
	if got := levelFor(404, nil); got != slog.LevelWarn {
		t.Fatalf("level 404 -> %v", got)
	}
	if got := levelFor(500, nil); got != slog.LevelError {
		t.Fatalf("level 500 -> %v", got)
	}
	if got := levelFor(200, io.EOF); got != slog.LevelError {
		t.Fatalf("error present -> %v", got)
	}
}

func Test_humanDuration_Branches(t *testing.T) {
	ns := 100 * time.Nanosecond
	us := 50 * time.Microsecond
	ms := 20 * time.Millisecond
	s := 2 * time.Second
	if !strings.HasSuffix(humanDuration(ns), "ns") {
		t.Fatalf("ns suffix, got %q", humanDuration(ns))
	}
	if !strings.HasSuffix(humanDuration(us), "µs") {
		t.Fatalf("us suffix, got %q", humanDuration(us))
	}
	if !strings.HasSuffix(humanDuration(ms), "ms") {
		t.Fatalf("ms suffix, got %q", humanDuration(ms))
	}
	if !strings.HasSuffix(humanDuration(s), "s") {
		t.Fatalf("s suffix, got %q", humanDuration(s))
	}
}

func Test_resolveMode_Branches(t *testing.T) {
	if resolveMode(Dev, &bytes.Buffer{}) != Dev {
		t.Fatal("explicit Dev should win")
	}
	if resolveMode(Prod, &bytes.Buffer{}) != Prod {
		t.Fatal("explicit Prod should win")
	}
	if resolveMode(Auto, &bytes.Buffer{}) != Prod {
		t.Fatal("Auto + non-terminal should be Prod")
	}
	_ = isTerminal(os.Stderr)
}

func Test_supportsColorEnv_And_forceColorOn(t *testing.T) {
	oldNo := os.Getenv("NO_COLOR")
	oldForce := os.Getenv("FORCE_COLOR")
	defer func() {
		_ = os.Setenv("NO_COLOR", oldNo)
		_ = os.Setenv("FORCE_COLOR", oldForce)
	}()

	_ = os.Setenv("NO_COLOR", "1")
	if supportsColorEnv() {
		t.Fatal("NO_COLOR=1 should disable color")
	}
	_ = os.Setenv("NO_COLOR", "0")
	_ = os.Setenv("FORCE_COLOR", "1")
	if !supportsColorEnv() || !forceColorOn() {
		t.Fatal("FORCE_COLOR=1 should enable color")
	}
	_ = os.Setenv("FORCE_COLOR", "0")
	if runtime.GOOS == "windows" && supportsColorEnv() {
		t.Fatal("Windows should not report supportsColorEnv without FORCE_COLOR")
	}
}

func Test_buildFallbackLogger_ProdJSON(t *testing.T) {
	var buf bytes.Buffer
	lg := buildFallbackLogger(Prod, &buf, false)
	lg.Info("hello", "status", 200)
	out := buf.String()
	var tmp map[string]any
	if err := json.Unmarshal([]byte(out), &tmp); err != nil {
		t.Fatalf("expected JSON log, got %q, err=%v", out, err)
	}
}

func Test_buildFallbackLogger_DevColorAndText(t *testing.T) {
	oldNo := os.Getenv("NO_COLOR")
	oldForce := os.Getenv("FORCE_COLOR")
	defer func() {
		_ = os.Setenv("NO_COLOR", oldNo)
		_ = os.Setenv("FORCE_COLOR", oldForce)
	}()

	var buf1 bytes.Buffer
	_ = os.Setenv("FORCE_COLOR", "1")
	lg1 := buildFallbackLogger(Dev, &buf1, true)
	lg1.Info("msg", slog.Int("status", 201))
	if s := buf1.String(); !strings.Contains(s, "\x1b[") {
		t.Fatalf("expected ANSI colored output, got %q", s)
	}

	var buf2 bytes.Buffer
	_ = os.Setenv("FORCE_COLOR", "0")
	_ = os.Setenv("NO_COLOR", "1")
	lg2 := buildFallbackLogger(Dev, &buf2, false)
	lg2.Info("msg", slog.Int("status", 201))
	if s := buf2.String(); strings.Contains(s, "\x1b[") {
		t.Fatalf("did not expect ANSI color, got %q", s)
	}
	if strings.HasPrefix(strings.TrimSpace(buf2.String()), "{") {
		t.Fatalf("did not expect JSON in Dev plain path, got %q", buf2.String())
	}
}

func Test_selectLogger_PrefersProvidedAndCtx(t *testing.T) {
	var provided bytes.Buffer
	o := LoggerOptions{}
	o = normalizeLoggerOptions(o)
	prov := slog.New(slog.NewTextHandler(&provided, nil))
	got := selectLogger(nil, LoggerOptions{Logger: prov, Output: o.Output}, nil, false)
	if got != prov {
		t.Fatal("selectLogger should return provided logger")
	}

	var ctxBuf bytes.Buffer
	c, _, _, ctxLogger := newCtxForLoggerTest(http.MethodGet, "http://x", &ctxBuf)
	fallback := slog.New(slog.NewTextHandler(io.Discard, nil))
	got2 := selectLogger(c, LoggerOptions{Mode: Auto, Output: os.Stderr}, fallback, true)
	if got2 != ctxLogger {
		t.Fatal("selectLogger should use ctx logger in pureAutoNoPrefs")
	}

	got3 := selectLogger(c, LoggerOptions{Mode: Auto, Output: os.Stderr}, fallback, false)
	if got3 != fallback {
		t.Fatal("selectLogger should return fallback")
	}
}

func Test_resolveRequestID_AllPaths(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://x", nil)
	w := httptest.NewRecorder()
	c := newCtx(w, r, nil)
	r.Header.Set("X-Request-Id", "rid1")
	if got := resolveRequestID(r, c, LoggerOptions{RequestIDHeader: "X-Request-Id"}); got != "rid1" {
		t.Fatalf("resolve from req header got %q", got)
	}

	r2 := httptest.NewRequest(http.MethodGet, "http://x", nil)
	w2 := httptest.NewRecorder()
	c2 := newCtx(w2, r2, nil)
	c2.Header().Set("X-Request-Id", "rid2")
	if got := resolveRequestID(r2, c2, LoggerOptions{RequestIDHeader: "X-Request-Id"}); got != "rid2" {
		t.Fatalf("resolve from resp header got %q", got)
	}

	r3 := httptest.NewRequest(http.MethodGet, "http://x", nil)
	w3 := httptest.NewRecorder()
	c3 := newCtx(w3, r3, nil)
	genCalled := false
	gen := func() string { genCalled = true; return "rid3" }
	if got := resolveRequestID(r3, c3, LoggerOptions{RequestIDHeader: "X-Request-Id", RequestIDGen: gen}); got != "rid3" || !genCalled {
		t.Fatalf("resolve from generator failed, got %q called=%v", got, genCalled)
	}
	if v := c3.Header().Get("X-Request-Id"); v != "rid3" {
		t.Fatalf("generator should set header, got %q", v)
	}

	r4 := httptest.NewRequest(http.MethodGet, "http://x", nil)
	w4 := httptest.NewRecorder()
	c4 := newCtx(w4, r4, nil)
	if got := resolveRequestID(r4, c4, LoggerOptions{RequestIDHeader: "X-Request-Id"}); got != "" {
		t.Fatalf("resolveRequestID should return empty, got %q", got)
	}
}

/* ==== helpers to keep test cyclomatic complexity low ==== */

func attrValString(v slog.Value) string {
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindInt64:
		return fmt.Sprintf("%d", v.Int64())
	case slog.KindUint64:
		return fmt.Sprintf("%d", v.Uint64())
	case slog.KindFloat64:
		// keep simple formatting for readability in tests
		return fmt.Sprintf("%g", v.Float64())
	case slog.KindBool:
		if v.Bool() {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprint(v.Any())
	}
}

func attrsToStringMap(attrs []slog.Attr) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		m[a.Key] = attrValString(a.Value)
	}
	return m
}

/* ========================================================= */

func Test_buildLogAttrs_AllFlags(t *testing.T) {
	c, _, r, _ := newCtxForLoggerTest(http.MethodGet, "http://x.test/a?x=1", io.Discard)
	r.Proto = "HTTP/1.1"
	r.Host = "x.test"
	r.Header.Set("User-Agent", "UA/1.0")
	c.Header().Set("X-Request-Id", "ridX")

	attrs := buildLogAttrs(c, r, 201, 12*time.Millisecond, "", Dev, LoggerOptions{
		RequestIDHeader: "X-Request-Id",
		UserAgent:       true,
		TraceExtractor: func(ctx context.Context) (string, string, bool) {
			return "trace123", "span456", true
		},
	})

	got := attrsToStringMap(attrs)

	if got["status"] != "201" || got["method"] != "GET" || got["host"] != "x.test" {
		t.Fatalf("basic attrs missing: %#v", got)
	}
	if _, ok := got["latency_human"]; !ok {
		t.Fatalf("latency_human should be present in Dev mode")
	}
	if got["query"] != "x=1" {
		t.Fatalf("query attr missing")
	}
	if got["request_id"] != "ridX" {
		t.Fatalf("request_id not resolved from header: %v", got["request_id"])
	}
	if got["user_agent"] != "UA/1.0" {
		t.Fatalf("user_agent missing")
	}
	if got["trace_id"] != "trace123" || got["span_id"] != "span456" || got["trace_sampled"] != "true" {
		t.Fatalf("trace attrs missing: %#v", got)
	}
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

func Test_colorTextHandler_Enabled_Handle_AttrsAndGroups(t *testing.T) {
	var buf bytes.Buffer
	h := newColorTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	if h.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("debug should be disabled at info level")
	}
	if !h.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("info should be enabled")
	}

	h2 := h.WithAttrs([]slog.Attr{slog.String("base", "v")}).(*colorTextHandler)
	h3 := h2.WithGroup("x").(*colorTextHandler)

	now := time.Now()
	rec := slog.NewRecord(now, slog.LevelInfo, "request", 0)
	rec.AddAttrs(
		slog.Int("status", 201),
		slog.Uint64("u", 7),
		slog.Float64("f", 1.5),
		slog.String("k", "v"),
		slog.Any("", "skip"),
	)

	if err := h3.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected colored output, got %q", out)
	}

	plain := stripANSI(out)
	if !strings.Contains(plain, "INFO") {
		t.Fatalf("missing INFO in plain: %q", plain)
	}
	for _, want := range []string{"base=v", "f=1.5", "k=v", "status=201", "u=7"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("missing %q in plain: %q", want, plain)
		}
	}
}

func Test_Logger_Middleware_TextAndErrorLevels(t *testing.T) {
	var buf bytes.Buffer
	_ = os.Setenv("NO_COLOR", "1")
	defer func() { _ = os.Setenv("NO_COLOR", "") }()

	o := LoggerOptions{
		Mode:      Dev,
		Output:    &buf,
		UserAgent: true,
		TraceExtractor: func(ctx context.Context) (string, string, bool) {
			return "tid", "sid", false
		},
	}
	mw := Logger(o)

	h := mw(func(c *Ctx) error {
		c.Status(503)
		c.Header().Set("X-Request-Id", "abc123")
		return io.EOF
	})

	r := httptest.NewRequest(http.MethodGet, "http://x.test/p?q=1", nil)
	r.Header.Set("User-Agent", "UA")
	w := httptest.NewRecorder()
	c := newCtx(w, r, nil)
	_ = h(c)

	out := buf.String()
	if !strings.Contains(out, "status=503") || !strings.Contains(out, "request_id=abc123") {
		t.Fatalf("logged line missing fields: %q", out)
	}
	if !strings.Contains(out, "error=") {
		t.Fatalf("expected error field in log: %q", out)
	}
}

func Test_Logger_Middleware_SelectsCtxLogger_WhenPureAutoNoPrefs(t *testing.T) {
	var ctxBuf bytes.Buffer
	c, _, _, ctxLogger := newCtxForLoggerTest(http.MethodGet, "http://x.test/ctx", &ctxBuf)

	o := LoggerOptions{Mode: Auto, Output: os.Stderr}
	mw := Logger(o)

	next := mw(func(c *Ctx) error {
		c.Status(204)
		return nil
	})

	_ = next(c)
	out := ctxBuf.String()
	if out == "" {
		t.Fatalf("expected logs to be written via ctx logger")
	}
	if ctxLogger == nil {
		t.Fatalf("ctx logger not set")
	}
}

func Test_attrInt_AllKinds(t *testing.T) {
	if v, ok := attrInt(slog.Int("x", 1)); !ok || v != 1 {
		t.Fatal("attrInt int64")
	}
	if v, ok := attrInt(slog.Uint64("x", 2)); !ok || v != 2 {
		t.Fatal("attrInt uint64")
	}
	if v, ok := attrInt(slog.Float64("x", 3.2)); !ok || v != 3 {
		t.Fatal("attrInt float64")
	}
	if _, ok := attrInt(slog.String("x", "nope")); ok {
		t.Fatal("attrInt should fail for non-number")
	}
}

func Test_colorTextHandler_LevelColorBranches(t *testing.T) {
	var buf bytes.Buffer
	h := newColorTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})

	levels := []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}
	wantNames := []string{"DEBUG", "INFO", "WARN", "ERROR"}

	for i, lvl := range levels {
		rec := slog.NewRecord(time.Now(), lvl, "msg", 0)
		rec.AddAttrs(slog.Int("status", 200))
		if err := h.Handle(context.Background(), rec); err != nil {
			t.Fatalf("Handle error for %v: %v", lvl, err)
		}
		out := buf.String()
		if !strings.Contains(out, wantNames[i]) {
			t.Fatalf("expected %q in output for %v, got %q", wantNames[i], lvl, out)
		}
		buf.Reset()
	}
}
