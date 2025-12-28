package assets

import (
	"embed"
	"io/fs"
)

//go:embed static/css/*.css static/js/*.js static/img/*.svg
var staticFS embed.FS

//go:embed views/layouts/*.html views/*.html
var viewsFS embed.FS

// StaticFS returns the static files filesystem
func StaticFS() (fs.FS, error) {
	return fs.Sub(staticFS, "static")
}

// ViewsFS returns the views filesystem
func ViewsFS() (fs.FS, error) {
	return fs.Sub(viewsFS, "views")
}
