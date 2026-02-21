package herd

import "sync"

// indexEntryFreelist is a mutex-protected freelist for recycling indexEntry allocations.
// Unlike sync.Pool, entries are NEVER drained by GC — they recycle indefinitely.
// This eliminates the 3.3 GB heap churn from sync.Pool re-allocation every GC cycle.
// Uses a simple mutex stack instead of lock-free CAS to avoid the ABA problem.
type indexEntryFreelist struct {
	mu   sync.Mutex
	head *indexEntry
}

// acquire pops an indexEntry from the freelist or allocates a new one.
// The returned entry is zero-initialized.
func (fl *indexEntryFreelist) acquire() *indexEntry {
	fl.mu.Lock()
	e := fl.head
	if e != nil {
		fl.head = e.flNext
		fl.mu.Unlock()
		e.flNext = nil
		e.valueOffset = 0
		e.size = 0
		e.contentType = ""
		e.created = 0
		e.updated = 0
		e.inline = nil
		return e
	}
	fl.mu.Unlock()
	return &indexEntry{}
}

// release pushes an indexEntry back onto the freelist for reuse.
func (fl *indexEntryFreelist) release(e *indexEntry) {
	if e == nil {
		return
	}
	e.inline = nil // help GC
	fl.mu.Lock()
	e.flNext = fl.head
	fl.head = e
	fl.mu.Unlock()
}
