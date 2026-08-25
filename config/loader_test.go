package config

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/toml"
)

// project writes a set of files under a temporary directory and returns it.
// Keys are slash separated whatever the platform is, because that is how a
// project is written down.
func project(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func open(t *testing.T, s Sources) *Loader {
	t.Helper()
	l, err := Open(s)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

// lookup is the value of one field as a string, and where it came from.
func lookup(t *testing.T, l *Loader, f Field) (string, string) {
	t.Helper()
	v, ok := l.Lookup(f)
	if !ok {
		return "", "unset"
	}
	return v.Display(), v.Source.String()
}

func TestLookupOrder(t *testing.T) {
	dir := project(t, map[string]string{
		"config/production.toml": "[app]\nname = \"from production\"\n",
		"config/local.toml":      "[app]\nname = \"from local\"\n",
		".env":                   "APP_NAME=from dotenv\n",
	})
	files := []string{
		filepath.Join(dir, "config", "production.toml"),
		filepath.Join(dir, "config", "local.toml"),
	}
	field := Field{Path: "app.name", Env: "APP_NAME", Default: "from the default"}

	// Each step adds one layer on top of the last, and the value has to
	// change every time.
	steps := []struct {
		name    string
		sources Sources
		want    string
		source  string
	}{
		{
			name:   "the default",
			want:   "from the default",
			source: "default",
		},
		{
			name:    "the first file",
			sources: Sources{Files: files[:1]},
			want:    "from production",
			source:  "file " + files[0] + ":2:8",
		},
		{
			name:    "the second file",
			sources: Sources{Files: files},
			want:    "from local",
			source:  "file " + files[1] + ":2:8",
		},
		{
			name:    "a dotenv file",
			sources: Sources{Files: files, DotEnv: []string{filepath.Join(dir, ".env")}},
			want:    "from dotenv",
			source:  "dotenv " + filepath.Join(dir, ".env") + ":1",
		},
		{
			name: "the environment",
			sources: Sources{
				Files:   files,
				DotEnv:  []string{filepath.Join(dir, ".env")},
				Environ: []string{"APP_NAME=from the environment"},
			},
			want:   "from the environment",
			source: "env APP_NAME",
		},
		{
			name: "the command line",
			sources: Sources{
				Files:   files,
				DotEnv:  []string{filepath.Join(dir, ".env")},
				Environ: []string{"APP_NAME=from the environment"},
				Args:    []string{"--config.app.name=from the command line"},
			},
			want:   "from the command line",
			source: "flag --config.app.name=from the command line",
		},
		{
			name: "the program itself",
			sources: Sources{
				Files:    files,
				DotEnv:   []string{filepath.Join(dir, ".env")},
				Environ:  []string{"APP_NAME=from the environment"},
				Args:     []string{"--config.app.name=from the command line"},
				Override: map[string]string{"app.name": "from the program"},
			},
			want:   "from the program",
			source: "override",
		},
	}

	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			got, source := lookup(t, open(t, step.sources), field)
			if got != step.want {
				t.Errorf("app.name is %q, want %q", got, step.want)
			}
			if source != step.source {
				t.Errorf("it came from %q, want %q", source, step.source)
			}
		})
	}
}

func TestLookupUnset(t *testing.T) {
	l := open(t, Sources{})
	if v, ok := l.Lookup(Field{Path: "app.name", Env: "APP_NAME"}); ok {
		t.Errorf("app.name is set to %q, want unset", v.Display())
	}
}

func TestLookupEmptyEnvironmentVariableCounts(t *testing.T) {
	l := open(t, Sources{Environ: []string{"APP_NAME="}})
	v, ok := l.Lookup(Field{Path: "app.name", Env: "APP_NAME", Default: "mizu"})
	if !ok || v.Display() != "" || v.Source.From != FromEnv {
		t.Errorf("app.name is %q from %s, want empty from the environment", v.Display(), v.Source)
	}
}

func TestLookupWithoutAName(t *testing.T) {
	// A field with no Path cannot come from a file or a flag, and one with no
	// Env cannot come from the environment.
	l := open(t, Sources{
		Environ:  []string{"APP_NAME=from the environment"},
		Args:     []string{"--config.app.name=from the command line"},
		Override: map[string]string{"app.name": "from the program"},
	})
	if v, _ := l.Lookup(Field{Env: "APP_NAME"}); v.Display() != "from the environment" {
		t.Errorf("a field with no path is %q, want the environment's value", v.Display())
	}
	if v, _ := l.Lookup(Field{Path: "app.name"}); v.Display() != "from the program" {
		t.Errorf("a field with no variable is %q, want the program's value", v.Display())
	}
}

