package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/toml"
)

// A From says which kind of place a value came from.
type From int

const (
	FromDefault  From = iota // the default written on the struct field
	FromFile                 // a TOML file
	FromDotEnv               // a .env file
	FromEnv                  // the process environment
	FromFlag                 // the command line
	FromOverride             // set by the program itself
	FromComputed             // worked out at startup, such as from the number of CPUs
)

func (f From) String() string {
	switch f {
	case FromDefault:
		return "default"
	case FromFile:
		return "file"
	case FromDotEnv:
		return "dotenv"
	case FromEnv:
		return "env"
	case FromFlag:
		return "flag"
	case FromOverride:
		return "override"
	case FromComputed:
		return "computed"
	}
	return "unknown"
}

// A Source is where a value came from, in enough detail to go and change it.
type Source struct {
	From From

	// Name says which one, and what it says depends on From: a file and a
	// position for FromFile, a file and a line for FromDotEnv, a variable name
	// for FromEnv, the argument for FromFlag, and an explanation for
	// FromComputed. It is empty for the layers where there is only one place
	// it could have been.
	Name string
}

func (s Source) String() string {
	if s.Name == "" {
		return s.From.String()
	}
	return s.From.String() + " " + s.Name
}

// A Field is a setting a caller is asking for.
//
// Generated code fills one in from what is written on a struct field. Path is
// how a file names the setting and Env is how the environment names it, and
// either may be empty when that layer has no name for it.
type Field struct {
	// Name is what the field is called in Go, such as DB.MaxOpenConns. It is
	// what an error about the field leads with, because that is the name the
	// person reading the error will search the code for. Generated code fills
	// it in, and it may be empty, in which case Path stands in for it.
	Name string

	// Path is the dotted path in a TOML file, such as database.max_open_conns.
	Path string

	// Env is the environment variable, such as DATABASE_URL.
	Env string

	// Default is the value to use when no layer has one, as text. An empty
	// default is the same as no default, because the field is already at its
	// zero value when nothing sets it.
	Default string

	// Secret marks a value that must not be printed. It changes nothing about
	// how the value is found, only how [Setting.Display] shows it.
	Secret bool
}

// A Value is a setting as one layer wrote it.
//
// Exactly one of TOML and Text holds the value, and TOML being non-nil is what
// says which. A file keeps the type it was written with, and every other layer
// is text, because an environment variable and a command line argument have
// nothing else to be.
type Value struct {
	Source Source
	Text   string
	TOML   *toml.Value
}

// Str is the value as text, which is what a [Parser] written by hand almost
// always wants.
//
// Every layer except a file is text already, because an environment variable
// and a command line argument have nothing else to be. A file keeps the type
// the value was written with, so a number or a boolean there is not text, and
// this reports that rather than inventing a spelling for it.
func (v Value) Str() (string, error) {
	if v.TOML == nil {
		return v.Text, nil
	}
	if v.TOML.Kind != toml.KindString {
		return "", wantErr("a string", v)
	}
	return v.TOML.Str, nil
}

// Display is the value written out for a person to read, for config:show and
// config:diff. Strings are shown without quotes, because the reader wants the
// value and not the syntax.
func (v Value) Display() string {
	if v.TOML == nil {
		return v.Text
	}
	return display(v.TOML)
}

// Errorf is an error about this value, at the place the value was written.
//
// A value from a file gets a file, a line and a column, and a value from
// anywhere else gets the name of its layer, so an error either way says where
// to go and change it.
//
// It is for code holding a value and no field, such as a check that runs after
// everything is read. A [Parse] does not need it, because [Get] already knows
// the field and the place and puts both in front of whatever the parser said.
func (v Value) Errorf(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	if v.TOML != nil {
		return &Error{
			File: v.TOML.Pos.File,
			Line: v.TOML.Pos.Line,
			Col:  v.TOML.Pos.Col,
			Msg:  msg,
		}
	}
	return &Error{Msg: v.Source.String() + ": " + msg}
}

