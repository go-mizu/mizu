package main

import (
	"os"

	"github.com/go-mizu/mizu/console"
)

func main() {
	os.Exit(newApp().Main(os.Args[1:]))
}

// registry is every command mizu has, in the order they are registered.
//
// It is a list of its own rather than something newApp builds inline so that a
// test can walk the tree, which is how the reference in doc.go is held to what
// the program does.
func registry() []console.Command {
	cmds := []console.Command{
		&About{},
		&Doctor{},
		&Gen{},
		&HashTune{},
		&Lint{},
		&New{},
		&Verify{quick: true},
		&Verify{},
		&Version{},
	}
	for _, g := range generators {
		cmds = append(cmds, &Gen{only: g.name})
	}
	return cmds
}

// newApp builds the command line.
//
// It is a function rather than a package variable so that a test gets a fresh
// one. Commands hold the values their flags parsed into, so an app that ran
// once is an app whose commands are already filled in.
func newApp() *console.App {
	var g globals
	a := &console.App{
		Name:    "mizu",
		Desc:    "The command line tool for the mizu toolkit",
		Version: self().Version,
		Globals: g.flags(),
		Before:  g.before,
	}
	a.Add(registry()...)
	return a
}
