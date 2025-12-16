package contract

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	mcpProtocolLatest   = "2025-06-18"
	mcpProtocolFallback = "2025-03-26"
)

// MountMCP mounts an MCP (Model Context Protocol) endpoint using Streamable HTTP.
//
// Deprecated: Use mcp.Mount from github.com/go-mizu/mizu/contract/transport/mcp instead.
// The new package provides additional options like custom server info, instructions,
// and allowed origins.
//
// Supported (tools-only):
//   - initialize
//   - notifications/initialized (accepted, 202)
//   - tools/list
//   - tools/call
//
// Transport behavior (HTTP):
//   - POST with JSON-RPC request -> 200 + application/json response
//   - POST with JSON-RPC notification/response -> 202 Accepted, no body
//   - GET -> returns a minimal SSE response (valid endpoint, no server-initiated messages)
func MountMCP(mux *http.ServeMux, path string, svc *Service) {
	if mux == nil || svc == nil {
		return
	}
	if path == "" {
		path = "/mcp"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	serverInfo := mcpServerInfo{
		Name:    "mizu-contract",
		Title:   "Mizu Contract MCP Server",
		Version: "0.1.0",
	}

	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if !mcpAllowOrigin(r) {
			http.Error(w, "invalid origin", http.StatusForbidden)
			return
		}

		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			_, _ = w.Write([]byte(": mcp\n\n"))
			return

		case http.MethodPost:
			if pv := strings.TrimSpace(r.Header.Get("MCP-Protocol-Version")); pv != "" {
				if !mcpProtocolSupported(pv) {
					http.Error(w, "unsupported MCP-Protocol-Version", http.StatusBadRequest)
					return
				}
			}

			var raw json.RawMessage
			dec := json.NewDecoder(r.Body)
			if err := dec.Decode(&raw); err != nil {
				mcpWriteRPCError(w, nil, rpcParseError(err))
				return
			}

			raw = TrimJSONSpace(raw)
			if len(raw) == 0 {
				mcpWriteRPCError(w, nil, rpcInvalidRequest(errors.New("empty body")))
				return
			}

			if isJSONRPCResponse(raw) || isJSONRPCNotification(raw) {
				w.WriteHeader(http.StatusAccepted)
				return
			}

			var req rpcRequest
			if err := json.Unmarshal(raw, &req); err != nil {
				mcpWriteRPCError(w, nil, rpcParseError(err))
				return
			}
			if req.JSONRPC != "2.0" {
				mcpWriteRPCError(w, req.ID, rpcInvalidRequest(errors.New("jsonrpc must be 2.0")))
				return
			}
			if len(req.ID) == 0 || IsJSONNull(req.ID) {
				w.WriteHeader(http.StatusAccepted)
				return
			}

			switch req.Method {
			case "initialize":
				mcpHandleInitialize(w, req.ID, req.Params, serverInfo)

			case "notifications/initialized":
				w.WriteHeader(http.StatusAccepted)

			case "tools/list":
				mcpHandleToolsList(w, req.ID, req.Params, svc)

			case "tools/call":
				mcpHandleToolsCall(w, r.Context(), req.ID, req.Params, svc)

			default:
				mcpWriteRPCError(w, req.ID, rpcMethodNotFound(req.Method))
			}
			return

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	})
}

type mcpServerInfo struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version"`
}

type mcpInitializeParams struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    json.RawMessage `json:"capabilities,omitempty"`
	ClientInfo      json.RawMessage `json:"clientInfo,omitempty"`
}

type mcpInitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      mcpServerInfo  `json:"serverInfo"`
	Instructions    string         `json:"instructions,omitempty"`
}

func mcpHandleInitialize(w http.ResponseWriter, id json.RawMessage, params json.RawMessage, info mcpServerInfo) {
	var p mcpInitializeParams
	_ = json.Unmarshal(params, &p)

	negotiated := mcpNegotiateProtocol(p.ProtocolVersion)
	if negotiated == "" {
		mcpWriteRPCError(w, id, &rpcError{
			Code:    -32602,
			Message: "Unsupported protocol version",
			Data: map[string]any{
				"supported": []string{mcpProtocolLatest, "2024-11-05"},
				"requested": p.ProtocolVersion,
			},
		})
		return
	}

	res := mcpInitializeResult{
		ProtocolVersion: negotiated,
		Capabilities: map[string]any{
			"tools": map[string]any{
				"listChanged": false,
			},
		},
		ServerInfo: info,
	}

	mcpWriteRPCResult(w, id, res)
}

func mcpHandleToolsList(w http.ResponseWriter, id json.RawMessage, params json.RawMessage, svc *Service) {
	type listParams struct {
		Cursor string `json:"cursor,omitempty"`
	}
	var _p listParams
	_ = json.Unmarshal(params, &_p)

	tools := make([]map[string]any, 0, len(svc.Methods))
	for _, m := range svc.Methods {
		tools = append(tools, mcpToolDef(svc, m))
	}

	mcpWriteRPCResult(w, id, map[string]any{
		"tools": tools,
	})
}

