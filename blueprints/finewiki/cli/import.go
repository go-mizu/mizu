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

func runImport(ctx context.Context, dataDir, lang string) error {
	// Get parquet URLs for this language
	urls, err := getParquetURLs(ctx, lang)
	if err != nil {
		return err
	}

	if len(urls) == 0 {
		return fmt.Errorf("no parquet files found for language: %s", lang)
	}

	langDir := LangDir(dataDir, lang)
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		return err
	}

	// Download all parquet files
	for i, url := range urls {
		var dst string
		if len(urls) == 1 {
			dst = filepath.Join(langDir, "data.parquet")
		} else {
			dst = filepath.Join(langDir, fmt.Sprintf("data-%03d.parquet", i))
		}

		fmt.Printf("downloading %s (%d/%d)...\n", filepath.Base(dst), i+1, len(urls))

		if err := downloadFile(ctx, url, dst); err != nil {
			return fmt.Errorf("download %s: %w", url, err)
		}

		fi, _ := os.Stat(dst)
		fmt.Printf("saved: %s (%.1f MB)\n", dst, float64(fi.Size())/(1024*1024))
	}

	fmt.Printf("\nimport complete. run 'finewiki serve %s' to start.\n", lang)
	return nil
}

func getParquetURLs(ctx context.Context, lang string) ([]string, error) {
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
			URL string `json:"url"`
		} `json:"parquet_files"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	urls := make([]string, 0, len(result.ParquetFiles))
	for _, f := range result.ParquetFiles {
		if f.URL != "" {
			urls = append(urls, f.URL)
		}
	}

	return urls, nil
}

func downloadFile(ctx context.Context, url, dst string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	if token := os.Getenv("HF_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 0} // No timeout for large files
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	tmp := dst + ".partial"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()

	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}

	return os.Rename(tmp, dst)
}
