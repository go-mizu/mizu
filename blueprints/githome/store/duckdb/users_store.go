package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/users"
)

// UsersStore handles user data access.
type UsersStore struct {
	db *sql.DB
}

// NewUsersStore creates a new users store.
func NewUsersStore(db *sql.DB) *UsersStore {
	return &UsersStore{db: db}
}

func (s *UsersStore) Create(ctx context.Context, u *users.User) error {
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO users (node_id, login, name, email, avatar_url, gravatar_id, type, site_admin,
			bio, blog, location, company, hireable, twitter_username, public_repos, public_gists,
			followers, following, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
		RETURNING id
	`, "", u.Login, u.Name, nullString(u.Email), u.AvatarURL, u.GravatarID, u.Type, u.SiteAdmin,
		u.Bio, u.Blog, u.Location, u.Company, u.Hireable, u.TwitterUsername, u.PublicRepos,
		u.PublicGists, u.Followers, u.Following, u.PasswordHash, u.CreatedAt, u.UpdatedAt,
	).Scan(&u.ID)
	if err != nil {
		return err
	}

	u.NodeID = generateNodeID("U", u.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE users SET node_id = $1 WHERE id = $2`, u.NodeID, u.ID)
	return err
}

func (s *UsersStore) GetByID(ctx context.Context, id int64) (*users.User, error) {
	u := &users.User{}
	var email sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, login, name, email, avatar_url, gravatar_id, type, site_admin,
			bio, blog, location, company, hireable, twitter_username, public_repos, public_gists,
			followers, following, password_hash, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.NodeID, &u.Login, &u.Name, &email, &u.AvatarURL, &u.GravatarID,
		&u.Type, &u.SiteAdmin, &u.Bio, &u.Blog, &u.Location, &u.Company, &u.Hireable,
		&u.TwitterUsername, &u.PublicRepos, &u.PublicGists, &u.Followers, &u.Following,
		&u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if email.Valid {
		u.Email = email.String
	}
	return u, err
}

func (s *UsersStore) GetByLogin(ctx context.Context, login string) (*users.User, error) {
	u := &users.User{}
	var email sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, login, name, email, avatar_url, gravatar_id, type, site_admin,
			bio, blog, location, company, hireable, twitter_username, public_repos, public_gists,
			followers, following, password_hash, created_at, updated_at
		FROM users WHERE login = $1
	`, login).Scan(&u.ID, &u.NodeID, &u.Login, &u.Name, &email, &u.AvatarURL, &u.GravatarID,
		&u.Type, &u.SiteAdmin, &u.Bio, &u.Blog, &u.Location, &u.Company, &u.Hireable,
		&u.TwitterUsername, &u.PublicRepos, &u.PublicGists, &u.Followers, &u.Following,
		&u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if email.Valid {
		u.Email = email.String
	}
	return u, err
}

func (s *UsersStore) GetByEmail(ctx context.Context, email string) (*users.User, error) {
	u := &users.User{}
	var emailVal sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, login, name, email, avatar_url, gravatar_id, type, site_admin,
			bio, blog, location, company, hireable, twitter_username, public_repos, public_gists,
			followers, following, password_hash, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.NodeID, &u.Login, &u.Name, &emailVal, &u.AvatarURL, &u.GravatarID,
		&u.Type, &u.SiteAdmin, &u.Bio, &u.Blog, &u.Location, &u.Company, &u.Hireable,
		&u.TwitterUsername, &u.PublicRepos, &u.PublicGists, &u.Followers, &u.Following,
		&u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if emailVal.Valid {
		u.Email = emailVal.String
	}
	return u, err
}

func (s *UsersStore) Update(ctx context.Context, id int64, in *users.UpdateIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET
			name = COALESCE($2, name),
			email = COALESCE($3, email),
			blog = COALESCE($4, blog),
			twitter_username = COALESCE($5, twitter_username),
			company = COALESCE($6, company),
			location = COALESCE($7, location),
			hireable = COALESCE($8, hireable),
			bio = COALESCE($9, bio),
			updated_at = $10
		WHERE id = $1
	`, id, nullStringPtr(in.Name), nullStringPtr(in.Email), nullStringPtr(in.Blog),
		nullStringPtr(in.TwitterUsername), nullStringPtr(in.Company), nullStringPtr(in.Location),
		nullBoolPtr(in.Hireable), nullStringPtr(in.Bio), time.Now())
	return err
}

