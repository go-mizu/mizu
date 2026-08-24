package ctxdata_test

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-mizu/mizu/ctxdata"
)

// TenantID is declared once, next to whatever owns the idea of a tenant. The
// name is what a log record calls it, and the options say what happens to it
// beyond being readable.
var TenantID = ctxdata.NewKey[string]("tenant_id", ctxdata.Logged(), ctxdata.Propagated())

// APIKey is in the record by name so that a support engineer can see which key
// a caller used, and never by value.
var APIKey = ctxdata.NewKey[string]("api_key", ctxdata.Redacted())

func Example() {
	// Middleware puts the tenant on the context once.
	ctx := ctxdata.With(context.Background(), TenantID, "acme")
	ctx = ctxdata.With(ctx, APIKey, "sk_live_7f3a")

	// Anything under it reads the tenant back without being passed it.
	if id, ok := ctxdata.Get(ctx, TenantID); ok {
		_ = id
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: dropTime,
	}))
	log.LogAttrs(ctx, slog.LevelInfo, "rebuilding the index", ctxdata.Attrs(ctx)...)

	// Output:
	// level=INFO msg="rebuilding the index" tenant_id=acme api_key=[redacted]
}

// ExampleAll is how a carrier other than logging reads the data: a queue
// writing an envelope takes the propagated ones and leaves the rest.
func ExampleAll() {
	ctx := ctxdata.With(context.Background(), TenantID, "acme")
	ctx = ctxdata.With(ctx, APIKey, "sk_live_7f3a")

	envelope := map[string]any{}
	for e := range ctxdata.All(ctx) {
		if e.Propagated {
			envelope[e.Name] = e.Value
		}
	}
	slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: dropTime})).
		Info("dispatching", "envelope", envelope)

	// Output:
	// level=INFO msg=dispatching envelope=map[tenant_id:acme]
}

func dropTime(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		return slog.Attr{}
	}
	return a
}
