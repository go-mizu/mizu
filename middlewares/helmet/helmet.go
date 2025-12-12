// Package helmet provides security headers middleware for Mizu.
package helmet

import (
	"fmt"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the helmet middleware.
type Options struct {
	// ContentSecurityPolicy sets the Content-Security-Policy header.
	ContentSecurityPolicy string

	// XFrameOptions sets the X-Frame-Options header.
	// Common values: "DENY", "SAMEORIGIN".
	XFrameOptions string

	// XContentTypeOptions enables X-Content-Type-Options: nosniff.
	XContentTypeOptions bool

	// ReferrerPolicy sets the Referrer-Policy header.
	ReferrerPolicy string

	// StrictTransportSecurity configures HSTS.
	StrictTransportSecurity *HSTSOptions

	// PermissionsPolicy sets the Permissions-Policy header.
	PermissionsPolicy string

	// CrossOriginOpenerPolicy sets the Cross-Origin-Opener-Policy header.
	CrossOriginOpenerPolicy string

	// CrossOriginEmbedderPolicy sets the Cross-Origin-Embedder-Policy header.
	CrossOriginEmbedderPolicy string

	// CrossOriginResourcePolicy sets the Cross-Origin-Resource-Policy header.
	CrossOriginResourcePolicy string

	// OriginAgentCluster enables Origin-Agent-Cluster: ?1.
	OriginAgentCluster bool

	// XDNSPrefetchControl sets X-DNS-Prefetch-Control.
	// nil means don't set, true = "on", false = "off".
	XDNSPrefetchControl *bool

	// XDownloadOptions enables X-Download-Options: noopen.
	XDownloadOptions bool

	// XPermittedCrossDomainPolicies sets X-Permitted-Cross-Domain-Policies header.
	XPermittedCrossDomainPolicies string
}

// HSTSOptions configures HTTP Strict Transport Security.
type HSTSOptions struct {
	// MaxAge is the time browsers should remember HTTPS-only.
	MaxAge time.Duration

	// IncludeSubDomains includes subdomains in HSTS.
	IncludeSubDomains bool

	// Preload enables HSTS preload list eligibility.
	Preload bool
}

// Default creates helmet middleware with recommended security headers.
func Default() mizu.Middleware {
	off := false
	return New(Options{
		XContentTypeOptions:           true,
		XFrameOptions:                 "SAMEORIGIN",
		XDNSPrefetchControl:           &off,
		XDownloadOptions:              true,
		XPermittedCrossDomainPolicies: "none",
		ReferrerPolicy:                "strict-origin-when-cross-origin",
		CrossOriginOpenerPolicy:       "same-origin",
		CrossOriginResourcePolicy:     "same-origin",
		OriginAgentCluster:            true,
	})
}

// New creates helmet middleware with custom options.
func New(opts Options) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			h := c.Header()

			if opts.ContentSecurityPolicy != "" {
				h.Set("Content-Security-Policy", opts.ContentSecurityPolicy)
			}

			if opts.XFrameOptions != "" {
				h.Set("X-Frame-Options", opts.XFrameOptions)
			}

			if opts.XContentTypeOptions {
				h.Set("X-Content-Type-Options", "nosniff")
			}

			if opts.ReferrerPolicy != "" {
				h.Set("Referrer-Policy", opts.ReferrerPolicy)
			}

			if opts.StrictTransportSecurity != nil {
				h.Set("Strict-Transport-Security", formatHSTS(opts.StrictTransportSecurity))
			}

			if opts.PermissionsPolicy != "" {
				h.Set("Permissions-Policy", opts.PermissionsPolicy)
			}

			if opts.CrossOriginOpenerPolicy != "" {
				h.Set("Cross-Origin-Opener-Policy", opts.CrossOriginOpenerPolicy)
			}

			if opts.CrossOriginEmbedderPolicy != "" {
				h.Set("Cross-Origin-Embedder-Policy", opts.CrossOriginEmbedderPolicy)
			}

			if opts.CrossOriginResourcePolicy != "" {
				h.Set("Cross-Origin-Resource-Policy", opts.CrossOriginResourcePolicy)
			}

			if opts.OriginAgentCluster {
				h.Set("Origin-Agent-Cluster", "?1")
			}

			if opts.XDNSPrefetchControl != nil {
				if *opts.XDNSPrefetchControl {
					h.Set("X-DNS-Prefetch-Control", "on")
				} else {
					h.Set("X-DNS-Prefetch-Control", "off")
				}
			}

			if opts.XDownloadOptions {
				h.Set("X-Download-Options", "noopen")
			}

			if opts.XPermittedCrossDomainPolicies != "" {
				h.Set("X-Permitted-Cross-Domain-Policies", opts.XPermittedCrossDomainPolicies)
			}

			return next(c)
		}
	}
}

