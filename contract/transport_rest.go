package contract

import (
	"encoding/json"
	"net/http"
	"strings"
)

// MountREST mounts a service contract as REST endpoints.
//
// Conventions:
//   - Method name determines HTTP verb:
//       Create*  -> POST
//       Get*     -> GET
//       List*    -> GET
//       Update*  -> PUT
//       Delete*  -> DELETE
//   - Resource path derived from service name (pluralized naively).
//   - ID path param inferred if input has field named "ID".
//
// Example:
//   svc.Name == "todo"
//   Method Create -> POST   /todos
//   Method Get    -> GET    /todos/{id}
func MountREST(mux *http.ServeMux, svc *Service) {
	base := "/" + pluralize(svc.Name)

	// Group methods by path
	pathHandlers := make(map[string]map[string]*Method)

	for _, m := range svc.Methods {
		verb := restVerb(m.Name)
		path := base

		if needsID(m) {
			path = base + "/{id}"
		}

		if pathHandlers[path] == nil {
			pathHandlers[path] = make(map[string]*Method)
		}
		pathHandlers[path][verb] = m
	}

	// Register one handler per path that dispatches by HTTP method
	for path, methods := range pathHandlers {
		methodsCopy := methods // capture for closure
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			m, ok := methodsCopy[r.Method]
			if !ok {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			restHandler(m)(w, r)
		})
	}
}

// ServeOpenAPI serves OpenAPI 3.1 JSON at the given path.
func ServeOpenAPI(mux *http.ServeMux, path string, svc *Service) {
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		doc := openAPIDoc(svc)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doc)
	})
}

// ---- REST handler ----

func restHandler(m *Method) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var in any
		if m.Input != nil {
			in = m.NewInput()
			_ = json.NewDecoder(r.Body).Decode(in)
		}

		out, err := m.Invoker.Call(ctx, in)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if out == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func methodHandler(method string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
	}
}

// ---- OpenAPI ----

func openAPIDoc(svc *Service) map[string]any {
	paths := map[string]any{}

	base := "/" + pluralize(svc.Name)

	for _, m := range svc.Methods {
		verb := strings.ToLower(restVerb(m.Name))
		path := base
		if needsID(m) {
			path = base + "/{id}"
		}

		op := map[string]any{
			"operationId": m.FullName,
			"responses": map[string]any{
				"200": map[string]any{
					"description": "OK",
				},
			},
		}

		if m.Output != nil {
			op["responses"].(map[string]any)["200"].(map[string]any)["content"] = map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{
						"$ref": "#/components/schemas/" + m.Output.ID,
					},
				},
			}
		}

		if m.Input != nil {
			op["requestBody"] = map[string]any{
				"required": true,
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": map[string]any{
							"$ref": "#/components/schemas/" + m.Input.ID,
						},
					},
				},
			}
		}

		if paths[path] == nil {
			paths[path] = map[string]any{}
		}
		paths[path].(map[string]any)[verb] = op
	}

	components := map[string]any{
		"schemas": map[string]any{},
	}

	for _, s := range svc.Types.Schemas() {
		components["schemas"].(map[string]any)[s.ID] = s.JSON
	}

	return map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":   svc.Name,
			"version": "0.1.0",
		},
		"paths":      paths,
		"components": components,
	}
}

// ---- helpers ----

func restVerb(name string) string {
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
	default:
		return http.MethodPost
	}
}

func needsID(m *Method) bool {
	if m.Input == nil {
		return false
	}
	// convention: field named ID implies /{id}
	return strings.Contains(strings.ToLower(m.Input.Name), "id")
}

func pluralize(s string) string {
	if strings.HasSuffix(s, "s") {
		return s
	}
	return s + "s"
}
