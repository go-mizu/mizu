// Package ipfilter provides IP whitelist/blacklist middleware for Mizu.
package ipfilter

import (
	"net"
	"net/http"

	"github.com/go-mizu/mizu"
)

// Options configures the IP filter middleware.
type Options struct {
	// AllowList is a list of IPs/CIDRs to allow.
	AllowList []string

	// DenyList is a list of IPs/CIDRs to deny.
	DenyList []string

	// DenyByDefault denies unless in allow list.
	// Default: false.
	DenyByDefault bool

	// TrustProxy uses X-Forwarded-For header.
	// Default: false.
	TrustProxy bool

	// ErrorHandler handles denied requests.
	ErrorHandler func(c *mizu.Ctx) error
}

// Allow creates middleware allowing only listed IPs.
func Allow(ips ...string) mizu.Middleware {
	return New(Options{
		AllowList:     ips,
		DenyByDefault: true,
	})
}

// Deny creates middleware denying listed IPs.
func Deny(ips ...string) mizu.Middleware {
	return New(Options{
		DenyList: ips,
	})
}

// New creates middleware with options.
func New(opts Options) mizu.Middleware {
	allowNets := parseNetworks(opts.AllowList)
	denyNets := parseNetworks(opts.DenyList)

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			var ip string
			if opts.TrustProxy {
				ip = getClientIP(c)
			} else {
				ip = extractIP(c.Request().RemoteAddr)
			}

			parsedIP := net.ParseIP(ip)
			if parsedIP == nil {
				return handleDenied(c, opts)
			}

			// Check deny list first
			for _, network := range denyNets {
				if network.Contains(parsedIP) {
					return handleDenied(c, opts)
				}
			}

			// Check allow list
			if opts.DenyByDefault {
				allowed := false
				for _, network := range allowNets {
					if network.Contains(parsedIP) {
						allowed = true
						break
					}
				}
				if !allowed {
					return handleDenied(c, opts)
				}
			}

			return next(c)
		}
	}
}

func handleDenied(c *mizu.Ctx, opts Options) error {
	if opts.ErrorHandler != nil {
		return opts.ErrorHandler(c)
	}
	return c.Text(http.StatusForbidden, "Forbidden")
}

func parseNetworks(ips []string) []*net.IPNet {
	var networks []*net.IPNet
	for _, ip := range ips {
		if _, network, err := net.ParseCIDR(ip); err == nil {
			networks = append(networks, network)
		} else if parsed := net.ParseIP(ip); parsed != nil {
			// Single IP - convert to /32 or /128
			if parsed.To4() != nil {
				_, network, _ := net.ParseCIDR(ip + "/32")
				networks = append(networks, network)
			} else {
				_, network, _ := net.ParseCIDR(ip + "/128")
				networks = append(networks, network)
			}
		}
	}
	return networks
}

func extractIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func getClientIP(c *mizu.Ctx) string {
	if xff := c.Request().Header.Get("X-Forwarded-For"); xff != "" {
		ip := extractFirstXFF(xff)
		if ip != "" && net.ParseIP(ip) != nil {
			return ip
		}
	}
	if xr := c.Request().Header.Get("X-Real-IP"); xr != "" && net.ParseIP(xr) != nil {
		return xr
	}
	return extractIP(c.Request().RemoteAddr)
}

func extractFirstXFF(xff string) string {
	end := len(xff)
	for i := 0; i < len(xff); i++ {
		if xff[i] == ',' {
			end = i
			break
		}
	}
	ip := xff[:end]
	for len(ip) > 0 && ip[0] == ' ' {
		ip = ip[1:]
	}
	return ip
}

// Private creates middleware allowing only private network IPs.
func Private() mizu.Middleware {
	return Allow(
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"::1/128",
		"fc00::/7",
	)
}

// Localhost creates middleware allowing only localhost.
func Localhost() mizu.Middleware {
	return Allow("127.0.0.1", "::1")
}
