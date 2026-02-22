package kestrel

import (
	"fmt"
	"sync"
	"testing"
)

// testArena creates a dataArena for use in tests.
func testArena() *dataArena { return newDataArena() }

// writeTestEntry writes [key][ct][value] to the arena and returns an htEntry.
func writeTestEntry(arena *dataArena, bucket, key string, size int64) htEntry {
	ck := compositeKey(bucket, key)
	ct := "test/plain"
	value := make([]byte, size)
	total := len(ck) + len(ct) + len(value)
	off, buf := arena.alloc(total)
	n := copy(buf, ck)
	copy(buf[n:], ct)
	copy(buf[n+len(ct):], value)
	return htEntry{
		hash:     htHash64(bucket, key),
		arenaOff: off,
		keyLen:   uint16(len(ck)),
		ctLen:    uint16(len(ct)),
		valueLen: uint32(len(value)),
		size:     size,
		created:  1000,
		updated:  1000,
	}
}

func TestHTableBasic(t *testing.T) {
	arena := testArena()
	defer arena.close()
	ht := newHTable(0)

	// Empty table
	if _, ok := ht.get(htHash64("b", "k1")); ok {
		t.Fatal("expected not found on empty table")
	}

	// Put and get via putOrUpdate
	e1 := writeTestEntry(arena, "b", "k1", 100)
	_, isUpdate := ht.putOrUpdate(e1.hash, e1)
	if isUpdate {
		t.Fatal("expected insert, not update")
	}

	got, ok := ht.get(e1.hash)
	if !ok || got.size != 100 {
		t.Fatal("expected to find k1 with size 100")
	}

	// Update existing
	e2 := writeTestEntry(arena, "b", "k1", 200)
	old, isUpdate := ht.putOrUpdate(e2.hash, e2)
	if !isUpdate || old.size != 100 {
		t.Fatal("expected update with old size 100")
	}

	got, ok = ht.get(e2.hash)
	if !ok || got.size != 200 {
		t.Fatal("expected updated size 200")
	}
}

func TestHTableRemove(t *testing.T) {
	arena := testArena()
	defer arena.close()
	ht := newHTable(0)

	e1 := writeTestEntry(arena, "b", "a", 1)
	e2 := writeTestEntry(arena, "b", "b", 2)
	e3 := writeTestEntry(arena, "b", "c", 3)

	ht.putOrUpdate(e1.hash, e1)
	ht.putOrUpdate(e2.hash, e2)
	ht.putOrUpdate(e3.hash, e3)

	// Remove middle
	old, ok := ht.remove(e2.hash)
	if !ok || old.size != 2 {
		t.Fatal("expected to remove b with size 2")
	}

	// Verify b is gone
	if _, ok := ht.get(e2.hash); ok {
		t.Fatal("b should be removed")
	}

	// Verify a and c still present
	if got, ok := ht.get(e1.hash); !ok || got.size != 1 {
		t.Fatal("a should still exist with size 1")
	}
	if got, ok := ht.get(e3.hash); !ok || got.size != 3 {
		t.Fatal("c should still exist with size 3")
	}

	// Remove rest
	ht.remove(e1.hash)
	ht.remove(e3.hash)
	if ht.count != 0 {
		t.Fatalf("expected count 0, got %d", ht.count)
	}
}

func TestHTableGrow(t *testing.T) {
	arena := testArena()
	defer arena.close()
	ht := newHTable(0)

	type kv struct {
		entry htEntry
	}
	var entries []kv

	for i := 0; i < 1000; i++ {
		k := fmt.Sprintf("key_%04d", i)
		e := writeTestEntry(arena, "bkt", k, int64(i))
		ht.putOrUpdate(e.hash, e)
		entries = append(entries, kv{entry: e})
	}

	if ht.count != 1000 {
		t.Fatalf("expected count 1000, got %d", ht.count)
	}

	// Verify all entries
	for _, kv := range entries {
		got, ok := ht.get(kv.entry.hash)
		if !ok {
			t.Fatalf("missing entry after grow")
		}
		if got.size != kv.entry.size {
			t.Fatalf("wrong size: got %d, want %d", got.size, kv.entry.size)
		}
	}
}

func TestHTableCollisions(t *testing.T) {
	arena := testArena()
	defer arena.close()
	ht := newHTable(8)
	n := 100

	type kv struct {
		entry htEntry
	}
	var kvs []kv

	for i := 0; i < n; i++ {
		k := fmt.Sprintf("%d", i)
		e := writeTestEntry(arena, "b", k, int64(i))
		ht.putOrUpdate(e.hash, e)
		kvs = append(kvs, kv{entry: e})
	}

	// Remove even entries
	for i := 0; i < n; i += 2 {
		ht.remove(kvs[i].entry.hash)
	}

	if ht.count != n/2 {
		t.Fatalf("expected count %d, got %d", n/2, ht.count)
	}

	// Verify odd entries still present
	for i := 1; i < n; i += 2 {
		got, ok := ht.get(kvs[i].entry.hash)
		if !ok {
			t.Fatalf("missing key %d after removing evens", i)
		}
		if got.size != int64(i) {
			t.Fatalf("wrong size for key %d: got %d", i, got.size)
		}
	}
}

func TestHTableUnsafeStringLookup(t *testing.T) {
	arena := testArena()
	defer arena.close()
	ht := newHTable(0)

	// Insert
	e := writeTestEntry(arena, "mybucket", "mykey", 42)
	ht.putOrUpdate(e.hash, e)

	// Lookup by hash
	got, ok := ht.get(e.hash)
	if !ok || got.size != 42 {
		t.Fatal("expected to find entry using hash-only lookup")
	}

	// Update
	e2 := writeTestEntry(arena, "mybucket", "mykey", 99)
	old, isUpdate := ht.putOrUpdate(e2.hash, e2)
	if !isUpdate || old.size != 42 {
		t.Fatal("expected update to succeed")
	}

	got, ok = ht.get(e.hash)
	if !ok || got.size != 99 {
		t.Fatal("expected updated value")
	}
}

func BenchmarkHTableGet(b *testing.B) {
	arena := testArena()
	defer arena.close()
	ht := newHTable(10000)

	hashes := make([]uint64, 10000)
	for i := 0; i < 10000; i++ {
		k := fmt.Sprintf("key_%06d", i)
		e := writeTestEntry(arena, "bkt", k, int64(i))
		hashes[i] = e.hash
		ht.putOrUpdate(e.hash, e)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ht.get(hashes[i%10000])
	}
}

func BenchmarkHTablePut(b *testing.B) {
	arena := testArena()
	defer arena.close()

	entries := make([]htEntry, 10000)
	for i := range entries {
		k := fmt.Sprintf("key_%06d", i)
		entries[i] = writeTestEntry(arena, "bkt", k, int64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ht := newHTable(10000)
		for j := 0; j < 10000; j++ {
			ht.putOrUpdate(entries[j].hash, entries[j])
		}
	}
}

// BenchmarkConcurrentHTWrite measures htable write under contention (1 shard).
func BenchmarkConcurrentHTWrite(b *testing.B) {
	arena := testArena()
	defer arena.close()
	var mu sync.RWMutex
	ht := newHTable(100000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			k := fmt.Sprintf("key_%d", i)
			e := writeTestEntry(arena, "bkt", k, 16384)
			mu.Lock()
			ht.putOrUpdate(e.hash, e)
			mu.Unlock()
			i++
		}
	})
}
