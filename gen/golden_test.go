package gen

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/types"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode"
)

var update = flag.Bool("update", false, "rewrite the files under testdata/golden")

// The columns generator is a whole generator in about sixty lines. It reads
// the markers, asks the type checker what the fields are, renders, and hands
// the files back for the writer to put somewhere.
//
// It is here because a golden test needs something to be golden about, and
// because the harness is worth proving end to end before the real generators
// are built on it. It is a test generator and not a shipped one: the real ORM
// generator reads struct tags, follows relations, and has opinions about
// plurals.

const columnsVersion = "1"

func columns(pkgs ...*Package) ([]File, error) {
	var files []File
	for _, p := range pkgs {
		targets, errs := Scan(p)
		if len(errs) > 0 {
			var joined []error
			for _, e := range errs {
				joined = append(joined, e)
			}
			return nil, errors.Join(joined...)
		}

		var body strings.Builder
		var source string
		for _, t := range targets {
			table, ok := tableName(t)
			if !ok {
				continue
			}
			st, ok := structOf(t.Object)
			if !ok {
				return nil, fmt.Errorf("%s: %s is marked as a table and is not a struct", t.Pos(), t.Name())
			}
			if source == "" {
				// One source line for the file, which is the first
				// declaration that asked for it. A generator writing one file
				// per type would name each of them.
				source = path.Join(dirOf(p), filepath.Base(t.Pos().Filename))
			}

			name := t.Name()
			fmt.Fprintf(&body, "// %sTable is where a %s is stored.\n", name, name)
			fmt.Fprintf(&body, "const %sTable = %q\n\n", name, table)
			fmt.Fprintf(&body, "// %sColumns are its columns, in the order the fields are declared.\n", name)
			fmt.Fprintf(&body, "var %sColumns = []string{\n", name)
			for i := range st.NumFields() {
				if f := st.Field(i); f.Exported() {
					fmt.Fprintf(&body, "%q,\n", column(f.Name()))
				}
			}
			body.WriteString("}\n\n")
		}
		if body.Len() == 0 {
			continue
		}

		var out strings.Builder
		out.WriteString(Header("columns", columnsVersion, source))
		fmt.Fprintf(&out, "\npackage %s\n\n", p.Name)
		out.WriteString(body.String())
		files = append(files, File{
			Path: path.Join(dirOf(p), "columns_gen.go"),
			Data: []byte(out.String()),
		})
	}
	return files, nil
}

// tableName reads the two markers this generator answers to.
func tableName(t Target) (string, bool) {
	for _, m := range t.Markers {
		switch m.Name {
		case "table":
			if w := m.Words(); len(w) == 1 {
				return w[0], true
			}
		case "model":
			if name, ok := m.Get("table"); ok {
				return name, true
			}
		}
	}
	return "", false
}

func structOf(obj types.Object) (*types.Struct, bool) {
	if obj == nil {
		return nil, false
	}
	st, ok := obj.Type().Underlying().(*types.Struct)
	return st, ok
}

// dirOf is the package directory relative to its module, which is what a path
// in a generated file is relative to.
func dirOf(p *Package) string {
	if p.Module == "" || p.PkgPath == p.Module {
		return ""
	}
	return strings.TrimPrefix(p.PkgPath, p.Module+"/")
}

// column turns a field name into a column name, so ID becomes id and
// HTTPStatus becomes http_status.
func column(name string) string {
	runes := []rune(name)
	var b strings.Builder
	for i, r := range runes {
		if !unicode.IsUpper(r) {
			b.WriteRune(r)
			continue
		}
		prevIsLower := i > 0 && !unicode.IsUpper(runes[i-1])
		nextIsLower := i+1 < len(runes) && !unicode.IsUpper(runes[i+1])
		if i > 0 && (prevIsLower || nextIsLower) {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func TestGoldenColumns(t *testing.T) {
	pkgs, err := Load(Config{Dir: fixture}, "./model", "./markers")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pkgs {
		if err := p.Err(); err != nil {
			t.Fatal(err)
		}
	}

	files, err := columns(pkgs...)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("generated %d files, want 2", len(files))
	}

	// Backwards, because the order files arrive in is the generator's business
	// and the output is not allowed to depend on it.
	backwards := slices.Clone(files)
	slices.Reverse(backwards)

	dir := t.TempDir()
	w := &Writer{Dir: dir}
	results, err := w.Write(backwards...)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Status != Created {
			t.Errorf("%s came back %v, want created", r.Path, r.Status)
		}
		got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(r.Path)))
		if err != nil {
			t.Fatal(err)
		}
		golden(t, r.Path, got)
	}

	// Running again finds nothing to do, which is what makes mizu gen --check
	// worth putting in CI.
	results, err = w.Write(files...)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Changed() {
			t.Errorf("%s came back %v on a second run, want unchanged", r.Path, r.Status)
		}
	}
}

func TestGoldenColumnsIsDeterministic(t *testing.T) {
	pkgs, err := Load(Config{Dir: fixture}, "./model", "./markers")
	if err != nil {
		t.Fatal(err)
	}
	first, err := columns(pkgs...)
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		again, err := columns(pkgs...)
		if err != nil {
			t.Fatal(err)
		}
		if len(again) != len(first) {
			t.Fatalf("got %d files, then %d", len(first), len(again))
		}
		for i := range again {
			if again[i].Path != first[i].Path {
				t.Fatalf("file %d is %s, then %s", i, first[i].Path, again[i].Path)
			}
			if !bytes.Equal(again[i].Data, first[i].Data) {
				t.Fatalf("%s came out differently the second time", first[i].Path)
			}
		}
	}
}

func TestColumnName(t *testing.T) {
	tests := []struct{ in, want string }{
		{"ID", "id"},
		{"Email", "email"},
		{"CreatedAt", "created_at"},
		{"HTTPStatus", "http_status"},
		{"OAuth2Token", "o_auth2_token"},
		{"A", "a"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := column(tt.in); got != tt.want {
			t.Errorf("column(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// golden compares got with testdata/golden/<name>, or writes it there when the
// test runs with -update.
func golden(t *testing.T, name string, got []byte) {
	t.Helper()
	file := filepath.Join("testdata", "golden", filepath.FromSlash(name))

	if *update {
		if err := os.MkdirAll(filepath.Dir(file), 0o777); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, got, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", file)
		return
	}

	want, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("%v\nrun go test ./gen -update to write the golden files", err)
	}
	if bytes.Equal(got, want) {
		return
	}
	if bytes.Equal(got, bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))) {
		t.Fatalf("%s differs from its golden file only in line endings, so this checkout turned LF into CRLF.\nthe repository's .gitattributes asks for LF, and git config core.autocrlf=false makes it stick", name)
	}
	t.Errorf("%s does not match its golden file.\nrun go test ./gen -update and read the diff\n\ngot:\n%s\nwant:\n%s", name, got, want)
}
