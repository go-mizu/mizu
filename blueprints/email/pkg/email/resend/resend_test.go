package resend

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/go-mizu/mizu/blueprints/email/pkg/email"
)

func TestNew_NoKey(t *testing.T) {
	// Ensure env var is not set for this test
	orig := os.Getenv("RESEND_API_KEY")
	os.Unsetenv("RESEND_API_KEY")
	defer func() {
		if orig != "" {
			os.Setenv("RESEND_API_KEY", orig)
		}
	}()

	_, err := New(Config{})
	if err == nil {
		t.Fatal("expected error when no API key")
	}
	if !errors.Is(err, email.ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got: %v", err)
	}
}

func TestNew_ExplicitKey(t *testing.T) {
	d, err := New(Config{APIKey: "re_test_key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d == nil {
		t.Fatal("expected non-nil driver")
	}
}

func TestNew_EnvKey(t *testing.T) {
	orig := os.Getenv("RESEND_API_KEY")
	os.Setenv("RESEND_API_KEY", "re_env_test_key")
	defer func() {
		if orig != "" {
			os.Setenv("RESEND_API_KEY", orig)
		} else {
			os.Unsetenv("RESEND_API_KEY")
		}
	}()

	d, err := New(Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d == nil {
		t.Fatal("expected non-nil driver")
	}
}

func TestDriver_Name(t *testing.T) {
	d, _ := New(Config{APIKey: "re_test"})
	if d.Name() != "resend" {
		t.Fatalf("expected %q, got %q", "resend", d.Name())
	}
}

func TestDriver_Send_InvalidMessage(t *testing.T) {
	d, _ := New(Config{APIKey: "re_test"})
	_, err := d.Send(context.Background(), &email.Message{})
	if err == nil {
		t.Fatal("expected error for invalid message")
	}
	if !errors.Is(err, email.ErrInvalidMessage) {
		t.Fatalf("expected ErrInvalidMessage, got: %v", err)
	}
}

func TestDriver_SendBatch_InvalidMessage(t *testing.T) {
	d, _ := New(Config{APIKey: "re_test"})
	msgs := []*email.Message{
		{From: "a@b.com", To: []string{"c@d.com"}, Subject: "OK", TextBody: "ok"},
		{}, // invalid
	}
	_, err := d.SendBatch(context.Background(), msgs)
	if err == nil {
		t.Fatal("expected error for batch with invalid message")
	}
	if !errors.Is(err, email.ErrInvalidMessage) {
		t.Fatalf("expected ErrInvalidMessage, got: %v", err)
	}
}

// Interface compliance check.
func TestDriverInterface(t *testing.T) {
	var _ email.Driver = &Driver{}
}

// --- Integration tests (require RESEND_API_KEY) ---

func getTestDriver(t *testing.T) *Driver {
	t.Helper()
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		t.Skip("RESEND_API_KEY not set, skipping integration test")
	}
	d, err := New(Config{APIKey: apiKey})
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}
	return d
}

func TestIntegration_Send(t *testing.T) {
	d := getTestDriver(t)
	ctx := context.Background()

	msg := &email.Message{
		From:     "onboarding@resend.dev",
		To:       []string{"delivered@resend.dev"},
		Subject:  "Test from email blueprint",
		TextBody: "This is a test email sent from the Go email driver integration tests.",
		HTMLBody: "<p>This is a <strong>test email</strong> sent from the Go email driver integration tests.</p>",
	}

	result, err := d.Send(ctx, msg)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if result.MessageID == "" {
		t.Fatal("expected non-empty MessageID")
	}
	t.Logf("Sent message ID: %s", result.MessageID)
}

func TestIntegration_Send_WithAttachment(t *testing.T) {
	d := getTestDriver(t)
	ctx := context.Background()

	msg := &email.Message{
		From:     "onboarding@resend.dev",
		To:       []string{"delivered@resend.dev"},
		Subject:  "Test with attachment",
		TextBody: "This email has an attachment.",
		Attachments: []email.Attachment{
			{
				Filename:    "test.txt",
				ContentType: "text/plain",
				Data:        []byte("Hello from the email driver test!"),
			},
		},
	}

	result, err := d.Send(ctx, msg)
	if err != nil {
		t.Fatalf("Send with attachment failed: %v", err)
	}
	if result.MessageID == "" {
		t.Fatal("expected non-empty MessageID")
	}
	t.Logf("Sent message with attachment, ID: %s", result.MessageID)
}

func TestIntegration_Send_WithCC(t *testing.T) {
	d := getTestDriver(t)
	ctx := context.Background()

	msg := &email.Message{
		From:     "onboarding@resend.dev",
		To:       []string{"delivered@resend.dev"},
		CC:       []string{"delivered@resend.dev"},
		ReplyTo:  "delivered@resend.dev",
		Subject:  "Test with CC and ReplyTo",
		TextBody: "This email tests CC and ReplyTo fields.",
	}

	result, err := d.Send(ctx, msg)
	if err != nil {
		t.Fatalf("Send with CC failed: %v", err)
	}
	if result.MessageID == "" {
		t.Fatal("expected non-empty MessageID")
	}
	t.Logf("Sent message with CC, ID: %s", result.MessageID)
}

func TestIntegration_Send_WithHeaders(t *testing.T) {
	d := getTestDriver(t)
	ctx := context.Background()

	msg := &email.Message{
		From:    "onboarding@resend.dev",
		To:      []string{"delivered@resend.dev"},
		Subject: "Test with custom headers",
		HTMLBody: "<p>Testing custom headers.</p>",
		Headers: map[string]string{
			"X-Custom-Header": "test-value",
			"In-Reply-To":     "<original-message-id@example.com>",
		},
	}

	result, err := d.Send(ctx, msg)
	if err != nil {
		t.Fatalf("Send with headers failed: %v", err)
	}
	if result.MessageID == "" {
		t.Fatal("expected non-empty MessageID")
	}
	t.Logf("Sent message with headers, ID: %s", result.MessageID)
}

func TestIntegration_SendBatch(t *testing.T) {
	d := getTestDriver(t)
	ctx := context.Background()

	msgs := []*email.Message{
		{
			From:     "onboarding@resend.dev",
			To:       []string{"delivered@resend.dev"},
			Subject:  "Batch test 1",
			TextBody: "First batch email.",
		},
		{
			From:     "onboarding@resend.dev",
			To:       []string{"delivered@resend.dev"},
			Subject:  "Batch test 2",
			TextBody: "Second batch email.",
		},
	}

	results, err := d.SendBatch(ctx, msgs)
	if err != nil {
		t.Fatalf("SendBatch failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for i, r := range results {
		if r.MessageID == "" {
			t.Fatalf("result[%d]: expected non-empty MessageID", i)
		}
		t.Logf("Batch result[%d] ID: %s", i, r.MessageID)
	}
}
