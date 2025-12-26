// Package assets provides embedded static files.
package assets

import (
	"embed"
	"io/fs"
)

//go:embed static
var FS embed.FS

// Static returns the static files filesystem.
func Static() fs.FS {
	sub, _ := fs.Sub(FS, "static")
	return sub
}
