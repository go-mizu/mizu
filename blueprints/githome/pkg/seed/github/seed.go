package github

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/comments"
	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/labels"
	"github.com/go-mizu/blueprints/githome/feature/milestones"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	gitpkg "github.com/go-mizu/blueprints/githome/pkg/git"
	"github.com/go-mizu/blueprints/githome/pkg/ulid"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

// SeedConfig configures the seeding process
type SeedConfig struct {
	Owner       string // GitHub owner (user or org)
	Repo        string // GitHub repository name
	LocalOwner  string // Local owner username (will be created if not exists)
	Token       string // GitHub token for higher rate limits (optional)
	MaxIssues   int    // Maximum number of issues to import (0 = all)
	MaxComments int    // Maximum comments per issue (0 = all)
}

// Seeder handles seeding GitHome from GitHub
type Seeder struct {
	db     *sql.DB
	client *Client
	config SeedConfig
	log    *slog.Logger

	// Stores
	usersStore      *duckdb.UsersStore
	reposStore      *duckdb.ReposStore
	issuesStore     *duckdb.IssuesStore
	labelsStore     *duckdb.LabelsStore
	milestonesStore *duckdb.MilestonesStore
	commentsStore   *duckdb.CommentsStore
	actorsStore     *duckdb.ActorsStore

	// ID mappings
	userIDs      map[string]string // GitHub login -> GitHome user ID
	labelIDs     map[string]string // GitHub label name -> GitHome label ID
	milestoneIDs map[int]string    // GitHub milestone number -> GitHome milestone ID
}

// NewSeeder creates a new seeder
func NewSeeder(db *sql.DB, config SeedConfig) (*Seeder, error) {
	return &Seeder{
		db:              db,
		client:          NewClient(config.Token),
		config:          config,
		log:             slog.Default(),
		usersStore:      duckdb.NewUsersStore(db),
		reposStore:      duckdb.NewReposStore(db),
		issuesStore:     duckdb.NewIssuesStore(db),
		labelsStore:     duckdb.NewLabelsStore(db),
		milestonesStore: duckdb.NewMilestonesStore(db),
		commentsStore:   duckdb.NewCommentsStore(db),
		actorsStore:     duckdb.NewActorsStore(db),
		userIDs:         make(map[string]string),
		labelIDs:        make(map[string]string),
		milestoneIDs:    make(map[int]string),
	}, nil
}

// Seed imports data from GitHub into GitHome
func (s *Seeder) Seed(ctx context.Context) error {
	s.log.Info("starting github seed", "owner", s.config.Owner, "repo", s.config.Repo)

	// Step 1: Fetch and create repository
	repo, err := s.seedRepository(ctx)
	if err != nil {
		return fmt.Errorf("seed repository: %w", err)
	}

	// Step 2: Fetch and create labels
	if err := s.seedLabels(ctx, repo.ID); err != nil {
		return fmt.Errorf("seed labels: %w", err)
	}

	// Step 3: Fetch and create milestones
	if err := s.seedMilestones(ctx, repo.ID); err != nil {
		return fmt.Errorf("seed milestones: %w", err)
	}

	// Step 4: Fetch and create issues
	if err := s.seedIssues(ctx, repo.ID); err != nil {
		return fmt.Errorf("seed issues: %w", err)
	}

	s.log.Info("github seed completed successfully")
	return nil
}

// seedRepository fetches and creates the repository
func (s *Seeder) seedRepository(ctx context.Context) (*repos.Repository, error) {
	s.log.Info("fetching repository metadata")

	ghRepo, err := s.client.FetchRepository(ctx, s.config.Owner, s.config.Repo)
	if err != nil {
		return nil, err
	}

	// Create or get owner user
	ownerID, actorID, err := s.ensureUser(ctx, s.config.LocalOwner, ghRepo.Owner.AvatarURL)
	if err != nil {
		return nil, fmt.Errorf("ensure owner: %w", err)
	}

	// Determine license
	license := ""
	if ghRepo.License != nil {
		license = ghRepo.License.SPDXID
	}

	// Create repository
	repo := &repos.Repository{
		ID:             ulid.New(),
		OwnerActorID:   actorID,
		OwnerID:        ownerID,
		OwnerType:      "user",
		OwnerName:      s.config.LocalOwner,
		Name:           ghRepo.Name,
		Slug:           strings.ToLower(ghRepo.Name),
		Description:    ghRepo.Description,
		DefaultBranch:  ghRepo.DefaultBranch,
		IsPrivate:      ghRepo.Private,
		IsArchived:     ghRepo.Archived,
		IsFork:         ghRepo.Fork,
		StarCount:      ghRepo.StargazersCount,
		ForkCount:      ghRepo.ForksCount,
		WatcherCount:   ghRepo.WatchersCount,
		OpenIssueCount: ghRepo.OpenIssuesCount,
		Topics:         ghRepo.Topics,
		License:        license,
		HasIssues:      ghRepo.HasIssues,
		HasWiki:        ghRepo.HasWiki,
		HasProjects:    ghRepo.HasProjects,
		Language:       ghRepo.Language,
		LanguageColor:  gitpkg.LanguageColor(ghRepo.Language),
		CreatedAt:      ghRepo.CreatedAt,
		UpdatedAt:      ghRepo.UpdatedAt,
		PushedAt:       ghRepo.PushedAt,
	}

	if err := s.reposStore.Create(ctx, repo); err != nil {
		return nil, err
	}

	s.log.Info("created repository",
		"name", repo.Name,
		"stars", repo.StarCount,
		"issues", repo.OpenIssueCount)

	return repo, nil
}

