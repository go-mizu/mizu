package serp

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// The invisible reCAPTCHA bypass works by:
// 1. GET the anchor page to extract an initial token
// 2. POST to the reload endpoint with that token to get a valid response token
// This exploits the fact that invisible reCAPTCHA validates via Google's endpoints,
// and some implementations accept tokens generated this way.

var (
	anchorTokenRe = regexp.MustCompile(`"recaptcha-token" value="([^"]+)"`)
	rrespTokenRe  = regexp.MustCompile(`"rresp","([^"]+)"`)
)

// SolveRecaptchaFree attempts to bypass invisible reCAPTCHA v2 without a paid service.
// Returns the g-recaptcha-response token, or error if bypass fails.
func SolveRecaptchaFree(siteKey, pageURL string) (string, error) {
	hc := &http.Client{Timeout: 15 * time.Second}

	// Encode origin for co parameter — must be base64 of "https://hostname:443"
	u, err := url.Parse(pageURL)
	if err != nil {
		return "", fmt.Errorf("parse pageURL: %w", err)
	}
	origin := u.Scheme + "://" + u.Hostname() + ":443"
	coParam := strings.ReplaceAll(base64.StdEncoding.EncodeToString([]byte(origin)), "=", ".")

	// Step 1: GET the anchor endpoint to get initial token
	anchorURL := fmt.Sprintf(
		"https://www.google.com/recaptcha/api2/anchor?ar=1&k=%s&co=%s&hl=en&v=QvLuXwupqtKMva7GIh5eGl3U&size=invisible",
		url.QueryEscape(siteKey),
		url.QueryEscape(coParam),
	)

	resp, err := hc.Get(anchorURL)
	if err != nil {
		return "", fmt.Errorf("anchor GET: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	match := anchorTokenRe.FindSubmatch(body)
	if len(match) < 2 {
		return "", fmt.Errorf("no recaptcha-token found in anchor response (len=%d)", len(body))
	}
	initialToken := string(match[1])

	// Step 2: POST to reload endpoint to get actual response token
	reloadURL := fmt.Sprintf(
		"https://www.google.com/recaptcha/api2/reload?k=%s",
		url.QueryEscape(siteKey),
	)

	payload := url.Values{
		"v":      {"QvLuXwupqtKMva7GIh5eGl3U"},
		"reason": {"q"},
		"c":      {initialToken},
		"k":      {siteKey},
		"co":     {coParam},
		"hl":     {"en"},
		"size":   {"invisible"},
	}

	req, _ := http.NewRequest("POST", reloadURL, strings.NewReader(payload.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err = hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("reload POST: %w", err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()

	match = rrespTokenRe.FindSubmatch(body)
	if len(match) < 2 {
		return "", fmt.Errorf("no rresp token in reload response (status=%d, len=%d)", resp.StatusCode, len(body))
	}

	return string(match[1]), nil
}

// proxySource defines a proxy list URL and what scheme to use.
type proxySource struct {
	url    string
	scheme string // "socks5", "http", "https"
}

// FetchFreeProxy fetches a working proxy from multiple free proxy lists.
// Tries SOCKS5, HTTP, and HTTPS proxies. Tests in parallel for speed.
func FetchFreeProxy() (string, error) {
	sources := []proxySource{
		{"https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/socks5.txt", "socks5"},
		{"https://raw.githubusercontent.com/proxifly/free-proxy-list/main/proxies/protocols/socks5/data.txt", "socks5"},
		{"https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/http.txt", "http"},
		{"https://raw.githubusercontent.com/proxifly/free-proxy-list/main/proxies/protocols/http/data.txt", "http"},
		{"https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/socks5.txt", "socks5"},
		{"https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/http.txt", "http"},
		{"https://raw.githubusercontent.com/hookzof/socks5_list/master/proxy.txt", "socks5"},
	}

	hc := &http.Client{Timeout: 10 * time.Second}

	type taggedProxy struct {
		addr   string
		scheme string
	}
	var allProxies []taggedProxy

	for _, src := range sources {
		resp, err := hc.Get(src.url)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				allProxies = append(allProxies, taggedProxy{line, src.scheme})
			}
		}
	}

	if len(allProxies) == 0 {
		return "", fmt.Errorf("no proxy lists reachable")
	}

	// Shuffle
	for i := len(allProxies) - 1; i > 0; i-- {
		j := int(time.Now().UnixNano()) % (i + 1)
		if j < 0 {
			j = -j
		}
		allProxies[i], allProxies[j] = allProxies[j], allProxies[i]
	}

	// Test up to 50 proxies, 10 at a time
	const batchSize = 10
	const maxTest = 50
	type result struct {
		proxy string
		ok    bool
	}
	tested := 0
	for tested < len(allProxies) && tested < maxTest {
		end := tested + batchSize
		if end > len(allProxies) {
			end = len(allProxies)
		}
		if end > maxTest {
			end = maxTest
		}

		batch := allProxies[tested:end]
		ch := make(chan result, len(batch))
		for _, tp := range batch {
			go func(p taggedProxy) {
				proxyURL := p.scheme + "://" + p.addr
				ch <- result{proxyURL, testProxy(proxyURL)}
			}(tp)
		}

		for range batch {
			r := <-ch
			if r.ok {
				return r.proxy, nil
			}
		}
		tested = end
	}
	return "", fmt.Errorf("no working proxy found in %d tested", tested)
}

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func testProxy(proxyURL string) bool {
	pURL, err := url.Parse(proxyURL)
	if err != nil {
		return false
	}
	hc := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(pURL),
		},
	}
	resp, err := hc.Get("https://httpbin.org/ip")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}