func mcpHandleToolsCall(w http.ResponseWriter, ctx context.Context, id json.RawMessage, params json.RawMessage, svc *Service) {
	type callParams struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments,omitempty"`
	}
	var p callParams
	if err := json.Unmarshal(params, &p); err != nil {
		mcpWriteRPCError(w, id, rpcInvalidParams(err))
		return
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		mcpWriteRPCError(w, id, rpcInvalidParams(errors.New("missing tool name")))
		return
	}

	m := mcpResolveTool(svc, name)
	if m == nil {
		mcpWriteRPCError(w, id, rpcMethodNotFound("tools/call: "+name))
		return
	}

	var in any
	if m.Input != nil {
		in = m.NewInput()

		a := TrimJSONSpace(p.Arguments)
		if len(a) == 0 || IsJSONNull(a) {
			// zero value input
		} else if len(a) > 0 && a[0] == '{' {
			if err := json.Unmarshal(a, in); err != nil {
				mcpWriteRPCError(w, id, rpcInvalidParams(err))
				return
			}
		} else {
			mcpWriteRPCError(w, id, rpcInvalidParams(errors.New("arguments must be an object")))
			return
		}
	} else {
		a := TrimJSONSpace(p.Arguments)
		if len(a) != 0 && !IsJSONNull(a) {
			mcpWriteRPCError(w, id, rpcInvalidParams(errors.New("tool takes no arguments")))
			return
		}
	}

	out, err := m.Invoker.Call(ctx, in)
	if err != nil {
		mcpWriteRPCResult(w, id, map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": err.Error(),
				},
			},
			"isError": true,
		})
		return
	}

	text := "ok"
	if out != nil {
		b, _ := json.Marshal(out)
		text = string(b)
	}

	mcpWriteRPCResult(w, id, map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": text,
			},
		},
		"isError": false,
	})
}

func mcpToolDef(svc *Service, m *Method) map[string]any {
	tool := map[string]any{
		"name":        mcpToolName(svc, m),
		"title":       m.FullName,
		"description": m.Description,
	}

	if m.Input != nil {
		if schema, ok := svc.Types.Schema(m.Input.ID); ok {
			tool["inputSchema"] = schema.JSON
		} else {
			tool["inputSchema"] = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}
		}
	} else {
		tool["inputSchema"] = map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}
	if m.Output != nil {
		if schema, ok := svc.Types.Schema(m.Output.ID); ok {
			tool["outputSchema"] = schema.JSON
		}
	}

	return tool
}

func mcpToolName(svc *Service, m *Method) string {
	return svc.Name + "." + m.Name
}

func mcpResolveTool(svc *Service, name string) *Method {
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		if len(parts) != 2 {
			return nil
		}
		if parts[0] != svc.Name {
			return nil
		}
		return svc.Method(parts[1])
	}
	return svc.Method(name)
}

func mcpNegotiateProtocol(requested string) string {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return mcpProtocolLatest
	}
	if mcpProtocolSupported(requested) {
		return requested
	}
	return ""
}

func mcpProtocolSupported(v string) bool {
	switch v {
	case mcpProtocolLatest, "2025-03-26", "2024-11-05":
		return true
	default:
		return false
	}
}

// ---- JSON-RPC framing (minimal) ----

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func mcpWriteRPCResult(w http.ResponseWriter, id json.RawMessage, result any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func mcpWriteRPCError(w http.ResponseWriter, id json.RawMessage, e *rpcError) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   e,
	})
}

func rpcParseError(err error) *rpcError {
	return &rpcError{Code: -32700, Message: "Parse error", Data: SafeErrorString(err)}
}

func rpcInvalidRequest(err error) *rpcError {
	return &rpcError{Code: -32600, Message: "Invalid Request", Data: SafeErrorString(err)}
}

func rpcMethodNotFound(method string) *rpcError {
	return &rpcError{Code: -32601, Message: "Method not found", Data: map[string]any{"method": method}}
}

func rpcInvalidParams(err error) *rpcError {
	return &rpcError{Code: -32602, Message: "Invalid params", Data: SafeErrorString(err)}
}

func isJSONRPCResponse(raw []byte) bool {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return false
	}
	if _, ok := m["result"]; ok {
		return true
	}
	if _, ok := m["error"]; ok {
		return true
	}
	return false
}

func isJSONRPCNotification(raw []byte) bool {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return false
	}
	_, hasMethod := m["method"]
	_, hasID := m["id"]
	return hasMethod && !hasID
}

// ---- Security: Origin validation ----

func mcpAllowOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	return sameHost(u.Host, r.Host)
}

func sameHost(a, b string) bool {
	a = stripDefaultPort(strings.ToLower(a))
	b = stripDefaultPort(strings.ToLower(b))
	return a == b
}

func stripDefaultPort(h string) string {
	if strings.HasSuffix(h, ":80") {
		return strings.TrimSuffix(h, ":80")
	}
	if strings.HasSuffix(h, ":443") {
		return strings.TrimSuffix(h, ":443")
	}
	return h
}

var _ = time.Second
