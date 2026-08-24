package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	dir := project(t, map[string]string{
		"config/local.toml": `
[app]
name = "shop"

[database]
max_open_conns = 25
timeout = "30s"
hosts = ["a", "b"]
`,
	})
	l := open(t, Sources{
		Files:   []string{filepath.Join(dir, "config", "local.toml")},
		Environ: []string{"APP_DEBUG=true"},
	})

	var c struct {
		Name    string
		Debug   bool
		Conns   int
		Timeout time.Duration
		Hosts   []string
		Missing string
	}
	c.Missing = "left alone"

	Get(l, &c.Name, Field{Name: "App.Name", Path: "app.name"}, String)
	Get(l, &c.Debug, Field{Name: "App.Debug", Path: "app.debug", Env: "APP_DEBUG"}, Bool)
	Get(l, &c.Conns, Field{Name: "DB.MaxOpenConns", Path: "database.max_open_conns"}, Int)
	Get(l, &c.Timeout, Field{Name: "DB.Timeout", Path: "database.timeout"}, Duration)
	Get(l, &c.Hosts, Field{Name: "DB.Hosts", Path: "database.hosts"}, Slice(String[string]))
	Get(l, &c.Missing, Field{Name: "App.Missing", Path: "app.missing"}, String)

	if err := l.Err(); err != nil {
		t.Fatal(err)
	}
	if c.Name != "shop" {
		t.Errorf("Name is %q", c.Name)
	}
	if !c.Debug {
		t.Error("Debug is false")
	}
	if c.Conns != 25 {
		t.Errorf("Conns is %d", c.Conns)
	}
	if c.Timeout != 30*time.Second {
		t.Errorf("Timeout is %v", c.Timeout)
	}
	if len(c.Hosts) != 2 || c.Hosts[0] != "a" {
		t.Errorf("Hosts are %q", c.Hosts)
	}
	// Nothing had a value for it, so what the struct came with stands.
	if c.Missing != "left alone" {
		t.Errorf("Missing is %q, want it untouched", c.Missing)
	}

	// Every field was recorded, whether it was set or not.
	if got := len(l.Settings()); got != 6 {
		t.Errorf("%d settings recorded, want 6", got)
	}
}

func TestGetCollectsErrors(t *testing.T) {
	dir := project(t, map[string]string{
		"config/local.toml": "[database]\nmax_open_conns = \"lots\"\ntimeout = \"soon\"\n",
	})
	l := open(t, Sources{Files: []string{filepath.Join(dir, "config", "local.toml")}})

	var conns int
	var timeout time.Duration
	var name string

	Get(l, &conns, Field{Name: "DB.MaxOpenConns", Path: "database.max_open_conns"}, Int)
	Get(l, &timeout, Field{Name: "DB.Timeout", Path: "database.timeout"}, Duration)
	Get(l, &name, Field{Name: "App.Name", Path: "app.name", Default: "shop"}, String)

	// Both bad fields are reported, not only the first one, and the good one
	// that came after them was still read.
	errs := l.Errors()
	if len(errs) != 2 {
		t.Fatalf("%d errors, want 2: %v", len(errs), errs)
	}
	if name != "shop" {
		t.Errorf("Name is %q, want the default to have been read anyway", name)
	}
	if conns != 0 || timeout != 0 {
		t.Errorf("a field that would not read was written to: %d, %v", conns, timeout)
	}

	err := l.Err()
	if err == nil {
		t.Fatal("Err is nil")
	}
	for _, want := range []string{
		"DB.MaxOpenConns",
		"DB.Timeout",
		"want an integer, got string",
		"want a length of time",
		filepath.Join(dir, "config", "local.toml") + ":2:18",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Err is %q, want it to mention %q", err, want)
		}
	}

	var fe *FieldError
	if !errors.As(err, &fe) {
		t.Fatal("Err does not unwrap to a *FieldError")
	}
	if fe.Field.Name != "DB.MaxOpenConns" {
		t.Errorf("the first error is about %q", fe.Field.Name)
	}
	if fe.Unwrap() == nil {
		t.Error("a FieldError wraps nothing")
	}
}

