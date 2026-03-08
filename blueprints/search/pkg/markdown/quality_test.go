package markdown

import (
	"bufio"
	"compress/gzip"
	"io"
	"sort"
	"math"
	"os"
	"strings"
	"testing"
)

// TestQualityParity compares ConvertLight vs Convert (trafilatura) on real
// HTML pages extracted from a Common Crawl .warc.gz file.
//
// Set WARC_TEST_FILE to a .warc.gz path:
//
//	WARC_TEST_FILE=~/data/common-crawl/CC-MAIN-2026-04/warc/CC-MAIN-...warc.gz \
//	  go test -run TestQualityParity -v ./pkg/markdown/
func TestQualityParity(t *testing.T) {
	warcPath := os.Getenv("WARC_TEST_FILE")
	if warcPath == "" {
		t.Skip("WARC_TEST_FILE not set")
	}

	pages := extractHTMLPages(t, warcPath, 100)
	if len(pages) < 10 {
		t.Fatalf("only extracted %d pages, need at least 10", len(pages))
	}
	t.Logf("extracted %d HTML pages from %s", len(pages), warcPath)

	var (
		bothOK       int
		trafOK       int
		lightOK      int
		neitherOK    int
		jaccard      []float64
		charRatio    []float64
		headingDelta []float64
		linkDelta    []float64
	)

	for i, page := range pages {
		traf := Convert(page, "")
		light := ConvertLight(page, "")

		tHas := traf.HasContent && traf.Markdown != ""
		lHas := light.HasContent && light.Markdown != ""

		switch {
		case tHas && lHas:
			bothOK++
		case tHas && !lHas:
			trafOK++
			if i < 5 {
				t.Logf("  page %d: trafilatura OK (%d bytes), light FAIL: %s", i, len(traf.Markdown), light.Error)
			}
		case !tHas && lHas:
			lightOK++
		default:
			neitherOK++
		}

		if tHas && lHas {
			j := jaccardSimilarity(traf.Markdown, light.Markdown)
			jaccard = append(jaccard, j)

			if j < 0.3 {
				tLinks := countPattern(traf.Markdown, "](")
				lLinks := countPattern(light.Markdown, "](")
				t.Logf("  LOW JACCARD page %d: j=%.3f traf=%d bytes light=%d bytes ratio=%.1f tLinks=%d lLinks=%d",
					i, j, len(traf.Markdown), len(light.Markdown),
					float64(len(light.Markdown))/float64(len(traf.Markdown)), tLinks, lLinks)
			}

			if len(traf.Markdown) > 0 {
				ratio := float64(len(light.Markdown)) / float64(len(traf.Markdown))
				charRatio = append(charRatio, ratio)
			}

			tHeadings := countPattern(traf.Markdown, "\n#")
			lHeadings := countPattern(light.Markdown, "\n#")
			if tHeadings > 0 {
				headingDelta = append(headingDelta, float64(lHeadings)/float64(tHeadings))
			}

			tLinks := countPattern(traf.Markdown, "](")
			lLinks := countPattern(light.Markdown, "](")
			if tLinks > 0 {
				linkDelta = append(linkDelta, float64(lLinks)/float64(tLinks))
			}
		}
	}

	total := len(pages)
	t.Logf("")
	t.Logf("=== Quality Parity Report (%d pages) ===", total)
	t.Logf("  Both OK:       %d (%.1f%%)", bothOK, pct(bothOK, total))
	t.Logf("  Trafilatura only: %d (%.1f%%)", trafOK, pct(trafOK, total))
	t.Logf("  Light only:    %d (%.1f%%)", lightOK, pct(lightOK, total))
	t.Logf("  Neither:       %d (%.1f%%)", neitherOK, pct(neitherOK, total))

	if len(jaccard) > 0 {
		avgJ := mean(jaccard)
		medJ := median(jaccard)
		t.Logf("  Jaccard similarity (word overlap): avg=%.3f median=%.3f min=%.3f", avgJ, medJ, min(jaccard))
		t.Logf("  Char count ratio (light/traf):     avg=%.2f median=%.2f", mean(charRatio), median(charRatio))
		if len(headingDelta) > 0 {
			t.Logf("  Heading ratio (light/traf):        avg=%.2f", mean(headingDelta))
		}
		if len(linkDelta) > 0 {
			t.Logf("  Link ratio (light/traf):           avg=%.2f median=%.2f", mean(linkDelta), median(linkDelta))
		}

		// Quality gates — use median (robust to outliers)
		if medJ < 0.55 {
			t.Errorf("FAIL: Jaccard median %.3f < 0.55 threshold", medJ)
		}
		successGap := pct(trafOK, total)
		if successGap > 5.0 {
			t.Errorf("FAIL: %.1f%% pages trafilatura-only (>5%% gap)", successGap)
		}
	}
}

// extractHTMLPages reads up to maxPages HTML response bodies from a .warc.gz.
func extractHTMLPages(t *testing.T, warcPath string, maxPages int) [][]byte {
	t.Helper()
	f, err := os.Open(warcPath)
	if err != nil {
		t.Fatalf("open %s: %v", warcPath, err)
	}
	defer f.Close()

	br := bufio.NewReaderSize(f, 64*1024)
	gz, err := gzip.NewReader(br)
	if err != nil {
		t.Fatalf("gzip: %v", err)
	}
	gz.Multistream(false)

	var pages [][]byte
	first := true

	for len(pages) < maxPages {
		if !first {
			io.Copy(io.Discard, gz)
			if err := gz.Reset(br); err != nil {
				break
			}
			gz.Multistream(false)
		}
		first = false

		body, err := io.ReadAll(gz)
		if err != nil {
			continue
		}

		// Check if it's an HTTP response with text/html
		if !isHTMLResponse(body) {
			continue
		}

		// Extract HTML body after HTTP headers
		if idx := findHTTPBodyStart(body); idx > 0 {
			htmlBody := body[idx:]
			if len(htmlBody) > 500 && len(htmlBody) < 512*1024 {
				pages = append(pages, htmlBody)
			}
		}
	}
	return pages
}

func isHTMLResponse(data []byte) bool {
	// Look for "200" status and "text/html" content type in first 2KB
	header := data
	if len(header) > 2048 {
		header = header[:2048]
	}
	lower := strings.ToLower(string(header))
	return strings.Contains(lower, "200") && strings.Contains(lower, "text/html")
}

func findHTTPBodyStart(data []byte) int {
	// Find \r\n\r\n separator between HTTP headers and body
	for i := 0; i < len(data)-3; i++ {
		if data[i] == '\r' && data[i+1] == '\n' && data[i+2] == '\r' && data[i+3] == '\n' {
			return i + 4
		}
	}
	return -1
}

func jaccardSimilarity(a, b string) float64 {
	wordsA := wordSet(a)
	wordsB := wordSet(b)

	intersection := 0
	for w := range wordsA {
		if wordsB[w] {
			intersection++
		}
	}
	union := len(wordsA) + len(wordsB) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func wordSet(s string) map[string]bool {
	set := make(map[string]bool)
	for _, w := range strings.Fields(s) {
		w = strings.ToLower(strings.Trim(w, ".,;:!?\"'()[]{}"))
		if len(w) >= 2 {
			set[w] = true
		}
	}
	return set
}

func countPattern(s, pattern string) int {
	return strings.Count(s, pattern)
}

func pct(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) * 100.0 / float64(total)
}

func median(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func min(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	m := math.MaxFloat64
	for _, v := range vals {
		if v < m {
			m = v
		}
	}
	return m
}

func max(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	m := -math.MaxFloat64
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	return m
}
