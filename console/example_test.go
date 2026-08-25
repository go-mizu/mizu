package console_test

import (
	"context"
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

// A section indents what the command says about itself. It does not indent the
// answer, so a command that groups its status messages is still the left-hand
// side of a pipe.
func ExampleIO_Section() {
	io := console.New(strings.NewReader(""), os.Stdout, os.Stdout, console.Options{Color: console.ColorNever})

	io.Info("checking the project")
	config := io.Section("Config")
	config.Success("loaded config/app.go")
	config.Warn("APP_KEY is not set")
	io.Info("done")

	// Output:
	// checking the project
	// Config
	//   loaded config/app.go
	//   warning: APP_KEY is not set
	// done
}

// A tree is data, so it goes to stdout and turns into JSON under --json.
func ExampleIO_Tree() {
	io := console.New(strings.NewReader(""), os.Stdout, os.Stdout, console.Options{})

	io.Tree(console.TreeNode{
		Label: "app",
		Children: []console.TreeNode{
			{Label: "cmd", Children: []console.TreeNode{{Label: "main.go"}}},
			{Label: "go.mod"},
		},
	})

	// Output:
	// app
	// ├── cmd
	// │   └── main.go
	// └── go.mod
}

// Flags and arguments are declared next to the fields they parse into. The
// command line here is what somebody typed after the command name.
func ExampleParse() {
	var (
		days   int
		dryRun bool
		tenant string
	)

	flags := []console.Flag{
		{Name: "days", Short: 'd', Default: "30", Desc: "Delete accounts older than this", Value: console.Int(&days)},
		{Name: "dry-run", Desc: "Report without deleting", Value: console.Bool(&dryRun)},
	}
	args := []console.Arg{
		{Name: "tenant", Required: true, Desc: "Tenant slug, or all", Value: console.String(&tenant)},
	}

	if err := console.Parse(flags, args, []string{"-d", "7", "--dry-run", "acme"}); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(days, dryRun, tenant)

	// Output:
	// 7 true acme
}

// A command line that could not be understood is a [console.UsageError], which
// is what a command exits 2 for. The message names the flag, and the nearest
// one that exists, because a typo is the usual reason to be reading it.
func ExampleParse_usage() {
	var days int
	flags := []console.Flag{{Name: "days", Value: console.Int(&days)}}

	fmt.Println(console.Parse(flags, nil, []string{"--dayz=7"}))
	fmt.Println(console.Parse(flags, nil, []string{"--days=soon"}))

	// Output:
	// unknown flag --dayz, did you mean --days
	// --days: "soon" is not a number
}

// greet is a command: fields, a spec that points the flags at them, and a Run
// that reads them as ordinary Go values.
type greet struct {
	loud bool
	name string
}

func (c *greet) Spec() console.Spec {
	return console.Spec{
		Name: "greet",
		Desc: "Say hello to somebody",
		Flags: []console.Flag{
			{Name: "loud", Desc: "Shout it", Value: console.Bool(&c.loud)},
		},
		Args: []console.Arg{
			{Name: "name", Required: true, Desc: "Who to greet", Value: console.String(&c.name)},
		},
	}
}

func (c *greet) Run(ctx context.Context, io *console.IO) error {
	hello := "hello, " + c.name
	if c.loud {
		hello = strings.ToUpper(hello)
	}
	io.Line(hello)
	return nil
}

func ExampleApp() {
	app := &console.App{Name: "hello", Desc: "hello greets people."}
	app.Add(&greet{})

	io := console.New(strings.NewReader(""), os.Stdout, os.Stdout, console.Options{Color: console.ColorNever})
	if err := app.Run(context.Background(), io, []string{"greet", "--loud", "Ada"}); err != nil {
		io.Error("%v", err)
	}

	// Output:
	// HELLO, ADA
}

// Asking what a command takes is not a failure, so help goes to stdout and the
// command does not run. The required argument is missing here, which is the
// usual reason to be asking.
func ExampleApp_help() {
	app := &console.App{Name: "hello", Desc: "hello greets people."}
	app.Add(&greet{})

	io := console.New(strings.NewReader(""), os.Stdout, os.Stdout, console.Options{Color: console.ColorNever})
	if err := app.Run(context.Background(), io, []string{"help", "greet"}); err != nil {
		io.Error("%v", err)
	}

	// Output:
	// Say hello to somebody
	//
	// Usage:
	//   hello greet [flags] <name>
	//
	// Arguments:
	//   name  Who to greet
	//
	// Flags:
	//       --loud  Shout it
	//   -h, --help  Show what this command takes
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
