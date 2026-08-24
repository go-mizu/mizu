// Package bootstrap is the shape of the problem the overlay exists for. The
// hand-written file below is fine. The generated file next to it was written
// against an older version of Config that had a Port field, so the package
// does not type-check, so the generator that would rewrite it cannot run.
package bootstrap

// A Config is a server address.
type Config struct {
	Addr string
}
