package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/blueprints/qa/feature/questions"
	"github.com/go-mizu/mizu/blueprints/qa/feature/tags"
)

// QuestionsStore implements questions.Store.
type QuestionsStore struct {
	db *sql.DB
}

// NewQuestionsStore creates a new questions store.
func NewQuestionsStore(db *sql.DB) *QuestionsStore {
	return &QuestionsStore{db: db}
}

// Create creates a question.
func (s *QuestionsStore) Create(ctx context.Context, question *questions.Question) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO questions (
			id, author_id, title, body, body_html, score, view_count,
			answer_count, comment_count, favorite_count, accepted_answer_id,
			bounty_amount, is_closed, close_reason, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, question.ID, question.AuthorID, question.Title, question.Body, question.BodyHTML,
		question.Score, question.ViewCount, question.AnswerCount, question.CommentCount,
		question.FavoriteCount, question.AcceptedAnswerID, question.BountyAmount,
		question.IsClosed, question.CloseReason, question.CreatedAt, question.UpdatedAt)
	return err
}

// GetByID retrieves a question by ID.
func (s *QuestionsStore) GetByID(ctx context.Context, id string) (*questions.Question, error) {
	return s.scanQuestion(s.db.QueryRowContext(ctx, `
		SELECT id, author_id, title, body, body_html, score, view_count,
			answer_count, comment_count, favorite_count, accepted_answer_id,
			bounty_amount, is_closed, close_reason, created_at, updated_at
		FROM questions WHERE id = $1
	`, id))
}

// GetByIDs retrieves questions by IDs.
func (s *QuestionsStore) GetByIDs(ctx context.Context, ids []string) (map[string]*questions.Question, error) {
	if len(ids) == 0 {
		return make(map[string]*questions.Question), nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := `
		SELECT id, author_id, title, body, body_html, score, view_count,
			answer_count, comment_count, favorite_count, accepted_answer_id,
			bounty_amount, is_closed, close_reason, created_at, updated_at
		FROM questions WHERE id IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*questions.Question)
	for rows.Next() {
		question, err := s.scanQuestionFromRows(rows)
		if err != nil {
			return nil, err
		}
		result[question.ID] = question
	}
	return result, rows.Err()
}

// Update updates a question.
func (s *QuestionsStore) Update(ctx context.Context, question *questions.Question) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE questions SET
			title = $2, body = $3, body_html = $4, score = $5, view_count = $6,
			answer_count = $7, comment_count = $8, favorite_count = $9,
			accepted_answer_id = $10, bounty_amount = $11, is_closed = $12,
			close_reason = $13, updated_at = $14
		WHERE id = $1
	`, question.ID, question.Title, question.Body, question.BodyHTML,
		question.Score, question.ViewCount, question.AnswerCount,
		question.CommentCount, question.FavoriteCount, question.AcceptedAnswerID,
		question.BountyAmount, question.IsClosed, question.CloseReason, question.UpdatedAt)
	return err
}

// Delete deletes a question.
func (s *QuestionsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM questions WHERE id = $1`, id)
	return err
}

// IncrementViews increments view count.
func (s *QuestionsStore) IncrementViews(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE questions SET view_count = view_count + 1 WHERE id = $1`, id)
	return err
}

// List lists questions.
func (s *QuestionsStore) List(ctx context.Context, opts questions.ListOpts) ([]*questions.Question, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 30
	}

	orderBy := "created_at DESC"
	switch opts.SortBy {
	case questions.SortScore:
		orderBy = "score DESC"
	case questions.SortActive:
		orderBy = "updated_at DESC"
	case questions.SortUnanswered:
		orderBy = "answer_count ASC, created_at DESC"
	}

	query := `
		SELECT id, author_id, title, body, body_html, score, view_count,
			answer_count, comment_count, favorite_count, accepted_answer_id,
			bounty_amount, is_closed, close_reason, created_at, updated_at
		FROM questions
		ORDER BY ` + orderBy + `
		LIMIT $1
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*questions.Question
	for rows.Next() {
		question, err := s.scanQuestionFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, question)
	}
	return result, rows.Err()
}

// ListByTag lists questions by tag.
func (s *QuestionsStore) ListByTag(ctx context.Context, tag string, opts questions.ListOpts) ([]*questions.Question, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 30
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT q.id, q.author_id, q.title, q.body, q.body_html, q.score, q.view_count,
			q.answer_count, q.comment_count, q.favorite_count, q.accepted_answer_id,
			q.bounty_amount, q.is_closed, q.close_reason, q.created_at, q.updated_at
		FROM questions q
		JOIN question_tags qt ON qt.question_id = q.id
		JOIN tags t ON t.id = qt.tag_id
		WHERE t.name = $1
		ORDER BY q.created_at DESC
		LIMIT $2
	`, tag, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*questions.Question
	for rows.Next() {
		question, err := s.scanQuestionFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, question)
	}
	return result, rows.Err()
}

// ListByAuthor lists questions by author.
func (s *QuestionsStore) ListByAuthor(ctx context.Context, authorID string, opts questions.ListOpts) ([]*questions.Question, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 30
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, author_id, title, body, body_html, score, view_count,
			answer_count, comment_count, favorite_count, accepted_answer_id,
			bounty_amount, is_closed, close_reason, created_at, updated_at
		FROM questions
		WHERE author_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, authorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*questions.Question
	for rows.Next() {
		question, err := s.scanQuestionFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, question)
	}
	return result, rows.Err()
}

