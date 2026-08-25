package broken

import (
	"context"
	"net/netip"
	"time"

	"github.com/go-mizu/mizu/console"
)

var (
	_ context.Context
	_ time.Time
	_ netip.Addr
	_ console.Spec
)

func (c *Command) Run(ctx context.Context, io *console.IO) error { return nil }

//mizu:command name=go
type Command struct{}

func (c *Command) Spec() console.Spec { return console.Spec{Name: "go"} }
