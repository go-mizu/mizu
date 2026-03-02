package crawl_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
)

func TestRunJob_Pass1Only(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	seeds := make([]crawl.SeedURL, 10)
	for i := range seeds {
		seeds[i] = crawl.SeedURL{
			URL:    fmt.Sprintf("%s/page/%d", srv.URL, i),
			Domain: "localhost",
			Host:   "localhost",
		}
	}

	result, err := crawl.RunJob(context.Background(), seeds, &crawl.NoopDNS{}, crawl.JobConfig{
		Engine:     "keepalive",
		Workers:    4,
		Timeout:    2 * time.Second,
		StatusOnly: true,
		// nil writers → devnull
	})
	if err != nil {
		t.Fatalf("RunJob: %v", err)
	}
	if result.Pass1 == nil {
		t.Fatal("Pass1 stats should not be nil")
	}
	if result.Pass1.OK != 10 {
		t.Errorf("want 10 OK, got %d", result.Pass1.OK)
	}
	if result.Pass2 != nil {
		t.Error("Pass2 should be nil when NoRetry/RetryTimeout=0")
	}
	if result.Total == nil {
		t.Fatal("Total stats should not be nil")
	}
}

func TestRunJob_AutoWorkers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	seeds := []crawl.SeedURL{{URL: srv.URL + "/", Domain: "localhost", Host: "localhost"}}

	result, err := crawl.RunJob(context.Background(), seeds, &crawl.NoopDNS{}, crawl.JobConfig{
		Engine:     "keepalive",
		Workers:    -1, // auto
		Timeout:    2 * time.Second,
		StatusOnly: true,
	})
	if err != nil {
		t.Fatalf("RunJob: %v", err)
	}
	if result.Workers <= 0 {
		t.Error("auto workers should be positive")
	}
}
