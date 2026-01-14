package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
)

// Integration tests require an S3-compatible service running.
// Set S3_TEST_ENDPOINT to enable these tests.

func TestIntegration_WriteReadDelete(t *testing.T) {
	dsn := getTestDSN(t)

	d := &driver{}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	b := st.Bucket("")
	if b == nil {
		t.Fatal("Bucket returned nil")
	}

	// Write an object
	key := fmt.Sprintf("test-object-%d", time.Now().UnixNano())
	content := []byte("Hello, S3 World!")

	obj, err := b.Write(ctx, key, bytes.NewReader(content), int64(len(content)), "text/plain", storage.Options{
		"metadata": map[string]string{"test": "value"},
	})
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if obj.Key != key {
		t.Errorf("Key = %q, want %q", obj.Key, key)
	}
	if obj.Size != int64(len(content)) {
		t.Errorf("Size = %d, want %d", obj.Size, len(content))
	}

	// Read back the object
	rc, readObj, err := b.Open(ctx, key, 0, 0, nil)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	readContent, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(readContent, content) {
		t.Errorf("Content = %q, want %q", string(readContent), string(content))
	}
	if readObj.Size != int64(len(content)) {
		t.Errorf("Read object Size = %d, want %d", readObj.Size, len(content))
	}

	// Stat the object
	statObj, err := b.Stat(ctx, key, nil)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if statObj.Size != int64(len(content)) {
		t.Errorf("Stat Size = %d, want %d", statObj.Size, len(content))
	}

	// Delete the object
	if err := b.Delete(ctx, key, nil); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err = b.Stat(ctx, key, nil)
	if err != storage.ErrNotExist {
		t.Errorf("Stat after delete: got %v, want ErrNotExist", err)
	}
}

func TestIntegration_RangeRead(t *testing.T) {
	dsn := getTestDSN(t)

	d := &driver{}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	b := st.Bucket("")
	key := fmt.Sprintf("test-range-%d", time.Now().UnixNano())
	content := []byte("0123456789ABCDEF")

	_, err = b.Write(ctx, key, bytes.NewReader(content), int64(len(content)), "application/octet-stream", nil)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	defer b.Delete(ctx, key, nil)

	// Read range
	rc, _, err := b.Open(ctx, key, 5, 5, nil)
	if err != nil {
		t.Fatalf("Open range failed: %v", err)
	}

	rangeContent, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	expected := "56789"
	if string(rangeContent) != expected {
		t.Errorf("Range content = %q, want %q", string(rangeContent), expected)
	}
}

