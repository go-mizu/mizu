package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/go-mizu/mizu/blueprints/search/pkg/qlocal"
	"github.com/spf13/cobra"
)

type qlocalRootFlags struct {
	indexName  string
	dbPath     string
	configPath string
}

type qlocalSearchFlags struct {
	n           int
	minScore    float64
	all         bool
	full        bool
	lineNums    bool
	csv         bool
	md          bool
	xml         bool
	files       bool
	json        bool
	collections []string
}

func (f *qlocalSearchFlags) outputFormat() qlocal.OutputFormat {
	switch {
	case f.csv:
		return qlocal.OutputCSV
	case f.md:
		return qlocal.OutputMD
	case f.xml:
		return qlocal.OutputXML
	case f.files:
		return qlocal.OutputFiles
	case f.json:
		return qlocal.OutputJSON
	default:
		return qlocal.OutputCLI
	}
}

func (f *qlocalSearchFlags) limit(defaultCLI, defaultMachine int) int {
	if f.all {
		return 100000
	}
	if f.n > 0 {
		return f.n
	}
	switch f.outputFormat() {
	case qlocal.OutputFiles, qlocal.OutputJSON:
		return defaultMachine
	default:
		return defaultCLI
	}
}

func NewLocal() *cobra.Command {
	var rf qlocalRootFlags
	cmd := &cobra.Command{
		Use:   "local",
		Short: "Local markdown search (qmd-inspired) with collections, contexts, FTS, and hybrid retrieval",
		Long: `Local markdown search (qmd-inspired) for indexing markdown collections and searching them with:
- BM25 full-text search (SQLite FTS5)
- deterministic local vector search (hash embeddings)
- hybrid query fusion (RRF + chunk scoring)

This is a Go-native port scaffold under pkg/qlocal integrated into the Search blueprint CLI.`,
	}

	cmd.PersistentFlags().StringVar(&rf.indexName, "index", "index", "Index name (separate DB+config namespace)")
	cmd.PersistentFlags().StringVar(&rf.dbPath, "db", "", "Override qlocal SQLite DB path")
	cmd.PersistentFlags().StringVar(&rf.configPath, "config", "", "Override qlocal YAML config path")

	withApp := func(run func(*cobra.Command, []string, *qlocal.App) error) func(*cobra.Command, []string) error {
		return func(cmd *cobra.Command, args []string) error {
			app, err := qlocal.Open(qlocal.OpenOptions{
				IndexName:  rf.indexName,
				DBPath:     rf.dbPath,
				ConfigPath: rf.configPath,
			})
			if err != nil {
				return err
			}
			defer app.Close()
			return run(cmd, args, app)
		}
	}

	cmd.AddCommand(newLocalStatus(withApp))
	cmd.AddCommand(newLocalCollection(withApp))
	cmd.AddCommand(newLocalContext(withApp))
	cmd.AddCommand(newLocalUpdate(withApp))
	cmd.AddCommand(newLocalEmbed(withApp))
	cmd.AddCommand(newLocalSearchLike("search", "BM25 full-text search", withApp, localRunSearch))
	vsearchCmd := newLocalSearchLike("vsearch", "Vector semantic search", withApp, localRunVSearch)
	vsearchCmd.Aliases = []string{"vector-search"}
	cmd.AddCommand(vsearchCmd)
	queryCmd := newLocalSearchLike("query", "Hybrid search with RRF + rerank-style chunk scoring", withApp, localRunQuery)
	queryCmd.Aliases = []string{"deep-search"}
	cmd.AddCommand(queryCmd)
	cmd.AddCommand(newLocalGet(withApp))
	cmd.AddCommand(newLocalMultiGet(withApp))
	cmd.AddCommand(newLocalLS(withApp))
	cmd.AddCommand(newLocalCleanup(withApp))
	cmd.AddCommand(newLocalPull(withApp))
	cmd.AddCommand(newLocalMCP(withApp))

	return cmd
}

