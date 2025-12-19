// Package sdkscala generates typed Scala SDK clients from contract.Service.
package sdkscala

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

// Config controls Scala SDK generation.
type Config struct {
	// Package is the Scala package name.
	// Default: sanitized lowercase service name.
	Package string

	// Version is the artifact version.
	// Default: "0.0.0".
	Version string

	// Organization is the SBT organization.
	// Default: "com.example".
	Organization string

	// ArtifactId is the SBT artifact ID.
	// Default: kebab-case service name.
	ArtifactId string

	// ScalaVersion is the Scala version.
	// Default: "2.13.12".
	ScalaVersion string

	// Scala3 enables Scala 3 syntax.
	// Default: false.
	Scala3 bool
}

// Generate produces a set of generated files for a typed Scala SDK client.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkscala: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("sdkscala").
		Funcs(template.FuncMap{
			"scalaQuote":    scalaQuote,
			"scalaString":   scalaQuote,
			"scalaName":     toScalaName,
			"scalaTypeName": toScalaTypeName,
			"camel":         toCamel,
			"pascal":        toPascal,
			"kebab":         toKebab,
			"httpMethod":    toHTTPMethodName,
			"upper":         strings.ToUpper,
			"join":          strings.Join,
			"trim":          strings.TrimSpace,
			"lower":         strings.ToLower,
			"indent":        indent,
			"hasPrefix":     strings.HasPrefix,
			"add":           func(a, b int) int { return a + b },
			"sub":           func(a, b int) int { return a - b },
			"len":           func(s interface{}) int { return lenHelper(s) },
		}).
		ParseFS(templateFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("sdkscala: parse templates: %w", err)
	}

	files := make([]*sdk.File, 0, 7)

	// Build package path
	pkgPath := strings.ReplaceAll(m.Package, ".", "/")

	// Generate each file from its template
	templates := []struct {
		name string
		path string
	}{
		{"build.sbt.tmpl", "build.sbt"},
		{"build.properties.tmpl", "project/build.properties"},
		{"Client.scala.tmpl", "src/main/scala/" + pkgPath + "/Client.scala"},
		{"Types.scala.tmpl", "src/main/scala/" + pkgPath + "/Types.scala"},
		{"Resources.scala.tmpl", "src/main/scala/" + pkgPath + "/Resources.scala"},
		{"Streaming.scala.tmpl", "src/main/scala/" + pkgPath + "/Streaming.scala"},
		{"Errors.scala.tmpl", "src/main/scala/" + pkgPath + "/Errors.scala"},
	}

	for _, t := range templates {
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, t.name, m); err != nil {
			return nil, fmt.Errorf("sdkscala: execute template %s: %w", t.name, err)
		}
		files = append(files, &sdk.File{Path: t.path, Content: out.String()})
	}

	return files, nil
}

type model struct {
	Package      string
	Organization string
	ArtifactId   string
	Version      string
	ScalaVersion string
	Scala3       bool

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
	ScalaName   string
	Description string
	Kind        contract.TypeKind

	Fields   []fieldModel
	Elem     string
	Tag      string
	Variants []variantModel
}

type fieldModel struct {
	Name        string
	ScalaName   string
	JSONName    string
	Description string
	ScalaType   string

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
	ScalaName   string
	Description string
}

