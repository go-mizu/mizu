package crawl

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/crawler"
	"golang.org/x/sync/errgroup"
)

// rawFetch is the result of an HTTP fetch before HTML extraction.
// Fetch workers produce these; parse workers consume them.
type rawFetch struct {
	seed        SeedURL
	statusCode  int
	bodyBytes   []byte // populated only for 200 HTML when !StatusOnly
	contentType string
	contentLen  int64
	redirectURL string
	fetchMs     int64
	errStr      string
}

// staticFrameCache implements DNSCache using data received from the queen.
type staticFrameCache struct {
	resolved map[string]string
	dead     map[string]bool
}

func (c *staticFrameCache) Lookup(host string) (string, bool) {
	ip, ok := c.resolved[host]
	return ip, ok
}
func (c *staticFrameCache) IsDead(host string) bool { return c.dead[host] }

// readDroneInput reads the DNS gob frame then seed JSON lines from r.
// Protocol: [4-byte big-endian length][gob dnsFrame][JSON lines…][EOF]
func readDroneInput(r io.Reader) (dnsFrame, []SeedURL, error) {
	var frameLen uint32
	if err := binary.Read(r, binary.BigEndian, &frameLen); err != nil {
		return dnsFrame{}, nil, fmt.Errorf("read frame length: %w", err)
	}
	frameBytes := make([]byte, frameLen)
	if _, err := io.ReadFull(r, frameBytes); err != nil {
		return dnsFrame{}, nil, fmt.Errorf("read frame bytes: %w", err)
	}
	var frame dnsFrame
	if err := gob.NewDecoder(bytes.NewReader(frameBytes)).Decode(&frame); err != nil {
		return dnsFrame{}, nil, fmt.Errorf("decode dns frame: %w", err)
	}
	var seeds []SeedURL
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		var s SeedURL
		if err := json.Unmarshal(scanner.Bytes(), &s); err == nil {
			seeds = append(seeds, s)
		}
	}
	return frame, seeds, nil
}

// keepaliveFetchRaw performs an HTTP GET and returns raw bytes without HTML extraction.
// For 200 HTML responses (and !cfg.StatusOnly) it reads up to 512 KB of body.
func keepaliveFetchRaw(ctx context.Context, client *http.Client,
	seed SeedURL, cfg Config) rawFetch {

	start := time.Now()
	ms := func() int64 { return time.Since(start).Milliseconds() }

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, seed.URL, nil)
	if err != nil {
		return rawFetch{seed: seed, fetchMs: ms(), errStr: err.Error()}
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return rawFetch{seed: seed, fetchMs: ms(), errStr: err.Error()}
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	cl := max(resp.ContentLength, 0)

	if cfg.StatusOnly {
		var buf [1]byte
		resp.Body.Read(buf[:]) //nolint:errcheck
		return rawFetch{
			seed:        seed,
			statusCode:  resp.StatusCode,
			contentType: ct,
			contentLen:  cl,
			redirectURL: resp.Header.Get("Location"),
			fetchMs:     ms(),
		}
	}

	// Full fetch: read body for 200 HTML pages only (up to 512 KB).
	isHTML := strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml")
	var bodyBytes []byte
	if resp.StatusCode == 200 && isHTML {
		bodyBytes, _ = io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	} else {
		io.Copy(io.Discard, resp.Body) //nolint:errcheck
	}
	bodyLen := max(cl, int64(len(bodyBytes)))

	return rawFetch{
		seed:        seed,
		statusCode:  resp.StatusCode,
		bodyBytes:   bodyBytes,
		contentType: ct,
		contentLen:  bodyLen,
		redirectURL: resp.Header.Get("Location"),
		fetchMs:     ms(),
	}
}

// parseRawFetch converts a rawFetch to a Result by running crawler.Extract.
func parseRawFetch(rf rawFetch) Result {
	if rf.errStr != "" {
		return Result{
			URL:         rf.seed.URL,
			Domain:      rf.seed.Domain,
			Error:       rf.errStr,
			FetchTimeMs: rf.fetchMs,
		}
	}
	var title, description, language, body string
	if rf.statusCode == 200 && len(rf.bodyBytes) > 0 {
		body = string(rf.bodyBytes)
		extracted := crawler.Extract(strings.NewReader(body), rf.seed.URL)
		title = extracted.Title
		description = extracted.Description
		language = extracted.Language
	}
	return Result{
		URL:           rf.seed.URL,
		Domain:        rf.seed.Domain,
		StatusCode:    rf.statusCode,
		ContentType:   rf.contentType,
		ContentLength: rf.contentLen,
		Body:          body,
		Title:         title,
		Description:   description,
		Language:      language,
		RedirectURL:   rf.redirectURL,
		FetchTimeMs:   rf.fetchMs,
		CrawledAt:     time.Now(),
	}
}


