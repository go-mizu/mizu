// Package forwarded provides X-Forwarded-* header handling middleware for Mizu.
package forwarded

import (
	"context"
	"net"
	"strings"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Info holds forwarded information.
type Info struct {
	For      string
	Host     string
	Proto    string
	Port     string
	Prefix   string
	ClientIP net.IP
}

// Options configures the forwarded middleware.
type Options struct {
	// TrustProxy indicates whether to trust proxy headers.
	// Default: true.
	TrustProxy bool

	// TrustedProxies is a list of trusted proxy IPs/CIDRs.
	// If empty, all proxies are trusted when TrustProxy is true.
	TrustedProxies []string

	// trustedNets is parsed from TrustedProxies (internal)
	trustedNets []*net.IPNet
}

// New creates forwarded middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{TrustProxy: true})
}

// WithOptions creates forwarded middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	// Parse trusted proxies
	for _, proxy := range opts.TrustedProxies {
		if !strings.Contains(proxy, "/") {
			proxy += "/32"
		}
		_, ipNet, err := net.ParseCIDR(proxy)
		if err == nil {
			opts.trustedNets = append(opts.trustedNets, ipNet)
		}
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			r := c.Request()

			info := &Info{}

			// Get remote IP
			remoteIP := parseIP(r.RemoteAddr)

			// Check if we should trust this request
			trusted := opts.TrustProxy && isTrusted(remoteIP, opts.trustedNets)

			if trusted {
				// X-Forwarded-For
				if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
					parts := strings.Split(xff, ",")
					// Get the first (original client) IP
					if len(parts) > 0 {
						info.For = strings.TrimSpace(parts[0])
						info.ClientIP = net.ParseIP(info.For)
					}
				}

				// X-Forwarded-Host
				if xfh := r.Header.Get("X-Forwarded-Host"); xfh != "" {
					info.Host = xfh
				}

				// X-Forwarded-Proto
				if xfp := r.Header.Get("X-Forwarded-Proto"); xfp != "" {
					info.Proto = xfp
				}

				// X-Forwarded-Port
				if xfport := r.Header.Get("X-Forwarded-Port"); xfport != "" {
					info.Port = xfport
				}

				// X-Forwarded-Prefix
				if xfprefix := r.Header.Get("X-Forwarded-Prefix"); xfprefix != "" {
					info.Prefix = xfprefix
				}

				// Standard Forwarded header (RFC 7239)
				if fwd := r.Header.Get("Forwarded"); fwd != "" {
					parseForwardedHeader(fwd, info)
				}
			}

			// Fall back to remote address if no forwarded info
			if info.ClientIP == nil {
				info.ClientIP = remoteIP
				info.For = remoteIP.String()
			}

			// Store info in context
			ctx := context.WithValue(c.Context(), contextKey{}, info)
			req := r.WithContext(ctx)
			*c.Request() = *req

			return next(c)
		}
	}
}

// Get retrieves forwarded info from context.
func Get(c *mizu.Ctx) *Info {
	if info, ok := c.Context().Value(contextKey{}).(*Info); ok {
		return info
	}
	return nil
}

// FromContext is an alias for Get.
func FromContext(c *mizu.Ctx) *Info {
	return Get(c)
}

// ClientIP returns the client IP from context.
func ClientIP(c *mizu.Ctx) net.IP {
	if info := Get(c); info != nil {
		return info.ClientIP
	}
	return nil
}

// Proto returns the protocol from context.
func Proto(c *mizu.Ctx) string {
	if info := Get(c); info != nil && info.Proto != "" {
		return info.Proto
	}
	return "http"
}

// Host returns the host from context.
func Host(c *mizu.Ctx) string {
	if info := Get(c); info != nil && info.Host != "" {
		return info.Host
	}
	return ""
}

func parseIP(addr string) net.IP {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return net.ParseIP(addr)
	}
	return net.ParseIP(host)
}

func isTrusted(ip net.IP, trustedNets []*net.IPNet) bool {
	if len(trustedNets) == 0 {
		return true // Trust all if no specific nets defined
	}
	for _, ipNet := range trustedNets {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

func parseForwardedHeader(header string, info *Info) {
	// Parse RFC 7239 Forwarded header
	// Example: for=192.0.2.60;proto=http;by=203.0.113.43
	parts := strings.Split(header, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		value := strings.Trim(strings.TrimSpace(kv[1]), `"`)

		switch key {
		case "for":
			// Handle IPv6 in brackets
			if strings.HasPrefix(value, "[") {
				if idx := strings.Index(value, "]"); idx > 0 {
					value = value[1:idx]
				}
			}
			if info.For == "" {
				info.For = value
				info.ClientIP = net.ParseIP(value)
			}
		case "proto":
			if info.Proto == "" {
				info.Proto = value
			}
		case "host":
			if info.Host == "" {
				info.Host = value
			}
		}
	}
}
