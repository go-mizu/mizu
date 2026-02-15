package dcrawler

import (
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// maxProxyBodySize is the maximum HTML body size the proxy will serve to Lightpanda.
// Pages larger than this overwhelm Lightpanda's parser and cause crashes.
const maxProxyBodySize = 2 * 1024 * 1024 // 2 MB

const chromeUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// polyfillScript is injected at the top of <head> in HTML responses.
// It overrides navigator properties, mocks window.chrome, and stubs missing APIs
// so that bot detection scripts (Cloudflare Turnstile, DataDome, etc.) see a Chrome-like environment.
const polyfillScript = `<script>
Object.defineProperty(navigator,'userAgent',{get:()=>'` + chromeUA + `',configurable:true});
Object.defineProperty(navigator,'vendor',{get:()=>'Google Inc.',configurable:true});
Object.defineProperty(navigator,'appVersion',{get:()=>'5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36',configurable:true});
Object.defineProperty(navigator,'platform',{get:()=>'MacIntel',configurable:true});
Object.defineProperty(navigator,'deviceMemory',{get:()=>8,configurable:true});
Object.defineProperty(navigator,'hardwareConcurrency',{get:()=>10,configurable:true});
Object.defineProperty(navigator,'maxTouchPoints',{get:()=>0,configurable:true});
Object.defineProperty(navigator,'language',{get:()=>'en-US',configurable:true});
Object.defineProperty(navigator,'languages',{get:()=>['en-US','en'],configurable:true});
try{delete navigator.__proto__.webdriver}catch(e){}
Object.defineProperty(navigator,'webdriver',{get:()=>false,configurable:true});
window.chrome={app:{isInstalled:false,InstallState:{DISABLED:'disabled',INSTALLED:'installed',NOT_INSTALLED:'not_installed'}},runtime:{id:undefined,connect:function(){},sendMessage:function(){},OnInstalledReason:{},OnRestartRequiredReason:{},PlatformArch:{},PlatformOs:{}},csi:function(){return{}},loadTimes:function(){return{}}};
if(!performance.timing){var _n=Date.now();Object.defineProperty(performance,'timing',{get:()=>({navigationStart:_n-500,fetchStart:_n-450,domainLookupStart:_n-400,domainLookupEnd:_n-380,connectStart:_n-380,connectEnd:_n-350,secureConnectionStart:_n-370,requestStart:_n-350,responseStart:_n-200,responseEnd:_n-100,domLoading:_n-90,domInteractive:_n-50,domContentLoadedEventStart:_n-40,domContentLoadedEventEnd:_n-35,domComplete:_n-10,loadEventStart:_n-5,loadEventEnd:_n,unloadEventStart:0,unloadEventEnd:0,redirectStart:0,redirectEnd:0})})}
if(typeof Notification==='undefined'){window.Notification={permission:'default',requestPermission:()=>Promise.resolve('default')}}
try{Object.defineProperty(navigator,'plugins',{get:()=>{var p=[{name:'Chrome PDF Plugin',filename:'internal-pdf-viewer',description:'PDF',length:1},{name:'Chrome PDF Viewer',filename:'mhjfbmdgcfjbbpaeojofohoefgiehjai',description:'',length:1},{name:'Native Client',filename:'internal-nacl-plugin',description:'',length:0}];p.item=i=>p[i];p.namedItem=n=>p.find(x=>x.name===n);p.refresh=()=>{};return p}})}catch(e){}
if(!navigator.permissions){navigator.permissions={query:p=>Promise.resolve({state:p.name==='notifications'?'prompt':'granted',onchange:null})}}
</script>`

// stealthProxy is a local HTTP proxy that makes requests with Chrome headers
// and injects polyfill scripts into HTML responses. This allows Lightpanda
// to load CF-protected pages by ensuring the HTTP-level fingerprint looks like Chrome
// and the JS environment has Chrome-like APIs available before page scripts run.
type stealthProxy struct {
	port     int
	listener net.Listener
	client   *http.Client
}

func newStealthProxy() (*stealthProxy, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 0 {
				origHost := via[0].URL.Hostname()
				newHost := req.URL.Hostname()
				if newHost != origHost {
					// Cross-domain redirect — stop following.
					return http.ErrUseLastResponse
				}
			}
			setChromeHeaders(req)
			return nil
		},
	}

	// Listen on a random available port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("stealth proxy listen: %w", err)
	}

	sp := &stealthProxy{
		port:     ln.Addr().(*net.TCPAddr).Port,
		listener: ln,
		client:   client,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/proxy", sp.handleProxy)
	go http.Serve(ln, mux) //nolint:errcheck

	return sp, nil
}

