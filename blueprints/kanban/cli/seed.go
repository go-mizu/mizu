package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/kanban/app/web"
	"github.com/go-mizu/blueprints/kanban/feature/columns"
	"github.com/go-mizu/blueprints/kanban/feature/issues"
	"github.com/go-mizu/blueprints/kanban/feature/projects"
	"github.com/go-mizu/blueprints/kanban/feature/teams"
	"github.com/go-mizu/blueprints/kanban/feature/users"
	"github.com/go-mizu/blueprints/kanban/feature/workspaces"
)

// NewSeed creates the seed command
func NewSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with sample data",
		Long: `Seed the Kanban database with sample data for testing.

Creates:
  - Demo user (demo@example.com / password)
  - Sample workspace and team
  - Sample project with columns
  - Sample issues

Examples:
  kanban seed                     # Seed with default data
  kanban seed --data /path/to    # Seed specific database`,
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

	// Create demo user
	user, _, err := srv.UserService().Register(ctx, &users.RegisterIn{
		Username: "demo",
		Email:    "demo@example.com",
		Password: "password",
	})
	if err != nil {
		stop()
		Warn(fmt.Sprintf("User may already exist: %v", err))
		// Try to get existing user
		user, _ = srv.UserService().GetByEmail(ctx, "demo@example.com")
		if user == nil {
			return fmt.Errorf("failed to create or get user: %w", err)
		}
	}

	// Create workspace
	ws, err := srv.WorkspaceService().Create(ctx, user.ID, &workspaces.CreateIn{
		Slug: "acme",
		Name: "Acme Inc",
	})
	if err != nil {
		stop()
		Warn(fmt.Sprintf("Workspace may already exist: %v", err))
		ws, _ = srv.WorkspaceService().GetBySlug(ctx, "acme")
		if ws == nil {
			return fmt.Errorf("failed to create or get workspace: %w", err)
		}
	}

	// Create team
	team, err := srv.TeamService().Create(ctx, ws.ID, &teams.CreateIn{
		Key:  "ENG",
		Name: "Engineering",
	})
	if err != nil {
		stop()
		Warn(fmt.Sprintf("Team may already exist: %v", err))
	}
	if team == nil {
		teamList, _ := srv.TeamService().ListByWorkspace(ctx, ws.ID)
		if len(teamList) > 0 {
			team = teamList[0]
		}
	}

	if team != nil {
		// Create project
		project, err := srv.ProjectService().Create(ctx, team.ID, &projects.CreateIn{
			Key:  "DEMO",
			Name: "Demo Project",
		})
		if err != nil {
			Warn(fmt.Sprintf("Project may already exist: %v", err))
		}
		if project == nil {
			projectList, _ := srv.ProjectService().ListByTeam(ctx, team.ID)
			if len(projectList) > 0 {
				project = projectList[0]
			}
		}

		if project != nil {
			// Create columns
			colNames := []string{"Backlog", "To Do", "In Progress", "Done"}
			for i, name := range colNames {
				_, err := srv.ColumnService().Create(ctx, project.ID, &columns.CreateIn{
					Name:     name,
					Position: i,
				})
				if err != nil {
					// Column may already exist
				}
			}

			// Create sample issues
			issueTitles := []string{
				"Setup project structure",
				"Implement user authentication",
				"Design database schema",
				"Create API endpoints",
				"Build frontend components",
			}
			for _, title := range issueTitles {
				_, err := srv.IssueService().Create(ctx, project.ID, user.ID, &issues.CreateIn{
					Title: title,
				})
				if err != nil {
					// Issue may already exist
				}
			}
		}
	}

	stop()
	Step("", "Database seeded", time.Since(start))
	Blank()
	Success("Sample data created")
	Blank()

	Summary(
		"User", "demo@example.com",
		"Password", "password",
		"Workspace", "acme",
	)
	Blank()
	Hint("Start the server with: kanban serve")
	Blank()

	return nil
}
