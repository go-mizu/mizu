package crawl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

)

// TestDNSFrameRoundTrip verifies encode→decode preserves resolved IPs and dead hosts.
func TestDNSFrameRoundTrip(t *testing.T) {
	frame := dnsFrame{
		Resolved: map[string]string{
			"example.com": "93.184.216.34",
			"golang.org":  "142.250.1.1",
		},
		Dead: map[string]bool{
			"nxdomain.example": true,
		},
	}

	var buf bytes.Buffer
	if err := writeDNSFrame(&buf, frame); err != nil {
		t.Fatalf("writeDNSFrame: %v", err)
	}

	seeds := []SeedURL{
		{URL: "http://example.com/", Domain: "example.com"},
	}

	// Append seed JSON to buf (simulates queen writing to stdin pipe)
	enc := json.NewEncoder(&buf)
	for _, s := range seeds {
		enc.Encode(s) //nolint:errcheck
	}

	got, gotSeeds, err := readDroneInput(&buf)
	if err != nil {
		t.Fatalf("readDroneInput: %v", err)
	}

	if got.Resolved["example.com"] != "93.184.216.34" {
		t.Errorf("resolved mismatch: got %v", got.Resolved)
	}
	if !got.Dead["nxdomain.example"] {
		t.Error("dead host not preserved")
	}
	if len(gotSeeds) != 1 || gotSeeds[0].URL != "http://example.com/" {
		t.Errorf("seeds mismatch: got %v", gotSeeds)
	}
}

// TestReadDroneInput_EmptySeeds verifies graceful handling of frame with no seeds.
func TestReadDroneInput_EmptySeeds(t *testing.T) {
	frame := dnsFrame{
		Resolved: map[string]string{"a.com": "1.2.3.4"},
		Dead:     map[string]bool{},
	}
	var buf bytes.Buffer
	writeDNSFrame(&buf, frame) //nolint:errcheck

	_, seeds, err := readDroneInput(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(seeds) != 0 {
		t.Errorf("want 0 seeds, got %d", len(seeds))
	}
}

// TestParseRawFetch_HTMLPage verifies crawler.Extract is called for 200 HTML responses.
func TestParseRawFetch_HTMLPage(t *testing.T) {
	html := `<html><head><title>Hello World</title></head><body>content</body></html>`
	rf := rawFetch{
		seed:        SeedURL{URL: "http://example.com/", Domain: "example.com"},
		statusCode:  200,
		bodyBytes:   []byte(html),
		contentType: "text/html; charset=utf-8",
		contentLen:  int64(len(html)),
		fetchMs:     50,
	}
	r := parseRawFetch(rf)
	if r.StatusCode != 200 {
		t.Errorf("status: want 200, got %d", r.StatusCode)
	}
	if r.Title != "Hello World" {
		t.Errorf("title: want 'Hello World', got %q", r.Title)
	}
	if r.Body == "" {
		t.Error("body should be set for 200 HTML")
	}
}

// TestParseRawFetch_ErrorResult verifies error rawFetches produce Results with Error set.
func TestParseRawFetch_ErrorResult(t *testing.T) {
	rf := rawFetch{
		seed:    SeedURL{URL: "http://example.com/fail", Domain: "example.com"},
		errStr:  "connection refused",
		fetchMs: 5,
	}
	r := parseRawFetch(rf)
	if r.Error == "" {
		t.Error("want non-empty Error")
	}
	if r.Body != "" {
		t.Error("body should be empty for error result")
	}
}

// TestBuildDNSFrame verifies only seeds' hosts are included.
func TestBuildDNSFrame(t *testing.T) {
	dns := &mockDNS{
		resolved: map[string]string{"live.com": "1.2.3.4"},
		dead:     map[string]bool{"dead.com": true},
	}
	seeds := []SeedURL{
		{URL: "http://live.com/a", Domain: "live.com", Host: "live.com"},
		{URL: "http://dead.com/b", Domain: "dead.com", Host: "dead.com"},
		{URL: "http://unknown.com/c", Domain: "unknown.com", Host: "unknown.com"},
	}
	frame := buildDNSFrame(seeds, dns)

	if frame.Resolved["live.com"] != "1.2.3.4" {
		t.Errorf("live.com not resolved: %v", frame.Resolved)
	}
	if !frame.Dead["dead.com"] {
		t.Error("dead.com not in Dead map")
	}
	if _, ok := frame.Resolved["unknown.com"]; ok {
		t.Error("unknown.com should not be in Resolved")
	}
}

type mockDNS struct {
	resolved map[string]string
	dead     map[string]bool
}

func (m *mockDNS) Lookup(host string) (string, bool) { ip, ok := m.resolved[host]; return ip, ok }
func (m *mockDNS) IsDead(host string) bool           { return m.dead[host] }

// TestKeepaliveFetchRaw_FullBody verifies body bytes are captured for 200 HTML.
func TestKeepaliveFetchRaw_FullBody(t *testing.T) {
	html := `<html><head><title>Test</title></head><body>` + strings.Repeat("x", 100) + `</body></html>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		fmt.Fprint(w, html)
	}))
	defer srv.Close()

	seed := SeedURL{URL: srv.URL + "/page", Domain: "localhost", Host: "localhost"}
	cfg := DefaultConfig()
	cfg.StatusOnly = false
	cfg.Timeout = 2 * time.Second
	cfg.InsecureTLS = false

	transport := &http.Transport{}
	client := &http.Client{Transport: transport, Timeout: cfg.Timeout}

	rf := keepaliveFetchRaw(context.Background(), client, seed, cfg)
	if rf.errStr != "" {
		t.Fatalf("unexpected error: %s", rf.errStr)
	}
	if rf.statusCode != 200 {
		t.Errorf("want 200, got %d", rf.statusCode)
	}
	if len(rf.bodyBytes) == 0 {
		t.Error("bodyBytes should be non-empty for 200 HTML")
	}
	if !strings.Contains(string(rf.bodyBytes), "Test") {
		t.Error("bodyBytes should contain HTML content")
	}
}

// TestKeepaliveFetchRaw_StatusOnly verifies no body bytes when StatusOnly=true.
func TestKeepaliveFetchRaw_StatusOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		fmt.Fprint(w, "<html><body>big body</body></html>")
	}))
	defer srv.Close()

	seed := SeedURL{URL: srv.URL + "/page", Domain: "localhost", Host: "localhost"}
	cfg := DefaultConfig()
	cfg.StatusOnly = true
	cfg.Timeout = 2 * time.Second
	cfg.InsecureTLS = false

	transport := &http.Transport{}
	client := &http.Client{Transport: transport, Timeout: cfg.Timeout}

	rf := keepaliveFetchRaw(context.Background(), client, seed, cfg)
	if rf.errStr != "" {
		t.Fatalf("unexpected error: %s", rf.errStr)
	}
	if rf.statusCode != 200 {
		t.Errorf("want 200, got %d", rf.statusCode)
	}
	if len(rf.bodyBytes) != 0 {
		t.Errorf("bodyBytes should be nil in status-only mode, got %d bytes", len(rf.bodyBytes))
	}
}
