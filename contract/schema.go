package contract

import (
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

// ContractEnum is an interface that types can implement to declare enum values.
type ContractEnum interface {
	ContractEnum() []any
}

// TypeRegistry manages type references and their JSON schemas.
type TypeRegistry struct {
	mu      sync.RWMutex
	types   map[string]*TypeRef
	schemas map[string]Schema
}

// TypeRef is a reference to a registered type.
type TypeRef struct {
	ID   string
	Name string
}

// Schema holds a JSON schema for a type.
type Schema struct {
	ID   string         `json:"id"`
	JSON map[string]any `json:"json"`
}

func newTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		types:   make(map[string]*TypeRef),
		schemas: make(map[string]Schema),
	}
}

// Add registers a type and returns its reference.
func (r *TypeRegistry) Add(t reflect.Type) (*TypeRef, error) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// Anonymous types use kind-based naming
	id := typeID(t)

	r.mu.Lock()
	defer r.mu.Unlock()

	if tr, ok := r.types[id]; ok {
		return tr, nil
	}

	tr := &TypeRef{ID: id, Name: typeName(t)}
	r.types[id] = tr

	s, err := schemaForType(t, r)
	if err != nil {
		delete(r.types, id)
		return nil, err
	}
	r.schemas[id] = Schema{ID: id, JSON: s}

	return tr, nil
}

// Get returns a type reference by ID.
func (r *TypeRegistry) Get(id string) *TypeRef {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.types[id]
}

// Schema returns the schema for a type ID.
func (r *TypeRegistry) Schema(id string) (Schema, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.schemas[id]
	return s, ok
}

// Schemas returns all registered schemas, sorted by ID.
func (r *TypeRegistry) Schemas() []Schema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Schema, 0, len(r.schemas))
	for _, s := range r.schemas {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Types returns all registered type references, sorted by ID.
func (r *TypeRegistry) Types() []*TypeRef {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*TypeRef, 0, len(r.types))
	for _, t := range r.types {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// typeID generates a unique identifier for a type.
func typeID(t reflect.Type) string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.PkgPath() == "" {
		// Built-in types or anonymous types
		return t.String()
	}

	parts := strings.Split(t.PkgPath(), "/")
	return parts[len(parts)-1] + "." + t.Name()
}

// typeName returns a human-readable name for a type.
func typeName(t reflect.Type) string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Name() != "" {
		return t.Name()
	}

	return t.String()
}

// schemaForType generates a JSON schema for a Go type.
func schemaForType(t reflect.Type, registry *TypeRegistry) (map[string]any, error) {
	return schemaForTypeInternal(t, registry, make(map[reflect.Type]bool))
}

func schemaForTypeInternal(t reflect.Type, registry *TypeRegistry, seen map[reflect.Type]bool) (map[string]any, error) {
	// Handle pointers by dereferencing
	isNullable := false
	for t.Kind() == reflect.Pointer {
		isNullable = true
		t = t.Elem()
	}

	// Check for enum interface
	if t.Kind() != reflect.Interface {
		ptr := reflect.New(t)
		if enum, ok := ptr.Interface().(ContractEnum); ok {
			vals := enum.ContractEnum()
			s := map[string]any{"enum": vals}
			if isNullable {
				s["nullable"] = true
			}
			return s, nil
		}
	}

	// Handle special types
	if t == typeTime {
		s := map[string]any{"type": "string", "format": "date-time"}
		if isNullable {
			s["nullable"] = true
		}
		return s, nil
	}

	switch t.Kind() {
	case reflect.String:
		s := map[string]any{"type": "string"}
		if isNullable {
			s["nullable"] = true
		}
		return s, nil

	case reflect.Bool:
		s := map[string]any{"type": "boolean"}
		if isNullable {
			s["nullable"] = true
		}
		return s, nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		s := map[string]any{"type": "integer"}
		if isNullable {
			s["nullable"] = true
		}
		return s, nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		s := map[string]any{"type": "integer", "minimum": float64(0)}
		if isNullable {
			s["nullable"] = true
		}
		return s, nil

	case reflect.Float32, reflect.Float64:
		s := map[string]any{"type": "number"}
		if isNullable {
			s["nullable"] = true
		}
		return s, nil

	case reflect.Slice, reflect.Array:
		itemSchema, err := schemaForTypeInternal(t.Elem(), registry, seen)
		if err != nil {
			return nil, err
		}
		s := map[string]any{
			"type":  "array",
			"items": itemSchema,
		}
		if isNullable {
			s["nullable"] = true
		}
		return s, nil

	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			// Only string keys are supported in JSON
			return map[string]any{"type": "object"}, nil
		}
		valueSchema, err := schemaForTypeInternal(t.Elem(), registry, seen)
		if err != nil {
			return nil, err
		}
		s := map[string]any{
			"type":                 "object",
			"additionalProperties": valueSchema,
		}
		if isNullable {
			s["nullable"] = true
		}
		return s, nil

	case reflect.Struct:
		// Prevent infinite recursion for circular types
		if seen[t] {
			return map[string]any{"$ref": "#/components/schemas/" + typeID(t)}, nil
		}
		seen[t] = true

		props := map[string]any{}
		var required []string

		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)

			// Skip unexported fields
			if f.PkgPath != "" {
				continue
			}

			// Handle embedded fields
			if f.Anonymous {
				embeddedSchema, err := schemaForTypeInternal(f.Type, registry, seen)
				if err != nil {
					return nil, err
				}
				// Merge embedded struct properties
				if embeddedProps, ok := embeddedSchema["properties"].(map[string]any); ok {
					for k, v := range embeddedProps {
						props[k] = v
					}
				}
				if embeddedReq, ok := embeddedSchema["required"].([]string); ok {
					required = append(required, embeddedReq...)
				}
				continue
			}

			// Get JSON field name
			name := jsonFieldName(f)
			if name == "-" {
				continue
			}

			fieldSchema, err := schemaForTypeInternal(f.Type, registry, seen)
			if err != nil {
				return nil, err
			}

			// Parse contract tags
			contractTag := f.Tag.Get("contract")
			if contractTag != "" {
				fieldSchema = applyContractTags(fieldSchema, contractTag)
			}

			props[name] = fieldSchema

			// Determine if required
			if isFieldRequired(f) {
				required = append(required, name)
			}
		}

		delete(seen, t)

		s := map[string]any{
			"type":       "object",
			"properties": props,
		}
		if len(required) > 0 {
			s["required"] = required
		}
		if isNullable {
			s["nullable"] = true
		}
		return s, nil

	case reflect.Interface:
		// interface{} or any
		return map[string]any{}, nil

	default:
		return nil, ErrUnsupportedType
	}
}

