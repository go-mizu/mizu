// Package sdkswift generates typed Swift SDK clients from contract.Service.
package sdkswift

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

//go:embed templates/*.swift.tmpl
var templateFS embed.FS

// Config controls Swift SDK generation.
type Config struct {
	// Package is the Swift module name.
	// Default: sanitized service name.
	Package string

	// Version is the package version for Package.swift.
	// Default: "0.0.0".
	Version string

	// Platforms specifies minimum platform versions.
	// Default: iOS 15.0, macOS 12.0, watchOS 8.0, tvOS 15.0.
	Platforms Platforms
}

// Platforms specifies minimum platform version requirements.
type Platforms struct {
	IOS     string // e.g., "15.0"
	MacOS   string // e.g., "12.0"
	WatchOS string // e.g., "8.0"
	TvOS    string // e.g., "15.0"
}

// Generate produces a set of generated files for a typed Swift SDK client.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkswift: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("sdkswift").
		Funcs(template.FuncMap{
			"swiftQuote":    swiftQuote,
			"swiftString":   swiftQuote,
			"swiftName":     toSwiftName,
			"swiftTypeName": toSwiftTypeName,
			"camel":         toCamel,
			"pascal":        toPascal,
			"join":          strings.Join,
			"trim":          strings.TrimSpace,
			"lower":         strings.ToLower,
			"indent":        indent,
			"hasPrefix":     strings.HasPrefix,
			"add":           func(a, b int) int { return a + b },
			"sub":           func(a, b int) int { return a - b },
			"len":           func(s interface{}) int { return lenHelper(s) },
		}).
		ParseFS(templateFS, "templates/*.swift.tmpl")
	if err != nil {
		return nil, fmt.Errorf("sdkswift: parse templates: %w", err)
	}

	files := make([]*sdk.File, 0, 6)

	// Generate each file from its template
	templates := []struct {
		name string
		path string
	}{
		{"Package.swift.tmpl", "Package.swift"},
		{"Client.swift.tmpl", "Sources/" + m.Package + "/Client.swift"},
		{"Types.swift.tmpl", "Sources/" + m.Package + "/Types.swift"},
		{"Resources.swift.tmpl", "Sources/" + m.Package + "/Resources.swift"},
		{"Streaming.swift.tmpl", "Sources/" + m.Package + "/Streaming.swift"},
		{"Errors.swift.tmpl", "Sources/" + m.Package + "/Errors.swift"},
	}

	for _, t := range templates {
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, t.name, m); err != nil {
			return nil, fmt.Errorf("sdkswift: execute template %s: %w", t.name, err)
		}
		files = append(files, &sdk.File{Path: t.path, Content: out.String()})
	}

	return files, nil
}

