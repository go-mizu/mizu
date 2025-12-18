package async

import "encoding/json"

// Request is the submit request structure.
type Request struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Response is the SSE event data structure.
type Response struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

// Error is the error structure for async responses.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// acceptedResponse is returned immediately on submit.
type acceptedResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}
