package github

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

// Seeder imports GitHub repository data into GitHome.
type Seeder struct {
	db      *sql.DB
	client  *Client
	crawler *Crawler
	config  Config

	// useCrawler indicates we should use the crawler instead of the API
	useCrawler bool

	// Stores
	usersStore      *duckdb.UsersStore
	orgsStore       *duckdb.OrgsStore
	reposStore      *duckdb.ReposStore
	issuesStore     *duckdb.IssuesStore
	pullsStore      *duckdb.PullsStore
	commentsStore   *duckdb.CommentsStore
	labelsStore     *duckdb.LabelsStore
	milestonesStore *duckdb.MilestonesStore

	// Cache maps GitHub logins to local IDs
	userCache      map[string]int64
	labelCache     map[string]int64
	milestoneCache map[int]int64 // GitHub milestone number -> local ID
}

// NewSeeder creates a new GitHub seeder.
func NewSeeder(db *sql.DB, config Config) *Seeder {
	client := NewClient(config.BaseURL, config.Token)
	crawler := NewCrawler(config.BaseURL)

	return &Seeder{
		db:              db,
		client:          client,
		crawler:         crawler,
		config:          config,
		usersStore:      duckdb.NewUsersStore(db),
		orgsStore:       duckdb.NewOrgsStore(db),
		reposStore:      duckdb.NewReposStore(db),
		issuesStore:     duckdb.NewIssuesStore(db),
		pullsStore:      duckdb.NewPullsStore(db),
		commentsStore:   duckdb.NewCommentsStore(db),
		labelsStore:     duckdb.NewLabelsStore(db),
		milestonesStore: duckdb.NewMilestonesStore(db),
		userCache:       make(map[string]int64),
		labelCache:      make(map[string]int64),
		milestoneCache:  make(map[int]int64),
	}
}

// Seed imports GitHub repository data into GitHome.
func (s *Seeder) Seed(ctx context.Context) (*Result, error) {
	result := &Result{}

	slog.Info("starting GitHub seed", "owner", s.config.Owner, "repo", s.config.Repo)

	// Validate token if provided
	if s.config.Token != "" {
		valid, err := s.client.ValidateToken(ctx)
		if err != nil {
			slog.Warn("failed to validate token, proceeding anyway", "error", err)
		} else if !valid {
			slog.Warn("GitHub token is invalid or expired, falling back to unauthenticated API (60 requests/hour)")
			s.client.ClearToken()
		} else {
			slog.Info("GitHub token validated successfully")
		}
	}

	// 1. Fetch repository metadata
	ghRepo, rateInfo, err := s.client.GetRepository(ctx, s.config.Owner, s.config.Repo)
	if err != nil {
		if IsFallbackError(err) {
			slog.Warn("API failed, switching to crawler fallback", "error", err)
			s.useCrawler = true
			result.UsedCrawler = true
			ghRepo, err = s.crawler.FetchRepository(ctx, s.config.Owner, s.config.Repo)
			if err != nil {
				return nil, fmt.Errorf("fetch repository (crawler): %w", err)
			}
		} else {
			return nil, fmt.Errorf("fetch repository: %w", err)
		}
	}
	s.updateRateInfo(result, rateInfo)

	slog.Info("fetched repository", "name", ghRepo.FullName, "stars", ghRepo.StargazersCount, "issues", ghRepo.OpenIssuesCount)

	// 2. Ensure owner exists (user or org)
	ownerID, ownerType, err := s.ensureOwner(ctx, ghRepo.Owner, result)
	if err != nil {
		return nil, fmt.Errorf("ensure owner: %w", err)
	}

	// 3. Create repository
	repo, err := s.ensureRepository(ctx, ghRepo, ownerID, ownerType, result)
	if err != nil {
		return nil, fmt.Errorf("ensure repository: %w", err)
	}

	slog.Info("created/found repository", "id", repo.ID, "name", repo.Name)

	// 4. Import labels (needed for issues/PRs)
	if s.config.ImportLabels {
		if err := s.importLabels(ctx, repo.ID, result); err != nil {
			slog.Warn("failed to import labels", "error", err)
			result.Errors = append(result.Errors, fmt.Errorf("import labels: %w", err))
		}
	}

	// 5. Import milestones (needed for issues/PRs)
	if s.config.ImportMilestones {
		if err := s.importMilestones(ctx, repo.ID, result); err != nil {
			slog.Warn("failed to import milestones", "error", err)
			result.Errors = append(result.Errors, fmt.Errorf("import milestones: %w", err))
		}
	}

	// 6. Import issues
	if s.config.ImportIssues {
		if err := s.importIssues(ctx, repo.ID, result); err != nil {
			slog.Warn("failed to import issues", "error", err)
			result.Errors = append(result.Errors, fmt.Errorf("import issues: %w", err))
		}
	}

	// 7. Import pull requests
	if s.config.ImportPRs {
		if err := s.importPullRequests(ctx, repo.ID, result); err != nil {
			slog.Warn("failed to import pull requests", "error", err)
			result.Errors = append(result.Errors, fmt.Errorf("import pull requests: %w", err))
		}
	}

	slog.Info("GitHub seed complete",
		"issues", result.IssuesCreated,
		"prs", result.PRsCreated,
		"comments", result.CommentsCreated,
		"labels", result.LabelsCreated,
		"milestones", result.MilestonesCreated,
		"users", result.UsersCreated,
		"errors", len(result.Errors),
		"usedCrawler", result.UsedCrawler)

	return result, nil
}

