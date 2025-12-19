// Package sdkcsharp generates typed C# SDK clients from contract.Service.
package sdkcsharp

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

//go:embed templates/*.cs.tmpl templates/*.csproj.tmpl
var templateFS embed.FS

// Config controls C# SDK generation.
type Config struct {
	// Namespace is the C# namespace for generated code.
	// Default: sanitized PascalCase service name + ".Sdk".
	Namespace string

	// PackageName is the NuGet package name.
	// Default: namespace.
	PackageName string

	// Version is the package version.
	// Default: "0.0.0".
	Version string

	// TargetFramework is the target .NET version.
	// Default: "net8.0".
	TargetFramework string

	// Nullable enables nullable reference types.
	// Default: true.
	Nullable bool
}

// Generate produces a set of generated files for a typed C# SDK client.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkcsharp: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("sdkcsharp").
		Funcs(template.FuncMap{
			"csharpQuote":      csharpQuote,
			"csharpString":     csharpQuote,
			"csharpName":       toCSharpName,
			"csharpTypeName":   toCSharpTypeName,
			"csharpMethodName": toCSharpMethodName,
			"pascal":           toPascal,
			"camel":            toCamel,
			"join":             strings.Join,
			"trim":             strings.TrimSpace,
			"lower":            strings.ToLower,
			"upper":            strings.ToUpper,
			"indent":           indent,
			"hasPrefix":        strings.HasPrefix,
			"hasSuffix":        strings.HasSuffix,
			"listElem":         listElem,
			"mapElem":          mapElem,
			"isPrimitive":      isPrimitive,
			"add":              func(a, b int) int { return a + b },
			"sub":              func(a, b int) int { return a - b },
			"len":              func(s interface{}) int { return lenHelper(s) },
		}).
		ParseFS(templateFS, "templates/*.cs.tmpl", "templates/*.csproj.tmpl")
	if err != nil {
		return nil, fmt.Errorf("sdkcsharp: parse templates: %w", err)
	}

	files := make([]*sdk.File, 0, 6)

	// Generate each file from its template
	templates := []struct {
		name string
		path string
	}{
		{"project.csproj.tmpl", m.PackageName + ".csproj"},
		{"Client.cs.tmpl", "src/" + m.Service.Sanitized + "Client.cs"},
		{"Types.cs.tmpl", "src/Models/Types.cs"},
		{"Resources.cs.tmpl", "src/Resources/Resources.cs"},
		{"Streaming.cs.tmpl", "src/Streaming.cs"},
		{"Exceptions.cs.tmpl", "src/Exceptions.cs"},
	}

	for _, t := range templates {
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, t.name, m); err != nil {
			return nil, fmt.Errorf("sdkcsharp: execute template %s: %w", t.name, err)
		}
		files = append(files, &sdk.File{Path: t.path, Content: out.String()})
	}

	return files, nil
}

type model struct {
	Namespace       string
	PackageName     string
	Version         string
	TargetFramework string
	Nullable        string

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
	CSharpName  string
	Description string
	Kind        contract.TypeKind

	Fields   []fieldModel
	Elem     string
	Tag      string
	Variants []variantModel

	// For struct types that are union variants
	IsUnionVariant  bool
	UnionBase       string // Base class name (e.g., "ContentBlock")
	UnionTag        string // Tag property name (e.g., "Type")
	UnionTagValue   string // Tag value (e.g., "text")
}

type fieldModel struct {
	Name        string
	CSharpName  string
	JSONName    string
	Description string
	CSharpType  string

	Optional   bool
	Nullable   bool
	IsRequired bool
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
	CSharpName  string
	ClassName   string
	Description string
}

