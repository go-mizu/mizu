package assets

import (
	"embed"
	"io/fs"
)

//go:embed static
var content embed.FS

// Static returns the static file system.
func Static() fs.FS {
	static, _ := fs.Sub(content, "static")
	return static
}
