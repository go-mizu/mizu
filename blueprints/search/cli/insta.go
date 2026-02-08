package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/dcrawler/insta"
	"github.com/spf13/cobra"
)

// NewInsta creates the insta command with subcommands.
func NewInsta() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "insta",
		Short: "Instagram search and scrape",
		Long: `Search and scrape Instagram data using the web API.

Public profiles and first 12 posts work without authentication.
For full pagination, comments, hashtags, and locations: login first.
Rate-limited to ~200 requests/11min. Use --delay to adjust.

Data: $HOME/data/instagram/
Sessions: $HOME/data/instagram/.sessions/

Subcommands:
  login      Login to Instagram (saves session)
  profile    Fetch and display user profile info
  posts      Download all posts for a user
  post       Fetch a single post by shortcode
  comments   Download comments for a post
  search     Search users, hashtags, places
  hashtag    Download posts for a hashtag
  location   Download posts for a location
  download   Download media files (images/videos)
  info       Show stored data statistics
  stories    Fetch user stories (requires auth)
  highlights Fetch user highlights (requires auth)
  reels      Fetch user reels (requires auth)
  followers  Fetch user followers (requires auth)
  following  Fetch user following (requires auth)
  tagged     Fetch posts user is tagged in (requires auth)
  saved      Fetch saved posts (requires auth, own only)
  likes      Fetch users who liked a post (requires auth)

Examples:
  search insta login myuser
  search insta profile natgeo --session myuser
  search insta posts natgeo --max-posts 100 --session myuser
  search insta search "landscape photography"
  search insta download natgeo --workers 4`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newInstaLogin())
	cmd.AddCommand(newInstaImportSession())
	cmd.AddCommand(newInstaProfile())
	cmd.AddCommand(newInstaPosts())
	cmd.AddCommand(newInstaPost())
	cmd.AddCommand(newInstaComments())
	cmd.AddCommand(newInstaSearch())
	cmd.AddCommand(newInstaHashtag())
	cmd.AddCommand(newInstaLocation())
	cmd.AddCommand(newInstaDownload())
	cmd.AddCommand(newInstaInfo())
	cmd.AddCommand(newInstaStories())
	cmd.AddCommand(newInstaHighlights())
	cmd.AddCommand(newInstaReels())
	cmd.AddCommand(newInstaFollowers())
	cmd.AddCommand(newInstaFollowing())
	cmd.AddCommand(newInstaTagged())
	cmd.AddCommand(newInstaSaved())
	cmd.AddCommand(newInstaLikes())

	return cmd
}

// initClient creates and initializes an Instagram client, optionally loading a session.
func initClient(cmd *cobra.Command, cfg insta.Config, session string) (*insta.Client, error) {
	client, err := insta.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	fmt.Println(labelStyle.Render("  Initializing session..."))
	if err := client.Init(cmd.Context()); err != nil {
		return nil, fmt.Errorf("init session: %w", err)
	}

	if session != "" {
		sessionPath := cfg.SessionPath(session)
		if err := client.LoadSessionFile(sessionPath); err != nil {
			return nil, fmt.Errorf("load session %q: %w", sessionPath, err)
		}
		fmt.Printf("  Logged in as %s\n", infoStyle.Render("@"+client.Username()))
	}

	return client, nil
}

// ── login ───────────────────────────────────────────────

func newInstaLogin() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login <username>",
		Short: "Login to Instagram and save session",
		Long: `Login to Instagram with username/password and save the session.

The session is saved to $HOME/data/instagram/.sessions/{username}.json
and can be loaded by other commands via --session flag.

Examples:
  search insta login myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runInstaLogin(cmd, username)
		},
	}
	return cmd
}

func runInstaLogin(cmd *cobra.Command, username string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Login"))
	fmt.Println()

	cfg := insta.DefaultConfig()

	client, err := insta.NewClient(cfg)
	if err != nil {
		return err
	}

	fmt.Println(labelStyle.Render("  Initializing..."))
	if err := client.Init(cmd.Context()); err != nil {
		return fmt.Errorf("init: %w", err)
	}

	// Get password from env or prompt
	fmt.Printf("  Username: %s\n", infoStyle.Render("@"+username))
	password := os.Getenv("INSTA_PWD")
	if password == "" {
		fmt.Print("  Password: ")
		fmt.Scanln(&password)
	} else {
		fmt.Println("  Password: (from INSTA_PWD env)")
	}

	if password == "" {
		return fmt.Errorf("password cannot be empty (set INSTA_PWD env or enter interactively)")
	}

	fmt.Println(labelStyle.Render("  Logging in..."))
	err = client.Login(cmd.Context(), username, password)
	if err != nil {
		// Check for checkpoint
		var checkpoint *insta.CheckpointError
		if errorAs(err, &checkpoint) {
			fmt.Println()
			fmt.Println(warningStyle.Render("  Instagram requires identity verification."))
			fmt.Println(labelStyle.Render("  Requesting verification code via email..."))

			if err := client.ChallengeStart(cmd.Context(), checkpoint.URL, 1); err != nil {
				fmt.Println(warningStyle.Render(fmt.Sprintf("  Auto-challenge failed: %v", err)))
				fmt.Printf("  Visit: %s\n", urlStyle.Render("https://www.instagram.com"+checkpoint.URL))
				fmt.Println(labelStyle.Render("  Complete verification in browser, then try login again."))
				return nil
			}

			fmt.Println(successStyle.Render("  Verification code sent to your email!"))
			code := os.Getenv("INSTA_CODE")
			if code == "" {
				fmt.Print("  Enter code: ")
				fmt.Scanln(&code)
			} else {
				fmt.Printf("  Code: %s (from INSTA_CODE env)\n", code)
			}
			if code == "" {
				return fmt.Errorf("verification code cannot be empty (set INSTA_CODE env or enter interactively)")
			}

			if err := client.ChallengeVerify(cmd.Context(), checkpoint.URL, code); err != nil {
				return fmt.Errorf("challenge verification: %w", err)
			}
			client.SetUsername(username)
		} else {
			// Check for 2FA
			var twoFA *insta.TwoFactorError
			if errorAs(err, &twoFA) {
				fmt.Print("  2FA Code: ")
				var code string
				fmt.Scanln(&code)
				if code == "" {
					return fmt.Errorf("2FA code cannot be empty")
				}
				if err := client.Login2FA(cmd.Context(), username, code, twoFA.Identifier); err != nil {
					return fmt.Errorf("2FA login: %w", err)
				}
			} else {
				return err
			}
		}
	}

	// Save session
	sessionPath := cfg.SessionPath(username)
	if err := client.SaveSession(sessionPath); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("  Login successful!"))
	fmt.Printf("  Session saved to: %s\n", labelStyle.Render(sessionPath))
	fmt.Printf("  Use with: %s\n", infoStyle.Render("--session "+username))

	return nil
}

// errorAs is a type-safe wrapper for errors.As.
func errorAs[T error](err error, target *T) bool {
	return err != nil && errors.As(err, target)
}

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ── import-session ──────────────────────────────────────

func newInstaImportSession() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-session <username>",
		Short: "Import session from browser cookies",
		Long: `Import an Instagram session using cookies from your browser.

