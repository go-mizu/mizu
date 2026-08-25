package console_test

import (
	"context"
	"io"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/console"
	"github.com/go-mizu/mizu/errs/diag/diagtest"
)

// TestDiagnostics runs the golden message corpus for this package.
//
// Each directory under testdata/diag holds an args file, which is a command
// line one token per line, and a want.txt, which is what somebody typing that
// command line sees. The command line goes through [console.App.Run]: the
// command is looked up, its flags and arguments are parsed, and the command
// runs if it got that far. What comes back is the error, which is the thing
// under review.
//
// Run rather than Start, because Start returns an exit code and the error is
// what an entry is about. So a command line here is the one a command sees,
// with the global flags already taken out of it.
//
// Run it with -update to rewrite the want.txt files, then read the diff. That
// diff is user-facing text and the five rules in doc 36 section 2.1 are the
// review checklist for it.
func TestDiagnostics(t *testing.T) {
	diagtest.Run(t, "testdata/diag", func(tb testing.TB, c diagtest.Case) error {
		return shop().Run(context.Background(), quiet(), c.Lines(tb, "args"))
	})
}

// quiet is an IO that keeps whatever a command writes out of the report.
//
// The corpus is about the error a command line produced, not about what the
// command printed on its way to it, and a command that printed something before
// failing is a real case worth having in there.
func quiet() *console.IO {
	return console.New(strings.NewReader(""), io.Discard, io.Discard, console.Options{
		Color:       console.ColorNever,
		Interaction: console.InteractionNever,
	})
}

// shop is the program the corpus runs against.
//
// It is a storefront's admin tool, written the way one would be rather than
// assembled out of the flags that reach particular branches. A corpus of
// messages about --flag-a and --flag-b reads as though the messages were
// written for the test, and the whole point of the golden files is that
// somebody can read one and tell whether it is the sentence they would want at
// the end of a bad afternoon.
func shop() *console.App {
	app := &console.App{
		Name:    "shop",
		Desc:    "shop administers a storefront.",
		Version: "1.4.0",
	}
	app.Add(&usersPrune{})
	app.Add(&usersInvite{})
	app.Add(&dbSeed{})
	app.Add(&serve{})
	return app
}

// usersPrune deletes accounts, so it is the command with the most ways to be
// asked for the wrong thing.
type usersPrune struct {
	days   int
	before time.Time
	reason string
	tags   []string
	dryRun bool
	tenant string
}

func (c *usersPrune) Spec() console.Spec {
	return console.Spec{
		Name: "users:prune",
		Desc: "Delete users who never verified their email",
		Flags: []console.Flag{
			{Name: "days", Short: 'd', Default: "30", Desc: "Delete accounts older than this many days", Value: console.Int(&c.days)},
			{Name: "before", Desc: "Delete accounts created before this date", Value: console.Time(&c.before)},
			{Name: "reason", Short: 'r', Required: true, Desc: "Why, for the audit log", Value: console.String(&c.reason)},
			{Name: "tag", Short: 't', Desc: "Only accounts with these tags", Value: console.Strings(&c.tags, ",")},
			{Name: "dry-run", Desc: "Report what would go without deleting it", Value: console.Bool(&c.dryRun)},
		},
		Args: []console.Arg{
			{Name: "tenant", Required: true, Desc: "Tenant slug, or all", Value: console.String(&c.tenant)},
		},
	}
}

func (c *usersPrune) Run(ctx context.Context, io *console.IO) error { return nil }

// usersInvite has the enum and the key=value pair.
type usersInvite struct {
	role   string
	meta   map[string]string
	sendAt time.Time
	emails []string
}

func (c *usersInvite) Spec() console.Spec {
	return console.Spec{
		Name: "users:invite",
		Desc: "Send an invitation",
		Flags: []console.Flag{
			{Name: "role", Default: "member", Desc: "What the invited user may do", Value: console.Enum(&c.role, "owner", "admin", "member", "viewer")},
			{Name: "meta", Desc: "Extra fields on the invitation, as key=value", Value: console.KeyValues(&c.meta)},
			{Name: "send-at", Desc: "Send it then rather than now", Value: console.Time(&c.sendAt)},
		},
		Args: []console.Arg{
			{Name: "email", Required: true, Rest: true, Desc: "Who to invite", Value: console.Strings(&c.emails, "")},
		},
	}
}

func (c *usersInvite) Run(ctx context.Context, io *console.IO) error { return nil }

// dbSeed is the one that fails at run time rather than at parse time, which is
// the other half of what a person sees.
type dbSeed struct {
	class string
	count uint
	fresh bool
}

func (c *dbSeed) Spec() console.Spec {
	return console.Spec{
		Name: "db:seed",
		Desc: "Fill the database with example data",
		Flags: []console.Flag{
			{Name: "class", Default: "DatabaseSeeder", Desc: "Which seeder to run", Value: console.String(&c.class)},
			{Name: "count", Short: 'c', Default: "10", Desc: "How many rows each seeder makes", Value: console.Uint(&c.count)},
			{Name: "fresh", Short: 'f', Desc: "Drop the tables and build them again first", Value: console.Bool(&c.fresh)},
		},
	}
}

func (c *dbSeed) Run(ctx context.Context, io *console.IO) error {
	io.Info("seeding %s", c.class)
	return console.Exit(console.CodeUnavailable, errTooEarly)
}

// errTooEarly is what db:seed fails with, so that the corpus holds a message
// from inside a command as well as the ones from the command line.
var errTooEarly = &notReady{what: "database", addr: "127.0.0.1:5432"}

type notReady struct {
	what string
	addr string
}

func (e *notReady) Error() string {
	return "the " + e.what + " at " + e.addr + " is not accepting connections, start it with docker compose up db"
}

// serve has the duration, the bounded number, the list of a type this package
// never heard of, and a hidden flag.
type serve struct {
	addr     string
	shutdown time.Duration
	workers  uint8
	trust    []netip.Prefix
	pprof    bool
}

func (c *serve) Spec() console.Spec {
	return console.Spec{
		Name: "serve",
		Desc: "Run the HTTP server",
		Long: "serve runs the storefront in the foreground until it is interrupted.",
		Flags: []console.Flag{
			{Name: "addr", Short: 'a', Default: ":8080", Desc: "Address to listen on", Value: console.String(&c.addr)},
			{Name: "shutdown", Default: "30s", Desc: "How long to let requests finish on the way out", Value: console.Duration(&c.shutdown)},
			{Name: "workers", Short: 'w', Default: "4", Desc: "Background workers to run alongside the server", Value: console.Uint(&c.workers)},
			{Name: "trust-proxy", Desc: "Believe X-Forwarded-For from these networks", Value: console.Slice(&c.trust, netip.ParsePrefix, ",")},
			{Name: "pprof", Hidden: true, Desc: "Serve the profiling endpoints", Value: console.Bool(&c.pprof)},
		},
	}
}

func (c *serve) Run(ctx context.Context, io *console.IO) error { return nil }
