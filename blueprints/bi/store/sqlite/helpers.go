package sqlite

import (
	"encoding/json"

	"github.com/oklog/ulid/v2"
)

// generateID generates a new ULID string for use as a primary key.
func generateID() string {
	return ulid.Make().String()
}

// toJSON marshals a value to a JSON string. Returns "{}" for nil values.
func toJSON(v interface{}) string {
	if v == nil {
		return "{}"
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// fromJSON unmarshals a JSON string into a value.
func fromJSON(s string, v interface{}) error {
	if s == "" {
		return nil
	}
	return json.Unmarshal([]byte(s), v)
}
