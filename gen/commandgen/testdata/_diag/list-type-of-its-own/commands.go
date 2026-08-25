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

type Tags []string

//mizu:command name=go
type Command struct {
	Tags Tags `flag:""`
}
