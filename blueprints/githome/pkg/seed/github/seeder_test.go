package github

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

// testToken returns the GitHub token from environment, or empty string.
func testToken() string {
	return os.Getenv("GITHUB_TOKEN")
}

// setupTestDB creates an in-memory DuckDB database for testing.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}

	// Initialize schema using the standard store
	_, err = duckdb.New(db)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create schema: %v", err)
	}

	return db
}

func TestClient_GetRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	token := testToken()
	client := NewClient("", token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo, rateInfo, err := client.GetRepository(ctx, "golang", "go")
	if err != nil {
		// Skip if we're rate limited or have auth issues without a token
		if token == "" {
			t.Skipf("skipping without token: %v", err)
		}
		t.Fatalf("GetRepository failed: %v", err)
	}

	if repo.Name != "go" {
		t.Errorf("expected repo name 'go', got %q", repo.Name)
	}
	if repo.FullName != "golang/go" {
		t.Errorf("expected full name 'golang/go', got %q", repo.FullName)
	}
	if repo.Owner == nil || repo.Owner.Login != "golang" {
		t.Errorf("expected owner 'golang', got %v", repo.Owner)
	}
	if repo.Owner.Type != "Organization" {
		t.Errorf("expected owner type 'Organization', got %q", repo.Owner.Type)
	}

	t.Logf("Repository: %s, Stars: %d, OpenIssues: %d", repo.FullName, repo.StargazersCount, repo.OpenIssuesCount)
	if rateInfo != nil {
		t.Logf("Rate limit remaining: %d", rateInfo.Remaining)
	}
}

