// Package sdkjava generates typed Java SDK clients from contract.Service.
package sdkjava

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

//go:embed templates/*.java.tmpl templates/*.xml.tmpl
var templateFS embed.FS

// Config controls Java SDK generation.
type Config struct {
	// Package is the Java package name.
	// Default: "com.example.{servicename}".
	Package string

	// GroupId is the Maven group ID.
	// Default: "com.example".
	GroupId string

	// ArtifactId is the Maven artifact ID.
	// Default: "{servicename}-sdk".
	ArtifactId string

	// Version is the package version.
	// Default: "0.0.0".
	Version string

	// JavaVersion is the target Java version (11, 17, 21).
	// Default: 11.
	JavaVersion int
}

// Generate produces a set of generated files for a typed Java SDK client.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkjava: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("sdkjava").
		Funcs(template.FuncMap{
			"javaQuote":      javaQuote,
			"javaString":     javaQuote,
			"javaName":       toJavaName,
			"javaTypeName":   toJavaTypeName,
			"javaMethodName": toJavaMethodName,
			"javaConstant":   toJavaConstant,
			"pascal":         toPascal,
			"camel":          toCamel,
			"upper":          strings.ToUpper,
			"lower":          strings.ToLower,
			"join":           strings.Join,
			"trim":           strings.TrimSpace,
			"indent":         indent,
			"hasPrefix":      strings.HasPrefix,
			"hasSuffix":      strings.HasSuffix,
			"httpMethod":     toHTTPMethodName,
			"add":            func(a, b int) int { return a + b },
			"sub":            func(a, b int) int { return a - b },
			"len":            func(s interface{}) int { return lenHelper(s) },
			"eq":             func(a, b string) bool { return a == b },
		}).
		ParseFS(templateFS, "templates/*.java.tmpl", "templates/*.xml.tmpl")
	if err != nil {
		return nil, fmt.Errorf("sdkjava: parse templates: %w", err)
	}

	files := make([]*sdk.File, 0, 10)

	// Build package path
	pkgPath := strings.ReplaceAll(m.Package, ".", "/")

	// Generate each file from its template
	templates := []struct {
		name string
		path string
	}{
		{"pom.xml.tmpl", "pom.xml"},
		{"Client.java.tmpl", "src/main/java/" + pkgPath + "/" + m.Service.Sanitized + "Client.java"},
		{"ClientOptions.java.tmpl", "src/main/java/" + pkgPath + "/ClientOptions.java"},
		{"AuthMode.java.tmpl", "src/main/java/" + pkgPath + "/AuthMode.java"},
		{"Types.java.tmpl", "src/main/java/" + pkgPath + "/model/Types.java"},
		{"Resources.java.tmpl", "src/main/java/" + pkgPath + "/resource/Resources.java"},
		{"HttpClientWrapper.java.tmpl", "src/main/java/" + pkgPath + "/internal/HttpClientWrapper.java"},
		{"SSEReader.java.tmpl", "src/main/java/" + pkgPath + "/internal/SSEReader.java"},
		{"Exceptions.java.tmpl", "src/main/java/" + pkgPath + "/exception/Exceptions.java"},
	}

	for _, t := range templates {
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, t.name, m); err != nil {
			return nil, fmt.Errorf("sdkjava: execute template %s: %w", t.name, err)
		}
		files = append(files, &sdk.File{Path: t.path, Content: out.String()})
	}

	return files, nil
}

type model struct {
	Package    string
	GroupId    string
	ArtifactId string
	Version    string

	JavaVersion int

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
	Resources []resourceModel
}

type kv struct {
	K string
	V string
}

type typeModel struct {
	Name        string
	JavaName    string
	Description string
	Kind        string // "struct", "slice", "map", "union"

	Fields   []fieldModel
	Elem     string
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
	JavaName    string
	GetterName  string
	SetterName  string
	JSONName    string
	Description string
	JavaType    string
	BoxedType   string

	Optional   bool
	Nullable   bool
	IsRequired bool
	HasEnum    bool
	EnumType   string
	Const      string
}

type enumModel struct {
	Name        string
	JavaName    string
	Description string
	Values      []enumValue
}

type enumValue struct {
	Name  string
	Value string
}

type variantModel struct {
	Value       string
	Type        string
	JavaName    string
	ClassName   string
	Description string
}

