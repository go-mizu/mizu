package live

import (
	"encoding/json"
	"testing"
)

func TestCommands(t *testing.T) {
	t.Run("redirect command", func(t *testing.T) {
		cmd := Redirect{To: "/dashboard", Replace: true}

		if cmd.commandType() != "redirect" {
			t.Error("expected redirect type")
		}

		env := wrapCommand(cmd)
		if env.Cmd != "redirect" {
			t.Error("expected redirect cmd")
		}

		data, _ := json.Marshal(env)
		if len(data) == 0 {
			t.Error("expected json output")
		}
	})

	t.Run("focus command", func(t *testing.T) {
		cmd := Focus{Selector: "#email-input"}

		if cmd.commandType() != "focus" {
			t.Error("expected focus type")
		}

		env := wrapCommand(cmd)
		data, _ := json.Marshal(env)

		var decoded struct {
			Cmd  string `json:"cmd"`
			Data struct {
				Selector string `json:"selector"`
			} `json:"data"`
		}
		json.Unmarshal(data, &decoded)

		if decoded.Cmd != "focus" {
			t.Error("cmd mismatch")
		}
		if decoded.Data.Selector != "#email-input" {
			t.Error("selector mismatch")
		}
	})

	t.Run("scroll command", func(t *testing.T) {
		cmd := Scroll{Selector: "#messages", Block: "end"}

		if cmd.commandType() != "scroll" {
			t.Error("expected scroll type")
		}
	})

	t.Run("download command", func(t *testing.T) {
		cmd := Download{URL: "/export.csv", Filename: "report.csv"}

		if cmd.commandType() != "download" {
			t.Error("expected download type")
		}
	})

	t.Run("js command", func(t *testing.T) {
		cmd := JS{
			Code: "console.log(args.msg)",
			Args: map[string]any{"msg": "hello"},
		}

		if cmd.commandType() != "js" {
			t.Error("expected js type")
		}

		env := wrapCommand(cmd)
		data, _ := json.Marshal(env)

		var decoded struct {
			Cmd  string `json:"cmd"`
			Data struct {
				Code string         `json:"code"`
				Args map[string]any `json:"args"`
			} `json:"data"`
		}
		json.Unmarshal(data, &decoded)

		if decoded.Data.Code != "console.log(args.msg)" {
			t.Error("code mismatch")
		}
	})

	t.Run("set title command", func(t *testing.T) {
		cmd := SetTitle{Title: "New Page Title"}

		if cmd.commandType() != "title" {
			t.Error("expected title type")
		}
	})

	t.Run("class commands", func(t *testing.T) {
		tests := []struct {
			cmd      Command
			expected string
		}{
			{AddClass{Selector: ".btn", Class: "active"}, "add_class"},
			{RemoveClass{Selector: ".btn", Class: "active"}, "remove_class"},
			{ToggleClass{Selector: ".btn", Class: "active"}, "toggle_class"},
		}

		for _, tt := range tests {
			if tt.cmd.commandType() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.cmd.commandType())
			}
		}
	})

	t.Run("attribute commands", func(t *testing.T) {
		set := SetAttribute{Selector: "input", Name: "disabled", Value: "true"}
		if set.commandType() != "set_attr" {
			t.Error("expected set_attr type")
		}

		rem := RemoveAttribute{Selector: "input", Name: "disabled"}
		if rem.commandType() != "remove_attr" {
			t.Error("expected remove_attr type")
		}
	})

	t.Run("command payload", func(t *testing.T) {
		payload := CommandPayload{
			Commands: []commandEnvelope{
				wrapCommand(Redirect{To: "/home"}),
				wrapCommand(Focus{Selector: "#name"}),
			},
		}

		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		var decoded CommandPayload
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		if len(decoded.Commands) != 2 {
			t.Errorf("expected 2 commands, got %d", len(decoded.Commands))
		}
	})
}
