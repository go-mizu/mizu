package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/kanban/app/web"
	"github.com/go-mizu/blueprints/kanban/feature/columns"
	"github.com/go-mizu/blueprints/kanban/feature/comments"
	"github.com/go-mizu/blueprints/kanban/feature/cycles"
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
		Short: "Seed the database with demo data",
		Long: `Seed the Kanban database with demo data for testing.

Creates a complete workspace with project and issues:
  - 5 users (alice, bob, charlie, diana, eve)
  - Acme Corp workspace
  - Engineering team
  - Product Launch project with columns
  - 15 realistic issues with assignees
  - 3 cycles (Sprint 1, 2, 3)
  - Comments on issues

Examples:
  kanban seed                     # Seed with demo data
  kanban seed --data /path/to    # Seed specific database`,
		RunE: runSeed,
	}

	return cmd
}

// seedUsers holds the test users data
var seedUsers = []struct {
	Username    string
	Email       string
	DisplayName string
	Password    string
}{
	{"alice", "alice@example.com", "Alice Johnson", "password123"},
	{"bob", "bob@example.com", "Bob Smith", "password123"},
	{"charlie", "charlie@example.com", "Charlie Brown", "password123"},
	{"diana", "diana@example.com", "Diana Prince", "password123"},
	{"eve", "eve@example.com", "Eve Wilson", "password123"},
}

