package console

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
)

// The table is the only thing in this package that does real work per call.
// Everything else is a Fprintf, and what it costs is what the terminal costs.
//
// The size is the one that matters: route:list on a large application prints a
// few hundred rows, and that is the point at which a quadratic width
// calculation or a per-cell allocation would start to show.
func benchRows(n int) [][]string {
	rows := make([][]string, n)
	for i := range rows {
		rows[i] = []string{
			"GET",
			"/posts/{post}/comments/{comment}/replies",
			"post.comment.reply.show",
			"handler.Comments.Replies",
			strconv.Itoa(i),
		}
	}
	return rows
}

var benchHeaders = []string{"Method", "URI", "Name", "Handler", "Order"}

func BenchmarkTable(b *testing.B) {
	for _, n := range []int{10, 400} {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			rows := benchRows(n)
			io := New(strings.NewReader(""), io.Discard, io.Discard, Options{Color: ColorNever})

			b.ReportAllocs()
			for b.Loop() {
				io.Table(benchHeaders, rows)
			}
		})
	}
}

func BenchmarkTableJSON(b *testing.B) {
	rows := benchRows(400)
	io := New(strings.NewReader(""), io.Discard, io.Discard, Options{JSON: true})

	b.ReportAllocs()
	for b.Loop() {
		io.Table(benchHeaders, rows)
	}
}

// repeated hands out the same answer forever, so a benchmark can ask as many
// questions as it likes without building the answers first.
type repeated struct {
	answer string
	at     int
}

func (r *repeated) Read(p []byte) (int, error) {
	n := copy(p, r.answer[r.at:])
	r.at += n
	if r.at == len(r.answer) {
		r.at = 0
	}
	return n, nil
}

// BenchmarkAsk measures the machinery around a prompt rather than the prompt.
// What a real one costs is a person typing.
//
// What it does catch is the version that builds a buffered reader per question,
// which is four kilobytes each time and, worse, reads ahead and eats the answer
// to the next one.
func BenchmarkAsk(b *testing.B) {
	io := New(&repeated{answer: "Ada\n"}, io.Discard, io.Discard, Options{Interaction: InteractionAlways})

	b.ReportAllocs()
	for b.Loop() {
		if _, err := io.Ask("Name", ""); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAdvance is what a loop pays for reporting its own progress. A
// command that deletes a million rows advances a bar a million times, and the
// answer should be that it never notices.
//
// The total is large enough that no step crosses into a new tenth, so this is
// the cost of the lock and the arithmetic rather than the cost of a line.
func BenchmarkAdvance(b *testing.B) {
	io := New(strings.NewReader(""), io.Discard, io.Discard, Options{})
	bar := io.Progress(1 << 40)

	b.ReportAllocs()
	for b.Loop() {
		bar.Advance(1)
	}
}

// BenchmarkInfo is the cost of a status line, and of the same line when
// --quiet turned it off. The second one is what a loop that reports progress
// pays when nobody is reading, which should be close to nothing.
func BenchmarkInfo(b *testing.B) {
	for _, tt := range []struct {
		name string
		opts Options
	}{
		{"on", Options{Color: ColorNever}},
		{"quiet", Options{Verbosity: Quiet}},
	} {
		b.Run(tt.name, func(b *testing.B) {
			io := New(strings.NewReader(""), io.Discard, io.Discard, tt.opts)

			b.ReportAllocs()
			for b.Loop() {
				io.Info("processed %d of %d", 4211, 10000)
			}
		})
	}
}
