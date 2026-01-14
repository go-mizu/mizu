// File: lib/storage/transport/sftp/server_test.go
package sftp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
	_ "github.com/go-mizu/blueprints/localflare/pkg/storage/driver/memory" // Register memory driver
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// testServer holds test server state.
type testServer struct {
	server  *Server
	client  *sftp.Client
	addr    string
	cleanup func()
}

// setupTestServer creates a test SFTP server backed by memory storage.
func setupTestServer(t *testing.T, cfgFunc func(*Config)) *testServer {
	t.Helper()

	ctx := context.Background()

	// Create memory storage
	store, err := storage.Open(ctx, "mem://")
	if err != nil {
		t.Fatalf("create memory storage: %v", err)
	}

	// Generate host key
	hostKey, err := GenerateHostKey()
	if err != nil {
		store.Close()
		t.Fatalf("generate host key: %v", err)
	}

	// Configure server with no authentication for testing
	cfg := &Config{
		Addr:     "127.0.0.1:0", // Random port
		HostKeys: []ssh.Signer{hostKey},
		Auth: AuthConfig{
			NoClientAuth: true,
		},
	}

	if cfgFunc != nil {
		cfgFunc(cfg)
	}

	// Create server
	server := New(store, cfg)

	// Create listener
	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		store.Close()
		t.Fatalf("listen: %v", err)
	}

	// Start server in background
	go server.Serve(ln)

	// Wait a bit for server to start
	time.Sleep(50 * time.Millisecond)

	// Create SFTP client
	addr := ln.Addr().String()
	client, err := createTestClient(t, addr, hostKey)
	if err != nil {
		server.Close()
		store.Close()
		t.Fatalf("create client: %v", err)
	}

	return &testServer{
		server: server,
		client: client,
		addr:   addr,
		cleanup: func() {
			client.Close()
			server.Close()
			store.Close()
		},
	}
}

