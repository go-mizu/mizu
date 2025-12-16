package cli

import (
	"encoding/json"
	"io"
)

// jsonEncoder wraps json.Encoder with consistent settings.
type jsonEncoder struct {
	w io.Writer
}

func newJSONEncoder(w io.Writer) *jsonEncoder {
	return &jsonEncoder{w: w}
}

func (e *jsonEncoder) encode(v any) error {
	enc := json.NewEncoder(e.w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// jsonResult is the standard envelope for JSON output.
type jsonResult struct {
	Success bool       `json:"success"`
	Data    any        `json:"data,omitempty"`
	Error   *jsonError `json:"error,omitempty"`
}

type jsonError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (o *output) writeJSON(data any) {
	result := jsonResult{
		Success: true,
		Data:    data,
	}
	enc := newJSONEncoder(o.stdout)
	_ = enc.encode(result)
}

func (o *output) writeJSONError(code, message string) {
	result := jsonResult{
		Success: false,
		Error: &jsonError{
			Code:    code,
			Message: message,
		},
	}
	enc := newJSONEncoder(o.stderr)
	_ = enc.encode(result)
}
