package cli

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/pkg/ulid"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
	"github.com/spf13/cobra"
)

func newSeedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Seed demo data",
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
			reposStore := duckdb.NewReposStore(db)

			// Create services
			usersSvc := users.NewService(usersStore)
			reposSvc := repos.NewService(reposStore, reposDir)

			// Create admin user
			admin, _, err := usersSvc.Register(ctx, &users.RegisterIn{
				Username: "admin",
				Email:    "admin@githome.local",
				Password: "password123",
				FullName: "Admin User",
			})
			if err != nil {
				slog.Warn("admin user may already exist", "error", err)
			} else {
				slog.Info("created admin user", "username", admin.Username)

				// Make admin
				admin.IsAdmin = true
				if err := usersStore.Update(ctx, admin); err != nil {
					return fmt.Errorf("update admin: %w", err)
				}
			}

			// Create demo user
			demo, _, err := usersSvc.Register(ctx, &users.RegisterIn{
				Username: "demo",
				Email:    "demo@githome.local",
				Password: "demo1234",
				FullName: "Demo User",
			})
			if err != nil {
				slog.Warn("demo user may already exist", "error", err)
			} else {
				slog.Info("created demo user", "username", demo.Username)
			}

			// Get users if they already exist
			if admin == nil {
				admin, _ = usersSvc.GetByUsername(ctx, "admin")
			}
			if demo == nil {
				demo, _ = usersSvc.GetByUsername(ctx, "demo")
			}

			if admin == nil {
				return fmt.Errorf("admin user not found")
			}

			// Create sample repositories
			sampleRepos := []struct {
				Name        string
				Description string
				OwnerID     string
				IsPrivate   bool
			}{
				{
					Name:        "hello-world",
					Description: "A simple hello world repository",
					OwnerID:     admin.ID,
					IsPrivate:   false,
				},
				{
					Name:        "my-project",
					Description: "My awesome project",
					OwnerID:     admin.ID,
					IsPrivate:   false,
				},
				{
					Name:        "private-repo",
					Description: "A private repository",
					OwnerID:     admin.ID,
					IsPrivate:   true,
				},
			}

			for _, r := range sampleRepos {
				repo, err := reposSvc.Create(ctx, r.OwnerID, &repos.CreateIn{
					Name:        r.Name,
					Description: r.Description,
					IsPrivate:   r.IsPrivate,
				})
				if err != nil {
					slog.Warn("repo may already exist", "name", r.Name, "error", err)
					continue
				}
				slog.Info("created repository", "name", repo.Name, "owner", admin.Username)
			}

			// Create sample issues for hello-world
			issuesStore := duckdb.NewIssuesStore(db)
			helloRepo, _ := reposStore.GetByOwnerAndName(ctx, admin.ID, "user", "hello-world")
			if helloRepo != nil {
				issues := []struct {
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

				for i, iss := range issues {
					issue := &struct {
						ID        string
						RepoID    string
						Number    int
						Title     string
						Body      string
						AuthorID  string
						State     string
					}{
						ID:       ulid.New(),
						RepoID:   helloRepo.ID,
						Number:   i + 1,
						Title:    iss.Title,
						Body:     iss.Body,
						AuthorID: admin.ID,
						State:    iss.State,
					}
					if err := issuesStore.Create(ctx, issue); err != nil {
						slog.Warn("issue may already exist", "title", iss.Title, "error", err)
					} else {
						slog.Info("created issue", "number", issue.Number, "title", issue.Title)
					}
				}

				// Update issue counter
				helloRepo.OpenIssueCount = 2
				reposStore.Update(ctx, helloRepo)
			}

			slog.Info("seeding complete")
			return nil
		},
	}
}