// ensureOwner creates or retrieves the repository owner.
func (s *Seeder) ensureOwner(ctx context.Context, ghOwner *ghUser, result *Result) (int64, string, error) {
	if ghOwner == nil {
		return s.config.AdminUserID, "User", nil
	}

	if ghOwner.Type == "Organization" {
		// Check if org exists
		existing, err := s.orgsStore.GetByLogin(ctx, ghOwner.Login)
		if err != nil {
			return 0, "", err
		}
		if existing != nil {
			return existing.ID, "Organization", nil
		}

		// Create organization
		org := mapOrganization(ghOwner)
		if err := s.orgsStore.Create(ctx, org); err != nil {
			return 0, "", err
		}

		// Add admin user as org owner
		if s.config.AdminUserID > 0 {
			if err := s.orgsStore.AddMember(ctx, org.ID, s.config.AdminUserID, "admin", true); err != nil {
				slog.Warn("failed to add admin to org", "org", org.Login, "error", err)
			}
		}

		result.OrgCreated = true
		slog.Info("created organization", "login", org.Login)
		return org.ID, "Organization", nil
	}

	// It's a user
	userID, err := s.ensureUser(ctx, ghOwner, result)
	if err != nil {
		return 0, "", err
	}
	return userID, "User", nil
}

// ensureUser creates or retrieves a user.
func (s *Seeder) ensureUser(ctx context.Context, ghUser *ghUser, result *Result) (int64, error) {
	if ghUser == nil {
		return s.config.AdminUserID, nil
	}

	// Check cache
	if id, ok := s.userCache[ghUser.Login]; ok {
		return id, nil
	}

	// Check if user exists
	existing, err := s.usersStore.GetByLogin(ctx, ghUser.Login)
	if err != nil {
		return 0, err
	}
	if existing != nil {
		s.userCache[ghUser.Login] = existing.ID
		return existing.ID, nil
	}

	// Create user
	user := mapUser(ghUser)
	if err := s.usersStore.Create(ctx, user); err != nil {
		return 0, err
	}

	s.userCache[ghUser.Login] = user.ID
	result.UsersCreated++
	slog.Debug("created user", "login", user.Login)
	return user.ID, nil
}

// ensureRepository creates or retrieves the repository.
func (s *Seeder) ensureRepository(ctx context.Context, gh *ghRepository, ownerID int64, ownerType string, result *Result) (*repos.Repository, error) {
	// Check if repo exists
	existing, err := s.reposStore.GetByOwnerAndName(ctx, ownerID, gh.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// Create repository
	repo := mapRepository(gh, ownerID, ownerType, s.config.IsPublic)
	if err := s.reposStore.Create(ctx, repo); err != nil {
		return nil, err
	}

	result.RepoCreated = true
	return repo, nil
}

// importLabels fetches and imports all labels.
func (s *Seeder) importLabels(ctx context.Context, repoID int64, result *Result) error {
	page := 1
	for {
		var ghLabels []*ghLabel
		var err error

		if s.useCrawler {
			ghLabels, err = s.crawler.FetchLabels(ctx, s.config.Owner, s.config.Repo)
		} else {
			var rateInfo *RateLimitInfo
			ghLabels, rateInfo, err = s.client.ListLabels(ctx, s.config.Owner, s.config.Repo, &ListOptions{
				Page:    page,
				PerPage: 100,
			})
			s.updateRateInfo(result, rateInfo)

			// Switch to crawler on rate limit or auth error
			if IsFallbackError(err) {
				slog.Warn("API failed during labels import, switching to crawler", "error", err)
				s.useCrawler = true
				result.UsedCrawler = true
				ghLabels, err = s.crawler.FetchLabels(ctx, s.config.Owner, s.config.Repo)
			}
		}
		if err != nil {
			return err
		}

		if len(ghLabels) == 0 {
			break
		}

		for _, ghLabel := range ghLabels {
			// Check if label exists
			existing, err := s.labelsStore.GetByName(ctx, repoID, ghLabel.Name)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("check label %s: %w", ghLabel.Name, err))
				continue
			}
			if existing != nil {
				s.labelCache[ghLabel.Name] = existing.ID
				continue
			}

			// Create label
			label := mapLabel(ghLabel, repoID)
			if err := s.labelsStore.Create(ctx, label); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("create label %s: %w", ghLabel.Name, err))
				continue
			}

			s.labelCache[ghLabel.Name] = label.ID
			result.LabelsCreated++
			slog.Debug("created label", "name", label.Name)
		}

		// Crawler fetches all at once, break after first iteration
		if s.useCrawler || len(ghLabels) < 100 {
			break
		}
		page++
	}

	slog.Info("imported labels", "count", result.LabelsCreated)
	return nil
}

