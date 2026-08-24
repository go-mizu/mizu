package log

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/go-mizu/mizu/ctxdata"
	"github.com/go-mizu/mizu/errs"
)

// JSONOptions configures [NewJSONHandler]. The zero value writes everything
// from debug up, with the usual secrets masked and no stacks.
type JSONOptions struct {
	// Level is the lowest level that gets written, and defaults to debug.
	Level slog.Leveler

	// TimeFormat is the layout the time is written in, and defaults to RFC 3339
	// with milliseconds.
	TimeFormat string

	// AddSource adds the file and line the record was made at, as a source
	// member holding file:line.
	AddSource bool

	// Stack adds the stack of an error attribute, as a stack member holding one
	// frame per line. It applies only to records at error level, since a stack
	// costs more to store than the rest of the record put together.
	Stack bool

	// Redact is the attribute keys whose values are masked. Nil means
	// [DefaultRedact], and an empty non-nil slice means nothing is masked.
	Redact []string
}

// jsonHandler is the handler [NewJSONHandler] returns.
type jsonHandler struct {
	w      io.Writer
	mu     *sync.Mutex
	level  slog.Leveler
	format string
	source bool
	stack  bool
	redact redactor

	// prefix is the attributes from [slog.Logger.With], already encoded, with
	// any group they are inside already opened.
	prefix []byte

	// open is how many braces prefix leaves open.
	open int

	// pending is the groups from [slog.Logger.WithGroup] that nothing has been
	// written into yet. A group with no attributes in it is not written at all,
	// so they are opened when the first attribute arrives.
	pending []string
}

// NewJSONHandler writes one JSON object per record, for a machine to read.
//
//	{"time":"2026-08-24T10:44:02.113+02:00","level":"ERROR","msg":"job failed","job":"SendWelcome","err":"dial tcp: connection refused","err_kind":"unavailable","err_code":"mail.down"}
//
// The members come out in a fixed order, time, level, msg, source, and then the
// attributes in the order they were given, so a line diffs against another line
// and a human can read one when they have to.
//
// An attribute holding an error is expanded into the message, and the kind and
// code when anything classified it. With [JSONOptions.Stack], a record at error
// level also carries the stack the error was made with.
func NewJSONHandler(w io.Writer, o JSONOptions) slog.Handler {
	h := &jsonHandler{
		w:      w,
		mu:     new(sync.Mutex),
		level:  o.Level,
		format: o.TimeFormat,
		source: o.AddSource,
		stack:  o.Stack,
		redact: newRedactor(o.Redact),
	}
	if h.level == nil {
		h.level = slog.LevelDebug
	}
	if h.format == "" {
		h.format = "2006-01-02T15:04:05.000Z07:00"
	}
	return h
}

func (h *jsonHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level.Level()
}

func (h *jsonHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	next := *h
	next.prefix = slices.Clip(h.prefix)
	for _, g := range h.pending {
		next.prefix = appendMember(next.prefix, g)
		next.prefix = append(next.prefix, '{')
		next.open++
	}
	next.pending = nil
	for _, a := range attrs {
		next.prefix, _ = h.appendAttr(next.prefix, a, nil)
	}
	return &next
}

func (h *jsonHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	next := *h
	next.pending = append(slices.Clip(h.pending), name)
	return &next
}

func (h *jsonHandler) Handle(ctx context.Context, r slog.Record) error {
	buf := newBuffer()
	defer freeBuffer(buf)
	b := *buf

	b = append(b, '{')
	if !r.Time.IsZero() {
		b = appendMember(b, slog.TimeKey)
		b = append(b, '"')
		b = r.Time.AppendFormat(b, h.format)
		b = append(b, '"')
	}
	b = appendMember(b, slog.LevelKey)
	b = appendJSONString(b, r.Level.String())
	b = appendMember(b, slog.MessageKey)
	b = appendJSONString(b, r.Message)

	if h.source && r.PC != 0 {
		f, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		b = appendMember(b, slog.SourceKey)
		b = append(b, '"')
		b = appendJSONInside(b, f.File)
		b = append(b, ':')
		b = strconv.AppendInt(b, int64(f.Line), 10)
		b = append(b, '"')
	}

	for e := range ctxdata.All(ctx) {
		if !e.Logged {
			continue
		}
		if e.Redacted {
			b = appendMember(b, e.Name)
			b = appendJSONString(b, Mask)
			continue
		}
		b, _ = h.appendAttr(b, slog.Any(e.Name, e.Value), nil)
	}

	b = append(b, h.prefix...)

	// A group with nothing in it is not written, so the groups the logger was
	// given are opened when the first attribute that writes something arrives.
	open := h.open
	pending := h.pending
	var deepest error
	r.Attrs(func(a slog.Attr) bool {
		if !writes(a) {
			return true
		}
		for _, g := range pending {
			b = appendMember(b, g)
			b = append(b, '{')
			open++
		}
		pending = nil
		b, deepest = h.appendAttr(b, a, deepest)
		return true
	})

	if h.stack && deepest != nil && r.Level >= slog.LevelError {
		if frames := errs.Stack(deepest); len(frames) > 0 {
			b = appendMember(b, "stack")
			b = append(b, '"')
			for i, f := range frames {
				if i > 0 {
					b = append(b, '\\', 'n')
				}
				b = appendJSONInside(b, f.String())
			}
			b = append(b, '"')
		}
	}

	for range open {
		b = append(b, '}')
	}
	b = append(b, '}', '\n')

	*buf = b
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(b)
	return err
}

