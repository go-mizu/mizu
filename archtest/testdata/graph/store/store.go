// Package store is a leaf that reaches the standard library and nothing else.
package store

import "encoding/json"

// Record is one stored thing.
type Record struct {
	ID   string `json:"id"`
	Body string `json:"body"`
}

// Encode writes r as JSON.
func Encode(r Record) ([]byte, error) {
	return json.Marshal(r)
}
