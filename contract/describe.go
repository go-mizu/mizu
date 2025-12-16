package contract

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Description is the machine-readable service description.
type Description struct {
	Services []ServiceDesc `json:"services"`
	Types    []TypeDesc    `json:"types"`
	Errors   []ErrorDesc   `json:"errors"`
}

// ServiceDesc describes a service.
type ServiceDesc struct {
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Version     string       `json:"version,omitempty"`
	Methods     []MethodDesc `json:"methods"`
}

// MethodDesc describes a method.
type MethodDesc struct {
	Name        string    `json:"name"`
	FullName    string    `json:"fullName"`
	Description string    `json:"description,omitempty"`
	Deprecated  bool      `json:"deprecated,omitempty"`
	Input       *TypeRef  `json:"input,omitempty"`
	Output      *TypeRef  `json:"output,omitempty"`
	HTTP        *HTTPDesc `json:"http,omitempty"`
	RPC         string    `json:"rpc,omitempty"`
}

// HTTPDesc provides REST transport hints.
type HTTPDesc struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// TypeRef references a type.
type TypeRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TypeDesc describes a type.
type TypeDesc struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
}

// ErrorDesc describes an error code.
type ErrorDesc struct {
	Code       string `json:"code"`
	HTTPStatus int    `json:"httpStatus"`
}

// Describe creates a Description from services.
func Describe(services ...*Service) *Description {
	desc := &Description{
		Services: make([]ServiceDesc, 0, len(services)),
		Types:    make([]TypeDesc, 0),
		Errors:   standardErrors(),
	}

	typeSet := make(map[string]bool)

	for _, svc := range services {
		sd := ServiceDesc{
			Name:        svc.Name,
			Description: svc.Description,
			Version:     svc.Version,
			Methods:     make([]MethodDesc, 0, len(svc.Methods)),
		}

		basePath := "/" + pluralize(svc.Name)

		for _, m := range svc.Methods {
			md := MethodDesc{
				Name:        m.Name,
				FullName:    m.FullName,
				Description: m.Description,
				Deprecated:  m.Deprecated,
				RPC:         m.FullName,
			}

			if m.Input != nil {
				md.Input = &TypeRef{ID: m.Input.ID, Name: m.Input.Name}
				typeSet[m.Input.ID] = true
			}

			if m.Output != nil {
				md.Output = &TypeRef{ID: m.Output.ID, Name: m.Output.Name}
				typeSet[m.Output.ID] = true
			}

			// Compute HTTP hints
			httpMethod := m.HTTPMethod
			if httpMethod == "" {
				httpMethod = inferHTTPMethod(m.Name)
			}
			httpPath := m.HTTPPath
			if httpPath == "" {
				httpPath = basePath
				if m.Input != nil && strings.Contains(strings.ToLower(m.Input.Name), "id") {
					httpPath = basePath + "/{id}"
				}
			}
			md.HTTP = &HTTPDesc{Method: httpMethod, Path: httpPath}

			sd.Methods = append(sd.Methods, md)
		}

		desc.Services = append(desc.Services, sd)

		// Collect types
		for _, t := range svc.Types.All() {
			if typeSet[t.ID] {
				desc.Types = append(desc.Types, TypeDesc{
					ID:     t.ID,
					Name:   t.Name,
					Schema: svc.Types.Schema(t.ID),
				})
				delete(typeSet, t.ID)
			}
		}
	}

	return desc
}

// ServeDescription mounts a description endpoint.
func ServeDescription(mux *http.ServeMux, path string, services ...*Service) {
	if path == "" {
		path = "/_describe"
	}

	desc := Describe(services...)

	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(desc)
	})
}

func standardErrors() []ErrorDesc {
	return []ErrorDesc{
		{Code: string(InvalidArgument), HTTPStatus: 400},
		{Code: string(NotFound), HTTPStatus: 404},
		{Code: string(AlreadyExists), HTTPStatus: 409},
		{Code: string(PermissionDenied), HTTPStatus: 403},
		{Code: string(Unauthenticated), HTTPStatus: 401},
		{Code: string(ResourceExhausted), HTTPStatus: 429},
		{Code: string(FailedPrecondition), HTTPStatus: 412},
		{Code: string(Unimplemented), HTTPStatus: 501},
		{Code: string(Internal), HTTPStatus: 500},
		{Code: string(Unavailable), HTTPStatus: 503},
	}
}

func pluralize(s string) string {
	if strings.HasSuffix(s, "s") {
		return s
	}
	return s + "s"
}

func inferHTTPMethod(name string) string {
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
