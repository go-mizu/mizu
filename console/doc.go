// Package console is what a command line program says and how it says it.
//
// An [IO] is the three streams a command has, plus what is known about the
// other end of them. Commands take one instead of writing to os.Stdout, which
// is what makes them testable without a subprocess and what makes the rules
// below hold everywhere rather than in the commands somebody remembered.
//
// # Data goes to stdout, everything else goes to stderr
//
// [IO.Print], [IO.Line], [IO.Table] and [IO.JSON] write to stdout. They are the
// answer to what was asked.
//
// [IO.Info], [IO.Success], [IO.Warn], [IO.Error] and [IO.Debug] write to
// stderr. They are the program talking about itself.
//
// The split is what makes a pipeline work. A command can report progress, warn
// about a deprecated flag and still be the left-hand side of a pipe, because
// none of that lands in the data. Getting this wrong is the most common way a
// command line tool becomes unusable in a script, and it is usually found by
// somebody whose parser broke on a status message.
//
// # Colour
//
// Colour is on when the stream is a terminal, and off otherwise. NO_COLOR with
// any value turns it off, and so does TERM=dumb. [ColorAlways] and [ColorNever]
// override all of that, because a user who passed a flag has already decided.
//
// stdout and stderr are decided separately, so redirecting one of them does not
// change the other.
//
// # How much a command says
//
// [Verbosity] runs from [Quiet] to [Trace]. Quiet keeps warnings and errors and
// drops the rest, because a warning nobody sees is the reason the flag gets
// blamed later. Debug lines appear from [Verbose] up.
//
// # Flags and arguments
//
// [Parse] fills a command's fields from a command line. A [Flag] and an [Arg]
// each name a [Value], which is the interface from the standard library's flag
// package, so anything already written against that works here.
//
// The forms it takes are the ones people type: --days 7 and --days=7 and -d7,
// -abc for three flags that take no argument, --no-dry-run for the opposite of
// a boolean, -vv for a count, -- for the end of the flags, and a flag before,
// after or between the positional arguments. A bare - and a negative number are
// arguments rather than flags, since that is what somebody meant by them.
//
// A flag that was not given takes its environment variable, then its default,
// and is an error if it was required and there was neither. All of those errors
// are a [UsageError], which is the difference between a command that failed and
// a command that never started, and it is what the caller exits 2 for.
//
// A mistake in the declaration is a panic instead: two flags with the same
// letter is a bug in the program, and nobody typing at a terminal can do
// anything about it.
//
// [ParseInt], [ParseUint], [ParseFloat], [ParseDuration], [ParseTime] and
// [ParseString] are what those constructors parse with, exported so that a list
// can have the same parsing and the same messages:
//
//	console.Slice(&c.Ports, console.ParseUint, ",")
//
// # Commands
//
// A [Command] is a struct with two methods: [Spec] says what it is called and
// what it takes, Run does it. The flags in the spec point at the struct's own
// fields, so Run reads them as ordinary Go values rather than looking anything
// up by name.
//
// An [App] holds a set of them and [App.Run] finds the one that was asked for.
// A colon in a name groups related commands, so db:seed and db:wipe appear
// together in help without either of them saying so.
//
// Asking for help is not a failure. --help, -h, help, help <command> and a bare
// command line write to stdout and return nil, so the answer can be piped into
// a pager or grepped for the command somebody half remembers. A command line
// that could not be understood is a [UsageError] instead, and an unknown
// command names the nearest one that exists.
//
// Help for a command is printed without parsing its arguments, because "what
// does this take" is the question somebody asks when they do not have them yet.
//
// # Running a command as a process
//
// [App.Main] is the whole of a main function. It returns the code to exit with:
//
//	func main() {
//		os.Exit(app.Main(os.Args[1:]))
//	}
//
// [Globals] are the flags every command takes: --verbose, --quiet, --json,
// --color, --no-color, --no-interaction and --timeout. They are taken out of
// the command line wherever they were written, so --json means the same thing
// before the command name and after it, and a command only ever sees its own
// flags. A program adds the ones that belong to it, such as --env, with
// [App.Globals].
//
// The exit codes are the ones from sysexits.h, so a shell script and a process
// supervisor already know how to read them: [CodeUsage] for a command line that
// could not be understood, [CodeConfig] for a configuration that does not make
// sense, [CodeUnavailable] for something the command depends on not being
// there, and [CodeFailure] for everything else. A command picks one by wrapping
// its error with [Exit], and an error type that classifies itself gets there by
// implementing [ExitCoder].
//
// SIGINT and SIGTERM cancel the command's context, so a command that honours it
// closes what it opened. A second signal exits at once, because somebody
// pressing Ctrl-C twice has stopped asking. [App.Start] is the same thing
// without the process, for a test: it takes the streams and a context and
// returns the code rather than reaching for os.
//
// The error a command returns is printed as one line. Its chain of causes is
// printed under it from --verbose up, because the answer is usually three wraps
// down and that is worth a flag rather than four lines on every failure.
//
// # Asking questions
//
// [IO.Ask], [IO.AskSecret], [IO.Confirm], [IO.Choice] and [IO.MultiChoice] read
// from stdin and write to stderr, so a command can ask something and still be
// the left-hand side of a pipe.
//
// They ask when stdin and stderr are both terminals, and --no-interaction turns
// that off. When they cannot ask, a prompt with a default takes it and a prompt
// without one is an error naming the question. That last rule is the one worth
// keeping: a command that reaches a question nobody can answer stops with a
// sentence about the missing value rather than holding a CI runner until it
// times out.
//
// Every prompt returns an error, which the same function in most CLI libraries
// does not. A question that could not be asked has no answer, and returning the
// zero value for one is how a command ends up deleting three users because
// nobody was there to say no.
//
// # Saying that something is happening
//
// [IO.Progress] is a bar for work whose size is known, [IO.Spinner] is for work
// whose size is not, and [IO.Task] is a spinner around one function call with a
// mark at the end saying how it went.
//
// All three are drawn on a terminal and written as lines anywhere else. The bar
// writes a line every ten percent, which is eleven lines for a job of any
// length, and the spinner writes one every thirty seconds. That is what keeps a
// CI log readable: the usual failure is a bar redrawing into a file, which
// leaves a megabyte of escape sequences on one line that nothing can read.
//
// A [Bar] is safe to use from several goroutines, which the rest of this
// package is not, because a worker pool reporting its own progress is the
// normal way to end up with one.
//
// # Grouping what is on the screen
//
// [IO.Section] prints a title and returns an IO whose status output is indented
// under it. Sections nest, and there is nothing to close, so an early return
// cannot leave one open.
//
// The indent applies to stderr only. [IO.Tree], like [IO.Table], is data and
// goes to stdout at the left margin wherever it was printed, because the shape
// of a command's output should not depend on where in the command it was
// written.
//
// # JSON
//
// An IO in JSON mode writes machine readable output and no decoration.
// [IO.Table] becomes an array of objects instead of columns, so a command that
// prints a list supports --json without writing the list twice. Warnings and
// errors still go to stderr, where they cannot corrupt what a parser is
// reading.
//
// # What is not here yet
//
// The generator that turns struct tags into a Spec is specified and not
// written, and neither is the test fixture that scripts an answer to a prompt.
//
// The prompts that are here read a line. There is no arrow key selection and no
// history, and a list is numbered instead. A number is something a person can
// read out to somebody else, it survives a terminal that reports key presses
// differently, and it is the difference between a prompt that works over a
// serial console or in a Docker exec and one that does not. A full screen
// selector is worth adding when something needs it, over the top of these
// rather than instead of them.
package console
