// Package configgen writes the decoder for an application's configuration
// struct. It is an implementation detail of the mizu command and is exempt from
// the compatibility promise in doc 31. Import it only if you are extending mizu
// itself.
//
// An application declares one struct and marks it:
//
//	//mizu:config
//	type Config struct {
//		App struct {
//			Name string `env:"APP_NAME" default:"blog"`
//			Key  []byte `env:"APP_KEY" secret:"true"`
//		}
//		Database struct {
//			DSN          string `env:"DATABASE_URL" secret:"true"`
//			MaxOpenConns int    `default:"25"`
//		}
//	}
//
// [Generate] walks it and writes config_gen.go next to it. The output calls the
// config package one field at a time, with the parser for the field's type
// picked out at generation time, so there is no reflection at startup and a
// field nothing can read is a build failure rather than a surprise on a
// Tuesday.
//
// # What it writes
//
// Four methods and a table, for a struct called Config:
//
//	LoadConfig(config.Sources) (*Config, error)   reads every setting
//	(*Config).Redact() *Config                    a copy with the secrets out
//	(*Config).Describe() []config.FieldDoc        every setting and its value
//	(*Config).Diff(*Config) []config.Change       what two of them disagree about
//
// LoadConfig reports every setting that will not read rather than the first,
// and then every setting written in a file that the struct has no field for.
// Describe and Diff are what config:show, config:doc and config:diff print.
// Redact writes out one line per secret rather than walking the struct, so
// adding a secret shows up as a line in the diff of the generated file.
//
// # Types
//
// A field is read by the parser its type matches, and a type nothing matches
// is an error naming the field:
//
//	string, bool, int, uint and float of every width, and any defined type over one of them
//	[]byte, written as base64
//	time.Duration, time.Time, log/slog.Level, net/netip.Addr, Prefix and AddrPort
//	a slice or a map keyed by a string, of any of the above
//	anything with a ParseConfig method, which is a config.Parser
//	anything with an UnmarshalText method, which is an encoding.TextUnmarshaler
//
// The last two are the way in for a type this package has never heard of. A
// ParseConfig gets the whole config.Value, and calling its Str method is how it
// reads the text, since a value from a file keeps the type it was written with.
// Anything else is a struct that holds more fields, and the walk goes into it.
//
// # Names
//
// A field's place in the struct is its path, in the lower case with underscores
// that a TOML file is written in, so App.Name is app.name and
// Database.MaxOpenConns is database.max_open_conns. Its environment variable
// is the same path in upper case with underscores, so DATABASE_MAX_OPEN_CONNS.
//
// Both are worth overriding often enough to be worth tags. An env tag names the
// variable outright, a toml tag renames one segment of the path, and env:"-"
// says the field has no variable at all.
//
//	DSN string `env:"DATABASE_URL"`
//	MaxOpenConns int `toml:"pool_size"`
//	Internal string `env:"-"`
//
// An embedded struct adds no segment, so the fields of an embedded mizu.Base
// sit at the top level where an application expects to find them.
//
// # Tags
//
//	env       the environment variable, or - for none
//	toml      the name of this segment of the path
//	default   the value when no layer has one, as it would be written in a file
//	secret    true means never print it, and allow file: and env: indirection
//
// A validate tag is read by the validate package rather than this one, and is
// left alone here.
//
// A field with no tags at all still gets a path and a variable, because both
// come from where the field is. Tags are for the times that is not what you
// want.
//
// # Limits
//
// One struct per package carries the marker. An application has one
// configuration, and a second one is a mistake rather than a second
// application.
//
// A field that is not exported is skipped rather than reported, since a
// private field in a configuration struct is nearly always something other
// than configuration. A field nested more than twelve deep is reported, which
// is further down than configuration has any reason to go.
//
// # What is not here yet
//
// Two forms from doc 05 are reported as errors rather than half implemented.
// A default with a bar in it, as in "console|json", means one value in local
// and another everywhere else, and needs the environment to have been decoded
// first. A default with braces in it, as in "{App.Name}:", refers to another
// field, and needs an order worked out across the struct. Both arrive with
// mizu.Base, which is what makes them meaningful.
//
// Validation is not here either. A validate tag says what a value has to be,
// and the code that checks it belongs with the validate package rather than
// with the code that reads the value.
package configgen
