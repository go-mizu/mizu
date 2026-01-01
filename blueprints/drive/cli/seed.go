package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/drive/app/web"
	"github.com/go-mizu/blueprints/drive/feature/accounts"
	"github.com/go-mizu/blueprints/drive/feature/files"
	"github.com/go-mizu/blueprints/drive/feature/folders"
)

// NewSeed creates the seed command
func NewSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with demo data",
		Long: `Seed the Drive database with demo data for testing.

Creates sample users with files and folders:
  - 3 users (alice, bob, charlie)
  - Sample folder structures
  - Various file types
  - Shared files and folders

To reset the database, delete the data directory first:
  rm -rf ~/data/blueprint/drive && drive seed

Examples:
  drive seed                     # Seed with demo data
  drive seed --data /path/to     # Seed specific database`,
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

// seedFolders defines the folder structure for each user
var seedFolders = []struct {
	Name     string
	Children []string
}{
	{"Documents", []string{"Work", "Personal", "Receipts"}},
	{"Photos", []string{"Vacation 2024", "Family", "Events"}},
	{"Projects", []string{"Website", "Mobile App", "Design"}},
	{"Music", nil},
	{"Videos", nil},
}

// seedFiles defines sample files to create
var seedFiles = []struct {
	Name     string
	Folder   string // Parent folder name, empty for root
	MimeType string
	Size     int64
}{
	{"README.md", "", "text/markdown", 2048},
	{"notes.txt", "Documents", "text/plain", 1024},
	{"report.pdf", "Documents/Work", "application/pdf", 102400},
	{"budget.xlsx", "Documents/Personal", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", 51200},
	{"receipt-amazon.pdf", "Documents/Receipts", "application/pdf", 25600},
	{"beach.jpg", "Photos/Vacation 2024", "image/jpeg", 2097152},
	{"sunset.jpg", "Photos/Vacation 2024", "image/jpeg", 1572864},
	{"family-dinner.jpg", "Photos/Family", "image/jpeg", 1835008},
	{"birthday.mp4", "Photos/Events", "video/mp4", 52428800},
	{"index.html", "Projects/Website", "text/html", 4096},
	{"styles.css", "Projects/Website", "text/css", 2048},
	{"app.js", "Projects/Website", "application/javascript", 8192},
	{"mockup.psd", "Projects/Design", "image/vnd.adobe.photoshop", 10485760},
	{"song.mp3", "Music", "audio/mpeg", 5242880},
	{"presentation.pptx", "", "application/vnd.openxmlformats-officedocument.presentationml.presentation", 2097152},
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
	createdUsers := make([]*accounts.User, 0, len(seedUsers))
	for _, u := range seedUsers {
		user, _, err := srv.AccountService().Register(ctx, &accounts.RegisterIn{
			Email:    u.Email,
			Name:     u.Name,
			Password: u.Password,
		})
		if err != nil {
			// Try to get existing user
			user, _ = srv.AccountService().GetByEmail(ctx, u.Email)
		}
		if user != nil {
			createdUsers = append(createdUsers, user)
		}
	}

	if len(createdUsers) == 0 {
		stop()
		return fmt.Errorf("failed to create any users")
	}

	// Create folder structure and files for the first user (Alice)
	alice := createdUsers[0]
	folderMap := make(map[string]string) // path -> folder ID

	// Create top-level folders
	for _, sf := range seedFolders {
		folder, err := srv.FolderService().Create(ctx, alice.ID, &folders.CreateIn{
			Name: sf.Name,
		})
		if err != nil {
			continue
		}
		folderMap[sf.Name] = folder.ID

		// Create child folders
		for _, child := range sf.Children {
			childFolder, err := srv.FolderService().Create(ctx, alice.ID, &folders.CreateIn{
				Name:     child,
				ParentID: folder.ID,
			})
			if err != nil {
				continue
			}
			folderMap[sf.Name+"/"+child] = childFolder.ID
		}
	}

	// Create files
	createdFiles := 0
	for _, sf := range seedFiles {
		var parentID string
		if sf.Folder != "" {
			parentID = folderMap[sf.Folder]
			if parentID == "" {
				continue // Skip if parent folder doesn't exist
			}
		}

		_, err := srv.FileService().Create(ctx, alice.ID, &files.CreateIn{
			Name:     sf.Name,
			ParentID: parentID,
			MimeType: sf.MimeType,
			Size:     sf.Size,
		})
		if err == nil {
			createdFiles++
		}
	}

	// Create some starred items for Alice
	allFolders, _ := srv.FolderService().ListByUser(ctx, alice.ID)
	for i, f := range allFolders {
		if i < 2 {
			_ = srv.FolderService().Star(ctx, f.ID, alice.ID)
		}
	}

	allFiles, _ := srv.FileService().ListByUser(ctx, alice.ID, "")
	for i, f := range allFiles {
		if i < 3 {
			_ = srv.FileService().Star(ctx, f.ID, alice.ID)
		}
	}

	// Create a shared folder between Alice and Bob
	if len(createdUsers) >= 2 {
		bob := createdUsers[1]

		// Create a shared folder
		sharedFolder, err := srv.FolderService().Create(ctx, alice.ID, &folders.CreateIn{
			Name: "Shared with Bob",
		})
		if err == nil {
			// Share it with Bob
			_, _ = srv.ShareService().Create(ctx, alice.ID, sharedFolder.ID, "folder", bob.ID, "editor")
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
		"Folders", fmt.Sprintf("%d folders", len(folderMap)),
		"Files", fmt.Sprintf("%d files", createdFiles),
	)
	Blank()
	Hint("Start the server with: drive serve")
	Hint("Login with: alice@example.com / password123")
	Hint("To reset: rm -rf " + dataDir + " && drive seed")
	Blank()

	return nil
}
