package mcp

import (
	"strings"

	contract "github.com/go-mizu/mizu/contract/v2"
)

// buildTools creates MCP tool definitions from a contract descriptor.
func buildTools(svc *contract.Service) []tool {
	var tools []tool
	for _, res := range svc.Resources {
		if res == nil {
			continue
		}
		for _, m := range res.Methods {
			if m == nil {
				continue
			}
			t := tool{
				Name:        res.Name + "_" + m.Name,
				Description: m.Description,
				InputSchema: buildInputSchema(m, svc),
			}
			tools = append(tools, t)
		}
	}
	return tools
}

// buildInputSchema creates a JSON Schema from a method's input type.
func buildInputSchema(m *contract.Method, svc *contract.Service) jsonSchema {
	schema := jsonSchema{Type: "object"}
	if m.Input == "" {
		return schema
	}

	// Find input type
	inputType := findType(svc, string(m.Input))
	if inputType == nil || inputType.Kind != contract.KindStruct {
		return schema
	}

	schema.Properties = make(map[string]jsonSchema)
	for _, f := range inputType.Fields {
		schema.Properties[f.Name] = typeRefToSchema(f.Type, svc)
		if !f.Optional {
			schema.Required = append(schema.Required, f.Name)
		}
	}
	return schema
}

// typeRefToSchema converts a contract TypeRef to a JSON Schema.
func typeRefToSchema(ref contract.TypeRef, svc *contract.Service) jsonSchema {
	s := string(ref)
	switch s {
	case "string":
		return jsonSchema{Type: "string"}
	case "int":
		return jsonSchema{Type: "integer"}
	case "float":
		return jsonSchema{Type: "number"}
	case "bool":
		return jsonSchema{Type: "boolean"}
	case "any":
		return jsonSchema{Type: "object"}
	}
	if strings.HasPrefix(s, "[]") {
		elem := typeRefToSchema(contract.TypeRef(s[2:]), svc)
		return jsonSchema{
			Type:  "array",
			Items: &elem,
		}
	}
	// Nested struct
	t := findType(svc, s)
	if t != nil && t.Kind == contract.KindStruct {
		schema := jsonSchema{Type: "object", Properties: make(map[string]jsonSchema)}
		for _, f := range t.Fields {
			schema.Properties[f.Name] = typeRefToSchema(f.Type, svc)
			if !f.Optional {
				schema.Required = append(schema.Required, f.Name)
			}
		}
		return schema
	}
	return jsonSchema{Type: "object"}
}

// findType finds a type by name in the service descriptor.
func findType(svc *contract.Service, name string) *contract.Type {
	for _, t := range svc.Types {
		if t != nil && t.Name == name {
			return t
		}
	}
	return nil
}

// parseTool parses a tool name into resource and method.
// Format: resource_method
func parseTool(name string) (resource, method string, ok bool) {
	i := strings.IndexByte(name, '_')
	if i <= 0 || i == len(name)-1 {
		return "", "", false
	}
	return name[:i], name[i+1:], true
}
