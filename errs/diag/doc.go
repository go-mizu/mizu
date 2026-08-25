// Package diag is one thing wrong, said once, in two languages.
//
// A [Diagnostic] is a value: what is wrong, where, why it matters and what to
// do about it. [Text] renders it for a person and [JSON] renders it for a
// program, and both read the same value, so the two cannot drift apart. That
// is the whole reason this is a package rather than a fmt.Errorf in each tool.
//
//	d := diag.Diagnostic{
//		Code:    "MZ1042",
//		Message: `unknown config key "database.pool_size"`,
//		File:    "config/app.toml",
//		Range:   diag.Span(14, 1, 10),
//		Detail:  "no such field in Config.Database",
//		Fix:     "mizu fix config --rule=rename-key --from=database.pool_size --to=database.max_open_conns",
//	}
//	diag.Text(os.Stderr, diag.List{d})
//
// prints
//
//	error[MZ1042]: unknown config key "database.pool_size"
//	  --> config/app.toml:14:1
//	   |
//	14 | pool_size = 25
//	   | ^^^^^^^^^ no such field in Config.Database
//	   |
//	   = fix: mizu fix config --rule=rename-key --from=database.pool_size --to=database.max_open_conns
//	   = explain: mizu explain MZ1042
//
// The shape is the one rustc uses, because it is the most read diagnostic
// format there is and inventing a second one buys nothing.
//
// # What a diagnostic owes the reader
//
// Name the thing, say what was expected and what was found, give the fix,
// point at the explanation once, and be quiet about everything else. The type
// is built so that the first and the fourth are hard to leave out: [Diagnostic.File]
// and [Diagnostic.Range] are where the place goes, and [Diagnostic.Code] is
// what the explain and docs lines are computed from, so a diagnostic with a
// code has both without anybody writing them.
//
// The rest is a matter of what the producer puts in the fields, which is why
// the golden corpus exists.
//
// # Machine-readable, always
//
// [JSON] writes the mizu.diag/1 document, which is the format every mizu
// command emits under --json. An [Edit] in it is a byte-accurate replacement a
// program can apply without parsing anything, and [Confidence] says whether it
// should.
//
// # Did you mean
//
// [Suggest] decides what to offer for a name nobody recognises and [Did] writes
// the sentence it goes in. They are here rather than in the packages that
// report unknown names because an unknown setting, an unknown flag and an
// unknown command are the same problem, and three implementations of it drift
// apart in the threshold, in the wording and in whether anything is offered at
// all. Where nothing qualifies, nothing comes back: a wrong suggestion sends
// the reader down a false path with confidence.
//
// # Errors
//
// A [Diagnostic] is an error and a [List] turns into one with [List.Err], so a
// package that finds problems returns them the ordinary way. [Of] goes back the
// other way and pulls the diagnostics out of an error, or makes one out of an
// error that carries none, so a command with an error always has something to
// print under --json.
package diag
