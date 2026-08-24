package ctxdata

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestAttrs(t *testing.T) {
	ctx := context.Background()
	ctx = With(ctx, tenantID, "acme")
	ctx = With(ctx, attempt, 3)
	ctx = With(ctx, userID, 12)
	ctx = With(ctx, token, "hunter2")

	attrs := Attrs(ctx)
	if len(attrs) != 3 {
		t.Fatalf("Attrs has %d attributes, want 3: %v", len(attrs), attrs)
	}
	want := []string{"tenant_id=acme", "user_id=12", "token=" + Mask}
	for i, attr := range attrs {
		if got := attr.String(); got != want[i] {
			t.Errorf("attribute %d is %s, want %s", i, got, want[i])
		}
	}
}

func TestAttrsOfNothing(t *testing.T) {
	if got := Attrs(context.Background()); got != nil {
		t.Errorf("Attrs of an empty context = %v", got)
	}
	if got := Attrs(With(context.Background(), attempt, 3)); got != nil {
		t.Errorf("a context with nothing logged in it produced %v", got)
	}
}

// TestAttrsInARecord is the whole point, written the way an application writes
// it.
func TestAttrsInARecord(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	})
	log := slog.New(h)

	ctx := With(With(context.Background(), tenantID, "acme"), token, "hunter2")
	log.LogAttrs(ctx, slog.LevelInfo, "rebuilding the index", Attrs(ctx)...)

	got := strings.TrimSpace(buf.String())
	want := `level=INFO msg="rebuilding the index" tenant_id=acme token=[redacted]`
	if got != want {
		t.Errorf("logged\n%s\nwant\n%s", got, want)
	}
}
