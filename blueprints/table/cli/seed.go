package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/table/app/web"
	"github.com/go-mizu/blueprints/table/feature/bases"
	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/feature/tables"
	"github.com/go-mizu/blueprints/table/feature/views"
	"github.com/go-mizu/blueprints/table/feature/workspaces"
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

	totalStart := time.Now()

	// Initialize server
	stepStart := time.Now()
	fmt.Print("  Initializing server... ")
	srv, err := web.New(web.Config{
		Addr:    ":0",
		DataDir: dataDir,
		Dev:     false,
	})
	if err != nil {
		fmt.Printf("✗ %v\n", err)
		return err
	}
	defer srv.Close()
	fmt.Printf("✓ (%v)\n", time.Since(stepStart).Round(time.Millisecond))

	ctx := context.Background()
	sectionStart := time.Now()

	// Create users
	fmt.Print("  Creating users...\n")
	var ownerUserID string
	userCount := 0
	for _, u := range seedUsers {
		stepStart = time.Now()
		fmt.Printf("    • %s... ", u.Email)

		user, _, err := srv.UserService().Register(ctx, u.Email, u.Name, u.Password)
		if err != nil {
			// Try to get existing user
			existingUser, _ := srv.UserService().GetByEmail(ctx, u.Email)
			if existingUser != nil {
				user = existingUser
				fmt.Printf("exists (%v)\n", time.Since(stepStart).Round(time.Millisecond))
			} else {
				fmt.Printf("✗ %v\n", err)
				continue
			}
		} else {
			fmt.Printf("✓ (%v)\n", time.Since(stepStart).Round(time.Millisecond))
		}

		userCount++
		if ownerUserID == "" {
			ownerUserID = user.ID
		}
	}
	Step("", fmt.Sprintf("Users ready (%d)", userCount), time.Since(sectionStart))

	if ownerUserID == "" {
		return fmt.Errorf("failed to create any users")
	}

	// Create workspace
	sectionStart = time.Now()
	stepStart = time.Now()
	fmt.Printf("  Creating workspace... ")
	ws, err := srv.WorkspaceService().Create(ctx, ownerUserID, workspaces.CreateIn{
		Name: "Alice's Workspace",
		Slug: "alice",
	})
	if err != nil {
		existingWs, _ := srv.WorkspaceService().GetBySlug(ctx, "alice")
		if existingWs != nil {
			ws = existingWs
			fmt.Printf("exists (%v)\n", time.Since(stepStart).Round(time.Millisecond))
		} else {
			fmt.Printf("✗ %v\n", err)
			return err
		}
	} else {
		fmt.Printf("✓ (%v)\n", time.Since(stepStart).Round(time.Millisecond))
	}
	Step("", "Workspace ready", time.Since(sectionStart))

	// Create base
	sectionStart = time.Now()
	stepStart = time.Now()
	fmt.Printf("  Creating base 'Project Tracker'... ")
	base, err := srv.BaseService().Create(ctx, ownerUserID, bases.CreateIn{
		WorkspaceID: ws.ID,
		Name:        "Project Tracker",
		Color:       "#2563EB",
	})
	if err != nil {
		fmt.Printf("✗ %v\n", err)
		return err
	}
	fmt.Printf("✓ (%v)\n", time.Since(stepStart).Round(time.Millisecond))
	Step("", "Base ready", time.Since(sectionStart))

	// Create Tasks table
	sectionStart = time.Now()
	stepStart = time.Now()
	fmt.Printf("  Creating table 'Tasks'... ")
	table, err := srv.TableService().Create(ctx, ownerUserID, tables.CreateIn{
		BaseID: base.ID,
		Name:   "Tasks",
	})
	if err != nil {
		fmt.Printf("✗ %v\n", err)
		return err
	}
	fmt.Printf("✓ (%v)\n", time.Since(stepStart).Round(time.Millisecond))
	Step("", "Tasks table ready", time.Since(sectionStart))

	// Create fields
	sectionStart = time.Now()
	fmt.Print("  Creating fields...\n")

	createField := func(name, fieldType string, options map[string]any) *fields.Field {
		stepStart = time.Now()
		fmt.Printf("    • %s (%s)... ", name, fieldType)

		var optionsJSON json.RawMessage
		if options != nil {
			optionsJSON, _ = json.Marshal(options)
		}

		field, err := srv.FieldService().Create(ctx, ownerUserID, fields.CreateIn{
			TableID: table.ID,
			Name:    name,
			Type:    fieldType,
			Options: optionsJSON,
		})
		if err != nil {
			fmt.Printf("✗ %v\n", err)
			return nil
		}
		fmt.Printf("✓ (%v)\n", time.Since(stepStart).Round(time.Millisecond))
		return field
	}

	nameField := createField("Name", "single_line_text", nil)
	statusField := createField("Status", "single_select", map[string]any{
		"choices": []map[string]any{
			{"id": "status-1", "name": "Not Started", "color": "#6B7280"},
			{"id": "status-2", "name": "In Progress", "color": "#3B82F6"},
			{"id": "status-3", "name": "Review", "color": "#8B5CF6"},
			{"id": "status-4", "name": "Done", "color": "#10B981"},
		},
	})
	priorityField := createField("Priority", "single_select", map[string]any{
		"choices": []map[string]any{
			{"id": "priority-1", "name": "Low", "color": "#6B7280"},
			{"id": "priority-2", "name": "Medium", "color": "#F59E0B"},
			{"id": "priority-3", "name": "High", "color": "#EF4444"},
			{"id": "priority-4", "name": "Critical", "color": "#DC2626"},
		},
	})
	dueDateField := createField("Due Date", "date", nil)
	startDateField := createField("Start Date", "date", nil)
	assigneeField := createField("Assignee", "collaborator", nil)
	notesField := createField("Notes", "long_text", nil)

	// Additional field types
	createField("Completed", "checkbox", nil)
	createField("Effort (pts)", "number", nil)
	createField("Confidence", "rating", map[string]any{"max": 5})
	createField("Tags", "multi_select", map[string]any{
		"choices": []map[string]any{
			{"id": "tag-1", "name": "Frontend", "color": "#3B82F6"},
			{"id": "tag-2", "name": "Backend", "color": "#10B981"},
			{"id": "tag-3", "name": "Design", "color": "#EC4899"},
			{"id": "tag-4", "name": "Bug", "color": "#EF4444"},
			{"id": "tag-5", "name": "Feature", "color": "#8B5CF6"},
			{"id": "tag-6", "name": "Docs", "color": "#F59E0B"},
		},
	})
	createField("Budget", "currency", nil)
	createField("Progress", "percent", nil)
	createField("Contact Email", "email", nil)
	createField("Reference URL", "url", nil)
	Step("", "Fields ready", time.Since(sectionStart))

	// Create sample records
	sectionStart = time.Now()
	fmt.Print("  Preparing records...\n")
	tasks := []struct {
		Name      string
		Status    string
		Priority  string
		StartDays int
		DueDays   int
		Notes     string
	}{
		{"Design new landing page", "status-2", "priority-3", -5, 3, "Create mockups in Figma"},
		{"Write documentation", "status-1", "priority-2", 0, 7, "Focus on API docs"},
		{"Fix login bug", "status-4", "priority-3", -10, -2, "Issue was in session handling"},
		{"Review pull requests", "status-3", "priority-2", -1, 1, "3 PRs pending review"},
		{"Update dependencies", "status-1", "priority-1", 7, 14, "Check for security updates"},
		{"Implement user settings", "status-1", "priority-2", 3, 10, "Add theme toggle and preferences"},
		{"Performance optimization", "status-2", "priority-4", -3, 5, "Focus on initial load time"},
		{"Write unit tests", "status-1", "priority-2", 7, 14, "Target 80% coverage"},
		{"Database migration", "status-2", "priority-3", -2, 4, "Migrate to new schema"},
		{"API rate limiting", "status-1", "priority-2", 5, 12, "Implement throttling"},
		{"Mobile responsive fixes", "status-3", "priority-2", -4, 2, "Fix breakpoints"},
		{"Security audit", "status-1", "priority-4", 1, 8, "Run penetration tests"},
	}

	var recordsData []map[string]any
	for _, task := range tasks {
		fmt.Printf("    • %s\n", task.Name)

		cells := make(map[string]any)
		if nameField != nil {
			cells[nameField.ID] = task.Name
		}
		if statusField != nil {
			cells[statusField.ID] = task.Status
		}
		if priorityField != nil {
			cells[priorityField.ID] = task.Priority
		}
		if startDateField != nil {
			cells[startDateField.ID] = time.Now().AddDate(0, 0, task.StartDays).Format("2006-01-02")
		}
		if dueDateField != nil {
			cells[dueDateField.ID] = time.Now().AddDate(0, 0, task.DueDays).Format("2006-01-02")
		}
		if notesField != nil {
			cells[notesField.ID] = task.Notes
		}
		if assigneeField != nil {
			cells[assigneeField.ID] = ownerUserID
		}

		recordsData = append(recordsData, cells)
	}

	stepStart = time.Now()
	fmt.Printf("  Inserting %d records... ", len(recordsData))
	createdRecords, err := srv.RecordService().CreateBatch(ctx, table.ID, recordsData, ownerUserID)
	if err != nil {
		fmt.Printf("✗ %v\n", err)
		return err
	}
	fmt.Printf("✓ (%v)\n", time.Since(stepStart).Round(time.Millisecond))
	recordCount := len(createdRecords)
	Step("", "Task records ready", time.Since(sectionStart))

	// Create views
	sectionStart = time.Now()
	fmt.Print("  Creating views...\n")
	viewTypes := []struct {
		Name   string
		Type   string
		Config map[string]any
	}{
		{"All Tasks", "grid", nil},
		{"By Status", "kanban", map[string]any{"groupBy": statusField.ID}},
		{"By Priority", "kanban", map[string]any{"groupBy": priorityField.ID}},
		{"Calendar", "calendar", map[string]any{"dateField": dueDateField.ID}},
		{"Gallery", "gallery", nil},
		{"Timeline", "timeline", map[string]any{"dateField": startDateField.ID, "endDateField": dueDateField.ID}},
		{"Submit Task", "form", map[string]any{"title": "Submit a New Task", "description": "Use this form to submit a new task."}},
	}

	viewCount := 0
	for _, vt := range viewTypes {
		stepStart = time.Now()
		fmt.Printf("    • %s (%s)... ", vt.Name, vt.Type)

		_, err := srv.ViewService().Create(ctx, ownerUserID, views.CreateIn{
			TableID: table.ID,
			Name:    vt.Name,
			Type:    vt.Type,
		})
		if err != nil {
			fmt.Printf("✗ %v\n", err)
		} else {
			viewCount++
			fmt.Printf("✓ (%v)\n", time.Since(stepStart).Round(time.Millisecond))
		}
	}
	Step("", "Views ready", time.Since(sectionStart))

	// Create Projects table
	sectionStart = time.Now()
	stepStart = time.Now()
	fmt.Printf("  Creating table 'Projects'... ")
	projectsTable, err := srv.TableService().Create(ctx, ownerUserID, tables.CreateIn{
		BaseID: base.ID,
		Name:   "Projects",
	})
	if err != nil {
		fmt.Printf("✗ %v\n", err)
	} else {
		fmt.Printf("✓ (%v)\n", time.Since(stepStart).Round(time.Millisecond))

		// Create project fields
		fieldsStart := time.Now()
		fmt.Print("  Creating project fields...\n")
		projNameField := createFieldForTable(srv, ctx, projectsTable.ID, "Name", "single_line_text", nil, ownerUserID)
		createFieldForTable(srv, ctx, projectsTable.ID, "Description", "long_text", nil, ownerUserID)
		createFieldForTable(srv, ctx, projectsTable.ID, "Status", "single_select", map[string]any{
			"choices": []map[string]any{
				{"id": "proj-1", "name": "Planning", "color": "#6B7280"},
				{"id": "proj-2", "name": "Active", "color": "#3B82F6"},
				{"id": "proj-3", "name": "Completed", "color": "#10B981"},
			},
		}, ownerUserID)
		createFieldForTable(srv, ctx, projectsTable.ID, "Start Date", "date", nil, ownerUserID)
		createFieldForTable(srv, ctx, projectsTable.ID, "End Date", "date", nil, ownerUserID)
		Step("", "Project fields ready", time.Since(fieldsStart))

		// Add sample projects
		projectStart := time.Now()
		fmt.Print("  Preparing project records...\n")
		projectNames := []string{"Website Redesign", "Mobile App", "API Integration", "Marketing Campaign"}
		var projectRecords []map[string]any
		for _, projName := range projectNames {
			fmt.Printf("    • %s\n", projName)
			cells := make(map[string]any)
			if projNameField != nil {
				cells[projNameField.ID] = projName
			}
			projectRecords = append(projectRecords, cells)
		}
		stepStart = time.Now()
		fmt.Printf("  Inserting %d project records... ", len(projectRecords))
		if _, err := srv.RecordService().CreateBatch(ctx, projectsTable.ID, projectRecords, ownerUserID); err != nil {
			fmt.Printf("✗ %v\n", err)
		} else {
			fmt.Printf("✓ (%v)\n", time.Since(stepStart).Round(time.Millisecond))
		}
		Step("", "Project records ready", time.Since(projectStart))
		Step("", "Projects table ready", time.Since(sectionStart))
	}

	Blank()
	Step("", "Database seeded", time.Since(totalStart))
	Blank()
	Success("Sample data created")
	Blank()

	Summary(
		"Users", fmt.Sprintf("%d users (alice, bob, charlie)", userCount),
		"Password", "password123",
		"Base", "Project Tracker",
		"Tables", fmt.Sprintf("Tasks (%d records, 15 fields), Projects (4 records)", recordCount),
		"Views", fmt.Sprintf("%d views (Grid, Kanban x2, Calendar, Gallery, Timeline, Form)", viewCount),
	)
	Blank()
	Hint("Start server: table serve")
	Hint("Login with: alice@example.com / password123")
	Hint("To reset: rm -rf " + dataDir + " && table seed")
	Blank()

	return nil
}

func createFieldForTable(srv *web.Server, ctx context.Context, tableID, name, fieldType string, options map[string]any, userID string) *fields.Field {
	stepStart := time.Now()
	fmt.Printf("    • %s (%s)... ", name, fieldType)

	var optionsJSON json.RawMessage
	if options != nil {
		optionsJSON, _ = json.Marshal(options)
	}

	field, err := srv.FieldService().Create(ctx, userID, fields.CreateIn{
		TableID: tableID,
		Name:    name,
		Type:    fieldType,
		Options: optionsJSON,
	})
	if err != nil {
		fmt.Printf("✗ %v\n", err)
		return nil
	}
	fmt.Printf("✓ (%v)\n", time.Since(stepStart).Round(time.Millisecond))
	return field
}
