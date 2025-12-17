package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-mizu/mizu/contract"
)

// Handler serves REST endpoints for a contract service.
type Handler struct {
	service  *contract.Service
	routes   map[string]map[string]*route // path -> method -> route
	basePath string
	patterns []*pathPattern // compiled path patterns for matching
}

type route struct {
	method   *contract.Method
	pathVars []string
}

type pathPattern struct {
	path    string
	pattern *regexp.Regexp
	vars    []string
	methods map[string]*route
}

// Option configures the handler.
type Option func(*Handler)

// WithBasePath sets a custom base path prefix.
func WithBasePath(path string) Option {
	return func(h *Handler) { h.basePath = path }
}

// NewHandler creates a REST handler for the service.
func NewHandler(svc *contract.Service, opts ...Option) *Handler {
	h := &Handler{
		service:  svc,
		routes:   make(map[string]map[string]*route),
		basePath: "/" + pluralize(svc.Name),
	}
	for _, opt := range opts {
		opt(h)
	}
	h.buildRoutes()
	return h
}

func (h *Handler) buildRoutes() {
	for _, m := range h.service.Methods {
		httpMethod := m.HTTPMethod
		if httpMethod == "" {
			httpMethod = inferHTTPMethod(m.Name)
		}

		path := m.HTTPPath
		if path == "" {
			path = h.inferPath(m)
		}

		pathVars := extractPathVars(path)

		if h.routes[path] == nil {
			h.routes[path] = make(map[string]*route)
		}
		h.routes[path][httpMethod] = &route{
			method:   m,
			pathVars: pathVars,
		}
	}

	// Compile path patterns for matching
	for path, methods := range h.routes {
		vars := extractPathVars(path)
		pattern := pathToRegexp(path)
		h.patterns = append(h.patterns, &pathPattern{
			path:    path,
			pattern: pattern,
			vars:    vars,
			methods: methods,
		})
	}
}

func (h *Handler) inferPath(m *contract.Method) string {
	switch {
	case strings.HasPrefix(m.Name, "Create"):
		return h.basePath
	case strings.HasPrefix(m.Name, "List"):
		return h.basePath
	case strings.HasPrefix(m.Name, "Get"), strings.HasPrefix(m.Name, "Update"),
		strings.HasPrefix(m.Name, "Delete"), strings.HasPrefix(m.Name, "Patch"):
		if needsID(m) {
			return h.basePath + "/{id}"
		}
		return h.basePath
	default:
		// Custom action: POST /resources/{action}
		action := strings.ToLower(m.Name)
		return h.basePath + "/" + action
	}
}

// Name returns the transport name.
func (h *Handler) Name() string { return "rest" }

// ServeHTTP handles HTTP requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Find matching route
	route, pathValues := h.matchRoute(r.URL.Path, r.Method)
	if route == nil {
		// Check if path exists but method is wrong
		for _, p := range h.patterns {
			if p.pattern.MatchString(r.URL.Path) {
				h.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
				return
			}
		}
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
		return
	}

	// Build input
	in, err := h.buildInput(r, route, pathValues)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error())
		return
	}

	// Invoke method
	out, err := route.method.Call(r.Context(), in)
	if err != nil {
		h.handleError(w, err)
		return
	}

	// Write response
	if out == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handler) matchRoute(path, method string) (*route, map[string]string) {
	for _, p := range h.patterns {
		matches := p.pattern.FindStringSubmatch(path)
		if matches == nil {
			continue
		}

		r, ok := p.methods[method]
		if !ok {
			continue
		}

		// Extract path values
		values := make(map[string]string)
		for i, name := range p.vars {
			if i+1 < len(matches) {
				values[name] = matches[i+1]
			}
		}

		return r, values
	}

	return nil, nil
}

func (h *Handler) buildInput(r *http.Request, route *route, pathValues map[string]string) (any, error) {
	if !route.method.HasInput() {
		return nil, nil
	}

	in := route.method.NewInput()

	// Apply path parameters
	if len(pathValues) > 0 {
		if err := applyPathParams(in, pathValues); err != nil {
			return nil, err
		}
	}

	// Apply query parameters for GET/DELETE
	if r.Method == http.MethodGet || r.Method == http.MethodDelete {
		if err := applyQueryParams(in, r.URL.Query()); err != nil {
			return nil, err
		}
	}

	// Apply request body for POST/PUT/PATCH
	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		if r.Body != nil && r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(in); err != nil {
				return nil, err
			}
		}
	}

	return in, nil
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	var ce *contract.Error
	if errors.As(err, &ce) {
		status := mapErrorToStatus(ce.Code)
		h.writeErrorResponse(w, status, string(ce.Code), ce.Message, ce.Details)
		return
	}

	h.writeError(w, http.StatusInternalServerError, "INTERNAL", err.Error())
}

func (h *Handler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeErrorResponse(w, status, code, message, nil)
}

