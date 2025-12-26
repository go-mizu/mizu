package cmd

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/kanban/app/web"
	"github.com/go-mizu/blueprints/kanban/feature/issues"
	"github.com/go-mizu/blueprints/kanban/feature/labels"
	"github.com/go-mizu/blueprints/kanban/feature/projects"
	"github.com/go-mizu/blueprints/kanban/feature/users"
	"github.com/go-mizu/blueprints/kanban/feature/workspaces"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed the database with sample data",
	Long: `Seed the database with sample data for development and testing.

This creates:
  - A demo user (demo@example.com / demo)
  - A sample workspace
  - Sample projects with issues
  - Labels and sample data

Examples:
  kanban seed                     # Seed the database
  kanban seed --db /path/to/db    # Seed specific database`,
	RunE: runSeed,
}

func init() {
	rootCmd.AddCommand(seedCmd)
}

func runSeed(cmd *cobra.Command, args []string) error {
	dbPath, _ := cmd.Root().PersistentFlags().GetString("db")

	log.Printf("Seeding database: %s", dbPath)

	dataDir := filepath.Dir(dbPath)
	if dataDir == "." {
		dataDir = "."
	}

	srv, err := web.New(web.Config{
		Addr:    ":8080",
		DataDir: dataDir,
		Dev:     false,
	})
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}
	defer srv.Close()

	ctx := context.Background()

	// Get services
	userSvc := srv.UserService()
	workspaceSvc := srv.WorkspaceService()
	projectSvc := srv.ProjectService()
	issueSvc := srv.IssueService()
	labelSvc := srv.LabelService()

	// Create demo user
	log.Println("Creating demo user...")
	user, _, err := userSvc.Register(ctx, &users.RegisterIn{
		Email:       "demo@example.com",
		Username:    "demo",
		Password:    "demo",
		DisplayName: "Demo User",
	})
	if err != nil {
		log.Printf("Note: %v (user may already exist, trying login)", err)
		// Try to login as existing user
		user, _, _ = userSvc.Login(ctx, &users.LoginIn{
			Email:    "demo@example.com",
			Password: "demo",
		})
	}
	if user == nil {
		return fmt.Errorf("failed to create or find demo user")
	}

	// Create workspace
	log.Println("Creating workspace...")
	workspace, err := workspaceSvc.Create(ctx, user.ID, &workspaces.CreateIn{
		Slug:        "acme",
		Name:        "Acme Corp",
		Description: "Main workspace for Acme Corporation",
	})
	if err != nil {
		log.Printf("Note: %v (workspace may already exist)", err)
		workspace, _ = workspaceSvc.GetBySlug(ctx, "acme")
	}
	if workspace == nil {
		return fmt.Errorf("failed to create or find workspace")
	}

	// Create projects
	log.Println("Creating projects...")
	projectData := []projects.CreateIn{
		{
			Key:         "WEB",
			Name:        "Website Redesign",
			Description: "Complete redesign of the company website",
			Color:       "#3b82f6",
		},
		{
			Key:         "MOB",
			Name:        "Mobile App",
			Description: "Native mobile application for iOS and Android",
			Color:       "#8b5cf6",
		},
		{
			Key:         "API",
			Name:        "Backend API",
			Description: "Core backend services and APIs",
			Color:       "#22c55e",
		},
	}

	var createdProjects []*projects.Project
	for _, pd := range projectData {
		pdCopy := pd // capture for pointer
		project, err := projectSvc.Create(ctx, workspace.ID, &pdCopy)
		if err != nil {
			log.Printf("Note: %v", err)
			project, _ = projectSvc.GetByKey(ctx, workspace.ID, pd.Key)
		}
		if project != nil {
			createdProjects = append(createdProjects, project)
		}
	}

	if len(createdProjects) == 0 {
		return fmt.Errorf("failed to create any projects")
	}

	// Create labels for first project
	log.Println("Creating labels...")
	project := createdProjects[0]
	labelData := []labels.CreateIn{
		{Name: "bug", Color: "#ef4444"},
		{Name: "enhancement", Color: "#22c55e"},
		{Name: "documentation", Color: "#3b82f6"},
		{Name: "urgent", Color: "#f97316"},
		{Name: "design", Color: "#8b5cf6"},
	}

	for _, ld := range labelData {
		ldCopy := ld // capture for pointer
		_, err := labelSvc.Create(ctx, project.ID, &ldCopy)
		if err != nil {
			log.Printf("Note: label %s: %v", ld.Name, err)
		}
	}

	// Create sample issues
	log.Println("Creating issues...")
	issueData := []issues.CreateIn{
		{
			Title:       "Implement user authentication",
			Description: "Add login, registration, and session management",
			Type:        "task",
			Status:      "done",
			Priority:    "high",
		},
		{
			Title:       "Design new landing page",
			Description: "Create mockups for the new landing page design",
			Type:        "story",
			Status:      "in_progress",
			Priority:    "high",
		},
		{
			Title:       "Fix navigation menu on mobile",
			Description: "The hamburger menu doesn't close properly on mobile devices",
			Type:        "bug",
			Status:      "in_progress",
			Priority:    "medium",
		},
		{
			Title:       "Add dark mode support",
			Description: "Implement dark mode theme toggle with system preference detection",
			Type:        "task",
			Status:      "todo",
			Priority:    "medium",
		},
		{
			Title:       "Optimize image loading",
			Description: "Implement lazy loading and WebP format for images",
			Type:        "task",
			Status:      "todo",
			Priority:    "low",
		},
		{
			Title:       "Write API documentation",
			Description: "Document all REST endpoints with examples",
			Type:        "task",
			Status:      "backlog",
			Priority:    "low",
		},
		{
			Title:       "Performance audit",
			Description: "Run Lighthouse audit and fix performance issues",
			Type:        "task",
			Status:      "backlog",
			Priority:    "medium",
		},
		{
			Title:       "Setup CI/CD pipeline",
			Description: "Configure GitHub Actions for automated testing and deployment",
			Type:        "epic",
			Status:      "in_review",
			Priority:    "high",
		},
	}

	for _, id := range issueData {
		idCopy := id // capture for pointer
		_, err := issueSvc.Create(ctx, project.ID, user.ID, &idCopy)
		if err != nil {
			log.Printf("Note: issue '%s': %v", id.Title, err)
		}
	}

	log.Println("✅ Database seeded successfully")
	log.Println("")
	log.Println("Demo credentials:")
	log.Println("  Email:    demo@example.com")
	log.Println("  Password: demo")
	log.Println("")
	log.Println("Run 'kanban serve' to start the server")

	return nil
}
