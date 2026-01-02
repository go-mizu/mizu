package comments

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/qa/feature/answers"
	"github.com/go-mizu/mizu/blueprints/qa/feature/questions"
	"github.com/go-mizu/mizu/blueprints/qa/pkg/ulid"
)

// Service implements the comments API.
type Service struct {
	store     Store
	accounts  accounts.API
	questions questions.API
	answers   answers.API
}

// NewService creates a new comments service.
func NewService(store Store, accounts accounts.API, questions questions.API, answers answers.API) *Service {
	return &Service{store: store, accounts: accounts, questions: questions, answers: answers}
}

// Create creates a new comment.
func (s *Service) Create(ctx context.Context, authorID string, in CreateIn) (*Comment, error) {
	if len(in.Body) == 0 || len(in.Body) > BodyMaxLen {
		return nil, ErrInvalid
	}

	comment := &Comment{
		ID:         ulid.New(),
		TargetType: in.TargetType,
		TargetID:   in.TargetID,
		AuthorID:   authorID,
		Body:       in.Body,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.store.Create(ctx, comment); err != nil {
		return nil, err
	}

	s.updateQuestionStats(ctx, in.TargetType, in.TargetID, 1)
	comment.Author, _ = s.accounts.GetByID(ctx, authorID)

	return comment, nil
}

// ListByTarget lists comments.
func (s *Service) ListByTarget(ctx context.Context, targetType TargetType, targetID string, opts ListOpts) ([]*Comment, error) {
	comments, err := s.store.ListByTarget(ctx, targetType, targetID, opts)
	if err != nil {
		return nil, err
	}

	// Batch load authors to avoid N+1
	authorIDs := make([]string, 0, len(comments))
	for _, comment := range comments {
		authorIDs = append(authorIDs, comment.AuthorID)
	}
	authors, _ := s.accounts.GetByIDs(ctx, authorIDs)
	for _, comment := range comments {
		comment.Author = authors[comment.AuthorID]
	}

	return comments, nil
}

// ListByTargets lists comments for multiple targets.
func (s *Service) ListByTargets(ctx context.Context, targetType TargetType, targetIDs []string, opts ListOpts) (map[string][]*Comment, error) {
	commentsMap, err := s.store.ListByTargets(ctx, targetType, targetIDs, opts)
	if err != nil {
		return nil, err
	}

	// Collect all unique author IDs
	authorIDs := make([]string, 0)
	seen := make(map[string]bool)
	for _, comments := range commentsMap {
		for _, comment := range comments {
			if !seen[comment.AuthorID] {
				seen[comment.AuthorID] = true
				authorIDs = append(authorIDs, comment.AuthorID)
			}
		}
	}

	// Batch load authors
	authors, _ := s.accounts.GetByIDs(ctx, authorIDs)

	// Assign authors to comments
	for _, comments := range commentsMap {
		for _, comment := range comments {
			comment.Author = authors[comment.AuthorID]
		}
	}

	return commentsMap, nil
}

// Delete deletes a comment.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// UpdateScore updates comment score.
func (s *Service) UpdateScore(ctx context.Context, id string, delta int64) error {
	return s.store.UpdateScore(ctx, id, delta)
}

func (s *Service) updateQuestionStats(ctx context.Context, targetType TargetType, targetID string, delta int64) {
	switch targetType {
	case TargetQuestion:
		_ = s.questions.UpdateStats(ctx, targetID, 0, delta, 0)
	case TargetAnswer:
		answer, err := s.answers.GetByID(ctx, targetID)
		if err == nil {
			_ = s.questions.UpdateStats(ctx, answer.QuestionID, 0, delta, 0)
		}
	}
}
