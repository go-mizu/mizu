module commandtest

go 1.27

require github.com/go-mizu/mizu v0.0.0

require (
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/term v0.45.0 // indirect
)

replace github.com/go-mizu/mizu => ../../..
