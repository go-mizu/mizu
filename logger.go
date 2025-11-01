package mizu

import (
	"context"
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
	RequestIDGen    func() string
	UserAgent       bool
	Output          io.Writer
	TraceExtractor  func(ctx context.Context) (traceID, spanID string, sampled bool)
}

// Logger returns a middleware that logs one line per request with useful fields.
func Logger(o LoggerOptions) Middleware {
	o = normalizeLoggerOptions(o)

	effectiveMode := resolveMode(o.Mode, o.Output)
	fallback := buildFallbackLogger(effectiveMode, o.Output, o.Color)
	pureAutoNoPrefs := o.Mode == Auto && !o.Color && o.Output == os.Stderr && o.Logger == nil

	return func(next Handler) Handler {
		return func(c *Ctx) error {
			start := time.Now()
			r := c.Request()

			lg := selectLogger(c, o, fallback, pureAutoNoPrefs)
			reqID := resolveRequestID(r, c, o)

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
		if forceColorOn() || (color && supportsColorEnv()) {
			return slog.New(newColorTextHandler(out, &slog.HandlerOptions{Level: slog.LevelInfo}))
		}
		return slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: slog.LevelInfo}))
	default:
		return slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
}

func selectLogger(c *Ctx, o LoggerOptions, fallback *slog.Logger, pureAutoNoPrefs bool) *slog.Logger {
	if o.Logger != nil {
		return o.Logger
	}
	if pureAutoNoPrefs {
		if ctxLog := c.Logger(); ctxLog != nil {
			return ctxLog
		}
	}
	return fallback
}

func resolveRequestID(r *http.Request, c *Ctx, o LoggerOptions) string {
	// read from request or response header
	if v := r.Header.Get(o.RequestIDHeader); v != "" {
		return v
	}
	if v := c.Header().Get(o.RequestIDHeader); v != "" {
		return v
	}

	// generate if configured
	if o.RequestIDGen != nil {
		if id := o.RequestIDGen(); id != "" {
			c.Header().Set(o.RequestIDHeader, id)
			return id
		}
	}

	// after handler run, response header might be set; call site rechecks
	return ""
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
	attrs := []slog.Attr{
		slog.Int("status", status),
		slog.String("method", r.Method),
		slog.String("path", requestPath(r)),
		slog.String("proto", r.Proto),
		slog.String("host", r.Host),
		slog.Int64("duration_ms", d.Milliseconds()),
		slog.String("remote_ip", c.ClientIP()),
	}
	if mode == Dev {
		attrs = append(attrs, slog.String("latency_human", humanDuration(d)))
	}
	if q := r.URL.RawQuery; q != "" {
		attrs = append(attrs, slog.String("query", q))
	}
	if reqID == "" {
		if v := c.Header().Get(o.RequestIDHeader); v != "" {
			reqID = v
		}
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
		if tid, sid, sampled := o.TraceExtractor(r.Context()); tid != "" {
			attrs = append(attrs, slog.String("trace_id", tid))
			if sid != "" {
				attrs = append(attrs, slog.String("span_id", sid))
			}
			attrs = append(attrs, slog.Bool("trace_sampled", sampled))
		}
	}
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

func forceColorOn() bool { return os.Getenv("FORCE_COLOR") == "1" }

func supportsColorEnv() bool {
	if os.Getenv("NO_COLOR") == "1" {
		return false
	}
	if forceColorOn() {
		return true
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

func (h *colorTextHandler) Handle(_ context.Context, r slog.Record) error {
	all := make([]slog.Attr, 0, len(h.attrs)+r.NumAttrs())
	all = append(all, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		all = append(all, a)
		return true
	})
	sort.SliceStable(all, func(i, j int) bool { return all[i].Key < all[j].Key })

	const (
		gray   = "\x1b[90m"
		bold   = "\x1b[1m"
		reset  = "\x1b[0m"
		cyan   = "\x1b[36m"
		green  = "\x1b[32m"
		yellow = "\x1b[33m"
		red    = "\x1b[31m"
	)

	levelName, levelColor := func(l slog.Level) (string, string) {
		switch {
		case l <= slog.LevelDebug:
			return "DEBUG", cyan + bold
		case l == slog.LevelInfo:
			return "INFO", green + bold
		case l == slog.LevelWarn:
			return "WARN", yellow + bold
		default:
			return "ERROR", red + bold
		}
	}(r.Level)

	ts := r.Time.Format(time.Kitchen)

	var b strings.Builder
	if ts != "" {
		b.WriteString(gray)
		b.WriteString(ts)
		b.WriteString(reset)
		b.WriteByte(' ')
	}
	b.WriteString(levelColor)
	b.WriteString(levelName)
	b.WriteString(reset)
	if r.Message != "" {
		b.WriteByte(' ')
		b.WriteString(r.Message)
	}

	for _, a := range all {
		if a.Key == "" {
			continue
		}
		b.WriteByte(' ')
		if a.Key == "status" {
			if code, ok := attrInt(a); ok {
				switch {
				case code < 300:
					b.WriteString(green)
				case code < 500:
					b.WriteString(yellow)
				default:
					b.WriteString(red)
				}
				b.WriteString(fmt.Sprintf("%s=%d", a.Key, code))
				b.WriteString(reset)
				continue
			}
		}
		b.WriteString(gray)
		b.WriteString(a.Key)
		b.WriteString("=")
		b.WriteString(reset)
		b.WriteString(fmt.Sprint(a.Value))
	}
	b.WriteByte('\n')

	_, err := io.WriteString(h.w, b.String())
	return err
}

func (h *colorTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cp := *h
	cp.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &cp
}

func (h *colorTextHandler) WithGroup(_ string) slog.Handler {
	cp := *h
	return &cp
}

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
