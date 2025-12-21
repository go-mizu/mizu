package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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

	var result struct {
		DatasetInfo struct {
			Configs []struct {
				ConfigName string `json:"config_name"`
			} `json:"config_names_with_splits,omitempty"`
		} `json:"dataset_info"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	langs := make([]string, 0, len(result.DatasetInfo.Configs))
	for _, c := range result.DatasetInfo.Configs {
		if c.ConfigName != "" {
			langs = append(langs, c.ConfigName)
		}
	}
	sort.Strings(langs)

	fmt.Printf("available languages (%d):\n", len(langs))
	for _, lang := range langs {
		fmt.Printf("  %s\n", lang)
	}

	return nil
}

func listInstalled(dataDir string) error {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("no languages installed")
			fmt.Printf("data directory: %s\n", dataDir)
			return nil
		}
		return err
	}

	var langs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		lang := e.Name()
		parquet := filepath.Join(dataDir, lang, "data.parquet")
		if _, err := os.Stat(parquet); err == nil {
			langs = append(langs, lang)
		} else {
			// Check for sharded parquets
			pattern := filepath.Join(dataDir, lang, "data-*.parquet")
			matches, _ := filepath.Glob(pattern)
			if len(matches) > 0 {
				langs = append(langs, lang)
			}
		}
	}

	if len(langs) == 0 {
		fmt.Println("no languages installed")
		fmt.Printf("data directory: %s\n", dataDir)
		fmt.Println("\nrun 'finewiki import <lang>' to download data")
		return nil
	}

	sort.Strings(langs)
	fmt.Printf("installed languages (%d):\n", len(langs))
	for _, lang := range langs {
		fmt.Printf("  %s\n", lang)
	}
	fmt.Printf("\ndata directory: %s\n", dataDir)

	return nil
}
