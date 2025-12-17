package live

import (
	"encoding/json"
)

// Message types for the wire protocol.
const (
	MsgTypeJoin        byte = 0x01
	MsgTypeLeave       byte = 0x02
	MsgTypeEvent       byte = 0x03
	MsgTypeHeartbeat   byte = 0x04
	MsgTypeReply       byte = 0x05
	MsgTypePatch       byte = 0x06
	MsgTypeCommand     byte = 0x07
	MsgTypeError       byte = 0x08
	MsgTypeRedirect    byte = 0x09
	MsgTypeClose       byte = 0x0A
	MsgTypePoke        byte = 0x0B // Sync: data changed notification
	MsgTypeSubscribe   byte = 0x0C // Sync: subscribe to scope
	MsgTypeUnsubscribe byte = 0x0D // Sync: unsubscribe from scope
)

// Message is the wire protocol envelope.
type Message struct {
	Type    byte            `json:"type"`
	Ref     uint32          `json:"ref,omitempty"`
	Topic   string          `json:"topic,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// JoinPayload is the payload for JOIN messages.
type JoinPayload struct {
	Token     string            `json:"token"`
	URL       string            `json:"url"`
	Params    map[string]string `json:"params,omitempty"`
	SessionID string            `json:"session,omitempty"`
	Reconnect bool              `json:"reconnect,omitempty"`
	Scopes    []string          `json:"scopes,omitempty"` // Sync scopes to subscribe to
}

// LeavePayload is the payload for LEAVE messages.
type LeavePayload struct {
	Reason string `json:"reason,omitempty"`
}

// HeartbeatPayload is the payload for HEARTBEAT messages.
type HeartbeatPayload struct {
	Ping int64 `json:"ping,omitempty"`
	Pong int64 `json:"pong,omitempty"`
}

// ReplyPayload is the payload for REPLY messages.
type ReplyPayload struct {
	Status    string            `json:"status"`
	SessionID string            `json:"session_id,omitempty"`
	Rendered  map[string]string `json:"rendered,omitempty"`
	Reason    string            `json:"reason,omitempty"`
	Message   string            `json:"message,omitempty"`
}

// PatchPayload is the payload for PATCH messages.
type PatchPayload struct {
	Regions []RegionPatch `json:"regions"`
	Title   string        `json:"title,omitempty"`
}

// RegionPatch represents a single region update.
type RegionPatch struct {
	ID     string `json:"id"`
	HTML   string `json:"html"`
	Action string `json:"action,omitempty"` // replace, morph, append, prepend, before, after, remove
}

// CommandPayload wraps commands for the wire.
type CommandPayload struct {
	Commands []commandEnvelope `json:"commands"`
}

// ErrorPayload is the payload for ERROR messages.
type ErrorPayload struct {
	Code        string `json:"code"`
	Message     string `json:"message,omitempty"`
	Recoverable bool   `json:"recoverable,omitempty"`
}

// ClosePayload is the payload for CLOSE messages.
type ClosePayload struct {
	Reason  string `json:"reason"`
	Message string `json:"message,omitempty"`
}

// PokePayload is the payload for POKE messages (sync integration).
type PokePayload struct {
	Scope  string `json:"scope"`
	Cursor uint64 `json:"cursor"`
}

// SubscribePayload is the payload for SUBSCRIBE messages.
type SubscribePayload struct {
	Scopes []string `json:"scopes"`
}

// UnsubscribePayload is the payload for UNSUBSCRIBE messages.
type UnsubscribePayload struct {
	Scopes []string `json:"scopes"`
}

// encodeMessage creates a wire message.
func encodeMessage(msgType byte, ref uint32, payload any) ([]byte, error) {
	var p json.RawMessage
	if payload != nil {
		var err error
		p, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}

	msg := Message{
		Type:    msgType,
		Ref:     ref,
		Payload: p,
	}
	return json.Marshal(msg)
}

// decodeMessage parses a wire message.
func decodeMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// parsePayload unmarshals the message payload into v.
func (m *Message) parsePayload(v any) error {
	if m.Payload == nil {
		return nil
	}
	return json.Unmarshal(m.Payload, v)
}
