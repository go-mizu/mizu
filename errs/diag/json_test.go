package diag_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs/diag"
	"github.com/go-mizu/mizu/golden"
)

// full is a diagnostic with every field in it, which is the one worth
// comparing against a golden file.
var full = diag.Diagnostic{
	Code:    "MZ1042",
	Message: `unknown config key "database.pool_size"`,
	File:    "config/app.toml",
	Range:   diag.Span(14, 1, 9),
	Detail:  "no such field in Config.Database",
	Suggestions: []diag.Suggestion{{
		Message:    `did you mean "max_open_conns"?`,
		Confidence: diag.High,
		Edits: []diag.Edit{{
			File:    "config/app.toml",
			Range:   diag.Span(14, 1, 9),
			NewText: "max_open_conns",
		}},
	}},
	Fix: "mizu fix config --rule=rename-key --from=database.pool_size --to=database.max_open_conns",
}

// The document, whole. This is the example in doc 37 section 4.2, and the
// golden file is what a change to the wire format has to walk past.
func TestJSONTheWholeDocument(t *testing.T) {
	var b strings.Builder
	err := diag.JSON(&b, diag.List{
		full,
		{Severity: diag.Warning, Message: "no APP_KEY is set", Fix: "mizu key:generate"},
		{Severity: diag.Note, Message: "the toolchain is a version behind"},
	}, diag.WithDuration(43))
	if err != nil {
		t.Fatal(err)
	}
	golden.AssertString(t, b.String())
}

// A run that found nothing is still a document. A reader should not have to
// tell an empty run from a crashed one by whether there was any output.
func TestJSONOfNothing(t *testing.T) {
	for _, l := range []diag.List{nil, {}} {
		var b strings.Builder
		if err := diag.JSON(&b, l); err != nil {
			t.Fatal(err)
		}
		var doc diag.Document
		if err := json.Unmarshal([]byte(b.String()), &doc); err != nil {
			t.Fatal(err)
		}
		if doc.Schema != diag.Schema {
			t.Errorf("schema is %q, want %q", doc.Schema, diag.Schema)
		}
		if doc.Diagnostics == nil {
			t.Error("diagnostics is null, want an empty list")
		}
		if doc.Summary != (diag.Counts{}) {
			t.Errorf("summary is %+v, want zero", doc.Summary)
		}
	}
}

func TestJSONSummaryCounts(t *testing.T) {
	var b strings.Builder
	err := diag.JSON(&b, diag.List{
		{}, {}, {Severity: diag.Warning}, {Severity: diag.Note},
	}, diag.WithDuration(1200))
	if err != nil {
		t.Fatal(err)
	}
	var doc diag.Document
	if err := json.Unmarshal([]byte(b.String()), &doc); err != nil {
		t.Fatal(err)
	}
	want := diag.Counts{Errors: 2, Warnings: 1, DurationMS: 1200}
	if doc.Summary != want {
		t.Errorf("summary is %+v, want %+v", doc.Summary, want)
	}
}

// A document round trips, because mizu fix reads one back and applies the
// edits in it.
func TestJSONRoundTrips(t *testing.T) {
	var b strings.Builder
	if err := diag.JSON(&b, diag.List{full}); err != nil {
		t.Fatal(err)
	}
	var doc diag.Document
	if err := json.Unmarshal([]byte(b.String()), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Diagnostics) != 1 {
		t.Fatalf("read back %d diagnostics, want 1", len(doc.Diagnostics))
	}
	got := doc.Diagnostics[0]
	if got.Code != full.Code || got.Message != full.Message || got.File != full.File {
		t.Errorf("read back %+v", got)
	}
	if got.Range != full.Range {
		t.Errorf("range came back %+v, want %+v", got.Range, full.Range)
	}
	if got.Fix != full.Fix {
		t.Errorf("fix came back %q", got.Fix)
	}
	if len(got.Suggestions) != 1 || got.Suggestions[0].Confidence != diag.High {
		t.Errorf("suggestions came back %+v", got.Suggestions)
	}
	if len(got.Suggestions[0].Edits) != 1 || got.Suggestions[0].Edits[0].NewText != "max_open_conns" {
		t.Errorf("edits came back %+v", got.Suggestions[0].Edits)
	}
}

// A diagnostic with no place leaves the range out rather than writing a range
// of zeroes, which reads as line 0 to a program that does not check.
func TestJSONLeavesOutWhatIsNotThere(t *testing.T) {
	b, err := json.Marshal(diag.Diagnostic{Message: "no APP_KEY is set"})
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, gone := range []string{"range", "code", "file", "detail", "suggestions", "fix_command", "explain", "docs"} {
		if strings.Contains(got, `"`+gone+`"`) {
			t.Errorf("%s is in the document with nothing in it: %s", gone, got)
		}
	}
	for _, want := range []string{`"severity":"error"`, `"message":"no APP_KEY is set"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in %s", want, got)
		}
	}
}

// explain and docs are computed from the code every time. A document whose
// explain line disagrees with its code is one this package will not reproduce.
func TestJSONDoesNotStoreTheExplanation(t *testing.T) {
	var d diag.Diagnostic
	err := json.Unmarshal([]byte(`{
		"code": "MZ1042",
		"severity": "error",
		"message": "unknown setting",
		"explain": "mizu explain MZ9999",
		"docs": "https://example.invalid/somewhere"
	}`), &d)
	if err != nil {
		t.Fatal(err)
	}

	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); !strings.Contains(got, `"explain":"mizu explain MZ1042"`) {
		t.Errorf("kept somebody else's explanation: %s", got)
	}
}

// A severity this version does not know is a document from a version it does
// not know, and reading it as an error beats reading it as the zero value.
func TestJSONRefusesASeverityItDoesNotKnow(t *testing.T) {
	var d diag.Diagnostic
	err := json.Unmarshal([]byte(`{"severity": "catastrophe", "message": "x"}`), &d)
	if err == nil {
		t.Fatal("accepted a severity that does not exist")
	}
}

func TestJSONReturnsAWriteError(t *testing.T) {
	err := diag.JSON(brokenWriter{}, diag.List{{Message: "unknown setting"}})
	if !errors.Is(err, errBrokenWriter) {
		t.Errorf("JSON() returned %v, want the write error", err)
	}
}

// The grouping Text does is for a person reading two hundred lines. A program
// reading them is not the one being spared.
func TestJSONKeepsEveryOne(t *testing.T) {
	var l diag.List
	for range 200 {
		l = append(l, diag.Diagnostic{Code: "MZ1042", Message: "unknown setting"})
	}
	var b strings.Builder
	if err := diag.JSON(&b, l, diag.WithLimit(3)); err != nil {
		t.Fatal(err)
	}
	var doc diag.Document
	if err := json.Unmarshal([]byte(b.String()), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Diagnostics) != 200 {
		t.Errorf("wrote %d of 200", len(doc.Diagnostics))
	}
}
