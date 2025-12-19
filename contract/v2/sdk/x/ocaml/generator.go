// Package sdkocaml generates typed OCaml SDK clients from contract.Service.
package sdkocaml

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

// Config controls OCaml SDK generation.
type Config struct {
	// PackageName is the opam/dune package name.
	// Default: sanitized lowercase service name with underscores.
	PackageName string

	// ModuleName is the root OCaml module name.
	// Default: PascalCase service name.
	ModuleName string

	// Version is the package version for opam.
	// Default: "0.0.0".
	Version string

	// Author is the package author.
	Author string

	// License is the package license.
	// Default: "MIT".
	License string

	// Synopsis is a one-line package description.
	Synopsis string
}

// Generate produces a set of generated files for a typed OCaml SDK client.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkocaml: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("sdkocaml").
		Funcs(template.FuncMap{
			"ocamlQuote":     ocamlQuote,
			"ocamlString":    ocamlQuote,
			"ocamlName":      toOCamlName,
			"ocamlTypeName":  toOCamlTypeName,
			"ocamlModName":   toOCamlModuleName,
			"ocamlField":     toOCamlFieldName,
			"ocamlVariant":   toOCamlVariantName,
			"snake":          toSnake,
			"camel":          toCamel,
			"pascal":         toPascal,
			"screamingSnake": toScreamingSnake,
			"upper":          strings.ToUpper,
			"lower":          strings.ToLower,
			"join":           strings.Join,
			"trim":           strings.TrimSpace,
			"trimSuffix":     strings.TrimSuffix,
			"trimOption":     trimOption,
			"indent":         indent,
			"hasPrefix":      strings.HasPrefix,
			"add":            func(a, b int) int { return a + b },
			"sub":            func(a, b int) int { return a - b },
			"len":            func(s interface{}) int { return lenHelper(s) },
		}).
		ParseFS(templateFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("sdkocaml: parse templates: %w", err)
	}

	files := make([]*sdk.File, 0, 16)

	// Generate each file from its template
	templates := []struct {
		name string
		path string
	}{
		{"dune-project.tmpl", "dune-project"},
		{"opam.tmpl", m.PackageName + ".opam"},
		{"lib_dune.tmpl", "lib/dune"},
		{"main.ml.tmpl", "lib/" + m.PackageName + ".ml"},
		{"main.mli.tmpl", "lib/" + m.PackageName + ".mli"},
		{"config.ml.tmpl", "lib/config.ml"},
		{"config.mli.tmpl", "lib/config.mli"},
		{"client.ml.tmpl", "lib/client.ml"},
		{"client.mli.tmpl", "lib/client.mli"},
		{"types.ml.tmpl", "lib/types.ml"},
		{"types.mli.tmpl", "lib/types.mli"},
		{"resources.ml.tmpl", "lib/resources.ml"},
		{"resources.mli.tmpl", "lib/resources.mli"},
		{"streaming.ml.tmpl", "lib/streaming.ml"},
		{"streaming.mli.tmpl", "lib/streaming.mli"},
		{"errors.ml.tmpl", "lib/errors.ml"},
		{"errors.mli.tmpl", "lib/errors.mli"},
	}

	for _, t := range templates {
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, t.name, m); err != nil {
			return nil, fmt.Errorf("sdkocaml: execute template %s: %w", t.name, err)
		}
		files = append(files, &sdk.File{Path: t.path, Content: out.String()})
	}

	return files, nil
}

type model struct {
	PackageName string
	ModuleName  string
	Version     string
	Author      string
	License     string
	Synopsis    string

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

	EnvPrefix string

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
	OCamlName   string
	ModuleName  string
	Description string
	Kind        contract.TypeKind

	Fields   []fieldModel
	Elem     string
	ElemSpec string
	Tag      string
	Variants []variantModel
}

type fieldModel struct {
	Name        string
	OCamlName   string
	JSONName    string
	Description string
	OCamlType   string

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
	OCamlName   string
	VariantName string // Constructor name like `Text
	Description string
}

