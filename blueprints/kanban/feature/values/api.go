// Package values provides field value management functionality.
package values

import (
	"context"
	"time"
)

// Value represents a field value for an issue.
type Value struct {
	IssueID   string     `json:"issue_id"`
	FieldID   string     `json:"field_id"`
	ValueText *string    `json:"value_text,omitempty"`
	ValueNum  *float64   `json:"value_num,omitempty"`
	ValueBool *bool      `json:"value_bool,omitempty"`
	ValueDate *time.Time `json:"value_date,omitempty"`
	ValueTS   *time.Time `json:"value_ts,omitempty"`
	ValueRef  *string    `json:"value_ref,omitempty"`
	ValueJSON *string    `json:"value_json,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// SetIn contains input for setting a field value.
type SetIn struct {
	ValueText *string    `json:"value_text,omitempty"`
	ValueNum  *float64   `json:"value_num,omitempty"`
	ValueBool *bool      `json:"value_bool,omitempty"`
	ValueDate *time.Time `json:"value_date,omitempty"`
	ValueTS   *time.Time `json:"value_ts,omitempty"`
	ValueRef  *string    `json:"value_ref,omitempty"`
	ValueJSON *string    `json:"value_json,omitempty"`
}

// API defines the values service contract.
type API interface {
	Set(ctx context.Context, issueID, fieldID string, in *SetIn) (*Value, error)
	Get(ctx context.Context, issueID, fieldID string) (*Value, error)
	ListByIssue(ctx context.Context, issueID string) ([]*Value, error)
	ListByField(ctx context.Context, fieldID string) ([]*Value, error)
	Delete(ctx context.Context, issueID, fieldID string) error
	DeleteByIssue(ctx context.Context, issueID string) error
	BulkSet(ctx context.Context, vs []*Value) error
	BulkGetByIssues(ctx context.Context, issueIDs []string) (map[string][]*Value, error)
}

// Store defines the data access contract for values.
type Store interface {
	Set(ctx context.Context, v *Value) error
	Get(ctx context.Context, issueID, fieldID string) (*Value, error)
	ListByIssue(ctx context.Context, issueID string) ([]*Value, error)
	ListByField(ctx context.Context, fieldID string) ([]*Value, error)
	Delete(ctx context.Context, issueID, fieldID string) error
	DeleteByIssue(ctx context.Context, issueID string) error
	BulkSet(ctx context.Context, vs []*Value) error
	BulkGetByIssues(ctx context.Context, issueIDs []string) (map[string][]*Value, error)
}
