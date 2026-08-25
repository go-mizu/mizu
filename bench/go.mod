module github.com/go-mizu/mizu/bench

go 1.27

require (
	github.com/go-mizu/mizu v0.0.0
	golang.org/x/tools v0.49.0
)

require (
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/mod v0.39.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/term v0.45.0 // indirect
)

// The benchmarks measure the toolkit in this repository rather than a published
// version of it, which is the whole point of them being here.
replace github.com/go-mizu/mizu => ../
