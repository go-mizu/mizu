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

	"github.com/go-mizu/mizu/blueprints/search/pkg/dcrawler/x"
	"github.com/go-mizu/mizu/blueprints/search/pkg/markdown"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/spf13/cobra"
)

// NewX creates the x command with subcommands.
func NewX() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "x",
		Short: "X/Twitter search and scrape",
		Long: `Search and scrape X/Twitter data using the internal GraphQL API.

All endpoints require authentication. Login first with:
  search x login <username>

Or import cookies from your browser:
  search x import-session <username>

Data: $HOME/data/x/
Sessions: $HOME/data/x/.sessions/

Subcommands:
  login          Login with username/password
  import-session Import cookies from browser
  profile        Fetch user profile
  tweets         Fetch user timeline tweets
  media          Fetch media-only timeline
  tweet          Fetch a single tweet with replies
  search         Search tweets
  search-users   Search for user profiles
  hashtag        Search by hashtag
  followers      Fetch follower list
  following      Fetch following list
  bookmarks      Fetch bookmarked tweets
  home           Fetch home timeline
  foryou         Fetch "For You" timeline
  retweeters     Fetch who retweeted a tweet
  favoriters     Fetch who liked a tweet
  list           Fetch list info
  list-tweets    Fetch tweets from a list
  list-members   Fetch members of a list
  space          Fetch audio space info
  download       Download media from stored tweets
  export         Export tweets to JSON/CSV/RSS
  trends         Show current trending topics
  info           Show stored data statistics

Examples:
  search x login myuser
  search x profile karpathy --session myuser
  search x tweets karpathy --max-tweets 100 --session myuser
  search x media karpathy --session myuser
  search x search "golang" --mode latest --session myuser
  search x bookmarks --session myuser
  search x download karpathy --photos --videos
  search x export karpathy --format csv`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newXLogin())
	cmd.AddCommand(newXImportSession())
	cmd.AddCommand(newXProfile())
	cmd.AddCommand(newXTweets())
	cmd.AddCommand(newXMedia())
	cmd.AddCommand(newXTweet())
	cmd.AddCommand(newXSearch())
	cmd.AddCommand(newXSearchUsers())
	cmd.AddCommand(newXHashtag())
	cmd.AddCommand(newXFollowers())
	cmd.AddCommand(newXFollowing())
	cmd.AddCommand(newXBookmarks())
	cmd.AddCommand(newXHome())
	cmd.AddCommand(newXForYou())
	cmd.AddCommand(newXRetweeters())
	cmd.AddCommand(newXFavoriters())
	cmd.AddCommand(newXList())
	cmd.AddCommand(newXListTweets())
	cmd.AddCommand(newXListMembers())
	cmd.AddCommand(newXSpace())
	cmd.AddCommand(newXDownload())
	cmd.AddCommand(newXExport())
	cmd.AddCommand(newXTrends())
	cmd.AddCommand(newXInfo())

	return cmd
}

// initXClient creates and optionally loads a session for an X client.
// If session is empty, it falls back to "default" session.
// After loading cookies, calls Activate() to switch the bearer token
// from guest to authenticated (required for search, bookmarks, etc).
func initXClient(cfg x.Config, session string) (*x.Client, error) {
	client := x.NewClient(cfg)

	// Fall back to "default" session if none specified
	if session == "" {
		session = "default"
	}

	sessionPath := cfg.SessionPath(session)
	sess, err := client.LoadSessionFile(sessionPath)
	if err != nil {
		if session == "default" {
			return nil, fmt.Errorf("no session found: run 'search x import-session' first or use --session flag")
		}
		return nil, fmt.Errorf("load session %q: %w", sessionPath, err)
	}
	fmt.Printf("  Session: %s\n", infoStyle.Render("@"+sess.Username))

	// Activate switches bearer token from guest to authenticated.
	// Without this, search/bookmarks/home endpoints return 401.
	if !client.Activate() {
		fmt.Println(warningStyle.Render("  Warning: session may be expired (Activate returned false)"))
	}

	return client, nil
}

// ── login ───────────────────────────────────────────────

func newXLogin() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login <username>",
		Short: "Login to X/Twitter and save session",
		Long: `Login to X/Twitter with username/password and save the session.

The session is saved to $HOME/data/x/.sessions/{username}.json
and can be loaded by other commands via --session flag.

Password is read from X_PWD environment variable or prompted interactively.

Examples:
  search x login myuser
  X_PWD=secret search x login myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runXLogin(username)
		},
	}
	return cmd
}

func runXLogin(username string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter Login"))
	fmt.Println()

	cfg := x.DefaultConfig()
	client := x.NewClient(cfg)

	fmt.Printf("  Username: %s\n", infoStyle.Render("@"+username))
	password := os.Getenv("X_PWD")
	if password == "" {
		fmt.Print("  Password: ")
		fmt.Scanln(&password)
	} else {
		fmt.Println("  Password: (from X_PWD env)")
	}

	if password == "" {
		return fmt.Errorf("password cannot be empty (set X_PWD env or enter interactively)")
	}

	fmt.Println(labelStyle.Render("  Logging in..."))
	if err := client.Login(username, password); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// Save session
	sessionPath := cfg.SessionPath(username)
	if err := client.SaveSessionFile(sessionPath, username); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("  Login successful!"))
	fmt.Printf("  Session saved to: %s\n", labelStyle.Render(sessionPath))
	fmt.Printf("  Use with: %s\n", infoStyle.Render("--session "+username))

	return nil
}

// ── import-session ──────────────────────────────────────

func newXImportSession() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-session [username]",
		Short: "Import session from browser cookies",
		Long: `Import an X/Twitter session using cookies from your browser.

Steps:
  1. Login to x.com in your browser
  2. Open DevTools (F12) > Application > Cookies > x.com
  3. Copy the values of: auth_token, ct0
  4. Set environment variables:
     export X_AUTH_TOKEN="your_auth_token"
     export X_CSRF_TOKEN="your_ct0_value"
  5. Run: search x import-session [username]

If username is omitted, saves as "default" session (used when --session is not set).

Examples:
  search x import-session           # saves as default session
  search x import-session myuser    # saves as named session`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := "default"
			if len(args) > 0 {
				username = strings.TrimPrefix(args[0], "@")
			}
			return runXImportSession(username)
		},
	}
	return cmd
}

func runXImportSession(username string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter Import Session"))
	fmt.Println()

	authToken := os.Getenv("X_AUTH_TOKEN")
	csrfToken := os.Getenv("X_CSRF_TOKEN")

	if authToken == "" {
		return fmt.Errorf("X_AUTH_TOKEN env not set (copy from browser DevTools > Application > Cookies > auth_token)")
	}
	if csrfToken == "" {
		return fmt.Errorf("X_CSRF_TOKEN env not set (copy from browser DevTools > Application > Cookies > ct0)")
	}

	cfg := x.DefaultConfig()
	client := x.NewClient(cfg)

	// Set auth token (uses library's proper cookie setup)
	client.SetAuthToken(authToken, csrfToken)

	// Activate to switch bearer token from guest to authenticated
	if !client.Activate() {
		fmt.Println(warningStyle.Render("  Warning: session activation failed (tokens may be expired)"))
	}

	// Save session
	sessionPath := cfg.SessionPath(username)
	if err := client.SaveSessionFile(sessionPath, username); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	// Verify by fetching profile
	fmt.Println(labelStyle.Render("  Verifying session..."))
	profile, err := client.GetProfile(username)
	if err != nil {
		fmt.Println(warningStyle.Render("  Warning: could not verify session (may still work)"))
		fmt.Println(warningStyle.Render(fmt.Sprintf("  %v", err)))
	} else if profile != nil {
		fmt.Printf("  Verified as: %s\n", infoStyle.Render("@"+profile.Username))
	}

	fmt.Println()
	fmt.Println(successStyle.Render("  Session saved!"))
	fmt.Printf("  Session saved to: %s\n", labelStyle.Render(sessionPath))
	if username == "default" {
		fmt.Println("  This is the default session (used when --session is not set)")
	} else {
		fmt.Printf("  Use with: %s\n", infoStyle.Render("--session "+username))
	}

	return nil
}

// ── profile ──────────────────────────────────────────────