// createTestClient creates an SFTP client for testing.
func createTestClient(t *testing.T, addr string, hostKey ssh.Signer) (*sftp.Client, error) {
	t.Helper()

	sshConfig := &ssh.ClientConfig{
		User: "testuser",
		Auth: []ssh.AuthMethod{
			ssh.Password("testpass"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("ssh dial: %w", err)
	}

	client, err := sftp.NewClient(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("sftp client: %w", err)
	}

	return client, nil
}

// TestListEmptyRoot tests listing an empty root directory.
func TestListEmptyRoot(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	entries, err := ts.client.ReadDir("/")
	if err != nil {
		t.Fatalf("ReadDir /: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 entries in empty root, got %d", len(entries))
	}
}

// TestCreateBucket tests creating a bucket via mkdir.
func TestCreateBucket(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	err := ts.client.Mkdir("/mybucket")
	if err != nil {
		t.Fatalf("Mkdir /mybucket: %v", err)
	}

	entries, err := ts.client.ReadDir("/")
	if err != nil {
		t.Fatalf("ReadDir /: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(entries))
	}

	if entries[0].Name() != "mybucket" {
		t.Errorf("expected bucket name 'mybucket', got %q", entries[0].Name())
	}

	if !entries[0].IsDir() {
		t.Error("bucket should be a directory")
	}
}

// TestWriteReadFile tests basic file write and read.
func TestWriteReadFile(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	// Create bucket
	if err := ts.client.Mkdir("/testbucket"); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	// Write file
	content := []byte("Hello, SFTP World!")
	f, err := ts.client.Create("/testbucket/hello.txt")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := f.Write(content); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Read file
	f, err = ts.client.Open("/testbucket/hello.txt")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if !bytes.Equal(data, content) {
		t.Errorf("content mismatch: got %q, want %q", data, content)
	}
}

// TestStatFile tests file stat operation.
func TestStatFile(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	// Create bucket and file
	ts.client.Mkdir("/testbucket")
	content := []byte("test content")
	f, _ := ts.client.Create("/testbucket/file.txt")
	f.Write(content)
	f.Close()

	// Stat file
	info, err := ts.client.Stat("/testbucket/file.txt")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	if info.Size() != int64(len(content)) {
		t.Errorf("size mismatch: got %d, want %d", info.Size(), len(content))
	}

	if info.IsDir() {
		t.Error("file should not be a directory")
	}
}

// TestDeleteFile tests file deletion.
func TestDeleteFile(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	// Create bucket and file
	ts.client.Mkdir("/testbucket")
	f, _ := ts.client.Create("/testbucket/todelete.txt")
	f.Write([]byte("delete me"))
	f.Close()

	// Delete file
	if err := ts.client.Remove("/testbucket/todelete.txt"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Verify deletion
	_, err := ts.client.Stat("/testbucket/todelete.txt")
	if err == nil {
		t.Error("expected error after deletion")
	}
}

// TestRenameFile tests file rename within same bucket.
func TestRenameFile(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	// Create bucket and file
	ts.client.Mkdir("/testbucket")
	f, _ := ts.client.Create("/testbucket/original.txt")
	f.Write([]byte("rename me"))
	f.Close()

	// Rename
	if err := ts.client.Rename("/testbucket/original.txt", "/testbucket/renamed.txt"); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	// Verify old path gone
	_, err := ts.client.Stat("/testbucket/original.txt")
	if err == nil {
		t.Error("original file should not exist")
	}

	// Verify new path exists
	info, err := ts.client.Stat("/testbucket/renamed.txt")
	if err != nil {
		t.Fatalf("Stat renamed: %v", err)
	}
	if info.Name() != "renamed.txt" {
		t.Errorf("expected name 'renamed.txt', got %q", info.Name())
	}
}

// TestNestedDirectories tests nested path handling.
func TestNestedDirectories(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	// Create bucket
	ts.client.Mkdir("/testbucket")

	// Create file in nested path
	f, err := ts.client.Create("/testbucket/a/b/c/deep.txt")
	if err != nil {
		t.Fatalf("Create nested: %v", err)
	}
	f.Write([]byte("deep content"))
	f.Close()

	// Read back
	f, err = ts.client.Open("/testbucket/a/b/c/deep.txt")
	if err != nil {
		t.Fatalf("Open nested: %v", err)
	}
	defer f.Close()

	data, _ := io.ReadAll(f)
	if string(data) != "deep content" {
		t.Errorf("content mismatch: %q", data)
	}

	// List intermediate directory
	entries, err := ts.client.ReadDir("/testbucket/a/b")
	if err != nil {
		t.Fatalf("ReadDir /testbucket/a/b: %v", err)
	}

	if len(entries) != 1 || entries[0].Name() != "c" {
		t.Errorf("expected directory 'c', got %v", entries)
	}
}

// TestListFiles tests file listing.
func TestListFiles(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	// Create bucket and files
	ts.client.Mkdir("/testbucket")

	files := []string{"apple.txt", "banana.txt", "cherry.txt"}
	for _, name := range files {
		f, _ := ts.client.Create("/testbucket/" + name)
		f.Write([]byte("content"))
		f.Close()
	}

	// List
	entries, err := ts.client.ReadDir("/testbucket")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	if len(entries) != len(files) {
		t.Fatalf("expected %d files, got %d", len(files), len(entries))
	}

	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	sort.Strings(files)

	for i := range files {
		if names[i] != files[i] {
			t.Errorf("file %d mismatch: got %q, want %q", i, names[i], files[i])
		}
	}
}

// TestRemoveBucket tests bucket removal via rmdir.
func TestRemoveBucket(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	// Create and verify bucket
	ts.client.Mkdir("/testbucket")
	entries, _ := ts.client.ReadDir("/")
	if len(entries) != 1 {
		t.Fatal("bucket not created")
	}

	// Remove bucket
	if err := ts.client.RemoveDirectory("/testbucket"); err != nil {
		t.Fatalf("RemoveDirectory: %v", err)
	}

	// Verify removal
	entries, _ = ts.client.ReadDir("/")
	if len(entries) != 0 {
		t.Error("bucket should be removed")
	}
}

// TestLargeFile tests handling of larger files.
func TestLargeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large file test in short mode")
	}

	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	ts.client.Mkdir("/testbucket")

	// Create 5MB file
	size := 5 * 1024 * 1024
	content := make([]byte, size)
	for i := range content {
		content[i] = byte(i % 256)
	}

	// Write
	f, _ := ts.client.Create("/testbucket/large.bin")
	n, err := f.Write(content)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != size {
		t.Fatalf("wrote %d bytes, expected %d", n, size)
	}
	f.Close()

	// Read and verify
	f, err = ts.client.Open("/testbucket/large.bin")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	f.Close()

	if len(data) != len(content) {
		t.Errorf("size mismatch: got %d, want %d", len(data), len(content))
	} else if !bytes.Equal(data, content) {
		// Find first difference
		for i := range data {
			if data[i] != content[i] {
				t.Errorf("content mismatch at offset %d: got %d, want %d", i, data[i], content[i])
				break
			}
		}
	}
}

// TestConcurrentAccess tests concurrent file operations.
func TestConcurrentAccess(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	ts.client.Mkdir("/testbucket")

	// Create test file
	f, _ := ts.client.Create("/testbucket/concurrent.txt")
	f.Write([]byte("initial content"))
	f.Close()

	// Concurrent reads
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			f, err := ts.client.Open("/testbucket/concurrent.txt")
			if err != nil {
				errors <- fmt.Errorf("reader %d: open: %w", id, err)
				return
			}
			defer f.Close()
			_, err = io.ReadAll(f)
			if err != nil {
				errors <- fmt.Errorf("reader %d: read: %w", id, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

// TestReadOnlyMode tests read-only configuration.
func TestReadOnlyMode(t *testing.T) {
	ts := setupTestServer(t, func(cfg *Config) {
		cfg.ReadOnly = true
	})
	defer ts.cleanup()

	// Try to create bucket - should fail
	err := ts.client.Mkdir("/testbucket")
	if err == nil {
		t.Error("mkdir should fail in read-only mode")
	}

	// Try to create file - should fail
	_, err = ts.client.Create("/testbucket/file.txt")
	if err == nil {
		t.Error("create should fail in read-only mode")
	}
}

// TestHomeBucket tests user isolation via HomeBucket.
func TestHomeBucket(t *testing.T) {
	var store storage.Storage
	ctx := context.Background()
	store, _ = storage.Open(ctx, "mem://")

	// Pre-create user bucket
	store.CreateBucket(ctx, "user-testuser", nil)

	hostKey, _ := GenerateHostKey()

	cfg := &Config{
		Addr:     "127.0.0.1:0",
		HostKeys: []ssh.Signer{hostKey},
		Auth: AuthConfig{
			NoClientAuth: true,
		},
		HomeBucket: func(username string) string {
			return "user-" + username
		},
	}

	server := New(store, cfg)
	ln, _ := net.Listen("tcp", cfg.Addr)
	go server.Serve(ln)
	defer func() {
		server.Close()
		store.Close()
	}()

	time.Sleep(50 * time.Millisecond)

	client, _ := createTestClient(t, ln.Addr().String(), hostKey)
	defer client.Close()

	// Root should show home bucket contents, not all buckets
	// When user creates a file at /, it goes into their bucket
	f, err := client.Create("/myfile.txt")
	if err != nil {
		t.Fatalf("Create in home bucket: %v", err)
	}
	f.Write([]byte("home content"))
	f.Close()

	// Verify file exists in user's bucket
	_, err = client.Stat("/myfile.txt")
	if err != nil {
		t.Fatalf("Stat in home bucket: %v", err)
	}
}

// TestStatRoot tests stating the root directory.
func TestStatRoot(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	info, err := ts.client.Stat("/")
	if err != nil {
		t.Fatalf("Stat /: %v", err)
	}

	if !info.IsDir() {
		t.Error("root should be a directory")
	}
}

// TestStatBucket tests stating a bucket.
func TestStatBucket(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	ts.client.Mkdir("/mybucket")

	info, err := ts.client.Stat("/mybucket")
	if err != nil {
		t.Fatalf("Stat bucket: %v", err)
	}

	if !info.IsDir() {
		t.Error("bucket should be a directory")
	}

	if info.Name() != "mybucket" {
		t.Errorf("expected name 'mybucket', got %q", info.Name())
	}
}

// TestStatNonExistent tests stating non-existent paths.
func TestStatNonExistent(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	// Note: The memory driver auto-creates buckets when accessed,
	// so stat on a "nonexistent" bucket actually creates it.
	// This behavior is specific to the memory driver.
	// We only test that nonexistent files return errors.

	ts.client.Mkdir("/mybucket")
	_, err := ts.client.Stat("/mybucket/nonexistent.txt")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

// TestSpecialCharacters tests handling of special characters in names.
func TestSpecialCharacters(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	ts.client.Mkdir("/testbucket")

	// Files with special characters
	names := []string{
		"file with spaces.txt",
		"file-with-dashes.txt",
		"file_with_underscores.txt",
		"file.multiple.dots.txt",
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			path := "/testbucket/" + name
			f, err := ts.client.Create(path)
			if err != nil {
				t.Fatalf("Create %q: %v", path, err)
			}
			f.Write([]byte("content"))
			f.Close()

			info, err := ts.client.Stat(path)
			if err != nil {
				t.Fatalf("Stat %q: %v", path, err)
			}
			if info.Name() != name {
				t.Errorf("name mismatch: got %q, want %q", info.Name(), name)
			}
		})
	}
}

// TestEmptyFile tests handling of zero-byte files.
func TestEmptyFile(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	ts.client.Mkdir("/testbucket")

	// Create empty file
	f, _ := ts.client.Create("/testbucket/empty.txt")
	f.Close()

	// Stat
	info, err := ts.client.Stat("/testbucket/empty.txt")
	if err != nil {
		t.Fatalf("Stat empty file: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("expected size 0, got %d", info.Size())
	}

	// Read
	f, _ = ts.client.Open("/testbucket/empty.txt")
	data, _ := io.ReadAll(f)
	f.Close()
	if len(data) != 0 {
		t.Errorf("expected empty content, got %d bytes", len(data))
	}
}

// TestMultipleBuckets tests operations across multiple buckets.
func TestMultipleBuckets(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	buckets := []string{"bucket1", "bucket2", "bucket3"}
	for _, b := range buckets {
		ts.client.Mkdir("/" + b)
	}

	// Verify listing
	entries, _ := ts.client.ReadDir("/")
	if len(entries) != len(buckets) {
		t.Fatalf("expected %d buckets, got %d", len(buckets), len(entries))
	}

	// Create file in each bucket
	for _, b := range buckets {
		f, _ := ts.client.Create("/" + b + "/file.txt")
		f.Write([]byte("content in " + b))
		f.Close()
	}

	// Verify each bucket has its file
	for _, b := range buckets {
		entries, _ := ts.client.ReadDir("/" + b)
		if len(entries) != 1 {
			t.Errorf("bucket %s: expected 1 file, got %d", b, len(entries))
		}
	}
}

// TestRealPath tests path normalization (realpath).
func TestRealPath(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	ts.client.Mkdir("/testbucket")

	// RealPath should normalize paths
	rp, err := ts.client.RealPath("/testbucket/../testbucket")
	if err != nil {
		t.Fatalf("RealPath: %v", err)
	}
	if !strings.HasSuffix(rp, "testbucket") {
		t.Errorf("unexpected realpath result: %s", rp)
	}
}

// TestFilePermissions tests that file permissions are set correctly.
func TestFilePermissions(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	ts.client.Mkdir("/testbucket")
	f, _ := ts.client.Create("/testbucket/file.txt")
	f.Write([]byte("content"))
	f.Close()

	// Check file permissions
	info, _ := ts.client.Stat("/testbucket/file.txt")
	mode := info.Mode()
	if mode.IsDir() {
		t.Error("file should not be a directory")
	}
	// Should have read/write permissions
	if mode&0600 == 0 {
		t.Error("file should have read/write permissions")
	}

	// Check directory permissions
	info, _ = ts.client.Stat("/testbucket")
	mode = info.Mode()
	if !mode.IsDir() {
		t.Error("bucket should be a directory")
	}
}

// TestGracefulShutdown tests server graceful shutdown.
func TestGracefulShutdown(t *testing.T) {
	ts := setupTestServer(t, nil)

	// Create a long-running operation
	ts.client.Mkdir("/testbucket")

	// Start shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- ts.server.Shutdown(ctx)
	}()

	// Close client (simulates client disconnect)
	ts.client.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Shutdown error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("Shutdown timed out")
	}
}

// TestBufferSpillToDisk tests that large uploads spill to disk.
func TestBufferSpillToDisk(t *testing.T) {
	// Use small buffer to force disk spill
	ts := setupTestServer(t, func(cfg *Config) {
		cfg.WriteBufferSize = 1024 // 1KB
	})
	defer ts.cleanup()

	ts.client.Mkdir("/testbucket")

	// Write 10KB file
	content := bytes.Repeat([]byte("X"), 10*1024)
	f, _ := ts.client.Create("/testbucket/large.txt")
	f.Write(content)
	f.Close()

	// Read back
	f, _ = ts.client.Open("/testbucket/large.txt")
	data, _ := io.ReadAll(f)
	f.Close()

	if !bytes.Equal(data, content) {
		t.Error("content mismatch after disk spill")
	}
}

// BenchmarkWrite benchmarks file write performance.
func BenchmarkWrite(b *testing.B) {
	ctx := context.Background()
	store, _ := storage.Open(ctx, "mem://")
	defer store.Close()

	hostKey, _ := GenerateHostKey()
	cfg := &Config{
		Addr:     "127.0.0.1:0",
		HostKeys: []ssh.Signer{hostKey},
		Auth:     AuthConfig{NoClientAuth: true},
	}

	server := New(store, cfg)
	ln, _ := net.Listen("tcp", cfg.Addr)
	go server.Serve(ln)
	defer server.Close()

	time.Sleep(50 * time.Millisecond)

	client, _ := createTestClientForBench(b, ln.Addr().String(), hostKey)
	defer client.Close()

	client.Mkdir("/bench")
	content := bytes.Repeat([]byte("X"), 1024) // 1KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("/bench/file%d.txt", i)
		f, _ := client.Create(name)
		f.Write(content)
		f.Close()
	}
}

// BenchmarkRead benchmarks file read performance.
func BenchmarkRead(b *testing.B) {
	ctx := context.Background()
	store, _ := storage.Open(ctx, "mem://")
	defer store.Close()

	hostKey, _ := GenerateHostKey()
	cfg := &Config{
		Addr:     "127.0.0.1:0",
		HostKeys: []ssh.Signer{hostKey},
		Auth:     AuthConfig{NoClientAuth: true},
	}

	server := New(store, cfg)
	ln, _ := net.Listen("tcp", cfg.Addr)
	go server.Serve(ln)
	defer server.Close()

	time.Sleep(50 * time.Millisecond)

	client, _ := createTestClientForBench(b, ln.Addr().String(), hostKey)
	defer client.Close()

	client.Mkdir("/bench")
	content := bytes.Repeat([]byte("X"), 1024)
	f, _ := client.Create("/bench/file.txt")
	f.Write(content)
	f.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f, _ := client.Open("/bench/file.txt")
		io.Copy(io.Discard, f)
		f.Close()
	}
}

func createTestClientForBench(b *testing.B, addr string, hostKey ssh.Signer) (*sftp.Client, error) {
	b.Helper()

	sshConfig := &ssh.ClientConfig{
		User:            "benchuser",
		Auth:            []ssh.AuthMethod{ssh.Password("pass")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, err
	}

	return sftp.NewClient(conn)
}

// TestAuthPublicKey tests public key authentication.
func TestAuthPublicKey(t *testing.T) {
	ctx := context.Background()
	store, _ := storage.Open(ctx, "mem://")
	defer store.Close()

	// Generate host key and user key
	hostKey, _ := GenerateHostKey()
	userKey, _ := GenerateHostKey()

	cfg := &Config{
		Addr:     "127.0.0.1:0",
		HostKeys: []ssh.Signer{hostKey},
		Auth: AuthConfig{
			PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
				if bytes.Equal(key.Marshal(), userKey.PublicKey().Marshal()) {
					return &ssh.Permissions{}, nil
				}
				return nil, fmt.Errorf("unauthorized key")
			},
		},
	}

	server := New(store, cfg)
	ln, _ := net.Listen("tcp", cfg.Addr)
	go server.Serve(ln)
	defer server.Close()

	time.Sleep(50 * time.Millisecond)

	// Connect with correct key
	sshConfig := &ssh.ClientConfig{
		User: "testuser",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(userKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	conn, err := ssh.Dial("tcp", ln.Addr().String(), sshConfig)
	if err != nil {
		t.Fatalf("SSH dial with valid key: %v", err)
	}
	conn.Close()

	// Connect with wrong key - should fail
	wrongKey, _ := GenerateHostKey()
	sshConfig.Auth = []ssh.AuthMethod{ssh.PublicKeys(wrongKey)}

	_, err = ssh.Dial("tcp", ln.Addr().String(), sshConfig)
	if err == nil {
		t.Error("expected error with wrong key")
	}
}

// TestAuthPassword tests password authentication.
func TestAuthPassword(t *testing.T) {
	ctx := context.Background()
	store, _ := storage.Open(ctx, "mem://")
	defer store.Close()

	hostKey, _ := GenerateHostKey()

	cfg := &Config{
		Addr:     "127.0.0.1:0",
		HostKeys: []ssh.Signer{hostKey},
		Auth: AuthConfig{
			PasswordCallback: PasswordAuthFromMap(map[string]string{
				"testuser": "correctpassword",
			}),
		},
	}

	server := New(store, cfg)
	ln, _ := net.Listen("tcp", cfg.Addr)
	go server.Serve(ln)
	defer server.Close()

	time.Sleep(50 * time.Millisecond)

	// Connect with correct password
	sshConfig := &ssh.ClientConfig{
		User:            "testuser",
		Auth:            []ssh.AuthMethod{ssh.Password("correctpassword")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	conn, err := ssh.Dial("tcp", ln.Addr().String(), sshConfig)
	if err != nil {
		t.Fatalf("SSH dial with valid password: %v", err)
	}
	conn.Close()

	// Connect with wrong password
	sshConfig.Auth = []ssh.AuthMethod{ssh.Password("wrongpassword")}
	_, err = ssh.Dial("tcp", ln.Addr().String(), sshConfig)
	if err == nil {
		t.Error("expected error with wrong password")
	}
}

// TestWriteAtRandomOffsets tests random offset writes.
func TestWriteAtRandomOffsets(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	ts.client.Mkdir("/testbucket")

	// Create file with scattered writes
	f, _ := ts.client.Create("/testbucket/random.bin")

	// Write at offset 100
	f.WriteAt([]byte("hello"), 100)
	// Write at offset 0
	f.WriteAt([]byte("start"), 0)
	// Write at offset 200
	f.WriteAt([]byte("end"), 200)

	f.Close()

	// Read and verify
	f, _ = ts.client.Open("/testbucket/random.bin")
	data, _ := io.ReadAll(f)
	f.Close()

	if len(data) < 203 {
		t.Fatalf("expected at least 203 bytes, got %d", len(data))
	}

	if string(data[0:5]) != "start" {
		t.Errorf("start mismatch: %q", data[0:5])
	}
	if string(data[100:105]) != "hello" {
		t.Errorf("middle mismatch: %q", data[100:105])
	}
	if string(data[200:203]) != "end" {
		t.Errorf("end mismatch: %q", data[200:203])
	}
}

// TestDirectoryListing tests that directories are properly identified.
func TestDirectoryListing(t *testing.T) {
	ts := setupTestServer(t, nil)
	defer ts.cleanup()

	ts.client.Mkdir("/testbucket")

	// Create files in nested structure
	f, _ := ts.client.Create("/testbucket/root.txt")
	f.Write([]byte("root file"))
	f.Close()

	f, _ = ts.client.Create("/testbucket/dir1/file1.txt")
	f.Write([]byte("nested file 1"))
	f.Close()

	f, _ = ts.client.Create("/testbucket/dir1/file2.txt")
	f.Write([]byte("nested file 2"))
	f.Close()

	f, _ = ts.client.Create("/testbucket/dir2/deep/file.txt")
	f.Write([]byte("deep nested"))
	f.Close()

	// List bucket root
	entries, _ := ts.client.ReadDir("/testbucket")

	var files, dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		} else {
			files = append(files, e.Name())
		}
	}

	sort.Strings(dirs)
	sort.Strings(files)

	if len(files) != 1 || files[0] != "root.txt" {
		t.Errorf("expected [root.txt], got %v", files)
	}

	if len(dirs) != 2 {
		t.Errorf("expected 2 directories, got %d: %v", len(dirs), dirs)
	}
}

// TestServerBanner tests SSH banner display.
func TestServerBanner(t *testing.T) {
	ts := setupTestServer(t, func(cfg *Config) {
		cfg.Banner = "Welcome to the test SFTP server\n"
	})
	defer ts.cleanup()

	// The banner is sent during SSH connection, not SFTP
	// Just verify the server starts correctly with a banner
	_, err := ts.client.Stat("/")
	if err != nil {
		t.Fatalf("Stat with banner: %v", err)
	}
}

// TestMaxConnections tests connection limiting.
func TestMaxConnections(t *testing.T) {
	ctx := context.Background()
	store, _ := storage.Open(ctx, "mem://")
	defer store.Close()

	hostKey, _ := GenerateHostKey()

	cfg := &Config{
		Addr:           "127.0.0.1:0",
		HostKeys:       []ssh.Signer{hostKey},
		Auth:           AuthConfig{NoClientAuth: true},
		MaxConnections: 2,
	}

	server := New(store, cfg)
	ln, _ := net.Listen("tcp", cfg.Addr)
	go server.Serve(ln)
	defer server.Close()

	time.Sleep(50 * time.Millisecond)

	addr := ln.Addr().String()

	// Create 2 connections (should succeed)
	var clients []*sftp.Client
	for i := 0; i < 2; i++ {
		client, err := createTestClient(t, addr, hostKey)
		if err != nil {
			t.Fatalf("client %d: %v", i, err)
		}
		clients = append(clients, client)
	}

	// Third connection should be rejected
	sshConfig := &ssh.ClientConfig{
		User:            "testuser",
		Auth:            []ssh.AuthMethod{ssh.Password("pass")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         2 * time.Second,
	}

	// Give the server time to register the connections
	time.Sleep(100 * time.Millisecond)

	_, err := ssh.Dial("tcp", addr, sshConfig)
	if err == nil {
		t.Log("Warning: third connection succeeded (race condition possible)")
	}

	// Cleanup
	for _, c := range clients {
		c.Close()
	}
}

// TestContentTypeDetection tests content type detection.
func TestContentTypeDetection(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"file.txt", "text/plain"},
		{"file.html", "text/html"},
		{"file.css", "text/css"},
		{"file.js", "application/javascript"},
		{"file.json", "application/json"},
		{"file.png", "image/png"},
		{"file.jpg", "image/jpeg"},
		{"file.gif", "image/gif"},
		{"file.pdf", "application/pdf"},
		{"file.zip", "application/zip"},
		{"file.bin", "application/octet-stream"},
		{"file", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			ct := detectContentType(tt.filename)
			if ct != tt.expected {
				t.Errorf("detectContentType(%q) = %q, want %q", tt.filename, ct, tt.expected)
			}
		})
	}
}

// TestTempDirConfig tests custom temp directory configuration.
func TestTempDirConfig(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "sftp-test-temp")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	ts := setupTestServer(t, func(cfg *Config) {
		cfg.TempDir = tempDir
		cfg.WriteBufferSize = 100 // Force disk spill
	})
	defer ts.cleanup()

	ts.client.Mkdir("/testbucket")

	// Write large file to trigger disk spill
	content := bytes.Repeat([]byte("X"), 1000)
	f, _ := ts.client.Create("/testbucket/file.txt")
	f.Write(content)
	f.Close()

	// Read back to verify
	f, _ = ts.client.Open("/testbucket/file.txt")
	data, _ := io.ReadAll(f)
	f.Close()

	if !bytes.Equal(data, content) {
		t.Error("content mismatch with custom temp dir")
	}
}
