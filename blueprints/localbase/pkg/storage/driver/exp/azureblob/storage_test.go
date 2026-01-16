package azureblob

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/localbase/pkg/storage"
)

func TestParseDSN(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		wantErr bool
	}{
		{"empty", "", true},
		{"wrong scheme", "http://acct/b", true},
		{"missing account", "azureblob:///bucket", true},
		{"ok", "azureblob://acct/b1?default_public=true", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			acct, bucket, public, err := parseDSN(tc.dsn)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDSN returned error: %v", err)
			}
			if acct != "acct" || bucket != "b1" || !public {
				t.Fatalf("unexpected values: %q %q %v", acct, bucket, public)
			}
		})
	}
}

func TestDriverOpenUsesContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	d := &driver{}
	if _, err := d.Open(ctx, "azureblob://acct/b"); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context error, got %v", err)
	}
}

func TestDriverOpenCustomStore(t *testing.T) {
	ctx := context.Background()
	called := false
	d := &driver{
		newStore: func(account, container string, public bool) *store {
			called = true
			if account != "acct" || container != "b" || public {
				t.Fatalf("unexpected params: %q %q %v", account, container, public)
			}
			return newStore(account, container, public)
		},
	}

	if _, err := d.Open(ctx, "azureblob://acct/b"); err != nil {
		t.Fatalf("open: %v", err)
	}
	if !called {
		t.Fatalf("expected custom store to be used")
	}
}

func TestStoreBucketLifecycle(t *testing.T) {
	st := newStore("acct", "default", true)

	// Default bucket creation via Bucket("")
	b := st.Bucket("")
	if b.Name() != "default" {
		t.Fatalf("unexpected bucket name: %s", b.Name())
	}

	// CreateBucket success and duplicate detection
	if _, err := st.CreateBucket(context.Background(), "photos", nil); err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	if _, err := st.CreateBucket(context.Background(), "photos", nil); !errors.Is(err, storage.ErrExist) {
		t.Fatalf("expected ErrExist, got %v", err)
	}

	// Buckets listing honors limit/offset
	iter, err := st.Buckets(context.Background(), 1, 1, nil)
	if err != nil {
		t.Fatalf("buckets: %v", err)
	}
	info, err := iter.Next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if info.Name == "" || info.Metadata["account"] != "acct" {
		t.Fatalf("unexpected bucket info: %+v", info)
	}
	if _, err := iter.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestDeleteBucketChecks(t *testing.T) {
	st := newStore("acct", "", false)
	if err := st.DeleteBucket(context.Background(), "missing", nil); !errors.Is(err, storage.ErrNotExist) {
		t.Fatalf("expected ErrNotExist, got %v", err)
	}

	b := st.Bucket("data")
	if _, err := b.Write(context.Background(), "k", strings.NewReader("v"), -1, "text/plain", nil); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := st.DeleteBucket(context.Background(), "data", storage.Options{}); !errors.Is(err, storage.ErrPermission) {
		t.Fatalf("expected ErrPermission, got %v", err)
	}
	if err := st.DeleteBucket(context.Background(), "data", storage.Options{"force": true}); err != nil {
		t.Fatalf("force delete: %v", err)
	}
}

func TestBucketObjectOperations(t *testing.T) {
	st := newStore("acct", "", false)
	b := st.Bucket("data")

	// Write and Stat
	obj, err := b.Write(context.Background(), "file.txt", strings.NewReader("hello"), 5, "text/plain", storage.Options{"metadata": map[string]string{"a": "b"}})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if obj.Key != "file.txt" || obj.Metadata["a"] != "b" {
		t.Fatalf("unexpected obj: %+v", obj)
	}
	if _, err := b.Write(context.Background(), "file.txt", strings.NewReader("dup"), -1, "text/plain", nil); !errors.Is(err, storage.ErrExist) {
		t.Fatalf("expected ErrExist, got %v", err)
	}

	stat, err := b.Stat(context.Background(), "file.txt", nil)
	if err != nil || stat.Key != "file.txt" {
		t.Fatalf("stat: %v %#v", err, stat)
	}

	// Open range
	rc, rangedObj, err := b.Open(context.Background(), "file.txt", 1, 2, nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	data, _ := io.ReadAll(rc)
	if string(data) != "el" || rangedObj.Size != 2 {
		t.Fatalf("unexpected range: %q size %d", string(data), rangedObj.Size)
	}
	_ = rc.Close()

	// Copy and Move
	if _, err := b.Copy(context.Background(), "copy.txt", "", "missing", nil); !errors.Is(err, storage.ErrNotExist) {
		t.Fatalf("expected ErrNotExist, got %v", err)
	}
	if _, err := b.Copy(context.Background(), "copy.txt", "other", "file.txt", nil); !errors.Is(err, storage.ErrUnsupported) {
		t.Fatalf("expected ErrUnsupported, got %v", err)
	}
	if _, err := b.Copy(context.Background(), "copy.txt", "", "file.txt", nil); err != nil {
		t.Fatalf("copy: %v", err)
	}
	if _, err := b.Move(context.Background(), "moved.txt", "", "copy.txt", nil); err != nil {
		t.Fatalf("move: %v", err)
	}
	if err := b.Delete(context.Background(), "copy.txt", nil); !errors.Is(err, storage.ErrNotExist) {
		t.Fatalf("expected ErrNotExist, got %v", err)
	}
}

func TestListAndSignedURL(t *testing.T) {
	st := newStore("acct", "", false)
	b := st.Bucket("docs")
	_, _ = b.Write(context.Background(), "dir/", strings.NewReader(""), 0, "", nil)
	_, _ = b.Write(context.Background(), "dir/file.txt", strings.NewReader("data"), -1, "", nil)

	iter, err := b.List(context.Background(), "dir", 0, 0, storage.Options{"dirs_only": true})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	obj, err := iter.Next()
	if err != nil || obj.Key != "dir/" {
		t.Fatalf("unexpected list result: %#v %v", obj, err)
	}
	if _, err := iter.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF, got %v", err)
	}

	// files_only branch
	iter, err = b.List(context.Background(), "dir", 10, 0, storage.Options{"files_only": true})
	if err != nil {
		t.Fatalf("list files: %v", err)
	}
	if _, err := iter.Next(); err != nil {
		t.Fatalf("expected file entry: %v", err)
	}

	url, err := b.SignedURL(context.Background(), "dir/file.txt", "GET", time.Minute, nil)
	if err != nil || !strings.Contains(url, "dir/file.txt") {
		t.Fatalf("signed url: %v %q", err, url)
	}
	if _, err := b.SignedURL(context.Background(), "missing", "GET", time.Minute, nil); !errors.Is(err, storage.ErrNotExist) {
		t.Fatalf("expected ErrNotExist, got %v", err)
	}
}

func TestWriteSizeMismatch(t *testing.T) {
	st := newStore("acct", "", false)
	b := st.Bucket("data")
	if _, err := b.Write(context.Background(), "f", strings.NewReader("abc"), 10, "", nil); err == nil {
		t.Fatalf("expected size mismatch error")
	}
}
