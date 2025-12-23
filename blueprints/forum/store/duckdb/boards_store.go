package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
)

// BoardsStore implements boards.Store.
type BoardsStore struct {
	db *sql.DB
}

// NewBoardsStore creates a new boards store.
func NewBoardsStore(db *sql.DB) *BoardsStore {
	return &BoardsStore{db: db}
}

// Create creates a board.
func (s *BoardsStore) Create(ctx context.Context, board *boards.Board) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO boards (
			id, name, title, description, sidebar, sidebar_html,
			icon_url, banner_url, primary_color, is_nsfw, is_private, is_archived,
			member_count, thread_count, created_at, created_by, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, board.ID, board.Name, board.Title, board.Description, board.Sidebar, board.SidebarHTML,
		board.IconURL, board.BannerURL, board.PrimaryColor, board.IsNSFW, board.IsPrivate, board.IsArchived,
		board.MemberCount, board.ThreadCount, board.CreatedAt, board.CreatedBy, board.UpdatedAt)
	return err
}

// GetByName retrieves a board by name.
func (s *BoardsStore) GetByName(ctx context.Context, name string) (*boards.Board, error) {
	return s.scanBoard(s.db.QueryRowContext(ctx, `
		SELECT id, name, title, description, sidebar, sidebar_html,
			icon_url, banner_url, primary_color, is_nsfw, is_private, is_archived,
			member_count, thread_count, created_at, created_by, updated_at
		FROM boards WHERE LOWER(name) = LOWER($1)
	`, name))
}

// GetByID retrieves a board by ID.
func (s *BoardsStore) GetByID(ctx context.Context, id string) (*boards.Board, error) {
	return s.scanBoard(s.db.QueryRowContext(ctx, `
		SELECT id, name, title, description, sidebar, sidebar_html,
			icon_url, banner_url, primary_color, is_nsfw, is_private, is_archived,
			member_count, thread_count, created_at, created_by, updated_at
		FROM boards WHERE id = $1
	`, id))
}

// Update updates a board.
func (s *BoardsStore) Update(ctx context.Context, board *boards.Board) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE boards SET
			name = $2, title = $3, description = $4, sidebar = $5, sidebar_html = $6,
			icon_url = $7, banner_url = $8, primary_color = $9, is_nsfw = $10,
			is_private = $11, is_archived = $12, member_count = $13, thread_count = $14,
			updated_at = $15
		WHERE id = $1
	`, board.ID, board.Name, board.Title, board.Description, board.Sidebar, board.SidebarHTML,
		board.IconURL, board.BannerURL, board.PrimaryColor, board.IsNSFW,
		board.IsPrivate, board.IsArchived, board.MemberCount, board.ThreadCount,
		board.UpdatedAt)
	return err
}

// Delete deletes a board.
func (s *BoardsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM boards WHERE id = $1`, id)
	return err
}

// AddMember adds a member to a board.
func (s *BoardsStore) AddMember(ctx context.Context, member *boards.BoardMember) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO board_members (board_id, account_id, joined_at)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`, member.BoardID, member.AccountID, member.JoinedAt)
	return err
}

// RemoveMember removes a member from a board.
func (s *BoardsStore) RemoveMember(ctx context.Context, boardID, accountID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM board_members WHERE board_id = $1 AND account_id = $2
	`, boardID, accountID)
	return err
}

// GetMember retrieves a member.
func (s *BoardsStore) GetMember(ctx context.Context, boardID, accountID string) (*boards.BoardMember, error) {
	member := &boards.BoardMember{}
	err := s.db.QueryRowContext(ctx, `
		SELECT board_id, account_id, joined_at
		FROM board_members WHERE board_id = $1 AND account_id = $2
	`, boardID, accountID).Scan(&member.BoardID, &member.AccountID, &member.JoinedAt)
	if err == sql.ErrNoRows {
		return nil, boards.ErrNotMember
	}
	if err != nil {
		return nil, err
	}
	return member, nil
}

