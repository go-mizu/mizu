package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/workspace/app/web"
	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/databases"
	"github.com/go-mizu/blueprints/workspace/feature/pages"
	"github.com/go-mizu/blueprints/workspace/feature/users"
	"github.com/go-mizu/blueprints/workspace/feature/views"
	"github.com/go-mizu/blueprints/workspace/feature/workspaces"
)

// NewSeed creates the seed command
func NewSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with demo data",
		Long: `Seed the Workspace database with demo data for testing.

Creates a complete workspace with pages and databases:
  - 3 users (alice, bob, charlie)
  - Personal workspace for Alice
  - Getting Started page with tutorial content
  - Sample database with tasks
  - Multiple views (table, board, calendar)

To reset the database, delete the data directory first:
  rm -rf ~/data/blueprint/workspace && workspace seed

Examples:
  workspace seed                     # Seed with demo data
  workspace seed --data /path/to    # Seed specific database`,
		RunE: runSeed,
	}

	return cmd
}

// seedUsers holds the test users data
var seedUsers = []struct {
	Email       string
	Name        string
	Password    string
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
	createdUsers := make([]*users.User, 0, len(seedUsers))
	for _, u := range seedUsers {
		user, _, err := srv.UserService().Register(ctx, &users.RegisterIn{
			Email:    u.Email,
			Name:     u.Name,
			Password: u.Password,
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
		Name: "Alice's Workspace",
		Slug: "alice",
	})
	if err != nil {
		ws, _ = srv.WorkspaceService().GetBySlug(ctx, "alice")
	}
	if ws == nil {
		stop()
		return fmt.Errorf("failed to create or get workspace")
	}

	// Add other users to workspace as members
	for i, u := range createdUsers {
		if i == 0 {
			continue // Skip owner
		}
		srv.MemberService().Add(ctx, ws.ID, u.ID, "member", ownerUser.ID)
	}

	// Create Getting Started page
	gettingStarted, err := srv.PageService().Create(ctx, &pages.CreateIn{
		WorkspaceID: ws.ID,
		ParentType:  pages.ParentWorkspace,
		Title:       "Getting Started",
		Icon:        "👋",
		CreatedBy:   ownerUser.ID,
	})
	if err != nil {
		stop()
		return fmt.Errorf("failed to create getting started page: %v", err)
	}

	// Add blocks to the page
	blockContents := []struct {
		Type    blocks.BlockType
		Content blocks.Content
	}{
		{blocks.BlockHeading1, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Welcome to Workspace!"}}}},
		{blocks.BlockParagraph, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "This is your personal workspace. Here you can create pages, databases, and collaborate with your team."}}}},
		{blocks.BlockHeading2, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Quick Tips"}}}},
		{blocks.BlockBulletList, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Press / to open the command menu and add new blocks"}}}},
		{blocks.BlockBulletList, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Drag blocks to reorder them"}}}},
		{blocks.BlockBulletList, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Use @mentions to reference people and pages"}}}},
		{blocks.BlockTodo, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Create your first page"}}, Checked: ptrBool(false)}},
		{blocks.BlockTodo, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Add a database"}}, Checked: ptrBool(false)}},
		{blocks.BlockTodo, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Invite a team member"}}, Checked: ptrBool(false)}},
		{blocks.BlockDivider, blocks.Content{}},
		{blocks.BlockCallout, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Need help? Check out our documentation or reach out to support."}}, Icon: "💡", Color: "blue"}},
	}

	for i, bc := range blockContents {
		srv.BlockService().Create(ctx, &blocks.CreateIn{
			PageID:    gettingStarted.ID,
			Type:      bc.Type,
			Content:   bc.Content,
			Position:  i,
			CreatedBy: ownerUser.ID,
		})
	}

	// Create a child page
	childPage, _ := srv.PageService().Create(ctx, &pages.CreateIn{
		WorkspaceID: ws.ID,
		ParentID:    gettingStarted.ID,
		ParentType:  pages.ParentPage,
		Title:       "My Notes",
		Icon:        "📝",
		CreatedBy:   ownerUser.ID,
	})
	if childPage != nil {
		srv.BlockService().Create(ctx, &blocks.CreateIn{
			PageID:    childPage.ID,
			Type:      blocks.BlockParagraph,
			Content:   blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Start writing your notes here..."}}},
			Position:  0,
			CreatedBy: ownerUser.ID,
		})
	}

	// Create a Tasks database page
	tasksPage, err := srv.PageService().Create(ctx, &pages.CreateIn{
		WorkspaceID: ws.ID,
		ParentType:  pages.ParentWorkspace,
		Title:       "Tasks",
		Icon:        "📋",
		CreatedBy:   ownerUser.ID,
	})
	if err != nil {
		stop()
		return fmt.Errorf("failed to create tasks page: %v", err)
	}

	// Create a database in the page
	db, err := srv.DatabaseService().Create(ctx, &databases.CreateIn{
		WorkspaceID: ws.ID,
		PageID:      tasksPage.ID,
		Title:       "Task Tracker",
		CreatedBy:   ownerUser.ID,
		Properties: []databases.Property{
			{Name: "Name", Type: databases.PropTitle},
			{Name: "Status", Type: databases.PropSelect, Config: databases.SelectConfig{
				Options: []databases.SelectOption{
					{Name: "Not Started", Color: "gray"},
					{Name: "In Progress", Color: "blue"},
					{Name: "Done", Color: "green"},
				},
			}},
			{Name: "Priority", Type: databases.PropSelect, Config: databases.SelectConfig{
				Options: []databases.SelectOption{
					{Name: "Low", Color: "gray"},
					{Name: "Medium", Color: "yellow"},
					{Name: "High", Color: "red"},
				},
			}},
			{Name: "Due Date", Type: databases.PropDate},
			{Name: "Assignee", Type: databases.PropPerson},
			{Name: "Attachments", Type: databases.PropFiles},
		},
	})
	if err != nil {
		stop()
		return fmt.Errorf("failed to create database: %v", err)
	}

	// Create views for the database
	srv.ViewService().Create(ctx, &views.CreateIn{
		DatabaseID: db.ID,
		Name:       "All Tasks",
		Type:       views.ViewTable,
		CreatedBy:  ownerUser.ID,
	})

	srv.ViewService().Create(ctx, &views.CreateIn{
		DatabaseID: db.ID,
		Name:       "Board",
		Type:       views.ViewBoard,
		GroupBy:    "Status",
		CreatedBy:  ownerUser.ID,
	})

	srv.ViewService().Create(ctx, &views.CreateIn{
		DatabaseID: db.ID,
		Name:       "Calendar",
		Type:       views.ViewCalendar,
		CalendarBy: "Due Date",
		CreatedBy:  ownerUser.ID,
	})

	// Add sample tasks as database items (pages with parent = database)
	tasks := []struct {
		Title       string
		Status      string
		Priority    string
		DueDays     int
		Attachments []map[string]interface{}
	}{
		{"Design new landing page", "In Progress", "High", 3, []map[string]interface{}{
			{"id": "file-1", "name": "mockup.png", "url": "https://via.placeholder.com/800x600?text=Landing+Page+Mockup", "type": "image/png"},
			{"id": "file-2", "name": "design-spec.pdf", "url": "https://example.com/design-spec.pdf", "type": "application/pdf"},
		}},
		{"Write documentation", "Not Started", "Medium", 7, []map[string]interface{}{
			{"id": "file-3", "name": "outline.md", "url": "https://example.com/outline.md", "type": "text/markdown"},
		}},
		{"Fix login bug", "Done", "High", -2, nil},
		{"Review pull requests", "In Progress", "Medium", 1, nil},
		{"Update dependencies", "Not Started", "Low", 14, []map[string]interface{}{
			{"id": "file-4", "name": "audit-report.txt", "url": "https://example.com/audit.txt", "type": "text/plain"},
		}},
	}

	for _, task := range tasks {
		props := pages.Properties{
			"Name":     pages.PropertyValue{Type: "title", Value: task.Title},
			"Status":   pages.PropertyValue{Type: "select", Value: task.Status},
			"Priority": pages.PropertyValue{Type: "select", Value: task.Priority},
		}
		if task.DueDays != 0 {
			dueDate := time.Now().AddDate(0, 0, task.DueDays).Format("2006-01-02")
			props["Due Date"] = pages.PropertyValue{Type: "date", Value: dueDate}
		}
		if task.Attachments != nil {
			props["Attachments"] = pages.PropertyValue{Type: "files", Value: task.Attachments}
		}

		srv.PageService().Create(ctx, &pages.CreateIn{
			WorkspaceID: ws.ID,
			ParentID:    db.ID,
			ParentType:  pages.ParentDatabase,
			Title:       task.Title,
			Properties:  props,
			CreatedBy:   ownerUser.ID,
		})
	}

	// Create a Meeting Notes page
	meetingPage, _ := srv.PageService().Create(ctx, &pages.CreateIn{
		WorkspaceID: ws.ID,
		ParentType:  pages.ParentWorkspace,
		Title:       "Meeting Notes",
		Icon:        "📅",
		CreatedBy:   ownerUser.ID,
	})
	if meetingPage != nil {
		meetingBlocks := []struct {
			Type    blocks.BlockType
			Content blocks.Content
		}{
			{blocks.BlockHeading2, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Team Sync - " + time.Now().Format("Jan 2, 2006")}}}},
			{blocks.BlockParagraph, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Attendees: Alice, Bob, Charlie"}}}},
			{blocks.BlockHeading3, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Agenda"}}}},
			{blocks.BlockNumberList, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Project updates"}}}},
			{blocks.BlockNumberList, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Blockers discussion"}}}},
			{blocks.BlockNumberList, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Next steps"}}}},
			{blocks.BlockHeading3, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Action Items"}}}},
			{blocks.BlockTodo, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Follow up on design review"}}, Checked: ptrBool(false)}},
			{blocks.BlockTodo, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Schedule demo with stakeholders"}}, Checked: ptrBool(false)}},
		}

		for i, bc := range meetingBlocks {
			srv.BlockService().Create(ctx, &blocks.CreateIn{
				PageID:    meetingPage.ID,
				Type:      bc.Type,
				Content:   bc.Content,
				Position:  i,
				CreatedBy: ownerUser.ID,
			})
		}
	}

	stop()
	Step("", "Database seeded", time.Since(start))
	Blank()
	Success("Sample data created")
	Blank()

	Summary(
		"Users", fmt.Sprintf("%d users (alice, bob, charlie)", len(createdUsers)),
		"Password", "password123",
		"Workspace", ws.Name,
		"Pages", "Getting Started, Tasks, Meeting Notes",
	)
	Blank()
	Hint("Start the server with: workspace serve")
	Hint("Login with: alice@example.com / password123")
	Hint("To reset: rm -rf " + dataDir + " && workspace seed")
	Blank()

	return nil
}

func ptrBool(b bool) *bool {
	return &b
}
