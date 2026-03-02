package qlocal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPullModels_HFCacheAndETag(t *testing.T) {
	var headCount, getCount int
	modelBody := []byte("fake-gguf-model")
	etag := "v1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/resolve/main/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("ETag", etag)
		if r.Method == http.MethodHead {
			headCount++
			return
		}
		if r.Method == http.MethodGet {
			getCount++
			_, _ = w.Write(modelBody)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}))
	defer srv.Close()

	t.Setenv("QLOCAL_HF_BASE_URL", srv.URL)
	cacheDir := filepath.Join(t.TempDir(), "models")
	t.Setenv("QLOCAL_MODEL_CACHE_DIR", cacheDir)
	app := newTestEnv(t).App

	modelURI := "hf:testuser/testrepo/model.gguf"
	results, err := app.Pull(context.Background(), PullOptions{Models: []string{modelURI}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Refreshed || !results[0].Downloaded {
		t.Fatalf("unexpected first pull result: %+v", results)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "model.gguf")); err != nil {
		t.Fatalf("model file missing: %v", err)
	}

	results, err = app.Pull(context.Background(), PullOptions{Models: []string{modelURI}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Refreshed {
		t.Fatalf("expected cached result, got %+v", results)
	}
	if getCount != 1 {
		t.Fatalf("GET count=%d want 1 (cached second pull)", getCount)
	}
	if headCount < 2 {
		t.Fatalf("HEAD count=%d want >=2", headCount)
	}
}
