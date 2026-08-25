package mizutest

import (
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"
)

// bodyLimit is how much of a body a failure message prints. Two kilobytes is
// enough to see an error page or the head of a list and short enough that three
// failures in one test still fit on a screen.
const bodyLimit = 2 << 10

// Dump prints the whole exchange through the test log: the request with its
// headers and body, the response with its headers and body, and everything the
// handler logged.
//
// It is a debugging tool rather than an assertion, and a call to it left in a
// passing test prints nothing, since go test hides the log of a test that
// passed unless asked with -v.
func (r *Response) Dump() *Response {
	r.tb.Helper()
	r.tb.Log("\n" + r.exchange())
	return r
}

// DumpHeaders prints the request and response headers alone, for when the
// bodies are large and beside the point.
func (r *Response) DumpHeaders() *Response {
	r.tb.Helper()

	var b strings.Builder
	r.writeRequestLine(&b)
	writeHeaders(&b, "> ", r.req.Header)
	r.writeStatusLine(&b)
	writeHeaders(&b, "< ", r.res.Header)
	r.tb.Log("\n" + b.String())
	return r
}

// DumpJSON prints the response body as indented JSON with its members sorted,
// which is what a body is usually being read for.
func (r *Response) DumpJSON() *Response {
	r.tb.Helper()
	doc, err := decodeJSON(r.body)
	if err != nil {
		r.tb.Logf("the body is not JSON (%v), and is:\n%s", err, r.Text())
		return r
	}
	r.tb.Log("\n" + strings.TrimPrefix(pretty(doc), "\n"))
	return r
}

// DumpLogs prints everything the handler logged during this request.
func (r *Response) DumpLogs() *Response {
	r.tb.Helper()
	if len(r.logs) == 0 {
		r.tb.Log("the handler logged nothing")
		return r
	}

	var b strings.Builder
	for _, e := range r.logs {
		b.WriteString(e.String())
		b.WriteByte('\n')
	}
	r.tb.Log("\n" + b.String())
	return r
}

// Logs is what the handler logged during this request, for asserting about it
// rather than reading it.
func (r *Response) Logs() []Entry { return slices.Clone(r.logs) }

// exchange is the whole thing as text, and is what the first failure on a
// response prints. Requests are prefixed with > and responses with <, which is
// the convention curl -v uses and one most people have already read.
func (r *Response) exchange() string {
	var b strings.Builder

	r.writeRequestLine(&b)
	writeHeaders(&b, "> ", r.req.Header)
	writeBody(&b, "> ", r.sent)

	r.writeStatusLine(&b)
	writeHeaders(&b, "< ", r.res.Header)
	writeBody(&b, "< ", r.body)

	// Errors first and in full, because the reason for an unexpected 500 is
	// almost always one of them. The rest is a count, since a request log line
	// per request is noise once the error is in view.
	var others int
	for _, e := range r.logs {
		if e.Level >= slog.LevelError {
			fmt.Fprintf(&b, "log %s\n", e)
			continue
		}
		others++
	}
	switch others {
	case 0:
	case 1:
		b.WriteString("log 1 more entry below error level, see DumpLogs\n")
	default:
		fmt.Fprintf(&b, "log %d more entries below error level, see DumpLogs\n", others)
	}
	return b.String()
}

func (r *Response) writeRequestLine(b *strings.Builder) {
	fmt.Fprintf(b, "> %s %s\n", r.req.Method, r.req.URL.RequestURI())
}

func (r *Response) writeStatusLine(b *strings.Builder) {
	fmt.Fprintf(b, "< %d %s\n", r.res.StatusCode, http.StatusText(r.res.StatusCode))
}

func writeHeaders(b *strings.Builder, prefix string, h http.Header) {
	for _, k := range slices.Sorted(maps.Keys(h)) {
		for _, v := range h[k] {
			fmt.Fprintf(b, "%s%s: %s\n", prefix, k, v)
		}
	}
}

// writeBody prints a body, cut at bodyLimit and cut at a rune boundary so the
// output is not left holding half a character.
func writeBody(b *strings.Builder, prefix string, body []byte) {
	if len(body) == 0 {
		return
	}

	shown, cut := body, 0
	if len(shown) > bodyLimit {
		cut = len(shown) - bodyLimit
		shown = shown[:bodyLimit]
		for len(shown) > 0 && !utf8.Valid(shown) {
			shown = shown[:len(shown)-1]
			cut++
		}
	}

	b.WriteString(prefix)
	b.WriteString(strings.ReplaceAll(strings.TrimRight(string(shown), "\n"), "\n", "\n"+prefix))
	b.WriteByte('\n')
	if cut > 0 {
		fmt.Fprintf(b, "%s... and %d more bytes\n", prefix, cut)
	}
}

// indent shifts a block one level in, so that the exchange under a failure
// message reads as part of it rather than as the next thing that happened.
func indent(s string) string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return ""
	}
	return "    " + strings.ReplaceAll(s, "\n", "\n    ")
}