// ListMembers lists board members.
func (s *BoardsStore) ListMembers(ctx context.Context, boardID string, opts boards.ListOpts) ([]*accounts.Account, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.username, a.email, a.password_hash, a.display_name, a.bio,
			a.avatar_url, a.banner_url, a.karma, a.post_karma, a.comment_karma,
			a.is_admin, a.is_suspended, a.suspend_reason, a.suspend_until,
			a.created_at, a.updated_at
		FROM accounts a
		JOIN board_members bm ON a.id = bm.account_id
		WHERE bm.board_id = $1
		ORDER BY bm.joined_at DESC
		LIMIT $2
	`, boardID, opts.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*accounts.Account
	for rows.Next() {
		account := &accounts.Account{}
		var suspendReason sql.NullString
		var suspendUntil sql.NullTime
		err := rows.Scan(
			&account.ID, &account.Username, &account.Email, &account.PasswordHash,
			&account.DisplayName, &account.Bio, &account.AvatarURL, &account.BannerURL,
			&account.Karma, &account.PostKarma, &account.CommentKarma,
			&account.IsAdmin, &account.IsSuspended, &suspendReason, &suspendUntil,
			&account.CreatedAt, &account.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if suspendReason.Valid {
			account.SuspendReason = suspendReason.String
		}
		if suspendUntil.Valid {
			account.SuspendUntil = &suspendUntil.Time
		}
		result = append(result, account)
	}
	return result, rows.Err()
}

// ListJoinedBoards lists boards a user has joined.
func (s *BoardsStore) ListJoinedBoards(ctx context.Context, accountID string) ([]*boards.Board, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT b.id, b.name, b.title, b.description, b.sidebar, b.sidebar_html,
			b.icon_url, b.banner_url, b.primary_color, b.is_nsfw, b.is_private, b.is_archived,
			b.member_count, b.thread_count, b.created_at, b.created_by, b.updated_at
		FROM boards b
		JOIN board_members bm ON b.id = bm.board_id
		WHERE bm.account_id = $1
		ORDER BY b.name
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanBoards(rows)
}

// AddModerator adds a moderator.
func (s *BoardsStore) AddModerator(ctx context.Context, mod *boards.BoardModerator) error {
	perms, _ := json.Marshal(mod.Permissions)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO board_moderators (board_id, account_id, permissions, added_at, added_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (board_id, account_id) DO UPDATE SET permissions = $3
	`, mod.BoardID, mod.AccountID, string(perms), mod.AddedAt, mod.AddedBy)
	return err
}

// RemoveModerator removes a moderator.
func (s *BoardsStore) RemoveModerator(ctx context.Context, boardID, accountID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM board_moderators WHERE board_id = $1 AND account_id = $2
	`, boardID, accountID)
	return err
}

// GetModerator retrieves a moderator.
func (s *BoardsStore) GetModerator(ctx context.Context, boardID, accountID string) (*boards.BoardModerator, error) {
	mod := &boards.BoardModerator{}
	var permsJSON sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT board_id, account_id, permissions, added_at, added_by
		FROM board_moderators WHERE board_id = $1 AND account_id = $2
	`, boardID, accountID).Scan(&mod.BoardID, &mod.AccountID, &permsJSON, &mod.AddedAt, &mod.AddedBy)
	if err == sql.ErrNoRows {
		return nil, boards.ErrNotModerator
	}
	if err != nil {
		return nil, err
	}
	if permsJSON.Valid {
		_ = json.Unmarshal([]byte(permsJSON.String), &mod.Permissions)
	}
	return mod, nil
}

