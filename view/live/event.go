package live

import (
	"net/url"
	"strconv"
)

// Event represents a client-originated event.
type Event struct {
	// Name is the event name (e.g., "click", "submit", "inc").
	Name string

	// Target is the optional component ID (from data-lv-target).
	Target string

	// Values contains data-lv-value-* attributes.
	Values map[string]string

	// Form contains form field values (for submit events).
	Form url.Values

	// Key is the keyboard key (for keydown/keyup events).
	Key string

	// Meta contains event metadata (shiftKey, ctrlKey, etc.).
	Meta EventMeta
}

// EventMeta contains keyboard/mouse modifier state.
type EventMeta struct {
	ShiftKey bool `json:"shift"`
	CtrlKey  bool `json:"ctrl"`
	AltKey   bool `json:"alt"`
	MetaKey  bool `json:"meta"`
}

// Get retrieves a value from Values, then Form.
func (e Event) Get(key string) string {
	if v, ok := e.Values[key]; ok {
		return v
	}
	if e.Form != nil {
		return e.Form.Get(key)
	}
	return ""
}

// GetInt retrieves an integer value.
func (e Event) GetInt(key string) int {
	s := e.Get(key)
	if s == "" {
		return 0
	}
	v, _ := strconv.Atoi(s)
	return v
}

// GetInt64 retrieves a 64-bit integer value.
func (e Event) GetInt64(key string) int64 {
	s := e.Get(key)
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// GetFloat retrieves a float value.
func (e Event) GetFloat(key string) float64 {
	s := e.Get(key)
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// GetBool retrieves a boolean value.
func (e Event) GetBool(key string) bool {
	s := e.Get(key)
	if s == "" {
		return false
	}
	v, _ := strconv.ParseBool(s)
	return v
}

// GetAll retrieves all values for a key from Form.
func (e Event) GetAll(key string) []string {
	if e.Form != nil {
		return e.Form[key]
	}
	return nil
}

// Has checks if a key exists in Values or Form.
func (e Event) Has(key string) bool {
	if _, ok := e.Values[key]; ok {
		return true
	}
	if e.Form != nil {
		_, ok := e.Form[key]
		return ok
	}
	return false
}

// eventPayload is the wire format for events.
type eventPayload struct {
	Name   string            `json:"name"`
	Target string            `json:"target,omitempty"`
	Values map[string]string `json:"values,omitempty"`
	Form   map[string][]string `json:"form,omitempty"`
	Key    string            `json:"key,omitempty"`
	Meta   EventMeta         `json:"meta,omitempty"`
}

func (p *eventPayload) toEvent() Event {
	return Event{
		Name:   p.Name,
		Target: p.Target,
		Values: p.Values,
		Form:   url.Values(p.Form),
		Key:    p.Key,
		Meta:   p.Meta,
	}
}