// seedIssues holds realistic issue data
var seedIssues = []struct {
	Title       string
	Description string
	Column      int // 0=Backlog, 1=Todo, 2=In Progress, 3=Done
	Priority    string
	Assignee    int // index into seedUsers, -1 for none
}{
	{"Implement user authentication flow", "Add login, logout, and session management with secure cookie handling", 3, "high", 0},
	{"Design database schema for workspaces", "Create tables for workspaces, teams, and member relationships", 3, "high", 1},
	{"Setup CI/CD pipeline", "Configure GitHub Actions for testing and deployment", 2, "medium", 1},
	{"Add dark mode support", "Implement theme switching with CSS variables and localStorage persistence", 2, "low", 0},
	{"Create kanban board UI", "Build drag-and-drop board interface with column management", 2, "high", 2},
	{"Implement issue comments", "Add commenting system with markdown support and real-time updates", 1, "medium", 3},
	{"Add email notifications", "Send email notifications for issue updates and mentions", 1, "low", -1},
	{"Build project settings page", "Create settings UI for project configuration and member management", 1, "medium", 2},
	{"Implement search functionality", "Add global search for issues with filters and keyboard shortcuts", 0, "medium", -1},
	{"Add file attachments", "Allow uploading and attaching files to issues", 0, "low", -1},
	{"Create mobile responsive layout", "Optimize UI for mobile devices and tablets", 0, "medium", -1},
	{"Add keyboard shortcuts", "Implement keyboard navigation for power users", 0, "low", 4},
	{"Build analytics dashboard", "Create dashboard showing project metrics and velocity", 0, "low", -1},
	{"Implement issue templates", "Allow creating reusable issue templates", 0, "low", -1},
	{"Add webhook integrations", "Support webhooks for external service notifications", 0, "low", -1},
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

	// Create all test users
	createdUsers := make([]*users.User, 0, len(seedUsers))
	for _, u := range seedUsers {
		user, _, err := srv.UserService().Register(ctx, &users.RegisterIn{
			Username:    u.Username,
			Email:       u.Email,
			DisplayName: u.DisplayName,
			Password:    u.Password,
		})
		if err != nil {
			// Try to get existing user
			user, _ = srv.UserService().GetByEmail(ctx, u.Email)
		}
		if user != nil {
			createdUsers = append(createdUsers, user)
		}
	}

	if len(createdUsers) == 0 {
		stop()
		return fmt.Errorf("failed to create any users")
	}

	ownerUser := createdUsers[0] // Alice is the owner

	// Create workspace
	ws, err := srv.WorkspaceService().Create(ctx, ownerUser.ID, &workspaces.CreateIn{
		Slug: "acme",
		Name: "Acme Corp",
	})
	if err != nil {
		ws, _ = srv.WorkspaceService().GetBySlug(ctx, "acme")
	}
	if ws == nil {
		stop()
		return fmt.Errorf("failed to create or get workspace")
	}

	// Add all users to workspace
	for i, u := range createdUsers {
		if i == 0 {
			continue // Skip owner, already a member
		}
		role := "member"
		if i == 1 {
			role = "admin"
		}
		_, _ = srv.WorkspaceService().AddMember(ctx, ws.ID, u.ID, role)
	}

	// Create team
	team, err := srv.TeamService().Create(ctx, ws.ID, &teams.CreateIn{
		Key:  "ENG",
		Name: "Engineering",
	})
	if err != nil {
		teamList, _ := srv.TeamService().ListByWorkspace(ctx, ws.ID)
		if len(teamList) > 0 {
			team = teamList[0]
		}
	}
	if team == nil {
		stop()
		return fmt.Errorf("failed to create or get team")
	}

	// Add users to team
	for i, u := range createdUsers {
		role := "member"
		if i == 0 || i == 1 {
			role = "lead"
		}
		_ = srv.TeamService().AddMember(ctx, team.ID, u.ID, role)
	}

	// Create project
	project, err := srv.ProjectService().Create(ctx, team.ID, &projects.CreateIn{
		Key:  "PROD",
		Name: "Product Launch",
	})
	if err != nil {
		projectList, _ := srv.ProjectService().ListByTeam(ctx, team.ID)
		if len(projectList) > 0 {
			project = projectList[0]
		}
	}
	if project == nil {
		stop()
		return fmt.Errorf("failed to create or get project")
	}

	// Create columns
	colNames := []string{"Backlog", "Todo", "In Progress", "Done"}
	var createdCols []*columns.Column
	for i, name := range colNames {
		col, err := srv.ColumnService().Create(ctx, project.ID, &columns.CreateIn{
			Name:     name,
			Position: i,
		})
		if err != nil {
			// Get existing columns
			cols, _ := srv.ColumnService().ListByProject(ctx, project.ID)
			for _, c := range cols {
				if c.Name == name {
					col = c
					break
				}
			}
		}
		if col != nil {
			createdCols = append(createdCols, col)
		}
	}

	// Set first column as default
	if len(createdCols) > 0 {
		_ = srv.ColumnService().SetDefault(ctx, project.ID, createdCols[0].ID)
	}

	// Create cycles
	now := time.Now()
	cycle1, _ := srv.CycleService().Create(ctx, team.ID, &cycles.CreateIn{
		Name:      "Sprint 1",
		StartDate: now.AddDate(0, 0, -14),
		EndDate:   now.AddDate(0, 0, -1),
	})

	cycle2, _ := srv.CycleService().Create(ctx, team.ID, &cycles.CreateIn{
		Name:      "Sprint 2",
		StartDate: now,
		EndDate:   now.AddDate(0, 0, 13),
	})

	cycle3, _ := srv.CycleService().Create(ctx, team.ID, &cycles.CreateIn{
		Name:      "Sprint 3",
		StartDate: now.AddDate(0, 0, 14),
		EndDate:   now.AddDate(0, 0, 27),
	})

	// Create issues
	var createdIssues []*issues.Issue
	for _, issueData := range seedIssues {
		var columnID string
		if issueData.Column < len(createdCols) {
			columnID = createdCols[issueData.Column].ID
		}

		issue, err := srv.IssueService().Create(ctx, project.ID, ownerUser.ID, &issues.CreateIn{
			Title:    issueData.Title,
			ColumnID: columnID,
		})
		if err != nil {
			continue
		}

		createdIssues = append(createdIssues, issue)

		// Assign to user if specified
		if issueData.Assignee >= 0 && issueData.Assignee < len(createdUsers) {
			_ = srv.AssigneeService().Add(ctx, issue.ID, createdUsers[issueData.Assignee].ID)
		}

		// Attach some issues to active cycle
		if cycle2 != nil && (issueData.Column == 1 || issueData.Column == 2) {
			_ = srv.IssueService().AttachCycle(ctx, issue.Key, cycle2.ID)
		}

		// Attach completed issues to completed cycle
		if cycle1 != nil && issueData.Column == 3 {
			_ = srv.IssueService().AttachCycle(ctx, issue.Key, cycle1.ID)
		}

		// Attach backlog issues to planning cycle
		if cycle3 != nil && issueData.Column == 0 && issueData.Priority == "medium" {
			_ = srv.IssueService().AttachCycle(ctx, issue.Key, cycle3.ID)
		}
	}

	// Add some comments to issues
	commentTexts := []string{
		"Great progress on this! Let me know if you need any help.",
		"I've reviewed the implementation and it looks good. A few minor suggestions in the PR.",
		"Can we schedule a quick sync to discuss the approach?",
		"Updated the design docs based on our discussion.",
		"This is blocked by the API changes. Should be unblocked by EOD.",
	}

	for i, issue := range createdIssues {
		if i >= 5 {
			break
		}
		// Add 1-2 comments per issue
		numComments := (i % 2) + 1
		for j := 0; j < numComments && j < len(commentTexts); j++ {
			authorIdx := (i + j) % len(createdUsers)
			_, _ = srv.CommentService().Create(ctx, issue.ID, createdUsers[authorIdx].ID, &comments.CreateIn{
				Content: commentTexts[(i+j)%len(commentTexts)],
			})
		}
	}

	stop()
	Step("", "Database seeded", time.Since(start))
	Blank()
	Success("Sample data created")
	Blank()

	Summary(
		"Users", fmt.Sprintf("%d users (alice, bob, charlie, diana, eve)", len(createdUsers)),
		"Password", "password123",
		"Workspace", "acme",
		"Project", "PROD - Product Launch",
		"Issues", fmt.Sprintf("%d issues", len(createdIssues)),
		"Cycles", "3 (Sprint 1, Sprint 2, Sprint 3)",
	)
	Blank()
	Hint("Start the server with: kanban serve")
	Hint("Login with: alice@example.com / password123")
	Blank()

	return nil
}
