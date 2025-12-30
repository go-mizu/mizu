package settings

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/cms/pkg/ulid"
)

var (
	ErrNotFound   = errors.New("setting not found")
	ErrMissingKey = errors.New("key is required")
)

// Service implements the settings API.
type Service struct {
	store Store
}

// NewService creates a new settings service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Get(ctx context.Context, key string) (*Setting, error) {
	setting, err := s.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		return nil, ErrNotFound
	}
	return setting, nil
}

func (s *Service) GetByGroup(ctx context.Context, group string) ([]*Setting, error) {
	return s.store.GetByGroup(ctx, group)
}

func (s *Service) GetAll(ctx context.Context) ([]*Setting, error) {
	return s.store.GetAll(ctx)
}

func (s *Service) GetPublic(ctx context.Context) ([]*Setting, error) {
	return s.store.GetPublic(ctx)
}

func (s *Service) Set(ctx context.Context, in *SetIn) (*Setting, error) {
	if in.Key == "" {
		return nil, ErrMissingKey
	}

	// Check if setting exists
	existing, _ := s.store.Get(ctx, in.Key)

	now := time.Now()
	setting := &Setting{
		Key:         in.Key,
		Value:       in.Value,
		ValueType:   in.ValueType,
		GroupName:   in.GroupName,
		Description: in.Description,
		UpdatedAt:   now,
	}

	if in.IsPublic != nil {
		setting.IsPublic = *in.IsPublic
	}

	if existing != nil {
		setting.ID = existing.ID
		setting.CreatedAt = existing.CreatedAt
	} else {
		setting.ID = ulid.New()
		setting.CreatedAt = now
	}

	if setting.ValueType == "" {
		setting.ValueType = "string"
	}

	if err := s.store.Set(ctx, setting); err != nil {
		return nil, err
	}

	return setting, nil
}

func (s *Service) SetBulk(ctx context.Context, settings []*SetIn) error {
	for _, in := range settings {
		if _, err := s.Set(ctx, in); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, key string) error {
	return s.store.Delete(ctx, key)
}
