package console_test

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-mizu/mizu/console"
)

// The zero Options are the ones a program wants: normal verbosity, output for
// a person, and colour decided by looking at the terminal.
func Example() {
	io := console.New(strings.NewReader(""), os.Stdout, os.Stdout, console.Options{Color: console.ColorNever})

	io.Info("looking for users who never verified their email")
	io.Table(
		[]string{"Name", "Email", "Posts"},
		[][]string{
			{"Ada Lovelace", "ada@example.com", "17"},
			{"Bo", "bo@example.com", "3"},
		},
		console.AlignRight(2),
	)
	io.Success("found 2")

	// Output:
	// looking for users who never verified their email
	// Name          Email            Posts
	// Ada Lovelace  ada@example.com     17
	// Bo            bo@example.com       3
	// found 2
}

// A table renders as JSON when the command was run with --json, so a command
// that prints a list supports both without building the list twice. The status
// messages are gone, because they are decoration and this output is going to a
// parser.
func ExampleIO_Table_json() {
	io := console.New(strings.NewReader(""), os.Stdout, os.Stdout, console.Options{JSON: true})

	io.Info("looking for users who never verified their email")
	io.Table(
		[]string{"Name", "Last seen"},
		[][]string{{"Ada Lovelace", "2026-08-01"}},
	)

	// Output:
	// [
	//   {"name": "Ada Lovelace", "last_seen": "2026-08-01"}
	// ]
}

// Data goes to stdout and everything else goes to stderr, so a command can
// talk about what it is doing and still be the left-hand side of a pipe. Here
// both streams are the same writer, which is what a terminal is.
func ExampleIO_Line() {
	io := console.New(strings.NewReader(""), os.Stdout, os.Stderr, console.Options{})

	io.Info("this goes to stderr and is not part of the answer")
	io.Line("ada@example.com")

	// Output:
	// ada@example.com
}

// A prompt guarding something destructive passes false, so that the answer
// nobody gives is the one that leaves the users alone.
//
// The answer here comes from a reader rather than from a person, so nothing
// echoes it. On a terminal the y the user typed would sit between the prompt
// and the line after it.
func ExampleIO_Confirm() {
	io := console.New(strings.NewReader("y\n"), os.Stdout, os.Stdout, console.Options{
		Color:       console.ColorNever,
		Interaction: console.InteractionAlways,
	})

	ok, err := io.Confirm("Delete 3 users?", false)
	if err != nil {
		return
	}
	if ok {
		io.Info("deleting")
	}

	// Output:
	// Delete 3 users? [y/N]: deleting
}

// A bar on a terminal redraws in place. Anywhere else, which includes every CI
// job, it is a line every ten percent: enough to see that something is
// happening, few enough that a job of a million steps is still ten lines.
func ExampleIO_Progress() {
	io := console.New(strings.NewReader(""), os.Stdout, os.Stdout, console.Options{})

	bar := io.Progress(5)
	for range 5 {
		bar.Advance(1)
	}
	bar.Done()

	// Output:
	// 20% (1/5)
	// 40% (2/5)
	// 60% (3/5)
	// 80% (4/5)
	// 100% (5/5)
}

// With no terminal, a prompt takes its default, and a prompt with no default is
// an error. That is the difference between a build that stops with a sentence
// about a missing value and one that holds a CI runner until it times out.
func ExampleIO_Ask_noInteraction() {
	io := console.New(strings.NewReader(""), os.Stdout, os.Stdout, console.Options{
		Interaction: console.InteractionNever,
	})

	name, err := io.Ask("Project name", "blog")
	fmt.Println(name, err)

	_, err = io.Ask("Database URL", "")
	fmt.Println(err)

	// Output:
	// blog <nil>
	// cannot ask "Database URL": there is no terminal and no default, so the value has to come from a flag
}
