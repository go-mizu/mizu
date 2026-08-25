package console

import (
	"context"
	"fmt"
	"strings"
)

// A Command is one thing a program can be asked to do.
//
// The pair of methods is the whole interface: Spec says what the command is
// called and what it takes, Run does it. A command holds its own fields, the
// spec points the flags at them, and Run reads them as ordinary Go values
// rather than looking anything up by name.
//
//	type UsersPrune struct {
//		Days   int
//		DryRun bool
//		Tenant string
//	}
//
//	func (c *UsersPrune) Spec() console.Spec {
//		return console.Spec{
//			Name: "users:prune",
//			Desc: "Delete users who never verified their email",
//			Flags: []console.Flag{
//				{Name: "days", Short: 'd', Default: "30", Value: console.Int(&c.Days)},
//			},
//			Args: []console.Arg{
//				{Name: "tenant", Required: true, Value: console.String(&c.Tenant)},
//			},
//		}
//	}
//
//	func (c *UsersPrune) Run(ctx context.Context, io *console.IO) error {
//		...
//	}
//
// The struct tags in the specification are a shorter way of writing that Spec
// method, and the generator emits exactly this. Nothing about running a command
// knows which of the two it came from.
type Command interface {
	Spec() Spec
	Run(ctx context.Context, io *IO) error
}

// A Spec is what a command is called and what it takes.
type Spec struct {
	// Name is what somebody types. A colon groups related commands, so
	// db:seed and db:wipe appear together in help, and the grouping is the
	// name rather than a field that can disagree with it.
	Name string

	// Desc is one line, printed next to the name in the command list. It reads
	// as an answer to what the command does: "Delete users who never verified
	// their email".
	Desc string

	// Long is the paragraphs under it in the command's own help, for what
	// somebody needs to know before running it rather than while reading the
	// list.
	Long string

	// Hidden keeps the command out of help. It still runs, which is what a
	// command kept for compatibility or written for the build needs.
	Hidden bool

	Flags []Flag
	Args  []Arg
}

// An App is a set of commands and the name they are run under.
//
// The zero value works. Name is what appears in help and in error messages, so
// it should be the name of the binary.
type App struct {
	Name    string
	Desc    string
	Version string

	// Globals are flags every command takes, on top of the ones in [Globals]
	// that every mizu command has. They are for what belongs to the program
	// rather than to a command: which environment, which config file.
	//
	// [App.Start] parses them and takes them out of the command line, so a
	// command never sees one and a command must not declare a flag with the
	// same name.
	Globals []Flag

	cmds []entry

	// globals is what Start parsed, kept so that help can list them. Help
	// printed by [App.Run] on its own does not, because a program that runs a
	// command without Start has not agreed to any of these.
	globals []Flag
}

// entry is a command and the spec it gave when it was registered.
//
// Spec is asked for once rather than per call, because a command that builds
// its flags in that method would otherwise hand out a second set of them that
// point at the same fields, and only one of the two would have been parsed.
type entry struct {
	spec Spec
	cmd  Command
}

// Add registers commands.
//
// It panics on a command that cannot work: no name, a name already taken, or a
// flag list with one of the mistakes [Parse] panics on. All of those belong to
// whoever wrote the program, and finding them when the binary starts is better
// than finding them when somebody runs the one command that has one.
func (a *App) Add(cmds ...Command) {
	for _, cmd := range cmds {
		spec := cmd.Spec()
		switch {
		case spec.Name == "":
			panic("console: a command has no name")
		case strings.ContainsAny(spec.Name, " \t"):
			panic("console: command name " + quote(spec.Name) + " has a space in it")
		case a.lookup(spec.Name) != nil:
			panic("console: command " + spec.Name + " is registered twice")
		}
		validate(spec.Flags, spec.Args)
		a.cmds = append(a.cmds, entry{spec: spec, cmd: cmd})
	}
}

// Run finds the command named by argv and runs it.
//
// argv is what follows the program name, so for "mizu users:prune --days 7" it
// is ["users:prune", "--days", "7"].
//
// Asking for help is not a failure: --help, -h, help, help <command> and a bare
// command line all write to stdout and return nil. Everything else that is
// wrong with the command line is a [UsageError], which the caller exits 2 for.
// An error from the command itself is returned as it is.
func (a *App) Run(ctx context.Context, c *IO, argv []string) error {
	if len(argv) == 0 {
		a.Help(c)
		return nil
	}

	name, rest := argv[0], argv[1:]
	switch {
	case name == "--help" || name == "-h" || (name == "help" && len(rest) == 0):
		a.Help(c)
		return nil

	case name == "--version" && a.Version != "":
		fmt.Fprintf(c.out, "%s %s\n", a.Name, a.Version)
		return nil

	case name == "help":
		if len(rest) > 1 {
			return usagef("help takes one command")
		}
		name, rest = rest[0], nil
		if e := a.lookup(name); e != nil {
			a.usage(c, e.spec)
			return nil
		}
		return a.unknown(name)

	case strings.HasPrefix(name, "-"):
		return usagef("unknown flag %s, run %q for what this takes", name, a.help())
	}

	e := a.lookup(name)
	if e == nil {
		return a.unknown(name)
	}
	if wantsHelp(e.spec, rest) {
		a.usage(c, e.spec)
		return nil
	}
	if err := Parse(e.spec.Flags, e.spec.Args, rest); err != nil {
		return err
	}
	return e.cmd.Run(ctx, c)
}

// lookup finds a command by name.
//
// It walks the list. A program with a thousand commands does this once, and the
// list is needed whole for help and for the suggestion on a name that is not
// here, so there is nothing for a map to save.
func (a *App) lookup(name string) *entry {
	for i, e := range a.cmds {
		if e.spec.Name == name {
			return &a.cmds[i]
		}
	}
	return nil
}

// unknown reports a command nobody registered, with the nearest one that
// somebody did.
func (a *App) unknown(name string) error {
	best, at := "", 3
	for _, e := range a.cmds {
		if e.spec.Hidden {
			continue
		}
		if d := distance(e.spec.Name, name); d < at {
			best, at = e.spec.Name, d
		}
	}
	if best != "" {
		return usagef("unknown command %s, did you mean %s", quote(name), best)
	}
	return usagef("unknown command %s, run %q for the list", quote(name), a.help())
}

// help is what to tell somebody to type. The name is empty in a test and in a
// program that never set one, and "run \"help\"" is still the right advice.
func (a *App) help() string {
	return strings.TrimSpace(a.Name + " help")
}

// wantsHelp reports whether the command line is asking what the command takes
// rather than asking it to run.
//
// A command that declares its own --help or -h keeps it. That is unlikely and
// it is not this package's business to forbid, and the alternative is a flag
// that parses everywhere except through the one function that dispatches.
func wantsHelp(spec Spec, argv []string) bool {
	long := find(spec.Flags, func(f Flag) bool { return f.Name == "help" }) < 0
	short := find(spec.Flags, func(f Flag) bool { return f.Short == 'h' }) < 0
	for _, arg := range argv {
		switch {
		case arg == "--":
			// Past this everything is an argument, and a command that takes a
			// file called --help should be able to say so.
			return false
		case long && arg == "--help", short && arg == "-h":
			return true
		}
	}
	return false
}
