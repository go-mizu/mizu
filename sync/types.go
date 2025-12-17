package sync

import "time"

// Mutation represents a client request to change state.
// It is a command, not a state patch.
type Mutation struct {
	// ID uniquely identifies this mutation for idempotency.
	ID string `json:"id"`

	// Name identifies the mutation type.
	Name string `json:"name"`

	// Scope identifies the data partition.
	Scope string `json:"scope,omitempty"`

	// Client identifies the originating client.
	Client string `json:"client,omitempty"`

	// Seq is a client-local sequence number.
	Seq uint64 `json:"seq,omitempty"`

	// Args contains mutation-specific arguments.
	Args map[string]any `json:"args,omitempty"`
}

// Result describes the outcome of applying a mutation.
type Result struct {
	OK      bool     `json:"ok"`
	Cursor  uint64   `json:"cursor,omitempty"`
	Code    string   `json:"code,omitempty"`
	Error   string   `json:"error,omitempty"`
	Changes []Change `json:"changes,omitempty"`
}

// Change is a single durable state change recorded in the log.
type Change struct {
	Cursor uint64    `json:"cursor"`
	Scope  string    `json:"scope"`
	Entity string    `json:"entity"`
	ID     string    `json:"id"`
	Op     Op        `json:"op"`
	Data   []byte    `json:"data,omitempty"`
	Time   time.Time `json:"time"`
}

// Op defines the type of change operation.
type Op string

const (
	Create Op = "create"
	Update Op = "update"
	Delete Op = "delete"
)
