// Package sdkc generates typed C SDK clients from contract.Service.
package sdkc

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"unicode"

	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/sdk"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// Config controls C SDK generation.
type Config struct {
	// Package is the C package/library name (used as prefix).
	// Default: sanitized snake_case service name.
	Package string

	// Version is the library version.
	// Default: "0.0.0".
	Version string

	// HeaderGuardPrefix is the prefix for header guards.
	// Default: UPPER_SNAKE_CASE of package.
	HeaderGuardPrefix string
}

// Generate produces a set of generated files for a typed C SDK client.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkc: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("sdkc").
		Funcs(template.FuncMap{
			"cQuote":         cQuote,
			"cName":          toCName,
			"cTypeName":      toCTypeName,
			"cConstant":      toCConstant,
			"cFieldName":     toCFieldName,
			"snake":          toSnake,
			"upper":          strings.ToUpper,
			"lower":          strings.ToLower,
			"join":           strings.Join,
			"trim":           strings.TrimSpace,
			"indent":         indent,
			"hasPrefix":      strings.HasPrefix,
			"hasSuffix":      strings.HasSuffix,
			"listElem":       listElem,
			"mapElem":        mapElem,
			"isPrimitive":    isPrimitive,
			"isPointerType":  isPointerType,
			"isStruct":       func(k contract.TypeKind) bool { return k == contract.KindStruct },
			"isSlice":        func(k contract.TypeKind) bool { return k == contract.KindSlice },
			"isMap":          func(k contract.TypeKind) bool { return k == contract.KindMap },
			"isUnion":        func(k contract.TypeKind) bool { return k == contract.KindUnion },
			"add":            func(a, b int) int { return a + b },
			"sub":            func(a, b int) int { return a - b },
			"len":            lenHelper,
		}).
		ParseFS(templateFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("sdkc: parse templates: %w", err)
	}

	files := make([]*sdk.File, 0, 12)

	// Generate each file from its template
	templates := []struct {
		name string
		path string
	}{
		{"CMakeLists.txt.tmpl", "CMakeLists.txt"},
		{"Makefile.tmpl", "Makefile"},
		{"main_header.h.tmpl", "include/" + m.Package + "/" + m.Package + ".h"},
		{"errors.h.tmpl", "include/" + m.Package + "/errors.h"},
		{"errors.c.tmpl", "src/errors.c"},
		{"client.h.tmpl", "include/" + m.Package + "/client.h"},
		{"client.c.tmpl", "src/client.c"},
		{"types.h.tmpl", "include/" + m.Package + "/types.h"},
		{"types.c.tmpl", "src/types.c"},
		{"resources.h.tmpl", "include/" + m.Package + "/resources.h"},
		{"resources.c.tmpl", "src/resources.c"},
		{"streaming.h.tmpl", "include/" + m.Package + "/streaming.h"},
		{"streaming.c.tmpl", "src/streaming.c"},
		{"internal.h.tmpl", "src/internal.h"},
	}

	for _, t := range templates {
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, t.name, m); err != nil {
			return nil, fmt.Errorf("sdkc: execute template %s: %w", t.name, err)
		}
		files = append(files, &sdk.File{Path: t.path, Content: out.String()})
	}

	return files, nil
}

type model struct {
	Package           string
	PackageUpper      string
	HeaderGuardPrefix string
	Version           string

	Service struct {
		Name        string
		Sanitized   string
		Description string
	}

	Defaults struct {
		BaseURL string
		Auth    string
		Headers []kv
	}

	HasDate bool
	HasSSE  bool
	HasAny  bool

	Types     []typeModel
	Resources []resourceModel
}

type kv struct {
	K string
	V string
}

type typeModel struct {
	Name        string
	CName       string
	CTypeName   string
	Description string
	Kind        contract.TypeKind

	Fields   []fieldModel
	Elem     string
	ElemC    string
	Tag      string
	Variants []variantModel

	// For struct types that are union variants
	IsUnionVariant bool
	UnionBase      string
	UnionTag       string
	UnionTagValue  string
}