// seedLabels fetches and creates labels
func (s *Seeder) seedLabels(ctx context.Context, repoID string) error {
	s.log.Info("fetching labels")

	ghLabels, err := s.client.FetchLabels(ctx, s.config.Owner, s.config.Repo)
	if err != nil {
		return err
	}

	for _, ghLabel := range ghLabels {
		label := &labels.Label{
			ID:          ulid.New(),
			RepoID:      repoID,
			Name:        ghLabel.Name,
			Color:       ghLabel.Color,
			Description: ghLabel.Description,
			CreatedAt:   time.Now(),
		}

		if err := s.labelsStore.Create(ctx, label); err != nil {
			s.log.Warn("failed to create label", "name", label.Name, "error", err)
			continue
		}

		s.labelIDs[ghLabel.Name] = label.ID
	}

	s.log.Info("created labels", "count", len(s.labelIDs))
	return nil
}

// seedMilestones fetches and creates milestones
func (s *Seeder) seedMilestones(ctx context.Context, repoID string) error {
	s.log.Info("fetching milestones")

	ghMilestones, err := s.client.FetchMilestones(ctx, s.config.Owner, s.config.Repo)
	if err != nil {
		return err
	}

	for _, ghMilestone := range ghMilestones {
		milestone := &milestones.Milestone{
			ID:          ulid.New(),
			RepoID:      repoID,
			Number:      ghMilestone.Number,
			Title:       ghMilestone.Title,
			Description: ghMilestone.Description,
			State:       ghMilestone.State,
			DueDate:     ghMilestone.DueOn,
			CreatedAt:   ghMilestone.CreatedAt,
			UpdatedAt:   ghMilestone.UpdatedAt,
			ClosedAt:    ghMilestone.ClosedAt,
		}

		if err := s.milestonesStore.Create(ctx, milestone); err != nil {
			s.log.Warn("failed to create milestone", "title", milestone.Title, "error", err)
			continue
		}

		s.milestoneIDs[ghMilestone.Number] = milestone.ID
	}

	s.log.Info("created milestones", "count", len(s.milestoneIDs))
	return nil
}

