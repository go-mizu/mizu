package live

// Message is the transport envelope for all live communications.
// It carries typed messages between clients and server.
type Message struct {
	// Type identifies the message purpose (e.g., "subscribe", "message", "ack").
	// The live package does not interpret this field; higher layers define semantics.
	Type string `json:"type"`

	// Topic is the routing key for pub/sub operations.
	// Empty for messages that don't target a specific topic.
	Topic string `json:"topic,omitempty"`

	// Ref is a client-generated reference for correlating request/response pairs.
	// Servers should echo Ref in acknowledgments.
	Ref string `json:"ref,omitempty"`

	// Body contains the message payload as opaque bytes.
	// Higher layers define the schema. When using JSON codec, this is base64-encoded.
	Body []byte `json:"body,omitempty"`
}

// Meta holds authenticated connection metadata.
// Populated by OnAuth callback and accessible via Session.Meta().
// The live package treats this as read-only after creation.
type Meta map[string]any

// Get returns the value for key, or nil if not present.
func (m Meta) Get(key string) any {
	if m == nil {
		return nil
	}
	return m[key]
}

// GetString returns the string value for key, or empty string if not present or not a string.
func (m Meta) GetString(key string) string {
	v, _ := m.Get(key).(string)
	return v
}
