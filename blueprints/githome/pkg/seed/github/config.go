package github

import "time"

// Config contains GitHub seeder configuration.
type Config struct {
	// Required
	Owner string // Repository owner (user or org)
	Repo  string // Repository name

	// Authentication
	Token string // GitHub personal access token (optional, higher rate limits)

	// Optional
	BaseURL     string // GitHub API base URL (for GitHub Enterprise), defaults to https://api.github.com
	AdminUserID int64  // Admin user ID for ownership (defaults to 1)
	IsPublic    bool   // Make imported repo public (default: true)

	// Import options
	ImportIssues     bool // Import issues (default: true)
	ImportPRs        bool // Import pull requests (default: true)
	ImportComments   bool // Import comments (default: true)
	ImportLabels     bool // Import labels (default: true)
	ImportMilestones bool // Import milestones (default: true)

	// Limits
	MaxIssues          int // Max issues to import (0 = all)
	MaxPRs             int // Max PRs to import (0 = all)
	MaxCommentsPerItem int // Max comments per issue/PR (0 = all)

	// Single item import
	SingleIssue int // Import only this issue number (0 = all)
	SinglePR    int // Import only this PR number (0 = all)
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig(owner, repo string) Config {
	return Config{
		Owner:            owner,
		Repo:             repo,
		BaseURL:          "https://api.github.com",
		AdminUserID:      1,
		IsPublic:         true,
		ImportIssues:     true,
		ImportPRs:        true,
		ImportComments:   true,
		ImportLabels:     true,
		ImportMilestones: true,
	}
}

// Result contains the result of a GitHub seeding operation.
type Result struct {
	// Counts
	RepoCreated       bool
	OrgCreated        bool
	UsersCreated      int
	IssuesCreated     int
	PRsCreated        int
	CommentsCreated   int
	LabelsCreated     int
	MilestonesCreated int

	// Skipped (already exist)
	IssuesSkipped   int
	PRsSkipped      int
	CommentsSkipped int

	// Errors
	Errors []error

	// Rate limit info
	RateLimitRemaining int
	RateLimitReset     time.Time

	// Fallback info
	UsedCrawler bool // True if crawler fallback was used due to API rate limit or auth failure
}