func TestClient_ListIssues(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	token := testToken()
	client := NewClient("", token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	issues, rateInfo, err := client.ListIssues(ctx, "golang", "go", &ListOptions{
		Page:    1,
		PerPage: 10,
		State:   "all",
	})
	if err != nil {
		if token == "" {
			t.Skipf("skipping without token: %v", err)
		}
		t.Fatalf("ListIssues failed: %v", err)
	}

	if len(issues) == 0 {
		t.Error("expected at least one issue")
	}

	// Find an actual issue (not a PR)
	var issueFound bool
	for _, issue := range issues {
		if issue.PullRequest == nil {
			issueFound = true
			t.Logf("Issue #%d: %s (state: %s)", issue.Number, issue.Title, issue.State)
			break
		}
	}

	if !issueFound {
		t.Log("No pure issues found in first 10 items (all are PRs)")
	}

	if rateInfo != nil {
		t.Logf("Rate limit remaining: %d", rateInfo.Remaining)
	}
}

func TestClient_ListPullRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	token := testToken()
	client := NewClient("", token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prs, rateInfo, err := client.ListPullRequests(ctx, "golang", "go", &ListOptions{
		Page:    1,
		PerPage: 5,
		State:   "all",
	})
	if err != nil {
		if token == "" {
			t.Skipf("skipping without token: %v", err)
		}
		t.Fatalf("ListPullRequests failed: %v", err)
	}

	if len(prs) == 0 {
		t.Error("expected at least one PR")
	}

	for _, pr := range prs {
		t.Logf("PR #%d: %s (state: %s, merged: %v)", pr.Number, pr.Title, pr.State, pr.Merged)
	}

	if rateInfo != nil {
		t.Logf("Rate limit remaining: %d", rateInfo.Remaining)
	}
}

func TestClient_ListLabels(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	token := testToken()
	client := NewClient("", token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	labels, rateInfo, err := client.ListLabels(ctx, "golang", "go", &ListOptions{
		Page:    1,
		PerPage: 100,
	})
	if err != nil {
		if token == "" {
			t.Skipf("skipping without token: %v", err)
		}
		t.Fatalf("ListLabels failed: %v", err)
	}

	if len(labels) == 0 {
		t.Error("expected at least one label")
	}

	t.Logf("Found %d labels", len(labels))
	for i, label := range labels {
		if i < 5 {
			t.Logf("Label: %s (color: #%s)", label.Name, label.Color)
		}
	}

	if rateInfo != nil {
		t.Logf("Rate limit remaining: %d", rateInfo.Remaining)
	}
}

func TestClient_ListMilestones(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	token := testToken()
	client := NewClient("", token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	milestones, rateInfo, err := client.ListMilestones(ctx, "golang", "go", &ListOptions{
		Page:    1,
		PerPage: 10,
		State:   "all",
	})
	if err != nil {
		if token == "" {
			t.Skipf("skipping without token: %v", err)
		}
		t.Fatalf("ListMilestones failed: %v", err)
	}

	t.Logf("Found %d milestones", len(milestones))
	for _, ms := range milestones {
		t.Logf("Milestone #%d: %s (state: %s, open: %d, closed: %d)",
			ms.Number, ms.Title, ms.State, ms.OpenIssues, ms.ClosedIssues)
	}

	if rateInfo != nil {
		t.Logf("Rate limit remaining: %d", rateInfo.Remaining)
	}
}

func TestClient_ListIssueComments(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	token := testToken()
	client := NewClient("", token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Issue #1 is the first issue in golang/go
	comments, rateInfo, err := client.ListIssueComments(ctx, "golang", "go", 1, &ListOptions{
		Page:    1,
		PerPage: 5,
	})
	if err != nil {
		if token == "" {
			t.Skipf("skipping without token: %v", err)
		}
		t.Fatalf("ListIssueComments failed: %v", err)
	}

	t.Logf("Found %d comments for issue #1", len(comments))
	for _, c := range comments {
		user := "unknown"
		if c.User != nil {
			user = c.User.Login
		}
		body := c.Body
		if len(body) > 50 {
			body = body[:50]
		}
		t.Logf("Comment by %s at %s: %s...", user, c.CreatedAt.Format("2006-01-02"), body)
	}

	if rateInfo != nil {
		t.Logf("Rate limit remaining: %d", rateInfo.Remaining)
	}
}

func TestSeeder_SeedGolangGo_LimitedIssues(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	config := DefaultConfig("golang", "go")
	config.Token = testToken()
	config.MaxIssues = 5
	config.MaxPRs = 0         // Skip PRs for this test
	config.ImportPRs = false
	config.ImportComments = false // Skip comments to reduce API calls
	config.AdminUserID = 1

	seeder := NewSeeder(db, config)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := seeder.Seed(ctx)
	if err != nil {
		t.Fatalf("Seed failed: %v", err)
	}

	// Verify results
	if !result.RepoCreated {
		t.Error("expected repo to be created")
	}
	if !result.OrgCreated {
		t.Error("expected org to be created")
	}
	if result.IssuesCreated == 0 {
		t.Error("expected at least one issue to be created")
	}
	if result.IssuesCreated > 5 {
		t.Errorf("expected at most 5 issues, got %d", result.IssuesCreated)
	}
	if result.LabelsCreated == 0 {
		t.Error("expected at least one label to be created")
	}

	t.Logf("Seed result: repo=%v, org=%v, users=%d, issues=%d, labels=%d, milestones=%d, errors=%d",
		result.RepoCreated, result.OrgCreated, result.UsersCreated,
		result.IssuesCreated, result.LabelsCreated, result.MilestonesCreated,
		len(result.Errors))

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Logf("Error: %v", e)
		}
	}

	// Verify data in database
	var repoCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM repositories").Scan(&repoCount); err != nil {
		t.Errorf("failed to count repos: %v", err)
	}
	if repoCount != 1 {
		t.Errorf("expected 1 repo, got %d", repoCount)
	}

	var issueCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM issues").Scan(&issueCount); err != nil {
		t.Errorf("failed to count issues: %v", err)
	}
	t.Logf("Issues in database: %d", issueCount)
}

func TestSeeder_SeedLabelsOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	config := DefaultConfig("golang", "go")
	config.Token = testToken()
	config.ImportIssues = false
	config.ImportPRs = false
	config.ImportComments = false
	config.ImportMilestones = false
	config.AdminUserID = 1

	seeder := NewSeeder(db, config)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	result, err := seeder.Seed(ctx)
	if err != nil {
		t.Fatalf("Seed failed: %v", err)
	}

	if result.LabelsCreated == 0 {
		t.Error("expected labels to be created")
	}
	if result.IssuesCreated != 0 {
		t.Errorf("expected 0 issues, got %d", result.IssuesCreated)
	}
	if result.PRsCreated != 0 {
		t.Errorf("expected 0 PRs, got %d", result.PRsCreated)
	}

	t.Logf("Labels created: %d", result.LabelsCreated)
}

