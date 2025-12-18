package async

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	contract "github.com/go-mizu/mizu/contract/v2"
)

// RequestTopic returns the canonical request topic name for AsyncAPI spec.
func RequestTopic(service, resource, method string) string {
	return sanitizeTopic(service) + "." + sanitizeTopic(resource) + "." + sanitizeTopic(method) + ".request"
}

// ResponseTopic returns the canonical response topic name for AsyncAPI spec.
func ResponseTopic(service, resource, method string) string {
	return sanitizeTopic(service) + "." + sanitizeTopic(resource) + "." + sanitizeTopic(method) + ".response"
}

func sanitizeTopic(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, ":", "_")
	return s
}

// AsyncAPIDocument builds an AsyncAPI document (JSON) for the async transport.
//
// v1 mapping:
//
// - For each method: two channels
//     <svc>.<resource>.<method>.request
//     <svc>.<resource>.<method>.response
//
// - Request message schema:
//     Envelope with params pointing to the method input schema
//
// - Response message schema:
//     Envelope with result pointing to the method output schema and optional error
func AsyncAPIDocument(svc *contract.Service) ([]byte, error) {
	doc, err := buildAsyncAPI(svc)
	if err != nil {
		return nil, err
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return b, nil
}

func buildAsyncAPI(svc *contract.Service) (map[string]any, error) {
	if svc == nil {
		return nil, fmt.Errorf("asyncapi: nil service")
	}

	declared := make(map[string]*contract.Type)
	for _, t := range svc.Types {
		if t == nil || strings.TrimSpace(t.Name) == "" {
			continue
		}
		declared[t.Name] = t
	}

	componentsSchemas := make(map[string]any)

	// Add declared type schemas
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
		componentsSchemas[n] = sc
	}

	// Add Envelope and Error schemas
	componentsSchemas["AsyncEnvelope"] = envelopeSchema()
	componentsSchemas["AsyncError"] = errorSchema()

	channels := make(map[string]any)

	for _, res := range svc.Resources {
		if res == nil {
			continue
		}
		for _, m := range res.Methods {
			if m == nil {
				continue
			}
			reqTopic := RequestTopic(svc.Name, res.Name, m.Name)
			respTopic := ResponseTopic(svc.Name, res.Name, m.Name)

			reqMsg := messageForRequest(res.Name, m.Name, m.Input, declared)
			respMsg := messageForResponse(res.Name, m.Name, m.Output, declared)

			channels[reqTopic] = map[string]any{
				"publish": map[string]any{
					"message": reqMsg,
				},
			}
			channels[respTopic] = map[string]any{
				"subscribe": map[string]any{
					"message": respMsg,
				},
			}
		}
	}

	doc := map[string]any{
		"asyncapi": "2.6.0",
		"info": map[string]any{
			"title":       svc.Name,
			"description": svc.Description,
			"version":     "0.1.0",
		},
		"channels": channels,
		"components": map[string]any{
			"schemas": componentsSchemas,
		},
	}

	return doc, nil
}

func messageForRequest(resource, method string, input contract.TypeRef, declared map[string]*contract.Type) map[string]any {
	props := map[string]any{
		"id":       map[string]any{"type": "string"},
		"reply_to": map[string]any{"type": "string"},
	}
	required := []string{"id"}

	if strings.TrimSpace(string(input)) != "" {
		props["params"] = schemaForTypeRef(input, declared)
		required = append(required, "params")
	} else {
		props["params"] = map[string]any{"type": "null"}
	}
	// reply_to is optional (notification)
	s := map[string]any{
		"type":       "object",
		"properties": props,
		"required":   required,
	}

	return map[string]any{
		"name": resource + "_" + method + "_request",
		"payload": s,
	}
}

func messageForResponse(resource, method string, output contract.TypeRef, declared map[string]*contract.Type) map[string]any {
	props := map[string]any{
		"id": map[string]any{"type": "string"},
		"error": map[string]any{
			"$ref": "#/components/schemas/AsyncError",
		},
	}
	required := []string{"id"}

	if strings.TrimSpace(string(output)) != "" {
		props["result"] = schemaForTypeRef(output, declared)
	} else {
		props["result"] = map[string]any{"type": "null"}
	}

	s := map[string]any{
		"type":       "object",
		"properties": props,
		"required":   required,
	}

	return map[string]any{
		"name": resource + "_" + method + "_response",
		"payload": s,
	}
}

func envelopeSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":       map[string]any{"type": "string"},
			"params":   map[string]any{},
			"reply_to": map[string]any{"type": "string"},
			"result":   map[string]any{},
			"error":    map[string]any{"$ref": "#/components/schemas/AsyncError"},
		},
	}
}

func errorSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code":    map[string]any{"type": "string"},
			"message": map[string]any{"type": "string"},
		},
		"required": []string{"code", "message"},
	}
}

// JSON Schema generation for declared types and TypeRef.

// schemaForDeclared converts contract.Type (struct/slice/map) into JSON Schema.
func schemaForDeclared(t *contract.Type, declared map[string]*contract.Type) (map[string]any, error) {
	if t == nil {
		return nil, fmt.Errorf("asyncapi: nil type")
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
			return nil, fmt.Errorf("asyncapi: slice %s missing elem", t.Name)
		}
		return map[string]any{
			"type":        "array",
			"items":       schemaForTypeRef(t.Elem, declared),
			"description": t.Description,
		}, nil

	case contract.KindMap:
		if t.Elem == "" {
			return nil, fmt.Errorf("asyncapi: map %s missing elem", t.Name)
		}
		return map[string]any{
			"type":                 "object",
			"additionalProperties": schemaForTypeRef(t.Elem, declared),
			"description":          t.Description,
		}, nil
	}
	return nil, fmt.Errorf("asyncapi: unsupported kind %q for type %s", t.Kind, t.Name)
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