func newXProfile() *cobra.Command {
	var session string

	cmd := &cobra.Command{
		Use:   "profile <username>",
		Short: "Fetch and display user profile",
		Long: `Fetch profile information for an X/Twitter user.

Displays username, bio, follower/following counts, tweet count, and more.
Profile data is saved to $HOME/data/x/{username}/profile.json

Examples:
  search x profile karpathy --session myuser
  search x profile elonmusk --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runXProfile(username, session)
		},
	}

	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	return cmd
}

func runXProfile(username, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter Profile"))
	fmt.Println()

	cfg := x.DefaultConfig()
	client, err := initXClient(cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  Fetching profile for %s\n", infoStyle.Render("@"+username))

	profile, err := client.GetProfile(username)
	if err != nil {
		return err
	}

	// Save profile
	if err := x.SaveProfile(cfg, profile); err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: could not save profile: %v", err)))
	}

	// Save to DB
	db, err := x.OpenDB(cfg.UserDBPath(username))
	if err == nil {
		db.InsertUser(profile)
		db.Close()
	}

	fmt.Println()
	displayXProfile(profile)

	return nil
}

func displayXProfile(p *x.Profile) {
	verified := ""
	if p.IsVerified || p.IsBlueVerified {
		verified = " [verified]"
	}
	private := ""
	if p.IsPrivate {
		private = " [private]"
	}

	fmt.Printf("  %s%s%s\n", titleStyle.Render(p.Name), verified, private)
	fmt.Printf("  @%s\n", infoStyle.Render(p.Username))
	if p.Biography != "" {
		fmt.Println()
		for _, line := range strings.Split(p.Biography, "\n") {
			fmt.Printf("  %s\n", line)
		}
	}
	fmt.Println()
	fmt.Printf("  Tweets:     %s\n", infoStyle.Render(formatLargeNumber(int64(p.TweetsCount))))
	fmt.Printf("  Followers:  %s\n", infoStyle.Render(formatLargeNumber(int64(p.FollowersCount))))
	fmt.Printf("  Following:  %s\n", infoStyle.Render(formatLargeNumber(int64(p.FollowingCount))))
	fmt.Printf("  Likes:      %s\n", infoStyle.Render(formatLargeNumber(int64(p.LikesCount))))
	fmt.Printf("  Media:      %s\n", infoStyle.Render(formatLargeNumber(int64(p.MediaCount))))
	if p.Location != "" {
		fmt.Printf("  Location:   %s\n", labelStyle.Render(p.Location))
	}
	if p.Website != "" {
		fmt.Printf("  Website:    %s\n", urlStyle.Render(p.Website))
	}
	if !p.Joined.IsZero() {
		fmt.Printf("  Joined:     %s\n", labelStyle.Render(p.Joined.Format("January 2006")))
	}
	fmt.Printf("  ID:         %s\n", labelStyle.Render(p.ID))
	fmt.Println()
}

// ── tweets ──────────────────────────────────────────────

func newXTweets() *cobra.Command {
	var (
		maxTweets int
		all       bool
		order     string
		session   string
	)

	cmd := &cobra.Command{
		Use:   "tweets <username>",
		Short: "Fetch user timeline tweets",
		Long: `Fetch timeline tweets for an X/Twitter user.

Tweets are stored in a DuckDB database at $HOME/data/x/{username}/tweets.duckdb
Use --all or --max-tweets 0 to fetch all available tweets.

Order controls which date windows are fetched first:
  oldest  - Start from account creation date (good for first full fetch)
  newest  - Start from today going backwards (good for catching up)

Windows that already have tweets in the DB are skipped by default.

Examples:
  search x tweets karpathy --session myuser
  search x tweets karpathy --all --session myuser
  search x tweets karpathy --all --order newest --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			if all {
				maxTweets = 0
			}
			return runXTweets(cmd.Context(), username, maxTweets, order, session)
		},
	}

	cmd.Flags().IntVar(&maxTweets, "max-tweets", 200, "Max tweets to fetch (0 = unlimited)")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all available tweets (overrides --max-tweets)")
	cmd.Flags().StringVar(&order, "order", "oldest", "Fetch order: oldest or newest first")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	return cmd
}

func runXTweets(ctx context.Context, username string, maxTweets int, order, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter Tweets"))
	fmt.Println()

	cfg := x.DefaultConfig()
	client, err := initXClient(cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  Fetching tweets for %s\n", infoStyle.Render("@"+username))
	if maxTweets > 0 {
		fmt.Printf("  Max tweets: %s\n", labelStyle.Render(fmt.Sprintf("%d", maxTweets)))
	} else {
		fmt.Printf("  Max tweets: %s\n", infoStyle.Render("unlimited (--all)"))
	}
	fmt.Printf("  Data:       %s\n", labelStyle.Render(cfg.UserDir(username)))
	fmt.Println()

	// Open DB early for incremental saves
	db, err := x.OpenDB(cfg.UserDBPath(username))
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	// Check existing tweet count
	stats, _ := db.GetStats()
	if stats.Tweets > 0 {
		fmt.Printf("  Existing:   %s tweets in DB\n", infoStyle.Render(formatLargeNumber(stats.Tweets)))
	}

	start := time.Now()
	var totalFetched int64
	batchSave := func(batch []x.Tweet) {
		if err := db.InsertTweets(batch); err != nil {
			fmt.Printf("\n  Warning: batch save failed: %v\n", err)
		} else {
			totalFetched += int64(len(batch))
		}
	}

	if maxTweets > 0 {
		// Limited fetch: use timeline API (fast, up to ~800 tweets)
		tweets, err := client.GetTweetsWithBatch(ctx, username, maxTweets,
			func(p x.Progress) {
				if !p.Done {
					if p.Message != "" {
						fmt.Printf("\r  tweets: %s  %s                    ",
							infoStyle.Render(formatLargeNumber(p.Current)),
							warningStyle.Render(p.Message))
					} else {
						fmt.Printf("\r  tweets: %s          ",
							infoStyle.Render(formatLargeNumber(p.Current)))
					}
				}
			}, batchSave)
		fmt.Println()
		if err != nil {
			fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d tweets)", err, len(tweets))))
		}
	} else {
		// --all: date-windowed search to get full history
		if err := fetchAllTweetsByDate(ctx, client, db, username, order, batchSave); err != nil {
			fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v", err)))
		}
	}

	// Get final DB stats
	finalStats, _ := db.GetStats()

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Fetched %d new tweets in %s",
		totalFetched, time.Since(start).Truncate(time.Second))))
	fmt.Printf("  Total in DB: %s tweets\n", infoStyle.Render(formatLargeNumber(finalStats.Tweets)))
	fmt.Printf("  Database:    %s\n", labelStyle.Render(cfg.UserDBPath(username)))

	// Show top 5 tweets from DB
	fmt.Println()
	fmt.Println(subtitleStyle.Render("  Top tweets by likes:"))
	top, _ := db.TopTweets(5)
	for i, t := range top {
		text := strings.ReplaceAll(t.Text, "\n", " ")
		if len(text) > 60 {
			text = text[:60] + "..."
		}
		fmt.Printf("  %d. %s likes  %s RT  %s views  %s\n",
			i+1,
			infoStyle.Render(formatLargeNumber(int64(t.Likes))),
			labelStyle.Render(formatLargeNumber(int64(t.Retweets))),
			labelStyle.Render(formatLargeNumber(int64(t.Views))),
			text)
	}

	return nil
}

