package contract

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
)

// ClientDescriptor contains all information needed to generate clients.
type ClientDescriptor struct {
	Package  string          `json:"package"`
	Services []ClientService `json:"services"`
	Types    []ClientType    `json:"types"`
	Errors   []ClientError   `json:"errors"`
}

// ClientService describes a service for client generation.
type ClientService struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Version     string         `json:"version,omitempty"`
	Methods     []ClientMethod `json:"methods"`
}

// ClientMethod describes a method for client generation.
type ClientMethod struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Input       *ClientTypeRef `json:"input,omitempty"`
	Output      *ClientTypeRef `json:"output,omitempty"`
	REST        *RESTHint      `json:"rest,omitempty"`
	RPC         *RPCHint       `json:"rpc,omitempty"`
}

// ClientTypeRef references a type.
type ClientTypeRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RESTHint provides REST transport hints.
type RESTHint struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// RPCHint provides RPC transport hints.
type RPCHint struct {
	Method string `json:"method"`
}

// ClientType describes a type for client generation.
type ClientType struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	GoType     string         `json:"goType"`
	TSType     string         `json:"tsType"`
	Fields     []ClientField  `json:"fields,omitempty"`
	EnumValues []any          `json:"enumValues,omitempty"`
	Schema     map[string]any `json:"schema"`
}

// ClientField describes a field in a type.
type ClientField struct {
	Name       string `json:"name"`
	JSONName   string `json:"jsonName"`
	Type       string `json:"type"`
	GoType     string `json:"goType"`
	TSType     string `json:"tsType"`
	Required   bool   `json:"required"`
	Nullable   bool   `json:"nullable"`
	IsArray    bool   `json:"isArray,omitempty"`
	IsMap      bool   `json:"isMap,omitempty"`
	ItemType   string `json:"itemType,omitempty"`
}

// ClientError describes an error code for clients.
type ClientError struct {
	Code        string `json:"code"`
	HTTPStatus  int    `json:"httpStatus"`
	Description string `json:"description"`
}

// GenerateClientDescriptor creates a client descriptor from services.
func GenerateClientDescriptor(packageName string, services ...*Service) *ClientDescriptor {
	desc := &ClientDescriptor{
		Package:  packageName,
		Services: make([]ClientService, 0, len(services)),
		Types:    make([]ClientType, 0),
		Errors:   standardErrors(),
	}

	typeSet := make(map[string]bool)

	for _, svc := range services {
		cs := ClientService{
			Name:        svc.Name,
			Description: svc.Description,
			Version:     svc.Version,
			Methods:     make([]ClientMethod, 0, len(svc.Methods)),
		}

		basePath := "/" + pluralize(svc.Name)

		for _, m := range svc.Methods {
			cm := ClientMethod{
				Name:        m.Name,
				Description: m.Description,
			}

			if m.Input != nil {
				cm.Input = &ClientTypeRef{
					ID:   m.Input.ID,
					Name: m.Input.Name,
				}
				typeSet[m.Input.ID] = true
			}

			if m.Output != nil {
				cm.Output = &ClientTypeRef{
					ID:   m.Output.ID,
					Name: m.Output.Name,
				}
				typeSet[m.Output.ID] = true
			}

			// REST hints
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
			cm.REST = &RESTHint{
				Method: httpMethod,
				Path:   httpPath,
			}

			// RPC hints
			cm.RPC = &RPCHint{
				Method: m.FullName,
			}

			cs.Methods = append(cs.Methods, cm)
		}

		desc.Services = append(desc.Services, cs)

		// Collect types
		for _, schema := range svc.Types.Schemas() {
			if typeSet[schema.ID] {
				ref := svc.Types.Get(schema.ID)
				ct := ClientType{
					ID:     schema.ID,
					Name:   ref.Name,
					GoType: goTypeName(ref.Name),
					TSType: tsTypeName(ref.Name),
					Schema: schema.JSON,
				}

				// Extract fields if it's an object
				if props, ok := schema.JSON["properties"].(map[string]any); ok {
					required := make(map[string]bool)
					if req, ok := schema.JSON["required"].([]string); ok {
						for _, r := range req {
							required[r] = true
						}
					}

					for fieldName, fieldSchema := range props {
						fs, _ := fieldSchema.(map[string]any)
						cf := ClientField{
							Name:     toPascalCase(fieldName),
							JSONName: fieldName,
							Required: required[fieldName],
						}

						// Determine type
						if nullable, ok := fs["nullable"].(bool); ok {
							cf.Nullable = nullable
						}
						if t, ok := fs["type"].(string); ok {
							cf.Type = t
							cf.GoType = jsonTypeToGo(t, fs)
							cf.TSType = jsonTypeToTS(t, fs)
							if t == "array" {
								cf.IsArray = true
								if items, ok := fs["items"].(map[string]any); ok {
									if itemType, ok := items["type"].(string); ok {
										cf.ItemType = itemType
									}
								}
							}
						}

						ct.Fields = append(ct.Fields, cf)
					}
				}

				// Check for enum
				if enum, ok := schema.JSON["enum"].([]any); ok {
					ct.EnumValues = enum
				}

				desc.Types = append(desc.Types, ct)
				delete(typeSet, schema.ID)
			}
		}
	}

	return desc
}

