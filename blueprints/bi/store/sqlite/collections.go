package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/bi/store"
)

// CollectionStore implements store.CollectionStore.
type CollectionStore struct {
	db *sql.DB
}

func (s *CollectionStore) Create(ctx context.Context, c *store.Collection) error {
	if c.ID == "" {
		c.ID = generateID()
	}
	c.CreatedAt = time.Now()

	var parentID interface{}
	if c.ParentID != "" {
		parentID = c.ParentID
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO collections (id, name, description, parent_id, color, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.Name, c.Description, parentID, c.Color, c.CreatedBy, c.CreatedAt)
	return err
}

func (s *CollectionStore) GetByID(ctx context.Context, id string) (*store.Collection, error) {
	var c store.Collection
	var parentID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, parent_id, color, created_by, created_at
		FROM collections WHERE id = ?
	`, id).Scan(&c.ID, &c.Name, &c.Description, &parentID, &c.Color, &c.CreatedBy, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.ParentID = parentID.String
	return &c, nil
}

func (s *CollectionStore) List(ctx context.Context) ([]*store.Collection, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, parent_id, color, created_by, created_at
		FROM collections ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Collection
	for rows.Next() {
		var c store.Collection
		var parentID sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &parentID, &c.Color, &c.CreatedBy, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.ParentID = parentID.String
		result = append(result, &c)
	}
	return result, rows.Err()
}

func (s *CollectionStore) ListByParent(ctx context.Context, parentID string) ([]*store.Collection, error) {
	var rows *sql.Rows
	var err error
	if parentID == "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, name, description, parent_id, color, created_by, created_at
			FROM collections WHERE parent_id IS NULL ORDER BY name
		`)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, name, description, parent_id, color, created_by, created_at
			FROM collections WHERE parent_id = ? ORDER BY name
		`, parentID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Collection
	for rows.Next() {
		var c store.Collection
		var pID sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &pID, &c.Color, &c.CreatedBy, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.ParentID = pID.String
		result = append(result, &c)
	}
	return result, rows.Err()
}

func (s *CollectionStore) Update(ctx context.Context, c *store.Collection) error {
	var parentID interface{}
	if c.ParentID != "" {
		parentID = c.ParentID
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE collections SET name=?, description=?, parent_id=?, color=?
		WHERE id=?
	`, c.Name, c.Description, parentID, c.Color, c.ID)
	return err
}

func (s *CollectionStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM collections WHERE id=?`, id)
	return err
}

// GetRootCollection returns the system root collection ("Our analytics").
func (s *CollectionStore) GetRootCollection(ctx context.Context) (*store.Collection, error) {
	var c store.Collection
	var parentID, ownerID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, parent_id, color, COALESCE(type, ''), COALESCE(owner_id, ''), created_by, created_at
		FROM collections WHERE type = 'root' LIMIT 1
	`).Scan(&c.ID, &c.Name, &c.Description, &parentID, &c.Color, &c.Type, &ownerID, &c.CreatedBy, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.ParentID = parentID.String
	c.OwnerID = ownerID.String
	return &c, nil
}

// GetPersonalCollection returns the personal collection for a specific user.
func (s *CollectionStore) GetPersonalCollection(ctx context.Context, userID string) (*store.Collection, error) {
	var c store.Collection
	var parentID, ownerID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, parent_id, color, COALESCE(type, ''), COALESCE(owner_id, ''), created_by, created_at
		FROM collections WHERE type = 'personal' AND owner_id = ? LIMIT 1
	`, userID).Scan(&c.ID, &c.Name, &c.Description, &parentID, &c.Color, &c.Type, &ownerID, &c.CreatedBy, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.ParentID = parentID.String
	c.OwnerID = ownerID.String
	return &c, nil
}

// EnsureRootCollection creates the root collection if it doesn't exist.
func (s *CollectionStore) EnsureRootCollection(ctx context.Context) (*store.Collection, error) {
	existing, err := s.GetRootCollection(ctx)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	c := &store.Collection{
		ID:        "root",
		Name:      "Our analytics",
		Type:      store.CollectionTypeRoot,
		CreatedBy: "system",
		CreatedAt: time.Now(),
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO collections (id, name, description, parent_id, color, type, owner_id, created_by, created_at)
		VALUES (?, ?, ?, NULL, ?, ?, NULL, ?, ?)
	`, c.ID, c.Name, c.Description, c.Color, c.Type, c.CreatedBy, c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// EnsurePersonalCollection creates a personal collection for a user if it doesn't exist.
func (s *CollectionStore) EnsurePersonalCollection(ctx context.Context, userID, userName string) (*store.Collection, error) {
	existing, err := s.GetPersonalCollection(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	c := &store.Collection{
		ID:        "personal-" + userID,
		Name:      userName + "'s personal collection",
		Type:      store.CollectionTypePersonal,
		OwnerID:   userID,
		CreatedBy: userID,
		CreatedAt: time.Now(),
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO collections (id, name, description, parent_id, color, type, owner_id, created_by, created_at)
		VALUES (?, ?, ?, NULL, ?, ?, ?, ?, ?)
	`, c.ID, c.Name, c.Description, c.Color, c.Type, c.OwnerID, c.CreatedBy, c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}
