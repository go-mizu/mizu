// Package sdkpy generates a modern typed Python SDK from contract.Service.
package sdkpy

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

//go:embed templates/*.py.tmpl templates/pyproject.toml.tmpl templates/README.md.tmpl
var templateFS embed.FS

// Config controls Python SDK generation.
type Config struct {
	// Package is the python import package name.
	// Default: lowercase sanitized service name, or "sdk".
	Package string

	// Project is the python distribution name (pyproject project.name).
	// Default: same as Package.
	Project string

	// Version is the python distribution version.
	// Default: "0.1.0".
	Version string
}

// Generate produces a set of generated files for a typed Python SDK.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkpy: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("sdkpy").
		Funcs(template.FuncMap{
			"pyIdent":   pyIdent,
			"pyString":  pyString,
			"snake":     toSnake,
			"title":     toTitle,
			"join":      strings.Join,
			"sortedKVs": sortedKVs,
		}).
		ParseFS(templateFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("sdkpy: parse templates: %w", err)
	}

	var files []*sdk.File

	// Static-ish top level
	{
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, "pyproject.toml.tmpl", m); err != nil {
			return nil, fmt.Errorf("sdkpy: execute pyproject: %w", err)
		}
		files = append(files, &sdk.File{Path: "pyproject.toml", Content: out.String()})
	}
	{
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, "README.md.tmpl", m); err != nil {
			return nil, fmt.Errorf("sdkpy: execute README: %w", err)
		}
		files = append(files, &sdk.File{Path: "README.md", Content: out.String()})
	}

	// Package files
	pkgDir := m.Package
	emitPkg := func(name, tmpl string) error {
		var out bytes.Buffer
		if err := tpl.ExecuteTemplate(&out, tmpl, m); err != nil {
			return fmt.Errorf("sdkpy: execute %s: %w", tmpl, err)
		}
		files = append(files, &sdk.File{
			Path:    pkgDir + "/" + name,
			Content: out.String(),
		})
		return nil
	}

	if err := emitPkg("__init__.py", "__init__.py.tmpl"); err != nil {
		return nil, err
	}
	if err := emitPkg("_client.py", "_client.py.tmpl"); err != nil {
		return nil, err
	}
	if err := emitPkg("_resources.py", "_resources.py.tmpl"); err != nil {
		return nil, err
	}
	if err := emitPkg("_types.py", "_types.py.tmpl"); err != nil {
		return nil, err
	}
	if err := emitPkg("_streaming.py", "_streaming.py.tmpl"); err != nil {
		return nil, err
	}

	return files, nil
}

type model struct {
	Package string
	Project string
	Version string

	Service struct {
		Name      string
		Sanitized string
	}

	Defaults struct {
		BaseURL string
		Auth    string
		Headers []kv
	}

	HasStreamingSSE bool

	Types     []typeModel
	Resources []resourceModel
}

type kv struct {
	K string
	V string
}

type typeModel struct {
	Name        string
	Description string
	Kind        contract.TypeKind

	Fields   []fieldModel
	Elem     string
	Tag      string
	Variants []variantModel
}

type fieldModel struct {
	Name        string
	Description string
	PyName      string
	PyType      string
	Optional    bool
	Nullable    bool
	Enum        []string
	Const       string
}

type variantModel struct {
	Value       string
	Type        string
	Description string
}

type resourceModel struct {
	Name        string
	Description string
	PyName      string
	ClassName   string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	Description string
	PyName      string

	HasInput  bool
	HasOutput bool
	InputType string
	OutputType string

	HTTPMethod string
	HTTPPath   string

	IsStreaming bool
	StreamMode  string
	StreamIsSSE bool
	StreamItem  string

	InputIsStruct bool
	InputFields   []callField
}

type callField struct {
	Name     string // json name
	PyName   string // snake python name
	PyType   string // python type
	Optional bool
	Nullable bool
}

