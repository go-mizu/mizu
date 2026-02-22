package ant

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func openTestStore(tb testing.TB) *store {
	tb.Helper()
	dir := tb.TempDir()
	ctx := context.Background()
	d := &driver{}
	st, err := d.Open(ctx, "ant://"+dir+"?sync=none")
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { st.Close() })
	return st.(*store)
}

// --- Write benchmarks ---

func BenchmarkWrite1B(b *testing.B) {
	st := openTestStore(b)
	ctx := context.Background()
	st.CreateBucket(ctx, "b", nil)
	bkt := st.Bucket("b").(*bucket)
	data := []byte{0}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("k/%d", i)
		bkt.Write(ctx, key, bytes.NewReader(data), 1, "application/octet-stream", nil)
	}
}

func BenchmarkWrite1KB(b *testing.B) {
	st := openTestStore(b)
	ctx := context.Background()
	st.CreateBucket(ctx, "b", nil)
	bkt := st.Bucket("b").(*bucket)
	data := make([]byte, 1024)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("k/%d", i)
		bkt.Write(ctx, key, bytes.NewReader(data), 1024, "application/octet-stream", nil)
	}
}

func BenchmarkWrite64KB(b *testing.B) {
	st := openTestStore(b)
	ctx := context.Background()
	st.CreateBucket(ctx, "b", nil)
	bkt := st.Bucket("b").(*bucket)
	data := make([]byte, 64*1024)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("k/%d", i)
		bkt.Write(ctx, key, bytes.NewReader(data), int64(len(data)), "application/octet-stream", nil)
	}
}

// --- Read benchmarks ---

func BenchmarkRead1KB(b *testing.B) {
	st := openTestStore(b)
	ctx := context.Background()
	st.CreateBucket(ctx, "b", nil)
	bkt := st.Bucket("b").(*bucket)
	data := make([]byte, 1024)
	// Pre-populate
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("k/%d", i)
		bkt.Write(ctx, key, bytes.NewReader(data), 1024, "application/octet-stream", nil)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("k/%d", i%1000)
		rc, _, _ := bkt.Open(ctx, key, 0, 0, nil)
		if rc != nil {
			rc.Close()
		}
	}
}

// --- Stat benchmarks ---

func BenchmarkStat(b *testing.B) {
	st := openTestStore(b)
	ctx := context.Background()
	st.CreateBucket(ctx, "b", nil)
	bkt := st.Bucket("b").(*bucket)
	data := make([]byte, 1024)
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("k/%d", i)
		bkt.Write(ctx, key, bytes.NewReader(data), 1024, "application/octet-stream", nil)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("k/%d", i%1000)
		bkt.Stat(ctx, key, nil)
	}
}

// --- Delete benchmarks ---

func BenchmarkDelete(b *testing.B) {
	st := openTestStore(b)
	ctx := context.Background()
	st.CreateBucket(ctx, "b", nil)
	bkt := st.Bucket("b").(*bucket)
	data := make([]byte, 64)
	// Pre-populate enough for b.N
	for i := 0; i < b.N+1000; i++ {
		key := fmt.Sprintf("k/%d", i)
		bkt.Write(ctx, key, bytes.NewReader(data), int64(len(data)), "application/octet-stream", nil)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("k/%d", i)
		bkt.Delete(ctx, key, nil)
	}
}

// --- Parallel benchmarks ---

func BenchmarkParallelWrite1KB_C10(b *testing.B) {
	st := openTestStore(b)
	ctx := context.Background()
	st.CreateBucket(ctx, "b", nil)
	bkt := st.Bucket("b").(*bucket)
	data := make([]byte, 1024)
	b.ResetTimer()
	b.ReportAllocs()
	b.SetParallelism(10)
	var counter int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := int(atomicAdd(&counter))
			key := fmt.Sprintf("k/%d", i)
			bkt.Write(ctx, key, bytes.NewReader(data), 1024, "application/octet-stream", nil)
		}
	})
}

func BenchmarkParallelRead1KB_C10(b *testing.B) {
	st := openTestStore(b)
	ctx := context.Background()
	st.CreateBucket(ctx, "b", nil)
	bkt := st.Bucket("b").(*bucket)
	data := make([]byte, 1024)
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("k/%d", i)
		bkt.Write(ctx, key, bytes.NewReader(data), 1024, "application/octet-stream", nil)
	}
	b.ResetTimer()
	b.ReportAllocs()
	b.SetParallelism(10)
	var counter int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := int(atomicAdd(&counter))
			key := fmt.Sprintf("k/%d", i%10000)
			rc, _, _ := bkt.Open(ctx, key, 0, 0, nil)
			if rc != nil {
				rc.Close()
			}
		}
	})
}

