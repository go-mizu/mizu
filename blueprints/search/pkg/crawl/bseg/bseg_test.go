package bseg_test

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/bseg"
)

// tempFile creates a temp file, returns it open for read/write. Caller must close.
func tempFile(t *testing.T) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "bseg_test_*.bseg2")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	return f
}

// writeRecords encodes records into f via bseg.Encoder, closes it, then reopens for reading.
func writeRecords(t *testing.T, f *os.File, records []*bseg.Record) *os.File {
	t.Helper()
	enc, err := bseg.NewEncoder(f, 0)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	for i, r := range records {
		if err := enc.Encode(r); err != nil {
			t.Fatalf("Encode[%d]: %v", i, err)
		}
	}
	if err := enc.Close(); err != nil {
		t.Fatalf("Encoder.Close: %v", err)
	}

	// Reopen the file for reading.
	rf, err := os.Open(f.Name())
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	return rf
}

func TestRoundTrip(t *testing.T) {
	want := []*bseg.Record{
		{
			Failed:      false,
			StatusCode:  200,
			ContentLen:  123456,
			FetchMs:     99,
			CrawledMs:   1700000000000,
			URL:         "https://example.com/page",
			ContentType: "text/html",
			BodyCID:     "bafybeiexample",
			Title:       "Example Page",
			Description: "A test page",
			Language:    "en",
			Domain:      "example.com",
			RedirectURL: "",
			Error:       "",
		},
		{
			// All string fields empty.
			Failed:     false,
			StatusCode: 404,
			ContentLen: 0,
			FetchMs:    5,
			CrawledMs:  42,
		},
		{
			Failed:      true,
			StatusCode:  0,
			ContentLen:  0,
			FetchMs:     3001,
			CrawledMs:   999,
			URL:         "https://dead.example/",
			ContentType: "",
			BodyCID:     "",
			Title:       "",
			Description: "",
			Language:    "",
			Domain:      "dead.example",
			RedirectURL: "",
			Error:       "timeout",
		},
	}

	f := tempFile(t)
	rf := writeRecords(t, f, want)
	defer rf.Close()

	dec, err := bseg.NewDecoder(rf)
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}

	hdr := dec.Header()
	if hdr.Version != bseg.Version {
		t.Errorf("header version: got %d, want %d", hdr.Version, bseg.Version)
	}
	if hdr.RecCount != uint32(len(want)) {
		t.Errorf("header rec_count: got %d, want %d", hdr.RecCount, len(want))
	}

	var got bseg.Record
	for i, w := range want {
		if err := dec.Decode(&got); err != nil {
			t.Fatalf("Decode[%d]: %v", i, err)
		}
		if got != *w {
			t.Errorf("record[%d] mismatch:\n  got  %+v\n  want %+v", i, got, *w)
		}
	}

	// Next read must return io.EOF.
	if err := dec.Decode(&got); err != io.EOF {
		t.Errorf("expected io.EOF after last record, got %v", err)
	}
}

func TestStringTruncation(t *testing.T) {
	longTitle := strings.Repeat("x", 70000)
	rec := &bseg.Record{
		StatusCode: 200,
		Title:      longTitle,
		URL:        "https://example.com/",
		Domain:     "example.com",
	}

	f := tempFile(t)
	rf := writeRecords(t, f, []*bseg.Record{rec})
	defer rf.Close()

	dec, err := bseg.NewDecoder(rf)
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}

	var got bseg.Record
	if err := dec.Decode(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if len(got.Title) != 65535 {
		t.Errorf("title length: got %d, want 65535", len(got.Title))
	}
	if got.Title != longTitle[:65535] {
		t.Error("title content mismatch after truncation")
	}
}

func TestEmptyFile(t *testing.T) {
	f := tempFile(t)
	rf := writeRecords(t, f, nil)
	defer rf.Close()

	dec, err := bseg.NewDecoder(rf)
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}

	hdr := dec.Header()
	if hdr.RecCount != 0 {
		t.Errorf("rec_count: got %d, want 0", hdr.RecCount)
	}

	var got bseg.Record
	if err := dec.Decode(&got); err != io.EOF {
		t.Errorf("expected io.EOF on empty file, got %v", err)
	}
}

func TestMultipleRecords(t *testing.T) {
	const N = 1000
	records := make([]*bseg.Record, N)
	for i := range records {
		records[i] = &bseg.Record{
			StatusCode:  int32(200 + i%5),
			ContentLen:  int64(i * 1024),
			FetchMs:     int64(i % 500),
			CrawledMs:   int64(1700000000000 + int64(i)*1000),
			URL:         strings.Repeat("u", i%200),
			ContentType: "text/html",
			Title:       strings.Repeat("t", i%100),
			Domain:      "example.com",
			Failed:      i%7 == 0,
			Error:       func() string {
				if i%7 == 0 {
					return "timeout"
				}
				return ""
			}(),
		}
	}

	f := tempFile(t)
	rf := writeRecords(t, f, records)
	defer rf.Close()

	dec, err := bseg.NewDecoder(rf)
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}

	hdr := dec.Header()
	if hdr.RecCount != N {
		t.Errorf("rec_count: got %d, want %d", hdr.RecCount, N)
	}

	var got bseg.Record
	for i, w := range records {
		if err := dec.Decode(&got); err != nil {
			t.Fatalf("Decode[%d]: %v", i, err)
		}
		if got != *w {
			t.Errorf("record[%d] mismatch:\n  got  %+v\n  want %+v", i, got, *w)
		}
	}

	if err := dec.Decode(&got); err != io.EOF {
		t.Errorf("expected io.EOF after %d records, got %v", N, err)
	}
}
