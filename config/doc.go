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
// # Reading a value into a field
//
// [Get] is [Loader.Lookup] with a type on the end of it. It takes a [Parse],
// which is any function that reads a value into a destination, and this package
// has one for each of the types a setting is usually written as.
//
//	config.Get(l, &c.DB.DSN, dsn, config.String)
//	config.Get(l, &c.HTTP.ReadTimeout, timeout, config.Duration)
//	config.Get(l, &c.HTTP.TrustedProxies, proxies, config.Slice(config.Prefix))
//
// A field of a type this package has never heard of is one function away, and
// a type that reads itself can be a [Parser] or an [encoding.TextUnmarshaler]
// and go through [Config] or [Text]. There is no reflection anywhere in it.
//
// A parser written by hand reads its text with [Value.Str], which is the value
// from whichever layer had it, and an error when a file wrote it as a number or
// a boolean instead. It returns an error that says what it wanted and nothing
// about where, since [Get] puts the field and the line in front of it.
//
// Nothing returns an error as it goes. A field that will not read is recorded
// and the next one is read anyway, so an application with three settings wrong
// hears about all three at once, from [Loader.Err] at the end.
//
// # Secrets
//
// A field marked [Field.Secret] never prints, and its value may point somewhere
// else instead of being the secret: file:/run/secrets/db reads a file, and
// env:OTHER_NAME reads another variable. Both are for a container that mounts
// its secrets rather than passing them in, and neither applies to a field that
// is not a secret, since file:/tmp/app.sqlite is a real database DSN.
//
// cmd:... runs a command and takes what it printed, and works only when the
// caller supplies [Sources.Command]. This package starts no processes of its
// own, because a configuration file that can run a program is something a
// caller has to ask for on purpose.
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
