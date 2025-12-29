package cli

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/pkg/seed/github"
	"github.com/go-mizu/blueprints/githome/pkg/seed/local"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
	"github.com/spf13/cobra"
)

func newSeedCmd() *cobra.Command {
	seedCmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed data into GitHome",
		Long: `Seed data into GitHome from various sources.

Available sources:
  demo   Create sample users, repositories, and issues
  local  Import git repositories from your local filesystem
  github Import issues, PRs, comments from a GitHub repository`,
	}

	seedCmd.AddCommand(
		newSeedDemoCmd(),
		newSeedLocalCmd(),
		newSeedGitHubCmd(),
	)

	return seedCmd
}

func newSeedDemoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "demo",
		Short: "Create sample users, repositories, and issues",
		Long: `Create sample data for testing and demonstration.

Creates:
  - admin user (login: admin, password: password123)
  - demo user (login: demo, password: demo1234)
  - Sample repositories with issues`,
		Example: "  githome seed demo",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := dataDir + "/githome.db"
			db, err := sql.Open("duckdb", dbPath)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer db.Close()

			_, err = duckdb.New(db)
			if err != nil {
				return fmt.Errorf("create store: %w", err)
			}

			ctx := context.Background()

			// Create stores
			usersStore := duckdb.NewUsersStore(db)
			orgsStore := duckdb.NewOrgsStore(db)
			reposStore := duckdb.NewReposStore(db)
			issuesStore := duckdb.NewIssuesStore(db)

			// Create services
			baseURL := "http://localhost:3000"
			usersSvc := users.NewService(usersStore, baseURL)
			reposSvc := repos.NewService(reposStore, usersStore, orgsStore, baseURL, reposDir)

			// Create admin user
			admin, err := usersSvc.Create(ctx, &users.CreateIn{
				Login:    "admin",
				Email:    "admin@githome.local",
				Password: "password123",
				Name:     "Admin User",
			})
			if err != nil {
				slog.Warn("admin user may already exist", "error", err)
				// Try to get existing admin
				admin, _ = usersSvc.GetByLogin(ctx, "admin")
			} else {
				slog.Info("created admin user", "username", admin.Login)

				// Make admin a site admin
				if err := setSiteAdmin(ctx, db, admin.ID, true); err != nil {
					return fmt.Errorf("update admin: %w", err)
				}
			}

			// Create demo user
			demo, err := usersSvc.Create(ctx, &users.CreateIn{
				Login:    "demo",
				Email:    "demo@githome.local",
				Password: "demo1234",
				Name:     "Demo User",
			})
			if err != nil {
				slog.Warn("demo user may already exist", "error", err)
				demo, _ = usersSvc.GetByLogin(ctx, "demo")
			} else {
				slog.Info("created demo user", "username", demo.Login)
			}

			if admin == nil {
				return fmt.Errorf("admin user not found")
			}

			// Create sample repositories
			sampleRepos := []struct {
				Name        string
				Description string
				IsPrivate   bool
			}{
				{
					Name:        "hello-world",
					Description: "A simple hello world repository",
					IsPrivate:   false,
				},
				{
					Name:        "my-project",
					Description: "My awesome project",
					IsPrivate:   false,
				},
				{
					Name:        "private-repo",
					Description: "A private repository",
					IsPrivate:   true,
				},
			}

			for _, r := range sampleRepos {
				repo, err := reposSvc.Create(ctx, admin.ID, &repos.CreateIn{
					Name:        r.Name,
					Description: r.Description,
					Private:     r.IsPrivate,
				})
				if err != nil {
					slog.Warn("repo may already exist", "name", r.Name, "error", err)
					continue
				}
				slog.Info("created repository", "name", repo.Name, "owner", admin.Login)
			}

			// Create sample issues for hello-world
			helloRepo, _ := reposStore.GetByOwnerAndName(ctx, admin.ID, "hello-world")
			if helloRepo != nil {
				sampleIssues := []struct {
					Title string
					Body  string
					State string
				}{
					{
						Title: "Add README documentation",
						Body:  "We need to add proper documentation to the README file.",
						State: "open",
					},
					{
						Title: "Bug: Application crashes on startup",
						Body:  "When running the application, it crashes immediately.\n\n## Steps to reproduce\n1. Run the app\n2. See error",
						State: "open",
					},
					{
						Title: "Feature: Add dark mode",
						Body:  "It would be great to have a dark mode option.",
						State: "closed",
					},
				}

				for i, iss := range sampleIssues {
					issue := &issues.Issue{
						RepoID:    helloRepo.ID,
						Number:    i + 1,
						Title:     iss.Title,
						Body:      iss.Body,
						CreatorID: admin.ID,
						State:     iss.State,
					}
					if err := issuesStore.Create(ctx, issue); err != nil {
						slog.Warn("issue may already exist", "title", iss.Title, "error", err)
					} else {
						slog.Info("created issue", "number", issue.Number, "title", issue.Title)
					}
				}

				// Update issue counter
				_ = reposStore.IncrementOpenIssues(ctx, helloRepo.ID, 2)
			}

			slog.Info("demo seeding complete")
			return nil
		},
	}
}

