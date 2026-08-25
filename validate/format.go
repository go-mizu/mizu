package validate

import (
	"net/netip"
	"net/url"
	"strings"
)

// The checks in this file are the ones that are not a line of Go.
//
// A generated validator writes the simple rules out rather than calling into
// this package, because utf8.RuneCountInString(v.Title) < 3 is something to
// read in a debugger and validate.IsMinLen(v.Title, 3) is something to step
// into. Nothing here is a line of Go, so everything here is a function, and
// the generated code, the reflective interpreter and the programmatic builder
// all call the same one.
//
// They take a string because that is what a form sends. A field that arrived
// as an int was already checked by the decoder that turned it into one.

// IsEmail is whether a string is an email address somebody could receive mail
// at.
//
// It is the address on its own: no display name, no angle brackets and no
// comments, since a form asking for an email address is asking for the address
// and net/mail.ParseAddress accepts a header field. The local part is the
// unquoted form, so a.b@example.com is an address and "a b"@example.com is
// refused. Quoted local parts are legal and are, in practice, a typo.
//
// The domain has to be a hostname with at least two labels and a top level
// domain of two or more letters, so user@localhost is refused here. That is
// stricter than the RFC and it is the rule a form wants: an address with no dot
// in the domain is a person who has not finished typing.
func IsEmail(s string) bool {
	at := strings.LastIndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return false
	}

	local, domain := s[:at], s[at+1:]
	if len(local) > 64 || !isLocalPart(local) {
		return false
	}
	if !IsHostname(domain) {
		return false
	}

	// A hostname may be one label and an email domain may not, and the last
	// label is a top level domain rather than a number.
	dot := strings.LastIndexByte(domain, '.')
	if dot < 0 {
		return false
	}
	tld := domain[dot+1:]
	if len(tld) < 2 {
		return false
	}
	for i := range len(tld) {
		if !isAlpha(tld[i]) {
			return false
		}
	}
	return true
}

// isLocalPart is whether the part before the @ is a dot-separated run of atoms,
// which is the unquoted form from RFC 5322.
func isLocalPart(s string) bool {
	if s == "" || s[0] == '.' || s[len(s)-1] == '.' {
		return false
	}
	prevDot := false
	for i := range len(s) {
		c := s[i]
		switch {
		case c == '.':
			if prevDot {
				return false
			}
			prevDot = true
		case isAtext(c):
			prevDot = false
		default:
			return false
		}
	}
	return true
}

// isAtext is the atext set from RFC 5322, which is what may appear in an
// unquoted local part.
func isAtext(c byte) bool {
	if isAlnum(c) {
		return true
	}
	return strings.IndexByte("!#$%&'*+-/=?^_`{|}~", c) >= 0
}