type fieldModel struct {
	Name        string
	CName       string
	JSONName    string
	Description string
	CType       string
	CTypeBase   string

	Optional   bool
	Nullable   bool
	IsRequired bool
	IsPointer  bool
	Enum       []enumValue
	Const      string
}

type enumValue struct {
	Name  string
	Value string
}

type variantModel struct {
	Value       string
	Type        string
	CName       string
	CTypeName   string
	Description string
}

type resourceModel struct {
	Name        string
	CName       string
	Description string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	CName       string
	Description string

	HasInput  bool
	HasOutput bool

	InputType   string
	InputCType  string
	OutputType  string
	OutputCType string

	HTTPMethod string
	HTTPPath   string

	IsStreaming    bool
	StreamMode     string
	StreamIsSSE    bool
	StreamItemType string
	StreamItemC    string
}

func buildModel(svc *contract.Service, cfg *Config) (*model, error) {
	m := &model{}

	// Package name (snake_case for C)
	if cfg != nil && cfg.Package != "" {
		m.Package = cfg.Package
	} else {
		m.Package = toSnake(sanitizeIdent(svc.Name))
		if m.Package == "" {
			m.Package = "sdk"
		}
	}

	// Package upper case
	m.PackageUpper = strings.ToUpper(m.Package)

	// Header guard prefix
	if cfg != nil && cfg.HeaderGuardPrefix != "" {
		m.HeaderGuardPrefix = cfg.HeaderGuardPrefix
	} else {
		m.HeaderGuardPrefix = m.PackageUpper
	}

	// Version
	if cfg != nil && cfg.Version != "" {
		m.Version = cfg.Version
	} else {
		m.Version = "0.0.0"
	}

	// Service info
	m.Service.Name = svc.Name
	m.Service.Sanitized = toSnake(sanitizeIdent(svc.Name))
	m.Service.Description = svc.Description

	// Defaults
	if svc.Defaults != nil {
		m.Defaults.BaseURL = strings.TrimRight(svc.Defaults.BaseURL, "/")
		m.Defaults.Auth = strings.TrimSpace(svc.Defaults.Auth)

		if len(svc.Defaults.Headers) > 0 {
			keys := make([]string, 0, len(svc.Defaults.Headers))
			for k := range svc.Defaults.Headers {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				m.Defaults.Headers = append(m.Defaults.Headers, kv{K: k, V: svc.Defaults.Headers[k]})
			}
		}
	}

	// Build type lookup
	typeByName := map[string]*contract.Type{}
	for _, t := range svc.Types {
		if t != nil && t.Name != "" {
			typeByName[t.Name] = t
		}
	}

	// Feature detection
	m.HasSSE = hasSSE(svc)
	m.HasDate = hasDate(svc)
	m.HasAny = hasAny(svc)

	// Build union variant lookup
	type unionInfo struct {
		Base     string
		Tag      string
		TagValue string
	}
	variantToUnion := make(map[string]unionInfo)
	for _, t := range svc.Types {
		if t != nil && t.Kind == contract.KindUnion {
			for _, v := range t.Variants {
				variantToUnion[string(v.Type)] = unionInfo{
					Base:     toCTypeName(m.Package, t.Name),
					Tag:      t.Tag,
					TagValue: v.Value,
				}
			}
		}
	}

	// Build types
	typeNames := make([]string, 0, len(typeByName))
	for name := range typeByName {
		typeNames = append(typeNames, name)
	}
	sort.Strings(typeNames)

	for _, name := range typeNames {
		t := typeByName[name]
		if t == nil {
			continue
		}

		tm := typeModel{
			Name:        t.Name,
			CName:       toCName(t.Name),
			CTypeName:   toCTypeName(m.Package, t.Name),
			Description: t.Description,
			Kind:        t.Kind,
			Tag:         t.Tag,
		}

		// Check if this type is a variant of a union
		if info, ok := variantToUnion[t.Name]; ok {
			tm.IsUnionVariant = true
			tm.UnionBase = info.Base
			tm.UnionTag = toCName(info.Tag)
			tm.UnionTagValue = info.TagValue
		}

		switch t.Kind {
		case contract.KindStruct:
			for _, f := range t.Fields {
				isOptional := f.Optional || f.Nullable
				ctype := cType(m.Package, typeByName, f.Type, f.Optional, f.Nullable)
				fm := fieldModel{
					Name:        f.Name,
					CName:       toCFieldName(f.Name),
					JSONName:    f.Name,
					Description: f.Description,
					CType:       ctype,
					CTypeBase:   cTypeBase(m.Package, typeByName, f.Type),
					Optional:    f.Optional,
					Nullable:    f.Nullable,
					IsRequired:  !isOptional,
					IsPointer:   isPointerType(ctype),
					Const:       f.Const,
				}

				// Handle enum values
				for _, e := range f.Enum {
					fm.Enum = append(fm.Enum, enumValue{
						Name:  toCConstant(e),
						Value: e,
					})
				}

				tm.Fields = append(tm.Fields, fm)
			}

		case contract.KindSlice:
			tm.Elem = string(t.Elem)
			tm.ElemC = cType(m.Package, typeByName, t.Elem, false, false)

		case contract.KindMap:
			tm.Elem = string(t.Elem)
			tm.ElemC = cType(m.Package, typeByName, t.Elem, false, false)

		case contract.KindUnion:
			for _, v := range t.Variants {
				tm.Variants = append(tm.Variants, variantModel{
					Value:       v.Value,
					Type:        string(v.Type),
					CName:       toCName(string(v.Type)),
					CTypeName:   toCTypeName(m.Package, string(v.Type)),
					Description: v.Description,
				})
			}
		}

		m.Types = append(m.Types, tm)
	}

	// Build resources
	for _, r := range svc.Resources {
		if r == nil {
			continue
		}
		rm := resourceModel{
			Name:        r.Name,
			CName:       toCName(r.Name),
			Description: r.Description,
		}

		for _, mm := range r.Methods {
			if mm == nil {
				continue
			}

			httpMethod := "POST"
			httpPath := "/" + r.Name + "/" + mm.Name
			if mm.HTTP != nil {
				if strings.TrimSpace(mm.HTTP.Method) != "" {
					httpMethod = strings.ToUpper(mm.HTTP.Method)
				}
				if strings.TrimSpace(mm.HTTP.Path) != "" {
					httpPath = mm.HTTP.Path
				}
			}

			hasInput := strings.TrimSpace(string(mm.Input)) != ""
			hasOutput := strings.TrimSpace(string(mm.Output)) != ""

			isStreaming := mm.Stream != nil
			streamMode := ""
			streamIsSSE := false
			streamItem := ""
			streamItemC := ""

			if isStreaming {
				streamMode = strings.TrimSpace(mm.Stream.Mode)
				streamIsSSE = streamMode == "" || strings.EqualFold(streamMode, "sse")
				streamItem = strings.TrimSpace(string(mm.Stream.Item))
				streamItemC = toCTypeName(m.Package, streamItem)
			}

			rm.Methods = append(rm.Methods, methodModel{
				Name:        mm.Name,
				CName:       toCName(mm.Name),
				Description: mm.Description,

				HasInput:    hasInput,
				HasOutput:   hasOutput,
				InputType:   string(mm.Input),
				InputCType:  toCTypeName(m.Package, string(mm.Input)),
				OutputType:  string(mm.Output),
				OutputCType: toCTypeName(m.Package, string(mm.Output)),
				HTTPMethod:  httpMethod,
				HTTPPath:    httpPath,
				IsStreaming: isStreaming,

				StreamMode:     streamMode,
				StreamIsSSE:    streamIsSSE,
				StreamItemType: streamItem,
				StreamItemC:    streamItemC,
			})
		}

		m.Resources = append(m.Resources, rm)
	}

	return m, nil
}

