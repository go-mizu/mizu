package handler

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-mizu/mizu/live"
)

// CounterView handles the counter live view.
type CounterView struct {
	sessions map[string]*CounterSession
	mu       sync.RWMutex
}

// CounterSession holds per-session state for the counter view.
type CounterSession struct {
	session *live.Session
	count   int
}

// NewCounterView creates a new counter view handler.
func NewCounterView() *CounterView {
	return &CounterView{
		sessions: make(map[string]*CounterSession),
	}
}

// Mount handles the mount message for a new session.
func (v *CounterView) Mount(s *live.Session, topic, ref string) {
	v.mu.Lock()
	v.sessions[s.ID()] = &CounterSession{
		session: s,
		count:   0,
	}
	v.mu.Unlock()

	// Send mounted confirmation
	v.sendMessage(s, topic, "mounted", ref, map[string]any{
		"sessionId": s.ID(),
	})
}

// HandleEvent processes UI events from the client.
func (v *CounterView) HandleEvent(s *live.Session, topic string, payload []byte) {
	var event Event
	if err := json.Unmarshal(payload, &event); err != nil {
		return
	}

	v.mu.Lock()
	cs := v.sessions[s.ID()]
	if cs == nil {
		v.mu.Unlock()
		return
	}

	switch event.Target {
	case "increment":
		cs.count++
	case "decrement":
		cs.count--
	case "reset":
		cs.count = 0
	case "add":
		// Handle custom value add
		if n, ok := event.Data["value"].(float64); ok {
			cs.count += int(n)
		}
	}
	count := cs.count
	v.mu.Unlock()

	// Send patch to update the UI
	patches := []Patch{
		{Op: "replace", Target: "#count", HTML: fmt.Sprintf("%d", count)},
	}

	v.sendMessage(s, topic, "patch", "", map[string]any{
		"patches": patches,
	})
}

// RemoveSession cleans up when a session disconnects.
func (v *CounterView) RemoveSession(sessionID string) {
	v.mu.Lock()
	delete(v.sessions, sessionID)
	v.mu.Unlock()
}

// sendMessage sends a typed message to the client.
func (v *CounterView) sendMessage(s *live.Session, topic, msgType, ref string, payload any) {
	msg := struct {
		Type    string `json:"type"`
		Ref     string `json:"ref,omitempty"`
		Payload any    `json:"payload,omitempty"`
	}{
		Type:    msgType,
		Ref:     ref,
		Payload: payload,
	}
	data, _ := json.Marshal(msg)
	_ = s.Send(live.Message{Topic: topic, Data: data})
}

// Event represents a UI event from the client.
type Event struct {
	Type   string         `json:"type"`
	Target string         `json:"target"`
	Value  string         `json:"value"`
	Data   map[string]any `json:"data"`
}

// Patch represents a DOM patch to send to the client.
type Patch struct {
	Op     string `json:"op"`
	Target string `json:"target"`
	HTML   string `json:"html,omitempty"`
	Attr   string `json:"attr,omitempty"`
	Value  string `json:"value,omitempty"`
}
