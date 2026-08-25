package main

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/go-mizu/mizu/console"
)

// version is what mizu knows about itself.
//
// Every field comes from the build info the compiler already embeds, so there
// are no -ldflags to remember and no generated file to keep in step. A binary
// from go install carries the module version it was installed at; one built
// from a checkout carries the commit instead. That is also what keeps the
// build reproducible, since none of these values is a clock reading taken at
// build time.
type version struct {
	Version  string `json:"version"`
	Revision string `json:"revision,omitempty"`
	Time     string `json:"time,omitempty"`
	Modified bool   `json:"modified,omitempty"`
	Go       string `json:"go"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
}

// self is what this binary was built from.
//
// It is read once because debug.ReadBuildInfo parses the embedded table every
// time it is called, and both the --version flag and the version command want
// the same answer out of it.
var self = sync.OnceValue(func() version { return versionOf(debug.ReadBuildInfo()) })

// versionOf reads what it can out of build info. Info is nil when the binary
// was built in a way that strips it, which is rare and not worth an error, so
// the version comes out as "unknown" and the rest of the fields still hold.
func versionOf(info *debug.BuildInfo, ok bool) version {
	v := version{
		Version: "unknown",
		Go:      runtime.Version(),
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
	if !ok || info == nil {
		return v
	}
	if info.Main.Version != "" {
		v.Version = info.Main.Version
	}
	if info.GoVersion != "" {
		v.Go = info.GoVersion
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			v.Revision = s.Value
		case "vcs.time":
			v.Time = s.Value
		case "vcs.modified":
			v.Modified = s.Value == "true"
		case "GOOS":
			v.OS = s.Value
		case "GOARCH":
			v.Arch = s.Value
		}
	}
	return v
}

// String renders the human form: what it is on the first line, what built it
// on the second.
func (v version) String() string {
	var b strings.Builder
	b.WriteString("mizu ")
	b.WriteString(v.Version)
	if v.Revision != "" {
		fmt.Fprintf(&b, " (%s", short(v.Revision))
		if v.Time != "" {
			fmt.Fprintf(&b, ", %s", v.Time)
		}
		// A pseudo-version already ends in +dirty, so saying it again would
		// print the word twice on the same line.
		if v.Modified && !strings.HasSuffix(v.Version, "+dirty") {
			b.WriteString(", dirty")
		}
		b.WriteByte(')')
	}
	fmt.Fprintf(&b, "\n%s %s/%s\n", v.Go, v.OS, v.Arch)
	return b.String()
}

// short trims a commit hash to the length people actually paste.
func short(rev string) string {
	if len(rev) > 12 {
		return rev[:12]
	}
	return rev
}

// Version prints what mizu knows about itself.
type Version struct{}

func (c *Version) Spec() console.Spec {
	return console.Spec{
		Name: "version",
		Desc: "Print version information",
		Long: versionLong,
	}
}

func (c *Version) Run(ctx context.Context, io *console.IO) error {
	v := self()
	if io.JSONMode() {
		return io.JSON(v)
	}
	io.Print("%s", v.String())
	return nil
}

const versionLong = `Every fact here comes from the build information the compiler embeds, so a
binary reports what it was built from without anything having been passed at
build time.

A binary from go install carries the module version it was installed at. One
built from a checkout carries the commit and the time instead, and says dirty
when the tree had uncommitted changes in it.

Run it with --json for the same facts as an object, which is what to paste into
a bug report.`
