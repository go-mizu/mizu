// Package sdkfsharp generates typed F# SDK clients from contract.Service.
package sdkfsharp

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

//go:embed templates/*.fs.tmpl templates/*.fsproj.tmpl
var templateFS embed.FS

// Config controls F# SDK generation.
type Config struct {
	// Namespace is the F# namespace for generated code.
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
}

// Generate produces a set of generated files for a typed F# SDK client.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkfsharp: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("sdkfsharp").
		Funcs(template.FuncMap{
			"fsharpQuote":      fsharpQuote,
			"fsharpString":     fsharpQuote,
			"fsharpName":       toFSharpName,
			"fsharpTypeName":   toFSharpTypeName,
			"fsharpMethodName": toFSharpMethodName,
			"fsharpFieldName":  toFSharpFieldName,
			"pascal":           toPascal,
			"camel":            toCamel,
			"httpMethod":       toHttpMethod,
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
			"isOption":         isOptionType,
			"add":              func(a, b int) int { return a + b },
			"sub":              func(a, b int) int { return a - b },
			"len":              func(s interface{}) int { return lenHelper(s) },
		}).
		ParseFS(templateFS, "templates/*.fs.tmpl", "templates/*.fsproj.tmpl")
	if err != nil {
		return nil, fmt.Errorf("sdkfsharp: parse templates: %w", err)
	}

	files := make([]*sdk.File, 0, 7)

	// Generate each file from its template
	// Note: F# requires files in compilation order
	templates := []struct {
		name string
		path string
	}{
		{"project.fsproj.tmpl", m.PackageName + ".fsproj"},
		{"Types.fs.tmpl", "src/Types.fs"},
		{"Errors.fs.tmpl", "src/Errors.fs"},
		{"Http.fs.tmpl", "src/Http.fs"},
		{"Streaming.fs.tmpl", "src/Streaming.fs"},
		{"Resources.fs.tmpl", "src/Resources.fs"},
		{"Client.fs.tmpl", "src/Client.fs"},
	}

	for _, t := range templates {
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, t.name, m); err != nil {
			return nil, fmt.Errorf("sdkfsharp: execute template %s: %w", t.name, err)
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
	Enums     []enumModel
	Unions    []unionModel
	Resources []resourceModel
}

type kv struct {
	K string
	V string
}

type typeModel struct {
	Name        string
	FSharpName  string
	Description string
	Kind        string // "struct", "slice", "map"

	Fields []fieldModel
	Elem   string
}

type fieldModel struct {
	Name        string
	FSharpName  string
	JSONName    string
	Description string
	FSharpType  string

	Optional bool
	Nullable bool
	Enum     []enumValue
	Const    string
}

type enumModel struct {
	Name       string
	FSharpName string
	Values     []enumValue
}

type enumValue struct {
	Name  string
	Value string
}

type unionModel struct {
	Name        string
	FSharpName  string
	Description string
	Tag         string
	Variants    []variantModel
}

type variantModel struct {
	Value       string
	Type        string
	FSharpName  string
	Description string
	Fields      []fieldModel
}

