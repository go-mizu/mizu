// Package settings provides site settings management.
package settings

import (
	"context"
	"time"
)

// Setting represents a site setting.
type Setting struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	ValueType   string    `json:"value_type"`
	GroupName   string    `json:"group_name,omitempty"`
	Description string    `json:"description,omitempty"`
	IsPublic    bool      `json:"is_public"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SetIn contains input for setting a value.
type SetIn struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueType   string `json:"value_type,omitempty"`
	GroupName   string `json:"group_name,omitempty"`
	Description string `json:"description,omitempty"`
	IsPublic    *bool  `json:"is_public,omitempty"`
}

// API defines the settings service contract.
type API interface {
	Get(ctx context.Context, key string) (*Setting, error)
	GetByGroup(ctx context.Context, group string) ([]*Setting, error)
	GetAll(ctx context.Context) ([]*Setting, error)
	GetPublic(ctx context.Context) ([]*Setting, error)
	Set(ctx context.Context, in *SetIn) (*Setting, error)
	SetBulk(ctx context.Context, settings []*SetIn) error
	Delete(ctx context.Context, key string) error
}

// Store defines the data access contract for settings.
type Store interface {
	Get(ctx context.Context, key string) (*Setting, error)
	GetByGroup(ctx context.Context, group string) ([]*Setting, error)
	GetAll(ctx context.Context) ([]*Setting, error)
	GetPublic(ctx context.Context) ([]*Setting, error)
	Set(ctx context.Context, s *Setting) error
	Delete(ctx context.Context, key string) error
}
