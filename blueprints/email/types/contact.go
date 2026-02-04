package types

import (
	"time"
)

// Contact represents an email contact.
type Contact struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	Name          string     `json:"name"`
	AvatarURL     string     `json:"avatar_url,omitempty"`
	IsFrequent    bool       `json:"is_frequent"`
	LastContacted *time.Time `json:"last_contacted,omitempty"`
	ContactCount  int        `json:"contact_count"`
	CreatedAt     time.Time  `json:"created_at"`
}
