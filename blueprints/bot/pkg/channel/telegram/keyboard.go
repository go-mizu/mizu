package telegram

import (
	"encoding/json"
	"strings"
)

// OutboundButton represents a button in an inline keyboard.
type OutboundButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callbackData,omitempty"`
	URL          string `json:"url,omitempty"`
}

// buildInlineKeyboard creates a Telegram inline keyboard markup from button rows.
// It converts OutboundButton slices into the Telegram Bot API format with
// the "inline_keyboard" key containing rows of button objects.
func buildInlineKeyboard(buttons [][]OutboundButton) map[string]any {
	rows := make([][]map[string]any, 0, len(buttons))

	for _, row := range buttons {
		tgRow := make([]map[string]any, 0, len(row))
		for _, btn := range row {
			tgBtn := map[string]any{
				"text": btn.Text,
			}
			if btn.CallbackData != "" {
				tgBtn["callback_data"] = btn.CallbackData
			}
			if btn.URL != "" {
				tgBtn["url"] = btn.URL
			}
			tgRow = append(tgRow, tgBtn)
		}
		rows = append(rows, tgRow)
	}

	return map[string]any{
		"inline_keyboard": rows,
	}
}

// buildInlineKeyboardJSON creates the JSON bytes for an inline keyboard.
// It marshals the result of buildInlineKeyboard into a json.RawMessage.
func buildInlineKeyboardJSON(buttons [][]OutboundButton) (json.RawMessage, error) {
	markup := buildInlineKeyboard(buttons)
	data, err := json.Marshal(markup)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// parseCallbackData parses a callback query's data string.
// The expected format is "command:arg1:arg2:..." where the first segment
// is the command and the remaining segments are arguments.
// An empty data string returns an empty command and nil args.
func parseCallbackData(data string) (command string, args []string) {
	if data == "" {
		return "", nil
	}

	parts := strings.SplitN(data, ":", 2)
	command = parts[0]

	if len(parts) > 1 && parts[1] != "" {
		args = strings.Split(parts[1], ":")
	}

	return command, args
}