type resourceModel struct {
	Name        string
	JavaName    string
	ClassName   string
	Description string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	JavaName    string
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

	// Service name
	serviceName := sanitizeIdent(svc.Name)
	if serviceName == "" {
		serviceName = "Service"
	}

	// Package name
	if cfg != nil && cfg.Package != "" {
		m.Package = cfg.Package
	} else {
		m.Package = "com.example." + strings.ToLower(serviceName)
	}

	// GroupId
	if cfg != nil && cfg.GroupId != "" {
		m.GroupId = cfg.GroupId
	} else {
		m.GroupId = "com.example"
	}

	// ArtifactId
	if cfg != nil && cfg.ArtifactId != "" {
		m.ArtifactId = cfg.ArtifactId
	} else {
		m.ArtifactId = strings.ToLower(serviceName) + "-sdk"
	}

	// Version
	if cfg != nil && cfg.Version != "" {
		m.Version = cfg.Version
	} else {
		m.Version = "0.0.0"
	}

	// Java version
	if cfg != nil && cfg.JavaVersion > 0 {
		m.JavaVersion = cfg.JavaVersion
	} else {
		m.JavaVersion = 11
	}

	// Service info
	m.Service.Name = svc.Name
	m.Service.Sanitized = toPascal(serviceName)
	m.Service.Description = svc.Description

	// Client defaults
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
					Base:     toJavaTypeName(t.Name),
					Tag:      t.Tag,
					TagValue: v.Value,
				}
			}
		}
	}

	// Collect enums from fields
	enumSet := make(map[string]enumModel)

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
			JavaName:    toJavaTypeName(t.Name),
			Description: t.Description,
			Kind:        string(t.Kind),
			Tag:         t.Tag,
		}

		// Check if this type is a variant of a union
		if info, ok := variantToUnion[t.Name]; ok {
			tm.IsUnionVariant = true
			tm.UnionBase = info.Base
			tm.UnionTag = toCamel(info.Tag)
			tm.UnionTagValue = info.TagValue
		}

		switch t.Kind {
		case contract.KindStruct:
			for _, f := range t.Fields {
				isOptional := f.Optional || f.Nullable
				javaType := javaType(typeByName, f.Type, f.Optional, f.Nullable)
				fm := fieldModel{
					Name:        f.Name,
					JavaName:    toJavaName(f.Name),
					GetterName:  toGetterName(f.Name, javaType),
					SetterName:  toSetterName(f.Name),
					JSONName:    f.Name,
					Description: f.Description,
					JavaType:    javaType,
					BoxedType:   toBoxedType(javaType),
					Optional:    f.Optional,
					Nullable:    f.Nullable,
					IsRequired:  !isOptional,
					Const:       f.Const,
				}

				// Handle enum values
				if len(f.Enum) > 0 {
					enumName := tm.JavaName + toPascal(f.Name)
					fm.HasEnum = true
					fm.EnumType = enumName

					// Create enum model
					em := enumModel{
						Name:     enumName,
						JavaName: enumName,
					}
					for _, e := range f.Enum {
						em.Values = append(em.Values, enumValue{
							Name:  toJavaConstant(e),
							Value: e,
						})
					}
					enumSet[enumName] = em
				}

				tm.Fields = append(tm.Fields, fm)
			}

		case contract.KindSlice:
			tm.Elem = javaType(typeByName, t.Elem, false, false)

		case contract.KindMap:
			tm.Elem = javaType(typeByName, t.Elem, false, false)

		case contract.KindUnion:
			for _, v := range t.Variants {
				tm.Variants = append(tm.Variants, variantModel{
					Value:       v.Value,
					Type:        string(v.Type),
					JavaName:    toCamel(string(v.Type)),
					ClassName:   toPascal(string(v.Type)),
					Description: v.Description,
				})
			}
		}

		m.Types = append(m.Types, tm)
	}

	// Convert enum set to slice
	enumNames := make([]string, 0, len(enumSet))
	for name := range enumSet {
		enumNames = append(enumNames, name)
	}
	sort.Strings(enumNames)
	for _, name := range enumNames {
		m.Enums = append(m.Enums, enumSet[name])
	}

	// Build resources
	for _, r := range svc.Resources {
		if r == nil {
			continue
		}
		rm := resourceModel{
			Name:        r.Name,
			JavaName:    toCamel(r.Name),
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
				streamItem = toJavaTypeName(strings.TrimSpace(string(mm.Stream.Item)))
			}

			rm.Methods = append(rm.Methods, methodModel{
				Name:        mm.Name,
				JavaName:    toJavaMethodName(mm.Name),
				Description: mm.Description,

				HasInput:    hasInput,
				HasOutput:   hasOutput,
				InputType:   toJavaTypeName(string(mm.Input)),
				OutputType:  toJavaTypeName(string(mm.Output)),
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

func javaType(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "JsonNode"
	}

	base := baseJavaType(typeByName, r)

	// For optional/nullable, we use the boxed type
	if optional || nullable {
		return toBoxedType(base)
	}
	return base
}

func baseJavaType(typeByName map[string]*contract.Type, r string) string {
	// Check if it's a known type
	if _, ok := typeByName[r]; ok {
		return toJavaTypeName(r)
	}

	switch r {
	case "string":
		return "String"
	case "bool", "boolean":
		return "boolean"
	case "int":
		return "int"
	case "int8":
		return "byte"
	case "int16":
		return "short"
	case "int32":
		return "int"
	case "int64":
		return "long"
	case "uint":
		return "int"
	case "uint8":
		return "short"
	case "uint16":
		return "int"
	case "uint32":
		return "long"
	case "uint64":
		return "long"
	case "float32":
		return "float"
	case "float64":
		return "double"
	case "time.Time":
		return "Instant"
	case "json.RawMessage":
		return "JsonNode"
	case "any", "interface{}":
		return "JsonNode"
	}

	// Handle slice types
	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		return "List<" + toBoxedType(baseJavaType(typeByName, elem)) + ">"
	}

	// Handle map types
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		return "Map<String, " + toBoxedType(baseJavaType(typeByName, elem)) + ">"
	}

	return "JsonNode"
}