// ListModerators lists board moderators.
func (s *BoardsStore) ListModerators(ctx context.Context, boardID string) ([]*boards.BoardModerator, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT board_id, account_id, permissions, added_at, added_by
		FROM board_moderators WHERE board_id = $1
		ORDER BY added_at
	`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*boards.BoardModerator
	for rows.Next() {
		mod := &boards.BoardModerator{}
		var permsJSON sql.NullString
		err := rows.Scan(&mod.BoardID, &mod.AccountID, &permsJSON, &mod.AddedAt, &mod.AddedBy)
		if err != nil {
			return nil, err
		}
		if permsJSON.Valid {
			_ = json.Unmarshal([]byte(permsJSON.String), &mod.Permissions)
		}
		result = append(result, mod)
	}
	return result, rows.Err()
}

// ListModeratedBoards lists boards a user moderates.
func (s *BoardsStore) ListModeratedBoards(ctx context.Context, accountID string) ([]*boards.Board, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT b.id, b.name, b.title, b.description, b.sidebar, b.sidebar_html,
			b.icon_url, b.banner_url, b.primary_color, b.is_nsfw, b.is_private, b.is_archived,
			b.member_count, b.thread_count, b.created_at, b.created_by, b.updated_at
		FROM boards b
		JOIN board_moderators bm ON b.id = bm.board_id
		WHERE bm.account_id = $1
		ORDER BY b.name
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanBoards(rows)
}

// List lists boards.
func (s *BoardsStore) List(ctx context.Context, opts boards.ListOpts) ([]*boards.Board, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, title, description, sidebar, sidebar_html,
			icon_url, banner_url, primary_color, is_nsfw, is_private, is_archived,
			member_count, thread_count, created_at, created_by, updated_at
		FROM boards
		WHERE NOT is_private
		ORDER BY member_count DESC
		LIMIT $1
	`, opts.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanBoards(rows)
}

// Search searches boards.
func (s *BoardsStore) Search(ctx context.Context, query string, limit int) ([]*boards.Board, error) {
	pattern := "%" + strings.ToLower(query) + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, title, description, sidebar, sidebar_html,
			icon_url, banner_url, primary_color, is_nsfw, is_private, is_archived,
			member_count, thread_count, created_at, created_by, updated_at
		FROM boards
		WHERE (LOWER(name) LIKE $1 OR LOWER(title) LIKE $1) AND NOT is_private
		ORDER BY member_count DESC
		LIMIT $2
	`, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanBoards(rows)
}

// ListPopular lists popular boards.
func (s *BoardsStore) ListPopular(ctx context.Context, limit int) ([]*boards.Board, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, title, description, sidebar, sidebar_html,
			icon_url, banner_url, primary_color, is_nsfw, is_private, is_archived,
			member_count, thread_count, created_at, created_by, updated_at
		FROM boards
		WHERE NOT is_private
		ORDER BY member_count DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanBoards(rows)
}

// ListNew lists new boards.
func (s *BoardsStore) ListNew(ctx context.Context, limit int) ([]*boards.Board, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, title, description, sidebar, sidebar_html,
			icon_url, banner_url, primary_color, is_nsfw, is_private, is_archived,
			member_count, thread_count, created_at, created_by, updated_at
		FROM boards
		WHERE NOT is_private
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanBoards(rows)
}

func (s *BoardsStore) scanBoard(row *sql.Row) (*boards.Board, error) {
	board := &boards.Board{}
	err := row.Scan(
		&board.ID, &board.Name, &board.Title, &board.Description,
		&board.Sidebar, &board.SidebarHTML, &board.IconURL, &board.BannerURL,
		&board.PrimaryColor, &board.IsNSFW, &board.IsPrivate, &board.IsArchived,
		&board.MemberCount, &board.ThreadCount, &board.CreatedAt, &board.CreatedBy, &board.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, boards.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return board, nil
}

func (s *BoardsStore) scanBoards(rows *sql.Rows) ([]*boards.Board, error) {
	var result []*boards.Board
	for rows.Next() {
		board := &boards.Board{}
		err := rows.Scan(
			&board.ID, &board.Name, &board.Title, &board.Description,
			&board.Sidebar, &board.SidebarHTML, &board.IconURL, &board.BannerURL,
			&board.PrimaryColor, &board.IsNSFW, &board.IsPrivate, &board.IsArchived,
			&board.MemberCount, &board.ThreadCount, &board.CreatedAt, &board.CreatedBy, &board.UpdatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, board)
	}
	return result, rows.Err()
}
