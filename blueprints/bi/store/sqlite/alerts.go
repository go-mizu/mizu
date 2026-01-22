package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/bi/store"
)

// AlertStore implements store.AlertStore.
type AlertStore struct {
	db *sql.DB
}

func (s *AlertStore) Create(ctx context.Context, a *store.Alert) error {
	if a.ID == "" {
		a.ID = generateID()
	}
	a.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO alerts (id, name, question_id, alert_type, condition, channels, enabled, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, a.ID, a.Name, a.QuestionID, a.AlertType, toJSON(a.Condition), toJSON(a.Channels), a.Enabled, a.CreatedBy, a.CreatedAt)
	return err
}

func (s *AlertStore) GetByID(ctx context.Context, id string) (*store.Alert, error) {
	var a store.Alert
	var cond, channels string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, question_id, alert_type, condition, channels, enabled, created_by, created_at
		FROM alerts WHERE id = ?
	`, id).Scan(&a.ID, &a.Name, &a.QuestionID, &a.AlertType, &cond, &channels, &a.Enabled, &a.CreatedBy, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	fromJSON(cond, &a.Condition)
	fromJSON(channels, &a.Channels)
	return &a, nil
}

func (s *AlertStore) List(ctx context.Context) ([]*store.Alert, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, question_id, alert_type, condition, channels, enabled, created_by, created_at
		FROM alerts ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Alert
	for rows.Next() {
		var a store.Alert
		var cond, channels string
		if err := rows.Scan(&a.ID, &a.Name, &a.QuestionID, &a.AlertType, &cond, &channels, &a.Enabled, &a.CreatedBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		fromJSON(cond, &a.Condition)
		fromJSON(channels, &a.Channels)
		result = append(result, &a)
	}
	return result, rows.Err()
}

func (s *AlertStore) ListByQuestion(ctx context.Context, questionID string) ([]*store.Alert, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, question_id, alert_type, condition, channels, enabled, created_by, created_at
		FROM alerts WHERE question_id = ? ORDER BY name
	`, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Alert
	for rows.Next() {
		var a store.Alert
		var cond, channels string
		if err := rows.Scan(&a.ID, &a.Name, &a.QuestionID, &a.AlertType, &cond, &channels, &a.Enabled, &a.CreatedBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		fromJSON(cond, &a.Condition)
		fromJSON(channels, &a.Channels)
		result = append(result, &a)
	}
	return result, rows.Err()
}

func (s *AlertStore) Update(ctx context.Context, a *store.Alert) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE alerts SET name=?, alert_type=?, condition=?, channels=?, enabled=?
		WHERE id=?
	`, a.Name, a.AlertType, toJSON(a.Condition), toJSON(a.Channels), a.Enabled, a.ID)
	return err
}

func (s *AlertStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM alerts WHERE id=?`, id)
	return err
}
