package cli

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/finewiki/app/web"
	"github.com/go-mizu/blueprints/finewiki/feature/search"
	"github.com/go-mizu/blueprints/finewiki/feature/view"
	"github.com/go-mizu/blueprints/finewiki/store/duckdb"

	_ "github.com/duckdb/duckdb-go/v2"
)

func serveCmd() *cobra.Command {
	var (
		addr    string
		dataDir string
	)

	c := &cobra.Command{
		Use:   "serve <lang>",
		Short: "Start the web server for a specific language",
		Long: `Start the FineWiki web server for a specific language.

The server reads from the DuckDB database at <data-dir>/<lang>/wiki.duckdb.
Run 'finewiki import <lang>' first to download and prepare the data.

Examples:
  finewiki serve vi                  # Serve Vietnamese wiki on :8080
  finewiki serve en --addr :3000     # Serve English wiki on port 3000
  finewiki serve ja --data ~/wiki    # Use custom data directory`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			lang := args[0]
			return runServe(cmd.Context(), addr, dataDir, lang)
		},
	}

	c.Flags().StringVar(&addr, "addr", ":8080", "HTTP listen address")
	c.Flags().StringVar(&dataDir, "data", DefaultDataDir(), "Base data directory")

	return c
}

func runServe(ctx context.Context, addr, dataDir, lang string) error {
	ui := NewUI()
	parquetGlob := ParquetGlob(dataDir, lang)
	dbPath := DuckDBPath(dataDir, lang)

	// Check if database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// Check if parquet exists but database doesn't
		if parquetExists(dataDir, lang) {
			ui.Error(fmt.Sprintf("Database not initialized for '%s'", lang))
			ui.Blank()
			ui.Hint("Parquet data exists but database hasn't been created.")
			ui.Hint(fmt.Sprintf("Run: finewiki import %s", lang))
			return fmt.Errorf("database not initialized")
		}

		ui.Error(fmt.Sprintf("No data found for '%s'", lang))
		ui.Blank()
		ui.Hint("Download the data first:")
		ui.Hint(fmt.Sprintf("Run: finewiki import %s", lang))
		return fmt.Errorf("data not found")
	}

	// Ensure directory exists for duckdb
	langDir := LangDir(dataDir, lang)
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		return err
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		ui.Error("Failed to open database")
		ui.Hint(err.Error())
		return err
	}
	defer db.Close()

	store, err := duckdb.New(db)
	if err != nil {
		return err
	}

	// Only verify schema exists, don't seed
	if err := store.Ensure(ctx, duckdb.Config{
		ParquetGlob: parquetGlob,
		EnableFTS:   true,
	}, duckdb.EnsureOptions{
		SeedIfEmpty: false, // Don't seed during serve
		BuildIndex:  false, // Already built during import
		BuildFTS:    false,
	}); err != nil {
		return err
	}

	// Verify database has data
	stats, _ := store.Stats(ctx)
	pageCount, _ := stats["pages"].(int64)
	if pageCount == 0 {
		ui.Error(fmt.Sprintf("Database is empty for '%s'", lang))
		ui.Blank()
		ui.Hint("Re-import the data:")
		ui.Hint(fmt.Sprintf("Run: finewiki import %s", lang))
		return fmt.Errorf("empty database")
	}

	tmpl, err := web.NewTemplates()
	if err != nil {
		return err
	}

	searchSvc := search.New(store)
	viewSvc := view.New(store)

	srv := web.New(viewSvc, searchSvc, tmpl)

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Print server info
	ui.Header(iconServer, "FineWiki Server")
	ui.Info("Language", fmt.Sprintf("%s", lang))
	ui.Info("Articles", formatNumber(pageCount))
	ui.Info("Database", dbPath)
	ui.Blank()
	ui.Info("Listening", fmt.Sprintf("http://localhost%s", addr))
	ui.Blank()
	ui.Hint("Press Ctrl+C to stop")
	ui.Blank()

	return httpSrv.ListenAndServe()
}

// parquetExists checks if parquet files exist for a language.
func parquetExists(dataDir, lang string) bool {
	// Check single file
	if _, err := os.Stat(ParquetPath(dataDir, lang)); err == nil {
		return true
	}

	// Check sharded files
	pattern := filepath.Join(LangDir(dataDir, lang), "data-*.parquet")
	matches, _ := filepath.Glob(pattern)
	return len(matches) > 0
}
