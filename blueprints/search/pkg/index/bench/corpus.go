package bench

import (
	"bufio"
	"compress/bzip2"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

// DefaultCorpusURL is the Wikipedia articles bzip2 download URL.
const DefaultCorpusURL = "https://www.dropbox.com/s/wwnfnu441w1ec9p/wiki-articles.json.bz2?dl=1"

var nonAlphaRe = regexp.MustCompile(`[^a-zA-Z]+`)

// NormalizeText replaces non-alpha runs with a single space and lowercases.
// Exported for testing.
func NormalizeText(s string) string {
	return strings.ToLower(nonAlphaRe.ReplaceAllString(s, " "))
}

// wikiRaw is the raw JSON shape of one Wikipedia article line.
type wikiRaw struct {
	URL  string `json:"url"`
	Body string `json:"body"`
}

// corpusDoc is the normalized output shape written to corpus.ndjson.
type corpusDoc struct {
	DocID string `json:"doc_id"`
	Text  string `json:"text"`
}

// TransformWikiLine parses one raw Wikipedia NDJSON line.
// Returns ok=false for empty URL or parse errors (caller should skip).
// Exported for testing.
func TransformWikiLine(line []byte) (corpusDoc, bool, error) {
	var raw wikiRaw
	if err := json.Unmarshal(line, &raw); err != nil {
		return corpusDoc{}, false, nil // skip malformed lines silently
	}
	if raw.URL == "" {
		return corpusDoc{}, false, nil
	}
	return corpusDoc{DocID: raw.URL, Text: NormalizeText(raw.Body)}, true, nil
}

// DownloadConfig controls the corpus download.
type DownloadConfig struct {
	URL     string // default: DefaultCorpusURL
	OutPath string // absolute path for corpus.ndjson
	MaxDocs int64  // 0 = unlimited
	Force   bool   // overwrite existing file
}

// DownloadStats tracks live download progress.
type DownloadStats struct {
	BytesDownloaded atomic.Int64 // compressed bytes received
	BytesWritten    atomic.Int64 // bytes written to corpus.ndjson
	DocsWritten     atomic.Int64
	StartTime       time.Time
	TotalBytes      int64 // from Content-Length (0 if unknown)
}

// countingReader wraps an io.Reader and counts bytes into stats.
type countingReader struct {
	r     io.Reader
	stats *DownloadStats
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	cr.stats.BytesDownloaded.Add(int64(n))
	return n, err
}

// writeCounting wraps a writer and counts bytes written.
type writeCounting struct {
	w     io.Writer
	stats *DownloadStats
}

func (wc *writeCounting) Write(p []byte) (int, error) {
	n, err := wc.w.Write(p)
	wc.stats.BytesWritten.Add(int64(n))
	return n, err
}

// Download streams the Wikipedia bzip2 corpus, normalizes, writes corpus.ndjson.
// progress is called every 200ms; pass nil to disable.
func Download(ctx context.Context, cfg DownloadConfig, progress func(*DownloadStats)) (*DownloadStats, error) {
	if cfg.URL == "" {
		cfg.URL = DefaultCorpusURL
	}
	if !cfg.Force {
		if _, err := os.Stat(cfg.OutPath); err == nil {
			return nil, fmt.Errorf("corpus already exists at %s (use --force to overwrite)", cfg.OutPath)
		}
	}
	if err := os.MkdirAll(filepath.Dir(cfg.OutPath), 0o755); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d from %s", resp.StatusCode, cfg.URL)
	}

	stats := &DownloadStats{
		StartTime:  time.Now(),
		TotalBytes: resp.ContentLength,
	}

	if progress != nil {
		stopTicker := make(chan struct{})
		ticker := time.NewTicker(200 * time.Millisecond)
		go func() {
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					progress(stats)
				case <-stopTicker:
					return
				}
			}
		}()
		defer close(stopTicker)
	}

	cr := &countingReader{r: resp.Body, stats: stats}
	bzr := bzip2.NewReader(cr)

	outFile, err := os.Create(cfg.OutPath)
	if err != nil {
		return nil, fmt.Errorf("create corpus: %w", err)
	}
	wc := &writeCounting{w: outFile, stats: stats}
	bw := bufio.NewWriterSize(wc, 4<<20)
	enc := json.NewEncoder(bw)
	enc.SetEscapeHTML(false)

	scanner := bufio.NewScanner(bzr)
	scanner.Buffer(make([]byte, 4<<20), 4<<20)

	for scanner.Scan() {
		if ctx.Err() != nil {
			bw.Flush()
			outFile.Close()
			os.Remove(cfg.OutPath)
			return stats, ctx.Err()
		}
		doc, ok, _ := TransformWikiLine(scanner.Bytes())
		if !ok {
			continue
		}
		if err := enc.Encode(doc); err != nil {
			bw.Flush()
			outFile.Close()
			os.Remove(cfg.OutPath)
			return stats, fmt.Errorf("encode: %w", err)
		}
		n := stats.DocsWritten.Add(1)
		if cfg.MaxDocs > 0 && n >= cfg.MaxDocs {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		bw.Flush()
		outFile.Close()
		os.Remove(cfg.OutPath)
		if ctx.Err() != nil {
			return stats, ctx.Err()
		}
		return stats, fmt.Errorf("scan: %w", err)
	}

	if err := bw.Flush(); err != nil {
		outFile.Close()
		os.Remove(cfg.OutPath)
		return stats, err
	}
	if err := outFile.Close(); err != nil {
		os.Remove(cfg.OutPath)
		return stats, err
	}
	return stats, nil
}

// CorpusReader reads corpus.ndjson and sends index.Document values on docCh.
// It closes docCh when done. maxDocs=0 means read all.
// Returns immediately; reading happens in a goroutine.
func CorpusReader(ctx context.Context, corpusPath string, maxDocs int64, docCh chan<- index.Document) error {
	f, err := os.Open(corpusPath)
	if err != nil {
		close(docCh)
		return fmt.Errorf("open corpus: %w", err)
	}
	go func() {
		defer f.Close()
		defer close(docCh)
		br := bufio.NewReaderSize(f, 4<<20)
		var count int64
		for {
			if ctx.Err() != nil {
				return
			}
			line, err := br.ReadBytes('\n')
			if len(line) > 0 {
				// trim newline
				if line[len(line)-1] == '\n' {
					line = line[:len(line)-1]
				}
				if len(line) > 0 {
					var doc corpusDoc  // zero-value each iteration
					if jsonErr := json.Unmarshal(line, &doc); jsonErr == nil && doc.DocID != "" {
						select {
						case docCh <- index.Document{DocID: doc.DocID, Text: []byte(doc.Text)}:
							count++
							if maxDocs > 0 && count >= maxDocs {
								return
							}
						case <-ctx.Done():
							return
						}
					}
				}
			}
			if err == io.EOF {
				return
			}
			if err != nil {
				return
			}
		}
	}()
	return nil
}