func TestLookupThroughSomethingThatIsNotATable(t *testing.T) {
	// app is a string here, so app.name names nothing, and the field falls
	// through to its default rather than the file.
	dir := project(t, map[string]string{"config/local.toml": "app = \"mizu\"\n"})
	l := open(t, Sources{Files: []string{filepath.Join(dir, "config", "local.toml")}})

	got, source := lookup(t, l, Field{Path: "app.name", Default: "from the default"})
	if got != "from the default" || source != "default" {
		t.Errorf("app.name is %q from %s, want the default", got, source)
	}
}

func TestFlagNamesFoldDashes(t *testing.T) {
	l := open(t, Sources{Args: []string{"--config.database.max-open-conns=25"}})
	got, source := lookup(t, l, Field{Path: "database.max_open_conns"})
	if got != "25" {
		t.Errorf("database.max_open_conns is %q, want 25", got)
	}
	if want := "flag --config.database.max-open-conns=25"; source != want {
		t.Errorf("it came from %q, want %q", source, want)
	}
}

func TestFlagWithoutAValue(t *testing.T) {
	_, err := Open(Sources{Args: []string{"--config.app.name"}})
	if err == nil {
		t.Fatal("no error, want one about a flag with no value")
	}
	if want := "--config.app.name needs a value"; !strings.Contains(err.Error(), want) {
		t.Errorf("error is %q, want it to mention %q", err, want)
	}
}

func TestMissingFilesAreFine(t *testing.T) {
	dir := t.TempDir()
	l := open(t, Discover(dir, nil, nil))
	if l.Env() != "local" {
		t.Errorf("the environment is %q, want local", l.Env())
	}
	if v, ok := l.Lookup(Field{Path: "app.name", Default: "mizu"}); !ok || v.Display() != "mizu" {
		t.Errorf("app.name is %q, want the default", v.Display())
	}
	if err := l.Check(); err != nil {
		t.Errorf("Check found something in a project with no files: %v", err)
	}
}

func TestBrokenFilesAreAllReported(t *testing.T) {
	dir := project(t, map[string]string{
		"config/production.toml": "app = = 1\n",
		"config/local.toml":      "[app\n",
		".env":                   "NOT A NAME=1\n",
	})
	s := Discover(dir, []string{"MIZU_ENV=production"}, nil)
	s.DotEnv = []string{filepath.Join(dir, ".env")}

	_, err := Open(s)
	if err == nil {
		t.Fatal("no error, want three")
	}
	for _, want := range []string{
		filepath.Join(dir, "config", "production.toml") + ":1:7",
		filepath.Join(dir, "config", "local.toml") + ":1:5",
		filepath.Join(dir, ".env") + ":1",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %s:\n%v", want, err)
		}
	}
}

func TestBrokenFileGivesAConfigError(t *testing.T) {
	dir := project(t, map[string]string{"config/local.toml": "app = = 1\n"})
	_, err := Open(Discover(dir, nil, nil))

	var cerr *Error
	if !errors.As(err, &cerr) {
		t.Fatalf("error is %T, want a *config.Error", err)
	}
	if cerr.Line != 1 || cerr.Col != 7 {
		t.Errorf("error is at %d:%d, want 1:7", cerr.Line, cerr.Col)
	}
	if cerr.File != filepath.Join(dir, "config", "local.toml") {
		t.Errorf("error is in %s, want config/local.toml", cerr.File)
	}
}

func TestUnreadableFile(t *testing.T) {
	dir := t.TempDir()
	// A directory where a file should be is readable in the sense that it
	// exists, and is not a file, which is the portable way to be unreadable.
	if err := os.MkdirAll(filepath.Join(dir, "config", "local.toml"), 0o777); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(Discover(dir, nil, nil)); err == nil {
		t.Fatal("no error, want one about the file")
	}
}

func TestUnreadableDotEnvFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".env"), 0o777); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(Discover(dir, nil, nil)); err == nil {
		t.Fatal("no error, want one about the .env file")
	}
}

