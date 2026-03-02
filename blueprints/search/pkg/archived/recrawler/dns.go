package recrawler

import (
	"context"
	"net"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/store"
)

// DNSResolver is deprecated. Use store.DNSResolver directly.
type DNSResolver = store.DNSResolver

// DNSProgress is deprecated. Use store.DNSProgress directly.
type DNSProgress = store.DNSProgress

// NewDNSResolver is deprecated. Use store.NewDNSResolver.
func NewDNSResolver(timeout time.Duration) *DNSResolver {
	return store.NewDNSResolver(timeout)
}

// DNSBatchPoolCapacity is deprecated. Use store.DNSBatchPoolCapacity.
func DNSBatchPoolCapacity() int {
	return store.DNSBatchPoolCapacity()
}

// DNSBatchRecommendedWorkerCap is deprecated. Use store.DNSBatchRecommendedWorkerCap.
func DNSBatchRecommendedWorkerCap() int {
	return store.DNSBatchRecommendedWorkerCap()
}

// makeResolver creates a net.Resolver that dials the given DNS server.
// Kept here for backward compatibility with verify.go.
// If addr is empty, uses the system default resolver.
func makeResolver(addr string, timeout time.Duration) *net.Resolver {
	if addr == "" {
		return &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: timeout}
				return d.DialContext(ctx, "udp", address)
			},
		}
	}
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: timeout}
			return d.DialContext(ctx, "udp", addr)
		},
	}
}
