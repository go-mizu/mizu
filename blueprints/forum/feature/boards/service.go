package boards

import (
	"context"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/pkg/markdown"
	"github.com/go-mizu/mizu/blueprints/forum/pkg/ulid"
)

// Service implements the boards API.
type Service struct {
	store Store
}

// NewService creates a new boards service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new board.
func (s *Service) Create(ctx context.Context, creatorID string, in CreateIn) (*Board, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	// Check if name is taken
	existing, err := s.store.GetByName(ctx, in.Name)
	if err != nil && err != ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, ErrNameTaken
	}

	// Truncate fields if needed
	title := in.Title
	if title == "" {
		title = in.Name
	}
	if len(title) > TitleMaxLen {
		title = title[:TitleMaxLen]
	}

	description := in.Description
	if len(description) > DescMaxLen {
		description = description[:DescMaxLen]
	}

	now := time.Now()

	// Use initial member count if provided (for seeding), otherwise default to 1 (creator auto-joins)
	memberCount := int64(1)
	if in.MemberCount > 0 {
		memberCount = in.MemberCount
	}

	board := &Board{
		ID:          ulid.New(),
		Name:        strings.ToLower(in.Name),
		Title:       title,
		Description: description,
		IsNSFW:      in.IsNSFW,
		IsPrivate:   in.IsPrivate,
		MemberCount: memberCount,
		CreatedAt:   now,
		CreatedBy:   creatorID,
		UpdatedAt:   now,
	}

	if err := s.store.Create(ctx, board); err != nil {
		return nil, err
	}

	// Auto-join creator
	member := &BoardMember{
		BoardID:   board.ID,
		AccountID: creatorID,
		JoinedAt:  now,
	}
	_ = s.store.AddMember(ctx, member)

	// Make creator a moderator with full permissions
	mod := &BoardModerator{
		BoardID:     board.ID,
		AccountID:   creatorID,
		Permissions: FullPerms(),
		AddedAt:     now,
		AddedBy:     creatorID,
	}
	_ = s.store.AddModerator(ctx, mod)

	board.IsJoined = true
	board.IsModerator = true

	return board, nil
}

// GetByName retrieves a board by name.
func (s *Service) GetByName(ctx context.Context, name string) (*Board, error) {
	return s.store.GetByName(ctx, strings.ToLower(name))
}

// GetByID retrieves a board by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Board, error) {
	return s.store.GetByID(ctx, id)
}

// GetByIDs retrieves multiple boards by their IDs.
func (s *Service) GetByIDs(ctx context.Context, ids []string) (map[string]*Board, error) {
	if len(ids) == 0 {
		return make(map[string]*Board), nil
	}
	return s.store.GetByIDs(ctx, ids)
}

// Update updates a board.
func (s *Service) Update(ctx context.Context, id string, in UpdateIn) (*Board, error) {
	board, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if board.IsArchived {
		return nil, ErrBoardArchived
	}

	if in.Title != nil {
		title := *in.Title
		if len(title) > TitleMaxLen {
			title = title[:TitleMaxLen]
		}
		board.Title = title
	}
	if in.Description != nil {
		desc := *in.Description
		if len(desc) > DescMaxLen {
			desc = desc[:DescMaxLen]
		}
		board.Description = desc
	}
	if in.Sidebar != nil {
		sidebar := *in.Sidebar
		if len(sidebar) > SidebarMaxLen {
			sidebar = sidebar[:SidebarMaxLen]
		}
		board.Sidebar = sidebar
		// Render markdown
		html, err := markdown.RenderSafe(sidebar)
		if err == nil {
			board.SidebarHTML = html
		}
	}
	if in.IconURL != nil {
		board.IconURL = *in.IconURL
	}
	if in.BannerURL != nil {
		board.BannerURL = *in.BannerURL
	}
	if in.PrimaryColor != nil {
		board.PrimaryColor = *in.PrimaryColor
	}
	if in.IsNSFW != nil {
		board.IsNSFW = *in.IsNSFW
	}

	board.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, board); err != nil {
		return nil, err
	}

	return board, nil
}

// Delete deletes a board.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// Archive archives a board.
func (s *Service) Archive(ctx context.Context, id string) error {
	board, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	board.IsArchived = true
	board.UpdatedAt = time.Now()

	return s.store.Update(ctx, board)
}

// Join joins a board.
func (s *Service) Join(ctx context.Context, boardID, accountID string) error {
	board, err := s.store.GetByID(ctx, boardID)
	if err != nil {
		return err
	}

	if board.IsArchived {
		return ErrBoardArchived
	}

	// Check if already member
	existing, err := s.store.GetMember(ctx, boardID, accountID)
	if err != nil && err != ErrNotMember {
		return err
	}
	if existing != nil {
		return ErrAlreadyMember
	}

	member := &BoardMember{
		BoardID:   boardID,
		AccountID: accountID,
		JoinedAt:  time.Now(),
	}

	if err := s.store.AddMember(ctx, member); err != nil {
		return err
	}

	// Update member count
	board.MemberCount++
	return s.store.Update(ctx, board)
}

// Leave leaves a board.
func (s *Service) Leave(ctx context.Context, boardID, accountID string) error {
	board, err := s.store.GetByID(ctx, boardID)
	if err != nil {
		return err
	}

	if err := s.store.RemoveMember(ctx, boardID, accountID); err != nil {
		return err
	}

	// Also remove moderator status
	_ = s.store.RemoveModerator(ctx, boardID, accountID)

	// Update member count
	if board.MemberCount > 0 {
		board.MemberCount--
		return s.store.Update(ctx, board)
	}
	return nil
}

