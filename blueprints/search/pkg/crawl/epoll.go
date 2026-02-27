package crawl

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/archived/recrawler"
	"golang.org/x/sync/errgroup"
)

// EpollEngine uses a fixed goroutine pool (4×nCPU) where each goroutine
// handles requests sequentially using raw net.Conn + explicit SetDeadline.
// Bypasses net/http to eliminate per-connection goroutine overhead.
// Pre-resolved IPs (from DNSCache) eliminate DNS lookup latency.
type EpollEngine struct{}

func (e *EpollEngine) Run(ctx context.Context, seeds []recrawler.SeedURL,
	dns DNSCache, cfg Config, results ResultWriter, failures FailureWriter) (*Stats, error) {

	// Filter out dead-domain seeds upfront
	live := make([]recrawler.SeedURL, 0, len(seeds))
	for _, s := range seeds {
		host := s.Host
		if host == "" {
			host = s.Domain
		}
		if dns.IsDead(host) {
			failures.AddURL(recrawler.FailedURL{
				URL:    s.URL,
				Domain: s.Domain,
				Reason: "domain_dead",
			})
			continue
		}
		live = append(live, s)
	}

	numWorkers := 4 * runtime.NumCPU()
	if cfg.Workers > 0 && cfg.Workers < numWorkers {
		numWorkers = cfg.Workers
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	workCh := make(chan recrawler.SeedURL, min(len(live), 10000))
	go func() {
		for _, s := range live {
			workCh <- s
		}
		close(workCh)
	}()

	var ok, failed, timeout, total atomic.Int64
	start := time.Now()
	peak := &peakTracker{}

	g, gctx := errgroup.WithContext(ctx)
	for range numWorkers {
		g.Go(func() error {
			for seed := range workCh {
				if gctx.Err() != nil {
					return nil
				}
				r := epollFetch(gctx, seed, dns, cfg)
				total.Add(1)
				peak.Record()

				switch {
				case r.Error != "" && isTimeoutErr(r.Error):
					timeout.Add(1)
					failures.AddURL(recrawler.FailedURL{
						URL:         seed.URL,
						Domain:      seed.Domain,
						Reason:      "http_timeout",
						Error:       r.Error,
						FetchTimeMs: r.FetchTimeMs,
					})
				case r.Error != "":
					failed.Add(1)
					failures.AddURL(recrawler.FailedURL{
						URL:         seed.URL,
						Domain:      seed.Domain,
						Reason:      "http_error",
						Error:       r.Error,
						FetchTimeMs: r.FetchTimeMs,
					})
				default:
					ok.Add(1)
				}
				results.Add(r)
			}
			return nil
		})
	}
	_ = g.Wait()

	dur := time.Since(start)
	tot := total.Load()
	avgRPS := 0.0
	if dur.Seconds() > 0 {
		avgRPS = float64(tot) / dur.Seconds()
	}
	return &Stats{
		Total:    tot,
		OK:       ok.Load(),
		Failed:   failed.Load(),
		Timeout:  timeout.Load(),
		PeakRPS:  peak.Peak(),
		AvgRPS:   avgRPS,
		Duration: dur,
		MemRSS:   rssNow(),
	}, nil
}

// epollFetch dials a raw TCP connection, sends a minimal HTTP/1.1 GET,
// reads only the status line. Uses pre-resolved IPs when available.
func epollFetch(ctx context.Context, seed recrawler.SeedURL, dns DNSCache, cfg Config) recrawler.Result {
	start := time.Now()
	ms := func() int64 { return time.Since(start).Milliseconds() }

	pu, err := parseRawURL(seed.URL)
	if err != nil {
		return recrawler.Result{URL: seed.URL, Domain: seed.Domain,
			Error: "parse: " + err.Error(), FetchTimeMs: ms()}
	}

	// Prefer pre-resolved IP to skip DNS at dial time
	dialHost := pu.Host
	if ip, ok := dns.Lookup(pu.Host); ok {
		dialHost = ip
	}
	dialAddr := net.JoinHostPort(dialHost, pu.Port)

	deadline := time.Now().Add(cfg.Timeout)
	dialCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	conn, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", dialAddr)
	if err != nil {
		return recrawler.Result{URL: seed.URL, Domain: seed.Domain,
			Error: err.Error(), FetchTimeMs: ms()}
	}
	defer conn.Close()
	conn.SetDeadline(deadline) //nolint:errcheck

	var rwConn net.Conn = conn
	if pu.Scheme == "https" {
		tlsConn := tls.Client(conn, &tls.Config{
			InsecureSkipVerify: cfg.InsecureTLS, //nolint:gosec
			ServerName:         pu.Host,
		})
		if err := tlsConn.Handshake(); err != nil {
			return recrawler.Result{URL: seed.URL, Domain: seed.Domain,
				Error: "tls: " + err.Error(), FetchTimeMs: ms()}
		}
		rwConn = tlsConn
	}

	// Minimal HTTP/1.1 request — no allocations beyond the string
	reqLine := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: %s\r\nConnection: close\r\n\r\n",
		pu.Path, pu.Host, cfg.UserAgent)
	if _, err := rwConn.Write([]byte(reqLine)); err != nil {
		return recrawler.Result{URL: seed.URL, Domain: seed.Domain,
			Error: "write: " + err.Error(), FetchTimeMs: ms()}
	}

	// Read only status line: "HTTP/1.1 200 OK\r\n"
	br := bufio.NewReaderSize(rwConn, 256)
	line, err := br.ReadString('\n')
	if err != nil && len(line) < 12 {
		return recrawler.Result{URL: seed.URL, Domain: seed.Domain,
			Error: "read: " + err.Error(), FetchTimeMs: ms()}
	}

	// Status code is at bytes 9-11: "HTTP/1.1 200 OK"
	code := 0
	if len(line) >= 12 {
		code, _ = strconv.Atoi(strings.TrimSpace(line[9:12]))
	}

	return recrawler.Result{
		URL:         seed.URL,
		Domain:      seed.Domain,
		StatusCode:  code,
		FetchTimeMs: ms(),
		CrawledAt:   time.Now(),
	}
}

// isTimeoutErr reports whether an error string indicates a network timeout.
func isTimeoutErr(errStr string) bool {
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "context deadline")
}
