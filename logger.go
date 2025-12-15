package mizu

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Mode selects the output style used by the request logger.
type Mode uint8

const (
	// Auto picks Dev for terminals and Prod for non terminals.
	Auto Mode = iota
	// Prod writes JSON lines for log collectors.
	Prod
	// Dev writes readable text for local development.
	Dev
)

// LoggerOptions configures how each request is logged.
type LoggerOptions struct {
	Mode            Mode
	Color           bool
	Logger          *slog.Logger
	RequestIDHeader string

	// RequestIDGen overrides the built-in generator.
	// If nil, the logger generates a request id when missing.
	RequestIDGen func() string

	UserAgent      bool
	Output         io.Writer
	TraceExtractor func(ctx context.Context) (traceID, spanID string, sampled bool)
}

// Logger returns a middleware that logs one line per request with useful fields.
// It always ensures a request id is present in the response header, generating one if missing.
func Logger(o LoggerOptions) Middleware {
	o = normalizeLoggerOptions(o)

	effectiveMode := resolveMode(o.Mode, o.Output)
	fallback := buildFallbackLogger(effectiveMode, o.Output, o.Color)

	return func(next Handler) Handler {
		return func(c *Ctx) error {
			start := time.Now()
			r := c.Request()

			lg := selectLogger(c, o, fallback)

			// Ensure request id exists early and is attached to the response header.
			reqID := ensureRequestID(r, c, o)

			err := next(c)

			status := statusOrOK(c.StatusCode())
			dur := time.Since(start)
			attrs := buildLogAttrs(c, r, status, dur, reqID, effectiveMode, o)

			level := levelFor(status, err)
			if err != nil {
				attrs = append(attrs, slog.String("error", err.Error()))
			}
			lg.LogAttrs(r.Context(), level, "request", attrs...)
			return err
		}
	}
}

func normalizeLoggerOptions(o LoggerOptions) LoggerOptions {
	if o.RequestIDHeader == "" {
		o.RequestIDHeader = "X-Request-Id"
	}
	if o.Output == nil {
		o.Output = os.Stderr
	}
	return o
}

func buildFallbackLogger(mode Mode, out io.Writer, color bool) *slog.Logger {
	switch mode {
	case Prod:
		return slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{Level: slog.LevelInfo}))
	case Dev:
		if shouldColor(out, color) {
			return slog.New(newColorTextHandler(out, &slog.HandlerOptions{Level: slog.LevelInfo}))
		}
		return slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: slog.LevelInfo}))
	default:
		return slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
}

func selectLogger(c *Ctx, o LoggerOptions, fallback *slog.Logger) *slog.Logger {
	if o.Logger != nil {
		return o.Logger
	}
	if ctxLog := c.Logger(); ctxLog != nil {
		return ctxLog
	}
	return fallback
}

func ensureRequestID(r *http.Request, c *Ctx, o LoggerOptions) string {
	h := o.RequestIDHeader

	// Prefer incoming request header.
	if v := r.Header.Get(h); v != "" {
		if c.Header().Get(h) == "" {
			c.Header().Set(h, v)
		}
		return v
	}

	// If already set by earlier middleware, use it.
	if v := c.Header().Get(h); v != "" {
		return v
	}

	// Generate if missing.
	gen := o.RequestIDGen
	if gen == nil {
		gen = defaultRequestID
	}
	id := strings.TrimSpace(gen())
	if id == "" {
		id = defaultRequestID()
	}
	c.Header().Set(h, id)
	return id
}

func defaultRequestID() string {
	// 128-bit random id, hex encoded (32 chars).
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}

	// Very unlikely fallback if crypto/rand fails.
	now := time.Now().UnixNano()
	buf := make([]byte, 8)
	for i := 0; i < 8; i++ {
		buf[i] = byte(now >> (8 * i))
	}
	return hex.EncodeToString(buf)
}

func statusOrOK(code int) int {
	if code == 0 {
		return http.StatusOK
	}
	return code
}