func hasSSE(svc *contract.Service) bool {
	for _, r := range svc.Resources {
		if r == nil {
			continue
		}
		for _, m := range r.Methods {
			if m == nil || m.Stream == nil {
				continue
			}
			mode := strings.TrimSpace(m.Stream.Mode)
			if mode == "" || strings.EqualFold(mode, "sse") {
				return true
			}
		}
	}
	return false
}

func hasDate(svc *contract.Service) bool {
	for _, t := range svc.Types {
		if t == nil {
			continue
		}
		for _, f := range t.Fields {
			if string(f.Type) == "time.Time" {
				return true
			}
		}
		if string(t.Elem) == "time.Time" {
			return true
		}
	}
	return false
}

func hasAny(svc *contract.Service) bool {
	for _, t := range svc.Types {
		if t == nil {
			continue
		}
		for _, f := range t.Fields {
			ref := string(f.Type)
			if ref == "any" || ref == "interface{}" || ref == "json.RawMessage" {
				return true
			}
		}
		ref := string(t.Elem)
		if ref == "any" || ref == "interface{}" || ref == "json.RawMessage" {
			return true
		}
	}
	return false
}

func cType(pkg string, typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "char *"
	}

	base := cTypeBase(pkg, typeByName, ref)

	// For optional/nullable types that are already pointers, no change needed
	if optional || nullable {
		// If it's already a pointer type, return as-is
		if strings.HasSuffix(base, "*") {
			return base
		}
		// For primitive types, we use a nullable wrapper or pointer
		return base
	}
	return base
}

