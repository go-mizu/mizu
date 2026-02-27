package crawl

import (
	"context"
	"net"
	"net/url"
	"runtime"
	"sync"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/archived/recrawler"
)

// staticDNSCache wraps recrawler.DNSResolver for the DNSCache interface.
// It snapshots the resolved IP map at construction time for O(1) lookups.
type staticDNSCache struct {
	resolved map[string][]string // host → IPs snapshot
	r        *recrawler.DNSResolver
}

func (s *staticDNSCache) Lookup(host string) (string, bool) {
	ips, ok := s.resolved[host]
	if !ok || len(ips) == 0 {
		return "", false
	}
	return ips[0], true
}

func (s *staticDNSCache) IsDead(host string) bool {
	return s.r.IsDead(host)
}

// WrapDNSResolver adapts a *recrawler.DNSResolver to DNSCache.
// It snapshots resolved IPs at call time; call again after resolution completes
// to get an up-to-date view.
func WrapDNSResolver(r *recrawler.DNSResolver) DNSCache {
	return &staticDNSCache{
		resolved: r.ResolvedIPs(),
		r:        r,
	}
}

// NoopDNS implements DNSCache with no pre-resolved IPs.
type NoopDNS struct{}

func (n *NoopDNS) Lookup(_ string) (string, bool) { return "", false }
func (n *NoopDNS) IsDead(_ string) bool           { return false }

// ResultDBWriter adapts recrawler.ResultDB to ResultWriter.
type ResultDBWriter struct{ DB *recrawler.ResultDB }

func (r *ResultDBWriter) Add(result recrawler.Result)     { r.DB.Add(result) }
func (r *ResultDBWriter) Flush(ctx context.Context) error { return r.DB.Flush(ctx) }
func (r *ResultDBWriter) Close() error                    { return r.DB.Close() }

// FailedDBWriter adapts recrawler.FailedDB to FailureWriter.
type FailedDBWriter struct{ DB *recrawler.FailedDB }

func (f *FailedDBWriter) AddURL(u recrawler.FailedURL) { f.DB.AddURL(u) }
func (f *FailedDBWriter) Close() error                 { return f.DB.Close() }

// rssNow returns current process RSS memory in bytes (approximated via runtime.MemStats).
func rssNow() int64 {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return int64(ms.Sys)
}

// dialWithIP returns a DialContext that connects to a pre-resolved IP
// but preserves the original hostname for TLS SNI / HTTP Host header.
func dialWithIP(ip string) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		_, port, err := net.SplitHostPort(addr)
		if err != nil {
			port = "443"
		}
		d := &net.Dialer{Timeout: 5 * time.Second}
		return d.DialContext(ctx, "tcp", net.JoinHostPort(ip, port))
	}
}

// parsedURL holds the components of a parsed URL needed for dialing.
type parsedURL struct {
	Scheme string
	Host   string // hostname only (no port)
	Port   string
	Path   string
	RawURL string
}

// parseRawURL parses a raw URL string into components for dialing.
func parseRawURL(rawURL string) (*parsedURL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	path := u.RequestURI()
	return &parsedURL{
		Scheme: u.Scheme,
		Host:   host,
		Port:   port,
		Path:   path,
		RawURL: rawURL,
	}, nil
}

// peakTracker computes peak RPS over a sliding 1-second window.
type peakTracker struct {
	mu   sync.Mutex
	last time.Time
	cur  int64
	peak float64
}

func (p *peakTracker) Record() {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	if p.last.IsZero() {
		p.last = now
	}
	p.cur++
	if now.Sub(p.last) >= time.Second {
		rps := float64(p.cur) / now.Sub(p.last).Seconds()
		if rps > p.peak {
			p.peak = rps
		}
		p.cur = 0
		p.last = now
	}
}

func (p *peakTracker) Peak() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.peak
}
