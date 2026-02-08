package cli

import (
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
		Short: "Instagram search and scrape (public data)",
		Long: `Search and scrape public Instagram data using the web GraphQL API.

No authentication required for public profiles, posts, hashtags, and locations.
Rate-limited to ~200 requests/hour. Use --delay to adjust request spacing.

Data is stored at $HOME/data/instagram/

Subcommands:
  profile    Fetch and display user profile info
  posts      Download all posts for a user
  post       Fetch a single post by shortcode
  comments   Download comments for a post
  search     Search users, hashtags, places
  hashtag    Download posts for a hashtag
  location   Download posts for a location
  download   Download media files (images/videos)
  info       Show stored data statistics

Examples:
  search insta profile natgeo
  search insta posts natgeo --max-posts 100
  search insta post CxYzAbC
  search insta comments CxYzAbC
  search insta search "landscape photography"
  search insta hashtag sunset --max-posts 50
  search insta download natgeo --workers 4`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newInstaProfile())
	cmd.AddCommand(newInstaPosts())
	cmd.AddCommand(newInstaPost())
	cmd.AddCommand(newInstaComments())
	cmd.AddCommand(newInstaSearch())
	cmd.AddCommand(newInstaHashtag())
	cmd.AddCommand(newInstaLocation())
	cmd.AddCommand(newInstaDownload())
	cmd.AddCommand(newInstaInfo())

	return cmd
}

// ── profile ──────────────────────────────────────────────

func newInstaProfile() *cobra.Command {
	var delay int

	cmd := &cobra.Command{
		Use:   "profile <username>",
		Short: "Fetch and display user profile",
		Long: `Fetch public profile information for an Instagram user.

Displays username, bio, follower/following counts, post count, and more.
Profile data is saved to $HOME/data/instagram/{username}/profile.json

Examples:
  search insta profile natgeo
  search insta profile nasa`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runInstaProfile(cmd, username, delay)
		},
	}

	cmd.Flags().IntVar(&delay, "delay", 3, "Delay between requests (seconds)")
	return cmd
}

func runInstaProfile(cmd *cobra.Command, username string, delay int) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Profile"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	cfg.Delay = time.Duration(delay) * time.Second

	client, err := insta.NewClient(cfg)
	if err != nil {
		return err
	}

	fmt.Println(labelStyle.Render("  Initializing session..."))
	if err := client.Init(cmd.Context()); err != nil {
		return fmt.Errorf("init session: %w", err)
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
	)

	cmd := &cobra.Command{
		Use:   "posts <username>",
		Short: "Download all posts for a user",
		Long: `Download all public posts for an Instagram user.

Posts are stored in a DuckDB database at $HOME/data/instagram/{username}/posts.duckdb
Includes image/video URLs, captions, like/comment counts, timestamps, and locations.

Examples:
  search insta posts natgeo
  search insta posts natgeo --max-posts 100
  search insta posts nasa --delay 5`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runInstaPosts(cmd, username, maxPosts, delay)
		},
	}

	cmd.Flags().IntVar(&maxPosts, "max-posts", 0, "Max posts to fetch (0=unlimited)")
	cmd.Flags().IntVar(&delay, "delay", 3, "Delay between requests (seconds)")
	return cmd
}

func runInstaPosts(cmd *cobra.Command, username string, maxPosts, delay int) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Posts"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	cfg.Delay = time.Duration(delay) * time.Second

	client, err := insta.NewClient(cfg)
	if err != nil {
		return err
	}

	fmt.Println(labelStyle.Render("  Initializing session..."))
	if err := client.Init(cmd.Context()); err != nil {
		return fmt.Errorf("init session: %w", err)
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
			return runInstaPost(cmd, shortcode)
		},
	}

	return cmd
}

func runInstaPost(cmd *cobra.Command, shortcode string) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Post"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	client, err := insta.NewClient(cfg)
	if err != nil {
		return err
	}

	fmt.Println(labelStyle.Render("  Initializing session..."))
	if err := client.Init(cmd.Context()); err != nil {
		return fmt.Errorf("init session: %w", err)
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
	)

	cmd := &cobra.Command{
		Use:   "comments <shortcode>",
		Short: "Download comments for a post",
		Long: `Download all comments for an Instagram post.

Comments are stored in the post owner's DuckDB database.

Examples:
  search insta comments CxYzAbC
  search insta comments CxYzAbC --max-comments 100`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstaComments(cmd, args[0], maxComments, delay)
		},
	}

	cmd.Flags().IntVar(&maxComments, "max-comments", 0, "Max comments to fetch (0=unlimited)")
	cmd.Flags().IntVar(&delay, "delay", 3, "Delay between requests (seconds)")
	return cmd
}

