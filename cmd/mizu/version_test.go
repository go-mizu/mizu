package main

import (
	"encoding/json"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/console"
	"github.com/go-mizu/mizu/console/consoletest"
)

func TestVersionOf(t *testing.T) {
	tests := []struct {
		name string
		info *debug.BuildInfo
		ok   bool
		want version
	}{
		{
			name: "no build info at all",
			info: nil,
			ok:   false,
			want: version{Version: "unknown", Go: runtime.Version(), OS: runtime.GOOS, Arch: runtime.GOARCH},
		},
		{
			name: "installed at a tag",
			ok:   true,
			info: &debug.BuildInfo{
				Main:      debug.Module{Version: "v1.2.3"},
				GoVersion: "go1.26.0",
				Settings: []debug.BuildSetting{
					{Key: "GOOS", Value: "linux"},
					{Key: "GOARCH", Value: "arm64"},
				},
			},
			want: version{Version: "v1.2.3", Go: "go1.26.0", OS: "linux", Arch: "arm64"},
		},
		{
			name: "built from a clean checkout",
			ok:   true,
			info: &debug.BuildInfo{
				Main:      debug.Module{Version: "(devel)"},
				GoVersion: "go1.26.0",
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "0123456789abcdef0123456789abcdef01234567"},
					{Key: "vcs.time", Value: "2026-08-24T09:00:00Z"},
					{Key: "vcs.modified", Value: "false"},
					{Key: "GOOS", Value: "darwin"},
					{Key: "GOARCH", Value: "amd64"},
				},
			},
			want: version{
				Version:  "(devel)",
				Revision: "0123456789abcdef0123456789abcdef01234567",
				Time:     "2026-08-24T09:00:00Z",
				Go:       "go1.26.0",
				OS:       "darwin",
				Arch:     "amd64",
			},
		},
		{
			name: "built from a dirty checkout",
			ok:   true,
			info: &debug.BuildInfo{
				Main:      debug.Module{Version: "(devel)"},
				GoVersion: "go1.26.0",
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc123"},
					{Key: "vcs.modified", Value: "true"},
					{Key: "GOOS", Value: "windows"},
					{Key: "GOARCH", Value: "amd64"},
				},
			},
			want: version{
				Version:  "(devel)",
				Revision: "abc123",
				Modified: true,
				Go:       "go1.26.0",
				OS:       "windows",
				Arch:     "amd64",
			},
		},
		{
			name: "build info with nothing in it",
			ok:   true,
			info: &debug.BuildInfo{},
			want: version{Version: "unknown", Go: runtime.Version(), OS: runtime.GOOS, Arch: runtime.GOARCH},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionOf(tt.info, tt.ok); got != tt.want {
				t.Errorf("versionOf() = %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestVersionString(t *testing.T) {
	tests := []struct {
		name string
		v    version
		want string
	}{
		{
			name: "a released binary says only what it is",
			v:    version{Version: "v1.2.3", Go: "go1.26.0", OS: "linux", Arch: "amd64"},
			want: "mizu v1.2.3\ngo1.26.0 linux/amd64\n",
		},
		{
			name: "a checkout build names the commit and shortens it",
			v: version{
				Version:  "(devel)",
				Revision: "0123456789abcdef0123456789abcdef01234567",
				Time:     "2026-08-24T09:00:00Z",
				Go:       "go1.26.0", OS: "darwin", Arch: "arm64",
			},
			want: "mizu (devel) (0123456789ab, 2026-08-24T09:00:00Z)\ngo1.26.0 darwin/arm64\n",
		},
		{
			name: "a dirty build says so",
			v: version{
				Version: "(devel)", Revision: "abc123", Modified: true,
				Go: "go1.26.0", OS: "darwin", Arch: "arm64",
			},
			want: "mizu (devel) (abc123, dirty)\ngo1.26.0 darwin/arm64\n",
		},
		{
			name: "a pseudo-version says dirty once, not twice",
			v: version{
				Version:  "v0.6.1-0.20260824104500-597d34503485+dirty",
				Revision: "597d34503485", Modified: true,
				Go: "go1.26.0", OS: "darwin", Arch: "arm64",
			},
			want: "mizu v0.6.1-0.20260824104500-597d34503485+dirty (597d34503485)\ngo1.26.0 darwin/arm64\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.String(); got != tt.want {
				t.Errorf("String() = %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestVersionCommand(t *testing.T) {
	r := consoletest.Run(t, &Version{}, consoletest.Args()).AssertSuccess()

	lines := strings.Split(strings.TrimRight(r.Stdout(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("output has %d lines, want 2:\n%s", len(lines), r.Stdout())
	}
	if !strings.HasPrefix(lines[0], "mizu ") {
		t.Errorf("first line = %q, want it to start with mizu", lines[0])
	}
	if !strings.Contains(lines[1], runtime.GOARCH) {
		t.Errorf("second line = %q, want it to name the architecture", lines[1])
	}
	r.AssertNoErrorOutput()
}

func TestVersionCommandJSON(t *testing.T) {
	r := consoletest.Run(t, &Version{},
		consoletest.Args(),
		consoletest.With(console.Options{JSON: true}),
	).AssertSuccess()

	var v version
	if err := json.Unmarshal([]byte(r.Stdout()), &v); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, r.Stdout())
	}
	if v.Version == "" {
		t.Error("version field is empty")
	}
	if v.Go != runtime.Version() {
		t.Errorf("go field = %q, want %q", v.Go, runtime.Version())
	}
	if v.OS != runtime.GOOS || v.Arch != runtime.GOARCH {
		t.Errorf("os/arch = %s/%s, want %s/%s", v.OS, v.Arch, runtime.GOOS, runtime.GOARCH)
	}
}

// --json is a global flag, so it means the same thing on either side of the
// command name and no command has to declare it.
func TestVersionJSONIsGlobal(t *testing.T) {
	for _, argv := range [][]string{
		{"version", "--json"},
		{"--json", "version"},
	} {
		out, errOut := say(t)
		if code := newApp().Start(t.Context(), nil, out, errOut, argv); code != console.CodeOK {
			t.Fatalf("%v exited %d: %s", argv, code, errOut)
		}
		var v version
		if err := json.Unmarshal(out.Bytes(), &v); err != nil {
			t.Errorf("%v printed something that is not JSON: %v\n%s", argv, err, out)
		}
	}
}

func TestVersionTakesNoArguments(t *testing.T) {
	out, errOut := say(t)
	if code := newApp().Start(t.Context(), nil, out, errOut, []string{"version", "extra"}); code != console.CodeUsage {
		t.Fatalf("exited %d, want %d", code, console.CodeUsage)
	}
	if !strings.Contains(errOut.String(), "extra") {
		t.Errorf("the error does not name the argument:\n%s", errOut)
	}
}
