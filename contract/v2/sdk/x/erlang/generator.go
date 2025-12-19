// Package sdkerlang generates typed Erlang SDK clients from contract.Service.
package sdkerlang

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

// Config controls Erlang SDK generation.
type Config struct {
	// AppName is the OTP application name.
	// Default: sanitized lowercase service name with underscores.
	AppName string

	// Version is the package version.
	// Default: "0.0.0".
	Version string

	// Description is the package description.
	Description string
}

// Generate produces a set of generated files for a typed Erlang SDK client.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkerlang: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("sdkerlang").
		Funcs(template.FuncMap{
			"erlangQuote":      erlangQuote,
			"erlangBinary":     erlangBinary,
			"erlangAtom":       erlangAtom,
			"erlangName":       toErlangName,
			"erlangTypeName":   toErlangTypeName,
			"erlangModuleName": toErlangModuleName,
			"erlangRecordName": toErlangRecordName,
			"snake":            toSnake,
			"pascal":           toPascal,
			"screamingSnake":   toScreamingSnake,
			"upper":            strings.ToUpper,
			"lower":            strings.ToLower,
			"join":             strings.Join,
			"trim":             strings.TrimSpace,
			"indent":           indent,
			"hasPrefix":        strings.HasPrefix,
			"add":              func(a, b int) int { return a + b },
			"sub":              func(a, b int) int { return a - b },
			"len":              func(s interface{}) int { return lenHelper(s) },
		}).
		ParseFS(templateFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("sdkerlang: parse templates: %w", err)
	}

	files := make([]*sdk.File, 0, 15)

	// Generate core files
	coreTemplates := []struct {
		name string
		path string
	}{
		{"rebar.config.tmpl", "rebar.config"},
		{"app.src.tmpl", "src/" + m.AppName + ".app.src"},
		{"main.erl.tmpl", "src/" + m.AppName + ".erl"},
		{"client.erl.tmpl", "src/" + m.AppName + "_client.erl"},
		{"config.erl.tmpl", "src/" + m.AppName + "_config.erl"},
		{"types.erl.tmpl", "src/" + m.AppName + "_types.erl"},
		{"streaming.erl.tmpl", "src/" + m.AppName + "_streaming.erl"},
		{"errors.erl.tmpl", "src/" + m.AppName + "_errors.erl"},
		{"include.hrl.tmpl", "include/" + m.AppName + ".hrl"},
	}

	for _, t := range coreTemplates {
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, t.name, m); err != nil {
			return nil, fmt.Errorf("sdkerlang: execute template %s: %w", t.name, err)
		}
		files = append(files, &sdk.File{Path: t.path, Content: out.String()})
	}

	// Generate resource modules
	for _, r := range m.Resources {
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, "resource.erl.tmpl", struct {
			*model
			Resource resourceModel
		}{m, r}); err != nil {
			return nil, fmt.Errorf("sdkerlang: execute resource template for %s: %w", r.Name, err)
		}
		files = append(files, &sdk.File{
			Path:    "src/" + m.AppName + "_" + r.ErlangName + ".erl",
			Content: out.String(),
		})
	}

	return files, nil
}

type model struct {
	AppName     string
	Version     string
	Description string

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
	ErlangName  string
	RecordName  string
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
	ErlangName  string
	AtomName    string
	JSONName    string
	Description string
	ErlangType  string
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
	ErlangName  string
	RecordName  string
	Description string
}

