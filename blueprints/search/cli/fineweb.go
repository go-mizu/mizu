package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
	fwdownloader "github.com/go-mizu/mizu/blueprints/search/pkg/fineweb"
	"github.com/spf13/cobra"
)

// NewFinewebSeed creates the fineweb seed command.
func NewFinewebSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fineweb",
		Short: "Download and index FineWeb-2 dataset",
		Long: `Downloads parquet files from HuggingFace's FineWeb-2 dataset
and imports them into DuckDB for local full-text search.

FineWeb-2 is a large-scale, high-quality web corpus containing
cleaned and deduplicated web documents in multiple languages.

Examples:
  # List available languages
  search seed fineweb --list

  # Download and index Vietnamese
  search seed fineweb --lang vie_Latn

  # Download multiple languages
  search seed fineweb --lang vie_Latn,eng_Latn

  # Check download status
  search seed fineweb --status`,
		RunE: runFinewebSeed,
	}

	cmd.Flags().StringSlice("lang", []string{"vie_Latn"}, "Languages to download (comma-separated)")
	cmd.Flags().Bool("list", false, "List available languages")
	cmd.Flags().Bool("status", false, "Show download/import status")
	cmd.Flags().Bool("download-only", false, "Only download, do not import")
	cmd.Flags().Bool("import-only", false, "Only import (skip download)")

	return cmd
}

func runFinewebSeed(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("FineWeb-2 Dataset Manager"))
	fmt.Println()

	listFlag, _ := cmd.Flags().GetBool("list")
	statusFlag, _ := cmd.Flags().GetBool("status")
	downloadOnly, _ := cmd.Flags().GetBool("download-only")
	importOnly, _ := cmd.Flags().GetBool("import-only")
	langs, _ := cmd.Flags().GetStringSlice("lang")

	// Handle --list
	if listFlag {
		return listLanguages()
	}

	// Handle --status
	if statusFlag {
		return showStatus(ctx, langs)
	}

	// Download and/or import
	if !importOnly {
		if err := downloadData(ctx, langs); err != nil {
			return err
		}
	}

	if !downloadOnly {
		if err := importData(ctx, langs); err != nil {
			return err
		}
	}

	fmt.Println()
	fmt.Println(successStyle.Render("FineWeb-2 setup complete!"))
	fmt.Println()

	return nil
}

func listLanguages() error {
	fmt.Println(infoStyle.Render("Available languages:"))
	fmt.Println()

	for _, lang := range fwdownloader.SupportedLanguages {
		fmt.Printf("  %s  %s (%s)\n",
			titleStyle.Render(lang.Code),
			lang.Name,
			mutedStyle.Render(lang.Script),
		)
	}
	fmt.Println()
	fmt.Println(mutedStyle.Render("Use --lang <code> to download a specific language"))
	fmt.Println()
	return nil
}

var mutedStyle = subtitleStyle // Alias for clarity

func showStatus(ctx context.Context, langs []string) error {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, "data", "fineweb-2")
	dbDir := filepath.Join(home, "data", "blueprints", "search", "fineweb-2")

	downloader := fwdownloader.NewDownloader(fwdownloader.Config{DataDir: dataDir})

	// Show downloaded languages (filter by langs if provided)
	downloaded, err := downloader.ListDownloaded()
	if err != nil {
		return fmt.Errorf("listing downloaded languages: %w", err)
	}

	// Filter if langs specified
	if len(langs) > 0 {
		langSet := make(map[string]bool)
		for _, l := range langs {
			langSet[l] = true
		}
		var filtered []string
		for _, d := range downloaded {
			if langSet[d] {
				filtered = append(filtered, d)
			}
		}
		downloaded = filtered
	}

	fmt.Println(infoStyle.Render("Download status:"))
	fmt.Println()

	if len(downloaded) == 0 {
		fmt.Println(mutedStyle.Render("  No languages downloaded yet"))
	} else {
		for _, lang := range downloaded {
			files, _ := downloader.ListFiles(lang)
			fmt.Printf("  %s  %d parquet files\n",
				successStyle.Render(lang),
				len(files),
			)
		}
	}
	fmt.Println()

	// Show import status
	fmt.Println(infoStyle.Render("Import status:"))
	fmt.Println()

	entries, err := os.ReadDir(dbDir)
	if os.IsNotExist(err) {
		fmt.Println(mutedStyle.Render("  No databases created yet"))
	} else if err != nil {
		return fmt.Errorf("reading database directory: %w", err)
	} else {
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".duckdb") {
				continue
			}
			lang := strings.TrimSuffix(entry.Name(), ".duckdb")

			// Try to get document count
			engine, err := fineweb.NewEngine(fineweb.Config{
				DataDir:   dbDir,
				SourceDir: dataDir,
				Languages: []string{lang},
			})
			if err != nil {
				fmt.Printf("  %s  error: %v\n", errorStyle.Render(lang), err)
				continue
			}

			count, _ := engine.GetDocumentCount(ctx)
			engine.Close()

			fmt.Printf("  %s  %d documents\n",
				successStyle.Render(lang),
				count,
			)
		}
	}
	fmt.Println()

	return nil
}

