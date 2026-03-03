package rose_test

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/flower/rose"
)

var benchQueries = []string{
	"machine learning",
	"climate change",
	"artificial intelligence",
	"United States",
	"open source software",
	"COVID-19 pandemic",
	"data privacy",
	"renewable energy",
	"blockchain technology",
	"neural network",
}

// BenchmarkRose_Index measures indexing throughput. Each operation indexes
// one document with ~200 tokens drawn from a 10,000-word vocabulary.
func BenchmarkRose_Index(b *testing.B) {
	dir := b.TempDir()
	eng, err := index.NewEngine("rose")
	if err != nil {
		b.Fatalf("NewEngine: %v", err)
	}
	if err := eng.Open(context.Background(), dir); err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	vocab := make([]string, 10000)
	for i := range vocab {
		vocab[i] = fmt.Sprintf("word%d", i)
	}

	rng := rand.New(rand.NewSource(42))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		words := make([]string, 200)
		for j := range words {
			words[j] = vocab[rng.Intn(len(vocab))]
		}
		body := strings.Join(words, " ")
		if err := eng.Index(context.Background(), []index.Document{
			{DocID: fmt.Sprintf("doc-%d", i), Text: []byte(body)},
		}); err != nil {
			b.Fatalf("Index: %v", err)
		}
	}
	b.ReportAllocs()
}

// BenchmarkRose_Search measures search latency over a pre-indexed corpus of
// 10,000 documents. Queries cycle through the 10-query standard set.
func BenchmarkRose_Search(b *testing.B) {
	dir := b.TempDir()
	eng, err := index.NewEngine("rose")
	if err != nil {
		b.Fatalf("NewEngine: %v", err)
	}
	if err := eng.Open(context.Background(), dir); err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	// Pre-index 10,000 docs with a 1,000-word vocab; inject query words
	// at ~10% frequency so queries return real hits.
	vocab := make([]string, 1000)
	for i := range vocab {
		vocab[i] = fmt.Sprintf("word%d", i)
	}
	queryWords := strings.Fields(strings.Join(benchQueries, " "))

	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 10000; i++ {
		words := make([]string, 200)
		for j := range words {
			if rng.Intn(10) == 0 && len(queryWords) > 0 {
				words[j] = queryWords[rng.Intn(len(queryWords))]
			} else {
				words[j] = vocab[rng.Intn(len(vocab))]
			}
		}
		body := strings.Join(words, " ")
		if err := eng.Index(context.Background(), []index.Document{
			{DocID: fmt.Sprintf("doc-%d", i), Text: []byte(body)},
		}); err != nil {
			b.Fatalf("Index: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := benchQueries[i%len(benchQueries)]
		if _, err := eng.Search(context.Background(), index.Query{Text: q, Limit: 10}); err != nil {
			b.Fatalf("Search: %v", err)
		}
	}
	b.ReportAllocs()
}

// TestRoseDiskUsage indexes 10,000 docs of ~200 tokens, flushes, and reports
// disk usage via Stats(). Intended to be run with -v to observe the output.
func TestRoseDiskUsage(t *testing.T) {
	dir := t.TempDir()
	eng, err := index.NewEngine("rose")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	if err := eng.Open(context.Background(), dir); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer eng.Close()

	vocab := make([]string, 10000)
	for i := range vocab {
		vocab[i] = fmt.Sprintf("word%d", i)
	}

	rng := rand.New(rand.NewSource(42))
	const numDocs = 10000
	for i := 0; i < numDocs; i++ {
		words := make([]string, 200)
		for j := range words {
			words[j] = vocab[rng.Intn(len(vocab))]
		}
		body := strings.Join(words, " ")
		if err := eng.Index(context.Background(), []index.Document{
			{DocID: fmt.Sprintf("doc-%d", i), Text: []byte(body)},
		}); err != nil {
			t.Fatalf("Index: %v", err)
		}
	}

	// Close flushes remaining in-memory data to disk before Stats.
	if err := eng.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Re-open to measure on-disk state.
	eng2, err := index.NewEngine("rose")
	if err != nil {
		t.Fatalf("NewEngine (reopen): %v", err)
	}
	if err := eng2.Open(context.Background(), dir); err != nil {
		t.Fatalf("Open (reopen): %v", err)
	}
	defer eng2.Close()

	stats, err := eng2.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	t.Logf("DocCount  = %d", stats.DocCount)
	t.Logf("DiskBytes = %d  (%.2f MB)", stats.DiskBytes, float64(stats.DiskBytes)/(1<<20))
}
