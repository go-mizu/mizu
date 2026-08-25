package main

import (
	"encoding/json"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/console"
)

// jsonRuns is one way of running each command that makes it print its answer.
//
// The key is the name in the command's spec. It is a table rather than a loop
// over the registry with no arguments because a command needs something to
// report on, and what that is differs: about and doctor need a project, new
// needs somewhere to write one, hash:tune needs a target it can reach before
// the test times out.
//
// The function sets up whatever the command needs, including the working
// directory, and returns the arguments to run it with. --json is added by the
// caller, so no entry here can pass it and no entry here can leave it out.
var jsonRuns = map[string]func(t *testing.T) []string{
	"about": func(t *testing.T) []string {
		scratch(t, commands)
		return nil
	},
	"check": func(t *testing.T) []string {
		pinEndings(t, scratch(t, commands))
		return nil
	},
	"doctor": func(t *testing.T) []string {
		pinEndings(t, scratch(t, commands))
		return nil
	},
	"gen": func(t *testing.T) []string {
		scratch(t, commands)
		return []string{"--check", "./..."}
	},
	"gen:agents": func(t *testing.T) []string {
		scratch(t, commands)
		return []string{"--check", "./..."}
	},
	"gen:command": func(t *testing.T) []string {
		scratch(t, commands)
		return []string{"--check", "./..."}
	},
	"gen:config": func(t *testing.T) []string {
		scratch(t, configs)
		return []string{"--check", "./..."}
	},
	"hash:tune": func(t *testing.T) []string {
		return []string{quick}
	},
	"lint": func(t *testing.T) []string {
		pinEndings(t, scratch(t, commands))
		return nil
	},
	"new": func(t *testing.T) []string {
		return []string{filepath.Join(t.TempDir(), "blog"), "--preset=api"}
	},
	"verify": func(t *testing.T) []string {
		pinEndings(t, scratch(t, commands))
		return nil
	},
	"version": func(t *testing.T) []string {
		return nil
	},
}

// TestEveryCommandAnswersInJSON is the promise that --json is worth writing a
// script against.
//
// The flag is global, so every command takes it whether or not it does anything
// with it, and a command that takes it and then prints a table for a person is
// worse than one that rejects it. What is checked here is that the output
// stream holds one JSON document and nothing else.
//
// The walk is over the registry rather than over the table, so a command that
// is added without deciding what document it writes fails this rather than
// being found by whoever pipes it into jq first.
func TestEveryCommandAnswersInJSON(t *testing.T) {
	for _, cmd := range registry() {
		name := cmd.Spec().Name

		setUp, ok := jsonRuns[name]
		if !ok {
			t.Errorf("mizu %s is registered and jsonRuns has no entry for it, so nothing checks that --json answers", name)
			continue
		}

		t.Run(name, func(t *testing.T) {
			argv := append([]string{name, "--json"}, setUp(t)...)
			out, errOut, code := start(t, argv...)

			// An unrecognised flag, a command that could not run, and a
			// command that answered in prose all land here the same way:
			// nothing a program can read came out.
			got := strings.TrimSpace(out.String())
			if got == "" {
				t.Fatalf("mizu %s wrote nothing to the output, and exited %d\n%s", strings.Join(argv, " "), code, errOut)
			}
			if !json.Valid([]byte(got)) {
				t.Errorf("mizu %s did not answer with JSON:\n%s", strings.Join(argv, " "), got)
			}
		})
	}
}

// --json is a global, so a command takes it by being registered rather than by
// declaring it, and the way to lose it is to declare a flag of the same name.
// The global parser runs first and takes the argument, so the command's own
// flag is never set and nothing says so.
//
// Every global is checked and not only --json, because the same mistake is the
// same mistake for --quiet or --timeout.
func TestNoCommandShadowsAGlobalFlag(t *testing.T) {
	var mine globals
	var shared console.Globals

	reserved := make(map[string]bool)
	for _, f := range append(shared.Flags(), mine.flags()...) {
		reserved[f.Name] = true
	}

	for _, cmd := range registry() {
		spec := cmd.Spec()
		for _, f := range spec.Flags {
			if reserved[f.Name] {
				t.Errorf("mizu %s declares --%s, which is a global flag, so the command never sees what was passed", spec.Name, f.Name)
			}
		}
	}
}

// A name in the table that no command answers to is a check that stopped
// checking anything when the command was renamed.
func TestTheJSONTableHasNothingSpare(t *testing.T) {
	registered := make(map[string]bool, len(registry()))
	for _, cmd := range registry() {
		registered[cmd.Spec().Name] = true
	}

	for _, name := range slices.Sorted(maps.Keys(jsonRuns)) {
		if !registered[name] {
			t.Errorf("jsonRuns has an entry for %q and no command is called that", name)
		}
	}
}
