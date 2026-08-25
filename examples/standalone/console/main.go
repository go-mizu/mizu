// Command console is a command line program with one command, which parses its
// own arguments and returns an exit code.
package main

import (
	"context"
	"os"

	"github.com/go-mizu/mizu/console"
)

type greet struct {
	name  string
	loud  bool
	times int
}

func (c *greet) Spec() console.Spec {
	return console.Spec{
		Name: "greet",
		Desc: "Say hello to somebody",
		Flags: []console.Flag{
			{Name: "loud", Desc: "Shout it", Value: console.Bool(&c.loud)},
			{Name: "times", Desc: "How many times", Default: "1", Value: console.Int(&c.times)},
		},
		Args: []console.Arg{
			{Name: "name", Desc: "Who to greet", Required: true, Value: console.String(&c.name)},
		},
	}
}

func (c *greet) Run(ctx context.Context, io *console.IO) error {
	for range c.times {
		if c.loud {
			io.Print("HELLO %s\n", c.name)
			continue
		}
		io.Print("hello %s\n", c.name)
	}
	return nil
}

func main() {
	app := &console.App{Name: "hello", Desc: "A greeter", Version: "1.0.0"}
	app.Add(&greet{})
	os.Exit(app.Main(os.Args[1:]))
}