Steps:
  1. Login to Instagram in your browser
  2. Open DevTools (F12) > Application > Cookies > instagram.com
  3. Copy the values of: sessionid, csrftoken, ds_user_id
  4. Set environment variables:
     export INSTA_SESSION_ID="your_sessionid"
     export INSTA_CSRF_TOKEN="your_csrftoken"
     export INSTA_DS_USER_ID="your_ds_user_id"
  5. Run: search insta import-session <username>

Examples:
  search insta import-session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runInstaImportSession(username)
		},
	}
	return cmd
}

func runInstaImportSession(username string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Import Session"))
	fmt.Println()

	sessionID := os.Getenv("INSTA_SESSION_ID")
	csrfToken := os.Getenv("INSTA_CSRF_TOKEN")
	dsUserID := os.Getenv("INSTA_DS_USER_ID")

	if sessionID == "" {
		return fmt.Errorf("INSTA_SESSION_ID env not set (copy from browser DevTools > Application > Cookies)")
	}
	if csrfToken == "" {
		return fmt.Errorf("INSTA_CSRF_TOKEN env not set")
	}
	if dsUserID == "" {
		return fmt.Errorf("INSTA_DS_USER_ID env not set")
	}

	cfg := insta.DefaultConfig()
	sess := &insta.Session{
		Username: username,
		UserID:   dsUserID,
		Cookies: map[string]string{
			"sessionid":  sessionID,
			"csrftoken":  csrfToken,
			"ds_user_id": dsUserID,
		},
	}

	client, err := insta.NewClient(cfg)
	if err != nil {
		return err
	}
	if err := client.ApplySession(sess); err != nil {
		return err
	}

	// Test session
	fmt.Println(labelStyle.Render("  Testing session..."))
	authUser, err := client.TestSession(context.Background())
	if err != nil {
		return fmt.Errorf("session invalid: %w", err)
	}

	fmt.Printf("  Authenticated as: %s\n", infoStyle.Render("@"+authUser))

	// Save session
	sessionPath := cfg.SessionPath(username)
	if err := client.SaveSession(sessionPath); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("  Session imported successfully!"))
	fmt.Printf("  Session saved to: %s\n", labelStyle.Render(sessionPath))
	fmt.Printf("  Use with: %s\n", infoStyle.Render("--session "+username))

	return nil
}

// ── profile ──────────────────────────────────────────────

