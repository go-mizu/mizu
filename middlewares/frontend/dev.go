package frontend

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
)

// newDevProxy creates a development proxy middleware.
func newDevProxy(opts Options) mizu.Middleware {
	target, err := url.Parse(opts.DevServer)
	if err != nil {
		panic(fmt.Sprintf("frontend: invalid dev server URL: %v", err))
	}

	proxy := &devProxy{
		target:         target,
		timeout:        opts.DevServerTimeout,
		proxyWebSocket: opts.ProxyWebSocket,
		ignorePaths:    opts.IgnorePaths,
		prefix:         opts.Prefix,
	}

	proxy.httpProxy = &httputil.ReverseProxy{
		Director:       proxy.director,
		ModifyResponse: proxy.modifyResponse,
		ErrorHandler:   proxy.errorHandler,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   opts.DevServerTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	return proxy.middleware()
}

type devProxy struct {
	target         *url.URL
	httpProxy      *httputil.ReverseProxy
	timeout        time.Duration
	proxyWebSocket bool
	ignorePaths    []string
	prefix         string
}

func (p *devProxy) middleware() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			path := c.Request().URL.Path

			// Check if path should be ignored
			for _, ignorePath := range p.ignorePaths {
				if strings.HasPrefix(path, ignorePath) {
					return next(c)
				}
			}

			// Handle WebSocket upgrade for HMR
			if p.proxyWebSocket && isWebSocketRequest(c.Request()) {
				return p.proxyWebSocket_(c)
			}

			// Proxy HTTP request
			p.httpProxy.ServeHTTP(c.Writer(), c.Request())
			return nil
		}
	}
}

func (p *devProxy) director(req *http.Request) {
	req.URL.Scheme = p.target.Scheme
	req.URL.Host = p.target.Host
	req.Host = p.target.Host

	// Strip prefix if configured
	if p.prefix != "" && strings.HasPrefix(req.URL.Path, p.prefix) {
		req.URL.Path = strings.TrimPrefix(req.URL.Path, p.prefix)
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
	}

	// Preserve original headers for HMR
	if _, ok := req.Header["User-Agent"]; !ok {
		req.Header.Set("User-Agent", "")
	}
}

func (p *devProxy) modifyResponse(resp *http.Response) error {
	// Allow cross-origin for dev server
	resp.Header.Set("Access-Control-Allow-Origin", "*")
	return nil
}

func (p *devProxy) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadGateway)
	_, _ = w.Write(devErrorPage(err, p.target.String()))
}

// isWebSocketRequest checks if the request is a WebSocket upgrade.
func isWebSocketRequest(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

// proxyWebSocket_ proxies WebSocket connections for HMR.
func (p *devProxy) proxyWebSocket_(c *mizu.Ctx) error {
	// Get the client connection
	hijacker, ok := c.Writer().(http.Hijacker)
	if !ok {
		return c.Text(http.StatusInternalServerError, "WebSocket not supported")
	}

	clientConn, clientBuf, err := hijacker.Hijack()
	if err != nil {
		return fmt.Errorf("hijack failed: %w", err)
	}
	defer func() { _ = clientConn.Close() }()

	// Connect to dev server
	serverURL := *p.target
	if p.target.Scheme == "https" {
		serverURL.Scheme = "wss"
	} else {
		serverURL.Scheme = "ws"
	}
	serverURL.Path = c.Request().URL.Path
	serverURL.RawQuery = c.Request().URL.RawQuery

	// Dial server
	serverConn, err := net.DialTimeout("tcp", p.target.Host, p.timeout)
	if err != nil {
		return fmt.Errorf("dial server failed: %w", err)
	}
	defer func() { _ = serverConn.Close() }()

	// Forward the original upgrade request to the server
	if err := c.Request().Write(serverConn); err != nil {
		return fmt.Errorf("write request failed: %w", err)
	}

	// Read server response and forward to client
	serverBuf := bufio.NewReadWriter(bufio.NewReader(serverConn), bufio.NewWriter(serverConn))
	resp, err := http.ReadResponse(serverBuf.Reader, c.Request())
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}

	if err := resp.Write(clientConn); err != nil {
		return fmt.Errorf("write response failed: %w", err)
	}

	// Bidirectional copy
	done := make(chan struct{}, 2)

	go func() {
		_, _ = io.Copy(serverConn, clientBuf)
		done <- struct{}{}
	}()

	go func() {
		_, _ = io.Copy(clientConn, serverBuf)
		done <- struct{}{}
	}()

	<-done
	return nil
}

// devErrorPage generates an error page for dev server connection failures.
func devErrorPage(err error, target string) []byte {
	return []byte(fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dev Server Error</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: #1a1a2e;
            color: #eaeaea;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            max-width: 600px;
            width: 100%%;
        }
        .error-card {
            background: #16213e;
            border: 1px solid #e94560;
            border-radius: 12px;
            padding: 32px;
            box-shadow: 0 4px 24px rgba(233, 69, 96, 0.2);
        }
        .icon {
            font-size: 48px;
            margin-bottom: 16px;
        }
        h1 {
            color: #e94560;
            font-size: 24px;
            margin-bottom: 16px;
        }
        .target {
            background: #0f0f23;
            padding: 12px 16px;
            border-radius: 8px;
            font-family: 'SF Mono', Monaco, 'Courier New', monospace;
            font-size: 14px;
            color: #00d9ff;
            margin: 16px 0;
            word-break: break-all;
        }
        .hint {
            color: #8b8b8b;
            margin-top: 16px;
            line-height: 1.6;
        }
        .hint code {
            background: #0f0f23;
            padding: 2px 8px;
            border-radius: 4px;
            color: #00ff88;
        }
        .retry-bar {
            margin-top: 24px;
            background: #0f0f23;
            height: 4px;
            border-radius: 2px;
            overflow: hidden;
        }
        .retry-progress {
            height: 100%%;
            background: linear-gradient(90deg, #e94560, #00d9ff);
            width: 0%%;
            animation: progress 2s linear infinite;
        }
        @keyframes progress {
            0%% { width: 0%%; }
            100%% { width: 100%%; }
        }
        .error-detail {
            margin-top: 16px;
            padding: 12px;
            background: rgba(233, 69, 96, 0.1);
            border-radius: 8px;
            font-size: 13px;
            color: #ff8a8a;
            font-family: 'SF Mono', Monaco, monospace;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="error-card">
            <div class="icon">&#x1F6A8;</div>
            <h1>Unable to connect to dev server</h1>
            <div class="target">%s</div>
            <p class="hint">
                Make sure your frontend dev server is running.<br><br>
                For Vite: <code>npm run dev</code><br>
                For Next.js: <code>npm run dev</code><br>
                For Vue CLI: <code>npm run serve</code>
            </p>
            <div class="error-detail">%s</div>
            <div class="retry-bar">
                <div class="retry-progress"></div>
            </div>
        </div>
    </div>
    <script>
        // Auto-retry every 2 seconds
        setTimeout(() => location.reload(), 2000);
    </script>
</body>
</html>`, target, err.Error()))
}
