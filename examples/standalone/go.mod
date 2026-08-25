// This module is the standalone claim compiled. Every directory in it is a
// program that uses one mizu package and the standard library, and the one
// requirement below is what a person adopting that package would add to their
// own go.mod.
//
// The replace is here because the module is not tagged yet. It goes away with
// the first tag.
module mizu.example/standalone

go 1.27

require github.com/go-mizu/mizu v0.0.0

require (
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/term v0.45.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

replace github.com/go-mizu/mizu => ../..
