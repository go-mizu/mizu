// Package broken parses and does not type-check. It is here so a test can
// check that a package with type errors comes back with its syntax intact
// instead of being dropped.
package broken

// Count calls a function that does not exist.
func Count() int { return missing() }
