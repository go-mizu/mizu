// Package badmarkers holds one of every mistake the scanner has to catch. It
// compiles, because none of these are Go errors, which is the whole problem
// with them.
package badmarkers

// Spaced has the space that turns a directive into a sentence.
//
// mizu:model table=things
type Spaced struct{}

// There is no fixture here for a marker with nothing after the colon. gofmt
// puts a space after the slashes when it sees one, because Go's rule for a
// directive is a non-space character right after the colon, so the mistake
// cannot survive a formatted file. parseMarker still rejects it, and there is
// a unit test for that.

// Twice gives one key two values.
//
//mizu:model table=a table=b
type Twice struct{}

// Unclosed opens a quote and never closes it.
//
//mizu:command name="posts:prune
type Unclosed struct{}

// Keyless starts an argument with an equals sign.
//
//mizu:model =posts
type Keyless struct{}

// Punctuated puts punctuation where the name ends.
//
//mizu:model! table=posts
type Punctuated struct{}

// Prose mentions a marker in a sentence and is not one, so the scanner should
// let it alone. The words after the colon read like English rather than like
// arguments, which is the difference.
//
// mizu:model is what marks a type as a model.
type Prose struct{}
