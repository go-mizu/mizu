package duckdb

import (
	"context"
	"database/sql"
	"time"
)

// Link represents a WordPress link (blogroll).
type Link struct {
	LinkID          string
	LinkURL         string
	LinkName        string
	LinkImage       string
	LinkTarget      string
	LinkDescription string
	LinkVisible     string
	LinkOwner       string
	LinkRating      int
	LinkUpdated     time.Time
	LinkRel         string
	LinkNotes       string
	LinkRSS         string
}

// LinksStore handles link persistence.
type LinksStore struct {
	db *sql.DB
}

// NewLinksStore creates a new links store.
func NewLinksStore(db *sql.DB) *LinksStore {
	return &LinksStore{db: db}
}

// Create creates a new link.
func (s *LinksStore) Create(ctx context.Context, l *Link) error {
	query := `INSERT INTO wp_links (link_id, link_url, link_name, link_image, link_target,
		link_description, link_visible, link_owner, link_rating, link_updated, link_rel, link_notes, link_rss)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err := s.db.ExecContext(ctx, query, l.LinkID, l.LinkURL, l.LinkName, l.LinkImage, l.LinkTarget,
		l.LinkDescription, l.LinkVisible, l.LinkOwner, l.LinkRating, l.LinkUpdated, l.LinkRel, l.LinkNotes, l.LinkRSS)
	return err
}

// GetByID retrieves a link by ID.
func (s *LinksStore) GetByID(ctx context.Context, id string) (*Link, error) {
	query := `SELECT link_id, link_url, link_name, link_image, link_target,
		link_description, link_visible, link_owner, link_rating, link_updated, link_rel, link_notes, link_rss
		FROM wp_links WHERE link_id = $1`
	l := &Link{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(&l.LinkID, &l.LinkURL, &l.LinkName, &l.LinkImage, &l.LinkTarget,
		&l.LinkDescription, &l.LinkVisible, &l.LinkOwner, &l.LinkRating, &l.LinkUpdated, &l.LinkRel, &l.LinkNotes, &l.LinkRSS)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return l, err
}

// Delete deletes a link.
func (s *LinksStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM wp_links WHERE link_id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// List lists visible links.
func (s *LinksStore) List(ctx context.Context) ([]*Link, error) {
	query := `SELECT link_id, link_url, link_name, link_image, link_target,
		link_description, link_visible, link_owner, link_rating, link_updated, link_rel, link_notes, link_rss
		FROM wp_links WHERE link_visible = 'Y' ORDER BY link_name`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*Link
	for rows.Next() {
		l := &Link{}
		if err := rows.Scan(&l.LinkID, &l.LinkURL, &l.LinkName, &l.LinkImage, &l.LinkTarget,
			&l.LinkDescription, &l.LinkVisible, &l.LinkOwner, &l.LinkRating, &l.LinkUpdated, &l.LinkRel, &l.LinkNotes, &l.LinkRSS); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

// Nonce represents a CSRF nonce.
type Nonce struct {
	NonceID   string
	UserID    string
	Action    string
	Nonce     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// NoncesStore handles nonce persistence.
type NoncesStore struct {
	db *sql.DB
}

// NewNoncesStore creates a new nonces store.
func NewNoncesStore(db *sql.DB) *NoncesStore {
	return &NoncesStore{db: db}
}

// Create creates a new nonce.
func (s *NoncesStore) Create(ctx context.Context, n *Nonce) error {
	query := `INSERT INTO wp_nonces (nonce_id, user_id, action, nonce, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := s.db.ExecContext(ctx, query, n.NonceID, n.UserID, n.Action, n.Nonce, n.ExpiresAt, n.CreatedAt)
	return err
}

// Verify verifies a nonce and deletes it if valid.
func (s *NoncesStore) Verify(ctx context.Context, nonce, action string) (bool, error) {
	query := `SELECT nonce_id FROM wp_nonces WHERE nonce = $1 AND action = $2 AND expires_at > $3`
	var id string
	err := s.db.QueryRowContext(ctx, query, nonce, action, time.Now()).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Delete the nonce after use
	deleteQuery := `DELETE FROM wp_nonces WHERE nonce_id = $1`
	_, _ = s.db.ExecContext(ctx, deleteQuery, id)

	return true, nil
}

// DeleteExpired deletes all expired nonces.
func (s *NoncesStore) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM wp_nonces WHERE expires_at < $1`
	_, err := s.db.ExecContext(ctx, query, time.Now())
	return err
}

// CronJob represents a scheduled job.
type CronJob struct {
	CronID          string
	Hook            string
	Args            string
	Schedule        string
	IntervalSeconds int
	NextRun         time.Time
	CreatedAt       time.Time
}

// CronStore handles cron job persistence.
type CronStore struct {
	db *sql.DB
}

// NewCronStore creates a new cron store.
func NewCronStore(db *sql.DB) *CronStore {
	return &CronStore{db: db}
}

// Create creates a new cron job.
func (s *CronStore) Create(ctx context.Context, c *CronJob) error {
	query := `INSERT INTO wp_cron (cron_id, hook, args, schedule, interval_seconds, next_run, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := s.db.ExecContext(ctx, query, c.CronID, c.Hook, c.Args, c.Schedule, c.IntervalSeconds, c.NextRun, c.CreatedAt)
	return err
}

// GetDue retrieves all jobs that are due to run.
func (s *CronStore) GetDue(ctx context.Context) ([]*CronJob, error) {
	query := `SELECT cron_id, hook, args, schedule, interval_seconds, next_run, created_at
		FROM wp_cron WHERE next_run <= $1 ORDER BY next_run`
	rows, err := s.db.QueryContext(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*CronJob
	for rows.Next() {
		c := &CronJob{}
		var args sql.NullString
		if err := rows.Scan(&c.CronID, &c.Hook, &args, &c.Schedule, &c.IntervalSeconds, &c.NextRun, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.Args = args.String
		jobs = append(jobs, c)
	}
	return jobs, rows.Err()
}

// UpdateNextRun updates the next run time for a job.
func (s *CronStore) UpdateNextRun(ctx context.Context, id string, nextRun time.Time) error {
	query := `UPDATE wp_cron SET next_run = $2 WHERE cron_id = $1`
	_, err := s.db.ExecContext(ctx, query, id, nextRun)
	return err
}

// Delete deletes a cron job.
func (s *CronStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM wp_cron WHERE cron_id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// DeleteByHook deletes all cron jobs with a specific hook.
func (s *CronStore) DeleteByHook(ctx context.Context, hook string) error {
	query := `DELETE FROM wp_cron WHERE hook = $1`
	_, err := s.db.ExecContext(ctx, query, hook)
	return err
}

// Transient represents a cached transient value.
type Transient struct {
	TransientID    string
	TransientKey   string
	TransientValue string
	ExpiresAt      *time.Time
	CreatedAt      time.Time
}

// TransientsStore handles transient persistence.
type TransientsStore struct {
	db *sql.DB
}

// NewTransientsStore creates a new transients store.
func NewTransientsStore(db *sql.DB) *TransientsStore {
	return &TransientsStore{db: db}
}

// Get retrieves a transient value.
func (s *TransientsStore) Get(ctx context.Context, key string) (string, error) {
	query := `SELECT transient_value FROM wp_transients WHERE transient_key = $1 AND (expires_at IS NULL OR expires_at > $2)`
	var value string
	err := s.db.QueryRowContext(ctx, query, key, time.Now()).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// Set sets a transient value with optional expiration.
func (s *TransientsStore) Set(ctx context.Context, key, value string, expiration *time.Duration) error {
	var expiresAt *time.Time
	if expiration != nil {
		t := time.Now().Add(*expiration)
		expiresAt = &t
	}

	// Try update first
	query := `UPDATE wp_transients SET transient_value = $2, expires_at = $3 WHERE transient_key = $1`
	result, err := s.db.ExecContext(ctx, query, key, value, expiresAt)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		return nil
	}

	// Insert if not exists
	insertQuery := `INSERT INTO wp_transients (transient_id, transient_key, transient_value, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)`
	_, err = s.db.ExecContext(ctx, insertQuery, NewID(), key, value, expiresAt, time.Now())
	return err
}

// Delete deletes a transient.
func (s *TransientsStore) Delete(ctx context.Context, key string) error {
	query := `DELETE FROM wp_transients WHERE transient_key = $1`
	_, err := s.db.ExecContext(ctx, query, key)
	return err
}

// DeleteExpired deletes all expired transients.
func (s *TransientsStore) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM wp_transients WHERE expires_at IS NOT NULL AND expires_at < $1`
	_, err := s.db.ExecContext(ctx, query, time.Now())
	return err
}

// AppPassword represents an application password.
type AppPassword struct {
	ID       string
	UserID   string
	UUID     string
	AppID    string
	Name     string
	Password string
	Created  time.Time
	LastUsed *time.Time
	LastIP   string
}

// AppPasswordsStore handles application password persistence.
type AppPasswordsStore struct {
	db *sql.DB
}

// NewAppPasswordsStore creates a new app passwords store.
func NewAppPasswordsStore(db *sql.DB) *AppPasswordsStore {
	return &AppPasswordsStore{db: db}
}

// Create creates a new application password.
func (s *AppPasswordsStore) Create(ctx context.Context, ap *AppPassword) error {
	query := `INSERT INTO wp_application_passwords (id, user_id, uuid, app_id, name, password, created, last_used, last_ip)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := s.db.ExecContext(ctx, query, ap.ID, ap.UserID, ap.UUID, ap.AppID, ap.Name, ap.Password, ap.Created, ap.LastUsed, ap.LastIP)
	return err
}

// GetByUUID retrieves an application password by UUID.
func (s *AppPasswordsStore) GetByUUID(ctx context.Context, uuid string) (*AppPassword, error) {
	query := `SELECT id, user_id, uuid, app_id, name, password, created, last_used, last_ip
		FROM wp_application_passwords WHERE uuid = $1`
	ap := &AppPassword{}
	var appID, lastIP sql.NullString
	var lastUsed sql.NullTime
	err := s.db.QueryRowContext(ctx, query, uuid).Scan(&ap.ID, &ap.UserID, &ap.UUID, &appID, &ap.Name, &ap.Password, &ap.Created, &lastUsed, &lastIP)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ap.AppID = appID.String
	ap.LastIP = lastIP.String
	if lastUsed.Valid {
		ap.LastUsed = &lastUsed.Time
	}
	return ap, nil
}

// GetByUserID retrieves all application passwords for a user.
func (s *AppPasswordsStore) GetByUserID(ctx context.Context, userID string) ([]*AppPassword, error) {
	query := `SELECT id, user_id, uuid, app_id, name, password, created, last_used, last_ip
		FROM wp_application_passwords WHERE user_id = $1 ORDER BY created DESC`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var passwords []*AppPassword
	for rows.Next() {
		ap := &AppPassword{}
		var appID, lastIP sql.NullString
		var lastUsed sql.NullTime
		if err := rows.Scan(&ap.ID, &ap.UserID, &ap.UUID, &appID, &ap.Name, &ap.Password, &ap.Created, &lastUsed, &lastIP); err != nil {
			return nil, err
		}
		ap.AppID = appID.String
		ap.LastIP = lastIP.String
		if lastUsed.Valid {
			ap.LastUsed = &lastUsed.Time
		}
		passwords = append(passwords, ap)
	}
	return passwords, rows.Err()
}

// Delete deletes an application password.
func (s *AppPasswordsStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM wp_application_passwords WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// UpdateLastUsed updates the last used time and IP for an application password.
func (s *AppPasswordsStore) UpdateLastUsed(ctx context.Context, id, ip string) error {
	query := `UPDATE wp_application_passwords SET last_used = $2, last_ip = $3 WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id, time.Now(), ip)
	return err
}

// SeedMapping represents a mapping between external and local IDs for seeding.
type SeedMapping struct {
	Source     string
	EntityType string
	ExternalID string
	LocalID    string
	CreatedAt  time.Time
}

// SeedMappingsStore handles seed mapping persistence.
type SeedMappingsStore struct {
	db *sql.DB
}

// NewSeedMappingsStore creates a new seed mappings store.
func NewSeedMappingsStore(db *sql.DB) *SeedMappingsStore {
	return &SeedMappingsStore{db: db}
}

// Create creates a new seed mapping.
func (s *SeedMappingsStore) Create(ctx context.Context, m *SeedMapping) error {
	query := `INSERT INTO seed_mappings (source, entity_type, external_id, local_id, created_at)
		VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`
	_, err := s.db.ExecContext(ctx, query, m.Source, m.EntityType, m.ExternalID, m.LocalID, m.CreatedAt)
	return err
}

// Get retrieves a local ID by source, entity type, and external ID.
func (s *SeedMappingsStore) Get(ctx context.Context, source, entityType, externalID string) (string, error) {
	query := `SELECT local_id FROM seed_mappings WHERE source = $1 AND entity_type = $2 AND external_id = $3`
	var localID string
	err := s.db.QueryRowContext(ctx, query, source, entityType, externalID).Scan(&localID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return localID, err
}