func TestTheLastFlagWins(t *testing.T) {
	l := open(t, Sources{Args: []string{"--config.app.name=first", "serve", "--config.app.name=second"}})
	got, _ := lookup(t, l, Field{Path: "app.name"})
	if got != "second" {
		t.Errorf("app.name is %q, want second", got)
	}

	l.Lookup(Field{Path: "app.name"})
	if unknown := l.Unknown(); len(unknown) != 0 {
		t.Errorf("a flag given twice is reported %d times, want none: %v", len(unknown), unknown)
	}
}

func TestWrapLeavesOtherErrorsAlone(t *testing.T) {
	err := errors.New("not from the parser")
	if got := wrap(err); got != err {
		t.Errorf("wrap changed an error it does not know about: %v", got)
	}
}

func TestErrorUnwrap(t *testing.T) {
	dir := project(t, map[string]string{"config/local.toml": "app = = 1\n"})
	_, err := Open(Discover(dir, nil, nil))

	var terr *toml.Error
	if !errors.As(err, &terr) {
		t.Fatalf("the parser's error did not survive being wrapped: %v", err)
	}
}

func TestDisplayOfNothing(t *testing.T) {
	if got := display(&toml.Value{}); got != "" {
		t.Errorf("a value of no kind shows as %q, want empty", got)
	}
	if got := (Value{}).Display(); got != "" {
		t.Errorf("the zero value shows as %q, want empty", got)
	}
}

func TestUnknownFieldsOfAnArrayOfTables(t *testing.T) {
	dir := project(t, map[string]string{
		"config/local.toml": "[[queue.connections]]\nname = \"default\"\ntires = 3\n",
	})
	l := open(t, Sources{Files: []string{filepath.Join(dir, "config", "local.toml")}})
	l.Lookup(Field{Path: "queue.connections.name"})
	l.Lookup(Field{Path: "queue.connections.tries"})

	unknown := l.Unknown()
	if len(unknown) != 1 {
		t.Fatalf("found %d unknown settings, want 1: %v", len(unknown), unknown)
	}
	want := []string{"queue.connections.tries"}
	if unknown[0].Path != "queue.connections.tires" || !slices.Equal(unknown[0].Near, want) {
		t.Errorf("found %q suggesting %v, want the misspelled tires suggesting tries", unknown[0].Path, unknown[0].Near)
	}
}

func TestDotEnvExpandsAgainstTheEnvironment(t *testing.T) {
	dir := project(t, map[string]string{
		".env": "DATABASE_URL=postgres://${DB_HOST:-localhost}:5432/app\n",
	})
	s := Sources{DotEnv: []string{filepath.Join(dir, ".env")}, Environ: []string{"DB_HOST=db.internal"}}

	got, _ := lookup(t, open(t, s), Field{Env: "DATABASE_URL"})
	if want := "postgres://db.internal:5432/app"; got != want {
		t.Errorf("DATABASE_URL is %q, want %q", got, want)
	}
}

