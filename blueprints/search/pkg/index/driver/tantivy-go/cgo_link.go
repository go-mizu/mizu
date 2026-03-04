//go:build tantivy

package tantivy

/*
// Add the platform-specific directory containing libtantivy_go.a to the
// linker search path.  ${SRCDIR} expands to the absolute path of the
// directory holding this source file.
#cgo darwin,arm64 LDFLAGS: -L${SRCDIR}/libs/darwin-arm64
#cgo darwin,amd64 LDFLAGS: -L${SRCDIR}/libs/darwin-amd64
#cgo linux,amd64 LDFLAGS: -L${SRCDIR}/libs/linux-amd64-musl
#cgo linux,arm64 LDFLAGS: -L${SRCDIR}/libs/linux-arm64-musl
*/
import "C"
