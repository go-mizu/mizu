package trpc

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu/contract"
)

// Handler handles tRPC requests over HTTP.
type Handler struct {
	service  *contract.Service
	resolver contract.Resolver
	invoker  contract.TransportInvoker
	basePath string
}

// Option configures the handler.
type Option func(*Handler)

// WithResolver sets a custom method resolver.
func WithResolver(r contract.Resolver) Option {
	return func(h *Handler) { h.resolver = r }
}

// WithInvoker sets a custom method invoker.
func WithInvoker(i contract.TransportInvoker) Option {
	return func(h *Handler) { h.invoker = i }
}

// NewHandler creates a new tRPC handler.
func NewHandler(basePath string, svc *contract.Service, opts ...Option) *Handler {
	h := &Handler{
		service:  svc,
		basePath: basePath,
		resolver: &contract.ServiceResolver{Service: svc},
		invoker:  &contract.DefaultInvoker{},
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

		method := h.resolver.Resolve(proc)
		if method == nil {
			h.writeError(w, http.StatusBadRequest, "unknown procedure")
			return
		}

		h.handleCall(w, r.Context(), method, r)
	}
}

func (h *Handler) handleCall(w http.ResponseWriter, ctx context.Context, m *contract.Method, r *http.Request) {
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		// EOF is acceptable (empty body)
		if !isEOFError(err) {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	result, err := h.invoker.Invoke(ctx, m, raw)
	if err != nil {
		// Application errors return 200 with error envelope
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ErrorEnvelope(CodeInternalError, err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(SuccessEnvelope(result))
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
	for _, s := range h.service.Types.Schemas() {
		schemas = append(schemas, Schema{
			ID:   s.ID,
			JSON: s.JSON,
		})
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
