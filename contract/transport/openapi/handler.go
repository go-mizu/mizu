package openapi

import (
	"net/http"
	"strings"

	"github.com/go-mizu/mizu/contract"
)

// Handler serves OpenAPI documents over HTTP.
type Handler struct {
	document *Document
	json     []byte
}

// NewHandler creates an OpenAPI handler for the given services.
func NewHandler(services ...*contract.Service) (*Handler, error) {
	doc := Generate(services...)
	data, err := doc.MarshalIndent()
	if err != nil {
		return nil, err
	}
	return &Handler{
		document: doc,
		json:     data,
	}, nil
}

// Name returns the transport name.
func (h *Handler) Name() string {
	return "openapi"
}

// ServeHTTP serves the OpenAPI document.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(h.json)
}

// Document returns the underlying OpenAPI document.
func (h *Handler) Document() *Document {
	return h.document
}

// JSON returns the cached JSON bytes.
func (h *Handler) JSON() []byte {
	return h.json
}

// Mount registers an OpenAPI handler at the given path.
func Mount(mux *http.ServeMux, path string, services ...*contract.Service) error {
	if mux == nil {
		return nil
	}
	if path == "" {
		path = "/openapi.json"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	h, err := NewHandler(services...)
	if err != nil {
		return err
	}

	mux.Handle(path, h)
	return nil
}

// MountWithDocs registers both OpenAPI spec and documentation UI handlers.
// The spec is served at specPath and docs at docsPath.
func MountWithDocs(mux *http.ServeMux, specPath, docsPath string, services ...*contract.Service) error {
	if mux == nil {
		return nil
	}
	if specPath == "" {
		specPath = "/openapi.json"
	}
	if docsPath == "" {
		docsPath = "/docs"
	}

	h, err := NewHandler(services...)
	if err != nil {
		return err
	}

	mux.Handle(specPath, h)
	mux.Handle(docsPath, NewDocsHandler(specPath))
	return nil
}

// DocsHandler serves a simple documentation UI redirect.
type DocsHandler struct {
	specURL string
}

// NewDocsHandler creates a documentation handler.
func NewDocsHandler(specURL string) *DocsHandler {
	return &DocsHandler{specURL: specURL}
}

// ServeHTTP serves documentation.
func (h *DocsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Simple Swagger UI redirect (can be enhanced with embedded UI)
	html := `<!DOCTYPE html>
<html>
<head>
    <title>API Documentation</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: "` + h.specURL + `",
                dom_id: '#swagger-ui',
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIBundle.SwaggerUIStandalonePreset
                ],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}