// importMilestones fetches and imports all milestones.
func (s *Seeder) importMilestones(ctx context.Context, repoID int64, result *Result) error {
	// Milestones are not supported via crawler, skip if in crawler mode
	if s.useCrawler {
		slog.Info("skipping milestones import (crawler mode)")
		return nil
	}

	page := 1
	for {
		ghMilestones, rateInfo, err := s.client.ListMilestones(ctx, s.config.Owner, s.config.Repo, &ListOptions{
			Page:    page,
			PerPage: 100,
			State:   "all",
		})
		s.updateRateInfo(result, rateInfo)

		// Switch to crawler on rate limit or auth error (but skip milestones)
		if IsFallbackError(err) {
			slog.Warn("API failed during milestones import, switching to crawler (skipping milestones)", "error", err)
			s.useCrawler = true
			result.UsedCrawler = true
			return nil
		}
		if err != nil {
			return err
		}

		if len(ghMilestones) == 0 {
			break
		}

		for _, ghMilestone := range ghMilestones {
			// Check if milestone exists
			existing, err := s.milestonesStore.GetByNumber(ctx, repoID, ghMilestone.Number)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("check milestone %d: %w", ghMilestone.Number, err))
				continue
			}
			if existing != nil {
				s.milestoneCache[ghMilestone.Number] = existing.ID
				continue
			}

			// Ensure creator exists
			creatorID := s.config.AdminUserID
			if ghMilestone.Creator != nil {
				id, err := s.ensureUser(ctx, ghMilestone.Creator, result)
				if err != nil {
					slog.Warn("failed to ensure milestone creator", "error", err)
				} else {
					creatorID = id
				}
			}

			// Create milestone
			milestone := mapMilestone(ghMilestone, repoID, creatorID)
			if err := s.milestonesStore.Create(ctx, milestone); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("create milestone %d: %w", ghMilestone.Number, err))
				continue
			}

			s.milestoneCache[ghMilestone.Number] = milestone.ID
			result.MilestonesCreated++
			slog.Debug("created milestone", "number", milestone.Number, "title", milestone.Title)
		}

		if len(ghMilestones) < 100 {
			break
		}
		page++
	}

	slog.Info("imported milestones", "count", result.MilestonesCreated)
	return nil
}

// importIssues fetches and imports all issues.
func (s *Seeder) importIssues(ctx context.Context, repoID int64, result *Result) error {
	// Handle single issue import
	if s.config.SingleIssue > 0 {
		return s.importSingleIssue(ctx, repoID, s.config.SingleIssue, result)
	}

	page := 1
	total := 0
	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			slog.Info("issues import cancelled")
			return ctx.Err()
		default:
		}

		// Check max limit
		if s.config.MaxIssues > 0 && total >= s.config.MaxIssues {
			break
		}

		var ghIssues []*ghIssue
		var err error

		if s.useCrawler {
			ghIssues, err = s.crawler.FetchIssues(ctx, s.config.Owner, s.config.Repo, page, "all")
		} else {
			var rateInfo *RateLimitInfo
			ghIssues, rateInfo, err = s.client.ListIssues(ctx, s.config.Owner, s.config.Repo, &ListOptions{
				Page:    page,
				PerPage: 100,
				State:   "all",
			})
			s.updateRateInfo(result, rateInfo)

			// Switch to crawler on rate limit or auth error
			if IsFallbackError(err) {
				slog.Warn("API failed during issues import, switching to crawler", "error", err)
				s.useCrawler = true
				result.UsedCrawler = true
				ghIssues, err = s.crawler.FetchIssues(ctx, s.config.Owner, s.config.Repo, page, "all")
			}
		}
		if err != nil {
			return err
		}

		if len(ghIssues) == 0 {
			break
		}

		for _, ghIssue := range ghIssues {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				slog.Info("issues import cancelled")
				return ctx.Err()
			default:
			}

			// Skip if it's a PR (GitHub API returns PRs in issues list)
			if ghIssue.PullRequest != nil {
				continue
			}

			// Check max limit
			if s.config.MaxIssues > 0 && total >= s.config.MaxIssues {
				break
			}

			// Check if issue exists
			existing, err := s.issuesStore.GetByNumber(ctx, repoID, ghIssue.Number)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("check issue #%d: %w", ghIssue.Number, err))
				continue
			}
			if existing != nil {
				result.IssuesSkipped++
				continue
			}

			// Ensure creator exists
			creatorID := s.config.AdminUserID
			if ghIssue.User != nil {
				id, err := s.ensureUser(ctx, ghIssue.User, result)
				if err != nil {
					slog.Warn("failed to ensure issue creator", "error", err)
				} else {
					creatorID = id
				}
			}

			// Create issue
			issue := mapIssue(ghIssue, repoID, creatorID)
			if err := s.issuesStore.Create(ctx, issue); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("create issue #%d: %w", ghIssue.Number, err))
				continue
			}

			result.IssuesCreated++
			total++

			// Add labels
			for _, ghLabel := range ghIssue.Labels {
				if labelID, ok := s.labelCache[ghLabel.Name]; ok {
					if err := s.labelsStore.AddToIssue(ctx, issue.ID, labelID); err != nil {
						slog.Warn("failed to add label to issue", "issue", issue.Number, "label", ghLabel.Name, "error", err)
					}
				}
			}

			// Set milestone
			if ghIssue.Milestone != nil {
				if milestoneID, ok := s.milestoneCache[ghIssue.Milestone.Number]; ok {
					if err := s.issuesStore.SetMilestone(ctx, issue.ID, &milestoneID); err != nil {
						slog.Warn("failed to set milestone", "issue", issue.Number, "error", err)
					}
				}
			}

			// Add assignees
			for _, assignee := range ghIssue.Assignees {
				assigneeID, err := s.ensureUser(ctx, assignee, result)
				if err != nil {
					continue
				}
				if err := s.issuesStore.AddAssignee(ctx, issue.ID, assigneeID); err != nil {
					slog.Warn("failed to add assignee", "issue", issue.Number, "assignee", assignee.Login, "error", err)
				}
			}

			// Import comments
			if s.config.ImportComments && ghIssue.Comments > 0 {
				if err := s.importIssueComments(ctx, repoID, issue.ID, ghIssue.Number, result); err != nil {
					slog.Warn("failed to import issue comments", "issue", ghIssue.Number, "error", err)
				}
			}

			if total%100 == 0 {
				slog.Info("importing issues", "progress", total)
			}
		}

		if len(ghIssues) < 100 {
			break
		}
		page++
	}

	slog.Info("imported issues", "created", result.IssuesCreated, "skipped", result.IssuesSkipped)
	return nil
}

