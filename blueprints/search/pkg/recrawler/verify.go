package recrawler

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// VerifyResult records the thorough verification of a single failed domain.
type VerifyResult struct {
	Domain         string
	OriginalReason string

	// DNS verification (each resolver independently, 10s timeout)
	DNSSystemIPs     string // IPs from system resolver
	DNSGoogleIPs     string // IPs from Google 8.8.8.8
	DNSCloudflareIPs string // IPs from Cloudflare 1.1.1.1
	DNSAlive         bool   // any resolver returned IPs

	// HTTP verification (30s timeout, tries both HTTP and HTTPS)
	HTTPStatus  int
	HTTPSStatus int
	HTTPError   string
	HTTPSError  string
	HTTPAlive   bool // either HTTP or HTTPS responded

	// Final verdict
	IsTrulyDead   bool // no DNS and no HTTP response
	FalsePositive bool // we marked dead but it's actually reachable

	VerifiedAt   time.Time
	VerifyTimeMs int64
}

// VerifyConfig configures the slow, thorough domain verification.
// Designed for correctness over speed: generous timeouts, few workers.
type VerifyConfig struct {
	Workers     int           // parallel verification workers (default: 10)
	DNSTimeout  time.Duration // per-resolver DNS timeout (default: 10s)
	HTTPTimeout time.Duration // HTTP request timeout (default: 30s)
}

// DefaultVerifyConfig returns conservative defaults for thorough verification.
func DefaultVerifyConfig() VerifyConfig {
	return VerifyConfig{
		Workers:     10,
		DNSTimeout:  10 * time.Second,
		HTTPTimeout: 30 * time.Second,
	}
}

// VerifyProgress reports live progress during verification.
type VerifyProgress struct {
	Total    int
	Done     int64
	Alive    int64
	Dead     int64
	FalsePos int64
	Speed    float64
	Elapsed  time.Duration
}