// fetchAllTweetsByDate fetches all tweets using date-windowed search.
// Uses "from:username since:YYYY-MM-DD until:YYYY-MM-DD" queries in monthly windows.
// order: "oldest" (join date → today) or "newest" (today → join date).
// Windows that already have tweets in the DB are skipped.
func fetchAllTweetsByDate(ctx context.Context, client *x.Client, db *x.DB, username, order string, batchSave x.BatchCallback) error {
	// Get profile for join date and tweet count
	profile, err := client.GetProfile(username)
	if err != nil {
		return fmt.Errorf("get profile: %w", err)
	}

	joinDate := profile.Joined
	if joinDate.IsZero() {
		joinDate = time.Date(2006, 3, 1, 0, 0, 0, 0, time.UTC) // Twitter launch date
	}

	now := time.Now().UTC()
	fmt.Printf("  Profile:    %s (@%s), joined %s, %s tweets\n",
		infoStyle.Render(profile.Name), username,
		labelStyle.Render(joinDate.Format("Jan 2006")),
		infoStyle.Render(formatLargeNumber(int64(profile.TweetsCount))))

	// Build monthly windows (always oldest→newest internally)
	type window struct {
		since, until time.Time
	}
	var windows []window

	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	cur := time.Date(joinDate.Year(), joinDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	for cur.Before(tomorrow) {
		until := cur.AddDate(0, 1, 0)
		if until.After(tomorrow) {
			until = tomorrow
		}
		windows = append(windows, window{since: cur, until: until})
		cur = until
	}

	// Reverse for newest-first order
	if order == "newest" {
		for i, j := 0, len(windows)-1; i < j; i, j = i+1, j-1 {
			windows[i], windows[j] = windows[j], windows[i]
		}
	}

	fmt.Printf("  Windows:    %d months (%s → %s), order: %s\n",
		len(windows),
		joinDate.Format("2006-01"),
		now.Format("2006-01"),
		order)

	// Check existing date range
	dbOldest, dbNewest, _ := db.TweetDateRange()
	if !dbOldest.IsZero() {
		fmt.Printf("  DB range:   %s → %s\n",
			labelStyle.Render(dbOldest.Format("2006-01-02")),
			labelStyle.Render(dbNewest.Format("2006-01-02")))
	}
	fmt.Println()

	oldMode := client.SearchMode()
	client.SetSearchMode(x.SearchLatest)
	defer client.SetSearchMode(oldMode)

	var grandTotal int64
	var skipped int
	for i, w := range windows {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		label := w.since.Format("Jan 2006")

		// Skip windows that already have tweets in DB
		existing := db.TweetCountInRange(w.since, w.until)
		if existing > 0 {
			skipped++
			grandTotal += existing
			fmt.Printf("  [%d/%d] %s: %s tweets (cached)          \n",
				i+1, len(windows), label,
				labelStyle.Render(formatLargeNumber(existing)))
			continue
		}

		sinceStr := w.since.Format("2006-01-02")
		untilStr := w.until.Format("2006-01-02")
		query := fmt.Sprintf("from:%s since:%s until:%s", username, sinceStr, untilStr)

		tweets, err := client.SearchTweetsWithBatch(ctx, query, 0,
			func(p x.Progress) {
				if !p.Done {
					if p.Message != "" {
						fmt.Printf("\r  [%d/%d] %s: %s  (total: %s)  %s                    ",
							i+1, len(windows), label,
							infoStyle.Render(formatLargeNumber(p.Current)),
							labelStyle.Render(formatLargeNumber(grandTotal+p.Current)),
							warningStyle.Render(p.Message))
					} else {
						fmt.Printf("\r  [%d/%d] %s: %s  (total: %s)          ",
							i+1, len(windows), label,
							infoStyle.Render(formatLargeNumber(p.Current)),
							labelStyle.Render(formatLargeNumber(grandTotal+p.Current)))
					}
				}
			}, batchSave)

		windowCount := int64(len(tweets))
		grandTotal += windowCount

		if err != nil {
			fmt.Printf("\n  Warning: %s: %v (got %d)\n", label, err, windowCount)
		}

		fmt.Printf("\r  [%d/%d] %s: %s tweets  (total: %s)          \n",
			i+1, len(windows), label,
			infoStyle.Render(formatLargeNumber(windowCount)),
			labelStyle.Render(formatLargeNumber(grandTotal)))
	}

	if skipped > 0 {
		fmt.Printf("\n  Skipped %d windows with existing data\n", skipped)
	}

	return nil
}

// ── tweet ───────────────────────────────────────────────

func newXTweet() *cobra.Command {
	var session, format string

	cmd := &cobra.Command{
		Use:   "tweet <id_or_url>",
		Short: "Fetch a single tweet with replies",
		Long: `Fetch a single tweet by ID or URL, including replies.

Accepts tweet ID or full URL:
  search x tweet 1234567890
  search x tweet https://x.com/user/status/1234567890

Use --format markdown to save the thread as a markdown article:
  search x tweet https://x.com/LangChain/status/2031055593360990358 --format markdown --session myuser

Examples:
  search x tweet 1234567890 --session myuser
  search x tweet https://x.com/user/status/123 --format markdown --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := extractTweetID(args[0])
			return runXTweet(cmd.Context(), id, session, format)
		},
	}

	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	cmd.Flags().StringVar(&format, "format", "", "Output format: markdown")
	return cmd
}

func extractTweetID(input string) string {
	// Handle URLs like https://x.com/user/status/1234567890
	if strings.Contains(input, "/status/") {
		parts := strings.Split(input, "/status/")
		if len(parts) == 2 {
			id := parts[1]
			// Remove query params
			if idx := strings.IndexByte(id, '?'); idx >= 0 {
				id = id[:idx]
			}
			return strings.TrimRight(id, "/")
		}
	}
	return input
}

func runXTweet(ctx context.Context, id, session, format string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter Tweet"))
	fmt.Println()

	cfg := x.DefaultConfig()
	client, err := initXClient(cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  Fetching tweet %s\n", infoStyle.Render(id))

	tweet, err := client.GetTweet(id)
	if err != nil {
		return err
	}

	fmt.Println()
	displayTweet(tweet)

	// Fetch replies
	fmt.Println(labelStyle.Render("  Fetching replies..."))
	replies, err := client.GetTweetReplies(id)
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v", err)))
	}

	// Always store tweet + replies in DuckDB
	db, dbErr := x.OpenDB(cfg.UserDBPath(tweet.Username))
	if dbErr == nil {
		allTweets := append([]x.Tweet{*tweet}, replies...)
		db.InsertTweets(allTweets)
	}

	if len(replies) > 0 {
		fmt.Printf("  %d replies found\n\n", len(replies))

		showN := min(5, len(replies))
		fmt.Println(subtitleStyle.Render("  Top replies:"))
		for i := range showN {
			r := replies[i]
			text := strings.ReplaceAll(r.Text, "\n", " ")
			if len(text) > 70 {
				text = text[:70] + "..."
			}
			fmt.Printf("  @%-16s %s likes  %s\n",
				infoStyle.Render(r.Username),
				labelStyle.Render(formatLargeNumber(int64(r.Likes))),
				text)
		}
	} else {
		fmt.Println(labelStyle.Render("  No replies found"))
	}

	// Markdown export
	if strings.ToLower(format) == "markdown" {
		thread := x.ExtractThread(*tweet, replies)

		// If the tweet is just a link, follow it and extract the full article body
		if isTweetJustALink(*tweet) || (tweet.Text == "" && len(tweet.URLs) > 0) {
			linkedURL := extractFirstURL(*tweet)
			if linkedURL != "" {
				fmt.Printf("  Fetching linked article: %s\n", infoStyle.Render(linkedURL))

				if extractXArticleID(linkedURL) != "" {
					// X Article — render with headless browser (public URL, no auth needed)
					articleURL := fmt.Sprintf("https://x.com/%s/article/%s", tweet.Username, tweet.ID)
					fmt.Println(labelStyle.Render("  Rendering with headless browser..."))
					artTitle, artBody, rodErr := fetchXArticleWithRod(ctx, articleURL, client.AuthToken(), client.CT0())
					if rodErr == nil && artBody != "" {
						if artTitle != "" && thread[0].Title == "" {
							thread[0].Title = artTitle
						}
						thread[0].Text = artBody
						thread[0].PermanentURL = articleURL
						fmt.Printf("  Article: %d chars (via headless browser)\n", len(artBody))
					} else {
						fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: X Article not accessible (%v)", rodErr)))
					}
				} else {
					// External URL — fetch HTML and extract with trafilatura
					artTitle, artBody, finalURL := fetchLinkedArticle(linkedURL)
					if artBody != "" {
						if artTitle != "" && thread[0].Title == "" {
							thread[0].Title = artTitle
						}
						thread[0].Text = artBody
						if finalURL != "" {
							thread[0].PermanentURL = finalURL
						}
						fmt.Printf("  Article: %d chars extracted\n", len(artBody))
					} else {
						fmt.Println(warningStyle.Render("  Warning: could not extract article body from linked URL"))
					}
				}
			}
		}

		mdPath := filepath.Join(cfg.UserDir(tweet.Username), id+".md")
		if err := x.ExportMarkdown(thread, mdPath); err != nil {
			fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: export markdown: %v", err)))
		} else {
			fmt.Println()
			fmt.Printf("  Thread: %d tweets\n", len(thread))
			fmt.Println(successStyle.Render(fmt.Sprintf("  Saved markdown: %s", mdPath)))
		}

		// Store assembled article in DuckDB
		if dbErr == nil {
			article := x.Article{
				ID:         tweet.ID,
				Username:   tweet.Username,
				Name:       tweet.Name,
				Title:      thread[0].Title,
				ContentMD:  x.TweetThreadToMarkdown(thread),
				TweetCount: len(thread),
				Likes:      tweet.Likes,
				Retweets:   tweet.Retweets,
				Replies:    tweet.Replies,
				Views:      tweet.Views,
				PostedAt:   tweet.PostedAt,
			}
			if err := db.InsertArticle(article); err != nil {
				fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: store article: %v", err)))
			} else {
				fmt.Println(successStyle.Render("  Saved article to database"))
			}
		}
	}

	if dbErr == nil {
		db.Close()
	}

	return nil
}

func displayTweet(t *x.Tweet) {
	fmt.Printf("  @%s", infoStyle.Render(t.Username))
	if t.Name != "" {
		fmt.Printf("  (%s)", t.Name)
	}
	fmt.Println()
	fmt.Printf("  %s\n", labelStyle.Render(t.PostedAt.Format("2006-01-02 15:04:05")))
	fmt.Println()

	// Print text (indented)
	for _, line := range strings.Split(t.Text, "\n") {
		fmt.Printf("  %s\n", line)
	}
	fmt.Println()

	fmt.Printf("  Likes:    %s\n", infoStyle.Render(formatLargeNumber(int64(t.Likes))))
	fmt.Printf("  Retweets: %s\n", infoStyle.Render(formatLargeNumber(int64(t.Retweets))))
	fmt.Printf("  Replies:  %s\n", infoStyle.Render(formatLargeNumber(int64(t.Replies))))
	fmt.Printf("  Views:    %s\n", infoStyle.Render(formatLargeNumber(int64(t.Views))))

	if len(t.Photos) > 0 {
		fmt.Printf("  Photos:   %d\n", len(t.Photos))
	}
	if len(t.Videos) > 0 {
		fmt.Printf("  Videos:   %d\n", len(t.Videos))
	}
	if len(t.Hashtags) > 0 {
		fmt.Printf("  Tags:     %s\n", labelStyle.Render(strings.Join(t.Hashtags, ", ")))
	}
	if t.PermanentURL != "" {
		fmt.Printf("  URL:      %s\n", urlStyle.Render(t.PermanentURL))
	}
	fmt.Printf("  ID:       %s\n", labelStyle.Render(t.ID))
	fmt.Println()
}

// ── search ──────────────────────────────────────────────

func newXSearch() *cobra.Command {
	var (
		maxTweets int
		mode      string
		session   string
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search tweets",
		Long: `Search X/Twitter for tweets matching a query.

Search modes:
  top     - Top/relevant results (default)
  latest  - Most recent results
  photos  - Tweets with photos
  videos  - Tweets with videos

Results are stored in a DuckDB database.

Examples:
  search x search "golang" --session myuser
  search x search "golang" --mode latest --max-tweets 50 --session myuser
  search x search "from:karpathy" --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runXSearch(cmd.Context(), args[0], maxTweets, mode, session)
		},
	}

	cmd.Flags().IntVar(&maxTweets, "max-tweets", 100, "Max tweets to fetch")
	cmd.Flags().StringVar(&mode, "mode", "top", "Search mode: top, latest, photos, videos")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	return cmd
}

