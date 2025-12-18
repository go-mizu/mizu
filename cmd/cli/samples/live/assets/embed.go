package assets

import "embed"

// ViewsFS contains the embedded view templates.
//
//go:embed views
var ViewsFS embed.FS

// StaticFS contains the embedded static assets.
//
//go:embed static
var StaticFS embed.FS