func cTypeBase(pkg string, typeByName map[string]*contract.Type, ref contract.TypeRef) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "char *"
	}

	// Check if it's a known type
	if _, ok := typeByName[r]; ok {
		return toCTypeName(pkg, r) + "_t *"
	}

	switch r {
	case "string":
		return "const char *"
	case "bool", "boolean":
		return "bool"
	case "int":
		return "int32_t"
	case "int8":
		return "int8_t"
	case "int16":
		return "int16_t"
	case "int32":
		return "int32_t"
	case "int64":
		return "int64_t"
	case "uint":
		return "uint32_t"
	case "uint8":
		return "uint8_t"
	case "uint16":
		return "uint16_t"
	case "uint32":
		return "uint32_t"
	case "uint64":
		return "uint64_t"
	case "float32":
		return "float"
	case "float64":
		return "double"
	case "time.Time":
		return "int64_t"
	case "json.RawMessage":
		return "char *"
	case "any", "interface{}":
		return "char *"
	}

	// Handle slice types
	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		// For slices, we use array types
		elemName := toSnake(sanitizeIdent(elem))
		return pkg + "_" + elemName + "_array_t *"
	}

	// Handle map types
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		elemName := toSnake(sanitizeIdent(elem))
		return pkg + "_string_" + elemName + "_map_t *"
	}

	return "char *"
}

// toCName converts a string to snake_case for C identifiers.
func toCName(s string) string {
	if s == "" {
		return ""
	}

	result := toSnake(s)

	// Check for reserved words
	if isCReserved(result) {
		return "_" + result
	}

	return result
}

// toCTypeName converts a string to snake_case with package prefix for C types.
func toCTypeName(pkg, s string) string {
	if s == "" {
		return ""
	}

	name := toSnake(sanitizeIdent(s))
	return pkg + "_" + name
}

// toCFieldName converts to snake_case for struct fields.
func toCFieldName(s string) string {
	if s == "" {
		return ""
	}

	result := toSnake(s)

	// Check for reserved words
	if isCReserved(result) {
		return result + "_"
	}

	return result
}

// toCConstant converts to UPPER_SNAKE_CASE for constants.
func toCConstant(s string) string {
	if s == "" {
		return ""
	}

	return strings.ToUpper(toSnake(sanitizeIdent(s)))
}

// toSnake converts to snake_case.
func toSnake(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	prevLower := false

	for i, r := range s {
		if r == '_' || r == '-' || r == '.' || r == ' ' {
			if b.Len() > 0 {
				b.WriteRune('_')
			}
			prevLower = false
			continue
		}

		if unicode.IsUpper(r) {
			if prevLower && i > 0 {
				b.WriteRune('_')
			}
			b.WriteRune(unicode.ToLower(r))
			prevLower = false
		} else {
			b.WriteRune(r)
			prevLower = unicode.IsLower(r)
		}
	}

	return b.String()
}

