package diagtest

import (
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs/diag"
)

// The rules are what a corpus enforces between reviews, so a rule that never
// fires is worse than no rule at all. Each entry here is a diagnostic and the
// complaint it should draw, or no complaint for the ones that are fine.
func TestRules(t *testing.T) {
	tests := []struct {
		name string
		d    diag.Diagnostic
		want string // a substring of the one complaint, or "" for silence
	}{
		{
			name: "a message that follows the rules",
			d:    diag.Diagnostic{Message: `--days: "soon" is not a number`},
		},
		{
			name: "a message starting with a name in capitals",
			d:    diag.Diagnostic{Message: "APP_KEY is not set, generate one with mizu key:generate"},
		},
		{
			name: "a message starting with a qualified name",
			d:    diag.Diagnostic{Message: "HTTP.Listen: want an address and a port, got a string"},
		},
		{
			name: "a message starting with a hyphenated name",
			d:    diag.Diagnostic{Message: "X-Forwarded-For was set by something that is not a trusted proxy"},
		},
		{
			name: "a message starting with a name holding a digit",
			d:    diag.Diagnostic{Message: "MZ1042 is reported by nothing in this build"},
		},
		{
			name: "no message",
			want: "has no message",
		},
		{
			name: "a message on two lines",
			d:    diag.Diagnostic{Message: "unknown setting \"app.nmae\"\ndid you mean \"app.name\"?"},
			want: "runs to more than one line",
		},
		{
			name: "a message ending in a full stop",
			d:    diag.Diagnostic{Message: "the file is not there."},
			want: "ends in punctuation",
		},
		{
			name: "a message ending in an exclamation mark",
			d:    diag.Diagnostic{Message: "the file is not there!"},
			want: "ends in punctuation",
		},
		{
			name: "a message starting with a sentence",
			d:    diag.Diagnostic{Message: "The file is not there"},
			want: "starts with a capital letter",
		},
		{
			name: "a message that says nothing",
			d:    diag.Diagnostic{Message: "failed to open the file"},
			want: `says "failed to"`,
		},
		{
			name: "a message that says nothing in other words",
			d:    diag.Diagnostic{Message: "the loader hit an Unexpected Error"},
			want: `says "unexpected error"`,
		},
		{
			name: "a message carrying colour",
			d:    diag.Diagnostic{Message: "\x1b[31mno such table\x1b[0m"},
			want: "terminal escapes",
		},
		{
			name: "a detail carrying colour",
			d:    diag.Diagnostic{Message: "no such table", Detail: "\x1b[31mhere\x1b[0m"},
			want: "terminal escapes",
		},
		{
			name: "a detail repeating the message",
			d:    diag.Diagnostic{Message: "no such table", Detail: "no such table"},
			want: "detail repeats the message",
		},
		{
			name: "a detail saying something else",
			d:    diag.Diagnostic{Message: "no such table", Detail: "want one of users, orders"},
		},
		{
			name: "a code that is registered",
			d:    diag.Diagnostic{Message: "no such table", Code: "MZ1042"},
		},
		{
			name: "a code that is not a code",
			d:    diag.Diagnostic{Message: "no such table", Code: "MZ42"},
			want: "is not a code",
		},
		{
			name: "a code nothing registered",
			d:    diag.Diagnostic{Message: "no such table", Code: "MZ9999"},
			want: "is not in the registry",
		},
		{
			name: "a position with a file",
			d:    diag.Diagnostic{Message: "no such table", File: "app.toml", Range: diag.At(2, 8)},
		},
		{
			name: "a position with no file",
			d:    diag.Diagnostic{Message: "no such table", Range: diag.At(2, 8)},
			want: "nowhere to look",
		},
		{
			name: "a suggestion with an edit",
			d: diag.Diagnostic{
				Message: "no such table",
				Suggestions: []diag.Suggestion{{
					Message: `did you mean "users"?`,
					Edits:   []diag.Edit{{File: "app.toml", Range: diag.Span(2, 8, 5), NewText: "users"}},
				}},
			},
		},
		{
			name: "a suggestion with no message",
			d: diag.Diagnostic{
				Message:     "no such table",
				Suggestions: []diag.Suggestion{{}},
			},
			want: "suggestion 0 has no message",
		},
		{
			name: "an edit naming no file",
			d: diag.Diagnostic{
				Message: "no such table",
				Suggestions: []diag.Suggestion{{
					Message: `did you mean "users"?`,
					Edits:   []diag.Edit{{Range: diag.Span(2, 8, 5), NewText: "users"}},
				}},
			},
			want: "edit 0 names no file",
		},
		{
			name: "an edit with no range",
			d: diag.Diagnostic{
				Message: "no such table",
				Suggestions: []diag.Suggestion{{
					Message: `did you mean "users"?`,
					Edits:   []diag.Edit{{File: "app.toml", NewText: "users"}},
				}},
			},
			want: "edit 0 has no range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := watch(t, func(tb testing.TB) { Check(tb, diag.List{tt.d}) })
			if tt.want == "" {
				if len(r.errs) > 0 {
					t.Fatalf("complained about a diagnostic that is fine: %q", r.errs)
				}
				return
			}
			if got := r.only(t); !strings.Contains(got, tt.want) {
				t.Errorf("says %q, want it to mention %q", got, tt.want)
			}
		})
	}
}

// A report holds more than one diagnostic often enough that a complaint about
// the wrong one is a real way to lose an afternoon.
func TestARuleNamesWhichDiagnosticBrokeIt(t *testing.T) {
	l := diag.List{
		{Message: "no such table"},
		{Message: "The file is not there"},
	}

	r := watch(t, func(tb testing.TB) { Check(tb, l) })
	if got := r.only(t); !strings.HasPrefix(got, "diagnostic 1:") {
		t.Errorf("says %q, want it to name diagnostic 1", got)
	}
}
