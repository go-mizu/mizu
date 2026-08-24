// Package wire is the fixture's composition root. It imports app, and nothing
// in the module imports it, which is the property the Forbid tests assert.
package wire

import "mizu.test/graph/app"

// Main builds the whole thing.
func Main() any { return app.Serve("1", "hello") }
