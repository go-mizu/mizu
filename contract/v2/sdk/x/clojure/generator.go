// Package sdkclojure generates typed Clojure SDK clients from contract.Service.
package sdkclojure

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

// Config controls Clojure SDK generation.
type Config struct {
	// Namespace is the root Clojure namespace.
	// Default: kebab-case service name (e.g., "my-api").
	Namespace string

	// GroupId for Maven/Clojars deployment.
	// Default: "com.example".
	GroupId string

	// ArtifactId for Maven/Clojars deployment.
	// Default: kebab-case service name.
	ArtifactId string

	// Version of the generated SDK.
	// Default: "0.0.0".
	Version string

	// GenerateSpecs enables clojure.spec generation.
	// Default: true.
	GenerateSpecs bool
}

// Generate produces a set of generated files for a typed Clojure SDK client.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkclojure: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("sdkclojure").
		Funcs(template.FuncMap{
			"cljQuote":    cljQuote,
			"cljString":   cljQuote,
			"cljName":     toCljName,
			"cljTypeName": toCljTypeName,
			"cljKeyword":  toCljKeyword,
			"camel":       toCamel,
			"pascal":      toPascal,
			"kebab":       toKebab,
			"snake":       toSnake,
			"httpMethod":  strings.ToLower,
			"upper":       strings.ToUpper,
			"lower":       strings.ToLower,
			"join":        strings.Join,
			"trim":        strings.TrimSpace,
			"indent":      indent,
			"hasPrefix":   strings.HasPrefix,
			"hasSuffix":   strings.HasSuffix,
			"add":         func(a, b int) int { return a + b },
			"sub":         func(a, b int) int { return a - b },
			"len":         lenHelper,
			"nsPath":      nsToPath,
		}).
		ParseFS(templateFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("sdkclojure: parse templates: %w", err)
	}

	files := make([]*sdk.File, 0, 8)

	// Build namespace path (e.g., "my-api" -> "my_api")
	nsPath := nsToPath(m.Namespace)

	// Generate each file from its template
	templates := []struct {
		name string
		path string
	}{
		{"deps.edn.tmpl", "deps.edn"},
		{"core.clj.tmpl", "src/" + nsPath + "/core.clj"},
		{"types.clj.tmpl", "src/" + nsPath + "/types.clj"},
		{"resources.clj.tmpl", "src/" + nsPath + "/resources.clj"},
		{"streaming.clj.tmpl", "src/" + nsPath + "/streaming.clj"},
		{"errors.clj.tmpl", "src/" + nsPath + "/errors.clj"},
	}

	// Add spec file if enabled
	if m.GenerateSpecs {
		templates = append(templates, struct {
			name string
			path string
		}{"spec.clj.tmpl", "src/" + nsPath + "/spec.clj"})
	}

	for _, t := range templates {
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, t.name, m); err != nil {
			return nil, fmt.Errorf("sdkclojure: execute template %s: %w", t.name, err)
		}
		files = append(files, &sdk.File{Path: t.path, Content: out.String()})
	}

	return files, nil
}

