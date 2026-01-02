package votes

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/qa/feature/answers"
	"github.com/go-mizu/mizu/blueprints/qa/feature/comments"
	"github.com/go-mizu/mizu/blueprints/qa/feature/questions"
	"github.com/go-mizu/mizu/blueprints/qa/pkg/ulid"
)

// Service implements the votes API.
type Service struct {
	store     Store
	accounts  accounts.API
	questions questions.API
	answers   answers.API
	comments  comments.API
}

// NewService creates a new votes service.
func NewService(store Store, accounts accounts.API, questions questions.API, answers answers.API, comments comments.API) *Service {
	return &Service{store: store, accounts: accounts, questions: questions, answers: answers, comments: comments}
}

// Cast casts a vote.
func (s *Service) Cast(ctx context.Context, voterID string, targetType TargetType, targetID string, value int) (*Vote, error) {
	if value != 1 && value != -1 {
		return nil, ErrInvalidVote
	}

	prev, _ := s.store.Get(ctx, voterID, targetType, targetID)
	prevValue := 0
	if prev != nil {
		prevValue = prev.Value
	}
	delta := value - prevValue

	vote := &Vote{
		ID:         ulid.New(),
		VoterID:    voterID,
		TargetType: targetType,
		TargetID:   targetID,
		Value:      value,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	stored, err := s.store.Upsert(ctx, vote)
	if err != nil {
		return nil, err
	}

	s.applyScoreDelta(ctx, targetType, targetID, int64(delta))

	return stored, nil
}

// Get retrieves a vote.
func (s *Service) Get(ctx context.Context, voterID string, targetType TargetType, targetID string) (*Vote, error) {
	return s.store.Get(ctx, voterID, targetType, targetID)
}

func (s *Service) applyScoreDelta(ctx context.Context, targetType TargetType, targetID string, delta int64) {
	switch targetType {
	case TargetQuestion:
		_ = s.questions.UpdateScore(ctx, targetID, delta)
		if delta != 0 {
			if question, err := s.questions.GetByID(ctx, targetID); err == nil {
				rep := int64(5)
				if delta < 0 {
					rep = -2
				}
				_ = s.accounts.UpdateReputation(ctx, question.AuthorID, rep)
			}
		}
	case TargetAnswer:
		_ = s.answers.UpdateScore(ctx, targetID, delta)
		if delta != 0 {
			if answer, err := s.answers.GetByID(ctx, targetID); err == nil {
				rep := int64(10)
				if delta < 0 {
					rep = -2
				}
				_ = s.accounts.UpdateReputation(ctx, answer.AuthorID, rep)
			}
		}
	case TargetComment:
		_ = s.comments.UpdateScore(ctx, targetID, delta)
	}
}
