package warc_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/warc"
)

func TestWriter_WriteAndReadBack(t *testing.T) {
	var buf bytes.Buffer
	w := warc.NewWriter(&buf)

	rec := &warc.Record{
		Header: warc.Header{
			"WARC-Type":       "response",
			"WARC-Target-URI": "https://example.com/",
			"WARC-Record-ID":  "<urn:uuid:test-456>",
			"Content-Length":  "5",
		},
		Body: strings.NewReader("hello"),
	}
	if err := w.WriteRecord(rec); err != nil {
		t.Fatalf("WriteRecord: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Read back
	r := warc.NewReader(&buf)
	if !r.Next() {
		t.Fatalf("expected record, err=%v", r.Err())
	}
	got := r.Record()
	if got.Header.Type() != warc.TypeResponse {
		t.Errorf("Type=%q", got.Header.Type())
	}
	if got.Header.TargetURI() != "https://example.com/" {
		t.Errorf("TargetURI=%q", got.Header.TargetURI())
	}
	body, _ := io.ReadAll(got.Body)
	if string(body) != "hello" {
		t.Errorf("body=%q, want %q", body, "hello")
	}
}

func TestWriter_MultiRecord_RoundTrip(t *testing.T) {
	urls := []string{"https://a.com/", "https://b.com/", "https://c.com/"}
	var buf bytes.Buffer
	w := warc.NewWriter(&buf)

	for _, u := range urls {
		body := "body for " + u
		rec := &warc.Record{
			Header: warc.Header{
				"WARC-Type":       "response",
				"WARC-Target-URI": u,
				"Content-Length":  fmt.Sprintf("%d", len(body)),
			},
			Body: strings.NewReader(body),
		}
		if err := w.WriteRecord(rec); err != nil {
			t.Fatalf("WriteRecord %s: %v", u, err)
		}
	}
	w.Close()

	r := warc.NewReader(&buf)
	count := 0
	for r.Next() {
		count++
		io.Copy(io.Discard, r.Record().Body)
	}
	if r.Err() != nil {
		t.Fatalf("unexpected error: %v", r.Err())
	}
	if count != len(urls) {
		t.Errorf("expected %d records, got %d", len(urls), count)
	}
}
