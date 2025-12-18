// contract/transport/rest/openapi.go
package rest

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	contract "github.com/go-mizu/mizu/contract/v2"
)

// OpenAPIDocument builds an OpenAPI 3.0 document for the HTTP bindings in svc.
// It returns JSON bytes with stable ordering (as much as encoding/json allows).
func OpenAPIDocument(svc *contract.Service) ([]byte, error) {
	doc, err := buildOpenAPI(svc)
	if err != nil {
		return nil, err
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return b, nil
}

func buildOpenAPI(svc *contract.Service) (map[string]any, error) {
	if svc == nil {
		return nil, fmt.Errorf("openapi: nil service")
	}

	schemas := make(map[string]any)
	declared := make(map[string]*contract.Type)
	for _, t := range svc.Types {
		if t == nil || strings.TrimSpace(t.Name) == "" {
			continue
		}
		declared[t.Name] = t
	}

	// Build component schemas for declared types.
	typeNames := make([]string, 0, len(declared))
	for name := range declared {
		typeNames = append(typeNames, name)
	}
	sort.Strings(typeNames)

	for _, name := range typeNames {
		sc, err := schemaForDeclared(declared[name], declared)
		if err != nil {
			return nil, err
		}
		schemas[name] = sc
	}

	paths := make(map[string]any)
	for _, res := range svc.Resources {
		if res == nil {
			continue
		}
		for _, m := range res.Methods {
			if m == nil || m.HTTP == nil {
				continue
			}
			p := strings.TrimSpace(m.HTTP.Path)
			verb := strings.ToLower(strings.TrimSpace(m.HTTP.Method))
			if p == "" || verb == "" {
				continue
			}

			pi, _ := paths[p].(map[string]any)
			if pi == nil {
				pi = make(map[string]any)
				paths[p] = pi
			}

			op := make(map[string]any)
			op["operationId"] = res.Name + "_" + m.Name
			if m.Description != "" {
				op["description"] = m.Description
			}

			// Parameters (path params always, query params for GET)
			params := buildParameters(p, m, declared)
			if len(params) > 0 {
				op["parameters"] = params
			}

			// Request body for non-GET when input exists.
			if verb != "get" && m.Input != "" {
				op["requestBody"] = map[string]any{
					"required": true,
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": schemaForTypeRef(m.Input, declared),
						},
					},
				}
			}

			// Responses
			resp := make(map[string]any)
			if m.Output == "" {
				resp["204"] = map[string]any{"description": "No Content"}
			} else {
				resp["200"] = map[string]any{
					"description": "OK",
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": schemaForTypeRef(m.Output, declared),
						},
					},
				}
			}
			op["responses"] = resp

			// Security: bearer if hinted.
			if svc.Defaults != nil && strings.EqualFold(svc.Defaults.Auth, "bearer") {
				op["security"] = []any{map[string]any{"bearerAuth": []any{}}}
			}

			pi[verb] = op
		}
	}

	doc := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       svc.Name,
			"description": svc.Description,
			"version":     "0.1.0",
		},
		"paths": paths,
		"components": map[string]any{
			"schemas": schemas,
		},
	}

	if svc.Defaults != nil && svc.Defaults.BaseURL != "" {
		doc["servers"] = []any{
			map[string]any{"url": strings.TrimRight(svc.Defaults.BaseURL, "/")},
		}
	}
	if svc.Defaults != nil && strings.EqualFold(svc.Defaults.Auth, "bearer") {
		doc["components"].(map[string]any)["securitySchemes"] = map[string]any{
			"bearerAuth": map[string]any{
				"type":   "http",
				"scheme": "bearer",
			},
		}
	}

	return doc, nil
}