func TestErrIsNilWhenEverythingReads(t *testing.T) {
	l := open(t, Sources{})
	var name string
	Get(l, &name, Field{Path: "app.name", Default: "shop"}, String)
	if err := l.Err(); err != nil {
		t.Fatalf("Err is %v, want nil", err)
	}
	if len(l.Errors()) != 0 {
		t.Errorf("Errors is %v, want nothing", l.Errors())
	}
}

func TestFieldErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  *FieldError
		want string
	}{
		{
			name: "a file says where it is",
			err: &FieldError{
				Field: Field{Name: "DB.Port", Path: "database.port"},
				Value: Value{Source: Source{From: FromFile, Name: "config/local.toml:3:8"}},
				Err:   errors.New("nope"),
			},
			want: "file config/local.toml:3:8: DB.Port: nope",
		},
		{
			name: "a variable says which one",
			err: &FieldError{
				Field: Field{Name: "DB.Port", Path: "database.port"},
				Value: Value{Source: Source{From: FromEnv, Name: "DB_PORT"}},
				Err:   errors.New("nope"),
			},
			want: "env DB_PORT: DB.Port: nope",
		},
		{
			name: "the path stands in for a missing name",
			err: &FieldError{
				Field: Field{Path: "database.port"},
				Value: Value{Source: Source{From: FromFlag, Name: "--config.database.port=x"}},
				Err:   errors.New("nope"),
			},
			want: "flag --config.database.port=x: database.port: nope",
		},
		{
			// A bad default is the program's own mistake, and there is nowhere
			// to send the reader but the code.
			name: "a default has nowhere to point",
			err: &FieldError{
				Field: Field{Name: "DB.Port"},
				Value: Value{Source: Source{From: FromDefault}},
				Err:   errors.New("nope"),
			},
			want: "DB.Port: nope",
		},
		{
			name: "nothing but the message",
			err:  &FieldError{Value: Value{Source: Source{From: FromDefault}}, Err: errors.New("nope")},
			want: "nope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSecretFromFile(t *testing.T) {
	dir := t.TempDir()
	secret := filepath.Join(dir, "db-password")
	if err := os.WriteFile(secret, []byte("hunter2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	l := open(t, Sources{Environ: []string{"DB_PASSWORD=file:" + secret}})
	field := Field{Name: "DB.Password", Path: "database.password", Env: "DB_PASSWORD", Secret: true}

	var got string
	Get(l, &got, field, String)
	if err := l.Err(); err != nil {
		t.Fatal(err)
	}
	// The trailing newline every text editor adds is not part of the secret.
	if got != "hunter2" {
		t.Errorf("read as %q, want hunter2", got)
	}

	// What config:show prints is the pointer and not what it points at.
	s := l.Settings()[0]
	if s.Value.Text != "file:"+secret {
		t.Errorf("the recorded value is %q, want the file: form", s.Value.Text)
	}
	if s.Display() != "***" {
		t.Errorf("a secret displays as %q", s.Display())
	}
}

func TestSecretFromMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-there")
	l := open(t, Sources{Environ: []string{"DB_PASSWORD=file:" + missing}})

	var got string
	Get(l, &got, Field{Name: "DB.Password", Env: "DB_PASSWORD", Secret: true}, String)

	err := l.Err()
	if err == nil {
		t.Fatal("a missing file was accepted")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Err is %v, want it to be about a file that is not there", err)
	}
	if got != "" {
		t.Errorf("read as %q, want nothing", got)
	}
}

func TestSecretFromEnv(t *testing.T) {
	dir := project(t, map[string]string{".env": "FROM_DOTENV=in the dotenv\n"})

	l := open(t, Sources{
		DotEnv:  []string{filepath.Join(dir, ".env")},
		Environ: []string{"REAL_PASSWORD=hunter2"},
		Override: map[string]string{
			"a.password": "env:REAL_PASSWORD",
			"b.password": "env:FROM_DOTENV",
			"c.password": "env:NOT_SET",
		},
	})

	var a, b, c string
	Get(l, &a, Field{Name: "A.Password", Path: "a.password", Secret: true}, String)
	Get(l, &b, Field{Name: "B.Password", Path: "b.password", Secret: true}, String)
	Get(l, &c, Field{Name: "C.Password", Path: "c.password", Secret: true}, String)

	if a != "hunter2" {
		t.Errorf("A is %q", a)
	}
	// A .env file counts, because it is where a developer keeps these.
	if b != "in the dotenv" {
		t.Errorf("B is %q", b)
	}

	errs := l.Errors()
	if len(errs) != 1 {
		t.Fatalf("%d errors, want 1: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Error(), "NOT_SET is not set") {
		t.Errorf("the error is %q", errs[0])
	}
}

func TestSecretFromCommand(t *testing.T) {
	// The core never starts a process. A caller that wants cmd: to work says
	// what running a command means.
	asked := ""
	l := open(t, Sources{
		Override: map[string]string{"a.password": "cmd:vault read db"},
		Command: func(name string) (string, error) {
			asked = name
			return "hunter2\n", nil
		},
	})

	var got string
	Get(l, &got, Field{Name: "A.Password", Path: "a.password", Secret: true}, String)
	if err := l.Err(); err != nil {
		t.Fatal(err)
	}
	if asked != "vault read db" {
		t.Errorf("the command was %q", asked)
	}
	if got != "hunter2" {
		t.Errorf("read as %q", got)
	}
}

func TestSecretFromCommandFails(t *testing.T) {
	broken := errors.New("vault is asleep")
	l := open(t, Sources{
		Override: map[string]string{"a.password": "cmd:vault read db"},
		Command:  func(string) (string, error) { return "", broken },
	})

	var got string
	Get(l, &got, Field{Name: "A.Password", Path: "a.password", Secret: true}, String)
	if err := l.Err(); !errors.Is(err, broken) {
		t.Errorf("Err is %v, want the command's own error", err)
	}
}

func TestSecretFromCommandRefused(t *testing.T) {
	// No Command, so a cmd: value is refused rather than run.
	l := open(t, Sources{Override: map[string]string{"a.password": "cmd:rm -rf /"}})

	var got string
	Get(l, &got, Field{Name: "A.Password", Path: "a.password", Secret: true}, String)

	err := l.Err()
	if err == nil {
		t.Fatal("a cmd: value was accepted with no way to run it")
	}
	if !strings.Contains(err.Error(), "turned off") {
		t.Errorf("Err is %q", err)
	}
	if got != "" {
		t.Errorf("read as %q, want nothing", got)
	}
}

func TestIndirectionOnlyForSecrets(t *testing.T) {
	// file: is a real scheme, and a DSN that names a file is a real DSN, so a
	// field that is not a secret keeps whatever it was given.
	l := open(t, Sources{Environ: []string{"DATABASE_URL=file:/tmp/app.sqlite"}})

	var dsn string
	Get(l, &dsn, Field{Name: "DB.DSN", Env: "DATABASE_URL"}, String)
	if err := l.Err(); err != nil {
		t.Fatal(err)
	}
	if dsn != "file:/tmp/app.sqlite" {
		t.Errorf("read as %q, want it left alone", dsn)
	}
}

func TestSecretWithoutIndirection(t *testing.T) {
	dir := project(t, map[string]string{"config/local.toml": "[database]\nport = 5432\n"})
	l := open(t, Sources{
		Files:    []string{filepath.Join(dir, "config", "local.toml")},
		Environ:  []string{"EMPTY="},
		Override: map[string]string{"a.password": "hunter2"},
	})

	// A plain secret, a secret with nothing in it, and a secret that is not
	// text at all: none of them go looking anywhere else.
	var plain, empty string
	var port int
	Get(l, &plain, Field{Path: "a.password", Secret: true}, String)
	Get(l, &empty, Field{Env: "EMPTY", Secret: true}, String)
	Get(l, &port, Field{Path: "database.port", Secret: true}, Int)

	if err := l.Err(); err != nil {
		t.Fatal(err)
	}
	if plain != "hunter2" || empty != "" || port != 5432 {
		t.Errorf("read as %q, %q and %d", plain, empty, port)
	}
}

func TestValueErrorf(t *testing.T) {
	dir := project(t, map[string]string{"config/local.toml": "[database]\nport = 5432\n"})
	file := filepath.Join(dir, "config", "local.toml")
	l := open(t, Sources{
		Files:   []string{file},
		Environ: []string{"DB_PORT=5432"},
	})

	v, _ := l.Lookup(Field{Path: "database.port"})
	if got, want := v.Errorf("no good: %d", 1).Error(), file+":2:8: no good: 1"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	v, _ = l.Lookup(Field{Env: "DB_PORT"})
	if got, want := v.Errorf("no good").Error(), "env DB_PORT: no good"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
