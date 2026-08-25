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