// ServeClientDescriptor serves the client descriptor as JSON.
func ServeClientDescriptor(mux *http.ServeMux, path string, packageName string, services ...*Service) {
	if path == "" {
		path = "/_client"
	}

	desc := GenerateClientDescriptor(packageName, services...)

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

// standardErrors returns the standard error codes for clients.
func standardErrors() []ClientError {
	return []ClientError{
		{Code: string(ErrCodeInvalidArgument), HTTPStatus: 400, Description: "Invalid argument provided"},
		{Code: string(ErrCodeNotFound), HTTPStatus: 404, Description: "Resource not found"},
		{Code: string(ErrCodeAlreadyExists), HTTPStatus: 409, Description: "Resource already exists"},
		{Code: string(ErrCodePermissionDenied), HTTPStatus: 403, Description: "Permission denied"},
		{Code: string(ErrCodeUnauthenticated), HTTPStatus: 401, Description: "Authentication required"},
		{Code: string(ErrCodeResourceExhausted), HTTPStatus: 429, Description: "Resource exhausted (rate limited)"},
		{Code: string(ErrCodeFailedPrecondition), HTTPStatus: 412, Description: "Precondition failed"},
		{Code: string(ErrCodeAborted), HTTPStatus: 409, Description: "Operation aborted"},
		{Code: string(ErrCodeUnimplemented), HTTPStatus: 501, Description: "Not implemented"},
		{Code: string(ErrCodeInternal), HTTPStatus: 500, Description: "Internal server error"},
		{Code: string(ErrCodeUnavailable), HTTPStatus: 503, Description: "Service unavailable"},
	}
}

func goTypeName(name string) string {
	return "*" + name
}

func tsTypeName(name string) string {
	return name
}

func toPascalCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func jsonTypeToGo(t string, schema map[string]any) string {
	switch t {
	case "string":
		if format, ok := schema["format"].(string); ok {
			if format == "date-time" {
				return "time.Time"
			}
		}
		return "string"
	case "integer":
		return "int64"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "array":
		if items, ok := schema["items"].(map[string]any); ok {
			if itemType, ok := items["type"].(string); ok {
				return "[]" + jsonTypeToGo(itemType, items)
			}
		}
		return "[]any"
	case "object":
		if addProps, ok := schema["additionalProperties"].(map[string]any); ok {
			if valType, ok := addProps["type"].(string); ok {
				return "map[string]" + jsonTypeToGo(valType, addProps)
			}
		}
		return "map[string]any"
	default:
		return "any"
	}
}

func jsonTypeToTS(t string, schema map[string]any) string {
	switch t {
	case "string":
		return "string"
	case "integer", "number":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		if items, ok := schema["items"].(map[string]any); ok {
			if itemType, ok := items["type"].(string); ok {
				return jsonTypeToTS(itemType, items) + "[]"
			}
		}
		return "any[]"
	case "object":
		if addProps, ok := schema["additionalProperties"].(map[string]any); ok {
			if valType, ok := addProps["type"].(string); ok {
				return "Record<string, " + jsonTypeToTS(valType, addProps) + ">"
			}
		}
		return "Record<string, any>"
	default:
		return "any"
	}
}

// GoTypeString returns the Go type string for a reflect.Type.
func GoTypeString(t reflect.Type) string {
	if t == nil {
		return ""
	}

	switch t.Kind() {
	case reflect.Pointer:
		return "*" + GoTypeString(t.Elem())
	case reflect.Slice:
		return "[]" + GoTypeString(t.Elem())
	case reflect.Array:
		return "[" + string(rune(t.Len())) + "]" + GoTypeString(t.Elem())
	case reflect.Map:
		return "map[" + GoTypeString(t.Key()) + "]" + GoTypeString(t.Elem())
	case reflect.Struct:
		if t.PkgPath() != "" {
			return t.Name()
		}
		return t.String()
	default:
		return t.String()
	}
}

// TSTypeString returns the TypeScript type string for a reflect.Type.
func TSTypeString(t reflect.Type) string {
	if t == nil {
		return "void"
	}

	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return TSTypeString(t.Elem()) + "[]"
	case reflect.Map:
		return "Record<string, " + TSTypeString(t.Elem()) + ">"
	case reflect.Struct:
		if t == typeTime {
			return "string" // ISO date string
		}
		return t.Name()
	case reflect.Interface:
		return "any"
	default:
		return "any"
	}
}
