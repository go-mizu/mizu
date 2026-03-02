package store

import (
	"testing"
	"time"
)

func TestDNSCacheSaveLoad_RoundTripDeadTimeout(t *testing.T) {
	d := NewDNSResolver(500 * time.Millisecond)

	// Seed caches directly (same package test).
	s := &d.shards[shardFor("example.com")]
	s.mu.Lock()
	s.resolved["example.com"] = []string{"93.184.216.34"}
	s.mu.Unlock()
	d.ok.Add(1)

	s = &d.shards[shardFor("tramitarlacurp.blogspot.com")]
	s.mu.Lock()
	s.dead["tramitarlacurp.blogspot.com"] = "lookup tramitarlacurp.blogspot.com: no such host"
	s.mu.Unlock()
	d.failed.Add(1)

	s = &d.shards[shardFor("slow.example")]
	s.mu.Lock()
	s.timeout["slow.example"] = "i/o timeout"
	s.mu.Unlock()
	d.timedOut.Add(1)

	// Malformed/dirty inputs should not break TSV bulk import.
	s = &d.shards[shardFor("bad\thost.example")]
	s.mu.Lock()
	s.resolved["bad\thost.example"] = []string{"1.2.3.4", "5.6.7.8"}
	s.dead["tabs.example"] = "lookup tabs.example:\tno such host"
	s.timeout["newline.example"] = "temporary failure\nretry later"
	s.mu.Unlock()
	d.ok.Add(1)
	d.failed.Add(1)
	d.timedOut.Add(1)

	dbPath := t.TempDir() + "/dns.duckdb"
	if err := d.SaveCache(dbPath); err != nil {
		t.Fatalf("SaveCache: %v", err)
	}

	loaded := NewDNSResolver(500 * time.Millisecond)
	n, err := loaded.LoadCache(dbPath)
	if err != nil {
		t.Fatalf("LoadCache: %v", err)
	}
	if n != 6 {
		t.Fatalf("loaded entries=%d want 6", n)
	}
	if loaded.LiveCount() != 2 {
		t.Fatalf("LiveCount=%d want 2", loaded.LiveCount())
	}
	if loaded.DeadCount() != 2 {
		t.Fatalf("DeadCount=%d want 2", loaded.DeadCount())
	}
	if loaded.TimeoutCount() != 2 {
		t.Fatalf("TimeoutCount=%d want 2", loaded.TimeoutCount())
	}
	if !loaded.IsDead("tramitarlacurp.blogspot.com") {
		t.Fatalf("expected dead domain loaded")
	}
	if !loaded.IsTimeout("slow.example") {
		t.Fatalf("expected timeout domain loaded")
	}
	if !loaded.IsDead("tabs.example") {
		t.Fatalf("expected tab error dead domain loaded")
	}
	if !loaded.IsTimeout("newline.example") {
		t.Fatalf("expected newline error timeout domain loaded")
	}
}
