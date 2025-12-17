package trpc

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu/contract/v1"
)

// Handler handles tRPC requests over HTTP.
type Handler struct {
	service  *contract.Service
	basePath string
}

// Option configures the handler.
type Option func(*Handler)

// NewHandler creates a new tRPC handler.
func NewHandler(basePath string, svc *contract.Service, opts ...Option) *Handler {
	h := &Handler{
		service:  svc,
		basePath: basePath,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Name returns the transport name.
func (h *Handler) Name() string {
	return "trpc"
}

// MetaHandler returns a handler for the .meta endpoint.
func (h *Handler) MetaHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		meta := h.buildMeta()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(meta)
	}
}

// CallHandler returns a handler for procedure calls.
func (h *Handler) CallHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		proc := strings.TrimPrefix(r.URL.Path, h.basePath+"/")
		proc = strings.TrimSpace(proc)

		if proc == "" {
			h.writeError(w, http.StatusBadRequest, "missing procedure")
			return
		}

		method := h.resolve(proc)
		if method == nil {
			h.writeError(w, http.StatusBadRequest, "unknown procedure")
			return
		}

		h.handleCall(w, r.Context(), method, r)
	}
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

func (h *Handler) handleCall(w http.ResponseWriter, ctx context.Context, m *contract.Method, r *http.Request) {
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		if !isEOFError(err) {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	result, err := h.invoke(ctx, m, raw)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ErrorEnvelope(CodeInternalError, err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(SuccessEnvelope(result))
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

func (h *Handler) writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorEnvelope(CodeBadRequest, msg))
}

func (h *Handler) buildMeta() ServiceMeta {
	methods := make([]ProcedureMeta, 0, len(h.service.Methods))
	for _, m := range h.service.Methods {
		pm := ProcedureMeta{
			Name:     m.Name,
			FullName: m.FullName,
			Proc:     h.service.Name + "." + m.Name,
		}
		if m.Input != nil {
			pm.Input = &TypeRef{ID: m.Input.ID, Name: m.Input.Name}
		}
		if m.Output != nil {
			pm.Output = &TypeRef{ID: m.Output.ID, Name: m.Output.Name}
		}
		methods = append(methods, pm)
	}

	schemas := make([]Schema, 0)
	for _, t := range h.service.Types.All() {
		if s := h.service.Types.Schema(t.ID); s != nil {
			schemas = append(schemas, Schema{
				ID:   t.ID,
				JSON: s,
			})
		}
	}

	return ServiceMeta{
		Service: h.service.Name,
		Methods: methods,
		Schemas: schemas,
	}
}

// Mount registers tRPC handlers at the given base path.
func Mount(mux *http.ServeMux, base string, svc *contract.Service, opts ...Option) {
	if mux == nil || svc == nil {
		return
	}
	if base == "" {
		base = "/trpc"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	base = strings.TrimRight(base, "/")

	h := NewHandler(base, svc, opts...)
	mux.HandleFunc(base+".meta", h.MetaHandler())
	mux.HandleFunc(base+"/", h.CallHandler())
}

func isEOFError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "EOF")
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
