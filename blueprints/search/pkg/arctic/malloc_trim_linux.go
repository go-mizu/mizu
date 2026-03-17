//go:build linux

package arctic

// #include <malloc.h>
import "C"

// mallocTrim releases free pages from the glibc heap back to the OS.
// After DuckDB closes its C-side allocations, glibc retains the pages
// as RSS. Calling malloc_trim(0) returns them to the kernel, keeping
// RSS bounded when processing many sequential chunks.
func mallocTrim() {
	C.malloc_trim(0)
}