// IsMember checks if a user is a member of a board.
func (s *Service) IsMember(ctx context.Context, boardID, accountID string) (bool, error) {
	member, err := s.store.GetMember(ctx, boardID, accountID)
	if err == ErrNotMember {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return member != nil, nil
}

// ListMembers lists members of a board.
func (s *Service) ListMembers(ctx context.Context, boardID string, opts ListOpts) ([]*accounts.Account, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 25
	}
	return s.store.ListMembers(ctx, boardID, opts)
}

// AddModerator adds a moderator to a board.
func (s *Service) AddModerator(ctx context.Context, boardID, accountID, addedBy string, perms ModPerms) error {
	board, err := s.store.GetByID(ctx, boardID)
	if err != nil {
		return err
	}

	if board.IsArchived {
		return ErrBoardArchived
	}

	// Ensure user is a member first
	isMember, err := s.IsMember(ctx, boardID, accountID)
	if err != nil {
		return err
	}
	if !isMember {
		// Auto-join them
		_ = s.Join(ctx, boardID, accountID)
	}

	mod := &BoardModerator{
		BoardID:     boardID,
		AccountID:   accountID,
		Permissions: perms,
		AddedAt:     time.Now(),
		AddedBy:     addedBy,
	}

	return s.store.AddModerator(ctx, mod)
}

// RemoveModerator removes a moderator from a board.
func (s *Service) RemoveModerator(ctx context.Context, boardID, accountID string) error {
	return s.store.RemoveModerator(ctx, boardID, accountID)
}

// IsModerator checks if a user is a moderator of a board.
func (s *Service) IsModerator(ctx context.Context, boardID, accountID string) (bool, error) {
	mod, err := s.store.GetModerator(ctx, boardID, accountID)
	if err == ErrNotModerator {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return mod != nil, nil
}

// GetModeratorPerms gets a moderator's permissions.
func (s *Service) GetModeratorPerms(ctx context.Context, boardID, accountID string) (*ModPerms, error) {
	mod, err := s.store.GetModerator(ctx, boardID, accountID)
	if err != nil {
		return nil, err
	}
	return &mod.Permissions, nil
}

// ListModerators lists moderators of a board.
func (s *Service) ListModerators(ctx context.Context, boardID string) ([]*BoardModerator, error) {
	return s.store.ListModerators(ctx, boardID)
}

// List lists boards.
func (s *Service) List(ctx context.Context, opts ListOpts) ([]*Board, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 25
	}
	return s.store.List(ctx, opts)
}

// Search searches for boards.
func (s *Service) Search(ctx context.Context, query string, limit int) ([]*Board, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	return s.store.Search(ctx, query, limit)
}

// ListPopular lists popular boards.
func (s *Service) ListPopular(ctx context.Context, limit int) ([]*Board, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	return s.store.ListPopular(ctx, limit)
}

// ListNew lists newly created boards.
func (s *Service) ListNew(ctx context.Context, limit int) ([]*Board, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	return s.store.ListNew(ctx, limit)
}

// ListJoined lists boards a user has joined.
func (s *Service) ListJoined(ctx context.Context, accountID string) ([]*Board, error) {
	return s.store.ListJoinedBoards(ctx, accountID)
}

// ListModerated lists boards a user moderates.
func (s *Service) ListModerated(ctx context.Context, accountID string) ([]*Board, error) {
	return s.store.ListModeratedBoards(ctx, accountID)
}

// EnrichBoard enriches a board with viewer state.
func (s *Service) EnrichBoard(ctx context.Context, board *Board, viewerID string) error {
	if viewerID == "" {
		return nil
	}

	isMember, err := s.IsMember(ctx, board.ID, viewerID)
	if err != nil {
		return err
	}
	board.IsJoined = isMember

	isMod, err := s.IsModerator(ctx, board.ID, viewerID)
	if err != nil {
		return err
	}
	board.IsModerator = isMod

	return nil
}

// EnrichBoards enriches multiple boards with viewer state.
func (s *Service) EnrichBoards(ctx context.Context, boards []*Board, viewerID string) error {
	if viewerID == "" || len(boards) == 0 {
		return nil
	}

	// Collect board IDs
	boardIDs := make([]string, len(boards))
	for i, b := range boards {
		boardIDs[i] = b.ID
	}

	// Batch fetch membership status
	members, err := s.store.GetMemberBoards(ctx, viewerID, boardIDs)
	if err != nil {
		return err
	}

	// Batch fetch moderator status
	moderators, err := s.store.GetModeratorBoards(ctx, viewerID, boardIDs)
	if err != nil {
		return err
	}

	// Assign states to boards
	for _, board := range boards {
		_, board.IsJoined = members[board.ID]
		_, board.IsModerator = moderators[board.ID]
	}

	return nil
}

// IncrementThreadCount updates the thread count.
func (s *Service) IncrementThreadCount(ctx context.Context, boardID string, delta int64) error {
	board, err := s.store.GetByID(ctx, boardID)
	if err != nil {
		return err
	}

	board.ThreadCount += delta
	if board.ThreadCount < 0 {
		board.ThreadCount = 0
	}

	return s.store.Update(ctx, board)
}
