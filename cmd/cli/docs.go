package cli

import "embed"

// DocsFS provides access to embedded documentation files.
//
//go:embed docs/*.md docs/commands/*.md
var DocsFS embed.FS
