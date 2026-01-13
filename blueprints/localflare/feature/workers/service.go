package workers

import (
	"context"
	"errors"
	"time"

	"github.com/oklog/ulid/v2"
)

// Service implements the workers API.
type Service struct {
	store Store
}

// NewService creates a new workers service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new worker.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Worker, error) {
	if in.Name == "" {
		return nil, errors.New("name is required")
	}

	// Check for duplicate name
	if existing, _ := s.store.GetByName(ctx, in.Name); existing != nil {
		return nil, ErrNameExists
	}

	bindings := in.Bindings
	if bindings == nil {
		bindings = make(map[string]string)
	}

	now := time.Now()
	worker := &Worker{
		ID:        ulid.Make().String(),
		Name:      in.Name,
		Script:    in.Script,
		Routes:    in.Routes,
		Bindings:  bindings,
		Enabled:   in.Enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Create(ctx, worker); err != nil {
		return nil, err
	}

	return worker, nil
}

// GetByID retrieves a worker by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Worker, error) {
	worker, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return worker, nil
}

// List lists all workers.
func (s *Service) List(ctx context.Context) ([]*Worker, error) {
	return s.store.List(ctx)
}

// Update updates a worker.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Worker, error) {
	worker, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	if in.Name != nil && *in.Name != "" {
		worker.Name = *in.Name
	}
	if in.Script != nil {
		worker.Script = *in.Script
	}
	if in.Routes != nil {
		worker.Routes = in.Routes
	}
	if in.Bindings != nil {
		worker.Bindings = in.Bindings
	}
	if in.Enabled != nil {
		worker.Enabled = *in.Enabled
	}
	worker.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, worker); err != nil {
		return nil, err
	}

	return worker, nil
}

// Delete deletes a worker.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// Deploy deploys a worker.
func (s *Service) Deploy(ctx context.Context, id string) (*DeployResult, error) {
	worker, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	// In a real implementation, this would compile and deploy the worker
	worker.UpdatedAt = time.Now()
	if err := s.store.Update(ctx, worker); err != nil {
		return nil, err
	}

	return &DeployResult{
		ID:         worker.ID,
		Name:       worker.Name,
		DeployedAt: worker.UpdatedAt,
	}, nil
}

// Logs returns worker logs.
func (s *Service) Logs(ctx context.Context, id string) ([]*LogEntry, error) {
	// Verify worker exists
	if _, err := s.store.GetByID(ctx, id); err != nil {
		return nil, ErrNotFound
	}

	// In a real implementation, this would return actual logs
	now := time.Now()
	return []*LogEntry{
		{Timestamp: now.Add(-5 * time.Minute), Level: "info", Message: "Worker started"},
		{Timestamp: now.Add(-3 * time.Minute), Level: "info", Message: "Received request"},
		{Timestamp: now.Add(-1 * time.Minute), Level: "info", Message: "Request completed"},
	}, nil
}

// CreateRoute creates a new worker route.
func (s *Service) CreateRoute(ctx context.Context, in *CreateRouteIn) (*Route, error) {
	if in.Pattern == "" || in.WorkerID == "" {
		return nil, errors.New("pattern and worker_id are required")
	}

	route := &Route{
		ID:       ulid.Make().String(),
		ZoneID:   in.ZoneID,
		Pattern:  in.Pattern,
		WorkerID: in.WorkerID,
		Enabled:  in.Enabled,
	}

	if err := s.store.CreateRoute(ctx, route); err != nil {
		return nil, err
	}

	return route, nil
}

// ListRoutes lists all routes for a zone.
func (s *Service) ListRoutes(ctx context.Context, zoneID string) ([]*Route, error) {
	return s.store.ListRoutes(ctx, zoneID)
}

// DeleteRoute deletes a route.
func (s *Service) DeleteRoute(ctx context.Context, id string) error {
	return s.store.DeleteRoute(ctx, id)
}

