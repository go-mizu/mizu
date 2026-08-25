package golden

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A golden assertion runs once per test rather than in a loop, so none of this
// is on a hot path. The numbers are here to say what a test pays for reading a
// file and normalising it, which is the difference between a suite of a
// thousand golden assertions costing a moment and costing a coffee.

var sink []byte

// corpus is a JSON document of roughly the size of an API response, which is
// what most of these assertions compare.
func corpus(items int) []byte {
	var b bytes.Buffer
	b.WriteString(`{"data":[`)
	for i := range items {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"id":%d,"name":"user %d","email":"u%d@example.com","active":%v}`,
			i, i, i, i%2 == 0)
	}
	b.WriteString(`],"total":`)
	fmt.Fprintf(&b, "%d}", items)
	return b.Bytes()
}

func BenchmarkAssert(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "BenchmarkAssert.golden")
	body := corpus(50)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		b.Fatal(err)
	}

	r := &recorder{name: "BenchmarkAssert"}
	b.ReportAllocs()

	for b.Loop() {
		Assert(r, body, Dir(dir))
	}
	if r.failed {
		b.Fatal(r.msg)
	}
}

func BenchmarkAssertJSON(b *testing.B) {
	dir := b.TempDir()
	body := corpus(50)

	r := &recorder{name: "BenchmarkAssertJSON"}
	*update = true
	AssertJSON(r, body, Dir(dir))
	*update = false
	if r.failed {
		b.Fatal(r.msg)
	}

	b.ReportAllocs()
	for b.Loop() {
		AssertJSON(r, body, Dir(dir))
	}
	if r.failed {
		b.Fatal(r.msg)
	}
}

func BenchmarkNormalizeJSON(b *testing.B) {
	body := corpus(50)
	b.ReportAllocs()

	for b.Loop() {
		sink = normalizeJSON(b, body)
	}
}

func BenchmarkNormalizeSQL(b *testing.B) {
	query := []byte(strings.Repeat("select id,\n       name\n  from users\n where id = ? and name = 'a  b'\n", 8))
	b.ReportAllocs()

	for b.Loop() {
		sink = normalizeSQL(b, query)
	}
}

func BenchmarkScrub(b *testing.B) {
	var o options
	ScrubUUIDs()(&o)
	ScrubTimes()(&o)
	ScrubDurations()(&o)

	line := []byte(strings.Repeat(
		"id=f81d4fae-7dec-11d0-a765-00a0c91e6bf6 at=2026-08-23T10:00:00Z took=1.5ms\n", 20))
	b.ReportAllocs()

	for b.Loop() {
		sink = o.scrub(line)
	}
}

// BenchmarkDiff is the failure path, which only runs when a test is already
// failing. It is here so that a golden file of a few thousand lines does not
// turn one failure into a long wait.
func BenchmarkDiff(b *testing.B) {
	want := []byte(strings.Repeat("a line of generated output\n", 2000))
	got := append(bytes.Clone(want[:len(want)/2]), []byte("something else\n")...)
	got = append(got, want[len(want)/2:]...)

	b.ReportAllocs()
	for b.Loop() {
		sink = []byte(diff("x.golden", want, got))
	}
}
