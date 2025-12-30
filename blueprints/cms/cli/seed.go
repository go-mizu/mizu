package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/cms/app/web"
	"github.com/go-mizu/blueprints/cms/feature/categories"
	"github.com/go-mizu/blueprints/cms/feature/comments"
	"github.com/go-mizu/blueprints/cms/feature/menus"
	"github.com/go-mizu/blueprints/cms/feature/pages"
	"github.com/go-mizu/blueprints/cms/feature/posts"
	"github.com/go-mizu/blueprints/cms/feature/settings"
	"github.com/go-mizu/blueprints/cms/feature/tags"
	"github.com/go-mizu/blueprints/cms/feature/users"
)

// NewSeed creates the seed command
func NewSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with demo data",
		Long: `Seed the CMS database with demo data for testing.

Creates sample content:
  - Admin user (admin@example.com / password123)
  - Categories (Technology, Business, Lifestyle)
  - Tags (Go, API, Tutorial, etc.)
  - Sample posts with content
  - Sample pages (About, Contact)
  - Navigation menus

To reset the database, delete the data directory first:
  rm -rf ~/data/blueprint/cms && cms seed

Examples:
  cms seed                     # Seed with demo data
  cms seed --data /path/to    # Seed specific database`,
		RunE: runSeed,
	}

	return cmd
}