func requestPath(r *http.Request) string {
	if p := r.URL.EscapedPath(); p != "" {
		return p
	}
	return r.URL.Path
}

func buildLogAttrs(c *Ctx, r *http.Request, status int, d time.Duration, reqID string, mode Mode, o LoggerOptions) []slog.Attr {
	attrs := buildCoreLogAttrs(c, r, status, d)
	if mode == Dev {
		attrs = append(attrs, slog.String("latency_human", humanDuration(d)))
	}
	attrs = appendOptionalLogAttrs(attrs, c, r, reqID, o)
	return attrs
}

func buildCoreLogAttrs(c *Ctx, r *http.Request, status int, d time.Duration) []slog.Attr {
	return []slog.Attr{
		slog.Int("status", status),
		slog.String("method", r.Method),
		slog.String("path", requestPath(r)),
		slog.String("proto", r.Proto),
		slog.String("host", r.Host),
		slog.Int64("duration_ms", d.Milliseconds()),
		slog.String("remote_ip", c.ClientIP()),
	}
}

func appendOptionalLogAttrs(attrs []slog.Attr, c *Ctx, r *http.Request, reqID string, o LoggerOptions) []slog.Attr {
	if q := r.URL.RawQuery; q != "" {
		attrs = append(attrs, slog.String("query", q))
	}
	if reqID != "" {
		attrs = append(attrs, slog.String("request_id", reqID))
	}
	if o.UserAgent {
		if ua := r.UserAgent(); ua != "" {
			attrs = append(attrs, slog.String("user_agent", ua))
		}
	}
	if o.TraceExtractor != nil {
		attrs = appendTraceAttrs(attrs, r, o.TraceExtractor)
	}
	return attrs
}

func appendTraceAttrs(attrs []slog.Attr, r *http.Request, extractor func(ctx context.Context) (traceID, spanID string, sampled bool)) []slog.Attr {
	tid, sid, sampled := extractor(r.Context())
	if tid == "" {
		return attrs
	}
	attrs = append(attrs, slog.String("trace_id", tid))
	if sid != "" {
		attrs = append(attrs, slog.String("span_id", sid))
	}
	attrs = append(attrs, slog.Bool("trace_sampled", sampled))
	return attrs
}

func resolveMode(m Mode, out io.Writer) Mode {
	if m != Auto {
		return m
	}
	if isTerminal(out) {
		return Dev
	}
	return Prod
}

func shouldColor(out io.Writer, prefer bool) bool {
	if !isTerminal(out) {
		return false
	}
	if _, ok := os.LookupEnv("NO_COLOR"); ok && !forceColorOn() {
		return false
	}
	if forceColorOn() {
		return true
	}
	if !prefer {
		return false
	}
	return supportsColorEnv()
}

func forceColorOn() bool { return os.Getenv("FORCE_COLOR") == "1" }

func supportsColorEnv() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	if runtime.GOOS == "windows" {
		return false
	}
	term := os.Getenv("TERM")
	return term != "" && term != "dumb"
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func levelFor(status int, err error) slog.Level {
	switch {
	case err != nil || status >= 500:
		return slog.LevelError
	case status >= 400:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

func humanDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.3fµs", float64(d)/float64(time.Microsecond))
	}
	if d < time.Second {
		return fmt.Sprintf("%.3fms", float64(d)/float64(time.Millisecond))
	}
	return fmt.Sprintf("%.3fs", float64(d)/float64(time.Second))
}

// colorTextHandler prints simple colored log lines for Dev mode.
type colorTextHandler struct {
	w     io.Writer
	opts  *slog.HandlerOptions
	attrs []slog.Attr
}

func newColorTextHandler(w io.Writer, opts *slog.HandlerOptions) *colorTextHandler {
	return &colorTextHandler{w: w, opts: opts}
}

func (h *colorTextHandler) Enabled(_ context.Context, level slog.Level) bool {
	min := slog.LevelInfo
	if h.opts.Level != nil {
		min = h.opts.Level.Level()
	}
	return level >= min
}

