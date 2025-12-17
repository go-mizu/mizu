package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-mizu/mizu/contract/v1"
)

// Handler handles MCP requests over HTTP (Streamable HTTP transport).
type Handler struct {
	service    *contract.Service
	serverInfo ServerInfo

	allowedOrigins []string
	instructions   string
}

// Option configures the handler.
type Option func(*Handler)

// WithServerInfo sets custom server info.
func WithServerInfo(info ServerInfo) Option {
	return func(h *Handler) { h.serverInfo = info }
}

// WithInstructions sets initialization instructions.
func WithInstructions(instructions string) Option {
	return func(h *Handler) { h.instructions = instructions }
}

// WithAllowedOrigins sets allowed CORS origins.
func WithAllowedOrigins(origins ...string) Option {
	return func(h *Handler) { h.allowedOrigins = origins }
}

// NewHandler creates a new MCP handler.
func NewHandler(svc *contract.Service, opts ...Option) *Handler {
	h := &Handler{
		service:    svc,
		serverInfo: DefaultServerInfo(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Name returns the transport name.
func (h *Handler) Name() string {
	return "mcp"
}

// ServeHTTP handles HTTP requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.allowOrigin(r) {
		http.Error(w, "invalid origin", http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleSSE(w, r)
	case http.MethodPost:
		h.handlePost(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	_, _ = w.Write([]byte(": mcp\n\n"))
}

func (h *Handler) handlePost(w http.ResponseWriter, r *http.Request) {
	if pv := strings.TrimSpace(r.Header.Get("MCP-Protocol-Version")); pv != "" {
		if !isProtocolSupported(pv) {
			http.Error(w, "unsupported MCP-Protocol-Version", http.StatusBadRequest)
			return
		}
	}

	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		h.writeRPCError(w, nil, parseError(err))
		return
	}

	raw = trimJSONSpace(raw)
	if len(raw) == 0 {
		h.writeRPCError(w, nil, invalidRequest("empty body"))
		return
	}

	if isRPCResponse(raw) || isRPCNotification(raw) {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	var req rpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		h.writeRPCError(w, nil, parseError(err))
		return
	}

	if req.JSONRPC != "2.0" {
		h.writeRPCError(w, req.ID, invalidRequest("jsonrpc must be 2.0"))
		return
	}

	if len(req.ID) == 0 || isJSONNull(req.ID) {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	h.handleMethod(w, r.Context(), req)
}

func (h *Handler) handleMethod(w http.ResponseWriter, ctx context.Context, req rpcRequest) {
	switch req.Method {
	case "initialize":
		h.handleInitialize(w, req.ID, req.Params)
	case "notifications/initialized":
		w.WriteHeader(http.StatusAccepted)
	case "tools/list":
		h.handleToolsList(w, req.ID, req.Params)
	case "tools/call":
		h.handleToolsCall(w, ctx, req.ID, req.Params)
	default:
		h.writeRPCError(w, req.ID, methodNotFound(req.Method))
	}
}

func (h *Handler) handleInitialize(w http.ResponseWriter, id json.RawMessage, params json.RawMessage) {
	var p InitializeParams
	_ = json.Unmarshal(params, &p)

	negotiated := negotiateProtocol(p.ProtocolVersion)
	if negotiated == "" {
		h.writeRPCError(w, id, &rpcError{
			Code:    codeInvalidParams,
			Message: "Unsupported protocol version",
			Data: map[string]any{
				"supported": []string{ProtocolLatest, ProtocolFallback, ProtocolLegacy},
				"requested": p.ProtocolVersion,
			},
		})
		return
	}

	result := InitializeResult{
		ProtocolVersion: negotiated,
		Capabilities: Capabilities{
			Tools: &ToolCapabilities{ListChanged: false},
		},
		ServerInfo:   h.serverInfo,
		Instructions: h.instructions,
	}

	h.writeRPCResult(w, id, result)
}

func (h *Handler) handleToolsList(w http.ResponseWriter, id json.RawMessage, params json.RawMessage) {
	tools := make([]Tool, 0, len(h.service.Methods))
	for _, m := range h.service.Methods {
		tools = append(tools, h.buildTool(m))
	}
	h.writeRPCResult(w, id, map[string]any{"tools": tools})
}

func (h *Handler) handleToolsCall(w http.ResponseWriter, ctx context.Context, id json.RawMessage, params json.RawMessage) {
	var p ToolCallParams
	if err := json.Unmarshal(params, &p); err != nil {
		h.writeRPCError(w, id, invalidParams(err))
		return
	}

	name := strings.TrimSpace(p.Name)
	if name == "" {
		h.writeRPCError(w, id, invalidParams(nil))
		return
	}

	method := h.resolve(name)
	if method == nil {
		h.writeRPCError(w, id, methodNotFound("tools/call: "+name))
		return
	}

	result, err := h.invoke(ctx, method, p.Arguments)
	if err != nil {
		h.writeRPCResult(w, id, ErrorResult(err.Error()))
		return
	}

	text := "ok"
	if result != nil {
		b, _ := json.Marshal(result)
		text = string(b)
	}

	h.writeRPCResult(w, id, SuccessResult(text))
}

func (h *Handler) resolve(name string) *contract.Method {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		if len(parts) != 2 || parts[0] != h.service.Name {
			return nil
		}
		return h.service.Method(parts[1])
	}
	return h.service.Method(name)
}

func (h *Handler) invoke(ctx context.Context, m *contract.Method, params []byte) (any, error) {
	var in any
	if m.HasInput() {
		in = m.NewInput()
		params = trimJSONSpace(params)
		if len(params) > 0 && !isJSONNull(params) {
			if len(params) > 0 && params[0] == '{' {
				if err := json.Unmarshal(params, in); err != nil {
					return nil, contract.NewError(contract.InvalidArgument, "invalid input: "+err.Error())
				}
			} else {
				return nil, contract.NewError(contract.InvalidArgument, "input must be a JSON object")
			}
		}
	} else {
		params = trimJSONSpace(params)
		if len(params) > 0 && !isJSONNull(params) {
			return nil, contract.NewError(contract.InvalidArgument, "method does not accept parameters")
		}
	}
	return m.Call(ctx, in)
}

func (h *Handler) buildTool(m *contract.Method) Tool {
	tool := Tool{
		Name:        h.service.Name + "." + m.Name,
		Title:       m.FullName,
		Description: m.Description,
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}

	if m.Input != nil {
		if schema := h.service.Types.Schema(m.Input.ID); schema != nil {
			tool.InputSchema = schema
		}
	}

	if m.Output != nil {
		if schema := h.service.Types.Schema(m.Output.ID); schema != nil {
			tool.OutputSchema = schema
		}
	}

	return tool
}

func (h *Handler) allowOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}

	if len(h.allowedOrigins) > 0 {
		for _, allowed := range h.allowedOrigins {
			if allowed == "*" || allowed == origin {
				return true
			}
		}
		return false
	}

	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	return sameHost(u.Host, r.Host)
}

func (h *Handler) writeRPCResult(w http.ResponseWriter, id json.RawMessage, result any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func (h *Handler) writeRPCError(w http.ResponseWriter, id json.RawMessage, e *rpcError) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   e,
	})
}

// Mount registers the MCP handler at the given path.
func Mount(mux *http.ServeMux, path string, svc *contract.Service, opts ...Option) {
	if mux == nil || svc == nil {
		return
	}
	if path == "" {
		path = "/mcp"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	mux.Handle(path, NewHandler(svc, opts...))
}

func isRPCResponse(raw []byte) bool {
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

func isRPCNotification(raw []byte) bool {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return false
	}
	_, hasMethod := m["method"]
	_, hasID := m["id"]
	return hasMethod && !hasID
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

func trimJSONSpace(b []byte) []byte {
	i, j := 0, len(b)
	for i < j && isSpace(b[i]) {
		i++
	}
	for j > i && isSpace(b[j-1]) {
		j--
	}
	return b[i:j]
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\n' || c == '\r' || c == '\t'
}

func isJSONNull(b []byte) bool {
	b = trimJSONSpace(b)
	if len(b) != 4 {
		return false
	}
	return (b[0] == 'n' || b[0] == 'N') &&
		(b[1] == 'u' || b[1] == 'U') &&
		(b[2] == 'l' || b[2] == 'L') &&
		(b[3] == 'l' || b[3] == 'L')
}
