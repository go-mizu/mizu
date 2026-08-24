// Package toml reads TOML 1.0.0 documents.
//
// [Parse] turns bytes into a [Table], which is a tree of [Value] with the line
// and column each one was written at.
//
//	doc, err := toml.ParseFile("config/production.toml")
//	if err != nil {
//		return err
//	}
//	if v := doc.Lookup("database", "dsn"); v != nil && v.Kind == toml.KindString {
//		dsn = v.Str
//	}
//
// # Positions
//
// Every value carries the place it was written. A configuration error is
// almost always about one line of one file, and an error that names it is the
// difference between fixing it and searching for it.
//
//	config/production.toml:14:8: database.pool: want an integer, got a string
//
// [Value.Errorf] writes that first part for you, so a caller that found the
// wrong kind only has to say what it wanted.
//
// # What is not here
//
// There is no encoder. Nothing in mizu writes TOML, and a marshaller that
// keeps comments and formatting is a much larger thing than a parser. If one
// is needed later it goes in its own file and does not change what is here.
//
// There is no struct decoding either, and no reflection. Configuration structs
// are filled by generated code, which reads a [Table] directly: check the
// kind, take the field. That keeps the reflect package out of the startup path
// and makes the mapping from a file to a struct something you can read.
//
// # Why the core carries a parser
//
// The core module of mizu requires nothing outside the standard library, and
// the standard library has no TOML. Configuration is TOML, so the choice was
// to write this or to break the promise on the first package that tested it.
// A parser for a format with a fixed specification and no extensions is a cost
// that gets paid once. See D-073 and D-074 in the decision register.
//
// # Conformance
//
// The whole of TOML 1.0.0, and nothing beyond it. Dates and times are the
// four kinds the specification names, and the ones without an offset are kept
// in UTC with the kind saying that the offset was never written. Seconds are
// required in a time, as 1.0.0 requires them.
//
// Integers are 64 bit and a number too large to fit is an error rather than a
// float. A bare \r that is not part of \r\n is an error, which is worth saying
// because a file edited on Windows and saved by a tool that mixes line endings
// is a real thing that happens.
package toml
