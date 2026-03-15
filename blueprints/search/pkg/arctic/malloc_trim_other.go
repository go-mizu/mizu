//go:build !linux

package arctic

// mallocTrim is a no-op on non-Linux platforms (malloc_trim is glibc-specific).
func mallocTrim() {}
