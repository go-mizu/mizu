package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	hfDataset   = "HuggingFaceFW/finewiki"
	hfServerAPI = "https://datasets-server.huggingface.co"
)

func importCmd() *cobra.Command {
	var dataDir string

	c := &cobra.Command{
		Use:   "import <lang>",
		Short: "Download parquet data for a specific language",
		Long: `Download the FineWiki parquet file for a specific language from HuggingFace.

The parquet file is saved to <data-dir>/<lang>/data.parquet.
After downloading, run 'finewiki serve <lang>' to start the server.

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
	// Get parquet file info for this language
	files, err := getParquetFiles(ctx, lang)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no parquet files found for language: %s", lang)
	}

	langDir := LangDir(dataDir, lang)

	// Print download info
	fmt.Printf("Downloading %s Wikipedia\n", strings.ToUpper(lang))
	fmt.Printf("Source: huggingface.co/datasets/%s\n", hfDataset)
	fmt.Printf("Target: %s/\n", langDir)
	fmt.Println()

	// Calculate total size
	var totalSize int64
	for _, f := range files {
		totalSize += f.Size
	}
	fmt.Printf("Files: %d (total: %s)\n", len(files), formatBytes(totalSize))

	// Create downloader
	dl := NewDownloader()
	if dl.UseCurl() {
		fmt.Println("Using: curl")
	} else {
		fmt.Println("Using: native Go")
	}
	fmt.Println()

	// Create language directory
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		return err
	}

	// Download all parquet files (skip if already exists with correct size)
	var downloaded, skipped int
	for i, f := range files {
		var dst string
		if len(files) == 1 {
			dst = filepath.Join(langDir, "data.parquet")
		} else {
			dst = filepath.Join(langDir, fmt.Sprintf("data-%03d.parquet", i))
		}

		// Check if file already exists with correct size
		if fi, err := os.Stat(dst); err == nil && fi.Size() == f.Size {
			fmt.Printf("[%d/%d] %s - skipped (already exists)\n", i+1, len(files), filepath.Base(dst))
			skipped++
			continue
		}

		fmt.Printf("[%d/%d] %s (%s)\n", i+1, len(files), filepath.Base(dst), formatBytes(f.Size))

		if err := dl.Download(ctx, f.URL, dst, f.Size); err != nil {
			return fmt.Errorf("download %s: %w", f.Filename, err)
		}

		// Verify file size
		fi, _ := os.Stat(dst)
		fmt.Printf("  Saved: %s\n", formatBytes(fi.Size()))
		downloaded++
	}

	if skipped > 0 && downloaded == 0 {
		fmt.Printf("\nAll files already exist. Run 'finewiki serve %s' to start.\n", lang)
		return nil
	}

	fmt.Printf("\nImport complete. Run 'finewiki serve %s' to start.\n", lang)
	return nil
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