type resourceModel struct {
	Name        string
	ErlangName  string
	ModuleName  string
	Description string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	ErlangName  string
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
		m.Description = toPascal(m.AppName) + " SDK"
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
			ErlangName:  toErlangTypeName(t.Name),
			RecordName:  toErlangRecordName(t.Name),
			Description: t.Description,
			Kind:        t.Kind,
			Tag:         t.Tag,
		}

		switch t.Kind {
		case contract.KindStruct:
			for _, f := range t.Fields {
				erlangType := erlangType(typeByName, f.Type, f.Optional, f.Nullable)
				typeSpec := erlangTypeSpec(typeByName, f.Type, f.Optional, f.Nullable)

				fm := fieldModel{
					Name:        f.Name,
					ErlangName:  toErlangName(f.Name),
					AtomName:    toErlangAtom(f.Name),
					JSONName:    f.Name,
					Description: f.Description,
					ErlangType:  erlangType,
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
						Atom:  toErlangAtom(e),
					})
				}

				tm.Fields = append(tm.Fields, fm)
			}

		case contract.KindSlice:
			tm.Elem = erlangType(typeByName, t.Elem, false, false)
			tm.ElemSpec = erlangTypeSpec(typeByName, t.Elem, false, false)

		case contract.KindMap:
			tm.Elem = erlangType(typeByName, t.Elem, false, false)
			tm.ElemSpec = erlangTypeSpec(typeByName, t.Elem, false, false)

		case contract.KindUnion:
			for _, v := range t.Variants {
				tm.Variants = append(tm.Variants, variantModel{
					Value:       v.Value,
					Type:        string(v.Type),
					ErlangName:  toErlangTypeName(string(v.Type)),
					RecordName:  toErlangRecordName(string(v.Type)),
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
			ErlangName:  toSnake(r.Name),
			ModuleName:  toPascal(r.Name),
			Description: r.Description,
		}

		for _, mm := range r.Methods {
			if mm == nil {
				continue
			}

			httpMethod := "post"
			httpPath := "/" + r.Name + "/" + mm.Name
			if mm.HTTP != nil {
				if strings.TrimSpace(mm.HTTP.Method) != "" {
					httpMethod = strings.ToLower(mm.HTTP.Method)
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
							ErlangName:  toErlangName(f.Name),
							AtomName:    toErlangAtom(f.Name),
							JSONName:    f.Name,
							Description: f.Description,
							ErlangType:  erlangType(typeByName, f.Type, f.Optional, f.Nullable),
							TypeSpec:    erlangTypeSpec(typeByName, f.Type, f.Optional, f.Nullable),
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
				streamItem = toErlangRecordName(strings.TrimSpace(string(mm.Stream.Item)))
			}

			rm.Methods = append(rm.Methods, methodModel{
				Name:        mm.Name,
				ErlangName:  toErlangName(mm.Name),
				Description: mm.Description,

				HasInput:      hasInput,
				HasOutput:     hasOutput,
				InputIsStruct: inputIsStruct,
				InputFields:   inputFields,
				InputType:     toErlangRecordName(string(mm.Input)),
				OutputType:    toErlangRecordName(string(mm.Output)),
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

func erlangType(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "term()"
	}

	base := baseErlangType(typeByName, r)

	if optional || nullable {
		return base + " | undefined"
	}
	return base
}

func erlangTypeSpec(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "term()"
	}

	base := baseErlangTypeSpec(typeByName, r)

	if optional || nullable {
		return base + " | undefined"
	}
	return base
}

func baseErlangType(typeByName map[string]*contract.Type, r string) string {
	// Check if it's a known type
	if _, ok := typeByName[r]; ok {
		return "#" + toErlangRecordName(r) + "{}"
	}

	switch r {
	case "string":
		return "binary()"
	case "bool", "boolean":
		return "boolean()"
	case "int", "int8", "int16", "int32", "int64":
		return "integer()"
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "non_neg_integer()"
	case "float32", "float64":
		return "float()"
	case "time.Time":
		return "calendar:datetime()"
	case "json.RawMessage":
		return "map()"
	case "any", "interface{}":
		return "term()"
	}

	// Handle slice types
	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		return "[" + baseErlangType(typeByName, elem) + "]"
	}

	// Handle map types
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		return "#{binary() => " + baseErlangType(typeByName, elem) + "}"
	}

	return "term()"
}

func baseErlangTypeSpec(typeByName map[string]*contract.Type, r string) string {
	// Check if it's a known type
	if _, ok := typeByName[r]; ok {
		return toErlangRecordName(r) + "()"
	}

	switch r {
	case "string":
		return "binary()"
	case "bool", "boolean":
		return "boolean()"
	case "int", "int8", "int16", "int32", "int64":
		return "integer()"
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "non_neg_integer()"
	case "float32", "float64":
		return "float()"
	case "time.Time":
		return "calendar:datetime()"
	case "json.RawMessage":
		return "map()"
	case "any", "interface{}":
		return "term()"
	}

	// Handle slice types
	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		return "list(" + baseErlangTypeSpec(typeByName, elem) + ")"
	}

	// Handle map types
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		return "#{binary() => " + baseErlangTypeSpec(typeByName, elem) + "}"
	}

	return "term()"
}

