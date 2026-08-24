package gen

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"go/scanner"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// A File is one file a generator produced, held in memory until a [Writer]
// puts it somewhere.
//
// Path is relative to the writer's directory and is written with forward
// slashes, because a generator thinks in import paths rather than in the local
// file system. Data is the whole file, header and all.
type File struct {
	Path string
	Data []byte
}

// A Status says what a [Writer] did with one file. In check mode it says what
// would have happened.
type Status int

const (
	Unchanged Status = iota // the file on disk already said this
	Created                 // there was no file
	Updated                 // there was a file and it differed
)

func (s Status) String() string {
	switch s {
	case Unchanged:
		return "unchanged"
	case Created:
		return "created"
	case Updated:
		return "updated"
	}
	return fmt.Sprintf("Status(%d)", int(s))
}

// A Result is what happened to one file.
type Result struct {
	Path   string
	Status Status
	Line   int // the first line that differs, when Status is Updated
}

// Changed reports whether the file on disk was not already what the generator
// produced. It is the question --check asks.
func (r Result) Changed() bool { return r.Status != Unchanged }

// A Writer puts generated files on disk.
//
// Files are formatted, compared with what is already there, and written only
// when they differ, through a temporary file and a rename. The comparison is
// not an optimisation: a generator that rewrites every file on every run
// retriggers every watcher and every rebuild that was waiting on an mtime.
//
// The zero Writer writes to the current directory.
type Writer struct {
	// Dir is where a file's path is resolved from. Empty means ".".
	Dir string

	// Check makes Write report what it would do and touch nothing. The results
	// come back the same either way, which is what makes --check trustworthy:
	// it runs the same code as the real thing.
	Check bool
}

// Write writes the given files.
//
// Every file has to carry the header from [Header], and a file already on disk
// without one is never overwritten. Between them those two rules mean the
// writer only ever replaces its own output, and it can tell the difference
// without keeping a list of what it wrote.
//
// Results come back sorted by path, whatever order the files were given in,
// because a generator's report has to read the same from one run to the next.
// Failures do not stop the other files: the error is every failure joined, so
// one bad file does not hide the rest.
func (w *Writer) Write(files ...File) ([]Result, error) {
	sorted := slices.Clone(files)
	slices.SortStableFunc(sorted, func(a, b File) int { return strings.Compare(a.Path, b.Path) })

	var results []Result
	var errs []error
	for i := 0; i < len(sorted); {
		// Duplicates are next to each other now. Which of them wins is not
		// something to decide by arrival order, so neither is written.
		j := i + 1
		for j < len(sorted) && sorted[j].Path == sorted[i].Path {
			j++
		}
		if n := j - i; n > 1 {
			errs = append(errs, fmt.Errorf("%s: given %d times in one call", sorted[i].Path, n))
			i = j
			continue
		}

		r, err := w.one(sorted[i])
		if err != nil {
			errs = append(errs, err)
		} else {
			results = append(results, r)
		}
		i = j
	}
	return results, errors.Join(errs...)
}

func (w *Writer) one(f File) (Result, error) {
	rel := filepath.FromSlash(f.Path)
	if !filepath.IsLocal(rel) {
		return Result{}, fmt.Errorf("%s: a generated file has to stay inside %s", f.Path, w.dir())
	}

	data := f.Data
	if filepath.Ext(rel) == ".go" {
		var err error
		if data, err = Format(f.Path, data); err != nil {
			return Result{}, err
		}
	}
	if !Generated(data) {
		return Result{}, fmt.Errorf("%s: has no generated header, so the next run would take it for a hand-written file and refuse to touch it; start it with gen.Header", f.Path)
	}

	name := filepath.Join(w.dir(), rel)
	old, err := os.ReadFile(name)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if !w.Check {
			if err := os.MkdirAll(filepath.Dir(name), 0o777); err != nil {
				return Result{}, fmt.Errorf("%s: %w", f.Path, err)
			}
			if err := writeFile(name, data); err != nil {
				return Result{}, fmt.Errorf("%s: %w", f.Path, err)
			}
		}
		return Result{Path: f.Path, Status: Created}, nil

	case err != nil:
		return Result{}, fmt.Errorf("%s: %w", f.Path, err)
	}

	if !Generated(old) {
		return Result{}, fmt.Errorf("refusing to overwrite hand-written file %s", f.Path)
	}
	if bytes.Equal(old, data) {
		return Result{Path: f.Path, Status: Unchanged}, nil
	}
	if !w.Check {
		if err := writeFile(name, data); err != nil {
			return Result{}, fmt.Errorf("%s: %w", f.Path, err)
		}
	}
	return Result{Path: f.Path, Status: Updated, Line: firstDiff(old, data)}, nil
}

func (w *Writer) dir() string {
	if w.Dir == "" {
		return "."
	}
	return w.Dir
}

// writeFile writes through a temporary file in the same directory, so a reader
// gets either the whole old file or the whole new one. A build running while a
// generator writes is the normal case and not a rare one, and half a file
// compiles about as well as it sounds.
//
// The result is mode 0644 whether the file was there before or not, because a
// rename replaces the directory entry along with the contents.
func writeFile(name string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(name), "."+filepath.Base(name)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // Does nothing once the rename below succeeds.

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), name)
}

