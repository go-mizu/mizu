module github.com/go-mizu/mizu/cmd

go 1.24.11

// Local development: replaced by the replace directive below
// Remote install: uses latest published version
require github.com/go-mizu/mizu v0.0.0

// Workspace replace directive (active for local development)
// When installing via `go install`, this is ignored and Go fetches
// the actual published version from the remote repository
replace github.com/go-mizu/mizu => ../
