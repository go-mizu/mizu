// Package sdkelixir generates typed Elixir SDK clients from contract.Service.
package sdkelixir

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

// Config controls Elixir SDK generation.
type Config struct {
	// AppName is the OTP application name.
	// Default: sanitized lowercase service name with underscores.
	AppName string

	// ModuleName is the root Elixir module name.
	// Default: PascalCase service name.
	ModuleName string

	// Version is the package version for mix.exs.
	// Default: "0.0.0".
	Version string

	// Description is the package description.
	Description string

	// Homepage is the project homepage URL.
	Homepage string
}

// Generate produces a set of generated files for a typed Elixir SDK client.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkelixir: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("sdkelixir").
		Funcs(template.FuncMap{
			"elixirQuote":    elixirQuote,
			"elixirString":   elixirQuote,
			"elixirAtom":     elixirAtom,
			"elixirName":     toElixirName,
			"elixirTypeName": toElixirTypeName,
			"elixirModName":  toElixirModuleName,
			"snake":          toSnake,
			"pascal":         toPascal,
			"screamingSnake": toScreamingSnake,
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
		ParseFS(templateFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("sdkelixir: parse templates: %w", err)
	}

	files := make([]*sdk.File, 0, 10)

	// Generate each file from its template
	templates := []struct {
		name string
		path string
	}{
		{"mix.exs.tmpl", "mix.exs"},
		{"main.ex.tmpl", "lib/" + m.AppName + ".ex"},
		{"client.ex.tmpl", "lib/" + m.AppName + "/client.ex"},
		{"config.ex.tmpl", "lib/" + m.AppName + "/config.ex"},
		{"types.ex.tmpl", "lib/" + m.AppName + "/types.ex"},
		{"resources.ex.tmpl", "lib/" + m.AppName + "/resources.ex"},
		{"streaming.ex.tmpl", "lib/" + m.AppName + "/streaming.ex"},
		{"errors.ex.tmpl", "lib/" + m.AppName + "/errors.ex"},
	}

	for _, t := range templates {
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, t.name, m); err != nil {
			return nil, fmt.Errorf("sdkelixir: execute template %s: %w", t.name, err)
		}
		files = append(files, &sdk.File{Path: t.path, Content: out.String()})
	}

	return files, nil
}

type model struct {
	AppName     string
	ModuleName  string
	Version     string
	Description string
	Homepage    string

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
	ElixirName  string
	Description string
	Kind        contract.TypeKind

	Fields   []fieldModel
	Elem     string
	Tag      string
	Variants []variantModel
}

type fieldModel struct {
	Name        string
	ElixirName  string
	AtomName    string
	JSONName    string
	Description string
	ElixirType  string
	TypeSpec    string

	Optional bool
	Nullable bool
	Enum     []enumValue
	Const    string
}

type enumValue struct {
	Name  string
	Value string
	Atom  string
}

type variantModel struct {
	Value       string
	Type        string
	ElixirName  string
	Description string
}