func newInstaProfile() *cobra.Command {
	var (
		delay   int
		session string
	)

	cmd := &cobra.Command{
		Use:   "profile <username>",
		Short: "Fetch and display user profile",
		Long: `Fetch public profile information for an Instagram user.

Displays username, bio, follower/following counts, post count, and more.
Profile data is saved to $HOME/data/instagram/{username}/profile.json

Examples:
  search insta profile natgeo
  search insta profile nasa --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runInstaProfile(cmd, username, delay, session)
		},
	}

	cmd.Flags().IntVar(&delay, "delay", 3, "Delay between requests (seconds)")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load")
	return cmd
}

func runInstaProfile(cmd *cobra.Command, username string, delay int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Profile"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	cfg.Delay = time.Duration(delay) * time.Second

	client, err := initClient(cmd, cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  Fetching profile for %s\n", infoStyle.Render("@"+username))

	profile, err := client.GetProfile(cmd.Context(), username)
	if err != nil {
		return err
	}

	// Save profile
	if err := client.SaveProfile(profile); err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: could not save profile: %v", err)))
	}

	fmt.Println()
	displayProfile(profile)

	return nil
}

func displayProfile(p *insta.Profile) {
	verified := ""
	if p.IsVerified {
		verified = " [verified]"
	}
	private := ""
	if p.IsPrivate {
		private = " [private]"
	}

	fmt.Printf("  %s%s%s\n", titleStyle.Render(p.FullName), verified, private)
	fmt.Printf("  @%s\n", infoStyle.Render(p.Username))
	if p.Biography != "" {
		fmt.Println()
		for _, line := range strings.Split(p.Biography, "\n") {
			fmt.Printf("  %s\n", line)
		}
	}
	fmt.Println()
	fmt.Printf("  Posts:      %s\n", infoStyle.Render(formatLargeNumber(p.PostCount)))
	fmt.Printf("  Followers:  %s\n", infoStyle.Render(formatLargeNumber(p.FollowerCount)))
	fmt.Printf("  Following:  %s\n", infoStyle.Render(formatLargeNumber(p.FollowingCount)))
	if p.CategoryName != "" {
		fmt.Printf("  Category:   %s\n", labelStyle.Render(p.CategoryName))
	}
	if p.ExternalURL != "" {
		fmt.Printf("  Website:    %s\n", urlStyle.Render(p.ExternalURL))
	}
	fmt.Printf("  ID:         %s\n", labelStyle.Render(p.ID))
	fmt.Println()
}

// ── posts ────────────────────────────────────────────────

func newInstaPosts() *cobra.Command {
	var (
		maxPosts int
		delay    int
		session  string
	)

	cmd := &cobra.Command{
		Use:   "posts <username>",
		Short: "Download all posts for a user",
		Long: `Download all public posts for an Instagram user.

Posts are stored in a DuckDB database at $HOME/data/instagram/{username}/posts.duckdb
Without auth: first 12 posts. With --session: full pagination.

Examples:
  search insta posts natgeo
  search insta posts natgeo --max-posts 100 --session myuser
  search insta posts nasa --delay 5`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runInstaPosts(cmd, username, maxPosts, delay, session)
		},
	}

	cmd.Flags().IntVar(&maxPosts, "max-posts", 0, "Max posts to fetch (0=unlimited)")
	cmd.Flags().IntVar(&delay, "delay", 3, "Delay between requests (seconds)")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load")
	return cmd
}

func runInstaPosts(cmd *cobra.Command, username string, maxPosts, delay int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Posts"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	cfg.Delay = time.Duration(delay) * time.Second

	client, err := initClient(cmd, cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  Fetching posts for %s\n", infoStyle.Render("@"+username))
	fmt.Printf("  Data:    %s\n", labelStyle.Render(cfg.UserDir(username)))
	fmt.Println()

	start := time.Now()
	posts, err := client.GetUserPosts(cmd.Context(), username, maxPosts, func(p insta.Progress) {
		if !p.Done {
			fmt.Printf("\r  Fetching posts: %s / %s",
				infoStyle.Render(formatLargeNumber(p.Current)),
				labelStyle.Render(formatLargeNumber(p.Total)))
		}
	})

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d posts)", err, len(posts))))
	}

	if len(posts) == 0 {
		fmt.Println(warningStyle.Render("  No posts found"))
		return nil
	}

	// Store in DuckDB
	db, err := insta.OpenDB(cfg.UserDBPath(username))
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := db.InsertPosts(posts); err != nil {
		return fmt.Errorf("insert posts: %w", err)
	}

	// Also store media URLs
	mediaItems := insta.CollectMediaItems(posts)
	if err := db.InsertMedia(mediaItems); err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: insert media: %v", err)))
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Fetched %d posts in %s",
		len(posts), time.Since(start).Truncate(time.Second))))
	fmt.Printf("  Database: %s\n", labelStyle.Render(cfg.UserDBPath(username)))
	fmt.Printf("  Media URLs: %d\n", len(mediaItems))

	// Show top 5 posts
	if len(posts) > 0 {
		fmt.Println()
		fmt.Println(subtitleStyle.Render("  Top posts by likes:"))
		topN := min(5, len(posts))
		top, _ := db.TopPosts(topN)
		for i, p := range top {
			caption := p.Caption
			if len(caption) > 60 {
				caption = caption[:60] + "..."
			}
			caption = strings.ReplaceAll(caption, "\n", " ")
			fmt.Printf("  %d. %s  %s likes  %s\n",
				i+1,
				infoStyle.Render(p.Shortcode),
				labelStyle.Render(formatLargeNumber(p.LikeCount)),
				caption)
		}
	}

	return nil
}

// ── post ─────────────────────────────────────────────────

func newInstaPost() *cobra.Command {
	var session string

	cmd := &cobra.Command{
		Use:   "post <shortcode>",
		Short: "Fetch a single post by shortcode",
		Long: `Fetch details for a single Instagram post.

The shortcode is the part of the URL after /p/, e.g. for
https://www.instagram.com/p/CxYzAbC/ the shortcode is CxYzAbC

Examples:
  search insta post CxYzAbC`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shortcode := args[0]
			// Extract shortcode from URL if full URL provided
			if strings.Contains(shortcode, "instagram.com") {
				parts := strings.Split(strings.Trim(shortcode, "/"), "/")
				for i, p := range parts {
					if (p == "p" || p == "reel") && i+1 < len(parts) {
						shortcode = parts[i+1]
						break
					}
				}
			}
			return runInstaPost(cmd, shortcode, session)
		},
	}

	cmd.Flags().StringVar(&session, "session", "", "Session username to load")
	return cmd
}

func runInstaPost(cmd *cobra.Command, shortcode, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Post"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	client, err := initClient(cmd, cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  Fetching post %s\n", infoStyle.Render(shortcode))

	post, err := client.GetPost(cmd.Context(), shortcode)
	if err != nil {
		return err
	}

	fmt.Println()
	displayPost(post)

	return nil
}

func displayPost(p *insta.Post) {
	fmt.Printf("  Type:       %s\n", infoStyle.Render(p.TypeName))
	fmt.Printf("  Shortcode:  %s\n", infoStyle.Render(p.Shortcode))
	fmt.Printf("  ID:         %s\n", labelStyle.Render(p.ID))
	if p.OwnerName != "" {
		fmt.Printf("  Owner:      @%s\n", infoStyle.Render(p.OwnerName))
	}
	fmt.Printf("  Date:       %s\n", labelStyle.Render(p.TakenAt.Format("2006-01-02 15:04:05")))
	fmt.Printf("  Size:       %dx%d\n", p.Width, p.Height)
	fmt.Printf("  Likes:      %s\n", infoStyle.Render(formatLargeNumber(p.LikeCount)))
	fmt.Printf("  Comments:   %s\n", infoStyle.Render(formatLargeNumber(p.CommentCount)))
	if p.IsVideo {
		fmt.Printf("  Views:      %s\n", infoStyle.Render(formatLargeNumber(p.ViewCount)))
	}
	if p.LocationName != "" {
		fmt.Printf("  Location:   %s\n", labelStyle.Render(p.LocationName))
	}
	if p.Caption != "" {
		fmt.Println()
		caption := p.Caption
		if len(caption) > 500 {
			caption = caption[:500] + "..."
		}
		for _, line := range strings.Split(caption, "\n") {
			fmt.Printf("  %s\n", line)
		}
	}
	if len(p.Children) > 0 {
		fmt.Printf("\n  Carousel items: %d\n", len(p.Children))
		for i, child := range p.Children {
			typeStr := "image"
			if child.IsVideo {
				typeStr = "video"
			}
			fmt.Printf("    %d. %s (%dx%d)\n", i+1, typeStr, child.Width, child.Height)
		}
	}
	fmt.Println()
}

// ── comments ─────────────────────────────────────────────

func newInstaComments() *cobra.Command {
	var (
		maxComments int
		delay       int
		session     string
	)

	cmd := &cobra.Command{
		Use:   "comments <shortcode>",
		Short: "Download comments for a post (requires auth)",
		Long: `Download all comments for an Instagram post.

Requires authentication. Use --session flag.
Comments are stored in the post owner's DuckDB database.

Examples:
  search insta comments CxYzAbC --session myuser
  search insta comments CxYzAbC --max-comments 100 --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstaComments(cmd, args[0], maxComments, delay, session)
		},
	}

	cmd.Flags().IntVar(&maxComments, "max-comments", 0, "Max comments to fetch (0=unlimited)")
	cmd.Flags().IntVar(&delay, "delay", 3, "Delay between requests (seconds)")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	return cmd
}

