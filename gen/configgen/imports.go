package configgen

import (
	"go/types"
	"path"
	"slices"
	"strconv"
	"strings"
)

// An importLine is one entry of the generated import block. Alias is empty
// unless two packages wanted the same name.
type importLine struct {
	Alias string
	Path  string
}

// imports collects what the output has to import while the output is being
// worked out, and hands back the block to write at the top of it.
//
// The generated file is formatted but not rewritten, so the imports have to be
// right when they are emitted rather than fixed up afterwards. That means only
// asking for a name when the output really is going to write one: a type that
// appears in a doc string and nowhere else must not drag in an import that
// nothing then uses, because an unused import does not compile.
type imports struct {
	self  string            // the package being generated into, which imports nothing of itself
	names map[string]string // import path to the name the output uses
	bases map[string]string // import path to the package's own name
	used  map[string]bool   // names already taken
}

func newImports(self string) *imports {
	return &imports{
		self:  self,
		names: map[string]string{},
		bases: map[string]string{},
		used:  map[string]bool{},
	}
}

// name is what the output calls a package, adding it to the block if this is
// the first time it has come up.
//
// A package's own name is usually the last element of its path, and this is
// the form for the handful of paths written into this generator by hand, where
// that is known to hold. Anything read out of a loaded program goes through
// [imports.pkg] instead, which does not have to guess.
func (i *imports) name(pkg string) string { return i.named(pkg, base(pkg)) }

// pkg is name for a package go/types loaded, whose real name is on hand and
// need not be read off the end of the path.
func (i *imports) pkg(p *types.Package) string { return i.named(p.Path(), p.Name()) }

func (i *imports) named(pkg, want string) string {
	if pkg == "" || pkg == i.self {
		return ""
	}
	if n, ok := i.names[pkg]; ok {
		return n
	}
	want = clean(want)
	name := want
	for n := 2; i.used[name]; n++ {
		name = want + strconv.Itoa(n)
	}
	i.names[pkg] = name
	i.bases[pkg] = want
	i.used[name] = true
	return name
}

// add takes a package the output needs whether or not anything asked for its
// name, which is the config package itself.
func (i *imports) add(pkg string) { i.name(pkg) }

// lines are the imports to write, standard library first and everything else
// after it, each group sorted, which is the order gofmt leaves them in.
func (i *imports) lines() []importLine {
	var std, rest []importLine
	for pkg, name := range i.names {
		line := importLine{Path: pkg}
		if name != base(pkg) {
			// Only say the name when the import would not give it anyway,
			// which is when two packages wanted one name or when a package is
			// not called what its path ends in.
			line.Alias = name
		}
		if isStd(pkg) {
			std = append(std, line)
		} else {
			rest = append(rest, line)
		}
	}
	byPath := func(a, b importLine) int { return strings.Compare(a.Path, b.Path) }
	slices.SortFunc(std, byPath)
	slices.SortFunc(rest, byPath)
	return append(std, rest...)
}

// typeString writes a type the way the generated file has to spell it, and
// records every package it mentions on the way through, because a type the
// output writes is a package the output imports.
func (i *imports) typeString(t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string { return i.pkg(p) })
}

// docString writes a type for a human to read, without asking for a single
// import, because this one ends up inside a quoted string.
func (i *imports) docString(t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string {
		if p.Path() == i.self {
			return ""
		}
		return p.Name()
	})
}

// base is the name an import of this path gives, which is the last element,
// except that a path ending in a major version is named by the element before
// it, so example.com/thing/v2 is thing.
func base(pkg string) string {
	b := path.Base(pkg)
	if len(b) > 1 && b[0] == 'v' && isDigits(b[1:]) {
		b = path.Base(path.Dir(pkg))
	}
	return clean(b)
}

// isStd is whether a package is in the standard library, which is what the
// first element of an import path having no dot in it means.
func isStd(pkg string) bool {
	first, _, _ := strings.Cut(pkg, "/")
	return !strings.Contains(first, ".")
}

// clean turns a path element into something that can be a Go identifier, since
// a directory may have a dash in it and an identifier may not.
func clean(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_':
			b.WriteRune(r)
		case r >= '0' && r <= '9' && b.Len() > 0:
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "pkg"
	}
	return b.String()
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}
