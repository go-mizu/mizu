package s3

import (
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

// MultiEndpointTransport distributes HTTP requests across multiple S3 endpoints.
// Two modes:
//   - "roundrobin": each request goes to the next endpoint in rotation.
//     Use for MinIO/RustFS/SeaweedFS where any node can serve any key.
//   - "rendezvous": each request goes to the endpoint with highest hash(endpoint+key).
//     Use for Herd where each node has independent storage (key affinity required).
type MultiEndpointTransport struct {
	endpoints  []string
	transports []*http.Transport
	counter    atomic.Uint64
	mode       string
}

// NewMultiEndpointTransport creates a transport that distributes requests across endpoints.
func NewMultiEndpointTransport(endpoints []string, mode string) *MultiEndpointTransport {
	t := &MultiEndpointTransport{
		endpoints:  endpoints,
		transports: make([]*http.Transport, len(endpoints)),
		mode:       mode,
	}
	for i := range endpoints {
		t.transports[i] = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          200,
			MaxIdleConnsPerHost:   200,
			MaxConnsPerHost:       200,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		}
	}
	return t
}

func (t *MultiEndpointTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var idx int
	switch t.mode {
	case "rendezvous":
		idx = t.rendezvousSelect(req.URL.Path)
	default:
		idx = int(t.counter.Add(1) % uint64(len(t.endpoints)))
	}
	req.URL.Host = t.endpoints[idx]
	req.Host = t.endpoints[idx]
	return t.transports[idx].RoundTrip(req)
}

func (t *MultiEndpointTransport) rendezvousSelect(key string) int {
	best, bestIdx := uint64(0), 0
	for i, ep := range t.endpoints {
		score := fnv1aHash(ep, key)
		if score > best {
			best = score
			bestIdx = i
		}
	}
	return bestIdx
}

// fnv1aHash computes FNV-1a hash of node+key for rendezvous hashing.
func fnv1aHash(node, key string) uint64 {
	const offset64 = 14695981039346656037
	const prime64 = 1099511628211
	h := uint64(offset64)
	for i := 0; i < len(node); i++ {
		h ^= uint64(node[i])
		h *= prime64
	}
	h ^= 0xFF
	h *= prime64
	for i := 0; i < len(key); i++ {
		h ^= uint64(key[i])
		h *= prime64
	}
	return h
}

// CloseIdleConnections closes idle connections on all underlying transports.
func (t *MultiEndpointTransport) CloseIdleConnections() {
	for _, tr := range t.transports {
		tr.CloseIdleConnections()
	}
}