func buildParameters(path string, m *contract.Method, declared map[string]*contract.Type) []any {
	pathParams := extractPathParams(path)

	var inputFields []contract.Field
	if m.Input != "" {
		if dt := declared[string(m.Input)]; dt != nil && dt.Kind == contract.KindStruct {
			inputFields = dt.Fields
		}
	}

	params := make([]any, 0)

	// Path params: required.
	for _, pp := range pathParams {
		params = append(params, map[string]any{
			"name":     pp,
			"in":       "path",
			"required": true,
			"schema":   map[string]any{"type": "string"},
		})
	}

	// Query params for GET: based on input struct fields not used in path.
	if m.HTTP != nil && strings.EqualFold(m.HTTP.Method, "GET") && len(inputFields) > 0 {
		used := make(map[string]bool)
		for _, pp := range pathParams {
			used[pp] = true
		}
		for _, f := range inputFields {
			if f.Name == "" || used[f.Name] {
				continue
			}
			params = append(params, map[string]any{
				"name":     f.Name,
				"in":       "query",
				"required": !f.Optional,
				"schema":   schemaForTypeRef(f.Type, declared),
				"description": f.Description,
			})
		}
	}

	return params
}

func extractPathParams(path string) []string {
	var out []string
	for {
		i := strings.Index(path, "{")
		if i < 0 {
			break
		}
		j := strings.Index(path[i:], "}")
		if j < 0 {
			break
		}
		j = i + j
		name := strings.TrimSpace(path[i+1 : j])
		if name != "" {
			out = append(out, name)
		}
		path = path[j+1:]
	}
	return out
}

func schemaForDeclared(t *contract.Type, declared map[string]*contract.Type) (map[string]any, error) {
	if t == nil {
		return nil, fmt.Errorf("openapi: nil type")
	}
	switch t.Kind {
	case contract.KindStruct:
		props := make(map[string]any)
		required := make([]string, 0)
		for _, f := range t.Fields {
			if f.Name == "" {
				continue
			}
			fs := schemaForTypeRef(f.Type, declared)
			if f.Nullable {
				fs = anyOfNull(fs)
			}
			props[f.Name] = fs
			if !f.Optional {
				required = append(required, f.Name)
			}
		}
		s := map[string]any{
			"type":        "object",
			"properties":  props,
			"description": t.Description,
		}
		if len(required) > 0 {
			sort.Strings(required)
			s["required"] = required
		}
		return s, nil

	case contract.KindSlice:
		if t.Elem == "" {
			return nil, fmt.Errorf("openapi: slice %s missing elem", t.Name)
		}
		return map[string]any{
			"type":        "array",
			"items":       schemaForTypeRef(t.Elem, declared),
			"description": t.Description,
		}, nil

	case contract.KindMap:
		if t.Elem == "" {
			return nil, fmt.Errorf("openapi: map %s missing elem", t.Name)
		}
		return map[string]any{
			"type":                 "object",
			"additionalProperties": schemaForTypeRef(t.Elem, declared),
			"description":          t.Description,
		}, nil
	}

	return nil, fmt.Errorf("openapi: unsupported kind %q for type %s", t.Kind, t.Name)
}

func schemaForTypeRef(ref contract.TypeRef, declared map[string]*contract.Type) map[string]any {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return map[string]any{}
	}

	// Declared types become refs.
	if declared != nil {
		if _, ok := declared[r]; ok {
			return map[string]any{"$ref": "#/components/schemas/" + r}
		}
	}

	// Primitives and common externals.
	switch r {
	case "string":
		return map[string]any{"type": "string"}
	case "bool", "boolean":
		return map[string]any{"type": "boolean"}
	case "int", "int32":
		return map[string]any{"type": "integer", "format": "int32"}
	case "int64":
		return map[string]any{"type": "integer", "format": "int64"}
	case "float32":
		return map[string]any{"type": "number", "format": "float"}
	case "float64", "number":
		return map[string]any{"type": "number", "format": "double"}
	case "time.Time":
		return map[string]any{"type": "string", "format": "date-time"}
	}

	// Unknown externals fall back to string to keep the doc usable.
	// Generators can override this via custom mappings.
	return map[string]any{"type": "string"}
}

func anyOfNull(s map[string]any) map[string]any {
	return map[string]any{
		"anyOf": []any{
			s,
			map[string]any{"type": "null"},
		},
	}
}
