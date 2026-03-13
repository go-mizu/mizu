package cli

import (
	"bufio"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/scrape/goodread"
	"github.com/spf13/cobra"
)

// NewGoodread creates the goodread CLI command.
func NewGoodread() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "goodread",
		Short: "Scrape Goodreads books, authors, lists, series, quotes, and more",
		Long: `Scrape public Goodreads data into a local DuckDB database.

Supports books, authors, series, lists, quotes, users, genres, and shelves.
Data is stored in $HOME/data/goodread/goodread.duckdb.

Examples:
  search goodread book 2767052            # Fetch a single book (The Hunger Games)
  search goodread book https://www.goodreads.com/book/show/2767052
  search goodread author 153394           # Fetch an author
  search goodread search "brandon sanderson"  # Search and enqueue results
  search goodread sitemap --limit 1000    # Seed queue from Goodreads sitemap
  search goodread crawl --workers 2       # Bulk crawl the queue
  search goodread info                    # Show database stats`,
	}

	cmd.AddCommand(newGoodreadBook())
	cmd.AddCommand(newGoodreadAuthor())
	cmd.AddCommand(newGoodreadSeries())
	cmd.AddCommand(newGoodreadList())
	cmd.AddCommand(newGoodreadQuote())
	cmd.AddCommand(newGoodreadUser())
	cmd.AddCommand(newGoodreadGenre())
	cmd.AddCommand(newGoodreadShelf())
	cmd.AddCommand(newGoodreadSearch())
	cmd.AddCommand(newGoodreadSitemap())
	cmd.AddCommand(newGoodreadCrawl())
	cmd.AddCommand(newGoodreadInfo())
	cmd.AddCommand(newGoodreadJobs())
	cmd.AddCommand(newGoodreadQueue())

	return cmd
}

// ── Shared flag helpers ───────────────────────────────────────────────────────

