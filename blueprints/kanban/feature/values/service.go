package values

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("value not found")
)

// Service implements the values API.
type Service struct {
	store Store
}

// NewService creates a new values service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Set(ctx context.Context, issueID, fieldID string, in *SetIn) (*Value, error) {
	value := &Value{
		IssueID:   issueID,
		FieldID:   fieldID,
		ValueText: in.ValueText,
		ValueNum:  in.ValueNum,
		ValueBool: in.ValueBool,
		ValueDate: in.ValueDate,
		ValueTS:   in.ValueTS,
		ValueRef:  in.ValueRef,
		ValueJSON: in.ValueJSON,
		UpdatedAt: time.Now(),
	}

	if err := s.store.Set(ctx, value); err != nil {
		return nil, err
	}

	return value, nil
}

func (s *Service) Get(ctx context.Context, issueID, fieldID string) (*Value, error) {
	v, err := s.store.Get(ctx, issueID, fieldID)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, ErrNotFound
	}
	return v, nil
}

func (s *Service) ListByIssue(ctx context.Context, issueID string) ([]*Value, error) {
	return s.store.ListByIssue(ctx, issueID)
}

func (s *Service) ListByField(ctx context.Context, fieldID string) ([]*Value, error) {
	return s.store.ListByField(ctx, fieldID)
}

func (s *Service) Delete(ctx context.Context, issueID, fieldID string) error {
	return s.store.Delete(ctx, issueID, fieldID)
}

func (s *Service) DeleteByIssue(ctx context.Context, issueID string) error {
	return s.store.DeleteByIssue(ctx, issueID)
}

func (s *Service) BulkSet(ctx context.Context, vs []*Value) error {
	now := time.Now()
	for _, v := range vs {
		v.UpdatedAt = now
	}
	return s.store.BulkSet(ctx, vs)
}

func (s *Service) BulkGetByIssues(ctx context.Context, issueIDs []string) (map[string][]*Value, error) {
	return s.store.BulkGetByIssues(ctx, issueIDs)
}