// firstDiff returns the 1-based line of the first byte where two files stop
// agreeing, which is the one thing worth saying about a difference without
// writing a diff.
func firstDiff(old, new []byte) int {
	i := 0
	for i < len(old) && i < len(new) && old[i] == new[i] {
		i++
	}
	return bytes.Count(old[:i], []byte("\n")) + 1
}

const (
	generatedPrefix = "// Code generated "
	generatedSuffix = " DO NOT EDIT."
)

// Header returns the comment that begins a generated file.
//
//	// Code generated by mizu gen orm v1; DO NOT EDIT.
//	// Source: model/post.go
//
// The first line is the one the go command looks for, so every tool that skips
// generated files skips these too. The second says where the declaration it
// was made from lives, so a reader who lands in the output has somewhere to
// go. An empty source leaves that line out, and an empty version leaves the
// version out.
func Header(generator, version, source string) string {
	var b strings.Builder
	b.WriteString(generatedPrefix)
	b.WriteString("by mizu gen ")
	b.WriteString(generator)
	if version != "" {
		b.WriteString(" v")
		b.WriteString(strings.TrimPrefix(version, "v"))
	}
	b.WriteString(";")
	b.WriteString(generatedSuffix)
	b.WriteString("\n")
	if source != "" {
		b.WriteString("// Source: ")
		b.WriteString(source)
		b.WriteString("\n")
	}
	return b.String()
}

// Generated reports whether data carries the header that marks a file as
// generated.
//
// The rule is the go command's own: a line matching
//
//	^// Code generated .* DO NOT EDIT\.$
//
// before the first line that is neither blank nor a comment. See
// https://go.dev/s/generatedcode. The line has to start at the left margin,
// because an indented comment is prose about the code around it.
//
// Comment means the opener of any language a generator emits, not only Go, so
// the same answer holds for a TypeScript client or a SQL migration. Nothing
// outside the go command reads the SQL one, but this package does, and it is
// how the writer knows a file is its own to replace.
func Generated(data []byte) bool {
	for len(data) > 0 {
		line := data
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			line, data = data[:i], data[i+1:]
		} else {
			data = nil
		}
		text := strings.TrimRight(string(line), "\r")
		switch {
		case generatedLine(text):
			return true
		case strings.TrimSpace(text) == "" || isComment(strings.TrimLeft(text, " \t")):
			// Keep looking.
		default:
			return false
		}
	}
	return false
}

// commentOpeners covers the languages a generator writes: Go and TypeScript,
// the continuation and closing lines of a block comment, shell and YAML and
// TOML, SQL, and HTML.
var commentOpeners = []string{"//", "/*", "*", "#", "--", "<!--"}

func generatedLine(text string) bool {
	if !strings.HasSuffix(text, generatedSuffix) {
		return false
	}
	for _, opener := range commentOpeners {
		if strings.HasPrefix(text, opener+" Code generated ") {
			return true
		}
	}
	return false
}

func isComment(text string) bool {
	for _, opener := range commentOpeners {
		if strings.HasPrefix(text, opener) {
			return true
		}
	}
	return false
}

// Format runs generated source through gofmt.
//
// The path is there for the error message, which matters more than usual: the
// file is not on disk to go and look at, so the error has to carry the line it
// went wrong on.
func Format(path string, src []byte) ([]byte, error) {
	out, err := format.Source(src)
	if err != nil {
		return nil, &FormatError{Path: path, Src: src, Err: err}
	}
	return out, nil
}

// A FormatError is generated source that is not valid Go.
//
// Src is what the generator produced, kept so a caller with somewhere to put
// it can write the file out and look at the whole thing.
type FormatError struct {
	Path string
	Src  []byte
	Err  error
}

func (e *FormatError) Error() string {
	var list scanner.ErrorList
	if !errors.As(e.Err, &list) || len(list) == 0 {
		return fmt.Sprintf("%s: %v", e.Path, e.Err)
	}

	first := list[0]
	var b strings.Builder
	fmt.Fprintf(&b, "%s:%d:%d: %s", e.Path, first.Pos.Line, first.Pos.Column, first.Msg)
	if rest := len(list) - 1; rest > 0 {
		fmt.Fprintf(&b, " (and %d more)", rest)
	}
	if line, ok := sourceLine(e.Src, first.Pos.Line); ok {
		fmt.Fprintf(&b, "\n%d | %s", first.Pos.Line, line)
	}
	return b.String()
}

func (e *FormatError) Unwrap() error { return e.Err }

func sourceLine(src []byte, n int) (string, bool) {
	if n < 1 {
		return "", false
	}
	for i := 1; ; i++ {
		line, rest, more := bytes.Cut(src, []byte("\n"))
		if i == n {
			return strings.TrimRight(string(line), "\r"), true
		}
		if !more {
			return "", false
		}
		src = rest
	}
}
