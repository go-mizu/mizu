package markdown_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	md "github.com/go-mizu/mizu/blueprints/search/pkg/markdown"
	warcpkg "github.com/go-mizu/mizu/blueprints/search/pkg/warc"
)

var sampleHTML [][]byte

func loadSamples(b *testing.B) {
	b.Helper()
	if len(sampleHTML) > 0 {
		return
	}
	warcPath := os.Getenv("WARC_PATH")
	if warcPath == "" {
		warcPath = os.ExpandEnv("$HOME/data/common-crawl/CC-MAIN-2026-08/warc/CC-MAIN-20260206181458-20260206211458-00000.warc.gz")
	}
	f, err := os.Open(warcPath)
	if err != nil {
		b.Skipf("WARC file not found: %s", warcPath)
	}
	defer f.Close()

	wr := warcpkg.NewReader(f)
	count := 0
	for wr.Next() && count < 200 {
		rec := wr.Record()
		if rec.Header.Type() != warcpkg.TypeResponse {
			io.Copy(io.Discard, rec.Body)
			continue
		}
		body, _ := io.ReadAll(rec.Body)
		if len(body) > 1024 && len(body) < 512*1024 && bytes.Contains(body[:min(len(body), 512)], []byte("200")) {
			sampleHTML = append(sampleHTML, body)
			count++
		}
	}
	if len(sampleHTML) == 0 {
		b.Skip("no HTML samples found in WARC file")
	}
	b.Logf("loaded %d HTML samples", len(sampleHTML))
}

func BenchmarkConvert(b *testing.B) {
	loadSamples(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		html := sampleHTML[i%len(sampleHTML)]
		md.Convert(html, "")
	}
}

func BenchmarkConvertFast(b *testing.B) {
	loadSamples(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		html := sampleHTML[i%len(sampleHTML)]
		md.ConvertFast(html, "")
	}
}

func BenchmarkConvertLight(b *testing.B) {
	loadSamples(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		html := sampleHTML[i%len(sampleHTML)]
		md.ConvertLight(html, "")
	}
}
