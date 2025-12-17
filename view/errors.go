package view

import (
	"errors"
	"fmt"
)

// Common errors returned by the view engine.
var (
	ErrTemplateNotFound  = errors.New("template not found")
	ErrLayoutNotFound    = errors.New("layout not found")
	ErrComponentNotFound = errors.New("component not found")
	ErrPartialNotFound   = errors.New("partial not found")
	ErrSlotNotDefined    = errors.New("slot not defined")
)

// TemplateError wraps a template error with context.
type TemplateError struct {
	// Name is the template name.
	Name string

	// File is the template file path.
	File string

	// Line is the line number where the error occurred (if known).
	Line int

	// Err is the underlying error.
	Err error
}

// Error returns the error message.
func (e *TemplateError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("template %q at line %d: %v", e.Name, e.Line, e.Err)
	}
	return fmt.Sprintf("template %q: %v", e.Name, e.Err)
}

// Unwrap returns the underlying error.
func (e *TemplateError) Unwrap() error {
	return e.Err
}

// NotFoundError is returned when a template is not found.
type NotFoundError struct {
	// Type is the template type (page, layout, component, partial).
	Type string

	// Name is the template name.
	Name string
}

// Error returns the error message.
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s %q not found", e.Type, e.Name)
}

// Is reports whether target matches this error.
func (e *NotFoundError) Is(target error) bool {
	switch e.Type {
	case "page":
		return target == ErrTemplateNotFound
	case "layout":
		return target == ErrLayoutNotFound
	case "component":
		return target == ErrComponentNotFound
	case "partial":
		return target == ErrPartialNotFound
	}
	return false
}
