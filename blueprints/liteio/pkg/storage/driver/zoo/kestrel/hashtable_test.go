package kestrel

import (
	"fmt"
	"sync"
	"testing"
)

func TestHTableBasic(t *testing.T) {
	ht := newHTable(0)

	// Empty table
	if _, ok := ht.get(htHash64("b", "k1"), "b\x00k1"); ok {
		t.Fatal("expected not found on empty table")
	}

	// Put and get
	r1 := &hotRecord{size: 100}
	h := htHash64("b", "k1")
	old, replaced := ht.put(h, "b\x00k1", r1)
	if replaced || old != nil {
		t.Fatal("expected no replacement on first put")
	}

	got, ok := ht.get(h, "b\x00k1")
	if !ok || got != r1 {
		t.Fatal("expected to find k1")
	}

	// Update existing
	r2 := &hotRecord{size: 200}
	old, replaced = ht.put(h, "b\x00k1", r2)
	if !replaced || old != r1 {
		t.Fatal("expected replacement with old value")
	}

	got, ok = ht.get(h, "b\x00k1")
	if !ok || got != r2 {
		t.Fatal("expected updated value")
	}
}

func TestHTableUpdate(t *testing.T) {
	ht := newHTable(0)

	r1 := &hotRecord{size: 100}
	h := htHash64("b", "k1")
	ht.put(h, "b\x00k1", r1)

	// Update via update method
	r2 := &hotRecord{size: 200}
	old, ok := ht.update(h, "b\x00k1", r2)
	if !ok || old != r1 {
		t.Fatal("expected update to succeed")
	}

	got, ok := ht.get(h, "b\x00k1")
	if !ok || got != r2 {
		t.Fatal("expected updated value")
	}

	// Update non-existent
	h2 := htHash64("b", "k2")
	_, ok = ht.update(h2, "b\x00k2", r2)
	if ok {
		t.Fatal("expected update to fail for non-existent key")
	}
}

func TestHTableRemove(t *testing.T) {
	ht := newHTable(0)

	r1 := &hotRecord{size: 1}
	r2 := &hotRecord{size: 2}
	r3 := &hotRecord{size: 3}

	h1 := htHash64("b", "a")
	h2 := htHash64("b", "b")
	h3 := htHash64("b", "c")

	ht.put(h1, "b\x00a", r1)
	ht.put(h2, "b\x00b", r2)
	ht.put(h3, "b\x00c", r3)

	// Remove middle
	old, ok := ht.remove(h2, "b\x00b")
	if !ok || old != r2 {
		t.Fatal("expected to remove b")
	}

	// Verify b is gone
	if _, ok := ht.get(h2, "b\x00b"); ok {
		t.Fatal("b should be removed")
	}

	// Verify a and c still present
	if got, ok := ht.get(h1, "b\x00a"); !ok || got != r1 {
		t.Fatal("a should still exist")
	}
	if got, ok := ht.get(h3, "b\x00c"); !ok || got != r3 {
		t.Fatal("c should still exist")
	}

	// Remove rest
	ht.remove(h1, "b\x00a")
	ht.remove(h3, "b\x00c")
	if ht.count != 0 {
		t.Fatalf("expected count 0, got %d", ht.count)
	}
}

func TestHTableGrow(t *testing.T) {
	ht := newHTable(0)
	entries := make(map[string]*hotRecord)

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("bkt\x00key_%04d", i)
		r := &hotRecord{size: int64(i)}
		h := htHash64("bkt", fmt.Sprintf("key_%04d", i))
		ht.put(h, key, r)
		entries[key] = r
	}

	if ht.count != 1000 {
		t.Fatalf("expected count 1000, got %d", ht.count)
	}

	// Verify all entries
	for key, expected := range entries {
		// Re-extract bucket and key parts for hashing
		var bkt, k string
		for j := 0; j < len(key); j++ {
			if key[j] == 0 {
				bkt = key[:j]
				k = key[j+1:]
				break
			}
		}
		h := htHash64(bkt, k)
		got, ok := ht.get(h, key)
		if !ok {
			t.Fatalf("missing key %q after grow", key)
		}
		if got != expected {
			t.Fatalf("wrong value for key %q", key)
		}
	}
}

