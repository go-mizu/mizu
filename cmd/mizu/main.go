// Command mizu is the toolkit's command line tool.
//
// It does one thing today.
//
//	mizu version          print the version, one fact per line
//	mizu version -json    print the same facts as JSON
//
// The rest of the command tree arrives with the generators. What is here now
// is the shape the rest hangs off: a run function that takes its arguments
// and its output streams, so every command is testable without a subprocess,
// and a dispatch table that fits on a screen.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "mizu:", err)
		os.Exit(1)
	}
}

// A command is one subcommand. Args are the ones after the name.
type command struct {
	name    string
	summary string
	run     func(args []string, stdout, stderr io.Writer) error
}

var commands = []command{
	{"version", "print version information", runVersion},
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		usage(stdout)
		return nil
	}

	name := args[0]
	if name == "help" || name == "-h" || name == "--help" {
		usage(stdout)
		return nil
	}

	i := slices.IndexFunc(commands, func(c command) bool { return c.name == name })
	if i < 0 {
		return fmt.Errorf("unknown command %q, run mizu help for the list", name)
	}
	return commands[i].run(args[1:], stdout, stderr)
}

func usage(w io.Writer) {
	var b strings.Builder
	b.WriteString("mizu is the command line tool for the mizu toolkit.\n\n")
	b.WriteString("Usage:\n\n\tmizu <command> [arguments]\n\nCommands:\n\n")
	for _, c := range commands {
		fmt.Fprintf(&b, "\t%-10s %s\n", c.name, c.summary)
	}
	b.WriteString("\nRun mizu <command> -h for the arguments a command takes.\n")
	io.WriteString(w, b.String())
}
