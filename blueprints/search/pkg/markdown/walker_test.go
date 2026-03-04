package markdown

import (
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestWalk_Integration(t *testing.T) {
	// Create temp dirs
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "bodies")
	outputDir := filepath.Join(tmpDir, "markdown")
	indexPath := filepath.Join(outputDir, "index.duckdb")

	// Write a fake bodystore file: ab/cd/rest.gz
	subDir := filepath.Join(inputDir, "ab", "cd")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	html := `<!DOCTYPE html><html><head><title>Test</title></head>
<body><article><h1>Test Article</h1><p>Hello world paragraph.</p></article></body></html>`

	gzPath := filepath.Join(subDir, "ef0123456789abcdef0123456789abcdef0123456789abcdef0123456789ab.gz")
	writeTestGz(t, gzPath, []byte(html))

	// Write a second file
	html2 := `<!DOCTYPE html><html><head><title>Second</title></head>
<body><article><h1>Second Article</h1><p>Another paragraph here.</p></article></body></html>`

	gzPath2 := filepath.Join(subDir, "1111111111111111111111111111111111111111111111111111111111111111.gz")
	writeTestGz(t, gzPath2, []byte(html2))

	// Run walker
	cfg := WalkConfig{
		InputDir:  inputDir,
		OutputDir: outputDir,
		IndexPath: indexPath,
		Workers:   2,
		BatchSize: 10,
	}

	stats, err := Walk(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}

	if stats.Converted < 1 {
		t.Errorf("expected at least 1 converted, got %d", stats.Converted)
	}
	if stats.Errors > 0 {
		t.Errorf("expected 0 errors, got %d", stats.Errors)
	}
	t.Logf("Converted: %d, Skipped: %d, Errors: %d, Duration: %s",
		stats.Converted, stats.Skipped, stats.Errors, stats.Duration)

	// Check output file exists
	outPath := filepath.Join(outputDir, "ab", "cd", "ef0123456789abcdef0123456789abcdef0123456789abcdef0123456789ab.md.gz")
	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("output file not found: %s", outPath)
	}

	// Read and verify output
	data, err := readGzTestFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("output file is empty")
	}
	t.Logf("Output markdown: %s", string(data))

	// Run again — should skip existing
	stats2, err := Walk(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if stats2.Skipped < 1 {
		t.Errorf("expected skips on second run, got %d", stats2.Skipped)
	}
	t.Logf("Second run: Converted: %d, Skipped: %d", stats2.Converted, stats2.Skipped)

	// Run with force — should re-convert
	cfg.Force = true
	stats3, err := Walk(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if stats3.Converted < 1 {
		t.Errorf("expected conversions with --force, got %d", stats3.Converted)
	}
	t.Logf("Force run: Converted: %d, Skipped: %d", stats3.Converted, stats3.Skipped)
}

func writeTestGz(t *testing.T, path string, data []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	if _, err := gz.Write(data); err != nil {
		f.Close()
		t.Fatal(err)
	}
	gz.Close()
	f.Close()
}

func readGzTestFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	var buf []byte
	tmp := make([]byte, 4096)
	for {
		n, err := gz.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			break
		}
	}
	return buf, nil
}
