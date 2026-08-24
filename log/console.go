package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu/ctxdata"
	"github.com/go-mizu/mizu/errs"
)

// Color says when the console handler writes escape sequences.
type Color uint8

const (
	// ColorAuto writes colour when the writer is a terminal, NO_COLOR is unset
	// and TERM is not dumb. It is the zero value.
	ColorAuto Color = iota

	// ColorAlways writes colour whatever the writer is, which is what a test
	// of the colour output wants.
	ColorAlways

	// ColorNever writes none.
	ColorNever
)

// ConsoleOptions configures [NewConsoleHandler]. The zero value is what a
// developer wants: everything from debug up, aligned, coloured when the
// terminal can take it, with the usual secrets masked.
type ConsoleOptions struct {
	// Level is the lowest level that gets written, and defaults to debug. A
	// [slog.LevelVar] here can be turned up while the process runs.
	Level slog.Leveler

	// TimeFormat is how the time of a record is written, in the layout
	// [time.Time.Format] takes. It defaults to 15:04:05.000, since a developer
	// reading a terminal knows what day it is.
	TimeFormat string

	// Color says whether to write colour. The zero value decides by looking at
	// the writer and the environment.
	Color Color

	// MsgWidth is the column the attributes line up at, and defaults to 28. A
	// message longer than that pushes its own attributes right rather than
	// truncating.
	MsgWidth int

	// AddSource adds the file and line the record was made at, as a source
	// attribute.
	AddSource bool

	// Redact is the attribute keys whose values are masked. Nil means
	// [DefaultRedact]. An empty non-nil slice means nothing is masked, which is
	// a thing to write on purpose rather than to arrive at by leaving a field
	// out.
	Redact []string
}

// console is the handler [NewConsoleHandler] returns.
type console struct {
	w      io.Writer
	mu     *sync.Mutex
	level  slog.Leveler
	format string
	width  int
	color  bool
	source bool
	redact redactor

	// prefix is the attributes from [slog.Logger.With], already formatted, so
	// a logger that carries ten attributes does not format them per record.
	prefix []byte

	// group is the dotted path from [slog.Logger.WithGroup], which the console
	// writes as a prefix on the key rather than as nesting.
	group string
}

// NewConsoleHandler writes records the way a person reads them: one line each,
// aligned, with the attributes after the message.
//
//	10:44:02.113 INF request                     method=GET path=/posts status=200 dur=4.2ms
//	10:44:02.118 WRN slow query                  dur=812ms rows=1204
//	10:44:02.119 ERR job failed                  job=SendWelcome attempt=3 err="dial tcp: connection refused"
//	             └─ /src/blog/job/welcome.go:41 blog/job.SendWelcome.Run
//
// An attribute holding an error is expanded: the message, and the kind and code
// when anything classified it. At error level, the line under the record is
// where the error was made, from the stack [errs] captured.
//
// This is the handler for a terminal. It is not the one to run in production,
// where [NewJSONHandler] writes something a machine can read.
func NewConsoleHandler(w io.Writer, o ConsoleOptions) slog.Handler {
	h := &console{
		w:      w,
		mu:     new(sync.Mutex),
		level:  o.Level,
		format: o.TimeFormat,
		width:  o.MsgWidth,
		color:  wantColor(w, o.Color),
		source: o.AddSource,
		redact: newRedactor(o.Redact),
	}
	if h.level == nil {
		h.level = slog.LevelDebug
	}
	if h.format == "" {
		h.format = "15:04:05.000"
	}
	if h.width == 0 {
		h.width = 28
	}
	return h
}

func (h *console) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level.Level()
}

func (h *console) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	next := *h
	next.prefix = make([]byte, len(h.prefix), len(h.prefix)+64)
	copy(next.prefix, h.prefix)
	for _, a := range attrs {
		next.prefix, _ = h.appendAttr(next.prefix, h.group, a, nil)
	}
	return &next
}

func (h *console) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	next := *h
	next.group = h.group + name + "."
	return &next
}

func (h *console) Handle(ctx context.Context, r slog.Record) error {
	buf := newBuffer()
	defer freeBuffer(buf)
	b := *buf

	indent := 0
	if !r.Time.IsZero() {
		start := len(b)
		b = h.paint(b, dim)
		b = r.Time.AppendFormat(b, h.format)
		b = h.reset(b)
		b = append(b, ' ')
		indent = len(b) - start - h.painted()
	}

	b = h.paint(b, levelColor(r.Level))
	b = append(b, levelTag(r.Level)...)
	b = h.reset(b)
	b = append(b, ' ')

	b = append(b, r.Message...)
	message := len(b)
	for range h.width - len(r.Message) {
		b = append(b, ' ')
	}
	padded := len(b)

	for e := range ctxdata.All(ctx) {
		if !e.Logged {
			continue
		}
		if e.Redacted {
			b = h.appendKey(b, "", e.Name)
			b = append(b, Mask...)
			continue
		}
		b, _ = h.appendAttr(b, "", slog.Any(e.Name, e.Value), nil)
	}

	b = append(b, h.prefix...)

	// deepest is the error the stack line comes from, which is the last error
	// attribute in the record. A record rarely has two.
	var deepest error
	r.Attrs(func(a slog.Attr) bool {
		b, deepest = h.appendAttr(b, h.group, a, deepest)
		return true
	})

	if h.source && r.PC != 0 {
		f, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		b = h.appendKey(b, "", "source")
		b = append(b, f.File...)
		b = append(b, ':')
		b = strconv.AppendInt(b, int64(f.Line), 10)
	}

	// The message is padded so that the attributes line up. A record with no
	// attributes has nothing to line up with, and the padding would be
	// trailing whitespace.
	if len(b) == padded {
		b = b[:message]
	}

	b = append(b, '\n')

	if deepest != nil && r.Level >= slog.LevelError {
		if frames := errs.Stack(deepest); len(frames) > 0 {
			for range indent {
				b = append(b, ' ')
			}
			b = h.paint(b, dim)
			b = append(b, "└─ "...)
			b = append(b, frames[0].File...)
			b = append(b, ':')
			b = strconv.AppendInt(b, int64(frames[0].Line), 10)
			b = append(b, ' ')
			b = append(b, frames[0].Func...)
			b = h.reset(b)
			b = append(b, '\n')
		}
	}

	*buf = b
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(b)
	return err
}