func (sp *stealthProxy) close() {
	if sp.listener != nil {
		sp.listener.Close()
	}
}

// rewriteURL converts a target URL to a proxy URL.
// The target URL is placed raw after "url=" — we use raw query string
// parsing in handleProxy to avoid double-encoding issues with browsers.
func (sp *stealthProxy) rewriteURL(targetURL string) string {
	return fmt.Sprintf("http://127.0.0.1:%d/proxy?url=%s", sp.port, targetURL)
}

// extractRealURL extracts the original target URL from a proxy URL.
func (sp *stealthProxy) extractRealURL(proxyURL string) string {
	if idx := strings.Index(proxyURL, "/proxy?url="); idx >= 0 {
		return proxyURL[idx+len("/proxy?url="):]
	}
	return proxyURL
}

func (sp *stealthProxy) handleProxy(w http.ResponseWriter, r *http.Request) {
	// Extract target URL from raw query string to avoid double-encoding.
	// Format: /proxy?url=https://example.com/path?q=foo&bar=1
	// Everything after "url=" is the target URL (including its own query params).
	rawQuery := r.URL.RawQuery
	if !strings.HasPrefix(rawQuery, "url=") {
		http.Error(w, "missing url param", http.StatusBadRequest)
		return
	}
	targetURL := rawQuery[4:]
	if targetURL == "" {
		http.Error(w, "missing url param", http.StatusBadRequest)
		return
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setChromeHeaders(req)

	resp, err := sp.client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Handle cross-domain redirects: return a minimal HTML page with the redirect
	// target as a link, instead of serving content from the foreign domain
	// (which can crash Lightpanda due to heavy SPAs like chatgpt.com).
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		loc := resp.Header.Get("Location")
		if loc != "" {
			body := fmt.Sprintf(`<html><head><title>Redirect</title></head><body><a href="%s">%s</a></body></html>`, loc, loc)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Content-Length", fmt.Sprint(len(body)))
			w.WriteHeader(200)
			w.Write([]byte(body)) //nolint:errcheck
			return
		}
	}

	// Check Content-Type: only serve HTML to Lightpanda.
	// Non-HTML content (XML, JSON, binary, JS, CSS) can cause Lightpanda to
	// hang on page.HTML() or crash on complex JS files.
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		// Return a minimal HTML page so Lightpanda doesn't choke.
		body := fmt.Sprintf(`<html><head><title>Non-HTML</title></head><body>Content-Type: %s</body></html>`, contentType)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
		w.WriteHeader(200)
		w.Write([]byte(body)) //nolint:errcheck
		return
	}

	// Also check the final URL for cross-domain redirect that returned 200
	// (some servers redirect via meta-refresh or JS, but Go follows HTTP redirects)
	finalHost := resp.Request.URL.Hostname()
	origParsed, _ := url.Parse(targetURL)
	if origParsed != nil && finalHost != origParsed.Hostname() {
		// Ended up on a different domain — return redirect page
		loc := resp.Request.URL.String()
		body := fmt.Sprintf(`<html><head><title>Redirect</title></head><body><a href="%s">%s</a></body></html>`, loc, loc)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
		w.WriteHeader(200)
		w.Write([]byte(body)) //nolint:errcheck
		return
	}

	// Read and decompress body (with size limit)
	reader := io.LimitReader(resp.Body, maxProxyBodySize+1)
	var body []byte
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gr, gerr := gzip.NewReader(reader)
		if gerr != nil {
			body, _ = io.ReadAll(reader)
		} else {
			body, _ = io.ReadAll(io.LimitReader(gr, maxProxyBodySize+1))
			gr.Close()
		}
	} else {
		body, _ = io.ReadAll(reader)
	}

	// Oversized pages: return a minimal HTML page instead of truncated HTML.
	// Truncated HTML has unclosed <script> tags that cause Lightpanda to hang.
	if len(body) > maxProxyBodySize {
		body = []byte(fmt.Sprintf(`<html><head><title>Large Page</title></head><body>Page size %d exceeds limit</body></html>`, len(body)))
	} else {
		// Strip third-party analytics/tracking scripts to reduce Lightpanda memory pressure.
		// Keep first-party scripts intact (needed for SPA rendering on sites like openai.com).
		body = stripTrackingScripts(body)
	}

	// Inject polyfills into HTML
	body = injectPolyfills(body)

	// Copy response headers (skip encoding/length since we decompressed)
	for k, v := range resp.Header {
		switch k {
		case "Content-Encoding", "Content-Length", "Transfer-Encoding":
			continue
		}
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.Header().Set("Content-Length", fmt.Sprint(len(body)))
	w.WriteHeader(resp.StatusCode)
	w.Write(body) //nolint:errcheck
}

