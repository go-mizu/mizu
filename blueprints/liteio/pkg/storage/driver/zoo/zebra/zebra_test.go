package zebra

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
)

func newTestStore(t *testing.T, opts string) *store {
	t.Helper()
	dir := t.TempDir()
	dsn := fmt.Sprintf("zebra:///%s?%s", dir, opts)
	d := &driver{}
	s, err := d.Open(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	return s.(*store)
}

func TestWriteReadRoundTrip(t *testing.T) {
	s := newTestStore(t, "stripes=4&sync=none&inline_kb=4")
	defer s.Close()

	b := s.Bucket("test")
	data := bytes.Repeat([]byte("x"), 1024)
	_, err := b.Write(context.Background(), "key1", bytes.NewReader(data), int64(len(data)), "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}

	rc, obj, err := b.Open(context.Background(), "key1", 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("data mismatch: got %d bytes, want %d", len(got), len(data))
	}
	if obj.ContentType != "text/plain" {
		t.Fatalf("content type: got %q, want %q", obj.ContentType, "text/plain")
	}
}

func TestInlineSmallValues(t *testing.T) {
	s := newTestStore(t, "stripes=4&sync=none&inline_kb=4")
	defer s.Close()

	b := s.Bucket("test")

	// Write 100 small values (inline path).
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%04d", i)
		data := []byte(fmt.Sprintf("value-%d", i))
		_, err := b.Write(context.Background(), key, bytes.NewReader(data), int64(len(data)), "text/plain", nil)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Read all back.
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%04d", i)
		expected := []byte(fmt.Sprintf("value-%d", i))

		rc, _, err := b.Open(context.Background(), key, 0, 0, nil)
		if err != nil {
			t.Fatalf("read %s: %v", key, err)
		}
		got, _ := io.ReadAll(rc)
		rc.Close()

		if !bytes.Equal(got, expected) {
			t.Fatalf("key %s: got %q, want %q", key, got, expected)
		}
	}
}

func TestLargeValueBypassesInline(t *testing.T) {
	s := newTestStore(t, "stripes=2&sync=none&inline_kb=1") // 1KB inline max
	defer s.Close()

	b := s.Bucket("test")
	// 2KB value should go through volume, not inline.
	data := bytes.Repeat([]byte("L"), 2048)
	_, err := b.Write(context.Background(), "big", bytes.NewReader(data), int64(len(data)), "application/octet-stream", nil)
	if err != nil {
		t.Fatal(err)
	}

	rc, obj, err := b.Open(context.Background(), "big", 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()

	got, _ := io.ReadAll(rc)
	if !bytes.Equal(got, data) {
		t.Fatalf("data mismatch: got %d bytes, want %d", len(got), len(data))
	}
	if obj.Size != 2048 {
		t.Fatalf("size: got %d, want 2048", obj.Size)
	}
}

func TestDeleteAndStat(t *testing.T) {
	s := newTestStore(t, "stripes=4&sync=none&inline_kb=4")
	defer s.Close()

	b := s.Bucket("test")
	data := []byte("hello")
	b.Write(context.Background(), "del-key", bytes.NewReader(data), int64(len(data)), "text/plain", nil)

	_, err := b.Stat(context.Background(), "del-key", nil)
	if err != nil {
		t.Fatal("stat before delete:", err)
	}

	if err := b.Delete(context.Background(), "del-key", nil); err != nil {
		t.Fatal("delete:", err)
	}

	_, err = b.Stat(context.Background(), "del-key", nil)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestCopy(t *testing.T) {
	s := newTestStore(t, "stripes=4&sync=none&inline_kb=4")
	defer s.Close()

	b := s.Bucket("test")
	data := []byte("copy-me")
	b.Write(context.Background(), "src", bytes.NewReader(data), int64(len(data)), "text/plain", nil)

	_, err := b.Copy(context.Background(), "dst", "test", "src", nil)
	if err != nil {
		t.Fatal("copy:", err)
	}

	rc, _, err := b.Open(context.Background(), "dst", 0, 0, nil)
	if err != nil {
		t.Fatal("open copy:", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()

	if !bytes.Equal(got, data) {
		t.Fatalf("copy data: got %q, want %q", got, data)
	}
}

func TestStripedDistribution(t *testing.T) {
	s := newTestStore(t, "stripes=8&sync=none&inline_kb=4")
	defer s.Close()

	b := s.Bucket("test")
	// Write 200 keys and verify they spread across stripes.
	for i := 0; i < 200; i++ {
		key := fmt.Sprintf("dist-key-%06d", i)
		data := []byte("v")
		b.Write(context.Background(), key, bytes.NewReader(data), 1, "text/plain", nil)
	}

	// Count keys per stripe.
	stripeCounts := make([]int, 8)
	for i, st := range s.stripes {
		results := st.idx.list("test", "")
		stripeCounts[i] = len(results)
	}

	// Verify at least 6 of 8 stripes have data (statistically near-certain).
	used := 0
	for _, c := range stripeCounts {
		if c > 0 {
			used++
		}
	}
	if used < 6 {
		t.Fatalf("expected at least 6 stripes with data, got %d: %v", used, stripeCounts)
	}
}

func TestUnknownSizeWrite(t *testing.T) {
	s := newTestStore(t, "stripes=4&sync=none&inline_kb=4")
	defer s.Close()

	b := s.Bucket("test")
	data := []byte("unknown-size-data")
	_, err := b.Write(context.Background(), "unknown", bytes.NewReader(data), -1, "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}

	rc, _, err := b.Open(context.Background(), "unknown", 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()

	if !bytes.Equal(got, data) {
		t.Fatalf("got %q, want %q", got, data)
	}
}

func TestRecovery(t *testing.T) {
	dir := t.TempDir()
	dsn := fmt.Sprintf("zebra:///%s?stripes=2&sync=batch&inline_kb=0", dir)

	// Write with sync=batch (CRC enabled, volume writes).
	d := &driver{}
	s1, err := d.Open(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	st1 := s1.(*store)

	b := st1.Bucket("test")
	data := []byte("persistent-data")
	_, err = b.Write(context.Background(), "persist-key", bytes.NewReader(data), int64(len(data)), "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	st1.Close()

	// Reopen — should recover.
	s2, err := d.Open(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	st2 := s2.(*store)
	defer st2.Close()

	b2 := st2.Bucket("test")
	rc, _, err := b2.Open(context.Background(), "persist-key", 0, 0, nil)
	if err != nil {
		t.Fatal("open after recovery:", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()

	if !bytes.Equal(got, data) {
		t.Fatalf("recovery: got %q, want %q", got, data)
	}
}

func BenchmarkWrite1KB(b *testing.B) {
	s := benchStore(b, "stripes=8&sync=none&inline_kb=4")
	defer s.Close()

	bkt := s.Bucket("bench")
	data := bytes.Repeat([]byte("W"), 1024)

	b.ResetTimer()
	b.SetBytes(1024)
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("w-%d", i)
		bkt.Write(context.Background(), key, bytes.NewReader(data), 1024, "application/octet-stream", nil)
	}
}

func BenchmarkRead1KB(b *testing.B) {
	s := benchStore(b, "stripes=8&sync=none&inline_kb=4")
	defer s.Close()

	bkt := s.Bucket("bench")
	data := bytes.Repeat([]byte("R"), 1024)

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("r-%d", i)
		bkt.Write(context.Background(), key, bytes.NewReader(data), 1024, "application/octet-stream", nil)
	}

	b.ResetTimer()
	b.SetBytes(1024)
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("r-%d", i%50)
		rc, _, _ := bkt.Open(context.Background(), key, 0, 0, nil)
		if rc != nil {
			rc.Close()
		}
	}
}

func BenchmarkStat(b *testing.B) {
	s := benchStore(b, "stripes=8&sync=none&inline_kb=4")
	defer s.Close()

	bkt := s.Bucket("bench")
	bkt.Write(context.Background(), "stat-key", bytes.NewReader([]byte("x")), 1, "text/plain", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bkt.Stat(context.Background(), "stat-key", nil)
	}
}

func BenchmarkParallelWrite1KB(b *testing.B) {
	s := benchStore(b, "stripes=8&sync=none&inline_kb=4")
	defer s.Close()

	bkt := s.Bucket("bench")
	data := bytes.Repeat([]byte("P"), 1024)

	b.ResetTimer()
	b.SetBytes(1024)
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("pw-%d", i)
			bkt.Write(context.Background(), key, bytes.NewReader(data), 1024, "application/octet-stream", nil)
			i++
		}
	})
}

func BenchmarkParallelRead1KB(b *testing.B) {
	s := benchStore(b, "stripes=8&sync=none&inline_kb=4")
	defer s.Close()

	bkt := s.Bucket("bench")
	data := bytes.Repeat([]byte("R"), 1024)

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("pr-%d", i)
		bkt.Write(context.Background(), key, bytes.NewReader(data), 1024, "application/octet-stream", nil)
	}

	b.ResetTimer()
	b.SetBytes(1024)
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("pr-%d", i%100)
			rc, _, _ := bkt.Open(context.Background(), key, 0, 0, nil)
			if rc != nil {
				rc.Close()
			}
			i++
		}
	})
}

func benchStore(b *testing.B, opts string) *store {
	b.Helper()
	dir := b.TempDir()
	dsn := fmt.Sprintf("zebra:///%s?%s", dir, opts)
	d := &driver{}
	s, err := d.Open(context.Background(), dsn)
	if err != nil {
		b.Fatal(err)
	}
	return s.(*store)
}
