package x

// proxy_pool.go — free proxy rotation for guest token rate-limit bypass.
//
// X's guest token rate limits are IP-based at the /1.1/guest/activate.json
// endpoint. By fetching guest tokens through different proxy IPs, each token
// has its own search rate-limit bucket — effectively multiplying the throughput.
//
// Proxy sources (public free lists):
//
//	https://raw.githubusercontent.com/proxifly/free-proxy-list/main/proxies/protocols/https/data.txt (HTTPS)
//	https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/http.txt (HTTP)
//	https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/socks5.txt (SOCKS5)

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

const (
	proxyBadExpiry    = 24 * time.Hour
	proxyTokenExpiry  = 45 * time.Minute
	proxyFetchTimeout = 10 * time.Second
	proxyMaxGood      = 100
	proxyMaxFetch     = 200
	proxyActivateURL  = "https://api.twitter.com/1.1/guest/activate.json"
)

// proxySource describes a public free-proxy list and the protocol it contains.
type proxySource struct {
	URL   string
	Proto string // "http", "https", "socks5"
}

var proxySources = []proxySource{
	{
		URL:   "https://raw.githubusercontent.com/proxifly/free-proxy-list/main/proxies/protocols/https/data.txt",
		Proto: "https",
	},
	{
		URL:   "https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/http.txt",
		Proto: "http",
	},
	{
		URL:   "https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/socks5.txt",
		Proto: "socks5",
	},
}

// goodProxy holds a proxy that has successfully produced a guest token.
type goodProxy struct {
	URL      string    `json:"url"`
	Proto    string    `json:"proto"`
	UseCount int       `json:"use_count"`
	LastUsed time.Time `json:"last_used"`
}

// tokenEntry is a cached guest token obtained via a specific proxy.
type tokenEntry struct {
	token string
	exp   time.Time
}

// ProxyPool manages a rotating set of proxies for guest token acquisition.
// Good proxies (those that successfully returned a token) are persisted to disk
// and tried first on subsequent runs. Bad proxies are blacklisted for 24 hours.
type ProxyPool struct {
	mu      sync.Mutex
	dataDir string                // ~/data/x/ — for good/bad proxy cache files
	good    []goodProxy           // proxies that worked, ordered most-recently-used first
	bad     map[string]time.Time  // proxy URL → when it was marked bad
	tokens  map[string]tokenEntry // proxy URL → cached guest token
	loaded  bool
}

// NewProxyPool creates a ProxyPool that stores its cache files under dataDir.
func NewProxyPool(dataDir string) *ProxyPool {
	return &ProxyPool{
		dataDir: dataDir,
		bad:     make(map[string]time.Time),
		tokens:  make(map[string]tokenEntry),
	}
}

// ── Public API ────────────────────────────────────────────────────────────────

// FetchGuestToken returns a valid guest token obtained through one of the
// pooled proxies. It tries known-good proxies first, then fetches fresh proxy
// lists when those are exhausted or all rate-limited.
func (p *ProxyPool) FetchGuestToken() (string, error) {
	p.mu.Lock()
	if !p.loaded {
		p.load()
	}
	p.mu.Unlock()

	// 1. Try cached tokens that haven't expired yet.
	if tok := p.pickCachedToken(); tok != "" {
		return tok, nil
	}

	// 2. Try good proxies (most recently used first).
	p.mu.Lock()
	goodCopy := make([]goodProxy, len(p.good))
	copy(goodCopy, p.good)
	p.mu.Unlock()

	for _, gp := range goodCopy {
		if p.isBad(gp.URL) {
			continue
		}
		tok, err := p.FetchGuestTokenViaProxy(gp.URL, gp.Proto)
		if err != nil {
			p.MarkBad(gp.URL)
			continue
		}
		p.MarkGood(gp.URL, gp.Proto)
		p.cacheToken(gp.URL, tok)
		return tok, nil
	}

	// 3. Fetch fresh proxy lists and try each.
	fresh, err := fetchProxyList()
	if err != nil {
		return "", fmt.Errorf("proxy pool: fetch list: %w", err)
	}

	for _, gp := range fresh {
		if p.isBad(gp.URL) {
			continue
		}
		tok, err := p.FetchGuestTokenViaProxy(gp.URL, gp.Proto)
		if err != nil {
			p.MarkBad(gp.URL)
			continue
		}
		p.MarkGood(gp.URL, gp.Proto)
		p.cacheToken(gp.URL, tok)
		return tok, nil
	}

	return "", fmt.Errorf("proxy pool: all %d proxies exhausted", len(fresh))
}

