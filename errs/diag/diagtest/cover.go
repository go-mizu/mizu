package diagtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Cover checks that every message the code in src can print has an entry in
// the corpus at dir.
//
// A corpus is only as good as the part of the code it reaches, and the part it
// does not reach is invisible: the tests pass, the golden files look complete,
// and the message nobody has read is the one somebody gets. Cover is the other
// half of [Run]. Run says that what the corpus covers still reads the way it
// was reviewed, and Cover says that the corpus covers everything.
//
// It reads the Go source in src, tests aside, and finds the format strings
// handed to errors.New, fmt.Errorf, and anything whose name ends in errf or
// Errorf. Each one becomes a pattern, with the text kept as it was written and
// each verb standing for whatever it printed, and the want.txt files under dir
// have to hold a line that matches it. Matching the whole format rather than a
// piece of it means the words around the verbs are checked too, which is where
// most of a message is. A format with almost no text in it matches anything
// and has to be named in skip.
//
// skip holds format strings written out in full, so adding one is a decision
// somebody made and a reviewer can see, rather than a threshold somebody
// moved. What belongs in it is a wrapper: a format that puts a position or a
// field name in front of a message that comes from somewhere else and has an
// entry of its own.
func Cover(t testing.TB, dir, src string, skip ...string) {
	t.Helper()

	golden := goldenText(t, dir)
	for _, f := range formats(t, src) {
		if slices.Contains(skip, f.format) {
			continue
		}
		runs := between(f.format)
		switch {
		case textLen(runs) < shortest:
			t.Errorf("%s: %q is mostly verbs, so it would match whatever the corpus holds.\nIf it wraps a message that has an entry of its own, name it in the skip list.", f.where, f.format)
		case !pattern(runs).MatchString(golden):
			t.Errorf("%s: nothing under %s prints %q.\nAdd a case whose input produces it, then run this test with -update and read what it wrote.", f.where, dir, f.format)
		}
	}
}

// shortest is how much text a format string has to have around its verbs for
// matching the corpus against it to mean anything.
//
// Two or three characters match every golden file in the corpus, so a format
// that short passes whatever the corpus holds, which is worse than failing.
const shortest = 10

// pattern is a format string as something to match a message against: the text
// as it was written, and a verb standing for whatever it printed.
//
// A dot does not match a newline, so a match is inside one line of one golden
// file, which is what a message is.
func pattern(runs []string) *regexp.Regexp {
	var b strings.Builder
	for i, s := range runs {
		if i > 0 {
			b.WriteString(".*")
		}
		b.WriteString(regexp.QuoteMeta(s))
	}
	return regexp.MustCompile(b.String())
}

// textLen is how much of a format string is text rather than spacing, which is
// how much of it a match is really checking.
func textLen(runs []string) int {
	var n int
	for _, s := range runs {
		n += len(strings.TrimSpace(s))
	}
	return n
}

// A found is one format string and where it was written.
type found struct {
	format string
	where  string
}

// formats is every message format in the Go source under src, in the order a
// walk of the directory finds them.
func formats(t testing.TB, src string) []found {
	t.Helper()

	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}

	var out []found
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(src, name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !makesAMessage(call.Fun) {
				return true
			}
			if s, ok := literal(call.Args); ok {
				out = append(out, found{format: s, where: fset.Position(call.Pos()).String()})
			}
			return true
		})
	}
	if len(out) == 0 {
		t.Fatalf("%s has no messages in it, so either the code moved or this is looking in the wrong place", src)
	}
	return out
}

// makesAMessage reports whether a call builds an error a person reads.
//
// The name is all there is to go on, since a test that type checked the
// package would need it to compile, and the packages worth running this on are
// the ones that report on code that does not.
func makesAMessage(fun ast.Expr) bool {
	var name string
	switch f := fun.(type) {
	case *ast.Ident:
		name = f.Name
	case *ast.SelectorExpr:
		name = f.Sel.Name
		if pkg, ok := f.X.(*ast.Ident); ok && pkg.Name == "errors" && name == "New" {
			return true
		}
	default:
		return false
	}
	return strings.HasSuffix(name, "errf") || strings.HasSuffix(name, "Errorf")
}

// literal is the first string literal in an argument list, which is where the
// format string is in all three of the calls this looks at.
func literal(args []ast.Expr) (string, bool) {
	for _, a := range args {
		lit, ok := a.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		s, err := strconv.Unquote(lit.Value)
		return s, err == nil
	}
	return "", false
}

// between splits a format string on its verbs. A doubled percent sign is text
// rather than a verb, so it stays in the run it was written in.
func between(format string) []string {
	var out []string
	var b strings.Builder
	for i := 0; i < len(format); i++ {
		switch {
		case format[i] != '%':
			b.WriteByte(format[i])
		case i+1 < len(format) && format[i+1] == '%':
			b.WriteByte('%')
			i++
		default:
			out = append(out, b.String())
			b.Reset()
			// The flags, the width and the precision, and then the letter
			// that ends the verb.
			for i++; i < len(format); i++ {
				if c := format[i] | 0x20; c >= 'a' && c <= 'z' {
					break
				}
			}
		}
	}
	return append(out, b.String())
}

// goldenText is every golden file under dir, joined, which is what a message
// is looked for in.
//
// Which entry prints a message does not matter here. What matters is that some
// entry does, so that somebody changing the message sees the diff.
func goldenText(t testing.TB, dir string) string {
	t.Helper()

	var b strings.Builder
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "want.txt" {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		b.Write(data)
		b.WriteByte('\n')
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// An empty golden file is the same as no golden file here, and both of
	// them would have every message look uncovered, which reads as a corpus
	// that is behind rather than as a corpus that is not there.
	out := b.String()
	if strings.TrimSpace(out) == "" {
		t.Fatalf("%s holds no golden files with anything in them, so every message would look uncovered", dir)
	}
	return out
}
