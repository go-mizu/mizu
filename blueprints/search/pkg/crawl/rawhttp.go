package crawl

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

// RawHTTPEngine uses raw net.Conn (bypassing net/http) with an optional
// per-host connection pool for keep-alive reuse.
// Workers scale like KeepAliveEngine (cfg.Workers), but without any net/http
// allocation overhead — no header maps, no response struct, no goroutine per conn.
type RawHTTPEngine struct{}

func (e *RawHTTPEngine) Run(ctx context.Context, seeds []SeedURL,
	dns DNSCache, cfg Config, results ResultWriter, failures FailureWriter) (*Stats, error) {

	// Filter dead domains upfront
	live := make([]SeedURL, 0, len(seeds))
	for _, s := range seeds {
		host := s.Host
		if host == "" {
			host = s.Domain
		}
		if dns.IsDead(host) {
			failures.AddURL(FailedURL{
				URL:    s.URL,
				Domain: s.Domain,
				Reason: "domain_dead",
			})
			continue
		}
		live = append(live, s)
	}

	pool := newRawConnPool(cfg.MaxConnsPerDomain, cfg.Timeout)
	defer pool.CloseAll()

	maxWorkers := cfg.Workers
	if maxWorkers <= 0 {
		maxWorkers = 500
	}

	workCh := make(chan SeedURL, min(len(live), 10000))
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
	for range maxWorkers {
		g.Go(func() error {
			for seed := range workCh {
				if gctx.Err() != nil {
					return nil
				}
				r := rawHTTPFetch(gctx, seed, dns, cfg, pool)
				total.Add(1)
				peak.Record()

				switch {
				case r.Error != "" && isTimeoutErr(r.Error):
					timeout.Add(1)
					failures.AddURL(FailedURL{
						URL:         seed.URL,
						Domain:      seed.Domain,
						Reason:      "http_timeout",
						Error:       r.Error,
						FetchTimeMs: r.FetchTimeMs,
					})
				case r.Error != "":
					failed.Add(1)
					failures.AddURL(FailedURL{
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

// rawHTTPFetch performs a single HTTP/1.1 GET over a raw net.Conn.
// Tries to reuse a pooled connection; opens a new one on miss.
// Returns the connection to the pool on success (after draining response).
func rawHTTPFetch(ctx context.Context, seed SeedURL,
	dns DNSCache, cfg Config, pool *rawConnPool) Result {

	start := time.Now()
	ms := func() int64 { return time.Since(start).Milliseconds() }

	pu, err := parseRawURL(seed.URL)
	if err != nil {
		return Result{URL: seed.URL, Domain: seed.Domain,
			Error: "parse: " + err.Error(), FetchTimeMs: ms()}
	}

	poolKey := pu.Scheme + "://" + pu.Host + ":" + pu.Port

	// Try pooled connection first
	conn, fromPool := pool.Get(poolKey)
	if !fromPool {
		// Open new connection
		dialHost := pu.Host
		if ip, ok := dns.Lookup(pu.Host); ok {
			dialHost = ip
		}
		dialAddr := net.JoinHostPort(dialHost, pu.Port)

		dialCtx, cancel := context.WithDeadline(ctx, time.Now().Add(cfg.Timeout))
		defer cancel()
		rawConn, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", dialAddr)
		if err != nil {
			return Result{URL: seed.URL, Domain: seed.Domain,
				Error: err.Error(), FetchTimeMs: ms()}
		}

		if pu.Scheme == "https" {
			// Set deadline BEFORE TLS handshake so hung servers don't block indefinitely.
			rawConn.SetDeadline(time.Now().Add(cfg.Timeout)) //nolint:errcheck
			tlsConn := tls.Client(rawConn, &tls.Config{
				InsecureSkipVerify: cfg.InsecureTLS, //nolint:gosec
				ServerName:         pu.Host,
			})
			if err := tlsConn.Handshake(); err != nil {
				rawConn.Close()
				return Result{URL: seed.URL, Domain: seed.Domain,
					Error: "tls: " + err.Error(), FetchTimeMs: ms()}
			}
			conn = tlsConn
		} else {
			conn = rawConn
		}
	}

	conn.SetDeadline(time.Now().Add(cfg.Timeout)) //nolint:errcheck

	// Send minimal HTTP/1.1 request with keep-alive
	reqLine := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: %s\r\nConnection: keep-alive\r\n\r\n",
		pu.Path, pu.Host, cfg.PickUserAgent())
	if _, err := conn.Write([]byte(reqLine)); err != nil {
		conn.Close()
		return Result{URL: seed.URL, Domain: seed.Domain,
			Error: "write: " + err.Error(), FetchTimeMs: ms()}
	}

	// Read status line only
	br := bufio.NewReaderSize(conn, 512)
	line, err := br.ReadString('\n')
	if err != nil && len(line) < 12 {
		conn.Close()
		return Result{URL: seed.URL, Domain: seed.Domain,
			Error: "read status: " + err.Error(), FetchTimeMs: ms()}
	}

	code := 0
	if len(line) >= 12 {
		code, _ = strconv.Atoi(strings.TrimSpace(line[9:12]))
	}

	// Drain headers + body before returning conn to pool.
	// Short deadline to avoid blocking on slow response bodies.
	conn.SetDeadline(time.Now().Add(500 * time.Millisecond)) //nolint:errcheck
	drainBuf := make([]byte, 8<<10)
	conn.Read(drainBuf) //nolint:errcheck
	conn.SetDeadline(time.Time{})

	pool.Put(poolKey, conn)

	return Result{
		URL:         seed.URL,
		Domain:      seed.Domain,
		StatusCode:  code,
		FetchTimeMs: ms(),
		CrawledAt:   time.Now(),
	}
}