func newLocalStatus(withApp func(func(*cobra.Command, []string, *qlocal.App) error) func(*cobra.Command, []string) error) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show qlocal index and collection health",
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			st, err := app.Status()
			if err != nil {
				return err
			}
			if asJSON {
				b, _ := json.MarshalIndent(st, "", "  ")
				fmt.Println(string(b))
				return nil
			}
			fmt.Println(titleStyle.Render("qlocal status"))
			fmt.Printf("Index:   %s\n", app.DBPath())
			fmt.Printf("Config:  %s\n", app.ConfigPath())
			fmt.Printf("Docs:    %d\n", st.TotalDocuments)
			fmt.Printf("Vectors: %v (needs embedding: %d)\n", st.HasVectorIndex, st.NeedsEmbedding)
			if st.MCP.Running {
				fmt.Printf("MCP:     running (PID %d)\n", st.MCP.PID)
			} else {
				fmt.Printf("MCP:     not running\n")
			}
			if len(st.Collections) == 0 {
				fmt.Println("Collections: none")
				return nil
			}
			fmt.Println("Collections:")
			for _, c := range st.Collections {
				last := c.LastUpdate
				if last == "" {
					last = "-"
				}
				fmt.Printf("  - %-16s %6d docs  pattern=%s  path=%s  updated=%s\n", c.Name, c.Documents, c.Pattern, c.Path, last)
			}
			return nil
		}),
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output JSON")
	return cmd
}

func newLocalCollection(withApp func(func(*cobra.Command, []string, *qlocal.App) error) func(*cobra.Command, []string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collection",
		Short: "Manage indexed collections",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List collections",
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			items, err := app.CollectionList()
			if err != nil {
				return err
			}
			if len(items) == 0 {
				fmt.Println("No collections.")
				return nil
			}
			for _, it := range items {
				include := "yes"
				if it.IncludeByDefault != nil && !*it.IncludeByDefault {
					include = "no"
				}
				fmt.Printf("%s\n", it.Name)
				fmt.Printf("  path:    %s\n", it.Path)
				fmt.Printf("  pattern: %s\n", firstNonEmpty(it.Pattern, qlocal.DefaultGlob))
				fmt.Printf("  include: %s\n", include)
				if strings.TrimSpace(it.Update) != "" {
					fmt.Printf("  update:  %s\n", it.Update)
				}
				if len(it.Context) > 0 {
					fmt.Printf("  contexts: %d\n", len(it.Context))
				}
			}
			return nil
		}),
	})

	var addName, addMask string
	addCmd := &cobra.Command{
		Use:   "add <path>",
		Short: "Add or update a collection",
		Args:  cobra.MaximumNArgs(1),
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			p := "."
			if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
				p = args[0]
			}
			item, err := app.CollectionAdd(p, addName, addMask)
			if err != nil {
				return err
			}
			fmt.Printf("Added collection %q -> %s (%s)\n", item.Name, item.Path, item.Pattern)
			return nil
		}),
	}
	addCmd.Flags().StringVar(&addName, "name", "", "Collection name")
	addCmd.Flags().StringVar(&addMask, "mask", qlocal.DefaultGlob, "Glob pattern (default **/*.md)")
	cmd.AddCommand(addCmd)

	cmd.AddCommand(&cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm"},
		Short:   "Remove a collection",
		Args:    cobra.ExactArgs(1),
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			if err := app.CollectionRemove(args[0]); err != nil {
				return err
			}
			fmt.Printf("Removed collection %q\n", args[0])
			return nil
		}),
	})

	cmd.AddCommand(&cobra.Command{
		Use:     "rename <old> <new>",
		Aliases: []string{"mv"},
		Short:   "Rename a collection",
		Args:    cobra.ExactArgs(2),
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			if err := app.CollectionRename(args[0], args[1]); err != nil {
				return err
			}
			fmt.Printf("Renamed %q -> %q\n", args[0], args[1])
			return nil
		}),
	})

	cmd.AddCommand(&cobra.Command{
		Use:     "show <name>",
		Aliases: []string{"info"},
		Short:   "Show collection details",
		Args:    cobra.ExactArgs(1),
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			c, err := app.CollectionShow(args[0])
			if err != nil {
				return err
			}
			b, _ := json.MarshalIndent(c, "", "  ")
			fmt.Println(string(b))
			return nil
		}),
	})

	cmd.AddCommand(&cobra.Command{
		Use:     "update-cmd <name> [command]",
		Aliases: []string{"set-update"},
		Short:   "Set/clear pre-update shell command for a collection",
		Args:    cobra.MinimumNArgs(1),
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			var shellCmd string
			if len(args) > 1 {
				shellCmd = strings.Join(args[1:], " ")
			}
			if err := app.CollectionSetUpdate(args[0], shellCmd); err != nil {
				return err
			}
			if shellCmd == "" {
				fmt.Printf("Cleared update command for %q\n", args[0])
			} else {
				fmt.Printf("Set update command for %q: %s\n", args[0], shellCmd)
			}
			return nil
		}),
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "include <name>",
		Short: "Include collection in default queries",
		Args:  cobra.ExactArgs(1),
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			if err := app.CollectionSetIncludeByDefault(args[0], true); err != nil {
				return err
			}
			fmt.Printf("Collection %q included in default queries\n", args[0])
			return nil
		}),
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "exclude <name>",
		Short: "Exclude collection from default queries",
		Args:  cobra.ExactArgs(1),
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			if err := app.CollectionSetIncludeByDefault(args[0], false); err != nil {
				return err
			}
			fmt.Printf("Collection %q excluded from default queries\n", args[0])
			return nil
		}),
	})

	return cmd
}

