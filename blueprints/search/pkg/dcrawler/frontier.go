package dcrawler

import (
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/bits-and-blooms/bloom/v3"
)

// Frontier manages the URL queue with bloom filter dedup and same-domain enforcement.
type Frontier struct {
	ch         chan CrawlItem
	filter     *bloom.BloomFilter
	mu         sync.Mutex
	domain     string
	includeSub bool
	robots     *RobotsChecker
	added      atomic.Int64
	rejected   atomic.Int64
}

// NewFrontier creates a new URL frontier with bloom filter dedup.
func NewFrontier(domain string, capacity int, bloomCap uint, bloomFPR float64, includeSub bool) *Frontier {
	return &Frontier{
		ch:         make(chan CrawlItem, capacity),
		filter:     bloom.NewWithEstimates(bloomCap, bloomFPR),
		domain:     NormalizeDomain(domain),
		includeSub: includeSub,
	}
}

// SetRobots sets the robots.txt checker for path filtering.
func (f *Frontier) SetRobots(r *RobotsChecker) {
	f.robots = r
}

// TryAdd normalizes, deduplicates, and enqueues a URL. Returns true if added.
func (f *Frontier) TryAdd(rawURL string, depth int) bool {
	normalized := NormalizeURL(rawURL)
	if normalized == "" {
		return false
	}
	u, err := url.Parse(normalized)
	if err != nil {
		return false
	}
	if !f.isSameDomain(u) {
		return false
	}
	if f.robots != nil && !f.robots.IsAllowed(u.Path) {
		f.rejected.Add(1)
		return false
	}
	f.mu.Lock()
	if f.filter.TestString(normalized) {
		f.mu.Unlock()
		return false
	}
	// Don't add to bloom yet — wait for successful enqueue.
	// Adding before enqueue permanently loses URLs when channel is full:
	// URL is marked "seen" but never enqueued, and can never be re-discovered.
	f.mu.Unlock()

	select {
	case f.ch <- CrawlItem{URL: normalized, Depth: depth}:
		f.mu.Lock()
		f.filter.AddString(normalized)
		f.mu.Unlock()
		f.added.Add(1)
		return true
	default:
		return false // frontier full — URL NOT in bloom, can be re-discovered
	}
}

// MarkSeen adds a URL to the bloom filter without enqueuing it (for resume).
func (f *Frontier) MarkSeen(rawURL string) {
	normalized := NormalizeURL(rawURL)
	if normalized == "" {
		return
	}
	f.mu.Lock()
	f.filter.AddString(normalized)
	f.mu.Unlock()
}

// Len returns the current number of items in the frontier channel.
func (f *Frontier) Len() int {
	return len(f.ch)
}

// BloomCount returns the approximate number of items in the bloom filter.
func (f *Frontier) BloomCount() uint32 {
	return f.filter.ApproximatedSize()
}

// Added returns the total number of URLs successfully added to the frontier.
func (f *Frontier) Added() int64 {
	return f.added.Load()
}

// Chan returns the frontier channel for workers to read from.
func (f *Frontier) Chan() <-chan CrawlItem {
	return f.ch
}

// Drain removes all items from the frontier channel and returns them.
// Used to save state on shutdown.
func (f *Frontier) Drain() []CrawlItem {
	var items []CrawlItem
	for {
		select {
		case item := <-f.ch:
			items = append(items, item)
		default:
			return items
		}
	}
}

// PushDirect adds an item to the frontier bypassing bloom/domain checks.
// Used when restoring from state DB (already validated).
func (f *Frontier) PushDirect(item CrawlItem) bool {
	// Mark as seen in bloom so it won't be re-discovered
	f.mu.Lock()
	f.filter.AddString(item.URL)
	f.mu.Unlock()
	select {
	case f.ch <- item:
		return true
	default:
		return false
	}
}

// trackingParams are URL parameters that serve only for analytics tracking.
// Stripping them prevents duplicate crawl entries for the same page.
var trackingParams = map[string]bool{
	"utm_source": true, "utm_medium": true, "utm_campaign": true,
	"utm_term": true, "utm_content": true, "utm_id": true,
	"fbclid": true, "gclid": true, "gclsrc": true,
	"msclkid": true, "twclid": true, "igshid": true,
	"mc_cid": true, "mc_eid": true,
	"_ga": true, "_gl": true, "_hsenc": true, "_hsmi": true,
}

func (f *Frontier) isSameDomain(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")
	if host == f.domain {
		return true
	}
	if f.includeSub && strings.HasSuffix(host, "."+f.domain) {
		return true
	}
	return false
}

// NormalizeURL normalizes a URL for dedup: lowercase scheme/host, remove fragment,
// remove default ports, sort query params, remove trailing slash (except root).
func NormalizeURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	// Remove fragment before parsing
	if i := strings.IndexByte(rawURL, '#'); i >= 0 {
		rawURL = rawURL[:i]
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	// Only HTTP(S)
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return ""
	}
	u.Scheme = scheme
	// Lowercase host
	u.Host = strings.ToLower(u.Host)
	// Remove default port
	host := u.Hostname()
	port := u.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		u.Host = host
	}
	// Remove trailing slash (except root)
	if u.Path != "/" && strings.HasSuffix(u.Path, "/") {
		u.Path = strings.TrimRight(u.Path, "/")
	}
	if u.Path == "" {
		u.Path = "/"
	}
	// Sort query parameters and strip tracking params
	if u.RawQuery != "" {
		params := u.Query()
		keys := make([]string, 0, len(params))
		for k := range params {
			if !trackingParams[strings.ToLower(k)] {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		var b strings.Builder
		for i, k := range keys {
			vals := params[k]
			sort.Strings(vals)
			for j, v := range vals {
				if i > 0 || j > 0 {
					b.WriteByte('&')
				}
				b.WriteString(url.QueryEscape(k))
				b.WriteByte('=')
				b.WriteString(url.QueryEscape(v))
			}
		}
		u.RawQuery = b.String()
	}
	u.Fragment = ""
	return u.String()
}
