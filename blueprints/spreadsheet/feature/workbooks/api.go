// Package workbooks provides workbook management functionality.
package workbooks

import (
	"context"
	"time"
)

// Workbook represents a spreadsheet workbook.
type Workbook struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	OwnerID   string    `json:"owner_id"`
	Settings  Settings  `json:"settings"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Settings contains workbook-level settings.
type Settings struct {
	Locale          string `json:"locale"`           // e.g., "en-US"
	TimeZone        string `json:"time_zone"`        // e.g., "America/New_York"
	CalculationMode string `json:"calculation_mode"` // "auto" or "manual"
	IterativeCalc   bool   `json:"iterative_calc"`
	MaxIterations   int    `json:"max_iterations"`
	MaxChange       float64 `json:"max_change"`
}

// CreateIn contains workbook creation input.
type CreateIn struct {
	Name      string   `json:"name"`
	OwnerID   string   `json:"owner_id"`
	Settings  Settings `json:"settings,omitempty"`
	CreatedBy string   `json:"created_by"`
}

// UpdateIn contains workbook update input.
type UpdateIn struct {
	Name     string   `json:"name,omitempty"`
	Settings Settings `json:"settings,omitempty"`
}

// API defines the workbooks service interface.
type API interface {
	// Create creates a new workbook.
	Create(ctx context.Context, in *CreateIn) (*Workbook, error)

	// GetByID retrieves a workbook by ID.
	GetByID(ctx context.Context, id string) (*Workbook, error)

	// List lists workbooks for a user.
	List(ctx context.Context, userID string) ([]*Workbook, error)

	// Update updates a workbook.
	Update(ctx context.Context, id string, in *UpdateIn) (*Workbook, error)

	// Delete deletes a workbook.
	Delete(ctx context.Context, id string) error

	// Copy creates a copy of a workbook.
	Copy(ctx context.Context, id string, newName string, userID string) (*Workbook, error)
}

// Store defines the workbooks data access interface.
type Store interface {
	Create(ctx context.Context, workbook *Workbook) error
	GetByID(ctx context.Context, id string) (*Workbook, error)
	ListByOwner(ctx context.Context, ownerID string) ([]*Workbook, error)
	Update(ctx context.Context, workbook *Workbook) error
	Delete(ctx context.Context, id string) error
}