func display(v *toml.Value) string {
	switch v.Kind {
	case toml.KindString:
		return v.Str
	case toml.KindInt:
		return strconv.FormatInt(v.Int, 10)
	case toml.KindFloat:
		return strconv.FormatFloat(v.Float, 'g', -1, 64)
	case toml.KindBool:
		return strconv.FormatBool(v.Bool)
	case toml.KindOffsetDateTime:
		return v.Time.Format(time.RFC3339Nano)
	case toml.KindLocalDateTime:
		return v.Time.Format("2006-01-02T15:04:05.999999999")
	case toml.KindLocalDate:
		return v.Time.Format("2006-01-02")
	case toml.KindLocalTime:
		return v.Time.Format("15:04:05.999999999")
	case toml.KindArray:
		parts := make([]string, len(v.Array))
		for i, e := range v.Array {
			parts[i] = display(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case toml.KindTable:
		var b strings.Builder
		b.WriteByte('{')
		for k, e := range v.Table.All() {
			if b.Len() > 1 {
				b.WriteString(", ")
			}
			b.WriteString(k)
			b.WriteString(" = ")
			b.WriteString(display(e))
		}
		b.WriteByte('}')
		return b.String()
	}
	return ""
}

// A Setting is one field and what the loader found for it, in the order it was
// asked for. config:show is a list of these.
type Setting struct {
	Field
	Value Value

	// Set is whether any layer had a value, the default included.
	Set bool
}

// Display is the value as it should be shown to a person, which is *** for a
// secret. Use this rather than Value.Display anywhere the result is going to
// end up on a screen or in a log.
func (s Setting) Display() string {
	if s.Secret && s.Set && s.Value.Display() != "" {
		return "***"
	}
	return s.Value.Display()
}

// A Loader answers what the value of a setting is and where it came from.
//
// It is not safe for concurrent use. Configuration is read once at startup by
// one goroutine, and a lock here would only hide a caller doing something
// stranger than that.
type Loader struct {
	env      string
	files    []loaded
	dotenv   map[string]entry
	environ  map[string]string
	flags    map[string]entry
	flagKeys []string // in the order they were given, for a stable report
	override map[string]string

	// command runs a cmd: indirection, and is nil when the caller did not
	// supply one, which is when a cmd: value is refused.
	command func(string) (string, error)

	settings []Setting
	errs     []*FieldError

	// asked is every path a field asked for, and open is every prefix of one.
	// A file key that is in asked was read, and one that is in open holds
	// something that was read, so neither is a key nobody wanted.
	asked map[string]bool
	open  map[string]bool
}

type loaded struct {
	name string
	doc  *toml.Table
}

type entry struct {
	value  string
	source Source
}

// Open reads everything the sources name.
//
// A file that is not there is skipped, because every file in the list is
// optional by design. A file that is there and cannot be read or parsed is an
// error, and all of those errors come back together, so a project with two
// broken files finds out about both at once.
func Open(s Sources) (*Loader, error) {
	l := &Loader{
		env:      s.Env,
		dotenv:   map[string]entry{},
		environ:  map[string]string{},
		flags:    map[string]entry{},
		override: s.Override,
		command:  s.Command,
		asked:    map[string]bool{},
		open:     map[string]bool{},
	}
	if l.env == "" {
		l.env = EnvDefault
	}

	var errs []error
	for _, name := range s.Files {
		data, err := os.ReadFile(name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		doc, err := toml.Parse(name, data)
		if err != nil {
			errs = append(errs, wrap(err))
			continue
		}
		l.files = append(l.files, loaded{name: name, doc: doc})
	}

	for _, kv := range s.Environ {
		if k, v, ok := strings.Cut(kv, "="); ok {
			l.environ[k] = v
		}
	}

	// The .env files are read after the process environment so that ${NAME}
	// in one of them can refer to a real variable, and read in order so that
	// a later file can refer to an earlier one.
	for _, name := range s.DotEnv {
		data, err := os.ReadFile(name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		vars, err := parseDotEnv(name, data, l.resolve)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, v := range vars {
			l.dotenv[v.name] = entry{
				value:  v.value,
				source: Source{From: FromDotEnv, Name: name + ":" + strconv.Itoa(v.line)},
			}
		}
	}

	if err := l.readFlags(s.Args); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return l, nil
}

// resolve is what a .env file's ${NAME} looks in: the .env files read so far,
// and then the process environment.
func (l *Loader) resolve(name string) (string, bool) {
	if e, ok := l.dotenv[name]; ok {
		return e.value, true
	}
	v, ok := l.environ[name]
	return v, ok
}

func (l *Loader) readFlags(args []string) error {
	var errs []error
	for _, arg := range args {
		rest, ok := configFlag(arg)
		if !ok {
			continue
		}
		name, value, ok := strings.Cut(rest, "=")
		if !ok {
			errs = append(errs, &Error{Msg: arg + " needs a value, as " + arg + "=..."})
			continue
		}
		key := flagKey(name)
		if _, seen := l.flags[key]; !seen {
			l.flagKeys = append(l.flagKeys, key)
		}
		l.flags[key] = entry{value: value, source: Source{From: FromFlag, Name: arg}}
	}
	return errors.Join(errs...)
}

// Env is the environment name the loader was opened with.
func (l *Loader) Env() string { return l.env }

// Lookup returns the value of a field and whether any layer had one.
//
// It also records the field, so that [Loader.Settings] can list what the
// application asked for and [Loader.Check] can tell which keys in the files
// nothing asked for.
func (l *Loader) Lookup(f Field) (Value, bool) {
	v, ok := l.find(f)
	l.mark(f.Path)
	l.settings = append(l.settings, Setting{Field: f, Value: v, Set: ok})
	return v, ok
}

func (l *Loader) find(f Field) (Value, bool) {
	if f.Path != "" {
		if text, ok := l.override[f.Path]; ok {
			return Value{Source: Source{From: FromOverride}, Text: text}, true
		}
		if e, ok := l.flags[flagKey(f.Path)]; ok {
			return Value{Source: e.source, Text: e.value}, true
		}
	}
	if f.Env != "" {
		if text, ok := l.environ[f.Env]; ok {
			return Value{Source: Source{From: FromEnv, Name: f.Env}, Text: text}, true
		}
		if e, ok := l.dotenv[f.Env]; ok {
			return Value{Source: e.source, Text: e.value}, true
		}
	}
	if f.Path != "" {
		for i := len(l.files) - 1; i >= 0; i-- {
			if v := lookupPath(l.files[i].doc, f.Path); v != nil {
				return Value{Source: Source{From: FromFile, Name: v.Pos.String()}, TOML: v}, true
			}
		}
	}
	if f.Default != "" {
		return Value{Source: Source{From: FromDefault}, Text: f.Default}, true
	}
	return Value{}, false
}

// lookupPath follows a dotted path down through a document, without splitting
// it into a slice on the way. This runs once for every field of every file, at
// startup, and there is nothing for it to allocate.
func lookupPath(t *toml.Table, path string) *toml.Value {
	for {
		key, rest, more := strings.Cut(path, ".")
		v := t.Get(key)
		if v == nil || !more {
			return v
		}
		if v.Kind != toml.KindTable {
			return nil // something above the leaf is not a table
		}
		t, path = v.Table, rest
	}
}

// mark records that a path was asked for, along with every table above it.
func (l *Loader) mark(path string) {
	if path == "" || l.asked[path] {
		return
	}
	l.asked[path] = true
	for rest := path; ; {
		i := strings.LastIndexByte(rest, '.')
		if i < 0 {
			break
		}
		rest = rest[:i]
		l.open[rest] = true
	}
}

// Settings are the fields that were looked up, in the order they were asked
// for, with what was found for each.
func (l *Loader) Settings() []Setting { return slices.Clone(l.settings) }

// Check reports every setting that was written down but that nothing asked
// for, joined into one error.
//
// Call it after the last [Loader.Lookup], because until then a key that
// nothing has asked for yet is only a key nothing has asked for yet.
func (l *Loader) Check() error {
	unknown := l.Unknown()
	if len(unknown) == 0 {
		return nil
	}
	errs := make([]error, len(unknown))
	for i, u := range unknown {
		errs[i] = u
	}
	return errors.Join(errs...)
}

// Unknown lists the settings in the files and on the command line that nothing
// asked for, in the order they were written.
//
// A table nobody asked anything under is reported once, rather than once for
// every key inside it, because a misspelled section header is one mistake.
func (l *Loader) Unknown() []Unknown {
	var out []Unknown
	known := l.known()
	for _, f := range l.files {
		l.walk(f.doc, "", known, &out)
	}
	for _, key := range l.flagKeys {
		if !l.asked[key] && !l.open[key] {
			out = append(out, Unknown{Path: key, From: l.flags[key].source, Near: nearest(key, known)})
		}
	}
	return out
}

func (l *Loader) walk(t *toml.Table, prefix string, known []string, out *[]Unknown) {
	for key, v := range t.All() {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		switch {
		case l.asked[path]:
			// Asked for, and everything under it belongs to whoever asked.
		case l.open[path] && v.Kind == toml.KindTable:
			l.walk(v.Table, path, known, out)
		case l.open[path] && v.Kind == toml.KindArray:
			for _, e := range v.Array {
				if e.Kind == toml.KindTable {
					l.walk(e.Table, path, known, out)
				}
			}
		default:
			*out = append(*out, Unknown{
				Path: path,
				From: Source{From: FromFile, Name: v.Pos.String()},
				Near: nearest(path, known),
			})
		}
	}
}

// known is every path a field asked for, and every table above one, which is
// the set a misspelling gets compared against.
func (l *Loader) known() []string {
	out := make([]string, 0, len(l.asked)+len(l.open))
	for p := range l.asked {
		out = append(out, p)
	}
	for p := range l.open {
		out = append(out, p)
	}
	slices.Sort(out)
	return out
}
