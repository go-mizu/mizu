package contract

import (
	"encoding/json"
	"net/http"
)

// IntrospectionResponse is the machine-readable contract description.
type IntrospectionResponse struct {
	Services []ServiceDescriptor `json:"services"`
	Types    []TypeDescriptor    `json:"types"`
}

// ServiceDescriptor describes a service in the introspection response.
type ServiceDescriptor struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Version     string             `json:"version,omitempty"`
	Tags        []string           `json:"tags,omitempty"`
	Methods     []MethodDescriptor `json:"methods"`
}

// MethodDescriptor describes a method in the introspection response.
type MethodDescriptor struct {
	Name        string   `json:"name"`
	FullName    string   `json:"fullName"`
	Description string   `json:"description,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Deprecated  bool     `json:"deprecated,omitempty"`
	Input       *TypeRef `json:"input,omitempty"`
	Output      *TypeRef `json:"output,omitempty"`
	// Transport hints
	REST *RESTDescriptor `json:"rest,omitempty"`
	RPC  *RPCDescriptor  `json:"rpc,omitempty"`
}

// RESTDescriptor provides REST-specific information.
type RESTDescriptor struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// RPCDescriptor provides RPC-specific information.
type RPCDescriptor struct {
	Method string `json:"method"`
}

// TypeDescriptor describes a type in the introspection response.
type TypeDescriptor struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
}

// Introspect generates an introspection response for the given services.
func Introspect(services ...*Service) *IntrospectionResponse {
	resp := &IntrospectionResponse{
		Services: make([]ServiceDescriptor, 0, len(services)),
		Types:    make([]TypeDescriptor, 0),
	}

	typeSet := make(map[string]bool)

	for _, svc := range services {
		sd := ServiceDescriptor{
			Name:        svc.Name,
			Description: svc.Description,
			Version:     svc.Version,
			Tags:        svc.Tags,
			Methods:     make([]MethodDescriptor, 0, len(svc.Methods)),
		}

		basePath := "/" + pluralize(svc.Name)

		for _, m := range svc.Methods {
			md := MethodDescriptor{
				Name:        m.Name,
				FullName:    m.FullName,
				Description: m.Description,
				Summary:     m.Summary,
				Tags:        m.Tags,
				Deprecated:  m.Deprecated,
				Input:       m.Input,
				Output:      m.Output,
			}

			// Compute REST hints
			httpMethod := m.HTTPMethod
			if httpMethod == "" {
				httpMethod = restVerb(m.Name)
			}
			httpPath := m.HTTPPath
			if httpPath == "" {
				httpPath = basePath
				if needsID(m) {
					httpPath = basePath + "/{id}"
				}
			}
			md.REST = &RESTDescriptor{
				Method: httpMethod,
				Path:   httpPath,
			}

			// RPC hints
			md.RPC = &RPCDescriptor{
				Method: m.FullName,
			}

			sd.Methods = append(sd.Methods, md)

			// Collect types
			if m.Input != nil && !typeSet[m.Input.ID] {
				typeSet[m.Input.ID] = true
			}
			if m.Output != nil && !typeSet[m.Output.ID] {
				typeSet[m.Output.ID] = true
			}
		}

		resp.Services = append(resp.Services, sd)

		// Add types from this service
		for _, schema := range svc.Types.Schemas() {
			if typeSet[schema.ID] {
				resp.Types = append(resp.Types, TypeDescriptor{
					ID:     schema.ID,
					Name:   svc.Types.Get(schema.ID).Name,
					Schema: schema.JSON,
				})
			}
		}
	}

	return resp
}

// ServeIntrospect mounts an introspection endpoint at the given path.
func ServeIntrospect(mux *http.ServeMux, path string, services ...*Service) {
	if path == "" {
		path = "/_introspect"
	}

	resp := Introspect(services...)

	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(resp)
	})
}

// MarshalJSON for TypeRef to include both ID and Name.
func (t *TypeRef) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"id":   t.ID,
		"name": t.Name,
	})
}
