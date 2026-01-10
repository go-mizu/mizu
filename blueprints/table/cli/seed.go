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
	ratingField := createField("Confidence", "rating", map[string]any{"max": 5})
	tagsField := createField("Tags", "multi_select", map[string]any{
		"choices": []map[string]any{
			{"id": "tag-1", "name": "Frontend", "color": "#3B82F6"},
			{"id": "tag-2", "name": "Backend", "color": "#10B981"},
			{"id": "tag-3", "name": "Design", "color": "#EC4899"},
			{"id": "tag-4", "name": "Bug", "color": "#EF4444"},
			{"id": "tag-5", "name": "Feature", "color": "#8B5CF6"},
			{"id": "tag-6", "name": "Docs", "color": "#F59E0B"},
		},
	})
	budgetField := createField("Budget", "currency", nil)
	progressField := createField("Progress", "percent", nil)
	emailField := createField("Contact Email", "email", nil)
	urlField := createField("Reference URL", "url", nil)
	thumbnailField := createField("Thumbnail", "attachment", nil)
	Step("", "Fields ready", time.Since(sectionStart))

	// Create sample records with realistic data
	sectionStart = time.Now()
	fmt.Print("  Preparing records...\n")
	tasks := []struct {
		Name      string
		Status    string
		Priority  string
		StartDays int
		DueDays   int
		Notes     string
		Progress  int
		Budget    float64
		Rating    int
		Tags      []string
		Email     string
		URL       string
		Thumbnail string
	}{
		// Currently In Progress Tasks with overlapping dates for timeline stacking demo
		{
			Name:      "Redesign Homepage Hero Section",
			Status:    "status-2",
			Priority:  "priority-3",
			StartDays: -5,
			DueDays:   3,
			Notes:     "Create a modern, eye-catching hero section with animated gradient background and clear CTA buttons. Use Figma for mockups.",
			Progress:  65,
			Budget:    5000,
			Rating:    4,
			Tags:      []string{"tag-1", "tag-3"},
			Email:     "design@example.com",
			URL:       "https://figma.com/file/hero-design",
			Thumbnail: "https://images.unsplash.com/photo-1581291518857-4e27b48ff24e?w=800&q=80",
		},
		{
			Name:      "Landing Page A/B Testing",
			Status:    "status-2",
			Priority:  "priority-2",
			StartDays: -4,
			DueDays:   4,
			Notes:     "Set up A/B tests for the new landing page variants. Test different headlines, CTA colors, and hero images.",
			Progress:  40,
			Budget:    2000,
			Rating:    4,
			Tags:      []string{"tag-1", "tag-5"},
			Email:     "growth@example.com",
			URL:       "https://optimizely.com",
			Thumbnail: "https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=800&q=80",
		},
		{
			Name:      "Database Schema Migration v2",
			Status:    "status-2",
			Priority:  "priority-3",
			StartDays: -2,
			DueDays:   4,
			Notes:     "Migrate from legacy schema to new normalized structure. Add proper indexes, foreign keys, and update ORM models. Zero downtime migration.",
			Progress:  70,
			Budget:    4000,
			Rating:    4,
			Tags:      []string{"tag-2"},
			Email:     "database@example.com",
			URL:       "https://dbdiagram.io/d/schema-v2",
			Thumbnail: "https://images.unsplash.com/photo-1558494949-ef010cbdcc31?w=800&q=80",
		},
		{
			Name:      "Performance Optimization - Initial Load",
			Status:    "status-2",
			Priority:  "priority-4",
			StartDays: -3,
			DueDays:   5,
			Notes:     "Reduce initial page load time from 4.2s to under 2s. Focus on code splitting, lazy loading, and image optimization.",
			Progress:  45,
			Budget:    6000,
			Rating:    5,
			Tags:      []string{"tag-1", "tag-2"},
			Email:     "perf@example.com",
			URL:       "https://web.dev/performance",
			Thumbnail: "https://images.unsplash.com/photo-1504868584819-f8e8b4b6d7e3?w=800&q=80",
		},
		{
			Name:      "Customer Onboarding Flow Redesign",
			Status:    "status-2",
			Priority:  "priority-3",
			StartDays: -7,
			DueDays:   6,
			Notes:     "Streamline the signup process. Add progress indicator, reduce form fields, implement social login, and create welcome email sequence.",
			Progress:  55,
			Budget:    7500,
			Rating:    4,
			Tags:      []string{"tag-1", "tag-3", "tag-5"},
			Email:     "growth@example.com",
			URL:       "https://hotjar.com/recordings/onboarding",
			Thumbnail: "https://images.unsplash.com/photo-1553484771-371a605b060b?w=800&q=80",
		},
		{
			Name:      "Accessibility Compliance (WCAG 2.1)",
			Status:    "status-2",
			Priority:  "priority-2",
			StartDays: -1,
			DueDays:   9,
			Notes:     "Audit and fix accessibility issues. Add ARIA labels, keyboard navigation, color contrast fixes, and screen reader support.",
			Progress:  35,
			Budget:    5500,
			Rating:    4,
			Tags:      []string{"tag-1"},
			Email:     "a11y@example.com",
			URL:       "https://www.w3.org/WAI/WCAG21/quickref",
			Thumbnail: "https://images.unsplash.com/photo-1573164713988-8665fc963095?w=800&q=80",
		},

		// Under Review Tasks
		{
			Name:      "Mobile Responsive Breakpoint Fixes",
			Status:    "status-3",
			Priority:  "priority-2",
			StartDays: -4,
			DueDays:   2,
			Notes:     "Fix layout issues on tablets (768-1024px). Navigation menu collapse, card grid adjustments, and form input sizing.",
			Progress:  90,
			Budget:    2000,
			Rating:    4,
			Tags:      []string{"tag-1", "tag-3"},
			Email:     "mobile@example.com",
			URL:       "https://responsively.app",
			Thumbnail: "https://images.unsplash.com/photo-1512941937669-90a1b58e7e9c?w=800&q=80",
		},
		{
			Name:      "Code Review Sprint - Q1 PRs",
			Status:    "status-3",
			Priority:  "priority-2",
			StartDays: -1,
			DueDays:   1,
			Notes:     "Review pending pull requests from the team. Focus on code quality, test coverage, and performance implications. 8 PRs in queue.",
			Progress:  80,
			Budget:    0,
			Rating:    4,
			Tags:      []string{"tag-1", "tag-2"},
			Email:     "reviews@example.com",
			URL:       "https://github.com/example/pulls",
			Thumbnail: "https://images.unsplash.com/photo-1522071820081-009f0129c71c?w=800&q=80",
		},

		// Completed Tasks (past dates)
		{
			Name:      "Fix Authentication Session Bug",
			Status:    "status-4",
			Priority:  "priority-3",
			StartDays: -10,
			DueDays:   -2,
			Notes:     "Critical bug in session handling causing intermittent logouts. Root cause: race condition in token refresh. Fixed with mutex lock.",
			Progress:  100,
			Budget:    1500,
			Rating:    5,
			Tags:      []string{"tag-2", "tag-4"},
			Email:     "security@example.com",
			URL:       "https://github.com/example/issues/142",
			Thumbnail: "https://images.unsplash.com/photo-1555066931-4365d14bab8c?w=800&q=80",
		},
		{
			Name:      "Email Template System",
			Status:    "status-4",
			Priority:  "priority-2",
			StartDays: -14,
			DueDays:   -5,
			Notes:     "Created reusable email template system with MJML. Includes welcome, password reset, invoice, and notification templates.",
			Progress:  100,
			Budget:    4500,
			Rating:    5,
			Tags:      []string{"tag-1", "tag-5"},
			Email:     "comms@example.com",
			URL:       "https://mjml.io",
			Thumbnail: "https://images.unsplash.com/photo-1596526131083-e8c633c948d2?w=800&q=80",
		},
		{
			Name:      "CI/CD Pipeline Setup",
			Status:    "status-4",
			Priority:  "priority-3",
			StartDays: -21,
			DueDays:   -14,
			Notes:     "Set up GitHub Actions for automated testing, linting, and deployment. Added staging and production environments.",
			Progress:  100,
			Budget:    3000,
			Rating:    5,
			Tags:      []string{"tag-2"},
			Email:     "devops@example.com",
			URL:       "https://github.com/features/actions",
			Thumbnail: "https://images.unsplash.com/photo-1618401471353-b98afee0b2eb?w=800&q=80",
		},
		{
			Name:      "User Research Interviews",
			Status:    "status-4",
			Priority:  "priority-2",
			StartDays: -18,
			DueDays:   -12,
			Notes:     "Conducted 15 user interviews to understand pain points. Synthesized findings into actionable insights for the product team.",
			Progress:  100,
			Budget:    2500,
			Rating:    4,
			Tags:      []string{"tag-3"},
			Email:     "research@example.com",
			URL:       "https://notion.so/user-research",
			Thumbnail: "https://images.unsplash.com/photo-1552664730-d307ca884978?w=800&q=80",
		},

		// Not Started / Upcoming Tasks (future dates)
		{
			Name:      "API Documentation Overhaul",
			Status:    "status-1",
			Priority:  "priority-2",
			StartDays: 0,
			DueDays:   7,
			Notes:     "Comprehensive API documentation with interactive examples using OpenAPI/Swagger. Include authentication flows and rate limiting info.",
			Progress:  15,
			Budget:    3000,
			Rating:    3,
			Tags:      []string{"tag-2", "tag-6"},
			Email:     "docs@example.com",
			URL:       "https://docs.example.com/api",
			Thumbnail: "https://images.unsplash.com/photo-1456406644174-8ddd4cd52a06?w=800&q=80",
		},
		{
			Name:      "Security Audit & Penetration Testing",
			Status:    "status-1",
			Priority:  "priority-4",
			StartDays: 1,
			DueDays:   8,
			Notes:     "Comprehensive security audit including OWASP Top 10 review, dependency scanning, and penetration testing with external vendor.",
			Progress:  0,
			Budget:    15000,
			Rating:    5,
			Tags:      []string{"tag-2"},
			Email:     "security@example.com",
			URL:       "https://owasp.org/Top10",
			Thumbnail: "https://images.unsplash.com/photo-1563986768609-322da13575f3?w=800&q=80",
		},
		{
			Name:      "User Preferences Dashboard",
			Status:    "status-1",
			Priority:  "priority-2",
			StartDays: 3,
			DueDays:   10,
			Notes:     "New settings page with dark/light theme toggle, notification preferences, language selection, and data export options.",
			Progress:  10,
			Budget:    8000,
			Rating:    4,
			Tags:      []string{"tag-1", "tag-5"},
			Email:     "product@example.com",
			URL:       "https://linear.app/example/issue/ENG-234",
			Thumbnail: "https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=800&q=80",
		},
		{
			Name:      "API Rate Limiting Implementation",
			Status:    "status-1",
			Priority:  "priority-2",
			StartDays: 5,
			DueDays:   12,
			Notes:     "Implement sliding window rate limiting with Redis. Configure limits per endpoint and user tier. Add rate limit headers to responses.",
			Progress:  0,
			Budget:    3500,
			Rating:    3,
			Tags:      []string{"tag-2", "tag-5"},
			Email:     "api@example.com",
			URL:       "https://redis.io/topics/rate-limiting",
			Thumbnail: "https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=800&q=80",
		},
		{
			Name:      "Dependency Security Updates",
			Status:    "status-1",
			Priority:  "priority-1",
			StartDays: 7,
			DueDays:   14,
			Notes:     "Update all npm packages to latest versions. Run security audit with npm audit and fix vulnerabilities. Update Go modules.",
			Progress:  0,
			Budget:    500,
			Rating:    2,
			Tags:      []string{"tag-2"},
			Email:     "devops@example.com",
			URL:       "https://snyk.io/dashboard",
			Thumbnail: "https://images.unsplash.com/photo-1558494949-ef010cbdcc31?w=800&q=80",
		},
		{
			Name:      "Unit Test Coverage Push to 80%",
			Status:    "status-1",
			Priority:  "priority-2",
			StartDays: 7,
			DueDays:   14,
			Notes:     "Current coverage at 62%. Need to add tests for auth module, API handlers, and utility functions. Set up coverage reports in CI.",
			Progress:  5,
			Budget:    2500,
			Rating:    3,
			Tags:      []string{"tag-2"},
			Email:     "qa@example.com",
			URL:       "https://codecov.io/gh/example",
			Thumbnail: "https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=800&q=80",
		},
		{
			Name:      "Analytics Dashboard Implementation",
			Status:    "status-1",
			Priority:  "priority-2",
			StartDays: 10,
			DueDays:   21,
			Notes:     "Build real-time analytics dashboard with charts for user engagement, revenue metrics, and conversion funnels. Use Chart.js and WebSockets.",
			Progress:  0,
			Budget:    12000,
			Rating:    4,
			Tags:      []string{"tag-1", "tag-2", "tag-5"},
			Email:     "analytics@example.com",
			URL:       "https://mixpanel.com",
			Thumbnail: "https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=800&q=80",
		},
		{
			Name:      "Mobile App Beta Launch",
			Status:    "status-1",
			Priority:  "priority-3",
			StartDays: 14,
			DueDays:   28,
			Notes:     "Prepare and launch mobile app beta on TestFlight and Google Play internal testing. Set up crash reporting and analytics.",
			Progress:  0,
			Budget:    8000,
			Rating:    4,
			Tags:      []string{"tag-1", "tag-5"},
			Email:     "mobile@example.com",
			URL:       "https://testflight.apple.com",
			Thumbnail: "https://images.unsplash.com/photo-1512941937669-90a1b58e7e9c?w=800&q=80",
		},
		{
			Name:      "Internationalization (i18n) Support",
			Status:    "status-1",
			Priority:  "priority-2",
			StartDays: 14,
			DueDays:   30,
			Notes:     "Add multi-language support starting with Spanish, French, and German. Extract all strings, set up translation workflow.",
			Progress:  0,
			Budget:    6000,
			Rating:    3,
			Tags:      []string{"tag-1", "tag-5"},
			Email:     "i18n@example.com",
			URL:       "https://crowdin.com",
			Thumbnail: "https://images.unsplash.com/photo-1516321497487-e288fb19713f?w=800&q=80",
		},
		{
			Name:      "Payment System Integration",
			Status:    "status-1",
			Priority:  "priority-3",
			StartDays: 21,
			DueDays:   35,
			Notes:     "Integrate Stripe for subscription billing. Support monthly and annual plans, promo codes, and invoice generation.",
			Progress:  0,
			Budget:    10000,
			Rating:    5,
			Tags:      []string{"tag-2", "tag-5"},
			Email:     "billing@example.com",
			URL:       "https://stripe.com/docs",
			Thumbnail: "https://images.unsplash.com/photo-1556742049-0cfed4f6a45d?w=800&q=80",
		},
		{
			Name:      "Video Tutorial Series",
			Status:    "status-1",
			Priority:  "priority-1",
			StartDays: 25,
			DueDays:   45,
			Notes:     "Create 10 video tutorials covering core features, advanced workflows, and integrations. Publish on YouTube and in-app.",
			Progress:  0,
			Budget:    5000,
			Rating:    3,
			Tags:      []string{"tag-6"},
			Email:     "content@example.com",
			URL:       "https://www.youtube.com",
			Thumbnail: "https://images.unsplash.com/photo-1492619375914-88005aa9e8fb?w=800&q=80",
		},
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
		if progressField != nil {
			cells[progressField.ID] = task.Progress
		}
		if budgetField != nil && task.Budget > 0 {
			cells[budgetField.ID] = task.Budget
		}
		if ratingField != nil {
			cells[ratingField.ID] = task.Rating
		}
		if tagsField != nil && len(task.Tags) > 0 {
			cells[tagsField.ID] = task.Tags
		}
		if emailField != nil {
			cells[emailField.ID] = task.Email
		}
		if urlField != nil {
			cells[urlField.ID] = task.URL
		}
		if thumbnailField != nil && task.Thumbnail != "" {
			// Create attachment structure for the thumbnail
			cells[thumbnailField.ID] = []map[string]any{
				{
					"id":        fmt.Sprintf("thumb-%d", len(recordsData)+1),
					"filename":  "thumbnail.jpg",
					"url":       task.Thumbnail,
					"mime_type": "image/jpeg",
					"size":      50000,
				},
			}
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
		{"Gallery", "gallery", map[string]any{"cover_field_id": thumbnailField.ID}},
		{"Timeline", "timeline", map[string]any{"dateField": startDateField.ID, "endDateField": dueDateField.ID}},
		{"Submit Task", "form", map[string]any{
			"title":                       "Submit a New Task",
			"description":                 "Use this form to submit a new task to the project tracker. All submissions will be reviewed by the team.",
			"submit_button_text":          "Submit Task",
			"success_message":             "Thank you for your submission! Your task has been added to the tracker and will be reviewed shortly.",
			"theme_color":                 "#2563eb",
			"cover_image_url":             "https://images.unsplash.com/photo-1522071820081-009f0129c71c?w=1200&q=80",
			"show_branding":               true,
			"allow_multiple_submissions":  true,
			"is_public":                   true,
		}},
		{"Task List", "list", nil},
	}

	viewCount := 0
	for _, vt := range viewTypes {
		stepStart = time.Now()
		fmt.Printf("    • %s (%s)... ", vt.Name, vt.Type)

		view, err := srv.ViewService().Create(ctx, ownerUserID, views.CreateIn{
			TableID: table.ID,
			Name:    vt.Name,
			Type:    vt.Type,
		})
		if err != nil {
			fmt.Printf("✗ %v\n", err)
		} else {
			// Set view config if provided
			if vt.Config != nil && view != nil {
				srv.ViewService().SetConfig(ctx, view.ID, vt.Config)
			}
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
		projDescField := createFieldForTable(srv, ctx, projectsTable.ID, "Description", "long_text", nil, ownerUserID)
		projStatusField := createFieldForTable(srv, ctx, projectsTable.ID, "Status", "single_select", map[string]any{
			"choices": []map[string]any{
				{"id": "proj-1", "name": "Planning", "color": "#6B7280"},
				{"id": "proj-2", "name": "Active", "color": "#3B82F6"},
				{"id": "proj-3", "name": "Completed", "color": "#10B981"},
				{"id": "proj-4", "name": "On Hold", "color": "#F59E0B"},
			},
		}, ownerUserID)
		projStartField := createFieldForTable(srv, ctx, projectsTable.ID, "Start Date", "date", nil, ownerUserID)
		projEndField := createFieldForTable(srv, ctx, projectsTable.ID, "End Date", "date", nil, ownerUserID)
		projBudgetField := createFieldForTable(srv, ctx, projectsTable.ID, "Budget", "currency", nil, ownerUserID)
		projProgressField := createFieldForTable(srv, ctx, projectsTable.ID, "Progress", "percent", nil, ownerUserID)
		projOwnerField := createFieldForTable(srv, ctx, projectsTable.ID, "Owner", "collaborator", nil, ownerUserID)
		projCoverField := createFieldForTable(srv, ctx, projectsTable.ID, "Cover Image", "attachment", nil, ownerUserID)
		Step("", "Project fields ready", time.Since(fieldsStart))

		// Add sample projects with full data
		projectStart := time.Now()
		fmt.Print("  Preparing project records...\n")
		projects := []struct {
			Name        string
			Description string
			Status      string
			StartDays   int
			EndDays     int
			Budget      float64
			Progress    int
			Cover       string
		}{
			{
				Name:        "Website Redesign",
				Description: "Complete overhaul of the company website with modern design, improved UX, and performance optimizations. Includes homepage, product pages, and checkout flow.",
				Status:      "proj-2",
				StartDays:   -14,
				EndDays:     30,
				Budget:      75000,
				Progress:    45,
				Cover:       "https://images.unsplash.com/photo-1467232004584-a241de8bcf5d?w=800&q=80",
			},
			{
				Name:        "Mobile App Launch",
				Description: "Native iOS and Android app development. Features include push notifications, offline mode, biometric auth, and sync with web platform.",
				Status:      "proj-2",
				StartDays:   -7,
				EndDays:     60,
				Budget:      120000,
				Progress:    25,
				Cover:       "https://images.unsplash.com/photo-1512941937669-90a1b58e7e9c?w=800&q=80",
			},
			{
				Name:        "API v2 Development",
				Description: "Next generation REST and GraphQL API with improved rate limiting, webhooks, and developer documentation portal.",
				Status:      "proj-1",
				StartDays:   14,
				EndDays:     90,
				Budget:      50000,
				Progress:    5,
				Cover:       "https://images.unsplash.com/photo-1558494949-ef010cbdcc31?w=800&q=80",
			},
			{
				Name:        "Marketing Campaign Q1",
				Description: "Multi-channel marketing campaign including social media, email, content marketing, and paid advertising for product launch.",
				Status:      "proj-3",
				StartDays:   -45,
				EndDays:     -15,
				Budget:      35000,
				Progress:    100,
				Cover:       "https://images.unsplash.com/photo-1533750349088-cd871a92f312?w=800&q=80",
			},
			{
				Name:        "Infrastructure Migration",
				Description: "Migrate from legacy infrastructure to Kubernetes on AWS. Includes CI/CD pipeline setup, monitoring, and auto-scaling.",
				Status:      "proj-4",
				StartDays:   -21,
				EndDays:     45,
				Budget:      80000,
				Progress:    35,
				Cover:       "https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=800&q=80",
			},
			{
				Name:        "Customer Portal",
				Description: "Self-service customer portal with account management, billing history, support tickets, and knowledge base.",
				Status:      "proj-2",
				StartDays:   0,
				EndDays:     45,
				Budget:      55000,
				Progress:    15,
				Cover:       "https://images.unsplash.com/photo-1553484771-371a605b060b?w=800&q=80",
			},
		}

		var projectRecords []map[string]any
		for _, proj := range projects {
			fmt.Printf("    • %s\n", proj.Name)
			cells := make(map[string]any)
			if projNameField != nil {
				cells[projNameField.ID] = proj.Name
			}
			if projDescField != nil {
				cells[projDescField.ID] = proj.Description
			}
			if projStatusField != nil {
				cells[projStatusField.ID] = proj.Status
			}
			if projStartField != nil {
				cells[projStartField.ID] = time.Now().AddDate(0, 0, proj.StartDays).Format("2006-01-02")
			}
			if projEndField != nil {
				cells[projEndField.ID] = time.Now().AddDate(0, 0, proj.EndDays).Format("2006-01-02")
			}
			if projBudgetField != nil {
				cells[projBudgetField.ID] = proj.Budget
			}
			if projProgressField != nil {
				cells[projProgressField.ID] = proj.Progress
			}
			if projOwnerField != nil {
				cells[projOwnerField.ID] = ownerUserID
			}
			if projCoverField != nil && proj.Cover != "" {
				cells[projCoverField.ID] = []map[string]any{
					{
						"id":        fmt.Sprintf("cover-%s", proj.Name),
						"filename":  "cover.jpg",
						"url":       proj.Cover,
						"mime_type": "image/jpeg",
						"size":      80000,
					},
				}
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
		"Tables", fmt.Sprintf("Tasks (%d records, 16 fields with images), Projects (6 records, 10 fields)", recordCount),
		"Views", fmt.Sprintf("%d views (Grid, Kanban x2, Calendar, Gallery, Timeline, Form, List)", viewCount),
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