func newLocalContext(withApp func(func(*cobra.Command, []string, *qlocal.App) error) func(*cobra.Command, []string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manage collection/global context annotations",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all contexts",
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			items, err := app.ContextList()
			if err != nil {
				return err
			}
			if len(items) == 0 {
				fmt.Println("No contexts.")
				return nil
			}
			for _, it := range items {
				fmt.Printf("%s %s -> %s\n", it.Collection, it.Path, it.Context)
			}
			return nil
		}),
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "add [path] <text>",
		Short: "Add context (defaults to current directory)",
		Args:  cobra.MinimumNArgs(1),
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			cwd, _ := os.Getwd()
			var pathArg, text string
			if len(args) == 1 {
				text = args[0]
			} else {
				pathArg = args[0]
				text = strings.Join(args[1:], " ")
			}
			target, err := app.ContextAdd(pathArg, text, cwd)
			if err != nil {
				return err
			}
			fmt.Printf("Context set for %s\n", target)
			return nil
		}),
	})
	cmd.AddCommand(&cobra.Command{
		Use:     "rm <path>",
		Aliases: []string{"remove"},
		Short:   "Remove context by path target",
		Args:    cobra.ExactArgs(1),
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			cwd, _ := os.Getwd()
			target, err := app.ContextRemove(args[0], cwd)
			if err != nil {
				return err
			}
			fmt.Printf("Context removed for %s\n", target)
			return nil
		}),
	})
	return cmd
}

func newLocalUpdate(withApp func(func(*cobra.Command, []string, *qlocal.App) error) func(*cobra.Command, []string) error) *cobra.Command {
	var pull bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Scan collections and update the local index",
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			stats, err := app.Update(cmd.Context(), qlocal.UpdateOptions{Pull: pull})
			if err != nil {
				return err
			}
			fmt.Printf("Collections: %d\nScanned: %d\nAdded: %d\nUpdated: %d\nUnchanged: %d\nDeactivated: %d\nErrors: %d\nDuration: %dms\n",
				stats.Collections, stats.Scanned, stats.Added, stats.Updated, stats.Unchanged, stats.Deactivated, stats.Errors, stats.DurationMS)
			return nil
		}),
	}
	cmd.Flags().BoolVar(&pull, "pull", false, "Run collection update command before indexing")
	return cmd
}

func newLocalEmbed(withApp func(func(*cobra.Command, []string, *qlocal.App) error) func(*cobra.Command, []string) error) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "embed",
		Short: "Generate deterministic local embeddings for vector search",
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			stats, err := app.Embed(cmd.Context(), qlocal.EmbedOptions{Force: force})
			if err != nil {
				return err
			}
			fmt.Printf("Documents embedded: %d\nChunks embedded: %d\nErrors: %d\nDuration: %dms\n", stats.Documents, stats.Chunks, stats.Errors, stats.DurationMS)
			return nil
		}),
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Clear embeddings and rebuild")
	return cmd
}

type localSearchRunner func(*cobra.Command, *qlocal.App, string, qlocalSearchFlags) error

