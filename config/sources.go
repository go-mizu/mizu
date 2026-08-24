package config

import (
	"path/filepath"
	"strings"
)

const (
	// EnvVar is the environment variable that names the environment.
	EnvVar = "MIZU_ENV"

	// EnvDefault is the environment when nothing names one. Nothing names one
	// on a laptop with a fresh checkout, which is the case worth being kind to.
	EnvDefault = "local"

	// FlagPrefix starts a command line argument that sets a setting, as in
	// --config.database.max-open-conns=25. A single dash works too.
	FlagPrefix = "config."
)

// Sources are the places configuration is read from.
//
// Everything is named outright and nothing is implied: [Open] reads what this
// struct says and looks for nothing else. [Discover] fills one in the way an
// application would, and printing the result shows exactly what will be read,
// which is what config:show puts at the top of its output.
type Sources struct {
	// Env is the environment name, such as local or production. The zero
	// value means EnvDefault.
	Env string

	// Files are TOML files, read in order, with a later one winning over an
	// earlier one. A file that is not there is skipped, so a project with no
	// config directory still starts.
	Files []string

	// DotEnv are .env files, read in order, with a later one winning over an
	// earlier one. They rank above Files and below Environ. A file that is not
	// there is skipped.
	DotEnv []string

	// Environ is the process environment, in the NAME=value form os.Environ
	// returns. A test that wants a clean environment leaves it nil.
	Environ []string

	// Args are command line arguments. Only --config.<path>=<value> is read
	// and everything else is ignored, because the flag package owns the rest
	// of the command line.
	Args []string

	// Override wins over every other layer, keyed by the dotted path of the
	// setting. It is for values the program works out for itself, and for
	// tests that want one setting to be a particular thing.
	Override map[string]string

	// Command runs a cmd: indirection on a secret field and returns what the
	// command printed. The zero value refuses them, because reading a file is
	// one thing and starting a process is another, and a caller has to ask for
	// the second one on purpose.
	//
	// It is a function rather than a flag so that the core never has to know
	// how to run a program. A caller that wants cmd:op read op://vault/db to
	// work supplies the part that runs it, and gets to decide what a command is
	// allowed to be.
	Command func(name string) (string, error)
}

// Discover returns the sources for an application rooted at dir.
//
// The environment name comes from the command line, then the process
// environment, then EnvDefault, and it decides the rest: config/<env>.toml
// and config/local.toml are the files, and the .env files are read only in
// local and testing. A .env file holds the credentials a developer uses on
// their own machine, and reading one on a production server would mean a file
// that got deployed by accident could quietly take over.
//
// Nothing here touches the disk. Everything it names is optional, and [Open]
// is what finds out which of them exist.
func Discover(dir string, environ, args []string) Sources {
	env := EnvName(environ, args)
	s := Sources{Env: env, Environ: environ, Args: args}

	s.Files = append(s.Files, filepath.Join(dir, "config", env+".toml"))
	if env != "local" {
		s.Files = append(s.Files, filepath.Join(dir, "config", "local.toml"))
	}
	if env == "local" || env == "testing" {
		// .env is shared and committed, .env.<env> is per environment and
		// committed, and .env.<env>.local is one developer's own and is not.
		// In local the middle one is already the private one, which is what
		// .env.local means everywhere else too, so there is no third file.
		s.DotEnv = append(s.DotEnv, filepath.Join(dir, ".env"), filepath.Join(dir, ".env."+env))
		if env != "local" {
			s.DotEnv = append(s.DotEnv, filepath.Join(dir, ".env."+env+".local"))
		}
	}
	return s
}

// EnvName is the environment named by the command line or the process
// environment, or EnvDefault when neither names one.
//
// This one setting has to be resolved before any file is read, because it is
// what says which files to read.
func EnvName(environ, args []string) string {
	if v, ok := flagValue(args, "env"); ok && v != "" {
		return v
	}
	if v, ok := environValue(environ, EnvVar); ok && v != "" {
		return v
	}
	return EnvDefault
}

// environValue finds a variable in an os.Environ slice. Later entries win,
// which is what the C library does when a name appears twice.
func environValue(environ []string, name string) (string, bool) {
	for i := len(environ) - 1; i >= 0; i-- {
		if k, v, ok := strings.Cut(environ[i], "="); ok && k == name {
			return v, true
		}
	}
	return "", false
}

// flagValue finds --config.<path>=<value> on a command line, without parsing
// the rest of it. Only EnvName uses this; everything else goes through the
// flags a Loader has already collected, which is where a flag with no value
// is reported.
func flagValue(args []string, path string) (string, bool) {
	for i := len(args) - 1; i >= 0; i-- {
		rest, ok := configFlag(args[i])
		if !ok {
			continue
		}
		if name, value, ok := strings.Cut(rest, "="); ok && flagKey(name) == flagKey(path) {
			return value, true
		}
	}
	return "", false
}

// configFlag strips the dashes and the config. prefix off an argument and
// reports whether it was one of ours. What is left is name=value, and both
// callers cut that themselves, because one of them has something to say about
// an argument with no value in it.
func configFlag(arg string) (string, bool) {
	rest, ok := strings.CutPrefix(arg, "--")
	if !ok {
		if rest, ok = strings.CutPrefix(arg, "-"); !ok {
			return "", false
		}
	}
	rest, ok = strings.CutPrefix(rest, FlagPrefix)
	if !ok || rest == "" {
		return "", false
	}
	return rest, true
}

// flagKey folds a path the way the command line writes it. A setting named
// max_open_conns is more naturally typed as --config.database.max-open-conns,
// so on the command line a dash and an underscore are the same character, and
// case does not count.
func flagKey(path string) string {
	return strings.ReplaceAll(strings.ToLower(path), "-", "_")
}
