package golden

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// update is the -update flag, which every assertion here reads.
//
// A package that declares a flag of its own by the same name and also imports
// this one will panic on startup with "flag redefined: update", which is the
// flag package saying two owners is one too many. The fix is to delete the
// other one: -update means the same thing wherever it is passed.
var update = flag.Bool("update", false, "rewrite the golden files under testdata instead of comparing against them")

// Assert compares got with the golden file for the running test, or writes it
// there when the test runs with -update.
//
//	golden.Assert(t, out.Bytes())
//
// The file is testdata/<test name>.golden. [Name] chooses another one, which is
// what a test asserting more than one thing needs.
func Assert(tb testing.TB, got []byte, opts ...Option) {
	tb.Helper()
	assert(tb, got, settle(tb, opts))
}

// AssertString is [Assert] for a string, which is what most callers have.
func AssertString(tb testing.TB, got string, opts ...Option) {
	tb.Helper()
	assert(tb, []byte(got), settle(tb, opts))
}

// assert is the whole of this package: scrub, read or write, compare, explain.
func assert(tb testing.TB, got []byte, o options) {
	tb.Helper()

	path := o.path()
	got = o.scrub(o.normalize(tb, got))

	if *update {
		write(tb, path, got)
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			tb.Fatalf("%s does not exist yet.\nrun go test %s -update to write it, then read the file before committing it", path, packageArg(tb))
			return
		}
		tb.Fatal(err)
		return
	}

	// The file goes through the same normaliser as the value under test, so a
	// file written by hand, reindented by an editor or checked out with
	// different line endings still matches.
	want = o.scrub(o.normalize(tb, want))

	if bytes.Equal(got, want) {
		return
	}
	if bytes.Equal(got, bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))) {
		tb.Fatalf("%s differs from the golden file only in line endings, so this checkout turned LF into CRLF.\n"+
			"the repository's .gitattributes asks for LF, and git config core.autocrlf=false makes it stick", path)
		return
	}

	tb.Fatalf("%s does not match the golden file.\n%s\nrun go test %s -update and read the diff before committing it",
		tb.Name(), diff(path, want, got), packageArg(tb))
}

// write puts got at path, making the directory if it is not there.
func write(tb testing.TB, path string, got []byte) {
	tb.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
		tb.Fatal(err)
		return
	}
	if err := os.WriteFile(path, got, 0o644); err != nil {
		tb.Fatal(err)
		return
	}
	tb.Logf("wrote %s", path)
}

// packageArg is what to tell the reader to pass to go test. A failure inside
// one package in a large tree is more useful with the package named than with
// ./... , which would rewrite everything.
func packageArg(tb testing.TB) string {
	dir, err := os.Getwd()
	if err != nil {
		return "./..."
	}
	return "./" + filepath.Base(dir)
}

// pathFor turns a test name into the file that holds its output.
//
// A subtest name has a slash in it, and that slash becomes a directory rather
// than an underscore, so a table test with thirty cases fills one directory
// instead of thirty long names in the middle of everything else.
func pathFor(dir, name string) string {
	parts := strings.Split(name, "/")
	for i, p := range parts {
		parts[i] = safe(p)
	}
	parts[len(parts)-1] += ".golden"
	return filepath.Join(append([]string{dir}, parts...)...)
}

// safe replaces the characters a file name cannot hold. Windows is the one that
// matters here, since it refuses several that Unix allows and a golden file has
// to be checked out on both.
func safe(name string) string {
	const reserved = `<>:"\|?*`

	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		switch {
		case r < 0x20, strings.ContainsRune(reserved, r):
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	if s := b.String(); s != "" {
		return s
	}
	return "_"
}

// options is what the variadic arguments add up to.
type options struct {
	dir       string // where the golden files live, "testdata" unless told otherwise
	name      string // the test's name, or whatever Name was given
	full      string // set by At, and then dir and name are not consulted
	normalize func(testing.TB, []byte) []byte
	scrubs    []scrubber
}

// path is the file this assertion reads or writes.
func (o options) path() string {
	if o.full != "" {
		return o.full
	}
	return pathFor(o.dir, o.name)
}

// Option changes where an assertion looks or what it does to the bytes before
// comparing them.
type Option func(*options)

// Name puts the output under <name>.golden rather than under the test's own
// name, which is what a test asserting more than one thing needs.
//
//	golden.Assert(t, header, golden.Name("TestGen/header"))
//	golden.Assert(t, body, golden.Name("TestGen/body"))
//
// A slash in the name is a directory, the same as in a subtest name.
func Name(name string) Option {
	return func(o *options) { o.name = name }
}

// Dir looks somewhere other than testdata. The name is still the test's, so a
// subtest still gets its own directory underneath.
func Dir(dir string) Option {
	return func(o *options) { o.dir = dir }
}

// At names the file outright, extension and all, for the case where the golden
// file is also an input to something else and has to be called what that
// something else expects. [Dir] and [Name] have no effect alongside it.
func At(path string) Option {
	return func(o *options) { o.full = filepath.FromSlash(path) }
}

// settle applies the options to the defaults.
//
// The normaliser is not one of the options, because it is decided by which
// assertion was called rather than by the caller. [Assert] compares bytes, so
// its normaliser hands them back.
func settle(tb testing.TB, opts []Option) options {
	tb.Helper()

	o := options{
		dir:       "testdata",
		name:      tb.Name(),
		normalize: asIs,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

func asIs(_ testing.TB, b []byte) []byte { return b }

// scrub runs every scrubber over b, in the order they were given.
func (o options) scrub(b []byte) []byte {
	for _, s := range o.scrubs {
		b = s(b)
	}
	return b
}
