package fts_zig

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// Vietnamese sample texts for synthetic benchmarking.
// Used when real parquet data is not available.
var vietnameseTemplates = []string{
	"Việt Nam là một quốc gia nằm ở Đông Nam Á với diện tích 331.212 km vuông và dân số hơn 100 triệu người",
	"Thành phố Hồ Chí Minh là trung tâm kinh tế lớn nhất cả nước với GDP chiếm hơn 20% tổng sản phẩm quốc nội",
	"Hà Nội là thủ đô nghìn năm văn hiến với nhiều di tích lịch sử và danh lam thắng cảnh nổi tiếng",
	"Đà Nẵng là thành phố đáng sống nhất Việt Nam với bãi biển đẹp và cơ sở hạ tầng hiện đại",
	"Phú Quốc là hòn đảo lớn nhất Việt Nam thu hút hàng triệu khách du lịch mỗi năm",
	"Nông nghiệp đóng vai trò quan trọng trong nền kinh tế Việt Nam đặc biệt là xuất khẩu gạo và cà phê",
	"Giáo dục Việt Nam đang trong quá trình đổi mới và hội nhập quốc tế với nhiều cải cách quan trọng",
	"Công nghệ thông tin là ngành phát triển nhanh nhất tại Việt Nam với hàng nghìn doanh nghiệp khởi nghiệp",
	"Ẩm thực Việt Nam nổi tiếng thế giới với phở bún chả bánh mì và nhiều món ăn đặc sắc khác",
	"Du lịch Việt Nam phát triển mạnh mẽ với Hạ Long Hội An Sapa là những điểm đến hàng đầu",
}

// generateSyntheticTexts creates n synthetic Vietnamese documents.
func generateSyntheticTexts(n int) []string {
	texts := make([]string, n)
	for i := range texts {
		// Combine 2-3 templates to create ~300-500 byte documents
		idx := i % len(vietnameseTemplates)
		idx2 := (i*7 + 3) % len(vietnameseTemplates)
		texts[i] = vietnameseTemplates[idx] + ". " + vietnameseTemplates[idx2]
	}
	return texts
}

// TestImportFromParquet verifies the bulk import function works.
func TestImportFromParquet(t *testing.T) {
	driver, err := NewIPCDriver(DefaultConfig())
	if err != nil {
		t.Fatalf("create driver: %v", err)
	}
	defer driver.Close()

	texts := generateSyntheticTexts(100)
	n, err := ImportFromParquet(driver, texts)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if n != 100 {
		t.Errorf("expected 100 docs, got %d", n)
	}

	if err := driver.Build(); err != nil {
		t.Fatalf("build: %v", err)
	}

	results, err := driver.Search("Việt Nam", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected search results for 'Việt Nam'")
	}
}

// TestImportFromParquetIter verifies the streaming import function works.
func TestImportFromParquetIter(t *testing.T) {
	driver, err := NewIPCDriver(DefaultConfig())
	if err != nil {
		t.Fatalf("create driver: %v", err)
	}
	defer driver.Close()

	texts := generateSyntheticTexts(1000)

	// Create batches of 100 to simulate parquet row group batches
	batchSize := 100
	batches := func(yield func([]string, error) bool) {
		for i := 0; i < len(texts); i += batchSize {
			end := i + batchSize
			if end > len(texts) {
				end = len(texts)
			}
			if !yield(texts[i:end], nil) {
				return
			}
		}
	}

	n, err := ImportFromParquetIter(driver, batches)
	if err != nil {
		t.Fatalf("import iter: %v", err)
	}
	if n != 1000 {
		t.Errorf("expected 1000 docs, got %d", n)
	}

	stats, err := driver.Stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.DocCount != 1000 {
		t.Errorf("expected 1000 doc count, got %d", stats.DocCount)
	}
}