func TestIntegration_Copy(t *testing.T) {
	dsn := getTestDSN(t)

	d := &driver{}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	b := st.Bucket("")
	srcKey := fmt.Sprintf("test-copy-src-%d", time.Now().UnixNano())
	dstKey := fmt.Sprintf("test-copy-dst-%d", time.Now().UnixNano())
	content := []byte("Content to copy")

	// Write source
	_, err = b.Write(ctx, srcKey, bytes.NewReader(content), int64(len(content)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	defer b.Delete(ctx, srcKey, nil)
	defer b.Delete(ctx, dstKey, nil)

	// Copy
	copyObj, err := b.Copy(ctx, dstKey, "", srcKey, nil)
	if err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	if copyObj.Key != dstKey {
		t.Errorf("Copy Key = %q, want %q", copyObj.Key, dstKey)
	}

	// Verify copy
	rc, _, err := b.Open(ctx, dstKey, 0, 0, nil)
	if err != nil {
		t.Fatalf("Open copy failed: %v", err)
	}

	copyContent, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(copyContent, content) {
		t.Errorf("Copy content = %q, want %q", string(copyContent), string(content))
	}
}

func TestIntegration_Move(t *testing.T) {
	dsn := getTestDSN(t)

	d := &driver{}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	b := st.Bucket("")
	srcKey := fmt.Sprintf("test-move-src-%d", time.Now().UnixNano())
	dstKey := fmt.Sprintf("test-move-dst-%d", time.Now().UnixNano())
	content := []byte("Content to move")

	// Write source
	_, err = b.Write(ctx, srcKey, bytes.NewReader(content), int64(len(content)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	defer b.Delete(ctx, dstKey, nil)

	// Move
	moveObj, err := b.Move(ctx, dstKey, "", srcKey, nil)
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	if moveObj.Key != dstKey {
		t.Errorf("Move Key = %q, want %q", moveObj.Key, dstKey)
	}

	// Verify source is gone
	_, err = b.Stat(ctx, srcKey, nil)
	if err != storage.ErrNotExist {
		t.Errorf("Stat source after move: got %v, want ErrNotExist", err)
	}

	// Verify destination exists
	_, err = b.Stat(ctx, dstKey, nil)
	if err != nil {
		t.Errorf("Stat destination after move failed: %v", err)
	}
}

func TestIntegration_List(t *testing.T) {
	dsn := getTestDSN(t)

	d := &driver{}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	b := st.Bucket("")
	prefix := fmt.Sprintf("test-list-%d/", time.Now().UnixNano())

	// Create several objects
	keys := []string{
		prefix + "file1.txt",
		prefix + "file2.txt",
		prefix + "subdir/file3.txt",
	}

	for _, key := range keys {
		_, err = b.Write(ctx, key, strings.NewReader("content"), 7, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write %s failed: %v", key, err)
		}
		defer b.Delete(ctx, key, nil)
	}

	// List all with prefix
	iter, err := b.List(ctx, prefix, 0, 0, storage.Options{"recursive": true})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	defer iter.Close()

	var listed []string
	for {
		obj, err := iter.Next()
		if err != nil {
			t.Fatalf("Next failed: %v", err)
		}
		if obj == nil {
			break
		}
		listed = append(listed, obj.Key)
	}

	if len(listed) != len(keys) {
		t.Errorf("Listed %d objects, want %d", len(listed), len(keys))
	}
}

func TestIntegration_SignedURL(t *testing.T) {
	dsn := getTestDSN(t)

	d := &driver{}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	b := st.Bucket("")
	key := fmt.Sprintf("test-signed-%d", time.Now().UnixNano())
	content := []byte("Signed URL content")

	// Write object
	_, err = b.Write(ctx, key, bytes.NewReader(content), int64(len(content)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	defer b.Delete(ctx, key, nil)

	// Get signed URL
	url, err := b.SignedURL(ctx, key, "GET", 15*time.Minute, nil)
	if err != nil {
		t.Fatalf("SignedURL failed: %v", err)
	}

	if url == "" {
		t.Error("SignedURL returned empty string")
	}

	// URL should contain expected components
	if !strings.Contains(url, key) {
		t.Errorf("SignedURL should contain key, got %q", url)
	}
}

func TestIntegration_Multipart(t *testing.T) {
	dsn := getTestDSN(t)

	d := &driver{}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	b := st.Bucket("")
	mp, ok := b.(storage.HasMultipart)
	if !ok {
		t.Fatal("Bucket does not implement HasMultipart")
	}

	key := fmt.Sprintf("test-multipart-%d", time.Now().UnixNano())
	defer b.Delete(ctx, key, nil)

	// Init multipart
	mu, err := mp.InitMultipart(ctx, key, "application/octet-stream", nil)
	if err != nil {
		t.Fatalf("InitMultipart failed: %v", err)
	}

	// Upload parts (minimum 5MB for S3, but smaller for testing)
	partSize := 5 * 1024 * 1024 // 5MB
	parts := make([]*storage.PartInfo, 0, 2)

	for i := 1; i <= 2; i++ {
		partData := make([]byte, partSize)
		for j := range partData {
			partData[j] = byte(i)
		}

		part, err := mp.UploadPart(ctx, mu, i, bytes.NewReader(partData), int64(len(partData)), nil)
		if err != nil {
			t.Fatalf("UploadPart %d failed: %v", i, err)
		}
		parts = append(parts, part)
	}

	// List parts
	listedParts, err := mp.ListParts(ctx, mu, 0, 0, nil)
	if err != nil {
		t.Fatalf("ListParts failed: %v", err)
	}

	if len(listedParts) != 2 {
		t.Errorf("ListParts returned %d parts, want 2", len(listedParts))
	}

	// Complete multipart
	obj, err := mp.CompleteMultipart(ctx, mu, parts, nil)
	if err != nil {
		t.Fatalf("CompleteMultipart failed: %v", err)
	}

	if obj.Key != key {
		t.Errorf("Completed object Key = %q, want %q", obj.Key, key)
	}

	expectedSize := int64(partSize * 2)
	if obj.Size != expectedSize {
		t.Errorf("Completed object Size = %d, want %d", obj.Size, expectedSize)
	}
}

func TestIntegration_MultipartAbort(t *testing.T) {
	dsn := getTestDSN(t)

	d := &driver{}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	b := st.Bucket("")
	mp, ok := b.(storage.HasMultipart)
	if !ok {
		t.Fatal("Bucket does not implement HasMultipart")
	}

	key := fmt.Sprintf("test-multipart-abort-%d", time.Now().UnixNano())

	// Init multipart
	mu, err := mp.InitMultipart(ctx, key, "application/octet-stream", nil)
	if err != nil {
		t.Fatalf("InitMultipart failed: %v", err)
	}

	// Upload a part
	partData := make([]byte, 5*1024*1024)
	_, err = mp.UploadPart(ctx, mu, 1, bytes.NewReader(partData), int64(len(partData)), nil)
	if err != nil {
		t.Fatalf("UploadPart failed: %v", err)
	}

	// Abort
	if err := mp.AbortMultipart(ctx, mu, nil); err != nil {
		t.Fatalf("AbortMultipart failed: %v", err)
	}

	// Verify no object was created
	_, err = b.Stat(ctx, key, nil)
	if err != storage.ErrNotExist {
		t.Errorf("Stat after abort: got %v, want ErrNotExist", err)
	}
}

func TestIntegration_Directory(t *testing.T) {
	dsn := getTestDSN(t)

	d := &driver{}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	b := st.Bucket("")
	hd, ok := b.(storage.HasDirectories)
	if !ok {
		t.Fatal("Bucket does not implement HasDirectories")
	}

	prefix := fmt.Sprintf("test-dir-%d", time.Now().UnixNano())
	keys := []string{
		prefix + "/file1.txt",
		prefix + "/file2.txt",
		prefix + "/subdir/file3.txt",
	}

	// Create files
	for _, key := range keys {
		_, err = b.Write(ctx, key, strings.NewReader("content"), 7, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write %s failed: %v", key, err)
		}
		defer b.Delete(ctx, key, nil)
	}

	// Get directory
	dir := hd.Directory(prefix)

	// Info
	info, err := dir.Info(ctx)
	if err != nil {
		t.Fatalf("Directory Info failed: %v", err)
	}
	if !info.IsDir {
		t.Error("Directory Info.IsDir = false, want true")
	}

	// List
	iter, err := dir.List(ctx, 0, 0, nil)
	if err != nil {
		t.Fatalf("Directory List failed: %v", err)
	}
	defer iter.Close()

	var listed []string
	for {
		obj, err := iter.Next()
		if err != nil {
			t.Fatalf("Next failed: %v", err)
		}
		if obj == nil {
			break
		}
		listed = append(listed, obj.Key)
	}

	// Should have 2 files + 1 subdir = at least 2 items
	if len(listed) < 2 {
		t.Errorf("Directory List returned %d items, want at least 2", len(listed))
	}
}

// Conformance test helpers

func s3StorageFactory(t *testing.T) (storage.Storage, func()) {
	dsn := getTestDSN(t)

	d := &driver{}
	ctx := context.Background()

	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	cleanup := func() {
		st.Close()
	}

	return st, cleanup
}

// TestConformance runs the storage conformance suite if available
func TestConformance(t *testing.T) {
	endpoint := os.Getenv("S3_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("S3_TEST_ENDPOINT not set, skipping conformance tests")
	}

	// The conformance suite would be called here if we had access to it
	// For now, the individual integration tests above cover the main functionality
	t.Log("Integration tests cover main storage conformance")
}
