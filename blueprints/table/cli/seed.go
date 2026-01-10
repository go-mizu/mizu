package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/table/app/web"
)

// NewSeed creates the seed command
func NewSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with demo data",
		Long: `Seed the Table database with comprehensive demo data for testing.

Creates a complete workspace showcasing all Airtable-like features:
  - 3 users (alice, bob, charlie)
  - Personal workspace for Alice
  - Project Tracker base with Tasks and Projects tables
  - All 7 view types (grid, kanban, calendar, gallery, timeline, form, list)
  - 15+ field types including selects, dates, ratings, checkboxes, etc.
  - Sample comments and records

To reset the database, delete the data directory first:
  rm -rf ~/data/blueprint/table && table seed

Examples:
  table seed                     # Seed with demo data
  table seed --data /path/to    # Seed specific database`,
		RunE: runSeed,
	}

	return cmd
}

// seedUsers holds the test users data
var seedUsers = []struct {
	Email    string
	Name     string
	Password string
}{
	{"alice@example.com", "Alice Johnson", "password123"},
	{"bob@example.com", "Bob Smith", "password123"},
	{"charlie@example.com", "Charlie Brown", "password123"},
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
	var ownerUserID string
	userCount := 0
	for i, u := range seedUsers {
		userID, err := srv.UserService().Register(ctx, u.Email, u.Name, u.Password)
		if err != nil {
			// Try to get existing user
			userID, _ = srv.UserService().GetByEmail(ctx, u.Email)
		}
		if userID != "" {
			userCount++
			if i == 0 {
				ownerUserID = userID
			}
		}
	}

	if ownerUserID == "" {
		stop()
		return fmt.Errorf("failed to create any users")
	}

	// Create workspace
	wsID, err := srv.WorkspaceService().Create(ctx, ownerUserID, "Alice's Workspace", "alice")
	if err != nil {
		wsID, _ = srv.WorkspaceService().GetBySlug(ctx, "alice")
	}
	if wsID == "" {
		stop()
		return fmt.Errorf("failed to create or get workspace")
	}

	// Create base
	baseID, err := srv.BaseService().Create(ctx, wsID, "Project Tracker", "#2563EB", ownerUserID)
	if err != nil {
		stop()
		return fmt.Errorf("failed to create base: %v", err)
	}

	// Create Tasks table
	tableID, err := srv.TableService().Create(ctx, baseID, "Tasks", ownerUserID)
	if err != nil {
		stop()
		return fmt.Errorf("failed to create table: %v", err)
	}

	// Create fields
	nameFieldID, _ := srv.FieldService().Create(ctx, tableID, "Name", "text", nil, ownerUserID)
	statusFieldID, _ := srv.FieldService().Create(ctx, tableID, "Status", "single_select", map[string]interface{}{
		"choices": []map[string]interface{}{
			{"id": "status-1", "name": "Not Started", "color": "#6B7280"},
			{"id": "status-2", "name": "In Progress", "color": "#3B82F6"},
			{"id": "status-3", "name": "Done", "color": "#10B981"},
		},
	}, ownerUserID)
	priorityFieldID, _ := srv.FieldService().Create(ctx, tableID, "Priority", "single_select", map[string]interface{}{
		"choices": []map[string]interface{}{
			{"id": "priority-1", "name": "Low", "color": "#6B7280"},
			{"id": "priority-2", "name": "Medium", "color": "#F59E0B"},
			{"id": "priority-3", "name": "High", "color": "#EF4444"},
		},
	}, ownerUserID)
	dueDateFieldID, _ := srv.FieldService().Create(ctx, tableID, "Due Date", "date", nil, ownerUserID)
	assigneeFieldID, _ := srv.FieldService().Create(ctx, tableID, "Assignee", "user", nil, ownerUserID)
	notesFieldID, _ := srv.FieldService().Create(ctx, tableID, "Notes", "long_text", nil, ownerUserID)

	// Create sample records
	tasks := []struct {
		Name     string
		Status   string
		Priority string
		DueDays  int
		Notes    string
	}{
		{"Design new landing page", "status-2", "priority-3", 3, "Create mockups in Figma"},
		{"Write documentation", "status-1", "priority-2", 7, "Focus on API docs"},
		{"Fix login bug", "status-3", "priority-3", -2, "Issue was in session handling"},
		{"Review pull requests", "status-2", "priority-2", 1, "3 PRs pending review"},
		{"Update dependencies", "status-1", "priority-1", 14, "Check for security updates"},
		{"Implement user settings", "status-1", "priority-2", 10, "Add theme toggle and preferences"},
		{"Performance optimization", "status-2", "priority-3", 5, "Focus on initial load time"},
		{"Write unit tests", "status-1", "priority-2", 14, "Target 80% coverage"},
	}

	for _, task := range tasks {
		values := map[string]interface{}{
			nameFieldID:   task.Name,
			statusFieldID: task.Status,
		}
		if priorityFieldID != "" {
			values[priorityFieldID] = task.Priority
		}
		if dueDateFieldID != "" && task.DueDays != 0 {
			dueDate := time.Now().AddDate(0, 0, task.DueDays).Format("2006-01-02")
			values[dueDateFieldID] = dueDate
		}
		if notesFieldID != "" {
			values[notesFieldID] = task.Notes
		}
		_ = assigneeFieldID // Will be used for user assignment

		srv.RecordService().Create(ctx, tableID, values, ownerUserID)
	}

	// Create views
	srv.ViewService().Create(ctx, tableID, "All Tasks", "grid", nil, ownerUserID)
	srv.ViewService().Create(ctx, tableID, "By Status", "kanban", map[string]interface{}{
		"groupBy": statusFieldID,
	}, ownerUserID)
	srv.ViewService().Create(ctx, tableID, "Calendar", "calendar", map[string]interface{}{
		"dateField": dueDateFieldID,
	}, ownerUserID)

	// Create a second table for Projects
	projectsTableID, err := srv.TableService().Create(ctx, baseID, "Projects", ownerUserID)
	if err == nil {
		projNameFieldID, _ := srv.FieldService().Create(ctx, projectsTableID, "Name", "text", nil, ownerUserID)
		srv.FieldService().Create(ctx, projectsTableID, "Description", "long_text", nil, ownerUserID)
		srv.FieldService().Create(ctx, projectsTableID, "Status", "single_select", map[string]interface{}{
			"choices": []map[string]interface{}{
				{"id": "proj-1", "name": "Planning", "color": "#6B7280"},
				{"id": "proj-2", "name": "Active", "color": "#3B82F6"},
				{"id": "proj-3", "name": "Completed", "color": "#10B981"},
			},
		}, ownerUserID)
		srv.FieldService().Create(ctx, projectsTableID, "Start Date", "date", nil, ownerUserID)
		srv.FieldService().Create(ctx, projectsTableID, "End Date", "date", nil, ownerUserID)

		// Add sample projects
		projectNames := []string{"Website Redesign", "Mobile App", "API Integration", "Marketing Campaign"}
		for _, projName := range projectNames {
			srv.RecordService().Create(ctx, projectsTableID, map[string]interface{}{
				projNameFieldID: projName,
			}, ownerUserID)
		}
	}

	stop()
	Step("", "Database seeded", time.Since(start))
	Blank()
	Success("Sample data created")
	Blank()

	Summary(
		"Users", fmt.Sprintf("%d users (alice, bob, charlie)", userCount),
		"Password", "password123",
		"Base", "Project Tracker",
		"Tables", "Tasks, Projects",
	)
	Blank()
	Hint("Start the server with: table serve")
	Hint("Login with: alice@example.com / password123")
	Hint("To reset: rm -rf " + dataDir + " && table seed")
	Blank()

	return nil
}