// BenchmarkImportBulk benchmarks bulk document import (load all, then index).
func BenchmarkImportBulk(b *testing.B) {
	sizes := []int{1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("docs=%d", size), func(b *testing.B) {
			texts := generateSyntheticTexts(size)

			// Measure total bytes for throughput reporting
			var totalBytes int64
			for _, t := range texts {
				totalBytes += int64(len(t))
			}

			b.SetBytes(totalBytes)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				driver, err := NewIPCDriver(DefaultConfig())
				if err != nil {
					b.Fatal(err)
				}
				_, err = ImportFromParquet(driver, texts)
				if err != nil {
					b.Fatal(err)
				}
				driver.Close()
			}
		})
	}
}

// BenchmarkImportStreaming benchmarks streaming batch import.
func BenchmarkImportStreaming(b *testing.B) {
	sizes := []int{1000, 10000, 100000}
	batchSize := 1000

	for _, size := range sizes {
		b.Run(fmt.Sprintf("docs=%d/batch=%d", size, batchSize), func(b *testing.B) {
			texts := generateSyntheticTexts(size)

			var totalBytes int64
			for _, t := range texts {
				totalBytes += int64(len(t))
			}

			b.SetBytes(totalBytes)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				driver, err := NewIPCDriver(DefaultConfig())
				if err != nil {
					b.Fatal(err)
				}

				batches := func(yield func([]string, error) bool) {
					for j := 0; j < len(texts); j += batchSize {
						end := j + batchSize
						if end > len(texts) {
							end = len(texts)
						}
						if !yield(texts[j:end], nil) {
							return
						}
					}
				}

				_, err = ImportFromParquetIter(driver, batches)
				if err != nil {
					b.Fatal(err)
				}
				driver.Close()
			}
		})
	}
}

// BenchmarkIndexingThroughput measures raw indexing throughput
// comparable to the Zig benchmark (docs/sec and MB/sec).
func BenchmarkIndexingThroughput(b *testing.B) {
	const numDocs = 50000
	texts := generateSyntheticTexts(numDocs)

	var totalBytes int64
	for _, t := range texts {
		totalBytes += int64(len(t))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		driver, err := NewIPCDriver(DefaultConfig())
		if err != nil {
			b.Fatal(err)
		}

		start := time.Now()
		_, _ = ImportFromParquet(driver, texts)
		elapsed := time.Since(start)

		driver.Close()

		docsPerSec := float64(numDocs) / elapsed.Seconds()
		mbPerSec := float64(totalBytes) / elapsed.Seconds() / 1024 / 1024

		if i == 0 {
			b.Logf("Go IPC driver: %.0f docs/sec, %.1f MB/sec (%d docs, %.1f MB)",
				docsPerSec, mbPerSec, numDocs, float64(totalBytes)/1024/1024)
		}
	}
}

// BenchmarkSearchAfterImport benchmarks search after importing documents.
func BenchmarkSearchAfterImport(b *testing.B) {
	driver, err := NewIPCDriver(DefaultConfig())
	if err != nil {
		b.Fatal(err)
	}
	defer driver.Close()

	texts := generateSyntheticTexts(10000)
	if _, err := ImportFromParquet(driver, texts); err != nil {
		b.Fatal(err)
	}
	if err := driver.Build(); err != nil {
		b.Fatal(err)
	}

	queries := []string{
		"Việt Nam",
		"kinh tế",
		"du lịch",
		"công nghệ",
		"giáo dục",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := queries[i%len(queries)]
		_, err := driver.Search(q, 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestImportFromParquetIterError verifies error propagation from iterator.
func TestImportFromParquetIterError(t *testing.T) {
	driver, err := NewIPCDriver(DefaultConfig())
	if err != nil {
		t.Fatalf("create driver: %v", err)
	}
	defer driver.Close()

	expectedErr := fmt.Errorf("parquet read error")
	batches := func(yield func([]string, error) bool) {
		if !yield([]string{"doc1", "doc2"}, nil) {
			return
		}
		yield(nil, expectedErr)
	}

	n, err := ImportFromParquetIter(driver, batches)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "parquet read error") {
		t.Errorf("expected parquet error, got: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 docs before error, got %d", n)
	}
}
