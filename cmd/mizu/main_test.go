package main

import (
	"flag"
	"strings"
	"testing"
)

func TestRunWithNoArgumentsPrintsUsage(t *testing.T) {
	var out, errOut strings.Builder
	if err := run(nil, &out, &errOut); err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, want := range []string{"Usage:", "version"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("usage does not mention %q:\n%s", want, out.String())
		}
	}
	if errOut.Len() != 0 {
		t.Errorf("usage went to stderr: %q", errOut.String())
	}
}

func TestRunHelpForms(t *testing.T) {
	for _, arg := range []string{"help", "-h", "--help"} {
		t.Run(arg, func(t *testing.T) {
			var out, errOut strings.Builder
			if err := run([]string{arg}, &out, &errOut); err != nil {
				t.Fatalf("run: %v", err)
			}
			if !strings.Contains(out.String(), "Usage:") {
				t.Errorf("%s printed no usage:\n%s", arg, out.String())
			}
		})
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var out, errOut strings.Builder
	err := run([]string{"serve"}, &out, &errOut)
	if err == nil {
		t.Fatal("unknown command was accepted")
	}
	if !strings.Contains(err.Error(), "serve") {
		t.Errorf("error = %q, want it to name the command", err)
	}
	if !strings.Contains(err.Error(), "mizu help") {
		t.Errorf("error = %q, want it to point at mizu help", err)
	}
}

// A -h on a subcommand comes back as flag.ErrHelp, which main turns into exit
// status 2 rather than an error message. The flag package has already printed
// the usage by then, so printing anything else would say it twice.
func TestSubcommandHelpReturnsErrHelp(t *testing.T) {
	var out, errOut strings.Builder
	err := run([]string{"version", "-h"}, &out, &errOut)
	if err != flag.ErrHelp {
		t.Fatalf("run returned %v, want flag.ErrHelp", err)
	}
	if !strings.Contains(errOut.String(), "-json") {
		t.Errorf("usage does not mention the flag:\n%s", errOut.String())
	}
}

func TestEveryCommandHasASummary(t *testing.T) {
	for _, c := range commands {
		if c.name == "" || c.summary == "" || c.run == nil {
			t.Errorf("command %+v is missing a field", c)
		}
		if strings.HasSuffix(c.summary, ".") {
			t.Errorf("summary %q ends in a full stop, and the usage list reads better without", c.summary)
		}
	}
}