func runXSearch(ctx context.Context, query string, maxTweets int, mode, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter Search"))
	fmt.Println()

	cfg := x.DefaultConfig()
	client, err := initXClient(cfg, session)
	if err != nil {
		return err
	}

	// Set search mode
	switch strings.ToLower(mode) {
	case "latest":
		client.SetSearchMode(x.SearchLatest)
	case "photos":
		client.SetSearchMode(x.SearchPhotos)
	case "videos":
		client.SetSearchMode(x.SearchVideos)
	default:
		client.SetSearchMode(x.SearchTop)
	}

	fmt.Printf("  Query:      %s\n", infoStyle.Render(query))
	fmt.Printf("  Mode:       %s\n", labelStyle.Render(mode))
	fmt.Printf("  Max tweets: %s\n", labelStyle.Render(fmt.Sprintf("%d", maxTweets)))
	fmt.Println()

	start := time.Now()
	tweets, err := client.SearchTweets(ctx, query, maxTweets, func(p x.Progress) {
		if !p.Done {
			fmt.Printf("\r  Searching: %s tweets",
				infoStyle.Render(formatLargeNumber(p.Current)))
		}
	})

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d tweets)", err, len(tweets))))
	}

	if len(tweets) == 0 {
		fmt.Println(warningStyle.Render("  No tweets found"))
		return nil
	}

	// Store in DuckDB
	dbPath := cfg.SearchDBPath(query)
	db, err := x.OpenDB(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := db.InsertTweets(tweets); err != nil {
		return fmt.Errorf("insert tweets: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Found %d tweets in %s",
		len(tweets), time.Since(start).Truncate(time.Second))))
	fmt.Printf("  Database: %s\n", labelStyle.Render(dbPath))

	// Count unique users
	users := make(map[string]bool)
	for _, t := range tweets {
		users[t.Username] = true
	}
	photoCount, videoCount := countMedia(tweets)
	fmt.Printf("  Users: %d  |  Photos: %d  |  Videos: %d\n", len(users), photoCount, videoCount)

	// Show top tweets
	fmt.Println()
	fmt.Println(subtitleStyle.Render("  Top results:"))
	top, _ := db.TopTweets(10)
	for i, t := range top {
		text := strings.ReplaceAll(t.Text, "\n", " ")
		if len(text) > 55 {
			text = text[:55] + "..."
		}
		fmt.Printf("  %d. @%-16s %s likes  %s\n",
			i+1,
			infoStyle.Render(t.Username),
			labelStyle.Render(formatLargeNumber(int64(t.Likes))),
			text)
	}

	return nil
}

// ── hashtag ─────────────────────────────────────────────

func newXHashtag() *cobra.Command {
	var (
		maxTweets int
		mode      string
		session   string
	)

	cmd := &cobra.Command{
		Use:   "hashtag <tag>",
		Short: "Search by hashtag",
		Long: `Search X/Twitter for tweets with a specific hashtag.

Results are stored in a DuckDB database.

Examples:
  search x hashtag golang --session myuser
  search x hashtag golang --mode latest --max-tweets 50 --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tag := strings.TrimPrefix(args[0], "#")
			return runXHashtag(cmd.Context(), tag, maxTweets, mode, session)
		},
	}

	cmd.Flags().IntVar(&maxTweets, "max-tweets", 100, "Max tweets to fetch")
	cmd.Flags().StringVar(&mode, "mode", "latest", "Search mode: top, latest")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	return cmd
}

func runXHashtag(ctx context.Context, tag string, maxTweets int, mode, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter Hashtag"))
	fmt.Println()

	cfg := x.DefaultConfig()
	client, err := initXClient(cfg, session)
	if err != nil {
		return err
	}

	// Set search mode
	switch strings.ToLower(mode) {
	case "latest":
		client.SetSearchMode(x.SearchLatest)
	default:
		client.SetSearchMode(x.SearchTop)
	}

	query := "#" + tag
	fmt.Printf("  Hashtag:    %s\n", infoStyle.Render(query))
	fmt.Printf("  Mode:       %s\n", labelStyle.Render(mode))
	fmt.Printf("  Max tweets: %s\n", labelStyle.Render(fmt.Sprintf("%d", maxTweets)))
	fmt.Printf("  Data:       %s\n", labelStyle.Render(cfg.HashtagDir(tag)))
	fmt.Println()

	start := time.Now()
	tweets, err := client.SearchTweets(ctx, query, maxTweets, func(p x.Progress) {
		if !p.Done {
			fmt.Printf("\r  Fetching: %s tweets",
				infoStyle.Render(formatLargeNumber(p.Current)))
		}
	})

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d tweets)", err, len(tweets))))
	}

	if len(tweets) == 0 {
		fmt.Println(warningStyle.Render("  No tweets found"))
		return nil
	}

	// Store in DuckDB
	dbPath := cfg.HashtagDBPath(tag)
	db, err := x.OpenDB(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := db.InsertTweets(tweets); err != nil {
		return fmt.Errorf("insert tweets: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Fetched %d tweets in %s",
		len(tweets), time.Since(start).Truncate(time.Second))))
	fmt.Printf("  Database: %s\n", labelStyle.Render(dbPath))

	// Show top tweets
	fmt.Println()
	fmt.Println(subtitleStyle.Render("  Top tweets by likes:"))
	top, _ := db.TopTweets(5)
	for i, t := range top {
		text := strings.ReplaceAll(t.Text, "\n", " ")
		if len(text) > 55 {
			text = text[:55] + "..."
		}
		fmt.Printf("  %d. @%-16s %s likes  %s\n",
			i+1,
			infoStyle.Render(t.Username),
			labelStyle.Render(formatLargeNumber(int64(t.Likes))),
			text)
	}

	return nil
}

// ── followers ───────────────────────────────────────────

func newXFollowers() *cobra.Command {
	var (
		maxUsers int
		session  string
	)

	cmd := &cobra.Command{
		Use:   "followers <username>",
		Short: "Fetch follower list",
		Long: `Fetch the follower list for an X/Twitter user.

Followers are stored in the users table.

Examples:
  search x followers karpathy --session myuser
  search x followers karpathy --max-users 100 --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runXFollowList(cmd.Context(), username, "followers", maxUsers, session)
		},
	}

	cmd.Flags().IntVar(&maxUsers, "max-users", 200, "Max users to fetch")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	return cmd
}

func newXFollowing() *cobra.Command {
	var (
		maxUsers int
		session  string
	)

	cmd := &cobra.Command{
		Use:   "following <username>",
		Short: "Fetch following list",
		Long: `Fetch the following list for an X/Twitter user.

Following users are stored in the users table.

Examples:
  search x following karpathy --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runXFollowList(cmd.Context(), username, "following", maxUsers, session)
		},
	}

	cmd.Flags().IntVar(&maxUsers, "max-users", 200, "Max users to fetch")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	return cmd
}

func runXFollowList(ctx context.Context, username, listType string, maxUsers int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter " + capitalizeFirst(listType)))
	fmt.Println()

	cfg := x.DefaultConfig()
	client, err := initXClient(cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  Fetching %s for %s\n", listType, infoStyle.Render("@"+username))
	fmt.Printf("  Max users: %s\n", labelStyle.Render(fmt.Sprintf("%d", maxUsers)))
	fmt.Println()

	start := time.Now()
	var users []x.FollowUser
	progressCb := func(p x.Progress) {
		if !p.Done {
			fmt.Printf("\r  %s: %s",
				capitalizeFirst(listType),
				infoStyle.Render(formatLargeNumber(p.Current)))
		}
	}

	if listType == "followers" {
		users, err = client.GetFollowers(ctx, username, maxUsers, progressCb)
	} else {
		users, err = client.GetFollowing(ctx, username, maxUsers, progressCb)
	}

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d users)", err, len(users))))
	}

	if len(users) == 0 {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  No %s found", listType)))
		return nil
	}

	// Store in DuckDB
	db, err := x.OpenDB(cfg.UserDBPath(username))
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := db.InsertFollowUsers(users); err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: insert users: %v", err)))
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Fetched %d %s in %s",
		len(users), listType, time.Since(start).Truncate(time.Second))))
	fmt.Printf("  Database: %s\n", labelStyle.Render(cfg.UserDBPath(username)))

	// Show first 10
	fmt.Println()
	showN := min(10, len(users))
	for i := range showN {
		u := users[i]
		verified := ""
		if u.IsVerified {
			verified = " [verified]"
		}
		followers := ""
		if u.FollowersCount > 0 {
			followers = fmt.Sprintf("  (%s followers)", formatLargeNumber(int64(u.FollowersCount)))
		}
		fmt.Printf("  @%-20s %s%s%s\n",
			infoStyle.Render(u.Username), u.Name, verified, followers)
	}
	if len(users) > showN {
		fmt.Printf("  ... and %d more\n", len(users)-showN)
	}

	return nil
}

// ── trends ──────────────────────────────────────────────

func newXTrends() *cobra.Command {
	var session string

	cmd := &cobra.Command{
		Use:   "trends",
		Short: "Show current trending topics",
		Long: `Show current trending topics on X/Twitter.

Examples:
  search x trends --session myuser`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runXTrends(session)
		},
	}

	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	return cmd
}

func runXTrends(session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter Trends"))
	fmt.Println()

	cfg := x.DefaultConfig()
	client, err := initXClient(cfg, session)
	if err != nil {
		return err
	}

	trends, err := client.GetTrends()
	if err != nil {
		return fmt.Errorf("get trends: %w", err)
	}

	if len(trends) == 0 {
		fmt.Println(warningStyle.Render("  No trends found"))
		return nil
	}

	fmt.Printf("  %d trending topics:\n\n", len(trends))
	for i, t := range trends {
		fmt.Printf("  %2d. %s\n", i+1, infoStyle.Render(t))
	}
	fmt.Println()

	return nil
}

// ── info ────────────────────────────────────────────────

func newXInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <username>",
		Short: "Show stored data statistics",
		Long: `Show statistics for previously scraped X/Twitter data.

Examples:
  search x info karpathy`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runXInfo(username)
		},
	}
	return cmd
}

func runXInfo(username string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter Info"))
	fmt.Println()

	cfg := x.DefaultConfig()

	// Check if data exists
	dir := cfg.UserDir(username)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("no data found for @%s (expected at %s)", username, dir)
	}

	// Load profile
	profile, err := x.LoadProfile(cfg, username)
	if err == nil {
		displayXProfile(profile)
	} else {
		fmt.Printf("  @%s\n\n", infoStyle.Render(username))
	}

	// Check database
	dbPath := cfg.UserDBPath(username)
	if _, err := os.Stat(dbPath); err == nil {
		db, err := x.OpenDB(dbPath)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		defer db.Close()

		stats, err := db.GetStats()
		if err != nil {
			return fmt.Errorf("get stats: %w", err)
		}

		fmt.Println(titleStyle.Render("  Database"))
		fmt.Printf("  Tweets:     %s\n", infoStyle.Render(formatLargeNumber(stats.Tweets)))
		fmt.Printf("  Users:      %s\n", infoStyle.Render(formatLargeNumber(stats.Users)))
		fmt.Printf("  DB Size:    %s\n", labelStyle.Render(formatBytes(stats.DBSize)))
		fmt.Printf("  Path:       %s\n", labelStyle.Render(dbPath))
		fmt.Println()

		// Show top tweets
		top, _ := db.TopTweets(5)
		if len(top) > 0 {
			fmt.Println(titleStyle.Render("  Top Tweets"))
			for i, t := range top {
				text := t.Text
				if len(text) > 50 {
					text = text[:50] + "..."
				}
				text = strings.ReplaceAll(text, "\n", " ")
				fmt.Printf("  %d. %s likes  %s RT  %s views  %s  %s\n",
					i+1,
					infoStyle.Render(formatLargeNumber(int64(t.Likes))),
					labelStyle.Render(formatLargeNumber(int64(t.Retweets))),
					labelStyle.Render(formatLargeNumber(int64(t.Views))),
					labelStyle.Render(t.PostedAt.Format("2006-01-02")),
					text)
			}
			fmt.Println()
		}
	} else {
		fmt.Println(warningStyle.Render("  No tweets database found"))
	}

	// Check media directory
	mediaDir := cfg.UserMediaDir(username)
	if entries, err := os.ReadDir(mediaDir); err == nil && len(entries) > 0 {
		var totalSize int64
		for _, e := range entries {
			if info, err := e.Info(); err == nil {
				totalSize += info.Size()
			}
		}
		fmt.Printf("  Media files: %d (%s)\n", len(entries), formatBytes(totalSize))
		fmt.Printf("  Media dir:   %s\n", labelStyle.Render(mediaDir))
	}

	return nil
}

// ── media ───────────────────────────────────────────────

func newXMedia() *cobra.Command {
	var (
		maxTweets int
		session   string
	)

	cmd := &cobra.Command{
		Use:   "media <username>",
		Short: "Fetch media-only timeline",
		Long: `Fetch tweets with media (photos/videos) for an X/Twitter user.

Only tweets containing media attachments are returned.
Tweets are stored in a DuckDB database at $HOME/data/x/{username}/tweets.duckdb

Examples:
  search x media karpathy --session myuser
  search x media karpathy --max-tweets 100 --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runXMedia(cmd.Context(), username, maxTweets, session)
		},
	}

	cmd.Flags().IntVar(&maxTweets, "max-tweets", 200, "Max tweets to fetch")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load")
	return cmd
}

