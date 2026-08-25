// Package store is the leaf. Nothing outside the standard library appears in
// its signatures, so it is the package that passes the rule.
package store

import "encoding/json"

// A Record is what the other packages in this module have to name.
type Record struct {
	ID   string
	Body []byte
}

// Encode takes a type of this package's own, which is not a dependency for
// anybody calling it.
func Encode(r Record) ([]byte, error) { return json.Marshal(r) }

// Decode returns one, which costs a caller nothing at all.
func Decode(b []byte) (Record, error) {
	var r Record
	err := json.Unmarshal(b, &r)
	return r, err
}

// A Batch collects records.
type Batch struct{ records []Record }

// Add is an exported method taking a type of this package.
func (b *Batch) Add(r Record) { b.records = append(b.records, r) }

// count is unexported, and so is out of the API.
func (b *Batch) count() int { return len(b.records) }

// A Key is a constraint, which a caller satisfies rather than names.
type Key interface{ ~string | ~int }
