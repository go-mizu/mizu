// Package local provides seeding GitHome from local git repositories.
// It scans a directory (e.g., $HOME/github) for repositories in the
// format $ORG/$REPO and imports them into GitHome.
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
	gitpkg "github.com/go-mizu/blueprints/githome/pkg/git"
	"github.com/go-mizu/blueprints/githome/pkg/slug"
	"github.com/go-mizu/blueprints/githome/pkg/ulid"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

// Config configures the local seeding process
type Config struct {
	ScanDir      string // Directory to scan for repos (e.g., $HOME/github)
	AdminUserID  string // Admin user ID (repos will be owned by this user or created orgs)
	AdminActorID string // Admin actor ID
	IsPublic     bool   // Whether imported repos should be public (default: true)
}

// Seeder handles seeding GitHome from local repositories
type Seeder struct {
	db     *sql.DB
	config Config
	log    *slog.Logger

	// Stores
	usersStore       *duckdb.UsersStore
	reposStore       *duckdb.ReposStore
	orgsStore        *duckdb.OrgsStore
	actorsStore      *duckdb.ActorsStore
	repoStorageStore *duckdb.RepoStorageStore

	// Caches
	orgActorIDs map[string]string // org slug -> actor ID
}

// Result contains the seeding results
type Result struct {
	OrgsCreated  int
	ReposCreated int
	ReposSkipped int
	Errors       []string
}

// NewSeeder creates a new local seeder
func NewSeeder(db *sql.DB, config Config) *Seeder {
	if config.IsPublic == false && config.ScanDir == "" {
		// Default to public if not explicitly set
		config.IsPublic = true
	}
	return &Seeder{
		db:               db,
		config:           config,
		log:              slog.Default(),
		usersStore:       duckdb.NewUsersStore(db),
		reposStore:       duckdb.NewReposStore(db),
		orgsStore:        duckdb.NewOrgsStore(db),
		actorsStore:      duckdb.NewActorsStore(db),
		repoStorageStore: duckdb.NewRepoStorageStore(db),
		orgActorIDs:      make(map[string]string),
	}
}

// Seed scans the configured directory and imports repositories
func (s *Seeder) Seed(ctx context.Context) (*Result, error) {
	result := &Result{}

	s.log.Info("starting local seed", "scan_dir", s.config.ScanDir)

	// Verify scan directory exists
	info, err := os.Stat(s.config.ScanDir)
	if err != nil {
		return nil, fmt.Errorf("scan directory not found: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scan path is not a directory: %s", s.config.ScanDir)
	}

	// Read org directories
	orgEntries, err := os.ReadDir(s.config.ScanDir)
	if err != nil {
		return nil, fmt.Errorf("read scan directory: %w", err)
	}

	for _, orgEntry := range orgEntries {
		if !orgEntry.IsDir() {
			continue
		}

		// Skip hidden directories
		if strings.HasPrefix(orgEntry.Name(), ".") {
			continue
		}

		orgName := orgEntry.Name()
		orgPath := filepath.Join(s.config.ScanDir, orgName)

		// Read repo directories within org
		repoEntries, err := os.ReadDir(orgPath)
		if err != nil {
			s.log.Warn("failed to read org directory", "org", orgName, "error", err)
			result.Errors = append(result.Errors, fmt.Sprintf("read org %s: %v", orgName, err))
			continue
		}

		var reposInOrg int
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

			// Check if it's a git repository
			if !isGitRepo(repoPath) {
				s.log.Debug("skipping non-git directory", "path", repoPath)
				continue
			}

			reposInOrg++

			// Import the repository
			created, err := s.importRepo(ctx, orgName, repoName, repoPath)
			if err != nil {
				s.log.Warn("failed to import repo", "org", orgName, "repo", repoName, "error", err)
				result.Errors = append(result.Errors, fmt.Sprintf("import %s/%s: %v", orgName, repoName, err))
				result.ReposSkipped++
				continue
			}

			if created {
				result.ReposCreated++
				s.log.Info("imported repository", "org", orgName, "repo", repoName)
			} else {
				result.ReposSkipped++
				s.log.Debug("repository already exists", "org", orgName, "repo", repoName)
			}
		}

		// If we imported any repos for this org, it was created
		if reposInOrg > 0 && s.orgActorIDs[orgName] != "" {
			result.OrgsCreated++
		}
	}

	s.log.Info("local seed completed",
		"orgs_created", result.OrgsCreated,
		"repos_created", result.ReposCreated,
		"repos_skipped", result.ReposSkipped,
		"errors", len(result.Errors))

	return result, nil
}

