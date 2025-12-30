package duckdb

import (
	"context"
	"database/sql"
	"time"
)

// User represents a WordPress user.
type User struct {
	ID                string
	UserLogin         string
	UserPass          string
	UserNicename      string
	UserEmail         string
	UserURL           string
	UserRegistered    time.Time
	UserActivationKey string
	UserStatus        int
	DisplayName       string
}

// UsersStore handles user persistence.
type UsersStore struct {
	db *sql.DB
}

// NewUsersStore creates a new users store.
func NewUsersStore(db *sql.DB) *UsersStore {
	return &UsersStore{db: db}
}

// Create creates a new user.
func (s *UsersStore) Create(ctx context.Context, u *User) error {
	query := `
		INSERT INTO wp_users (ID, user_login, user_pass, user_nicename, user_email, user_url,
			user_registered, user_activation_key, user_status, display_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := s.db.ExecContext(ctx, query,
		u.ID, u.UserLogin, u.UserPass, u.UserNicename, u.UserEmail, u.UserURL,
		u.UserRegistered, u.UserActivationKey, u.UserStatus, u.DisplayName,
	)
	return err
}

// GetByID retrieves a user by ID.
func (s *UsersStore) GetByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT ID, user_login, user_pass, user_nicename, user_email, user_url,
			user_registered, user_activation_key, user_status, display_name
		FROM wp_users WHERE ID = $1
	`
	u := &User{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.UserLogin, &u.UserPass, &u.UserNicename, &u.UserEmail, &u.UserURL,
		&u.UserRegistered, &u.UserActivationKey, &u.UserStatus, &u.DisplayName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// GetByLogin retrieves a user by login name.
func (s *UsersStore) GetByLogin(ctx context.Context, login string) (*User, error) {
	query := `
		SELECT ID, user_login, user_pass, user_nicename, user_email, user_url,
			user_registered, user_activation_key, user_status, display_name
		FROM wp_users WHERE LOWER(user_login) = LOWER($1)
	`
	u := &User{}
	err := s.db.QueryRowContext(ctx, query, login).Scan(
		&u.ID, &u.UserLogin, &u.UserPass, &u.UserNicename, &u.UserEmail, &u.UserURL,
		&u.UserRegistered, &u.UserActivationKey, &u.UserStatus, &u.DisplayName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// GetByEmail retrieves a user by email.
func (s *UsersStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT ID, user_login, user_pass, user_nicename, user_email, user_url,
			user_registered, user_activation_key, user_status, display_name
		FROM wp_users WHERE LOWER(user_email) = LOWER($1)
	`
	u := &User{}
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.UserLogin, &u.UserPass, &u.UserNicename, &u.UserEmail, &u.UserURL,
		&u.UserRegistered, &u.UserActivationKey, &u.UserStatus, &u.DisplayName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// GetByNicename retrieves a user by nicename (slug).
func (s *UsersStore) GetByNicename(ctx context.Context, nicename string) (*User, error) {
	query := `
		SELECT ID, user_login, user_pass, user_nicename, user_email, user_url,
			user_registered, user_activation_key, user_status, display_name
		FROM wp_users WHERE user_nicename = $1
	`
	u := &User{}
	err := s.db.QueryRowContext(ctx, query, nicename).Scan(
		&u.ID, &u.UserLogin, &u.UserPass, &u.UserNicename, &u.UserEmail, &u.UserURL,
		&u.UserRegistered, &u.UserActivationKey, &u.UserStatus, &u.DisplayName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// Update updates a user.
func (s *UsersStore) Update(ctx context.Context, u *User) error {
	query := `
		UPDATE wp_users SET
			user_login = $2,
			user_pass = $3,
			user_nicename = $4,
			user_email = $5,
			user_url = $6,
			user_activation_key = $7,
			user_status = $8,
			display_name = $9
		WHERE ID = $1
	`
	_, err := s.db.ExecContext(ctx, query,
		u.ID, u.UserLogin, u.UserPass, u.UserNicename, u.UserEmail, u.UserURL,
		u.UserActivationKey, u.UserStatus, u.DisplayName,
	)
	return err
}

// Delete deletes a user by ID.
func (s *UsersStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM wp_users WHERE ID = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// ListOpts contains options for listing users.
type UserListOpts struct {
	Limit   int
	Offset  int
	OrderBy string
	Order   string
	Search  string
	Role    string
	Include []string
	Exclude []string
}

// List lists users with pagination.
func (s *UsersStore) List(ctx context.Context, opts UserListOpts) ([]*User, int, error) {
	// Build query
	query := `SELECT ID, user_login, user_pass, user_nicename, user_email, user_url,
		user_registered, user_activation_key, user_status, display_name FROM wp_users`
	countQuery := `SELECT COUNT(*) FROM wp_users`

	var args []interface{}
	var where []string
	argNum := 1

	if opts.Search != "" {
		where = append(where, "(LOWER(user_login) LIKE $"+string(rune('0'+argNum))+" OR LOWER(user_email) LIKE $"+string(rune('0'+argNum))+" OR LOWER(display_name) LIKE $"+string(rune('0'+argNum))+")")
		args = append(args, "%"+opts.Search+"%")
		argNum++
	}

	if len(where) > 0 {
		query += " WHERE " + where[0]
		countQuery += " WHERE " + where[0]
		for i := 1; i < len(where); i++ {
			query += " AND " + where[i]
			countQuery += " AND " + where[i]
		}
	}

	// Order
	orderBy := "user_registered"
	if opts.OrderBy != "" {
		orderBy = opts.OrderBy
	}
	order := "DESC"
	if opts.Order != "" {
		order = opts.Order
	}
	query += " ORDER BY " + orderBy + " " + order

	// Pagination
	if opts.Limit > 0 {
		query += " LIMIT " + string(rune('0'+opts.Limit))
	}
	if opts.Offset > 0 {
		query += " OFFSET " + string(rune('0'+opts.Offset))
	}

	// Execute count query
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Execute main query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(
			&u.ID, &u.UserLogin, &u.UserPass, &u.UserNicename, &u.UserEmail, &u.UserURL,
			&u.UserRegistered, &u.UserActivationKey, &u.UserStatus, &u.DisplayName,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}

	return users, total, rows.Err()
}

// Count returns the total number of users.
func (s *UsersStore) Count(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM wp_users`).Scan(&count)
	return count, err
}
