// Package sdkdart generates typed Dart SDK clients from contract.Service.
package sdkdart

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

//go:embed templates/*.dart.tmpl templates/*.yaml.tmpl
var templateFS embed.FS

// Config controls Dart SDK generation.
type Config struct {
	// Package is the Dart package name (snake_case).
	// Default: sanitized lowercase service name.
	Package string

	// Version is the package version for pubspec.yaml.
	// Default: "0.0.0".
	Version string

	// Description is the package description.
	// Default: service description or "Generated Dart SDK".
	Description string

	// MinSDK is the minimum Dart SDK version.
	// Default: "3.0.0".
	MinSDK string
}

// Generate produces a set of generated files for a typed Dart SDK client.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkdart: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("sdkdart").
		Funcs(template.FuncMap{
			"dartQuote":    dartQuote,
			"dartString":   dartQuote,
			"dartName":     toDartName,
			"dartTypeName": toDartTypeName,
			"camel":        toCamel,
			"pascal":       toPascal,
			"snake":        toSnake,
			"join":         strings.Join,
			"trim":         strings.TrimSpace,
			"lower":        strings.ToLower,
			"indent":       indent,
			"hasPrefix":    strings.HasPrefix,
			"hasSuffix":    strings.HasSuffix,
			"listElem":     listElem,
			"mapElem":      mapElem,
			"isPrimitive":  isPrimitive,
			"add":          func(a, b int) int { return a + b },
			"sub":          func(a, b int) int { return a - b },
			"len":          func(s interface{}) int { return lenHelper(s) },
		}).
		ParseFS(templateFS, "templates/*.dart.tmpl", "templates/*.yaml.tmpl")
	if err != nil {
		return nil, fmt.Errorf("sdkdart: parse templates: %w", err)
	}

	files := make([]*sdk.File, 0, 7)

	// Generate each file from its template
	templates := []struct {
		name string
		path string
	}{
		{"pubspec.yaml.tmpl", "pubspec.yaml"},
		{"lib.dart.tmpl", "lib/" + m.Package + ".dart"},
		{"client.dart.tmpl", "lib/src/client.dart"},
		{"types.dart.tmpl", "lib/src/types.dart"},
		{"resources.dart.tmpl", "lib/src/resources.dart"},
		{"streaming.dart.tmpl", "lib/src/streaming.dart"},
		{"errors.dart.tmpl", "lib/src/errors.dart"},
	}

	for _, t := range templates {
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, t.name, m); err != nil {
			return nil, fmt.Errorf("sdkdart: execute template %s: %w", t.name, err)
		}
		files = append(files, &sdk.File{Path: t.path, Content: out.String()})
	}

	return files, nil
}

type model struct {
	Package     string
	Version     string
	Description string
	MinSDK      string

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
}

type kv struct {
	K string
	V string
}

type typeModel struct {
	Name        string
	DartName    string
	Description string
	Kind        contract.TypeKind

	Fields   []fieldModel
	Elem     string
	Tag      string
	Variants []variantModel
}

type fieldModel struct {
	Name        string
	DartName    string
	JSONName    string
	Description string
	DartType    string

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
	DartName    string
	ClassName   string
	Description string
}

