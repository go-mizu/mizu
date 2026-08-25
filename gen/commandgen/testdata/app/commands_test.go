package app

import (
	"context"
	"io"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/console"
)

// parse runs a command line through the generated spec, which is the round trip
// the generator exists for: tags in, fields out.
func parse(t *testing.T, cmd console.Command, argv ...string) {
	t.Helper()
	spec := cmd.Spec()
	if err := console.Parse(spec.Flags, spec.Args, argv); err != nil {
		t.Fatalf("%s %s: %v", spec.Name, strings.Join(argv, " "), err)
	}
}

// refuse is the same thing for a command line that should not be accepted.
func refuse(t *testing.T, cmd console.Command, argv ...string) string {
	t.Helper()
	spec := cmd.Spec()
	err := console.Parse(spec.Flags, spec.Args, argv)
	if err == nil {
		t.Fatalf("%s %s was accepted", spec.Name, strings.Join(argv, " "))
	}
	return err.Error()
}

func TestUsersPrune(t *testing.T) {
	c := new(UsersPrune)
	parse(t, c, "--days", "7", "--dry-run", "-f", "json", "acme")

	if c.Tenant != "acme" {
		t.Errorf("Tenant = %q, want acme", c.Tenant)
	}
	if c.Days != 7 {
		t.Errorf("Days = %d, want 7", c.Days)
	}
	if !c.DryRun {
		t.Error("DryRun is off")
	}
	if c.Format != "json" {
		t.Errorf("Format = %q, want json", c.Format)
	}
	if c.Wait != 5*time.Second {
		t.Errorf("Wait = %v, want the 5s default", c.Wait)
	}
}

func TestDefaultsAndEnvironment(t *testing.T) {
	t.Setenv("MIZU_PRUNE_WAIT", "90s")

	c := new(UsersPrune)
	parse(t, c, "acme")

	if c.Days != 30 {
		t.Errorf("Days = %d, want the 30 default", c.Days)
	}
	if c.Format != "text" {
		t.Errorf("Format = %q, want the text default", c.Format)
	}
	if c.Wait != 90*time.Second {
		t.Errorf("Wait = %v, want the 90s from the environment", c.Wait)
	}
}

func TestRequiredArgument(t *testing.T) {
	if msg := refuse(t, new(UsersPrune)); !strings.Contains(msg, "tenant") {
		t.Errorf("the error does not name the argument: %s", msg)
	}
}

func TestEnum(t *testing.T) {
	msg := refuse(t, new(UsersPrune), "--format", "yaml", "acme")
	for _, want := range []string{"yaml", "text", "json"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the error does not mention %q: %s", want, msg)
		}
	}
}

func TestDbWipe(t *testing.T) {
	c := new(DbWipe)
	parse(t, c, "-lll", "--force", "--url", "postgres://localhost/app")

	if c.Loud != 3 {
		t.Errorf("Loud = %d, want 3", c.Loud)
	}
	if !c.Force {
		t.Error("Force is off")
	}
	if c.URL != "postgres://localhost/app" {
		t.Errorf("URL = %q", c.URL)
	}

	if msg := refuse(t, new(DbWipe)); !strings.Contains(msg, "--url") {
		t.Errorf("the error does not name the required flag: %s", msg)
	}
}

func TestServe(t *testing.T) {
	c := new(Serve)
	parse(t, c,
		"--bind", "0.0.0.0", "-p", "9000", "--sample", "0.5", "-t", "5s",
		"--built", "2026-01-02", "-o", "a.example", "-o", "b.example",
		"--redirect", "80,443", "-H", "X-One=1", "-H", "X-Two=2",
		"--include", "a,b.conf")

	if want := netip.MustParseAddr("0.0.0.0"); c.Bind != want {
		t.Errorf("Bind = %v, want %v", c.Bind, want)
	}
	if c.Port != 9000 {
		t.Errorf("Port = %d, want 9000", c.Port)
	}
	if c.Sample != 0.5 {
		t.Errorf("Sample = %v, want 0.5", c.Sample)
	}
	if c.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", c.Timeout)
	}
	if want := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC); !c.Built.Equal(want) {
		t.Errorf("Built = %v, want %v", c.Built, want)
	}
	if strings.Join(c.Origins, " ") != "a.example b.example" {
		t.Errorf("Origins = %v", c.Origins)
	}
	if len(c.Redirect) != 2 || c.Redirect[0] != 80 || c.Redirect[1] != 443 {
		t.Errorf("Redirect = %v, want [80 443]", c.Redirect)
	}
	if c.Header["X-One"] != "1" || c.Header["X-Two"] != "2" {
		t.Errorf("Header = %v", c.Header)
	}

	// sep:"" is what keeps a comma inside a value rather than between two.
	if len(c.Include) != 1 || c.Include[0] != "a,b.conf" {
		t.Errorf("Include = %v, want one name with a comma in it", c.Include)
	}
}

// TestListElementErrors checks that a list reports a bad element the way a
// single value of the same type would, which is why the parsers are exported.
func TestListElementErrors(t *testing.T) {
	msg := refuse(t, new(Serve), "--redirect", "80,https")
	if !strings.Contains(msg, "https") {
		t.Errorf("the error does not name the value: %s", msg)
	}
	if strings.Contains(msg, "ParseUint") {
		t.Errorf("the error is about strconv rather than about the flag: %s", msg)
	}
}

func TestDeployArguments(t *testing.T) {
	c := new(Deploy)
	parse(t, c, "staging")

	if c.Target != "staging" {
		t.Errorf("Target = %q", c.Target)
	}
	if c.Ref != "HEAD" {
		t.Errorf("Ref = %q, want the HEAD default", c.Ref)
	}
	if len(c.Services) != 0 {
		t.Errorf("Services = %v, want none", c.Services)
	}

	all := new(Deploy)
	parse(t, all, "production", "v2.1.0", "api", "worker", "cron")
	if strings.Join(all.Services, " ") != "api worker cron" {
		t.Errorf("Services = %v", all.Services)
	}
}

// TestApp runs the commands the way a binary does, which is the other half of
// what the generator produces: a list to hand to console.App.
func TestApp(t *testing.T) {
	a := &console.App{Name: "commandtest"}
	a.Add(Commands()...)

	var out, errs strings.Builder
	code := a.Start(context.Background(), strings.NewReader(""), &out, &errs, []string{"deploy", "staging"})
	if code != console.CodeOK {
		t.Fatalf("exit %d\n%s", code, errs.String())
	}
	if !strings.Contains(out.String(), `"Target": "staging"`) {
		t.Errorf("the command did not print what it parsed:\n%s", out.String())
	}

	out.Reset()
	if code := a.Start(context.Background(), strings.NewReader(""), &out, io.Discard, []string{"--help"}); code != console.CodeOK {
		t.Fatalf("help exited %d", code)
	}
	help := out.String()
	for _, want := range []string{"users:prune", "serve", "deploy", "Runs the HTTP server"} {
		if !strings.Contains(help, want) {
			t.Errorf("help does not mention %q:\n%s", want, help)
		}
	}
	if strings.Contains(help, "db:wipe") {
		t.Errorf("the hidden command is in help:\n%s", help)
	}
}