type resourceModel struct {
	Name        string
	ElixirName  string
	ModuleName  string
	Description string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	ElixirName  string
	Description string

	HasInput      bool
	HasOutput     bool
	InputIsStruct bool
	InputFields   []fieldModel

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

	// App name
	if cfg != nil && cfg.AppName != "" {
		m.AppName = cfg.AppName
	} else {
		m.AppName = toSnake(sanitizeIdent(svc.Name))
		if m.AppName == "" {
			m.AppName = "sdk"
		}
	}

	// Module name
	if cfg != nil && cfg.ModuleName != "" {
		m.ModuleName = cfg.ModuleName
	} else {
		m.ModuleName = toPascal(sanitizeIdent(svc.Name))
		if m.ModuleName == "" {
			m.ModuleName = "SDK"
		}
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
		m.Description = m.ModuleName + " SDK"
	}

	// Homepage
	if cfg != nil && cfg.Homepage != "" {
		m.Homepage = cfg.Homepage
	} else {
		m.Homepage = "https://github.com/example/" + m.AppName
	}

	// Service info
	m.Service.Name = svc.Name
	m.Service.Sanitized = toPascal(sanitizeIdent(svc.Name))
	m.Service.Description = svc.Description

	// Client config
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
			ElixirName:  toPascal(t.Name),
			Description: t.Description,
			Kind:        t.Kind,
			Tag:         t.Tag,
		}

		switch t.Kind {
		case contract.KindStruct:
			for _, f := range t.Fields {
				elixirType := elixirType(typeByName, f.Type, f.Optional, f.Nullable)
				typeSpec := elixirTypeSpec(typeByName, f.Type, f.Optional, f.Nullable)

				fm := fieldModel{
					Name:        f.Name,
					ElixirName:  toElixirName(f.Name),
					AtomName:    elixirAtom(toSnake(f.Name)),
					JSONName:    f.Name,
					Description: f.Description,
					ElixirType:  elixirType,
					TypeSpec:    typeSpec,
					Optional:    f.Optional,
					Nullable:    f.Nullable,
					Const:       f.Const,
				}

				// Handle enum values
				for _, e := range f.Enum {
					fm.Enum = append(fm.Enum, enumValue{
						Name:  toSnake(e),
						Value: e,
						Atom:  elixirAtom(toSnake(e)),
					})
				}

				tm.Fields = append(tm.Fields, fm)
			}

		case contract.KindSlice:
			tm.Elem = elixirType(typeByName, t.Elem, false, false)

		case contract.KindMap:
			tm.Elem = elixirType(typeByName, t.Elem, false, false)

		case contract.KindUnion:
			for _, v := range t.Variants {
				tm.Variants = append(tm.Variants, variantModel{
					Value:       v.Value,
					Type:        string(v.Type),
					ElixirName:  toPascal(string(v.Type)),
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
			ElixirName:  toSnake(r.Name),
			ModuleName:  toPascal(r.Name),
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

			// Check if input type is a struct and get its fields
			inputIsStruct := false
			var inputFields []fieldModel
			if hasInput {
				inputTypeName := strings.TrimSpace(string(mm.Input))
				if t, ok := typeByName[inputTypeName]; ok && t.Kind == contract.KindStruct {
					inputIsStruct = true
					for _, f := range t.Fields {
						inputFields = append(inputFields, fieldModel{
							Name:        f.Name,
							ElixirName:  toElixirName(f.Name),
							AtomName:    elixirAtom(toSnake(f.Name)),
							JSONName:    f.Name,
							Description: f.Description,
							ElixirType:  elixirType(typeByName, f.Type, f.Optional, f.Nullable),
							TypeSpec:    elixirTypeSpec(typeByName, f.Type, f.Optional, f.Nullable),
							Optional:    f.Optional,
							Nullable:    f.Nullable,
							Const:       f.Const,
						})
					}
				}
			}

			isStreaming := mm.Stream != nil
			streamMode := ""
			streamIsSSE := false
			streamItem := ""

			if isStreaming {
				streamMode = strings.TrimSpace(mm.Stream.Mode)
				streamIsSSE = streamMode == "" || strings.EqualFold(streamMode, "sse")
				streamItem = toPascal(strings.TrimSpace(string(mm.Stream.Item)))
			}

			rm.Methods = append(rm.Methods, methodModel{
				Name:        mm.Name,
				ElixirName:  toSnake(mm.Name),
				Description: mm.Description,

				HasInput:      hasInput,
				HasOutput:     hasOutput,
				InputIsStruct: inputIsStruct,
				InputFields:   inputFields,
				InputType:     toPascal(string(mm.Input)),
				OutputType:    toPascal(string(mm.Output)),
				HTTPMethod:    httpMethod,
				HTTPPath:      httpPath,
				IsStreaming:   isStreaming,

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

func elixirType(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "term()"
	}

	base := baseElixirType(typeByName, r)

	if optional || nullable {
		return base + " | nil"
	}
	return base
}

func elixirTypeSpec(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "term()"
	}

	base := baseElixirTypeSpec(typeByName, r)

	if optional || nullable {
		return base + " | nil"
	}
	return base
}

func baseElixirType(typeByName map[string]*contract.Type, r string) string {
	// Check if it's a known type
	if _, ok := typeByName[r]; ok {
		return toPascal(r) + ".t()"
	}

	switch r {
	case "string":
		return "String.t()"
	case "bool", "boolean":
		return "boolean()"
	case "int", "int8", "int16", "int32", "int64":
		return "integer()"
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "non_neg_integer()"
	case "float32", "float64":
		return "float()"
	case "time.Time":
		return "DateTime.t()"
	case "json.RawMessage":
		return "map()"
	case "any", "interface{}":
		return "term()"
	}

	// Handle slice types
	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		return "[" + baseElixirType(typeByName, elem) + "]"
	}

	// Handle map types
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		return "%{String.t() => " + baseElixirType(typeByName, elem) + "}"
	}

	return "term()"
}

func baseElixirTypeSpec(typeByName map[string]*contract.Type, r string) string {
	// Check if it's a known type
	if _, ok := typeByName[r]; ok {
		return toPascal(r) + ".t()"
	}

	switch r {
	case "string":
		return "String.t()"
	case "bool", "boolean":
		return "boolean()"
	case "int", "int8", "int16", "int32", "int64":
		return "integer()"
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "non_neg_integer()"
	case "float32", "float64":
		return "float()"
	case "time.Time":
		return "DateTime.t()"
	case "json.RawMessage":
		return "map()"
	case "any", "interface{}":
		return "term()"
	}

	// Handle slice types
	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		return "list(" + baseElixirTypeSpec(typeByName, elem) + ")"
	}

	// Handle map types
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		return "%{String.t() => " + baseElixirTypeSpec(typeByName, elem) + "}"
	}

	return "term()"
}