func runSeed(cmd *cobra.Command, args []string) error {
	Blank()
	Header("", "Seed Database")
	Blank()

	Summary("Data", dataDir)
	Blank()

	start := time.Now()
	stop := StartSpinner("Seeding database...")

	srv, err := web.New(web.Config{
		Addr:    ":0",
		DataDir: dataDir,
		Dev:     false,
	})
	if err != nil {
		stop()
		Error(fmt.Sprintf("Failed to create server: %v", err))
		return err
	}
	defer srv.Close()

	ctx := context.Background()

	// Create admin user
	admin, _, err := srv.UserService().Register(ctx, &users.RegisterIn{
		Email:    "admin@example.com",
		Password: "password123",
		Name:     "Admin User",
	})
	if err != nil {
		admin, _ = srv.UserService().GetByEmail(ctx, "admin@example.com")
	}
	if admin == nil {
		stop()
		return fmt.Errorf("failed to create admin user")
	}

	// Update admin role
	adminRole := "admin"
	srv.UserService().Update(ctx, admin.ID, &users.UpdateIn{Role: &adminRole})

	// Create categories
	catTech, _ := srv.CategoryService().Create(ctx, &categories.CreateIn{
		Name:        "Technology",
		Description: "Latest tech news and tutorials",
	})
	catBiz, _ := srv.CategoryService().Create(ctx, &categories.CreateIn{
		Name:        "Business",
		Description: "Business insights and strategies",
	})
	catLife, _ := srv.CategoryService().Create(ctx, &categories.CreateIn{
		Name:        "Lifestyle",
		Description: "Tips for a better life",
	})

	// Create tags
	tagGo, _ := srv.TagService().Create(ctx, &tags.CreateIn{Name: "Go"})
	tagAPI, _ := srv.TagService().Create(ctx, &tags.CreateIn{Name: "API"})
	tagTutorial, _ := srv.TagService().Create(ctx, &tags.CreateIn{Name: "Tutorial"})
	tagTips, _ := srv.TagService().Create(ctx, &tags.CreateIn{Name: "Tips"})
	srv.TagService().Create(ctx, &tags.CreateIn{Name: "Best Practices"})

	// Create posts
	samplePosts := []struct {
		Title      string
		Excerpt    string
		Content    string
		CategoryID string
		TagIDs     []string
		Status     string
	}{
		{
			Title:   "Getting Started with Go",
			Excerpt: "Learn the basics of Go programming language",
			Content: `# Getting Started with Go

Go is a statically typed, compiled programming language designed at Google. It is syntactically similar to C, but with memory safety, garbage collection, structural typing, and CSP-style concurrency.

## Why Go?

- **Simple and Clean Syntax**: Go is designed to be easy to learn and use.
- **Fast Compilation**: Go compiles very quickly.
- **Built-in Concurrency**: Goroutines make concurrent programming easy.
- **Strong Standard Library**: Go comes with a comprehensive standard library.

## Hello World

` + "```go\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}\n```" + `

Start your Go journey today!`,
			CategoryID: catTech.ID,
			TagIDs:     []string{tagGo.ID, tagTutorial.ID},
			Status:     "published",
		},
		{
			Title:   "Building RESTful APIs",
			Excerpt: "A comprehensive guide to building REST APIs",
			Content: `# Building RESTful APIs

REST (Representational State Transfer) is an architectural style for designing networked applications.

## Key Principles

1. **Stateless**: Each request contains all information needed.
2. **Client-Server**: Clear separation of concerns.
3. **Uniform Interface**: Consistent resource identification.
4. **Cacheable**: Responses can be cached.

## Best Practices

- Use proper HTTP methods (GET, POST, PUT, DELETE)
- Return appropriate status codes
- Use JSON for data exchange
- Implement proper error handling
- Add authentication and authorization

Happy coding!`,
			CategoryID: catTech.ID,
			TagIDs:     []string{tagAPI.ID, tagTutorial.ID},
			Status:     "published",
		},
		{
			Title:   "10 Productivity Tips for Developers",
			Excerpt: "Boost your productivity with these simple tips",
			Content: `# 10 Productivity Tips for Developers

Being productive as a developer is about working smarter, not harder.

## Tips

1. **Use keyboard shortcuts** - Learn your IDE shortcuts.
2. **Take regular breaks** - Use the Pomodoro technique.
3. **Minimize distractions** - Turn off notifications.
4. **Write clean code** - It saves time in the long run.
5. **Automate repetitive tasks** - Don't repeat yourself.
6. **Learn to say no** - Focus on what matters.
7. **Document as you go** - Future you will thank you.
8. **Keep learning** - Stay updated with new technologies.
9. **Use version control** - Git is your friend.
10. **Get enough sleep** - A rested mind is more productive.

Start implementing these tips today!`,
			CategoryID: catLife.ID,
			TagIDs:     []string{tagTips.ID},
			Status:     "published",
		},
	}

	var createdPosts []*posts.Post
	for _, p := range samplePosts {
		allowComments := true
		post, err := srv.PostService().Create(ctx, admin.ID, &posts.CreateIn{
			Title:         p.Title,
			Excerpt:       p.Excerpt,
			Content:       p.Content,
			Status:        p.Status,
			CategoryIDs:   []string{p.CategoryID},
			TagIDs:        p.TagIDs,
			AllowComments: &allowComments,
		})
		if err == nil {
			createdPosts = append(createdPosts, post)
			if p.Status == "published" {
				srv.PostService().Publish(ctx, post.ID)
			}
		}
	}

	// Create pages
	srv.PageService().Create(ctx, admin.ID, &pages.CreateIn{
		Title:   "About Us",
		Content: "# About Us\n\nWelcome to our CMS. This is a modern content management system built with Go.\n\nWe believe in clean, fast, and secure content management.",
		Status:  "published",
	})

	srv.PageService().Create(ctx, admin.ID, &pages.CreateIn{
		Title:   "Contact",
		Content: "# Contact Us\n\nGet in touch with us:\n\n- Email: hello@example.com\n- Phone: +1 234 567 890\n- Address: 123 Main St, City, Country",
		Status:  "published",
	})

	// Create comments on first post
	if len(createdPosts) > 0 {
		srv.CommentService().Create(ctx, &comments.CreateIn{
			PostID:      createdPosts[0].ID,
			AuthorName:  "John Doe",
			AuthorEmail: "john@example.com",
			Content:     "Great article! Very helpful for beginners.",
		})
		srv.CommentService().Create(ctx, &comments.CreateIn{
			PostID:      createdPosts[0].ID,
			AuthorName:  "Jane Smith",
			AuthorEmail: "jane@example.com",
			Content:     "Thanks for sharing. Looking forward to more content!",
		})
	}

	// Create menus
	mainMenu, _ := srv.MenuService().CreateMenu(ctx, &menus.CreateMenuIn{
		Name:     "Main Menu",
		Location: "header",
	})
	if mainMenu != nil {
		srv.MenuService().CreateItem(ctx, mainMenu.ID, &menus.CreateItemIn{
			Title:     "Home",
			URL:       "/",
			SortOrder: 0,
		})
		srv.MenuService().CreateItem(ctx, mainMenu.ID, &menus.CreateItemIn{
			Title:     "Blog",
			URL:       "/blog",
			SortOrder: 1,
		})
		srv.MenuService().CreateItem(ctx, mainMenu.ID, &menus.CreateItemIn{
			Title:     "About",
			URL:       "/about",
			SortOrder: 2,
		})
		srv.MenuService().CreateItem(ctx, mainMenu.ID, &menus.CreateItemIn{
			Title:     "Contact",
			URL:       "/contact",
			SortOrder: 3,
		})
	}

	footerMenu, _ := srv.MenuService().CreateMenu(ctx, &menus.CreateMenuIn{
		Name:     "Footer Menu",
		Location: "footer",
	})
	if footerMenu != nil {
		srv.MenuService().CreateItem(ctx, footerMenu.ID, &menus.CreateItemIn{
			Title:     "Privacy Policy",
			URL:       "/privacy",
			SortOrder: 0,
		})
		srv.MenuService().CreateItem(ctx, footerMenu.ID, &menus.CreateItemIn{
			Title:     "Terms of Service",
			URL:       "/terms",
			SortOrder: 1,
		})
	}

	// Create settings
	srv.SettingsService().Set(ctx, &settings.SetIn{
		Key:       "site_title",
		Value:     "My CMS",
		GroupName: "general",
		IsPublic:  ptrBool(true),
	})
	srv.SettingsService().Set(ctx, &settings.SetIn{
		Key:       "site_description",
		Value:     "A modern content management system",
		GroupName: "general",
		IsPublic:  ptrBool(true),
	})
	srv.SettingsService().Set(ctx, &settings.SetIn{
		Key:       "posts_per_page",
		Value:     "10",
		ValueType: "number",
		GroupName: "reading",
		IsPublic:  ptrBool(true),
	})

	// Count created items
	var categoryCount, tagCount int
	if catTech != nil {
		categoryCount++
	}
	if catBiz != nil {
		categoryCount++
	}
	if catLife != nil {
		categoryCount++
	}
	if tagGo != nil {
		tagCount++
	}
	if tagAPI != nil {
		tagCount++
	}
	if tagTutorial != nil {
		tagCount++
	}
	if tagTips != nil {
		tagCount++
	}

	stop()
	Step("", "Database seeded", time.Since(start))
	Blank()
	Success("Sample data created")
	Blank()

	Summary(
		"User", "admin@example.com",
		"Password", "password123",
		"Posts", fmt.Sprintf("%d posts", len(createdPosts)),
		"Categories", fmt.Sprintf("%d categories", categoryCount),
		"Tags", fmt.Sprintf("%d+ tags", tagCount),
		"Menus", "2 menus (header, footer)",
	)
	Blank()
	Hint("Start the server with: cms serve")
	Hint("Login with: admin@example.com / password123")
	Hint("To reset: rm -rf " + dataDir + " && cms seed")
	Blank()

	return nil
}

func ptrBool(b bool) *bool {
	return &b
}