// jsonFieldName extracts the JSON field name from struct field tags.
func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		// Default: lowercase first letter
		return strings.ToLower(f.Name[:1]) + f.Name[1:]
	}

	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "" {
		return strings.ToLower(f.Name[:1]) + f.Name[1:]
	}
	return name
}

// isFieldRequired determines if a field is required.
func isFieldRequired(f reflect.StructField) bool {
	// Check contract tag
	contractTag := f.Tag.Get("contract")
	if strings.Contains(contractTag, "required") {
		return true
	}

	// Check json tag for omitempty
	jsonTag := f.Tag.Get("json")
	if strings.Contains(jsonTag, "omitempty") {
		return false
	}

	// Pointers are optional by default
	if f.Type.Kind() == reflect.Pointer {
		return false
	}

	// Slices and maps are optional by default
	if f.Type.Kind() == reflect.Slice || f.Type.Kind() == reflect.Map {
		return false
	}

	return true
}

// applyContractTags applies contract struct tags to a schema.
func applyContractTags(schema map[string]any, tag string) map[string]any {
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Handle key=value pairs
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			switch key {
			case "minLength":
				if n := parseInt(value); n > 0 {
					schema["minLength"] = n
				}
			case "maxLength":
				if n := parseInt(value); n > 0 {
					schema["maxLength"] = n
				}
			case "minimum":
				if n := parseFloat(value); n != 0 {
					schema["minimum"] = n
				}
			case "maximum":
				if n := parseFloat(value); n != 0 {
					schema["maximum"] = n
				}
			case "minItems":
				if n := parseInt(value); n > 0 {
					schema["minItems"] = n
				}
			case "maxItems":
				if n := parseInt(value); n > 0 {
					schema["maxItems"] = n
				}
			case "pattern":
				schema["pattern"] = value
			case "format":
				schema["format"] = value
			case "description":
				schema["description"] = value
			case "default":
				schema["default"] = value
			}
		} else {
			// Handle boolean flags
			switch part {
			case "required":
				// Handled separately
			case "nullable":
				schema["nullable"] = true
			case "uniqueItems":
				schema["uniqueItems"] = true
			}
		}
	}

	return schema
}

func parseInt(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

func parseFloat(s string) float64 {
	var n float64
	var dec float64 = 1
	inDecimal := false
	negative := false

	for i, c := range s {
		if i == 0 && c == '-' {
			negative = true
			continue
		}
		if c == '.' {
			inDecimal = true
			continue
		}
		if c >= '0' && c <= '9' {
			if inDecimal {
				dec *= 10
				n += float64(c-'0') / dec
			} else {
				n = n*10 + float64(c-'0')
			}
		}
	}

	if negative {
		return -n
	}
	return n
}

// deref dereferences pointer types.
func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// Common time type reference
var typeTime = reflect.TypeOf(time.Time{})
