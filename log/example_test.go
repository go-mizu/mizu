package log_test

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/go-mizu/mizu/ctxdata"
	"github.com/go-mizu/mizu/errs"
	"github.com/go-mizu/mizu/log"
)

// TenantID is the sort of datum middleware puts on the context once, and every
// record made under it carries.
var TenantID = ctxdata.NewKey[string]("tenant_id", ctxdata.Logged())

func Example() {
	h := log.NewConsoleHandler(os.Stdout, log.ConsoleOptions{})
	l := slog.New(fixed{h})

	l.Info("listening", "addr", ":8080")
	l.Info("request", "method", "GET", "path", "/posts", "status", 200, "dur", 4200*time.Microsecond)
	l.Warn("slow query", "dur", 812*time.Millisecond, "rows", 1204)

	// Output:
	// 10:44:02.113 INF listening                    addr=:8080
	// 10:44:02.113 INF request                      method=GET path=/posts status=200 dur=4.2ms
	// 10:44:02.113 WRN slow query                   dur=812ms rows=1204
}

// ExampleNewConsoleHandler is the handler to install while developing. In a
// terminal the level and the keys are coloured, and a message longer than the
// column width pushes its own attributes right rather than being cut.
func ExampleNewConsoleHandler() {
	h := log.NewConsoleHandler(os.Stdout, log.ConsoleOptions{
		Level:      slog.LevelDebug,
		TimeFormat: time.TimeOnly,
	})
	l := slog.New(fixed{h})

	l.Debug("cache miss", "key", "post:12")
	l.Info("rebuilt the index in the background because the last run was interrupted", "docs", 4021)

	// Output:
	// 10:44:02 DBG cache miss                   key=post:12
	// 10:44:02 INF rebuilt the index in the background because the last run was interrupted docs=4021
}

// ExampleNewJSONHandler is the handler to run in production. An error is
// expanded into its message, its kind and its code, so that a query can count
// the unavailable ones without matching on text.
func ExampleNewJSONHandler() {
	h := log.NewJSONHandler(os.Stdout, log.JSONOptions{Level: slog.LevelInfo})
	l := slog.New(fixed{h})

	err := errs.New(errs.Unavailable, "mail.down", "dial tcp 10.0.0.4:25: connection refused")
	l.Error("could not send the welcome mail", "user_id", 4021, "err", err)

	// Output:
	// {"time":"2026-08-24T10:44:02.113Z","level":"ERROR","msg":"could not send the welcome mail","user_id":4021,"err":"dial tcp 10.0.0.4:25: connection refused","err_kind":"unavailable","err_code":"mail.down"}
}

// ExampleDefaultRedact shows the backstop. A value that arrives under one of
// these keys is written as [log.Mask], whatever put it there.
func ExampleDefaultRedact() {
	h := log.NewJSONHandler(os.Stdout, log.JSONOptions{})
	l := slog.New(fixed{h})

	l.Info("signing in", "user", "ada", "password", "hunter2")

	// A handler built with an empty list masks nothing, which is a thing to
	// write down on purpose.
	plain := slog.New(fixed{log.NewJSONHandler(os.Stdout, log.JSONOptions{Redact: []string{}})})
	plain.Info("signing in", "user", "ada", "password", "hunter2")

	// Output:
	// {"time":"2026-08-24T10:44:02.113Z","level":"INFO","msg":"signing in","user":"ada","password":"[redacted]"}
	// {"time":"2026-08-24T10:44:02.113Z","level":"INFO","msg":"signing in","user":"ada","password":"hunter2"}
}

// Example_contextData is the reason a request id ends up on every line without
// being passed down. Middleware puts it on the context, and the handler reads
// it back.
func Example_contextData() {
	l := slog.New(fixed{log.NewConsoleHandler(os.Stdout, log.ConsoleOptions{})})

	ctx := ctxdata.With(context.Background(), TenantID, "acme")
	l.InfoContext(ctx, "rebuilding the index")
	l.InfoContext(ctx, "rebuilt", "docs", 4021)

	// Output:
	// 10:44:02.113 INF rebuilding the index         tenant_id=acme
	// 10:44:02.113 INF rebuilt                      tenant_id=acme docs=4021
}

// fixed gives every record the same time, since an example has to write down
// what it prints. A program has no use for it.
type fixed struct{ slog.Handler }

func (h fixed) Handle(ctx context.Context, r slog.Record) error {
	r.Time = time.Date(2026, 8, 24, 10, 44, 2, 113_000_000, time.UTC)
	return h.Handler.Handle(ctx, r)
}
