package amazon

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestReviewTask404IsSkipped(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	db, err := OpenDB(filepath.Join(t.TempDir(), "amazon.duckdb"))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	state, err := OpenState(filepath.Join(t.TempDir(), "state.duckdb"))
	if err != nil {
		t.Fatalf("OpenState: %v", err)
	}
	defer state.Close()

	client := NewClient(Config{Workers: 1, Timeout: DefaultTimeout})
	task := &ReviewTask{
		URL:     srv.URL + "/product-reviews/B0GL7WD892",
		ASIN:    "B0GL7WD892",
		Client:  client,
		DB:      db,
		StateDB: state,
	}

	metric, err := task.Run(context.Background(), func(*ReviewState) {})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if metric.Skipped != 1 || metric.Fetched != 0 || metric.Pages != 0 {
		t.Fatalf("unexpected metric: %+v", metric)
	}
}

func TestQATask404IsSkipped(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	db, err := OpenDB(filepath.Join(t.TempDir(), "amazon.duckdb"))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	state, err := OpenState(filepath.Join(t.TempDir(), "state.duckdb"))
	if err != nil {
		t.Fatalf("OpenState: %v", err)
	}
	defer state.Close()

	client := NewClient(Config{Workers: 1, Timeout: DefaultTimeout})
	task := &QATask{
		URL:     srv.URL + "/ask/B0GL7WD892",
		ASIN:    "B0GL7WD892",
		Client:  client,
		DB:      db,
		StateDB: state,
	}

	metric, err := task.Run(context.Background(), func(*QAState) {})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if metric.Skipped != 1 || metric.Fetched != 0 || metric.Pages != 0 {
		t.Fatalf("unexpected metric: %+v", metric)
	}
}
