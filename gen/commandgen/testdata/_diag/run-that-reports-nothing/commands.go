package broken

import (
	"context"

	"github.com/go-mizu/mizu/console"
)

//mizu:command name=go
type Command struct{}

func (c *Command) Run(ctx context.Context, io *console.IO) {}
