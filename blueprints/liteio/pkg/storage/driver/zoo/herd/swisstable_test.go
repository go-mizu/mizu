package herd

import (
	"fmt"
	"testing"
)

func TestSwissTableBasic(t *testing.T) {
	st := newSwissTable(0)

	// Empty table
	if _, ok := st.get("a"); ok {
		t.Fatal("expected not found on empty table")
	}
	if st.len() != 0 {
		t.Fatalf("expected len 0, got %d", st.len())
	}

	// Put and get
	e1 := &indexEntry{size: 100}
	old, replaced := st.put("key1", e1)
	if replaced || old != nil {
		t.Fatal("expected no replacement on first put")
	}
	if st.len() != 1 {
		t.Fatalf("expected len 1, got %d", st.len())
	}

	got, ok := st.get("key1")
	if !ok || got != e1 {
		t.Fatal("expected to find key1")
	}

	// Update existing key
	e2 := &indexEntry{size: 200}
	old, replaced = st.put("key1", e2)
	if !replaced || old != e1 {
		t.Fatal("expected replacement with old value")
	}
	if st.len() != 1 {
		t.Fatalf("expected len 1 after update, got %d", st.len())
	}

	got, ok = st.get("key1")
	if !ok || got != e2 {
		t.Fatal("expected to find updated key1")
	}
}

func TestSwissTableRemove(t *testing.T) {
	st := newSwissTable(0)

	// Remove from empty
	_, removed := st.remove("nope")
	if removed {
		t.Fatal("expected no removal from empty table")
	}

	e1 := &indexEntry{size: 1}
	e2 := &indexEntry{size: 2}
	e3 := &indexEntry{size: 3}

	st.put("a", e1)
	st.put("b", e2)
	st.put("c", e3)

	if st.len() != 3 {
		t.Fatalf("expected len 3, got %d", st.len())
	}

	// Remove middle
	old, ok := st.remove("b")
	if !ok || old != e2 {
		t.Fatal("expected to remove b")
	}
	if st.len() != 2 {
		t.Fatalf("expected len 2, got %d", st.len())
	}

	// Verify b is gone
	if _, ok := st.get("b"); ok {
		t.Fatal("b should be removed")
	}

	// Verify a and c still present
	if got, ok := st.get("a"); !ok || got != e1 {
		t.Fatal("a should still exist")
	}
	if got, ok := st.get("c"); !ok || got != e3 {
		t.Fatal("c should still exist")
	}

	// Remove a and c
	st.remove("a")
	st.remove("c")
	if st.len() != 0 {
		t.Fatalf("expected len 0, got %d", st.len())
	}
}

func TestSwissTableGrow(t *testing.T) {
	st := newSwissTable(0)
	entries := make(map[string]*indexEntry)

	// Insert enough entries to trigger multiple grows.
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%04d", i)
		e := &indexEntry{size: int64(i)}
		st.put(key, e)
		entries[key] = e
	}

	if st.len() != 1000 {
		t.Fatalf("expected len 1000, got %d", st.len())
	}

	// Verify all entries are retrievable.
	for key, expected := range entries {
		got, ok := st.get(key)
		if !ok {
			t.Fatalf("missing key %q after grow", key)
		}
		if got != expected {
			t.Fatalf("wrong value for key %q", key)
		}
	}
}

func TestSwissTableForEach(t *testing.T) {
	st := newSwissTable(0)
	for i := 0; i < 50; i++ {
		st.put(fmt.Sprintf("k%d", i), &indexEntry{size: int64(i)})
	}

	count := 0
	st.forEach(func(key string, value *indexEntry) {
		count++
	})
	if count != 50 {
		t.Fatalf("forEach visited %d entries, expected 50", count)
	}
}

func TestSwissTableCollisions(t *testing.T) {
	// Test with keys that may hash to the same slots.
	st := newSwissTable(8)
	n := 100
	for i := 0; i < n; i++ {
		st.put(fmt.Sprintf("%d", i), &indexEntry{size: int64(i)})
	}

	// Remove half and verify the other half is intact.
	for i := 0; i < n; i += 2 {
		st.remove(fmt.Sprintf("%d", i))
	}

	if st.len() != n/2 {
		t.Fatalf("expected len %d, got %d", n/2, st.len())
	}

	for i := 1; i < n; i += 2 {
		got, ok := st.get(fmt.Sprintf("%d", i))
		if !ok {
			t.Fatalf("missing key %d after removing evens", i)
		}
		if got.size != int64(i) {
			t.Fatalf("wrong size for key %d: got %d", i, got.size)
		}
	}
}

func TestSwissTableRemoveNonExistent(t *testing.T) {
	st := newSwissTable(0)
	st.put("exists", &indexEntry{size: 1})

	_, ok := st.remove("does_not_exist")
	if ok {
		t.Fatal("expected no removal for non-existent key")
	}
	if st.len() != 1 {
		t.Fatalf("expected len 1, got %d", st.len())
	}
}

func BenchmarkSwissTableGet(b *testing.B) {
	st := newSwissTable(10000)
	for i := 0; i < 10000; i++ {
		st.put(fmt.Sprintf("key_%06d", i), &indexEntry{size: int64(i)})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		st.get(fmt.Sprintf("key_%06d", i%10000))
	}
}

func BenchmarkSwissTablePut(b *testing.B) {
	entries := make([]*indexEntry, 10000)
	for i := range entries {
		entries[i] = &indexEntry{size: int64(i)}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		st := newSwissTable(10000)
		for j := 0; j < 10000; j++ {
			st.put(fmt.Sprintf("key_%06d", j), entries[j])
		}
	}
}

func BenchmarkGoMapGet(b *testing.B) {
	m := make(map[string]*indexEntry, 10000)
	for i := 0; i < 10000; i++ {
		m[fmt.Sprintf("key_%06d", i)] = &indexEntry{size: int64(i)}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m[fmt.Sprintf("key_%06d", i%10000)]
	}
}