// RunDrone is the entry point for the hidden "cc recrawl-drone" subcommand.
// It reads a DNS frame + seed URLs from stdin, runs the 3-stage pipeline,
// and writes result/failure DBs to the paths in cfg.
func RunDrone(ctx context.Context, cfg Config) error {
	frame, seeds, err := readDroneInput(os.Stdin)
	if err != nil {
		return fmt.Errorf("read drone input: %w", err)
	}
	if len(seeds) == 0 {
		return nil
	}

	dns := &staticFrameCache{resolved: frame.Resolved, dead: frame.Dead}

	// Raise fd limit so we can have thousands of concurrent sockets.
	if err := raiseRlimit(65536); err != nil {
		fmt.Fprintf(os.Stderr, "[drone] raiseRlimit: %v (continuing)\n", err)
	}

	// Open result DB.
	if cfg.SwarmResultDir == "" {
		return fmt.Errorf("--result-dir is required for drone")
	}
	if err := os.MkdirAll(cfg.SwarmResultDir, 0o755); err != nil {
		return fmt.Errorf("mkdir result dir: %w", err)
	}
	batchSz := max(cfg.BatchSize, 50)
	if DroneResultDBFactory == nil {
		return fmt.Errorf("DroneResultDBFactory not set; import pkg/crawl/store to initialize")
	}
	rdb, err := DroneResultDBFactory(cfg.SwarmResultDir, 8, batchSz, 0)
	if err != nil {
		return fmt.Errorf("open result db: %w", err)
	}
	defer rdb.Close()

	// Open failed DB (optional – no error if path is empty).
	var failDB FailureWriter
	if cfg.SwarmFailedDB != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.SwarmFailedDB), 0o755); err != nil {
			return fmt.Errorf("mkdir failed db dir: %w", err)
		}
		if DroneFailedDBFactory == nil {
			return fmt.Errorf("DroneFailedDBFactory not set; import pkg/crawl/store to initialize")
		}
		failDB, err = DroneFailedDBFactory(cfg.SwarmFailedDB)
		if err != nil {
			return fmt.Errorf("open failed db: %w", err)
		}
		defer failDB.Close()
	}

	// Counters (updated by parse workers, read by stats reporter).
	var okCount, failCount, timeoutCount atomic.Int64

	// Stage 3: DB write goroutine.
	writeCh := make(chan Result, 256)
	var writeWg sync.WaitGroup
	writeWg.Go(func() {
		for r := range writeCh {
			rdb.Add(r)
		}
		rdb.Flush(context.Background()) //nolint:errcheck
	})

	// Stage 2: parse workers (CPU-bound, NumCPU goroutines).
	parseN := max(runtime.NumCPU(), 2)
	fetchCh := make(chan rawFetch, 2000)
	var parseWg sync.WaitGroup
	for range parseN {
		parseWg.Go(func() {
			for rf := range fetchCh {
				r := parseRawFetch(rf)
				writeCh <- r
				switch {
				case r.Error != "" && isTimeoutErr(r.Error):
					timeoutCount.Add(1)
					if failDB != nil {
						failDB.AddURL(FailedURL{
							URL: r.URL, Domain: r.Domain,
							Reason: "http_timeout", Error: r.Error,
							FetchTimeMs: r.FetchTimeMs,
						})
					}
				case r.Error != "":
					failCount.Add(1)
					if failDB != nil {
						failDB.AddURL(FailedURL{
							URL: r.URL, Domain: r.Domain,
							Reason: "http_error", Error: r.Error,
							FetchTimeMs: r.FetchTimeMs,
						})
					}
				default:
					okCount.Add(1)
				}
			}
		})
	}

	// Stats reporter: write cumulative JSON to stdout every 500ms.
	ticker := time.NewTicker(500 * time.Millisecond)
	reportCtx, cancelReport := context.WithCancel(ctx)
	var reportWg sync.WaitGroup
	reportWg.Go(func() {
		defer ticker.Stop()
		enc := json.NewEncoder(os.Stdout)
		for {
			select {
			case <-ticker.C:
				enc.Encode(droneStats{ //nolint:errcheck
					OK:      okCount.Load(),
					Failed:  failCount.Load(),
					Timeout: timeoutCount.Load(),
					Total:   okCount.Load() + failCount.Load() + timeoutCount.Load(),
				})
			case <-reportCtx.Done():
				return
			}
		}
	})

	// Stage 1: domain-affine fetch pipeline.
	trk := &adaptiveTracker{}
	runSwarmFetch(ctx, seeds, dns, cfg, trk, fetchCh, failDB)

	// Shutdown pipeline in order: fetch → parse → write.
	close(fetchCh)
	parseWg.Wait()
	close(writeCh)
	writeWg.Wait()

	// Stop stats reporter.
	cancelReport()
	reportWg.Wait()

	// Final report.
	ok := okCount.Load()
	fa := failCount.Load()
	to := timeoutCount.Load()
	json.NewEncoder(os.Stdout).Encode(droneStats{ //nolint:errcheck
		OK: ok, Failed: fa, Timeout: to, Total: ok + fa + to,
	})
	return nil
}

