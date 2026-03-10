package dcrawler

import (
	"bufio"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds crawler configuration.
type Config struct {
	Domain            string
	SeedURLs          []string
	Workers           int
	MaxConns          int
	MaxIdleConns      int
	Timeout           time.Duration
	MaxDepth          int
	MaxPages          int
	MaxBodySize       int64
	UserAgent         string
	DataDir           string
	ShardCount        int
	BatchSize         int
	StoreBody         bool
	StoreLinks        bool
	RespectRobots     bool
	FollowSitemap     bool
	Resume            bool
	FrontierSize      int
	BloomCapacity     uint
	BloomFPR          float64
	RateLimit         int
	IncludeSubdomain  bool
	ForceHTTP1        bool
	TransportShards   int
	SeedFile          string
	Continuous        bool          // Run non-stop, re-seed when frontier drains
	ReseedInterval    time.Duration // Min interval between re-seeds (default 30s)
	UseRod            bool          // Use headless Chrome via rod for JS-rendered pages
	RodWorkers        int           // Number of browser pages (default 40)
	RodHeadless       bool          // Run rod in headless mode (default true)
	ScrollCount       int           // Browser mode: scroll N times for infinite scroll (0=no scroll)
	ExtractImages     bool          // Extract <img> URLs and store in links table
	RodBlockResources bool          // Block images/fonts/CSS in browser mode for faster loads
	StaleHours        int           // Resume: re-crawl pages older than N hours (0=disabled)
	UseLightpanda     bool          // Use Lightpanda browser via CDP (alternative to Chrome/rod)
	DomainAliases     []string      // Additional domains to treat as same-domain (e.g., "new.qq.com" for "news.qq.com")
	RodNoRenderWait   bool          // Skip DOM stabilization wait (faster for SSG/static Next.js sites)
	UserDataDir       string        // Chrome user-data-dir for persistent cookies/localStorage across restarts
	ProxyURL          string        // HTTP or SOCKS5 proxy URL for Chrome, e.g. "http://user:pass@host:port"
	ProxyFile         string        // File with one proxy URL per line; enables multi-browser-instance mode
	UseWorker         bool          // Proxy fetches through CF Worker (tools/crawler)
	WorkerURL         string        // Worker endpoint (default https://crawler.go-mizu.workers.dev)
	WorkerToken       string        // Bearer token for worker auth
	WorkerBrowser     bool          // Enable CF Browser Rendering on worker side
	WorkerBatch       int           // URLs per batch request (default 10)
	WorkerParallel    int           // Concurrent batch requests (default 20)
}

// DefaultConfig returns optimal defaults for high-throughput single-domain crawling.
// Targets 10K+ pages/sec via HTTP/2 multiplexing.
func DefaultConfig() Config {
	return Config{
		Workers:         1000,
		MaxConns:        200, // ~200 TCP conns × ~250 H2 streams = 50K concurrent
		MaxIdleConns:    500,
		Timeout:         10 * time.Second,
		MaxBodySize:     512 * 1024, // 512KB
		UserAgent:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		DataDir:         defaultDataDir(),
		ShardCount:      8,
		BatchSize:       500,
		StoreBody:       true,
		StoreLinks:      true,
		RespectRobots:   true,
		FollowSitemap:   true,
		FrontierSize:    4_000_000,
		BloomCapacity:   50_000_000,
		BloomFPR:        0.001,
		TransportShards: 16,
	}
}

// AutoBrowserPages returns the optimal number of concurrent browser tabs
// based on available RAM. Formula: clamp(availRAMMB / 50, 20, 150).
//
// Benchmarks (openai.com --no-render-wait --scroll 0, Cloudflare+Next.js):
//
//	server2 (~11 GB avail) → 150 tabs → ~6 pages/s (1012 pages / 173s)
//
// Note: openai.com is rate-limited by Cloudflare and heavy JS (~25s/page).
// Lighter sites without anti-bot will achieve much higher throughput.
func AutoBrowserPages(availRAMMB int) int {
	n := availRAMMB / 50
	if n < 20 {
		n = 20
	}
	if n > 150 {
		n = 150
	}
	return n
}

// readProcMemAvailMB reads MemAvailable from /proc/meminfo (Linux).
// Returns 4000 MB as fallback on non-Linux or read failure.
func readProcMemAvailMB() int {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 4000 // fallback: 4GB
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.Atoi(fields[1])
				if err == nil {
					return kb / 1024
				}
			}
		}
	}
	return 4000 // fallback: 4 GB → AutoBrowserPages(4000) = 80 tabs (dev/macOS default)
}

func defaultDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "crawler")
}

// NormalizeDomain extracts and normalizes the domain from a URL or domain string.
// Strips www. prefix, lowercases, and handles full URLs (extracts hostname only).
func NormalizeDomain(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	// If it looks like a URL, parse it to extract hostname
	if strings.Contains(input, "/") || strings.HasPrefix(input, "http") {
		if !strings.Contains(input, "://") {
			input = "https://" + input
		}
		if u, err := url.Parse(input); err == nil && u.Hostname() != "" {
			input = u.Hostname()
		}
	}
	input = strings.TrimPrefix(input, "www.")
	return input
}

// ExtractSeedURL extracts a seed URL from user input.
// If input is a full URL, returns it as-is. Otherwise builds https://{domain}/.
func ExtractSeedURL(input string) string {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return input
	}
	if strings.Contains(input, "/") {
		return "https://" + input
	}
	return ""
}

// DomainDir returns the directory for storing crawl data for a domain.
func (c *Config) DomainDir() string {
	return filepath.Join(c.DataDir, NormalizeDomain(c.Domain))
}

// ResultDir returns the directory for sharded result DuckDB files.
func (c *Config) ResultDir() string {
	return filepath.Join(c.DomainDir(), "results")
}
