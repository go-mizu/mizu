// contract/transport/jsonrpc/openrpc.go
package jsonrpc

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	contract "github.com/go-mizu/mizu/contract/v2"
)

// OpenRPCDocument returns an OpenRPC document (JSON) for the JSON-RPC surface.
//
// It lists methods as "<resource>.<method>" and emits JSON Schema components for declared types.
func OpenRPCDocument(svc *contract.Service) ([]byte, error) {
	doc, err := buildOpenRPC(svc)
	if err != nil {
		return nil, err
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return b, nil
}

func buildOpenRPC(svc *contract.Service) (map[string]any, error) {
	if svc == nil {
		return nil, fmt.Errorf("openrpc: nil service")
	}

	declared := make(map[string]*contract.Type)
	for _, t := range svc.Types {
		if t == nil || strings.TrimSpace(t.Name) == "" {
			continue
		}
		declared[t.Name] = t
	}

	schemas := make(map[string]any)
	typeNames := make([]string, 0, len(declared))
	for n := range declared {
		typeNames = append(typeNames, n)
	}
	sort.Strings(typeNames)
	for _, n := range typeNames {
		sc, err := schemaForDeclared(declared[n], declared)
		if err != nil {
			return nil, err
		}
		schemas[n] = sc
	}

	var methods []any
	for _, res := range svc.Resources {
		if res == nil {
			continue
		}
		for _, m := range res.Methods {
			if m == nil {
				continue
			}
			full := res.Name + "." + m.Name

			var params []any
			if m.Input != "" {
				// Named params, single object argument.
				params = append(params, map[string]any{
					"name":        "params",
					"description": "Named parameters object",
					"required":    true,
					"schema":      schemaForTypeRef(m.Input, declared),
				})
			}

			resultSchema := map[string]any{"type": "null"}
			if m.Output != "" {
				resultSchema = schemaForTypeRef(m.Output, declared)
			}

			entry := map[string]any{
				"name":        full,
				"description": m.Description,
				"params":      params,
				"result": map[string]any{
					"name":   "result",
					"schema": resultSchema,
				},
				"errors": []any{
					map[string]any{"code": errParse, "message": "parse error"},
					map[string]any{"code": errInvalidRequest, "message": "invalid request"},
					map[string]any{"code": errMethodNotFound, "message": "method not found"},
					map[string]any{"code": errInvalidParams, "message": "invalid params"},
					map[string]any{"code": errInternal, "message": "internal error"},
					map[string]any{"code": errServer, "message": "server error"},
				},
			}
			methods = append(methods, entry)
		}
	}

	doc := map[string]any{
		"openrpc": "1.2.6",
		"info": map[string]any{
			"title":       svc.Name,
			"description": svc.Description,
			"version":     "0.1.0",
		},
		"methods": methods,
		"components": map[string]any{
			"schemas": schemas,
		},
	}

	return doc, nil
}

func schemaForDeclared(t *contract.Type, declared map[string]*contract.Type) (map[string]any, error) {
	if t == nil {
		return nil, fmt.Errorf("openrpc: nil type")
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
			if f.Description != "" {
				fs = cloneWithDescription(fs, f.Description)
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
			return nil, fmt.Errorf("openrpc: slice %s missing elem", t.Name)
		}
		return map[string]any{
			"type":        "array",
			"items":       schemaForTypeRef(t.Elem, declared),
			"description": t.Description,
		}, nil

	case contract.KindMap:
		if t.Elem == "" {
			return nil, fmt.Errorf("openrpc: map %s missing elem", t.Name)
		}
		return map[string]any{
			"type":                 "object",
			"additionalProperties": schemaForTypeRef(t.Elem, declared),
			"description":          t.Description,
		}, nil
	}
	return nil, fmt.Errorf("openrpc: unsupported kind %q for type %s", t.Kind, t.Name)
}

func schemaForTypeRef(ref contract.TypeRef, declared map[string]*contract.Type) map[string]any {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return map[string]any{}
	}

	if declared != nil {
		if _, ok := declared[r]; ok {
			return map[string]any{"$ref": "#/components/schemas/" + r}
		}
	}

	switch r {
	case "string":
		return map[string]any{"type": "string"}
	case "bool", "boolean":
		return map[string]any{"type": "boolean"}
	case "int", "int32":
		return map[string]any{"type": "integer"}
	case "int64":
		return map[string]any{"type": "integer"}
	case "float32", "float64", "number":
		return map[string]any{"type": "number"}
	case "time.Time":
		return map[string]any{"type": "string", "format": "date-time"}
	}

	// Unknown external types default to string in docs.
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

func cloneWithDescription(s map[string]any, desc string) map[string]any {
	out := make(map[string]any, len(s)+1)
	for k, v := range s {
		out[k] = v
	}
	if desc != "" {
		out["description"] = desc
	}
	return out
}