// IsURL is whether a string is an absolute http or https URL with a host
// somebody could reach.
//
// The scheme is checked because url is the rule a form uses for a link
// somebody will click, and javascript:alert(1) parses. A rule that takes any
// scheme is [IsURI].
func IsURL(s string) bool {
	u, ok := absolute(s)
	if !ok {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return isHost(u.Host)
}

// IsURI is whether a string is an absolute URI of any scheme, so mailto: and
// urn: pass and a bare path does not.
//
// The host, if the scheme has one, is not checked. urn:isbn:0451450523 has
// nothing to check and mailto:a@b has an address rather than a host.
func IsURI(s string) bool {
	_, ok := absolute(s)
	return ok
}

// absolute parses and requires a scheme, and refuses the control characters
// and spaces that net/url is willing to keep.
func absolute(s string) (*url.URL, bool) {
	if s == "" || strings.ContainsFunc(s, func(r rune) bool { return r <= ' ' || r == 0x7f }) {
		return nil, false
	}
	u, err := url.Parse(s)
	if err != nil || !u.IsAbs() {
		return nil, false
	}
	return u, true
}

// isHost is whether a URL's host is a name or an address, with an optional
// port, which is what the authority of an http URL is allowed to hold.
func isHost(host string) bool {
	if after, ok := strings.CutPrefix(host, "["); ok {
		end := strings.IndexByte(after, ']')
		if end < 0 {
			return false
		}
		if _, ok := addr(after[:end]); !ok {
			return false
		}
		return after[end+1:] == "" || isPortSuffix(after[end+1:])
	}
	if colon := strings.LastIndexByte(host, ':'); colon >= 0 {
		if !isPortSuffix(host[colon:]) {
			return false
		}
		host = host[:colon]
	}
	if _, ok := addr(host); ok {
		return true
	}
	return IsHostname(host)
}

// isPortSuffix is a colon and a port, which is how a port arrives still
// attached to the host net/url handed over.
func isPortSuffix(s string) bool {
	return strings.HasPrefix(s, ":") && IsPort(s[1:])
}

// IsHostname is whether a string is a hostname in the RFC 1123 form: labels of
// letters, digits and hyphens, each 1 to 63 long and neither starting nor
// ending in a hyphen, joined by dots, 253 or fewer in total.
//
// One label is a hostname, so localhost passes. A trailing dot is the root and
// is refused, because a form asking for a hostname is not asking for a DNS
// name to look up.
func IsHostname(s string) bool {
	if s == "" || len(s) > 253 {
		return false
	}
	for label := range strings.SplitSeq(s, ".") {
		if !isLabel(label) {
			return false
		}
	}
	return true
}

func isLabel(s string) bool {
	if s == "" || len(s) > 63 || s[0] == '-' || s[len(s)-1] == '-' {
		return false
	}
	for i := range len(s) {
		if c := s[i]; !isAlnum(c) && c != '-' {
			return false
		}
	}
	return true
}

// IsIP is whether a string is an IP address, of either family.
func IsIP(s string) bool {
	_, ok := addr(s)
	return ok
}

// IsIPv4 is whether a string is an IPv4 address written as one.
//
// ::ffff:192.0.2.1 is an IPv6 address that holds an IPv4 one and it is refused
// here, because a field checked with ipv4 is a field something is going to put
// in a four byte slot.
func IsIPv4(s string) bool {
	a, ok := addr(s)
	return ok && a.Is4()
}

// IsIPv6 is whether a string is an IPv6 address, including the forms that hold
// an IPv4 address inside them.
func IsIPv6(s string) bool {
	a, ok := addr(s)
	return ok && !a.Is4()
}

// addr parses without a zone, since fe80::1%eth0 is an address on one machine
// and a string somewhere else.
func addr(s string) (netip.Addr, bool) {
	a, err := netip.ParseAddr(s)
	if err != nil || a.Zone() != "" {
		return netip.Addr{}, false
	}
	return a, true
}

// IsCIDR is whether a string is an address and a prefix length, such as
// 10.0.0.0/8 or 2001:db8::/32.
//
// The address does not have to be the first in the block, so 10.0.0.7/8 passes.
// That is what a person types when they mean the machine and its network, and
// refusing it means explaining prefix arithmetic in a form error. A zone is
// refused, which netip.ParsePrefix already does.
func IsCIDR(s string) bool {
	_, err := netip.ParsePrefix(s)
	return err == nil
}

// IsMAC is whether a string is a hardware address, in either of the two forms
// that are written down.
//
// Six or eight groups of two hex digits separated by colons or hyphens, which
// is EUI-48 and EUI-64, or three or four groups of four hex digits separated by
// dots, which is how the same addresses are written on network equipment. The
// separator has to be the same all the way through.
func IsMAC(s string) bool {
	switch {
	case strings.Count(s, ".") > 0:
		return isMACGroups(s, '.', 4, 3, 4)
	case strings.Count(s, "-") > 0:
		return isMACGroups(s, '-', 2, 6, 8)
	default:
		return isMACGroups(s, ':', 2, 6, 8)
	}
}

// isMACGroups is width hex digits per group, sep between them, and either of
// two group counts.
func isMACGroups(s string, sep byte, width, a, b int) bool {
	n := strings.Count(s, string(sep)) + 1
	if n != a && n != b {
		return false
	}
	if len(s) != n*width+(n-1) {
		return false
	}
	for i := range len(s) {
		if (i+1)%(width+1) == 0 {
			if s[i] != sep {
				return false
			}
			continue
		}
		if !isHex(s[i]) {
			return false
		}
	}
	return true
}

// IsPort is whether a string is a port a service can listen on or a client can
// reach, which is 1 to 65535 written in decimal with no leading zero.
//
// Zero is refused. It is a real value and it means "give me whichever one is
// free", which is not a thing a form is asking for.
func IsPort(s string) bool {
	if s == "" || len(s) > 5 || (s[0] == '0' && len(s) > 1) {
		return false
	}
	var n int
	for i := range len(s) {
		c := s[i]
		if !isDigit(c) {
			return false
		}
		n = n*10 + int(c-'0')
	}
	return n >= 1 && n <= 65535
}

// IsUUID is whether a string is a UUID in the canonical form: 32 hex digits in
// five groups of 8, 4, 4, 4 and 12, joined by hyphens. Either case is
// accepted and neither braces nor a urn: prefix is.
//
// No version or variant is checked, because a rule that refuses a nil UUID or a
// version nobody has heard of yet is a rule that will be wrong later.
func IsUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := range 36 {
		switch i {
		case 8, 13, 18, 23:
			if s[i] != '-' {
				return false
			}
		default:
			if !isHex(s[i]) {
				return false
			}
		}
	}
	return true
}