// seedIssues fetches and creates issues with comments
func (s *Seeder) seedIssues(ctx context.Context, repoID string) error {
	s.log.Info("fetching issues")

	ghIssues, err := s.client.FetchIssues(ctx, s.config.Owner, s.config.Repo)
	if err != nil {
		return err
	}

	// Limit if configured
	if s.config.MaxIssues > 0 && len(ghIssues) > s.config.MaxIssues {
		ghIssues = ghIssues[:s.config.MaxIssues]
	}

	for i, ghIssue := range ghIssues {
		// Create issue author if needed
		authorID, err := s.ensureUserFromGH(ctx, &ghIssue.User)
		if err != nil {
			s.log.Warn("failed to ensure author", "login", ghIssue.User.Login, "error", err)
			continue
		}

		issue := &issues.Issue{
			ID:             ulid.New(),
			RepoID:         repoID,
			Number:         ghIssue.Number,
			Title:          ghIssue.Title,
			Body:           ghIssue.Body,
			AuthorID:       authorID,
			State:          ghIssue.State,
			StateReason:    ghIssue.StateReason,
			IsLocked:       ghIssue.Locked,
			LockReason:     ghIssue.ActiveLockReason,
			CommentCount:   ghIssue.Comments,
			CreatedAt:      ghIssue.CreatedAt,
			UpdatedAt:      ghIssue.UpdatedAt,
			ClosedAt:       ghIssue.ClosedAt,
		}

		// Set milestone if present
		if ghIssue.Milestone != nil {
			if mid, ok := s.milestoneIDs[ghIssue.Milestone.Number]; ok {
				issue.MilestoneID = mid
			}
		}

		// Set closed by if present
		if ghIssue.ClosedBy != nil {
			closedByID, _ := s.ensureUserFromGH(ctx, ghIssue.ClosedBy)
			issue.ClosedByID = closedByID
		}

		if err := s.issuesStore.Create(ctx, issue); err != nil {
			s.log.Warn("failed to create issue", "number", issue.Number, "error", err)
			continue
		}

		// Add labels
		for _, ghLabel := range ghIssue.Labels {
			if labelID, ok := s.labelIDs[ghLabel.Name]; ok {
				il := &issues.IssueLabel{
					IssueID:   issue.ID,
					LabelID:   labelID,
					CreatedAt: time.Now(),
				}
				s.issuesStore.AddLabel(ctx, il)
			}
		}

		// Add assignees
		for _, ghAssignee := range ghIssue.Assignees {
			assigneeID, _ := s.ensureUserFromGH(ctx, &ghAssignee)
			if assigneeID != "" {
				ia := &issues.IssueAssignee{
					IssueID:   issue.ID,
					UserID:    assigneeID,
					CreatedAt: time.Now(),
				}
				s.issuesStore.AddAssignee(ctx, ia)
			}
		}

		// Fetch and create comments
		if ghIssue.Comments > 0 {
			if err := s.seedIssueComments(ctx, issue.ID, ghIssue.Number); err != nil {
				s.log.Warn("failed to seed comments", "issue", ghIssue.Number, "error", err)
			}
		}

		if (i+1)%10 == 0 {
			s.log.Info("seeding issues progress", "current", i+1, "total", len(ghIssues))
		}
	}

	s.log.Info("created issues", "count", len(ghIssues))
	return nil
}

// seedIssueComments fetches and creates comments for an issue
func (s *Seeder) seedIssueComments(ctx context.Context, issueID string, issueNumber int) error {
	ghComments, err := s.client.FetchComments(ctx, s.config.Owner, s.config.Repo, issueNumber)
	if err != nil {
		return err
	}

	// Limit if configured
	if s.config.MaxComments > 0 && len(ghComments) > s.config.MaxComments {
		ghComments = ghComments[:s.config.MaxComments]
	}

	for _, ghComment := range ghComments {
		authorID, _ := s.ensureUserFromGH(ctx, &ghComment.User)
		if authorID == "" {
			continue
		}

		comment := &comments.Comment{
			ID:         ulid.New(),
			TargetType: "issue",
			TargetID:   issueID,
			UserID:     authorID,
			Body:       ghComment.Body,
			CreatedAt:  ghComment.CreatedAt,
			UpdatedAt:  ghComment.UpdatedAt,
		}

		if err := s.commentsStore.Create(ctx, comment); err != nil {
			s.log.Warn("failed to create comment", "error", err)
		}
	}

	return nil
}

// ensureUser creates or retrieves a local user and returns (userID, actorID)
func (s *Seeder) ensureUser(ctx context.Context, username, avatarURL string) (userID, actorID string, err error) {
	// Check cache
	if id, ok := s.userIDs[username]; ok {
		// For simplicity, we assume actor was created with user
		actor, _ := s.actorsStore.GetByUserID(ctx, id)
		if actor != nil {
			return id, actor.ID, nil
		}
		return id, id, nil
	}

	// Check if user exists
	existingUser, _ := s.usersStore.GetByUsername(ctx, username)
	if existingUser != nil {
		s.userIDs[username] = existingUser.ID
		actor, _ := s.actorsStore.GetByUserID(ctx, existingUser.ID)
		if actor != nil {
			return existingUser.ID, actor.ID, nil
		}
		return existingUser.ID, existingUser.ID, nil
	}

	// Create new user
	userID = ulid.New()
	user := &users.User{
		ID:           userID,
		Username:     username,
		Email:        username + "@githome.local",
		PasswordHash: "$2a$10$XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX", // Placeholder
		AvatarURL:    avatarURL,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.usersStore.Create(ctx, user); err != nil {
		return "", "", err
	}

	// Create actor
	actor, err := s.actorsStore.GetOrCreateForUser(ctx, userID)
	if err != nil {
		s.log.Warn("failed to create actor", "user", username, "error", err)
		actorID = userID // Fallback
	} else {
		actorID = actor.ID
	}

	s.userIDs[username] = userID
	return userID, actorID, nil
}

// ensureUserFromGH creates or retrieves a user from GitHub user data
func (s *Seeder) ensureUserFromGH(ctx context.Context, ghUser *User) (string, error) {
	if ghUser == nil {
		return "", nil
	}
	userID, _, err := s.ensureUser(ctx, ghUser.Login, ghUser.AvatarURL)
	return userID, err
}
