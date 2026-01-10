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

	// Additional field types demonstrating all capabilities
	completedField := createField("Completed", "checkbox", nil)
	effortField := createField("Effort (pts)", "number", map[string]any{"precision": 0})
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
	budgetField := createField("Budget", "currency", map[string]any{
		"currency_symbol": "$",
		"precision":       2,
	})
	progressField := createField("Progress", "percent", nil)
	emailField := createField("Contact Email", "email", nil)
	urlField := createField("Reference URL", "url", nil)
	thumbnailField := createField("Thumbnail", "attachment", nil)

	// New field types for comprehensive demo
	phoneField := createField("Phone", "phone", nil)
	durationField := createField("Time Spent", "duration", map[string]any{"format": "h:mm"})
	barcodeField := createField("Asset ID", "barcode", map[string]any{"barcode_type": "CODE128"})
	// Button field - displays a clickable button, no data value needed
	_ = createField("View Docs", "button", map[string]any{
		"label": "Open Docs",
		"url":   "https://docs.example.com",
		"color": "#2563eb",
	})
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
		Phone     string
		Duration  int // in seconds
		Barcode   string
		Completed bool
		Effort    int
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
			Phone:     "+1 (415) 555-0101",
			Duration:  14400, // 4 hours
			Barcode:   "TASK-2024-001",
			Completed: false,
			Effort:    8,
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
			Phone:     "+1 (415) 555-0102",
			Duration:  7200, // 2 hours
			Barcode:   "TASK-2024-002",
			Completed: false,
			Effort:    5,
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
			Phone:     "+1 (415) 555-0103",
			Duration:  28800, // 8 hours
			Barcode:   "TASK-2024-003",
			Completed: false,
			Effort:    13,
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
			Phone:     "+1 (415) 555-0104",
			Duration:  21600, // 6 hours
			Barcode:   "TASK-2024-004",
			Completed: false,
			Effort:    13,
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
			Phone:     "+1 (415) 555-0105",
			Duration:  36000, // 10 hours
			Barcode:   "TASK-2024-005",
			Completed: false,
			Effort:    21,
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
			Phone:     "+1 (415) 555-0106",
			Duration:  10800, // 3 hours
			Barcode:   "TASK-2024-006",
			Completed: false,
			Effort:    8,
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
			Phone:     "+1 (415) 555-0107",
			Duration:  18000, // 5 hours
			Barcode:   "TASK-2024-007",
			Completed: false,
			Effort:    5,
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
			Phone:     "+1 (415) 555-0108",
			Duration:  14400, // 4 hours
			Barcode:   "TASK-2024-008",
			Completed: false,
			Effort:    3,
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
			Phone:     "+1 (415) 555-0109",
			Duration:  32400, // 9 hours
			Barcode:   "TASK-2024-009",
			Completed: true,
			Effort:    8,
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
			Phone:     "+1 (415) 555-0110",
			Duration:  43200, // 12 hours
			Barcode:   "TASK-2024-010",
			Completed: true,
			Effort:    13,
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
			Phone:     "+1 (415) 555-0111",
			Duration:  28800, // 8 hours
			Barcode:   "TASK-2024-011",
			Completed: true,
			Effort:    8,
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
			Phone:     "+1 (415) 555-0112",
			Duration:  54000, // 15 hours
			Barcode:   "TASK-2024-012",
			Completed: true,
			Effort:    13,
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
			Phone:     "+1 (415) 555-0113",
			Duration:  3600, // 1 hour (just started)
			Barcode:   "TASK-2024-013",
			Completed: false,
			Effort:    8,
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
			Phone:     "+1 (415) 555-0114",
			Duration:  0,
			Barcode:   "TASK-2024-014",
			Completed: false,
			Effort:    21,
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
			Phone:     "+1 (415) 555-0115",
			Duration:  1800, // 30 minutes
			Barcode:   "TASK-2024-015",
			Completed: false,
			Effort:    13,
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
			Phone:     "+1 (415) 555-0116",
			Duration:  0,
			Barcode:   "TASK-2024-016",
			Completed: false,
			Effort:    8,
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
			Phone:     "+1 (415) 555-0117",
			Duration:  0,
			Barcode:   "TASK-2024-017",
			Completed: false,
			Effort:    3,
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
			Phone:     "+1 (415) 555-0118",
			Duration:  900, // 15 minutes
			Barcode:   "TASK-2024-018",
			Completed: false,
			Effort:    8,
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
			Phone:     "+1 (415) 555-0119",
			Duration:  0,
			Barcode:   "TASK-2024-019",
			Completed: false,
			Effort:    21,
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
			Phone:     "+1 (415) 555-0120",
			Duration:  0,
			Barcode:   "TASK-2024-020",
			Completed: false,
			Effort:    34,
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
			Phone:     "+1 (415) 555-0121",
			Duration:  0,
			Barcode:   "TASK-2024-021",
			Completed: false,
			Effort:    21,
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
			Phone:     "+1 (415) 555-0122",
			Duration:  0,
			Barcode:   "TASK-2024-022",
			Completed: false,
			Effort:    21,
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
			Phone:     "+1 (415) 555-0123",
			Duration:  0,
			Barcode:   "TASK-2024-023",
			Completed: false,
			Effort:    34,
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
		// New field types
		if phoneField != nil && task.Phone != "" {
			cells[phoneField.ID] = task.Phone
		}
		if durationField != nil && task.Duration > 0 {
			cells[durationField.ID] = task.Duration
		}
		if barcodeField != nil && task.Barcode != "" {
			cells[barcodeField.ID] = task.Barcode
		}
		if completedField != nil {
			cells[completedField.ID] = task.Completed
		}
		if effortField != nil && task.Effort > 0 {
			cells[effortField.ID] = task.Effort
		}
		// Button field is computed, no value needed

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

	// Create Team Members table
	teamStart := time.Now()
	stepStart = time.Now()
	fmt.Printf("  Creating table 'Team Members'... ")
	teamTable, err := srv.TableService().Create(ctx, ownerUserID, tables.CreateIn{
		BaseID: base.ID,
		Name:   "Team Members",
	})
	if err != nil {
		fmt.Printf("✗ %v\n", err)
	} else {
		fmt.Printf("✓ (%v)\n", time.Since(stepStart).Round(time.Millisecond))

		// Create team member fields
		fmt.Print("  Creating team fields...\n")
		teamNameField := createFieldForTable(srv, ctx, teamTable.ID, "Name", "single_line_text", nil, ownerUserID)
		teamEmailField := createFieldForTable(srv, ctx, teamTable.ID, "Email", "email", nil, ownerUserID)
		teamRoleField := createFieldForTable(srv, ctx, teamTable.ID, "Role", "single_select", map[string]any{
			"choices": []map[string]any{
				{"id": "role-1", "name": "Engineer", "color": "#3B82F6"},
				{"id": "role-2", "name": "Designer", "color": "#EC4899"},
				{"id": "role-3", "name": "Product Manager", "color": "#8B5CF6"},
				{"id": "role-4", "name": "QA Engineer", "color": "#10B981"},
				{"id": "role-5", "name": "DevOps", "color": "#F59E0B"},
				{"id": "role-6", "name": "Team Lead", "color": "#EF4444"},
			},
		}, ownerUserID)
		teamDeptField := createFieldForTable(srv, ctx, teamTable.ID, "Department", "single_select", map[string]any{
			"choices": []map[string]any{
				{"id": "dept-1", "name": "Engineering", "color": "#3B82F6"},
				{"id": "dept-2", "name": "Design", "color": "#EC4899"},
				{"id": "dept-3", "name": "Product", "color": "#8B5CF6"},
				{"id": "dept-4", "name": "Marketing", "color": "#F59E0B"},
			},
		}, ownerUserID)
		teamPhoneField := createFieldForTable(srv, ctx, teamTable.ID, "Phone", "phone", nil, ownerUserID)
		teamStartField := createFieldForTable(srv, ctx, teamTable.ID, "Start Date", "date", nil, ownerUserID)
		teamSkillsField := createFieldForTable(srv, ctx, teamTable.ID, "Skills", "multi_select", map[string]any{
			"choices": []map[string]any{
				{"id": "skill-1", "name": "React", "color": "#61DAFB"},
				{"id": "skill-2", "name": "Go", "color": "#00ADD8"},
				{"id": "skill-3", "name": "Python", "color": "#3776AB"},
				{"id": "skill-4", "name": "TypeScript", "color": "#3178C6"},
				{"id": "skill-5", "name": "Figma", "color": "#F24E1E"},
				{"id": "skill-6", "name": "SQL", "color": "#336791"},
				{"id": "skill-7", "name": "Docker", "color": "#2496ED"},
				{"id": "skill-8", "name": "AWS", "color": "#FF9900"},
			},
		}, ownerUserID)
		teamActiveField := createFieldForTable(srv, ctx, teamTable.ID, "Active", "checkbox", nil, ownerUserID)
		teamAvatarField := createFieldForTable(srv, ctx, teamTable.ID, "Avatar", "attachment", nil, ownerUserID)
		teamSalaryField := createFieldForTable(srv, ctx, teamTable.ID, "Salary", "currency", map[string]any{
			"currency_symbol": "$",
			"precision":       0,
		}, ownerUserID)

		// Team member data
		teamMembers := []struct {
			Name       string
			Email      string
			Role       string
			Dept       string
			Phone      string
			StartDays  int
			Skills     []string
			Active     bool
			Avatar     string
			Salary     float64
		}{
			{"Alice Johnson", "alice@example.com", "role-6", "dept-1", "+1 (415) 555-1001", -730, []string{"skill-1", "skill-4", "skill-7"}, true, "https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=200", 145000},
			{"Bob Smith", "bob@example.com", "role-1", "dept-1", "+1 (415) 555-1002", -365, []string{"skill-2", "skill-6", "skill-8"}, true, "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=200", 125000},
			{"Charlie Brown", "charlie@example.com", "role-2", "dept-2", "+1 (415) 555-1003", -180, []string{"skill-5"}, true, "https://images.unsplash.com/photo-1500648767791-00dcc994a43e?w=200", 115000},
			{"Diana Martinez", "diana@example.com", "role-1", "dept-1", "+1 (415) 555-1004", -90, []string{"skill-1", "skill-3", "skill-4"}, true, "https://images.unsplash.com/photo-1438761681033-6461ffad8d80?w=200", 120000},
			{"Edward Kim", "edward@example.com", "role-3", "dept-3", "+1 (415) 555-1005", -540, []string{}, true, "https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=200", 140000},
			{"Fiona Chen", "fiona@example.com", "role-4", "dept-1", "+1 (415) 555-1006", -270, []string{"skill-3", "skill-6"}, true, "https://images.unsplash.com/photo-1544005313-94ddf0286df2?w=200", 105000},
			{"George Wilson", "george@example.com", "role-5", "dept-1", "+1 (415) 555-1007", -450, []string{"skill-7", "skill-8"}, true, "https://images.unsplash.com/photo-1506794778202-cad84cf45f1d?w=200", 135000},
			{"Hannah Lee", "hannah@example.com", "role-1", "dept-1", "+1 (415) 555-1008", -60, []string{"skill-1", "skill-2", "skill-4"}, true, "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=200", 110000},
		}

		var teamRecords []map[string]any
		for _, tm := range teamMembers {
			fmt.Printf("    • %s\n", tm.Name)
			cells := make(map[string]any)
			if teamNameField != nil {
				cells[teamNameField.ID] = tm.Name
			}
			if teamEmailField != nil {
				cells[teamEmailField.ID] = tm.Email
			}
			if teamRoleField != nil {
				cells[teamRoleField.ID] = tm.Role
			}
			if teamDeptField != nil {
				cells[teamDeptField.ID] = tm.Dept
			}
			if teamPhoneField != nil {
				cells[teamPhoneField.ID] = tm.Phone
			}
			if teamStartField != nil {
				cells[teamStartField.ID] = time.Now().AddDate(0, 0, tm.StartDays).Format("2006-01-02")
			}
			if teamSkillsField != nil && len(tm.Skills) > 0 {
				cells[teamSkillsField.ID] = tm.Skills
			}
			if teamActiveField != nil {
				cells[teamActiveField.ID] = tm.Active
			}
			if teamAvatarField != nil && tm.Avatar != "" {
				cells[teamAvatarField.ID] = []map[string]any{
					{
						"id":        fmt.Sprintf("avatar-%s", tm.Email),
						"filename":  "avatar.jpg",
						"url":       tm.Avatar,
						"mime_type": "image/jpeg",
						"size":      10000,
					},
				}
			}
			if teamSalaryField != nil {
				cells[teamSalaryField.ID] = tm.Salary
			}
			teamRecords = append(teamRecords, cells)
		}

		stepStart = time.Now()
		fmt.Printf("  Inserting %d team member records... ", len(teamRecords))
		if _, err := srv.RecordService().CreateBatch(ctx, teamTable.ID, teamRecords, ownerUserID); err != nil {
			fmt.Printf("✗ %v\n", err)
		} else {
			fmt.Printf("✓ (%v)\n", time.Since(stepStart).Round(time.Millisecond))
		}

		// Create team views
		teamViewTypes := []struct {
			Name   string
			Type   string
			Config map[string]any
		}{
			{"All Members", "grid", nil},
			{"By Role", "kanban", map[string]any{"groupBy": teamRoleField.ID}},
			{"By Department", "kanban", map[string]any{"groupBy": teamDeptField.ID}},
			{"Team Gallery", "gallery", map[string]any{"cover_field_id": teamAvatarField.ID}},
		}

		for _, vt := range teamViewTypes {
			view, err := srv.ViewService().Create(ctx, ownerUserID, views.CreateIn{
				TableID: teamTable.ID,
				Name:    vt.Name,
				Type:    vt.Type,
			})
			if err == nil && vt.Config != nil && view != nil {
				srv.ViewService().SetConfig(ctx, view.ID, vt.Config)
			}
		}
		Step("", "Team Members table ready", time.Since(teamStart))
	}

	// Create Clients table
	clientStart := time.Now()
	stepStart = time.Now()
	fmt.Printf("  Creating table 'Clients'... ")
	clientTable, err := srv.TableService().Create(ctx, ownerUserID, tables.CreateIn{
		BaseID: base.ID,
		Name:   "Clients",
	})
	if err != nil {
		fmt.Printf("✗ %v\n", err)
	} else {
		fmt.Printf("✓ (%v)\n", time.Since(stepStart).Round(time.Millisecond))

		// Create client fields
		fmt.Print("  Creating client fields...\n")
		clientNameField := createFieldForTable(srv, ctx, clientTable.ID, "Company", "single_line_text", nil, ownerUserID)
		clientContactField := createFieldForTable(srv, ctx, clientTable.ID, "Contact Name", "single_line_text", nil, ownerUserID)
		clientEmailField := createFieldForTable(srv, ctx, clientTable.ID, "Email", "email", nil, ownerUserID)
		clientPhoneField := createFieldForTable(srv, ctx, clientTable.ID, "Phone", "phone", nil, ownerUserID)
		clientWebsiteField := createFieldForTable(srv, ctx, clientTable.ID, "Website", "url", nil, ownerUserID)
		clientStatusField := createFieldForTable(srv, ctx, clientTable.ID, "Status", "single_select", map[string]any{
			"choices": []map[string]any{
				{"id": "cstat-1", "name": "Lead", "color": "#6B7280"},
				{"id": "cstat-2", "name": "Prospect", "color": "#F59E0B"},
				{"id": "cstat-3", "name": "Active", "color": "#10B981"},
				{"id": "cstat-4", "name": "Churned", "color": "#EF4444"},
			},
		}, ownerUserID)
		clientContractField := createFieldForTable(srv, ctx, clientTable.ID, "Contract Value", "currency", map[string]any{
			"currency_symbol": "$",
			"precision":       0,
		}, ownerUserID)
		clientRenewalField := createFieldForTable(srv, ctx, clientTable.ID, "Renewal Date", "date", nil, ownerUserID)
		clientNotesField := createFieldForTable(srv, ctx, clientTable.ID, "Notes", "long_text", nil, ownerUserID)
		clientLogoField := createFieldForTable(srv, ctx, clientTable.ID, "Logo", "attachment", nil, ownerUserID)
		clientSatisfactionField := createFieldForTable(srv, ctx, clientTable.ID, "Satisfaction", "rating", map[string]any{"max": 5}, ownerUserID)

		// Client data
		clients := []struct {
			Company      string
			Contact      string
			Email        string
			Phone        string
			Website      string
			Status       string
			Contract     float64
			RenewalDays  int
			Notes        string
			Logo         string
			Satisfaction int
		}{
			{"Acme Corporation", "John Davis", "john@acme.com", "+1 (800) 555-2001", "https://acme.example.com", "cstat-3", 250000, 90, "Enterprise client since 2021. Very engaged with product roadmap.", "https://logo.clearbit.com/acme.com", 5},
			{"TechStart Inc", "Sarah Miller", "sarah@techstart.io", "+1 (800) 555-2002", "https://techstart.io", "cstat-3", 75000, 180, "Fast-growing startup. Interested in advanced features.", "https://logo.clearbit.com/stripe.com", 4},
			{"Global Industries", "Michael Brown", "m.brown@global.com", "+1 (800) 555-2003", "https://global-ind.example.com", "cstat-2", 500000, 45, "Large enterprise prospect. Currently in pilot phase.", "https://logo.clearbit.com/ibm.com", 3},
			{"Creative Agency Co", "Emily White", "emily@creative.co", "+1 (800) 555-2004", "https://creative.co", "cstat-3", 45000, 270, "Design agency with multiple teams using the product.", "https://logo.clearbit.com/figma.com", 5},
			{"DataFlow Systems", "Robert Johnson", "robert@dataflow.io", "+1 (800) 555-2005", "https://dataflow.io", "cstat-3", 120000, 120, "Data analytics company. Heavy API usage.", "https://logo.clearbit.com/snowflake.com", 4},
			{"Retail Plus", "Lisa Anderson", "lisa@retailplus.com", "+1 (800) 555-2006", "https://retailplus.com", "cstat-4", 80000, -30, "Churned due to budget cuts. Keep in touch for Q2.", "https://logo.clearbit.com/shopify.com", 2},
			{"Healthcare Solutions", "David Wilson", "david@healthsol.org", "+1 (800) 555-2007", "https://healthsol.org", "cstat-1", 0, 0, "New lead from conference. Schedule demo for next week.", "https://logo.clearbit.com/epic.com", 0},
			{"EduTech Learning", "Jennifer Taylor", "j.taylor@edutech.edu", "+1 (800) 555-2008", "https://edutech.edu", "cstat-2", 150000, 60, "Education sector prospect. RFP submitted.", "https://logo.clearbit.com/coursera.org", 3},
		}

		var clientRecords []map[string]any
		for _, c := range clients {
			fmt.Printf("    • %s\n", c.Company)
			cells := make(map[string]any)
			if clientNameField != nil {
				cells[clientNameField.ID] = c.Company
			}
			if clientContactField != nil {
				cells[clientContactField.ID] = c.Contact
			}
			if clientEmailField != nil {
				cells[clientEmailField.ID] = c.Email
			}
			if clientPhoneField != nil {
				cells[clientPhoneField.ID] = c.Phone
			}
			if clientWebsiteField != nil {
				cells[clientWebsiteField.ID] = c.Website
			}
			if clientStatusField != nil {
				cells[clientStatusField.ID] = c.Status
			}
			if clientContractField != nil && c.Contract > 0 {
				cells[clientContractField.ID] = c.Contract
			}
			if clientRenewalField != nil && c.RenewalDays != 0 {
				cells[clientRenewalField.ID] = time.Now().AddDate(0, 0, c.RenewalDays).Format("2006-01-02")
			}
			if clientNotesField != nil {
				cells[clientNotesField.ID] = c.Notes
			}
			if clientLogoField != nil && c.Logo != "" {
				cells[clientLogoField.ID] = []map[string]any{
					{
						"id":        fmt.Sprintf("logo-%s", c.Company),
						"filename":  "logo.png",
						"url":       c.Logo,
						"mime_type": "image/png",
						"size":      5000,
					},
				}
			}
			if clientSatisfactionField != nil && c.Satisfaction > 0 {
				cells[clientSatisfactionField.ID] = c.Satisfaction
			}
			clientRecords = append(clientRecords, cells)
		}

		stepStart = time.Now()
		fmt.Printf("  Inserting %d client records... ", len(clientRecords))
		if _, err := srv.RecordService().CreateBatch(ctx, clientTable.ID, clientRecords, ownerUserID); err != nil {
			fmt.Printf("✗ %v\n", err)
		} else {
			fmt.Printf("✓ (%v)\n", time.Since(stepStart).Round(time.Millisecond))
		}

		// Create client views
		clientViewTypes := []struct {
			Name   string
			Type   string
			Config map[string]any
		}{
			{"All Clients", "grid", nil},
			{"Pipeline", "kanban", map[string]any{"groupBy": clientStatusField.ID}},
			{"Client Gallery", "gallery", map[string]any{"cover_field_id": clientLogoField.ID}},
			{"Renewals", "calendar", map[string]any{"dateField": clientRenewalField.ID}},
		}

		for _, vt := range clientViewTypes {
			view, err := srv.ViewService().Create(ctx, ownerUserID, views.CreateIn{
				TableID: clientTable.ID,
				Name:    vt.Name,
				Type:    vt.Type,
			})
			if err == nil && vt.Config != nil && view != nil {
				srv.ViewService().SetConfig(ctx, view.ID, vt.Config)
			}
		}
		Step("", "Clients table ready", time.Since(clientStart))
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
		"Tables", fmt.Sprintf("Tasks (%d records), Projects (6), Team Members (8), Clients (8)", recordCount),
		"Views", fmt.Sprintf("%d+ views across all tables (Grid, Kanban, Calendar, Gallery, Timeline, Form, List)", viewCount),
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
