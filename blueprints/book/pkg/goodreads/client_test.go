package goodreads

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetPopularListsFallsBackToPopularListsPage(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><script id="__NEXT_DATA__" type="application/json">{}</script></html>`))
	})
	mux.HandleFunc("/list/popular_lists", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`
<div>
  <a class="listTitle" href="/list/show/1.Best_Books_Ever">Best Books Ever</a>
  <span>100 books — 2,000 voters</span>
</div>`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient()
	c.baseURL = srv.URL

	lists, err := c.GetPopularLists(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lists) != 1 {
		t.Fatalf("expected 1 list, got %d", len(lists))
	}
	if lists[0].GoodreadsID != "1" {
		t.Fatalf("expected id 1, got %q", lists[0].GoodreadsID)
	}
}

func TestGetPopularListsByTag(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/list/tag/fantasy", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`
<div>
  <a class="listTitle" href="/list/show/50.The_Best_Epic_Fantasy_fiction_">The Best Epic Fantasy (fiction)</a>
  <span>4,248 books — 25,925 voters</span>
</div>`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient()
	c.baseURL = srv.URL

	lists, err := c.GetPopularLists(context.Background(), "fantasy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lists) != 1 {
		t.Fatalf("expected 1 list, got %d", len(lists))
	}
	if lists[0].Tag != "fantasy" {
		t.Fatalf("expected tag fantasy, got %q", lists[0].Tag)
	}
}

