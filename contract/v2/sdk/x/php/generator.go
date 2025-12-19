// Package sdkphp generates typed PHP SDK clients from contract.Service.
package sdkphp

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

//go:embed templates/*.php.tmpl templates/*.json.tmpl
var templateFS embed.FS

// Config controls PHP SDK generation.
type Config struct {
	// Namespace is the PHP namespace.
	// Default: PascalCase service name.
	Namespace string

	// PackageName is the Composer package name (vendor/package).
	// Default: "vendor/{service}-sdk".
	PackageName string

	// Version is the package version for composer.json.
	// Default: "0.0.0".
	Version string

	// Author is the package author.
	Author string

	// License is the package license.
	// Default: "MIT".
	License string
}

// Generate produces a set of generated files for a typed PHP SDK client.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkphp: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("sdkphp").
		Funcs(template.FuncMap{
			"phpQuote":    phpQuote,
			"phpString":   phpQuote,
			"jsonQuote":   jsonQuote,
			"phpName":     toPHPName,
			"phpTypeName": toPHPTypeName,
			"phpVarName":  toPHPVarName,
			"camel":       toCamel,
			"pascal":      toPascal,
			"snake":       toSnake,
			"upper":       strings.ToUpper,
			"lower":       strings.ToLower,
			"join":        strings.Join,
			"trim":        strings.TrimSpace,
			"indent":      indent,
			"hasPrefix":   strings.HasPrefix,
			"add":         func(a, b int) int { return a + b },
			"sub":         func(a, b int) int { return a - b },
			"len":         lenHelper,
			"last":        lastHelper,
			"first":       firstHelper,
		}).
		ParseFS(templateFS, "templates/*.php.tmpl", "templates/*.json.tmpl")
	if err != nil {
		return nil, fmt.Errorf("sdkphp: parse templates: %w", err)
	}

	files := make([]*sdk.File, 0, 10)

	// Generate composer.json
	var out bytes.Buffer
	if err := tpl.ExecuteTemplate(&out, "composer.json.tmpl", m); err != nil {
		return nil, fmt.Errorf("sdkphp: execute template composer.json: %w", err)
	}
	files = append(files, &sdk.File{Path: "composer.json", Content: out.String()})

	// Generate main PHP files
	coreTemplates := []struct {
		name string
		path string
	}{
		{"Client.php.tmpl", "src/Client.php"},
		{"ClientOptions.php.tmpl", "src/ClientOptions.php"},
		{"AuthMode.php.tmpl", "src/AuthMode.php"},
		{"Exceptions.php.tmpl", "src/Exceptions/SDKException.php"},
	}

	for _, t := range coreTemplates {
		out.Reset()
		if err := tpl.ExecuteTemplate(&out, t.name, m); err != nil {
			return nil, fmt.Errorf("sdkphp: execute template %s: %w", t.name, err)
		}
		files = append(files, &sdk.File{Path: t.path, Content: out.String()})
	}

	// Generate individual type files
	for _, t := range m.Types {
		out.Reset()
		typeData := struct {
			*model
			Type typeModel
		}{m, t}

		if t.Kind == contract.KindUnion {
			if err := tpl.ExecuteTemplate(&out, "UnionType.php.tmpl", typeData); err != nil {
				return nil, fmt.Errorf("sdkphp: execute template UnionType for %s: %w", t.Name, err)
			}
		} else if t.IsEnum {
			if err := tpl.ExecuteTemplate(&out, "EnumType.php.tmpl", typeData); err != nil {
				return nil, fmt.Errorf("sdkphp: execute template EnumType for %s: %w", t.Name, err)
			}
		} else {
			if err := tpl.ExecuteTemplate(&out, "Type.php.tmpl", typeData); err != nil {
				return nil, fmt.Errorf("sdkphp: execute template Type for %s: %w", t.Name, err)
			}
		}
		files = append(files, &sdk.File{
			Path:    "src/Types/" + t.PHPName + ".php",
			Content: out.String(),
		})
	}

	// Generate resource files
	for _, r := range m.Resources {
		out.Reset()
		resourceData := struct {
			*model
			Resource resourceModel
		}{m, r}
		if err := tpl.ExecuteTemplate(&out, "Resource.php.tmpl", resourceData); err != nil {
			return nil, fmt.Errorf("sdkphp: execute template Resource for %s: %w", r.Name, err)
		}
		files = append(files, &sdk.File{
			Path:    "src/Resources/" + r.ClassName + ".php",
			Content: out.String(),
		})
	}

	return files, nil
}

