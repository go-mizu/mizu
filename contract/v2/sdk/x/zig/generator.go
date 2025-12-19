// Package sdkzig generates typed Zig SDK clients from contract.Service.
package sdkzig

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

//go:embed templates/*.zig.tmpl templates/*.zon.tmpl
var templateFS embed.FS

// Config controls Zig SDK generation.
type Config struct {
	// Package is the Zig package name (snake_case).
	// Default: sanitized snake_case service name.
	Package string

	// Version is the package version.
	// Default: "0.1.0".
	Version string

	// MinZigVersion is the minimum Zig version required.
	// Default: "0.11.0".
	MinZigVersion string
}

// Generate produces a set of generated files for a typed Zig SDK client.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkzig: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("sdkzig").
		Funcs(template.FuncMap{
			"zigQuote":    zigQuote,
			"zigString":   zigQuote,
			"zigName":     toZigName,
			"zigTypeName": toZigTypeName,
			"zigType":     func(ref string, opt, null bool) string { return zigType(m.typeByName, contract.TypeRef(ref), opt, null) },
			"snake":       toSnake,
			"pascal":      toPascal,
			"camel":       toCamel,
			"screaming":   toScreamingSnake,
			"httpMethod":  strings.ToLower,
			"upper":       strings.ToUpper,
			"join":        strings.Join,
			"trim":        strings.TrimSpace,
			"lower":       strings.ToLower,
			"indent":      indent,
			"hasPrefix":   strings.HasPrefix,
			"add":         func(a, b int) int { return a + b },
			"sub":         func(a, b int) int { return a - b },
			"len":         func(s interface{}) int { return lenHelper(s) },
		}).
		ParseFS(templateFS, "templates/*.zig.tmpl", "templates/*.zon.tmpl")
	if err != nil {
		return nil, fmt.Errorf("sdkzig: parse templates: %w", err)
	}

	files := make([]*sdk.File, 0, 8)

	// Generate each file from its template
	templates := []struct {
		name string
		path string
	}{
		{"build.zig.tmpl", "build.zig"},
		{"build.zig.zon.tmpl", "build.zig.zon"},
		{"root.zig.tmpl", "src/root.zig"},
		{"client.zig.tmpl", "src/client.zig"},
		{"types.zig.tmpl", "src/types.zig"},
		{"resources.zig.tmpl", "src/resources.zig"},
		{"streaming.zig.tmpl", "src/streaming.zig"},
		{"errors.zig.tmpl", "src/errors.zig"},
	}

	for _, t := range templates {
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, t.name, m); err != nil {
			return nil, fmt.Errorf("sdkzig: execute template %s: %w", t.name, err)
		}
		files = append(files, &sdk.File{Path: t.path, Content: out.String()})
	}

	return files, nil
}

type model struct {
	Package       string
	Version       string
	MinZigVersion string

	Service struct {
		Name        string
		Sanitized   string
		Description string
	}

	Client struct {
		BaseURL string
		Auth    string
		Headers []kv
	}

	HasDate bool
	HasSSE  bool
	HasAny  bool

	Types     []typeModel
	Resources []resourceModel

	typeByName map[string]*contract.Type
}

type kv struct {
	K string
	V string
}

type typeModel struct {
	Name        string
	ZigName     string
	Description string
	Kind        contract.TypeKind

	Fields   []fieldModel
	Elem     string
	ElemZig  string
	Tag      string
	Variants []variantModel
	HasEnum  bool
	Enum     []enumValue
}

type fieldModel struct {
	Name        string
	ZigName     string
	JSONName    string
	Description string
	ZigType     string
	ZigTypeRaw  string

	Optional bool
	Nullable bool
	Enum     []enumValue
	Const    string
}

type enumValue struct {
	Name    string
	ZigName string
	Value   string
}

type variantModel struct {
	Value       string
	Type        string
	ZigName     string
	ZigType     string
	PascalName  string
	Description string
}

type resourceModel struct {
	Name        string
	ZigName     string
	StructName  string
	Description string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	ZigName     string
	Description string

	HasInput  bool
	HasOutput bool

	InputType  string
	OutputType string

	HTTPMethod string
	HTTPPath   string

	IsStreaming    bool
	StreamMode     string
	StreamIsSSE    bool
	StreamItemType string
}