// Search searches questions.
func (s *QuestionsStore) Search(ctx context.Context, query string, limit int) ([]*questions.Question, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, author_id, title, body, body_html, score, view_count,
			answer_count, comment_count, favorite_count, accepted_answer_id,
			bounty_amount, is_closed, close_reason, created_at, updated_at
		FROM questions
		WHERE LOWER(title) LIKE LOWER($1) OR LOWER(body) LIKE LOWER($1)
		ORDER BY score DESC
		LIMIT $2
	`, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*questions.Question
	for rows.Next() {
		question, err := s.scanQuestionFromRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, question)
	}
	return result, rows.Err()
}

// SetAcceptedAnswer sets accepted answer.
func (s *QuestionsStore) SetAcceptedAnswer(ctx context.Context, id string, answerID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE questions SET accepted_answer_id = $2 WHERE id = $1`, id, answerID)
	return err
}

// SetClosed closes or reopens a question.
func (s *QuestionsStore) SetClosed(ctx context.Context, id string, closed bool, reason string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE questions SET is_closed = $2, close_reason = $3 WHERE id = $1`, id, closed, reason)
	return err
}

// UpdateStats updates counts.
func (s *QuestionsStore) UpdateStats(ctx context.Context, id string, answerDelta, commentDelta, favoriteDelta int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE questions SET
			answer_count = answer_count + $2,
			comment_count = comment_count + $3,
			favorite_count = favorite_count + $4
		WHERE id = $1
	`, id, answerDelta, commentDelta, favoriteDelta)
	return err
}

// UpdateScore updates score.
func (s *QuestionsStore) UpdateScore(ctx context.Context, id string, delta int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE questions SET score = score + $2 WHERE id = $1`, id, delta)
	return err
}

// SetTags sets tags for a question.
func (s *QuestionsStore) SetTags(ctx context.Context, questionID string, tagNames []string) error {
	_, _ = s.db.ExecContext(ctx, `DELETE FROM question_tags WHERE question_id = $1`, questionID)
	if len(tagNames) == 0 {
		return nil
	}

	// Batch fetch all tag IDs in one query
	placeholders := make([]string, len(tagNames))
	args := make([]any, len(tagNames))
	for i, name := range tagNames {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = name
	}

	query := `SELECT id, name FROM tags WHERE name IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	tagIDs := make(map[string]string)
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		tagIDs[name] = id
	}

	// Batch insert all question_tags
	if len(tagIDs) == 0 {
		return nil
	}

	insertPlaceholders := make([]string, 0, len(tagIDs))
	insertArgs := make([]any, 0, len(tagIDs)*2)
	i := 0
	for _, tagID := range tagIDs {
		insertPlaceholders = append(insertPlaceholders, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		insertArgs = append(insertArgs, questionID, tagID)
		i++
	}

	insertQuery := `INSERT INTO question_tags (question_id, tag_id) VALUES ` + strings.Join(insertPlaceholders, ", ")
	_, err = s.db.ExecContext(ctx, insertQuery, insertArgs...)
	return err
}

// GetTags retrieves tags for a question.
func (s *QuestionsStore) GetTags(ctx context.Context, questionID string) ([]*tags.Tag, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.excerpt, t.wiki, t.question_count, t.created_at
		FROM tags t
		JOIN question_tags qt ON qt.tag_id = t.id
		WHERE qt.question_id = $1
		ORDER BY t.name ASC
	`, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*tags.Tag
	for rows.Next() {
		tag := &tags.Tag{}
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Excerpt, &tag.Wiki, &tag.QuestionCount, &tag.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, tag)
	}
	return result, rows.Err()
}

// GetTagsForQuestions retrieves tags for multiple questions.
func (s *QuestionsStore) GetTagsForQuestions(ctx context.Context, questionIDs []string) (map[string][]*tags.Tag, error) {
	if len(questionIDs) == 0 {
		return make(map[string][]*tags.Tag), nil
	}

	placeholders := make([]string, len(questionIDs))
	args := make([]any, len(questionIDs))
	for i, id := range questionIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := `
		SELECT qt.question_id, t.id, t.name, t.excerpt, t.wiki, t.question_count, t.created_at
		FROM tags t
		JOIN question_tags qt ON qt.tag_id = t.id
		WHERE qt.question_id IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY t.name ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]*tags.Tag)
	for rows.Next() {
		var questionID string
		tag := &tags.Tag{}
		if err := rows.Scan(&questionID, &tag.ID, &tag.Name, &tag.Excerpt, &tag.Wiki, &tag.QuestionCount, &tag.CreatedAt); err != nil {
			return nil, err
		}
		result[questionID] = append(result[questionID], tag)
	}
	return result, rows.Err()
}

func (s *QuestionsStore) scanQuestion(row *sql.Row) (*questions.Question, error) {
	question := &questions.Question{}
	if err := row.Scan(
		&question.ID,
		&question.AuthorID,
		&question.Title,
		&question.Body,
		&question.BodyHTML,
		&question.Score,
		&question.ViewCount,
		&question.AnswerCount,
		&question.CommentCount,
		&question.FavoriteCount,
		&question.AcceptedAnswerID,
		&question.BountyAmount,
		&question.IsClosed,
		&question.CloseReason,
		&question.CreatedAt,
		&question.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, questions.ErrNotFound
		}
		return nil, err
	}
	return question, nil
}

func (s *QuestionsStore) scanQuestionFromRows(rows *sql.Rows) (*questions.Question, error) {
	question := &questions.Question{}
	if err := rows.Scan(
		&question.ID,
		&question.AuthorID,
		&question.Title,
		&question.Body,
		&question.BodyHTML,
		&question.Score,
		&question.ViewCount,
		&question.AnswerCount,
		&question.CommentCount,
		&question.FavoriteCount,
		&question.AcceptedAnswerID,
		&question.BountyAmount,
		&question.IsClosed,
		&question.CloseReason,
		&question.CreatedAt,
		&question.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return question, nil
}