// appendAttr writes one attribute, and returns the error it wrote, which is the
// one the stack comes from. Whether that error has a stack is asked once, in
// Handle, since resolving one costs more than writing the rest of the record.
func (h *jsonHandler) appendAttr(b []byte, a slog.Attr, deepest error) ([]byte, error) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return b, deepest
	}

	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return b, deepest
		}
		// A group with no name is written into whatever object it is in,
		// which is what log/slog says an empty key means.
		if a.Key == "" {
			for _, sub := range attrs {
				b, deepest = h.appendAttr(b, sub, deepest)
			}
			return b, deepest
		}
		b = appendMember(b, a.Key)
		b = append(b, '{')
		for _, sub := range attrs {
			b, deepest = h.appendAttr(b, sub, deepest)
		}
		return append(b, '}'), deepest
	}

	if h.redact.hides(a.Key) {
		b = appendMember(b, a.Key)
		return appendJSONString(b, Mask), deepest
	}

	if err, ok := errorValue(a.Value); ok {
		b = appendMember(b, a.Key)
		b = appendJSONString(b, err.Error())
		if k, known := kindOf(err); known {
			b = appendSuffixMember(b, a.Key, "_kind")
			b = appendJSONString(b, k.String())
		}
		if code := errs.CodeOf(err); code != "" {
			b = appendSuffixMember(b, a.Key, "_code")
			b = appendJSONString(b, code)
		}
		return b, err
	}

	b = appendMember(b, a.Key)
	return appendJSONValue(b, a.Value), deepest
}

// writes is whether an attribute puts anything in the record, which decides
// whether a group waiting to be opened has anything to hold.
func writes(a slog.Attr) bool {
	if a.Value.Kind() == slog.KindGroup {
		return len(a.Value.Group()) > 0
	}
	return !a.Equal(slog.Attr{})
}

// appendMember writes the comma and the key, and knows not to write a comma
// as the first thing in an object.
func appendMember(b []byte, key string) []byte {
	return appendSuffixMember(b, key, "")
}

// appendSuffixMember is a member whose key is two strings, such as err and
// _kind. Joining them first would allocate a string per error per record.
func appendSuffixMember(b []byte, key, suffix string) []byte {
	if len(b) == 0 || b[len(b)-1] != '{' {
		b = append(b, ',')
	}
	b = append(b, '"')
	b = appendJSONInside(b, key)
	b = appendJSONInside(b, suffix)
	b = append(b, '"')
	return append(b, ':')
}

func appendJSONValue(b []byte, v slog.Value) []byte {
	switch v.Kind() {
	case slog.KindString:
		return appendJSONString(b, v.String())
	case slog.KindInt64:
		return strconv.AppendInt(b, v.Int64(), 10)
	case slog.KindUint64:
		return strconv.AppendUint(b, v.Uint64(), 10)
	case slog.KindFloat64:
		return appendJSONFloat(b, v.Float64())
	case slog.KindBool:
		return strconv.AppendBool(b, v.Bool())
	case slog.KindDuration:
		// Nanoseconds, the way log/slog writes one, so that a dashboard can do
		// arithmetic on it. The console handler is where 4.2ms belongs.
		return strconv.AppendInt(b, int64(v.Duration()), 10)
	case slog.KindTime:
		b = append(b, '"')
		b = v.Time().AppendFormat(b, time.RFC3339Nano)
		return append(b, '"')
	default:
		return appendJSONAny(b, v.Any())
	}
}

// appendJSONFloat writes a number, or a string for the three values JSON has no
// number for, since a log line that will not parse is worse than an odd value.
func appendJSONFloat(b []byte, f float64) []byte {
	switch {
	case f != f:
		return append(b, `"NaN"`...)
	case f > 1.7976931348623157e308:
		return append(b, `"+Inf"`...)
	case f < -1.7976931348623157e308:
		return append(b, `"-Inf"`...)
	}
	return strconv.AppendFloat(b, f, 'g', -1, 64)
}

// appendJSONAny writes a value this package has no case for, by asking
// encoding/json, and writes what it printed as a string when that fails.
func appendJSONAny(b []byte, v any) []byte {
	if v == nil {
		return append(b, "null"...)
	}
	out, err := json.Marshal(v)
	if err != nil {
		return appendJSONString(b, fmt.Sprintf("%+v", v))
	}
	return append(b, out...)
}

const hex = "0123456789abcdef"

func appendJSONString(b []byte, s string) []byte {
	b = append(b, '"')
	b = appendJSONInside(b, s)
	return append(b, '"')
}

// appendJSONInside escapes a string into the quotes somebody else wrote.
//
// It escapes what RFC 8259 requires and nothing else. HTML escaping, which
// encoding/json does by default, is for a document put inside a script tag and
// makes a log line harder to read for no gain.
func appendJSONInside(b []byte, s string) []byte {
	start := 0
	for i := 0; i < len(s); {
		if c := s[i]; c < utf8.RuneSelf {
			if c >= 0x20 && c != '"' && c != '\\' {
				i++
				continue
			}
			b = append(b, s[start:i]...)
			switch c {
			case '"':
				b = append(b, '\\', '"')
			case '\\':
				b = append(b, '\\', '\\')
			case '\n':
				b = append(b, '\\', 'n')
			case '\r':
				b = append(b, '\\', 'r')
			case '\t':
				b = append(b, '\\', 't')
			default:
				b = append(b, '\\', 'u', '0', '0', hex[c>>4], hex[c&0xf])
			}
			i++
			start = i
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			b = append(b, s[start:i]...)
			b = append(b, `�`...)
			i++
			start = i
			continue
		}
		i += size
	}
	return append(b, s[start:]...)
}
