package branches

import (
	"context"
	"errors"
)

var (
	ErrNotFound        = errors.New("branch not found")
	ErrBranchExists    = errors.New("branch already exists")
	ErrProtected       = errors.New("branch is protected")
)

// Branch represents a Git branch
type Branch struct {
	Name      string     `json:"name"`
	Commit    *CommitRef `json:"commit"`
	Protected bool       `json:"protected"`
}

// CommitRef represents a commit reference
type CommitRef struct {
	SHA string `json:"sha"`
	URL string `json:"url"`
}

// BranchProtection represents branch protection settings
type BranchProtection struct {
	URL                            string                      `json:"url"`
	Enabled                        bool                        `json:"enabled"`
	RequiredStatusChecks           *RequiredStatusChecks       `json:"required_status_checks,omitempty"`
	EnforceAdmins                  *EnforceAdmins              `json:"enforce_admins,omitempty"`
	RequiredPullRequestReviews     *RequiredPullRequestReviews `json:"required_pull_request_reviews,omitempty"`
	Restrictions                   *BranchRestrictions         `json:"restrictions,omitempty"`
	RequiredLinearHistory          *EnabledSetting             `json:"required_linear_history,omitempty"`
	AllowForcePushes               *EnabledSetting             `json:"allow_force_pushes,omitempty"`
	AllowDeletions                 *EnabledSetting             `json:"allow_deletions,omitempty"`
	RequiredConversationResolution *EnabledSetting             `json:"required_conversation_resolution,omitempty"`
	RequiredSignatures             *EnabledSetting             `json:"required_signatures,omitempty"`
}

// RequiredStatusChecks represents required status checks
type RequiredStatusChecks struct {
	URL            string   `json:"url"`
	Strict         bool     `json:"strict"`
	Contexts       []string `json:"contexts"`
	ContextsURL    string   `json:"contexts_url"`
	Checks         []*Check `json:"checks"`
}

// Check represents a required status check
type Check struct {
	Context string `json:"context"`
	AppID   *int64 `json:"app_id,omitempty"`
}

// EnforceAdmins represents enforcement for admins
type EnforceAdmins struct {
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

// RequiredPullRequestReviews represents required PR review settings
type RequiredPullRequestReviews struct {
	URL                          string              `json:"url"`
	DismissalRestrictions        *DismissalRestrictions `json:"dismissal_restrictions,omitempty"`
	DismissStaleReviews          bool                `json:"dismiss_stale_reviews"`
	RequireCodeOwnerReviews      bool                `json:"require_code_owner_reviews"`
	RequiredApprovingReviewCount int                 `json:"required_approving_review_count"`
	RequireLastPushApproval      bool                `json:"require_last_push_approval"`
}

// DismissalRestrictions represents dismissal restrictions
type DismissalRestrictions struct {
	URL    string   `json:"url"`
	Users  []User   `json:"users"`
	Teams  []Team   `json:"teams"`
	Apps   []App    `json:"apps,omitempty"`
}

// BranchRestrictions represents push restrictions
type BranchRestrictions struct {
	URL      string  `json:"url"`
	UsersURL string  `json:"users_url"`
	TeamsURL string  `json:"teams_url"`
	AppsURL  string  `json:"apps_url"`
	Users    []User  `json:"users"`
	Teams    []Team  `json:"teams"`
	Apps     []App   `json:"apps"`
}

// EnabledSetting represents a simple enabled setting
type EnabledSetting struct {
	Enabled bool `json:"enabled"`
}

// User is a minimal user reference
type User struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Type      string `json:"type"`
	SiteAdmin bool   `json:"site_admin"`
}