func runInstaComments(cmd *cobra.Command, shortcode string, maxComments, delay int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Comments"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	cfg.Delay = time.Duration(delay) * time.Second

	client, err := initClient(cmd, cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  Fetching comments for post %s\n", infoStyle.Render(shortcode))
	fmt.Println()

	start := time.Now()
	comments, err := client.GetComments(cmd.Context(), shortcode, maxComments, func(p insta.Progress) {
		if !p.Done {
			fmt.Printf("\r  Comments: %s / %s",
				infoStyle.Render(formatLargeNumber(p.Current)),
				labelStyle.Render(formatLargeNumber(p.Total)))
		}
	})

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d comments)", err, len(comments))))
	}

	if len(comments) == 0 {
		fmt.Println(warningStyle.Render("  No comments found"))
		return nil
	}

	// Store in a generic comments database
	dbPath := cfg.UserDBPath("_comments")
	db, err := insta.OpenDB(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := db.InsertComments(comments); err != nil {
		return fmt.Errorf("insert comments: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Fetched %d comments in %s",
		len(comments), time.Since(start).Truncate(time.Second))))
	fmt.Printf("  Database: %s\n", labelStyle.Render(dbPath))

	// Show sample comments
	fmt.Println()
	fmt.Println(subtitleStyle.Render("  Sample comments:"))
	showN := min(5, len(comments))
	for i := range showN {
		c := comments[i]
		text := strings.ReplaceAll(c.Text, "\n", " ")
		if len(text) > 80 {
			text = text[:80] + "..."
		}
		fmt.Printf("  @%-20s %s\n", infoStyle.Render(c.AuthorName), text)
	}

	return nil
}

// ── search ───────────────────────────────────────────────

func newInstaSearch() *cobra.Command {
	var (
		count   int
		session string
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search users, hashtags, places",
		Long: `Search Instagram for users, hashtags, and places.

May require authentication. Use --session for better results.

Examples:
  search insta search "landscape photography"
  search insta search golang --count 20 --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstaSearch(cmd, args[0], count, session)
		},
	}

	cmd.Flags().IntVar(&count, "count", 50, "Number of results")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load")
	return cmd
}

func runInstaSearch(cmd *cobra.Command, query string, count int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Search"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	client, err := initClient(cmd, cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  Searching for %s\n", infoStyle.Render(query))
	fmt.Println()

	result, err := client.Search(cmd.Context(), query, count)
	if err != nil {
		return err
	}

	// Display users
	if len(result.Users) > 0 {
		fmt.Println(titleStyle.Render("  Users"))
		for _, u := range result.Users {
			verified := ""
			if u.IsVerified {
				verified = " [verified]"
			}
			private := ""
			if u.IsPrivate {
				private = " [private]"
			}
			followers := ""
			if u.Followers > 0 {
				followers = fmt.Sprintf("  (%s followers)", formatLargeNumber(u.Followers))
			}
			fmt.Printf("  @%-20s %s%s%s%s\n",
				infoStyle.Render(u.Username),
				u.FullName,
				verified, private, followers)
		}
		fmt.Println()
	}

	// Display hashtags
	if len(result.Hashtags) > 0 {
		fmt.Println(titleStyle.Render("  Hashtags"))
		for _, h := range result.Hashtags {
			fmt.Printf("  #%-20s %s posts\n",
				infoStyle.Render(h.Name),
				labelStyle.Render(formatLargeNumber(h.MediaCount)))
		}
		fmt.Println()
	}

	// Display places
	if len(result.Places) > 0 {
		fmt.Println(titleStyle.Render("  Places"))
		for _, p := range result.Places {
			addr := p.Address
			if p.City != "" {
				if addr != "" {
					addr += ", "
				}
				addr += p.City
			}
			fmt.Printf("  %-25s %s\n",
				infoStyle.Render(p.Title),
				labelStyle.Render(addr))
		}
		fmt.Println()
	}

	total := len(result.Users) + len(result.Hashtags) + len(result.Places)
	if total == 0 {
		fmt.Println(warningStyle.Render("  No results found"))
	} else {
		fmt.Printf("  Total: %d users, %d hashtags, %d places\n",
			len(result.Users), len(result.Hashtags), len(result.Places))
	}

	return nil
}

// ── hashtag ──────────────────────────────────────────────

func newInstaHashtag() *cobra.Command {
	var (
		maxPosts int
		delay    int
		session  string
	)

	cmd := &cobra.Command{
		Use:   "hashtag <tag>",
		Short: "Download posts for a hashtag (requires auth)",
		Long: `Download posts for an Instagram hashtag.

Requires authentication. Use --session flag.
Posts are stored in a DuckDB database at $HOME/data/instagram/hashtag/{tag}/posts.duckdb

Examples:
  search insta hashtag sunset --session myuser
  search insta hashtag sunset --max-posts 50 --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tag := strings.TrimPrefix(args[0], "#")
			return runInstaHashtag(cmd, tag, maxPosts, delay, session)
		},
	}

	cmd.Flags().IntVar(&maxPosts, "max-posts", 0, "Max posts to fetch (0=unlimited)")
	cmd.Flags().IntVar(&delay, "delay", 3, "Delay between requests (seconds)")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	return cmd
}

