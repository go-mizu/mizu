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
	"sort"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	var (
		installed bool
		dataDir   string
	)

	c := &cobra.Command{
		Use:   "list",
		Short: "List available languages",
		Long: `List available languages from the FineWiki dataset.

Examples:
  finewiki list              # Show all available languages
  finewiki list --installed  # Show only locally installed languages`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if installed {
				return listInstalled(dataDir)
			}
			return listAvailable(cmd.Context())
		},
	}

	c.Flags().BoolVar(&installed, "installed", false, "Show only installed languages")
	c.Flags().StringVar(&dataDir, "data", DefaultDataDir(), "Base data directory")

	return c
}

// configInfo contains metadata for a language config from HuggingFace API
type configInfo struct {
	ConfigName   string               `json:"config_name"`
	DownloadSize int64                `json:"download_size"`
	DatasetSize  int64                `json:"dataset_size"`
	Splits       map[string]splitInfo `json:"splits"`
}

// splitInfo contains split-level metadata
type splitInfo struct {
	Name        string `json:"name"`
	NumBytes    int64  `json:"num_bytes"`
	NumExamples int64  `json:"num_examples"`
}

func listAvailable(ctx context.Context) error {
	url := fmt.Sprintf("%s/info?dataset=%s", hfServerAPI, hfDataset)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	if token := os.Getenv("HF_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("HuggingFace API error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	// The API returns dataset_info as a map where keys are language codes
	var result struct {
		DatasetInfo map[string]configInfo `json:"dataset_info"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	langs := make([]string, 0, len(result.DatasetInfo))
	for lang := range result.DatasetInfo {
		langs = append(langs, lang)
	}
	sort.Strings(langs)

	fmt.Printf("Available languages (%d):\n\n", len(langs))

	// Print in columns for better readability
	cols := 8
	for i, lang := range langs {
		fmt.Printf("  %-10s", lang)
		if (i+1)%cols == 0 {
			fmt.Println()
		}
	}
	if len(langs)%cols != 0 {
		fmt.Println()
	}

	return nil
}

// langInfo holds details about an installed language
type langInfo struct {
	Lang      string
	Files     int
	SizeBytes int64
	Pages     int64
}

func listInstalled(dataDir string) error {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No languages installed")
			fmt.Printf("Data directory: %s\n", dataDir)
			fmt.Println("\nRun 'finewiki import <lang>' to download data")
			return nil
		}
		return err
	}

	var infos []langInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		lang := e.Name()
		info := gatherLangInfo(dataDir, lang)
		if info.Files > 0 {
			infos = append(infos, info)
		}
	}

	if len(infos) == 0 {
		fmt.Println("No languages installed")
		fmt.Printf("Data directory: %s\n", dataDir)
		fmt.Println("\nRun 'finewiki import <lang>' to download data")
		return nil
	}

	// Sort by language code
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Lang < infos[j].Lang
	})

	fmt.Printf("Installed Languages (%d):\n\n", len(infos))

	// Print table header
	fmt.Printf("  %-10s %6s %12s %12s\n", "LANG", "FILES", "SIZE", "PAGES")
	fmt.Printf("  %-10s %6s %12s %12s\n", "----", "-----", "----", "-----")

	var totalSize int64
	var totalPages int64
	for _, info := range infos {
		pagesStr := "-"
		if info.Pages > 0 {
			pagesStr = formatNumber(info.Pages)
			totalPages += info.Pages
		}
		fmt.Printf("  %-10s %6d %12s %12s\n",
			info.Lang,
			info.Files,
			formatBytes(info.SizeBytes),
			pagesStr,
		)
		totalSize += info.SizeBytes
	}

	fmt.Println()
	fmt.Printf("Data directory: %s\n", dataDir)
	fmt.Printf("Total size: %s\n", formatBytes(totalSize))
	if totalPages > 0 {
		fmt.Printf("Total pages: %s\n", formatNumber(totalPages))
	}

	return nil
}

func gatherLangInfo(dataDir, lang string) langInfo {
	langDir := filepath.Join(dataDir, lang)

	var files int
	var sizeBytes int64

	// Check for single parquet file
	single := filepath.Join(langDir, "data.parquet")
	if fi, err := os.Stat(single); err == nil {
		files = 1
		sizeBytes = fi.Size()
	} else {
		// Check for sharded parquet files
		pattern := filepath.Join(langDir, "data-*.parquet")
		matches, _ := filepath.Glob(pattern)
		files = len(matches)
		for _, m := range matches {
			if fi, err := os.Stat(m); err == nil {
				sizeBytes += fi.Size()
			}
		}
	}

	// Try to get page count from DuckDB if available
	var pages int64
	dbPath := filepath.Join(langDir, "wiki.duckdb")
	if _, err := os.Stat(dbPath); err == nil {
		pages = countPagesFromDB(dbPath)
	}

	return langInfo{
		Lang:      lang,
		Files:     files,
		SizeBytes: sizeBytes,
		Pages:     pages,
	}
}

func countPagesFromDB(dbPath string) int64 {
	// Open DuckDB in read-only mode
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return 0
	}
	defer db.Close()

	var count int64
	row := db.QueryRow("SELECT COUNT(*) FROM titles")
	if err := row.Scan(&count); err != nil {
		return 0
	}
	return count
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatNumber(n int64) string {
	str := fmt.Sprintf("%d", n)
	// Add commas for thousands
	result := ""
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}