// ANSI color constants for terminal output.
const (
	colorGray   = "\x1b[90m"
	colorBold   = "\x1b[1m"
	colorReset  = "\x1b[0m"
	colorCyan   = "\x1b[36m"
	colorGreen  = "\x1b[32m"
	colorYellow = "\x1b[33m"
	colorRed    = "\x1b[31m"
)

func (h *colorTextHandler) Handle(_ context.Context, r slog.Record) error {
	all := h.collectAttrs(r)

	// Stable output helps humans scan logs.
	sort.SliceStable(all, func(i, j int) bool { return all[i].Key < all[j].Key })

	var b strings.Builder
	h.writeTimestamp(&b, r.Time)
	h.writeLevel(&b, r.Level)
	h.writeMessage(&b, r.Message)
	h.writeAttrs(&b, all)
	b.WriteByte('\n')

	_, err := io.WriteString(h.w, b.String())
	return err
}

func (h *colorTextHandler) collectAttrs(r slog.Record) []slog.Attr {
	all := make([]slog.Attr, 0, len(h.attrs)+r.NumAttrs())
	all = append(all, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		all = append(all, a)
		return true
	})
	return all
}

func (h *colorTextHandler) writeTimestamp(b *strings.Builder, t time.Time) {
	// Include seconds for easier debugging.
	ts := t.Format("15:04:05")
	b.WriteString(colorGray)
	b.WriteString(ts)
	b.WriteString(colorReset)
	b.WriteByte(' ')
}

func (h *colorTextHandler) writeLevel(b *strings.Builder, level slog.Level) {
	levelName, levelColor := levelNameAndColor(level)
	b.WriteString(levelColor)
	b.WriteString(levelName)
	b.WriteString(colorReset)
}

func levelNameAndColor(l slog.Level) (string, string) {
	switch {
	case l <= slog.LevelDebug:
		return "DEBUG", colorCyan + colorBold
	case l == slog.LevelInfo:
		return "INFO", colorGreen + colorBold
	case l == slog.LevelWarn:
		return "WARN", colorYellow + colorBold
	default:
		return "ERROR", colorRed + colorBold
	}
}

func (h *colorTextHandler) writeMessage(b *strings.Builder, msg string) {
	if msg == "" {
		return
	}
	b.WriteByte(' ')
	b.WriteString(msg)
}

func (h *colorTextHandler) writeAttrs(b *strings.Builder, attrs []slog.Attr) {
	for _, a := range attrs {
		if a.Key == "" {
			continue
		}
		b.WriteByte(' ')
		if a.Key == "status" {
			if h.writeStatusAttr(b, a) {
				continue
			}
		}
		b.WriteString(colorGray)
		b.WriteString(a.Key)
		b.WriteString("=")
		b.WriteString(colorReset)
		fmt.Fprint(b, a.Value)
	}
}

func (h *colorTextHandler) writeStatusAttr(b *strings.Builder, a slog.Attr) bool {
	code, ok := attrInt(a)
	if !ok {
		return false
	}
	b.WriteString(statusColor(code))
	fmt.Fprintf(b, "%s=%d", a.Key, code)
	b.WriteString(colorReset)
	return true
}

func statusColor(code int) string {
	switch {
	case code < 300:
		return colorGreen
	case code < 500:
		return colorYellow
	default:
		return colorRed
	}
}

func (h *colorTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cp := *h
	cp.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &cp
}

func (h *colorTextHandler) WithGroup(_ string) slog.Handler {
	// Groups are intentionally ignored to keep output compact.
	cp := *h
	return &cp
}

//nolint:gosec // G115: HTTP status codes are always safe to convert to int
func attrInt(a slog.Attr) (int, bool) {
	switch a.Value.Kind() {
	case slog.KindInt64:
		return int(a.Value.Int64()), true
	case slog.KindUint64:
		return int(a.Value.Uint64()), true
	case slog.KindFloat64:
		return int(a.Value.Float64()), true
	}
	return 0, false
}
