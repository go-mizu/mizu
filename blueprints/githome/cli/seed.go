package cli

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/pkg/seed/local"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
	"github.com/spf13/cobra"
)

func newSeedCmd() *cobra.Command {
	seedCmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed data into GitHome",
		Long:  "Seed demo data or import repositories from various sources",
	}

	seedCmd.AddCommand(
		newSeedDemoCmd(),
		newSeedLocalCmd(),
	)

	return seedCmd
}

func newSeedDemoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "demo",
		Short: "Seed demo data (users, repos, issues)",
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
		Short: "Import local git repositories from $HOME/github",
		Long: `Scan a directory for git repositories in the format $ORG/$REPO and import them into GitHome.

By default, scans $HOME/github for repositories organized as:
  $HOME/github/
    org1/
      repo1/
      repo2/
    org2/
      repo3/

Each unique org directory becomes an organization in GitHome,
and each repository is linked (not copied) to its local path.`,
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

// setSiteAdmin directly updates the site_admin field for a user.
func setSiteAdmin(ctx context.Context, db *sql.DB, userID int64, isAdmin bool) error {
	_, err := db.ExecContext(ctx, `UPDATE users SET site_admin = $2 WHERE id = $1`, userID, isAdmin)
	return err
}
