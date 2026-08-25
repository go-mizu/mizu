/*
Command mizu is the command line tool for the mizu toolkit.

It writes projects, runs the generators, and answers the questions somebody has
about a project they have just opened: what is this, is my change finished, and
what do I need to know before I touch anything.

Install it with go install:

	go install github.com/go-mizu/mizu/cmd/mizu@latest

# The commands

	mizu new <dir>     write a project that builds, tests and runs
	mizu about         print what this project is made of
	mizu check         type check and vet, the fastest answer there is
	mizu verify        run everything that has to pass before a change is done
	mizu lint          report the mistakes the compiler cannot
	mizu gen           write what the markers in the project ask for
	mizu gen:agents    write AGENTS.md, what an agent reads before it edits
	mizu gen:bind      write the BindRequest methods for the //mizu:bind structs
	mizu gen:command   write the Spec methods for the //mizu:command structs
	mizu gen:config    write the decoder for the //mizu:config struct
	mizu doctor        check the project and the environment
	mizu hash:tune     measure argon2id here and print the cost to configure
	mizu version       print the version, one fact per line

A colon groups related commands, so gen and the gen:* commands appear together
in help. The rest of the tree arrives with the packages it drives.

# Writing a project

	mizu new blog --preset=api

The directory is the name of the project, so the last element of the path is
what the module is called and what the program prints about itself. It has to
be empty, apart from a .git, since starting the repository first and the
project second is an ordinary way to begin.

--preset says what to write. With no --preset and somebody at the terminal it
is a question rather than a guess, and with nobody to ask it is the first
preset, so a script that forgot the flag gets a project rather than a prompt it
cannot answer. Every preset writes the same files and differs in what is in
them.

What comes out builds, tests and runs before anything is added to it. It has an
AGENTS.md from the first commit, written by the same generator that maintains
it from then on.

# The loop

	mizu check     after an edit
	mizu verify    before saying a change is finished

verify runs seven stages in dependency order and stops at the first failure:
gen, fmt, vet, lint, build, test and doctor. A green run means a green CI, which
is the whole point of having one command rather than a list somebody has to
remember. check is the same command with the quick stages only, taken from the
same list, so passing check and failing verify is possible and the reverse is
not.

	mizu verify --fix

writes what can be written, which is the generated files and the formatting,
and then carries on with the rest. Those stages come back as fixed rather than
ok, so a run that changed the tree does not read as a run that had nothing to
do.

Each stage prints as it finishes. Under --json the whole run is one document,
skipped stages included, so a failure localises without reading any output.

# Generating

	mizu gen             write every generator's files
	mizu gen --check     report what is out of date and write nothing
	mizu gen ./app/...   only these packages

A generator is driven by a marker comment on a declaration, //mizu:bind,
//mizu:command or //mizu:config, and writes a file next to the one the marker
is in. Nothing is generated from a registry of types held somewhere else, so a
declaration and the code written from it move together.

	mizu gen:bind

writes a BindRequest method for every struct marked //mizu:bind. web.Bind calls
that method when the type has one and falls back to reflection when it does
not, so a marker is a change to the struct and to nothing else. What it buys is
the query string and the form body: a generated binder reads them a pair at a
time instead of asking net/http for a map of every name the request carries.

--check is what CI runs. It exits non-zero and names the first file that would
have changed, since a generated file that disagrees with its source is a build
that works on the machine where somebody last ran the generator.

Generated files carry a header saying so, and mizu refuses to overwrite a file
without one rather than taking a hand-written file for its own.

# Checking the rules a compiler has no opinion about

	mizu lint                  every check over ./...
	mizu lint --check=ctx      one check
	mizu lint ./app/...        only these packages

A mizu package sometimes makes a rule the type system cannot: a *web.Ctx comes
from a pool and stops being valid when the handler returns, so keeping one in a
field or handing one to a goroutine is wrong in a way that compiles. Each check
reads the types in a package and says where a rule like that was broken, with
the line quoted and a code to look up.

A check reports what it is sure of. Nothing follows a value through an
interface or an any, so a check missing something is expected and a check
inventing something is a bug. The other half of the rule is the guarded build,
which catches at run time what reading the source cannot.

verify runs this as a stage, so a rule broken in an editor is a rule somebody
hears about before CI does.

# Finding out about a project

	mizu about

prints the module and the toolchain, the toolkit packages the project imports
with a count of how many of its packages import each, and the files a generator
wrote, grouped by the generator. All of it is read off the project rather than
out of a file that says what the project is meant to be, because the two
disagree eventually and the one that is wrong is the one somebody wrote by
hand.

	mizu gen:agents

writes AGENTS.md at the module root, which is what an agent reads before it
edits anything: the project, its commands, its layout, and whatever somebody
wrote between the mizu:keep markers. It is regenerated by mizu gen along with
everything else.

	mizu doctor

checks the project and the environment and says what is wrong with it, one line
per check. --ci exits non-zero when anything is an error, which is the
difference between a check somebody reads and a check that gates a merge.

# Flags every command takes

	-v, --verbose      say more, twice for more again
	-q, --quiet        warnings and errors only
	    --json         machine readable output
	    --diag-file    also write diagnostics as JSON to this file
	    --color        auto, always or never
	    --no-color     never colour output
	-n, --no-interaction   never ask a question, take the defaults
	    --timeout      give up after this long
	    --profile      write a CPU profile here
	    --trace        write an execution trace here

They can be written before the command name or after it. --quiet beats
--verbose, because somebody who passed both meant the one that asks for less.

# Machine readable output

--json is true of every command, including the ones that have nothing to say
but whether they worked. Stdout is the answer the command was asked for, and a
failure prints a mizu.diag/1 document on stderr with the code, the message and
where it happened. A command's output under --json is one document, so a caller
reads it with a decoder rather than a regular expression.

--diag-file writes the same diagnostic to a file, for a CI job that wants the
human output on the console and the structured one for an annotation.

# Exit codes

From sysexits.h, which is what a shell script and a process supervisor already
know how to read.

	0     it did what it was asked
	1     something went wrong while it was doing it
	2     the command line could not be understood, so nothing ran
	69    something it depends on is not there
	70    a bug in mizu rather than in the command line
	77    a file, a socket or an API said no
	78    a configuration that does not make sense
	130   interrupted, or a question answered no

# How a command is put together

Every command is a struct with two methods, Spec and Run, both taking a
[console.IO] rather than reaching for the process. A command is tested by
calling it, with buffers for the streams and scripted answers to the questions
it asks, and there is no test in this package that starts a process.

The flags a command takes are the fields of that struct, and the [console.Spec]
that describes them is written by mizu gen:command from the //mizu:command
marker. This program's own commands are hand-written for now, since a generator
that cannot build the tool it ships in is a generator with a bootstrap problem
rather than a feature.
*/
package main
