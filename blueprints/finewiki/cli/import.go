package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/finewiki/store/duckdb"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	hfDataset   = "HuggingFaceFW/finewiki"
	hfServerAPI = "https://datasets-server.huggingface.co"
)

func importCmd() *cobra.Command {
	var dataDir string

	c := &cobra.Command{
		Use:   "import <lang>",
		Short: "Download and index Wikipedia data for a language",
		Long: `Download the FineWiki parquet file for a specific language from HuggingFace
and prepare the DuckDB database for serving.

The data is saved to <data-dir>/<lang>/.
After importing, run 'finewiki serve <lang>' to start the server.

Examples:
  finewiki import vi              # Download Vietnamese data
  finewiki import en              # Download English data
  finewiki import ja --data ~/wiki`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			lang := args[0]
			return runImport(cmd.Context(), dataDir, lang)
		},
	}

	c.Flags().StringVar(&dataDir, "data", DefaultDataDir(), "Base data directory")

	return c
}

// parquetFileInfo contains URL and size for a parquet file
type parquetFileInfo struct {
	URL      string
	Filename string
	Size     int64
}

func runImport(ctx context.Context, dataDir, lang string) error {
	ui := NewUI()
	importStart := time.Now()

	// Get parquet file info for this language
	files, err := getParquetFiles(ctx, lang)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to fetch file info for '%s'", lang))
		ui.Hint(err.Error())
		ui.Blank()
		ui.Hint("Check available languages at:")
		ui.Hint("https://huggingface.co/datasets/HuggingFaceFW/finewiki")
		return err
	}

	if len(files) == 0 {
		ui.Error(fmt.Sprintf("No data found for language '%s'", lang))
		ui.Blank()
		ui.Hint("Check available languages at:")
		ui.Hint("https://huggingface.co/datasets/HuggingFaceFW/finewiki")
		return fmt.Errorf("no parquet files found for language: %s", lang)
	}

	langDir := LangDir(dataDir, lang)

	// Print header
	ui.Header(iconDownload, fmt.Sprintf("Downloading %s Wikipedia", strings.ToUpper(lang)))
	ui.Info("Source", fmt.Sprintf("huggingface.co/datasets/%s", hfDataset))
	ui.Info("Target", langDir)
	ui.Blank()

	// Calculate total size
	var totalSize int64
	for _, f := range files {
		totalSize += f.Size
	}
	ui.Info("Files", fmt.Sprintf("%d (%s total)", len(files), formatBytes(totalSize)))

	// Create downloader
	dl := NewDownloader()
	if dl.UseCurl() {
		ui.Info("Using", "curl")
	} else {
		ui.Info("Using", "native Go HTTP")
	}
	ui.Blank()

	// Create language directory
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		return err
	}

	// Download all parquet files
	var downloaded, skipped int
	for i, f := range files {
		var dst string
		if len(files) == 1 {
			dst = filepath.Join(langDir, "data.parquet")
		} else {
			dst = filepath.Join(langDir, fmt.Sprintf("data-%03d.parquet", i))
		}

		filename := filepath.Base(dst)

		// Check if file already exists with correct size
		if fi, err := os.Stat(dst); err == nil && fi.Size() == f.Size {
			ui.Progress(i+1, len(files), filename, formatBytes(f.Size), true)
			skipped++
			continue
		}

		// Download file
		dlStart := time.Now()
		if err := dl.Download(ctx, f.URL, dst, f.Size); err != nil {
			ui.Error(fmt.Sprintf("Failed to download %s", filename))
			ui.Hint(err.Error())
			return fmt.Errorf("download %s: %w", f.Filename, err)
		}

		// Verify and print completion
		fi, _ := os.Stat(dst)
		ui.ProgressDone(i+1, len(files), filename, formatBytes(fi.Size()), time.Since(dlStart))
		downloaded++
	}

	ui.Blank()

	// Seed database
	if err := seedDatabase(ctx, ui, dataDir, lang); err != nil {
		return err
	}

	// Get article count for summary
	articleCount := getArticleCount(dataDir, lang)

	// Print summary
	ui.Success("Import complete!")
	ui.Summary([][2]string{
		{"Articles", formatNumber(articleCount)},
		{"Database", DuckDBPath(dataDir, lang)},
		{"Duration", time.Since(importStart).Round(100 * time.Millisecond).String()},
	})
	ui.Blank()
	ui.Hint(fmt.Sprintf("Run: finewiki serve %s", lang))
	ui.Blank()

	return nil
}

func seedDatabase(ctx context.Context, ui *UI, dataDir, lang string) error {
	dbPath := DuckDBPath(dataDir, lang)
	parquetGlob := ParquetGlob(dataDir, lang)

	ui.StartSpinner("Preparing database...")

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		ui.StopSpinnerError("Failed to open database")
		return err
	}
	defer db.Close()

	store, err := duckdb.New(db)
	if err != nil {
		ui.StopSpinnerError("Failed to initialize store")
		return err
	}

	start := time.Now()

	if err := store.Ensure(ctx, duckdb.Config{
		ParquetGlob: parquetGlob,
		EnableFTS:   true,
	}, duckdb.EnsureOptions{
		SeedIfEmpty: true,
		BuildIndex:  true,
		BuildFTS:    true,
	}); err != nil {
		ui.StopSpinnerError("Failed to prepare database")
		return err
	}

	ui.StopSpinner("Database ready", time.Since(start))
	return nil
}

func getArticleCount(dataDir, lang string) int64 {
	dbPath := DuckDBPath(dataDir, lang)
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return 0
	}
	defer db.Close()

	var count int64
	db.QueryRow("SELECT count(*) FROM pages").Scan(&count)
	return count
}

func getParquetFiles(ctx context.Context, lang string) ([]parquetFileInfo, error) {
	// Query HuggingFace datasets server API
	url := fmt.Sprintf("%s/parquet?dataset=%s&config=%s", hfServerAPI, hfDataset, lang)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if token := os.Getenv("HF_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("HuggingFace API error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		ParquetFiles []struct {
			URL      string `json:"url"`
			Filename string `json:"filename"`
			Size     int64  `json:"size"`
		} `json:"parquet_files"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	files := make([]parquetFileInfo, 0, len(result.ParquetFiles))
	for _, f := range result.ParquetFiles {
		if f.URL != "" {
			files = append(files, parquetFileInfo{
				URL:      f.URL,
				Filename: f.Filename,
				Size:     f.Size,
			})
		}
	}

	return files, nil
}

