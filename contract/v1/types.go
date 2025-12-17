package contract

import (
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

// Enum is an interface that types can implement to declare enum values.
type Enum interface {
	ContractEnum() []any
}

// Types manages registered types and their schemas.
type Types struct {
	mu      sync.RWMutex
	types   map[string]*Type
	schemas map[string]map[string]any
}

// Type represents a registered type.
type Type struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func newTypes() *Types {
	return &Types{
		types:   make(map[string]*Type),
		schemas: make(map[string]map[string]any),
	}
}

// Add registers a type and returns its reference.
func (t *Types) Add(rt reflect.Type) (*Type, error) {
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}

	id := typeID(rt)

	t.mu.Lock()
	defer t.mu.Unlock()

	if typ, ok := t.types[id]; ok {
		return typ, nil
	}

	typ := &Type{ID: id, Name: typeName(rt)}
	t.types[id] = typ

	schema, err := buildSchema(rt, t)
	if err != nil {
		delete(t.types, id)
		return nil, err
	}
	t.schemas[id] = schema

	return typ, nil
}

// Get returns a type by ID.
func (t *Types) Get(id string) *Type {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.types[id]
}

// Schema returns the JSON schema for a type ID.
func (t *Types) Schema(id string) map[string]any {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.schemas[id]
}

// All returns all registered types, sorted by ID.
func (t *Types) All() []*Type {
	t.mu.RLock()
	defer t.mu.RUnlock()

	out := make([]*Type, 0, len(t.types))
	for _, typ := range t.types {
		out = append(out, typ)
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

var typeTime = reflect.TypeOf(time.Time{})

// buildSchema generates a JSON schema for a Go type.
func buildSchema(t reflect.Type, types *Types) (map[string]any, error) {
	return buildSchemaInternal(t, types, make(map[reflect.Type]bool))
}

func buildSchemaInternal(t reflect.Type, types *Types, seen map[reflect.Type]bool) (map[string]any, error) {
	isNullable := false
	for t.Kind() == reflect.Pointer {
		isNullable = true
		t = t.Elem()
	}

	// Check for enum interface
	if t.Kind() != reflect.Interface {
		ptr := reflect.New(t)
		if enum, ok := ptr.Interface().(Enum); ok {
			vals := enum.ContractEnum()
			s := map[string]any{"enum": vals}
			if isNullable {
				s["nullable"] = true
			}
			return s, nil
		}
	}

	// Handle time.Time
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
		itemSchema, err := buildSchemaInternal(t.Elem(), types, seen)
		if err != nil {
			return nil, err
		}
		s := map[string]any{"type": "array", "items": itemSchema}
		if isNullable {
			s["nullable"] = true
		}
		return s, nil

	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return map[string]any{"type": "object"}, nil
		}
		valueSchema, err := buildSchemaInternal(t.Elem(), types, seen)
		if err != nil {
			return nil, err
		}
		s := map[string]any{"type": "object", "additionalProperties": valueSchema}
		if isNullable {
			s["nullable"] = true
		}
		return s, nil

	case reflect.Struct:
		if seen[t] {
			return map[string]any{"$ref": "#/components/schemas/" + typeID(t)}, nil
		}
		seen[t] = true

		props := map[string]any{}
		var required []string

		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue
			}

			if f.Anonymous {
				embeddedSchema, err := buildSchemaInternal(f.Type, types, seen)
				if err != nil {
					return nil, err
				}
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

			name := jsonFieldName(f)
			if name == "-" {
				continue
			}

			fieldSchema, err := buildSchemaInternal(f.Type, types, seen)
			if err != nil {
				return nil, err
			}

			if tag := f.Tag.Get("contract"); tag != "" {
				fieldSchema = applyTags(fieldSchema, tag)
			}

			props[name] = fieldSchema

			if isRequired(f) {
				required = append(required, name)
			}
		}

		delete(seen, t)

		s := map[string]any{"type": "object", "properties": props}
		if len(required) > 0 {
			s["required"] = required
		}
		if isNullable {
			s["nullable"] = true
		}
		return s, nil

	case reflect.Interface:
		return map[string]any{}, nil

	default:
		return nil, ErrUnsupportedType
	}
}

func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return strings.ToLower(f.Name[:1]) + f.Name[1:]
	}
	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "" {
		return strings.ToLower(f.Name[:1]) + f.Name[1:]
	}
	return name
}

func isRequired(f reflect.StructField) bool {
	if strings.Contains(f.Tag.Get("contract"), "required") {
		return true
	}
	if strings.Contains(f.Tag.Get("json"), "omitempty") {
		return false
	}
	k := f.Type.Kind()
	if k == reflect.Pointer || k == reflect.Slice || k == reflect.Map {
		return false
	}
	return true
}

func applyTags(schema map[string]any, tag string) map[string]any {
	for _, part := range strings.Split(tag, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			key, value := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
			switch key {
			case "minLength", "maxLength", "minItems", "maxItems":
				if n := parseInt(value); n > 0 {
					schema[key] = n
				}
			case "minimum", "maximum":
				if n := parseFloat(value); n != 0 {
					schema[key] = n
				}
			case "pattern", "format", "description", "default":
				schema[key] = value
			}
		} else if part == "nullable" {
			schema["nullable"] = true
		} else if part == "uniqueItems" {
			schema["uniqueItems"] = true
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
