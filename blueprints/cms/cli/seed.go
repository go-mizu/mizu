package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/cms/app/web"
	"github.com/go-mizu/blueprints/cms/feature/auth"
	"github.com/go-mizu/blueprints/cms/feature/collections"
	"github.com/spf13/cobra"
)

// NewSeed creates the seed command.
func NewSeed() *cobra.Command {
	var dbPath string

	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with sample data",
		Long:  "Create sample users, pages, posts, and other content.",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv, err := web.New(web.Config{
				DBPath: dbPath,
			})
			if err != nil {
				return fmt.Errorf("create server: %w", err)
			}
			defer srv.Close()

			ctx := context.Background()

			// Create admin user
			fmt.Println("Creating admin user...")
			admin, err := srv.AuthService().Register(ctx, "users", &auth.RegisterInput{
				Email:     "admin@example.com",
				Password:  "password",
				FirstName: "Admin",
				LastName:  "User",
			})
			if err != nil {
				fmt.Printf("Warning: Could not create admin user: %v\n", err)
			} else {
				fmt.Printf("Created admin: %s\n", admin.User.Email)

				// Update to admin role
				srv.CollectionsService().UpdateByID(ctx, "users", admin.User.ID, &collections.UpdateInput{
					Data: map[string]any{
						"roles": []string{"admin"},
					},
				})
			}

			// Create sample categories
			fmt.Println("Creating categories...")
			categories := []map[string]any{
				{"name": "Technology", "slug": "technology", "description": "Tech news and tutorials"},
				{"name": "Design", "slug": "design", "description": "UI/UX and design articles"},
				{"name": "Business", "slug": "business", "description": "Business and marketing"},
			}
			categoryIDs := make(map[string]string)
			for _, cat := range categories {
				doc, err := srv.CollectionsService().Create(ctx, "categories", &collections.CreateInput{Data: cat})
				if err != nil {
					fmt.Printf("Warning: Could not create category %s: %v\n", cat["name"], err)
				} else {
					categoryIDs[cat["slug"].(string)] = doc["id"].(string)
					fmt.Printf("Created category: %s\n", cat["name"])
				}
			}

			// Create sample tags
			fmt.Println("Creating tags...")
			tags := []map[string]any{
				{"name": "Go", "slug": "go"},
				{"name": "JavaScript", "slug": "javascript"},
				{"name": "React", "slug": "react"},
				{"name": "CSS", "slug": "css"},
			}
			tagIDs := make(map[string]string)
			for _, tag := range tags {
				doc, err := srv.CollectionsService().Create(ctx, "tags", &collections.CreateInput{Data: tag})
				if err != nil {
					fmt.Printf("Warning: Could not create tag %s: %v\n", tag["name"], err)
				} else {
					tagIDs[tag["slug"].(string)] = doc["id"].(string)
					fmt.Printf("Created tag: %s\n", tag["name"])
				}
			}

			// Create sample pages
			fmt.Println("Creating pages...")
			pages := []map[string]any{
				{
					"title":   "Home",
					"slug":    "home",
					"content": `{"root":{"children":[{"children":[{"text":"Welcome to our CMS!"}],"type":"h1"}],"type":"root"}}`,
					"status":  "published",
				},
				{
					"title":   "About",
					"slug":    "about",
					"content": `{"root":{"children":[{"children":[{"text":"About Us"}],"type":"h1"},{"children":[{"text":"We are a great company."}],"type":"p"}],"type":"root"}}`,
					"status":  "published",
				},
				{
					"title":   "Contact",
					"slug":    "contact",
					"content": `{"root":{"children":[{"children":[{"text":"Contact Us"}],"type":"h1"},{"children":[{"text":"Email: hello@example.com"}],"type":"p"}],"type":"root"}}`,
					"status":  "published",
				},
			}
			for _, page := range pages {
				_, err := srv.CollectionsService().Create(ctx, "pages", &collections.CreateInput{Data: page})
				if err != nil {
					fmt.Printf("Warning: Could not create page %s: %v\n", page["title"], err)
				} else {
					fmt.Printf("Created page: %s\n", page["title"])
				}
			}

			// Create sample posts
			fmt.Println("Creating posts...")
			var authorID string
			if admin != nil {
				authorID = admin.User.ID
			}
			posts := []map[string]any{
				{
					"title":       "Getting Started with Go",
					"slug":        "getting-started-with-go",
					"excerpt":     "Learn the basics of Go programming language.",
					"content":     `{"root":{"children":[{"children":[{"text":"Getting Started with Go"}],"type":"h1"},{"children":[{"text":"Go is a statically typed, compiled language designed at Google."}],"type":"p"}],"type":"root"}}`,
					"author":      authorID,
					"categories":  []string{categoryIDs["technology"]},
					"tags":        []string{tagIDs["go"]},
					"status":      "published",
					"publishedAt": time.Now().Format(time.RFC3339),
				},
				{
					"title":       "Modern CSS Techniques",
					"slug":        "modern-css-techniques",
					"excerpt":     "Explore modern CSS features and techniques.",
					"content":     `{"root":{"children":[{"children":[{"text":"Modern CSS Techniques"}],"type":"h1"},{"children":[{"text":"CSS has evolved significantly in recent years."}],"type":"p"}],"type":"root"}}`,
					"author":      authorID,
					"categories":  []string{categoryIDs["design"]},
					"tags":        []string{tagIDs["css"]},
					"status":      "published",
					"publishedAt": time.Now().Format(time.RFC3339),
				},
				{
					"title":   "Building React Applications",
					"slug":    "building-react-applications",
					"excerpt": "A guide to building modern React applications.",
					"content": `{"root":{"children":[{"children":[{"text":"Building React Applications"}],"type":"h1"},{"children":[{"text":"React is a popular JavaScript library for building user interfaces."}],"type":"p"}],"type":"root"}}`,
					"author":  authorID,
					"categories": []string{
						categoryIDs["technology"],
						categoryIDs["design"],
					},
					"tags":        []string{tagIDs["javascript"], tagIDs["react"]},
					"status":      "draft",
					"publishedAt": nil,
				},
			}
			for _, post := range posts {
				_, err := srv.CollectionsService().Create(ctx, "posts", &collections.CreateInput{Data: post})
				if err != nil {
					fmt.Printf("Warning: Could not create post %s: %v\n", post["title"], err)
				} else {
					fmt.Printf("Created post: %s\n", post["title"])
				}
			}

			// Seed site settings global
			fmt.Println("Creating site settings...")
			_, err = srv.GlobalsService().Update(ctx, "site-settings", map[string]any{
				"siteName":        "My CMS Site",
				"siteDescription": "A Payload CMS compatible content management system",
				"social": map[string]any{
					"twitter":  "https://twitter.com/example",
					"github":   "https://github.com/example",
					"linkedin": "https://linkedin.com/company/example",
				},
				"contact": map[string]any{
					"email":   "hello@example.com",
					"phone":   "+1 (555) 123-4567",
					"address": "123 Main St, City, Country",
				},
				"seo": map[string]any{
					"titleSuffix":        " | My CMS Site",
					"defaultDescription": "Welcome to our content management system.",
				},
			})
			if err != nil {
				fmt.Printf("Warning: Could not create site settings: %v\n", err)
			} else {
				fmt.Println("Created site settings")
			}

			// Seed navigation global
			fmt.Println("Creating navigation...")
			_, err = srv.GlobalsService().Update(ctx, "navigation", map[string]any{
				"header": []map[string]any{
					{"label": "Home", "type": "custom", "url": "/"},
					{"label": "About", "type": "custom", "url": "/about"},
					{"label": "Blog", "type": "custom", "url": "/blog"},
					{"label": "Contact", "type": "custom", "url": "/contact"},
				},
				"footer": []map[string]any{
					{"label": "Privacy Policy", "type": "custom", "url": "/privacy"},
					{"label": "Terms of Service", "type": "custom", "url": "/terms"},
				},
			})
			if err != nil {
				fmt.Printf("Warning: Could not create navigation: %v\n", err)
			} else {
				fmt.Println("Created navigation")
			}

			fmt.Println("\nDatabase seeded successfully!")
			fmt.Println("\nAdmin credentials:")
			fmt.Println("  Email: admin@example.com")
			fmt.Println("  Password: password")

			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")

	return cmd
}
