package searxng

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine"
)

func getSearXNGURL() string {
	if url := os.Getenv("SEARXNG_URL"); url != "" {
		return url
	}
	return "http://localhost:8888"
}

func skipIfNoSearXNG(t *testing.T) *Engine {
	t.Helper()
	eng := New(getSearXNGURL())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := eng.Healthz(ctx); err != nil {
		t.Skipf("SearXNG not available: %v", err)
	}
	return eng
}

func TestEngine_Name(t *testing.T) {
	eng := New("http://localhost:8888")
	if eng.Name() != "searxng" {
		t.Errorf("expected name 'searxng', got %q", eng.Name())
	}
}

func TestEngine_Categories(t *testing.T) {
	eng := New("http://localhost:8888")
	cats := eng.Categories()

	if len(cats) == 0 {
		t.Error("expected at least one category")
	}

	// Check that general is included
	var hasGeneral bool
	for _, c := range cats {
		if c == engine.CategoryGeneral {
			hasGeneral = true
			break
		}
	}
	if !hasGeneral {
		t.Error("expected 'general' category to be included")
	}
}

func TestEngine_Search_General(t *testing.T) {
	eng := skipIfNoSearXNG(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := eng.Search(ctx, "golang", engine.SearchOptions{
		Category: engine.CategoryGeneral,
		Page:     1,
		PerPage:  10,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if resp.Query != "golang" {
		t.Errorf("expected query 'golang', got %q", resp.Query)
	}

	if len(resp.Results) == 0 {
		t.Error("expected at least one result")
	}

	// Check first result has required fields
	if len(resp.Results) > 0 {
		r := resp.Results[0]
		if r.URL == "" {
			t.Error("expected result to have URL")
		}
		if r.Title == "" {
			t.Error("expected result to have title")
		}
		if r.Engine == "" {
			t.Error("expected result to have engine")
		}
	}

	t.Logf("Got %d results for 'golang', search time: %.2fms", len(resp.Results), resp.SearchTimeMs)
}

func TestEngine_Search_Images(t *testing.T) {
	eng := skipIfNoSearXNG(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := eng.Search(ctx, "golang logo", engine.SearchOptions{
		Category: engine.CategoryImages,
		Page:     1,
		PerPage:  10,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(resp.Results) == 0 {
		t.Skip("no image results returned (may depend on enabled engines)")
	}

	// Check first result has image-specific fields
	r := resp.Results[0]
	t.Logf("First image result: %+v", r)

	if r.URL == "" {
		t.Error("expected result to have URL")
	}
}

func TestEngine_Search_Videos(t *testing.T) {
	eng := skipIfNoSearXNG(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := eng.Search(ctx, "golang tutorial", engine.SearchOptions{
		Category: engine.CategoryVideos,
		Page:     1,
		PerPage:  10,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(resp.Results) == 0 {
		t.Skip("no video results returned (may depend on enabled engines)")
	}

	t.Logf("Got %d video results", len(resp.Results))
}

func TestEngine_Search_News(t *testing.T) {
	eng := skipIfNoSearXNG(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := eng.Search(ctx, "technology", engine.SearchOptions{
		Category: engine.CategoryNews,
		Page:     1,
		PerPage:  10,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(resp.Results) == 0 {
		t.Skip("no news results returned (may depend on enabled engines)")
	}

	t.Logf("Got %d news results", len(resp.Results))
}

func TestEngine_Search_IT(t *testing.T) {
	eng := skipIfNoSearXNG(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := eng.Search(ctx, "mizu framework", engine.SearchOptions{
		Category: engine.CategoryIT,
		Page:     1,
		PerPage:  10,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	t.Logf("Got %d IT results", len(resp.Results))
}

func TestEngine_Search_Science(t *testing.T) {
	eng := skipIfNoSearXNG(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := eng.Search(ctx, "machine learning", engine.SearchOptions{
		Category: engine.CategoryScience,
		Page:     1,
		PerPage:  10,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	t.Logf("Got %d science results", len(resp.Results))
}

func TestEngine_Search_WithTimeRange(t *testing.T) {
	eng := skipIfNoSearXNG(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := eng.Search(ctx, "golang", engine.SearchOptions{
		Category:  engine.CategoryGeneral,
		Page:      1,
		PerPage:   10,
		TimeRange: "week",
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	t.Logf("Got %d results for 'golang' in the past week", len(resp.Results))
}

func TestEngine_Search_WithLanguage(t *testing.T) {
	eng := skipIfNoSearXNG(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := eng.Search(ctx, "programming", engine.SearchOptions{
		Category: engine.CategoryGeneral,
		Page:     1,
		PerPage:  10,
		Language: "en",
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	t.Logf("Got %d results for 'programming' in English", len(resp.Results))
}

func TestEngine_Healthz(t *testing.T) {
	eng := New(getSearXNGURL())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := eng.Healthz(ctx)
	if err != nil {
		t.Logf("SearXNG health check failed (this is expected if SearXNG is not running): %v", err)
	} else {
		t.Log("SearXNG is healthy")
	}
}