func newSeedLocalCmd() *cobra.Command {
	var scanDir string
	var isPublic bool

	cmd := &cobra.Command{
		Use:   "local",
		Short: "Import git repositories from local filesystem",
		Long: `Import git repositories from your local filesystem into GitHome.

Scans a directory for repositories organized as:
  <scan-dir>/
    org1/
      repo1/
      repo2/
    org2/
      repo3/

Each org directory becomes an organization in GitHome.
Repositories are linked (not copied) to their local paths.`,
		Example: `  githome seed local
  githome seed local --scan-dir ~/projects
  githome seed local --public=false`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := dataDir + "/githome.db"
			db, err := sql.Open("duckdb", dbPath)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer db.Close()

			_, err = duckdb.New(db)
			if err != nil {
				return fmt.Errorf("create store: %w", err)
			}

			ctx := context.Background()

			// Ensure admin user exists
			usersStore := duckdb.NewUsersStore(db)
			adminUserID, _, err := local.EnsureAdminUser(ctx, usersStore, nil)
			if err != nil {
				return fmt.Errorf("ensure admin user: %w", err)
			}

			// Create seeder
			seeder := local.NewSeeder(db, local.Config{
				ScanDir:      scanDir,
				AdminUserID:  adminUserID,
				AdminActorID: adminUserID,
				IsPublic:     isPublic,
			})

			// Run seeding
			result, err := seeder.Seed(ctx)
			if err != nil {
				return fmt.Errorf("seed local: %w", err)
			}

			slog.Info("local seeding complete",
				"orgs_created", result.OrgsCreated,
				"repos_created", result.ReposCreated,
				"repos_skipped", result.ReposSkipped)

			if len(result.Errors) > 0 {
				slog.Warn("some errors occurred during seeding", "count", len(result.Errors))
				for _, e := range result.Errors {
					slog.Warn("seed error", "error", e)
				}
			}

			return nil
		},
	}

	// Default scan directory
	defaultScanDir := os.Getenv("HOME") + "/github"

	cmd.Flags().StringVar(&scanDir, "scan-dir", defaultScanDir, "Directory to scan for git repositories")
	cmd.Flags().BoolVar(&isPublic, "public", true, "Make imported repositories public")

	return cmd
}