func runInstaHashtag(cmd *cobra.Command, tag string, maxPosts, delay int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Hashtag"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	cfg.Delay = time.Duration(delay) * time.Second

	client, err := initClient(cmd, cfg, session)
	if err != nil {
		return err
	}

	maxStr := "all"
	if maxPosts > 0 {
		maxStr = fmt.Sprintf("%d", maxPosts)
	}
	fmt.Printf("  Hashtag: %s  |  Fetching: %s\n", infoStyle.Render("#"+tag), maxStr)
	fmt.Printf("  Data:    %s\n", labelStyle.Render(cfg.HashtagDir(tag)))
	fmt.Println()

	start := time.Now()
	posts, err := client.GetHashtagPosts(cmd.Context(), tag, maxPosts, func(p insta.Progress) {
		if !p.Done {
			fmt.Printf("\r  Fetching posts: %s / %s",
				infoStyle.Render(formatLargeNumber(p.Current)),
				labelStyle.Render(formatLargeNumber(p.Total)))
		}
	})

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d posts)", err, len(posts))))
	}

	if len(posts) == 0 {
		fmt.Println(warningStyle.Render("  No posts found"))
		return nil
	}

	db, err := insta.OpenDB(cfg.HashtagDBPath(tag))
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := db.InsertPosts(posts); err != nil {
		return fmt.Errorf("insert posts: %w", err)
	}

	mediaItems := insta.CollectMediaItems(posts)
	if err := db.InsertMedia(mediaItems); err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: insert media: %v", err)))
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Fetched %d posts in %s",
		len(posts), time.Since(start).Truncate(time.Second))))
	fmt.Printf("  Database: %s\n", labelStyle.Render(cfg.HashtagDBPath(tag)))

	return nil
}

// ── location ─────────────────────────────────────────────

func newInstaLocation() *cobra.Command {
	var (
		maxPosts int
		delay    int
		session  string
	)

	cmd := &cobra.Command{
		Use:   "location <id>",
		Short: "Download posts for a location (requires auth)",
		Long: `Download posts for an Instagram location.

Requires authentication. Use --session flag.
Location IDs can be found via 'search insta search' or from Instagram URLs.

Examples:
  search insta location 213385402 --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstaLocation(cmd, args[0], maxPosts, delay, session)
		},
	}

	cmd.Flags().IntVar(&maxPosts, "max-posts", 0, "Max posts to fetch (0=unlimited)")
	cmd.Flags().IntVar(&delay, "delay", 3, "Delay between requests (seconds)")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	return cmd
}

func runInstaLocation(cmd *cobra.Command, locationID string, maxPosts, delay int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Location"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	cfg.Delay = time.Duration(delay) * time.Second

	client, err := initClient(cmd, cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  Location: %s\n", infoStyle.Render(locationID))
	fmt.Printf("  Data:     %s\n", labelStyle.Render(cfg.LocationDir(locationID)))
	fmt.Println()

	start := time.Now()
	posts, err := client.GetLocationPosts(cmd.Context(), locationID, maxPosts, func(p insta.Progress) {
		if !p.Done {
			fmt.Printf("\r  Fetching posts: %s / %s",
				infoStyle.Render(formatLargeNumber(p.Current)),
				labelStyle.Render(formatLargeNumber(p.Total)))
		}
	})

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d posts)", err, len(posts))))
	}

	if len(posts) == 0 {
		fmt.Println(warningStyle.Render("  No posts found"))
		return nil
	}

	db, err := insta.OpenDB(cfg.LocationDBPath(locationID))
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := db.InsertPosts(posts); err != nil {
		return fmt.Errorf("insert posts: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Fetched %d posts in %s",
		len(posts), time.Since(start).Truncate(time.Second))))
	fmt.Printf("  Database: %s\n", labelStyle.Render(cfg.LocationDBPath(locationID)))

	return nil
}

// ── download ─────────────────────────────────────────────

func newInstaDownload() *cobra.Command {
	var (
		workers  int
		noImages bool
		noVideos bool
		maxPosts int
		delay    int
		session  string
	)

	cmd := &cobra.Command{
		Use:   "download <username>",
		Short: "Download media files (images/videos)",
		Long: `Download media files for a user's posts.

First fetches all posts (or uses cached data), then downloads images and videos.
Media is saved to $HOME/data/instagram/{username}/media/
Use --session for full pagination (all posts).

Examples:
  search insta download natgeo
  search insta download natgeo --workers 4 --session myuser
  search insta download natgeo --no-videos
  search insta download natgeo --max-posts 50`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runInstaDownload(cmd, username, workers, !noImages, !noVideos, maxPosts, delay, session)
		},
	}

	cmd.Flags().IntVar(&workers, "workers", 8, "Concurrent download workers")
	cmd.Flags().BoolVar(&noImages, "no-images", false, "Skip image downloads")
	cmd.Flags().BoolVar(&noVideos, "no-videos", false, "Skip video downloads")
	cmd.Flags().IntVar(&maxPosts, "max-posts", 0, "Max posts to fetch (0=unlimited)")
	cmd.Flags().IntVar(&delay, "delay", 3, "Delay between API requests (seconds)")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load")
	return cmd
}

func runInstaDownload(cmd *cobra.Command, username string, workers int, images, videos bool, maxPosts, delay int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Download"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	cfg.Delay = time.Duration(delay) * time.Second
	cfg.Workers = workers

	client, err := initClient(cmd, cfg, session)
	if err != nil {
		return err
	}

	// Fetch posts (also fetches profile internally)
	fmt.Printf("  Fetching posts for %s\n", infoStyle.Render("@"+username))
	posts, err := client.GetUserPosts(cmd.Context(), username, maxPosts, func(p insta.Progress) {
		if !p.Done {
			fmt.Printf("\r  Posts: %s / %s",
				infoStyle.Render(formatLargeNumber(p.Current)),
				labelStyle.Render(formatLargeNumber(p.Total)))
		}
	})
	fmt.Println()

	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d posts)", err, len(posts))))
	}

	if len(posts) == 0 {
		fmt.Println(warningStyle.Render("  No posts found"))
		return nil
	}

	// Store posts
	db, err := insta.OpenDB(cfg.UserDBPath(username))
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := db.InsertPosts(posts); err != nil {
		return fmt.Errorf("insert posts: %w", err)
	}

	// Collect media items
	mediaItems := insta.CollectMediaItems(posts)
	_ = db.InsertMedia(mediaItems)

	imageCount := 0
	videoCount := 0
	for _, m := range mediaItems {
		if m.Type == "image" {
			imageCount++
		} else {
			videoCount++
		}
	}

	fmt.Println()
	fmt.Printf("  Media: %d images, %d videos\n", imageCount, videoCount)
	fmt.Printf("  Dir:   %s\n", labelStyle.Render(cfg.UserMediaDir(username)))
	fmt.Println()

	// Download media
	start := time.Now()
	err = insta.DownloadMedia(cmd.Context(), mediaItems, cfg.UserMediaDir(username), workers, images, videos, func(p insta.DownloadProgress) {
		if !p.Done {
			fmt.Printf("\r  Downloading: %d/%d  |  Skipped: %d  |  Failed: %d  |  %s",
				p.Downloaded, p.Total, p.Skipped, p.Failed,
				formatBytes(p.Bytes))
		}
	})

	fmt.Println()
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Download complete in %s",
		time.Since(start).Truncate(time.Second))))

	return nil
}

// ── info ─────────────────────────────────────────────────

func newInstaInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <username>",
		Short: "Show stored data statistics",
		Long: `Show statistics for previously scraped Instagram data.

Examples:
  search insta info natgeo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runInstaInfo(username)
		},
	}
	return cmd
}

