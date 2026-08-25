// Package app is a set of commands written the way an application writes them,
// for the generator to read. The file beside this one is what it wrote, checked
// in and compiled, so a change to the output is a change in a diff.
package app

import (
	"context"
	"net/netip"
	"time"

	"github.com/go-mizu/mizu/console"
)

// Format is how a command prints what it did, and is here so that a defined
// type over a string gets the same treatment as a string.
type Format string

// UsersPrune deletes users who never verified their email.
//
//mizu:command name=users:prune
type UsersPrune struct {
	// Which tenant to prune.
	Tenant string `arg:"0"`

	// How long unverified is too long.
	Days int `flag:"days,d" default:"30"`

	// Say what would go and delete nothing.
	DryRun bool `flag:""`

	// How long to wait between batches.
	Wait time.Duration `flag:"" env:"MIZU_PRUNE_WAIT" default:"5s"`

	// How to print what happened.
	Format Format `flag:"format,f" enum:"text|json" default:"text"`

	pruned int // what the command did, which is not part of the command line
}

func (c *UsersPrune) Run(ctx context.Context, io *console.IO) error {
	return io.JSON(c)
}

//mizu:command name=db:wipe desc="Drop every table and start again" long="Every table goes, including the one the migrations are recorded in. Only the local and test environments allow it." hidden
type DbWipe struct {
	// Answer the question this would otherwise ask.
	Force bool `flag:"force,f"`

	// The database to wipe.
	URL string `flag:"url" env:"DATABASE_URL" required:"true" desc:"Which database, since this one has no safe default"`

	// How much to say, given once per level.
	Loud int `flag:"loud,l" count:"true"`

	// Kept for the build that still passes it.
	Legacy bool `flag:"legacy" hidden:"true"`
}

func (c *DbWipe) Run(ctx context.Context, io *console.IO) error {
	return io.JSON(c)
}

// Serve runs the HTTP server until it is told to stop.
//
//mizu:command name=serve
type Serve struct {
	// The address to listen on.
	Bind netip.Addr `flag:"bind,b" default:"127.0.0.1"`

	// The port to listen on.
	Port uint16 `flag:"port,p" default:"8080"`

	// The share of requests to trace, from 0 to 1.
	Sample float64 `flag:"sample" default:"0.01"`

	// How long a request may take.
	Timeout time.Duration `flag:"timeout,t" default:"30s"`

	// When the build being served was made.
	Built time.Time `flag:"built"`

	// Where a browser request may come from.
	Origins []string `flag:"origin,o"`

	// The ports to redirect to this one.
	Redirect []uint16 `flag:"redirect"`

	// Headers to add to every response.
	Header map[string]string `flag:"header,H"`

	// Files to read before starting, which may have commas in their names.
	Include []string `flag:"include" sep:""`
}

func (c *Serve) Run(ctx context.Context, io *console.IO) error {
	return io.JSON(c)
}

// Deploy sends a build somewhere.
//
//mizu:command name=deploy
type Deploy struct {
	// Where to send it.
	Target string `arg:"0" enum:"staging|production"`

	// Which build, or the last one.
	Ref string `arg:"1" default:"HEAD"`

	// The services to send, or all of them.
	Services []string `arg:"2..." required:"false" sep:""`
}

func (c *Deploy) Run(ctx context.Context, io *console.IO) error {
	return io.JSON(c)
}
