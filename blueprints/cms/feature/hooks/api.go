// Package hooks provides lifecycle hook execution for collections and globals.
package hooks

import (
	"context"
	"net/http"

	"github.com/go-mizu/blueprints/cms/config"
)

// HookContext provides context for collection and global hooks.
type HookContext struct {
	Ctx         context.Context
	Req         *http.Request
	Collection  string
	Global      string
	Operation   Operation
	ID          string
	Data        map[string]any // Mutable data for before hooks
	OriginalDoc map[string]any // Previous document state
	Doc         map[string]any // Result document for after hooks
	User        map[string]any // Authenticated user
	FindArgs    *FindArgs      // For read operations
	Result      any            // Operation result for afterOperation
	Custom      map[string]any // Custom context for passing data between hooks
}

// Operation represents the type of operation being performed.
type Operation string

const (
	OpCreate Operation = "create"
	OpRead   Operation = "read"
	OpUpdate Operation = "update"
	OpDelete Operation = "delete"
	OpLogin  Operation = "login"
	OpLogout Operation = "logout"
)

// FindArgs holds find operation arguments.
type FindArgs struct {
	Where          map[string]any
	Sort           string
	Limit          int
	Page           int
	Depth          int
	Locale         string
	FallbackLocale string
}

// FieldHookContext provides context for field-level hooks.
type FieldHookContext struct {
	Ctx          context.Context
	Value        any             // Current field value
	PreviousValue any            // Previous value (for afterChange)
	OriginalDoc  map[string]any  // Previous document state
	Data         map[string]any  // Full incoming data
	SiblingData  map[string]any  // Data at the same level
	Field        *config.Field   // Field configuration
	Collection   string
	Operation    Operation
	Req          *http.Request
	User         map[string]any
	Path         string          // Field path (e.g., "content.blocks.0.text")
}

// Executor defines the hook execution interface.
type Executor interface {
	// Collection hooks
	ExecuteBeforeOperation(ctx *HookContext, hooks *config.CollectionHooks) error
	ExecuteBeforeValidate(ctx *HookContext, hooks *config.CollectionHooks) error
	ExecuteBeforeChange(ctx *HookContext, hooks *config.CollectionHooks) error
	ExecuteAfterChange(ctx *HookContext, hooks *config.CollectionHooks) error
	ExecuteBeforeRead(ctx *HookContext, hooks *config.CollectionHooks) error
	ExecuteAfterRead(ctx *HookContext, hooks *config.CollectionHooks) error
	ExecuteBeforeDelete(ctx *HookContext, hooks *config.CollectionHooks) error
	ExecuteAfterDelete(ctx *HookContext, hooks *config.CollectionHooks) error
	ExecuteAfterOperation(ctx *HookContext, hooks *config.CollectionHooks) error
	ExecuteAfterError(ctx *HookContext, hooks *config.CollectionHooks, err error) error

	// Auth hooks (for auth-enabled collections)
	ExecuteBeforeLogin(ctx *HookContext, hooks *config.CollectionHooks) error
	ExecuteAfterLogin(ctx *HookContext, hooks *config.CollectionHooks) error
	ExecuteAfterLogout(ctx *HookContext, hooks *config.CollectionHooks) error
	ExecuteAfterMe(ctx *HookContext, hooks *config.CollectionHooks) error
	ExecuteAfterRefresh(ctx *HookContext, hooks *config.CollectionHooks) error
	ExecuteAfterForgotPassword(ctx *HookContext, hooks *config.CollectionHooks) error

	// Global hooks
	ExecuteGlobalBeforeChange(ctx *HookContext, hooks *config.GlobalHooks) error
	ExecuteGlobalAfterChange(ctx *HookContext, hooks *config.GlobalHooks) error
	ExecuteGlobalBeforeRead(ctx *HookContext, hooks *config.GlobalHooks) error
	ExecuteGlobalAfterRead(ctx *HookContext, hooks *config.GlobalHooks) error
}

// FieldExecutor defines the field-level hook execution interface.
type FieldExecutor interface {
	// Field hooks - return modified value
	ExecuteFieldBeforeValidate(ctx *FieldHookContext, hooks *config.FieldHooks) (any, error)
	ExecuteFieldBeforeChange(ctx *FieldHookContext, hooks *config.FieldHooks) (any, error)
	ExecuteFieldAfterChange(ctx *FieldHookContext, hooks *config.FieldHooks) (any, error)
	ExecuteFieldAfterRead(ctx *FieldHookContext, hooks *config.FieldHooks) (any, error)

	// Process all fields in a document
	ProcessFieldsBeforeValidate(ctx *HookContext, fields []config.Field, data map[string]any) (map[string]any, error)
	ProcessFieldsBeforeChange(ctx *HookContext, fields []config.Field, data map[string]any) (map[string]any, error)
	ProcessFieldsAfterChange(ctx *HookContext, fields []config.Field, data map[string]any) (map[string]any, error)
	ProcessFieldsAfterRead(ctx *HookContext, fields []config.Field, data map[string]any) (map[string]any, error)
}
