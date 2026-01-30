//go:build !cgo

package fts_zig

func newCGODriver(_ Config) (Driver, error) {
	return nil, ErrCGODisabled
}