type resourceModel struct {
	Name        string
	FSharpName  string
	TypeName    string
	Description string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	FSharpName  string
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

	// Namespace (PascalCase.Sdk for F#)
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

	// Collect enums from field constraints
	enumsByKey := make(map[string]enumModel)

	// Build types, enums, and unions
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

		switch t.Kind {
		case contract.KindStruct:
			tm := typeModel{
				Name:        t.Name,
				FSharpName:  toFSharpTypeName(t.Name),
				Description: t.Description,
				Kind:        "struct",
			}

			for _, f := range t.Fields {
				isOptional := f.Optional || f.Nullable
				fm := fieldModel{
					Name:        f.Name,
					FSharpName:  toFSharpFieldName(f.Name),
					JSONName:    f.Name,
					Description: f.Description,
					FSharpType:  fsharpType(typeByName, f.Type, f.Optional, f.Nullable),
					Optional:    f.Optional,
					Nullable:    f.Nullable,
					Const:       f.Const,
				}

				// Handle enum values - collect them for enum generation
				if len(f.Enum) > 0 {
					enumName := toPascal(t.Name) + toPascal(f.Name)
					enumKey := strings.Join(f.Enum, "|")

					if _, exists := enumsByKey[enumKey]; !exists {
						em := enumModel{
							Name:       enumName,
							FSharpName: toFSharpTypeName(enumName),
						}
						for _, e := range f.Enum {
							em.Values = append(em.Values, enumValue{
								Name:  toPascal(sanitizeIdent(e)),
								Value: e,
							})
						}
						enumsByKey[enumKey] = em
						m.Enums = append(m.Enums, em)
					}

					// Update field type to use enum
					if isOptional {
						fm.FSharpType = enumName + " option"
					} else {
						fm.FSharpType = enumName
					}

					for _, e := range f.Enum {
						fm.Enum = append(fm.Enum, enumValue{
							Name:  toPascal(sanitizeIdent(e)),
							Value: e,
						})
					}
				}

				tm.Fields = append(tm.Fields, fm)
			}

			m.Types = append(m.Types, tm)

		case contract.KindSlice:
			tm := typeModel{
				Name:        t.Name,
				FSharpName:  toFSharpTypeName(t.Name),
				Description: t.Description,
				Kind:        "slice",
				Elem:        fsharpType(typeByName, t.Elem, false, false),
			}
			m.Types = append(m.Types, tm)

		case contract.KindMap:
			tm := typeModel{
				Name:        t.Name,
				FSharpName:  toFSharpTypeName(t.Name),
				Description: t.Description,
				Kind:        "map",
				Elem:        fsharpType(typeByName, t.Elem, false, false),
			}
			m.Types = append(m.Types, tm)

		case contract.KindUnion:
			um := unionModel{
				Name:        t.Name,
				FSharpName:  toFSharpTypeName(t.Name),
				Description: t.Description,
				Tag:         t.Tag,
			}

			for _, v := range t.Variants {
				vm := variantModel{
					Value:       v.Value,
					Type:        string(v.Type),
					FSharpName:  toPascal(string(v.Type)),
					Description: v.Description,
				}

				// If the variant type exists, get its fields
				if variantType, ok := typeByName[string(v.Type)]; ok && variantType.Kind == contract.KindStruct {
					for _, f := range variantType.Fields {
						// Skip the tag field
						if f.Name == t.Tag {
							continue
						}
						isOptional := f.Optional || f.Nullable
						vm.Fields = append(vm.Fields, fieldModel{
							Name:        f.Name,
							FSharpName:  toFSharpFieldName(f.Name),
							JSONName:    f.Name,
							Description: f.Description,
							FSharpType:  fsharpType(typeByName, f.Type, f.Optional, f.Nullable),
							Optional:    isOptional,
						})
					}
				}

				um.Variants = append(um.Variants, vm)
			}

			m.Unions = append(m.Unions, um)
		}
	}

	// Build resources
	for _, r := range svc.Resources {
		if r == nil {
			continue
		}
		rm := resourceModel{
			Name:        r.Name,
			FSharpName:  toPascal(r.Name),
			TypeName:    toPascal(r.Name) + "Resource",
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
				streamItem = toFSharpTypeName(strings.TrimSpace(string(mm.Stream.Item)))
			}

			rm.Methods = append(rm.Methods, methodModel{
				Name:        mm.Name,
				FSharpName:  toFSharpMethodName(mm.Name),
				Description: mm.Description,

				HasInput:    hasInput,
				HasOutput:   hasOutput,
				InputType:   toFSharpTypeName(string(mm.Input)),
				OutputType:  toFSharpTypeName(string(mm.Output)),
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

func fsharpType(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "JsonElement"
	}

	base := baseFSharpType(typeByName, r)

	if optional || nullable {
		return base + " option"
	}
	return base
}

func baseFSharpType(typeByName map[string]*contract.Type, r string) string {
	// Check if it's a known type
	if t, ok := typeByName[r]; ok {
		// If it's a union, use the union type directly
		if t.Kind == contract.KindUnion {
			return toFSharpTypeName(r)
		}
		return toFSharpTypeName(r)
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
		return "int16"
	case "int32":
		return "int"
	case "int64":
		return "int64"
	case "uint":
		return "uint32"
	case "uint8":
		return "byte"
	case "uint16":
		return "uint16"
	case "uint32":
		return "uint32"
	case "uint64":
		return "uint64"
	case "float32":
		return "float32"
	case "float64":
		return "float"
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
		return baseFSharpType(typeByName, elem) + " list"
	}

	// Handle map types
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		return "Map<string, " + baseFSharpType(typeByName, elem) + ">"
	}

	return "JsonElement"
}

// toFSharpName converts a string to PascalCase for F# identifiers.
func toFSharpName(s string) string {
	if s == "" {
		return ""
	}

	result := toPascal(s)

	// Check for reserved words
	if isFSharpReserved(strings.ToLower(result)) {
		return "``" + result + "``"
	}

	return result
}

// toFSharpTypeName converts a string to PascalCase for F# types.
func toFSharpTypeName(s string) string {
	if s == "" {
		return ""
	}

	return toPascal(s)
}

// toFSharpMethodName converts a string to PascalCase + Async for F# methods.
func toFSharpMethodName(s string) string {
	if s == "" {
		return ""
	}

	result := toPascal(s)

	// Check for reserved words
	if isFSharpReserved(strings.ToLower(result)) {
		return "``" + result + "Async``"
	}

	return result + "Async"
}

// toFSharpFieldName converts a string to PascalCase for F# record fields.
func toFSharpFieldName(s string) string {
	if s == "" {
		return ""
	}

	result := toPascal(s)

	// Check for reserved words
	if isFSharpReserved(strings.ToLower(result)) {
		return "``" + result + "``"
	}

	return result
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

// toHttpMethod converts an HTTP method to .NET HttpMethod format (e.g., "GET" -> "Get").
func toHttpMethod(s string) string {
	if s == "" {
		return "Post"
	}
	// Convert to proper casing: GET -> Get, POST -> Post, DELETE -> Delete
	s = strings.ToLower(s)
	return string(unicode.ToUpper(rune(s[0]))) + s[1:]
}

// isFSharpReserved checks if a name is an F# reserved word.
func isFSharpReserved(s string) bool {
	reserved := map[string]bool{
		"abstract": true, "and": true, "as": true, "assert": true,
		"base": true, "begin": true, "class": true, "default": true,
		"delegate": true, "do": true, "done": true, "downcast": true,
		"downto": true, "elif": true, "else": true, "end": true,
		"exception": true, "extern": true, "false": true, "finally": true,
		"fixed": true, "for": true, "fun": true, "function": true,
		"global": true, "if": true, "in": true, "inherit": true,
		"inline": true, "interface": true, "internal": true, "lazy": true,
		"let": true, "match": true, "member": true, "module": true,
		"mutable": true, "namespace": true, "new": true, "not": true,
		"null": true, "of": true, "open": true, "or": true,
		"override": true, "private": true, "public": true, "rec": true,
		"return": true, "select": true, "static": true, "struct": true,
		"then": true, "to": true, "true": true, "try": true,
		"type": true, "upcast": true, "use": true, "val": true,
		"void": true, "when": true, "while": true, "with": true,
		"yield": true, "asr": true, "land": true, "lor": true,
		"lsl": true, "lsr": true, "lxor": true, "mod": true,
		"sig": true,
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

// fsharpQuote returns a quoted F# string literal.
func fsharpQuote(s string) string {
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

// listElem extracts the element type from a list type string.
func listElem(s string) string {
	if strings.HasSuffix(s, " list") {
		return strings.TrimSuffix(s, " list")
	}
	return s
}

// mapElem extracts the value type from a Map<string, T> type string.
func mapElem(s string) string {
	if strings.HasPrefix(s, "Map<string, ") && strings.HasSuffix(s, ">") {
		return strings.TrimSuffix(strings.TrimPrefix(s, "Map<string, "), ">")
	}
	return s
}

// isPrimitive checks if a type is an F# primitive.
func isPrimitive(s string) bool {
	s = strings.TrimSuffix(s, " option")
	primitives := map[string]bool{
		"string": true, "int": true, "int64": true, "int16": true,
		"byte": true, "sbyte": true, "uint32": true, "uint64": true,
		"uint16": true, "float32": true, "float": true, "decimal": true,
		"bool": true, "char": true, "obj": true, "JsonElement": true,
		"DateTimeOffset": true, "DateTime": true, "TimeSpan": true,
		"Guid": true,
	}
	return primitives[s]
}

// isOptionType checks if a type is an F# option type.
func isOptionType(s string) bool {
	return strings.HasSuffix(s, " option")
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
	case []enumModel:
		return len(v)
	case []unionModel:
		return len(v)
	case []kv:
		return len(v)
	case []string:
		return len(v)
	default:
		return 0
	}
}
