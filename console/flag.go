package console

import (
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-mizu/mizu/errs/diag"
)

// A Flag is one option a command takes.
//
// The zero Short means the flag has no single letter form. Name is the long
// form without its dashes, and it is required: a flag with only a letter is a
// flag nobody can read in a script six months later.
type Flag struct {
	Name  string
	Short rune
	Desc  string

	// Env is an environment variable consulted when the flag is absent from
	// the command line. The flag wins, then the variable, then Default.
	Env string

	// Default is applied when neither the command line nor Env supplied a
	// value. It is a string because that is what help text prints and what a
	// struct tag holds, and it goes through Value.Set like anything else, so a
	// default that does not parse is caught the first time the command runs.
	Default string

	// Required makes the absence of the flag an error. A required flag with a
	// default is a contradiction, and is reported as one.
	Required bool

	// Hidden keeps the flag out of help text. It still parses. This is for a
	// flag kept for compatibility, or one that only makes sense to whoever
	// wrote the build.
	Hidden bool

	// Value is where the flag parses into. See [Var] and the constructors
	// around it.
	Value Value
}

// An Arg is one positional argument.
type Arg struct {
	Name string
	Desc string

	// Required makes the absence of the argument an error naming it, rather
	// than a command that runs against the zero value and does something
	// surprising with it.
	Required bool

	// Rest makes this argument take everything left over. It has to be the
	// last one, and its Value has to be one that appends, which is [Slice].
	Rest bool

	// Default is applied when the argument was not given. See [Flag.Default].
	Default string

	Value Value
}

// A UsageError is a command line that could not be understood.
//
// It is the difference between the command failing and the command never
// having started, which is why it exits 2 and prints usage rather than a
// stack. Everything a command returns from Run is the other kind.
type UsageError struct {
	msg string
}

func (e *UsageError) Error() string { return e.msg }

func usagef(format string, a ...any) *UsageError {
	return &UsageError{msg: fmt.Sprintf(format, a...)}
}

// Parse fills the values behind flags and args from a command line.
//
// The command line is what is left after the command name, so for
// "mizu users:prune --days 7 acme" it is ["--days", "7", "acme"].
//
// Flags may come before, after, or between the positional arguments, because
// that is where people put them. Everything after a bare -- is positional
// whatever it looks like, and a bare - is positional as well, since that is how
// a program is asked to read from stdin.
//
// Every error it returns is a [UsageError]. A mistake in the definition, a flag
// with no name or two flags with the same letter, is a panic instead: that is a
// bug in the program rather than in what somebody typed.
func Parse(flags []Flag, args []Arg, argv []string) error {
	return parse(flags, args, argv, os.Getenv)
}

func parse(flags []Flag, args []Arg, argv []string, getenv func(string) string) error {
	validate(flags, args)

	seen := make([]bool, len(flags))
	var rest []string

	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		switch {
		case arg == "--":
			rest = append(rest, argv[i+1:]...)
			i = len(argv)

		case strings.HasPrefix(arg, "--"):
			used, err := long(flags, seen, arg[2:], argv[i+1:])
			if err != nil {
				return err
			}
			i += used

		case len(arg) > 1 && arg[0] == '-' && !negative(arg):
			used, err := short(flags, seen, arg[1:], argv[i+1:])
			if err != nil {
				return err
			}
			i += used

		default:
			rest = append(rest, arg)
		}
	}

	if err := fill(flags, seen, getenv); err != nil {
		return err
	}
	return positional(args, rest)
}

// negative reports whether a token that starts with a dash is a number.
//
// -5 is an argument to a command that takes an offset, not a flag called 5.
// Nothing else about the parse depends on what the letters mean, and this is
// the one case where reading them saves an error message that would make no
// sense to whoever typed it.
func negative(arg string) bool {
	r, _ := utf8.DecodeRuneInString(arg[1:])
	return unicode.IsDigit(r) || r == '.'
}