// importRepo imports a single repository
func (s *Seeder) importRepo(ctx context.Context, orgName, repoName, repoPath string) (bool, error) {
	// Get or create org actor
	actorID, err := s.ensureOrg(ctx, orgName)
	if err != nil {
		return false, fmt.Errorf("ensure org: %w", err)
	}

	// Generate slug
	repoSlug := slug.Make(repoName)

	// Check if repo already exists
	existing, err := s.reposStore.GetByOwnerAndName(ctx, actorID, "org", repoSlug)
	if err != nil {
		return false, fmt.Errorf("check existing: %w", err)
	}
	if existing != nil {
		return false, nil // Already exists
	}

	// Open git repo to get metadata
	gitRepo, err := gitpkg.Open(repoPath)
	if err != nil {
		return false, fmt.Errorf("open git repo: %w", err)
	}

	// Get default branch
	defaultBranch, err := gitRepo.GetDefaultBranch(ctx)
	if err != nil {
		defaultBranch = "main"
	}

	// Get primary language
	language, languageColor, _ := gitRepo.GetPrimaryLanguage(ctx, defaultBranch)

	// Create repo record
	now := time.Now()
	repo := &repos.Repository{
		ID:            ulid.New(),
		OwnerActorID:  actorID,
		OwnerType:     "org",
		OwnerName:     orgName,
		Name:          repoName,
		Slug:          repoSlug,
		Description:   fmt.Sprintf("Imported from %s", repoPath),
		DefaultBranch: defaultBranch,
		IsPrivate:     !s.config.IsPublic,
		Language:      language,
		LanguageColor: languageColor,
		HasIssues:     true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.reposStore.Create(ctx, repo); err != nil {
		return false, fmt.Errorf("create repo: %w", err)
	}

	// Store the local path in repo_storage
	storage := &duckdb.RepoStorage{
		RepoID:         repo.ID,
		StorageBackend: "fs",
		StoragePath:    repoPath,
		CreatedAt:      now,
	}
	if err := s.repoStorageStore.Create(ctx, storage); err != nil {
		// Log but don't fail - the repo was created successfully
		s.log.Warn("failed to store repo path", "repo", repoName, "error", err)
	}

	return true, nil
}

// ensureOrg creates or retrieves an org and returns its actor ID
func (s *Seeder) ensureOrg(ctx context.Context, orgName string) (string, error) {
	orgSlug := slug.Make(orgName)

	// Check cache
	if actorID, ok := s.orgActorIDs[orgSlug]; ok {
		return actorID, nil
	}

	// Check if org exists
	existingOrg, _ := s.orgsStore.GetBySlug(ctx, orgSlug)
	if existingOrg != nil {
		// Get actor for org
		actor, err := s.actorsStore.GetByOrgID(ctx, existingOrg.ID)
		if err != nil {
			return "", err
		}
		if actor != nil {
			s.orgActorIDs[orgSlug] = actor.ID
			return actor.ID, nil
		}
		// Create actor if missing
		actor, err = s.actorsStore.GetOrCreateForOrg(ctx, existingOrg.ID)
		if err != nil {
			return "", err
		}
		s.orgActorIDs[orgSlug] = actor.ID
		return actor.ID, nil
	}

	// Create new org
	now := time.Now()
	org := &orgs.Organization{
		ID:          ulid.New(),
		Name:        orgName,
		Slug:        orgSlug,
		DisplayName: orgName,
		Description: fmt.Sprintf("Organization: %s", orgName),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.orgsStore.Create(ctx, org); err != nil {
		return "", fmt.Errorf("create org: %w", err)
	}

	// Add admin user as owner
	member := &orgs.Member{
		OrgID:     org.ID,
		UserID:    s.config.AdminUserID,
		Role:      orgs.RoleOwner,
		CreatedAt: now,
	}
	if err := s.orgsStore.AddMember(ctx, member); err != nil {
		s.log.Warn("failed to add admin as org owner", "org", orgName, "error", err)
	}

	// Create actor for org
	actor, err := s.actorsStore.GetOrCreateForOrg(ctx, org.ID)
	if err != nil {
		return "", fmt.Errorf("create actor: %w", err)
	}

	s.orgActorIDs[orgSlug] = actor.ID
	s.log.Info("created organization", "name", orgName, "id", org.ID)

	return actor.ID, nil
}

// isGitRepo checks if a path contains a git repository
func isGitRepo(path string) bool {
	// Check for .git directory (normal repo)
	gitDir := filepath.Join(path, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		return true
	}

	// Check for HEAD file (bare repo)
	headFile := filepath.Join(path, "HEAD")
	if info, err := os.Stat(headFile); err == nil && !info.IsDir() {
		return true
	}

	return false
}

// EnsureAdminUser ensures an admin user exists and returns their ID and actor ID
func EnsureAdminUser(ctx context.Context, usersStore *duckdb.UsersStore, actorsStore *duckdb.ActorsStore) (userID, actorID string, err error) {
	// Check if admin user exists
	admin, err := usersStore.GetByUsername(ctx, "admin")
	if err != nil {
		return "", "", err
	}

	if admin == nil {
		// Create admin user
		now := time.Now()
		admin = &users.User{
			ID:           ulid.New(),
			Username:     "admin",
			Email:        "admin@githome.local",
			PasswordHash: "$2a$10$XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX", // Placeholder
			FullName:     "Admin User",
			IsAdmin:      true,
			IsActive:     true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := usersStore.Create(ctx, admin); err != nil {
			return "", "", fmt.Errorf("create admin user: %w", err)
		}
	}

	// Get or create actor
	actor, err := actorsStore.GetOrCreateForUser(ctx, admin.ID)
	if err != nil {
		return "", "", fmt.Errorf("get or create actor: %w", err)
	}

	return admin.ID, actor.ID, nil
}
