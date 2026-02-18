// File: lib/openapi/handle_docs.go
package openapi

import (
	"embed"
	"html/template"
	"net/http"
	"strings"
	"sync"
)

//go:embed docs/*.html
var docsFS embed.FS

// Config for the docs handler.
type Config struct {
	// SpecURL is the URL of your OpenAPI spec, eg "/openapi.json".
	SpecURL string

	// DefaultUI is used when ?ui= is not provided.
	// Examples: "scalar", "redoc", "rapidoc", "stoplight", "swagger"
	DefaultUI string
}

// minimal template data
type docsPageData struct {
	SpecURL string
}

// Handler implements http.Handler and renders OpenAPI docs.
// Single route: GET /docs
// UI is selected by ?ui=scalar|redoc|rapidoc|stoplight|swagger
type Handler struct {
	cfg     Config
	once    sync.Once
	tmpl    *template.Template
	initErr error
}

func NewHandler(cfg Config) (*Handler, error) {
	h := &Handler{
		cfg: cfg,
	}
	if h.cfg.DefaultUI == "" {
		h.cfg.DefaultUI = "scalar"
	}
	if err := h.init(); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *Handler) init() error {
	h.once.Do(func() {
		t, err := template.New("openapi").ParseFS(docsFS, "docs/*.html")
		if err != nil {
			h.initErr = err
			return
		}
		h.tmpl = t
	})
	return h.initErr
}

// ServeHTTP implements http.Handler
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h.init(); err != nil {
		http.Error(w, "failed to load templates: "+err.Error(), http.StatusInternalServerError)
		return
	}

	ui := h.chooseUI(r)
	tmplName := h.templateName(ui)
	if tmplName == "" {
		http.Error(w, "docs UI not available", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := docsPageData{SpecURL: h.cfg.SpecURL}

	if err := h.tmpl.ExecuteTemplate(w, tmplName, data); err != nil {
		http.Error(w, "failed to render: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) chooseUI(r *http.Request) string {
	ui := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("ui")))
	if ui == "" {
		return h.cfg.DefaultUI
	}
	if h.templateName(ui) != "" {
		return ui
	}
	return h.cfg.DefaultUI
}

func (h *Handler) templateName(ui string) string {
	switch ui {
	case "swagger":
		return "swagger.html"
	case "redoc":
		return "redoc.html"
	case "scalar":
		return "scalar.html"
	case "stoplight":
		return "stoplight.html"
	case "rapidoc":
		return "rapidoc.html"
	default:
		return ""
	}
}