// FetchGuestTokenViaProxy fetches a guest token by routing the activate request
// through the given proxy. proxyURL should be "host:port"; proto is one of
// "http", "https", or "socks5".
func (p *ProxyPool) FetchGuestTokenViaProxy(proxyAddr, proto string) (string, error) {
	transport, err := buildTransport(proxyAddr, proto)
	if err != nil {
		return "", fmt.Errorf("build transport for %s: %w", proxyAddr, err)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   proxyFetchTimeout,
	}

	req, err := http.NewRequest("POST", proxyActivateURL, nil)
	if err != nil {
		return "", fmt.Errorf("proxy guest activate request: %w", err)
	}
	req.Header.Set("Authorization", bearerToken)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("proxy guest activate fetch (%s): %w", proxyAddr, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", fmt.Errorf("proxy guest activate read: %w", err)
	}
	if resp.StatusCode != 200 {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 120 {
			snippet = snippet[:120]
		}
		return "", fmt.Errorf("proxy guest activate HTTP %d via %s: %s", resp.StatusCode, proxyAddr, snippet)
	}

	var result struct {
		GuestToken string `json:"guest_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("proxy guest activate parse: %w", err)
	}
	if result.GuestToken == "" {
		return "", fmt.Errorf("proxy guest activate via %s: empty token", proxyAddr)
	}
	return result.GuestToken, nil
}

// MarkBad records a proxy as non-functional. It will be ignored for 24 hours.
func (p *ProxyPool) MarkBad(proxyURL string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.bad[proxyURL] = time.Now()
	// Remove from good list.
	good := p.good[:0]
	for _, gp := range p.good {
		if gp.URL != proxyURL {
			good = append(good, gp)
		}
	}
	p.good = good
	p.save()
}

// MarkGood records a proxy as successful, incrementing its use count and
// moving it to the front of the good list.
func (p *ProxyPool) MarkGood(proxyURL, proto string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove existing entry if present.
	updated := goodProxy{URL: proxyURL, Proto: proto, UseCount: 1, LastUsed: time.Now()}
	filtered := p.good[:0]
	for _, gp := range p.good {
		if gp.URL == proxyURL {
			updated.UseCount = gp.UseCount + 1
		} else {
			filtered = append(filtered, gp)
		}
	}
	// Prepend (most recently used first).
	p.good = append([]goodProxy{updated}, filtered...)
	// Cap the list.
	if len(p.good) > proxyMaxGood {
		p.good = p.good[:proxyMaxGood]
	}
	// Remove from bad list.
	delete(p.bad, proxyURL)
	p.save()
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func (p *ProxyPool) isBad(proxyURL string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	t, ok := p.bad[proxyURL]
	if !ok {
		return false
	}
	if time.Since(t) > proxyBadExpiry {
		delete(p.bad, proxyURL)
		return false
	}
	return true
}

func (p *ProxyPool) pickCachedToken() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	for proxyURL, entry := range p.tokens {
		if now.Before(entry.exp) {
			_ = proxyURL
			return entry.token
		}
		delete(p.tokens, proxyURL)
	}
	return ""
}

func (p *ProxyPool) cacheToken(proxyURL, token string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tokens[proxyURL] = tokenEntry{
		token: token,
		exp:   time.Now().Add(proxyTokenExpiry),
	}
}

// load reads good_proxies.json and bad_proxies.json from dataDir.
// Expired bad proxies (>24h) are discarded automatically.
// Must be called with p.mu held.
func (p *ProxyPool) load() {
	p.loaded = true

	// Load good proxies.
	goodPath := filepath.Join(p.dataDir, "good_proxies.json")
	if data, err := os.ReadFile(goodPath); err == nil {
		var list []goodProxy
		if json.Unmarshal(data, &list) == nil {
			p.good = list
		}
	}

	// Load bad proxies, purge expired entries.
	badPath := filepath.Join(p.dataDir, "bad_proxies.json")
	if data, err := os.ReadFile(badPath); err == nil {
		var raw map[string]time.Time
		if json.Unmarshal(data, &raw) == nil {
			cutoff := time.Now().Add(-proxyBadExpiry)
			for k, t := range raw {
				if t.After(cutoff) {
					p.bad[k] = t
				}
			}
		}
	}
}

// save writes good_proxies.json and bad_proxies.json to dataDir.
// Must be called with p.mu held.
func (p *ProxyPool) save() {
	if err := os.MkdirAll(p.dataDir, 0o755); err != nil {
		return
	}

	if data, err := json.MarshalIndent(p.good, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(p.dataDir, "good_proxies.json"), data, 0o644)
	}

	if data, err := json.MarshalIndent(p.bad, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(p.dataDir, "bad_proxies.json"), data, 0o644)
	}
}

// ── Proxy list fetcher ────────────────────────────────────────────────────────

// fetchProxyList downloads proxies from all public sources, shuffles, and
// returns up to proxyMaxFetch entries.
func fetchProxyList() ([]goodProxy, error) {
	var all []goodProxy
	client := &http.Client{Timeout: 15 * time.Second}

	for _, src := range proxySources {
		entries, err := fetchOneProxyList(client, src.URL, src.Proto)
		if err != nil {
			// Non-fatal: skip failing sources.
			continue
		}
		all = append(all, entries...)
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("all proxy list sources failed or returned empty results")
	}

	// Shuffle for fairness.
	rand.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	if len(all) > proxyMaxFetch {
		all = all[:proxyMaxFetch]
	}
	return all, nil
}

// fetchOneProxyList downloads a single proxy list URL.
// Each non-empty, non-comment line is expected to be either:
//   - "host:port"
//   - "proto://host:port"
func fetchOneProxyList(client *http.Client, listURL, defaultProto string) ([]goodProxy, error) {
	resp, err := client.Get(listURL)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", listURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("fetch %s: HTTP %d", listURL, resp.StatusCode)
	}

	var proxies []goodProxy
	scanner := bufio.NewScanner(io.LimitReader(resp.Body, 2*1024*1024))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		addr, proto := parseProxyLine(line, defaultProto)
		if addr == "" {
			continue
		}
		proxies = append(proxies, goodProxy{
			URL:   addr,
			Proto: proto,
		})
	}
	return proxies, scanner.Err()
}

// parseProxyLine extracts the "host:port" address and protocol from a proxy
// list line. Lines may be bare "host:port" or "proto://host:port".
func parseProxyLine(line, defaultProto string) (addr, proto string) {
	if strings.Contains(line, "://") {
		u, err := url.Parse(line)
		if err != nil || u.Host == "" {
			return "", ""
		}
		return u.Host, u.Scheme
	}
	// Bare host:port — validate.
	host, port, err := net.SplitHostPort(line)
	if err != nil || host == "" || port == "" {
		return "", ""
	}
	return line, defaultProto
}

// ── HTTP transport builder ────────────────────────────────────────────────────

// buildTransport constructs an http.Transport that routes connections through
// the given proxy. proxyAddr is "host:port"; proto is "http", "https", or "socks5".
func buildTransport(proxyAddr, proto string) (http.RoundTripper, error) {
	switch proto {
	case "socks5":
		dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("socks5 dialer: %w", err)
		}
		return &http.Transport{
			Dial: dialer.Dial,
		}, nil

	case "http", "https":
		scheme := "http"
		if proto == "https" {
			scheme = "https"
		}
		proxyURL, err := url.Parse(scheme + "://" + proxyAddr)
		if err != nil {
			return nil, fmt.Errorf("parse proxy url: %w", err)
		}
		return &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}, nil

	default:
		// Treat unknown protocols as HTTP.
		proxyURL, err := url.Parse("http://" + proxyAddr)
		if err != nil {
			return nil, fmt.Errorf("parse proxy url (default): %w", err)
		}
		return &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}, nil
	}
}