func runXMedia(ctx context.Context, username string, maxTweets int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter Media"))
	fmt.Println()

	cfg := x.DefaultConfig()
	client, err := initXClient(cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  Fetching media tweets for %s\n", infoStyle.Render("@"+username))
	fmt.Printf("  Max tweets: %s\n", labelStyle.Render(fmt.Sprintf("%d", maxTweets)))
	fmt.Println()

	start := time.Now()
	tweets, err := client.GetMediaTweets(ctx, username, maxTweets, func(p x.Progress) {
		if !p.Done {
			fmt.Printf("\r  Fetching media: %s",
				infoStyle.Render(formatLargeNumber(p.Current)))
		}
	})

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d tweets)", err, len(tweets))))
	}

	if len(tweets) == 0 {
		fmt.Println(warningStyle.Render("  No media tweets found"))
		return nil
	}

	// Store in DuckDB
	db, err := x.OpenDB(cfg.UserDBPath(username))
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := db.InsertTweets(tweets); err != nil {
		return fmt.Errorf("insert tweets: %w", err)
	}

	photoCount, videoCount, gifCount := countAllMedia(tweets)
	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Fetched %d media tweets in %s",
		len(tweets), time.Since(start).Truncate(time.Second))))
	fmt.Printf("  Photos: %d  |  Videos: %d  |  GIFs: %d\n", photoCount, videoCount, gifCount)
	fmt.Printf("  Database: %s\n", labelStyle.Render(cfg.UserDBPath(username)))

	return nil
}

// ── search-users ────────────────────────────────────────

func newXSearchUsers() *cobra.Command {
	var (
		maxUsers int
		session  string
	)

	cmd := &cobra.Command{
		Use:   "search-users <query>",
		Short: "Search for user profiles",
		Long: `Search X/Twitter for user profiles matching a query.

Results are stored in the users table.

Examples:
  search x search-users "golang" --session myuser
  search x search-users "AI researcher" --max-users 50 --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runXSearchUsers(cmd.Context(), args[0], maxUsers, session)
		},
	}

	cmd.Flags().IntVar(&maxUsers, "max-users", 50, "Max users to fetch")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load")
	return cmd
}

func runXSearchUsers(ctx context.Context, query string, maxUsers int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter Search Users"))
	fmt.Println()

	cfg := x.DefaultConfig()
	client, err := initXClient(cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  Query:     %s\n", infoStyle.Render(query))
	fmt.Printf("  Max users: %s\n", labelStyle.Render(fmt.Sprintf("%d", maxUsers)))
	fmt.Println()

	start := time.Now()
	users, err := client.SearchProfiles(ctx, query, maxUsers, func(p x.Progress) {
		if !p.Done {
			fmt.Printf("\r  Searching: %s users",
				infoStyle.Render(formatLargeNumber(p.Current)))
		}
	})

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d users)", err, len(users))))
	}

	if len(users) == 0 {
		fmt.Println(warningStyle.Render("  No users found"))
		return nil
	}

	// Store in DuckDB
	dbPath := filepath.Join(cfg.DataDir, "search", sanitizeXDirName(query), "tweets.duckdb")
	db, err := x.OpenDB(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := db.InsertFollowUsers(users); err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: insert users: %v", err)))
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Found %d users in %s",
		len(users), time.Since(start).Truncate(time.Second))))

	// Show results
	fmt.Println()
	showN := min(10, len(users))
	for i := range showN {
		u := users[i]
		verified := ""
		if u.IsVerified {
			verified = " [verified]"
		}
		followers := ""
		if u.FollowersCount > 0 {
			followers = fmt.Sprintf("  (%s followers)", formatLargeNumber(int64(u.FollowersCount)))
		}
		fmt.Printf("  @%-20s %s%s%s\n",
			infoStyle.Render(u.Username), u.Name, verified, followers)
	}
	if len(users) > showN {
		fmt.Printf("  ... and %d more\n", len(users)-showN)
	}

	return nil
}

// ── bookmarks ───────────────────────────────────────────

func newXBookmarks() *cobra.Command {
	var (
		maxTweets int
		session   string
	)

	cmd := &cobra.Command{
		Use:   "bookmarks",
		Short: "Fetch bookmarked tweets",
		Long: `Fetch the authenticated user's bookmarked tweets.

Requires authentication. Bookmarks are stored in $HOME/data/x/bookmarks/tweets.duckdb

Examples:
  search x bookmarks --session myuser
  search x bookmarks --max-tweets 200 --session myuser`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runXBookmarks(cmd.Context(), maxTweets, session)
		},
	}

	cmd.Flags().IntVar(&maxTweets, "max-tweets", 200, "Max tweets to fetch")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load")
	return cmd
}

func runXBookmarks(ctx context.Context, maxTweets int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter Bookmarks"))
	fmt.Println()

	cfg := x.DefaultConfig()
	client, err := initXClient(cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  Max tweets: %s\n", labelStyle.Render(fmt.Sprintf("%d", maxTweets)))
	fmt.Println()

	start := time.Now()
	tweets, err := client.GetBookmarks(ctx, maxTweets, func(p x.Progress) {
		if !p.Done {
			fmt.Printf("\r  Fetching bookmarks: %s",
				infoStyle.Render(formatLargeNumber(p.Current)))
		}
	})

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d tweets)", err, len(tweets))))
	}

	if len(tweets) == 0 {
		fmt.Println(warningStyle.Render("  No bookmarks found"))
		return nil
	}

	// Store in DuckDB
	dbPath := filepath.Join(cfg.DataDir, "bookmarks", "tweets.duckdb")
	db, err := x.OpenDB(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := db.InsertTweets(tweets); err != nil {
		return fmt.Errorf("insert tweets: %w", err)
	}

	// Count unique users
	users := make(map[string]bool)
	for _, t := range tweets {
		users[t.Username] = true
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Fetched %d bookmarks in %s",
		len(tweets), time.Since(start).Truncate(time.Second))))
	fmt.Printf("  Users: %d\n", len(users))
	fmt.Printf("  Database: %s\n", labelStyle.Render(dbPath))

	// Show top bookmarks
	fmt.Println()
	fmt.Println(subtitleStyle.Render("  Top bookmarks by likes:"))
	top, _ := db.TopTweets(5)
	for i, t := range top {
		text := strings.ReplaceAll(t.Text, "\n", " ")
		if len(text) > 55 {
			text = text[:55] + "..."
		}
		fmt.Printf("  %d. @%-16s %s likes  %s\n",
			i+1,
			infoStyle.Render(t.Username),
			labelStyle.Render(formatLargeNumber(int64(t.Likes))),
			text)
	}

	return nil
}

// ── home ────────────────────────────────────────────────

func newXHome() *cobra.Command {
	var (
		maxTweets int
		session   string
	)

	cmd := &cobra.Command{
		Use:   "home",
		Short: "Fetch home timeline",
		Long: `Fetch the authenticated user's home timeline (Following tab).

Stored in $HOME/data/x/home/tweets.duckdb

Examples:
  search x home --session myuser
  search x home --max-tweets 100 --session myuser`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runXTimeline(cmd.Context(), "home", maxTweets, session)
		},
	}

	cmd.Flags().IntVar(&maxTweets, "max-tweets", 100, "Max tweets to fetch")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load")
	return cmd
}

func newXForYou() *cobra.Command {
	var (
		maxTweets int
		session   string
	)

	cmd := &cobra.Command{
		Use:   "foryou",
		Short: `Fetch "For You" timeline`,
		Long: `Fetch the authenticated user's "For You" algorithmic timeline.

Stored in $HOME/data/x/foryou/tweets.duckdb

Examples:
  search x foryou --session myuser
  search x foryou --max-tweets 100 --session myuser`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runXTimeline(cmd.Context(), "foryou", maxTweets, session)
		},
	}

	cmd.Flags().IntVar(&maxTweets, "max-tweets", 100, "Max tweets to fetch")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load")
	return cmd
}

func runXTimeline(ctx context.Context, timelineType string, maxTweets int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter " + capitalizeFirst(timelineType) + " Timeline"))
	fmt.Println()

	cfg := x.DefaultConfig()
	client, err := initXClient(cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  Max tweets: %s\n", labelStyle.Render(fmt.Sprintf("%d", maxTweets)))
	fmt.Println()

	start := time.Now()
	var tweets []x.Tweet
	progressCb := func(p x.Progress) {
		if !p.Done {
			fmt.Printf("\r  Fetching %s: %s tweets",
				timelineType,
				infoStyle.Render(formatLargeNumber(p.Current)))
		}
	}

	switch timelineType {
	case "home":
		tweets, err = client.GetHomeTweets(ctx, maxTweets, progressCb)
	case "foryou":
		tweets, err = client.GetForYouTweets(ctx, maxTweets, progressCb)
	}

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d tweets)", err, len(tweets))))
	}

	if len(tweets) == 0 {
		fmt.Println(warningStyle.Render("  No tweets found"))
		return nil
	}

	// Store in DuckDB
	dbPath := filepath.Join(cfg.DataDir, timelineType, "tweets.duckdb")
	db, err := x.OpenDB(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := db.InsertTweets(tweets); err != nil {
		return fmt.Errorf("insert tweets: %w", err)
	}

	users := make(map[string]bool)
	for _, t := range tweets {
		users[t.Username] = true
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Fetched %d tweets in %s",
		len(tweets), time.Since(start).Truncate(time.Second))))
	fmt.Printf("  Users: %d\n", len(users))
	fmt.Printf("  Database: %s\n", labelStyle.Render(dbPath))

	// Show top tweets
	fmt.Println()
	fmt.Println(subtitleStyle.Render("  Top tweets by likes:"))
	top, _ := db.TopTweets(5)
	for i, t := range top {
		text := strings.ReplaceAll(t.Text, "\n", " ")
		if len(text) > 55 {
			text = text[:55] + "..."
		}
		fmt.Printf("  %d. @%-16s %s likes  %s\n",
			i+1,
			infoStyle.Render(t.Username),
			labelStyle.Render(formatLargeNumber(int64(t.Likes))),
			text)
	}

	return nil
}

// ── retweeters ──────────────────────────────────────────

func newXRetweeters() *cobra.Command {
	var (
		maxUsers int
		session  string
	)

	cmd := &cobra.Command{
		Use:   "retweeters <tweet_id_or_url>",
		Short: "Fetch who retweeted a tweet",
		Long: `Fetch the list of users who retweeted a specific tweet.

Accepts tweet ID or full URL.

Examples:
  search x retweeters 1234567890 --session myuser
  search x retweeters https://x.com/user/status/1234567890 --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := extractTweetID(args[0])
			return runXRetweeters(id, maxUsers, session)
		},
	}

	cmd.Flags().IntVar(&maxUsers, "max-users", 100, "Max users to fetch")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load")
	return cmd
}

