package mizu

// The JSON these three functions speak is the same JSON either way, but the
// package that speaks it is not. Go 1.27 builds encoding/json/v2 by default and
// leaves the old one in place, so mizu still compiles under
// GOEXPERIMENT=nojsonv2 and json_v1.go is what it compiles then. The two files
// are held to the same behaviour rather than to the same code: one JSON value
// in, unknown members and trailing data refused, map keys sorted and nothing
// HTML escaped on the way out. json_test.go is that contract written down, and
// running the package both ways is what checks it.
//
// What the two do not promise is the same error text. A body with two values in
// it fails either way and says so differently, and no caller reads further into
// it than the message.
//
// Nothing outside those two files names encoding/json, so a third version, or a
// decision to stop carrying the old one, is a change to two files and no call
// sites.
