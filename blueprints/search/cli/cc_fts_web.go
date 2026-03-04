package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web"
	"github.com/spf13/cobra"
)

func newCCFTSWeb() *cobra.Command {
	var (
		port    int
		engine  string
		crawlID string
		addr    string
		open    bool
	)

	cmd := &cobra.Command{
		Use:   "web",
		Short: "Launch web GUI for FTS search and markdown browsing",
		Long: `Start an embedded HTTP server with a modern web interface for
searching the FTS index and browsing/previewing markdown documents.`,
		Example: `  search cc fts web
  search cc fts web --port 8080 --engine sqlite
  search cc fts web --open`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if crawlID == "" {
				crawlID = detectLatestCrawl()
			}

			homeDir, _ := os.UserHomeDir()
			baseDir := filepath.Join(homeDir, "data", "common-crawl", crawlID)

			srv := web.New(engine, crawlID, addr, baseDir)

			url := fmt.Sprintf("http://localhost:%d", port)
			fmt.Fprintf(os.Stderr, "FTS Web GUI\n")
			fmt.Fprintf(os.Stderr, "  url:     %s\n", url)
			fmt.Fprintf(os.Stderr, "  engine:  %s\n", engine)
			fmt.Fprintf(os.Stderr, "  crawl:   %s\n", crawlID)
			fmt.Fprintf(os.Stderr, "  data:    %s\n", baseDir)
			fmt.Fprintf(os.Stderr, "\nPress Ctrl+C to stop.\n")

			if open {
				openBrowser(url)
			}

			return srv.ListenAndServe(cmd.Context(), port)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 3456, "Listen port")
	cmd.Flags().StringVar(&engine, "engine", "tantivy", "FTS engine")
	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&addr, "addr", "", "External engine address")
	cmd.Flags().BoolVar(&open, "open", false, "Open browser on start")
	return cmd
}

func newCCFTSDashboard() *cobra.Command {
	var (
		port            int
		engine          string
		crawlID         string
		addr            string
		open            bool
		metaDriver      string
		metaDSN         string
		metaRefreshTTL  time.Duration
		metaPrewarm     bool
		metaBusyTimeout time.Duration
		metaJournalMode string
	)

	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Launch admin dashboard for FTS pipeline management",
		Long: `Start the FTS dashboard — a web interface for managing the full
CC FTS pipeline: download WARCs, extract markdown, pack data, build indexes,
search, and browse documents. Real-time progress via WebSocket.`,
		Example: `  search cc fts dashboard
  search cc fts dashboard --port 8080
  search cc fts dashboard --open`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if crawlID == "" {
				crawlID = detectLatestCrawl()
			}
			homeDir, _ := os.UserHomeDir()
			baseDir := filepath.Join(homeDir, "data", "common-crawl", crawlID)

			srv := web.NewDashboardWithOptions(engine, crawlID, addr, baseDir, web.DashboardOptions{
				MetaDriver:      metaDriver,
				MetaDSN:         metaDSN,
				MetaRefreshTTL:  metaRefreshTTL,
				MetaPrewarm:     metaPrewarm,
				MetaBusyTimeout: metaBusyTimeout,
				MetaJournalMode: metaJournalMode,
			})

			url := fmt.Sprintf("http://localhost:%d", port)
			fmt.Fprintf(os.Stderr, "FTS Dashboard\n")
			fmt.Fprintf(os.Stderr, "  url:     %s\n", url)
			fmt.Fprintf(os.Stderr, "  engine:  %s\n", engine)
			fmt.Fprintf(os.Stderr, "  crawl:   %s\n", crawlID)
			fmt.Fprintf(os.Stderr, "  data:    %s\n", baseDir)
			fmt.Fprintf(os.Stderr, "  meta:    driver=%s ttl=%s\n", metaDriver, metaRefreshTTL)
			fmt.Fprintf(os.Stderr, "\nPress Ctrl+C to stop.\n")

			if open {
				openBrowser(url)
			}

			return srv.ListenAndServe(cmd.Context(), port)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 3456, "Listen port")
	cmd.Flags().StringVar(&engine, "engine", "duckdb", "Default FTS engine")
	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&addr, "addr", "", "External engine address")
	cmd.Flags().BoolVar(&open, "open", false, "Open browser on start")
	cmd.Flags().StringVar(&metaDriver, "meta-driver", "sqlite", "Metadata cache driver: sqlite, duckdb, none")
	cmd.Flags().StringVar(&metaDSN, "meta-dsn", "", "Metadata cache DB path (default: ~/data/common-crawl/.meta/dashboard_meta.*)")
	cmd.Flags().DurationVar(&metaRefreshTTL, "meta-refresh-ttl", 30*time.Second, "Metadata stale threshold for background refresh")
	cmd.Flags().BoolVar(&metaPrewarm, "meta-prewarm", true, "Prewarm metadata cache for active crawl on startup")
	cmd.Flags().DurationVar(&metaBusyTimeout, "meta-busy-timeout", 5*time.Second, "Metadata DB busy timeout")
	cmd.Flags().StringVar(&metaJournalMode, "meta-journal-mode", "WAL", "Metadata DB journal mode (sqlite)")
	return cmd
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return
	}
	cmd.Start()
}