func runXRetweeters(tweetID string, maxUsers int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter Retweeters"))
	fmt.Println()

	cfg := x.DefaultConfig()
	client, err := initXClient(cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  Tweet: %s\n", infoStyle.Render(tweetID))
	fmt.Println()

	users, err := client.GetRetweeters(tweetID, maxUsers)
	if err != nil {
		return err
	}

	if len(users) == 0 {
		fmt.Println(warningStyle.Render("  No retweeters found"))
		return nil
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("  Found %d retweeters", len(users))))
	fmt.Println()

	for i, u := range users {
		verified := ""
		if u.IsVerified {
			verified = " [verified]"
		}
		followers := ""
		if u.FollowersCount > 0 {
			followers = fmt.Sprintf("  (%s followers)", formatLargeNumber(int64(u.FollowersCount)))
		}
		fmt.Printf("  %d. @%-20s %s%s%s\n",
			i+1,
			infoStyle.Render(u.Username), u.Name, verified, followers)
	}

	return nil
}

// ── favoriters ──────────────────────────────────────────

func newXFavoriters() *cobra.Command {
	var (
		maxUsers int
		session  string
	)

	cmd := &cobra.Command{
		Use:   "favoriters <tweet_id_or_url>",
		Short: "Fetch who liked a tweet",
		Long: `Fetch the list of users who liked (favorited) a specific tweet.

Accepts tweet ID or full URL.

Examples:
  search x favoriters 1234567890 --session myuser
  search x favoriters https://x.com/user/status/1234567890 --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := extractTweetID(args[0])
			return runXFavoriters(id, maxUsers, session)
		},
	}

	cmd.Flags().IntVar(&maxUsers, "max-users", 100, "Max users to fetch")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load")
	return cmd
}

func runXFavoriters(tweetID string, maxUsers int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter Favoriters"))
	fmt.Println()

	cfg := x.DefaultConfig()
	client, err := initXClient(cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  Tweet: %s\n", infoStyle.Render(tweetID))
	fmt.Println()

	users, err := client.GetFavoriters(tweetID, maxUsers)
	if err != nil {
		return err
	}

	if len(users) == 0 {
		fmt.Println(warningStyle.Render("  No favoriters found"))
		return nil
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("  Found %d favoriters", len(users))))
	fmt.Println()

	for i, u := range users {
		verified := ""
		if u.IsVerified {
			verified = " [verified]"
		}
		followers := ""
		if u.FollowersCount > 0 {
			followers = fmt.Sprintf("  (%s followers)", formatLargeNumber(int64(u.FollowersCount)))
		}
		fmt.Printf("  %d. @%-20s %s%s%s\n",
			i+1,
			infoStyle.Render(u.Username), u.Name, verified, followers)
	}

	return nil
}

// ── list ────────────────────────────────────────────────

func newXList() *cobra.Command {
	var session string

	cmd := &cobra.Command{
		Use:   "list <id>",
		Short: "Fetch list info",
		Long: `Fetch information about an X/Twitter list by ID.

Examples:
  search x list 1234567890 --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runXList(args[0], session)
		},
	}

	cmd.Flags().StringVar(&session, "session", "", "Session username to load")
	return cmd
}

func runXList(id, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter List"))
	fmt.Println()

	cfg := x.DefaultConfig()
	client, err := initXClient(cfg, session)
	if err != nil {
		return err
	}

	list, err := client.GetListByID(id)
	if err != nil {
		return err
	}

	fmt.Printf("  Name:        %s\n", titleStyle.Render(list.Name))
	fmt.Printf("  ID:          %s\n", labelStyle.Render(list.ID))
	if list.Description != "" {
		fmt.Printf("  Description: %s\n", list.Description)
	}
	fmt.Printf("  Members:     %s\n", infoStyle.Render(formatLargeNumber(int64(list.MemberCount))))
	if list.OwnerName != "" {
		fmt.Printf("  Owner:       %s\n", infoStyle.Render("@"+list.OwnerName))
	}
	fmt.Println()

	return nil
}

// ── list-tweets ─────────────────────────────────────────

func newXListTweets() *cobra.Command {
	var (
		maxTweets int
		session   string
	)

	cmd := &cobra.Command{
		Use:   "list-tweets <list_id>",
		Short: "Fetch tweets from a list",
		Long: `Fetch tweets from an X/Twitter list timeline.

Stored in $HOME/data/x/list/{id}/tweets.duckdb

Examples:
  search x list-tweets 1234567890 --session myuser
  search x list-tweets 1234567890 --max-tweets 200 --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runXListTweets(cmd.Context(), args[0], maxTweets, session)
		},
	}

	cmd.Flags().IntVar(&maxTweets, "max-tweets", 200, "Max tweets to fetch")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load")
	return cmd
}

func runXListTweets(ctx context.Context, listID string, maxTweets int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter List Tweets"))
	fmt.Println()

	cfg := x.DefaultConfig()
	client, err := initXClient(cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  List ID:    %s\n", infoStyle.Render(listID))
	fmt.Printf("  Max tweets: %s\n", labelStyle.Render(fmt.Sprintf("%d", maxTweets)))
	fmt.Println()

	start := time.Now()
	tweets, err := client.GetListTweets(ctx, listID, maxTweets, func(p x.Progress) {
		if !p.Done {
			fmt.Printf("\r  Fetching: %s tweets",
				infoStyle.Render(formatLargeNumber(p.Current)))
		}
	})

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d tweets)", err, len(tweets))))
	}

	if len(tweets) == 0 {
		fmt.Println(warningStyle.Render("  No tweets found"))
		return nil
	}

	// Store in DuckDB
	dbPath := filepath.Join(cfg.DataDir, "list", listID, "tweets.duckdb")
	db, err := x.OpenDB(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := db.InsertTweets(tweets); err != nil {
		return fmt.Errorf("insert tweets: %w", err)
	}

	users := make(map[string]bool)
	for _, t := range tweets {
		users[t.Username] = true
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Fetched %d tweets in %s",
		len(tweets), time.Since(start).Truncate(time.Second))))
	fmt.Printf("  Users: %d\n", len(users))
	fmt.Printf("  Database: %s\n", labelStyle.Render(dbPath))

	// Show top tweets
	fmt.Println()
	fmt.Println(subtitleStyle.Render("  Top tweets by likes:"))
	top, _ := db.TopTweets(5)
	for i, t := range top {
		text := strings.ReplaceAll(t.Text, "\n", " ")
		if len(text) > 55 {
			text = text[:55] + "..."
		}
		fmt.Printf("  %d. @%-16s %s likes  %s\n",
			i+1,
			infoStyle.Render(t.Username),
			labelStyle.Render(formatLargeNumber(int64(t.Likes))),
			text)
	}

	return nil
}

// ── list-members ────────────────────────────────────────

func newXListMembers() *cobra.Command {
	var (
		maxUsers int
		session  string
	)

	cmd := &cobra.Command{
		Use:   "list-members <list_id>",
		Short: "Fetch members of a list",
		Long: `Fetch the members of an X/Twitter list.

Examples:
  search x list-members 1234567890 --session myuser
  search x list-members 1234567890 --max-users 500 --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runXListMembers(cmd.Context(), args[0], maxUsers, session)
		},
	}

	cmd.Flags().IntVar(&maxUsers, "max-users", 200, "Max users to fetch")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load")
	return cmd
}

