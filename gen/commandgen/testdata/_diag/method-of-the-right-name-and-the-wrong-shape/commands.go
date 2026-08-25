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

type NoArgs struct{}

func (n *NoArgs) UnmarshalText() error { return nil }

type NotBytes struct{}

func (n *NotBytes) UnmarshalText(s string) error { return nil }

type NotAMethod struct {
	UnmarshalText func([]byte) error
}

//mizu:command name=go
type Command struct {
	A NoArgs     `flag:"a"`
	B NotBytes   `flag:"b"`
	C NotAMethod `flag:"c"`
}
