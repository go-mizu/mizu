package recrawl_v3

import (
	"net"
	"sync"
	"time"
)

// rawConnPool maintains a per-host pool of reusable net.Conn connections.
// Connections are returned via Put() after a successful response body drain.
// Thread-safe. Excess connections (above maxPerHost) are closed immediately.
type rawConnPool struct {
	mu         sync.Mutex
	pools      map[string][]net.Conn
	maxPerHost int
	timeout    time.Duration
}

func newRawConnPool(maxPerHost int, timeout time.Duration) *rawConnPool {
	if maxPerHost <= 0 {
		maxPerHost = 4
	}
	return &rawConnPool{
		pools:      make(map[string][]net.Conn),
		maxPerHost: maxPerHost,
		timeout:    timeout,
	}
}

// Get returns a pooled connection for key, or (nil, false) if none available.
func (p *rawConnPool) Get(key string) (net.Conn, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	conns := p.pools[key]
	if len(conns) == 0 {
		return nil, false
	}
	c := conns[len(conns)-1]
	p.pools[key] = conns[:len(conns)-1]
	return c, true
}

// Put returns a connection to the pool. If the pool is full, the connection is closed.
func (p *rawConnPool) Put(key string, c net.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.pools[key]) >= p.maxPerHost {
		c.Close()
		return
	}
	p.pools[key] = append(p.pools[key], c)
}

// CloseAll closes every pooled connection and clears the pool.
func (p *rawConnPool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, conns := range p.pools {
		for _, c := range conns {
			c.Close()
		}
	}
	p.pools = make(map[string][]net.Conn)
}
