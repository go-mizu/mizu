package store

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/email/types"
)

// EmailFilter contains filter options for listing emails.
type EmailFilter struct {
	LabelID   string
	Query     string
	IsRead    *bool
	IsStarred *bool
	IsDraft   *bool
	Page      int
	PerPage   int
}

// Store defines the interface for all storage operations.
type Store interface {
	// Schema management
	Ensure(ctx context.Context) error
	Close() error

	// Emails
	ListEmails(ctx context.Context, filter EmailFilter) (*types.EmailListResponse, error)
	GetEmail(ctx context.Context, id string) (*types.Email, error)
	CreateEmail(ctx context.Context, email *types.Email) error
	UpdateEmail(ctx context.Context, id string, updates map[string]any) error
	DeleteEmail(ctx context.Context, id string, permanent bool) error
	BatchUpdateEmails(ctx context.Context, action *types.BatchAction) error
	SearchEmails(ctx context.Context, query string, page, perPage int) (*types.EmailListResponse, error)

	// Threads
	ListThreads(ctx context.Context, filter EmailFilter) (*types.ThreadListResponse, error)
	GetThread(ctx context.Context, id string) (*types.Thread, error)

	// Labels
	ListLabels(ctx context.Context) ([]types.Label, error)
	CreateLabel(ctx context.Context, label *types.Label) error
	UpdateLabel(ctx context.Context, id string, updates map[string]any) error
	DeleteLabel(ctx context.Context, id string) error
	AddEmailLabel(ctx context.Context, emailID, labelID string) error
	RemoveEmailLabel(ctx context.Context, emailID, labelID string) error

	// Contacts
	ListContacts(ctx context.Context, query string) ([]types.Contact, error)
	CreateContact(ctx context.Context, contact *types.Contact) error
	UpdateContact(ctx context.Context, id string, updates map[string]any) error
	DeleteContact(ctx context.Context, id string) error

	// Attachments
	ListAttachments(ctx context.Context, emailID string) ([]types.Attachment, error)
	GetAttachment(ctx context.Context, id string) (*types.Attachment, []byte, error)
	CreateAttachment(ctx context.Context, attachment *types.Attachment, data []byte) error
	DeleteAttachment(ctx context.Context, id string) error

	// Settings
	GetSettings(ctx context.Context) (*types.Settings, error)
	UpdateSettings(ctx context.Context, settings *types.Settings) error

	// Seed
	SeedEmails(ctx context.Context) error
	SeedContacts(ctx context.Context) error
	SeedLabels(ctx context.Context) error
}