func buildModel(svc *contract.Service, cfg *Config) (*model, error) {
	m := &model{}

	// Package name
	if cfg != nil && cfg.Package != "" {
		m.Package = cfg.Package
	} else {
		m.Package = toSnake(sanitizeIdent(svc.Name))
		if m.Package == "" {
			m.Package = "sdk"
		}
	}

	// Version
	if cfg != nil && cfg.Version != "" {
		m.Version = cfg.Version
	} else {
		m.Version = "0.1.0"
	}

	// Min Zig version
	if cfg != nil && cfg.MinZigVersion != "" {
		m.MinZigVersion = cfg.MinZigVersion
	} else {
		m.MinZigVersion = "0.11.0"
	}

	// Service info
	m.Service.Name = svc.Name
	m.Service.Sanitized = toPascal(sanitizeIdent(svc.Name))
	m.Service.Description = svc.Description

	// Defaults
	if svc.Client != nil {
		m.Client.BaseURL = strings.TrimRight(svc.Client.BaseURL, "/")
		m.Client.Auth = strings.TrimSpace(svc.Client.Auth)

		if len(svc.Client.Headers) > 0 {
			keys := make([]string, 0, len(svc.Client.Headers))
			for k := range svc.Client.Headers {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				m.Client.Headers = append(m.Client.Headers, kv{K: k, V: svc.Client.Headers[k]})
			}
		}
	}

	// Build type lookup
	m.typeByName = map[string]*contract.Type{}
	for _, t := range svc.Types {
		if t != nil && t.Name != "" {
			m.typeByName[t.Name] = t
		}
	}

	// Feature detection
	m.HasSSE = hasSSE(svc)
	m.HasDate = hasDate(svc)
	m.HasAny = hasAny(svc)

	// Build types
	typeNames := make([]string, 0, len(m.typeByName))
	for name := range m.typeByName {
		typeNames = append(typeNames, name)
	}
	sort.Strings(typeNames)

	for _, name := range typeNames {
		t := m.typeByName[name]
		if t == nil {
			continue
		}

		tm := typeModel{
			Name:        t.Name,
			ZigName:     toZigTypeName(t.Name),
			Description: t.Description,
			Kind:        t.Kind,
			Tag:         t.Tag,
		}

		switch t.Kind {
		case contract.KindStruct:
			for _, f := range t.Fields {
				fm := fieldModel{
					Name:        f.Name,
					ZigName:     toZigName(f.Name),
					JSONName:    f.Name,
					Description: f.Description,
					ZigType:     zigType(m.typeByName, f.Type, f.Optional, f.Nullable),
					ZigTypeRaw:  zigType(m.typeByName, f.Type, false, false),
					Optional:    f.Optional,
					Nullable:    f.Nullable,
					Const:       f.Const,
				}

				// Handle enum values
				for _, e := range f.Enum {
					fm.Enum = append(fm.Enum, enumValue{
						Name:    e,
						ZigName: toZigName(e),
						Value:   e,
					})
				}

				tm.Fields = append(tm.Fields, fm)
			}

		case contract.KindSlice:
			tm.Elem = string(t.Elem)
			tm.ElemZig = zigType(m.typeByName, t.Elem, false, false)

		case contract.KindMap:
			tm.Elem = string(t.Elem)
			tm.ElemZig = zigType(m.typeByName, t.Elem, false, false)

		case contract.KindUnion:
			for _, v := range t.Variants {
				tm.Variants = append(tm.Variants, variantModel{
					Value:       v.Value,
					Type:        string(v.Type),
					ZigName:     toZigName(v.Value),
					ZigType:     toZigTypeName(string(v.Type)),
					PascalName:  toPascal(v.Value),
					Description: v.Description,
				})
			}
		}

		// Check if this struct has enum fields (for generating separate enum types)
		for _, f := range tm.Fields {
			if len(f.Enum) > 0 {
				tm.HasEnum = true
				// Create enum values at the type level
				tm.Enum = append(tm.Enum, f.Enum...)
				break
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
			ZigName:     toZigName(r.Name),
			StructName:  toPascal(r.Name),
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

			if isStreaming {
				streamMode = strings.TrimSpace(mm.Stream.Mode)
				streamIsSSE = streamMode == "" || strings.EqualFold(streamMode, "sse")
				streamItem = toZigTypeName(strings.TrimSpace(string(mm.Stream.Item)))
			}

			rm.Methods = append(rm.Methods, methodModel{
				Name:        mm.Name,
				ZigName:     toZigName(mm.Name),
				Description: mm.Description,

				HasInput:    hasInput,
				HasOutput:   hasOutput,
				InputType:   toZigTypeName(string(mm.Input)),
				OutputType:  toZigTypeName(string(mm.Output)),
				HTTPMethod:  httpMethod,
				HTTPPath:    httpPath,
				IsStreaming: isStreaming,

				StreamMode:     streamMode,
				StreamIsSSE:    streamIsSSE,
				StreamItemType: streamItem,
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

func zigType(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "std.json.Value"
	}

	base := baseZigType(typeByName, r)

	if optional || nullable {
		return "?" + base
	}
	return base
}

func baseZigType(typeByName map[string]*contract.Type, r string) string {
	// Check if it's a known type
	if _, ok := typeByName[r]; ok {
		return toZigTypeName(r)
	}

	switch r {
	case "string":
		return "[]const u8"
	case "bool", "boolean":
		return "bool"
	case "int":
		return "i32"
	case "int8":
		return "i8"
	case "int16":
		return "i16"
	case "int32":
		return "i32"
	case "int64":
		return "i64"
	case "uint":
		return "u32"
	case "uint8":
		return "u8"
	case "uint16":
		return "u16"
	case "uint32":
		return "u32"
	case "uint64":
		return "u64"
	case "float32":
		return "f32"
	case "float64":
		return "f64"
	case "time.Time":
		return "i64"
	case "json.RawMessage":
		return "std.json.Value"
	case "any", "interface{}":
		return "std.json.Value"
	}

	// Handle slice types
	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		return "[]const " + baseZigType(typeByName, elem)
	}

	// Handle map types
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		return "std.StringHashMap(" + baseZigType(typeByName, elem) + ")"
	}

	return "std.json.Value"
}

// toZigName converts a string to snake_case for Zig fields/methods.
func toZigName(s string) string {
	if s == "" {
		return ""
	}

	result := toSnake(s)

	// Check for reserved words
	if isZigReserved(result) {
		return "@\"" + result + "\""
	}

	return result
}

// toZigTypeName converts a string to PascalCase for Zig types.
func toZigTypeName(s string) string {
	if s == "" {
		return ""
	}

	return toPascal(s)
}

// toSnake converts to snake_case.
func toSnake(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	prevWasUpper := false
	prevWasLower := false

	for i, r := range s {
		if r == '_' || r == '-' || r == '.' || r == ' ' {
			if b.Len() > 0 {
				b.WriteRune('_')
			}
			prevWasUpper = false
			prevWasLower = false
			continue
		}

		if unicode.IsUpper(r) {
			if i > 0 && (prevWasLower || (prevWasUpper && i+1 < len(s) && unicode.IsLower(rune(s[i+1])))) {
				b.WriteRune('_')
			}
			b.WriteRune(unicode.ToLower(r))
			prevWasUpper = true
			prevWasLower = false
		} else {
			b.WriteRune(r)
			prevWasUpper = false
			prevWasLower = true
		}
	}

	return b.String()
}

// toPascal converts to PascalCase.
func toPascal(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	capNext := true

	for _, r := range s {
		if r == '_' || r == '-' || r == '.' || r == ' ' {
			capNext = true
			continue
		}
		if capNext {
			b.WriteRune(unicode.ToUpper(r))
			capNext = false
			continue
		}
		b.WriteRune(r)
	}

	return b.String()
}

// toCamel converts to camelCase.
func toCamel(s string) string {
	pascal := toPascal(s)
	if len(pascal) == 0 {
		return ""
	}
	return strings.ToLower(string(pascal[0])) + pascal[1:]
}

// toScreamingSnake converts to SCREAMING_SNAKE_CASE.
func toScreamingSnake(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	prevWasUpper := false
	prevWasLower := false

	for i, r := range s {
		if r == '_' || r == '-' || r == '.' || r == ' ' {
			if b.Len() > 0 {
				b.WriteRune('_')
			}
			prevWasUpper = false
			prevWasLower = false
			continue
		}

		if unicode.IsUpper(r) {
			if i > 0 && (prevWasLower || (prevWasUpper && i+1 < len(s) && unicode.IsLower(rune(s[i+1])))) {
				b.WriteRune('_')
			}
			b.WriteRune(r)
			prevWasUpper = true
			prevWasLower = false
		} else {
			b.WriteRune(unicode.ToUpper(r))
			prevWasUpper = false
			prevWasLower = true
		}
	}

	return b.String()
}

// isZigReserved checks if a name is a Zig reserved word.
func isZigReserved(s string) bool {
	reserved := map[string]bool{
		// Keywords
		"addrspace": true, "align": true, "allowzero": true, "and": true, "anyframe": true,
		"anytype": true, "asm": true, "async": true, "await": true, "break": true,
		"catch": true, "comptime": true, "const": true, "continue": true, "defer": true,
		"else": true, "enum": true, "errdefer": true, "error": true, "export": true,
		"extern": true, "false": true, "fn": true, "for": true, "if": true,
		"inline": true, "noalias": true, "nosuspend": true, "null": true, "opaque": true,
		"or": true, "orelse": true, "packed": true, "pub": true, "resume": true,
		"return": true, "struct": true, "suspend": true, "switch": true, "test": true,
		"threadlocal": true, "true": true, "try": true, "type": true, "undefined": true,
		"union": true, "unreachable": true, "usingnamespace": true, "var": true, "volatile": true,
		"while": true,
		// Primitives
		"void": true, "anyerror": true, "noreturn": true,
	}
	return reserved[s]
}

// sanitizeIdent removes invalid characters from an identifier.
func sanitizeIdent(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// zigQuote returns a quoted Zig string literal.
func zigQuote(s string) string {
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