// crockford is the base32 alphabet a ULID is written in, which leaves out I, L,
// O and U so that a person reading one aloud cannot produce a different one.
const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// IsULID is whether a string is a ULID: 26 characters of Crockford base32,
// where the first is 7 or lower so that the timestamp fits the 48 bits it has.
//
// Lower case is accepted, since the alphabet has no pair that differ only in
// case and a ULID read off a screen is often retyped in whichever case was
// easier.
func IsULID(s string) bool {
	if len(s) != 26 || strings.IndexByte(crockford, upper(s[0])) > 7 {
		return false
	}
	for i := range 26 {
		if strings.IndexByte(crockford, upper(s[i])) < 0 {
			return false
		}
	}
	return true
}

// IsE164 is whether a string is a telephone number in international format: a
// plus, a country code that does not start with zero, and 15 digits at most in
// total.
//
// Nothing here says the number is in service or that the country code exists.
// It says the number is written the way a gateway takes it, which is what a
// field can check without asking somebody else.
func IsE164(s string) bool {
	if len(s) < 3 || len(s) > 16 || s[0] != '+' || s[1] == '0' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isDigit(s[i]) {
			return false
		}
	}
	return true
}

// formats is every rule in this file, by the name a tag spells it.
//
// The reflective interpreter reads this rather than a switch, and
// TestEveryFormatHasAMessage walks it, so a rule added here without a sentence
// to go with it fails a test rather than telling somebody their address is not
// valid.
var formats = map[string]func(string) bool{
	"cidr":     IsCIDR,
	"e164":     IsE164,
	"email":    IsEmail,
	"hostname": IsHostname,
	"ip":       IsIP,
	"ipv4":     IsIPv4,
	"ipv6":     IsIPv6,
	"mac":      IsMAC,
	"port":     IsPort,
	"ulid":     IsULID,
	"uri":      IsURI,
	"url":      IsURL,
	"uuid":     IsUUID,
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isAlpha(c byte) bool { return c|0x20 >= 'a' && c|0x20 <= 'z' }
func isAlnum(c byte) bool { return isDigit(c) || isAlpha(c) }

func isHex(c byte) bool {
	return isDigit(c) || (c|0x20 >= 'a' && c|0x20 <= 'f')
}

// upper is ASCII upper case, which is all the alphabets in this file use.
func upper(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - 0x20
	}
	return c
}