func runInstaInfo(username string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Info"))
	fmt.Println()

	cfg := insta.DefaultConfig()

	// Check if data exists
	dir := cfg.UserDir(username)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("no data found for @%s (expected at %s)", username, dir)
	}

	// Load profile
	profile, err := insta.LoadProfile(cfg, username)
	if err == nil {
		displayProfile(profile)
	} else {
		fmt.Printf("  @%s\n\n", infoStyle.Render(username))
	}

	// Check database
	dbPath := cfg.UserDBPath(username)
	if _, err := os.Stat(dbPath); err == nil {
		db, err := insta.OpenDB(dbPath)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		defer db.Close()

		stats, err := db.GetStats()
		if err != nil {
			return fmt.Errorf("get stats: %w", err)
		}

		fmt.Println(titleStyle.Render("  Database"))
		fmt.Printf("  Posts:      %s\n", infoStyle.Render(formatLargeNumber(stats.Posts)))
		fmt.Printf("  Comments:   %s\n", infoStyle.Render(formatLargeNumber(stats.Comments)))
		fmt.Printf("  Media URLs: %s\n", infoStyle.Render(formatLargeNumber(stats.Media)))
		fmt.Printf("  DB Size:    %s\n", labelStyle.Render(formatBytes(stats.DBSize)))
		fmt.Printf("  Path:       %s\n", labelStyle.Render(dbPath))
		fmt.Println()

		// Show top posts
		top, _ := db.TopPosts(5)
		if len(top) > 0 {
			fmt.Println(titleStyle.Render("  Top Posts"))
			for i, p := range top {
				caption := p.Caption
				if len(caption) > 50 {
					caption = caption[:50] + "..."
				}
				caption = strings.ReplaceAll(caption, "\n", " ")
				fmt.Printf("  %d. %s  %s likes  %s  %s\n",
					i+1,
					infoStyle.Render(p.Shortcode),
					labelStyle.Render(formatLargeNumber(p.LikeCount)),
					labelStyle.Render(p.TakenAt.Format("2006-01-02")),
					caption)
			}
			fmt.Println()
		}
	} else {
		fmt.Println(warningStyle.Render("  No posts database found"))
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

// ── stories ──────────────────────────────────────────────

func newInstaStories() *cobra.Command {
	var session string

	cmd := &cobra.Command{
		Use:   "stories <username>",
		Short: "Fetch user stories (requires auth)",
		Long: `Fetch current stories for an Instagram user.

Requires authentication and uses the iPhone API for higher quality.

Examples:
  search insta stories natgeo --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runInstaStories(cmd, username, session)
		},
	}

	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	return cmd
}

func runInstaStories(cmd *cobra.Command, username, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Stories"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	client, err := initClient(cmd, cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  Fetching stories for %s\n", infoStyle.Render("@"+username))

	story, err := client.GetStoriesByUsername(cmd.Context(), username)
	if err != nil {
		return err
	}

	if len(story.Items) == 0 {
		fmt.Println(warningStyle.Render("  No active stories"))
		return nil
	}

	fmt.Println()
	fmt.Printf("  Stories: %d items\n\n", len(story.Items))
	for i, item := range story.Items {
		typeStr := "image"
		if item.IsVideo {
			typeStr = "video"
		}
		fmt.Printf("  %d. %s  %s (%dx%d)  %s\n",
			i+1, infoStyle.Render(item.ID), typeStr,
			item.Width, item.Height,
			labelStyle.Render(item.TakenAt.Format("2006-01-02 15:04")))
	}
	fmt.Println()

	return nil
}

// ── highlights ───────────────────────────────────────────

func newInstaHighlights() *cobra.Command {
	var session string

	cmd := &cobra.Command{
		Use:   "highlights <username>",
		Short: "Fetch user highlights (requires auth)",
		Long: `Fetch highlight reels for an Instagram user.

Requires authentication.

Examples:
  search insta highlights natgeo --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runInstaHighlights(cmd, username, session)
		},
	}

	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	return cmd
}

func runInstaHighlights(cmd *cobra.Command, username, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Highlights"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	client, err := initClient(cmd, cfg, session)
	if err != nil {
		return err
	}

	// Get user ID
	profile, err := client.GetProfile(cmd.Context(), username)
	if err != nil {
		return err
	}

	fmt.Printf("  Fetching highlights for %s\n", infoStyle.Render("@"+username))

	highlights, err := client.GetHighlights(cmd.Context(), profile.ID)
	if err != nil {
		return err
	}

	if len(highlights) == 0 {
		fmt.Println(warningStyle.Render("  No highlights found"))
		return nil
	}

	fmt.Println()
	fmt.Printf("  Highlights: %d\n\n", len(highlights))
	for i, h := range highlights {
		fmt.Printf("  %d. %s  (%d items)  ID: %s\n",
			i+1, infoStyle.Render(h.Title), h.ItemCount, labelStyle.Render(h.ID))
	}
	fmt.Println()

	return nil
}