func TestLaterDotEnvFilesWin(t *testing.T) {
	dir := project(t, map[string]string{
		".env":       "A=one\nB=one\n",
		".env.local": "B=two\nC=${A}-two\n",
	})
	l := open(t, Discover(dir, nil, nil))

	for _, tt := range []struct{ name, want string }{{"A", "one"}, {"B", "two"}, {"C", "one-two"}} {
		if got, _ := lookup(t, l, Field{Env: tt.name}); got != tt.want {
			t.Errorf("%s is %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestUnknownSettings(t *testing.T) {
	dir := project(t, map[string]string{
		"config/local.toml": `
[app]
name = "mizu"
nmae = "typo"

[database]
dsn = "postgres://localhost/app"
max_conns = 25

[cach]
store = "redis"
ttl = "1h"
`,
	})
	l := open(t, Sources{
		Files: []string{filepath.Join(dir, "config", "local.toml")},
		Args:  []string{"--config.app.name=mizu", "--config.app.nmae=typo"},
	})
	for _, path := range []string{"app.name", "database.dsn", "database.max_open_conns", "cache.store"} {
		l.Lookup(Field{Path: path})
	}

	want := []string{
		`app.nmae"`,
		`did you mean "app.name"?`,
		`database.max_conns"`,
		`did you mean "database.max_open_conns"?`,
		`"cach"`,
		`did you mean "cache"?`,
	}
	err := l.Check()
	if err == nil {
		t.Fatal("Check found nothing, want three unknown settings")
	}
	for _, w := range want {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("the report does not mention %s:\n%v", w, err)
		}
	}

	unknown := l.Unknown()
	if len(unknown) != 4 {
		t.Fatalf("found %d unknown settings, want 4: %v", len(unknown), unknown)
	}
	// The misspelled table is one mistake and is reported once, not once for
	// each of the two keys inside it.
	if unknown[2].Path != "cach" {
		t.Errorf("the third is %q, want the table cach", unknown[2].Path)
	}
	if unknown[2].From.From != FromFile {
		t.Errorf("the third came from %s, want a file", unknown[2].From)
	}
	// The flag comes after the files, whatever order it was given in.
	if unknown[3].Path != "app.nmae" || unknown[3].From.From != FromFlag {
		t.Errorf("the fourth is %q from %s, want app.nmae from a flag", unknown[3].Path, unknown[3].From)
	}
}

func TestUnknownSettingsWithNothingClose(t *testing.T) {
	dir := project(t, map[string]string{"config/local.toml": "wildly_different = 1\n"})
	l := open(t, Sources{Files: []string{filepath.Join(dir, "config", "local.toml")}})
	l.Lookup(Field{Path: "app.name"})

	unknown := l.Unknown()
	if len(unknown) != 1 {
		t.Fatalf("found %d unknown settings, want 1", len(unknown))
	}
	if len(unknown[0].Near) != 0 {
		t.Errorf("it suggested %v, and nothing there is close enough to suggest", unknown[0].Near)
	}
	if want := "unknown setting \"wildly_different\""; !strings.Contains(unknown[0].Error(), want) {
		t.Errorf("the report is %q, want it to say %q with no suggestion", unknown[0].Error(), want)
	}
}

func TestAskingForATableTakesEverythingUnderIt(t *testing.T) {
	dir := project(t, map[string]string{
		"config/local.toml": "[storage.disks.s3]\nbucket = \"b\"\nregion = \"r\"\n",
	})
	l := open(t, Sources{Files: []string{filepath.Join(dir, "config", "local.toml")}})
	l.Lookup(Field{Path: "storage.disks"})

	if err := l.Check(); err != nil {
		t.Errorf("Check reported something under a table that was asked for: %v", err)
	}
}

func TestArraysOfTables(t *testing.T) {
	dir := project(t, map[string]string{
		"config/local.toml": "[[queue.connections]]\nname = \"default\"\n\n[[queue.connections]]\nname = \"emails\"\n",
	})
	l := open(t, Sources{Files: []string{filepath.Join(dir, "config", "local.toml")}})

	v, ok := l.Lookup(Field{Path: "queue.connections"})
	if !ok || v.TOML == nil || len(v.TOML.Array) != 2 {
		t.Fatalf("queue.connections is %v, want an array of two tables", v.Display())
	}
	if err := l.Check(); err != nil {
		t.Errorf("Check reported something inside an array that was asked for: %v", err)
	}
}

func TestSettingsAreRecordedInOrder(t *testing.T) {
	dir := project(t, map[string]string{"config/local.toml": "[app]\nname = \"mizu\"\n"})
	l := open(t, Sources{Files: []string{filepath.Join(dir, "config", "local.toml")}})

	l.Lookup(Field{Path: "app.name", Env: "APP_NAME"})
	l.Lookup(Field{Path: "app.key", Env: "APP_KEY", Secret: true, Default: "shh"})
	l.Lookup(Field{Path: "app.debug", Env: "APP_DEBUG"})

	settings := l.Settings()
	if len(settings) != 3 {
		t.Fatalf("recorded %d settings, want 3", len(settings))
	}
	for i, want := range []struct {
		path    string
		display string
		set     bool
	}{
		{"app.name", "mizu", true},
		{"app.key", "***", true},
		{"app.debug", "", false},
	} {
		got := settings[i]
		if got.Path != want.path || got.Display() != want.display || got.Set != want.set {
			t.Errorf("setting %d is %q=%q set=%v, want %q=%q set=%v",
				i, got.Path, got.Display(), got.Set, want.path, want.display, want.set)
		}
	}
	if settings[1].Value.Display() != "shh" {
		t.Errorf("the secret itself is %q, and Value.Display is meant to be the real one", settings[1].Value.Display())
	}
}

func TestSettingsIsACopy(t *testing.T) {
	l := open(t, Sources{})
	l.Lookup(Field{Path: "app.name", Default: "mizu"})
	settings := l.Settings()
	settings[0].Path = "changed"
	if got := l.Settings()[0].Path; got != "app.name" {
		t.Errorf("the first setting is now %q, so Settings handed out the loader's own slice", got)
	}
}

func TestDisplay(t *testing.T) {
	dir := project(t, map[string]string{
		"config/local.toml": `
text = "a string"
count = 25
ratio = 1.5
enabled = true
at = 1979-05-27T07:32:00Z
stamp = 1979-05-27T07:32:00.5
day = 1979-05-27
clock = 07:32:00
list = ["a", 1, true]
empty = []
inline = { name = "x", port = 8080 }
`,
	})
	l := open(t, Sources{Files: []string{filepath.Join(dir, "config", "local.toml")}})

	tests := []struct{ path, want string }{
		{"text", "a string"},
		{"count", "25"},
		{"ratio", "1.5"},
		{"enabled", "true"},
		{"at", "1979-05-27T07:32:00Z"},
		{"stamp", "1979-05-27T07:32:00.5"},
		{"day", "1979-05-27"},
		{"clock", "07:32:00"},
		{"list", "[a, 1, true]"},
		{"empty", "[]"},
		{"inline", "{name = x, port = 8080}"},
	}
	for _, tt := range tests {
		if got, _ := lookup(t, l, Field{Path: tt.path}); got != tt.want {
			t.Errorf("%s shows as %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestFromString(t *testing.T) {
	tests := []struct {
		from From
		want string
	}{
		{FromDefault, "default"},
		{FromFile, "file"},
		{FromDotEnv, "dotenv"},
		{FromEnv, "env"},
		{FromFlag, "flag"},
		{FromOverride, "override"},
		{FromComputed, "computed"},
		{From(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.from.String(); got != tt.want {
			t.Errorf("From(%d) is %q, want %q", int(tt.from), got, tt.want)
		}
	}
}

func TestSourceString(t *testing.T) {
	tests := []struct {
		source Source
		want   string
	}{
		{Source{From: FromDefault}, "default"},
		{Source{From: FromFile, Name: "config/local.toml:3:8"}, "file config/local.toml:3:8"},
		{Source{From: FromEnv, Name: "APP_NAME"}, "env APP_NAME"},
		{Source{From: FromComputed, Name: "GOMAXPROCS*4"}, "computed GOMAXPROCS*4"},
	}
	for _, tt := range tests {
		if got := tt.source.String(); got != tt.want {
			t.Errorf("%#v is %q, want %q", tt.source, got, tt.want)
		}
	}
}

func TestErrorMessage(t *testing.T) {
	tests := []struct {
		err  *Error
		want string
	}{
		{&Error{File: "a.toml", Line: 2, Col: 5, Msg: "no"}, "a.toml:2:5: no"},
		{&Error{File: ".env", Line: 2, Msg: "no"}, ".env:2: no"},
		{&Error{File: ".env", Msg: "no"}, ".env: no"},
		{&Error{Msg: "no"}, "no"},
	}
	for _, tt := range tests {
		if got := tt.err.Error(); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestUnknownMessage(t *testing.T) {
	tests := []struct {
		unknown Unknown
		want    string
	}{
		{
			Unknown{Path: "database.max_conns", From: Source{From: FromFile, Name: "config/production.toml:14:1"}, Near: []string{"database.max_open_conns"}},
			`file config/production.toml:14:1: unknown setting "database.max_conns", did you mean "database.max_open_conns"?`,
		},
		{
			Unknown{
				Path: "database.max_conns",
				From: Source{From: FromFile, Name: "config/production.toml:14:1"},
				Near: []string{"database.max_open_conns", "database.max_idle_conns"},
			},
			`file config/production.toml:14:1: unknown setting "database.max_conns", did you mean "database.max_open_conns" or "database.max_idle_conns"?`,
		},
		{
			Unknown{Path: "nope", From: Source{From: FromFlag, Name: "--config.nope=1"}},
			`flag --config.nope=1: unknown setting "nope"`,
		},
	}
	for _, tt := range tests {
		if got := tt.unknown.Error(); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}