func TestHTableCollisions(t *testing.T) {
	ht := newHTable(8)
	n := 100

	type kv struct {
		hash uint64
		key  string
		rec  *hotRecord
	}
	var kvs []kv

	for i := 0; i < n; i++ {
		k := fmt.Sprintf("b\x00%d", i)
		h := htHash64("b", fmt.Sprintf("%d", i))
		r := &hotRecord{size: int64(i)}
		ht.put(h, k, r)
		kvs = append(kvs, kv{h, k, r})
	}

	// Remove even entries
	for i := 0; i < n; i += 2 {
		ht.remove(kvs[i].hash, kvs[i].key)
	}

	if ht.count != n/2 {
		t.Fatalf("expected count %d, got %d", n/2, ht.count)
	}

	// Verify odd entries still present
	for i := 1; i < n; i += 2 {
		got, ok := ht.get(kvs[i].hash, kvs[i].key)
		if !ok {
			t.Fatalf("missing key %d after removing evens", i)
		}
		if got.size != int64(i) {
			t.Fatalf("wrong size for key %d: got %d", i, got.size)
		}
	}
}

func TestHTableUnsafeStringLookup(t *testing.T) {
	ht := newHTable(0)

	// Insert with heap key
	r1 := &hotRecord{size: 42}
	h := htHash64("mybucket", "mykey")
	heapKey := compositeKey("mybucket", "mykey")
	ht.put(h, heapKey, r1)

	// Lookup with stack-backed key (simulating unsafeString)
	var buf [256]byte
	stackKey := unsafeString(compositeKeyBuf(buf[:0], "mybucket", "mykey"))

	got, ok := ht.get(h, stackKey)
	if !ok || got != r1 {
		t.Fatal("expected to find entry using stack-backed key")
	}

	// Update with stack-backed key
	r2 := &hotRecord{size: 99}
	old, ok := ht.update(h, stackKey, r2)
	if !ok || old != r1 {
		t.Fatal("expected update with stack key to succeed")
	}

	got, ok = ht.get(h, heapKey)
	if !ok || got != r2 {
		t.Fatal("expected updated value")
	}
}

func BenchmarkHTableGet(b *testing.B) {
	ht := newHTable(10000)
	keys := make([]string, 10000)
	hashes := make([]uint64, 10000)
	for i := 0; i < 10000; i++ {
		k := fmt.Sprintf("key_%06d", i)
		keys[i] = "bkt\x00" + k
		hashes[i] = htHash64("bkt", k)
		ht.put(hashes[i], keys[i], &hotRecord{size: int64(i)})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := i % 10000
		ht.get(hashes[idx], keys[idx])
	}
}

func BenchmarkGoMapGet(b *testing.B) {
	m := make(map[string]*hotRecord, 10000)
	keys := make([]string, 10000)
	for i := 0; i < 10000; i++ {
		k := fmt.Sprintf("bkt\x00key_%06d", i)
		keys[i] = k
		m[k] = &hotRecord{size: int64(i)}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m[keys[i%10000]]
	}
}

func BenchmarkHTablePut(b *testing.B) {
	recs := make([]*hotRecord, 10000)
	keys := make([]string, 10000)
	hashes := make([]uint64, 10000)
	for i := range recs {
		k := fmt.Sprintf("key_%06d", i)
		recs[i] = &hotRecord{size: int64(i)}
		keys[i] = "bkt\x00" + k
		hashes[i] = htHash64("bkt", k)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ht := newHTable(10000)
		for j := 0; j < 10000; j++ {
			ht.put(hashes[j], keys[j], recs[j])
		}
	}
}

func BenchmarkGoMapPut(b *testing.B) {
	recs := make([]*hotRecord, 10000)
	keys := make([]string, 10000)
	for i := range recs {
		recs[i] = &hotRecord{size: int64(i)}
		keys[i] = fmt.Sprintf("bkt\x00key_%06d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := make(map[string]*hotRecord, 10000)
		for j := 0; j < 10000; j++ {
			m[keys[j]] = recs[j]
		}
	}
}

// BenchmarkConcurrentHTWrite measures htable write under C200 contention (1 shard).
func BenchmarkConcurrentHTWrite(b *testing.B) {
	var mu sync.RWMutex
	ht := newHTable(100000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("bkt\x00key_%d", i)
			h := htHash64("bkt", fmt.Sprintf("key_%d", i))
			rec := &hotRecord{size: 16384}
			mu.Lock()
			ht.put(h, key, rec)
			mu.Unlock()
			i++
		}
	})
}

// BenchmarkConcurrentGoMapWrite measures Go map write under C200 contention (1 shard).
func BenchmarkConcurrentGoMapWrite(b *testing.B) {
	var mu sync.RWMutex
	m := make(map[string]*hotRecord, 100000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("bkt\x00key_%d", i)
			rec := &hotRecord{size: 16384}
			mu.Lock()
			m[key] = rec
			mu.Unlock()
			i++
		}
	})
}
