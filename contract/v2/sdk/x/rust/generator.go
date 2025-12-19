// Package sdkrust generates typed Rust SDK clients from contract.Service.
package sdkrust

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

//go:embed templates/*.rs.tmpl templates/*.toml.tmpl
var templateFS embed.FS

// Config controls Rust SDK generation.
type Config struct {
	// Crate is the Rust crate name (kebab-case).
	// Default: sanitized kebab-case service name.
	Crate string

	// Version is the crate version.
	// Default: "0.1.0".
	Version string

	// Authors is the list of crate authors.
	Authors []string

	// Repository is the crate repository URL.
	Repository string

	// Documentation is the docs.rs URL.
	Documentation string

	// Edition is the Rust edition (2018, 2021).
	// Default: "2021".
	Edition string
}

// Generate produces a set of generated files for a typed Rust SDK client.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkrust: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("sdkrust").
		Funcs(template.FuncMap{
			"rustQuote":    rustQuote,
			"rustString":   rustQuote,
			"rustName":     toRustName,
			"rustTypeName": toRustTypeName,
			"rustType":     func(ref string, opt, null bool) string { return rustType(m.typeByName, contract.TypeRef(ref), opt, null) },
			"snake":        toSnake,
			"pascal":       toPascal,
			"screaming":    toScreamingSnake,
			"httpMethod":   strings.ToLower,
			"upper":        strings.ToUpper,
			"join":         strings.Join,
			"trim":         strings.TrimSpace,
			"lower":        strings.ToLower,
			"indent":       indent,
			"hasPrefix":    strings.HasPrefix,
			"add":          func(a, b int) int { return a + b },
			"sub":          func(a, b int) int { return a - b },
			"len":          func(s interface{}) int { return lenHelper(s) },
		}).
		ParseFS(templateFS, "templates/*.rs.tmpl", "templates/*.toml.tmpl")
	if err != nil {
		return nil, fmt.Errorf("sdkrust: parse templates: %w", err)
	}

	files := make([]*sdk.File, 0, 8)

	// Generate each file from its template
	templates := []struct {
		name string
		path string
	}{
		{"Cargo.toml.tmpl", "Cargo.toml"},
		{"lib.rs.tmpl", "src/lib.rs"},
		{"client.rs.tmpl", "src/client.rs"},
		{"types.rs.tmpl", "src/types.rs"},
		{"resources.rs.tmpl", "src/resources.rs"},
		{"streaming.rs.tmpl", "src/streaming.rs"},
		{"error.rs.tmpl", "src/error.rs"},
	}

	for _, t := range templates {
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, t.name, m); err != nil {
			return nil, fmt.Errorf("sdkrust: execute template %s: %w", t.name, err)
		}
		files = append(files, &sdk.File{Path: t.path, Content: out.String()})
	}

	return files, nil
}

type model struct {
	Crate         string
	Version       string
	Authors       []string
	Repository    string
	Documentation string
	Edition       string

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

	typeByName map[string]*contract.Type
}

type kv struct {
	K string
	V string
}

type typeModel struct {
	Name        string
	RustName    string
	Description string
	Kind        contract.TypeKind

	Fields   []fieldModel
	Elem     string
	Tag      string
	Variants []variantModel
}

type fieldModel struct {
	Name        string
	RustName    string
	JSONName    string
	Description string
	RustType    string
	RustTypeRaw string

	Optional bool
	Nullable bool
	Enum     []enumValue
	Const    string
}

type enumValue struct {
	Name  string
	Value string
}

type variantModel struct {
	Value       string
	Type        string
	RustName    string
	Description string
}

