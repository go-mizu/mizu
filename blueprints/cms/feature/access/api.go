// Package access provides document and field-level access control.
package access

import (
	"context"
	"net/http"

	"github.com/go-mizu/blueprints/cms/config"
)

// AccessContext provides context for access control functions.
type AccessContext struct {
	Ctx        context.Context
	Req        *http.Request
	User       map[string]any
	ID         string
	Data       map[string]any
	Doc        map[string]any
	Collection string
	Global     string
}

// Result represents the result of an access control check.
type Result struct {
	Allowed bool           // Direct allow/deny
	Where   map[string]any // Query constraint for filtered access
}

// Checker defines the access control checking interface.
type Checker interface {
	// Document-level access
	CanCreate(ctx *AccessContext, access *config.AccessConfig) (*Result, error)
	CanRead(ctx *AccessContext, access *config.AccessConfig) (*Result, error)
	CanUpdate(ctx *AccessContext, access *config.AccessConfig) (*Result, error)
	CanDelete(ctx *AccessContext, access *config.AccessConfig) (*Result, error)
	CanAdmin(ctx *AccessContext, access *config.AccessConfig) (*Result, error)

	// Apply query constraints
	ApplyAccessFilter(where map[string]any, accessWhere map[string]any) map[string]any
}

// FieldChecker defines field-level access control.
type FieldChecker interface {
	// Field-level access
	CanCreateField(ctx *AccessContext, field *config.Field) (bool, error)
	CanReadField(ctx *AccessContext, field *config.Field) (bool, error)
	CanUpdateField(ctx *AccessContext, field *config.Field) (bool, error)

	// Filter document based on field access
	FilterReadableFields(doc map[string]any, fields []config.Field, ctx *AccessContext) (map[string]any, error)
	FilterWritableFields(data map[string]any, fields []config.Field, ctx *AccessContext, isCreate bool) (map[string]any, error)
}
