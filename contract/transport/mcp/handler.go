package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-mizu/mizu/contract"
)

// Handler handles MCP requests over HTTP (Streamable HTTP transport).
type Handler struct {
	service    *contract.Service
	resolver   contract.Resolver
	invoker    contract.TransportInvoker
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

// WithResolver sets a custom method resolver.
func WithResolver(r contract.Resolver) Option {
	return func(h *Handler) { h.resolver = r }
}

// WithInvoker sets a custom method invoker.
func WithInvoker(i contract.TransportInvoker) Option {
	return func(h *Handler) { h.invoker = i }
}

// NewHandler creates a new MCP handler.
func NewHandler(svc *contract.Service, opts ...Option) *Handler {
	h := &Handler{
		service:    svc,
		serverInfo: DefaultServerInfo(),
		resolver:   &contract.ServiceResolver{Service: svc},
		invoker:    &contract.DefaultInvoker{},
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
	// Minimal SSE response for endpoint validation
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	_, _ = w.Write([]byte(": mcp\n\n"))
}

func (h *Handler) handlePost(w http.ResponseWriter, r *http.Request) {
	// Validate protocol version if present
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

	raw = contract.TrimJSONSpace(raw)
	if len(raw) == 0 {
		h.writeRPCError(w, nil, invalidRequest("empty body"))
		return
	}

	// Check for response or notification (return 202)
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

	// Handle notification (no ID)
	if len(req.ID) == 0 || contract.IsJSONNull(req.ID) {
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

	method := h.resolver.Resolve(name)
	if method == nil {
		h.writeRPCError(w, id, methodNotFound("tools/call: "+name))
		return
	}

	result, err := h.invoker.Invoke(ctx, method, p.Arguments)
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
		if schema, ok := h.service.Types.Schema(m.Input.ID); ok {
			tool.InputSchema = schema.JSON
		}
	}

	if m.Output != nil {
		if schema, ok := h.service.Types.Schema(m.Output.ID); ok {
			tool.OutputSchema = schema.JSON
		}
	}

	return tool
}

func (h *Handler) allowOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}

	// Check allowed origins
	if len(h.allowedOrigins) > 0 {
		for _, allowed := range h.allowedOrigins {
			if allowed == "*" || allowed == origin {
				return true
			}
		}
		return false
	}

	// Default: same-host check
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

// isRPCResponse checks if raw JSON is a JSON-RPC response.
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

// isRPCNotification checks if raw JSON is a JSON-RPC notification.
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
