// Package recrawl_v3 implements four independent high-performance recrawl engines.
// All engines implement the Engine interface and share ResultWriter / FailureWriter.
package recrawl_v3

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/recrawler"
)

// Engine is implemented by all four v3 strategies.
type Engine interface {
	// Run crawls all seeds, writing results and failures to the provided writers.
	// Returns Stats when done or ctx is cancelled.
	Run(ctx context.Context, seeds []recrawler.SeedURL, dns DNSCache, cfg Config,
		results ResultWriter, failures FailureWriter) (*Stats, error)
}

// Stats holds performance counters returned after a Run.
type Stats struct {
	Total    int64
	OK       int64
	Failed   int64
	Timeout  int64
	PeakRPS  float64
	AvgRPS   float64
	Duration time.Duration
	P95LatMs int64
	MemRSS   int64 // bytes at end of run
}

// Config configures any engine.
type Config struct {
	Workers           int           // concurrent workers (engines A, D, C-drone)
	Timeout           time.Duration // per-request HTTP timeout
	StatusOnly        bool          // discard body, read status line only
	MaxConnsPerDomain int           // max simultaneous connections per domain (engine A)
	UserAgent         string
	InsecureTLS       bool   // skip TLS verification
	DroneCount        int    // swarm engine: number of drone processes (engine C)
	SearchBinary      string // path to self binary (engine C drones re-exec it)
}

// DefaultConfig returns sensible defaults for the remote server.
func DefaultConfig() Config {
	return Config{
		Workers:           1500,
		Timeout:           5 * time.Second,
		StatusOnly:        true,
		MaxConnsPerDomain: 4,
		UserAgent:         "MizuCrawler/3.0",
		InsecureTLS:       true,
		DroneCount:        4,
	}
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
	Add(r recrawler.Result)
	Flush(ctx context.Context) error
	Close() error
}

// FailureWriter accepts failed URLs.
type FailureWriter interface {
	AddURL(u recrawler.FailedURL)
	Close() error
}

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

// Stub types — replaced by full implementations in keepalive.go, epoll.go, swarm.go, rawhttp.go
// KeepAliveEngine is defined in keepalive.go
// EpollEngine is defined in epoll.go
// SwarmEngine is defined in swarm.go
type RawHTTPEngine struct{}

func (e *RawHTTPEngine) Run(_ context.Context, _ []recrawler.SeedURL, _ DNSCache, _ Config, _ ResultWriter, _ FailureWriter) (*Stats, error) {
	return nil, fmt.Errorf("RawHTTPEngine not yet implemented")
}