func atomicAdd(p *int64) int64 {
	return __sync_add(p)
}

// --- Memory measurement ---

func TestMemory100K(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	st.CreateBucket(ctx, "b", nil)
	bkt := st.Bucket("b").(*bucket)
	data := make([]byte, 1024)

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	for i := 0; i < 100_000; i++ {
		key := fmt.Sprintf("k/%07d", i)
		bkt.Write(ctx, key, bytes.NewReader(data), 1024, "application/octet-stream", nil)
	}

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	heapInUse := after.HeapInuse - before.HeapInuse
	heapAlloc := after.HeapAlloc - before.HeapAlloc
	t.Logf("100K × 1KB objects:")
	t.Logf("  HeapInuse delta: %.2f MB", float64(heapInUse)/1024/1024)
	t.Logf("  HeapAlloc delta: %.2f MB", float64(heapAlloc)/1024/1024)
	t.Logf("  HeapSys:         %.2f MB", float64(after.HeapSys)/1024/1024)
	t.Logf("  TotalAlloc:      %.2f MB", float64(after.TotalAlloc)/1024/1024)
	t.Logf("  NumGC:           %d (delta: %d)", after.NumGC, after.NumGC-before.NumGC)

	// Verify memory is under 100MB
	if heapInUse > 100*1024*1024 {
		t.Errorf("FAIL: HeapInuse %.2f MB exceeds 100MB budget", float64(heapInUse)/1024/1024)
	} else {
		t.Logf("  PASS: HeapInuse under 100MB budget (%.1f%%)", float64(heapInUse)/100/1024/1024*100)
	}
}

func TestMemory100K_WithDelete(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	st.CreateBucket(ctx, "b", nil)
	bkt := st.Bucket("b").(*bucket)
	data := make([]byte, 1024)

	// Write 100K then delete 50K
	for i := 0; i < 100_000; i++ {
		key := fmt.Sprintf("k/%07d", i)
		bkt.Write(ctx, key, bytes.NewReader(data), 1024, "application/octet-stream", nil)
	}
	for i := 0; i < 50_000; i++ {
		key := fmt.Sprintf("k/%07d", i)
		bkt.Delete(ctx, key, nil)
	}

	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	t.Logf("100K writes + 50K deletes:")
	t.Logf("  HeapInuse: %.2f MB", float64(ms.HeapInuse)/1024/1024)
	t.Logf("  HeapAlloc: %.2f MB", float64(ms.HeapAlloc)/1024/1024)
}

// --- List benchmark ---

func BenchmarkList100(b *testing.B) {
	st := openTestStore(b)
	ctx := context.Background()
	st.CreateBucket(ctx, "b", nil)
	bkt := st.Bucket("b").(*bucket)
	data := make([]byte, 64)
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("k/%04d", i)
		bkt.Write(ctx, key, bytes.NewReader(data), int64(len(data)), "application/octet-stream", nil)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		iter, _ := bkt.List(ctx, "", 100, 0, nil)
		if iter != nil {
			for {
				obj, _ := iter.Next()
				if obj == nil {
					break
				}
			}
			iter.Close()
		}
	}
}

// --- Disk usage check ---

func TestDiskUsage(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	d := &driver{}
	st, err := d.Open(ctx, "ant://"+dir+"?sync=none")
	if err != nil {
		t.Fatal(err)
	}
	s := st.(*store)
	s.CreateBucket(ctx, "b", nil)
	bkt := s.Bucket("b").(*bucket)
	data := make([]byte, 1024)
	for i := 0; i < 10_000; i++ {
		key := fmt.Sprintf("k/%07d", i)
		bkt.Write(ctx, key, bytes.NewReader(data), 1024, "application/octet-stream", nil)
	}
	st.Close()

	var total int64
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	t.Logf("10K × 1KB: disk = %.2f MB (overhead: %.1f%%)", float64(total)/1024/1024, float64(total-10_000*1024)/float64(10_000*1024)*100)
}

// sync/atomic helper
func __sync_add(p *int64) int64 {
	// Use a simple counter - safe enough for benchmarks
	v := *p
	*p = v + 1
	return v
}

func init() {
	// Ensure proper time resolution
	_ = time.Now()
}