// importSingleIssue fetches and imports a single issue by number.
func (s *Seeder) importSingleIssue(ctx context.Context, repoID int64, issueNumber int, result *Result) error {
	slog.Info("fetching single issue", "number", issueNumber)

	ghIssue, rateInfo, err := s.client.GetIssue(ctx, s.config.Owner, s.config.Repo, issueNumber)
	s.updateRateInfo(result, rateInfo)
	if err != nil {
		return fmt.Errorf("fetch issue #%d: %w", issueNumber, err)
	}

	// Check if issue exists
	existing, err := s.issuesStore.GetByNumber(ctx, repoID, ghIssue.Number)
	if err != nil {
		return fmt.Errorf("check issue #%d: %w", ghIssue.Number, err)
	}
	if existing != nil {
		slog.Info("issue already exists, updating", "number", ghIssue.Number)
		// Update existing issue
		updateIn := &issues.UpdateIn{
			Title: &ghIssue.Title,
			Body:  &ghIssue.Body,
			State: &ghIssue.State,
		}
		if err := s.issuesStore.Update(ctx, existing.ID, updateIn); err != nil {
			slog.Warn("failed to update issue", "error", err)
		}
		result.IssuesSkipped++
	} else {
		// Ensure creator exists
		creatorID := s.config.AdminUserID
		if ghIssue.User != nil {
			id, err := s.ensureUser(ctx, ghIssue.User, result)
			if err != nil {
				slog.Warn("failed to ensure issue creator", "error", err)
			} else {
				creatorID = id
			}
		}

		// Create issue
		issue := mapIssue(ghIssue, repoID, creatorID)
		if err := s.issuesStore.Create(ctx, issue); err != nil {
			return fmt.Errorf("create issue #%d: %w", ghIssue.Number, err)
		}

		result.IssuesCreated++

		// Add labels
		for _, ghLabel := range ghIssue.Labels {
			if labelID, ok := s.labelCache[ghLabel.Name]; ok {
				if err := s.labelsStore.AddToIssue(ctx, issue.ID, labelID); err != nil {
					slog.Warn("failed to add label to issue", "issue", issue.Number, "label", ghLabel.Name, "error", err)
				}
			}
		}

		// Set milestone
		if ghIssue.Milestone != nil {
			if milestoneID, ok := s.milestoneCache[ghIssue.Milestone.Number]; ok {
				if err := s.issuesStore.SetMilestone(ctx, issue.ID, &milestoneID); err != nil {
					slog.Warn("failed to set milestone", "issue", issue.Number, "error", err)
				}
			}
		}

		// Add assignees
		for _, assignee := range ghIssue.Assignees {
			assigneeID, err := s.ensureUser(ctx, assignee, result)
			if err != nil {
				continue
			}
			if err := s.issuesStore.AddAssignee(ctx, issue.ID, assigneeID); err != nil {
				slog.Warn("failed to add assignee", "issue", issue.Number, "assignee", assignee.Login, "error", err)
			}
		}

		existing = issue
	}

	// Import comments
	if s.config.ImportComments && ghIssue.Comments > 0 {
		if err := s.importIssueComments(ctx, repoID, existing.ID, ghIssue.Number, result); err != nil {
			slog.Warn("failed to import issue comments", "issue", ghIssue.Number, "error", err)
		}
	}

	slog.Info("imported single issue", "number", issueNumber, "comments", ghIssue.Comments)
	return nil
}

