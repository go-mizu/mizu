package mw

import (
	"net/http"
	"net/netip"
	"slices"
	"strings"

	"github.com/go-mizu/mizu/web"
)

// RealIP rewrites RemoteAddr from a forwarding header, for requests that
// arrived through one of the proxies named in trusted.
//
//	mw.RealIP(mw.Private()...)                          // a proxy on the same network
//	mw.RealIP(netip.MustParsePrefix("10.0.4.0/24"))     // the load balancer subnet
//
// After it,
// [github.com/go-mizu/mizu/web.Ctx.IP] is the client rather than the proxy, and
// so is the ip on every record [Logger] writes.
//
// The list is the whole of the security of this. X-Forwarded-For is a header,
// which means anybody who can reach the service can write whatever they like in
// it, so believing it without a list is how a rate limit gets bypassed and how
// an audit log ends up naming an address the request never came from. Passing
// nothing is allowed and does nothing, because a trusted list with no proxies in
// it trusts nothing.
//
// What it does with the list: the connection the request came in on has to be
// one of the trusted addresses, or nothing happens. Then it reads the hops right
// to left, which is the order they were added in, and takes the first one that
// is not itself trusted. That is the last address a proxy you run wrote down,
// and everything to the left of it is whatever the client sent.
//
// Forwarded is read in preference to X-Forwarded-For, since it is the one with
// an RFC. Only the for parameter is read; by, host and proto are ignored. A hop
// that is not an address, which includes RFC 7239's obfuscated identifiers and
// the literal unknown, stops the walk and leaves RemoteAddr alone, because a hop
// that cannot be checked against the list cannot be trusted past.
//
// The port on the rewritten RemoteAddr is zero. The client has one and the
// service never learns it, and a proxy's port is not it.
func RealIP(trusted ...netip.Prefix) web.Middleware {
	nets := slices.Clone(trusted)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if addr, ok := client(r, nets); ok {
				// The request the server handed over belongs to the server, so
				// the rewrite goes on a copy. It is a shallow one: nothing under
				// it is touched, only the field.
				copied := *r
				copied.RemoteAddr = netip.AddrPortFrom(addr, 0).String()
				r = &copied
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Private is the address ranges a proxy running on your own network sits in:
// loopback, the three RFC 1918 blocks, and IPv6 loopback and unique local.
//
//	mw.RealIP(mw.Private()...)
//
// It is the right list for a service behind nginx on the same host or behind an
// ingress in the same cluster, and the wrong one for anything reachable from
// outside that network, where the answer is the addresses of the proxies you
// actually run.
//
// Link local is deliberately not in it. 169.254.0.0/16 is where a cloud
// instance's metadata service lives, and a request that appears to come from
// there is more likely to be somebody who found a request forgery than a proxy.
func Private() []netip.Prefix {
	return []netip.Prefix{
		netip.MustParsePrefix("127.0.0.0/8"),
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("172.16.0.0/12"),
		netip.MustParsePrefix("192.168.0.0/16"),
		netip.MustParsePrefix("::1/128"),
		netip.MustParsePrefix("fc00::/7"),
	}
}

// client is the address a forwarding header says the request came from, for a
// request that arrived through a trusted proxy.
func client(r *http.Request, trusted []netip.Prefix) (netip.Addr, bool) {
	peer, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil || !within(peer.Addr().Unmap(), trusted) {
		return netip.Addr{}, false
	}

	hops := forwarded(r.Header.Values("Forwarded"))
	if len(hops) == 0 {
		hops = xff(r.Header.Values("X-Forwarded-For"))
	}

	for i := len(hops) - 1; i >= 0; i-- {
		if !hops[i].IsValid() {
			return netip.Addr{}, false
		}
		if !within(hops[i], trusted) {
			return hops[i], true
		}
	}

	// Every hop was a proxy of ours, so the client is not among them and the
	// connection is the best answer there is.
	return netip.Addr{}, false
}

// within reports whether a is in any of the prefixes.
func within(a netip.Addr, trusted []netip.Prefix) bool {
	return slices.ContainsFunc(trusted, func(p netip.Prefix) bool { return p.Contains(a) })
}

// xff is the hops in the X-Forwarded-For headers, left to right, with the
// invalid address standing for one that could not be read.
func xff(values []string) []netip.Addr {
	var hops []netip.Addr
	for _, v := range values {
		for hop := range strings.SplitSeq(v, ",") {
			hops = append(hops, address(hop))
		}
	}
	return hops
}

// forwarded is the same for the RFC 7239 Forwarded header.
//
// One header holds a comma separated list of elements, and one element holds a
// semicolon separated list of parameters, of which for is the only one this
// reads. An element with no for parameter contributes no hop, which is what a
// proxy that recorded only proto or host sends.
func forwarded(values []string) []netip.Addr {
	var hops []netip.Addr
	for _, v := range values {
		for elem := range strings.SplitSeq(v, ",") {
			for param := range strings.SplitSeq(elem, ";") {
				name, value, ok := strings.Cut(param, "=")
				if ok && strings.EqualFold(strings.TrimSpace(name), "for") {
					hops = append(hops, address(value))
				}
			}
		}
	}
	return hops
}

// address is the IP in one hop, which is written half a dozen ways: bare, with
// a port, quoted, or in brackets with or without a port.
//
// The invalid address comes back for anything else, and the caller treats that
// as a wall rather than as a hop to skip.
func address(s string) netip.Addr {
	s = strings.Trim(strings.TrimSpace(s), `"`)

	if a, err := netip.ParseAddr(s); err == nil {
		return a.Unmap()
	}
	if ap, err := netip.ParseAddrPort(s); err == nil {
		return ap.Addr().Unmap()
	}
	if host, _, ok := strings.Cut(strings.TrimPrefix(s, "["), "]"); ok {
		if a, err := netip.ParseAddr(host); err == nil {
			return a.Unmap()
		}
	}
	return netip.Addr{}
}
