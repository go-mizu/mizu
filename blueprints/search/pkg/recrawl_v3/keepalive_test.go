package recrawl_v3

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/recrawler"
)

func TestKeepAliveEngine_BasicCrawl(t *testing.T) {
	var reqCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount++
		w.WriteHeader(200)
	}))
	defer srv.Close()

	seeds := make([]recrawler.SeedURL, 20)
	for i := range seeds {
		seeds[i] = recrawler.SeedURL{
			URL:    srv.URL + "/page/" + string(rune('a'+i)),
			Domain: "localhost",
			Host:   "localhost",
		}
	}

	cfg := DefaultConfig()
	cfg.Workers = 4
	cfg.Timeout = 2 * time.Second
	cfg.InsecureTLS = false
	cfg.StatusOnly = true

	eng := &KeepAliveEngine{}
	stats, err := eng.Run(context.Background(), seeds, &NoopDNS{}, cfg,
		&noopResultWriter{}, &noopFailureWriter{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if stats.OK != 20 {
		t.Errorf("want 20 OK, got %d (failed=%d)", stats.OK, stats.Failed)
	}
	if stats.Total != 20 {
		t.Errorf("want Total=20, got %d", stats.Total)
	}
	if stats.Duration <= 0 {
		t.Error("Duration should be positive")
	}
}

func TestKeepAliveEngine_DeadDomainSkipped(t *testing.T) {
	seeds := []recrawler.SeedURL{
		{URL: "http://dead.example.com/page", Domain: "dead.example.com", Host: "dead.example.com"},
		{URL: "http://dead.example.com/page2", Domain: "dead.example.com", Host: "dead.example.com"},
	}

	cfg := DefaultConfig()
	cfg.Workers = 2
	cfg.Timeout = 1 * time.Second

	deadDNS := &mockDeadDNS{deadHost: "dead.example.com"}
	eng := &KeepAliveEngine{}
	fw := &countFailureWriter{}
	stats, err := eng.Run(context.Background(), seeds, deadDNS, cfg,
		&noopResultWriter{}, fw)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if stats.Total != 0 {
		t.Errorf("dead domain URLs should be skipped (not counted), got Total=%d", stats.Total)
	}
	if fw.count != 2 {
		t.Errorf("want 2 failures recorded for dead domain, got %d", fw.count)
	}
}

// ── test stubs ──────────────────────────────────────────────

type noopResultWriter struct{}

func (n *noopResultWriter) Add(_ recrawler.Result)        {}
func (n *noopResultWriter) Flush(_ context.Context) error { return nil }
func (n *noopResultWriter) Close() error                  { return nil }

type noopFailureWriter struct{}

func (n *noopFailureWriter) AddURL(_ recrawler.FailedURL) {}
func (n *noopFailureWriter) Close() error                 { return nil }

type countFailureWriter struct{ count int }

func (c *countFailureWriter) AddURL(_ recrawler.FailedURL) { c.count++ }
func (c *countFailureWriter) Close() error                 { return nil }

type mockDeadDNS struct{ deadHost string }

func (m *mockDeadDNS) Lookup(_ string) (string, bool) { return "", false }
func (m *mockDeadDNS) IsDead(host string) bool        { return host == m.deadHost }