func TestMapper_Issue(t *testing.T) {
	now := time.Now()
	closed := now.Add(-time.Hour)

	ghIssue := &ghIssue{
		ID:          12345,
		NodeID:      "I_kwDOAHR9X851234",
		Number:      100,
		Title:       "Test Issue",
		Body:        "This is a test issue body",
		State:       "closed",
		StateReason: "completed",
		User:        &ghUser{Login: "testuser", ID: 1},
		Locked:      true,
		ActiveLockReason: "resolved",
		Comments:    5,
		ClosedAt:    &closed,
		CreatedAt:   now.Add(-24 * time.Hour),
		UpdatedAt:   now,
	}

	issue := mapIssue(ghIssue, 1, 2)

	if issue.Number != 100 {
		t.Errorf("expected number 100, got %d", issue.Number)
	}
	if issue.Title != "Test Issue" {
		t.Errorf("expected title 'Test Issue', got %q", issue.Title)
	}
	if issue.State != "closed" {
		t.Errorf("expected state 'closed', got %q", issue.State)
	}
	if issue.StateReason != "completed" {
		t.Errorf("expected state reason 'completed', got %q", issue.StateReason)
	}
	if issue.RepoID != 1 {
		t.Errorf("expected repo_id 1, got %d", issue.RepoID)
	}
	if issue.CreatorID != 2 {
		t.Errorf("expected creator_id 2, got %d", issue.CreatorID)
	}
	if !issue.Locked {
		t.Error("expected locked to be true")
	}
	if issue.ActiveLockReason != "resolved" {
		t.Errorf("expected lock reason 'resolved', got %q", issue.ActiveLockReason)
	}
	if issue.ClosedAt == nil {
		t.Error("expected closed_at to be set")
	}
}

func TestMapper_PullRequest(t *testing.T) {
	now := time.Now()
	merged := now.Add(-time.Hour)

	ghPR := &ghPullRequest{
		ID:          67890,
		NodeID:      "PR_kwDOAHR9X867890",
		Number:      200,
		Title:       "Test PR",
		Body:        "This is a test PR body",
		State:       "closed",
		User:        &ghUser{Login: "testuser", ID: 1},
		Head: &ghBranch{
			Label: "testuser:feature",
			Ref:   "feature",
			SHA:   "abc123",
		},
		Base: &ghBranch{
			Label: "golang:main",
			Ref:   "main",
			SHA:   "def456",
		},
		Draft:          false,
		Merged:         true,
		MergedAt:       &merged,
		MergeCommitSHA: "ghi789",
		Comments:       3,
		ReviewComments: 10,
		Commits:        5,
		Additions:      100,
		Deletions:      50,
		ChangedFiles:   8,
		CreatedAt:      now.Add(-48 * time.Hour),
		UpdatedAt:      now,
	}

	pr := mapPullRequest(ghPR, 1, 2)

	if pr.Number != 200 {
		t.Errorf("expected number 200, got %d", pr.Number)
	}
	if pr.Title != "Test PR" {
		t.Errorf("expected title 'Test PR', got %q", pr.Title)
	}
	if pr.State != "closed" {
		t.Errorf("expected state 'closed', got %q", pr.State)
	}
	if pr.Head == nil || pr.Head.Ref != "feature" {
		t.Error("expected head ref to be 'feature'")
	}
	if pr.Base == nil || pr.Base.Ref != "main" {
		t.Error("expected base ref to be 'main'")
	}
	if !pr.Merged {
		t.Error("expected merged to be true")
	}
	if pr.MergeCommitSHA != "ghi789" {
		t.Errorf("expected merge commit SHA 'ghi789', got %q", pr.MergeCommitSHA)
	}
	if pr.Commits != 5 {
		t.Errorf("expected 5 commits, got %d", pr.Commits)
	}
	if pr.Additions != 100 {
		t.Errorf("expected 100 additions, got %d", pr.Additions)
	}
}

