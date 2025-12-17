package live

import (
	"encoding/json"
)

// Codec defines message encoding for the wire protocol.
type Codec interface {
	// Encode serializes a message to bytes.
	Encode(Message) ([]byte, error)

	// Decode deserializes bytes to a message.
	Decode([]byte) (Message, error)
}

// JSONCodec encodes messages as JSON.
// This is the default codec.
type JSONCodec struct{}

// Encode serializes a message to JSON.
func (JSONCodec) Encode(m Message) ([]byte, error) {
	return json.Marshal(m)
}

// Decode deserializes JSON to a message.
func (JSONCodec) Decode(data []byte) (Message, error) {
	var m Message
	if err := json.Unmarshal(data, &m); err != nil {
		return Message{}, err
	}
	return m, nil
}

// Ensure JSONCodec implements Codec.
var _ Codec = JSONCodec{}
