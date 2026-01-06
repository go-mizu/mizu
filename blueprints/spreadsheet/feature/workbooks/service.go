package workbooks

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/spreadsheet/pkg/ulid"
)

var (
	ErrNotFound = errors.New("workbook not found")
)

// Service implements the workbooks API.
type Service struct {
	store Store
}

// NewService creates a new workbooks service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new workbook.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Workbook, error) {
	now := time.Now()

	// Set defaults for settings
	settings := in.Settings
	if settings.Locale == "" {
		settings.Locale = "en-US"
	}
	if settings.TimeZone == "" {
		settings.TimeZone = "UTC"
	}
	if settings.CalculationMode == "" {
		settings.CalculationMode = "auto"
	}
	if settings.MaxIterations == 0 {
		settings.MaxIterations = 100
	}
	if settings.MaxChange == 0 {
		settings.MaxChange = 0.001
	}

	workbook := &Workbook{
		ID:        ulid.New(),
		Name:      in.Name,
		OwnerID:   in.OwnerID,
		Settings:  settings,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Create(ctx, workbook); err != nil {
		return nil, err
	}

	return workbook, nil
}

// GetByID retrieves a workbook by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Workbook, error) {
	return s.store.GetByID(ctx, id)
}

// List lists workbooks for a user.
func (s *Service) List(ctx context.Context, userID string) ([]*Workbook, error) {
	return s.store.ListByOwner(ctx, userID)
}

// Update updates a workbook.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Workbook, error) {
	workbook, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != "" {
		workbook.Name = in.Name
	}
	// Merge settings if provided
	if in.Settings.Locale != "" {
		workbook.Settings.Locale = in.Settings.Locale
	}
	if in.Settings.TimeZone != "" {
		workbook.Settings.TimeZone = in.Settings.TimeZone
	}
	if in.Settings.CalculationMode != "" {
		workbook.Settings.CalculationMode = in.Settings.CalculationMode
	}

	workbook.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, workbook); err != nil {
		return nil, err
	}

	return workbook, nil
}

// Delete deletes a workbook.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// Copy creates a copy of a workbook.
func (s *Service) Copy(ctx context.Context, id string, newName string, userID string) (*Workbook, error) {
	original, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.Create(ctx, &CreateIn{
		Name:      newName,
		OwnerID:   userID,
		Settings:  original.Settings,
		CreatedBy: userID,
	})
}
