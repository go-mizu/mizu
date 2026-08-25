// Command mizu is the toolkit's command line tool.
//
// It does four things today.
//
//	mizu gen              write what the markers in a project ask for
//	mizu gen --check      report what is out of date and write nothing
//	mizu doctor           check the project and say what is wrong with it
//	mizu version          print the version, one fact per line
//	mizu hash:tune        measure argon2id here and print the cost to configure
//
// The rest of the command tree arrives with the packages it drives. What is
// here now is the shape the rest hangs off: every command is a struct with a
// Spec and a Run, and both of those are ordinary methods taking a
// [console.IO], so a command is tested by calling it rather than by starting a
// process.
//
// Every command takes the flags in [console.Globals] on top of its own:
// --verbose, --quiet, --json, --color, --no-color, --no-interaction and
// --timeout. They can be written before the command name or after it. The two
// this program adds, --profile and --trace, are in [globals].
package main

import (
	"os"

	"github.com/go-mizu/mizu/console"
)

func main() {
	os.Exit(newApp().Main(os.Args[1:]))
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
	a.Add(
		&Doctor{},
		&Gen{},
		&HashTune{},
		&Version{},
	)
	for _, g := range generators {
		a.Add(&Gen{only: g.name})
	}
	return a
}
