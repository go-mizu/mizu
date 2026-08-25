package main

import (
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/console"
)

// reference is the package comment in doc.go, which is what go doc prints and
// what pkg.go.dev shows.
//
// It is read from the file rather than from the built binary because a comment
// is not in one, and the point of these tests is that the comment and the
// program say the same thing.
func reference(tb testing.TB) string {
	tb.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "doc.go", nil, parser.PackageClauseOnly|parser.ParseComments)
	if err != nil {
		tb.Fatal(err)
	}
	if f.Doc == nil {
		tb.Fatal("doc.go has no package comment")
	}
	return f.Doc.Text()
}

// A reference that has fallen behind the program is worse than no reference,
// since somebody reading it has no way to tell which of the two is wrong.
func TestTheReferenceListsEveryCommand(t *testing.T) {
	doc := reference(t)
	for _, cmd := range registry() {
		name := cmd.Spec().Name
		if !strings.Contains(doc, "mizu "+name) {
			t.Errorf("doc.go says nothing about mizu %s", name)
		}
	}
}

func TestTheReferenceListsEveryGlobalFlag(t *testing.T) {
	doc := reference(t)
	var mine globals
	var shared console.Globals
	for _, f := range append(shared.Flags(), mine.flags()...) {
		if !strings.Contains(doc, "--"+f.Name) {
			t.Errorf("doc.go says nothing about --%s", f.Name)
		}
	}
}

// The exit codes are the part of the interface a shell script reads, so they
// are named here rather than copied into the comment and left there.
func TestTheReferenceListsEveryExitCode(t *testing.T) {
	doc := reference(t)
	codes := []int{
		console.CodeOK,
		console.CodeFailure,
		console.CodeUsage,
		console.CodeUnavailable,
		console.CodeInternal,
		console.CodeNoPermission,
		console.CodeConfig,
		console.CodeInterrupted,
	}
	for _, code := range codes {
		if !strings.Contains(doc, "\t"+strconv.Itoa(code)+" ") {
			t.Errorf("doc.go has no line for exit code %d", code)
		}
	}
}