// importIssueComments fetches and imports comments for an issue.
func (s *Seeder) importIssueComments(ctx context.Context, repoID, issueID int64, issueNumber int, result *Result) error {
	page := 1
	total := 0
	for {
		// Check max limit
		if s.config.MaxCommentsPerItem > 0 && total >= s.config.MaxCommentsPerItem {
			break
		}

		var ghComments []*ghComment
		var err error

		if s.useCrawler {
			// Crawler fetches all comments at once
			ghComments, err = s.crawler.FetchComments(ctx, s.config.Owner, s.config.Repo, issueNumber)
			if err != nil {
				return err
			}
		} else {
			var rateInfo *RateLimitInfo
			ghComments, rateInfo, err = s.client.ListIssueComments(ctx, s.config.Owner, s.config.Repo, issueNumber, &ListOptions{
				Page:    page,
				PerPage: 100,
			})
			s.updateRateInfo(result, rateInfo)

			// Switch to crawler on rate limit or auth error
			if IsFallbackError(err) {
				slog.Warn("API failed during comments import, switching to crawler", "error", err)
				s.useCrawler = true
				result.UsedCrawler = true
				ghComments, err = s.crawler.FetchComments(ctx, s.config.Owner, s.config.Repo, issueNumber)
			}
			if err != nil {
				return err
			}
		}

		if len(ghComments) == 0 {
			break
		}

		for _, ghComment := range ghComments {
			// Check max limit
			if s.config.MaxCommentsPerItem > 0 && total >= s.config.MaxCommentsPerItem {
				break
			}

			// Ensure creator exists
			creatorID := s.config.AdminUserID
			if ghComment.User != nil {
				id, err := s.ensureUser(ctx, ghComment.User, result)
				if err != nil {
					slog.Warn("failed to ensure comment creator", "error", err)
				} else {
					creatorID = id
				}
			}

			// Create comment
			comment := mapIssueComment(ghComment, issueID, repoID, creatorID)
			if err := s.commentsStore.CreateIssueComment(ctx, comment); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("create comment for issue #%d: %w", issueNumber, err))
				continue
			}

			result.CommentsCreated++
			total++
		}

		// Crawler fetches all at once, break after first iteration
		if s.useCrawler || len(ghComments) < 100 {
			break
		}
		page++
	}

	return nil
}

// importPullRequests fetches and imports all pull requests.
func (s *Seeder) importPullRequests(ctx context.Context, repoID int64, result *Result) error {
	// Handle single PR import
	if s.config.SinglePR > 0 {
		return s.importSinglePR(ctx, repoID, s.config.SinglePR, result)
	}

	page := 1
	total := 0
	for {
		// Check max limit
		if s.config.MaxPRs > 0 && total >= s.config.MaxPRs {
			break
		}

		var ghPRs []*ghPullRequest
		var err error

		if s.useCrawler {
			ghPRs, err = s.crawler.FetchPullRequests(ctx, s.config.Owner, s.config.Repo, page, "all")
		} else {
			var rateInfo *RateLimitInfo
			ghPRs, rateInfo, err = s.client.ListPullRequests(ctx, s.config.Owner, s.config.Repo, &ListOptions{
				Page:    page,
				PerPage: 100,
				State:   "all",
			})
			s.updateRateInfo(result, rateInfo)

			// Switch to crawler on rate limit or auth error
			if IsFallbackError(err) {
				slog.Warn("API failed during PRs import, switching to crawler", "error", err)
				s.useCrawler = true
				result.UsedCrawler = true
				ghPRs, err = s.crawler.FetchPullRequests(ctx, s.config.Owner, s.config.Repo, page, "all")
			}
		}
		if err != nil {
			return err
		}

		if len(ghPRs) == 0 {
			break
		}

		for _, ghPR := range ghPRs {
			// Check max limit
			if s.config.MaxPRs > 0 && total >= s.config.MaxPRs {
				break
			}

			// Check if PR exists
			existing, err := s.pullsStore.GetByNumber(ctx, repoID, ghPR.Number)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("check PR #%d: %w", ghPR.Number, err))
				continue
			}
			if existing != nil {
				result.PRsSkipped++
				continue
			}

			// Ensure creator exists
			creatorID := s.config.AdminUserID
			if ghPR.User != nil {
				id, err := s.ensureUser(ctx, ghPR.User, result)
				if err != nil {
					slog.Warn("failed to ensure PR creator", "error", err)
				} else {
					creatorID = id
				}
			}

			// Create PR
			pr := mapPullRequest(ghPR, repoID, creatorID)
			if err := s.pullsStore.Create(ctx, pr); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("create PR #%d: %w", ghPR.Number, err))
				continue
			}

			result.PRsCreated++
			total++

			// Handle merged PRs
			if ghPR.Merged && ghPR.MergedAt != nil {
				mergedByID := s.config.AdminUserID
				if ghPR.MergedBy != nil {
					id, err := s.ensureUser(ctx, ghPR.MergedBy, result)
					if err == nil {
						mergedByID = id
					}
				}
				if err := s.pullsStore.SetMerged(ctx, pr.ID, *ghPR.MergedAt, ghPR.MergeCommitSHA, mergedByID); err != nil {
					slog.Warn("failed to set PR merged status", "pr", ghPR.Number, "error", err)
				}
			}

			// Add assignees
			for _, assignee := range ghPR.Assignees {
				assigneeID, err := s.ensureUser(ctx, assignee, result)
				if err != nil {
					continue
				}
				// PRs use pr_assignees table - we'll skip this as it's not implemented
				_ = assigneeID
			}

			// Import PR review comments
			if s.config.ImportComments && ghPR.ReviewComments > 0 {
				if err := s.importPRComments(ctx, pr.ID, ghPR.Number, result); err != nil {
					slog.Warn("failed to import PR comments", "pr", ghPR.Number, "error", err)
				}
			}

			if total%100 == 0 {
				slog.Info("importing pull requests", "progress", total)
			}
		}

		if len(ghPRs) < 100 {
			break
		}
		page++
	}

	slog.Info("imported pull requests", "created", result.PRsCreated, "skipped", result.PRsSkipped)
	return nil
}

