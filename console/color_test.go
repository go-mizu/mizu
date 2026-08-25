package console

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// env returns a getenv for a fixed set of variables, so the rules can be
// checked without changing the environment the test runs in.
func env(pairs ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return func(k string) string { return m[k] }
}

func TestColorEnabled(t *testing.T) {
	tests := map[string]struct {
		mode   Color
		getenv func(string) string
		want   bool
	}{
		"a pipe with nothing set":  {ColorAuto, env(), false},
		"NO_COLOR":                 {ColorAuto, env("NO_COLOR", "1"), false},
		"NO_COLOR set to nothing":  {ColorAuto, env("NO_COLOR", ""), false},
		"NO_COLOR set to false":    {ColorAuto, env("NO_COLOR", "false"), false},
		"a dumb terminal":          {ColorAuto, env("TERM", "dumb"), false},
		"a real terminal name":     {ColorAuto, env("TERM", "xterm-256color"), false},
		"forced":                   {ColorAlways, env(), true},
		"forced over NO_COLOR":     {ColorAlways, env("NO_COLOR", "1"), true},
		"forced over a dumb term":  {ColorAlways, env("TERM", "dumb"), true},
		"refused on a terminal":    {ColorNever, env(), false},
		"refused with TERM normal": {ColorNever, env("TERM", "xterm"), false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// A buffer is not a terminal, which is why every auto case here
			// is false. The terminal side is covered by TestIsTerminal.
			if got := colorEnabled(&bytes.Buffer{}, tt.mode, tt.getenv); got != tt.want {
				t.Errorf("colorEnabled = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNoColorWinsOverATerminal is the case a buffer cannot cover, so it uses
// the process's own stdout and only asserts something when that is a terminal.
func TestNoColorWinsOverATerminal(t *testing.T) {
	if !isTerminal(os.Stdout) {
		t.Skip("stdout is not a terminal here")
	}

	if colorEnabled(os.Stdout, ColorAuto, env("NO_COLOR", "1")) {
		t.Error("NO_COLOR did not turn colour off on a terminal")
	}
	if !colorEnabled(os.Stdout, ColorAuto, env()) {
		t.Error("colour is off on a terminal with nothing set")
	}
}

func TestIsTerminal(t *testing.T) {
	if isTerminal(&bytes.Buffer{}) {
		t.Error("a buffer says it is a terminal")
	}

	// A file has an Fd and is not a terminal, which is the case that catches
	// an implementation that only checks for the method.
	f, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if isTerminal(f) {
		t.Error("a regular file says it is a terminal")
	}
}

func TestTerminalWidth(t *testing.T) {
	if got := terminalWidth(&bytes.Buffer{}, 0); got != 0 {
		t.Errorf("the width of a buffer is %d, want 0", got)
	}
	if got := terminalWidth(&bytes.Buffer{}, 120); got != 120 {
		t.Errorf("the override gave %d, want 120", got)
	}
	if got := terminalWidth(os.Stdout, -1); got < 0 {
		t.Errorf("a negative override gave %d, want the terminal or 0", got)
	}
}

// TestColorsSeparately is why the decision is made per stream. Sending the
// data to a file and watching the messages in the terminal is an ordinary
// thing to do, and the messages should still be readable.
func TestColorsSeparately(t *testing.T) {
	if !isTerminal(os.Stderr) {
		t.Skip("stderr is not a terminal here")
	}

	io := New(strings.NewReader(""), &bytes.Buffer{}, os.Stderr, Options{})

	if io.colorOut {
		t.Error("colour is on for a buffer")
	}
	if !io.colorErr {
		t.Error("colour is off for the terminal")
	}
}

func TestParseColor(t *testing.T) {
	tests := map[string]struct {
		want Color
		bad  bool
	}{
		"auto":   {ColorAuto, false},
		"":       {ColorAuto, false},
		"always": {ColorAlways, false},
		"never":  {ColorNever, false},
		"yes":    {ColorAuto, true},
		"Always": {ColorAuto, true},
	}
	for in, tt := range tests {
		got, err := ParseColor(in)
		if (err != nil) != tt.bad {
			t.Errorf("ParseColor(%q) gave error %v", in, err)
		}
		if got != tt.want {
			t.Errorf("ParseColor(%q) = %v, want %v", in, got, tt.want)
		}
	}
}

// TestParseColorSaysWhatItWanted is the error message being the documentation
// for the flag, which is where somebody who typed it wrong is looking.
func TestParseColorSaysWhatItWanted(t *testing.T) {
	_, err := ParseColor("yes")
	if err == nil {
		t.Fatal("a bad colour was accepted")
	}
	for _, want := range []string{`"yes"`, "auto", "always", "never"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error is %q, want it to mention %s", err, want)
		}
	}
}

func TestColorString(t *testing.T) {
	tests := map[Color]string{ColorAuto: "auto", ColorAlways: "always", ColorNever: "never", Color(9): "auto"}
	for c, want := range tests {
		if got := c.String(); got != want {
			t.Errorf("Color(%d).String() = %q, want %q", c, got, want)
		}
	}
}

func TestStyleWrap(t *testing.T) {
	if got := styleRed.wrap("no", true); got != "\x1b[31mno\x1b[0m" {
		t.Errorf("wrap gave %q", got)
	}
	if got := styleRed.wrap("no", false); got != "no" {
		t.Errorf("with colour off wrap gave %q, want the text alone", got)
	}
	if got := styleNone.wrap("plain", true); got != "plain" {
		t.Errorf("styleNone gave %q, want no escapes", got)
	}
}

// TestMessagesCarryColor checks the styles reach the output, using a forced
// colour so the test does not need a terminal.
func TestMessagesCarryColor(t *testing.T) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	io := New(strings.NewReader(""), out, errOut, Options{Color: ColorAlways, Verbosity: Verbose})

	io.Success("done")
	io.Warn("careful")
	io.Error("no")
	io.Debug("4ms")

	for _, want := range []string{"\x1b[32mdone", "\x1b[33mwarning: careful", "\x1b[31merror: no", "\x1b[2mdebug: 4ms"} {
		if !strings.Contains(errOut.String(), want) {
			t.Errorf("stderr is %q, want it to contain %q", errOut.String(), want)
		}
	}
}

// TestInfoIsNotColoured keeps the palette meaning something. If every line is
// coloured then no line stands out, which is the state most tools end up in.
func TestInfoIsNotColoured(t *testing.T) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	io := New(strings.NewReader(""), out, errOut, Options{Color: ColorAlways})

	io.Info("loading the config")

	if got := errOut.String(); got != "loading the config\n" {
		t.Errorf("Info wrote %q, want no escapes", got)
	}
}