// appendAttr writes one attribute, and returns the error it wrote, which is the
// one the stack line under the record comes from. Whether that error has a
// stack is asked once, in Handle, since resolving one costs more than writing
// the rest of the record.
func (h *console) appendAttr(b []byte, group string, a slog.Attr, deepest error) ([]byte, error) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return b, deepest
	}

	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return b, deepest
		}
		if a.Key != "" {
			group += a.Key + "."
		}
		for _, sub := range attrs {
			b, deepest = h.appendAttr(b, group, sub, deepest)
		}
		return b, deepest
	}

	if h.redact.hides(a.Key) {
		b = h.appendKey(b, group, a.Key)
		return append(b, Mask...), deepest
	}

	if err, ok := errorValue(a.Value); ok {
		b = h.appendKey(b, group, a.Key)
		b = h.paint(b, red)
		b = appendConsoleString(b, err.Error())
		b = h.reset(b)
		if k, known := kindOf(err); known {
			b = h.appendSuffixKey(b, group, a.Key, "_kind")
			b = append(b, k.String()...)
		}
		if code := errs.CodeOf(err); code != "" {
			b = h.appendSuffixKey(b, group, a.Key, "_code")
			b = append(b, code...)
		}
		return b, err
	}

	b = h.appendKey(b, group, a.Key)
	return appendConsoleValue(b, a.Value), deepest
}

// appendKey writes the separator and the key, which is where the colour goes
// so that the eye finds the values.
func (h *console) appendKey(b []byte, group, key string) []byte {
	return h.appendSuffixKey(b, group, key, "")
}

// appendSuffixKey is a key made of two strings, such as err and _kind. Joining
// them first would allocate a string per error per record.
func (h *console) appendSuffixKey(b []byte, group, key, suffix string) []byte {
	b = append(b, ' ')
	b = h.paint(b, dim)
	b = append(b, group...)
	b = append(b, key...)
	b = append(b, suffix...)
	b = append(b, '=')
	return h.reset(b)
}

func appendConsoleValue(b []byte, v slog.Value) []byte {
	switch v.Kind() {
	case slog.KindString:
		return appendConsoleString(b, v.String())
	case slog.KindInt64:
		return strconv.AppendInt(b, v.Int64(), 10)
	case slog.KindUint64:
		return strconv.AppendUint(b, v.Uint64(), 10)
	case slog.KindFloat64:
		return strconv.AppendFloat(b, v.Float64(), 'g', -1, 64)
	case slog.KindBool:
		return strconv.AppendBool(b, v.Bool())
	case slog.KindDuration:
		return append(b, v.Duration().String()...)
	case slog.KindTime:
		return v.Time().AppendFormat(b, time.RFC3339Nano)
	default:
		return appendConsoleString(b, fmt.Sprint(v.Any()))
	}
}

// appendConsoleString quotes only when it has to, since a line of quoted words
// is harder to read than a line of words.
func appendConsoleString(b []byte, s string) []byte {
	if s == "" {
		return append(b, `""`...)
	}
	if strings.ContainsAny(s, " \t\r\n\"=") || !isPrintable(s) {
		return strconv.AppendQuote(b, s)
	}
	return append(b, s...)
}

func isPrintable(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

// errorValue is the error an attribute holds, if it holds one.
func errorValue(v slog.Value) (error, bool) {
	if v.Kind() != slog.KindAny {
		return nil, false
	}
	err, ok := v.Any().(error)
	return err, ok
}

const (
	reset  = "\033[0m"
	dim    = "\033[90m"
	red    = "\033[31m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
)

func levelColor(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return red
	case l >= slog.LevelWarn:
		return yellow
	case l >= slog.LevelInfo:
		return cyan
	default:
		return dim
	}
}

func (h *console) paint(b []byte, color string) []byte {
	if !h.color {
		return b
	}
	return append(b, color...)
}

func (h *console) reset(b []byte) []byte {
	if !h.color {
		return b
	}
	return append(b, reset...)
}

// painted is how many bytes of a coloured field are escape sequences rather
// than something on the screen, which is what the stack line indents past.
func (h *console) painted() int {
	if !h.color {
		return 0
	}
	return len(dim) + len(reset)
}

// wantColor decides whether the writer can take escape sequences.
func wantColor(w io.Writer, c Color) bool {
	switch c {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	}
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	return isTerminal(w)
}
