// Package commandgen writes the Spec method for a command line command. It is
// an implementation detail of the mizu command and is exempt from the
// compatibility promise in doc 31. Import it only if you are extending mizu
// itself.
//
// A command is a struct that says what it takes and does it. The taking part is
// tags on the fields, and the doing part is a Run method:
//
//	//mizu:command name=users:prune
//	type UsersPrune struct {
//		Tenant string        `arg:"0" desc:"which tenant to prune"`
//		Days   int           `flag:"days,d" default:"30" desc:"how long unverified is too long"`
//		DryRun bool          `flag:"" desc:"say what would go and delete nothing"`
//		Wait   time.Duration `flag:"" env:"PRUNE_WAIT" default:"5s"`
//	}
//
//	func (c *UsersPrune) Run(ctx context.Context, io *console.IO) error {
//		...
//	}
//
// [Generate] walks the package and writes commands_gen.go next to it, holding
// one Spec method per command and a Commands function that lists them:
//
//	app.Add(commands.Commands()...)
//
// The generated Spec points console.Value constructors at the struct's own
// fields, so Run reads Days as an int rather than looking a flag up by name,
// and a field nothing can parse is a build failure rather than a surprise at a
// terminal.
//
// # Why a generator
//
// Writing the Spec by hand is a list of flags beside a list of fields, and the
// two drift. The usual answer is reflection at startup, which turns a typo in a
// tag into a panic on the first run, in front of whoever ran it. Reading the
// tags at build time puts the same mistake in a compiler message, and leaves
// the running program with plain field access and no reflection at all.
//
// The Spec stays a public type that can be written by hand, because a command
// whose flags depend on something read at startup cannot come from tags. The
// generator writes the ordinary case and gets out of the way of the rest.
//
// # Tags
//
//	flag:"name,short"   a flag, as in --days or -d; an empty name comes from the field
//	arg:"0"             a positional argument, and its place on the line
//	arg:"1..."          an argument that takes the rest of the line
//	desc:"..."          the one line of help, when the doc comment is not it
//	default:"v"         the value when it was not given
//	env:"VAR"           the environment variable to fall back to
//	required:"true"     a flag that has to be given
//	required:"false"    an argument that does not
//	hidden:"true"       leave it out of the help
//	enum:"a|b|c"        the only values it takes
//	sep:";"             what separates the values of a list, and "" for none
//	count:"true"        an int that counts how many times it was given, as in -vv
//
// An empty flag name comes from the field in kebab case, so DryRun is --dry-run
// and MaxOpenConns is --max-open-conns. An argument's name comes from the field
// the same way, since a positional argument is named for the help text rather
// than for typing.
//
// A field with no flag and no arg tag is not part of the command line, which is
// how a command holds a field of its own. A field carrying one of the other
// tags without either of those is reported, because nothing would read it.
//
// A validate tag is read by the validate package rather than this one, and is
// left alone here.
//
// # What a field says without any tags
//
// The doc comment on a field is its help text, so the explanation lives beside
// the field it is about and there is one place to change it:
//
//	// How long unverified is too long.
//	Days int `flag:"days,d" default:"30"`
//
// The first sentence is what help has room for. A trailing full stop is
// dropped, since help lists read as labels rather than as prose. A desc tag
// wins when there is one. The same holds for the marker on the struct: its desc
// argument wins, and the struct's doc comment is used when it has none.
//
// # Types
//
// A field is read by the constructor its type matches, and a type nothing
// matches is an error naming the field:
//
//	string, bool, int, uint and float of every width, and any defined type over one of them
//	time.Duration and time.Time
//	a slice of any of the above, which collects rather than replaces
//	map[string]string, given as --label k=v and repeated
//	anything with an UnmarshalText method, which is an encoding.TextUnmarshaler
//
// A slice splits on a comma as well as collecting, so --tag a,b and --tag a
// --tag b mean the same thing. A sep tag changes the separator and sep:"" turns
// the splitting off, for values that have commas in them.
//
// A list of anything but strings parses each element with the exported parser
// for its type, so --port 80,443 into a []uint16 reports a bad number the same
// way a single --port would.
//
// # Rules the generator checks
//
// Arguments are numbered from zero with no gaps, an argument taking the rest of
// the line is last, and a required argument does not follow an optional one.
// All three are the kind of mistake that produces a command nobody can run
// rather than one that fails to build.
//
// An argument is required unless it has a default or says required:"false",
// because writing one on the line is what a positional argument is for. A flag
// is the other way around.
//
// A flag that is required and has a default is reported here, since
// console.Parse panics on the pair: a flag with a default is never missing.
//
// A struct that already has a Spec method is reported rather than overwritten,
// and so is one whose Run has the wrong shape, since half an interface is a
// build failure whose message is about the interface rather than about the
// command.
//
// # Names
//
// The marker carries the name, and a marker without one is an error that
// suggests the likely spelling. A command name is what somebody types, and
// UsersPrune could be users:prune or user:sprune. Guessing wrongly here means a
// command whose name nobody can predict, which is worth one argument on the
// marker to avoid.
//
//	//mizu:command name=db:wipe desc="Drop every table" hidden
//
// desc is the one line of help, long is the paragraph under it, and hidden
// keeps the command out of the listing while leaving it runnable.
package commandgen
