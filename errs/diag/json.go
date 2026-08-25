package diag

import (
	"encoding/json"
	"io"
)

// Schema is the name in every document this package writes.
//
// The number goes up when a reader that understands this one would get a
// document wrong. Adding a field does not do that, so it does not move the
// number; changing what a field means does.
const Schema = "mizu.diag/1"

// A Document is what [JSON] writes and what a program reads.
//
// It is exported so that a command building one out of several runs, or a test
// reading one back, has the type rather than a map.
type Document struct {
	Schema      string `json:"schema"`
	Diagnostics List   `json:"diagnostics"`
	Summary     Counts `json:"summary"`
}

// Counts is the tally at the end of a document.
type Counts struct {
	Errors     int   `json:"errors"`
	Warnings   int   `json:"warnings"`
	DurationMS int64 `json:"duration_ms"`
}

// JSON writes l as a mizu.diag/1 document.
//
// Every mizu command that can print a diagnostic prints this under --json,
// including the ones invoked through go generate, which have no flag to pass
// and write it to the path in MIZU_DIAG_FILE instead.
//
// The document is a document even when nothing went wrong: an empty
// diagnostics list and a summary of zero. A reader should not have to tell an
// empty run from a crashed one by whether there was any output.
//
// [WithLimit] does not apply here. The grouping [Text] does is for a person
// reading two hundred lines, and a program reading them is not the one being
// spared.
func JSON(w io.Writer, l List, opts ...Option) error {
	o := newOptions(opts)
	if l == nil {
		l = List{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(Document{
		Schema:      Schema,
		Diagnostics: l,
		Summary: Counts{
			Errors:     l.Count(Error),
			Warnings:   l.Count(Warning),
			DurationMS: o.dur,
		},
	})
}

// wire is the JSON shape of a diagnostic.
//
// It is a type of its own rather than tags on [Diagnostic] because two of the
// fields are not stored. explain and docs are computed from the code, so a
// document cannot carry an explanation for a different diagnostic than the one
// it is on, and the Go value has one field where the wire has three.
type wire struct {
	Code        Code         `json:"code,omitempty"`
	Severity    Severity     `json:"severity"`
	Message     string       `json:"message"`
	File        string       `json:"file,omitempty"`
	Range       *Range       `json:"range,omitempty"`
	Detail      string       `json:"detail,omitempty"`
	Suggestions []Suggestion `json:"suggestions,omitempty"`
	FixCommand  string       `json:"fix_command,omitempty"`
	Explain     string       `json:"explain,omitempty"`
	Docs        string       `json:"docs,omitempty"`
}

// MarshalJSON writes the mizu.diag/1 shape of one diagnostic.
func (d Diagnostic) MarshalJSON() ([]byte, error) {
	w := wire{
		Code:        d.Code,
		Severity:    d.Severity,
		Message:     d.Message,
		File:        d.File,
		Detail:      d.Detail,
		Suggestions: d.Suggestions,
		FixCommand:  d.Fix,
		Explain:     d.Code.Explain(),
		Docs:        d.Code.Docs(),
	}
	if d.Range.IsValid() {
		w.Range = &d.Range
	}
	return json.Marshal(w)
}

// UnmarshalJSON reads back what [Diagnostic.MarshalJSON] wrote.
//
// explain and docs are dropped rather than stored, since they are what the
// code says they are. A document whose explain line disagrees with its code is
// one this package will not reproduce, which is the right way round.
func (d *Diagnostic) UnmarshalJSON(b []byte) error {
	var w wire
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	*d = Diagnostic{
		Code:        w.Code,
		Severity:    w.Severity,
		Message:     w.Message,
		File:        w.File,
		Detail:      w.Detail,
		Suggestions: w.Suggestions,
		Fix:         w.FixCommand,
	}
	if w.Range != nil {
		d.Range = *w.Range
	}
	return nil
}
