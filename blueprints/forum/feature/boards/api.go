package boards

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
)

// Errors
var (
	ErrNotFound      = errors.New("board not found")
	ErrNameTaken     = errors.New("board name already taken")
	ErrInvalidName   = errors.New("invalid board name")
	ErrNotMember     = errors.New("not a member of this board")
	ErrAlreadyMember = errors.New("already a member of this board")
	ErrNotModerator  = errors.New("not a moderator of this board")
	ErrBoardArchived = errors.New("board is archived")
)

// Validation constants
const (
	NameMinLen    = 3
	NameMaxLen    = 21
	TitleMaxLen   = 100
	DescMaxLen    = 500
	SidebarMaxLen = 10000
)

var NameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]+$`)

// Board represents a discussion board.
type Board struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Sidebar      string    `json:"sidebar"`
	SidebarHTML  string    `json:"sidebar_html"`
	IconURL      string    `json:"icon_url"`
	BannerURL    string    `json:"banner_url"`
	PrimaryColor string    `json:"primary_color"`
	IsNSFW       bool      `json:"is_nsfw"`
	IsPrivate    bool      `json:"is_private"`
	IsArchived   bool      `json:"is_archived"`
	MemberCount  int64     `json:"member_count"`
	ThreadCount  int64     `json:"thread_count"`
	CreatedAt    time.Time `json:"created_at"`
	CreatedBy    string    `json:"created_by"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Viewer state
	IsJoined    bool `json:"is_joined,omitempty"`
	IsModerator bool `json:"is_moderator,omitempty"`
}

// BoardMember represents a board subscription.
type BoardMember struct {
	BoardID   string    `json:"board_id"`
	AccountID string    `json:"account_id"`
	JoinedAt  time.Time `json:"joined_at"`
}

// BoardModerator represents a moderator assignment.
type BoardModerator struct {
	BoardID     string            `json:"board_id"`
	AccountID   string            `json:"account_id"`
	Permissions ModPerms          `json:"permissions"`
	AddedAt     time.Time         `json:"added_at"`
	AddedBy     string            `json:"added_by"`
	Account     *accounts.Account `json:"account,omitempty"`
}

// ModPerms defines moderator permissions.
type ModPerms struct {
	ManagePosts    bool `json:"manage_posts"`
	ManageComments bool `json:"manage_comments"`
	ManageUsers    bool `json:"manage_users"`
	ManageMods     bool `json:"manage_mods"`
	ManageSettings bool `json:"manage_settings"`
}

// FullPerms returns permissions with all flags enabled.
func FullPerms() ModPerms {
	return ModPerms{
		ManagePosts:    true,
		ManageComments: true,
		ManageUsers:    true,
		ManageMods:     true,
		ManageSettings: true,
	}
}

// CreateIn contains input for creating a board.
type CreateIn struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	IsNSFW      bool   `json:"is_nsfw"`
	IsPrivate   bool   `json:"is_private"`

	// Seeding fields (optional, for importing from external sources)
	MemberCount int64 `json:"member_count,omitempty"`
}

// Validate validates the create input.
func (in *CreateIn) Validate() error {
	if len(in.Name) < NameMinLen || len(in.Name) > NameMaxLen {
		return ErrInvalidName
	}
	if !NameRegex.MatchString(in.Name) {
		return ErrInvalidName
	}
	return nil
}

// UpdateIn contains input for updating a board.
type UpdateIn struct {
	Title        *string `json:"title,omitempty"`
	Description  *string `json:"description,omitempty"`
	Sidebar      *string `json:"sidebar,omitempty"`
	IconURL      *string `json:"icon_url,omitempty"`
	BannerURL    *string `json:"banner_url,omitempty"`
	PrimaryColor *string `json:"primary_color,omitempty"`
	IsNSFW       *bool   `json:"is_nsfw,omitempty"`
}

// ListOpts contains options for listing boards.
type ListOpts struct {
	Limit   int
	Cursor  string
	OrderBy string
}

// API defines the boards service interface.
type API interface {
	// Board management
	Create(ctx context.Context, creatorID string, in CreateIn) (*Board, error)
	GetByName(ctx context.Context, name string) (*Board, error)
	GetByID(ctx context.Context, id string) (*Board, error)
	Update(ctx context.Context, id string, in UpdateIn) (*Board, error)
	Delete(ctx context.Context, id string) error
	Archive(ctx context.Context, id string) error

	// Membership
	Join(ctx context.Context, boardID, accountID string) error
	Leave(ctx context.Context, boardID, accountID string) error
	IsMember(ctx context.Context, boardID, accountID string) (bool, error)
	ListMembers(ctx context.Context, boardID string, opts ListOpts) ([]*accounts.Account, error)

	// Moderation
	AddModerator(ctx context.Context, boardID, accountID, addedBy string, perms ModPerms) error
	RemoveModerator(ctx context.Context, boardID, accountID string) error
	IsModerator(ctx context.Context, boardID, accountID string) (bool, error)
	GetModeratorPerms(ctx context.Context, boardID, accountID string) (*ModPerms, error)
	ListModerators(ctx context.Context, boardID string) ([]*BoardModerator, error)

	// Discovery
	List(ctx context.Context, opts ListOpts) ([]*Board, error)
	Search(ctx context.Context, query string, limit int) ([]*Board, error)
	ListPopular(ctx context.Context, limit int) ([]*Board, error)
	ListNew(ctx context.Context, limit int) ([]*Board, error)

	// User's boards
	ListJoined(ctx context.Context, accountID string) ([]*Board, error)
	ListModerated(ctx context.Context, accountID string) ([]*Board, error)

	// Viewer state
	EnrichBoard(ctx context.Context, board *Board, viewerID string) error
	EnrichBoards(ctx context.Context, boards []*Board, viewerID string) error

	// Stats
	IncrementThreadCount(ctx context.Context, boardID string, delta int64) error
}

// Store defines the data storage interface for boards.
type Store interface {
	Create(ctx context.Context, board *Board) error
	GetByName(ctx context.Context, name string) (*Board, error)
	GetByID(ctx context.Context, id string) (*Board, error)
	Update(ctx context.Context, board *Board) error
	Delete(ctx context.Context, id string) error

	// Membership
	AddMember(ctx context.Context, member *BoardMember) error
	RemoveMember(ctx context.Context, boardID, accountID string) error
	GetMember(ctx context.Context, boardID, accountID string) (*BoardMember, error)
	ListMembers(ctx context.Context, boardID string, opts ListOpts) ([]*accounts.Account, error)
	ListJoinedBoards(ctx context.Context, accountID string) ([]*Board, error)

	// Moderation
	AddModerator(ctx context.Context, mod *BoardModerator) error
	RemoveModerator(ctx context.Context, boardID, accountID string) error
	GetModerator(ctx context.Context, boardID, accountID string) (*BoardModerator, error)
	ListModerators(ctx context.Context, boardID string) ([]*BoardModerator, error)
	ListModeratedBoards(ctx context.Context, accountID string) ([]*Board, error)

	// Lists
	List(ctx context.Context, opts ListOpts) ([]*Board, error)
	Search(ctx context.Context, query string, limit int) ([]*Board, error)
	ListPopular(ctx context.Context, limit int) ([]*Board, error)
	ListNew(ctx context.Context, limit int) ([]*Board, error)
}