func runInstaComments(cmd *cobra.Command, shortcode string, maxComments, delay int) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Comments"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	cfg.Delay = time.Duration(delay) * time.Second

	client, err := insta.NewClient(cfg)
	if err != nil {
		return err
	}

	fmt.Println(labelStyle.Render("  Initializing session..."))
	if err := client.Init(cmd.Context()); err != nil {
		return fmt.Errorf("init session: %w", err)
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
	var count int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search users, hashtags, places",
		Long: `Search Instagram for users, hashtags, and places.

Examples:
  search insta search "landscape photography"
  search insta search golang --count 20`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstaSearch(cmd, args[0], count)
		},
	}

	cmd.Flags().IntVar(&count, "count", 50, "Number of results")
	return cmd
}

func runInstaSearch(cmd *cobra.Command, query string, count int) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Search"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	client, err := insta.NewClient(cfg)
	if err != nil {
		return err
	}

	fmt.Println(labelStyle.Render("  Initializing session..."))
	if err := client.Init(cmd.Context()); err != nil {
		return fmt.Errorf("init session: %w", err)
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
	)

	cmd := &cobra.Command{
		Use:   "hashtag <tag>",
		Short: "Download posts for a hashtag",
		Long: `Download public posts for an Instagram hashtag.

Posts are stored in a DuckDB database at $HOME/data/instagram/hashtag/{tag}/posts.duckdb

Examples:
  search insta hashtag sunset
  search insta hashtag sunset --max-posts 50`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tag := strings.TrimPrefix(args[0], "#")
			return runInstaHashtag(cmd, tag, maxPosts, delay)
		},
	}

	cmd.Flags().IntVar(&maxPosts, "max-posts", 0, "Max posts to fetch (0=unlimited)")
	cmd.Flags().IntVar(&delay, "delay", 3, "Delay between requests (seconds)")
	return cmd
}

func runInstaHashtag(cmd *cobra.Command, tag string, maxPosts, delay int) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Hashtag"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	cfg.Delay = time.Duration(delay) * time.Second

	client, err := insta.NewClient(cfg)
	if err != nil {
		return err
	}

	fmt.Println(labelStyle.Render("  Initializing session..."))
	if err := client.Init(cmd.Context()); err != nil {
		return fmt.Errorf("init session: %w", err)
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
	)

	cmd := &cobra.Command{
		Use:   "location <id>",
		Short: "Download posts for a location",
		Long: `Download public posts for an Instagram location.

Location IDs can be found via 'search insta search' or from Instagram URLs.

Examples:
  search insta location 213385402`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstaLocation(cmd, args[0], maxPosts, delay)
		},
	}

	cmd.Flags().IntVar(&maxPosts, "max-posts", 0, "Max posts to fetch (0=unlimited)")
	cmd.Flags().IntVar(&delay, "delay", 3, "Delay between requests (seconds)")
	return cmd
}

func runInstaLocation(cmd *cobra.Command, locationID string, maxPosts, delay int) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Location"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	cfg.Delay = time.Duration(delay) * time.Second

	client, err := insta.NewClient(cfg)
	if err != nil {
		return err
	}

	fmt.Println(labelStyle.Render("  Initializing session..."))
	if err := client.Init(cmd.Context()); err != nil {
		return fmt.Errorf("init session: %w", err)
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
	)

	cmd := &cobra.Command{
		Use:   "download <username>",
		Short: "Download media files (images/videos)",
		Long: `Download media files for a user's posts.

First fetches all posts (or uses cached data), then downloads images and videos.
Media is saved to $HOME/data/instagram/{username}/media/

Examples:
  search insta download natgeo
  search insta download natgeo --workers 4
  search insta download natgeo --no-videos
  search insta download natgeo --max-posts 50`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := strings.TrimPrefix(args[0], "@")
			return runInstaDownload(cmd, username, workers, !noImages, !noVideos, maxPosts, delay)
		},
	}

	cmd.Flags().IntVar(&workers, "workers", 8, "Concurrent download workers")
	cmd.Flags().BoolVar(&noImages, "no-images", false, "Skip image downloads")
	cmd.Flags().BoolVar(&noVideos, "no-videos", false, "Skip video downloads")
	cmd.Flags().IntVar(&maxPosts, "max-posts", 0, "Max posts to fetch (0=unlimited)")
	cmd.Flags().IntVar(&delay, "delay", 3, "Delay between API requests (seconds)")
	return cmd
}

func runInstaDownload(cmd *cobra.Command, username string, workers int, images, videos bool, maxPosts, delay int) error {
	fmt.Println(Banner())
	fmt.Println(subtitleStyle.Render("Instagram Download"))
	fmt.Println()

	cfg := insta.DefaultConfig()
	cfg.Delay = time.Duration(delay) * time.Second
	cfg.Workers = workers

	client, err := insta.NewClient(cfg)
	if err != nil {
		return err
	}

	fmt.Println(labelStyle.Render("  Initializing session..."))
	if err := client.Init(cmd.Context()); err != nil {
		return fmt.Errorf("init session: %w", err)
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