// long handles one --name token. It returns how many of the tokens after it
// were consumed.
func long(flags []Flag, seen []bool, token string, next []string) (int, error) {
	name, val, hasVal := strings.Cut(token, "=")
	if name == "" {
		return 0, usagef("--%s is not a flag name", token)
	}

	i := find(flags, func(f Flag) bool { return f.Name == name })
	if i < 0 {
		// --no-dry-run is how a boolean flag is turned off, and it only means
		// that when there is a boolean flag to turn off.
		if off, ok := strings.CutPrefix(name, "no-"); ok {
			if j := find(flags, func(f Flag) bool { return f.Name == off && isBool(f.Value) }); j >= 0 {
				if hasVal {
					return 0, usagef("--no-%s takes no value", off)
				}
				seen[j] = true
				return 0, set(flags[j], "--no-"+off, "false")
			}
		}
		return 0, unknown(flags, "--"+name, name)
	}

	f := flags[i]
	seen[i] = true
	switch {
	case hasVal:
	case isBool(f.Value):
		if c, ok := f.Value.(counter); ok {
			c.Count()
			return 0, nil
		}
		val = "true"
	case len(next) == 0:
		return 0, usagef("--%s needs a value", name)
	default:
		val = next[0]
		return 1, set(f, "--"+name, val)
	}
	return 0, set(f, "--"+name, val)
}

// short handles one -abc token, where a and b are flags that take no value and
// c may take the rest of the token or the one after it.
func short(flags []Flag, seen []bool, token string, next []string) (int, error) {
	for i, r := range token {
		j := find(flags, func(f Flag) bool { return f.Short == r })
		if j < 0 {
			return 0, unknown(flags, "-"+string(r), "")
		}

		f := flags[j]
		seen[j] = true
		tail := token[i+utf8.RuneLen(r):]

		if isBool(f.Value) && !strings.HasPrefix(tail, "=") {
			if c, ok := f.Value.(counter); ok {
				c.Count()
			} else if err := set(f, "-"+string(r), "true"); err != nil {
				return 0, err
			}
			continue
		}

		// Whatever is left of the token is the value, with or without an
		// equals sign, so -n5 and -n=5 and -n 5 all say five.
		if val, ok := strings.CutPrefix(tail, "="); ok {
			return 0, set(f, "-"+string(r), val)
		}
		if tail != "" {
			return 0, set(f, "-"+string(r), tail)
		}
		if len(next) == 0 {
			return 0, usagef("-%s needs a value", string(r))
		}
		return 1, set(f, "-"+string(r), next[0])
	}
	return 0, nil
}

// set parses one value into a flag, naming the flag when it does not.
func set(f Flag, as, val string) error {
	if err := f.Value.Set(val); err != nil {
		return usagef("%s: %v", as, err)
	}
	return nil
}

// fill applies the environment and the defaults to the flags that were not
// given, and reports the required ones that are still empty.
func fill(flags []Flag, seen []bool, getenv func(string) string) error {
	for i, f := range flags {
		if seen[i] {
			continue
		}
		if f.Env != "" {
			if val := getenv(f.Env); val != "" {
				if err := f.Value.Set(val); err != nil {
					return usagef("%s: %v", f.Env, err)
				}
				continue
			}
		}
		if f.Default != "" {
			if err := f.Value.Set(f.Default); err != nil {
				// The default is written by whoever declared the flag, so this
				// is their mistake and not the caller's. It is still an error
				// rather than a panic, because a generated default can come
				// from a struct tag in a file nobody has run yet.
				return usagef("the default for --%s does not parse: %v", f.Name, err)
			}
			continue
		}
		if f.Required {
			return usagef("--%s is required%s", f.Name, because(f.Desc))
		}
	}
	return nil
}