// VerifyFailedDomains loads failed domains from failedDBPath, verifies each one
// thoroughly (slow, correct), and writes results to outputPath.
// If limit > 0, only the top N domains (by URL count) are verified.
//
// For each domain:
//  1. DNS: tries system, Google 8.8.8.8, Cloudflare 1.1.1.1 (10s timeout each)
//  2. HTTP: tries https://{domain}/ and http://{domain}/ (30s timeout each)
//  3. Verdict: truly dead (all fail) or false positive (any succeed)
func VerifyFailedDomains(ctx context.Context, failedDBPath, outputPath string, cfg VerifyConfig, limit int, onProgress func(VerifyProgress)) error {
	domains, err := LoadFailedDomains(failedDBPath)
	if err != nil {
		return fmt.Errorf("loading failed domains: %w", err)
	}
	if len(domains) == 0 {
		return fmt.Errorf("no failed domains found in %s", failedDBPath)
	}
	if limit > 0 && limit < len(domains) {
		domains = domains[:limit]
	}

	outDB, err := sql.Open("duckdb", outputPath)
	if err != nil {
		return fmt.Errorf("opening output db: %w", err)
	}
	defer outDB.Close()

	if err := initVerifySchema(outDB); err != nil {
		return err
	}

	// Resolvers with generous timeouts (correctness over speed)
	systemResolver := makeResolver("", cfg.DNSTimeout)
	googleResolver := makeResolver("8.8.8.8:53", cfg.DNSTimeout)
	cloudflareResolver := makeResolver("1.1.1.1:53", cfg.DNSTimeout)

	httpClient := &http.Client{
		Timeout: cfg.HTTPTimeout,
		Transport: &http.Transport{
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: false},
			MaxIdleConns:          100,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: cfg.HTTPTimeout,
			DisableCompression:    true,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	var done, aliveCount, deadCount, falsePosCount atomic.Int64
	start := time.Now()

	ch := make(chan FailedDomain, cfg.Workers*2)
	resultCh := make(chan VerifyResult, cfg.Workers*2)

	// Verification workers
	var wg sync.WaitGroup
	for range cfg.Workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for d := range ch {
				select {
				case <-ctx.Done():
					return
				default:
				}
				r := verifyOneDomain(ctx, d, systemResolver, googleResolver, cloudflareResolver, httpClient, cfg)
				resultCh <- r
				done.Add(1)
				if r.FalsePositive {
					falsePosCount.Add(1)
					aliveCount.Add(1)
				} else {
					deadCount.Add(1)
				}
			}
		}()
	}

	// Result writer
	var writerWg sync.WaitGroup
	writerWg.Add(1)
	go func() {
		defer writerWg.Done()
		batch := make([]VerifyResult, 0, 100)
		for r := range resultCh {
			batch = append(batch, r)
			if len(batch) >= 100 {
				writeVerifyBatch(outDB, batch)
				batch = batch[:0]
			}
		}
		if len(batch) > 0 {
			writeVerifyBatch(outDB, batch)
		}
	}()

	// Progress reporter
	progressDone := make(chan struct{})
	if onProgress != nil {
		go func() {
			defer close(progressDone)
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					d := done.Load()
					elapsed := time.Since(start)
					speed := float64(0)
					if elapsed.Seconds() > 0 {
						speed = float64(d) / elapsed.Seconds()
					}
					onProgress(VerifyProgress{
						Total:    len(domains),
						Done:     d,
						Alive:    aliveCount.Load(),
						Dead:     deadCount.Load(),
						FalsePos: falsePosCount.Load(),
						Speed:    speed,
						Elapsed:  elapsed,
					})
					if d >= int64(len(domains)) {
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	} else {
		close(progressDone)
	}

	// Feed domains
	for _, d := range domains {
		select {
		case ch <- d:
		case <-ctx.Done():
			break
		}
	}
	close(ch)
	wg.Wait()
	close(resultCh)
	writerWg.Wait()
	<-progressDone

	// Save summary metadata
	outDB.Exec(`CREATE TABLE IF NOT EXISTS meta (key VARCHAR PRIMARY KEY, value VARCHAR)`)
	outDB.Exec("INSERT OR REPLACE INTO meta VALUES ('total_domains', ?)", fmt.Sprintf("%d", len(domains)))
	outDB.Exec("INSERT OR REPLACE INTO meta VALUES ('alive', ?)", fmt.Sprintf("%d", aliveCount.Load()))
	outDB.Exec("INSERT OR REPLACE INTO meta VALUES ('dead', ?)", fmt.Sprintf("%d", deadCount.Load()))
	outDB.Exec("INSERT OR REPLACE INTO meta VALUES ('false_positives', ?)", fmt.Sprintf("%d", falsePosCount.Load()))
	fpRate := float64(0)
	if len(domains) > 0 {
		fpRate = float64(falsePosCount.Load()) / float64(len(domains)) * 100
	}
	outDB.Exec("INSERT OR REPLACE INTO meta VALUES ('false_positive_rate', ?)", fmt.Sprintf("%.2f%%", fpRate))
	outDB.Exec("INSERT OR REPLACE INTO meta VALUES ('verified_at', ?)", time.Now().Format(time.RFC3339))

	return nil
}

func initVerifySchema(db *sql.DB) error {
	db.Exec("DROP TABLE IF EXISTS verified_domains")
	_, err := db.Exec(`
		CREATE TABLE verified_domains (
			domain VARCHAR PRIMARY KEY,
			original_reason VARCHAR,
			dns_system_ips VARCHAR DEFAULT '',
			dns_google_ips VARCHAR DEFAULT '',
			dns_cloudflare_ips VARCHAR DEFAULT '',
			dns_alive BOOLEAN,
			http_status INTEGER DEFAULT 0,
			https_status INTEGER DEFAULT 0,
			http_error VARCHAR DEFAULT '',
			https_error VARCHAR DEFAULT '',
			http_alive BOOLEAN,
			is_truly_dead BOOLEAN,
			false_positive BOOLEAN,
			verified_at TIMESTAMP,
			verify_time_ms BIGINT
		)
	`)
	return err
}

func verifyOneDomain(ctx context.Context, d FailedDomain, sysRes, googleRes, cfRes *net.Resolver, client *http.Client, cfg VerifyConfig) VerifyResult {
	start := time.Now()
	r := VerifyResult{
		Domain:         d.Domain,
		OriginalReason: d.Reason,
		VerifiedAt:     time.Now(),
	}

	// DNS verification: try all 3 resolvers independently with generous timeout
	r.DNSSystemIPs = verifyLookupIPs(ctx, sysRes, d.Domain, cfg.DNSTimeout)
	r.DNSGoogleIPs = verifyLookupIPs(ctx, googleRes, d.Domain, cfg.DNSTimeout)
	r.DNSCloudflareIPs = verifyLookupIPs(ctx, cfRes, d.Domain, cfg.DNSTimeout)
	r.DNSAlive = r.DNSSystemIPs != "" || r.DNSGoogleIPs != "" || r.DNSCloudflareIPs != ""

	// HTTP verification: only if DNS alive, try both HTTPS and HTTP
	if r.DNSAlive {
		r.HTTPSStatus, r.HTTPSError = verifyHTTP(ctx, client, "https://"+d.Domain+"/")
		r.HTTPStatus, r.HTTPError = verifyHTTP(ctx, client, "http://"+d.Domain+"/")
		r.HTTPAlive = r.HTTPSStatus > 0 || r.HTTPStatus > 0
	}

	// Verdict
	r.IsTrulyDead = !r.DNSAlive && !r.HTTPAlive
	r.FalsePositive = r.DNSAlive || r.HTTPAlive
	r.VerifyTimeMs = time.Since(start).Milliseconds()

	return r
}

func verifyLookupIPs(ctx context.Context, resolver *net.Resolver, domain string, timeout time.Duration) string {
	lookupCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addrs, err := resolver.LookupHost(lookupCtx, domain)
	if err != nil || len(addrs) == 0 {
		return ""
	}
	return strings.Join(addrs, ",")
}

func verifyHTTP(ctx context.Context, client *http.Client, url string) (status int, errMsg string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, truncateStr(err.Error(), 200)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MizuVerify/1.0)")
	req.Header.Set("Accept", "text/html,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return 0, truncateStr(err.Error(), 200)
	}
	resp.Body.Close()
	return resp.StatusCode, ""
}

