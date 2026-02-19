package horse

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/liteio-dev/liteio/pkg/storage"
)

func TestWriteReadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	dsn := "horse:///" + filepath.Join(dir, "data") + "?sync=none"

	ctx := context.Background()
	d := &driver{}
	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	b := st.Bucket("test")

	// Write a 1KB value.
	data := bytes.Repeat([]byte("A"), 1024)
	obj, err := b.Write(ctx, "key1", bytes.NewReader(data), int64(len(data)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if obj.Size != 1024 {
		t.Fatalf("expected size 1024, got %d", obj.Size)
	}

	// Read it back.
	rc, obj2, err := b.Open(ctx, "key1", 0, 0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("data mismatch: got %d bytes, want %d", len(got), len(data))
	}
	if obj2.ContentType != "text/plain" {
		t.Fatalf("content type mismatch: got %q", obj2.ContentType)
	}
}

func TestWriteReadMultipleKeys(t *testing.T) {
	dir := t.TempDir()
	dsn := "horse:///" + filepath.Join(dir, "data") + "?sync=none"

	ctx := context.Background()
	d := &driver{}
	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	b := st.Bucket("test")

	// Write multiple keys to exercise buffer ring.
	for i := 0; i < 100; i++ {
		data := bytes.Repeat([]byte{byte(i)}, 1024)
		key := "key" + string(rune('0'+i%10)) + string(rune('0'+i/10))
		_, err := b.Write(ctx, key, bytes.NewReader(data), int64(len(data)), "application/octet-stream", nil)
		if err != nil {
			t.Fatalf("Write %d: %v", i, err)
		}
	}

	// Read back and verify.
	for i := 0; i < 100; i++ {
		key := "key" + string(rune('0'+i%10)) + string(rune('0'+i/10))
		rc, _, err := b.Open(ctx, key, 0, 0, nil)
		if err != nil {
			t.Fatalf("Open %d: %v", i, err)
		}
		got, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("ReadAll %d: %v", i, err)
		}
		expected := bytes.Repeat([]byte{byte(i)}, 1024)
		if !bytes.Equal(got, expected) {
			t.Fatalf("data mismatch at key %d: got[0]=%d, want[0]=%d", i, got[0], expected[0])
		}
	}
}

