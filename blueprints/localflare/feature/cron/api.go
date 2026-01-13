// Package cron provides Cron Trigger management.
package cron

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound       = errors.New("trigger not found")
	ErrScriptRequired = errors.New("script_name is required")
	ErrCronRequired   = errors.New("cron expression is required")
)

// Trigger represents a cron trigger.
type Trigger struct {
	ID         string    `json:"id"`
	ScriptName string    `json:"script_name"`
	Cron       string    `json:"cron"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Execution represents a cron execution record.
type Execution struct {
	ID          string     `json:"id"`
	TriggerID   string     `json:"trigger_id"`
	ScheduledAt time.Time  `json:"scheduled_at"`
	StartedAt   time.Time  `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	Status      string     `json:"status"`
	Error       string     `json:"error,omitempty"`
}

// CreateTriggerIn contains input for creating a trigger.
type CreateTriggerIn struct {
	ScriptName string `json:"script_name"`
	Cron       string `json:"cron"`
	Enabled    bool   `json:"enabled"`
}

// UpdateTriggerIn contains input for updating a trigger.
type UpdateTriggerIn struct {
	Cron    *string `json:"cron,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
}

// API defines the Cron service contract.
type API interface {
	Create(ctx context.Context, in *CreateTriggerIn) (*Trigger, error)
	Get(ctx context.Context, id string) (*Trigger, error)
	List(ctx context.Context) ([]*Trigger, error)
	ListByScript(ctx context.Context, scriptName string) ([]*Trigger, error)
	Update(ctx context.Context, id string, in *UpdateTriggerIn) (*Trigger, error)
	Delete(ctx context.Context, id string) error
	GetExecutions(ctx context.Context, triggerID string, limit int) ([]*Execution, error)
}

// Store defines the data access contract.
type Store interface {
	CreateTrigger(ctx context.Context, trigger *Trigger) error
	GetTrigger(ctx context.Context, id string) (*Trigger, error)
	ListTriggers(ctx context.Context) ([]*Trigger, error)
	ListTriggersByScript(ctx context.Context, scriptName string) ([]*Trigger, error)
	UpdateTrigger(ctx context.Context, trigger *Trigger) error
	DeleteTrigger(ctx context.Context, id string) error
	GetRecentExecutions(ctx context.Context, triggerID string, limit int) ([]*Execution, error)
}
