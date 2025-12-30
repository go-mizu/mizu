// Package validation provides field validation for documents.
package validation

import (
	"context"

	"github.com/go-mizu/blueprints/cms/config"
)

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   any    `json:"value,omitempty"`
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return e.Message
}

// ValidationResult holds the result of validation.
type ValidationResult struct {
	Valid  bool               `json:"valid"`
	Errors []*ValidationError `json:"errors,omitempty"`
}

// IsValid returns true if validation passed.
func (r *ValidationResult) IsValid() bool {
	return r.Valid && len(r.Errors) == 0
}

// AddError adds a validation error.
func (r *ValidationResult) AddError(field, message string, value any) {
	r.Valid = false
	r.Errors = append(r.Errors, &ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// ValidationContext provides context for validation functions.
type ValidationContext struct {
	Ctx         context.Context
	Data        map[string]any // Full document data
	SiblingData map[string]any // Data at the same level
	Operation   string         // "create" or "update"
	User        map[string]any // Authenticated user
	ID          string         // Document ID (for update)
	Collection  string
}

// Validator defines the validation service interface.
type Validator interface {
	// ValidateDocument validates an entire document.
	ValidateDocument(ctx *ValidationContext, data map[string]any, fields []config.Field) *ValidationResult

	// ValidateField validates a single field.
	ValidateField(ctx *ValidationContext, value any, field *config.Field, path string) *ValidationResult

	// ValidateRequired checks if a required field has a value.
	ValidateRequired(value any) bool

	// ValidateMinLength checks minimum string length.
	ValidateMinLength(value string, min int) bool

	// ValidateMaxLength checks maximum string length.
	ValidateMaxLength(value string, max int) bool

	// ValidateMin checks minimum numeric value.
	ValidateMin(value float64, min float64) bool

	// ValidateMax checks maximum numeric value.
	ValidateMax(value float64, max float64) bool

	// ValidateEmail validates email format.
	ValidateEmail(value string) bool

	// ValidateUnique checks if a value is unique in the collection.
	ValidateUnique(ctx context.Context, collection, field string, value any, excludeID string) (bool, error)
}

// UniqueChecker defines the interface for checking uniqueness.
type UniqueChecker interface {
	IsUnique(ctx context.Context, collection, field string, value any, excludeID string) (bool, error)
}
