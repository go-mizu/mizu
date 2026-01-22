package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/bi/store"
)

// DashboardStore implements store.DashboardStore.
type DashboardStore struct {
	db *sql.DB
}

func (s *DashboardStore) Create(ctx context.Context, d *store.Dashboard) error {
	if d.ID == "" {
		d.ID = generateID()
	}
	now := time.Now()
	d.CreatedAt = now
	d.UpdatedAt = now

	var collID interface{}
	if d.CollectionID != "" {
		collID = d.CollectionID
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dashboards (id, name, description, collection_id, auto_refresh, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, d.ID, d.Name, d.Description, collID, d.AutoRefresh, d.CreatedBy, d.CreatedAt, d.UpdatedAt)
	return err
}

func (s *DashboardStore) GetByID(ctx context.Context, id string) (*store.Dashboard, error) {
	var d store.Dashboard
	var collID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, collection_id, auto_refresh, created_by, created_at, updated_at
		FROM dashboards WHERE id = ?
	`, id).Scan(&d.ID, &d.Name, &d.Description, &collID, &d.AutoRefresh, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d.CollectionID = collID.String
	return &d, nil
}

func (s *DashboardStore) List(ctx context.Context) ([]*store.Dashboard, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, collection_id, auto_refresh, created_by, created_at, updated_at
		FROM dashboards ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Dashboard
	for rows.Next() {
		var d store.Dashboard
		var collID sql.NullString
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &collID, &d.AutoRefresh, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		d.CollectionID = collID.String
		result = append(result, &d)
	}
	return result, rows.Err()
}

func (s *DashboardStore) ListByCollection(ctx context.Context, collectionID string) ([]*store.Dashboard, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, collection_id, auto_refresh, created_by, created_at, updated_at
		FROM dashboards WHERE collection_id = ? ORDER BY name
	`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Dashboard
	for rows.Next() {
		var d store.Dashboard
		var collID sql.NullString
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &collID, &d.AutoRefresh, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		d.CollectionID = collID.String
		result = append(result, &d)
	}
	return result, rows.Err()
}

func (s *DashboardStore) Update(ctx context.Context, d *store.Dashboard) error {
	d.UpdatedAt = time.Now()

	var collID interface{}
	if d.CollectionID != "" {
		collID = d.CollectionID
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE dashboards SET name=?, description=?, collection_id=?, auto_refresh=?, updated_at=?
		WHERE id=?
	`, d.Name, d.Description, collID, d.AutoRefresh, d.UpdatedAt, d.ID)
	return err
}

func (s *DashboardStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dashboards WHERE id=?`, id)
	return err
}

func (s *DashboardStore) CreateCard(ctx context.Context, card *store.DashboardCard) error {
	if card.ID == "" {
		card.ID = generateID()
	}

	var questionID interface{}
	if card.QuestionID != "" {
		questionID = card.QuestionID
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dashboard_cards (id, dashboard_id, question_id, card_type, tab_id, row_num, col_num, width, height, settings)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, card.ID, card.DashboardID, questionID, card.CardType, card.TabID, card.Row, card.Col, card.Width, card.Height, toJSON(card.Settings))
	return err
}

func (s *DashboardStore) GetCard(ctx context.Context, id string) (*store.DashboardCard, error) {
	var card store.DashboardCard
	var settings string
	var qID, tabID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, dashboard_id, question_id, card_type, tab_id, row_num, col_num, width, height, settings
		FROM dashboard_cards WHERE id = ?
	`, id).Scan(&card.ID, &card.DashboardID, &qID, &card.CardType, &tabID, &card.Row, &card.Col, &card.Width, &card.Height, &settings)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	card.QuestionID = qID.String
	card.TabID = tabID.String
	fromJSON(settings, &card.Settings)
	return &card, nil
}

func (s *DashboardStore) ListCards(ctx context.Context, dashboardID string) ([]*store.DashboardCard, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, dashboard_id, question_id, card_type, tab_id, row_num, col_num, width, height, settings
		FROM dashboard_cards WHERE dashboard_id = ? ORDER BY row_num, col_num
	`, dashboardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.DashboardCard
	for rows.Next() {
		var card store.DashboardCard
		var settings string
		var qID, tabID sql.NullString
		if err := rows.Scan(&card.ID, &card.DashboardID, &qID, &card.CardType, &tabID, &card.Row, &card.Col, &card.Width, &card.Height, &settings); err != nil {
			return nil, err
		}
		card.QuestionID = qID.String
		card.TabID = tabID.String
		fromJSON(settings, &card.Settings)
		result = append(result, &card)
	}
	return result, rows.Err()
}

func (s *DashboardStore) UpdateCard(ctx context.Context, card *store.DashboardCard) error {
	var questionID, tabID interface{}
	if card.QuestionID != "" {
		questionID = card.QuestionID
	}
	if card.TabID != "" {
		tabID = card.TabID
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE dashboard_cards SET question_id=?, card_type=?, tab_id=?, row_num=?, col_num=?, width=?, height=?, settings=?
		WHERE id=?
	`, questionID, card.CardType, tabID, card.Row, card.Col, card.Width, card.Height, toJSON(card.Settings), card.ID)
	return err
}

func (s *DashboardStore) DeleteCard(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dashboard_cards WHERE id=?`, id)
	return err
}

func (s *DashboardStore) CreateFilter(ctx context.Context, filter *store.DashboardFilter) error {
	if filter.ID == "" {
		filter.ID = generateID()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dashboard_filters (id, dashboard_id, name, type, default_value, required, targets)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, filter.ID, filter.DashboardID, filter.Name, filter.Type, filter.Default, filter.Required, toJSON(filter.Targets))
	return err
}

func (s *DashboardStore) ListFilters(ctx context.Context, dashboardID string) ([]*store.DashboardFilter, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, dashboard_id, name, type, default_value, required, targets
		FROM dashboard_filters WHERE dashboard_id = ?
	`, dashboardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.DashboardFilter
	for rows.Next() {
		var f store.DashboardFilter
		var targets string
		if err := rows.Scan(&f.ID, &f.DashboardID, &f.Name, &f.Type, &f.Default, &f.Required, &targets); err != nil {
			return nil, err
		}
		fromJSON(targets, &f.Targets)
		result = append(result, &f)
	}
	return result, rows.Err()
}

func (s *DashboardStore) DeleteFilter(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dashboard_filters WHERE id=?`, id)
	return err
}

func (s *DashboardStore) CreateTab(ctx context.Context, tab *store.DashboardTab) error {
	if tab.ID == "" {
		tab.ID = generateID()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dashboard_tabs (id, dashboard_id, name, position)
		VALUES (?, ?, ?, ?)
	`, tab.ID, tab.DashboardID, tab.Name, tab.Position)
	return err
}

func (s *DashboardStore) ListTabs(ctx context.Context, dashboardID string) ([]*store.DashboardTab, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, dashboard_id, name, position
		FROM dashboard_tabs WHERE dashboard_id = ? ORDER BY position
	`, dashboardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.DashboardTab
	for rows.Next() {
		var t store.DashboardTab
		if err := rows.Scan(&t.ID, &t.DashboardID, &t.Name, &t.Position); err != nil {
			return nil, err
		}
		result = append(result, &t)
	}
	return result, rows.Err()
}

func (s *DashboardStore) DeleteTab(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dashboard_tabs WHERE id=?`, id)
	return err
}