// toElixirName converts a string to snake_case for Elixir functions/variables.
func toElixirName(s string) string {
	if s == "" {
		return ""
	}

	result := toSnake(s)

	// Check for reserved words
	if isElixirReserved(result) {
		return result + "_"
	}

	return result
}

// toElixirTypeName converts a string to PascalCase for Elixir modules.
func toElixirTypeName(s string) string {
	if s == "" {
		return ""
	}
	return toPascal(s)
}

// toElixirModuleName converts a string to a valid Elixir module name.
func toElixirModuleName(s string) string {
	if s == "" {
		return ""
	}
	return toPascal(s)
}

// toSnake converts a string to snake_case.
func toSnake(s string) string {
	if s == "" {
		return ""
	}

	runes := []rune(s)
	var b strings.Builder

	isSep := func(r rune) bool {
		return r == '_' || r == '-' || r == '.' || r == ' '
	}

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		if isSep(r) {
			if b.Len() > 0 && b.String()[b.Len()-1] != '_' {
				b.WriteByte('_')
			}
			continue
		}

		if unicode.IsLower(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}

		if unicode.IsUpper(r) {
			// Add underscore if previous was lowercase or digit (new word)
			if i > 0 && !isSep(runes[i-1]) {
				prev := runes[i-1]
				if unicode.IsLower(prev) || unicode.IsDigit(prev) {
					if b.Len() > 0 && b.String()[b.Len()-1] != '_' {
						b.WriteByte('_')
					}
				}
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}

		b.WriteRune(r)
	}

	result := strings.Trim(b.String(), "_")
	if result == "" {
		return "x"
	}
	return result
}

// toPascal converts a string to PascalCase.
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

	result := b.String()
	if result == "" {
		return "X"
	}
	return result
}

// toScreamingSnake converts a string to SCREAMING_SNAKE_CASE.
func toScreamingSnake(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	prevWasUpper := false
	prevWasLower := false

	for i, r := range s {
		if r == '_' || r == '-' || r == '.' || r == ' ' {
			if b.Len() > 0 && b.String()[b.Len()-1] != '_' {
				b.WriteByte('_')
			}
			prevWasUpper = false
			prevWasLower = false
			continue
		}

		if unicode.IsUpper(r) {
			if prevWasLower || (prevWasUpper && i+1 < len(s) && unicode.IsLower(rune(s[i+1]))) {
				if b.Len() > 0 && b.String()[b.Len()-1] != '_' {
					b.WriteByte('_')
				}
			}
			b.WriteRune(r)
			prevWasUpper = true
			prevWasLower = false
			continue
		}

		b.WriteRune(unicode.ToUpper(r))
		prevWasLower = unicode.IsLetter(r)
		prevWasUpper = false
	}

	result := strings.Trim(b.String(), "_")
	if result == "" {
		return "X"
	}
	return result
}

// isElixirReserved checks if a name is an Elixir reserved word.
func isElixirReserved(s string) bool {
	reserved := map[string]bool{
		"after": true, "and": true, "catch": true, "cond": true,
		"do": true, "else": true, "end": true, "false": true,
		"fn": true, "for": true, "if": true, "in": true,
		"nil": true, "not": true, "or": true, "raise": true,
		"receive": true, "rescue": true, "true": true, "try": true,
		"unless": true, "when": true, "with": true,
		// Additional keywords that can cause issues
		"def": true, "defp": true, "defmodule": true, "defstruct": true,
		"defmacro": true, "defmacrop": true, "defprotocol": true,
		"defimpl": true, "defdelegate": true, "defexception": true,
		"defguard": true, "defguardp": true, "alias": true,
		"import": true, "require": true, "use": true,
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

// elixirQuote returns a quoted Elixir string literal.
func elixirQuote(s string) string {
	// Use Elixir string escaping
	escaped := strings.ReplaceAll(s, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	escaped = strings.ReplaceAll(escaped, "\n", "\\n")
	escaped = strings.ReplaceAll(escaped, "\r", "\\r")
	escaped = strings.ReplaceAll(escaped, "\t", "\\t")
	return "\"" + escaped + "\""
}

// elixirAtom returns an Elixir atom.
func elixirAtom(s string) string {
	// Check if it needs quoting (contains special chars or starts with number)
	needsQuote := len(s) > 0 && unicode.IsDigit(rune(s[0]))
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '@' {
			needsQuote = true
			break
		}
	}
	if needsQuote {
		return ":\"" + s + "\""
	}
	return ":" + s
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
