package types

import (
	"time"
)

// LabelType represents the type of a label.
type LabelType string

const (
	// LabelTypeSystem represents a built-in system label (inbox, sent, trash, etc.).
	LabelTypeSystem LabelType = "system"
	// LabelTypeUser represents a user-created custom label.
	LabelTypeUser LabelType = "user"
)

// Label represents a label that can be applied to emails.
type Label struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Color       string    `json:"color,omitempty"`
	Type        LabelType `json:"type"`
	Visible     bool      `json:"visible"`
	Position    int       `json:"position"`
	UnreadCount int       `json:"unread_count"`
	TotalCount  int       `json:"total_count"`
	CreatedAt   time.Time `json:"created_at"`
}
