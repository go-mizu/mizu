package crawler

import (
	"sync"
	"testing"
	"time"
)

func TestFrontierPushPop(t *testing.T) {
	f := NewFrontier(0)

	f.Push(URLEntry{URL: "https://example.com/a", Depth: 0, Priority: 1})
	f.Push(URLEntry{URL: "https://example.com/b", Depth: 0, Priority: 0})
	f.Push(URLEntry{URL: "https://example.com/c", Depth: 0, Priority: 2})

	// Should pop in priority order (lower = higher priority)
	entry, ok := f.TryPop()
	if !ok {
		t.Fatal("expected entry")
	}
	if entry.URL != "https://example.com/b" {
		t.Errorf("first pop URL = %q, want /b (priority 0)", entry.URL)
	}

	entry, ok = f.TryPop()
	if !ok {
		t.Fatal("expected entry")
	}
	if entry.URL != "https://example.com/a" {
		t.Errorf("second pop URL = %q, want /a (priority 1)", entry.URL)
	}
}

func TestFrontierDedup(t *testing.T) {
	f := NewFrontier(0)

	added := f.Push(URLEntry{URL: "https://example.com/page"})
	if !added {
		t.Error("first push should succeed")
	}

	added = f.Push(URLEntry{URL: "https://example.com/page"})
	if added {
		t.Error("duplicate push should return false")
	}

	if f.Len() != 1 {
		t.Errorf("Len = %d, want 1", f.Len())
	}
}

func TestFrontierNormalization(t *testing.T) {
	f := NewFrontier(0)

	f.Push(URLEntry{URL: "https://example.com/page"})
	// Same URL with trailing slash should be deduped
	added := f.Push(URLEntry{URL: "https://EXAMPLE.COM/page"})
	if added {
		t.Error("normalized duplicate should be rejected")
	}
}

func TestFrontierVisited(t *testing.T) {
	f := NewFrontier(0)

	f.Push(URLEntry{URL: "https://example.com/a"})
	f.TryPop()

	if !f.IsVisited("https://example.com/a") {
		t.Error("should be visited after push")
	}
	if f.IsVisited("https://example.com/b") {
		t.Error("should not be visited")
	}
	if f.VisitedCount() != 1 {
		t.Errorf("VisitedCount = %d, want 1", f.VisitedCount())
	}
}

func TestFrontierClose(t *testing.T) {
	f := NewFrontier(0)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, ok := f.Pop()
		if ok {
			t.Error("Pop should return false when closed and empty")
		}
	}()

	time.Sleep(10 * time.Millisecond)
	f.Close()
	wg.Wait()
}

func TestFrontierConcurrent(t *testing.T) {
	f := NewFrontier(0)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			f.Push(URLEntry{
				URL:      "https://example.com/" + string(rune('a'+i%26)),
				Priority: i % 5,
			})
		}(i)
	}
	wg.Wait()

	// All URLs should be in visited
	if f.VisitedCount() == 0 {
		t.Error("expected some visited URLs")
	}
}

func TestFrontierDomainDelay(t *testing.T) {
	f := NewFrontier(50 * time.Millisecond)

	start := time.Now()
	f.WaitForDomain("example.com")
	f.WaitForDomain("example.com")
	elapsed := time.Since(start)

	if elapsed < 40*time.Millisecond {
		t.Errorf("second WaitForDomain should have waited ~50ms, elapsed %v", elapsed)
	}
}
