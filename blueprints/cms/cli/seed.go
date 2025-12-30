package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/mizu/blueprints/cms/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/cms/feature/options"
	"github.com/go-mizu/mizu/blueprints/cms/feature/posts"
	"github.com/go-mizu/mizu/blueprints/cms/feature/terms"
	"github.com/go-mizu/mizu/blueprints/cms/store/duckdb"
)

// NewSeed creates the seed command.
func NewSeed() *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Seed sample data",
		Long: `Populates the database with sample content for development and testing.

This creates:
  - Sample users (admin, editor, author)
  - Sample posts and pages
  - Categories and tags
  - Sample comments`,
		RunE: runSeed,
	}
}

func runSeed(cmd *cobra.Command, args []string) error {
	ui := NewUI()

	ui.Header(iconDatabase, "Seeding CMS Data")
	ui.Blank()

	// Open database
	start := time.Now()
	ui.StartSpinner("Opening database...")

	store, err := duckdb.Open(dataDir)
	if err != nil {
		ui.StopSpinnerError("Failed to open database")
		return err
	}
	defer store.Close()

	ui.StopSpinner("Database ready", time.Since(start))

	ctx := context.Background()

	// Initialize services
	accountsSvc := accounts.NewService(store.Users(), store.Usermeta(), store.Sessions())
	optionsSvc := options.NewService(store.Options())
	termsSvc := terms.NewService(store.Terms(), store.TermTaxonomy(), store.Termmeta())
	postsSvc := posts.NewService(store.Posts(), store.Postmeta(), store.TermRelationships(), store.TermTaxonomy(), store.Options())

	// Initialize options
	ui.StartSpinner("Initializing options...")
	start = time.Now()
	_ = optionsSvc.InitDefaults(ctx, "http://localhost:8080", "My CMS Site", "admin@example.com")
	ui.StopSpinner("Options initialized", time.Since(start))

	// Create users
	ui.StartSpinner("Creating users...")
	start = time.Now()

	users := []struct {
		username string
		email    string
		password string
		role     string
	}{
		{"admin", "admin@example.com", "admin123", "administrator"},
		{"editor", "editor@example.com", "editor123", "editor"},
		{"author", "author@example.com", "author123", "author"},
		{"subscriber", "subscriber@example.com", "sub123", "subscriber"},
	}

	userIDs := make(map[string]string)
	for _, u := range users {
		existing, _ := accountsSvc.GetByLogin(ctx, u.username)
		if existing != nil {
			userIDs[u.username] = existing.ID
			continue
		}
		user, err := accountsSvc.Create(ctx, accounts.CreateIn{
			Username:    u.username,
			Email:       u.email,
			Password:    u.password,
			DisplayName: u.username,
			Roles:       []string{u.role},
		})
		if err != nil {
			continue
		}
		userIDs[u.username] = user.ID
	}

	ui.StopSpinner(fmt.Sprintf("Created %d users", len(users)), time.Since(start))

	// Create categories
	ui.StartSpinner("Creating categories...")
	start = time.Now()

	categories := []string{"Technology", "News", "Tutorials", "Reviews", "Opinion"}
	for _, cat := range categories {
		existing, _ := termsSvc.GetBySlug(ctx, cat, "category")
		if existing == nil {
			_, _ = termsSvc.Create(ctx, terms.CreateIn{
				Name:     cat,
				Taxonomy: "category",
			})
		}
	}

	ui.StopSpinner(fmt.Sprintf("Created %d categories", len(categories)), time.Since(start))

	// Create tags
	ui.StartSpinner("Creating tags...")
	start = time.Now()

	tags := []string{"golang", "web", "api", "rest", "wordpress", "mizu", "tutorial", "guide"}
	for _, tag := range tags {
		existing, _ := termsSvc.GetBySlug(ctx, tag, "post_tag")
		if existing == nil {
			_, _ = termsSvc.Create(ctx, terms.CreateIn{
				Name:     tag,
				Taxonomy: "post_tag",
			})
		}
	}

	ui.StopSpinner(fmt.Sprintf("Created %d tags", len(tags)), time.Since(start))

	// Create posts
	ui.StartSpinner("Creating posts...")
	start = time.Now()

	samplePosts := []struct {
		title   string
		content string
		author  string
	}{
		{
			title:   "Getting Started with CMS",
			content: "Welcome to your new CMS! This post will help you get started with the basics of content management.",
			author:  "admin",
		},
		{
			title:   "Understanding the REST API",
			content: "The CMS provides a WordPress-compatible REST API. Learn how to use it to build headless applications.",
			author:  "editor",
		},
		{
			title:   "Theme Development Guide",
			content: "Learn how to create custom themes for your CMS installation with this comprehensive guide.",
			author:  "author",
		},
		{
			title:   "Plugin Architecture",
			content: "Extend your CMS with plugins. This post covers the plugin architecture and how to create your own.",
			author:  "admin",
		},
		{
			title:   "SEO Best Practices",
			content: "Optimize your content for search engines with these SEO best practices and tips.",
			author:  "editor",
		},
	}

	for _, p := range samplePosts {
		existing, _ := postsSvc.GetBySlug(ctx, p.title, "post")
		if existing != nil {
			continue
		}
		authorID := userIDs[p.author]
		if authorID == "" {
			authorID = userIDs["admin"]
		}
		_, _ = postsSvc.Create(ctx, posts.CreateIn{
			Title:   p.title,
			Content: p.content,
			Status:  "publish",
			Author:  authorID,
			Type:    "post",
		})
	}

	ui.StopSpinner(fmt.Sprintf("Created %d posts", len(samplePosts)), time.Since(start))

	// Create pages
	ui.StartSpinner("Creating pages...")
	start = time.Now()

	pages := []struct {
		title   string
		content string
	}{
		{"About", "Learn more about our CMS and the team behind it."},
		{"Contact", "Get in touch with us. We'd love to hear from you!"},
		{"Privacy Policy", "This privacy policy explains how we handle your data."},
		{"Terms of Service", "By using this site, you agree to these terms of service."},
	}

	for _, p := range pages {
		existing, _ := postsSvc.GetBySlug(ctx, p.title, "page")
		if existing != nil {
			continue
		}
		_, _ = postsSvc.Create(ctx, posts.CreateIn{
			Title:   p.title,
			Content: p.content,
			Status:  "publish",
			Author:  userIDs["admin"],
			Type:    "page",
		})
	}

	ui.StopSpinner(fmt.Sprintf("Created %d pages", len(pages)), time.Since(start))

	ui.Summary([][2]string{
		{"Users", fmt.Sprintf("%d", len(users))},
		{"Categories", fmt.Sprintf("%d", len(categories))},
		{"Tags", fmt.Sprintf("%d", len(tags))},
		{"Posts", fmt.Sprintf("%d", len(samplePosts))},
		{"Pages", fmt.Sprintf("%d", len(pages))},
	})

	ui.Success("Sample data seeded successfully!")
	ui.Blank()
	ui.Hint("Run 'cms serve' to start the server and explore the content")

	return nil
}
