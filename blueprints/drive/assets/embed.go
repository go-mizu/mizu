// Package assets provides embedded static files and templates.
package assets

import (
	"embed"
	"io/fs"
)

//go:embed static views
var FS embed.FS

// Static returns the static files filesystem.
func Static() fs.FS {
	sub, _ := fs.Sub(FS, "static")
	return sub
}

// Views returns the views filesystem.
func Views() fs.FS {
	sub, _ := fs.Sub(FS, "views")
	return sub
}
