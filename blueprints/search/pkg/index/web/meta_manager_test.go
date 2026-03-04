package web

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestMetaManager_ScanFallback(t *testing.T) {
	root := t.TempDir()
	warcDir := filepath.Join(root, "warc")
	mustMkdir(t, warcDir)
	writeFile(t, filepath.Join(warcDir, "00000.warc.gz"), 1024)

	m, err := NewMetaManager(context.Background(), MetaConfig{
		Driver:      "none",
		ActiveCrawl: "CC-TEST-2026",
		ActiveDir:   root,
		CommonCrawl: filepath.Dir(root),
	})
	if err != nil {
		t.Fatalf("NewMetaManager: %v", err)
	}

	got := m.GetSummary(context.Background(), "CC-TEST-2026", root)
	if got.MetaBackend != "scan-fallback" {
		t.Fatalf("meta_backend = %q, want scan-fallback", got.MetaBackend)
	}
	if got.WARCCount != 1 {
		t.Fatalf("warc_count = %d, want 1", got.WARCCount)
	}
}

func TestMetaManager_SQLiteCacheAndRefresh(t *testing.T) {
	root := t.TempDir()
	warcDir := filepath.Join(root, "warc")
	mustMkdir(t, warcDir)
	writeFile(t, filepath.Join(warcDir, "00000.warc.gz"), 1024)

	m, err := NewMetaManager(context.Background(), MetaConfig{
		Driver:      "sqlite",
		DSN:         filepath.Join(t.TempDir(), "meta.sqlite"),
		RefreshTTL:  time.Hour,
		Prewarm:     false,
		ActiveCrawl: "CC-TEST-2026",
		ActiveDir:   root,
		CommonCrawl: filepath.Dir(root),
		BusyTimeout: 3 * time.Second,
		JournalMode: "WAL",
	})
	if err != nil {
		t.Fatalf("NewMetaManager: %v", err)
	}
	defer m.Close()

	// First read: cache miss -> sync refresh.
	first := m.GetSummary(context.Background(), "CC-TEST-2026", root)
	if first.MetaBackend != "sqlite" {
		t.Fatalf("meta_backend = %q, want sqlite", first.MetaBackend)
	}
	if first.WARCCount != 1 {
		t.Fatalf("first warc_count = %d, want 1", first.WARCCount)
	}
	if first.MetaGeneratedAt == "" {
		t.Fatal("expected meta_generated_at on cached response")
	}
	warcs, _, err := m.ListWARCs(context.Background(), "CC-TEST-2026", root)
	if err != nil {
		t.Fatalf("ListWARCs: %v", err)
	}
	if len(warcs) == 0 {
		t.Fatal("expected at least one warc record")
	}

	// Change FS data after cache snapshot.
	writeFile(t, filepath.Join(warcDir, "00001.warc.gz"), 1024)

	// TTL is long: should still serve cached count.
	second := m.GetSummary(context.Background(), "CC-TEST-2026", root)
	if second.WARCCount != 1 {
		t.Fatalf("second warc_count = %d, want cached value 1", second.WARCCount)
	}

	accepted := m.TriggerRefresh("CC-TEST-2026", root, true)
	if !accepted {
		t.Fatal("TriggerRefresh returned false, want true")
	}
	waitForSummaryWARCCount(t, m, "CC-TEST-2026", root, 2)

	third := m.GetSummary(context.Background(), "CC-TEST-2026", root)
	if third.WARCCount != 2 {
		t.Fatalf("third warc_count = %d, want 2 after refresh", third.WARCCount)
	}
}

func waitForSummaryWARCCount(t *testing.T, m *MetaManager, crawlID, crawlDir string, want int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got := m.GetSummary(context.Background(), crawlID, crawlDir)
		if got.WARCCount == want {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("summary warc_count did not reach %d before timeout", want)
}

func TestMetaManager_IsStale_ScanAwareThreshold(t *testing.T) {
	m := &MetaManager{refreshTTL: 30 * time.Second}
	now := time.Now()

	// Without scan-awareness, 40s old data is stale under a 30s TTL.
	if !m.isStale(now.Add(-40*time.Second), 0) {
		t.Fatal("expected stale without scan duration")
	}

	// With a 25s scan duration, stale threshold becomes 55s.
	if m.isStale(now.Add(-40*time.Second), 25*time.Second) {
		t.Fatal("expected fresh when scan-aware threshold is not exceeded")
	}

	if !m.isStale(now.Add(-70*time.Second), 25*time.Second) {
		t.Fatal("expected stale when scan-aware threshold is exceeded")
	}
}
