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
  - 45 realistic issues with priorities, assignees, and dates
  - 3 cycles (Sprint 1, 2, 3)
  - Comments on issues

To reset the database, delete the data directory first:
  rm -rf ~/.kanban && kanban seed

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
// Priority: 0=none, 1=urgent, 2=high, 3=medium, 4=low
var seedIssues = []struct {
	Title       string
	Description string
	Column      int // 0=Backlog, 1=Todo, 2=In Progress, 3=Done
	Priority    int // 0=none, 1=urgent, 2=high, 3=medium, 4=low
	Assignee    int // index into seedUsers, -1 for none
	DaysOffset  int // days from now for due date, 0 = no date
	Duration    int // duration in days for gantt chart
}{
	// Completed Issues (Column 3) - past due dates
	{"Implement user authentication flow", "Add login, logout, and session management with secure cookie handling", 3, 2, 0, -10, 5},
	{"Design database schema for workspaces", "Create tables for workspaces, teams, and member relationships", 3, 2, 1, -8, 3},
	{"Setup project structure", "Initialize Go modules, configure linting, and set up directory structure", 3, 2, 0, -15, 2},
	{"Create user registration API", "Build POST /api/auth/register endpoint with validation", 3, 2, 1, -12, 3},
	{"Add password hashing", "Implement bcrypt password hashing for secure storage", 3, 2, 0, -11, 1},
	{"Setup logging middleware", "Add structured logging with request/response tracking", 3, 3, 2, -9, 2},
	{"Create session management", "Implement cookie-based sessions with expiry handling", 3, 2, 0, -7, 2},
	{"Build workspace creation flow", "Allow users to create and configure new workspaces", 3, 3, 1, -6, 3},
	{"Add team management", "Create, update, and delete teams within workspaces", 3, 3, 2, -5, 2},
	{"Implement role-based access", "Add owner, admin, member roles with permission checks", 3, 2, 0, -4, 3},

	// In Progress Issues (Column 2) - current/near future dates
	{"Setup CI/CD pipeline", "Configure GitHub Actions for testing and deployment", 2, 3, 1, 2, 4},
	{"Add dark mode support", "Implement theme switching with CSS variables and localStorage persistence", 2, 4, 0, 5, 3},
	{"Create kanban board UI", "Build drag-and-drop board interface with column management", 2, 1, 2, 3, 5},
	{"Implement real-time updates", "Add WebSocket support for live board updates", 2, 2, 3, 4, 4},
	{"Build issue detail page", "Create detailed view with description, comments, and activity", 2, 3, 0, 6, 3},
	{"Add assignee selection", "Allow assigning team members to issues", 2, 3, 1, 4, 2},
	{"Implement cycle management", "Create sprints/cycles with start and end dates", 2, 1, 2, 5, 4},
	{"Add issue priorities", "Implement urgent, high, medium, low priority levels", 2, 4, 3, 3, 2},
	{"Create project settings", "Build settings page for project configuration", 2, 3, 0, 7, 3},
	{"Implement issue labels", "Add customizable labels for issue categorization", 2, 4, 1, 8, 2},

	// Todo Issues (Column 1) - upcoming dates
	{"Implement issue comments", "Add commenting system with markdown support and real-time updates", 1, 3, 3, 10, 3},
	{"Add email notifications", "Send email notifications for issue updates and mentions", 1, 4, -1, 12, 4},
	{"Build project settings page", "Create settings UI for project configuration and member management", 1, 3, 2, 11, 3},
	{"Add due date picker", "Implement date selection with calendar UI", 1, 3, 0, 9, 2},
	{"Create notification center", "Build in-app notification system with badge counts", 1, 3, 1, 14, 4},
	{"Implement issue linking", "Allow linking related issues (blocks, blocked by, duplicates)", 1, 2, 2, 13, 3},
	{"Add bulk issue operations", "Enable selecting multiple issues for bulk updates", 1, 3, 3, 15, 3},
	{"Create issue export", "Export issues to CSV/JSON format", 1, 4, 0, 16, 2},
	{"Build activity feed", "Show recent activity across all projects", 1, 3, 1, 17, 3},
	{"Implement @mentions", "Add user mentions in comments and descriptions", 1, 2, 2, 12, 2},

	// Backlog Issues (Column 0) - future dates for planning
	{"Implement search functionality", "Add global search for issues with filters and keyboard shortcuts", 0, 3, -1, 25, 4},
	{"Add file attachments", "Allow uploading and attaching files to issues", 0, 4, -1, 28, 3},
	{"Create mobile responsive layout", "Optimize UI for mobile devices and tablets", 0, 3, -1, 30, 5},
	{"Build analytics dashboard", "Create dashboard showing project metrics and velocity", 0, 4, -1, 35, 6},
	{"Implement issue templates", "Allow creating reusable issue templates", 0, 4, -1, 32, 3},
	{"Add webhook integrations", "Support webhooks for external service notifications", 0, 4, -1, 40, 4},
	{"Create API documentation", "Generate OpenAPI docs for all endpoints", 0, 3, -1, 33, 3},
	{"Implement rate limiting", "Add rate limiting to prevent API abuse", 0, 2, -1, 26, 2},
	{"Add OAuth providers", "Support Google, GitHub, Microsoft login", 0, 3, -1, 45, 5},
	{"Build audit logging", "Track all user actions for compliance", 0, 2, -1, 38, 4},
	{"Create backup system", "Implement automated database backups", 0, 2, -1, 42, 3},
	{"Add data export", "Allow users to export all their data", 0, 3, -1, 36, 3},
	{"Implement SSO support", "Add SAML/OIDC single sign-on", 0, 2, -1, 50, 6},
	{"Create workspace templates", "Pre-built workspace configurations", 0, 4, -1, 48, 3},
	{"Add issue automations", "Automate status changes and assignments", 0, 3, -1, 55, 5},
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

	// Create a second project for testing project selector
	project2, err := srv.ProjectService().Create(ctx, team.ID, &projects.CreateIn{
		Key:  "DOCS",
		Name: "Documentation",
	})
	if err != nil {
		projectList, _ := srv.ProjectService().ListByTeam(ctx, team.ID)
		for _, p := range projectList {
			if p.Key == "DOCS" {
				project2 = p
				break
			}
		}
	}

	// Create columns for second project
	if project2 != nil {
		for i, name := range colNames {
			col, _ := srv.ColumnService().Create(ctx, project2.ID, &columns.CreateIn{
				Name:     name,
				Position: i,
			})
			if i == 0 && col != nil {
				_ = srv.ColumnService().SetDefault(ctx, project2.ID, col.ID)
			}
		}
	}

	// Create issues
	var createdIssues []*issues.Issue
	for _, issueData := range seedIssues {
		var columnID string
		if issueData.Column < len(createdCols) {
			columnID = createdCols[issueData.Column].ID
		}

		issue, err := srv.IssueService().Create(ctx, project.ID, ownerUser.ID, &issues.CreateIn{
			Title:       issueData.Title,
			Description: issueData.Description,
			ColumnID:    columnID,
			Priority:    issueData.Priority,
		})
		if err != nil {
			continue
		}

		createdIssues = append(createdIssues, issue)

		// Update issue with dates for calendar/gantt chart views
		if issueData.DaysOffset != 0 || issueData.Duration > 0 {
			dueDate := now.AddDate(0, 0, issueData.DaysOffset)
			var startDate, endDate *time.Time
			if issueData.Duration > 0 {
				start := dueDate.AddDate(0, 0, -issueData.Duration)
				startDate = &start
				endDate = &dueDate
			}
			_, _ = srv.IssueService().Update(ctx, issue.ID, &issues.UpdateIn{
				DueDate:   &dueDate,
				StartDate: startDate,
				EndDate:   endDate,
			})
		}

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

		// Attach backlog issues to planning cycle (medium priority = 3)
		if cycle3 != nil && issueData.Column == 0 && issueData.Priority == 3 {
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
	Hint("To reset: rm -rf " + dataDir + " && kanban seed")
	Blank()

	return nil
}