func TestMapper_Label(t *testing.T) {
	ghLabel := &ghLabel{
		ID:          111,
		NodeID:      "LA_kwDOAHR9X8111",
		Name:        "bug",
		Description: "Something isn't working",
		Color:       "d73a4a",
		Default:     true,
	}

	label := mapLabel(ghLabel, 1)

	if label.Name != "bug" {
		t.Errorf("expected name 'bug', got %q", label.Name)
	}
	if label.Description != "Something isn't working" {
		t.Errorf("expected description, got %q", label.Description)
	}
	if label.Color != "d73a4a" {
		t.Errorf("expected color 'd73a4a', got %q", label.Color)
	}
	if !label.Default {
		t.Error("expected default to be true")
	}
	if label.RepoID != 1 {
		t.Errorf("expected repo_id 1, got %d", label.RepoID)
	}
}

func TestMapper_Milestone(t *testing.T) {
	now := time.Now()
	dueOn := now.Add(7 * 24 * time.Hour)

	ghMilestone := &ghMilestone{
		ID:           222,
		NodeID:       "MI_kwDOAHR9X8222",
		Number:       10,
		Title:        "v1.0",
		Description:  "First stable release",
		State:        "open",
		OpenIssues:   15,
		ClosedIssues: 45,
		DueOn:        &dueOn,
		CreatedAt:    now.Add(-30 * 24 * time.Hour),
		UpdatedAt:    now,
	}

	milestone := mapMilestone(ghMilestone, 1, 2)

	if milestone.Number != 10 {
		t.Errorf("expected number 10, got %d", milestone.Number)
	}
	if milestone.Title != "v1.0" {
		t.Errorf("expected title 'v1.0', got %q", milestone.Title)
	}
	if milestone.State != "open" {
		t.Errorf("expected state 'open', got %q", milestone.State)
	}
	if milestone.OpenIssues != 15 {
		t.Errorf("expected 15 open issues, got %d", milestone.OpenIssues)
	}
	if milestone.ClosedIssues != 45 {
		t.Errorf("expected 45 closed issues, got %d", milestone.ClosedIssues)
	}
	if milestone.DueOn == nil {
		t.Error("expected due_on to be set")
	}
	if milestone.RepoID != 1 {
		t.Errorf("expected repo_id 1, got %d", milestone.RepoID)
	}
	if milestone.CreatorID != 2 {
		t.Errorf("expected creator_id 2, got %d", milestone.CreatorID)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig("owner", "repo")

	if config.Owner != "owner" {
		t.Errorf("expected owner 'owner', got %q", config.Owner)
	}
	if config.Repo != "repo" {
		t.Errorf("expected repo 'repo', got %q", config.Repo)
	}
	if config.BaseURL != "https://api.github.com" {
		t.Errorf("expected base URL, got %q", config.BaseURL)
	}
	if !config.IsPublic {
		t.Error("expected IsPublic to be true")
	}
	if !config.ImportIssues {
		t.Error("expected ImportIssues to be true")
	}
	if !config.ImportPRs {
		t.Error("expected ImportPRs to be true")
	}
	if !config.ImportComments {
		t.Error("expected ImportComments to be true")
	}
	if !config.ImportLabels {
		t.Error("expected ImportLabels to be true")
	}
	if !config.ImportMilestones {
		t.Error("expected ImportMilestones to be true")
	}
}

func TestClient_RateLimitHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	token := testToken()
	client := NewClient("", token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Make a request and check rate limit info
	_, rateInfo, err := client.GetRepository(ctx, "golang", "go")
	if err != nil {
		if token == "" {
			t.Skipf("skipping without token: %v", err)
		}
		t.Fatalf("GetRepository failed: %v", err)
	}

	if rateInfo == nil {
		t.Fatal("expected rate info to be returned")
	}

	t.Logf("Rate limit: remaining=%d, reset=%s",
		rateInfo.Remaining, rateInfo.Reset.Format(time.RFC3339))

	// Rate limit should be positive (unless we hit the limit)
	if rateInfo.Remaining < 0 {
		t.Errorf("unexpected negative rate limit: %d", rateInfo.Remaining)
	}

	// Reset time should be in the future (or very recent past)
	if rateInfo.Reset.Before(time.Now().Add(-5 * time.Minute)) {
		t.Errorf("reset time is too far in the past: %s", rateInfo.Reset)
	}
}

func TestClient_NotFoundError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	token := testToken()
	client := NewClient("", token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, _, err := client.GetRepository(ctx, "nonexistent-org-12345", "nonexistent-repo-67890")
	if err == nil {
		t.Error("expected error for non-existent repo")
	}

	t.Logf("Expected error: %v", err)
}
