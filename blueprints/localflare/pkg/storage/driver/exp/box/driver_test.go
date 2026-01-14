package box

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mizu/blueprints/drive/lib/storage"
)

// ensure driver implements storage.Driver
var _ storage.Driver = (*driver)(nil)

func TestParseRoot(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "data")

	tests := []struct {
		name    string
		dsn     string
		want    string
		wantErr string
	}{
		{name: "simple", dsn: "box:/" + filepath.ToSlash(tmp), want: filepath.Clean("/" + filepath.ToSlash(tmp))},
		{name: "url_style", dsn: "box:///var/data", want: filepath.FromSlash("/var/data")},
		{name: "empty_dsn", dsn: "", wantErr: "empty dsn"},
		{name: "missing_path", dsn: "box:", wantErr: "missing path"},
		{name: "unsupported", dsn: "local:/tmp", wantErr: "unsupported"},
		{name: "relative", dsn: "box:relative/path", wantErr: "absolute"},
		{name: "parse_error", dsn: "box://%", wantErr: "parse dsn"},
		{name: "empty_url_path", dsn: "box://", wantErr: "empty path"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseRoot(tc.dsn)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseRoot returned error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("parseRoot(%q) = %q, want %q", tc.dsn, got, tc.want)
			}
		})
	}
}

func TestOpenDelegatesToLocal(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()

	st, err := storage.Open(ctx, "box:"+root)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer func() {
		_ = st.Close()
	}()

	// basic bucket operations using delegated local backend
	_, err = st.CreateBucket(ctx, "bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	b := st.Bucket("bucket")
	obj, err := b.Write(ctx, "file.txt", strings.NewReader("data"), 4, "text/plain", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if obj.Key != "file.txt" {
		t.Fatalf("unexpected object key %q", obj.Key)
	}

	rc, meta, err := b.Open(ctx, "file.txt", 0, 0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() {
		_ = rc.Close()
	}()

	buf, _ := io.ReadAll(rc)
	if string(buf) != "data" {
		t.Fatalf("unexpected content %q", string(buf))
	}
	if meta == nil || meta.Key != "file.txt" {
		t.Fatalf("unexpected metadata %#v", meta)
	}
}
