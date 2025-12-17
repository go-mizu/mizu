package live

import (
	"encoding/json"
	"testing"
)

func TestProtocol(t *testing.T) {
	t.Run("encode and decode message", func(t *testing.T) {
		payload := ReplyPayload{
			Status:    "ok",
			SessionID: "abc123",
		}

		data, err := encodeMessage(MsgTypeReply, 42, payload)
		if err != nil {
			t.Fatalf("encode error: %v", err)
		}

		msg, err := decodeMessage(data)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}

		if msg.Type != MsgTypeReply {
			t.Errorf("expected type %d, got %d", MsgTypeReply, msg.Type)
		}
		if msg.Ref != 42 {
			t.Errorf("expected ref 42, got %d", msg.Ref)
		}

		var decoded ReplyPayload
		if err := msg.parsePayload(&decoded); err != nil {
			t.Fatalf("parse payload error: %v", err)
		}

		if decoded.Status != "ok" {
			t.Errorf("expected status ok, got %s", decoded.Status)
		}
		if decoded.SessionID != "abc123" {
			t.Errorf("expected session abc123, got %s", decoded.SessionID)
		}
	})

	t.Run("encode nil payload", func(t *testing.T) {
		data, err := encodeMessage(MsgTypeHeartbeat, 1, nil)
		if err != nil {
			t.Fatalf("encode error: %v", err)
		}

		msg, err := decodeMessage(data)
		if err != nil {
			t.Fatalf("decode error: %v", err)
		}

		if msg.Type != MsgTypeHeartbeat {
			t.Error("expected heartbeat type")
		}
	})

	t.Run("join payload", func(t *testing.T) {
		payload := JoinPayload{
			Token:     "csrf-token",
			URL:       "/counter",
			Params:    map[string]string{"id": "123"},
			SessionID: "existing",
			Reconnect: true,
		}

		data, _ := json.Marshal(payload)
		var decoded JoinPayload
		json.Unmarshal(data, &decoded)

		if decoded.Token != "csrf-token" {
			t.Error("token mismatch")
		}
		if decoded.URL != "/counter" {
			t.Error("url mismatch")
		}
		if decoded.Params["id"] != "123" {
			t.Error("params mismatch")
		}
		if !decoded.Reconnect {
			t.Error("reconnect mismatch")
		}
	})

	t.Run("patch payload", func(t *testing.T) {
		payload := PatchPayload{
			Regions: []RegionPatch{
				{ID: "stats", HTML: "<div>42</div>", Action: "morph"},
				{ID: "log", HTML: "<ul><li>inc</li></ul>", Action: "replace"},
			},
			Title: "New Title",
		}

		data, _ := json.Marshal(payload)
		var decoded PatchPayload
		json.Unmarshal(data, &decoded)

		if len(decoded.Regions) != 2 {
			t.Errorf("expected 2 regions, got %d", len(decoded.Regions))
		}
		if decoded.Regions[0].ID != "stats" {
			t.Error("region id mismatch")
		}
		if decoded.Title != "New Title" {
			t.Error("title mismatch")
		}
	})

	t.Run("error payload", func(t *testing.T) {
		payload := ErrorPayload{
			Code:        "session_expired",
			Message:     "Your session has expired",
			Recoverable: true,
		}

		data, _ := json.Marshal(payload)
		var decoded ErrorPayload
		json.Unmarshal(data, &decoded)

		if decoded.Code != "session_expired" {
			t.Error("code mismatch")
		}
		if !decoded.Recoverable {
			t.Error("recoverable mismatch")
		}
	})

	t.Run("message types", func(t *testing.T) {
		tests := []struct {
			msgType byte
			name    string
		}{
			{MsgTypeJoin, "join"},
			{MsgTypeLeave, "leave"},
			{MsgTypeEvent, "event"},
			{MsgTypeHeartbeat, "heartbeat"},
			{MsgTypeReply, "reply"},
			{MsgTypePatch, "patch"},
			{MsgTypeCommand, "command"},
			{MsgTypeError, "error"},
			{MsgTypeRedirect, "redirect"},
			{MsgTypeClose, "close"},
		}

		for _, tt := range tests {
			data, err := encodeMessage(tt.msgType, 0, nil)
			if err != nil {
				t.Errorf("%s: encode error: %v", tt.name, err)
				continue
			}

			msg, err := decodeMessage(data)
			if err != nil {
				t.Errorf("%s: decode error: %v", tt.name, err)
				continue
			}

			if msg.Type != tt.msgType {
				t.Errorf("%s: type mismatch: expected %d, got %d", tt.name, tt.msgType, msg.Type)
			}
		}
	})
}