func toBoxedType(t string) string {
	switch t {
	case "boolean":
		return "Boolean"
	case "byte":
		return "Byte"
	case "short":
		return "Short"
	case "int":
		return "Integer"
	case "long":
		return "Long"
	case "float":
		return "Float"
	case "double":
		return "Double"
	case "char":
		return "Character"
	default:
		return t
	}
}

// toJavaName converts a string to lowerCamelCase for Java fields/methods.
func toJavaName(s string) string {
	if s == "" {
		return ""
	}

	result := toCamel(s)

	// Check for reserved words
	if isJavaReserved(result) {
		return "_" + result
	}

	return result
}

// toJavaTypeName converts a string to UpperCamelCase for Java types.
func toJavaTypeName(s string) string {
	if s == "" {
		return ""
	}

	return toPascal(s)
}

// toJavaMethodName converts a string to lowerCamelCase for Java methods.
func toJavaMethodName(s string) string {
	if s == "" {
		return ""
	}

	result := toCamel(s)

	// Check for reserved words
	if isJavaReserved(result) {
		return "_" + result
	}

	return result
}

// toGetterName creates a getter name for a field.
func toGetterName(s string, javaType string) string {
	if s == "" {
		return ""
	}

	prefix := "get"
	if javaType == "boolean" || javaType == "Boolean" {
		prefix = "is"
	}

	return prefix + toPascal(s)
}

// toSetterName creates a setter name for a field.
func toSetterName(s string) string {
	if s == "" {
		return ""
	}

	return "set" + toPascal(s)
}

// toJavaConstant converts a value to SCREAMING_SNAKE_CASE for Java constants.
func toJavaConstant(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	prevWasUpper := false
	prevWasDelimiter := false

	for i, r := range s {
		if r == '_' || r == '-' || r == '.' || r == ' ' {
			if !prevWasDelimiter && b.Len() > 0 {
				b.WriteRune('_')
			}
			prevWasDelimiter = true
			prevWasUpper = false
			continue
		}

		if unicode.IsUpper(r) {
			if i > 0 && !prevWasUpper && !prevWasDelimiter {
				b.WriteRune('_')
			}
			b.WriteRune(r)
			prevWasUpper = true
		} else {
			b.WriteRune(unicode.ToUpper(r))
			prevWasUpper = false
		}
		prevWasDelimiter = false
	}

	result := b.String()

	// If it starts with a number, prefix with underscore
	if len(result) > 0 && unicode.IsDigit(rune(result[0])) {
		result = "_" + result
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

// isJavaReserved checks if a name is a Java reserved word.
func isJavaReserved(s string) bool {
	reserved := map[string]bool{
		"abstract": true, "assert": true, "boolean": true, "break": true,
		"byte": true, "case": true, "catch": true, "char": true,
		"class": true, "const": true, "continue": true, "default": true,
		"do": true, "double": true, "else": true, "enum": true,
		"extends": true, "final": true, "finally": true, "float": true,
		"for": true, "goto": true, "if": true, "implements": true,
		"import": true, "instanceof": true, "int": true, "interface": true,
		"long": true, "native": true, "new": true, "package": true,
		"private": true, "protected": true, "public": true, "return": true,
		"short": true, "static": true, "strictfp": true, "super": true,
		"switch": true, "synchronized": true, "this": true, "throw": true,
		"throws": true, "transient": true, "try": true, "void": true,
		"volatile": true, "while": true,
		// Also reserved: true, false, null (but these are literals)
		"true": true, "false": true, "null": true,
		// Common problematic names
		"object": true, "string": true,
	}
	return reserved[strings.ToLower(s)]
}

// sanitizeIdent removes invalid characters from an identifier.
func sanitizeIdent(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsLetter(r) || r == '_' || (i > 0 && unicode.IsDigit(r)) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// javaQuote returns a quoted Java string literal.
func javaQuote(s string) string {
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

// toHTTPMethodName returns the HTTP method for use in Java code.
func toHTTPMethodName(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
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
	case []enumModel:
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