func downloadData(ctx context.Context, langs []string) error {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, "data", "fineweb-2")

	downloader := fwdownloader.NewDownloader(fwdownloader.Config{DataDir: dataDir})

	fmt.Println(infoStyle.Render("Downloading FineWeb-2 data..."))
	fmt.Println()

	var lastFile string
	var lastLang string

	progress := func(p fwdownloader.DownloadProgress) {
		if p.Done {
			fmt.Printf("\r\033[K  %s  Download complete (%d files)\n",
				successStyle.Render(p.Language),
				p.TotalFiles)
			lastLang = ""
			return
		}
		if p.Error != nil {
			fmt.Printf("\r\033[K  %s  Error: %v\n", errorStyle.Render(p.Language), p.Error)
			return
		}

		// Print language header on first file
		if lastLang != p.Language {
			if lastLang != "" {
				fmt.Println()
			}
			fmt.Printf("  %s  Starting download (%d files)...\n",
				infoStyle.Render(p.Language),
				p.TotalFiles)
			lastLang = p.Language
		}

		if p.BytesReceived == p.TotalBytes && p.TotalBytes > 0 {
			// File complete - overwrite progress line
			fmt.Printf("\r\033[K    [%d/%d] %s %.1f MB %s\n",
				p.FileIndex,
				p.TotalFiles,
				successStyle.Render("OK"),
				float64(p.TotalBytes)/(1024*1024),
				p.CurrentFile,
			)
			lastFile = ""
		} else if lastFile != p.CurrentFile {
			// Starting new file
			fmt.Printf("    [%d/%d] Downloading %s...",
				p.FileIndex,
				p.TotalFiles,
				p.CurrentFile,
			)
			lastFile = p.CurrentFile
		}
	}

	if err := downloader.Download(ctx, langs, progress); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	fmt.Println()
	return nil
}

func importData(ctx context.Context, langs []string) error {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, "data", "fineweb-2")
	dbDir := filepath.Join(home, "data", "blueprints", "search", "fineweb-2")

	fmt.Println(infoStyle.Render("Importing data into DuckDB..."))
	fmt.Println()

	engine, err := fineweb.NewEngine(fineweb.Config{
		DataDir:   dbDir,
		SourceDir: dataDir,
	})
	if err != nil {
		return fmt.Errorf("creating engine: %w", err)
	}
	defer engine.Close()

	for _, lang := range langs {
		// Count parquet files for progress
		parquetDir := filepath.Join(dataDir, lang, "train")
		entries, _ := os.ReadDir(parquetDir)
		var fileCount int
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".parquet") {
				fileCount++
			}
		}

		fmt.Printf("  %s  Importing %d parquet files...\n", infoStyle.Render(lang), fileCount)

		var totalRows int64
		var fileIndex int
		progress := func(file string, rows int64) {
			fileIndex++
			totalRows += rows
			fmt.Printf("    [%d/%d] %s %s (%s rows, %s total)\n",
				fileIndex,
				fileCount,
				successStyle.Render("OK"),
				file,
				formatNumber(rows),
				formatNumber(totalRows),
			)
		}

		if err := engine.ImportLanguage(ctx, lang, progress); err != nil {
			fmt.Printf("  %s  Error: %v\n", errorStyle.Render(lang), err)
			continue
		}

		fmt.Printf("  %s  Import complete: %s documents\n",
			successStyle.Render(lang),
			formatNumber(totalRows))
	}

	fmt.Println()
	return nil
}

// formatNumber formats a number with thousand separators
func formatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000)
}
