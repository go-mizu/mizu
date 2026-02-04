// Package resend implements the email.Driver interface using the Resend API.
package resend

import (
	"context"
	"fmt"
	"os"

	"github.com/go-mizu/mizu/blueprints/email/pkg/email"
	resendgo "github.com/resend/resend-go/v2"
)

// Config holds Resend driver configuration.
type Config struct {
	// APIKey is the Resend API key. If empty, reads RESEND_API_KEY from environment.
	APIKey string
}

// Driver implements email.Driver using the Resend API.
type Driver struct {
	client *resendgo.Client
}

// New creates a new Resend driver.
func New(cfg Config) (*Driver, error) {
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("RESEND_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("%w: RESEND_API_KEY is required", email.ErrNotConfigured)
	}
	return &Driver{client: resendgo.NewClient(apiKey)}, nil
}

// Name returns "resend".
func (d *Driver) Name() string { return "resend" }

// Send delivers a single email message via the Resend API.
func (d *Driver) Send(ctx context.Context, msg *email.Message) (*email.SendResult, error) {
	if err := msg.Validate(); err != nil {
		return nil, err
	}

	req := buildRequest(msg)

	resp, err := d.client.Emails.SendWithContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", email.ErrSendFailed, err)
	}

	return &email.SendResult{MessageID: resp.Id}, nil
}

// SendBatch delivers multiple email messages via the Resend batch API.
func (d *Driver) SendBatch(ctx context.Context, msgs []*email.Message) ([]*email.SendResult, error) {
	for _, msg := range msgs {
		if err := msg.Validate(); err != nil {
			return nil, err
		}
	}

	reqs := make([]*resendgo.SendEmailRequest, len(msgs))
	for i, msg := range msgs {
		reqs[i] = buildRequest(msg)
	}

	resp, err := d.client.Batch.SendWithContext(ctx, reqs)
	if err != nil {
		return nil, fmt.Errorf("%w: batch: %v", email.ErrSendFailed, err)
	}

	results := make([]*email.SendResult, len(resp.Data))
	for i, r := range resp.Data {
		results[i] = &email.SendResult{MessageID: r.Id}
	}

	return results, nil
}

func buildRequest(msg *email.Message) *resendgo.SendEmailRequest {
	req := &resendgo.SendEmailRequest{
		From:    msg.From,
		To:      msg.To,
		Subject: msg.Subject,
		Html:    msg.HTMLBody,
		Text:    msg.TextBody,
		Cc:      msg.CC,
		Bcc:     msg.BCC,
		ReplyTo: msg.ReplyTo,
		Headers: msg.Headers,
	}

	for _, a := range msg.Attachments {
		req.Attachments = append(req.Attachments, &resendgo.Attachment{
			Filename: a.Filename,
			Content:  a.Data,
		})
	}

	return req
}
