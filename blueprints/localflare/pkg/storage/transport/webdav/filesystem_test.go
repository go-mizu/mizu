// File: lib/storage/transport/webdav/filesystem_test.go

package webdav

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/go-mizu/blueprints/localflare/pkg/storage/driver/local"
)

func newTestFS(t *testing.T) (*StorageFileSystem, func()) {
	t.Helper()

	ctx := context.Background()
	store, err := local.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("open local storage: %v", err)
	}

	fs := &StorageFileSystem{
		store:              store,
		defaultContentType: "application/octet-stream",
		writeBufferSize:    32 << 20,
	}

	cleanup := func() {
		_ = store.Close()
	}

	return fs, cleanup
}

func TestStorageFileSystem_ParsePath_MultiBucket(t *testing.T) {
	fs := &StorageFileSystem{}

	tests := []struct {
		name       string
		path       string
		wantBucket string
		wantKey    string
	}{
		{"root", "/", "", ""},
		{"bucket", "/mybucket", "mybucket", ""},
		{"bucket trailing slash", "/mybucket/", "mybucket", ""},
		{"object", "/mybucket/file.txt", "mybucket", "file.txt"},
		{"nested object", "/mybucket/dir/subdir/file.txt", "mybucket", "dir/subdir/file.txt"},
		{"clean path", "//mybucket//dir//file.txt", "mybucket", "dir/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, key, err := fs.parsePath(tt.path)
			if err != nil {
				t.Fatalf("parsePath(%q) error: %v", tt.path, err)
			}
			if bucket != tt.wantBucket {
				t.Errorf("bucket = %q, want %q", bucket, tt.wantBucket)
			}
			if key != tt.wantKey {
				t.Errorf("key = %q, want %q", key, tt.wantKey)
			}
		})
	}
}

func TestStorageFileSystem_ParsePath_SingleBucket(t *testing.T) {
	fs := &StorageFileSystem{bucket: "fixed"}

	tests := []struct {
		name       string
		path       string
		wantBucket string
		wantKey    string
	}{
		{"root", "/", "fixed", ""},
		{"file at root", "/file.txt", "fixed", "file.txt"},
		{"nested file", "/dir/subdir/file.txt", "fixed", "dir/subdir/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, key, err := fs.parsePath(tt.path)
			if err != nil {
				t.Fatalf("parsePath(%q) error: %v", tt.path, err)
			}
			if bucket != tt.wantBucket {
				t.Errorf("bucket = %q, want %q", bucket, tt.wantBucket)
			}
			if key != tt.wantKey {
				t.Errorf("key = %q, want %q", key, tt.wantKey)
			}
		})
	}
}

func TestStorageFileSystem_Mkdir(t *testing.T) {
	fs, cleanup := newTestFS(t)
	defer cleanup()

	ctx := context.Background()

	// Create bucket
	err := fs.Mkdir(ctx, "/newbucket", 0755)
	if err != nil {
		t.Fatalf("Mkdir bucket: %v", err)
	}

	// Verify bucket exists
	info, err := fs.Stat(ctx, "/newbucket")
	if err != nil {
		t.Fatalf("Stat bucket: %v", err)
	}
	if !info.IsDir() {
		t.Error("bucket should be directory")
	}

	// Create nested directory (virtual, no-op)
	err = fs.Mkdir(ctx, "/newbucket/subdir", 0755)
	if err != nil {
		t.Fatalf("Mkdir subdir: %v", err)
	}
}

func TestStorageFileSystem_Stat_Root(t *testing.T) {
	fs, cleanup := newTestFS(t)
	defer cleanup()

	ctx := context.Background()

	info, err := fs.Stat(ctx, "/")
	if err != nil {
		t.Fatalf("Stat root: %v", err)
	}

	if !info.IsDir() {
		t.Error("root should be directory")
	}
	if info.Name() != "/" {
		t.Errorf("root name = %q, want %q", info.Name(), "/")
	}
}