func (s *UsersStore) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET password_hash = $2, updated_at = $3 WHERE id = $1
	`, id, passwordHash, time.Now())
	return err
}

func (s *UsersStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

func (s *UsersStore) List(ctx context.Context, opts *users.ListOpts) ([]*users.User, error) {
	query := `
		SELECT id, node_id, login, name, email, avatar_url, gravatar_id, type, site_admin,
			bio, blog, location, company, hireable, twitter_username, public_repos, public_gists,
			followers, following, password_hash, created_at, updated_at
		FROM users`

	var args []any
	if opts != nil && opts.Since > 0 {
		query += ` WHERE id > $1`
		args = append(args, opts.Since)
	}
	query += ` ORDER BY id ASC`

	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanUsers(rows)
}

func (s *UsersStore) CreateFollow(ctx context.Context, followerID, followedID int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_follows (follower_id, followed_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`, followerID, followedID, time.Now())
	return err
}

func (s *UsersStore) DeleteFollow(ctx context.Context, followerID, followedID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM user_follows WHERE follower_id = $1 AND followed_id = $2
	`, followerID, followedID)
	return err
}

func (s *UsersStore) IsFollowing(ctx context.Context, followerID, followedID int64) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM user_follows WHERE follower_id = $1 AND followed_id = $2
	`, followerID, followedID).Scan(&count)
	return count > 0, err
}

func (s *UsersStore) ListFollowers(ctx context.Context, userID int64, opts *users.ListOpts) ([]*users.SimpleUser, error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	query := applyPagination(`
		SELECT u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin
		FROM users u
		JOIN user_follows f ON f.follower_id = u.id
		WHERE f.followed_id = $1
		ORDER BY f.created_at DESC
	`, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSimpleUsers(rows)
}

func (s *UsersStore) ListFollowing(ctx context.Context, userID int64, opts *users.ListOpts) ([]*users.SimpleUser, error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	query := applyPagination(`
		SELECT u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin
		FROM users u
		JOIN user_follows f ON f.followed_id = u.id
		WHERE f.follower_id = $1
		ORDER BY f.created_at DESC
	`, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSimpleUsers(rows)
}

func (s *UsersStore) IncrementFollowers(ctx context.Context, userID int64, delta int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET followers = followers + $2, updated_at = $3 WHERE id = $1
	`, userID, delta, time.Now())
	return err
}

func (s *UsersStore) IncrementFollowing(ctx context.Context, userID int64, delta int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET following = following + $2, updated_at = $3 WHERE id = $1
	`, userID, delta, time.Now())
	return err
}

// Helper functions

func scanUsers(rows *sql.Rows) ([]*users.User, error) {
	var list []*users.User
	for rows.Next() {
		u := &users.User{}
		var email sql.NullString
		if err := rows.Scan(&u.ID, &u.NodeID, &u.Login, &u.Name, &email, &u.AvatarURL,
			&u.GravatarID, &u.Type, &u.SiteAdmin, &u.Bio, &u.Blog, &u.Location, &u.Company,
			&u.Hireable, &u.TwitterUsername, &u.PublicRepos, &u.PublicGists, &u.Followers,
			&u.Following, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		if email.Valid {
			u.Email = email.String
		}
		list = append(list, u)
	}
	return list, rows.Err()
}

func scanSimpleUsers(rows *sql.Rows) ([]*users.SimpleUser, error) {
	var list []*users.SimpleUser
	for rows.Next() {
		u := &users.SimpleUser{}
		var email sql.NullString
		if err := rows.Scan(&u.ID, &u.NodeID, &u.Login, &u.Name, &email, &u.AvatarURL,
			&u.Type, &u.SiteAdmin); err != nil {
			return nil, err
		}
		if email.Valid {
			u.Email = email.String
		}
		list = append(list, u)
	}
	return list, rows.Err()
}