// ── reels ────────────────────────────────────────────────

func newInstaReels() *cobra.Command {
	var (
		maxReels int
		delay    int
		session  string
	)

	cmd := &cobra.Command{
		Use:   "reels <username>",
		Short: "Fetch user reels (requires auth)",
		Long: `Fetch reels for an Instagram user.

Requires authentication. Reels are short-form video posts.

Examples:
  search insta reels natgeo --session myuser
  search insta reels natgeo --max-posts 50 --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runInstaReels(cmd, username, maxReels, delay, session)
		},
	}

	cmd.Flags().IntVar(&maxReels, "max-posts", 0, "Max reels to fetch (0=unlimited)")
	cmd.Flags().IntVar(&delay, "delay", 3, "Delay between requests (seconds)")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	return cmd
}

func runInstaReels(cmd *cobra.Command, username string, maxReels, delay int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Reels"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	cfg.Delay = time.Duration(delay) * time.Second

	client, err := initClient(cmd, cfg, session)
	if err != nil {
		return err
	}

	// Get user ID
	profile, err := client.GetProfile(cmd.Context(), username)
	if err != nil {
		return err
	}

	fmt.Printf("  Fetching reels for %s\n", infoStyle.Render("@"+username))
	fmt.Println()

	start := time.Now()
	reels, err := client.GetReels(cmd.Context(), profile.ID, maxReels, func(p insta.Progress) {
		if !p.Done {
			fmt.Printf("\r  Reels: %s",
				infoStyle.Render(formatLargeNumber(p.Current)))
		}
	})

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d reels)", err, len(reels))))
	}

	if len(reels) == 0 {
		fmt.Println(warningStyle.Render("  No reels found"))
		return nil
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Fetched %d reels in %s",
		len(reels), time.Since(start).Truncate(time.Second))))

	// Show top 5 reels
	fmt.Println()
	fmt.Println(subtitleStyle.Render("  Top reels by views:"))
	showN := min(5, len(reels))
	for i := range showN {
		r := reels[i]
		caption := r.Caption
		if len(caption) > 50 {
			caption = caption[:50] + "..."
		}
		caption = strings.ReplaceAll(caption, "\n", " ")
		fmt.Printf("  %d. %s  %s views  %s\n",
			i+1, infoStyle.Render(r.Shortcode),
			labelStyle.Render(formatLargeNumber(r.ViewCount)),
			caption)
	}

	return nil
}

// ── followers ────────────────────────────────────────────

func newInstaFollowers() *cobra.Command {
	var (
		maxUsers int
		delay    int
		session  string
	)

	cmd := &cobra.Command{
		Use:   "followers <username>",
		Short: "Fetch user followers (requires auth)",
		Long: `Fetch the follower list for an Instagram user.

Requires authentication.

Examples:
  search insta followers natgeo --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runInstaFollowList(cmd, username, "followers", maxUsers, delay, session)
		},
	}

	cmd.Flags().IntVar(&maxUsers, "max-users", 0, "Max users to fetch (0=unlimited)")
	cmd.Flags().IntVar(&delay, "delay", 3, "Delay between requests (seconds)")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	return cmd
}

func newInstaFollowing() *cobra.Command {
	var (
		maxUsers int
		delay    int
		session  string
	)

	cmd := &cobra.Command{
		Use:   "following <username>",
		Short: "Fetch user following (requires auth)",
		Long: `Fetch the following list for an Instagram user.

Requires authentication.

Examples:
  search insta following natgeo --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runInstaFollowList(cmd, username, "following", maxUsers, delay, session)
		},
	}

	cmd.Flags().IntVar(&maxUsers, "max-users", 0, "Max users to fetch (0=unlimited)")
	cmd.Flags().IntVar(&delay, "delay", 3, "Delay between requests (seconds)")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	return cmd
}

func runInstaFollowList(cmd *cobra.Command, username, listType string, maxUsers, delay int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram " + capitalizeFirst(listType)))
	fmt.Println()

	cfg := insta.DefaultConfig()
	cfg.Delay = time.Duration(delay) * time.Second

	client, err := initClient(cmd, cfg, session)
	if err != nil {
		return err
	}

	profile, err := client.GetProfile(cmd.Context(), username)
	if err != nil {
		return err
	}

	fmt.Printf("  Fetching %s for %s\n", listType, infoStyle.Render("@"+username))
	fmt.Println()

	start := time.Now()
	var users []insta.FollowUser
	progressCb := func(p insta.Progress) {
		if !p.Done {
			fmt.Printf("\r  %s: %s / %s",
				capitalizeFirst(listType),
				infoStyle.Render(formatLargeNumber(p.Current)),
				labelStyle.Render(formatLargeNumber(p.Total)))
		}
	}

	if listType == "followers" {
		users, err = client.GetFollowers(cmd.Context(), profile.ID, maxUsers, progressCb)
	} else {
		users, err = client.GetFollowing(cmd.Context(), profile.ID, maxUsers, progressCb)
	}

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d users)", err, len(users))))
	}

	if len(users) == 0 {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  No %s found", listType)))
		return nil
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Fetched %d %s in %s",
		len(users), listType, time.Since(start).Truncate(time.Second))))

	// Show first 10
	fmt.Println()
	showN := min(10, len(users))
	for i := range showN {
		u := users[i]
		verified := ""
		if u.IsVerified {
			verified = " [verified]"
		}
		fmt.Printf("  @%-20s %s%s\n",
			infoStyle.Render(u.Username), u.FullName, verified)
	}
	if len(users) > showN {
		fmt.Printf("  ... and %d more\n", len(users)-showN)
	}

	return nil
}

// ── tagged ───────────────────────────────────────────────

func newInstaTagged() *cobra.Command {
	var (
		maxPosts int
		delay    int
		session  string
	)

	cmd := &cobra.Command{
		Use:   "tagged <username>",
		Short: "Fetch posts user is tagged in (requires auth)",
		Long: `Fetch posts where a user is tagged.

Requires authentication.

Examples:
  search insta tagged natgeo --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runInstaTagged(cmd, username, maxPosts, delay, session)
		},
	}

	cmd.Flags().IntVar(&maxPosts, "max-posts", 0, "Max posts to fetch (0=unlimited)")
	cmd.Flags().IntVar(&delay, "delay", 3, "Delay between requests (seconds)")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	return cmd
}

