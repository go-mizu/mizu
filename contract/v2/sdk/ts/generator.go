// Package sdkts generates a modern TypeScript SDK from contract.Service.
package sdkts

import (
	"bytes"
	"embed"
	"fmt"
	"path"
	"sort"
	"strings"
	"text/template"
	"unicode"

	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/sdk"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// Config controls TypeScript SDK generation.
type Config struct {
	// Package is the npm package name.
	// Default: sanitized lowercase service name, or "sdk".
	Package string

	// Version is the package version written to package.json.
	// Default: "0.0.0".
	Version string
}

// Generate produces a set of files for a TypeScript SDK.
// Output is an npm-compatible project with ES modules.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkts: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl := template.New("sdkts").Funcs(template.FuncMap{
		"tsQuote":  tsQuote,
		"tsString": tsQuote,
		"tsIdent":  tsIdent,
		"camel":    toCamel,
		"pascal":   toPascal,
		"join":     strings.Join,
		"trim":     strings.TrimSpace,
		"lower":    strings.ToLower,
		"indent":   indent,
	})

	need := []string{
		"templates/package.json.tmpl",
		"templates/tsconfig.json.tmpl",
		"templates/_client.ts.tmpl",
		"templates/_types.ts.tmpl",
		"templates/_streaming.ts.tmpl",
		"templates/_resources.ts.tmpl",
		"templates/index.ts.tmpl",
	}
	var errParse error
	tpl, errParse = tpl.ParseFS(templateFS, need...)
	if errParse != nil {
		return nil, fmt.Errorf("sdkts: parse templates: %w", errParse)
	}

	outPlan := []struct {
		Path string
		Tpl  string
	}{
		{Path: "package.json", Tpl: "package.json.tmpl"},
		{Path: "tsconfig.json", Tpl: "tsconfig.json.tmpl"},

		{Path: "src/index.ts", Tpl: "index.ts.tmpl"},
		{Path: "src/_client.ts", Tpl: "_client.ts.tmpl"},
		{Path: "src/_types.ts", Tpl: "_types.ts.tmpl"},
		{Path: "src/_streaming.ts", Tpl: "_streaming.ts.tmpl"},
		{Path: "src/_resources.ts", Tpl: "_resources.ts.tmpl"},
	}

	var files []*sdk.File
	for _, item := range outPlan {
		var buf bytes.Buffer
		if err := tpl.ExecuteTemplate(&buf, item.Tpl, m); err != nil {
			return nil, fmt.Errorf("sdkts: execute template %s: %w", item.Tpl, err)
		}
		files = append(files, &sdk.File{
			Path:    path.Clean(item.Path),
			Content: buf.String(),
		})
	}

	return files, nil
}

type model struct {
	Package string
	Version string

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

	Types     []typeModel
	Resources []resourceModel

	HasSSE bool
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
	TSName      string
	Description string
	TSType      string

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
	TSName      string
	ClassName   string
	Description string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	TSName      string
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
	StreamItem     string
}

