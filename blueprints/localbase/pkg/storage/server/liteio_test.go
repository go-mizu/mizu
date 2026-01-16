package server_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/localbase/pkg/storage"
	_ "github.com/go-mizu/blueprints/localbase/pkg/storage/driver/exp/s3"
)

// TestLiteIOConnection tests connecting to a running liteio instance.
// Set LITEIO_ENDPOINT to the liteio endpoint (default: localhost:9200)
func TestLiteIOConnection(t *testing.T) {
	endpoint := os.Getenv("LITEIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9200"
	}

	// Check if liteio is running
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dsn := "s3://liteio:liteio123@" + endpoint + "/test-bucket?insecure=true&force_path_style=true"
	t.Logf("Connecting to liteio at %s", endpoint)

	st, err := storage.Open(ctx, dsn)
	if err != nil {
		t.Skipf("liteio not available at %s: %v", endpoint, err)
	}
	defer st.Close()

	// Test bucket operations
	t.Run("CreateBucket", func(t *testing.T) {
		ctx := context.Background()
		_, err := st.CreateBucket(ctx, "test-bucket", nil)
		if err != nil && err != storage.ErrExist {
			t.Fatalf("create bucket: %v", err)
		}
	})

	t.Run("ListBuckets", func(t *testing.T) {
		ctx := context.Background()
		iter, err := st.Buckets(ctx, 0, 0, nil)
		if err != nil {
			t.Fatalf("list buckets: %v", err)
		}
		defer iter.Close()

		count := 0
		for {
			info, err := iter.Next()
			if err != nil {
				t.Fatalf("iter next: %v", err)
			}
			if info == nil {
				break
			}
			t.Logf("Bucket: %s (created: %v)", info.Name, info.CreatedAt)
			count++
		}
		t.Logf("Found %d buckets", count)
	})

	bucket := st.Bucket("test-bucket")

	t.Run("WriteObject", func(t *testing.T) {
		ctx := context.Background()
		data := []byte("Hello, LiteIO!")
		obj, err := bucket.Write(ctx, "test-key.txt", bytes.NewReader(data), int64(len(data)), "text/plain", nil)
		if err != nil {
			t.Fatalf("write object: %v", err)
		}
		t.Logf("Wrote object: %s (size: %d)", obj.Key, obj.Size)
	})

	t.Run("ReadObject", func(t *testing.T) {
		ctx := context.Background()
		rc, obj, err := bucket.Open(ctx, "test-key.txt", 0, 0, nil)
		if err != nil {
			t.Fatalf("open object: %v", err)
		}
		defer rc.Close()

		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read object: %v", err)
		}

		expected := "Hello, LiteIO!"
		if string(data) != expected {
			t.Errorf("got %q, want %q", string(data), expected)
		}
		t.Logf("Read object: %s (size: %d)", obj.Key, len(data))
	})

	t.Run("StatObject", func(t *testing.T) {
		ctx := context.Background()
		obj, err := bucket.Stat(ctx, "test-key.txt", nil)
		if err != nil {
			t.Fatalf("stat object: %v", err)
		}
		t.Logf("Stat: key=%s size=%d", obj.Key, obj.Size)
	})

	t.Run("ListObjects", func(t *testing.T) {
		ctx := context.Background()
		iter, err := bucket.List(ctx, "", 0, 0, nil)
		if err != nil {
			t.Fatalf("list objects: %v", err)
		}
		defer iter.Close()

		count := 0
		for {
			obj, err := iter.Next()
			if err != nil {
				t.Fatalf("iter next: %v", err)
			}
			if obj == nil {
				break
			}
			t.Logf("Object: %s (size: %d)", obj.Key, obj.Size)
			count++
		}
		t.Logf("Found %d objects", count)
	})

	t.Run("DeleteObject", func(t *testing.T) {
		ctx := context.Background()
		err := bucket.Delete(ctx, "test-key.txt", nil)
		if err != nil {
			t.Fatalf("delete object: %v", err)
		}
		t.Log("Deleted test-key.txt")
	})

	t.Log("All LiteIO connection tests passed!")
}
