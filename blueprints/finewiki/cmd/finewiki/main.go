package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/finewiki/app/web"
	"github.com/go-mizu/blueprints/finewiki/feature/search"
	"github.com/go-mizu/blueprints/finewiki/feature/view"
	"github.com/go-mizu/blueprints/finewiki/store/duckdb"

	_ "github.com/duckdb/duckdb-go/v2"
)

//go:embed views/**/*
var viewsFS embed.FS

type templates struct {
	t *template.Template
}

func newTemplates() (*templates, error) {
	funcs := template.FuncMap{
		"dict": dict,
	}

	t := template.New("views").Funcs(funcs)

	patterns := []string{
		"views/layout/*.html",
		"views/component/*.html",
		"views/page/*.html",
	}

	var err error
	for _, p := range patterns {
		t, err = t.ParseFS(viewsFS, p)
		if err != nil {
			return nil, err
		}
	}

	return &templates{t: t}, nil
}

func (x *templates) Render(w any, name string, data any) error {
	ww, ok := w.(io.Writer)
	if !ok {
		return errors.New("templates: writer does not implement io.Writer")
	}
	return x.t.ExecuteTemplate(ww, name, data)
}

func dict(kv ...any) (map[string]any, error) {
	if len(kv)%2 != 0 {
		return nil, errors.New("dict: odd args")
	}
	m := make(map[string]any, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		k, ok := kv[i].(string)
		if !ok {
			return nil, errors.New("dict: key is not string")
		}
		m[k] = kv[i+1]
	}
	return m, nil
}

func main() {
	root := &cobra.Command{
		Use:   "finewiki",
		Short: "FineWiki: fast read-only wiki viewer (DuckDB + Parquet)",
		Long:  "FineWiki serves FineWiki Parquet shards with fast title search and SSR pages.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(cmd.Context(), cmd)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.SetVersionTemplate("finewiki {{.Version}}\n")
	root.Version = versionString()

	root.Flags().String("addr", envDefault("FINEWIKI_ADDR", ":8080"), "HTTP listen address")
	root.Flags().String("db", envDefault("FINEWIKI_DUCKDB", "finewiki.duckdb"), "DuckDB database file")
	root.Flags().String("parquet", envDefault("FINEWIKI_PARQUET", ""), "Parquet path or glob (required)")
	root.Flags().Bool("fts", envBool("FINEWIKI_FTS", false), "Enable DuckDB FTS fallback for title search")

	root.AddCommand(importCmd())
	root.AddCommand(listCmd())

	if err := fang.Execute(context.Background(), root); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func runServe(ctx context.Context, cmd *cobra.Command) error {
	addr, _ := cmd.Flags().GetString("addr")
	dbPath, _ := cmd.Flags().GetString("db")
	parquetGlob, _ := cmd.Flags().GetString("parquet")
	enableFTS, _ := cmd.Flags().GetBool("fts")

	if strings.TrimSpace(parquetGlob) == "" {
		return errors.New("missing --parquet (or FINEWIKI_PARQUET)")
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil && filepath.Dir(dbPath) != "." {
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
		EnableFTS:   enableFTS,
	}, duckdb.EnsureOptions{
		SeedIfEmpty: true,
		BuildIndex:  true,
		BuildFTS:    enableFTS,
	}); err != nil {
		return err
	}

	tmpl, err := newTemplates()
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

	fmt.Fprintf(os.Stdout, "listening on %s\n", addr)
	return httpSrv.ListenAndServe()
}

func importCmd() *cobra.Command {
	var outDir string

	c := &cobra.Command{
		Use:   "import <path-or-url>",
		Short: "Import a Parquet shard into a local directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := duckdb.ImportParquet(cmd.Context(), args[0], outDir)
			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout, p)
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	c.Flags().StringVar(&outDir, "dir", envDefault("FINEWIKI_DATA", "data"), "Destination directory")
	return c
}

func listCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list <hf-dataset>",
		Short: "List Parquet shard URLs for a Hugging Face dataset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			urls, err := duckdb.ListParquet(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			for _, u := range urls {
				fmt.Fprintln(os.Stdout, u)
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	return c
}

func versionString() string {
	if v := os.Getenv("FINEWIKI_VERSION"); strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			return bi.Main.Version
		}
	}
	return "dev"
}

func envDefault(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func envBool(k string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "t", "true", "y", "yes", "on":
		return true
	case "0", "f", "false", "n", "no", "off":
		return false
	default:
		return def
	}
}
