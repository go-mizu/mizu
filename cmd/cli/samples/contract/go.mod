module example.com/contract

go 1.24.11

require github.com/go-mizu/mizu v0.5.21

// For local development, use replace directive:
// replace github.com/go-mizu/mizu => /path/to/mizu
replace github.com/go-mizu/mizu => ../../../..