func runInstaTagged(cmd *cobra.Command, username string, maxPosts, delay int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Tagged Posts"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	cfg.Delay = time.Duration(delay) * time.Second

	client, err := initClient(cmd, cfg, session)
	if err != nil {
		return err
	}

	profile, err := client.GetProfile(cmd.Context(), username)
	if err != nil {
		return err
	}

	fmt.Printf("  Fetching tagged posts for %s\n", infoStyle.Render("@"+username))
	fmt.Println()

	start := time.Now()
	posts, err := client.GetTaggedPosts(cmd.Context(), profile.ID, maxPosts, func(p insta.Progress) {
		if !p.Done {
			fmt.Printf("\r  Tagged posts: %s / %s",
				infoStyle.Render(formatLargeNumber(p.Current)),
				labelStyle.Render(formatLargeNumber(p.Total)))
		}
	})

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d posts)", err, len(posts))))
	}

	if len(posts) == 0 {
		fmt.Println(warningStyle.Render("  No tagged posts found"))
		return nil
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Fetched %d tagged posts in %s",
		len(posts), time.Since(start).Truncate(time.Second))))

	return nil
}

// ── saved ────────────────────────────────────────────────

func newInstaSaved() *cobra.Command {
	var (
		maxPosts int
		delay    int
		session  string
	)

	cmd := &cobra.Command{
		Use:   "saved",
		Short: "Fetch saved posts (requires auth, own only)",
		Long: `Fetch the logged-in user's saved posts.

Only works for the authenticated user's own saved posts.
Requires authentication.

Examples:
  search insta saved --session myuser`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstaSaved(cmd, maxPosts, delay, session)
		},
	}

	cmd.Flags().IntVar(&maxPosts, "max-posts", 0, "Max posts to fetch (0=unlimited)")
	cmd.Flags().IntVar(&delay, "delay", 3, "Delay between requests (seconds)")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	return cmd
}

func runInstaSaved(cmd *cobra.Command, maxPosts, delay int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Saved Posts"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	cfg.Delay = time.Duration(delay) * time.Second

	client, err := initClient(cmd, cfg, session)
	if err != nil {
		return err
	}

	fmt.Println("  Fetching your saved posts...")
	fmt.Println()

	start := time.Now()
	posts, err := client.GetSavedPosts(cmd.Context(), maxPosts, func(p insta.Progress) {
		if !p.Done {
			fmt.Printf("\r  Saved posts: %s / %s",
				infoStyle.Render(formatLargeNumber(p.Current)),
				labelStyle.Render(formatLargeNumber(p.Total)))
		}
	})

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d posts)", err, len(posts))))
	}

	if len(posts) == 0 {
		fmt.Println(warningStyle.Render("  No saved posts found"))
		return nil
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Fetched %d saved posts in %s",
		len(posts), time.Since(start).Truncate(time.Second))))

	// Show top 5
	showN := min(5, len(posts))
	fmt.Println()
	for i := range showN {
		p := posts[i]
		caption := p.Caption
		if len(caption) > 50 {
			caption = caption[:50] + "..."
		}
		caption = strings.ReplaceAll(caption, "\n", " ")
		fmt.Printf("  %d. %s  @%s  %s\n",
			i+1, infoStyle.Render(p.Shortcode), p.OwnerName, caption)
	}

	return nil
}

// ── likes ────────────────────────────────────────────────

func newInstaLikes() *cobra.Command {
	var (
		maxUsers int
		delay    int
		session  string
	)

	cmd := &cobra.Command{
		Use:   "likes <shortcode>",
		Short: "Fetch users who liked a post (requires auth)",
		Long: `Fetch the list of users who liked an Instagram post.

Requires authentication.

Examples:
  search insta likes CxYzAbC --session myuser`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shortcode := args[0]
			if strings.Contains(shortcode, "instagram.com") {
				parts := strings.Split(strings.Trim(shortcode, "/"), "/")
				for i, p := range parts {
					if (p == "p" || p == "reel") && i+1 < len(parts) {
						shortcode = parts[i+1]
						break
					}
				}
			}
			return runInstaLikes(cmd, shortcode, maxUsers, delay, session)
		},
	}

	cmd.Flags().IntVar(&maxUsers, "max-users", 0, "Max users to fetch (0=unlimited)")
	cmd.Flags().IntVar(&delay, "delay", 3, "Delay between requests (seconds)")
	cmd.Flags().StringVar(&session, "session", "", "Session username to load (required)")
	return cmd
}

func runInstaLikes(cmd *cobra.Command, shortcode string, maxUsers, delay int, session string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Post Likes"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	cfg.Delay = time.Duration(delay) * time.Second

	client, err := initClient(cmd, cfg, session)
	if err != nil {
		return err
	}

	fmt.Printf("  Fetching likes for post %s\n", infoStyle.Render(shortcode))
	fmt.Println()

	start := time.Now()
	users, err := client.GetPostLikes(cmd.Context(), shortcode, maxUsers, func(p insta.Progress) {
		if !p.Done {
			fmt.Printf("\r  Likes: %s / %s",
				infoStyle.Render(formatLargeNumber(p.Current)),
				labelStyle.Render(formatLargeNumber(p.Total)))
		}
	})

	fmt.Println()
	if err != nil {
		fmt.Println(warningStyle.Render(fmt.Sprintf("  Warning: %v (got %d users)", err, len(users))))
	}

	if len(users) == 0 {
		fmt.Println(warningStyle.Render("  No likes found"))
		return nil
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("  Fetched %d likes in %s",
		len(users), time.Since(start).Truncate(time.Second))))

	// Show first 10
	fmt.Println()
	showN := min(10, len(users))
	for i := range showN {
		u := users[i]
		verified := ""
		if u.IsVerified {
			verified = " [verified]"
		}
		fmt.Printf("  @%-20s %s%s\n",
			infoStyle.Render(u.Username), u.FullName, verified)
	}
	if len(users) > showN {
		fmt.Printf("  ... and %d more\n", len(users)-showN)
	}

	return nil
}