func runXListMembers(ctx context.Context, listID string, maxUsers int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter List Members"))
	fmt.Println()

	cfg := x.DefaultConfig()
	client, err := initXClient(cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  List ID:   %s\n", infoStyle.Render(listID))
	fmt.Printf("  Max users: %s\n", labelStyle.Render(fmt.Sprintf("%d", maxUsers)))
	fmt.Println()

	start := time.Now()
	users, err := client.GetListMembers(ctx, listID, maxUsers, func(p x.Progress) {
		if !p.Done {
			fmt.Printf("\r  Fetching: %s members",
				infoStyle.Render(formatLargeNumber(p.Current)))
		}
	})

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d users)", err, len(users))))
	}

	if len(users) == 0 {
		fmt.Println(warningStyle.Render("  No members found"))
		return nil
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("  Found %d members in %s",
		len(users), time.Since(start).Truncate(time.Second))))
	fmt.Println()

	showN := min(20, len(users))
	for i := range showN {
		u := users[i]
		verified := ""
		if u.IsVerified {
			verified = " [verified]"
		}
		followers := ""
		if u.FollowersCount > 0 {
			followers = fmt.Sprintf("  (%s followers)", formatLargeNumber(int64(u.FollowersCount)))
		}
		fmt.Printf("  @%-20s %s%s%s\n",
			infoStyle.Render(u.Username), u.Name, verified, followers)
	}
	if len(users) > showN {
		fmt.Printf("  ... and %d more\n", len(users)-showN)
	}

	return nil
}

// ── space ───────────────────────────────────────────────