// Team is a minimal team reference
type Team struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// App is a minimal app reference
type App struct {
	ID   int64  `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// ListOpts contains options for listing branches
type ListOpts struct {
	Page      int    `json:"page,omitempty"`
	PerPage   int    `json:"per_page,omitempty"`
	Protected *bool  `json:"protected,omitempty"`
}

// UpdateProtectionIn represents input for updating branch protection
type UpdateProtectionIn struct {
	RequiredStatusChecks       *RequiredStatusChecksIn       `json:"required_status_checks"`
	EnforceAdmins              bool                          `json:"enforce_admins"`
	RequiredPullRequestReviews *RequiredPullRequestReviewsIn `json:"required_pull_request_reviews"`
	Restrictions               *RestrictionsIn               `json:"restrictions"`
	RequiredLinearHistory      bool                          `json:"required_linear_history,omitempty"`
	AllowForcePushes           *bool                         `json:"allow_force_pushes,omitempty"`
	AllowDeletions             bool                          `json:"allow_deletions,omitempty"`
	RequiredConversationResolution bool                      `json:"required_conversation_resolution,omitempty"`
}

// RequiredStatusChecksIn represents input for required status checks
type RequiredStatusChecksIn struct {
	Strict   bool           `json:"strict"`
	Contexts []string       `json:"contexts,omitempty"`
	Checks   []*CheckIn     `json:"checks,omitempty"`
}

// CheckIn represents input for a check
type CheckIn struct {
	Context string `json:"context"`
	AppID   *int64 `json:"app_id,omitempty"`
}

// RequiredPullRequestReviewsIn represents input for required PR reviews
type RequiredPullRequestReviewsIn struct {
	DismissalRestrictions        *DismissalRestrictionsIn `json:"dismissal_restrictions,omitempty"`
	DismissStaleReviews          bool                     `json:"dismiss_stale_reviews,omitempty"`
	RequireCodeOwnerReviews      bool                     `json:"require_code_owner_reviews,omitempty"`
	RequiredApprovingReviewCount int                      `json:"required_approving_review_count,omitempty"`
	RequireLastPushApproval      bool                     `json:"require_last_push_approval,omitempty"`
}

// DismissalRestrictionsIn represents input for dismissal restrictions
type DismissalRestrictionsIn struct {
	Users []string `json:"users,omitempty"`
	Teams []string `json:"teams,omitempty"`
	Apps  []string `json:"apps,omitempty"`
}

// RestrictionsIn represents input for push restrictions
type RestrictionsIn struct {
	Users []string `json:"users"`
	Teams []string `json:"teams"`
	Apps  []string `json:"apps,omitempty"`
}

// API defines the branches service interface
type API interface {
	// List returns branches for a repository
	List(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Branch, error)

	// Get retrieves a branch by name
	Get(ctx context.Context, owner, repo, branch string) (*Branch, error)

	// Rename renames a branch
	Rename(ctx context.Context, owner, repo, branch, newName string) (*Branch, error)

	// GetProtection retrieves branch protection settings
	GetProtection(ctx context.Context, owner, repo, branch string) (*BranchProtection, error)

	// UpdateProtection updates branch protection settings
	UpdateProtection(ctx context.Context, owner, repo, branch string, in *UpdateProtectionIn) (*BranchProtection, error)

	// DeleteProtection removes branch protection
	DeleteProtection(ctx context.Context, owner, repo, branch string) error

	// GetRequiredStatusChecks retrieves required status checks
	GetRequiredStatusChecks(ctx context.Context, owner, repo, branch string) (*RequiredStatusChecks, error)

	// UpdateRequiredStatusChecks updates required status checks
	UpdateRequiredStatusChecks(ctx context.Context, owner, repo, branch string, in *RequiredStatusChecksIn) (*RequiredStatusChecks, error)

	// RemoveRequiredStatusChecks removes required status checks
	RemoveRequiredStatusChecks(ctx context.Context, owner, repo, branch string) error

	// GetRequiredSignatures retrieves required signatures setting
	GetRequiredSignatures(ctx context.Context, owner, repo, branch string) (*EnabledSetting, error)

	// CreateRequiredSignatures enables required signatures
	CreateRequiredSignatures(ctx context.Context, owner, repo, branch string) (*EnabledSetting, error)

	// DeleteRequiredSignatures disables required signatures
	DeleteRequiredSignatures(ctx context.Context, owner, repo, branch string) error
}

// Store defines the data access interface for branches
type Store interface {
	// Branch protection settings
	GetProtection(ctx context.Context, repoID int64, branch string) (*BranchProtection, error)
	SetProtection(ctx context.Context, repoID int64, branch string, protection *BranchProtection) error
	DeleteProtection(ctx context.Context, repoID int64, branch string) error
}
