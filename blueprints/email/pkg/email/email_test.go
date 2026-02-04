package email

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestMessage_Validate(t *testing.T) {
	tests := []struct {
		name    string
		msg     Message
		wantErr string
	}{
		{
			name:    "valid message with text body",
			msg:     Message{From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi", TextBody: "hello"},
			wantErr: "",
		},
		{
			name:    "valid message with html body",
			msg:     Message{From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi", HTMLBody: "<p>hello</p>"},
			wantErr: "",
		},
		{
			name:    "valid message with both bodies",
			msg:     Message{From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi", TextBody: "hello", HTMLBody: "<p>hello</p>"},
			wantErr: "",
		},
		{
			name:    "missing from",
			msg:     Message{To: []string{"c@d.com"}, Subject: "hi", TextBody: "hello"},
			wantErr: "from address is required",
		},
		{
			name:    "missing to",
			msg:     Message{From: "a@b.com", Subject: "hi", TextBody: "hello"},
			wantErr: "at least one recipient is required",
		},
		{
			name:    "empty to slice",
			msg:     Message{From: "a@b.com", To: []string{}, Subject: "hi", TextBody: "hello"},
			wantErr: "at least one recipient is required",
		},
		{
			name:    "missing subject",
			msg:     Message{From: "a@b.com", To: []string{"c@d.com"}, TextBody: "hello"},
			wantErr: "subject is required",
		},
		{
			name:    "missing body",
			msg:     Message{From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi"},
			wantErr: "text or html body is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tt.wantErr, err)
			}
			if !errors.Is(err, ErrInvalidMessage) {
				t.Fatalf("expected error to wrap ErrInvalidMessage, got: %v", err)
			}
		})
	}
}

func TestNoopDriver_Name(t *testing.T) {
	d := Noop()
	if d.Name() != "noop" {
		t.Fatalf("expected name %q, got %q", "noop", d.Name())
	}
}

func TestNoopDriver_Send(t *testing.T) {
	d := Noop()
	ctx := context.Background()

	msg := &Message{
		From:     "sender@example.com",
		To:       []string{"recipient@example.com"},
		Subject:  "Test",
		TextBody: "Hello",
	}

	result, err := d.Send(ctx, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MessageID == "" {
		t.Fatal("expected non-empty MessageID")
	}
	if !strings.HasPrefix(result.MessageID, "noop-") {
		t.Fatalf("expected MessageID to start with 'noop-', got %q", result.MessageID)
	}
}

func TestNoopDriver_Send_InvalidMessage(t *testing.T) {
	d := Noop()
	ctx := context.Background()

	msg := &Message{} // missing everything
	_, err := d.Send(ctx, msg)
	if err == nil {
		t.Fatal("expected error for invalid message")
	}
	if !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("expected ErrInvalidMessage, got: %v", err)
	}
}

func TestNoopDriver_SendBatch(t *testing.T) {
	d := Noop()
	ctx := context.Background()

	msgs := []*Message{
		{From: "a@b.com", To: []string{"c@d.com"}, Subject: "One", TextBody: "1"},
		{From: "a@b.com", To: []string{"e@f.com"}, Subject: "Two", HTMLBody: "<p>2</p>"},
	}

	results, err := d.SendBatch(ctx, msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for i, r := range results {
		if r.MessageID == "" {
			t.Fatalf("result[%d]: expected non-empty MessageID", i)
		}
		if !strings.HasPrefix(r.MessageID, "noop-") {
			t.Fatalf("result[%d]: expected MessageID to start with 'noop-', got %q", i, r.MessageID)
		}
	}
}

func TestNoopDriver_SendBatch_InvalidMessage(t *testing.T) {
	d := Noop()
	ctx := context.Background()

	msgs := []*Message{
		{From: "a@b.com", To: []string{"c@d.com"}, Subject: "OK", TextBody: "ok"},
		{}, // invalid
	}

	_, err := d.SendBatch(ctx, msgs)
	if err == nil {
		t.Fatal("expected error for batch with invalid message")
	}
	if !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("expected ErrInvalidMessage, got: %v", err)
	}
}

// TestDriverInterface verifies the interface is satisfied at compile time.
func TestDriverInterface(t *testing.T) {
	var _ Driver = Noop()
	var _ Driver = &noopDriver{}
}
