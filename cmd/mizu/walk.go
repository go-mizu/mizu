package main

import (
	"io/fs"
	"iter"
	"slices"
	"strings"
)

// skipped are the directories a walk of a project does not go into.
//
// A dot directory is somebody's tooling, node_modules is an ecosystem of its
// own, vendor is a copy of other people's code, and testdata is a fixture. What
// is in any of them is not this project's own, and walking them is most of the
// time a walk of the project would take.
//
// A directory whose name starts with an underscore is skipped for the same
// reason the go command skips one: ./... does not match it, go build does not
// compile it, and go test does not run it. A tool that describes the project
// has to agree with the tool that builds it, or AGENTS.md lists packages that
// nothing in the module can import.
var skipped = []string{"node_modules", "testdata", "vendor"}

// projectFiles yields every ordinary file in the project, by its path from the
// root, in the order a walk finds them.
//
// A symlink is not one of them. Following it reports the same file twice, or a
// file that is not in the project at all.
//
// Nothing here fails. A directory that cannot be read is not evidence of
// anything, and the commands that walk a project are reporting on it rather
// than guarding it.
func projectFiles(fsys fs.FS) iter.Seq[string] {
	return func(yield func(string) bool) {
		fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
			switch {
			case err != nil:
				return nil
			case d.IsDir():
				if name != "." && (strings.HasPrefix(d.Name(), ".") || strings.HasPrefix(d.Name(), "_") || slices.Contains(skipped, d.Name())) {
					return fs.SkipDir
				}
				return nil
			case !d.Type().IsRegular():
				return nil
			}
			if !yield(name) {
				// SkipAll ends the walk, so nothing after this reaches the
				// caller that already said it had enough.
				return fs.SkipAll
			}
			return nil
		})
	}
}
