// Package realip provides real client IP extraction middleware for Mizu.
package realip

import (
	"context"
	"net"
	"strings"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Options configures the realip middleware.
type Options struct {
	// TrustedProxies is a list of trusted proxy IPs/CIDRs.
	// If empty, all proxies are trusted (not recommended for production).
	TrustedProxies []string

	// TrustedHeaders is a list of headers to check for the real IP.
	// Default: X-Forwarded-For, X-Real-IP.
	TrustedHeaders []string
}

var defaultHeaders = []string{
	"X-Forwarded-For",
	"X-Real-IP",
	"CF-Connecting-IP",
	"True-Client-IP",
}

// New creates realip middleware with defaults.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithTrustedProxies creates middleware with trusted proxy list.
func WithTrustedProxies(proxies ...string) mizu.Middleware {
	return WithOptions(Options{TrustedProxies: proxies})
}

// WithOptions creates realip middleware with options.
//
//nolint:cyclop // Real IP detection requires multiple header and proxy checks
func WithOptions(opts Options) mizu.Middleware {
	if len(opts.TrustedHeaders) == 0 {
		opts.TrustedHeaders = defaultHeaders
	}

	// Parse trusted proxies into CIDRs
	var trustedNets []*net.IPNet
	for _, p := range opts.TrustedProxies {
		if strings.Contains(p, "/") {
			_, network, err := net.ParseCIDR(p)
			if err == nil {
				trustedNets = append(trustedNets, network)
			}
		} else {
			ip := net.ParseIP(p)
			if ip != nil {
				if ip.To4() != nil {
					_, network, _ := net.ParseCIDR(p + "/32")
					trustedNets = append(trustedNets, network)
				} else {
					_, network, _ := net.ParseCIDR(p + "/128")
					trustedNets = append(trustedNets, network)
				}
			}
		}
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			remoteIP := extractIP(c.Request().RemoteAddr)

			// Check if remote IP is trusted
			trusted := len(trustedNets) == 0 || isTrusted(remoteIP, trustedNets)

			var realIP string
			if trusted {
				// Try headers in order
				for _, header := range opts.TrustedHeaders {
					value := c.Request().Header.Get(header)
					if value != "" {
						ip := extractFirstIP(value)
						if ip != "" {
							realIP = ip
							break
						}
					}
				}
			}

			if realIP == "" {
				realIP = remoteIP
			}

			// Store in context
			ctx := context.WithValue(c.Context(), contextKey{}, realIP)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			return next(c)
		}
	}
}

// FromContext extracts the real IP from context.
func FromContext(c *mizu.Ctx) string {
	if ip, ok := c.Context().Value(contextKey{}).(string); ok {
		return ip
	}
	return getClientIP(c)
}

// Get is an alias for FromContext.
func Get(c *mizu.Ctx) string {
	return FromContext(c)
}

func extractIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func extractFirstIP(header string) string {
	// X-Forwarded-For can be comma-separated
	parts := strings.Split(header, ",")
	for _, part := range parts {
		ip := strings.TrimSpace(part)
		if net.ParseIP(ip) != nil {
			return ip
		}
	}
	return ""
}

func isTrusted(ip string, networks []*net.IPNet) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	for _, network := range networks {
		if network.Contains(parsedIP) {
			return true
		}
	}
	return false
}

func getClientIP(c *mizu.Ctx) string {
	// Check X-Forwarded-For header
	if xff := c.Request().Header.Get("X-Forwarded-For"); xff != "" {
		ip := strings.TrimSpace(strings.Split(xff, ",")[0])
		if net.ParseIP(ip) != nil {
			return ip
		}
	}
	// Check X-Real-IP header
	if xr := c.Request().Header.Get("X-Real-IP"); xr != "" && net.ParseIP(xr) != nil {
		return xr
	}
	// Fallback to RemoteAddr
	return extractIP(c.Request().RemoteAddr)
}
