package x

// proxy_guest.go — global proxy pool integration for guest token rotation.
//
// When the direct guest token is rate limited, callers can obtain a token
// through a different IP by calling FetchGuestTokenFromPool(). Each proxy
// has its own rate-limit bucket at X's activate endpoint, multiplying the
// effective throughput for anonymous API access.

import (
	"os"
	"path/filepath"
	"sync"
)

// proxyGuestPool is the global proxy pool for guest token rotation.
// Initialized lazily on first use.
var (
	proxyGuestPool *ProxyPool
	proxyPoolOnce  sync.Once
)

// getProxyPool returns the global ProxyPool, initializing it on first call.
func getProxyPool() *ProxyPool {
	proxyPoolOnce.Do(func() {
		homeDir, _ := os.UserHomeDir()
		proxyGuestPool = NewProxyPool(filepath.Join(homeDir, "data", "x"))
	})
	return proxyGuestPool
}

// FetchGuestTokenFromPool obtains a guest token via the proxy pool.
// It returns a token that was acquired through a proxy IP different from the
// host machine, giving it a separate rate-limit bucket at X's activate endpoint.
//
// Typical usage: call this when the direct guest token is rate limited and a
// second direct fetch also fails.
func FetchGuestTokenFromPool() (string, error) {
	return getProxyPool().FetchGuestToken()
}