func buildModel(svc *contract.Service, cfg *Config) (*model, error) {
	m := &model{}

	if cfg != nil && strings.TrimSpace(cfg.Package) != "" {
		m.Package = tsIdent(cfg.Package)
	} else {
		p := strings.ToLower(sanitizeIdent(svc.Name))
		if p == "" {
			p = "sdk"
		}
		m.Package = tsIdent(p)
	}

	if cfg != nil && strings.TrimSpace(cfg.Version) != "" {
		m.Version = strings.TrimSpace(cfg.Version)
	} else {
		m.Version = "0.0.0"
	}

	m.Service.Name = svc.Name
	m.Service.Sanitized = sanitizeIdent(svc.Name)
	m.Service.Description = svc.Description

	if svc.Client != nil {
		m.Client.BaseURL = strings.TrimRight(strings.TrimSpace(svc.Client.BaseURL), "/")
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

	typeByName := map[string]*contract.Type{}
	for _, t := range svc.Types {
		if t != nil && strings.TrimSpace(t.Name) != "" {
			typeByName[t.Name] = t
		}
	}

	m.HasSSE = hasSSE(svc)

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
				tm.Fields = append(tm.Fields, fieldModel{
					Name:        f.Name,
					TSName:      toCamel(f.Name),
					Description: f.Description,
					TSType:      tsType(typeByName, f.Type, f.Optional, f.Nullable, f.Enum, f.Const),
					Optional:    f.Optional,
					Nullable:    f.Nullable,
					Enum:        append([]string(nil), f.Enum...),
					Const:       f.Const,
				})
			}

		case contract.KindSlice:
			tm.Elem = tsType(typeByName, t.Elem, false, false, nil, "")

		case contract.KindMap:
			tm.Elem = tsType(typeByName, t.Elem, false, false, nil, "")

		case contract.KindUnion:
			for _, v := range t.Variants {
				tm.Variants = append(tm.Variants, variantModel{
					Value:       v.Value,
					Type:        string(v.Type),
					Description: v.Description,
					FieldName:   toPascal(string(v.Type)),
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
			TSName:      toCamel(r.Name),
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
							TSName:      toCamel(f.Name),
							Description: f.Description,
							TSType:      tsType(typeByName, f.Type, f.Optional, f.Nullable, f.Enum, f.Const),
							Optional:    f.Optional,
							Nullable:    f.Nullable,
							Enum:        append([]string(nil), f.Enum...),
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
				streamItem = strings.TrimSpace(string(mm.Stream.Item))
			}

			rm.Methods = append(rm.Methods, methodModel{
				Name:        mm.Name,
				TSName:      toCamel(mm.Name),
				Description: mm.Description,

				HasInput:      hasInput,
				HasOutput:     hasOutput,
				InputIsStruct: inputIsStruct,
				InputFields:   inputFields,
				InputType:     string(mm.Input),
				OutputType:    string(mm.Output),

				HTTPMethod: httpMethod,
				HTTPPath:   httpPath,

				IsStreaming:    isStreaming,
				StreamMode:     streamMode,
				StreamIsSSE:    streamIsSSE,
				StreamItemType: streamItem,
				StreamItem:     streamItem,
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

func tsType(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool, enum []string, constVal string) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "unknown"
	}

	// Handle const values - they become literal types
	if constVal != "" {
		return fmt.Sprintf("%q", constVal)
	}

	// Handle enum values - they become union of literal types
	if len(enum) > 0 {
		literals := make([]string, len(enum))
		for i, e := range enum {
			literals[i] = fmt.Sprintf("%q", e)
		}
		base := strings.Join(literals, " | ")
		if nullable {
			return base + " | null"
		}
		return base
	}

	base := baseTSType(typeByName, r)
	if nullable {
		return base + " | null"
	}
	return base
}

func baseTSType(typeByName map[string]*contract.Type, r string) string {
	if _, ok := typeByName[r]; ok {
		return r
	}

	switch r {
	case "string":
		return "string"
	case "bool", "boolean":
		return "boolean"
	case "int", "int8", "int16", "int32", "int64":
		return "number"
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "number"
	case "float32", "float64":
		return "number"
	case "time.Time":
		return "string" // ISO 8601 string on wire
	case "json.RawMessage", "any", "interface{}":
		return "unknown"
	}

	if strings.HasPrefix(r, "[]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "[]"))
		return baseTSType(typeByName, elem) + "[]"
	}
	if strings.HasPrefix(r, "map[string]") {
		elem := strings.TrimSpace(strings.TrimPrefix(r, "map[string]"))
		return "Record<string, " + baseTSType(typeByName, elem) + ">"
	}

	return "unknown"
}

func tsQuote(s string) string { return fmt.Sprintf("%q", s) }

func tsIdent(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "sdk"
	}
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "sdk"
	}
	r0 := rune(out[0])
	if !(unicode.IsLetter(r0) || r0 == '_') {
		out = "_" + out
	}
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

func toCamel(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	capNext := false
	for i, r := range s {
		if r == '_' || r == '-' || r == '.' || r == ' ' {
			capNext = true
			continue
		}
		if i == 0 {
			b.WriteRune(unicode.ToLower(r))
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
	if out == "" {
		return "x"
	}
	return out
}

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
		} else {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "X"
	}
	return out
}

func indent(n int, s string) string {
	if n <= 0 || s == "" {
		return s
	}
	pad := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i := range lines {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		lines[i] = pad + lines[i]
	}
	return strings.Join(lines, "\n")
}