type resourceModel struct {
	Name        string
	OCamlName   string
	ModuleName  string
	Description string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	OCamlName   string
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

	// Package name (lowercase, underscores)
	if cfg != nil && cfg.PackageName != "" {
		m.PackageName = cfg.PackageName
	} else {
		m.PackageName = toSnake(sanitizeIdent(svc.Name))
		if m.PackageName == "" {
			m.PackageName = "sdk"
		}
	}

	// Module name (PascalCase)
	if cfg != nil && cfg.ModuleName != "" {
		m.ModuleName = cfg.ModuleName
	} else {
		m.ModuleName = toPascal(sanitizeIdent(svc.Name))
		if m.ModuleName == "" {
			m.ModuleName = "Sdk"
		}
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
	} else {
		m.Author = "Generated"
	}

	// License
	if cfg != nil && cfg.License != "" {
		m.License = cfg.License
	} else {
		m.License = "MIT"
	}

	// Synopsis
	if cfg != nil && cfg.Synopsis != "" {
		m.Synopsis = cfg.Synopsis
	} else if svc.Description != "" {
		m.Synopsis = svc.Description
	} else {
		m.Synopsis = m.ModuleName + " SDK"
	}

	// Environment variable prefix
	m.EnvPrefix = toScreamingSnake(sanitizeIdent(svc.Name))
	if m.EnvPrefix == "" {
		m.EnvPrefix = "SDK"
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

		ocamlTypeName := toSnake(t.Name)
		tm := typeModel{
			Name:        t.Name,
			OCamlName:   ocamlTypeName,
			ModuleName:  toPascal(t.Name),
			Description: t.Description,
			Kind:        t.Kind,
			Tag:         t.Tag,
		}

		switch t.Kind {
		case contract.KindStruct:
			for _, f := range t.Fields {
				ocamlType := ocamlType(typeByName, f.Type, f.Optional, f.Nullable)

				fm := fieldModel{
					Name:        f.Name,
					OCamlName:   toOCamlFieldName(f.Name),
					JSONName:    f.Name,
					Description: f.Description,
					OCamlType:   ocamlType,
					Optional:    f.Optional,
					Nullable:    f.Nullable,
					Const:       f.Const,
				}

				// Handle enum values
				for _, e := range f.Enum {
					fm.Enum = append(fm.Enum, enumValue{
						Name:  toOCamlVariantName(e),
						Value: e,
					})
				}

				tm.Fields = append(tm.Fields, fm)
			}

		case contract.KindSlice:
			tm.Elem = ocamlType(typeByName, t.Elem, false, false)
			tm.ElemSpec = tm.Elem + " list"

		case contract.KindMap:
			tm.Elem = ocamlType(typeByName, t.Elem, false, false)
			tm.ElemSpec = "(string * " + tm.Elem + ") list"

		case contract.KindUnion:
			for _, v := range t.Variants {
				variantTypeName := toSnake(string(v.Type))
				tm.Variants = append(tm.Variants, variantModel{
					Value:       v.Value,
					Type:        string(v.Type),
					OCamlName:   variantTypeName,
					VariantName: toOCamlVariantName(string(v.Type)),
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
			OCamlName:   toSnake(r.Name),
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
							OCamlName:   toOCamlFieldName(f.Name),
							JSONName:    f.Name,
							Description: f.Description,
							OCamlType:   ocamlType(typeByName, f.Type, f.Optional, f.Nullable),
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
				streamItem = toSnake(strings.TrimSpace(string(mm.Stream.Item)))
			}

			methodName := toSnake(mm.Name)
			if isOCamlReserved(methodName) {
				methodName = methodName + "_"
			}

			rm.Methods = append(rm.Methods, methodModel{
				Name:        mm.Name,
				OCamlName:   methodName,
				Description: mm.Description,

				HasInput:      hasInput,
				HasOutput:     hasOutput,
				InputIsStruct: inputIsStruct,
				InputFields:   inputFields,
				InputType:     toSnake(string(mm.Input)),
				OutputType:    toSnake(string(mm.Output)),
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

func ocamlType(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "Yojson.Safe.t"
	}

	base := baseOCamlType(typeByName, r)

	if optional || nullable {
		return base + " option"
	}
	return base
}

func baseOCamlType(typeByName map[string]*contract.Type, r string) string {
	// Check if it's a known type
	if _, ok := typeByName[r]; ok {
		return toSnake(r)
	}

	switch r {
	case "string":
		return "string"
	case "bool", "boolean":
		return "bool"
	case "int", "int8", "int16":
		return "int"
	case "int32":
		return "int32"
	case "int64":
		return "int64"
	case "uint", "uint8", "uint16":
		return "int"
	case "uint32":
		return "int32"
	case "uint64":
		return "int64"
	case "float32", "float64":
		return "float"
	case "time.Time":
		return "Ptime.t"
	case "json.RawMessage":
		return "Yojson.Safe.t"
	case "any", "interface{}":
		return "Yojson.Safe.t"
	}

	// Handle slice types
	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		elemType := baseOCamlType(typeByName, elem)
		return elemType + " list"
	}

	// Handle map types
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		elemType := baseOCamlType(typeByName, elem)
		return "(string * " + elemType + ") list"
	}

	return "Yojson.Safe.t"
}

// toOCamlName converts a string to snake_case for OCaml values/functions.
func toOCamlName(s string) string {
	if s == "" {
		return ""
	}

	result := toSnake(s)

	// Check for reserved words
	if isOCamlReserved(result) {
		return result + "_"
	}

	return result
}

// toOCamlTypeName converts a string to snake_case for OCaml types.
func toOCamlTypeName(s string) string {
	if s == "" {
		return ""
	}

	result := toSnake(s)

	// Check for reserved words
	if isOCamlReserved(result) {
		return result + "_"
	}

	return result
}

// toOCamlModuleName converts a string to a valid OCaml module name (PascalCase).
func toOCamlModuleName(s string) string {
	if s == "" {
		return ""
	}
	return toPascal(s)
}

// toOCamlFieldName generates a snake_case field name.
func toOCamlFieldName(s string) string {
	result := toSnake(s)
	if isOCamlReserved(result) {
		return result + "_"
	}
	return result
}

// toOCamlVariantName generates a PascalCase variant name for polymorphic variants.
func toOCamlVariantName(s string) string {
	result := toPascal(s)
	// OCaml polymorphic variants use backtick, regular variants use PascalCase
	return result
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

// toCamel converts a string to camelCase.
func toCamel(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	capNext := false
	first := true

	for _, r := range s {
		if r == '_' || r == '-' || r == '.' || r == ' ' {
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

// isOCamlReserved checks if a name is an OCaml reserved word.
func isOCamlReserved(s string) bool {
	reserved := map[string]bool{
		"and": true, "as": true, "assert": true, "asr": true,
		"begin": true, "class": true, "constraint": true, "do": true,
		"done": true, "downto": true, "else": true, "end": true,
		"exception": true, "external": true, "false": true, "for": true,
		"fun": true, "function": true, "functor": true, "if": true,
		"in": true, "include": true, "inherit": true, "initializer": true,
		"land": true, "lazy": true, "let": true, "lor": true,
		"lsl": true, "lsr": true, "lxor": true, "match": true,
		"method": true, "mod": true, "module": true, "mutable": true,
		"new": true, "nonrec": true, "object": true, "of": true,
		"open": true, "or": true, "private": true, "rec": true,
		"sig": true, "struct": true, "then": true, "to": true,
		"true": true, "try": true, "type": true, "val": true,
		"virtual": true, "when": true, "while": true, "with": true,
		// Common function names that might cause issues
		"t": true, "compare": true, "equal": true, "hash": true,
		"pp": true, "show": true, "of_yojson": true, "yojson_of": true,
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

// ocamlQuote returns a quoted OCaml string literal.
func ocamlQuote(s string) string {
	escaped := strings.ReplaceAll(s, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	escaped = strings.ReplaceAll(escaped, "\n", "\\n")
	escaped = strings.ReplaceAll(escaped, "\r", "\\r")
	escaped = strings.ReplaceAll(escaped, "\t", "\\t")
	return "\"" + escaped + "\""
}

// trimOption removes " option" suffix from OCaml types.
func trimOption(s string) string {
	return strings.TrimSuffix(s, " option")
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
