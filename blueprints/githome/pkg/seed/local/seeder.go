package local

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

// Config contains seeder configuration.
type Config struct {
	ScanDir      string
	AdminUserID  int64
	AdminActorID int64 // unused, kept for API compatibility
	IsPublic     bool
}

// Result contains the result of a seeding operation.
type Result struct {
	OrgsCreated  int
	ReposCreated int
	ReposSkipped int
	Errors       []error
}

// Seeder seeds data from local repositories.
type Seeder struct {
	db           *sql.DB
	usersStore   *duckdb.UsersStore
	orgsStore    *duckdb.OrgsStore
	reposStore   *duckdb.ReposStore
	config       Config
}

// NewSeeder creates a new Seeder.
func NewSeeder(db *sql.DB, config Config) *Seeder {
	return &Seeder{
		db:         db,
		usersStore: duckdb.NewUsersStore(db),
		orgsStore:  duckdb.NewOrgsStore(db),
		reposStore: duckdb.NewReposStore(db),
		config:     config,
	}
}

// Seed scans the configured directory and imports repositories.
func (s *Seeder) Seed(ctx context.Context) (*Result, error) {
	result := &Result{}

	// Scan for org directories
	entries, err := os.ReadDir(s.config.ScanDir)
	if err != nil {
		return nil, fmt.Errorf("read scan dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip hidden directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		orgName := entry.Name()
		orgPath := filepath.Join(s.config.ScanDir, orgName)

		// Ensure org exists
		org, err := s.ensureOrg(ctx, orgName)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("ensure org %s: %w", orgName, err))
			continue
		}
		if org != nil {
			result.OrgsCreated++
		}

		// Get or retrieve the org
		existingOrg, err := s.orgsStore.GetByLogin(ctx, orgName)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("get org %s: %w", orgName, err))
			continue
		}
		if existingOrg == nil {
			result.Errors = append(result.Errors, fmt.Errorf("org %s not found after creation", orgName))
			continue
		}

		// Scan repos in this org
		repoEntries, err := os.ReadDir(orgPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("read org dir %s: %w", orgPath, err))
			continue
		}

		for _, repoEntry := range repoEntries {
			if !repoEntry.IsDir() {
				continue
			}

			// Skip hidden directories
			if strings.HasPrefix(repoEntry.Name(), ".") {
				continue
			}

			repoName := repoEntry.Name()
			repoPath := filepath.Join(orgPath, repoName)

			// Check if it's a git repo
			if !isGitRepo(repoPath) {
				continue
			}

			// Check if repo already exists
			existing, err := s.reposStore.GetByOwnerAndName(ctx, existingOrg.ID, repoName)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("check repo %s/%s: %w", orgName, repoName, err))
				continue
			}
			if existing != nil {
				result.ReposSkipped++
				continue
			}

			// Create repo
			now := time.Now()
			visibility := "public"
			if !s.config.IsPublic {
				visibility = "private"
			}

			repo := &repos.Repository{
				Name:          repoName,
				FullName:      fmt.Sprintf("%s/%s", orgName, repoName),
				OwnerID:       existingOrg.ID,
				OwnerType:     "Organization",
				Private:       !s.config.IsPublic,
				Visibility:    visibility,
				DefaultBranch: "main",
				HasIssues:     true,
				HasProjects:   true,
				HasWiki:       true,
				HasDownloads:  true,
				AllowSquashMerge:  true,
				AllowMergeCommit:  true,
				AllowRebaseMerge:  true,
				AllowForking:      true,
				CreatedAt:     now,
				UpdatedAt:     now,
			}

			if err := s.reposStore.Create(ctx, repo); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("create repo %s/%s: %w", orgName, repoName, err))
				continue
			}

			result.ReposCreated++
			slog.Info("imported repository", "org", orgName, "repo", repoName)
		}
	}

	return result, nil
}

// ensureOrg creates an organization if it doesn't exist.
func (s *Seeder) ensureOrg(ctx context.Context, login string) (*orgs.Organization, error) {
	existing, err := s.orgsStore.GetByLogin(ctx, login)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, nil // Already exists
	}

	now := time.Now()
	org := &orgs.Organization{
		Login:                       login,
		Name:                        login,
		Type:                        "Organization",
		HasOrganizationProjects:     true,
		HasRepositoryProjects:       true,
		MembersCanCreateRepositories: true,
		MembersCanCreatePublicRepositories: true,
		MembersCanCreatePrivateRepositories: true,
		DefaultRepositoryPermission: "read",
		CreatedAt:                   now,
		UpdatedAt:                   now,
	}

	if err := s.orgsStore.Create(ctx, org); err != nil {
		return nil, err
	}

	// Add admin user as owner
	if s.config.AdminUserID > 0 {
		if err := s.orgsStore.AddMember(ctx, org.ID, s.config.AdminUserID, "admin", true); err != nil {
			slog.Warn("failed to add admin to org", "org", login, "error", err)
		}
	}

	slog.Info("created organization", "login", login)
	return org, nil
}

func isGitRepo(path string) bool {
	// Check for .git directory (regular repo) or HEAD file (bare repo)
	gitDir := filepath.Join(path, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		return true
	}

	// Check for bare repo
	headFile := filepath.Join(path, "HEAD")
	if _, err := os.Stat(headFile); err == nil {
		return true
	}

	return false
}

// EnsureAdminUser ensures an admin user exists and returns their ID.
func EnsureAdminUser(ctx context.Context, usersStore *duckdb.UsersStore, _ interface{}) (int64, int64, error) {
	// Check if admin user exists
	admin, err := usersStore.GetByLogin(ctx, "admin")
	if err != nil {
		return 0, 0, err
	}
	if admin != nil {
		return admin.ID, admin.ID, nil
	}

	// Create admin user
	now := time.Now()
	admin = &users.User{
		Login:     "admin",
		Name:      "Admin User",
		Email:     "admin@githome.local",
		Type:      "User",
		SiteAdmin: true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := usersStore.Create(ctx, admin); err != nil {
		return 0, 0, err
	}

	slog.Info("created admin user", "login", admin.Login, "id", admin.ID)
	return admin.ID, admin.ID, nil
}
