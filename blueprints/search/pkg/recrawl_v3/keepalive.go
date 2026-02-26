package recrawl_v3

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/recrawler"
	"golang.org/x/sync/errgroup"
)

// KeepAliveEngine groups URLs by domain and processes each domain's URLs
// sequentially using a single http.Client with persistent keep-alive connections.
// This eliminates per-request TLS handshake overhead (~100–300ms per request).
type KeepAliveEngine struct{}

func (e *KeepAliveEngine) Run(ctx context.Context, seeds []recrawler.SeedURL,
	dns DNSCache, cfg Config, results ResultWriter, failures FailureWriter) (*Stats, error) {

	// Skip dead domains up front; group live URLs by domain
	byDomain := make(map[string][]recrawler.SeedURL, 1024)
	for _, s := range seeds {
		if dns.IsDead(s.Host) {
			failures.AddURL(recrawler.FailedURL{
				URL:    s.URL,
				Domain: s.Domain,
				Reason: "domain_dead",
			})
			continue
		}
		byDomain[s.Domain] = append(byDomain[s.Domain], s)
	}

	type domainWork struct {
		domain string
		urls   []recrawler.SeedURL
	}

	workCh := make(chan domainWork, len(byDomain))
	for d, us := range byDomain {
		workCh <- domainWork{d, us}
	}
	close(workCh)

	maxWorkers := cfg.Workers
	if maxWorkers <= 0 {
		maxWorkers = 500
	}
	if maxWorkers > len(byDomain) && len(byDomain) > 0 {
		maxWorkers = len(byDomain)
	}
	if maxWorkers == 0 {
		// No live domains
		return &Stats{}, nil
	}

	var (
		ok      atomic.Int64
		failed  atomic.Int64
		timeout atomic.Int64
		total   atomic.Int64
	)

	start := time.Now()
	peak := &peakTracker{}

	g, gctx := errgroup.WithContext(ctx)
	for range maxWorkers {
		g.Go(func() error {
			for work := range workCh {
				if gctx.Err() != nil {
					return nil
				}
				processOneDomain(gctx, work.urls, dns, cfg,
					results, failures, &ok, &failed, &timeout, &total, peak)
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

func processOneDomain(ctx context.Context, urls []recrawler.SeedURL,
	dns DNSCache, cfg Config, results ResultWriter, failures FailureWriter,
	ok, failed, timeout, total *atomic.Int64, peak *peakTracker) {

	if len(urls) == 0 {
		return
	}
	domain := urls[0].Domain
	host := urls[0].Host
	if host == "" {
		host = domain
	}

	tlsCfg := &tls.Config{
		InsecureSkipVerify: cfg.InsecureTLS, //nolint:gosec
		ServerName:         host,
	}
	transport := &http.Transport{
		TLSClientConfig:     tlsCfg,
		MaxIdleConnsPerHost: cfg.MaxConnsPerDomain,
		IdleConnTimeout:     15 * time.Second,
		DisableCompression:  true,
	}
	if ip, found := dns.Lookup(host); found {
		transport.DialContext = dialWithIP(ip)
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	for _, seed := range urls {
		if ctx.Err() != nil {
			transport.CloseIdleConnections()
			return
		}
		r := keepaliveFetchOne(ctx, client, seed, cfg)
		total.Add(1)
		peak.Record()

		isTimeout := strings.Contains(r.Error, "timeout") ||
			strings.Contains(r.Error, "deadline exceeded") ||
			strings.Contains(r.Error, "context deadline")

		switch {
		case r.Error != "" && isTimeout:
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
	transport.CloseIdleConnections()
}

func keepaliveFetchOne(ctx context.Context, client *http.Client,
	seed recrawler.SeedURL, cfg Config) recrawler.Result {

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, seed.URL, nil)
	if err != nil {
		return recrawler.Result{
			URL: seed.URL, Domain: seed.Domain,
			Error: err.Error(), FetchTimeMs: time.Since(start).Milliseconds(),
		}
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	ms := time.Since(start).Milliseconds()
	if err != nil {
		return recrawler.Result{
			URL: seed.URL, Domain: seed.Domain,
			Error: err.Error(), FetchTimeMs: ms,
		}
	}
	defer resp.Body.Close()

	if cfg.StatusOnly {
		// Read 1 byte to allow connection reuse, then discard
		buf := [1]byte{}
		resp.Body.Read(buf[:]) //nolint:errcheck
	}

	return recrawler.Result{
		URL:         seed.URL,
		Domain:      seed.Domain,
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		RedirectURL: resp.Header.Get("Location"),
		FetchTimeMs: ms,
		CrawledAt:   time.Now(),
	}
}
