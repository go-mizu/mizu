// Package types contains shared data types for the email blueprint.
package types

import (
	"time"
)

// Email represents a single email message.
type Email struct {
	ID             string      `json:"id"`
	ThreadID       string      `json:"thread_id"`
	MessageID      string      `json:"message_id"`
	InReplyTo      string      `json:"in_reply_to,omitempty"`
	References     []string    `json:"references,omitempty"`
	FromAddress    string      `json:"from_address"`
	FromName       string      `json:"from_name"`
	ToAddresses    []Recipient `json:"to_addresses"`
	CCAddresses    []Recipient `json:"cc_addresses,omitempty"`
	BCCAddresses   []Recipient `json:"bcc_addresses,omitempty"`
	Subject        string      `json:"subject"`
	BodyText       string      `json:"body_text,omitempty"`
	BodyHTML       string      `json:"body_html,omitempty"`
	Snippet        string      `json:"snippet"`
	IsRead         bool        `json:"is_read"`
	IsStarred      bool        `json:"is_starred"`
	IsImportant    bool        `json:"is_important"`
	IsDraft        bool        `json:"is_draft"`
	IsSent         bool        `json:"is_sent"`
	HasAttachments bool        `json:"has_attachments"`
	SizeBytes      int64       `json:"size_bytes"`
	Labels         []string    `json:"labels,omitempty"`
	SentAt         *time.Time  `json:"sent_at,omitempty"`
	ReceivedAt     time.Time   `json:"received_at"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	SnoozedUntil   *time.Time  `json:"snoozed_until,omitempty"`
	ScheduledAt    *time.Time  `json:"scheduled_at,omitempty"`
	IsMuted        bool        `json:"is_muted"`
}

// Recipient represents an email address with an optional display name.
type Recipient struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address"`
}

// Thread represents a conversation thread containing one or more emails.
type Thread struct {
	ID          string    `json:"id"`
	Subject     string    `json:"subject"`
	Snippet     string    `json:"snippet"`
	Emails      []Email   `json:"emails"`
	EmailCount  int       `json:"email_count"`
	UnreadCount int       `json:"unread_count"`
	IsStarred   bool      `json:"is_starred"`
	IsImportant bool      `json:"is_important"`
	Labels      []string  `json:"labels"`
	LastEmailAt time.Time `json:"last_email_at"`
}

// Attachment represents a file attached to an email.
type Attachment struct {
	ID          string    `json:"id"`
	EmailID     string    `json:"email_id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	CreatedAt   time.Time `json:"created_at"`
}

// ComposeRequest represents a request to compose or send an email.
type ComposeRequest struct {
	To        []Recipient `json:"to"`
	CC        []Recipient `json:"cc,omitempty"`
	BCC       []Recipient `json:"bcc,omitempty"`
	Subject   string      `json:"subject"`
	BodyHTML  string      `json:"body_html"`
	BodyText  string      `json:"body_text"`
	InReplyTo string      `json:"in_reply_to,omitempty"`
	ThreadID  string      `json:"thread_id,omitempty"`
	IsDraft   bool        `json:"is_draft"`
}

// BatchAction represents a batch operation on multiple emails.
type BatchAction struct {
	IDs     []string `json:"ids"`
	Action  string   `json:"action"`   // archive, trash, delete, read, unread, star, unstar, important, unimportant, mute, unmute
	LabelID string   `json:"label_id,omitempty"`
}

// ScheduleRequest represents a request to schedule an email for later sending.
type ScheduleRequest struct {
	SendAt time.Time `json:"send_at"`
}

// EmailListResponse represents a paginated list of emails.
type EmailListResponse struct {
	Emails     []Email `json:"emails"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	PerPage    int     `json:"per_page"`
	TotalPages int     `json:"total_pages"`
}

// ThreadListResponse represents a paginated list of threads.
type ThreadListResponse struct {
	Threads    []Thread `json:"threads"`
	Total      int      `json:"total"`
	Page       int      `json:"page"`
	PerPage    int      `json:"per_page"`
	TotalPages int      `json:"total_pages"`
}
