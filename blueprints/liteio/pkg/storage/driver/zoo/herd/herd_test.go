package herd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/liteio-dev/liteio/pkg/storage"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "herd-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func openTestStore(t *testing.T) storage.Storage {
	t.Helper()
	dir := tempDir(t)
	dsn := fmt.Sprintf("herd://%s?stripes=4&sync=none&inline_kb=8&prealloc=16", dir)
	st, err := storage.Open(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestWriteRead(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	bkt := st.Bucket("test")
	data := []byte("hello, herd!")

	obj, err := bkt.Write(ctx, "greeting.txt", bytes.NewReader(data), int64(len(data)), "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	if obj.Size != int64(len(data)) {
		t.Fatalf("expected size %d, got %d", len(data), obj.Size)
	}

	rc, obj2, err := bkt.Open(ctx, "greeting.txt", 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("expected %q, got %q", data, got)
	}
	if obj2.ContentType != "text/plain" {
		t.Fatalf("expected content-type text/plain, got %q", obj2.ContentType)
	}
}

func TestInlineValues(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	bkt := st.Bucket("test")

	// 1KB value should be inlined (< 8KB threshold).
	data := bytes.Repeat([]byte("x"), 1024)
	_, err := bkt.Write(ctx, "inline.bin", bytes.NewReader(data), int64(len(data)), "application/octet-stream", nil)
	if err != nil {
		t.Fatal(err)
	}

	rc, _, err := bkt.Open(ctx, "inline.bin", 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()

	if !bytes.Equal(got, data) {
		t.Fatal("inline value mismatch")
	}
}

func TestLargeValues(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	bkt := st.Bucket("test")

	// 64KB value should go through volume (> 8KB threshold).
	data := bytes.Repeat([]byte("L"), 64*1024)
	_, err := bkt.Write(ctx, "large.bin", bytes.NewReader(data), int64(len(data)), "application/octet-stream", nil)
	if err != nil {
		t.Fatal(err)
	}

	rc, obj, err := bkt.Open(ctx, "large.bin", 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()

	if !bytes.Equal(got, data) {
		t.Fatal("large value mismatch")
	}
	if obj.Size != int64(len(data)) {
		t.Fatalf("expected size %d, got %d", len(data), obj.Size)
	}
}

func TestStat(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	bkt := st.Bucket("test")

	data := []byte("stat me")
	bkt.Write(ctx, "s.txt", bytes.NewReader(data), int64(len(data)), "text/plain", nil)

	obj, err := bkt.Stat(ctx, "s.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	if obj.Size != int64(len(data)) {
		t.Fatalf("expected size %d, got %d", len(data), obj.Size)
	}
	if obj.ContentType != "text/plain" {
		t.Fatalf("expected text/plain, got %q", obj.ContentType)
	}
}

func TestDelete(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	bkt := st.Bucket("test")

	data := []byte("delete me")
	bkt.Write(ctx, "d.txt", bytes.NewReader(data), int64(len(data)), "text/plain", nil)

	err := bkt.Delete(ctx, "d.txt", nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = bkt.Stat(ctx, "d.txt", nil)
	if err != storage.ErrNotExist {
		t.Fatalf("expected ErrNotExist, got %v", err)
	}
}

func TestBloomFilterReject(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	bkt := st.Bucket("test")

	// Read a key that was never written — bloom should reject fast.
	_, _, err := bkt.Open(ctx, "nonexistent.txt", 0, 0, nil)
	if err != storage.ErrNotExist {
		t.Fatalf("expected ErrNotExist, got %v", err)
	}

	_, err = bkt.Stat(ctx, "nonexistent.txt", nil)
	if err != storage.ErrNotExist {
		t.Fatalf("expected ErrNotExist for stat, got %v", err)
	}
}

func TestList(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	bkt := st.Bucket("test")

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("dir/file_%02d.txt", i)
		data := []byte(fmt.Sprintf("content %d", i))
		bkt.Write(ctx, key, bytes.NewReader(data), int64(len(data)), "text/plain", nil)
	}

	iter, err := bkt.List(ctx, "dir/", 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()

	count := 0
	for {
		obj, err := iter.Next()
		if err != nil {
			t.Fatal(err)
		}
		if obj == nil {
			break
		}
		if !strings.HasPrefix(obj.Key, "dir/") {
			t.Fatalf("unexpected key: %s", obj.Key)
		}
		count++
	}
	if count != 10 {
		t.Fatalf("expected 10, got %d", count)
	}
}

func TestCopy(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	bkt := st.Bucket("test")

	data := []byte("copy me")
	bkt.Write(ctx, "src.txt", bytes.NewReader(data), int64(len(data)), "text/plain", nil)

	obj, err := bkt.Copy(ctx, "dst.txt", "test", "src.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	if obj.Size != int64(len(data)) {
		t.Fatalf("expected size %d, got %d", len(data), obj.Size)
	}

	rc, _, err := bkt.Open(ctx, "dst.txt", 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()

	if !bytes.Equal(got, data) {
		t.Fatal("copy value mismatch")
	}
}

func TestRangeRead(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	bkt := st.Bucket("test")

	data := []byte("0123456789")
	bkt.Write(ctx, "range.txt", bytes.NewReader(data), int64(len(data)), "text/plain", nil)

	// Read bytes 3-6.
	rc, _, err := bkt.Open(ctx, "range.txt", 3, 4, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()

	if string(got) != "3456" {
		t.Fatalf("expected '3456', got %q", got)
	}
}

func TestConcurrentWriteRead(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	bkt := st.Bucket("test")

	const n = 1000
	var wg sync.WaitGroup

	// Concurrent writes.
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key_%04d", i)
			data := []byte(fmt.Sprintf("value_%04d", i))
			_, err := bkt.Write(ctx, key, bytes.NewReader(data), int64(len(data)), "text/plain", nil)
			if err != nil {
				t.Errorf("write %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	// Concurrent reads.
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key_%04d", i)
			expected := fmt.Sprintf("value_%04d", i)
			rc, _, err := bkt.Open(ctx, key, 0, 0, nil)
			if err != nil {
				t.Errorf("read %d: %v", i, err)
				return
			}
			got, _ := io.ReadAll(rc)
			rc.Close()
			if string(got) != expected {
				t.Errorf("value mismatch for %s: got %q", key, got)
			}
		}(i)
	}
	wg.Wait()
}

func TestMultiNodeWriteRead(t *testing.T) {
	dir := tempDir(t)
	dsn := fmt.Sprintf("herd://%s?nodes=3&stripes=4&sync=none&inline_kb=8&prealloc=16", dir)
	st, err := storage.Open(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	bkt := st.Bucket("test")

	// Write 100 objects, verify they distribute across nodes.
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("obj_%03d.txt", i)
		data := []byte(fmt.Sprintf("data_%d", i))
		_, err := bkt.Write(ctx, key, bytes.NewReader(data), int64(len(data)), "text/plain", nil)
		if err != nil {
			t.Fatalf("write %s: %v", key, err)
		}
	}

	// Read them all back.
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("obj_%03d.txt", i)
		expected := fmt.Sprintf("data_%d", i)
		rc, _, err := bkt.Open(ctx, key, 0, 0, nil)
		if err != nil {
			t.Fatalf("open %s: %v", key, err)
		}
		got, _ := io.ReadAll(rc)
		rc.Close()
		if string(got) != expected {
			t.Fatalf("key %s: expected %q, got %q", key, expected, got)
		}
	}

	// List should return all 100.
	iter, err := bkt.List(ctx, "", 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for {
		obj, err := iter.Next()
		if err != nil {
			t.Fatal(err)
		}
		if obj == nil {
			break
		}
		count++
	}
	iter.Close()
	if count != 100 {
		t.Fatalf("expected 100 objects in list, got %d", count)
	}

	// Delete and verify.
	if err := bkt.Delete(ctx, "obj_050.txt", nil); err != nil {
		t.Fatal(err)
	}
	_, _, err = bkt.Open(ctx, "obj_050.txt", 0, 0, nil)
	if err != storage.ErrNotExist {
		t.Fatalf("expected ErrNotExist after delete, got %v", err)
	}

	// Copy across nodes.
	_, err = bkt.Copy(ctx, "obj_copy.txt", "", "obj_001.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	rc, _, err := bkt.Open(ctx, "obj_copy.txt", 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if string(got) != "data_1" {
		t.Fatalf("copy: expected 'data_1', got %q", got)
	}
}

func TestMultiNodeConcurrent(t *testing.T) {
	dir := tempDir(t)
	dsn := fmt.Sprintf("herd://%s?nodes=3&stripes=4&sync=none&inline_kb=8&prealloc=16", dir)
	st, err := storage.Open(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	bkt := st.Bucket("test")

	// Concurrent writes across 3 nodes.
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("concurrent_%04d", n)
			data := []byte(fmt.Sprintf("value_%d", n))
			_, err := bkt.Write(ctx, key, bytes.NewReader(data), int64(len(data)), "text/plain", nil)
			if err != nil {
				t.Errorf("write %s: %v", key, err)
			}
		}(i)
	}
	wg.Wait()

	// Verify all 200.
	for i := 0; i < 200; i++ {
		key := fmt.Sprintf("concurrent_%04d", i)
		expected := fmt.Sprintf("value_%d", i)
		rc, _, err := bkt.Open(ctx, key, 0, 0, nil)
		if err != nil {
			t.Fatalf("open %s: %v", key, err)
		}
		got, _ := io.ReadAll(rc)
		rc.Close()
		if string(got) != expected {
			t.Fatalf("%s: expected %q, got %q", key, expected, got)
		}
	}
}

func TestMultipart(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	bkt := st.Bucket("test")

	mp, ok := bkt.(storage.HasMultipart)
	if !ok {
		t.Skip("bucket does not support multipart")
	}

	mu, err := mp.InitMultipart(ctx, "multi.txt", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}

	part1Data := []byte("hello ")
	part2Data := []byte("world!")

	p1, err := mp.UploadPart(ctx, mu, 1, bytes.NewReader(part1Data), int64(len(part1Data)), nil)
	if err != nil {
		t.Fatal(err)
	}

	p2, err := mp.UploadPart(ctx, mu, 2, bytes.NewReader(part2Data), int64(len(part2Data)), nil)
	if err != nil {
		t.Fatal(err)
	}

	obj, err := mp.CompleteMultipart(ctx, mu, []*storage.PartInfo{p1, p2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if obj.Size != int64(len(part1Data)+len(part2Data)) {
		t.Fatalf("expected size %d, got %d", len(part1Data)+len(part2Data), obj.Size)
	}

	rc, _, err := bkt.Open(ctx, "multi.txt", 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()

	if string(got) != "hello world!" {
		t.Fatalf("expected 'hello world!', got %q", got)
	}
}