type resourceModel struct {
	Name        string
	ScalaName   string
	ClassName   string
	Description string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	ScalaName   string
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

	// Organization
	if cfg != nil && cfg.Organization != "" {
		m.Organization = cfg.Organization
	} else {
		m.Organization = "com.example"
	}

	// ArtifactId
	if cfg != nil && cfg.ArtifactId != "" {
		m.ArtifactId = cfg.ArtifactId
	} else {
		m.ArtifactId = toKebab(sanitizeIdent(svc.Name))
		if m.ArtifactId == "" {
			m.ArtifactId = "sdk"
		}
	}

	// Version
	if cfg != nil && cfg.Version != "" {
		m.Version = cfg.Version
	} else {
		m.Version = "0.0.0"
	}

	// ScalaVersion
	if cfg != nil && cfg.ScalaVersion != "" {
		m.ScalaVersion = cfg.ScalaVersion
	} else {
		m.ScalaVersion = "2.13.12"
	}

	// Scala3
	if cfg != nil {
		m.Scala3 = cfg.Scala3
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
			ScalaName:   toScalaTypeName(t.Name),
			Description: t.Description,
			Kind:        t.Kind,
			Tag:         t.Tag,
		}

		switch t.Kind {
		case contract.KindStruct:
			for _, f := range t.Fields {
				fm := fieldModel{
					Name:        f.Name,
					ScalaName:   toScalaName(f.Name),
					JSONName:    f.Name,
					Description: f.Description,
					ScalaType:   scalaType(typeByName, f.Type, f.Optional, f.Nullable),
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
			tm.Elem = scalaType(typeByName, t.Elem, false, false)

		case contract.KindMap:
			tm.Elem = scalaType(typeByName, t.Elem, false, false)

		case contract.KindUnion:
			for _, v := range t.Variants {
				tm.Variants = append(tm.Variants, variantModel{
					Value:       v.Value,
					Type:        string(v.Type),
					ScalaName:   toPascal(string(v.Type)),
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
			ScalaName:   toCamel(r.Name),
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
				streamItem = toScalaTypeName(strings.TrimSpace(string(mm.Stream.Item)))
			}

			rm.Methods = append(rm.Methods, methodModel{
				Name:        mm.Name,
				ScalaName:   toCamel(mm.Name),
				Description: mm.Description,

				HasInput:    hasInput,
				HasOutput:   hasOutput,
				InputType:   toScalaTypeName(string(mm.Input)),
				OutputType:  toScalaTypeName(string(mm.Output)),
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

func scalaType(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "io.circe.Json"
	}

	base := baseScalaType(typeByName, r)

	if optional || nullable {
		return "Option[" + base + "]"
	}
	return base
}

func baseScalaType(typeByName map[string]*contract.Type, r string) string {
	// Check if it's a known type
	if _, ok := typeByName[r]; ok {
		return toScalaTypeName(r)
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
		return "Int"
	case "uint8":
		return "Short"
	case "uint16":
		return "Int"
	case "uint32":
		return "Long"
	case "uint64":
		return "BigInt"
	case "float32":
		return "Float"
	case "float64":
		return "Double"
	case "time.Time":
		return "java.time.Instant"
	case "json.RawMessage":
		return "io.circe.Json"
	case "any", "interface{}":
		return "io.circe.Json"
	}

	// Handle slice types
	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		return "List[" + baseScalaType(typeByName, elem) + "]"
	}

	// Handle map types
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		return "Map[String, " + baseScalaType(typeByName, elem) + "]"
	}

	return "io.circe.Json"
}

// toScalaName converts a string to camelCase for Scala properties/methods.
func toScalaName(s string) string {
	if s == "" {
		return ""
	}

	result := toCamel(s)

	// Check for reserved words
	if isScalaReserved(result) {
		return "`" + result + "`"
	}

	return result
}

// toScalaTypeName converts a string to PascalCase for Scala types.
func toScalaTypeName(s string) string {
	if s == "" {
		return ""
	}

	return toPascal(s)
}

// toCamel converts to camelCase.
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

// toPascal converts to PascalCase.
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

// toKebab converts to kebab-case.
func toKebab(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	prevWasUpper := false

	for i, r := range s {
		if r == '_' || r == '.' || r == ' ' {
			b.WriteRune('-')
			prevWasUpper = false
			continue
		}

		if unicode.IsUpper(r) {
			if i > 0 && !prevWasUpper {
				b.WriteRune('-')
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

// isScalaReserved checks if a name is a Scala reserved word.
func isScalaReserved(s string) bool {
	reserved := map[string]bool{
		"abstract": true, "case": true, "catch": true, "class": true,
		"def": true, "do": true, "else": true, "extends": true,
		"false": true, "final": true, "finally": true, "for": true,
		"forSome": true, "if": true, "implicit": true, "import": true,
		"lazy": true, "match": true, "new": true, "null": true,
		"object": true, "override": true, "package": true, "private": true,
		"protected": true, "return": true, "sealed": true, "super": true,
		"this": true, "throw": true, "trait": true, "try": true,
		"true": true, "type": true, "val": true, "var": true,
		"while": true, "with": true, "yield": true,
		// Scala 3 additions
		"enum": true, "export": true, "given": true, "then": true,
		"extension": true, "using": true, "transparent": true, "inline": true,
		"opaque": true, "open": true, "derives": true, "end": true,
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

// scalaQuote returns a quoted Scala string literal.
func scalaQuote(s string) string {
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

// toHTTPMethodName converts HTTP method to sttp Method reference.
// e.g., "GET" -> "Get", "POST" -> "Post", "DELETE" -> "Delete"
func toHTTPMethodName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "POST"
	}
	return strings.ToUpper(s)
}