// toErlangName converts a string to snake_case for Erlang functions/variables.
func toErlangName(s string) string {
	if s == "" {
		return ""
	}

	result := toSnake(s)

	// Check for reserved words
	if isErlangReserved(result) {
		return result + "_"
	}

	return result
}

// toErlangTypeName converts a string to snake_case for Erlang type names.
func toErlangTypeName(s string) string {
	if s == "" {
		return ""
	}
	return toSnake(s)
}

// toErlangModuleName converts a string to a valid Erlang module name (snake_case).
func toErlangModuleName(s string) string {
	if s == "" {
		return ""
	}
	return toSnake(s)
}

// toErlangRecordName converts a string to a valid Erlang record name (snake_case).
func toErlangRecordName(s string) string {
	if s == "" {
		return ""
	}
	return toSnake(s)
}

// toErlangAtom converts a string to a valid Erlang atom.
func toErlangAtom(s string) string {
	if s == "" {
		return ""
	}

	result := toSnake(s)

	// Check if it needs quoting
	if needsQuoting(result) {
		return "'" + result + "'"
	}

	return result
}

// needsQuoting checks if an Erlang atom needs single quotes.
func needsQuoting(s string) bool {
	if s == "" {
		return false
	}

	// Must start with lowercase letter
	r := rune(s[0])
	if !unicode.IsLower(r) {
		return true
	}

	// Can only contain alphanumeric, underscore, @
	for _, c := range s {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' && c != '@' {
			return true
		}
	}

	// Reserved words need quoting
	return isErlangReserved(s)
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

// isErlangReserved checks if a name is an Erlang reserved word.
func isErlangReserved(s string) bool {
	reserved := map[string]bool{
		"after": true, "and": true, "andalso": true, "band": true,
		"begin": true, "bnot": true, "bor": true, "bsl": true,
		"bsr": true, "bxor": true, "case": true, "catch": true,
		"cond": true, "div": true, "end": true, "fun": true,
		"if": true, "let": true, "not": true, "of": true,
		"or": true, "orelse": true, "receive": true, "rem": true,
		"try": true, "when": true, "xor": true,
		// Also avoid some common atoms
		"true": true, "false": true, "undefined": true,
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

// erlangQuote returns a quoted Erlang string literal (list).
func erlangQuote(s string) string {
	escaped := strings.ReplaceAll(s, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	escaped = strings.ReplaceAll(escaped, "\n", "\\n")
	escaped = strings.ReplaceAll(escaped, "\r", "\\r")
	escaped = strings.ReplaceAll(escaped, "\t", "\\t")
	return "\"" + escaped + "\""
}

// erlangBinary returns an Erlang binary literal.
func erlangBinary(s string) string {
	escaped := strings.ReplaceAll(s, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	escaped = strings.ReplaceAll(escaped, "\n", "\\n")
	escaped = strings.ReplaceAll(escaped, "\r", "\\r")
	escaped = strings.ReplaceAll(escaped, "\t", "\\t")
	return "<<\"" + escaped + "\">>"
}

// erlangAtom returns an Erlang atom.
func erlangAtom(s string) string {
	return toErlangAtom(s)
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
