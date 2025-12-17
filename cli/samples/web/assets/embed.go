package assets

import "embed"

// ViewsFS contains all view templates.
//
//go:embed views
var ViewsFS embed.FS

// StaticFS contains all static assets.
//
//go:embed static
var StaticFS embed.FS
