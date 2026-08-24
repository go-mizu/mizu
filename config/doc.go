// Package config finds the value of a setting and says where it came from.
//
// An application has one configuration struct, and generated code fills it in
// by asking a [Loader] for one field at a time. This package is the half that
// knows about places: files, .env files, the process environment, the command
// line. It does not know what any setting means, and it holds no struct.
//
//	src := config.Discover(".", os.Environ(), os.Args[1:])
//	l, err := config.Open(src)
//	if err != nil {
//		return err
//	}
//	v, ok := l.Lookup(config.Field{Path: "database.dsn", Env: "DATABASE_URL"})
//
// # Order
//
// Six layers, and a later one wins over an earlier one.
//
//  1. the default, which is [Field.Default]
//  2. the files, config/<env>.toml then config/local.toml
//  3. the .env files, in local and testing only
//  4. the process environment
//  5. the command line, as --config.database.dsn=...
//  6. whatever the program set itself, which is [Sources.Override]
//
// The application's defaults and the framework's are one layer here, because
// by the time a field reaches [Loader.Lookup] the generator has already worked
// out which of the two applies.
//
// # Where a value came from
//
// Every answer carries a [Source], so config:show can print the value next to
// the file and line it was written on, and an error about a value can name the
// place to go and fix it. Keeping that is most of the reason this package
// exists rather than a map lookup.
//
// # Settings nobody asked for
//
// [Loader.Check] reports keys in the files that no field claimed. A typo in a
// configuration file is otherwise silent, and silent is the worst thing a
// configuration file can be. The report names the file, the line, and the
// closest key that does exist.
//
//	config/production.toml:14:1: unknown setting "database.max_conns", did you mean "database.max_open_conns"?
package config