// runSwarmFetch groups seeds by domain and runs domain-affine fetch workers.
// Each domain's URLs go to one inner goroutine group sharing a single http.Transport.
// Results are sent to fetchCh (never calling crawler.Extract).
func runSwarmFetch(ctx context.Context, seeds []SeedURL,
	dns DNSCache, cfg Config, trk *adaptiveTracker, fetchCh chan<- rawFetch, failDB FailureWriter) {

	byDomain := make(map[string][]SeedURL, 1024)
	for _, s := range seeds {
		h := s.Host
		if h == "" {
			h = s.Domain
		}
		if dns.IsDead(h) {
			if failDB != nil {
				failDB.AddURL(FailedURL{
					URL: s.URL, Domain: s.Domain, Reason: "domain_dead",
				})
			}
			continue
		}
		byDomain[s.Domain] = append(byDomain[s.Domain], s)
	}
	if len(byDomain) == 0 {
		return
	}

	workCh := make(chan []SeedURL, min(len(byDomain), 4096))
	go func() {
		for _, us := range byDomain {
			workCh <- us
		}
		close(workCh)
	}()

	maxWorkers := cfg.Workers
	if maxWorkers <= 0 {
		maxWorkers = 500
	}
	innerN := cfg.MaxConnsPerDomain
	if innerN <= 0 {
		innerN = 1
	}

	g, gctx := errgroup.WithContext(ctx)
	for range maxWorkers {
		g.Go(func() error {
			for urls := range workCh {
				if gctx.Err() != nil {
					return nil
				}
				swarmProcessDomain(gctx, urls, dns, cfg, trk, innerN, fetchCh, failDB)
			}
			return nil
		})
	}
	g.Wait() //nolint:errcheck
}

// swarmProcessDomain handles all URLs for one domain using a shared http.Transport.
// It mirrors processOneDomain from keepalive.go but sends rawFetch to fetchCh.
func swarmProcessDomain(ctx context.Context, urls []SeedURL,
	dns DNSCache, cfg Config, trk *adaptiveTracker, innerN int, fetchCh chan<- rawFetch,
	failDB FailureWriter) {

	if len(urls) == 0 {
		return
	}
	domain := urls[0].Domain
	host := urls[0].Host
	if host == "" {
		host = domain
	}

	domainCtx := ctx
	if cfg.DomainTimeout > 0 {
		var cancel context.CancelFunc
		domainCtx, cancel = context.WithTimeout(ctx, cfg.DomainTimeout)
		defer cancel()
	}

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

	urlCh := make(chan SeedURL, len(urls))
	for _, u := range urls {
		urlCh <- u
	}
	close(urlCh)

	var domainTimeouts atomic.Int64
	abandonCh := make(chan struct{})
	var abandonOnce sync.Once

	effectiveThreshold := int64(0)
	if cfg.DomainFailThreshold > 0 {
		effectiveThreshold = int64(cfg.DomainFailThreshold) * int64(max(innerN, 1))
	}

	n := min(innerN, len(urls))
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			client := &http.Client{
				Transport: transport,
				Timeout:   cfg.Timeout,
			}
			for seed := range urlCh {
				select {
				case <-ctx.Done():
					return
				default:
				}
				select {
				case <-abandonCh:
					if failDB != nil {
						failDB.AddURL(FailedURL{
							URL: seed.URL, Domain: seed.Domain,
							Reason: "domain_http_timeout_killed",
						})
					}
					continue
				default:
				}
				if cfg.DomainTimeout > 0 {
					select {
					case <-domainCtx.Done():
						if failDB != nil {
							failDB.AddURL(FailedURL{
								URL: seed.URL, Domain: seed.Domain,
								Reason: "domain_deadline_exceeded",
							})
						}
						continue
					default:
					}
				}

				// Apply adaptive timeout (P95×2) if enough samples exist.
				if t := trk.Timeout(cfg.Timeout); t > 0 {
					client.Timeout = t
				} else {
					client.Timeout = cfg.Timeout
				}

				rf := keepaliveFetchRaw(domainCtx, client, seed, cfg)

				// Record successful latencies for adaptive ceiling.
				if rf.errStr == "" {
					trk.record(rf.fetchMs)
				}

				isTimeout := rf.errStr != "" && isTimeoutErr(rf.errStr)
				if isTimeout {
					nt := domainTimeouts.Add(1)
					if effectiveThreshold > 0 && nt >= effectiveThreshold {
						abandonOnce.Do(func() { close(abandonCh) })
					}
				}

				select {
				case fetchCh <- rf:
				case <-ctx.Done():
					return
				}
			}
		})
	}
	wg.Wait()
	_ = domain
}
