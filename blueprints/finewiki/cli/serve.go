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

The server reads from the parquet file at <data-dir>/<lang>/data.parquet
and uses a DuckDB index at <data-dir>/<lang>/wiki.duckdb.

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
	parquetGlob := ParquetGlob(dataDir, lang)
	dbPath := DuckDBPath(dataDir, lang)

	// Check parquet exists (either single or sharded)
	if !parquetExists(dataDir, lang) {
		return fmt.Errorf("parquet file not found for language '%s'\nrun 'finewiki import %s' first", lang, lang)
	}

	// Ensure directory exists for duckdb
	langDir := LangDir(dataDir, lang)
	if err := os.MkdirAll(langDir, 0o755); err != nil {
		return err
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	store, err := duckdb.New(db)
	if err != nil {
		return err
	}

	if err := store.Ensure(ctx, duckdb.Config{
		ParquetGlob: parquetGlob,
		EnableFTS:   true,
	}, duckdb.EnsureOptions{
		SeedIfEmpty: true,
		BuildIndex:  true,
		BuildFTS:    true,
	}); err != nil {
		return err
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

	fmt.Fprintf(os.Stdout, "serving %s wiki on %s\n", lang, addr)
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
