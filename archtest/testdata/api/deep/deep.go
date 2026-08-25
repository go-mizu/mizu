// Package deep spells store.Record out in every way a signature can, so that a
// walk over parameter types has something to be wrong about.
package deep

import "mizu.test/api/store"

// A Pair is generic, so that a type argument is somewhere to hide a package.
type Pair[T any] struct{ A, B T }

// An Alias is another name for somebody else's type.
type Alias = store.Record

// Chan has it behind a channel.
func Chan(c chan store.Record) {}

// Array has it behind an array.
func Array(a [2]store.Record) {}

// Variadic has it behind the ... which is a slice.
func Variadic(rs ...store.Record) {}

// Args has it as a type argument.
func Args(p Pair[store.Record]) {}

// Anon has it inside a struct written out at the call, which the caller has to
// write out too. It calls hidden so that hidden is small enough and near enough
// to an exported function to end up in the export data.
func Anon(o struct{ R store.Record }) { hidden(o.R) }

// Iface has it in a method of an interface written out at the call, which
// whatever satisfies the interface has to name.
func Iface(x interface{ Put(store.Record) error }) {}

// Result has it on the way back, which costs a caller nothing.
func Result() store.Record { return store.Record{} }

// Aliased has it under another name.
func Aliased(r Alias) {}

// Constrained has it as a constraint, which is satisfied by an int or a string
// rather than named.
func Constrained[T store.Key](v T) {}

// Nothing stays inside the standard library.
func Nothing(s string, n int) error { return nil }

// hidden is unexported.
func hidden(r store.Record) {}
