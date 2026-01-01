// Package activity provides activity logging functionality.
package activity

import (
	"context"
	"time"
)

// Action types
const (
	ActionFileUpload   = "file.upload"
	ActionFileDownload = "file.download"
	ActionFileView     = "file.view"
	ActionFileRename   = "file.rename"
	ActionFileMove     = "file.move"
	ActionFileCopy     = "file.copy"
	ActionFileTrash    = "file.trash"
	ActionFileRestore  = "file.restore"
	ActionFileDelete   = "file.delete"
	ActionFileStar     = "file.star"
	ActionFileUnstar   = "file.unstar"

	ActionFolderCreate  = "folder.create"
	ActionFolderRename  = "folder.rename"
	ActionFolderMove    = "folder.move"
	ActionFolderTrash   = "folder.trash"
	ActionFolderRestore = "folder.restore"
	ActionFolderDelete  = "folder.delete"
	ActionFolderStar    = "folder.star"
	ActionFolderUnstar  = "folder.unstar"

	ActionShareCreate = "share.create"
	ActionShareUpdate = "share.update"
	ActionShareDelete = "share.delete"

	ActionUserLogin  = "user.login"
	ActionUserLogout = "user.logout"
)

// Activity represents an activity log entry.
type Activity struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	ResourceName string    `json:"resource_name,omitempty"`
	Details      string    `json:"details,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// LogIn contains input for logging an activity.
type LogIn struct {
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	ResourceName string `json:"resource_name,omitempty"`
	Details      string `json:"details,omitempty"`
	IPAddress    string `json:"ip_address,omitempty"`
	UserAgent    string `json:"user_agent,omitempty"`
}

// API defines the activity service contract.
type API interface {
	Log(ctx context.Context, userID string, in *LogIn) (*Activity, error)
	ListByUser(ctx context.Context, userID string, limit int) ([]*Activity, error)
	ListForResource(ctx context.Context, resourceType, resourceID string, limit int) ([]*Activity, error)
	ListRecent(ctx context.Context, limit int) ([]*Activity, error)
}
