package micro

import "time"

// listing is the struct the binding benchmarks fill: twelve fields, in the shape
// a search page sends, with a slice and a date in it because both cost more than
// a string does.
//
// The tags are the names the fields would get anyway, since a field with no tag
// is read under its own name in snake case. They are written out so the JSON
// benchmark and the form benchmark are filling the same struct from the same
// names, rather than one of them leaning on a decoder that matches names
// loosely.
type listing struct {
	Q        string    `json:"q"`
	Tags     []string  `json:"tags"`
	Page     int       `json:"page"`
	PerPage  int       `json:"per_page"`
	Sort     string    `json:"sort"`
	Order    string    `json:"order"`
	MinPrice float64   `json:"min_price"`
	MaxPrice float64   `json:"max_price"`
	InStock  bool      `json:"in_stock"`
	Since    time.Time `json:"since"`
	Kind     string    `json:"kind"`
	Cursor   string    `json:"cursor"`
}

// genListing is the same struct with a generated binder on it.
//
// A defined type has the fields and the tags of the type it is over and none of
// its methods, so the bind rows and the bind/gen rows are filling the same
// twelve fields from the same names and the only difference between them is
// which binder ran. Writing the fields out twice would let the two drift, and a
// pair of numbers that are not measuring the same work is worse than no numbers.
//
// This file is not a test file, because a generator reads the package rather
// than the tests in it. bind_gen.go next to it is the output, checked in, and
// TestGeneratedBinderIsCurrent is what keeps it that way.
//
//mizu:bind
type genListing listing
