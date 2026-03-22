package cc_v2

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ParseFileSelector parses a file selector like "0", "0-9", "1,2,5-10".
func ParseFileSelector(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "all" {
		return nil, nil
	}
	if strings.Contains(s, ",") {
		seen := make(map[int]bool)
		var out []int
		for _, part := range strings.Split(s, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			sub, err := ParseFileSelector(part)
			if err != nil {
				return nil, err
			}
			for _, n := range sub {
				if !seen[n] {
					seen[n] = true
					out = append(out, n)
				}
			}
		}
		return out, nil
	}
	if strings.Contains(s, "-") {
		parts := strings.SplitN(s, "-", 2)
		lo, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		hi, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("bad range: %s", s)
		}
		if lo > hi {
			return nil, fmt.Errorf("inverted range: %d > %d", lo, hi)
		}
		out := make([]int, 0, hi-lo+1)
		for i := lo; i <= hi; i++ {
			out = append(out, i)
		}
		return out, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil, fmt.Errorf("bad file index: %s", s)
	}
	return []int{n}, nil
}

// FmtBytes formats bytes as human-readable.
func FmtBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