func TestStorageFileSystem_OpenFile_Write(t *testing.T) {
	fs, cleanup := newTestFS(t)
	defer cleanup()

	ctx := context.Background()

	// Create bucket first
	err := fs.Mkdir(ctx, "/writebucket", 0755)
	if err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	// Open file for writing
	file, err := fs.OpenFile(ctx, "/writebucket/test.txt", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		t.Fatalf("OpenFile for write: %v", err)
	}

	// Write content
	content := []byte("hello world")
	n, err := file.Write(content)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(content) {
		t.Errorf("wrote %d bytes, want %d", n, len(content))
	}

	// Close to commit
	err = file.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Verify content by reading
	file, err = fs.OpenFile(ctx, "/writebucket/test.txt", os.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("OpenFile for read: %v", err)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	_ = file.Close()

	if string(data) != string(content) {
		t.Errorf("read = %q, want %q", string(data), string(content))
	}
}

func TestStorageFileSystem_RemoveAll(t *testing.T) {
	fs, cleanup := newTestFS(t)
	defer cleanup()

	ctx := context.Background()

	// Create bucket and files
	err := fs.Mkdir(ctx, "/delbucket", 0755)
	if err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	// Create some files (using flat structure to match object storage behavior)
	for _, name := range []string{"file1.txt", "file2.txt", "file3.txt"} {
		file, err := fs.OpenFile(ctx, "/delbucket/"+name, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			t.Fatalf("OpenFile: %v", err)
		}
		_, _ = file.Write([]byte("content"))
		_ = file.Close()
	}

	// Delete a single file
	err = fs.RemoveAll(ctx, "/delbucket/file1.txt")
	if err != nil {
		t.Fatalf("RemoveAll file: %v", err)
	}

	// Verify file is gone
	_, err = fs.Stat(ctx, "/delbucket/file1.txt")
	if !os.IsNotExist(err) {
		t.Errorf("file1.txt should not exist, got err: %v", err)
	}

	// Verify other files still exist
	_, err = fs.Stat(ctx, "/delbucket/file2.txt")
	if err != nil {
		t.Errorf("file2.txt should still exist, got err: %v", err)
	}

	// Delete bucket with remaining files (force mode handles non-empty)
	err = fs.RemoveAll(ctx, "/delbucket")
	if err != nil {
		t.Fatalf("RemoveAll bucket: %v", err)
	}

	// Verify bucket is gone
	_, err = fs.Stat(ctx, "/delbucket")
	if !os.IsNotExist(err) {
		t.Errorf("bucket should not exist, got err: %v", err)
	}
}

func TestStorageFileSystem_Rename(t *testing.T) {
	fs, cleanup := newTestFS(t)
	defer cleanup()

	ctx := context.Background()

	// Create bucket and file
	err := fs.Mkdir(ctx, "/renamebucket", 0755)
	if err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	file, err := fs.OpenFile(ctx, "/renamebucket/original.txt", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	_, _ = file.Write([]byte("rename me"))
	_ = file.Close()

	// Rename file
	err = fs.Rename(ctx, "/renamebucket/original.txt", "/renamebucket/renamed.txt")
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}

	// Verify original is gone
	_, err = fs.Stat(ctx, "/renamebucket/original.txt")
	if !os.IsNotExist(err) {
		t.Errorf("original.txt should not exist, got err: %v", err)
	}

	// Verify renamed file exists
	info, err := fs.Stat(ctx, "/renamebucket/renamed.txt")
	if err != nil {
		t.Fatalf("Stat renamed: %v", err)
	}
	if info.Name() != "renamed.txt" {
		t.Errorf("renamed name = %q, want %q", info.Name(), "renamed.txt")
	}
}

func TestStorageFileSystem_ReadOnly(t *testing.T) {
	store, err := local.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("open local storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	fs := &StorageFileSystem{
		store:    store,
		readOnly: true,
	}

	ctx := context.Background()

	// Mkdir should fail
	err = fs.Mkdir(ctx, "/bucket", 0755)
	if err != os.ErrPermission {
		t.Errorf("Mkdir in read-only should return ErrPermission, got: %v", err)
	}

	// RemoveAll should fail
	err = fs.RemoveAll(ctx, "/bucket/file.txt")
	if err != os.ErrPermission {
		t.Errorf("RemoveAll in read-only should return ErrPermission, got: %v", err)
	}

	// Rename should fail
	err = fs.Rename(ctx, "/bucket/a.txt", "/bucket/b.txt")
	if err != os.ErrPermission {
		t.Errorf("Rename in read-only should return ErrPermission, got: %v", err)
	}
}

func TestStorageFile_Seek(t *testing.T) {
	fs, cleanup := newTestFS(t)
	defer cleanup()

	ctx := context.Background()

	// Create bucket and file
	err := fs.Mkdir(ctx, "/seekbucket", 0755)
	if err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	content := "0123456789ABCDEF"
	file, _ := fs.OpenFile(ctx, "/seekbucket/seek.txt", os.O_WRONLY|os.O_CREATE, 0644)
	_, _ = file.Write([]byte(content))
	_ = file.Close()

	// Open for reading
	file, err = fs.OpenFile(ctx, "/seekbucket/seek.txt", os.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	defer func() { _ = file.Close() }()

	// Seek to position 5
	pos, err := file.Seek(5, io.SeekStart)
	if err != nil {
		t.Fatalf("Seek: %v", err)
	}
	if pos != 5 {
		t.Errorf("position = %d, want 5", pos)
	}

	// Read from position 5
	buf := make([]byte, 5)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read: %v", err)
	}
	if string(buf[:n]) != "56789" {
		t.Errorf("read = %q, want %q", string(buf[:n]), "56789")
	}
}

func TestStorageFile_Readdir(t *testing.T) {
	fs, cleanup := newTestFS(t)
	defer cleanup()

	ctx := context.Background()

	// Create bucket and files
	err := fs.Mkdir(ctx, "/listbucket", 0755)
	if err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	files := []string{"a.txt", "b.txt", "c.txt", "dir/nested.txt"}
	for _, name := range files {
		file, _ := fs.OpenFile(ctx, "/listbucket/"+name, os.O_WRONLY|os.O_CREATE, 0644)
		_, _ = file.Write([]byte("content"))
		_ = file.Close()
	}

	// Open directory
	dir, err := fs.OpenFile(ctx, "/listbucket", os.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("OpenFile dir: %v", err)
	}
	defer func() { _ = dir.Close() }()

	// Read directory entries
	entries, err := dir.Readdir(0)
	if err != nil && err != io.EOF {
		t.Fatalf("Readdir: %v", err)
	}

	// Should have 4 entries: a.txt, b.txt, c.txt, and dir (virtual)
	if len(entries) != 4 {
		t.Errorf("got %d entries, want 4", len(entries))
		for _, e := range entries {
			t.Logf("  entry: %s (dir=%v)", e.Name(), e.IsDir())
		}
	}

	// Check for expected entries
	found := make(map[string]bool)
	for _, e := range entries {
		found[e.Name()] = true
	}

	for _, expected := range []string{"a.txt", "b.txt", "c.txt", "dir"} {
		if !found[expected] {
			t.Errorf("missing entry: %s", expected)
		}
	}
}

func TestStorageFileInfo_ContentType(t *testing.T) {
	tests := []struct {
		filename    string
		contentType string
	}{
		{"file.txt", "text/plain"},
		{"file.html", "text/html"},
		{"file.css", "text/css"},
		{"file.js", "application/javascript"},
		{"file.json", "application/json"},
		{"file.png", "image/png"},
		{"file.jpg", "image/jpeg"},
		{"file.gif", "image/gif"},
		{"file.svg", "image/svg+xml"},
		{"file.pdf", "application/pdf"},
		{"file.zip", "application/zip"},
		{"file.unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			ct := detectContentType(tt.filename)
			if ct != tt.contentType {
				t.Errorf("detectContentType(%q) = %q, want %q", tt.filename, ct, tt.contentType)
			}
		})
	}
}

func TestMapError(t *testing.T) {
	tests := []struct {
		name  string
		input error
		want  error
	}{
		{"nil", nil, nil},
		{"not exist", os.ErrNotExist, os.ErrNotExist},
		{"exist", os.ErrExist, os.ErrExist},
		{"permission", os.ErrPermission, os.ErrPermission},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapError(tt.input)
			if got != tt.want {
				t.Errorf("mapError(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
