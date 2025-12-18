// Package sdkgo generates typed Go SDK clients from contract.Service.
package sdkgo

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

//go:embed templates/*.go.tmpl
var templateFS embed.FS

// Config controls Go SDK generation.
type Config struct {
	// Package is the Go package name for generated code.
	// Default: lowercase sanitized service name, or "sdk".
	Package string

	// Filename is the output file path.
	// Default: "client.go".
	Filename string
}

// Generate produces a set of generated files for a typed Go SDK client.
// The output is intentionally small: by default a single Go file in one package.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkgo: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("sdkgo").
		Funcs(template.FuncMap{
			"quote":  quote,
			"goName": toGoName,
			"join":   strings.Join,
		}).
		ParseFS(templateFS, "templates/*.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("sdkgo: parse templates: %w", err)
	}

	var out bytes.Buffer
	if err := tpl.ExecuteTemplate(&out, "client.go.tmpl", m); err != nil {
		return nil, fmt.Errorf("sdkgo: execute template: %w", err)
	}

	filename := "client.go"
	if cfg != nil && cfg.Filename != "" {
		filename = cfg.Filename
	}

	return []*sdk.File{
		{Path: filename, Content: out.String()},
	}, nil
}

type model struct {
	Package string

	Service struct {
		Name        string
		Sanitized   string
		Description string
	}

	Defaults struct {
		BaseURL  string
		Auth    string
		Headers []kv
	}

	Imports []string

	HasTime bool
	HasSSE  bool

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
	GoName      string
	Description string
	GoType      string
	Tag         string

	Optional bool
	Nullable bool
	Enum     []string
	Const    string
}

type variantModel struct {
	Value       string
	Type        string
	Description string
	FieldName   string
}

type resourceModel struct {
	Name        string
	GoName      string
	Description string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	GoName      string
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

	if cfg != nil && cfg.Package != "" {
		m.Package = cfg.Package
	} else {
		p := strings.ToLower(sanitizeIdent(svc.Name))
		if p == "" {
			p = "sdk"
		}
		m.Package = p
	}

	m.Service.Name = svc.Name
	m.Service.Sanitized = sanitizeIdent(svc.Name)
	m.Service.Description = svc.Description

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

	typeByName := map[string]*contract.Type{}
	for _, t := range svc.Types {
		if t != nil && t.Name != "" {
			typeByName[t.Name] = t
		}
	}

	m.HasSSE = hasSSE(svc)
	m.HasTime = hasTime(svc)

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
			Description: t.Description,
			Kind:        t.Kind,
			Elem:        string(t.Elem),
			Tag:         t.Tag,
		}

		switch t.Kind {
		case contract.KindStruct:
			for _, f := range t.Fields {
				tag := f.Name
				if f.Optional {
					tag += ",omitempty"
				}
				tm.Fields = append(tm.Fields, fieldModel{
					Name:        f.Name,
					GoName:      toGoName(f.Name),
					Description: f.Description,
					GoType:      goType(typeByName, f.Type, f.Optional, f.Nullable),
					Tag:         tag,
					Optional:    f.Optional,
					Nullable:    f.Nullable,
					Enum:        append([]string(nil), f.Enum...),
					Const:       f.Const,
				})
			}

		case contract.KindSlice:
			tm.Elem = goType(typeByName, t.Elem, false, false)

		case contract.KindMap:
			tm.Elem = goType(typeByName, t.Elem, false, false)

		case contract.KindUnion:
			for _, v := range t.Variants {
				tm.Variants = append(tm.Variants, variantModel{
					Value:       v.Value,
					Type:        string(v.Type),
					Description: v.Description,
					FieldName:   toGoName(string(v.Type)),
				})
			}
		}

		m.Types = append(m.Types, tm)
	}

	for _, r := range svc.Resources {
		if r == nil {
			continue
		}
		rm := resourceModel{
			Name:        r.Name,
			GoName:      toGoName(r.Name),
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
				streamItem = strings.TrimSpace(string(mm.Stream.Item))
			}

			rm.Methods = append(rm.Methods, methodModel{
				Name:        mm.Name,
				GoName:      toGoName(mm.Name),
				Description: mm.Description,

				HasInput:    hasInput,
				HasOutput:   hasOutput,
				InputType:   string(mm.Input),
				OutputType:  string(mm.Output),
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

	m.Imports = computeImports(m.HasSSE, m.HasTime)
	return m, nil
}

func computeImports(hasSSE, hasTime bool) []string {
	imports := []string{
		"bytes",
		"context",
		"encoding/json",
		"fmt",
		"io",
		"net/http",
		"strings",
	}
	if hasSSE {
		imports = append(imports, "bufio")
	}
	if hasTime {
		imports = append(imports, "time")
	}
	sort.Strings(imports)
	return imports
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

func hasTime(svc *contract.Service) bool {
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

func goType(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "json.RawMessage"
	}

	base := baseGoType(typeByName, r)

	if optional || nullable {
		if !strings.HasPrefix(base, "[]") &&
			!strings.HasPrefix(base, "map[") &&
			base != "json.RawMessage" &&
			base != "any" &&
			base != "interface{}" {
			return "*" + base
		}
	}
	return base
}

func baseGoType(typeByName map[string]*contract.Type, r string) string {
	if _, ok := typeByName[r]; ok {
		return r
	}

	switch r {
	case "string":
		return "string"
	case "bool", "boolean":
		return "bool"
	case "int":
		return "int"
	case "int8":
		return "int8"
	case "int16":
		return "int16"
	case "int32":
		return "int32"
	case "int64":
		return "int64"
	case "uint":
		return "uint"
	case "uint8":
		return "uint8"
	case "uint16":
		return "uint16"
	case "uint32":
		return "uint32"
	case "uint64":
		return "uint64"
	case "float32":
		return "float32"
	case "float64":
		return "float64"
	case "time.Time":
		return "time.Time"
	case "json.RawMessage":
		return "json.RawMessage"
	case "any":
		return "any"
	case "interface{}":
		return "interface{}"
	}

	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		return "[]" + baseGoType(typeByName, elem)
	}
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		return "map[string]" + baseGoType(typeByName, elem)
	}

	return "json.RawMessage"
}

func toGoName(s string) string {
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

	out := b.String()
	out = strings.ReplaceAll(out, "Id", "ID")
	out = strings.ReplaceAll(out, "Url", "URL")
	out = strings.ReplaceAll(out, "Http", "HTTP")
	out = strings.ReplaceAll(out, "Api", "API")
	out = strings.ReplaceAll(out, "Sse", "SSE")
	out = strings.ReplaceAll(out, "Json", "JSON")
	return out
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

func quote(s string) string {
	return fmt.Sprintf("%q", s)
}