type resourceModel struct {
	Name        string
	DartName    string
	ClassName   string
	Description string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	DartName    string
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

	// Package name (snake_case for Dart)
	if cfg != nil && cfg.Package != "" {
		m.Package = cfg.Package
	} else {
		p := toSnake(sanitizeIdent(svc.Name))
		if p == "" {
			p = "sdk"
		}
		m.Package = p
	}

	// Version
	if cfg != nil && cfg.Version != "" {
		m.Version = cfg.Version
	} else {
		m.Version = "0.0.0"
	}

	// Description
	if cfg != nil && cfg.Description != "" {
		m.Description = cfg.Description
	} else if svc.Description != "" {
		m.Description = svc.Description
	} else {
		m.Description = "Generated Dart SDK"
	}

	// MinSDK
	if cfg != nil && cfg.MinSDK != "" {
		m.MinSDK = cfg.MinSDK
	} else {
		m.MinSDK = "3.0.0"
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
			DartName:    toDartTypeName(t.Name),
			Description: t.Description,
			Kind:        t.Kind,
			Tag:         t.Tag,
		}

		switch t.Kind {
		case contract.KindStruct:
			for _, f := range t.Fields {
				fm := fieldModel{
					Name:        f.Name,
					DartName:    toDartName(f.Name),
					JSONName:    f.Name,
					Description: f.Description,
					DartType:    dartType(typeByName, f.Type, f.Optional, f.Nullable),
					Optional:    f.Optional,
					Nullable:    f.Nullable,
					Const:       f.Const,
				}

				// Handle enum values
				for _, e := range f.Enum {
					fm.Enum = append(fm.Enum, enumValue{
						Name:  toCamel(sanitizeIdent(e)),
						Value: e,
					})
				}

				tm.Fields = append(tm.Fields, fm)
			}

		case contract.KindSlice:
			tm.Elem = dartType(typeByName, t.Elem, false, false)

		case contract.KindMap:
			tm.Elem = dartType(typeByName, t.Elem, false, false)

		case contract.KindUnion:
			for _, v := range t.Variants {
				tm.Variants = append(tm.Variants, variantModel{
					Value:       v.Value,
					Type:        string(v.Type),
					DartName:    toCamel(string(v.Type)),
					ClassName:   toPascal(string(v.Type)),
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
			DartName:    toCamel(r.Name),
			ClassName:   toPascal(r.Name) + "Resource",
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
				streamItem = toDartTypeName(strings.TrimSpace(string(mm.Stream.Item)))
			}

			rm.Methods = append(rm.Methods, methodModel{
				Name:        mm.Name,
				DartName:    toCamel(mm.Name),
				Description: mm.Description,

				HasInput:    hasInput,
				HasOutput:   hasOutput,
				InputType:   toDartTypeName(string(mm.Input)),
				OutputType:  toDartTypeName(string(mm.Output)),
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

func dartType(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "Object"
	}

	base := baseDartType(typeByName, r)

	if optional || nullable {
		return base + "?"
	}
	return base
}

func baseDartType(typeByName map[string]*contract.Type, r string) string {
	// Check if it's a known type
	if _, ok := typeByName[r]; ok {
		return toDartTypeName(r)
	}

	switch r {
	case "string":
		return "String"
	case "bool", "boolean":
		return "bool"
	case "int", "int8", "int16", "int32", "int64":
		return "int"
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "int"
	case "float32", "float64":
		return "double"
	case "time.Time":
		return "DateTime"
	case "json.RawMessage":
		return "Object"
	case "any", "interface{}":
		return "Object"
	}

	// Handle slice types
	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		return "List<" + baseDartType(typeByName, elem) + ">"
	}

	// Handle map types
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		return "Map<String, " + baseDartType(typeByName, elem) + ">"
	}

	return "Object"
}

// toDartName converts a string to lowerCamelCase for Dart properties/methods.
func toDartName(s string) string {
	if s == "" {
		return ""
	}

	result := toCamel(s)

	// Check for reserved words
	if isDartReserved(result) {
		return "$" + result
	}

	return result
}

// toDartTypeName converts a string to UpperCamelCase for Dart types.
func toDartTypeName(s string) string {
	if s == "" {
		return ""
	}

	return toPascal(s)
}

// toCamel converts to lowerCamelCase.
func toCamel(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	capNext := false
	first := true

	for _, r := range s {
		if r == '_' || r == '-' || r == '.' {
			capNext = true
			continue
		}
		if first {
			b.WriteRune(unicode.ToLower(r))
			first = false
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

// toPascal converts to UpperCamelCase.
func toPascal(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	capNext := true

	for _, r := range s {
		if r == '_' || r == '-' || r == '.' {
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

// toSnake converts to snake_case.
func toSnake(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	prevWasUpper := false

	for i, r := range s {
		if r == '_' || r == '-' || r == '.' || r == ' ' {
			if b.Len() > 0 {
				b.WriteRune('_')
			}
			prevWasUpper = false
			continue
		}

		if unicode.IsUpper(r) {
			if i > 0 && !prevWasUpper && b.Len() > 0 {
				b.WriteRune('_')
			}
			b.WriteRune(unicode.ToLower(r))
			prevWasUpper = true
		} else {
			b.WriteRune(r)
			prevWasUpper = false
		}
	}

	return b.String()
}

// isDartReserved checks if a name is a Dart reserved word.
func isDartReserved(s string) bool {
	reserved := map[string]bool{
		"abstract": true, "as": true, "assert": true, "async": true,
		"await": true, "break": true, "case": true, "catch": true,
		"class": true, "const": true, "continue": true, "covariant": true,
		"default": true, "deferred": true, "do": true, "dynamic": true,
		"else": true, "enum": true, "export": true, "extends": true,
		"extension": true, "external": true, "factory": true, "false": true,
		"final": true, "finally": true, "for": true, "Function": true,
		"get": true, "hide": true, "if": true, "implements": true,
		"import": true, "in": true, "interface": true, "is": true,
		"late": true, "library": true, "mixin": true, "new": true,
		"null": true, "of": true, "on": true, "operator": true,
		"part": true, "required": true, "rethrow": true, "return": true,
		"sealed": true, "set": true, "show": true, "static": true,
		"super": true, "switch": true, "sync": true, "this": true,
		"throw": true, "true": true, "try": true, "type": true,
		"typedef": true, "var": true, "void": true, "when": true,
		"while": true, "with": true, "yield": true,
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

// dartQuote returns a quoted Dart string literal.
func dartQuote(s string) string {
	// Use single quotes for Dart
	var b strings.Builder
	b.WriteRune('\'')
	for _, r := range s {
		switch r {
		case '\'':
			b.WriteString("\\'")
		case '\\':
			b.WriteString("\\\\")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		case '$':
			b.WriteString("\\$")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteRune('\'')
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

// listElem extracts the element type from a List<T> type string.
func listElem(s string) string {
	if strings.HasPrefix(s, "List<") && strings.HasSuffix(s, ">") {
		return strings.TrimSuffix(strings.TrimPrefix(s, "List<"), ">")
	}
	return s
}

// mapElem extracts the value type from a Map<String, T> type string.
func mapElem(s string) string {
	if strings.HasPrefix(s, "Map<String, ") && strings.HasSuffix(s, ">") {
		return strings.TrimSuffix(strings.TrimPrefix(s, "Map<String, "), ">")
	}
	return s
}

// isPrimitive checks if a type is a Dart primitive that doesn't need .toJson()/.fromJson().
func isPrimitive(s string) bool {
	s = strings.TrimSuffix(s, "?")
	primitives := map[string]bool{
		"String": true, "int": true, "double": true, "bool": true, "Object": true,
		"DateTime": true, "dynamic": true, "num": true,
	}
	return primitives[s]
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
