package cli

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/table/app/web"
	"github.com/go-mizu/blueprints/table/feature/records"
	"github.com/go-mizu/blueprints/table/feature/workspaces"
)

//go:embed seed/*
var seedFS embed.FS

// NewSeed creates the seed command
func NewSeed() *cobra.Command {
	var seedName string

	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with demo data",
		Long: `Seed the Table database with demo data from a seed directory.

Available seeds:
  - project_tracker (default): Tasks, Projects, Team Members, Clients
  - crm: Leads, Contacts, Deals, Companies (Sales CRM)
  - inventory: Products, Suppliers, Orders, Warehouses
  - hr: Employees, Departments, Leave Requests, Performance Reviews
  - real_estate: Properties, Agents, Showings, Clients
  - restaurant: Menu Items, Orders, Tables, Reservations
  - ecommerce: Products, Categories, Orders, Customers
  - event_management: Events, Venues, Speakers, Attendees
  - bug_tracker: Issues, Sprints, Components, Milestones
  - library: Books, Authors, Members, Loans

Creates:
  - 3 users (alice, bob, charlie)
  - Personal workspace for Alice
  - Selected base with tables, fields, views, and sample records

To reset the database, delete the data directory first:
  rm -rf ~/data/blueprint/table && table seed

Examples:
  table seed                              # Seed with project_tracker (default)
  table seed --name crm                   # Seed with Sales CRM data
  table seed --name ecommerce             # Seed with E-Commerce data
  table seed --data /path/to              # Seed specific database`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSeed(seedName)
		},
	}

	cmd.Flags().StringVarP(&seedName, "name", "n", "project_tracker", "Name of the seed dataset to use")

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

func runSeed(seedName string) error {
	Blank()
	Header("", "Seed Database")
	Blank()

	Summary("Data", dataDir)
	Summary("Seed", seedName)
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

	// Extract seed data to temp directory and import
	sectionStart = time.Now()
	stepStart = time.Now()
	fmt.Printf("  Extracting seed data '%s'... ", seedName)

	tempDir, err := extractSeedData(seedName)
	if err != nil {
		fmt.Printf("✗ %v\n", err)
		return err
	}
	defer os.RemoveAll(tempDir)
	fmt.Printf("✓ (%v)\n", time.Since(stepStart).Round(time.Millisecond))

	// Import the base using importexport service
	stepStart = time.Now()
	fmt.Printf("  Importing base from seed... ")
	base, err := srv.ImportExportService().Import(ctx, ws.ID, ownerUserID, tempDir)
	if err != nil {
		fmt.Printf("✗ %v\n", err)
		return err
	}
	fmt.Printf("✓ (%v)\n", time.Since(stepStart).Round(time.Millisecond))
	Step("", fmt.Sprintf("Base '%s' imported", base.Name), time.Since(sectionStart))

	// Get statistics
	tables, _ := srv.TableService().ListByBase(ctx, base.ID)
	tableCount := len(tables)
	var viewCount, recordCount int
	for _, tbl := range tables {
		views, _ := srv.ViewService().ListByTable(ctx, tbl.ID)
		viewCount += len(views)
		recs, _ := srv.RecordService().List(ctx, tbl.ID, records.ListOpts{Limit: 1})
		recordCount += recs.Total
	}

	Blank()
	Step("", "Database seeded", time.Since(totalStart))
	Blank()
	Success("Sample data created")
	Blank()

	Summary(
		"Users", fmt.Sprintf("%d users (alice, bob, charlie)", userCount),
		"Password", "password123",
		"Base", base.Name,
		"Tables", fmt.Sprintf("%d tables", tableCount),
		"Views", fmt.Sprintf("%d views", viewCount),
		"Records", fmt.Sprintf("%d records", recordCount),
	)
	Blank()
	Hint("Start server: table serve")
	Hint("Login with: alice@example.com / password123")
	Hint("To reset: rm -rf " + dataDir + " && table seed")
	Blank()

	return nil
}

// extractSeedData extracts embedded seed data to a temporary directory
func extractSeedData(seedName string) (string, error) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "table-seed-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	seedPath := filepath.Join("seed", seedName)

	// Check if seed exists
	if _, err := seedFS.ReadDir(seedPath); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("seed '%s' not found", seedName)
	}

	// Extract all files from embedded filesystem
	err = fs.WalkDir(seedFS, seedPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from seed directory
		relPath, err := filepath.Rel(seedPath, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(tempDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// Read and write file
		content, err := seedFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		return os.WriteFile(destPath, content, 0644)
	})

	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("extract seed: %w", err)
	}

	return tempDir, nil
}
