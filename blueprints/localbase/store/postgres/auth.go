package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuthStore implements store.AuthStore using PostgreSQL.
type AuthStore struct {
	pool *pgxpool.Pool
}

// CreateUser creates a new user.
func (s *AuthStore) CreateUser(ctx context.Context, user *store.User) error {
	sql := `
	INSERT INTO auth.users (
		id, email, phone, encrypted_password, email_confirmed_at, phone_confirmed_at,
		raw_app_meta_data, raw_user_meta_data, is_super_admin, role,
		confirmation_token, recovery_token
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
	)
	`

	appMeta := user.AppMetadata
	if appMeta == nil {
		appMeta = make(map[string]any)
	}
	userMeta := user.UserMetadata
	if userMeta == nil {
		userMeta = make(map[string]any)
	}

	_, err := s.pool.Exec(ctx, sql,
		user.ID,
		nullIfEmpty(user.Email),
		nullIfEmpty(user.Phone),
		user.EncryptedPassword,
		user.EmailConfirmedAt,
		user.PhoneConfirmedAt,
		appMeta,
		userMeta,
		user.IsSuperAdmin,
		user.Role,
		nullIfEmpty(user.ConfirmationToken),
		nullIfEmpty(user.RecoveryToken),
	)

	return err
}

// GetUserByID retrieves a user by ID.
func (s *AuthStore) GetUserByID(ctx context.Context, id string) (*store.User, error) {
	sql := `
	SELECT id, email, phone, encrypted_password, email_confirmed_at, phone_confirmed_at,
		raw_app_meta_data, raw_user_meta_data, is_super_admin, role,
		created_at, updated_at, last_sign_in_at, banned_until,
		confirmation_token, recovery_token
	FROM auth.users
	WHERE id = $1
	`

	user := &store.User{}
	var email, phone, confirmationToken, recoveryToken *string

	err := s.pool.QueryRow(ctx, sql, id).Scan(
		&user.ID,
		&email,
		&phone,
		&user.EncryptedPassword,
		&user.EmailConfirmedAt,
		&user.PhoneConfirmedAt,
		&user.AppMetadata,
		&user.UserMetadata,
		&user.IsSuperAdmin,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastSignInAt,
		&user.BannedUntil,
		&confirmationToken,
		&recoveryToken,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}

	if email != nil {
		user.Email = *email
	}
	if phone != nil {
		user.Phone = *phone
	}
	if confirmationToken != nil {
		user.ConfirmationToken = *confirmationToken
	}
	if recoveryToken != nil {
		user.RecoveryToken = *recoveryToken
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email.
func (s *AuthStore) GetUserByEmail(ctx context.Context, email string) (*store.User, error) {
	sql := `
	SELECT id FROM auth.users WHERE email = $1
	`

	var id string
	err := s.pool.QueryRow(ctx, sql, email).Scan(&id)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}

	return s.GetUserByID(ctx, id)
}

// GetUserByPhone retrieves a user by phone.
func (s *AuthStore) GetUserByPhone(ctx context.Context, phone string) (*store.User, error) {
	sql := `
	SELECT id FROM auth.users WHERE phone = $1
	`

	var id string
	err := s.pool.QueryRow(ctx, sql, phone).Scan(&id)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}

	return s.GetUserByID(ctx, id)
}

// UpdateUser updates a user.
func (s *AuthStore) UpdateUser(ctx context.Context, user *store.User) error {
	sql := `
	UPDATE auth.users
	SET email = $2, phone = $3, encrypted_password = $4, email_confirmed_at = $5,
		phone_confirmed_at = $6, raw_app_meta_data = $7, raw_user_meta_data = $8,
		is_super_admin = $9, role = $10, last_sign_in_at = $11, banned_until = $12,
		confirmation_token = $13, recovery_token = $14, updated_at = NOW()
	WHERE id = $1
	`

	_, err := s.pool.Exec(ctx, sql,
		user.ID,
		nullIfEmpty(user.Email),
		nullIfEmpty(user.Phone),
		user.EncryptedPassword,
		user.EmailConfirmedAt,
		user.PhoneConfirmedAt,
		user.AppMetadata,
		user.UserMetadata,
		user.IsSuperAdmin,
		user.Role,
		user.LastSignInAt,
		user.BannedUntil,
		nullIfEmpty(user.ConfirmationToken),
		nullIfEmpty(user.RecoveryToken),
	)

	return err
}

