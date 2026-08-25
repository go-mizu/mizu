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

//mizu:command
type Command struct{}
