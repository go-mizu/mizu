package golden

import "regexp"

// scrubber replaces something that changes between runs with something that
// does not.
type scrubber func([]byte) []byte

// Scrub replaces everything pattern matches with replacement, in the output and
// in the golden file, before the two are compared.
//
//	golden.Assert(t, out, golden.Scrub(tempDir, "<tmp>"))
//
// replacement goes through [regexp.Regexp.ReplaceAll], so $1 is the first
// group. A literal dollar sign has to be written $$.
//
// Scrubbing costs something worth knowing about: whatever is scrubbed is no
// longer being tested. A timestamp replaced by <time> asserts that there was a
// timestamp there, not that it was the right one. Where the value comes from
// code you control, injecting a clock or a fixed ID source asserts more and
// scrubs nothing.
func Scrub(pattern *regexp.Regexp, replacement string) Option {
	repl := []byte(replacement)
	return func(o *options) {
		o.scrubs = append(o.scrubs, func(b []byte) []byte {
			return pattern.ReplaceAll(b, repl)
		})
	}
}

// The patterns are deliberately narrow. A loose one eats something it was not
// meant to and turns a real difference into a match, which is the one failure a
// golden file must not have: a test that passes because the assertion stopped
// looking.
var (
	uuidRE = regexp.MustCompile(`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}\b`)

	ulidRE = regexp.MustCompile(`\b[0-7][0-9ABCDEFGHJKMNPQRSTVWXYZ]{25}\b`)

	// RFC 3339, which is what time.Time marshals to and what an API returns.
	// The offset is required, since a bare date is usually a birthday rather
	// than a timestamp and scrubbing it would hide a real change.
	timeRE = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}[Tt ]\d{2}:\d{2}:\d{2}(\.\d+)?([Zz]|[+-]\d{2}:\d{2})`)

	// What time.Duration prints, which is what shows up in a log line or a
	// timing field.
	durationRE = regexp.MustCompile(`\b\d+(\.\d+)?(ns|µs|us|ms|s|m|h)\b`)
)

// ScrubUUIDs replaces every version 1 to 8 UUID with <uuid>.
//
// The version and variant digits are part of the pattern, so a plain hex string
// of the same shape is left alone.
func ScrubUUIDs() Option { return Scrub(uuidRE, "<uuid>") }

// ScrubULIDs replaces every ULID with <ulid>.
func ScrubULIDs() Option { return Scrub(ulidRE, "<ulid>") }

// ScrubTimes replaces every RFC 3339 timestamp with <time>.
//
// A timestamp without an offset is left alone, since a bare date is more often
// a value the test is about than a value the clock produced.
func ScrubTimes() Option { return Scrub(timeRE, "<time>") }

// ScrubDurations replaces every duration in the form [time.Duration.String]
// prints with <duration>.
//
// This one is the loosest of the four, because 5m in a body of prose is a
// duration as far as the pattern is concerned. Prefer [Scrub] with a pattern
// that includes the field name when the output has prose in it.
func ScrubDurations() Option { return Scrub(durationRE, "<duration>") }