func newSeedGitHubCmd() *cobra.Command {
	var token string
	var baseURL string
	var isPublic bool
	var maxIssues int
	var maxPRs int
	var maxComments int
	var noComments bool
	var noPRs bool

	cmd := &cobra.Command{
		Use:   "github <owner/repo>",
		Short: "Import issues, PRs, comments from a GitHub repository",
		Long: `Import issues, pull requests, comments, labels, and milestones from a
GitHub repository into GitHome for offline viewing.

Requires a GitHub repository in the format 'owner/repo'.
For higher rate limits, provide a GitHub personal access token via --token or GITHUB_TOKEN env var.

Note: Without authentication, GitHub API allows 60 requests/hour.
With a token, this increases to 5,000 requests/hour.`,
		Example: `  githome seed github golang/go
  githome seed github golang/go --token $GITHUB_TOKEN
  githome seed github golang/go --max-issues 100 --max-prs 50
  githome seed github golang/go --no-comments
  githome seed github mycompany/repo --base-url https://github.mycompany.com/api/v3`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse owner/repo
			parts := strings.SplitN(args[0], "/", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid repository format, expected 'owner/repo'")
			}
			owner, repo := parts[0], parts[1]

			// Get token from env if not provided
			if token == "" {
				token = os.Getenv("GITHUB_TOKEN")
			}

			dbPath := dataDir + "/githome.db"
			db, err := sql.Open("duckdb", dbPath)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer db.Close()

			_, err = duckdb.New(db)
			if err != nil {
				return fmt.Errorf("create store: %w", err)
			}

			ctx := context.Background()

			// Ensure admin user exists
			usersStore := duckdb.NewUsersStore(db)
			adminUserID, err := github.EnsureAdminUser(ctx, usersStore)
			if err != nil {
				return fmt.Errorf("ensure admin user: %w", err)
			}

			// Create config
			config := github.DefaultConfig(owner, repo)
			config.Token = token
			config.AdminUserID = adminUserID
			config.IsPublic = isPublic
			config.MaxIssues = maxIssues
			config.MaxPRs = maxPRs
			config.MaxCommentsPerItem = maxComments
			config.ImportComments = !noComments
			config.ImportPRs = !noPRs

			if baseURL != "" {
				config.BaseURL = baseURL
			}

			// Create seeder and run
			seeder := github.NewSeeder(db, config)
			result, err := seeder.Seed(ctx)
			if err != nil {
				return fmt.Errorf("seed github: %w", err)
			}

			// Print summary
			slog.Info("GitHub seeding complete",
				"repo_created", result.RepoCreated,
				"org_created", result.OrgCreated,
				"users_created", result.UsersCreated,
				"issues_created", result.IssuesCreated,
				"issues_skipped", result.IssuesSkipped,
				"prs_created", result.PRsCreated,
				"prs_skipped", result.PRsSkipped,
				"comments_created", result.CommentsCreated,
				"labels_created", result.LabelsCreated,
				"milestones_created", result.MilestonesCreated)

			if result.RateLimitRemaining < 100 {
				slog.Warn("rate limit running low",
					"remaining", result.RateLimitRemaining,
					"reset", result.RateLimitReset.Format("15:04:05"))
			}

			if len(result.Errors) > 0 {
				slog.Warn("some errors occurred during seeding", "count", len(result.Errors))
				for _, e := range result.Errors {
					slog.Warn("seed error", "error", e)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "GitHub personal access token (or GITHUB_TOKEN env)")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "GitHub API base URL (for GitHub Enterprise)")
	cmd.Flags().BoolVar(&isPublic, "public", true, "Make imported repository public")
	cmd.Flags().IntVar(&maxIssues, "max-issues", 0, "Maximum issues to import (0 = all)")
	cmd.Flags().IntVar(&maxPRs, "max-prs", 0, "Maximum PRs to import (0 = all)")
	cmd.Flags().IntVar(&maxComments, "max-comments", 0, "Maximum comments per issue/PR (0 = all)")
	cmd.Flags().BoolVar(&noComments, "no-comments", false, "Skip importing comments")
	cmd.Flags().BoolVar(&noPRs, "no-prs", false, "Skip importing pull requests")

	return cmd
}

// setSiteAdmin directly updates the site_admin field for a user.
func setSiteAdmin(ctx context.Context, db *sql.DB, userID int64, isAdmin bool) error {
	_, err := db.ExecContext(ctx, `UPDATE users SET site_admin = $2 WHERE id = $1`, userID, isAdmin)
	return err
}