func newLocalSearchLike(name, short string, withApp func(func(*cobra.Command, []string, *qlocal.App) error) func(*cobra.Command, []string) error, runner localSearchRunner) *cobra.Command {
	var f qlocalSearchFlags
	cmd := &cobra.Command{
		Use:   name + " [query]",
		Short: short,
		Args:  cobra.MinimumNArgs(1),
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			q := strings.Join(args, " ")
			return runner(cmd, app, q, f)
		}),
	}
	addLocalSearchFlags(cmd, &f)
	return cmd
}

func addLocalSearchFlags(cmd *cobra.Command, f *qlocalSearchFlags) {
	cmd.Flags().IntVarP(&f.n, "n", "n", 0, "Max results")
	cmd.Flags().Float64Var(&f.minScore, "min-score", 0, "Minimum score")
	cmd.Flags().BoolVar(&f.all, "all", false, "Return all matches (large limit)")
	cmd.Flags().BoolVar(&f.full, "full", false, "Return full document body")
	cmd.Flags().BoolVar(&f.lineNums, "line-numbers", false, "Include line numbers")
	cmd.Flags().BoolVar(&f.csv, "csv", false, "CSV output")
	cmd.Flags().BoolVar(&f.md, "md", false, "Markdown output")
	cmd.Flags().BoolVar(&f.xml, "xml", false, "XML output")
	cmd.Flags().BoolVar(&f.files, "files", false, "Files list output")
	cmd.Flags().BoolVar(&f.json, "json", false, "JSON output")
	cmd.Flags().StringSliceVarP(&f.collections, "collection", "c", nil, "Filter to collection(s)")
}

func localRunSearch(cmd *cobra.Command, app *qlocal.App, q string, f qlocalSearchFlags) error {
	results, err := app.SearchFTS(q, qlocal.SearchOptions{
		Limit:       f.limit(5, 20),
		MinScore:    f.minScore,
		Collections: f.collections,
		IncludeBody: true,
	})
	if err != nil {
		return err
	}
	return printSearchResults(results, q, f)
}

func localRunVSearch(cmd *cobra.Command, app *qlocal.App, q string, f qlocalSearchFlags) error {
	minScore := f.minScore
	if !cmd.Flags().Changed("min-score") {
		minScore = 0.3
	}
	results, err := app.VectorSearch(q, qlocal.SearchOptions{
		Limit:       f.limit(5, 20),
		MinScore:    minScore,
		Collections: f.collections,
		IncludeBody: true,
	})
	if err != nil {
		return err
	}
	return printSearchResults(results, q, f)
}

func localRunQuery(cmd *cobra.Command, app *qlocal.App, q string, f qlocalSearchFlags) error {
	results, err := app.QueryContext(cmd.Context(), q, qlocal.HybridOptions{
		Limit:       f.limit(5, 20),
		MinScore:    f.minScore,
		Collections: f.collections,
	})
	if err != nil {
		return err
	}
	return printSearchResults(results, q, f)
}

func printSearchResults(results []qlocal.SearchResult, q string, f qlocalSearchFlags) error {
	out, err := qlocal.FormatSearchResults(results, qlocal.OutputOptions{
		Format:      f.outputFormat(),
		Full:        f.full,
		LineNumbers: f.lineNums,
		Query:       q,
	})
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

func newLocalGet(withApp func(func(*cobra.Command, []string, *qlocal.App) error) func(*cobra.Command, []string) error) *cobra.Command {
	var from int
	var maxLines int
	var lineNumbers bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "get <file>[:line]",
		Short: "Get a single indexed document by path or #docid",
		Args:  cobra.ExactArgs(1),
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			doc, err := app.Get(args[0], qlocal.GetOptions{
				FromLine:    from,
				MaxLines:    maxLines,
				Full:        true,
				LineNumbers: lineNumbers && !asJSON,
			})
			if err != nil {
				return err
			}
			if asJSON {
				b, _ := json.MarshalIndent(doc, "", "  ")
				fmt.Println(string(b))
				return nil
			}
			fmt.Printf("%s (%s)\n", doc.DisplayPath, "#"+doc.DocID)
			if doc.Context != "" {
				fmt.Printf("Context: %s\n", doc.Context)
			}
			fmt.Println(doc.Body)
			return nil
		}),
	}
	cmd.Flags().IntVar(&from, "from", 0, "Start line (1-based)")
	cmd.Flags().IntVarP(&maxLines, "lines", "l", 0, "Max lines")
	cmd.Flags().BoolVar(&lineNumbers, "line-numbers", false, "Include line numbers")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output JSON")
	return cmd
}

