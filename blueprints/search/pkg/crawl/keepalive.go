package crawl

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/crawler"
	"golang.org/x/sync/errgroup"
)

// KeepAliveEngine groups URLs by domain and processes each domain's URLs
// using a pool of MaxConnsPerDomain concurrent goroutines that all share one
// http.Transport — so connections are reused across the goroutines (keep-alive).
//
// Adaptive timeout: after ≥5 successful samples, uses P95×2 instead of cfg.Timeout.
// Domain abandonment: after DomainFailThreshold total timeouts, remaining URLs are skipped.
type KeepAliveEngine struct{}

// adaptiveTracker is a lock-free latency histogram for P95-based adaptive timeout.
// Workers share a single instance via atomic operations — no mutex needed.
type adaptiveTracker struct {
	buckets [8]atomic.Int64
	total   atomic.Int64
}

var adaptiveEdgesKA = [8]int64{100, 250, 500, 1000, 2000, 3500, 5000, 10000}

func (t *adaptiveTracker) record(ms int64) {
	t.total.Add(1)
	for i, edge := range adaptiveEdgesKA {
		if ms < edge {
			t.buckets[i].Add(1)
			return
		}
	}
	t.buckets[len(t.buckets)-1].Add(1)
}

// Timeout returns P95×2 clamped to [500ms, ceiling]. Returns 0 if <5 samples.
func (t *adaptiveTracker) Timeout(ceiling time.Duration) time.Duration {
	n := t.total.Load()
	if n < 5 {
		return 0
	}
	target := int64(float64(n) * 0.95)
	var cum int64
	for i, edge := range adaptiveEdgesKA {
		cum += t.buckets[i].Load()
		if cum >= target {
			ms := max(edge*2, 500)
			if ceil := ceiling.Milliseconds(); ms > ceil {
				ms = ceil
			}
			return time.Duration(ms) * time.Millisecond
		}
	}
	return ceiling
}

// P95Ms returns the current P95 latency bucket in ms. Returns 0 if <10 samples.
func (t *adaptiveTracker) P95Ms() int64 {
	n := t.total.Load()
	if n < 10 {
		return 0
	}
	target := int64(float64(n) * 0.95)
	var cum int64
	for i, edge := range adaptiveEdgesKA {
		cum += t.buckets[i].Load()
		if cum >= target {
			return edge
		}
	}
	return adaptiveEdgesKA[len(adaptiveEdgesKA)-1]
}