// injectPolyfills inserts the polyfill <script> tag after the <head> tag.
func injectPolyfills(body []byte) []byte {
	html := string(body)
	lower := strings.ToLower(html)

	// Try <head> tag first
	if idx := strings.Index(lower, "<head>"); idx >= 0 {
		return []byte(html[:idx+6] + polyfillScript + html[idx+6:])
	}
	// Try <head with attributes
	if idx := strings.Index(lower, "<head"); idx >= 0 {
		closeIdx := strings.Index(html[idx:], ">")
		if closeIdx >= 0 {
			insertAt := idx + closeIdx + 1
			return []byte(html[:insertAt] + polyfillScript + html[insertAt:])
		}
	}
	// Fallback: prepend
	return append([]byte(polyfillScript), body...)
}

// trackingDomains are third-party analytics/tracking script sources that
// add memory pressure without contributing crawlable content.
var trackingDomains = []string{
	"googletagmanager.com",
	"google-analytics.com",
	"googleads.",
	"googlesyndication.com",
	"facebook.net",
	"fbevents.js",
	"connect.facebook",
	"analytics.",
	"hotjar.com",
	"hubspot.com",
	"hsforms.net",
	"hs-scripts.com",
	"hs-analytics.net",
	"intellimize.co",
	"datadome.",
	"sentry.io",
	"segment.com",
	"segment.io",
	"cdn.mxpnl.com",
	"mixpanel.com",
	"amplitude.com",
	"clarity.ms",
	"cloudflareinsights.com",
	"newrelic.com",
	"nr-data.net",
	"intercom.io",
	"intercomcdn.com",
	"crisp.chat",
	"zendesk.com",
	"drift.com",
	"tawk.to",
	"doubleclick.net",
}

// scriptTagRe matches <script ...>...</script> tags.
var scriptTagRe = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`)

// stripTrackingScripts removes only third-party analytics/tracking <script> tags.
// First-party scripts are preserved for SPA rendering (e.g., openai.com needs
// React/Next.js to render content and links).
func stripTrackingScripts(body []byte) []byte {
	return scriptTagRe.ReplaceAllFunc(body, func(match []byte) []byte {
		lower := strings.ToLower(string(match))
		for _, domain := range trackingDomains {
			if strings.Contains(lower, domain) {
				return nil // Remove this script tag
			}
		}
		return match // Keep non-tracking scripts
	})
}

// setChromeHeaders sets HTTP headers that match a real Chrome 131 browser.
func setChromeHeaders(req *http.Request) {
	req.Header.Set("User-Agent", chromeUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Sec-CH-UA", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
	req.Header.Set("Sec-CH-UA-Mobile", "?0")
	req.Header.Set("Sec-CH-UA-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}
