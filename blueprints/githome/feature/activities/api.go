package activities

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("event not found")
)

// Event represents an activity event
type Event struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Actor     *Actor      `json:"actor"`
	Repo      *EventRepo  `json:"repo"`
	Org       *Actor      `json:"org,omitempty"`
	Payload   interface{} `json:"payload"`
	Public    bool        `json:"public"`
	CreatedAt time.Time   `json:"created_at"`
}

// Actor represents an event actor
type Actor struct {
	ID           int64  `json:"id"`
	Login        string `json:"login"`
	DisplayLogin string `json:"display_login,omitempty"`
	GravatarID   string `json:"gravatar_id"`
	URL          string `json:"url"`
	AvatarURL    string `json:"avatar_url"`
}

// EventRepo represents an event repository
type EventRepo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Feeds represents activity feeds
type Feeds struct {
	TimelineURL                 string `json:"timeline_url"`
	UserURL                     string `json:"user_url"`
	CurrentUserPublicURL        string `json:"current_user_public_url,omitempty"`
	CurrentUserURL              string `json:"current_user_url,omitempty"`
	CurrentUserActorURL         string `json:"current_user_actor_url,omitempty"`
	CurrentUserOrganizationURL  string `json:"current_user_organization_url,omitempty"`
	CurrentUserOrganizationsURL []string `json:"current_user_organization_urls,omitempty"`
}

// Event types
const (
	EventCommitComment            = "CommitCommentEvent"
	EventCreate                   = "CreateEvent"
	EventDelete                   = "DeleteEvent"
	EventFork                     = "ForkEvent"
	EventGollum                   = "GollumEvent"
	EventIssueComment             = "IssueCommentEvent"
	EventIssues                   = "IssuesEvent"
	EventMember                   = "MemberEvent"
	EventPublic                   = "PublicEvent"
	EventPullRequest              = "PullRequestEvent"
	EventPullRequestReview        = "PullRequestReviewEvent"
	EventPullRequestReviewComment = "PullRequestReviewCommentEvent"
	EventPush                     = "PushEvent"
	EventRelease                  = "ReleaseEvent"
	EventSponsor                  = "SponsorshipEvent"
	EventWatch                    = "WatchEvent"
)

// ListOpts contains options for listing events
type ListOpts struct {
	Page    int `json:"page,omitempty"`
	PerPage int `json:"per_page,omitempty"`
}

// API defines the activities service interface
type API interface {
	// ListPublic returns public events
	ListPublic(ctx context.Context, opts *ListOpts) ([]*Event, error)

	// ListForRepo returns events for a repository
	ListForRepo(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Event, error)

	// ListNetworkEvents returns events for a repo's network
	ListNetworkEvents(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Event, error)

	// ListForOrg returns events for an organization
	ListForOrg(ctx context.Context, org string, opts *ListOpts) ([]*Event, error)

	// ListPublicForOrg returns public events for an organization
	ListPublicForOrg(ctx context.Context, org string, opts *ListOpts) ([]*Event, error)

	// ListForUser returns events performed by a user
	ListForUser(ctx context.Context, username string, opts *ListOpts) ([]*Event, error)

	// ListPublicForUser returns public events performed by a user
	ListPublicForUser(ctx context.Context, username string, opts *ListOpts) ([]*Event, error)

	// ListOrgEventsForUser returns org events for a user
	ListOrgEventsForUser(ctx context.Context, username, org string, opts *ListOpts) ([]*Event, error)

	// ListReceivedEvents returns events received by a user
	ListReceivedEvents(ctx context.Context, username string, opts *ListOpts) ([]*Event, error)

	// ListPublicReceivedEvents returns public events received by a user
	ListPublicReceivedEvents(ctx context.Context, username string, opts *ListOpts) ([]*Event, error)

	// GetFeeds returns feed URLs for the authenticated user
	GetFeeds(ctx context.Context, userID int64) (*Feeds, error)

	// Create creates an event (internal use)
	Create(ctx context.Context, eventType string, actorID, repoID int64, orgID *int64, payload interface{}, public bool) (*Event, error)
}

// Store defines the data access interface for activities
type Store interface {
	Create(ctx context.Context, e *Event) error
	GetByID(ctx context.Context, id string) (*Event, error)
	ListPublic(ctx context.Context, opts *ListOpts) ([]*Event, error)
	ListForRepo(ctx context.Context, repoID int64, opts *ListOpts) ([]*Event, error)
	ListForOrg(ctx context.Context, orgID int64, opts *ListOpts) ([]*Event, error)
	ListForUser(ctx context.Context, userID int64, opts *ListOpts) ([]*Event, error)
	ListReceivedByUser(ctx context.Context, userID int64, opts *ListOpts) ([]*Event, error)
}