func writeVerifyBatch(db *sql.DB, batch []VerifyResult) {
	const maxPerStmt = 100
	for i := 0; i < len(batch); i += maxPerStmt {
		end := min(i+maxPerStmt, len(batch))
		chunk := batch[i:end]

		var b strings.Builder
		b.WriteString("INSERT OR REPLACE INTO verified_domains (domain, original_reason, dns_system_ips, dns_google_ips, dns_cloudflare_ips, dns_alive, http_status, https_status, http_error, https_error, http_alive, is_truly_dead, false_positive, verified_at, verify_time_ms) VALUES ")
		args := make([]any, 0, len(chunk)*15)

		for j, r := range chunk {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString("(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
			args = append(args, r.Domain, r.OriginalReason, r.DNSSystemIPs, r.DNSGoogleIPs,
				r.DNSCloudflareIPs, r.DNSAlive, r.HTTPStatus, r.HTTPSStatus, r.HTTPError,
				r.HTTPSError, r.HTTPAlive, r.IsTrulyDead, r.FalsePositive, r.VerifiedAt, r.VerifyTimeMs)
		}

		db.Exec(b.String(), args...)
	}
}

// VerifySummary reads the verified_domains table and returns summary stats.
func VerifySummary(dbPath string) (total, alive, dead, falsePos int, fpRate float64, err error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}
	defer db.Close()

	err = db.QueryRow("SELECT COUNT(*) FROM verified_domains").Scan(&total)
	if err != nil {
		return
	}
	db.QueryRow("SELECT COUNT(*) FROM verified_domains WHERE false_positive = true").Scan(&falsePos)
	alive = falsePos
	dead = total - alive
	if total > 0 {
		fpRate = float64(falsePos) / float64(total) * 100
	}
	return
}

// VerifyFalsePositives returns the domains that were incorrectly marked dead.
func VerifyFalsePositives(dbPath string, limit int) ([]VerifyResult, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT domain, original_reason, dns_system_ips, dns_google_ips, dns_cloudflare_ips,
	                 dns_alive, http_status, https_status, http_error, https_error, http_alive,
	                 is_truly_dead, false_positive, verified_at, verify_time_ms
	          FROM verified_domains WHERE false_positive = true ORDER BY domain`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []VerifyResult
	for rows.Next() {
		var r VerifyResult
		rows.Scan(&r.Domain, &r.OriginalReason, &r.DNSSystemIPs, &r.DNSGoogleIPs,
			&r.DNSCloudflareIPs, &r.DNSAlive, &r.HTTPStatus, &r.HTTPSStatus,
			&r.HTTPError, &r.HTTPSError, &r.HTTPAlive, &r.IsTrulyDead, &r.FalsePositive,
			&r.VerifiedAt, &r.VerifyTimeMs)
		results = append(results, r)
	}
	return results, nil
}