// DeleteUser deletes a user.
func (s *AuthStore) DeleteUser(ctx context.Context, id string) error {
	sql := `DELETE FROM auth.users WHERE id = $1`
	_, err := s.pool.Exec(ctx, sql, id)
	return err
}

// ListUsers lists users with pagination.
func (s *AuthStore) ListUsers(ctx context.Context, page, perPage int) ([]*store.User, int, error) {
	countSQL := `SELECT COUNT(*) FROM auth.users`
	var total int
	if err := s.pool.QueryRow(ctx, countSQL).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	sql := `
	SELECT id, email, phone, email_confirmed_at, phone_confirmed_at,
		raw_app_meta_data, raw_user_meta_data, is_super_admin, role,
		created_at, updated_at, last_sign_in_at, banned_until
	FROM auth.users
	ORDER BY created_at DESC
	LIMIT $1 OFFSET $2
	`

	rows, err := s.pool.Query(ctx, sql, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*store.User
	for rows.Next() {
		user := &store.User{}
		var email, phone *string

		err := rows.Scan(
			&user.ID,
			&email,
			&phone,
			&user.EmailConfirmedAt,
			&user.PhoneConfirmedAt,
			&user.AppMetadata,
			&user.UserMetadata,
			&user.IsSuperAdmin,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.LastSignInAt,
			&user.BannedUntil,
		)
		if err != nil {
			return nil, 0, err
		}

		if email != nil {
			user.Email = *email
		}
		if phone != nil {
			user.Phone = *phone
		}

		users = append(users, user)
	}

	return users, total, nil
}

// CreateSession creates a new session.
func (s *AuthStore) CreateSession(ctx context.Context, session *store.Session) error {
	sql := `
	INSERT INTO auth.sessions (id, user_id, factor_id, aal, not_after)
	VALUES ($1, $2, $3, $4, $5)
	`

	_, err := s.pool.Exec(ctx, sql,
		session.ID,
		session.UserID,
		nullIfEmpty(session.FactorID),
		session.AAL,
		session.NotAfter,
	)

	return err
}

// GetSession retrieves a session by ID.
func (s *AuthStore) GetSession(ctx context.Context, id string) (*store.Session, error) {
	sql := `
	SELECT id, user_id, created_at, updated_at, factor_id, aal, not_after
	FROM auth.sessions
	WHERE id = $1
	`

	session := &store.Session{}
	var factorID *string

	err := s.pool.QueryRow(ctx, sql, id).Scan(
		&session.ID,
		&session.UserID,
		&session.CreatedAt,
		&session.UpdatedAt,
		&factorID,
		&session.AAL,
		&session.NotAfter,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, err
	}

	if factorID != nil {
		session.FactorID = *factorID
	}

	return session, nil
}

// DeleteSession deletes a session.
func (s *AuthStore) DeleteSession(ctx context.Context, id string) error {
	sql := `DELETE FROM auth.sessions WHERE id = $1`
	_, err := s.pool.Exec(ctx, sql, id)
	return err
}

// DeleteUserSessions deletes all sessions for a user.
func (s *AuthStore) DeleteUserSessions(ctx context.Context, userID string) error {
	sql := `DELETE FROM auth.sessions WHERE user_id = $1`
	_, err := s.pool.Exec(ctx, sql, userID)
	return err
}

// CreateRefreshToken creates a new refresh token.
func (s *AuthStore) CreateRefreshToken(ctx context.Context, token *store.RefreshToken) error {
	sql := `
	INSERT INTO auth.refresh_tokens (token, user_id, session_id, parent)
	VALUES ($1, $2, $3, $4)
	`

	_, err := s.pool.Exec(ctx, sql,
		token.Token,
		token.UserID,
		nullIfEmpty(token.SessionID),
		nullIfEmpty(token.Parent),
	)

	return err
}

// GetRefreshToken retrieves a refresh token.
func (s *AuthStore) GetRefreshToken(ctx context.Context, token string) (*store.RefreshToken, error) {
	sql := `
	SELECT id, token, user_id, session_id, parent, revoked, created_at, updated_at
	FROM auth.refresh_tokens
	WHERE token = $1
	`

	rt := &store.RefreshToken{}
	var sessionID, parent *string

	err := s.pool.QueryRow(ctx, sql, token).Scan(
		&rt.ID,
		&rt.Token,
		&rt.UserID,
		&sessionID,
		&parent,
		&rt.Revoked,
		&rt.CreatedAt,
		&rt.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("refresh token not found")
	}
	if err != nil {
		return nil, err
	}

	if sessionID != nil {
		rt.SessionID = *sessionID
	}
	if parent != nil {
		rt.Parent = *parent
	}

	return rt, nil
}

// RevokeRefreshToken revokes a refresh token.
func (s *AuthStore) RevokeRefreshToken(ctx context.Context, token string) error {
	sql := `UPDATE auth.refresh_tokens SET revoked = TRUE, updated_at = NOW() WHERE token = $1`
	_, err := s.pool.Exec(ctx, sql, token)
	return err
}

// RotateRefreshToken rotates a refresh token.
func (s *AuthStore) RotateRefreshToken(ctx context.Context, oldToken, newToken string) error {
	// Get old token info
	old, err := s.GetRefreshToken(ctx, oldToken)
	if err != nil {
		return err
	}

	// Revoke old token
	if err := s.RevokeRefreshToken(ctx, oldToken); err != nil {
		return err
	}

	// Create new token
	newRT := &store.RefreshToken{
		Token:     newToken,
		UserID:    old.UserID,
		SessionID: old.SessionID,
		Parent:    oldToken,
	}

	return s.CreateRefreshToken(ctx, newRT)
}

// CreateMFAFactor creates a new MFA factor.
func (s *AuthStore) CreateMFAFactor(ctx context.Context, factor *store.MFAFactor) error {
	sql := `
	INSERT INTO auth.mfa_factors (id, user_id, friendly_name, factor_type, status, secret)
	VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.pool.Exec(ctx, sql,
		factor.ID,
		factor.UserID,
		nullIfEmpty(factor.FriendlyName),
		factor.FactorType,
		factor.Status,
		factor.Secret,
	)

	return err
}

// GetMFAFactor retrieves an MFA factor by ID.
func (s *AuthStore) GetMFAFactor(ctx context.Context, id string) (*store.MFAFactor, error) {
	sql := `
	SELECT id, user_id, friendly_name, factor_type, status, secret, created_at, updated_at
	FROM auth.mfa_factors
	WHERE id = $1
	`

	factor := &store.MFAFactor{}
	var friendlyName *string

	err := s.pool.QueryRow(ctx, sql, id).Scan(
		&factor.ID,
		&factor.UserID,
		&friendlyName,
		&factor.FactorType,
		&factor.Status,
		&factor.Secret,
		&factor.CreatedAt,
		&factor.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("MFA factor not found")
	}
	if err != nil {
		return nil, err
	}

	if friendlyName != nil {
		factor.FriendlyName = *friendlyName
	}

	return factor, nil
}

// GetUserMFAFactors retrieves all MFA factors for a user.
func (s *AuthStore) GetUserMFAFactors(ctx context.Context, userID string) ([]*store.MFAFactor, error) {
	sql := `
	SELECT id, user_id, friendly_name, factor_type, status, created_at, updated_at
	FROM auth.mfa_factors
	WHERE user_id = $1
	ORDER BY created_at
	`

	rows, err := s.pool.Query(ctx, sql, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var factors []*store.MFAFactor
	for rows.Next() {
		factor := &store.MFAFactor{}
		var friendlyName *string

		err := rows.Scan(
			&factor.ID,
			&factor.UserID,
			&friendlyName,
			&factor.FactorType,
			&factor.Status,
			&factor.CreatedAt,
			&factor.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if friendlyName != nil {
			factor.FriendlyName = *friendlyName
		}

		factors = append(factors, factor)
	}

	return factors, nil
}

// UpdateMFAFactor updates an MFA factor.
func (s *AuthStore) UpdateMFAFactor(ctx context.Context, factor *store.MFAFactor) error {
	sql := `
	UPDATE auth.mfa_factors
	SET friendly_name = $2, status = $3, secret = $4, updated_at = NOW()
	WHERE id = $1
	`

	_, err := s.pool.Exec(ctx, sql,
		factor.ID,
		nullIfEmpty(factor.FriendlyName),
		factor.Status,
		factor.Secret,
	)

	return err
}

// DeleteMFAFactor deletes an MFA factor.
func (s *AuthStore) DeleteMFAFactor(ctx context.Context, id string) error {
	sql := `DELETE FROM auth.mfa_factors WHERE id = $1`
	_, err := s.pool.Exec(ctx, sql, id)
	return err
}

// CreateIdentity creates a new OAuth identity.
func (s *AuthStore) CreateIdentity(ctx context.Context, identity *store.Identity) error {
	sql := `
	INSERT INTO auth.identities (id, user_id, provider, provider_id, identity_data, last_sign_in_at)
	VALUES ($1, $2, $3, $4, $5, $6)
	`

	// provider_id is the email for email provider
	providerID := identity.Email
	if providerID == "" {
		providerID = identity.UserID
	}

	_, err := s.pool.Exec(ctx, sql,
		identity.IdentityID,
		identity.UserID,
		identity.Provider,
		providerID,
		identity.IdentityData,
		identity.LastSignInAt,
	)

	return err
}

// GetIdentity retrieves an identity by provider and provider ID.
func (s *AuthStore) GetIdentity(ctx context.Context, provider, providerIDValue string) (*store.Identity, error) {
	sql := `
	SELECT id, user_id, provider, provider_id, identity_data, last_sign_in_at, created_at, updated_at
	FROM auth.identities
	WHERE provider = $1 AND provider_id = $2
	`

	identity := &store.Identity{}
	var providerID string

	err := s.pool.QueryRow(ctx, sql, provider, providerIDValue).Scan(
		&identity.IdentityID,
		&identity.UserID,
		&identity.Provider,
		&providerID,
		&identity.IdentityData,
		&identity.LastSignInAt,
		&identity.CreatedAt,
		&identity.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("identity not found")
	}
	if err != nil {
		return nil, err
	}

	// Set ID to match user_id (Supabase compatibility)
	identity.ID = identity.UserID
	// Set email from provider_id for email provider
	if identity.Provider == "email" {
		identity.Email = providerID
	}

	return identity, nil
}

// GetUserIdentities retrieves all identities for a user.
func (s *AuthStore) GetUserIdentities(ctx context.Context, userID string) ([]*store.Identity, error) {
	sql := `
	SELECT id, user_id, provider, provider_id, identity_data, last_sign_in_at, created_at, updated_at
	FROM auth.identities
	WHERE user_id = $1
	ORDER BY created_at
	`

	rows, err := s.pool.Query(ctx, sql, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var identities []*store.Identity
	for rows.Next() {
		identity := &store.Identity{}
		var providerID string

		err := rows.Scan(
			&identity.IdentityID,
			&identity.UserID,
			&identity.Provider,
			&providerID,
			&identity.IdentityData,
			&identity.LastSignInAt,
			&identity.CreatedAt,
			&identity.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Set ID to match user_id (Supabase compatibility)
		identity.ID = identity.UserID
		// Set email from provider_id for email provider
		if identity.Provider == "email" {
			identity.Email = providerID
		}

		identities = append(identities, identity)
	}

	return identities, nil
}

// DeleteIdentity deletes an identity.
func (s *AuthStore) DeleteIdentity(ctx context.Context, id string) error {
	sql := `DELETE FROM auth.identities WHERE id = $1`
	_, err := s.pool.Exec(ctx, sql, id)
	return err
}

// Helper functions

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}
