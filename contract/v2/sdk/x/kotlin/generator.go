// Package sdkkotlin generates typed Kotlin SDK clients from contract.Service.
package sdkkotlin

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

//go:embed templates/*.kt.tmpl templates/*.kts.tmpl
var templateFS embed.FS

// Config controls Kotlin SDK generation.
type Config struct {
	// Package is the Kotlin package name.
	// Default: sanitized lowercase service name.
	Package string

	// Version is the package version for build.gradle.kts.
	// Default: "0.0.0".
	Version string

	// GroupId is the Maven group ID.
	// Default: "com.example".
	GroupId string
}

// Generate produces a set of generated files for a typed Kotlin SDK client.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkkotlin: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("sdkkotlin").
		Funcs(template.FuncMap{
			"kotlinQuote":    kotlinQuote,
			"kotlinString":   kotlinQuote,
			"kotlinName":     toKotlinName,
			"kotlinTypeName": toKotlinTypeName,
			"camel":          toCamel,
			"pascal":         toPascal,
			"httpMethod":     toHTTPMethodName,
			"upper":          strings.ToUpper,
			"join":           strings.Join,
			"trim":           strings.TrimSpace,
			"lower":          strings.ToLower,
			"indent":         indent,
			"hasPrefix":      strings.HasPrefix,
			"add":            func(a, b int) int { return a + b },
			"sub":            func(a, b int) int { return a - b },
			"len":            func(s interface{}) int { return lenHelper(s) },
		}).
		ParseFS(templateFS, "templates/*.kt.tmpl", "templates/*.kts.tmpl")
	if err != nil {
		return nil, fmt.Errorf("sdkkotlin: parse templates: %w", err)
	}

	files := make([]*sdk.File, 0, 6)

	// Build package path
	pkgPath := strings.ReplaceAll(m.Package, ".", "/")

	// Generate each file from its template
	templates := []struct {
		name string
		path string
	}{
		{"build.gradle.kts.tmpl", "build.gradle.kts"},
		{"Client.kt.tmpl", "src/main/kotlin/" + pkgPath + "/Client.kt"},
		{"Types.kt.tmpl", "src/main/kotlin/" + pkgPath + "/Types.kt"},
		{"Resources.kt.tmpl", "src/main/kotlin/" + pkgPath + "/Resources.kt"},
		{"Streaming.kt.tmpl", "src/main/kotlin/" + pkgPath + "/Streaming.kt"},
		{"Errors.kt.tmpl", "src/main/kotlin/" + pkgPath + "/Errors.kt"},
	}

	for _, t := range templates {
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, t.name, m); err != nil {
			return nil, fmt.Errorf("sdkkotlin: execute template %s: %w", t.name, err)
		}
		files = append(files, &sdk.File{Path: t.path, Content: out.String()})
	}

	return files, nil
}

type model struct {
	Package string
	GroupId string
	Version string

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
	KotlinName  string
	Description string
	Kind        contract.TypeKind

	Fields   []fieldModel
	Elem     string
	Tag      string
	Variants []variantModel
}

type fieldModel struct {
	Name        string
	KotlinName  string
	JSONName    string
	Description string
	KotlinType  string

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
	KotlinName  string
	Description string
}