type resourceModel struct {
	Name        string
	RustName    string
	StructName  string
	Description string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	RustName    string
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

	// Crate name
	if cfg != nil && cfg.Crate != "" {
		m.Crate = cfg.Crate
	} else {
		m.Crate = toKebab(sanitizeIdent(svc.Name))
		if m.Crate == "" {
			m.Crate = "sdk"
		}
	}

	// Version
	if cfg != nil && cfg.Version != "" {
		m.Version = cfg.Version
	} else {
		m.Version = "0.1.0"
	}

	// Authors
	if cfg != nil && len(cfg.Authors) > 0 {
		m.Authors = cfg.Authors
	}

	// Repository
	if cfg != nil && cfg.Repository != "" {
		m.Repository = cfg.Repository
	}

	// Documentation
	if cfg != nil && cfg.Documentation != "" {
		m.Documentation = cfg.Documentation
	}

	// Edition
	if cfg != nil && cfg.Edition != "" {
		m.Edition = cfg.Edition
	} else {
		m.Edition = "2021"
	}

	// Service info
	m.Service.Name = svc.Name
	m.Service.Sanitized = toPascal(sanitizeIdent(svc.Name))
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
			RustName:    toRustTypeName(t.Name),
			Description: t.Description,
			Kind:        t.Kind,
			Tag:         t.Tag,
		}

		switch t.Kind {
		case contract.KindStruct:
			for _, f := range t.Fields {
				fm := fieldModel{
					Name:        f.Name,
					RustName:    toRustName(f.Name),
					JSONName:    f.Name,
					Description: f.Description,
					RustType:    rustType(m.typeByName, f.Type, f.Optional, f.Nullable),
					RustTypeRaw: rustType(m.typeByName, f.Type, false, false),
					Optional:    f.Optional,
					Nullable:    f.Nullable,
					Const:       f.Const,
				}

				// Handle enum values
				for _, e := range f.Enum {
					fm.Enum = append(fm.Enum, enumValue{
						Name:  toPascal(e),
						Value: e,
					})
				}

				tm.Fields = append(tm.Fields, fm)
			}

		case contract.KindSlice:
			tm.Elem = rustType(m.typeByName, contract.TypeRef(t.Elem), false, false)

		case contract.KindMap:
			tm.Elem = rustType(m.typeByName, contract.TypeRef(t.Elem), false, false)

		case contract.KindUnion:
			for _, v := range t.Variants {
				tm.Variants = append(tm.Variants, variantModel{
					Value:       v.Value,
					Type:        string(v.Type),
					RustName:    toPascal(v.Value),
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
			RustName:    toSnake(r.Name),
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
				streamItem = toRustTypeName(strings.TrimSpace(string(mm.Stream.Item)))
			}

			rm.Methods = append(rm.Methods, methodModel{
				Name:        mm.Name,
				RustName:    toSnake(mm.Name),
				Description: mm.Description,

				HasInput:    hasInput,
				HasOutput:   hasOutput,
				InputType:   toRustTypeName(string(mm.Input)),
				OutputType:  toRustTypeName(string(mm.Output)),
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

func rustType(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "serde_json::Value"
	}

	base := baseRustType(typeByName, r)

	if optional || nullable {
		return "Option<" + base + ">"
	}
	return base
}

func baseRustType(typeByName map[string]*contract.Type, r string) string {
	// Check if it's a known type
	if _, ok := typeByName[r]; ok {
		return toRustTypeName(r)
	}

	switch r {
	case "string":
		return "String"
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
		return "chrono::DateTime<chrono::Utc>"
	case "json.RawMessage":
		return "serde_json::Value"
	case "any", "interface{}":
		return "serde_json::Value"
	}

	// Handle slice types
	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		return "Vec<" + baseRustType(typeByName, elem) + ">"
	}

	// Handle map types
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		return "std::collections::HashMap<String, " + baseRustType(typeByName, elem) + ">"
	}

	return "serde_json::Value"
}

// toRustName converts a string to snake_case for Rust fields/methods.
func toRustName(s string) string {
	if s == "" {
		return ""
	}

	result := toSnake(s)

	// Check for reserved words
	if isRustReserved(result) {
		return "r#" + result
	}

	return result
}

// toRustTypeName converts a string to PascalCase for Rust types.
func toRustTypeName(s string) string {
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

// toKebab converts to kebab-case.
func toKebab(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	prevWasUpper := false
	prevWasLower := false

	for i, r := range s {
		if r == '_' || r == '-' || r == '.' || r == ' ' {
			if b.Len() > 0 {
				b.WriteRune('-')
			}
			prevWasUpper = false
			prevWasLower = false
			continue
		}

		if unicode.IsUpper(r) {
			if i > 0 && (prevWasLower || (prevWasUpper && i+1 < len(s) && unicode.IsLower(rune(s[i+1])))) {
				b.WriteRune('-')
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

// isRustReserved checks if a name is a Rust reserved word.
func isRustReserved(s string) bool {
	reserved := map[string]bool{
		// Strict keywords
		"as": true, "async": true, "await": true, "break": true, "const": true,
		"continue": true, "crate": true, "dyn": true, "else": true, "enum": true,
		"extern": true, "false": true, "fn": true, "for": true, "if": true,
		"impl": true, "in": true, "let": true, "loop": true, "match": true,
		"mod": true, "move": true, "mut": true, "pub": true, "ref": true,
		"return": true, "self": true, "Self": true, "static": true, "struct": true,
		"super": true, "trait": true, "true": true, "type": true, "unsafe": true,
		"use": true, "where": true, "while": true,
		// Reserved keywords
		"abstract": true, "become": true, "box": true, "do": true, "final": true,
		"macro": true, "override": true, "priv": true, "try": true, "typeof": true,
		"unsized": true, "virtual": true, "yield": true,
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

// rustQuote returns a quoted Rust string literal.
func rustQuote(s string) string {
	return fmt.Sprintf("%q", s)
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