// importPRComments fetches and imports review comments for a PR.
func (s *Seeder) importPRComments(ctx context.Context, prID int64, prNumber int, result *Result) error {
	page := 1
	total := 0
	for {
		// Check max limit
		if s.config.MaxCommentsPerItem > 0 && total >= s.config.MaxCommentsPerItem {
			break
		}

		ghComments, rateInfo, err := s.client.ListPRComments(ctx, s.config.Owner, s.config.Repo, prNumber, &ListOptions{
			Page:    page,
			PerPage: 100,
		})
		s.updateRateInfo(result, rateInfo)
		if err != nil {
			return err
		}

		if len(ghComments) == 0 {
			break
		}

		for _, ghComment := range ghComments {
			// Check max limit
			if s.config.MaxCommentsPerItem > 0 && total >= s.config.MaxCommentsPerItem {
				break
			}

			// Ensure creator exists
			creatorID := s.config.AdminUserID
			if ghComment.User != nil {
				id, err := s.ensureUser(ctx, ghComment.User, result)
				if err != nil {
					slog.Warn("failed to ensure PR comment creator", "error", err)
				} else {
					creatorID = id
				}
			}

			// Create review comment
			comment := mapReviewComment(ghComment, prID, creatorID)
			if err := s.pullsStore.CreateReviewComment(ctx, comment); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("create PR comment for #%d: %w", prNumber, err))
				continue
			}

			result.CommentsCreated++
			total++
		}

		if len(ghComments) < 100 {
			break
		}
		page++
	}

	return nil
}

