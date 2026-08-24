package config

import (
	"log/slog"
	"net/netip"
	"strings"
	"testing"
	"time"
)

type appEnv string

func TestShow(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"a string", Show("shop"), "shop"},
		{"a named string", Show(appEnv("production")), "production"},
		{"a boolean", Show(true), "true"},
		{"an integer", Show(25), "25"},
		{"an unsigned integer", Show(uint16(5432)), "5432"},
		{"a float", Show(0.25), "0.25"},
		{"a length of time", Show(30 * time.Second), "30s"},
		{"a level", Show(slog.LevelInfo), "INFO"},
		{"a network", Show(netip.MustParsePrefix("10.0.0.0/8")), "10.0.0.0/8"},
		{"an address", Show(netip.MustParseAddr("::1")), "::1"},
		{"bytes", Show([]byte("hello")), "aGVsbG8="},
		{"no bytes", Show([]byte(nil)), ""},
		{"a moment", Show(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)), "2026-01-02T03:04:05Z"},
		{"nothing", Show(""), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestShowSlice(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"strings", ShowSlice([]string{"a", "b"}), "[a, b]"},
		{"one", ShowSlice([]string{"a"}), "[a]"},
		{"none", ShowSlice([]string(nil)), "[]"},
		{"networks", ShowSlice([]netip.Prefix{netip.MustParsePrefix("::1/128")}), "[::1/128]"},
		{"times", ShowSlice([]time.Duration{time.Second, time.Minute}), "[1s, 1m0s]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestShowMap(t *testing.T) {
	// In key order, so two runs of config:show agree with each other.
	got := ShowMap(map[string]int{"z": 3, "a": 1, "m": 2})
	if want := "{a = 1, m = 2, z = 3}"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := ShowMap(map[string]string(nil)); got != "{}" {
		t.Errorf("an empty map is %q", got)
	}
}

// docs is what generated code holds: the part of a setting that comes from the
// struct and never changes.
var docs = []FieldDoc{
	{Field: Field{Name: "App.Name", Path: "app.name", Env: "APP_NAME"}, Type: "string", Doc: "Name is what the application calls itself."},
	{Field: Field{Name: "App.Debug", Path: "app.debug", Env: "APP_DEBUG"}, Type: "bool"},
	{Field: Field{Name: "DB.DSN", Path: "database.dsn", Env: "DATABASE_URL", Secret: true}, Type: "string"},
	{Field: Field{Name: "DB.Password", Path: "database.password", Secret: true}, Type: "string"},
}

func TestDescribe(t *testing.T) {
	got := Describe(docs, []string{"shop", "false", "postgres://localhost/app", ""})
	if len(got) != 4 {
		t.Fatalf("%d fields, want 4", len(got))
	}
	if got[0].Value != "shop" || got[0].Doc != "Name is what the application calls itself." {
		t.Errorf("the first field is %+v", got[0])
	}
	if got[2].Value != Redacted {
		t.Errorf("a secret that is set shows as %q", got[2].Value)
	}
	// A secret that is not set says so, since knowing one is missing is the
	// reason to look.
	if got[3].Value != "" {
		t.Errorf("a secret that is not set shows as %q", got[3].Value)
	}

	// The values a caller passed in are not written back over the generated
	// ones, which are shared by every Config in the process.
	if docs[0].Value != "" {
		t.Errorf("Describe wrote %q back into the generated fields", docs[0].Value)
	}
}

func TestDescribeShortValues(t *testing.T) {
	// A caller with fewer values than fields gets what there is, rather than a
	// panic. The two come from the same generated walk, so this is only a
	// guard against somebody calling it by hand.
	got := Describe(docs, []string{"shop"})
	if len(got) != 4 || got[0].Value != "shop" || got[1].Value != "" {
		t.Errorf("got %+v", got)
	}
}

func TestDiff(t *testing.T) {
	from := []string{"shop", "false", "postgres://a/app", ""}
	to := []string{"shop", "true", "postgres://b/app", "hunter2"}

	got := Diff(docs, from, to)
	if len(got) != 3 {
		t.Fatalf("%d changes, want 3: %v", len(got), got)
	}
	if got[0].Path != "app.debug" || got[0].From != "false" || got[0].To != "true" {
		t.Errorf("the first change is %v", got[0])
	}
	if got[0].String() != "app.debug: false -> true" {
		t.Errorf("a change reads as %q", got[0])
	}
	// A secret that changed is reported as changed, and neither value is
	// printed.
	if got[1].From != Redacted || got[1].To != Redacted {
		t.Errorf("a secret change is %v", got[1])
	}
	// One that was not set and now is says that much.
	if got[2].From != "" || got[2].To != Redacted {
		t.Errorf("a secret that appeared is %v", got[2])
	}
}

func TestDiffFindsNothing(t *testing.T) {
	same := []string{"shop", "false", "", ""}
	if got := Diff(docs, same, same); got != nil {
		t.Errorf("two identical configurations differ by %v", got)
	}
	// Short input stops rather than reading past the end.
	if got := Diff(docs, []string{"shop"}, []string{"other"}); len(got) != 1 {
		t.Errorf("got %v, want the one field there were values for", got)
	}
}

func TestDescribeReadsAsATable(t *testing.T) {
	// What config:doc prints, near enough, so the parts line up.
	var b strings.Builder
	for _, f := range Describe(docs, []string{"shop", "false", "postgres://localhost/app", ""}) {
		b.WriteString(f.Name + "\t" + f.Type + "\t" + f.Env + "\t" + f.Value + "\n")
	}
	want := "App.Name\tstring\tAPP_NAME\tshop\n" +
		"App.Debug\tbool\tAPP_DEBUG\tfalse\n" +
		"DB.DSN\tstring\tDATABASE_URL\t***\n" +
		"DB.Password\tstring\t\t\n"
	if b.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", b.String(), want)
	}
}