func newXSpace() *cobra.Command {
	var session string

	cmd := &cobra.Command{
		Use:   "space <id>",
		Short: "Fetch audio space info",
		Long: `Fetch information about an X/Twitter audio space.

Examples:
  search x space 1vAxRkrPeqjKl --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runXSpace(args[0], session)
		},
	}

	cmd.Flags().StringVar(&session, "session", "", "Session username to load")
	return cmd
}

func runXSpace(id, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter Space"))
	fmt.Println()

	cfg := x.DefaultConfig()
	client, err := initXClient(cfg, session)
	if err != nil {
		return err
	}

	space, err := client.GetSpace(id)
	if err != nil {
		return err
	}

	fmt.Printf("  Title:     %s\n", titleStyle.Render(space.Title))
	fmt.Printf("  ID:        %s\n", labelStyle.Render(space.ID))
	fmt.Printf("  State:     %s\n", infoStyle.Render(space.State))
	if !space.CreatedAt.IsZero() {
		fmt.Printf("  Created:   %s\n", labelStyle.Render(space.CreatedAt.Format("2006-01-02 15:04:05")))
	}
	if !space.StartedAt.IsZero() {
		fmt.Printf("  Started:   %s\n", labelStyle.Render(space.StartedAt.Format("2006-01-02 15:04:05")))
	}
	if !space.UpdatedAt.IsZero() {
		fmt.Printf("  Updated:   %s\n", labelStyle.Render(space.UpdatedAt.Format("2006-01-02 15:04:05")))
	}
	fmt.Println()

	return nil
}

// ── download ────────────────────────────────────────────

func newXDownload() *cobra.Command {
	var (
		photos  bool
		videos  bool
		gifs    bool
		workers int
	)

	cmd := &cobra.Command{
		Use:   "download <username>",
		Short: "Download media from stored tweets",
		Long: `Download media (photos, videos, GIFs) from previously scraped tweets.

Reads from the DuckDB database and downloads to $HOME/data/x/{username}/media/
Skips already-downloaded files.

Examples:
  search x download karpathy --photos --videos
  search x download karpathy --photos --videos --gifs --workers 16`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			if !photos && !videos && !gifs {
				photos = true // default to photos if nothing specified
			}
			return runXDownload(cmd.Context(), username, photos, videos, gifs, workers)
		},
	}

	cmd.Flags().BoolVar(&photos, "photos", false, "Download photos")
	cmd.Flags().BoolVar(&videos, "videos", false, "Download videos")
	cmd.Flags().BoolVar(&gifs, "gifs", false, "Download GIFs")
	cmd.Flags().IntVar(&workers, "workers", 8, "Number of download workers")
	return cmd
}

func runXDownload(ctx context.Context, username string, photos, videos, gifs bool, workers int) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter Download"))
	fmt.Println()

	cfg := x.DefaultConfig()
	dbPath := cfg.UserDBPath(username)

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("no tweets found for @%s (run 'search x tweets %s' first)", username, username)
	}

	db, err := x.OpenDB(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	tweets, err := db.AllTweets()
	if err != nil {
		return fmt.Errorf("query tweets: %w", err)
	}

	items := x.ExtractMedia(tweets, photos, videos, gifs)
	if len(items) == 0 {
		fmt.Println(warningStyle.Render("  No media to download"))
		return nil
	}

	mediaTypes := []string{}
	if photos {
		mediaTypes = append(mediaTypes, "photos")
	}
	if videos {
		mediaTypes = append(mediaTypes, "videos")
	}
	if gifs {
		mediaTypes = append(mediaTypes, "GIFs")
	}

	mediaDir := cfg.UserMediaDir(username)
	fmt.Printf("  User:    %s\n", infoStyle.Render("@"+username))
	fmt.Printf("  Items:   %s (%s)\n", infoStyle.Render(formatLargeNumber(int64(len(items)))), strings.Join(mediaTypes, ", "))
	fmt.Printf("  Workers: %s\n", labelStyle.Render(fmt.Sprintf("%d", workers)))
	fmt.Printf("  Output:  %s\n", labelStyle.Render(mediaDir))
	fmt.Println()

	start := time.Now()
	err = x.DownloadMedia(ctx, items, mediaDir, workers, func(p x.DownloadProgress) {
		if !p.Done {
			fmt.Printf("\r  Progress: %d/%d downloaded  %d skipped  %d failed  %s",
				p.Downloaded, p.Total, p.Skipped, p.Failed,
				formatBytes(p.Bytes))
		}
	})

	fmt.Println()
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Download complete in %s",
		time.Since(start).Truncate(time.Second))))
	fmt.Printf("  Media dir: %s\n", labelStyle.Render(mediaDir))

	return nil
}

// ── export ──────────────────────────────────────────────

func newXExport() *cobra.Command {
	var (
		format string
	)

	cmd := &cobra.Command{
		Use:   "export <username>",
		Short: "Export tweets to JSON/CSV/RSS",
		Long: `Export previously scraped tweets to JSON, CSV, or RSS format.

Reads from the DuckDB database and exports to $HOME/data/x/{username}/export/

Formats:
  json  - JSON array of tweet objects
  csv   - CSV with headers
  rss   - RSS 2.0 XML feed

Examples:
  search x export karpathy --format json
  search x export karpathy --format csv
  search x export karpathy --format rss`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runXExport(username, format)
		},
	}

	cmd.Flags().StringVar(&format, "format", "json", "Export format: json, csv, rss")
	return cmd
}

func runXExport(username, format string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("X/Twitter Export"))
	fmt.Println()

	cfg := x.DefaultConfig()
	dbPath := cfg.UserDBPath(username)

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("no tweets found for @%s (run 'search x tweets %s' first)", username, username)
	}

	db, err := x.OpenDB(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	tweets, err := db.AllTweets()
	if err != nil {
		return fmt.Errorf("query tweets: %w", err)
	}

	if len(tweets) == 0 {
		fmt.Println(warningStyle.Render("  No tweets to export"))
		return nil
	}

	exportDir := filepath.Join(cfg.UserDir(username), "export")
	var outPath string

	switch strings.ToLower(format) {
	case "json":
		outPath = filepath.Join(exportDir, "tweets.json")
		err = x.ExportJSON(tweets, outPath)
	case "csv":
		outPath = filepath.Join(exportDir, "tweets.csv")
		err = x.ExportCSV(tweets, outPath)
	case "rss":
		outPath = filepath.Join(exportDir, "tweets.rss")
		title := fmt.Sprintf("@%s tweets", username)
		link := fmt.Sprintf("https://x.com/%s", username)
		err = x.ExportRSS(tweets, title, link, outPath)
	default:
		return fmt.Errorf("unknown format %q (use json, csv, or rss)", format)
	}

	if err != nil {
		return fmt.Errorf("export %s: %w", format, err)
	}

	fmt.Printf("  User:    %s\n", infoStyle.Render("@"+username))
	fmt.Printf("  Tweets:  %s\n", infoStyle.Render(formatLargeNumber(int64(len(tweets)))))
	fmt.Printf("  Format:  %s\n", labelStyle.Render(format))
	fmt.Printf("  Output:  %s\n", labelStyle.Render(outPath))
	fmt.Println()
	fmt.Println(successStyle.Render("  Export complete!"))

	return nil
}

// ── helpers ─────────────────────────────────────────────

func countMedia(tweets []x.Tweet) (photos, videos int) {
	for _, t := range tweets {
		photos += len(t.Photos)
		videos += len(t.Videos)
	}
	return
}

func countAllMedia(tweets []x.Tweet) (photos, videos, gifs int) {
	for _, t := range tweets {
		photos += len(t.Photos)
		videos += len(t.Videos)
		gifs += len(t.GIFs)
	}
	return
}

func sanitizeXDirName(s string) string {
	var b []byte
	for i := range len(s) {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9',
			c == '-', c == '_', c == '.':
			b = append(b, c)
		default:
			b = append(b, '_')
		}
	}
	if len(b) == 0 {
		return "_"
	}
	return string(b)
}

// fetchLinkedArticle follows a URL (including t.co redirects), fetches the HTML,
// and extracts the article title + markdown body using trafilatura.
// Returns empty strings on failure.
func fetchLinkedArticle(rawURL string) (title, body, finalURL string) {
	client := &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	finalURL = resp.Request.URL.String()

	htmlBytes, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024)) // 5 MB cap
	if err != nil || len(htmlBytes) == 0 {
		return
	}

	result := markdown.Convert(htmlBytes, finalURL)
	if result.Error != "" || result.Markdown == "" {
		return
	}

	title = result.Title
	body = result.Markdown
	return
}

// extractFirstURL returns the first HTTP(S) URL found in tweet text or URLs list.
func extractFirstURL(t x.Tweet) string {
	// Prefer URLs from the entities list (already expanded from t.co)
	for _, u := range t.URLs {
		if strings.HasPrefix(u, "http") {
			return u
		}
	}
	// Fallback: scan text for t.co or other URLs
	for _, word := range strings.Fields(t.Text) {
		if strings.HasPrefix(word, "http") {
			return strings.TrimRight(word, ".,;)")
		}
	}
	return ""
}

// extractXArticleID returns the article tweet ID from an x.com/i/article/<id> URL,
// or empty string if it's not an X Article URL.
func extractXArticleID(rawURL string) string {
	if !strings.Contains(rawURL, "/i/article/") {
		return ""
	}
	parts := strings.SplitN(rawURL, "/i/article/", 2)
	if len(parts) != 2 {
		return ""
	}
	id := parts[1]
	if idx := strings.IndexAny(id, "/?#"); idx >= 0 {
		id = id[:idx]
	}
	return strings.TrimSpace(id)
}

// fetchXArticleWithRod uses a headless Chrome browser to render an X Article page
// and extract its content. Auth cookies are needed to bypass the login wall.
func fetchXArticleWithRod(ctx context.Context, articleURL, authToken, ct0 string) (title, body string, err error) {
	chromeBin := detectChromeBinForX()

	l := launcher.New().
		Headless(true).
		Set("disable-blink-features", "AutomationControlled").
		Set("disable-dev-shm-usage", "").
		Set("no-sandbox", "").
		Set("window-size", "1280,800")
	if chromeBin != "" {
		l = l.Bin(chromeBin)
	}

	u, launchErr := l.Launch()
	if launchErr != nil {
		return "", "", fmt.Errorf("rod launch: %w", launchErr)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return "", "", fmt.Errorf("rod connect: %w", err)
	}
	defer browser.Close()

	page, err := browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return "", "", fmt.Errorf("rod page: %w", err)
	}

	pageCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	// Set auth cookies at browser level before navigation
	if authToken != "" && ct0 != "" {
		if err := browser.SetCookies([]*proto.NetworkCookieParam{
			{Name: "auth_token", Value: authToken, Domain: ".x.com", Path: "/"},
			{Name: "ct0", Value: ct0, Domain: ".x.com", Path: "/"},
		}); err != nil {
			return "", "", fmt.Errorf("rod set cookies: %w", err)
		}
	}

	if err := page.Context(pageCtx).Navigate(articleURL); err != nil {
		return "", "", fmt.Errorf("rod navigate: %w", err)
	}
	_ = page.Context(pageCtx).WaitLoad()

	// Wait specifically for the X Article content element to appear.
	for i := range 20 {
		time.Sleep(1 * time.Second)
		rendered, evalErr := page.Eval(`() => {
			const article = document.querySelector('[data-testid="twitterArticleRichTextView"]');
			if (article) return article.innerText.length;
			return 0;
		}`)
		if evalErr == nil && rendered.Value.Int() > 100 {
			break
		}
		if i == 19 {
			fmt.Fprintf(os.Stderr, "  Rod: twitterArticleRichTextView not found after 20s\n")
		}
	}

	html, err := page.HTML()
	if err != nil {
		return "", "", fmt.Errorf("rod get html: %w", err)
	}
	// Extract article content from X Article DOM and convert to markdown.
	// X Articles use Draft.js rendering with specific class names and data-testid attributes.
	extracted, _ := page.Eval(`() => {
		const titleEl = document.querySelector('[data-testid="twitter-article-title"]');
		const bodyEl = document.querySelector('[data-testid="twitterArticleReadView"]')
			|| document.querySelector('[data-testid="twitterArticleRichTextView"]');
		if (!bodyEl || bodyEl.innerText.length < 100) return '';

		const lines = [];

		function processNode(node) {
			// Code blocks: data-testid="markdown-code-block"
			if (node.dataset && node.dataset.testid === 'markdown-code-block') {
				// First child span contains the language label
				const langEl = node.querySelector('span');
				const lang = langEl ? langEl.textContent.trim() : '';
				// The code text is everything after the language + copy button
				const allText = node.innerText.trim();
				// Remove the language label and "Copy" button text from the start
				let code = allText;
				if (lang && code.startsWith(lang)) code = code.slice(lang.length).trim();
				if (code.startsWith('Copy')) code = code.slice(4).trim();
				if (code) {
					lines.push('');
					lines.push('` + "```" + `' + lang);
					lines.push(code);
					lines.push('` + "```" + `');
					lines.push('');
				}
				return;
			}

			// Images
			if (node.dataset && node.dataset.testid === 'tweetPhoto') {
				const img = node.querySelector('img[src]');
				if (img) {
					let src = img.src;
					// Use original size instead of small
					src = src.replace(/name=small/, 'name=large');
					const alt = img.alt || 'Image';
					lines.push('');
					lines.push('![' + alt + '](' + src + ')');
					lines.push('');
				}
				return;
			}

			// Videos
			if (node.tagName === 'VIDEO' || (node.dataset && node.dataset.testid === 'videoPlayer')) {
				const src = node.querySelector('source')?.src || node.querySelector('video')?.src;
				if (src) {
					lines.push('');
					lines.push('[Video](' + src + ')');
					lines.push('');
				}
				return;
			}

			// Headers (h1-h3)
			if (/^H[123]$/.test(node.tagName)) {
				const level = node.tagName[1];
				const prefix = '#'.repeat(parseInt(level)) + ' ';
				lines.push('');
				lines.push(prefix + node.innerText.trim());
				lines.push('');
				return;
			}

			// Regular text blocks (Draft.js unstyled blocks)
			if (node.classList && node.classList.contains('longform-unstyled') && node.dataset && node.dataset.block === 'true') {
				const text = processInlineContent(node);
				if (text.trim()) {
					lines.push('');
					lines.push(text.trim());
				}
				return;
			}

			// Ordered/unordered list items
			if (node.classList && (node.classList.contains('longform-ordered-list-item') || node.classList.contains('longform-unordered-list-item'))) {
				const text = processInlineContent(node);
				const prefix = node.classList.contains('longform-ordered-list-item') ? '1. ' : '- ';
				if (text.trim()) lines.push(prefix + text.trim());
				return;
			}

			// Blockquotes
			if (node.classList && node.classList.contains('longform-blockquote')) {
				const text = node.innerText.trim();
				if (text) {
					lines.push('');
					lines.push('> ' + text.replace(/\n/g, '\n> '));
					lines.push('');
				}
				return;
			}

			// Recurse into children
			for (const child of node.children || []) {
				processNode(child);
			}
		}

		function processInlineContent(node) {
			let result = '';
			for (const child of node.querySelectorAll('[data-text="true"]')) {
				result += child.textContent;
			}
			// Also pick up links
			if (!result) result = node.innerText;
			// Inline links: find <a> tags and convert to markdown links
			const links = node.querySelectorAll('a[href]');
			for (const a of links) {
				const text = a.innerText;
				const href = a.href;
				if (text && href && !href.startsWith('javascript:')) {
					result = result.replace(text, '[' + text + '](' + href + ')');
				}
			}
			return result;
		}

		processNode(bodyEl);

		const title = titleEl ? titleEl.innerText.trim() : '';
		const body = lines.join('\n').replace(/\n{3,}/g, '\n\n').trim();
		return JSON.stringify({ title, body });
	}`)

	if extracted != nil && extracted.Value.Str() != "" {
		var result struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		}
		if json.Unmarshal([]byte(extracted.Value.Str()), &result) == nil && len(result.Body) > 100 {
			return result.Title, result.Body, nil
		}
	}

	// Article element not found — likely a login wall or rate limit
	return "", "", fmt.Errorf("article content not found in rendered page (html=%d bytes)", len(html))
}

// detectChromeBinForX finds the Chrome binary path (reuses dcrawler's detection logic).
func detectChromeBinForX() string {
	if p := os.Getenv("CHROME_BIN"); p != "" {
		return p
	}
	candidates := []string{
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append([]string{
			home + "/bin/chromium",
			home + "/.local/bin/chromium",
		}, candidates...)
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// isTweetJustALink returns true when a tweet's text is essentially only a URL.
func isTweetJustALink(t x.Tweet) bool {
	text := strings.TrimSpace(t.Text)
	// Strip any URL at the end
	words := strings.Fields(text)
	if len(words) == 0 {
		return false
	}
	// Count non-URL words
	nonURL := 0
	for _, w := range words {
		if !strings.HasPrefix(w, "http") {
			nonURL++
		}
	}
	return nonURL == 0
}