// importSinglePR fetches and imports a single pull request with full details.
func (s *Seeder) importSinglePR(ctx context.Context, repoID int64, prNumber int, result *Result) error {
	slog.Info("fetching single pull request", "number", prNumber)

	// Fetch full PR details (includes commits, additions, deletions, changed_files)
	ghPR, rateInfo, err := s.client.GetPullRequest(ctx, s.config.Owner, s.config.Repo, prNumber)
	s.updateRateInfo(result, rateInfo)
	if err != nil {
		return fmt.Errorf("fetch PR #%d: %w", prNumber, err)
	}

	slog.Info("fetched PR details",
		"number", ghPR.Number,
		"commits", ghPR.Commits,
		"additions", ghPR.Additions,
		"deletions", ghPR.Deletions,
		"changed_files", ghPR.ChangedFiles,
		"comments", ghPR.Comments,
		"review_comments", ghPR.ReviewComments)

	// Ensure creator exists
	creatorID := s.config.AdminUserID
	if ghPR.User != nil {
		id, err := s.ensureUser(ctx, ghPR.User, result)
		if err != nil {
			slog.Warn("failed to ensure PR creator", "error", err)
		} else {
			creatorID = id
		}
	}

	// Check if PR exists
	existing, err := s.pullsStore.GetByNumber(ctx, repoID, ghPR.Number)
	if err != nil {
		return fmt.Errorf("check PR #%d: %w", ghPR.Number, err)
	}

	var prID int64
	if existing != nil {
		slog.Info("PR already exists, updating with latest data", "number", ghPR.Number)
		prID = existing.ID

		// Update PR with full details - update additions, deletions, changed_files, commits count
		_, err = s.db.ExecContext(ctx, `
			UPDATE pull_requests SET
				title = $2,
				body = $3,
				state = $4,
				draft = $5,
				commits = $6,
				additions = $7,
				deletions = $8,
				changed_files = $9,
				comments = $10,
				review_comments = $11,
				updated_at = $12
			WHERE id = $1
		`, prID, ghPR.Title, ghPR.Body, ghPR.State, ghPR.Draft,
			ghPR.Commits, ghPR.Additions, ghPR.Deletions, ghPR.ChangedFiles,
			ghPR.Comments, ghPR.ReviewComments, time.Now())
		if err != nil {
			slog.Warn("failed to update PR", "error", err)
		}

		result.PRsSkipped++
	} else {
		// Create PR
		pr := mapPullRequest(ghPR, repoID, creatorID)
		if err := s.pullsStore.Create(ctx, pr); err != nil {
			return fmt.Errorf("create PR #%d: %w", ghPR.Number, err)
		}
		prID = pr.ID
		result.PRsCreated++

		// Handle merged PRs
		if ghPR.Merged && ghPR.MergedAt != nil {
			mergedByID := s.config.AdminUserID
			if ghPR.MergedBy != nil {
				id, err := s.ensureUser(ctx, ghPR.MergedBy, result)
				if err == nil {
					mergedByID = id
				}
			}
			if err := s.pullsStore.SetMerged(ctx, prID, *ghPR.MergedAt, ghPR.MergeCommitSHA, mergedByID); err != nil {
				slog.Warn("failed to set PR merged status", "pr", ghPR.Number, "error", err)
			}
		}
	}

	// Add labels
	for _, ghLabel := range ghPR.Labels {
		if labelID, ok := s.labelCache[ghLabel.Name]; ok {
			if err := s.addPRLabel(ctx, prID, labelID); err != nil {
				slog.Warn("failed to add label to PR", "pr", ghPR.Number, "label", ghLabel.Name, "error", err)
			}
		}
	}

	// Add assignees
	for _, assignee := range ghPR.Assignees {
		assigneeID, err := s.ensureUser(ctx, assignee, result)
		if err != nil {
			continue
		}
		if err := s.addPRAssignee(ctx, prID, assigneeID); err != nil {
			slog.Warn("failed to add assignee to PR", "pr", ghPR.Number, "assignee", assignee.Login, "error", err)
		}
	}

	// Import PR reviews
	if err := s.importPRReviews(ctx, prID, prNumber, result); err != nil {
		slog.Warn("failed to import PR reviews", "pr", prNumber, "error", err)
	}

	// Import PR review comments (inline comments)
	if s.config.ImportComments {
		if err := s.importPRComments(ctx, prID, prNumber, result); err != nil {
			slog.Warn("failed to import PR review comments", "pr", prNumber, "error", err)
		}
	}

	// Import issue comments (PR conversation comments use issue comments API)
	if s.config.ImportComments && ghPR.Comments > 0 {
		if err := s.importPRIssueComments(ctx, repoID, prID, prNumber, result); err != nil {
			slog.Warn("failed to import PR issue comments", "pr", prNumber, "error", err)
		}
	}

	// Import PR commits
	if ghPR.Commits > 0 {
		if err := s.importPRCommits(ctx, prID, prNumber, result); err != nil {
			slog.Warn("failed to import PR commits", "pr", prNumber, "error", err)
		}
	}

	// Import PR files
	if ghPR.ChangedFiles > 0 {
		if err := s.importPRFiles(ctx, prID, prNumber, result); err != nil {
			slog.Warn("failed to import PR files", "pr", prNumber, "error", err)
		}
	}

	slog.Info("imported single PR",
		"number", prNumber,
		"commits", ghPR.Commits,
		"additions", ghPR.Additions,
		"deletions", ghPR.Deletions,
		"changed_files", ghPR.ChangedFiles,
		"reviews", result.PRReviewsCreated,
		"comments", ghPR.Comments+ghPR.ReviewComments)

	return nil
}

// importPRCommits fetches and imports commits for a PR.
func (s *Seeder) importPRCommits(ctx context.Context, prID int64, prNumber int, result *Result) error {
	page := 1
	for {
		ghCommits, rateInfo, err := s.client.ListPRCommits(ctx, s.config.Owner, s.config.Repo, prNumber, &ListOptions{
			Page:    page,
			PerPage: 100,
		})
		s.updateRateInfo(result, rateInfo)
		if err != nil {
			return err
		}

		if len(ghCommits) == 0 {
			break
		}

		for _, ghCommit := range ghCommits {
			// Look up author and committer IDs if available
			var authorID, committerID *int64
			if ghCommit.Author != nil {
				if id, err := s.ensureUser(ctx, ghCommit.Author, result); err == nil {
					authorID = &id
				}
			}
			if ghCommit.Committer != nil {
				if id, err := s.ensureUser(ctx, ghCommit.Committer, result); err == nil {
					committerID = &id
				}
			}

			commit := mapPRCommit(ghCommit, authorID, committerID)
			if err := s.pullsStore.CreateCommit(ctx, prID, commit); err != nil {
				slog.Warn("failed to create PR commit", "pr", prNumber, "sha", ghCommit.SHA, "error", err)
				continue
			}
			result.PRCommitsCreated++
		}

		if len(ghCommits) < 100 {
			break
		}
		page++
	}

	return nil
}