// ContentSecurityPolicy sets the Content-Security-Policy header.
func ContentSecurityPolicy(policy string) mizu.Middleware {
	return New(Options{ContentSecurityPolicy: policy})
}

// XFrameOptions sets the X-Frame-Options header.
func XFrameOptions(value string) mizu.Middleware {
	return New(Options{XFrameOptions: value})
}

// XContentTypeOptions sets X-Content-Type-Options: nosniff.
func XContentTypeOptions() mizu.Middleware {
	return New(Options{XContentTypeOptions: true})
}

// ReferrerPolicy sets the Referrer-Policy header.
func ReferrerPolicy(policy string) mizu.Middleware {
	return New(Options{ReferrerPolicy: policy})
}

// StrictTransportSecurity sets the Strict-Transport-Security header.
func StrictTransportSecurity(maxAge time.Duration, includeSubDomains, preload bool) mizu.Middleware {
	return New(Options{
		StrictTransportSecurity: &HSTSOptions{
			MaxAge:            maxAge,
			IncludeSubDomains: includeSubDomains,
			Preload:           preload,
		},
	})
}

// PermissionsPolicy sets the Permissions-Policy header.
func PermissionsPolicy(policy string) mizu.Middleware {
	return New(Options{PermissionsPolicy: policy})
}

// CrossOriginOpenerPolicy sets the Cross-Origin-Opener-Policy header.
func CrossOriginOpenerPolicy(policy string) mizu.Middleware {
	return New(Options{CrossOriginOpenerPolicy: policy})
}

// CrossOriginEmbedderPolicy sets the Cross-Origin-Embedder-Policy header.
func CrossOriginEmbedderPolicy(policy string) mizu.Middleware {
	return New(Options{CrossOriginEmbedderPolicy: policy})
}

// CrossOriginResourcePolicy sets the Cross-Origin-Resource-Policy header.
func CrossOriginResourcePolicy(policy string) mizu.Middleware {
	return New(Options{CrossOriginResourcePolicy: policy})
}

// OriginAgentCluster sets the Origin-Agent-Cluster header.
func OriginAgentCluster() mizu.Middleware {
	return New(Options{OriginAgentCluster: true})
}

// XDNSPrefetchControl sets the X-DNS-Prefetch-Control header.
func XDNSPrefetchControl(on bool) mizu.Middleware {
	return New(Options{XDNSPrefetchControl: &on})
}

// XDownloadOptions sets the X-Download-Options header.
func XDownloadOptions() mizu.Middleware {
	return New(Options{XDownloadOptions: true})
}

// XPermittedCrossDomainPolicies sets the X-Permitted-Cross-Domain-Policies header.
func XPermittedCrossDomainPolicies(policy string) mizu.Middleware {
	return New(Options{XPermittedCrossDomainPolicies: policy})
}

func formatHSTS(opts *HSTSOptions) string {
	maxAge := int(opts.MaxAge.Seconds())
	result := fmt.Sprintf("max-age=%d", maxAge)
	if opts.IncludeSubDomains {
		result += "; includeSubDomains"
	}
	if opts.Preload {
		result += "; preload"
	}
	return result
}
