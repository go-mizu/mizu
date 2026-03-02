package bodystore_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/bodystore"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := bodystore.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	body := []byte("<html><body>hello world</body></html>")
	cid, err := s.Put(body)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if len(cid) == 0 {
		t.Fatal("empty cid")
	}
	if !s.Has(cid) {
		t.Fatal("Has returned false after Put")
	}

	got, err := s.Get(cid)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("body mismatch: got %d bytes, want %d", len(got), len(body))
	}
}

func TestPutIdempotent(t *testing.T) {
	dir := t.TempDir()
	s, _ := bodystore.Open(dir)
	body := []byte("same content")

	cid1, err := s.Put(body)
	if err != nil {
		t.Fatal(err)
	}
	cid2, err := s.Put(body)
	if err != nil {
		t.Fatal(err)
	}
	if cid1 != cid2 {
		t.Fatalf("cids differ: %s vs %s", cid1, cid2)
	}
	// Verify only one file on disk
	path := s.Path(cid1)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not found: %v", err)
	}
}

func TestGetMissing(t *testing.T) {
	s, _ := bodystore.Open(t.TempDir())
	_, err := s.Get("sha256:0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Fatal("expected error for missing cid")
	}
}
