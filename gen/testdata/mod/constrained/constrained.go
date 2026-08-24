//go:build mizu.test.never

// Package constrained has no files once build constraints are applied, so the
// go command has something to say about it and the loader has nothing to
// parse. It is here for the one case where the go command's error is the only
// error there is.
package constrained

func Nothing() {}
