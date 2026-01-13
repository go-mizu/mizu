package cron

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"
)

// Service implements the Cron API.
type Service struct {
	store Store
}

// NewService creates a new Cron service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new trigger.
func (s *Service) Create(ctx context.Context, in *CreateTriggerIn) (*Trigger, error) {
	if in.ScriptName == "" {
		return nil, ErrScriptRequired
	}
	if in.Cron == "" {
		return nil, ErrCronRequired
	}

	now := time.Now()
	trigger := &Trigger{
		ID:         ulid.Make().String(),
		ScriptName: in.ScriptName,
		Cron:       in.Cron,
		Enabled:    in.Enabled,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.store.CreateTrigger(ctx, trigger); err != nil {
		return nil, err
	}

	return trigger, nil
}

// Get retrieves a trigger by ID.
func (s *Service) Get(ctx context.Context, id string) (*Trigger, error) {
	trigger, err := s.store.GetTrigger(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return trigger, nil
}

// List lists all triggers.
func (s *Service) List(ctx context.Context) ([]*Trigger, error) {
	return s.store.ListTriggers(ctx)
}

// ListByScript lists triggers for a script.
func (s *Service) ListByScript(ctx context.Context, scriptName string) ([]*Trigger, error) {
	return s.store.ListTriggersByScript(ctx, scriptName)
}

// Update updates a trigger.
func (s *Service) Update(ctx context.Context, id string, in *UpdateTriggerIn) (*Trigger, error) {
	trigger, err := s.store.GetTrigger(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	if in.Cron != nil {
		trigger.Cron = *in.Cron
	}
	if in.Enabled != nil {
		trigger.Enabled = *in.Enabled
	}
	trigger.UpdatedAt = time.Now()

	if err := s.store.UpdateTrigger(ctx, trigger); err != nil {
		return nil, err
	}

	return trigger, nil
}

// Delete deletes a trigger.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.DeleteTrigger(ctx, id)
}

// GetExecutions gets recent executions for a trigger.
func (s *Service) GetExecutions(ctx context.Context, triggerID string, limit int) ([]*Execution, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	return s.store.GetRecentExecutions(ctx, triggerID, limit)
}
