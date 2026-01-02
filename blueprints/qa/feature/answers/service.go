package answers

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/qa/feature/questions"
	"github.com/go-mizu/mizu/blueprints/qa/pkg/markdown"
	"github.com/go-mizu/mizu/blueprints/qa/pkg/ulid"
)

// Service implements the answers API.
type Service struct {
	store     Store
	accounts  accounts.API
	questions questions.API
}

// NewService creates a new answers service.
func NewService(store Store, accounts accounts.API, questions questions.API) *Service {
	return &Service{store: store, accounts: accounts, questions: questions}
}

// Create creates a new answer.
func (s *Service) Create(ctx context.Context, authorID string, in CreateIn) (*Answer, error) {
	if len(in.Body) == 0 || len(in.Body) > BodyMaxLen {
		return nil, ErrInvalid
	}

	html, _ := markdown.RenderSafe(in.Body)
	now := time.Now()
	answer := &Answer{
		ID:         ulid.New(),
		QuestionID: in.QuestionID,
		AuthorID:   authorID,
		Body:       in.Body,
		BodyHTML:   html,
		Score:      0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.store.Create(ctx, answer); err != nil {
		return nil, err
	}

	_ = s.questions.UpdateStats(ctx, in.QuestionID, 1, 0, 0)
	answer.Author, _ = s.accounts.GetByID(ctx, authorID)

	return answer, nil
}

// GetByID retrieves an answer.
func (s *Service) GetByID(ctx context.Context, id string) (*Answer, error) {
	answer, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	answer.Author, _ = s.accounts.GetByID(ctx, answer.AuthorID)
	return answer, nil
}

// ListByQuestion lists answers for a question.
func (s *Service) ListByQuestion(ctx context.Context, questionID string, opts ListOpts) ([]*Answer, error) {
	answers, err := s.store.ListByQuestion(ctx, questionID, opts)
	if err != nil {
		return nil, err
	}

	// Batch load authors to avoid N+1
	authorIDs := make([]string, 0, len(answers))
	for _, answer := range answers {
		authorIDs = append(authorIDs, answer.AuthorID)
	}
	authors, _ := s.accounts.GetByIDs(ctx, authorIDs)
	for _, answer := range answers {
		answer.Author = authors[answer.AuthorID]
	}

	return answers, nil
}

// Update updates an answer.
func (s *Service) Update(ctx context.Context, id string, in UpdateIn) (*Answer, error) {
	answer, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Body != nil {
		if len(*in.Body) == 0 || len(*in.Body) > BodyMaxLen {
			return nil, ErrInvalid
		}
		answer.Body = *in.Body
		answer.BodyHTML, _ = markdown.RenderSafe(*in.Body)
	}

	answer.UpdatedAt = time.Now()
	if err := s.store.Update(ctx, answer); err != nil {
		return nil, err
	}
	answer.Author, _ = s.accounts.GetByID(ctx, answer.AuthorID)
	return answer, nil
}

// Delete deletes an answer.
func (s *Service) Delete(ctx context.Context, id string) error {
	answer, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.questions.UpdateStats(ctx, answer.QuestionID, -1, 0, 0)
	return nil
}

// SetAccepted toggles accepted state.
func (s *Service) SetAccepted(ctx context.Context, id string, accepted bool) error {
	if err := s.store.SetAccepted(ctx, id, accepted); err != nil {
		return err
	}
	answer, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if accepted {
		return s.questions.SetAcceptedAnswer(ctx, answer.QuestionID, answer.ID)
	}
	return s.questions.SetAcceptedAnswer(ctx, answer.QuestionID, "")
}

// UpdateScore updates answer score.
func (s *Service) UpdateScore(ctx context.Context, id string, delta int64) error {
	return s.store.UpdateScore(ctx, id, delta)
}
