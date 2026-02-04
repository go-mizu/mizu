// Package email defines a driver-based interface for sending emails.
// Use Noop() for local-only mode, or a concrete driver like resend.New()
// for actual delivery.
package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Driver defines the interface for sending emails.
// Implementations must be safe for concurrent use.
type Driver interface {
	// Send delivers a single email message.
	Send(ctx context.Context, msg *Message) (*SendResult, error)

	// SendBatch delivers multiple email messages.
	// Returns results for each message. On partial failure,
	// successful sends still have their MessageID populated.
	SendBatch(ctx context.Context, msgs []*Message) ([]*SendResult, error)

	// Name returns the driver name (e.g. "resend", "smtp", "noop").
	Name() string
}

// Message represents an email to be sent.
type Message struct {
	From        string            // "Name <addr>" or "addr"
	To          []string          // recipient addresses
	CC          []string          // carbon-copy
	BCC         []string          // blind carbon-copy
	ReplyTo     string            // reply-to address
	Subject     string            // subject line
	TextBody    string            // plain text body
	HTMLBody    string            // HTML body
	Headers     map[string]string // custom headers (Message-ID, In-Reply-To, References)
	Attachments []Attachment      // file attachments
}

// Attachment represents a file attached to an email.
type Attachment struct {
	Filename    string // e.g. "report.pdf"
	ContentType string // MIME type, e.g. "application/pdf"
	Data        []byte // raw file bytes
}

// SendResult is the response after a successful send.
type SendResult struct {
	MessageID string // provider-assigned message ID
}

var (
	// ErrInvalidMessage is returned when a message fails validation.
	ErrInvalidMessage = errors.New("email: invalid message")

	// ErrSendFailed is returned when the provider rejects the send.
	ErrSendFailed = errors.New("email: send failed")

	// ErrRateLimit is returned when the provider rate-limits the request.
	ErrRateLimit = errors.New("email: rate limited")

	// ErrNotConfigured is returned when the driver is not properly configured.
	ErrNotConfigured = errors.New("email: driver not configured")
)

// Validate checks that the message has required fields.
func (m *Message) Validate() error {
	if m.From == "" {
		return fmt.Errorf("%w: from address is required", ErrInvalidMessage)
	}
	if len(m.To) == 0 {
		return fmt.Errorf("%w: at least one recipient is required", ErrInvalidMessage)
	}
	if m.Subject == "" {
		return fmt.Errorf("%w: subject is required", ErrInvalidMessage)
	}
	if m.TextBody == "" && m.HTMLBody == "" {
		return fmt.Errorf("%w: text or html body is required", ErrInvalidMessage)
	}
	return nil
}

// Noop returns a no-op driver that accepts all sends without delivering.
// Useful for development and testing.
func Noop() Driver { return &noopDriver{} }

type noopDriver struct{}

func (d *noopDriver) Name() string { return "noop" }

func (d *noopDriver) Send(_ context.Context, msg *Message) (*SendResult, error) {
	if err := msg.Validate(); err != nil {
		return nil, err
	}
	return &SendResult{MessageID: "noop-" + uuid.New().String()}, nil
}

func (d *noopDriver) SendBatch(_ context.Context, msgs []*Message) ([]*SendResult, error) {
	results := make([]*SendResult, len(msgs))
	for i, msg := range msgs {
		r, err := d.Send(context.Background(), msg)
		if err != nil {
			return nil, err
		}
		results[i] = r
	}
	return results, nil
}
