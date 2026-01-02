package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/mizu/blueprints/qa/feature/answers"
)

// AnswersStore implements answers.Store.
type AnswersStore struct {
	db *sql.DB
}

// NewAnswersStore creates a new answers store.
func NewAnswersStore(db *sql.DB) *AnswersStore {
	return &AnswersStore{db: db}
}

// Create creates an answer.
func (s *AnswersStore) Create(ctx context.Context, answer *answers.Answer) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO answers (
			id, question_id, author_id, body, body_html, score,
			is_accepted, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, answer.ID, answer.QuestionID, answer.AuthorID, answer.Body,
		answer.BodyHTML, answer.Score, answer.IsAccepted, answer.CreatedAt, answer.UpdatedAt)
	return err
}

// GetByID retrieves an answer by ID.
func (s *AnswersStore) GetByID(ctx context.Context, id string) (*answers.Answer, error) {
	answer := &answers.Answer{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, question_id, author_id, body, body_html, score,
			is_accepted, created_at, updated_at
		FROM answers WHERE id = $1
	`, id).Scan(
		&answer.ID, &answer.QuestionID, &answer.AuthorID, &answer.Body,
		&answer.BodyHTML, &answer.Score, &answer.IsAccepted, &answer.CreatedAt, &answer.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, answers.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return answer, nil
}

// ListByQuestion lists answers for a question.
func (s *AnswersStore) ListByQuestion(ctx context.Context, questionID string, opts answers.ListOpts) ([]*answers.Answer, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, question_id, author_id, body, body_html, score,
			is_accepted, created_at, updated_at
		FROM answers
		WHERE question_id = $1
		ORDER BY is_accepted DESC, score DESC, created_at ASC
		LIMIT $2
	`, questionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*answers.Answer
	for rows.Next() {
		answer := &answers.Answer{}
		if err := rows.Scan(
			&answer.ID, &answer.QuestionID, &answer.AuthorID, &answer.Body,
			&answer.BodyHTML, &answer.Score, &answer.IsAccepted, &answer.CreatedAt, &answer.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, answer)
	}
	return result, rows.Err()
}

// Update updates an answer.
func (s *AnswersStore) Update(ctx context.Context, answer *answers.Answer) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE answers SET body = $2, body_html = $3, score = $4,
			is_accepted = $5, updated_at = $6
		WHERE id = $1
	`, answer.ID, answer.Body, answer.BodyHTML, answer.Score, answer.IsAccepted, answer.UpdatedAt)
	return err
}

// Delete deletes an answer.
func (s *AnswersStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM answers WHERE id = $1`, id)
	return err
}

// SetAccepted updates accepted flag.
func (s *AnswersStore) SetAccepted(ctx context.Context, id string, accepted bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE answers SET is_accepted = $2 WHERE id = $1`, id, accepted)
	return err
}

// UpdateScore updates score.
func (s *AnswersStore) UpdateScore(ctx context.Context, id string, delta int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE answers SET score = score + $2 WHERE id = $1`, id, delta)
	return err
}
