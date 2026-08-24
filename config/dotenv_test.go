package config

import (
	"errors"
	"strings"
	"testing"
)

// env is the process environment a .env file is expanded against in these
// tests. Nothing here reads the real one.
func env(name string) (string, bool) {
	v, ok := map[string]string{"HOST": "db.internal", "EMPTY": ""}[name]
	return v, ok
}

func TestDotEnv(t *testing.T) {
	src := "" +
		"# a comment line\n" +
		"\n" +
		"FOO=bar\n" +
		"export EXPORTED=yes\n" +
		"SPACED   =   value with spaces   \n" +
		"EMPTY=\n" +
		"QUOTED=\"one two\"\n" +
		"LITERAL='no ${FOO} here'\n" +
		"ESCAPES=\"a\\nb\\tc\\\\d\\\"e\\qf\"\n" +
		"MULTI=\"first\nsecond\"\n" +
		"COMMENTED=value # not part of it\n" +
		"HASH=value#part-of-it\n" +
		"URL=postgres://user:pass@localhost:5432/db?sslmode=disable\n" +
		"EXPAND=${FOO}/x\n" +
		"CHAIN=${EXPAND}/y\n" +
		"FROMENV=${HOST}\n" +
		"FALLBACK=${NOPE:-a default}\n" +
		"FALLBACK_EMPTY=${EMPTY:-used}\n" +
		"MISSING=[${NOPE}]\n" +
		"DOLLAR=pa$$word\n" +
		"AFTER_QUOTE=\"x\" # a comment\n"

	want := []struct {
		name  string
		value string
		line  int
	}{
		{"FOO", "bar", 3},
		{"EXPORTED", "yes", 4},
		{"SPACED", "value with spaces", 5},
		{"EMPTY", "", 6},
		{"QUOTED", "one two", 7},
		{"LITERAL", "no ${FOO} here", 8},
		{"ESCAPES", "a\nb\tc\\d\"e\\qf", 9},
		{"MULTI", "first\nsecond", 10},
		{"COMMENTED", "value", 12},
		{"HASH", "value#part-of-it", 13},
		{"URL", "postgres://user:pass@localhost:5432/db?sslmode=disable", 14},
		{"EXPAND", "bar/x", 15},
		{"CHAIN", "bar/x/y", 16},
		{"FROMENV", "db.internal", 17},
		{"FALLBACK", "a default", 18},
		{"FALLBACK_EMPTY", "used", 19},
		{"MISSING", "[]", 20},
		{"DOLLAR", "pa$$word", 21},
		{"AFTER_QUOTE", "x", 22},
	}

	vars, err := parseDotEnv(".env", []byte(src), env)
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != len(want) {
		t.Fatalf("read %d variables, want %d: %v", len(vars), len(want), vars)
	}
	for i, w := range want {
		got := vars[i]
		if got.name != w.name || got.value != w.value || got.line != w.line {
			t.Errorf("variable %d is %q=%q on line %d, want %q=%q on line %d",
				i, got.name, got.value, got.line, w.name, w.value, w.line)
		}
	}
}

func TestDotEnvSingleQuotesRunOverLines(t *testing.T) {
	vars, err := parseDotEnv(".env", []byte("KEY='-----BEGIN KEY-----\nabc\n-----END KEY-----'\nNEXT=1\n"), env)
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 2 {
		t.Fatalf("read %d variables, want 2", len(vars))
	}
	if want := "-----BEGIN KEY-----\nabc\n-----END KEY-----"; vars[0].value != want {
		t.Errorf("KEY is %q, want %q", vars[0].value, want)
	}
	if vars[1].line != 4 {
		t.Errorf("NEXT is on line %d, want 4", vars[1].line)
	}
}

func TestDotEnvExportIsOnlyAWholeWord(t *testing.T) {
	vars, err := parseDotEnv(".env", []byte("exports=1\nexport=2\nexport A=3\n"), env)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"exports", "export", "A"}
	for i, w := range want {
		if vars[i].name != w {
			t.Errorf("variable %d is named %q, want %q", i, vars[i].name, w)
		}
	}
}

func TestDotEnvEmpty(t *testing.T) {
	for _, src := range []string{"", "\n\n", "# only a comment\n", "   \n\t\n"} {
		vars, err := parseDotEnv(".env", []byte(src), env)
		if err != nil {
			t.Errorf("parsing %q: %v", src, err)
		}
		if len(vars) != 0 {
			t.Errorf("parsing %q read %d variables, want none", src, len(vars))
		}
	}
}

func TestDotEnvErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
		line int
	}{
		{"no equals", "FOO\n", `"FOO" has no value`, 1},
		{"no equals at the end", "A=1\nFOO", `"FOO" has no value`, 2},
		{"no name", "=bar\n", "a line starts with = and so names nothing", 1},
		{"a digit first", "1FOO=bar\n", `"1FOO" is not a variable name`, 1},
		{"a space in the name", "FOO BAR=x\n", `"FOO BAR" is not a variable name`, 1},
		{"a dash in the name", "FOO-BAR=x\n", `"FOO-BAR" is not a variable name`, 1},
		{"no closing double quote", "A=1\nFOO=\"open\n", `the value of FOO has no closing "`, 2},
		{"no closing single quote", "FOO='open\n", "the value of FOO has no closing '", 1},
		{"no closing brace", "FOO=${OPEN\n", "the value of FOO has a ${ with no closing }", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDotEnv("test.env", []byte(tt.src), env)
			if err == nil {
				t.Fatalf("parsing %q gave no error, want %q", tt.src, tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error is %q, want it to mention %q", err, tt.want)
			}
			var cerr *Error
			if !errors.As(err, &cerr) {
				t.Fatalf("error is %T, want a *config.Error", err)
			}
			if cerr.File != "test.env" || cerr.Line != tt.line {
				t.Errorf("error is at %s:%d, want test.env:%d", cerr.File, cerr.Line, tt.line)
			}
		})
	}
}

func TestUnescape(t *testing.T) {
	tests := []struct{ in, want string }{
		{"plain", "plain"},
		{`a\nb`, "a\nb"},
		{`a\rb`, "a\rb"},
		{`a\tb`, "a\tb"},
		{`a\\b`, `a\b`},
		{`a\"b`, `a"b`},
		{`a\'b`, `a'b`},
		{`a\$b`, `a$b`},
		{`C:\Users\me`, `C:\Users\me`},
		{`ends with a backslash \`, `ends with a backslash \`},
	}
	for _, tt := range tests {
		if got := unescape(tt.in); got != tt.want {
			t.Errorf("unescape(%q) is %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestCommentAt(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"value # comment", 6},
		{"value\t# comment", 6},
		{"#comment", 0},
		{"value#part", -1},
		{"value", -1},
		{"", -1},
	}
	for _, tt := range tests {
		if got := commentAt(tt.in); got != tt.want {
			t.Errorf("commentAt(%q) is %d, want %d", tt.in, got, tt.want)
		}
	}
}