// isCReserved checks if a name is a C reserved word.
func isCReserved(s string) bool {
	reserved := map[string]bool{
		"auto": true, "break": true, "case": true, "char": true,
		"const": true, "continue": true, "default": true, "do": true,
		"double": true, "else": true, "enum": true, "extern": true,
		"float": true, "for": true, "goto": true, "if": true,
		"inline": true, "int": true, "long": true, "register": true,
		"restrict": true, "return": true, "short": true, "signed": true,
		"sizeof": true, "static": true, "struct": true, "switch": true,
		"typedef": true, "union": true, "unsigned": true, "void": true,
		"volatile": true, "while": true,
		// C11 additions
		"_Alignas": true, "_Alignof": true, "_Atomic": true, "_Bool": true,
		"_Complex": true, "_Generic": true, "_Imaginary": true, "_Noreturn": true,
		"_Static_assert": true, "_Thread_local": true,
		// Common additions
		"bool": true, "true": true, "false": true, "NULL": true,
	}
	return reserved[s]
}

// sanitizeIdent removes invalid characters from an identifier.
// Non-alphanumeric characters become underscores to preserve word boundaries.
func sanitizeIdent(s string) string {
	var b strings.Builder
	prevUnderscore := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevUnderscore = false
		} else if r == '_' || r == ' ' || r == '-' {
			// Convert separators to underscore, avoiding duplicates
			if !prevUnderscore && b.Len() > 0 {
				b.WriteRune('_')
				prevUnderscore = true
			}
		}
		// Other characters (like !) are simply ignored
	}
	// Trim trailing underscore
	result := b.String()
	return strings.TrimSuffix(result, "_")
}

// cQuote returns a quoted C string literal.
func cQuote(s string) string {
	var b strings.Builder
	b.WriteRune('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString("\\\"")
		case '\\':
			b.WriteString("\\\\")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		case '\x00':
			b.WriteString("\\0")
		default:
			if r < 32 || r > 126 {
				b.WriteString(fmt.Sprintf("\\x%02x", r))
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteRune('"')
	return b.String()
}

// indent adds n spaces of indentation to each line of s.
func indent(n int, s string) string {
	prefix := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

// listElem extracts the element type from an array type string.
func listElem(s string) string {
	if strings.HasSuffix(s, "_array_t *") {
		// Extract element name from {pkg}_{elem}_array_t *
		s = strings.TrimSuffix(s, "_array_t *")
		parts := strings.Split(s, "_")
		if len(parts) > 1 {
			return strings.Join(parts[1:], "_")
		}
	}
	return s
}

// mapElem extracts the value type from a map type string.
func mapElem(s string) string {
	if strings.HasSuffix(s, "_map_t *") {
		// Extract element name from {pkg}_string_{elem}_map_t *
		s = strings.TrimSuffix(s, "_map_t *")
		parts := strings.Split(s, "_string_")
		if len(parts) > 1 {
			return parts[1]
		}
	}
	return s
}

// isPrimitive checks if a type is a C primitive.
func isPrimitive(s string) bool {
	s = strings.TrimSpace(s)
	primitives := map[string]bool{
		"bool": true, "int": true, "long": true, "short": true,
		"char": true, "float": true, "double": true,
		"int8_t": true, "int16_t": true, "int32_t": true, "int64_t": true,
		"uint8_t": true, "uint16_t": true, "uint32_t": true, "uint64_t": true,
		"size_t": true, "ssize_t": true, "ptrdiff_t": true,
	}
	return primitives[s]
}

// isPointerType checks if a type string represents a pointer type.
func isPointerType(s string) bool {
	return strings.HasSuffix(strings.TrimSpace(s), "*")
}

// lenHelper returns the length of a slice or array.
func lenHelper(s interface{}) int {
	switch v := s.(type) {
	case []fieldModel:
		return len(v)
	case []typeModel:
		return len(v)
	case []resourceModel:
		return len(v)
	case []methodModel:
		return len(v)
	case []variantModel:
		return len(v)
	case []enumValue:
		return len(v)
	case []kv:
		return len(v)
	case []string:
		return len(v)
	default:
		return 0
	}
}
