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
// Prompts, progress bars, spinners, sections and trees are specified and not
// written. So are the command structs and the flag generator that will call all
// of this, and the test fixture that scripts an answer to a prompt. What is
// here is the part they all sit on: the streams, and what is known about the
// other end of them.
package console