// positional hands the leftover words to the arguments in order.
func positional(args []Arg, rest []string) error {
	for i, a := range args {
		if a.Rest {
			words := rest[min(i, len(rest)):]
			for _, word := range words {
				if err := a.Value.Set(word); err != nil {
					return usagef("%s: %v", a.Name, err)
				}
			}
			if len(words) > 0 {
				return nil
			}
			return missing(a)
		}
		if i < len(rest) {
			if err := a.Value.Set(rest[i]); err != nil {
				return usagef("%s: %v", a.Name, err)
			}
			continue
		}
		if err := missing(a); err != nil {
			return err
		}
	}

	if len(rest) > len(args) {
		return usagef("unexpected argument %s%s", quote(rest[len(args)]), takes(args))
	}
	return nil
}

// missing applies an argument's default, or says it was needed.
func missing(a Arg) error {
	if a.Default != "" {
		if err := a.Value.Set(a.Default); err != nil {
			return usagef("the default for %s does not parse: %v", a.Name, err)
		}
		return nil
	}
	if a.Required {
		return usagef("%s is required%s", a.Name, because(a.Desc))
	}
	return nil
}

// because turns a description into the second half of a sentence, so that the
// error says what the missing thing is for rather than only its name.
func because(desc string) string {
	if desc == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(desc)
	return ": " + string(unicode.ToLower(r)) + desc[n:]
}

// takes says how many arguments there were room for.
func takes(args []Arg) string {
	switch len(args) {
	case 0:
		return ", this command takes none"
	case 1:
		return ", this command takes one"
	default:
		return fmt.Sprintf(", this command takes %d", len(args))
	}
}

func quote(s string) string { return `"` + s + `"` }

// unknown reports a flag nobody declared, with the nearest one that was.
func unknown(flags []Flag, as, name string) error {
	if did := diag.Did(nearest(flags, name), dashed); did != "" {
		return usagef("unknown flag %s, %s", as, did)
	}
	return usagef("unknown flag %s", as)
}

// dashed is a flag name as it is written on a command line.
func dashed(name string) string { return "--" + name }

// nearest returns the visible flags worth offering for name.
func nearest(flags []Flag, name string) []string {
	return diag.Suggest(name, func(yield func(string) bool) {
		for _, f := range flags {
			if f.Hidden {
				continue
			}
			if !yield(f.Name) {
				return
			}
		}
	})
}

// isBool reports whether a value takes no argument on the command line.
func isBool(v Value) bool {
	b, ok := v.(boolean)
	return ok && b.IsBoolFlag()
}

// find is slices.IndexFunc, spelled out here because the flag lists are short
// enough that the shape of the loop is the whole story.
func find(flags []Flag, match func(Flag) bool) int {
	for i, f := range flags {
		if match(f) {
			return i
		}
	}
	return -1
}

// validate catches the mistakes that belong to whoever declared the command.
//
// They panic rather than return, because a duplicate flag letter is not
// something the person running the command can do anything about, and a
// command that ships with one is broken for everybody. The generator makes
// most of these impossible; a hand-written command is where they happen.
func validate(flags []Flag, args []Arg) {
	names := make(map[string]bool, len(flags))
	shorts := make(map[rune]bool, len(flags))
	for _, f := range flags {
		switch {
		case f.Name == "":
			panic("console: a flag has no name")
		case f.Value == nil:
			panic("console: flag --" + f.Name + " has no value to parse into")
		case names[f.Name]:
			panic("console: flag --" + f.Name + " is declared twice")
		case f.Short != 0 && shorts[f.Short]:
			panic("console: flag -" + string(f.Short) + " is declared twice")
		case f.Required && f.Default != "":
			panic("console: flag --" + f.Name + " is required and has a default")
		}
		names[f.Name] = true
		if f.Short != 0 {
			shorts[f.Short] = true
		}
	}

	for i, a := range args {
		switch {
		case a.Name == "":
			panic("console: an argument has no name")
		case a.Value == nil:
			panic("console: argument " + a.Name + " has no value to parse into")
		case a.Rest && i != len(args)-1:
			panic("console: argument " + a.Name + " takes the rest and is not last")
		}
	}
}