type resourceModel struct {
	Name        string
	CSharpName  string
	ClassName   string
	Description string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	CSharpName  string
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

	// Namespace (PascalCase.Sdk for C#)
	if cfg != nil && cfg.Namespace != "" {
		m.Namespace = cfg.Namespace
	} else {
		ns := toPascal(sanitizeIdent(svc.Name))
		if ns == "" {
			ns = "Sdk"
		} else {
			ns += ".Sdk"
		}
		m.Namespace = ns
	}

	// Package name
	if cfg != nil && cfg.PackageName != "" {
		m.PackageName = cfg.PackageName
	} else {
		m.PackageName = m.Namespace
	}

	// Version
	if cfg != nil && cfg.Version != "" {
		m.Version = cfg.Version
	} else {
		m.Version = "0.0.0"
	}

	// Target framework
	if cfg != nil && cfg.TargetFramework != "" {
		m.TargetFramework = cfg.TargetFramework
	} else {
		m.TargetFramework = "net8.0"
	}

	// Nullable
	if cfg != nil && !cfg.Nullable {
		m.Nullable = "disable"
	} else {
		m.Nullable = "enable"
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

	// Build union variant lookup: maps variant type name -> union info
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
					Base:     toCSharpTypeName(t.Name),
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
			CSharpName:  toCSharpTypeName(t.Name),
			Description: t.Description,
			Kind:        t.Kind,
			Tag:         t.Tag,
		}

		// Check if this type is a variant of a union
		if info, ok := variantToUnion[t.Name]; ok {
			tm.IsUnionVariant = true
			tm.UnionBase = info.Base
			tm.UnionTag = toPascal(info.Tag)
			tm.UnionTagValue = info.TagValue
		}

		switch t.Kind {
		case contract.KindStruct:
			for _, f := range t.Fields {
				isOptional := f.Optional || f.Nullable
				fm := fieldModel{
					Name:        f.Name,
					CSharpName:  toCSharpName(f.Name),
					JSONName:    f.Name,
					Description: f.Description,
					CSharpType:  csharpType(typeByName, f.Type, f.Optional, f.Nullable),
					Optional:    f.Optional,
					Nullable:    f.Nullable,
					IsRequired:  !isOptional,
					Const:       f.Const,
				}

				// Handle enum values
				for _, e := range f.Enum {
					fm.Enum = append(fm.Enum, enumValue{
						Name:  toPascal(sanitizeIdent(e)),
						Value: e,
					})
				}

				tm.Fields = append(tm.Fields, fm)
			}

		case contract.KindSlice:
			tm.Elem = csharpType(typeByName, t.Elem, false, false)

		case contract.KindMap:
			tm.Elem = csharpType(typeByName, t.Elem, false, false)

		case contract.KindUnion:
			for _, v := range t.Variants {
				tm.Variants = append(tm.Variants, variantModel{
					Value:       v.Value,
					Type:        string(v.Type),
					CSharpName:  toCamel(string(v.Type)),
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
			CSharpName:  toPascal(r.Name),
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
				streamItem = toCSharpTypeName(strings.TrimSpace(string(mm.Stream.Item)))
			}

			rm.Methods = append(rm.Methods, methodModel{
				Name:        mm.Name,
				CSharpName:  toCSharpMethodName(mm.Name),
				Description: mm.Description,

				HasInput:    hasInput,
				HasOutput:   hasOutput,
				InputType:   toCSharpTypeName(string(mm.Input)),
				OutputType:  toCSharpTypeName(string(mm.Output)),
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

func csharpType(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "JsonElement"
	}

	base := baseCSharpType(typeByName, r)

	if optional || nullable {
		return base + "?"
	}
	return base
}

func baseCSharpType(typeByName map[string]*contract.Type, r string) string {
	// Check if it's a known type
	if _, ok := typeByName[r]; ok {
		return toCSharpTypeName(r)
	}

	switch r {
	case "string":
		return "string"
	case "bool", "boolean":
		return "bool"
	case "int":
		return "int"
	case "int8":
		return "sbyte"
	case "int16":
		return "short"
	case "int32":
		return "int"
	case "int64":
		return "long"
	case "uint":
		return "uint"
	case "uint8":
		return "byte"
	case "uint16":
		return "ushort"
	case "uint32":
		return "uint"
	case "uint64":
		return "ulong"
	case "float32":
		return "float"
	case "float64":
		return "double"
	case "time.Time":
		return "DateTimeOffset"
	case "json.RawMessage":
		return "JsonElement"
	case "any", "interface{}":
		return "JsonElement"
	}

	// Handle slice types
	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		return "IReadOnlyList<" + baseCSharpType(typeByName, elem) + ">"
	}

	// Handle map types
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		return "IReadOnlyDictionary<string, " + baseCSharpType(typeByName, elem) + ">"
	}

	return "JsonElement"
}

// toCSharpName converts a string to PascalCase for C# properties.
func toCSharpName(s string) string {
	if s == "" {
		return ""
	}

	result := toPascal(s)

	// Check for reserved words
	if isCSharpReserved(strings.ToLower(result)) {
		return "@" + result
	}

	return result
}

// toCSharpTypeName converts a string to PascalCase for C# types.
func toCSharpTypeName(s string) string {
	if s == "" {
		return ""
	}

	return toPascal(s)
}

// toCSharpMethodName converts a string to PascalCase + Async for C# methods.
func toCSharpMethodName(s string) string {
	if s == "" {
		return ""
	}

	result := toPascal(s)

	// Check for reserved words
	if isCSharpReserved(strings.ToLower(result)) {
		return "@" + result + "Async"
	}

	return result + "Async"
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

// isCSharpReserved checks if a name is a C# reserved word.
func isCSharpReserved(s string) bool {
	reserved := map[string]bool{
		"abstract": true, "as": true, "base": true, "bool": true,
		"break": true, "byte": true, "case": true, "catch": true,
		"char": true, "checked": true, "class": true, "const": true,
		"continue": true, "decimal": true, "default": true, "delegate": true,
		"do": true, "double": true, "else": true, "enum": true,
		"event": true, "explicit": true, "extern": true, "false": true,
		"finally": true, "fixed": true, "float": true, "for": true,
		"foreach": true, "goto": true, "if": true, "implicit": true,
		"in": true, "int": true, "interface": true, "internal": true,
		"is": true, "lock": true, "long": true, "namespace": true,
		"new": true, "null": true, "object": true, "operator": true,
		"out": true, "override": true, "params": true, "private": true,
		"protected": true, "public": true, "readonly": true, "ref": true,
		"return": true, "sbyte": true, "sealed": true, "short": true,
		"sizeof": true, "stackalloc": true, "static": true, "string": true,
		"struct": true, "switch": true, "this": true, "throw": true,
		"true": true, "try": true, "typeof": true, "uint": true,
		"ulong": true, "unchecked": true, "unsafe": true, "ushort": true,
		"using": true, "virtual": true, "void": true, "volatile": true,
		"while": true,
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

// csharpQuote returns a quoted C# string literal.
func csharpQuote(s string) string {
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
			b.WriteRune(r)
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

// listElem extracts the element type from an IReadOnlyList<T> type string.
func listElem(s string) string {
	if strings.HasPrefix(s, "IReadOnlyList<") && strings.HasSuffix(s, ">") {
		return strings.TrimSuffix(strings.TrimPrefix(s, "IReadOnlyList<"), ">")
	}
	return s
}

// mapElem extracts the value type from an IReadOnlyDictionary<string, T> type string.
func mapElem(s string) string {
	if strings.HasPrefix(s, "IReadOnlyDictionary<string, ") && strings.HasSuffix(s, ">") {
		return strings.TrimSuffix(strings.TrimPrefix(s, "IReadOnlyDictionary<string, "), ">")
	}
	return s
}

// isPrimitive checks if a type is a C# primitive that doesn't need custom serialization.
func isPrimitive(s string) bool {
	s = strings.TrimSuffix(s, "?")
	primitives := map[string]bool{
		"string": true, "int": true, "long": true, "short": true,
		"byte": true, "sbyte": true, "uint": true, "ulong": true,
		"ushort": true, "float": true, "double": true, "decimal": true,
		"bool": true, "char": true, "object": true, "JsonElement": true,
		"DateTimeOffset": true, "DateTime": true, "TimeSpan": true,
		"Guid": true,
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