func newLocalMultiGet(withApp func(func(*cobra.Command, []string, *qlocal.App) error) func(*cobra.Command, []string) error) *cobra.Command {
	var maxLines int
	var maxBytes int
	var lineNumbers bool
	var f qlocalSearchFlags
	cmd := &cobra.Command{
		Use:   "multi-get <pattern>",
		Short: "Get multiple documents by glob or comma-separated list",
		Args:  cobra.ExactArgs(1),
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			results, errs, err := app.MultiGet(args[0], maxLines, maxBytes, true)
			if err != nil {
				return err
			}
			for _, e := range errs {
				fmt.Fprintln(os.Stderr, warningStyle.Render(e))
			}
			out, err := qlocal.FormatMultiGet(results, f.outputFormat(), lineNumbers)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		}),
	}
	cmd.Flags().IntVarP(&maxLines, "lines", "l", 0, "Max lines per file")
	cmd.Flags().IntVar(&maxBytes, "max-bytes", qlocal.DefaultMultiGetMaxBytes, "Skip files larger than this many bytes")
	cmd.Flags().BoolVar(&lineNumbers, "line-numbers", false, "Include line numbers")
	// Reuse format flags only
	cmd.Flags().BoolVar(&f.csv, "csv", false, "CSV output")
	cmd.Flags().BoolVar(&f.md, "md", false, "Markdown output")
	cmd.Flags().BoolVar(&f.xml, "xml", false, "XML output")
	cmd.Flags().BoolVar(&f.files, "files", false, "Files list output")
	cmd.Flags().BoolVar(&f.json, "json", false, "JSON output")
	return cmd
}

func newLocalLS(withApp func(func(*cobra.Command, []string, *qlocal.App) error) func(*cobra.Command, []string) error) *cobra.Command {
	var limit int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "ls [collection[/prefix]]",
		Short: "List indexed files",
		Args:  cobra.MaximumNArgs(1),
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			filter := ""
			if len(args) > 0 {
				filter = args[0]
			}
			items, err := app.List(filter, limit)
			if err != nil {
				return err
			}
			if asJSON {
				b, _ := json.MarshalIndent(items, "", "  ")
				fmt.Println(string(b))
				return nil
			}
			for _, it := range items {
				fmt.Printf("%s (%s, %d bytes)\n", it.DisplayPath, "#"+it.DocID, it.BodyLength)
			}
			return nil
		}),
	}
	cmd.Flags().IntVar(&limit, "limit", 200, "Max files to list (0 = all)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output JSON")
	return cmd
}

func newLocalCleanup(withApp func(func(*cobra.Command, []string, *qlocal.App) error) func(*cobra.Command, []string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Remove caches/orphans and vacuum the qlocal DB",
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			stats, err := app.Cleanup(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Printf("llm_cache deleted: %d\ninactive docs deleted: %d\norphaned content deleted: %d\norphaned vectors deleted: %d\n",
				stats.LLMCacheDeleted, stats.InactiveDeleted, stats.OrphanedContent, stats.OrphanedVectors)
			return nil
		}),
	}
	return cmd
}

func newLocalPull(withApp func(func(*cobra.Command, []string, *qlocal.App) error) func(*cobra.Command, []string) error) *cobra.Command {
	var refresh bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Download/cache qmd-compatible GGUF model files from HuggingFace",
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			results, err := app.Pull(cmd.Context(), qlocal.PullOptions{Refresh: refresh})
			if err != nil {
				return err
			}
			if asJSON {
				b, _ := json.MarshalIndent(results, "", "  ")
				fmt.Println(string(b))
				return nil
			}
			if len(results) == 0 {
				fmt.Println("No models.")
				return nil
			}
			for _, r := range results {
				note := "cached/checked"
				if r.Refreshed {
					note = "downloaded"
				}
				fmt.Printf("- %s -> %s (%d bytes, %s)\n", r.Model, r.Path, r.SizeBytes, note)
			}
			return nil
		}),
	}
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Refresh models")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output JSON")
	return cmd
}

