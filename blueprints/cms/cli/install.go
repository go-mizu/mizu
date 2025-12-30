package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/mizu/blueprints/cms/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/cms/feature/options"
	"github.com/go-mizu/mizu/blueprints/cms/feature/posts"
	"github.com/go-mizu/mizu/blueprints/cms/feature/terms"
	"github.com/go-mizu/mizu/blueprints/cms/store/duckdb"
)

var (
	siteTitle  string
	siteURL    string
	adminUser  string
	adminEmail string
	adminPass  string
)

// NewInstall creates the install command.
func NewInstall() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Run WordPress-style installation",
		Long: `Runs the WordPress 5-minute installation process.

This sets up:
  - Site configuration (title, URL)
  - Administrator account
  - Default content (Hello World post, Sample Page)
  - Default category (Uncategorized)
  - Default options`,
		RunE: runInstall,
	}

	cmd.Flags().StringVar(&siteTitle, "title", "", "Site title")
	cmd.Flags().StringVar(&siteURL, "url", "http://localhost:8080", "Site URL")
	cmd.Flags().StringVar(&adminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&adminEmail, "admin-email", "", "Admin email")
	cmd.Flags().StringVar(&adminPass, "admin-pass", "", "Admin password")

	return cmd
}

func runInstall(cmd *cobra.Command, args []string) error {
	ui := NewUI()

	ui.Header(iconDatabase, "WordPress-Compatible CMS Installation")
	ui.Blank()

	// Interactive prompts if flags not provided
	reader := bufio.NewReader(os.Stdin)

	if siteTitle == "" {
		fmt.Print("  Site Title: ")
		siteTitle, _ = reader.ReadString('\n')
		siteTitle = strings.TrimSpace(siteTitle)
		if siteTitle == "" {
			siteTitle = "My CMS Site"
		}
	}

	if adminUser == "" {
		fmt.Print("  Admin Username: ")
		adminUser, _ = reader.ReadString('\n')
		adminUser = strings.TrimSpace(adminUser)
		if adminUser == "" {
			adminUser = "admin"
		}
	}

	if adminEmail == "" {
		fmt.Print("  Admin Email: ")
		adminEmail, _ = reader.ReadString('\n')
		adminEmail = strings.TrimSpace(adminEmail)
		if adminEmail == "" {
			adminEmail = "admin@example.com"
		}
	}

	if adminPass == "" {
		fmt.Print("  Admin Password: ")
		adminPass, _ = reader.ReadString('\n')
		adminPass = strings.TrimSpace(adminPass)
		if adminPass == "" {
			adminPass = "admin123"
		}
	}

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

	// Initialize default options
	ui.StartSpinner("Setting up options...")
	start = time.Now()

	if err := optionsSvc.InitDefaults(ctx, siteURL, siteTitle, adminEmail); err != nil {
		ui.StopSpinnerError("Failed to initialize options")
		return err
	}

	ui.StopSpinner("Options configured", time.Since(start))

	// Create admin user
	ui.StartSpinner("Creating administrator...")
	start = time.Now()

	existingUser, _ := accountsSvc.GetByLogin(ctx, adminUser)
	var adminID string
	if existingUser != nil {
		adminID = existingUser.ID
		ui.StopSpinner("Administrator exists", time.Since(start))
	} else {
		admin, err := accountsSvc.Create(ctx, accounts.CreateIn{
			Username:    adminUser,
			Email:       adminEmail,
			Password:    adminPass,
			DisplayName: adminUser,
			Roles:       []string{"administrator"},
		})
		if err != nil {
			ui.StopSpinnerError("Failed to create administrator")
			return err
		}
		adminID = admin.ID
		ui.StopSpinner("Administrator created", time.Since(start))
	}

	// Create default category
	ui.StartSpinner("Creating default category...")
	start = time.Now()

	existingCat, _ := termsSvc.GetBySlug(ctx, "uncategorized", "category")
	if existingCat == nil {
		_, err := termsSvc.Create(ctx, terms.CreateIn{
			Name:     "Uncategorized",
			Slug:     "uncategorized",
			Taxonomy: "category",
		})
		if err != nil {
			ui.StopSpinnerError("Failed to create category")
			return err
		}
	}

	ui.StopSpinner("Default category ready", time.Since(start))

	// Create Hello World post
	ui.StartSpinner("Creating sample content...")
	start = time.Now()

	existingPost, _ := postsSvc.GetBySlug(ctx, "hello-world", "post")
	if existingPost == nil {
		_, err := postsSvc.Create(ctx, posts.CreateIn{
			Title:   "Hello World!",
			Content: "Welcome to your new CMS site. This is your first post. Edit or delete it, then start writing!",
			Status:  "publish",
			Author:  adminID,
			Type:    "post",
		})
		if err != nil {
			ui.StopSpinnerError("Failed to create post")
			return err
		}
	}

	// Create Sample Page
	existingPage, _ := postsSvc.GetBySlug(ctx, "sample-page", "page")
	if existingPage == nil {
		_, err := postsSvc.Create(ctx, posts.CreateIn{
			Title:   "Sample Page",
			Content: "This is a sample page. It's different from a blog post because it stays in one place and will show up in your site navigation (in most themes).",
			Status:  "publish",
			Author:  adminID,
			Type:    "page",
		})
		if err != nil {
			ui.StopSpinnerError("Failed to create page")
			return err
		}
	}

	ui.StopSpinner("Sample content created", time.Since(start))

	// Summary
	ui.Summary([][2]string{
		{"Site Title", siteTitle},
		{"Site URL", siteURL},
		{"Admin User", adminUser},
		{"Admin Email", adminEmail},
		{"Data Dir", dataDir},
	})

	ui.Success("Installation complete!")
	ui.Blank()
	ui.Hint("Run 'cms serve' to start the server")
	ui.Hint("Login at " + siteURL + "/wp-admin/ with your admin credentials")

	return nil
}
