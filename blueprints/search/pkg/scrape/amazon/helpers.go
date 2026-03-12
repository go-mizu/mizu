package amazon

import (
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

// parsePrice strips currency symbols and commas then parses as float64.
func parsePrice(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// parseFloatStr parses a float64 from a string, returning 0 on failure.
func parseFloatStr(s string) float64 {
	s = strings.TrimSpace(s)
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// parseInt64Str parses an int64 from a string, returning 0 on failure.
func parseInt64Str(s string) int64 {
	s = strings.TrimSpace(s)
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// parseInt64Digits extracts all digit runes from s and parses as int64.
// Used for strings like "1,234 ratings" → 1234.
func parseInt64Digits(s string) int64 {
	var b strings.Builder
	for _, ch := range s {
		if unicode.IsDigit(ch) {
			b.WriteRune(ch)
		}
	}
	if b.Len() == 0 {
		return 0
	}
	v, _ := strconv.ParseInt(b.String(), 10, 64)
	return v
}

// extractQueryParam extracts a named query parameter from a URL string.
func extractQueryParam(rawURL, param string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Query().Get(param)
}

// extractPathSegmentAfter returns the path segment immediately following
// the given prefix segment in rawURL. E.g. prefix="/author/" in
// "/author/john-doe?ref=..." returns "john-doe".
func extractPathSegmentAfter(rawURL, prefix string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		// fallback: plain string search
		idx := strings.Index(rawURL, prefix)
		if idx < 0 {
			return ""
		}
		rest := rawURL[idx+len(prefix):]
		end := strings.IndexAny(rest, "/?#")
		if end < 0 {
			return rest
		}
		return rest[:end]
	}
	path := u.Path
	idx := strings.Index(path, prefix)
	if idx < 0 {
		return ""
	}
	rest := path[idx+len(prefix):]
	end := strings.IndexAny(rest, "/")
	if end < 0 {
		return rest
	}
	return rest[:end]
}

// starRatingFromClass converts an Amazon star class name like "a-star-4-5"
// into a float64 rating (4.5). Returns 0 on parse failure.
func starRatingFromClass(class string) float64 {
	// Find segment matching a-star-N or a-star-N-M
	for _, part := range strings.Fields(class) {
		if !strings.HasPrefix(part, "a-star-") {
			continue
		}
		suffix := strings.TrimPrefix(part, "a-star-")
		// also handle a-star-small-N-M
		suffix = strings.TrimPrefix(suffix, "small-")
		segments := strings.Split(suffix, "-")
		switch len(segments) {
		case 1:
			v, err := strconv.ParseFloat(segments[0], 64)
			if err == nil {
				return v
			}
		case 2:
			v, err := strconv.ParseFloat(segments[0]+"."+segments[1], 64)
			if err == nil {
				return v
			}
		}
	}
	return 0
}

// absoluteURL prepends baseURL to href if href is relative.
func absoluteURL(base, href string) string {
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(href, "/")
}

// dedup returns a new slice with duplicates removed, preserving order.
func dedup(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
