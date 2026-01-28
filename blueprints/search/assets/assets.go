package assets

import "embed"

// StaticFS contains the embedded static files for the frontend.
//
//go:embed all:static
var StaticFS embed.FS
