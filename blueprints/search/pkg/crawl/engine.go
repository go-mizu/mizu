// Package recrawl_v3 implements four independent high-performance recrawl engines.
// All engines implement the Engine interface and share ResultWriter / FailureWriter.
package crawl

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"
)

// Engine is implemented by all four v3 strategies.
type Engine interface {
	// Run crawls all seeds, writing results and failures to the provided writers.
	// Returns Stats when done or ctx is cancelled.
	Run(ctx context.Context, seeds []SeedURL, dns DNSCache, cfg Config,
		results ResultWriter, failures FailureWriter) (*Stats, error)
}

// Stats holds performance counters returned after a Run.
type Stats struct {
	Total    int64
	OK       int64
	Failed   int64
	Timeout  int64
	Skipped  int64         // URLs skipped because domain was abandoned
	Bytes    int64         // total bytes received (body size)
	PeakRPS  float64
	AvgRPS   float64
	Duration time.Duration
	P95LatMs int64
	MemRSS   int64 // bytes at end of run
	Workers  int   // resolved worker count after auto-config
}

// Config configures any engine.
type Config struct {
	Workers             int           // concurrent workers (engines A, D, C-drone)
	Timeout             time.Duration // per-request HTTP timeout
	StatusOnly          bool          // discard body, read status line only
	MaxConnsPerDomain   int           // max simultaneous connections per domain (engine A)
	UserAgent           string   // used if UserAgents is empty
	UserAgents          []string // rotation pool; if non-empty, picked randomly per request
	InsecureTLS         bool     // skip TLS verification
	DroneCount          int    // swarm engine: number of drone processes (engine C)
	SearchBinary        string // path to self binary (engine C drones re-exec it)
	DomainFailThreshold      int           // consecutive timeouts before abandoning a domain (0=disabled)
	DomainTimeout            time.Duration // per-domain context deadline; cancel remaining URLs after this (0=disabled, <0=adaptive: 2×sweep time, clamped [30s, AdaptiveTimeoutMax])
	AdaptiveTimeoutMax       time.Duration // upper bound for adaptive DomainTimeout (0=default 10min); lower values free workers from large slow domains faster
	DomainDeadProbe          int           // abandon domain after this many timeouts with 0 successes (0=disabled); catches dead-HTTP domains early
	DomainStallRatio         int           // abandon domain when timeouts ≥ successes×ratio after DomainDeadProbe timeouts (0=disabled); e.g. 20 = >95% timeout rate
	DisableAdaptiveTimeout   bool          // use cfg.Timeout directly; never shrink via adaptive P95 (useful for pass-2 retries where fast domains skew P95 down)
	Notifier                 DomainNotifier // optional domain lifecycle callbacks (nil = disabled)

	// Swarm engine – used by queen to tell drones where to write.
	SwarmResultDir string // base dir; drone i writes to SwarmResultDir/d{i}/
	SwarmFailedDir string // base dir; drone i writes to SwarmFailedDir/failed_{i}.duckdb

	// Swarm drone – set from --result-dir / --failed-db CLI flags.
	SwarmFailedDB string // this drone's failed DB path (e.g. SwarmFailedDir/failed_0.duckdb)
	BatchSize     int    // DB write batch size for ResultDB

	// ProgressFunc is called by the swarm engine every 500ms with cumulative
	// ok/failed/timeout totals from all drones. Nil-safe.
	ProgressFunc func(ok, failed, timeout int64)

	// BodyStore is optional. When set, HTML bodies are written to the CAS store
	// and Result.BodyCID is populated; Result.Body is left empty.
	BodyStore interface {
		Put(body []byte) (cid string, err error)
	}
}

// DomainNotifier receives domain lifecycle events from the engine.
// All methods are nil-safe to call — engines check cfg.Notifier != nil first.
type DomainNotifier interface {
	// StartDomain is called when the engine begins processing a domain's URLs.
	StartDomain(domain string, urlCount int)
	// EndDomain is called when all URLs for a domain have been processed or abandoned.
	EndDomain(domain string)
}

// BrowserUserAgents is the default rotation pool of realistic browser User-Agents.
// Using browser UAs bypasses bot-detection that holds connections open for crawler UAs.
var BrowserUserAgents = []string{
	// Chrome 131 – Windows (most common desktop)
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	// Chrome 131 – macOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	// Chrome 131 – Linux
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	// Firefox 132 – Windows
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:132.0) Gecko/20100101 Firefox/132.0",
	// Firefox 132 – macOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14.7; rv:132.0) Gecko/20100101 Firefox/132.0",
	// Safari 17 – macOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_7_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Safari/605.1.15",
	// Edge 131 – Windows
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0",
	// Chrome 131 – Android (mobile)
	"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.6778.204 Mobile Safari/537.36",
}

// DefaultConfig returns sensible defaults for the remote server.
func DefaultConfig() Config {
	return Config{
		Workers:             1500,
		Timeout:             5 * time.Second,
		StatusOnly:          false, // default to full body download
		MaxConnsPerDomain:   4,
		UserAgent:           "MizuCrawler/3.0",
		UserAgents:          BrowserUserAgents,
		InsecureTLS:         true,
		DroneCount:          4,
		DomainFailThreshold: 3, // abandon domain after 3 consecutive timeouts
		BatchSize:           5000,
	}
}

// PickUserAgent returns a random UA from UserAgents, falling back to UserAgent.
func (c Config) PickUserAgent() string {
	if len(c.UserAgents) > 0 {
		return c.UserAgents[rand.IntN(len(c.UserAgents))]
	}
	return c.UserAgent
}

// DNSCache is a read-only pre-resolved host→IP mapping.
type DNSCache interface {
	// Lookup returns the first resolved IP for host, or ok=false.
	Lookup(host string) (ip string, ok bool)
	// IsDead returns true if host resolved to NXDOMAIN.
	IsDead(host string) bool
}

// ResultWriter accepts crawl results.
type ResultWriter interface {
	Add(r Result)
	Flush(ctx context.Context) error
	Close() error
}

// FailureWriter accepts failed URLs.
type FailureWriter interface {
	AddURL(u FailedURL)
	Close() error
}

// DroneResultDBFactory is set by pkg/crawl/store via init() to break the import cycle.
// swarm_drone.go calls this to create a ResultDB without importing store directly.
// Signature matches store.NewResultDB.
var DroneResultDBFactory func(dir string, shardCount, batchSize, duckMemPerShardMB int) (ResultWriter, error)

// DroneFailedDBFactory is set by pkg/crawl/store via init() to break the import cycle.
// swarm_drone.go calls this to create a FailedDB without importing store directly.
// Signature matches store.OpenFailedDB.
var DroneFailedDBFactory func(path string) (FailureWriter, error)

// New returns the named engine. Valid names: "keepalive", "epoll", "swarm", "rawhttp".
func New(name string) (Engine, error) {
	switch name {
	case "keepalive":
		return &KeepAliveEngine{}, nil
	case "epoll":
		return &EpollEngine{}, nil
	case "swarm":
		return &SwarmEngine{}, nil
	case "rawhttp":
		return &RawHTTPEngine{}, nil
	default:
		return nil, fmt.Errorf("unknown engine %q (valid: keepalive, epoll, swarm, rawhttp)", name)
	}
}

// Full implementations are in their respective files:
// KeepAliveEngine is defined in keepalive.go
// EpollEngine is defined in epoll.go
// SwarmEngine is defined in swarm.go
// RawHTTPEngine is defined in rawhttp.go