type model struct {
	Namespace   string
	PackageName string
	Version     string
	Author      string
	License     string

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
	PHPName     string
	Description string
	Kind        contract.TypeKind

	Fields   []fieldModel
	Elem     string
	Tag      string
	Variants []variantModel

	// For enums extracted from fields
	IsEnum     bool
	EnumValues []enumValue
}

type fieldModel struct {
	Name        string
	PHPName     string
	JSONName    string
	Description string
	PHPType     string
	PHPDocType  string

	Optional bool
	Nullable bool
	IsArray  bool
	IsMap    bool
	ArrayOf  string
	MapOf    string
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
	PHPName     string
	Description string
}

type resourceModel struct {
	Name        string
	PHPName     string
	ClassName   string
	Description string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	PHPName     string
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

	// Namespace
	if cfg != nil && cfg.Namespace != "" {
		m.Namespace = cfg.Namespace
	} else {
		m.Namespace = toPascal(sanitizeIdent(svc.Name))
		if m.Namespace == "" {
			m.Namespace = "SDK"
		}
	}

	// Package name
	if cfg != nil && cfg.PackageName != "" {
		m.PackageName = cfg.PackageName
	} else {
		pkgName := strings.ToLower(sanitizeIdent(svc.Name))
		if pkgName == "" {
			pkgName = "sdk"
		}
		m.PackageName = "vendor/" + pkgName + "-sdk"
	}

	// Version
	if cfg != nil && cfg.Version != "" {
		m.Version = cfg.Version
	} else {
		m.Version = "0.0.0"
	}

	// Author
	if cfg != nil && cfg.Author != "" {
		m.Author = cfg.Author
	}

	// License
	if cfg != nil && cfg.License != "" {
		m.License = cfg.License
	} else {
		m.License = "MIT"
	}

	// Service info
	m.Service.Name = svc.Name
	m.Service.Sanitized = toPascal(sanitizeIdent(svc.Name))
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
			PHPName:     toPHPTypeName(t.Name),
			Description: t.Description,
			Kind:        t.Kind,
			Tag:         t.Tag,
		}

		switch t.Kind {
		case contract.KindStruct:
			for _, f := range t.Fields {
				phpType, phpDocType, isArray, arrayOf, isMap, mapOf := phpTypeInfo(typeByName, f.Type, f.Optional, f.Nullable)

				fm := fieldModel{
					Name:        f.Name,
					PHPName:     toPHPVarName(f.Name),
					JSONName:    f.Name,
					Description: f.Description,
					PHPType:     phpType,
					PHPDocType:  phpDocType,
					Optional:    f.Optional,
					Nullable:    f.Nullable,
					IsArray:     isArray,
					ArrayOf:     arrayOf,
					IsMap:       isMap,
					MapOf:       mapOf,
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
			tm.Elem = phpType(typeByName, t.Elem, false, false)

		case contract.KindMap:
			tm.Elem = phpType(typeByName, t.Elem, false, false)

		case contract.KindUnion:
			for _, v := range t.Variants {
				tm.Variants = append(tm.Variants, variantModel{
					Value:       v.Value,
					Type:        string(v.Type),
					PHPName:     toPHPTypeName(string(v.Type)),
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
			PHPName:     toCamel(r.Name),
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
				streamItem = toPHPTypeName(strings.TrimSpace(string(mm.Stream.Item)))
			}

			rm.Methods = append(rm.Methods, methodModel{
				Name:        mm.Name,
				PHPName:     toCamel(mm.Name),
				Description: mm.Description,

				HasInput:    hasInput,
				HasOutput:   hasOutput,
				InputType:   toPHPTypeName(string(mm.Input)),
				OutputType:  toPHPTypeName(string(mm.Output)),
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

func phpType(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "mixed"
	}

	base := basePHPType(typeByName, r)

	if optional || nullable {
		return "?" + base
	}
	return base
}

func phpTypeInfo(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) (phpType, phpDocType string, isArray bool, arrayOf string, isMap bool, mapOf string) {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "mixed", "mixed", false, "", false, ""
	}

	// Handle slice types
	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		elemType := basePHPType(typeByName, elem)
		phpType = "array"
		phpDocType = "array<" + elemType + ">"
		isArray = true
		arrayOf = elemType
		if optional || nullable {
			phpType = "?" + phpType
			phpDocType = phpDocType + "|null"
		}
		return
	}

	// Handle map types
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		elemType := basePHPType(typeByName, elem)
		phpType = "array"
		phpDocType = "array<string, " + elemType + ">"
		isMap = true
		mapOf = elemType
		if optional || nullable {
			phpType = "?" + phpType
			phpDocType = phpDocType + "|null"
		}
		return
	}

	base := basePHPType(typeByName, r)
	phpType = base
	phpDocType = base

	if optional || nullable {
		phpType = "?" + base
		phpDocType = base + "|null"
	}
	return
}

func basePHPType(typeByName map[string]*contract.Type, r string) string {
	// Check if it's a known type
	if _, ok := typeByName[r]; ok {
		return toPHPTypeName(r)
	}

	switch r {
	case "string":
		return "string"
	case "bool", "boolean":
		return "bool"
	case "int", "int8", "int16", "int32", "int64":
		return "int"
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "int"
	case "float32", "float64":
		return "float"
	case "time.Time":
		return "\\DateTimeImmutable"
	case "json.RawMessage":
		return "array"
	case "any", "interface{}":
		return "mixed"
	}

	// Handle slice types
	if strings.HasPrefix(r, "[]") {
		return "array"
	}

	// Handle map types
	if strings.HasPrefix(r, "map[string]") {
		return "array"
	}

	return "mixed"
}

// toPHPName converts a string to camelCase for PHP methods.
func toPHPName(s string) string {
	if s == "" {
		return ""
	}

	result := toCamel(s)

	if isPHPReserved(result) {
		return "_" + result
	}
	return result
}

// toPHPTypeName converts a string to PascalCase for PHP classes.
func toPHPTypeName(s string) string {
	if s == "" {
		return ""
	}

	result := toPascal(s)

	if isPHPReserved(result) {
		return result + "_"
	}
	return result
}

// toPHPVarName converts a string to camelCase for PHP variables/properties.
func toPHPVarName(s string) string {
	if s == "" {
		return ""
	}

	result := toCamel(s)

	if isPHPReserved(result) {
		return "_" + result
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
			b.WriteRune('_')
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
			prevWasLower = unicode.IsLower(r)
		}
	}

	return b.String()
}

// toEnumCase converts a value to SCREAMING_SNAKE_CASE for PHP enum constants.
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

// isPHPReserved checks if a name is a PHP reserved word.
func isPHPReserved(s string) bool {
	lower := strings.ToLower(s)
	reserved := map[string]bool{
		"abstract": true, "and": true, "array": true, "as": true,
		"break": true, "callable": true, "case": true, "catch": true,
		"class": true, "clone": true, "const": true, "continue": true,
		"declare": true, "default": true, "die": true, "do": true,
		"echo": true, "else": true, "elseif": true, "empty": true,
		"enddeclare": true, "endfor": true, "endforeach": true, "endif": true,
		"endswitch": true, "endwhile": true, "enum": true, "eval": true,
		"exit": true, "extends": true, "final": true, "finally": true,
		"fn": true, "for": true, "foreach": true, "function": true,
		"global": true, "goto": true, "if": true, "implements": true,
		"include": true, "include_once": true, "instanceof": true, "insteadof": true,
		"interface": true, "isset": true, "list": true, "match": true,
		"namespace": true, "new": true, "or": true, "print": true,
		"private": true, "protected": true, "public": true, "readonly": true,
		"require": true, "require_once": true, "return": true, "static": true,
		"switch": true, "throw": true, "trait": true, "try": true,
		"unset": true, "use": true, "var": true, "while": true,
		"xor": true, "yield": true, "yield_from": true,
		// Soft reserved
		"int": true, "float": true, "bool": true, "string": true,
		"true": true, "false": true, "null": true, "void": true,
		"iterable": true, "object": true, "resource": true, "mixed": true,
		"numeric": true, "never": true,
	}
	return reserved[lower]
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

// phpQuote returns a quoted PHP string literal.
func phpQuote(s string) string {
	// Use single quotes for simple strings, escape as needed
	escaped := strings.ReplaceAll(s, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "'", "\\'")
	return "'" + escaped + "'"
}

// jsonQuote returns a quoted JSON string literal.
func jsonQuote(s string) string {
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

// lastHelper checks if index is the last element.
func lastHelper(index, length int) bool {
	return index == length-1
}

// firstHelper checks if index is the first element.
func firstHelper(index int) bool {
	return index == 0
}
