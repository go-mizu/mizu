// Package store imports model, so that a test can check the loader resolves a
// type across two packages it loaded from source in the same run.
package store

import (
	"encoding/json"

	"mizu.test/gen/model"
)

// Find returns a user, or nil if there is not one.
//
//mizu:query find
func Find(id int64) *model.User { return nil }

// Encode writes a user as JSON, which is here so the package reaches the
// standard library as well as its neighbour.
func Encode(u *model.User) ([]byte, error) { return json.Marshal(u) }
