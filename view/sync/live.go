package sync

import (
	"encoding/json"
	"strconv"
)

// LiveMessage represents a sync notification from the live package.
type LiveMessage struct {
	Type   string `json:"type"`
	Topic  string `json:"topic"`
	Cursor uint64 `json:"cursor"`
}

// ParseLiveMessage parses a sync notification from live message body.
func ParseLiveMessage(body []byte) (LiveMessage, error) {
	var msg LiveMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return msg, err
	}
	return msg, nil
}

// LiveHandler returns a message handler function for live integration.
// It should be called when receiving messages from a live connection.
//
// Usage:
//
//	handler := syncClient.LiveHandler()
//	liveConn.OnMessage = func(msg live.Message) {
//	    if msg.Type == "sync" {
//	        handler(msg.Topic, msg.Body)
//	    }
//	}
func (c *Client) LiveHandler() func(topic string, body []byte) {
	return func(topic string, body []byte) {
		msg, err := ParseLiveMessage(body)
		if err != nil {
			// Try parsing cursor directly
			cursor, err := parseCursor(body)
			if err != nil {
				return
			}
			c.NotifyLive(cursor)
			return
		}
		c.NotifyLive(msg.Cursor)
	}
}

// LiveTopic returns the topic name for this client's scope.
// This should be subscribed to on the live connection.
func (c *Client) LiveTopic() string {
	return "sync:" + c.opts.Scope
}

// parseCursor attempts to parse a cursor from JSON like {"cursor":123}
func parseCursor(body []byte) (uint64, error) {
	var data struct {
		Cursor uint64 `json:"cursor"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		// Try parsing as plain number
		n, err := strconv.ParseUint(string(body), 10, 64)
		if err != nil {
			return 0, err
		}
		return n, nil
	}
	return data.Cursor, nil
}

// WithLive is a helper that sets up live integration for the client.
// It returns a function that should be called to process live messages.
//
// Example with live package:
//
//	client := sync.New(opts)
//	handler := sync.WithLive(client)
//
//	liveConn.OnMessage = func(msg live.Message) {
//	    if msg.Type == "sync" && msg.Topic == client.LiveTopic() {
//	        handler(msg.Body)
//	    }
//	}
func WithLive(c *Client) func(body []byte) {
	return func(body []byte) {
		cursor, err := parseCursor(body)
		if err != nil {
			return
		}
		c.NotifyLive(cursor)
	}
}