func (e *KeepAliveEngine) Run(ctx context.Context, seeds []SeedURL,
	dns DNSCache, cfg Config, results ResultWriter, failures FailureWriter) (*Stats, error) {

	// Raise fd limit so large worker counts don't hit the 1024 soft limit ceiling.
	// Idempotent: no-op if already ≥ 65536.
	_ = raiseRlimit(65536)

	if len(seeds) == 0 {
		return &Stats{}, nil
	}

	// Sort seeds in-place by domain so we can stream domain-by-domain without
	// building a full byDomain map (~331 MB for 6.9M seeds). At any time only
	// one domain's URL slice is held in the goroutine (~4.5 KB avg).
	sort.Slice(seeds, func(i, j int) bool { return seeds[i].Domain < seeds[j].Domain })

	type domainWork struct {
		urls []SeedURL
	}

	workCh := make(chan domainWork, 4096)
	go func() {
		defer close(workCh)
		var currentDomain string
		var batch []SeedURL
		flush := func() {
			if len(batch) == 0 {
				return
			}
			select {
			case workCh <- domainWork{batch}:
			case <-ctx.Done():
			}
			batch = nil
		}
		for i, s := range seeds {
			// Zero seeds[i] immediately after use so the GC can reclaim the
			// URL/Domain/Host string data progressively as domains complete,
			// rather than holding all 6.9M seeds in memory for the full run.
			seeds[i] = SeedURL{}
			if dns.IsDead(s.Host) {
				failures.AddURL(FailedURL{
					URL:    s.URL,
					Domain: s.Domain,
					Reason: "domain_dead",
				})
				continue
			}
			if s.Domain != currentDomain {
				flush()
				currentDomain = s.Domain
			}
			batch = append(batch, s)
		}
		flush()
	}()

	// Inner parallelism: concurrent requests within one domain (keep-alive pool).
	innerN := cfg.MaxConnsPerDomain
	if innerN <= 0 {
		innerN = 1
	}

	// Outer workers: pick up domains. Do NOT cap at len(byDomain) — extra workers
	// just exit immediately when the channel is drained.
	maxWorkers := cfg.Workers
	if maxWorkers <= 0 {
		maxWorkers = 500
	}

	var (
		ok      atomic.Int64
		failed  atomic.Int64
		timeout atomic.Int64
		skipped atomic.Int64
		total   atomic.Int64
		bytes   atomic.Int64
	)

	adaptive := &adaptiveTracker{}
	start := time.Now()
	peak := &peakTracker{}

	g, gctx := errgroup.WithContext(ctx)
	for range maxWorkers {
		g.Go(func() error {
			for work := range workCh {
				if gctx.Err() != nil {
					return nil
				}
				processOneDomain(gctx, work.urls, dns, cfg, adaptive, innerN,
					results, failures,
					&ok, &failed, &timeout, &skipped, &total, &bytes, peak)
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
		Skipped:  skipped.Load(),
		Bytes:    bytes.Load(),
		PeakRPS:  peak.Peak(),
		AvgRPS:   avgRPS,
		Duration: dur,
		P95LatMs: adaptive.P95Ms(),
		MemRSS:   rssNow(),
	}, nil
}

func processOneDomain(ctx context.Context, urls []SeedURL,
	dns DNSCache, cfg Config, adaptive *adaptiveTracker, innerN int,
	results ResultWriter, failures FailureWriter,
	ok, failed, timeout, skipped, total *atomic.Int64, bytesTotal *atomic.Int64, peak *peakTracker) {

	if len(urls) == 0 {
		return
	}
	domain := urls[0].Domain
	host := urls[0].Host
	if host == "" {
		host = domain
	}
	if cfg.Notifier != nil {
		cfg.Notifier.StartDomain(domain, len(urls))
		defer cfg.Notifier.EndDomain(domain)
	}

	// Per-domain context deadline.
	// DomainTimeout > 0: explicit deadline.
	// DomainTimeout < 0: adaptive — 2× worst-case sweep time, clamped [30s, 10min].
	// DomainTimeout == 0: disabled.
	domainCtx := ctx
	effectiveDomainTimeout := cfg.DomainTimeout
	if effectiveDomainTimeout < 0 {
		sweep := time.Duration(len(urls)) * cfg.Timeout / time.Duration(max(innerN, 1)) * 2
		if sweep < 30*time.Second {
			sweep = 30 * time.Second
		}
		maxCap := cfg.AdaptiveTimeoutMax
		if maxCap <= 0 {
			maxCap = 10 * time.Minute
		}
		if sweep > maxCap {
			sweep = maxCap
		}
		effectiveDomainTimeout = sweep
	}
	if effectiveDomainTimeout > 0 {
		var cancelDomain context.CancelFunc
		domainCtx, cancelDomain = context.WithTimeout(ctx, effectiveDomainTimeout)
		defer cancelDomain()
	}

	// Shared transport — all innerN goroutines reuse the same connection pool.
	tlsCfg := &tls.Config{
		InsecureSkipVerify: cfg.InsecureTLS, //nolint:gosec
		ServerName:         host,
	}
	transport := &http.Transport{
		TLSClientConfig:     tlsCfg,
		MaxIdleConnsPerHost: max(innerN, 1),
		IdleConnTimeout:     15 * time.Second,
		DisableCompression:  cfg.StatusOnly,
	}
	if ip, found := dns.Lookup(host); found {
		transport.DialContext = dialWithIP(ip)
	}
	defer transport.CloseIdleConnections()

	// Feed all URLs into a channel; innerN goroutines drain it concurrently.
	urlCh := make(chan SeedURL, len(urls))
	for _, u := range urls {
		urlCh <- u
	}
	close(urlCh)

	var domainTimeouts atomic.Int64
	var domainSuccesses atomic.Int64
	abandonCh := make(chan struct{})
	var abandonOnce sync.Once

	n := min(innerN, len(urls))

	// Domain abandonment: after DomainFailThreshold *rounds* of timeouts, skip remaining.
	// "Round" = n concurrent workers. This prevents premature abandonment when
	// n parallel workers all timeout on the first batch simultaneously.
	// effectiveThreshold = DomainFailThreshold × n
	effectiveThreshold := int64(0)
	if cfg.DomainFailThreshold > 0 {
		effectiveThreshold = int64(cfg.DomainFailThreshold) * int64(max(n, 1))
	}
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{
				Transport: transport,
				Timeout:   cfg.Timeout,
			}
			for seed := range urlCh {
				// Fast-abort on global context cancellation
				select {
				case <-ctx.Done():
					return
				default:
				}
				// Domain abandoned?
				select {
				case <-abandonCh:
					skipped.Add(1)
					failures.AddURL(FailedURL{
						URL:    seed.URL,
						Domain: seed.Domain,
						Reason: "domain_http_timeout_killed",
					})
					continue
				default:
				}
				// Per-domain context deadline exceeded?
				if effectiveDomainTimeout > 0 {
					select {
					case <-domainCtx.Done():
						skipped.Add(1)
						failures.AddURL(FailedURL{
							URL:    seed.URL,
							Domain: seed.Domain,
							Reason: "domain_deadline_exceeded",
						})
						continue
					default:
					}
				}

				// Apply adaptive timeout (skip if disabled — pass-2 uses full timeout
				// to avoid fast domains skewing P95 down and cutting off slow retries).
				if !cfg.DisableAdaptiveTimeout {
					if t := adaptive.Timeout(cfg.Timeout); t > 0 {
						client.Timeout = t
					} else {
						client.Timeout = cfg.Timeout
					}
				} else {
					client.Timeout = cfg.Timeout
				}

				r := keepaliveFetchOne(domainCtx, client, seed, cfg)
				total.Add(1)
				peak.Record()
				bytesTotal.Add(r.ContentLength)

				isTimeout := strings.Contains(r.Error, "timeout") ||
					strings.Contains(r.Error, "deadline exceeded") ||
					strings.Contains(r.Error, "context deadline")

				switch {
				case r.Error != "" && isTimeout:
					timeout.Add(1)
					n := domainTimeouts.Add(1)
					if effectiveThreshold > 0 && n >= effectiveThreshold {
						abandonOnce.Do(func() { close(abandonCh) })
					}
					// Dead-domain probe: if DomainDeadProbe timeouts with 0 successes,
					// the domain's TCP/HTTP layer is dead — abandon remaining URLs.
					// Stall probe: if timeouts ≥ successes×DomainStallRatio after the probe
					// window (e.g. ratio=20 → >95% timeout rate), abandon — domain is alive
					// but too slow to be worth crawling further.
					if cfg.DomainDeadProbe > 0 && n >= int64(cfg.DomainDeadProbe) {
						s := domainSuccesses.Load()
						isDead := s == 0
						isStalling := cfg.DomainStallRatio > 0 && s > 0 && n >= int64(s)*int64(cfg.DomainStallRatio)
						if isDead || isStalling {
							abandonOnce.Do(func() { close(abandonCh) })
						}
					}
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
					domainSuccesses.Add(1)
					adaptive.record(r.FetchTimeMs) // only successful latencies
				}
				results.Add(r)
			}
		}()
	}
	wg.Wait()
}

func keepaliveFetchOne(ctx context.Context, client *http.Client,
	seed SeedURL, cfg Config) Result {

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, seed.URL, nil)
	if err != nil {
		return Result{
			URL: seed.URL, Domain: seed.Domain,
			Error: err.Error(), FetchTimeMs: time.Since(start).Milliseconds(),
		}
	}
	req.Header.Set("User-Agent", cfg.PickUserAgent())

	resp, err := client.Do(req)
	if err != nil {
		return Result{
			URL: seed.URL, Domain: seed.Domain,
			Error: err.Error(), FetchTimeMs: time.Since(start).Milliseconds(),
		}
	}
	defer resp.Body.Close()

	if cfg.StatusOnly {
		// Read 1 byte to allow connection reuse, then discard
		var buf [1]byte
		resp.Body.Read(buf[:]) //nolint:errcheck
		return Result{
			URL:           seed.URL,
			Domain:        seed.Domain,
			StatusCode:    resp.StatusCode,
			ContentType:   resp.Header.Get("Content-Type"),
			ContentLength: max(resp.ContentLength, 0),
			RedirectURL:   resp.Header.Get("Location"),
			FetchTimeMs:   time.Since(start).Milliseconds(),
			CrawledAt:     time.Now(),
		}
	}

	// Full fetch: read body (up to 256 KB), extract metadata for HTML pages
	ct := resp.Header.Get("Content-Type")
	isHTML := strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml")
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	bodySize := max(resp.ContentLength, int64(len(bodyBytes)))

	var title, description, language, bodyCID string
	if resp.StatusCode == 200 && isHTML && len(bodyBytes) > 0 {
		bodyStr := string(bodyBytes)
		extracted := crawler.Extract(strings.NewReader(bodyStr), seed.URL)
		title = extracted.Title
		description = extracted.Description
		language = extracted.Language
		if cfg.BodyStore != nil {
			if cid, err := cfg.BodyStore.Put(bodyBytes); err == nil {
				bodyCID = cid
			}
		}
		// Always discard body from Result (overflow string fix — Result.Body is always "").
		// Bodies are only accessible via BodyCID when cfg.BodyStore is set.
	}

	return Result{
		URL:           seed.URL,
		Domain:        seed.Domain,
		StatusCode:    resp.StatusCode,
		ContentType:   ct,
		ContentLength: bodySize,
		BodyCID:       bodyCID,
		Title:         title,
		Description:   description,
		Language:      language,
		RedirectURL:   resp.Header.Get("Location"),
		FetchTimeMs:   time.Since(start).Milliseconds(),
		CrawledAt:     time.Now(),
	}
}