type resourceModel struct {
	Name        string
	KotlinName  string
	ClassName   string
	Description string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	KotlinName  string
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
		p := strings.ToLower(sanitizeIdent(svc.Name))
		if p == "" {
			p = "sdk"
		}
		m.Package = "com.example." + p
	}

	// GroupId
	if cfg != nil && cfg.GroupId != "" {
		m.GroupId = cfg.GroupId
	} else {
		m.GroupId = "com.example"
	}

	// Version
	if cfg != nil && cfg.Version != "" {
		m.Version = cfg.Version
	} else {
		m.Version = "0.0.0"
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
			KotlinName:  toKotlinTypeName(t.Name),
			Description: t.Description,
			Kind:        t.Kind,
			Tag:         t.Tag,
		}

		switch t.Kind {
		case contract.KindStruct:
			for _, f := range t.Fields {
				fm := fieldModel{
					Name:        f.Name,
					KotlinName:  toKotlinName(f.Name),
					JSONName:    f.Name,
					Description: f.Description,
					KotlinType:  kotlinType(typeByName, f.Type, f.Optional, f.Nullable),
					Optional:    f.Optional,
					Nullable:    f.Nullable,
					Const:       f.Const,
				}

				// Handle enum values
				for _, e := range f.Enum {
					fm.Enum = append(fm.Enum, enumValue{
						Name:  toEnumCase(e),
						Value: e,
					})
				}

				tm.Fields = append(tm.Fields, fm)
			}

		case contract.KindSlice:
			tm.Elem = kotlinType(typeByName, t.Elem, false, false)

		case contract.KindMap:
			tm.Elem = kotlinType(typeByName, t.Elem, false, false)

		case contract.KindUnion:
			for _, v := range t.Variants {
				tm.Variants = append(tm.Variants, variantModel{
					Value:       v.Value,
					Type:        string(v.Type),
					KotlinName:  toPascal(string(v.Type)),
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
			KotlinName:  toCamel(r.Name),
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
				streamItem = toKotlinTypeName(strings.TrimSpace(string(mm.Stream.Item)))
			}

			rm.Methods = append(rm.Methods, methodModel{
				Name:        mm.Name,
				KotlinName:  toCamel(mm.Name),
				Description: mm.Description,

				HasInput:    hasInput,
				HasOutput:   hasOutput,
				InputType:   toKotlinTypeName(string(mm.Input)),
				OutputType:  toKotlinTypeName(string(mm.Output)),
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

func kotlinType(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "JsonElement"
	}

	base := baseKotlinType(typeByName, r)

	if optional || nullable {
		return base + "?"
	}
	return base
}

func baseKotlinType(typeByName map[string]*contract.Type, r string) string {
	// Check if it's a known type
	if _, ok := typeByName[r]; ok {
		return toKotlinTypeName(r)
	}

	switch r {
	case "string":
		return "String"
	case "bool", "boolean":
		return "Boolean"
	case "int":
		return "Int"
	case "int8":
		return "Byte"
	case "int16":
		return "Short"
	case "int32":
		return "Int"
	case "int64":
		return "Long"
	case "uint":
		return "UInt"
	case "uint8":
		return "UByte"
	case "uint16":
		return "UShort"
	case "uint32":
		return "UInt"
	case "uint64":
		return "ULong"
	case "float32":
		return "Float"
	case "float64":
		return "Double"
	case "time.Time":
		return "Instant"
	case "json.RawMessage":
		return "JsonElement"
	case "any", "interface{}":
		return "JsonElement"
	}

	// Handle slice types
	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		return "List<" + baseKotlinType(typeByName, elem) + ">"
	}

	// Handle map types
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		return "Map<String, " + baseKotlinType(typeByName, elem) + ">"
	}

	return "JsonElement"
}

// toKotlinName converts a string to lowerCamelCase for Kotlin properties/methods.
func toKotlinName(s string) string {
	if s == "" {
		return ""
	}

	// Check for reserved words
	if isKotlinReserved(s) {
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
	if isKotlinReserved(result) {
		return "`" + result + "`"
	}
	return result
}

// toKotlinTypeName converts a string to UpperCamelCase for Kotlin types.
func toKotlinTypeName(s string) string {
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

// toEnumCase converts a value to SCREAMING_SNAKE_CASE for Kotlin enum constants.
func toEnumCase(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	prevWasUpper := false

	for i, r := range s {
		if r == '_' || r == '-' || r == '.' || r == ' ' {
			b.WriteRune('_')
			prevWasUpper = false
			continue
		}

		if unicode.IsUpper(r) {
			if i > 0 && !prevWasUpper {
				b.WriteRune('_')
			}
			b.WriteRune(r)
			prevWasUpper = true
		} else {
			b.WriteRune(unicode.ToUpper(r))
			prevWasUpper = false
		}
	}

	return b.String()
}

// isKotlinReserved checks if a name is a Kotlin reserved word.
func isKotlinReserved(s string) bool {
	reserved := map[string]bool{
		"as": true, "break": true, "class": true, "continue": true,
		"do": true, "else": true, "false": true, "for": true,
		"fun": true, "if": true, "in": true, "interface": true,
		"is": true, "null": true, "object": true, "package": true,
		"return": true, "super": true, "this": true, "throw": true,
		"true": true, "try": true, "typealias": true, "typeof": true,
		"val": true, "var": true, "when": true, "while": true,
		// Soft keywords that are reserved in certain contexts
		"by": true, "catch": true, "constructor": true, "delegate": true,
		"dynamic": true, "field": true, "file": true, "finally": true,
		"get": true, "import": true, "init": true, "param": true,
		"property": true, "receiver": true, "set": true, "setparam": true,
		"value": true, "where": true,
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

// kotlinQuote returns a quoted Kotlin string literal.
func kotlinQuote(s string) string {
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

// toHTTPMethodName converts HTTP method to Ktor HttpMethod constant name.
// e.g., "GET" -> "Get", "POST" -> "Post", "DELETE" -> "Delete"
func toHTTPMethodName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "Post"
	}
	// Convert to proper case: first letter uppercase, rest lowercase
	return strings.ToUpper(string(s[0])) + strings.ToLower(s[1:])
}