func TestWriteReadLargeValue(t *testing.T) {
	dir := t.TempDir()
	dsn := "horse:///" + filepath.Join(dir, "data") + "?sync=none&bufsize=1048576" // 1MB buffer

	ctx := context.Background()
	d := &driver{}
	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	b := st.Bucket("test")

	// Write a 2MB value — larger than buffer size, will force buffer swap.
	data := make([]byte, 2*1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	_, err = b.Write(ctx, "big", bytes.NewReader(data), int64(len(data)), "application/octet-stream", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Read back.
	rc, obj, err := b.Open(ctx, "big", 0, 0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if obj.Size != int64(len(data)) {
		t.Fatalf("size mismatch: got %d, want %d", obj.Size, len(data))
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("data mismatch for large value")
	}
}

func TestDeleteAndStat(t *testing.T) {
	dir := t.TempDir()
	dsn := "horse:///" + filepath.Join(dir, "data") + "?sync=none"

	ctx := context.Background()
	d := &driver{}
	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	b := st.Bucket("test")

	data := []byte("hello world")
	_, err = b.Write(ctx, "del-me", bytes.NewReader(data), int64(len(data)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Stat should work.
	obj, err := b.Stat(ctx, "del-me", nil)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if obj.Size != int64(len(data)) {
		t.Fatalf("size mismatch: got %d", obj.Size)
	}

	// Delete.
	err = b.Delete(ctx, "del-me", nil)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Stat should fail.
	_, err = b.Stat(ctx, "del-me", nil)
	if err != storage.ErrNotExist {
		t.Fatalf("expected ErrNotExist, got %v", err)
	}
}

func TestCopy(t *testing.T) {
	dir := t.TempDir()
	dsn := "horse:///" + filepath.Join(dir, "data") + "?sync=none"

	ctx := context.Background()
	d := &driver{}
	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	b := st.Bucket("test")

	data := []byte("copy me")
	_, err = b.Write(ctx, "src", bytes.NewReader(data), int64(len(data)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Copy.
	_, err = b.Copy(ctx, "dst", "test", "src", nil)
	if err != nil {
		t.Fatalf("Copy: %v", err)
	}

	// Read copy.
	rc, _, err := b.Open(ctx, "dst", 0, 0, nil)
	if err != nil {
		t.Fatalf("Open dst: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if !bytes.Equal(got, data) {
		t.Fatalf("copy data mismatch")
	}
}

func TestUnknownSizeWrite(t *testing.T) {
	dir := t.TempDir()
	dsn := "horse:///" + filepath.Join(dir, "data") + "?sync=none"

	ctx := context.Background()
	d := &driver{}
	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	b := st.Bucket("test")

	data := []byte("unknown size data")
	// Pass size=-1 to trigger unknown-size path.
	_, err = b.Write(ctx, "unk", bytes.NewReader(data), -1, "text/plain", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	rc, _, err := b.Open(ctx, "unk", 0, 0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if !bytes.Equal(got, data) {
		t.Fatalf("data mismatch: got %q, want %q", got, data)
	}
}

// Benchmarks for quick performance verification.

func BenchmarkWrite1KB(b *testing.B) {
	dir := b.TempDir()
	dsn := "horse:///" + filepath.Join(dir, "data") + "?sync=none"
	ctx := context.Background()
	d := &driver{}
	st, err := d.Open(ctx, dsn)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer st.Close()
	bkt := st.Bucket("test")
	data := bytes.Repeat([]byte("A"), 1024)

	b.SetBytes(1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "write/" + strconv.Itoa(i)
		bkt.Write(ctx, key, bytes.NewReader(data), 1024, "application/octet-stream", nil)
	}
}

func BenchmarkRead1KB(b *testing.B) {
	dir := b.TempDir()
	dsn := "horse:///" + filepath.Join(dir, "data") + "?sync=none"
	ctx := context.Background()
	d := &driver{}
	st, err := d.Open(ctx, dsn)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer st.Close()
	bkt := st.Bucket("test")
	data := bytes.Repeat([]byte("A"), 1024)
	bkt.Write(ctx, "read-key", bytes.NewReader(data), 1024, "text/plain", nil)

	b.SetBytes(1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rc, _, _ := bkt.Open(ctx, "read-key", 0, 0, nil)
		io.Copy(io.Discard, rc)
		rc.Close()
	}
}

func BenchmarkStat(b *testing.B) {
	dir := b.TempDir()
	dsn := "horse:///" + filepath.Join(dir, "data") + "?sync=none"
	ctx := context.Background()
	d := &driver{}
	st, err := d.Open(ctx, dsn)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer st.Close()
	bkt := st.Bucket("test")
	data := bytes.Repeat([]byte("A"), 1024)
	bkt.Write(ctx, "stat-key", bytes.NewReader(data), 1024, "text/plain", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bkt.Stat(ctx, "stat-key", nil)
	}
}

func BenchmarkParallelWrite1KB(b *testing.B) {
	dir := b.TempDir()
	dsn := "horse:///" + filepath.Join(dir, "data") + "?sync=none"
	ctx := context.Background()
	d := &driver{}
	st, err := d.Open(ctx, dsn)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer st.Close()
	bkt := st.Bucket("test")
	data := bytes.Repeat([]byte("A"), 1024)

	var counter uint64
	b.SetBytes(1024)
	b.SetParallelism(100)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n := atomic.AddUint64(&counter, 1)
			key := "pw/" + strconv.FormatUint(n, 10)
			bkt.Write(ctx, key, bytes.NewReader(data), 1024, "application/octet-stream", nil)
		}
	})
}

func BenchmarkParallelRead1KB(b *testing.B) {
	dir := b.TempDir()
	dsn := "horse:///" + filepath.Join(dir, "data") + "?sync=none"
	ctx := context.Background()
	d := &driver{}
	st, err := d.Open(ctx, dsn)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer st.Close()
	bkt := st.Bucket("test")
	data := bytes.Repeat([]byte("A"), 1024)
	bkt.Write(ctx, "pr-key", bytes.NewReader(data), 1024, "text/plain", nil)

	b.SetBytes(1024)
	b.SetParallelism(100)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rc, _, _ := bkt.Open(ctx, "pr-key", 0, 0, nil)
			io.Copy(io.Discard, rc)
			rc.Close()
		}
	})
}

func TestRecoveryAfterReopen(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	// Use sync=batch so CRC is computed during writes (needed for recovery verification).
	dsn := "horse:///" + dataDir + "?sync=batch"

	ctx := context.Background()
	d := &driver{}

	// Write some data.
	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	b := st.Bucket("test")
	data := []byte("persist me")
	_, err = b.Write(ctx, "key1", bytes.NewReader(data), int64(len(data)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Close (this flushes the buffer ring + syncs).
	st.Close()

	// Verify the volume file exists.
	volPath := filepath.Join(dataDir, "volume.dat")
	info, err := os.Stat(volPath)
	if err != nil {
		t.Fatalf("volume file missing: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("volume file is empty")
	}

	// Reopen and verify data via CRC-verified recovery.
	st2, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	defer st2.Close()

	b2 := st2.Bucket("test")
	rc, _, err := b2.Open(ctx, "key1", 0, 0, nil)
	if err != nil {
		t.Fatalf("Open after reopen: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if !bytes.Equal(got, data) {
		t.Fatalf("data mismatch after recovery: got %q, want %q", got, data)
	}
}