func newLocalMCP(withApp func(func(*cobra.Command, []string, *qlocal.App) error) func(*cobra.Command, []string) error) *cobra.Command {
	var http bool
	var daemon bool
	var port int
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start qlocal MCP-compatible server (stdio or HTTP)",
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			if http {
				addr := fmt.Sprintf("127.0.0.1:%d", port)
				if daemon {
					return startLocalMCPDaemon(cmd, addr)
				}
				fmt.Fprintf(os.Stderr, "qlocal MCP HTTP listening on http://%s/mcp\n", addr)
				return qlocal.StartMCPHTTPServer(cmd.Context(), app, addr)
			}
			return qlocal.ServeMCPStdio(cmd.Context(), app, os.Stdin, os.Stdout)
		}),
	}
	cmd.Flags().BoolVar(&http, "http", false, "Use HTTP transport")
	cmd.Flags().BoolVar(&daemon, "daemon", false, "Run in background")
	cmd.Flags().IntVar(&port, "port", 8181, "HTTP port")
	cmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stop MCP daemon started with --http --daemon",
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			pidPath := qlocal.MCPPIDPathForIndex("index")
			if strings.TrimSpace(app.IndexName) != "" {
				pidPath = qlocal.MCPPIDPathForIndex(app.IndexName)
			}
			data, err := os.ReadFile(pidPath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("Not running (no PID file).")
					return nil
				}
				return err
			}
			pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
			if pid <= 0 {
				_ = os.Remove(pidPath)
				fmt.Println("Removed invalid PID file.")
				return nil
			}
			proc, err := os.FindProcess(pid)
			if err == nil {
				_ = proc.Signal(syscall.SIGTERM)
			}
			_ = os.Remove(pidPath)
			fmt.Printf("Stopped qlocal MCP daemon (PID %d)\n", pid)
			return nil
		}),
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show MCP daemon status",
		RunE: withApp(func(cmd *cobra.Command, args []string, app *qlocal.App) error {
			st := qlocal.GetMCPDaemonStatus(app.IndexName)
			if st.Running {
				fmt.Printf("qlocal MCP daemon running (PID %d)\nPID file: %s\n", st.PID, st.PIDPath)
			} else {
				fmt.Printf("qlocal MCP daemon not running\nPID file: %s\n", st.PIDPath)
			}
			return nil
		}),
	})
	return cmd
}

func startLocalMCPDaemon(cmd *cobra.Command, addr string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	pidPath := qlocal.MCPPIDPathForIndex("index")
	if idx, _ := cmd.Flags().GetString("index"); strings.TrimSpace(idx) != "" {
		pidPath = qlocal.MCPPIDPathForIndex(idx)
	}
	if data, err := os.ReadFile(pidPath); err == nil {
		if pid, _ := strconv.Atoi(strings.TrimSpace(string(data))); pid > 0 {
			if p, e := os.FindProcess(pid); e == nil {
				if sigErr := p.Signal(syscall.Signal(0)); sigErr == nil {
					return fmt.Errorf("MCP daemon already running (PID %d)", pid)
				}
			}
		}
		_ = os.Remove(pidPath)
	}
	logDir := filepath.Dir(pidPath)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return err
	}
	logPath := filepath.Join(logDir, "mcp.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	args := []string{"local"}
	if idx, _ := cmd.Flags().GetString("index"); strings.TrimSpace(idx) != "" {
		args = append(args, "--index", idx)
	}
	if dbp, _ := cmd.Flags().GetString("db"); strings.TrimSpace(dbp) != "" {
		args = append(args, "--db", dbp)
	}
	if cfg, _ := cmd.Flags().GetString("config"); strings.TrimSpace(cfg) != "" {
		args = append(args, "--config", cfg)
	}
	args = append(args, "mcp", "--http", "--port", strings.TrimPrefix(addr, "127.0.0.1:"))
	child := exec.Command(exe, args...)
	child.Stdout = logFile
	child.Stderr = logFile
	child.Stdin = nil
	child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := child.Start(); err != nil {
		_ = logFile.Close()
		return err
	}
	if child.Process != nil {
		_ = child.Process.Release()
	}
	_ = logFile.Close()
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(child.Process.Pid)+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Printf("Started qlocal MCP daemon on http://%s/mcp (PID %d)\n", addr, child.Process.Pid)
	fmt.Printf("Logs: %s\n", logPath)
	return nil
}

func firstNonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

// localRunSearch-like functions above expect args as one query string. For structured query docs
// containing newlines, the user can pass shell-escaped newlines; Cobra preserves them in args.