type model struct {
	Namespace     string
	NsPath        string
	GroupId       string
	ArtifactId    string
	Version       string
	GenerateSpecs bool

	Service struct {
		Name        string
		Sanitized   string
		CljName     string
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
	CljName     string
	RecordName  string
	Description string
	Kind        contract.TypeKind

	Fields   []fieldModel
	Elem     string
	ElemClj  string
	Tag      string
	Variants []variantModel
}

type fieldModel struct {
	Name        string
	CljName     string
	CljKeyword  string
	JSONName    string
	Description string
	CljType     string

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
	CljName     string
	Description string
}

type resourceModel struct {
	Name        string
	CljName     string
	Description string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	CljName     string
	FullName    string
	Description string

	HasInput  bool
	HasOutput bool

	InputType   string
	OutputType  string
	InputClj    string
	OutputClj   string

	HTTPMethod string
	HTTPPath   string

	IsStreaming    bool
	StreamMode     string
	StreamIsSSE    bool
	StreamItemType string
	StreamItemClj  string
}

func buildModel(svc *contract.Service, cfg *Config) (*model, error) {
	m := &model{}

	// Namespace
	if cfg != nil && cfg.Namespace != "" {
		m.Namespace = cfg.Namespace
	} else {
		ns := toKebab(sanitizeIdent(svc.Name))
		if ns == "" {
			ns = "sdk"
		}
		m.Namespace = ns
	}
	m.NsPath = nsToPath(m.Namespace)

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

	// GenerateSpecs
	if cfg != nil {
		m.GenerateSpecs = cfg.GenerateSpecs
	} else {
		m.GenerateSpecs = true
	}

	// Service info
	m.Service.Name = svc.Name
	m.Service.Sanitized = toPascal(sanitizeIdent(svc.Name))
	m.Service.CljName = toKebab(sanitizeIdent(svc.Name))
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
			CljName:     toKebab(t.Name),
			RecordName:  toPascal(t.Name),
			Description: t.Description,
			Kind:        t.Kind,
			Tag:         t.Tag,
		}

		switch t.Kind {
		case contract.KindStruct:
			for _, f := range t.Fields {
				fm := fieldModel{
					Name:        f.Name,
					CljName:     toKebab(f.Name),
					CljKeyword:  toCljKeyword(f.Name),
					JSONName:    f.Name,
					Description: f.Description,
					CljType:     cljType(typeByName, f.Type, f.Optional, f.Nullable),
					Optional:    f.Optional,
					Nullable:    f.Nullable,
					Const:       f.Const,
				}

				// Handle enum values
				for _, e := range f.Enum {
					fm.Enum = append(fm.Enum, enumValue{
						Name:  toCljKeyword(e),
						Value: e,
					})
				}

				tm.Fields = append(tm.Fields, fm)
			}

		case contract.KindSlice:
			tm.Elem = string(t.Elem)
			tm.ElemClj = cljType(typeByName, t.Elem, false, false)

		case contract.KindMap:
			tm.Elem = string(t.Elem)
			tm.ElemClj = cljType(typeByName, t.Elem, false, false)

		case contract.KindUnion:
			for _, v := range t.Variants {
				tm.Variants = append(tm.Variants, variantModel{
					Value:       v.Value,
					Type:        string(v.Type),
					CljName:     toCljKeyword(v.Value),
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
			CljName:     toKebab(r.Name),
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

			isStreaming := mm.Stream != nil
			streamMode := ""
			streamIsSSE := false
			streamItem := ""
			streamItemClj := ""

			if isStreaming {
				streamMode = strings.TrimSpace(mm.Stream.Mode)
				streamIsSSE = streamMode == "" || strings.EqualFold(streamMode, "sse")
				streamItem = strings.TrimSpace(string(mm.Stream.Item))
				streamItemClj = toCljTypeName(streamItem)
			}

			rm.Methods = append(rm.Methods, methodModel{
				Name:        mm.Name,
				CljName:     toKebab(mm.Name),
				FullName:    toKebab(r.Name) + "-" + toKebab(mm.Name),
				Description: mm.Description,

				HasInput:    hasInput,
				HasOutput:   hasOutput,
				InputType:   string(mm.Input),
				OutputType:  string(mm.Output),
				InputClj:    toCljTypeName(string(mm.Input)),
				OutputClj:   toCljTypeName(string(mm.Output)),
				HTTPMethod:  httpMethod,
				HTTPPath:    httpPath,
				IsStreaming: isStreaming,

				StreamMode:     streamMode,
				StreamIsSSE:    streamIsSSE,
				StreamItemType: streamItem,
				StreamItemClj:  streamItemClj,
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

func cljType(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "any?"
	}

	base := baseCljType(typeByName, r)

	// In Clojure, optional/nullable just means the value can be nil
	// No wrapper type needed - specs handle this
	if optional || nullable {
		return "(s/nilable " + base + ")"
	}
	return base
}

func baseCljType(typeByName map[string]*contract.Type, r string) string {
	// Check if it's a known type
	if _, ok := typeByName[r]; ok {
		return "::" + toKebab(r)
	}

	switch r {
	case "string":
		return "string?"
	case "bool", "boolean":
		return "boolean?"
	case "int":
		return "int?"
	case "int8", "int16", "int32":
		return "int?"
	case "int64":
		return "int?"
	case "uint", "uint8", "uint16", "uint32":
		return "nat-int?"
	case "uint64":
		return "nat-int?"
	case "float32", "float64":
		return "number?"
	case "time.Time":
		return "inst?"
	case "json.RawMessage":
		return "any?"
	case "any", "interface{}":
		return "any?"
	}

	// Handle slice types
	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		return "(s/coll-of " + baseCljType(typeByName, elem) + ")"
	}

	// Handle map types
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		return "(s/map-of keyword? " + baseCljType(typeByName, elem) + ")"
	}

	return "any?"
}

// toCljName converts a string to kebab-case for Clojure functions/vars.
func toCljName(s string) string {
	if s == "" {
		return ""
	}

	result := toKebab(s)

	// Check for reserved words
	if isCljReserved(result) {
		return result + "-val"
	}

	return result
}

// toCljTypeName converts a string to PascalCase for Clojure records.
func toCljTypeName(s string) string {
	if s == "" {
		return ""
	}

	return toPascal(s)
}

// toCljKeyword converts a string to a kebab-case keyword.
func toCljKeyword(s string) string {
	if s == "" {
		return ""
	}

	return ":" + toKebab(s)
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

// toSnake converts to snake_case.
func toSnake(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	prevWasUpper := false

	for i, r := range s {
		if r == '-' || r == '.' || r == ' ' {
			b.WriteRune('_')
			prevWasUpper = false
			continue
		}

		if unicode.IsUpper(r) {
			if i > 0 && !prevWasUpper {
				b.WriteRune('_')
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

// isCljReserved checks if a name is a Clojure reserved word or core function.
func isCljReserved(s string) bool {
	reserved := map[string]bool{
		// Special forms
		"def": true, "if": true, "do": true, "let": true, "quote": true,
		"var": true, "fn": true, "loop": true, "recur": true, "throw": true,
		"try": true, "catch": true, "finally": true, "monitor-enter": true,
		"monitor-exit": true, "new": true, "set!": true,
		// Core macros/functions that are commonly used
		"ns": true, "import": true, "require": true, "use": true,
		"defn": true, "defmacro": true, "defonce": true, "defprotocol": true,
		"defrecord": true, "deftype": true, "defmulti": true, "defmethod": true,
		"cond": true, "case": true, "when": true, "when-let": true,
		"if-let": true, "if-not": true, "when-not": true,
		"and": true, "or": true, "not": true,
		// Commonly shadowed names
		"name": true, "type": true, "count": true, "first": true, "rest": true,
		"map": true, "filter": true, "reduce": true, "get": true, "assoc": true,
		"update": true, "keys": true, "vals": true, "merge": true, "into": true,
		"conj": true, "cons": true, "list": true, "vector": true, "set": true,
		"str": true, "print": true, "println": true, "pr": true, "prn": true,
		"format": true, "read": true, "load": true, "apply": true, "partial": true,
		"comp": true, "identity": true, "constantly": true,
		"true": true, "false": true, "nil": true,
		"some": true, "any": true, "every": true,
		"class": true, "meta": true, "with-meta": true,
		"atom": true, "ref": true, "agent": true, "deref": true,
		"swap!": true, "reset!": true, "alter": true, "send": true,
		"future": true, "promise": true, "deliver": true,
		"time": true, "delay": true, "force": true,
	}
	return reserved[s]
}

// sanitizeIdent removes invalid characters from an identifier.
func sanitizeIdent(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// cljQuote returns a quoted Clojure string literal.
func cljQuote(s string) string {
	return fmt.Sprintf("%q", s)
}

// nsToPath converts a Clojure namespace to a file path.
// e.g., "my-api.core" -> "my_api/core"
func nsToPath(ns string) string {
	// Replace dots with slashes, hyphens with underscores
	result := strings.ReplaceAll(ns, ".", "/")
	result = strings.ReplaceAll(result, "-", "_")
	return result
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
