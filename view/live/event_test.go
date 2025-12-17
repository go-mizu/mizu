package live

import (
	"net/url"
	"testing"
)

func TestEvent(t *testing.T) {
	t.Run("get from values", func(t *testing.T) {
		e := Event{
			Name:   "click",
			Values: map[string]string{"id": "123", "action": "delete"},
		}

		if e.Get("id") != "123" {
			t.Errorf("expected id=123, got %s", e.Get("id"))
		}
		if e.Get("action") != "delete" {
			t.Errorf("expected action=delete, got %s", e.Get("action"))
		}
		if e.Get("missing") != "" {
			t.Error("expected empty for missing key")
		}
	})

	t.Run("get from form", func(t *testing.T) {
		e := Event{
			Name: "submit",
			Form: url.Values{
				"email":    {"user@example.com"},
				"password": {"secret"},
			},
		}

		if e.Get("email") != "user@example.com" {
			t.Errorf("expected email, got %s", e.Get("email"))
		}
	})

	t.Run("values take precedence over form", func(t *testing.T) {
		e := Event{
			Name:   "submit",
			Values: map[string]string{"name": "values"},
			Form:   url.Values{"name": {"form"}},
		}

		if e.Get("name") != "values" {
			t.Errorf("expected values to take precedence, got %s", e.Get("name"))
		}
	})

	t.Run("get int", func(t *testing.T) {
		e := Event{
			Values: map[string]string{
				"count": "42",
				"bad":   "not-a-number",
			},
		}

		if e.GetInt("count") != 42 {
			t.Errorf("expected 42, got %d", e.GetInt("count"))
		}
		if e.GetInt("bad") != 0 {
			t.Error("expected 0 for invalid int")
		}
		if e.GetInt("missing") != 0 {
			t.Error("expected 0 for missing int")
		}
	})

	t.Run("get int64", func(t *testing.T) {
		e := Event{
			Values: map[string]string{"big": "9223372036854775807"},
		}

		if e.GetInt64("big") != 9223372036854775807 {
			t.Error("expected max int64")
		}
	})

	t.Run("get float", func(t *testing.T) {
		e := Event{
			Values: map[string]string{"price": "19.99"},
		}

		if e.GetFloat("price") != 19.99 {
			t.Errorf("expected 19.99, got %f", e.GetFloat("price"))
		}
	})

	t.Run("get bool", func(t *testing.T) {
		e := Event{
			Values: map[string]string{
				"enabled":  "true",
				"disabled": "false",
				"one":      "1",
			},
		}

		if !e.GetBool("enabled") {
			t.Error("expected enabled=true")
		}
		if e.GetBool("disabled") {
			t.Error("expected disabled=false")
		}
		if !e.GetBool("one") {
			t.Error("expected one=true")
		}
	})

	t.Run("get all", func(t *testing.T) {
		e := Event{
			Form: url.Values{
				"tags": {"go", "web", "live"},
			},
		}

		tags := e.GetAll("tags")
		if len(tags) != 3 {
			t.Errorf("expected 3 tags, got %d", len(tags))
		}
	})

	t.Run("has", func(t *testing.T) {
		e := Event{
			Values: map[string]string{"a": "1"},
			Form:   url.Values{"b": {"2"}},
		}

		if !e.Has("a") {
			t.Error("expected has(a)=true")
		}
		if !e.Has("b") {
			t.Error("expected has(b)=true")
		}
		if e.Has("c") {
			t.Error("expected has(c)=false")
		}
	})
}

func TestEventPayload(t *testing.T) {
	t.Run("to event", func(t *testing.T) {
		p := eventPayload{
			Name:   "click",
			Target: "btn",
			Values: map[string]string{"id": "1"},
			Form:   map[string][]string{"name": {"test"}},
			Key:    "Enter",
			Meta:   EventMeta{ShiftKey: true},
		}

		e := p.toEvent()

		if e.Name != "click" {
			t.Errorf("expected name=click, got %s", e.Name)
		}
		if e.Target != "btn" {
			t.Errorf("expected target=btn, got %s", e.Target)
		}
		if e.Get("id") != "1" {
			t.Error("expected values")
		}
		if e.Form.Get("name") != "test" {
			t.Error("expected form")
		}
		if e.Key != "Enter" {
			t.Error("expected key")
		}
		if !e.Meta.ShiftKey {
			t.Error("expected shift key")
		}
	})
}
