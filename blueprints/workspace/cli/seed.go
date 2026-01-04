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

	// Create Development Test Page with all block types
	devTestPage, _ := srv.PageService().Create(ctx, &pages.CreateIn{
		WorkspaceID: ws.ID,
		ParentType:  pages.ParentWorkspace,
		Title:       "Development Test Page",
		Icon:        "🧪",
		CreatedBy:   ownerUser.ID,
	})
	if devTestPage != nil {
		createDevTestBlocks(ctx, srv.BlockService(), devTestPage.ID, ownerUser.ID)
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
		"Pages", "Getting Started, Tasks, Meeting Notes, Development Test Page",
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

// createDevTestBlocks creates comprehensive test blocks for the Development Test Page
func createDevTestBlocks(ctx context.Context, blockSvc blocks.API, pageID, userID string) {
	testBlocks := []struct {
		Type    blocks.BlockType
		Content blocks.Content
	}{
		// ========================================
		// Section: Title & Introduction
		// ========================================
		{blocks.BlockHeading1, blocks.Content{RichText: rt("Block Editor Showcase")}},
		{blocks.BlockParagraph, blocks.Content{RichText: []blocks.RichText{
			{Type: "text", Text: "This page demonstrates all available block types with various configurations. Use this page to test the editor functionality and verify styling matches Notion."},
		}}},
		{blocks.BlockDivider, blocks.Content{}},

		// ========================================
		// Section: Text Formatting
		// ========================================
		{blocks.BlockHeading1, blocks.Content{RichText: rt("1. Text Formatting")}},
		{blocks.BlockParagraph, blocks.Content{RichText: []blocks.RichText{
			{Type: "text", Text: "Normal text, "},
			{Type: "text", Text: "bold text", Annotations: blocks.Annotations{Bold: true}},
			{Type: "text", Text: ", "},
			{Type: "text", Text: "italic text", Annotations: blocks.Annotations{Italic: true}},
			{Type: "text", Text: ", "},
			{Type: "text", Text: "underlined", Annotations: blocks.Annotations{Underline: true}},
			{Type: "text", Text: ", "},
			{Type: "text", Text: "strikethrough", Annotations: blocks.Annotations{Strikethrough: true}},
			{Type: "text", Text: ", and "},
			{Type: "text", Text: "inline code", Annotations: blocks.Annotations{Code: true}},
			{Type: "text", Text: "."},
		}}},
		{blocks.BlockParagraph, blocks.Content{RichText: []blocks.RichText{
			{Type: "text", Text: "Combined: "},
			{Type: "text", Text: "bold italic", Annotations: blocks.Annotations{Bold: true, Italic: true}},
			{Type: "text", Text: ", "},
			{Type: "text", Text: "bold underline", Annotations: blocks.Annotations{Bold: true, Underline: true}},
			{Type: "text", Text: ", "},
			{Type: "text", Text: "all styles", Annotations: blocks.Annotations{Bold: true, Italic: true, Underline: true, Strikethrough: true}},
			{Type: "text", Text: "."},
		}}},
		{blocks.BlockParagraph, blocks.Content{RichText: []blocks.RichText{
			{Type: "text", Text: "Colors: "},
			{Type: "text", Text: "gray", Annotations: blocks.Annotations{Color: "gray"}},
			{Type: "text", Text: " "},
			{Type: "text", Text: "brown", Annotations: blocks.Annotations{Color: "brown"}},
			{Type: "text", Text: " "},
			{Type: "text", Text: "orange", Annotations: blocks.Annotations{Color: "orange"}},
			{Type: "text", Text: " "},
			{Type: "text", Text: "yellow", Annotations: blocks.Annotations{Color: "yellow"}},
			{Type: "text", Text: " "},
			{Type: "text", Text: "green", Annotations: blocks.Annotations{Color: "green"}},
			{Type: "text", Text: " "},
			{Type: "text", Text: "blue", Annotations: blocks.Annotations{Color: "blue"}},
			{Type: "text", Text: " "},
			{Type: "text", Text: "purple", Annotations: blocks.Annotations{Color: "purple"}},
			{Type: "text", Text: " "},
			{Type: "text", Text: "pink", Annotations: blocks.Annotations{Color: "pink"}},
			{Type: "text", Text: " "},
			{Type: "text", Text: "red", Annotations: blocks.Annotations{Color: "red"}},
		}}},
		{blocks.BlockParagraph, blocks.Content{RichText: []blocks.RichText{
			{Type: "text", Text: "Highlights: "},
			{Type: "text", Text: "gray bg", Annotations: blocks.Annotations{Color: "gray_background"}},
			{Type: "text", Text: " "},
			{Type: "text", Text: "yellow bg", Annotations: blocks.Annotations{Color: "yellow_background"}},
			{Type: "text", Text: " "},
			{Type: "text", Text: "green bg", Annotations: blocks.Annotations{Color: "green_background"}},
			{Type: "text", Text: " "},
			{Type: "text", Text: "blue bg", Annotations: blocks.Annotations{Color: "blue_background"}},
			{Type: "text", Text: " "},
			{Type: "text", Text: "purple bg", Annotations: blocks.Annotations{Color: "purple_background"}},
			{Type: "text", Text: " "},
			{Type: "text", Text: "pink bg", Annotations: blocks.Annotations{Color: "pink_background"}},
			{Type: "text", Text: " "},
			{Type: "text", Text: "red bg", Annotations: blocks.Annotations{Color: "red_background"}},
		}}},

		// ========================================
		// Section: Headings
		// ========================================
		{blocks.BlockHeading1, blocks.Content{RichText: rt("2. Headings")}},
		{blocks.BlockHeading1, blocks.Content{RichText: rt("Heading 1 - Main Title")}},
		{blocks.BlockHeading2, blocks.Content{RichText: rt("Heading 2 - Section Title")}},
		{blocks.BlockHeading3, blocks.Content{RichText: rt("Heading 3 - Subsection Title")}},
		{blocks.BlockParagraph, blocks.Content{RichText: rt("Regular paragraph text for comparison.")}},

		// ========================================
		// Section: Lists
		// ========================================
		{blocks.BlockHeading1, blocks.Content{RichText: rt("3. List Types")}},

		{blocks.BlockHeading2, blocks.Content{RichText: rt("Bulleted List")}},
		{blocks.BlockBulletList, blocks.Content{RichText: rt("First bullet point")}},
		{blocks.BlockBulletList, blocks.Content{RichText: rt("Second bullet point")}},
		{blocks.BlockBulletList, blocks.Content{RichText: rt("Third bullet point with longer text to test wrapping behavior")}},

		{blocks.BlockHeading2, blocks.Content{RichText: rt("Numbered List")}},
		{blocks.BlockNumberList, blocks.Content{RichText: rt("First numbered item")}},
		{blocks.BlockNumberList, blocks.Content{RichText: rt("Second numbered item")}},
		{blocks.BlockNumberList, blocks.Content{RichText: rt("Third numbered item")}},

		{blocks.BlockHeading2, blocks.Content{RichText: rt("To-Do List")}},
		{blocks.BlockTodo, blocks.Content{RichText: rt("Unchecked task"), Checked: ptrBool(false)}},
		{blocks.BlockTodo, blocks.Content{RichText: rt("Checked/completed task"), Checked: ptrBool(true)}},
		{blocks.BlockTodo, blocks.Content{RichText: rt("Another pending task"), Checked: ptrBool(false)}},

		{blocks.BlockHeading2, blocks.Content{RichText: rt("Toggle List")}},
		// Note: Nested toggles are created separately below with parent_id references

		// ========================================
		// Section: Callouts
		// ========================================
		{blocks.BlockHeading1, blocks.Content{RichText: rt("4. Callout Blocks")}},
		{blocks.BlockCallout, blocks.Content{RichText: rt("Default callout - great for tips and information"), Icon: "💡", Color: "default"}},
		{blocks.BlockCallout, blocks.Content{RichText: rt("Gray callout - subtle and neutral"), Icon: "📝", Color: "gray"}},
		{blocks.BlockCallout, blocks.Content{RichText: rt("Brown callout - warm and earthy"), Icon: "🌰", Color: "brown"}},
		{blocks.BlockCallout, blocks.Content{RichText: rt("Orange callout - attention-grabbing"), Icon: "🔶", Color: "orange"}},
		{blocks.BlockCallout, blocks.Content{RichText: rt("Yellow callout - warning or important note"), Icon: "⚠️", Color: "yellow"}},
		{blocks.BlockCallout, blocks.Content{RichText: rt("Green callout - success or positive feedback"), Icon: "✅", Color: "green"}},
		{blocks.BlockCallout, blocks.Content{RichText: rt("Blue callout - informational or tips"), Icon: "ℹ️", Color: "blue"}},
		{blocks.BlockCallout, blocks.Content{RichText: rt("Purple callout - creative or ideas"), Icon: "💜", Color: "purple"}},
		{blocks.BlockCallout, blocks.Content{RichText: rt("Pink callout - playful or feminine"), Icon: "🌸", Color: "pink"}},
		{blocks.BlockCallout, blocks.Content{RichText: rt("Red callout - error or danger"), Icon: "❌", Color: "red"}},

		// ========================================
		// Section: Quote & Divider
		// ========================================
		{blocks.BlockHeading1, blocks.Content{RichText: rt("5. Quote & Divider")}},
		{blocks.BlockQuote, blocks.Content{RichText: rt("This is a blockquote. Great for highlighting important quotes or references. It can span multiple lines and should maintain proper styling.")}},
		{blocks.BlockDivider, blocks.Content{}},
		{blocks.BlockParagraph, blocks.Content{RichText: rt("Content after the divider.")}},

		// ========================================
		// Section: Code Blocks
		// ========================================
		{blocks.BlockHeading1, blocks.Content{RichText: rt("6. Code Blocks")}},

		{blocks.BlockHeading3, blocks.Content{RichText: rt("JavaScript")}},
		{blocks.BlockCode, blocks.Content{
			RichText: []blocks.RichText{{Type: "text", Text: `function greeting(name) {
  console.log('Hello, ' + name + '!');
  return {
    message: 'Welcome',
    timestamp: new Date(),
  };
}

// Call the function
greeting('World');`}},
			Language: "javascript",
		}},

		{blocks.BlockHeading3, blocks.Content{RichText: rt("Python")}},
		{blocks.BlockCode, blocks.Content{
			RichText: []blocks.RichText{{Type: "text", Text: `def fibonacci(n):
    """Generate Fibonacci sequence up to n."""
    a, b = 0, 1
    result = []
    while a < n:
        result.append(a)
        a, b = b, a + b
    return result

# Example usage
print(fibonacci(100))`}},
			Language: "python",
		}},

		{blocks.BlockHeading3, blocks.Content{RichText: rt("Go")}},
		{blocks.BlockCode, blocks.Content{
			RichText: []blocks.RichText{{Type: "text", Text: `package main

import "fmt"

func main() {
    // Simple hello world
    message := "Hello, Go!"
    fmt.Println(message)

    // Loop example
    for i := 0; i < 5; i++ {
        fmt.Printf("Count: %d\n", i)
    }
}`}},
			Language: "go",
		}},

		{blocks.BlockHeading3, blocks.Content{RichText: rt("CSS")}},
		{blocks.BlockCode, blocks.Content{
			RichText: []blocks.RichText{{Type: "text", Text: `.notion-callout {
  display: flex;
  padding: 16px 16px 16px 12px;
  border-radius: 3px;
  background: rgba(241, 241, 239, 1);
}

.notion-callout:hover {
  background: rgba(235, 235, 233, 1);
}`}},
			Language: "css",
		}},

		{blocks.BlockHeading3, blocks.Content{RichText: rt("SQL")}},
		{blocks.BlockCode, blocks.Content{
			RichText: []blocks.RichText{{Type: "text", Text: `SELECT
    u.id,
    u.name,
    COUNT(o.id) AS order_count,
    SUM(o.total) AS total_spent
FROM users u
LEFT JOIN orders o ON u.id = o.user_id
WHERE u.created_at > '2024-01-01'
GROUP BY u.id, u.name
HAVING COUNT(o.id) > 5
ORDER BY total_spent DESC
LIMIT 10;`}},
			Language: "sql",
		}},

		{blocks.BlockHeading3, blocks.Content{RichText: rt("HTML")}},
		{blocks.BlockCode, blocks.Content{
			RichText: []blocks.RichText{{Type: "text", Text: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Sample Page</title>
</head>
<body>
    <header class="site-header">
        <h1>Welcome</h1>
    </header>
    <main>
        <p>Hello, World!</p>
    </main>
</body>
</html>`}},
			Language: "html",
		}},

		{blocks.BlockHeading3, blocks.Content{RichText: rt("JSON")}},
		{blocks.BlockCode, blocks.Content{
			RichText: []blocks.RichText{{Type: "text", Text: `{
  "name": "workspace",
  "version": "1.0.0",
  "description": "A Notion-like block editor",
  "features": ["rich-text", "databases", "collaboration"],
  "config": {
    "theme": "light",
    "autosave": true
  }
}`}},
			Language: "json",
		}},

		{blocks.BlockHeading3, blocks.Content{RichText: rt("TypeScript")}},
		{blocks.BlockCode, blocks.Content{
			RichText: []blocks.RichText{{Type: "text", Text: `interface Block {
  id: string;
  type: BlockType;
  content: RichText[];
  children?: Block[];
}

type BlockType = 'paragraph' | 'heading' | 'callout' | 'code';

const createBlock = <T extends BlockType>(
  type: T,
  content: string
): Block => ({
  id: crypto.randomUUID(),
  type,
  content: [{ text: content }],
});`}},
			Language: "typescript",
		}},

		{blocks.BlockHeading3, blocks.Content{RichText: rt("Bash / Shell")}},
		{blocks.BlockCode, blocks.Content{
			RichText: []blocks.RichText{{Type: "text", Text: `#!/bin/bash
set -e

echo "Starting deployment..."

# Build and test
npm run build
npm test

# Deploy
rsync -avz ./dist/ user@server:/var/www/app/

echo "Deployment complete!"
exit 0`}},
			Language: "bash",
		}},

		// ========================================
		// Section: Equations
		// ========================================
		{blocks.BlockHeading1, blocks.Content{RichText: rt("7. Equations (LaTeX)")}},
		{blocks.BlockParagraph, blocks.Content{RichText: rt("Mathematical equations rendered with KaTeX:")}},
		{blocks.BlockEquation, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "E = mc^2"}}}},
		{blocks.BlockEquation, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "\\frac{-b \\pm \\sqrt{b^2 - 4ac}}{2a}"}}}},
		{blocks.BlockEquation, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "\\int_{-\\infty}^{\\infty} e^{-x^2} dx = \\sqrt{\\pi}"}}}},
		{blocks.BlockEquation, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "\\sum_{n=1}^{\\infty} \\frac{1}{n^2} = \\frac{\\pi^2}{6}"}}}},
		{blocks.BlockEquation, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "\\nabla \\times \\mathbf{E} = -\\frac{\\partial \\mathbf{B}}{\\partial t}"}}}},
		{blocks.BlockEquation, blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "\\begin{bmatrix} a & b \\\\ c & d \\end{bmatrix} \\begin{bmatrix} x \\\\ y \\end{bmatrix} = \\begin{bmatrix} ax + by \\\\ cx + dy \\end{bmatrix}"}}}},

		// ========================================
		// Section: Media Blocks
		// ========================================
		{blocks.BlockHeading1, blocks.Content{RichText: rt("8. Media Blocks")}},

		{blocks.BlockHeading2, blocks.Content{RichText: rt("Image")}},
		{blocks.BlockImage, blocks.Content{
			URL:     "https://images.unsplash.com/photo-1506905925346-21bda4d32df4?w=800",
			Caption: []blocks.RichText{{Type: "text", Text: "Beautiful mountain landscape"}},
		}},

		{blocks.BlockHeading2, blocks.Content{RichText: rt("Bookmark")}},
		{blocks.BlockBookmark, blocks.Content{
			URL:         "https://github.com",
			Title:       "GitHub: Where the world builds software",
			Description: "GitHub is where over 100 million developers shape the future of software, together.",
		}},
		{blocks.BlockBookmark, blocks.Content{
			URL:         "https://notion.so",
			Title:       "Notion – The all-in-one workspace",
			Description: "A new tool that blends your everyday work apps into one. It's the all-in-one workspace for you and your team.",
		}},

		{blocks.BlockHeading2, blocks.Content{RichText: rt("Video (External)")}},
		{blocks.BlockVideo, blocks.Content{
			URL:     "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			Caption: []blocks.RichText{{Type: "text", Text: "Sample video embed"}},
		}},

		{blocks.BlockHeading2, blocks.Content{RichText: rt("File Attachments")}},
		{blocks.BlockFile, blocks.Content{
			URL:   "https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf",
			Title: "Sample PDF Document.pdf",
		}},
		{blocks.BlockFile, blocks.Content{
			URL:   "https://sample-videos.com/csv/Sample-Spreadsheet-10-rows.csv",
			Title: "Data Export.csv",
		}},

		{blocks.BlockHeading2, blocks.Content{RichText: rt("Embed (External Content)")}},
		{blocks.BlockEmbed, blocks.Content{
			URL:     "https://codepen.io/pen/embed",
			Caption: []blocks.RichText{{Type: "text", Text: "Embedded external content"}},
		}},

		// ========================================
		// Section: Table
		// ========================================
		{blocks.BlockHeading1, blocks.Content{RichText: rt("9. Tables")}},
		{blocks.BlockTable, blocks.Content{
			TableWidth: 3,
			HasHeader:  true,
		}},

		// ========================================
		// Section: Advanced Blocks
		// ========================================
		{blocks.BlockHeading1, blocks.Content{RichText: rt("10. Advanced Blocks")}},

		{blocks.BlockHeading2, blocks.Content{RichText: rt("Synced Block (Mock)")}},
		{blocks.BlockSyncedBlock, blocks.Content{
			SyncedFrom: "original-block-id",
			RichText:   rt("This is synced content that appears in multiple places."),
		}},

		{blocks.BlockHeading2, blocks.Content{RichText: rt("Linked Database (Mock)")}},
		{blocks.BlockLinkedDB, blocks.Content{
			DatabaseID: "sample-database-id",
		}},

		{blocks.BlockHeading2, blocks.Content{RichText: rt("Breadcrumb")}},
		{blocks.BlockBreadcrumb, blocks.Content{}},

		{blocks.BlockHeading2, blocks.Content{RichText: rt("Template Button")}},
		{blocks.BlockTemplateButton, blocks.Content{
			ButtonText:  "Add New Task",
			ButtonStyle: "primary",
		}},

		// ========================================
		// Section: Column Layout
		// ========================================
		{blocks.BlockHeading1, blocks.Content{RichText: rt("11. Column Layout")}},
		{blocks.BlockParagraph, blocks.Content{RichText: rt("Multi-column layouts allow side-by-side content arrangement. Use the / menu to insert columns.")}},
		{blocks.BlockColumnList, blocks.Content{
			ColumnCount: 2,
		}},

		// ========================================
		// Section: Child Pages
		// ========================================
		{blocks.BlockHeading1, blocks.Content{RichText: rt("12. Child Pages & Databases")}},
		{blocks.BlockChildPage, blocks.Content{
			Title: "Sub-page Example",
			Icon:  "📄",
		}},
		{blocks.BlockChildDB, blocks.Content{
			Title: "Inline Database Example",
		}},

		// ========================================
		// Section: Summary
		// ========================================
		{blocks.BlockDivider, blocks.Content{}},
		{blocks.BlockHeading1, blocks.Content{RichText: rt("Summary")}},
		{blocks.BlockCallout, blocks.Content{
			RichText: []blocks.RichText{
				{Type: "text", Text: "This page includes ", Annotations: blocks.Annotations{}},
				{Type: "text", Text: "30+ block types", Annotations: blocks.Annotations{Bold: true}},
				{Type: "text", Text: " demonstrating the full capabilities of the block editor:"},
			},
			Icon:  "🎯",
			Color: "blue",
		}},
		{blocks.BlockBulletList, blocks.Content{RichText: []blocks.RichText{
			{Type: "text", Text: "Text formatting", Annotations: blocks.Annotations{Bold: true}},
			{Type: "text", Text: " - Bold, italic, colors, highlights, inline code"},
		}}},
		{blocks.BlockBulletList, blocks.Content{RichText: []blocks.RichText{
			{Type: "text", Text: "Lists", Annotations: blocks.Annotations{Bold: true}},
			{Type: "text", Text: " - Bulleted, numbered, to-do, toggle"},
		}}},
		{blocks.BlockBulletList, blocks.Content{RichText: []blocks.RichText{
			{Type: "text", Text: "Callouts", Annotations: blocks.Annotations{Bold: true}},
			{Type: "text", Text: " - All 10 color variants with icons"},
		}}},
		{blocks.BlockBulletList, blocks.Content{RichText: []blocks.RichText{
			{Type: "text", Text: "Code blocks", Annotations: blocks.Annotations{Bold: true}},
			{Type: "text", Text: " - JS, Python, Go, CSS, SQL, HTML, JSON, TS, Bash"},
		}}},
		{blocks.BlockBulletList, blocks.Content{RichText: []blocks.RichText{
			{Type: "text", Text: "Media", Annotations: blocks.Annotations{Bold: true}},
			{Type: "text", Text: " - Images, videos, files, bookmarks, embeds"},
		}}},
		{blocks.BlockBulletList, blocks.Content{RichText: []blocks.RichText{
			{Type: "text", Text: "Advanced", Annotations: blocks.Annotations{Bold: true}},
			{Type: "text", Text: " - Equations, tables, synced blocks, linked databases"},
		}}},
		{blocks.BlockDivider, blocks.Content{}},
		{blocks.BlockParagraph, blocks.Content{RichText: []blocks.RichText{
			{Type: "text", Text: "Use this page to test drag-and-drop, formatting, and visual consistency with Notion.", Annotations: blocks.Annotations{Italic: true, Color: "gray"}},
		}}},
	}

	// Create all blocks
	for i, bc := range testBlocks {
		blockSvc.Create(ctx, &blocks.CreateIn{
			PageID:    pageID,
			Type:      bc.Type,
			Content:   bc.Content,
			Position:  i,
			CreatedBy: userID,
		})
	}

	// Create nested toggle examples
	createNestedToggles(ctx, blockSvc, pageID, userID, len(testBlocks))
}

// createNestedToggles creates toggle blocks with nested children to demonstrate the feature
func createNestedToggles(ctx context.Context, blockSvc blocks.API, pageID, userID string, startPos int) {
	// Create first toggle with nested content
	toggle1, _ := blockSvc.Create(ctx, &blocks.CreateIn{
		PageID:    pageID,
		Type:      blocks.BlockToggle,
		Content:   blocks.Content{RichText: rt("Getting Started Guide")},
		Position:  startPos,
		CreatedBy: userID,
	})
	if toggle1 != nil {
		// Add children to the first toggle
		blockSvc.Create(ctx, &blocks.CreateIn{
			PageID:    pageID,
			ParentID:  toggle1.ID,
			Type:      blocks.BlockParagraph,
			Content:   blocks.Content{RichText: rt("Welcome! This toggle contains helpful information to get you started.")},
			Position:  0,
			CreatedBy: userID,
		})
		blockSvc.Create(ctx, &blocks.CreateIn{
			PageID:    pageID,
			ParentID:  toggle1.ID,
			Type:      blocks.BlockBulletList,
			Content:   blocks.Content{RichText: rt("Step 1: Create a new page")},
			Position:  1,
			CreatedBy: userID,
		})
		blockSvc.Create(ctx, &blocks.CreateIn{
			PageID:    pageID,
			ParentID:  toggle1.ID,
			Type:      blocks.BlockBulletList,
			Content:   blocks.Content{RichText: rt("Step 2: Add some blocks")},
			Position:  2,
			CreatedBy: userID,
		})
		blockSvc.Create(ctx, &blocks.CreateIn{
			PageID:    pageID,
			ParentID:  toggle1.ID,
			Type:      blocks.BlockBulletList,
			Content:   blocks.Content{RichText: rt("Step 3: Share with your team")},
			Position:  3,
			CreatedBy: userID,
		})
	}

	// Create second toggle with nested toggle (deeply nested)
	toggle2, _ := blockSvc.Create(ctx, &blocks.CreateIn{
		PageID:    pageID,
		Type:      blocks.BlockToggle,
		Content:   blocks.Content{RichText: rt("Advanced Features")},
		Position:  startPos + 1,
		CreatedBy: userID,
	})
	if toggle2 != nil {
		// Add a paragraph and a nested toggle
		blockSvc.Create(ctx, &blocks.CreateIn{
			PageID:    pageID,
			ParentID:  toggle2.ID,
			Type:      blocks.BlockParagraph,
			Content:   blocks.Content{RichText: rt("Explore these advanced capabilities:")},
			Position:  0,
			CreatedBy: userID,
		})

		// Nested toggle inside toggle2
		nestedToggle, _ := blockSvc.Create(ctx, &blocks.CreateIn{
			PageID:    pageID,
			ParentID:  toggle2.ID,
			Type:      blocks.BlockToggle,
			Content:   blocks.Content{RichText: rt("Keyboard Shortcuts")},
			Position:  1,
			CreatedBy: userID,
		})
		if nestedToggle != nil {
			blockSvc.Create(ctx, &blocks.CreateIn{
				PageID:    pageID,
				ParentID:  nestedToggle.ID,
				Type:      blocks.BlockParagraph,
				Content:   blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Cmd/Ctrl + B", Annotations: blocks.Annotations{Code: true}}, {Type: "text", Text: " - Bold text"}}},
				Position:  0,
				CreatedBy: userID,
			})
			blockSvc.Create(ctx, &blocks.CreateIn{
				PageID:    pageID,
				ParentID:  nestedToggle.ID,
				Type:      blocks.BlockParagraph,
				Content:   blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Cmd/Ctrl + I", Annotations: blocks.Annotations{Code: true}}, {Type: "text", Text: " - Italic text"}}},
				Position:  1,
				CreatedBy: userID,
			})
			blockSvc.Create(ctx, &blocks.CreateIn{
				PageID:    pageID,
				ParentID:  nestedToggle.ID,
				Type:      blocks.BlockParagraph,
				Content:   blocks.Content{RichText: []blocks.RichText{{Type: "text", Text: "Cmd/Ctrl + /", Annotations: blocks.Annotations{Code: true}}, {Type: "text", Text: " - Open slash menu"}}},
				Position:  2,
				CreatedBy: userID,
			})
		}

		// Another nested toggle
		nestedToggle2, _ := blockSvc.Create(ctx, &blocks.CreateIn{
			PageID:    pageID,
			ParentID:  toggle2.ID,
			Type:      blocks.BlockToggle,
			Content:   blocks.Content{RichText: rt("Database Features")},
			Position:  2,
			CreatedBy: userID,
		})
		if nestedToggle2 != nil {
			blockSvc.Create(ctx, &blocks.CreateIn{
				PageID:    pageID,
				ParentID:  nestedToggle2.ID,
				Type:      blocks.BlockBulletList,
				Content:   blocks.Content{RichText: rt("Create tables, boards, and calendars")},
				Position:  0,
				CreatedBy: userID,
			})
			blockSvc.Create(ctx, &blocks.CreateIn{
				PageID:    pageID,
				ParentID:  nestedToggle2.ID,
				Type:      blocks.BlockBulletList,
				Content:   blocks.Content{RichText: rt("Filter and sort your data")},
				Position:  1,
				CreatedBy: userID,
			})
			blockSvc.Create(ctx, &blocks.CreateIn{
				PageID:    pageID,
				ParentID:  nestedToggle2.ID,
				Type:      blocks.BlockBulletList,
				Content:   blocks.Content{RichText: rt("Link databases together")},
				Position:  2,
				CreatedBy: userID,
			})
		}
	}

	// Create third toggle - simple with callout
	toggle3, _ := blockSvc.Create(ctx, &blocks.CreateIn{
		PageID:    pageID,
		Type:      blocks.BlockToggle,
		Content:   blocks.Content{RichText: rt("Pro Tips")},
		Position:  startPos + 2,
		CreatedBy: userID,
	})
	if toggle3 != nil {
		blockSvc.Create(ctx, &blocks.CreateIn{
			PageID:    pageID,
			ParentID:  toggle3.ID,
			Type:      blocks.BlockCallout,
			Content:   blocks.Content{RichText: rt("Use toggle lists to organize FAQ sections, documentation, or any content that benefits from progressive disclosure."), Icon: "💡", Color: "blue"},
			Position:  0,
			CreatedBy: userID,
		})
	}
}

// rt is a helper to create simple rich text
func rt(text string) []blocks.RichText {
	return []blocks.RichText{{Type: "text", Text: text}}
}
