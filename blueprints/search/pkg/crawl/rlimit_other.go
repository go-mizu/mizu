//go:build !linux

package crawl

func raiseRlimit(_ uint64) error { return nil }
