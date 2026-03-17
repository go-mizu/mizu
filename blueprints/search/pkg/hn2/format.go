package hn2

import (
	"fmt"
	"strings"
)

// fmtInt formats n with comma separators (e.g. 1234567 → "1,234,567").
func fmtInt(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b = append(b, ',')
		}
		b = append(b, byte(c))
	}
	return string(b)
}

// fmtCount formats n as a human-readable magnitude (e.g. 1234567 → "1.2M").
func fmtCount(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// blockFilename returns the parquet filename for a live 5-min block,
// replacing ":" with "_" for filesystem compatibility.
// e.g. blockFilename("2026-03-14", "00:05") → "2026-03-14_00_05.parquet"
func blockFilename(date, hhmm string) string {
	return date + "_" + strings.ReplaceAll(hhmm, ":", "_") + ".parquet"
}