type model struct {
	Package string
	Version string

	Platforms platformsModel

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

type platformsModel struct {
	IOS     string
	MacOS   string
	WatchOS string
	TvOS    string
}

type kv struct {
	K string
	V string
}

type typeModel struct {
	Name        string
	SwiftName   string
	Description string
	Kind        contract.TypeKind

	Fields   []fieldModel
	Elem     string
	Tag      string
	Variants []variantModel
}

type fieldModel struct {
	Name        string
	SwiftName   string
	JSONName    string
	Description string
	SwiftType   string

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
	SwiftName   string
	Description string
}

type resourceModel struct {
	Name        string
	SwiftName   string
	ClassName   string
	Description string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	SwiftName   string
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
		p := toPascal(sanitizeIdent(svc.Name))
		if p == "" {
			p = "SDK"
		}
		m.Package = p
	}

	// Version
	if cfg != nil && cfg.Version != "" {
		m.Version = cfg.Version
	} else {
		m.Version = "0.0.0"
	}

	// Platforms
	m.Platforms = platformsModel{
		IOS:     "15.0",
		MacOS:   "12.0",
		WatchOS: "8.0",
		TvOS:    "15.0",
	}
	if cfg != nil {
		if cfg.Platforms.IOS != "" {
			m.Platforms.IOS = cfg.Platforms.IOS
		}
		if cfg.Platforms.MacOS != "" {
			m.Platforms.MacOS = cfg.Platforms.MacOS
		}
		if cfg.Platforms.WatchOS != "" {
			m.Platforms.WatchOS = cfg.Platforms.WatchOS
		}
		if cfg.Platforms.TvOS != "" {
			m.Platforms.TvOS = cfg.Platforms.TvOS
		}
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
			SwiftName:   toSwiftTypeName(t.Name),
			Description: t.Description,
			Kind:        t.Kind,
			Tag:         t.Tag,
		}

		switch t.Kind {
		case contract.KindStruct:
			for _, f := range t.Fields {
				fm := fieldModel{
					Name:        f.Name,
					SwiftName:   toSwiftName(f.Name),
					JSONName:    f.Name,
					Description: f.Description,
					SwiftType:   swiftType(typeByName, f.Type, f.Optional, f.Nullable),
					Optional:    f.Optional,
					Nullable:    f.Nullable,
					Const:       f.Const,
				}

				// Handle enum values
				for _, e := range f.Enum {
					fm.Enum = append(fm.Enum, enumValue{
						Name:  toSwiftName(e),
						Value: e,
					})
				}

				tm.Fields = append(tm.Fields, fm)
			}

		case contract.KindSlice:
			tm.Elem = swiftType(typeByName, t.Elem, false, false)

		case contract.KindMap:
			tm.Elem = swiftType(typeByName, t.Elem, false, false)

		case contract.KindUnion:
			for _, v := range t.Variants {
				tm.Variants = append(tm.Variants, variantModel{
					Value:       v.Value,
					Type:        string(v.Type),
					SwiftName:   toCamel(string(v.Type)),
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
			SwiftName:   toCamel(r.Name),
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
				streamItem = toSwiftTypeName(strings.TrimSpace(string(mm.Stream.Item)))
			}

			rm.Methods = append(rm.Methods, methodModel{
				Name:        mm.Name,
				SwiftName:   toCamel(mm.Name),
				Description: mm.Description,

				HasInput:    hasInput,
				HasOutput:   hasOutput,
				InputType:   toSwiftTypeName(string(mm.Input)),
				OutputType:  toSwiftTypeName(string(mm.Output)),
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

func swiftType(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "AnyCodable"
	}

	base := baseSwiftType(typeByName, r)

	if optional || nullable {
		return base + "?"
	}
	return base
}

func baseSwiftType(typeByName map[string]*contract.Type, r string) string {
	// Check if it's a known type
	if _, ok := typeByName[r]; ok {
		return toSwiftTypeName(r)
	}

	switch r {
	case "string":
		return "String"
	case "bool", "boolean":
		return "Bool"
	case "int":
		return "Int"
	case "int8":
		return "Int8"
	case "int16":
		return "Int16"
	case "int32":
		return "Int32"
	case "int64":
		return "Int64"
	case "uint":
		return "UInt"
	case "uint8":
		return "UInt8"
	case "uint16":
		return "UInt16"
	case "uint32":
		return "UInt32"
	case "uint64":
		return "UInt64"
	case "float32":
		return "Float"
	case "float64":
		return "Double"
	case "time.Time":
		return "Date"
	case "json.RawMessage":
		return "AnyCodable"
	case "any", "interface{}":
		return "AnyCodable"
	}

	// Handle slice types
	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		return "[" + baseSwiftType(typeByName, elem) + "]"
	}

	// Handle map types
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		return "[String: " + baseSwiftType(typeByName, elem) + "]"
	}

	return "AnyCodable"
}

// toSwiftName converts a string to lowerCamelCase for Swift properties/methods.
func toSwiftName(s string) string {
	if s == "" {
		return ""
	}

	// Check for reserved words
	if isSwiftReserved(s) {
		return "`" + s + "`"
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

	result := b.String()
	result = fixAcronyms(result, false)
	return result
}

// toSwiftTypeName converts a string to UpperCamelCase for Swift types.
func toSwiftTypeName(s string) string {
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

	result := b.String()
	result = fixAcronyms(result, true)
	return result
}

// toCamel converts to lowerCamelCase.
func toCamel(s string) string {
	return toSwiftName(s)
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

	result := b.String()
	result = fixAcronyms(result, true)
	return result
}

// fixAcronyms fixes common acronyms to be properly cased.
func fixAcronyms(s string, isTypeName bool) string {
	// For type names, acronyms stay uppercase
	if isTypeName {
		s = strings.ReplaceAll(s, "Id", "ID")
		s = strings.ReplaceAll(s, "Url", "URL")
		s = strings.ReplaceAll(s, "Http", "HTTP")
		s = strings.ReplaceAll(s, "Api", "API")
		s = strings.ReplaceAll(s, "Sse", "SSE")
		s = strings.ReplaceAll(s, "Json", "JSON")
	} else {
		// For properties, acronyms at the start are lowercase
		// "ID" at start -> "id", "ID" elsewhere -> "ID"
		// This is tricky, we'll keep it simple for now
	}
	return s
}

// isSwiftReserved checks if a name is a Swift reserved word.
func isSwiftReserved(s string) bool {
	reserved := map[string]bool{
		"class": true, "deinit": true, "enum": true, "extension": true,
		"func": true, "import": true, "init": true, "inout": true,
		"internal": true, "let": true, "operator": true, "private": true,
		"protocol": true, "public": true, "static": true, "struct": true,
		"subscript": true, "typealias": true, "var": true, "break": true,
		"case": true, "continue": true, "default": true, "defer": true,
		"do": true, "else": true, "fallthrough": true, "for": true,
		"guard": true, "if": true, "in": true, "repeat": true,
		"return": true, "switch": true, "where": true, "while": true,
		"as": true, "catch": true, "false": true, "is": true,
		"nil": true, "rethrows": true, "super": true, "self": true,
		"Self": true, "throw": true, "throws": true, "true": true,
		"try": true, "associatedtype": true, "Type": true, "type": true,
		"Any": true, "async": true, "await": true, "actor": true,
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

// swiftQuote returns a quoted Swift string literal.
func swiftQuote(s string) string {
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
