package fts_zig

import (
	"testing"
)

func TestIPCDriver(t *testing.T) {
	driver, err := NewIPCDriver(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create IPC driver: %v", err)
	}
	defer driver.Close()

	// Add documents
	_, err = driver.AddDocument("hello world")
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	_, err = driver.AddDocument("hello there")
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	_, err = driver.AddDocument("world peace")
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	// Build
	if err := driver.Build(); err != nil {
		t.Fatalf("failed to build: %v", err)
	}

	// Search
	results, err := driver.Search("hello", 10)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestMmapDriver(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BasePath = t.TempDir()

	driver, err := NewMmapDriver(cfg)
	if err != nil {
		t.Fatalf("failed to create mmap driver: %v", err)
	}
	defer driver.Close()

	// Add documents
	_, err = driver.AddDocument("the quick brown fox")
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	_, err = driver.AddDocument("the lazy brown dog")
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	// Build
	if err := driver.Build(); err != nil {
		t.Fatalf("failed to build: %v", err)
	}

	// Search
	results, err := driver.Search("brown", 10)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestDriverStats(t *testing.T) {
	driver, err := NewIPCDriver(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}
	defer driver.Close()

	for i := 0; i < 100; i++ {
		_, err := driver.AddDocument("test document")
		if err != nil {
			t.Fatalf("failed to add document: %v", err)
		}
	}

	stats, err := driver.Stats()
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if stats.DocCount != 100 {
		t.Errorf("expected 100 docs, got %d", stats.DocCount)
	}
}

func BenchmarkIPCAddDocument(b *testing.B) {
	driver, err := NewIPCDriver(DefaultConfig())
	if err != nil {
		b.Fatalf("failed to create driver: %v", err)
	}
	defer driver.Close()

	text := "This is a test document with some words for benchmarking purposes."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = driver.AddDocument(text)
	}
}

func BenchmarkMmapAddDocument(b *testing.B) {
	cfg := DefaultConfig()
	cfg.BasePath = b.TempDir()

	driver, err := NewMmapDriver(cfg)
	if err != nil {
		b.Fatalf("failed to create driver: %v", err)
	}
	defer driver.Close()

	text := "This is a test document with some words for benchmarking purposes."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = driver.AddDocument(text)
	}
}