func addDBFlags(cmd *cobra.Command, dbPath, statePath *string, delay *int) {
	cfg := goodread.DefaultConfig()
	cmd.Flags().StringVar(dbPath, "db", cfg.DBPath, "Path to goodread.duckdb")
	cmd.Flags().StringVar(statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(delay, "delay", int(cfg.Delay/time.Millisecond), "Delay between requests in milliseconds")
}

func openDBs(dbPath, statePath string, delay int) (*goodread.DB, *goodread.State, *goodread.Client, error) {
	cfg := goodread.DefaultConfig()
	cfg.DBPath = dbPath
	cfg.StatePath = statePath
	cfg.Delay = time.Duration(delay) * time.Millisecond

	db, err := goodread.OpenDB(dbPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open db: %w", err)
	}

	stateDB, err := goodread.OpenState(statePath)
	if err != nil {
		db.Close()
		return nil, nil, nil, fmt.Errorf("open state: %w", err)
	}

	client := goodread.NewClient(cfg)
	return db, stateDB, client, nil
}

// ── book ─────────────────────────────────────────────────────────────────────

func newGoodreadBook() *cobra.Command {
	var dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "book <id|url>",
		Short: "Fetch a single Goodreads book",
		Args:  cobra.ExactArgs(1),
		Example: `  search goodread book 2767052
  search goodread book https://www.goodreads.com/book/show/2767052`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			url := normalizeBookURL(args[0])
			fmt.Printf("Fetching %s ...\n", url)

			task := &goodread.BookTask{
				URL:     url,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *goodread.BookState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addDBFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

// ── author ────────────────────────────────────────────────────────────────────

func newGoodreadAuthor() *cobra.Command {
	var dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "author <id|url>",
		Short: "Fetch a single Goodreads author",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			url := normalizeAuthorURL(args[0])
			fmt.Printf("Fetching %s ...\n", url)

			task := &goodread.AuthorTask{
				URL:     url,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *goodread.AuthorState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addDBFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

// ── series ────────────────────────────────────────────────────────────────────

func newGoodreadSeries() *cobra.Command {
	var dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "series <id|url>",
		Short: "Fetch a single Goodreads series",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			url := normalizeSeriesURL(args[0])
			fmt.Printf("Fetching %s ...\n", url)

			task := &goodread.SeriesTask{
				URL:     url,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *goodread.SeriesState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addDBFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

// ── list ─────────────────────────────────────────────────────────────────────

func newGoodreadList() *cobra.Command {
	var dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "list <id|url>",
		Short: "Fetch a single Goodreads listopia list",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			url := normalizeListURL(args[0])
			fmt.Printf("Fetching %s ...\n", url)

			task := &goodread.ListTask{
				URL:     url,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *goodread.ListState) {
				fmt.Printf("  [%s] %s (books=%d)\n", s.Status, s.URL, s.BooksFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addDBFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

// ── quote ─────────────────────────────────────────────────────────────────────

func newGoodreadQuote() *cobra.Command {
	var dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "quote <url>",
		Short: "Fetch quotes from a Goodreads quotes page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			url := args[0]
			task := &goodread.QuoteTask{
				URL:     url,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *goodread.QuoteState) {
				fmt.Printf("  [%s] %s (quotes=%d)\n", s.Status, s.URL, s.QuotesFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addDBFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

// ── user ─────────────────────────────────────────────────────────────────────

func newGoodreadUser() *cobra.Command {
	var dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "user <id|username>",
		Short: "Fetch a Goodreads user profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			url := normalizeUserURL(args[0])
			task := &goodread.UserTask{
				URL:     url,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *goodread.UserState) {
				fmt.Printf("  [%s] %s\n", s.Status, s.URL)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addDBFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

// ── genre ─────────────────────────────────────────────────────────────────────

func newGoodreadGenre() *cobra.Command {
	var dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "genre <slug>",
		Short: "Fetch a Goodreads genre page (e.g. fantasy, science-fiction)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, client, err := openDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()

			slug := args[0]
			url := goodread.BaseURL + "/genres/" + slug

			task := &goodread.GenreTask{
				URL:     url,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}
			m, err := task.Run(cmd.Context(), func(s *goodread.GenreState) {
				fmt.Printf("  [%s] %s (books=%d)\n", s.Status, s.URL, s.BooksFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d skipped=%d failed=%d\n", m.Fetched, m.Skipped, m.Failed)
			return nil
		},
	}

	addDBFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

// ── shelf ─────────────────────────────────────────────────────────────────────

func newGoodreadShelf() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int
	var shelf, cookiesFile string
	var auth bool

	cmd := &cobra.Command{
		Use:   "shelf <user_id>",
		Short: "Fetch a Goodreads user shelf",
		Args:  cobra.ExactArgs(1),
		Example: `  search goodread shelf 12345678 --shelf read
  search goodread shelf 12345678 --shelf to-read
  search goodread shelf 12345678 --auth`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := goodread.DefaultConfig()
			cfg.DBPath = dbPath
			cfg.StatePath = statePath
			cfg.Delay = time.Duration(delay) * time.Millisecond

			db, err := goodread.OpenDB(dbPath)
			if err != nil {
				return fmt.Errorf("open db: %w", err)
			}
			defer db.Close()

			stateDB, err := goodread.OpenState(statePath)
			if err != nil {
				return fmt.Errorf("open state: %w", err)
			}
			defer stateDB.Close()

			var client *goodread.Client
			if auth {
				cookies, err := goodread.LoadCookiesFromFile(cookiesFile)
				if err != nil {
					return fmt.Errorf("load cookies: %w", err)
				}
				client, err = goodread.NewClientWithCookies(cfg, cookies)
				if err != nil {
					return fmt.Errorf("create auth client: %w", err)
				}
				fmt.Printf("Using authenticated client (%d cookies)\n", len(cookies))
			} else {
				client = goodread.NewClient(cfg)
			}

			userID := args[0]
			shelfName := shelf
			if shelfName == "" {
				shelfName = "read"
			}
			shelfURL := goodread.BaseURL + "/review/list/" + userID + "?shelf=" + shelfName

			task := &goodread.ShelfTask{
				URL:       shelfURL,
				UserID:    userID,
				ShelfName: shelfName,
				Client:    client,
				DB:        db,
				StateDB:   stateDB,
				MaxPages:  maxPages,
			}
			m, err := task.Run(cmd.Context(), func(s *goodread.ShelfState) {
				fmt.Printf("  [%s] page=%d books=%d\n", s.Status, s.Pages, s.BooksFound)
			})
			if err != nil {
				return err
			}
			fmt.Printf("Done: fetched=%d pages=%d\n", m.Fetched, m.Pages)
			return nil
		},
	}

	addDBFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().StringVar(&shelf, "shelf", "read", "Shelf name (read, to-read, currently-reading, or custom)")
	cmd.Flags().IntVar(&maxPages, "max-pages", 0, "Max pages to fetch (0 = unlimited)")
	cmd.Flags().BoolVar(&auth, "auth", false, "Use authenticated client with cookies from --cookies-file")
	cmd.Flags().StringVar(&cookiesFile, "cookies-file", "", "Path to cookies.json (default: ~/data/goodread/cookies.json)")
	return cmd
}

// ── search ────────────────────────────────────────────────────────────────────

func newGoodreadSearch() *cobra.Command {
	var dbPath, statePath string
	var delay, maxPages int
	var auth bool
	var cookiesFile string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search Goodreads and enqueue results",
		Long: `Search Goodreads and enqueue results.

Without --auth: uses GET /book/auto_complete?format=json&q=<query> (no login required,
up to ~20 results, no pagination).

With --auth: uses the full HTML search page (/search?q=...&tab=books) with pagination,
returning 10 results per page. Works with or without cookies.
Cookies improve rate limits. Export with: goodread-tool cookies export`,
		Args: cobra.ExactArgs(1),
		Example: `  search goodread search "Dune"
  search goodread search "Frank Herbert"
  search goodread search "Dune" --auth
  search goodread search "Dune" --auth --max-pages 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := goodread.DefaultConfig()
			cfg.DBPath = dbPath
			cfg.StatePath = statePath
			cfg.Delay = time.Duration(delay) * time.Millisecond

			stateDB, err := goodread.OpenState(statePath)
			if err != nil {
				return fmt.Errorf("open state: %w", err)
			}
			defer stateDB.Close()

			query := args[0]

			if !auth {
				// Unauthenticated: autocomplete API
				client := goodread.NewClient(cfg)
				apiURL := goodread.BaseURL + "/book/auto_complete?format=json&q=" + url.QueryEscape(query)
				fmt.Printf("Searching (autocomplete): %s\n", apiURL)

				body, code, err := client.Fetch(cmd.Context(), apiURL)
				if err != nil {
					return fmt.Errorf("fetch: %w", err)
				}
				if code != 200 {
					return fmt.Errorf("unexpected HTTP %d", code)
				}

				results := goodread.ParseSearchAutocomplete(body)
				if len(results) == 0 {
					fmt.Println("No results found.")
					return nil
				}

				total := 0
				for _, r := range results {
					if err := stateDB.Enqueue(r.URL, r.EntityType, 5); err == nil {
						fmt.Printf("  Enqueued [%s] %s\n", r.EntityType, r.Title)
						total++
					}
				}
				fmt.Printf("Enqueued %d URLs\n", total)
				return nil
			}

			// Authenticated: HTML search with pagination
			cookies, err := goodread.LoadCookiesFromFile(cookiesFile)
			if err != nil {
				return fmt.Errorf("load cookies (run: goodread-tool cookies export): %w", err)
			}
			client, err := goodread.NewClientWithCookies(cfg, cookies)
			if err != nil {
				return fmt.Errorf("create auth client: %w", err)
			}
			fmt.Printf("Searching (authenticated, %d cookies): %q\n", len(cookies), query)

			searchURL := goodread.BaseURL + "/search?q=" + url.QueryEscape(query) + "&tab=books"
			total := 0
			page := 1

			for searchURL != "" {
				if maxPages > 0 && page > maxPages {
					break
				}
				fmt.Printf("  Page %d: %s\n", page, searchURL)

				doc, code, err := client.FetchHTML(cmd.Context(), searchURL)
				if err != nil {
					return fmt.Errorf("fetch page %d: %w", page, err)
				}
				if code == 401 {
					return fmt.Errorf("login required: cookies may be expired (run: goodread-tool cookies export)")
				}
				if code != 200 {
					return fmt.Errorf("unexpected HTTP %d on page %d", code, page)
				}

				results := goodread.ParseSearchHTML(doc)
				if len(results) == 0 {
					fmt.Printf("  No results on page %d (may be last page or login-gated)\n", page)
					break
				}
				for _, r := range results {
					if err := stateDB.Enqueue(r.URL, r.EntityType, 5); err == nil {
						total++
					}
				}
				fmt.Printf("  Page %d: found %d results (total=%d)\n", page, len(results), total)

				searchURL = goodread.ParseSearchHTMLNextPage(doc)
				page++
			}

			fmt.Printf("Enqueued %d URLs\n", total)
			return nil
		},
	}

	addDBFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().BoolVar(&auth, "auth", false, "Use authenticated HTML search with pagination (requires cookies)")
	cmd.Flags().StringVar(&cookiesFile, "cookies-file", "", "Path to cookies.json (default: ~/data/goodread/cookies.json)")
	cmd.Flags().IntVar(&maxPages, "max-pages", 0, "Max pages to fetch in --auth mode (0 = unlimited)")
	return cmd
}

// ── sitemap ───────────────────────────────────────────────────────────────────

func newGoodreadSitemap() *cobra.Command {
	var dbPath, statePath string
	var delay, limit int
	var entityFilter string

	cmd := &cobra.Command{
		Use:   "sitemap",
		Short: "Seed the queue from Goodreads sitemaps",
		Long: `Seed the crawl queue by parsing Goodreads sitemaps discovered from robots.txt.

Goodreads publishes per-type siteindex files (author, list, quote, etc.) each
pointing to gzipped sitemap files. URLs are filtered by entity type and
enqueued for later crawling.`,
		Args: cobra.NoArgs,
		Example: `  search goodread sitemap --limit 1000
  search goodread sitemap --limit 500 --type author`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, stateDB, _, err := openDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			// Discover siteindex URLs from robots.txt.
			fmt.Printf("Reading %s ...\n", goodread.RobotsTxtURL)
			siteindexes, err := parseSitemapsFromRobots(goodread.RobotsTxtURL)
			if err != nil {
				return fmt.Errorf("parse robots.txt: %w", err)
			}
			fmt.Printf("Found %d siteindex files\n", len(siteindexes))

			// Filter by entity type if requested.
			filter := strings.ToLower(strings.TrimSpace(entityFilter))

			total := 0
			for _, si := range siteindexes {
				if limit > 0 && total >= limit {
					break
				}
				// Infer entity type from siteindex filename.
				siType := inferSiteindexType(si)
				if filter != "" && !strings.Contains(siType, filter) {
					continue
				}

				fmt.Printf("  Siteindex [%s] %s ...\n", siType, si)
				gzURLs, err := fetchSitemapIndex(si)
				if err != nil {
					fmt.Printf("    Warning: %v\n", err)
					continue
				}

				for _, gzURL := range gzURLs {
					if limit > 0 && total >= limit {
						break
					}
					n, err := fetchAndEnqueueGzSitemap(gzURL, stateDB, limit-total)
					if err != nil {
						fmt.Printf("    Warning (%s): %v\n", gzURL, err)
						continue
					}
					total += n
					if n > 0 {
						fmt.Printf("    %s → enqueued %d (total=%d)\n", gzURL, n, total)
					}
					time.Sleep(200 * time.Millisecond)
				}
			}

			fmt.Printf("Total enqueued from sitemaps: %d\n", total)
			return nil
		},
	}

	addDBFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().IntVar(&limit, "limit", 0, "Max URLs to enqueue (0 = unlimited)")
	cmd.Flags().StringVar(&entityFilter, "type", "", "Filter by entity type: book, author, list, quote, user (default: all)")
	return cmd
}

// parseSitemapsFromRobots fetches robots.txt and extracts all Sitemap: lines.
func parseSitemapsFromRobots(robotsURL string) ([]string, error) {
	resp, err := http.Get(robotsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var urls []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if after, ok := strings.CutPrefix(line, "Sitemap:"); ok {
			u := strings.TrimSpace(after)
			if u != "" {
				urls = append(urls, u)
			}
		}
	}
	return urls, scanner.Err()
}

// inferSiteindexType guesses the entity type from a siteindex URL filename.
// e.g. "https://www.goodreads.com/siteindex.author.xml" → "author"
func inferSiteindexType(u string) string {
	lastSlash := strings.LastIndex(u, "/")
	if lastSlash < 0 {
		return "unknown"
	}
	filename := u[lastSlash+1:] // "siteindex.author.xml"
	parts := strings.Split(filename, ".")
	for i, p := range parts {
		if p == "siteindex" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return "unknown"
}

// fetchSitemapIndex fetches a siteindex XML and returns the list of .gz sitemap URLs.
func fetchSitemapIndex(siteindexURL string) ([]string, error) {
	resp, err := http.Get(siteindexURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var idx struct {
		Sitemaps []struct {
			Loc string `xml:"loc"`
		} `xml:"sitemap"`
	}
	if err := xml.Unmarshal(body, &idx); err != nil {
		return nil, err
	}

	var urls []string
	for _, sm := range idx.Sitemaps {
		if sm.Loc != "" {
			urls = append(urls, sm.Loc)
		}
	}
	return urls, nil
}

// fetchAndEnqueueGzSitemap downloads a gzipped sitemap, decompresses it, and enqueues URLs.
func fetchAndEnqueueGzSitemap(gzURL string, stateDB *goodread.State, limit int) (int, error) {
	resp, err := http.Get(gzURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var reader io.Reader = resp.Body
	if strings.HasSuffix(gzURL, ".gz") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return 0, fmt.Errorf("gzip: %w", err)
		}
		defer gz.Close()
		reader = gz
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return 0, err
	}

	var urlSet struct {
		URLs []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal(body, &urlSet); err != nil {
		return 0, err
	}

	n := 0
	for _, u := range urlSet.URLs {
		if limit > 0 && n >= limit {
			break
		}
		entityType := goodread.InferEntityType(u.Loc)
		if stateDB.Enqueue(u.Loc, entityType, 1) == nil {
			n++
		}
	}
	return n, nil
}

// ── crawl ─────────────────────────────────────────────────────────────────────

func newGoodreadCrawl() *cobra.Command {
	var dbPath, statePath string
	var delay, workers, maxPages int

	cmd := &cobra.Command{
		Use:   "crawl",
		Short: "Bulk crawl from the queue (use sitemap or search to seed first)",
		Args:  cobra.NoArgs,
		Example: `  search goodread sitemap --limit 500 && search goodread crawl --workers 2
  search goodread crawl --workers 1 --delay 3000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := goodread.DefaultConfig()
			cfg.DBPath = dbPath
			cfg.StatePath = statePath
			cfg.Workers = workers
			cfg.Delay = time.Duration(delay) * time.Millisecond
			cfg.MaxPages = maxPages

			db, err := goodread.OpenDB(dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			stateDB, err := goodread.OpenState(statePath)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			client := goodread.NewClient(cfg)

			pending, _, done, failed := stateDB.QueueStats()
			fmt.Printf("Queue: pending=%d done=%d failed=%d\n", pending, done, failed)
			if pending == 0 {
				fmt.Println("Queue is empty. Run 'search goodread sitemap' or 'search goodread search' first.")
				return nil
			}

			fmt.Printf("Starting crawl with %d workers, delay=%s ...\n", workers, cfg.Delay)
			stateDB.CreateJob("crawl-"+fmt.Sprintf("%d", time.Now().Unix()), "bulk-crawl", "crawl")

			task := &goodread.CrawlTask{
				Config:  cfg,
				Client:  client,
				DB:      db,
				StateDB: stateDB,
			}

			m, err := task.Run(cmd.Context(), func(s *goodread.CrawlState) {
				goodread.PrintCrawlProgress(s)
			})
			if err != nil {
				return err
			}

			fmt.Printf("\nCrawl complete: done=%d failed=%d duration=%s\n",
				m.Done, m.Failed, m.Duration.Round(time.Second))
			return nil
		},
	}

	cfg := goodread.DefaultConfig()
	cmd.Flags().StringVar(&dbPath, "db", cfg.DBPath, "Path to goodread.duckdb")
	cmd.Flags().StringVar(&statePath, "state", cfg.StatePath, "Path to state.duckdb")
	cmd.Flags().IntVar(&workers, "workers", cfg.Workers, "Concurrent fetch workers")
	cmd.Flags().IntVar(&delay, "delay", int(cfg.Delay/time.Millisecond), "Delay between requests in milliseconds")
	cmd.Flags().IntVar(&maxPages, "max-pages", 0, "Max pages per entity (0 = unlimited)")
	return cmd
}

// ── info ─────────────────────────────────────────────────────────────────────

func newGoodreadInfo() *cobra.Command {
	var dbPath, statePath string
	var delay int

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show Goodreads database stats and queue depth",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, stateDB, _, err := openDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer db.Close()
			defer stateDB.Close()
			return goodread.PrintStats(db, stateDB)
		},
	}

	addDBFlags(cmd, &dbPath, &statePath, &delay)
	return cmd
}

// ── jobs ─────────────────────────────────────────────────────────────────────

func newGoodreadJobs() *cobra.Command {
	var dbPath, statePath string
	var delay, limit int

	cmd := &cobra.Command{
		Use:   "jobs",
		Short: "List recent crawl jobs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, stateDB, _, err := openDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			jobs, err := stateDB.ListJobs(limit)
			if err != nil {
				return err
			}

			if len(jobs) == 0 {
				fmt.Println("No jobs found.")
				return nil
			}

			fmt.Println("── Recent Jobs ──")
			for _, j := range jobs {
				dur := ""
				if !j.CompletedAt.IsZero() {
					dur = j.CompletedAt.Sub(j.StartedAt).Round(time.Second).String()
				}
				fmt.Printf("  [%s] %s  %s  started=%s  duration=%s\n",
					j.Status, j.JobID, j.Name,
					j.StartedAt.Format("2006-01-02 15:04"), dur)
			}
			return nil
		},
	}

	addDBFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of jobs to show")
	return cmd
}

// ── queue ─────────────────────────────────────────────────────────────────────

func newGoodreadQueue() *cobra.Command {
	var dbPath, statePath string
	var delay, limit int
	var status string

	cmd := &cobra.Command{
		Use:   "queue",
		Short: "Inspect queue items",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, stateDB, _, err := openDBs(dbPath, statePath, delay)
			if err != nil {
				return err
			}
			defer stateDB.Close()

			items, err := stateDB.ListQueue(status, limit)
			if err != nil {
				return err
			}

			if len(items) == 0 {
				fmt.Printf("No %s items in queue.\n", status)
				return nil
			}

			fmt.Printf("── Queue items (status=%s) ──\n", status)
			for _, it := range items {
				fmt.Printf("  [%s] %s\n", it.EntityType, it.URL)
			}
			return nil
		},
	}

	addDBFlags(cmd, &dbPath, &statePath, &delay)
	cmd.Flags().StringVar(&status, "status", "pending", "Filter by status: pending, failed, done, in_progress")
	cmd.Flags().IntVar(&limit, "limit", 20, "Number of items to show")
	return cmd
}

// ── URL normalization helpers ─────────────────────────────────────────────────

func normalizeBookURL(s string) string {
	if strings.HasPrefix(s, "http") {
		return s
	}
	return goodread.BaseURL + "/book/show/" + s
}

func normalizeAuthorURL(s string) string {
	if strings.HasPrefix(s, "http") {
		return s
	}
	return goodread.BaseURL + "/author/show/" + s
}

func normalizeSeriesURL(s string) string {
	if strings.HasPrefix(s, "http") {
		return s
	}
	return goodread.BaseURL + "/series/" + s
}

func normalizeListURL(s string) string {
	if strings.HasPrefix(s, "http") {
		return s
	}
	return goodread.BaseURL + "/list/show/" + s
}

func normalizeUserURL(s string) string {
	if strings.HasPrefix(s, "http") {
		return s
	}
	return goodread.BaseURL + "/user/show/" + s
}