func (h *Handler) writeErrorResponse(w http.ResponseWriter, status int, code, message string, details any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := map[string]any{
		"code":    code,
		"message": message,
	}
	if details != nil {
		resp["details"] = details
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// Mount registers the REST handler with explicit paths.
func Mount(mux *http.ServeMux, svc *contract.Service, opts ...Option) {
	h := NewHandler(svc, opts...)
	// Register a catch-all pattern for the base path
	mux.Handle(h.basePath+"/", h)
	mux.Handle(h.basePath, h)
}

// MountWithSpec registers both REST handler and OpenAPI spec.
func MountWithSpec(mux *http.ServeMux, specPath string, svc *contract.Service, opts ...Option) error {
	Mount(mux, svc, opts...)

	if specPath == "" {
		specPath = "/openapi.json"
	}

	sh, err := NewSpecHandler(svc)
	if err != nil {
		return err
	}
	mux.Handle(specPath, sh)
	return nil
}

// Helper functions

func inferHTTPMethod(name string) string {
	switch {
	case strings.HasPrefix(name, "Create"):
		return http.MethodPost
	case strings.HasPrefix(name, "Get"):
		return http.MethodGet
	case strings.HasPrefix(name, "List"):
		return http.MethodGet
	case strings.HasPrefix(name, "Update"):
		return http.MethodPut
	case strings.HasPrefix(name, "Delete"):
		return http.MethodDelete
	case strings.HasPrefix(name, "Patch"):
		return http.MethodPatch
	default:
		return http.MethodPost
	}
}

func extractPathVars(path string) []string {
	var vars []string
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(path, -1)
	for _, m := range matches {
		if len(m) > 1 {
			vars = append(vars, m[1])
		}
	}
	return vars
}

func pathToRegexp(path string) *regexp.Regexp {
	// Escape special regex characters except for our path variables
	escaped := regexp.QuoteMeta(path)
	// Replace escaped {var} patterns with capture groups
	re := regexp.MustCompile(`\\\{([^}]+)\\\}`)
	pattern := re.ReplaceAllString(escaped, `([^/]+)`)
	return regexp.MustCompile("^" + pattern + "$")
}

func applyPathParams(in any, values map[string]string) error {
	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldVal := v.Field(i)

		if !fieldVal.CanSet() {
			continue
		}

		// Check json tag first, then field name
		name := field.Name
		if tag := field.Tag.Get("json"); tag != "" {
			parts := strings.Split(tag, ",")
			if parts[0] != "" && parts[0] != "-" {
				name = parts[0]
			}
		}

		// Match by lowercase comparison
		for paramName, paramValue := range values {
			if strings.EqualFold(name, paramName) || strings.EqualFold(field.Name, paramName) {
				if fieldVal.Kind() == reflect.String {
					fieldVal.SetString(paramValue)
				}
			}
		}
	}

	return nil
}

func applyQueryParams(in any, query map[string][]string) error {
	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldVal := v.Field(i)

		if !fieldVal.CanSet() {
			continue
		}

		// Check json tag first, then field name
		name := field.Name
		if tag := field.Tag.Get("json"); tag != "" {
			parts := strings.Split(tag, ",")
			if parts[0] != "" && parts[0] != "-" {
				name = parts[0]
			}
		}

		values, ok := query[name]
		if !ok {
			// Try lowercase
			values, ok = query[strings.ToLower(name)]
		}
		if !ok {
			continue
		}

		if len(values) == 0 {
			continue
		}

		switch fieldVal.Kind() {
		case reflect.String:
			fieldVal.SetString(values[0])
		case reflect.Int, reflect.Int64, reflect.Int32:
			var intVal int64
			if err := json.Unmarshal([]byte(values[0]), &intVal); err == nil {
				fieldVal.SetInt(intVal)
			}
		case reflect.Bool:
			fieldVal.SetBool(values[0] == "true" || values[0] == "1")
		case reflect.Slice:
			if fieldVal.Type().Elem().Kind() == reflect.String {
				fieldVal.Set(reflect.ValueOf(values))
			}
		}
	}

	return nil
}

func mapErrorToStatus(code contract.Code) int {
	return contract.CodeToHTTP(code)
}

func pluralize(s string) string {
	if strings.HasSuffix(s, "s") {
		return s
	}
	return s + "s"
}

func needsID(m *contract.Method) bool {
	if m.Input == nil {
		return false
	}
	return strings.Contains(strings.ToLower(m.Input.Name), "id")
}

func restVerb(name string) string {
	switch {
	case strings.HasPrefix(name, "Create"):
		return http.MethodPost
	case strings.HasPrefix(name, "Get"), strings.HasPrefix(name, "List"):
		return http.MethodGet
	case strings.HasPrefix(name, "Update"):
		return http.MethodPut
	case strings.HasPrefix(name, "Delete"):
		return http.MethodDelete
	case strings.HasPrefix(name, "Patch"):
		return http.MethodPatch
	default:
		return http.MethodPost
	}
}