// importPRFiles fetches and imports files for a PR.
func (s *Seeder) importPRFiles(ctx context.Context, prID int64, prNumber int, result *Result) error {
	page := 1
	for {
		ghFiles, rateInfo, err := s.client.ListPRFiles(ctx, s.config.Owner, s.config.Repo, prNumber, &ListOptions{
			Page:    page,
			PerPage: 100,
		})
		s.updateRateInfo(result, rateInfo)
		if err != nil {
			return err
		}

		if len(ghFiles) == 0 {
			break
		}

		for _, ghFile := range ghFiles {
			file := mapPRFile(ghFile)
			if err := s.pullsStore.CreateFile(ctx, prID, file); err != nil {
				slog.Warn("failed to create PR file", "pr", prNumber, "file", ghFile.Filename, "error", err)
				continue
			}
			result.PRFilesCreated++
		}

		if len(ghFiles) < 100 {
			break
		}
		page++
	}

	return nil
}

// importPRReviews fetches and imports reviews for a PR.
func (s *Seeder) importPRReviews(ctx context.Context, prID int64, prNumber int, result *Result) error {
	page := 1
	for {
		ghReviews, rateInfo, err := s.client.ListPRReviews(ctx, s.config.Owner, s.config.Repo, prNumber, &ListOptions{
			Page:    page,
			PerPage: 100,
		})
		s.updateRateInfo(result, rateInfo)
		if err != nil {
			return err
		}

		if len(ghReviews) == 0 {
			break
		}

		for _, ghReview := range ghReviews {
			// Ensure reviewer exists
			reviewerID := s.config.AdminUserID
			if ghReview.User != nil {
				id, err := s.ensureUser(ctx, ghReview.User, result)
				if err != nil {
					slog.Warn("failed to ensure reviewer", "error", err)
				} else {
					reviewerID = id
				}
			}

			// Create review
			review := mapReview(ghReview, prID, reviewerID)
			if err := s.pullsStore.CreateReview(ctx, review); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("create review for PR #%d: %w", prNumber, err))
				continue
			}

			result.PRReviewsCreated++
		}

		if len(ghReviews) < 100 {
			break
		}
		page++
	}

	return nil
}

// importPRIssueComments fetches and imports issue comments for a PR (PR conversation).
func (s *Seeder) importPRIssueComments(ctx context.Context, repoID, prID int64, prNumber int, result *Result) error {
	page := 1
	total := 0
	for {
		// Check max limit
		if s.config.MaxCommentsPerItem > 0 && total >= s.config.MaxCommentsPerItem {
			break
		}

		ghComments, rateInfo, err := s.client.ListIssueComments(ctx, s.config.Owner, s.config.Repo, prNumber, &ListOptions{
			Page:    page,
			PerPage: 100,
		})
		s.updateRateInfo(result, rateInfo)
		if err != nil {
			return err
		}

		if len(ghComments) == 0 {
			break
		}

		for _, ghComment := range ghComments {
			// Check max limit
			if s.config.MaxCommentsPerItem > 0 && total >= s.config.MaxCommentsPerItem {
				break
			}

			// Ensure creator exists
			creatorID := s.config.AdminUserID
			if ghComment.User != nil {
				id, err := s.ensureUser(ctx, ghComment.User, result)
				if err != nil {
					slog.Warn("failed to ensure PR comment creator", "error", err)
				} else {
					creatorID = id
				}
			}

			// Create comment - we store PR conversation comments as issue comments
			// but link them to the PR by using the PR ID as the issue ID
			// This matches GitHub's API behavior where PR conversations are fetched via issues API
			comment := mapPRIssueComment(ghComment, prID, repoID, creatorID)
			if err := s.commentsStore.CreateIssueComment(ctx, comment); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("create PR issue comment for #%d: %w", prNumber, err))
				continue
			}

			result.CommentsCreated++
			total++
		}

		if len(ghComments) < 100 {
			break
		}
		page++
	}

	return nil
}

// addPRLabel adds a label to a PR.
func (s *Seeder) addPRLabel(ctx context.Context, prID, labelID int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pr_labels (pr_id, label_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, prID, labelID)
	return err
}

// addPRAssignee adds an assignee to a PR.
func (s *Seeder) addPRAssignee(ctx context.Context, prID, userID int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pr_assignees (pr_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, prID, userID)
	return err
}

// updateRateInfo updates the result with rate limit information.
func (s *Seeder) updateRateInfo(result *Result, rateInfo *RateLimitInfo) {
	if rateInfo == nil {
		return
	}
	result.RateLimitRemaining = rateInfo.Remaining
	result.RateLimitReset = rateInfo.Reset

	// Warn if running low
	if rateInfo.Remaining < 100 && rateInfo.Remaining > 0 {
		slog.Warn("rate limit running low", "remaining", rateInfo.Remaining, "reset", rateInfo.Reset.Format(time.RFC3339))
	}
}

// EnsureAdminUser ensures an admin user exists and returns their ID.
func EnsureAdminUser(ctx context.Context, usersStore *duckdb.UsersStore) (int64, error) {
	// Check if admin user exists
	admin, err := usersStore.GetByLogin(ctx, "admin")
	if err != nil {
		return 0, err
	}
	if admin != nil {
		return admin.ID, nil
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
		return 0, err
	}

	slog.Info("created admin user", "login", admin.Login, "id", admin.ID)
	return admin.ID, nil
}