func buildModel(svc *contract.Service, cfg *Config) (*model, error) {
	m := &model{}

	// package + project
	if cfg != nil && cfg.Package != "" {
		m.Package = cfg.Package
	} else {
		p := strings.ToLower(sanitizeIdent(svc.Name))
		if p == "" {
			p = "sdk"
		}
		m.Package = p
	}

	if cfg != nil && cfg.Project != "" {
		m.Project = cfg.Project
	} else {
		m.Project = m.Package
	}

	if cfg != nil && cfg.Version != "" {
		m.Version = cfg.Version
	} else {
		m.Version = "0.1.0"
	}

	m.Service.Name = svc.Name
	m.Service.Sanitized = sanitizeIdent(svc.Name)

	// defaults
	if svc.Defaults != nil {
		m.Defaults.BaseURL = strings.TrimRight(strings.TrimSpace(svc.Defaults.BaseURL), "/")
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

	typeByName := map[string]*contract.Type{}
	for _, t := range svc.Types {
		if t != nil && t.Name != "" {
			typeByName[t.Name] = t
		}
	}

	// streaming presence (SSE)
	for _, r := range svc.Resources {
		if r == nil {
			continue
		}
		for _, mm := range r.Methods {
			if mm == nil || mm.Stream == nil {
				continue
			}
			mode := strings.TrimSpace(mm.Stream.Mode)
			if mode == "" || strings.EqualFold(mode, "sse") {
				m.HasStreamingSSE = true
				break
			}
		}
	}

	// types (stable order)
	typeNames := make([]string, 0, len(typeByName))
	for n := range typeByName {
		typeNames = append(typeNames, n)
	}
	sort.Strings(typeNames)

	for _, n := range typeNames {
		t := typeByName[n]
		if t == nil {
			continue
		}
		tm := typeModel{
			Name:        t.Name,
			Description: t.Description,
			Kind:        t.Kind,
			Elem:        pyType(typeByName, t.Elem, false, false),
			Tag:         t.Tag,
		}

		switch t.Kind {
		case contract.KindStruct:
			for _, f := range t.Fields {
				tm.Fields = append(tm.Fields, fieldModel{
					Name:        f.Name,
					Description: f.Description,
					PyName:      toSnake(f.Name),
					PyType:      pyType(typeByName, f.Type, f.Optional, f.Nullable),
					Optional:    f.Optional,
					Nullable:    f.Nullable,
					Enum:        append([]string(nil), f.Enum...),
					Const:       f.Const,
				})
			}
		case contract.KindSlice, contract.KindMap:
			// Elem already set
		case contract.KindUnion:
			for _, v := range t.Variants {
				tm.Variants = append(tm.Variants, variantModel{
					Value:       v.Value,
					Type:        string(v.Type),
					Description: v.Description,
				})
			}
		}

		m.Types = append(m.Types, tm)
	}

	// resources
	for _, r := range svc.Resources {
		if r == nil {
			continue
		}
		rm := resourceModel{
			Name:        r.Name,
			Description: r.Description,
			PyName:      toSnake(r.Name),
			ClassName:   toTitle(toSnake(r.Name)) + "Resource",
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
				streamItem = strings.TrimSpace(string(mm.Stream.Item))
			}

			inType := strings.TrimSpace(string(mm.Input))
			outType := strings.TrimSpace(string(mm.Output))

			mmModel := methodModel{
				Name:        mm.Name,
				Description: mm.Description,
				PyName:      toSnake(mm.Name),

				HasInput:   hasInput,
				HasOutput:  hasOutput,
				InputType:  inType,
				OutputType: outType,

				HTTPMethod: httpMethod,
				HTTPPath:   httpPath,

				IsStreaming: isStreaming,
				StreamMode:  streamMode,
				StreamIsSSE: streamIsSSE,
				StreamItem:  streamItem,
			}

			// input struct expansion into kwargs (nice DX)
			if hasInput {
				if it, ok := typeByName[inType]; ok && it != nil && it.Kind == contract.KindStruct {
					mmModel.InputIsStruct = true
					for _, f := range it.Fields {
						mmModel.InputFields = append(mmModel.InputFields, callField{
							Name:     f.Name,
							PyName:   toSnake(f.Name),
							PyType:   pyType(typeByName, f.Type, f.Optional, f.Nullable),
							Optional: f.Optional,
							Nullable: f.Nullable,
						})
					}
				}
			}

			rm.Methods = append(rm.Methods, mmModel)
		}

		m.Resources = append(m.Resources, rm)
	}

	return m, nil
}

func pyType(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "Any"
	}

	base := basePyType(typeByName, r)

	if optional {
		return "Optional[" + base + "]"
	}
	if nullable {
		return "Optional[" + base + "]"
	}
	return base
}

func basePyType(typeByName map[string]*contract.Type, r string) string {
	if _, ok := typeByName[r]; ok {
		return r
	}

	// Contract may include Go-ish collection syntax. Support same ones used in your Go generator.
	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		return "List[" + basePyType(typeByName, elem) + "]"
	}
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		return "Dict[str, " + basePyType(typeByName, elem) + "]"
	}

	switch r {
	case "string":
		return "str"
	case "bool", "boolean":
		return "bool"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return "int"
	case "float32", "float64":
		return "float"
	case "time.Time":
		return "datetime"
	case "json.RawMessage", "any", "interface{}":
		return "Any"
	}

	return "Any"
}

func toSnake(s string) string {
	if s == "" {
		return ""
	}
	// basic snake conversion: split on - _ . and case transitions
	var out strings.Builder
	prevLower := false
	for _, r := range s {
		if r == '-' || r == '_' || r == '.' || r == ' ' {
			if out.Len() > 0 && out.String()[out.Len()-1] != '_' {
				out.WriteByte('_')
			}
			prevLower = false
			continue
		}
		if unicode.IsUpper(r) && prevLower {
			out.WriteByte('_')
		}
		out.WriteRune(unicode.ToLower(r))
		prevLower = unicode.IsLower(r) || unicode.IsDigit(r)
	}
	return strings.Trim(out.String(), "_")
}

func toTitle(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "_")
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		runes := []rune(parts[i])
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, "")
}

func sanitizeIdent(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func pyIdent(s string) string {
	// simple: ensure not empty and avoid hyphens/spaces
	s = toSnake(s)
	if s == "" {
		return "x"
	}
	if unicode.IsDigit([]rune(s)[0]) {
		return "x_" + s
	}
	return s
}

func pyString(s string) string {
	// emits a JSON-safe python string literal via %q semantics
	return fmt.Sprintf("%q", s)
}

func sortedKVs(in []kv) []kv {
	out := append([]kv(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].K < out[j].K })
	return out
}
