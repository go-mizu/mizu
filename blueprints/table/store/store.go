// Package store provides data access abstractions.
package store

import (
	"github.com/go-mizu/blueprints/table/feature/attachments"
	"github.com/go-mizu/blueprints/table/feature/bases"
	"github.com/go-mizu/blueprints/table/feature/comments"
	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/feature/operations"
	"github.com/go-mizu/blueprints/table/feature/records"
	"github.com/go-mizu/blueprints/table/feature/shares"
	"github.com/go-mizu/blueprints/table/feature/tables"
	"github.com/go-mizu/blueprints/table/feature/users"
	"github.com/go-mizu/blueprints/table/feature/views"
	"github.com/go-mizu/blueprints/table/feature/webhooks"
	"github.com/go-mizu/blueprints/table/feature/workspaces"
)

// Store provides access to all feature stores.
type Store interface {
	Users() users.Store
	Workspaces() workspaces.Store
	Bases() bases.Store
	Tables() tables.Store
	Fields() fields.Store
	Records() records.Store
	Views() views.Store
	Operations() operations.Store
	Shares() shares.Store
	Attachments() attachments.Store
	Comments() comments.Store
	Webhooks() webhooks.Store
	Close() error
}
