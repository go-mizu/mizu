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

// Templates are organized to mirror output paths.
//
//go:embed templates/**/*.tmpl templates/*.tmpl
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
// Output is ESM-first and fetch-based for Node.js, Bun, and Deno.
func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error) {
	if svc == nil {
		return nil, fmt.Errorf("sdkts: nil service")
	}

	m, err := buildModel(svc, cfg)
	if err != nil {
		return nil, err
	}

	tpl := template.New("sdkts").Funcs(template.FuncMap{
		"tsQuote": tsQuote,
		"tsIdent": tsIdent,
		"snake":   toSnake,
		"pascal":  toPascal,
		"camel":   toLowerCamel,
		"join":    strings.Join,
		"trim":    strings.TrimSpace,
		"lower":   strings.ToLower,
		"indent":  indent,
	})

	// Parse templates by explicit paths (stable, avoids surprises).
	paths := []string{
		"templates/package.json.tmpl",
		"templates/tsconfig.json.tmpl",

		"templates/src/index.ts.tmpl",
		"templates/src/client.ts.tmpl",
		"templates/src/core.ts.tmpl",
		"templates/src/errors.ts.tmpl",
		"templates/src/streaming.ts.tmpl",
		"templates/src/types.ts.tmpl",

		"templates/src/resources/resource.ts.tmpl",
	}

	tpl, err = tpl.ParseFS(templateFS, paths...)
	if err != nil {
		return nil, fmt.Errorf("sdkts: parse templates: %w", err)
	}

	var files []*sdk.File

	// Top-level files.
	files = append(files, execToFile(tpl, "templates/package.json.tmpl", "package.json", m))
	files = append(files, execToFile(tpl, "templates/tsconfig.json.tmpl", "tsconfig.json", m))

	// src files.
	files = append(files, execToFile(tpl, "templates/src/index.ts.tmpl", "src/index.ts", m))
	files = append(files, execToFile(tpl, "templates/src/client.ts.tmpl", "src/client.ts", m))
	files = append(files, execToFile(tpl, "templates/src/core.ts.tmpl", "src/core.ts", m))
	files = append(files, execToFile(tpl, "templates/src/errors.ts.tmpl", "src/errors.ts", m))
	files = append(files, execToFile(tpl, "templates/src/streaming.ts.tmpl", "src/streaming.ts", m))
	files = append(files, execToFile(tpl, "templates/src/types.ts.tmpl", "src/types.ts", m))

	// Per-resource.
	for _, r := range m.Resources {
		rc := resourceCtx{
			Model:    m,
			Resource: r,
		}
		outPath := "src/resources/" + r.ClassName + ".ts"
		files = append(files, execToFile(tpl, "templates/src/resources/resource.ts.tmpl", outPath, rc))
	}

	return files, nil
}

func execToFile(tpl *template.Template, tplName, outPath string, data any) *sdk.File {
	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, tplName, data); err != nil {
		panic(fmt.Sprintf("sdkts: execute template %s: %v", tplName, err))
	}
	return &sdk.File{
		Path:    path.Clean(outPath),
		Content: buf.String(),
	}
}

type model struct {
	Package string
	Version string

	Service struct {
		Name        string
		Sanitized   string
		Description string
	}

	Defaults struct {
		BaseURL string
		Auth    string
		Headers []kv
	}

	Types     []typeModel
	Resources []resourceModel

	HasSSE bool
}

type resourceCtx struct {
	Model    *model
	Resource resourceModel
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
}

type resourceModel struct {
	Name        string
	PropName    string // client.<propName>
	ClassName   string // exported class name
	Description string
	Methods     []methodModel
}

type methodModel struct {
	Name        string
	PropName    string
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

	// Package name.
	if cfg != nil && strings.TrimSpace(cfg.Package) != "" {
		m.Package = strings.TrimSpace(cfg.Package)
	} else {
		p := strings.ToLower(sanitizeIdent(svc.Name))
		if p == "" {
			p = "sdk"
		}
		m.Package = p
	}

	// Version.
	if cfg != nil && strings.TrimSpace(cfg.Version) != "" {
		m.Version = strings.TrimSpace(cfg.Version)
	} else {
		m.Version = "0.0.0"
	}

	m.Service.Name = svc.Name
	m.Service.Sanitized = sanitizeIdent(svc.Name)
	m.Service.Description = svc.Description

	// Defaults.
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

	// Index types by name.
	typeByName := map[string]*contract.Type{}
	for _, t := range svc.Types {
		if t != nil && strings.TrimSpace(t.Name) != "" {
			typeByName[t.Name] = t
		}
	}

	// Streaming presence.
	m.HasSSE = hasSSE(svc)

	// Stable type order.
	typeNames := make([]string, 0, len(typeByName))
	for name := range typeByName {
		typeNames = append(typeNames, name)
	}
	sort.Strings(typeNames)

	// Types model.
	for _, name := range typeNames {
		t := typeByName[name]
		if t == nil {
			continue
		}

		tm := typeModel{
			Name:        toPascal(t.Name),
			Description: strings.TrimSpace(t.Description),
			Kind:        t.Kind,
			Elem:        "",
			Tag:         strings.TrimSpace(t.Tag),
		}

		switch t.Kind {
		case contract.KindStruct:
			for _, f := range t.Fields {
				tsName := toLowerCamel(sanitizeIdent(f.Name))
				if tsName == "" {
					tsName = "x"
				}
				tm.Fields = append(tm.Fields, fieldModel{
					Name:        f.Name,
					TSName:      tsName,
					Description: strings.TrimSpace(f.Description),
					TSType:      tsType(typeByName, f.Type, f.Optional, f.Nullable),
					Optional:    f.Optional,
					Nullable:    f.Nullable,
					Enum:        append([]string(nil), f.Enum...),
					Const:       f.Const,
				})
			}

		case contract.KindSlice:
			tm.Elem = tsType(typeByName, t.Elem, false, false)

		case contract.KindMap:
			tm.Elem = tsType(typeByName, t.Elem, false, false)

		case contract.KindUnion:
			for _, v := range t.Variants {
				tm.Variants = append(tm.Variants, variantModel{
					Value:       v.Value,
					Type:        toPascal(string(v.Type)),
					Description: strings.TrimSpace(v.Description),
				})
			}
		}

		m.Types = append(m.Types, tm)
	}

	// Resources.
	for _, r := range svc.Resources {
		if r == nil || strings.TrimSpace(r.Name) == "" {
			continue
		}

		prop := toLowerCamel(sanitizeIdent(r.Name))
		if prop == "" {
			prop = "resource"
		}

		rm := resourceModel{
			Name:        r.Name,
			PropName:    prop,
			ClassName:   toPascal(r.Name),
			Description: strings.TrimSpace(r.Description),
		}

		for _, mm := range r.Methods {
			if mm == nil || strings.TrimSpace(mm.Name) == "" {
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
				PropName:    toLowerCamel(sanitizeIdent(mm.Name)),
				Description: strings.TrimSpace(mm.Description),

				HasInput:   hasInput,
				HasOutput:  hasOutput,
				InputType:  toPascal(string(mm.Input)),
				OutputType: toPascal(string(mm.Output)),

				HTTPMethod: httpMethod,
				HTTPPath:   httpPath,

				IsStreaming:    isStreaming,
				StreamMode:     streamMode,
				StreamIsSSE:    streamIsSSE,
				StreamItemType: toPascal(streamItem),
			})
		}

		m.Resources = append(m.Resources, rm)
	}

	// Stable resources order.
	sort.Slice(m.Resources, func(i, j int) bool {
		return m.Resources[i].PropName < m.Resources[j].PropName
	})

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

func tsType(typeByName map[string]*contract.Type, ref contract.TypeRef, optional, nullable bool) string {
	r := strings.TrimSpace(string(ref))
	if r == "" {
		return "unknown"
	}

	base := baseTSType(typeByName, r)

	if nullable {
		base = base + " | null"
	}
	// Optional is encoded at property level with ?:, but for non-property contexts,
	// allowing undefined improves composability.
	if optional {
		base = base + " | undefined"
	}
	return base
}

func baseTSType(typeByName map[string]*contract.Type, r string) string {
	if t, ok := typeByName[r]; ok && t != nil {
		switch t.Kind {
		case contract.KindStruct:
			return toPascal(r)
		case contract.KindSlice:
			elem := tsType(typeByName, t.Elem, false, false)
			return elem + "[]"
		case contract.KindMap:
			elem := tsType(typeByName, t.Elem, false, false)
			return "Record<string, " + elem + ">"
		case contract.KindUnion:
			if len(t.Variants) == 0 {
				return "unknown"
			}
			parts := make([]string, 0, len(t.Variants))
			for _, v := range t.Variants {
				parts = append(parts, baseTSType(typeByName, strings.TrimSpace(string(v.Type))))
			}
			parts = uniqueStringsStable(parts)
			return strings.Join(parts, " | ")
		default:
			return "unknown"
		}
	}

	switch r {
	case "string":
		return "string"
	case "bool", "boolean":
		return "boolean"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64", "number":
		return "number"
	case "time.Time":
		return "string"
	case "json.RawMessage", "any", "interface{}", "object":
		return "unknown"
	}

	return "unknown"
}

func uniqueStringsStable(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func tsQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return `"` + s + `"`
}

func tsIdent(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "x"
	}
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "x"
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

func toSnake(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	var prevLower bool
	var lastUnderscore bool
	for _, r := range s {
		if r == '-' || r == '.' || r == ' ' {
			r = '_'
		}
		if r == '_' {
			if b.Len() > 0 && !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
			prevLower = false
			continue
		}
		if unicode.IsUpper(r) {
			if prevLower && !lastUnderscore {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
			prevLower = false
			lastUnderscore = false
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			prevLower = unicode.IsLetter(r) && unicode.IsLower(r)
			lastUnderscore = false
			continue
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "x"
	}
	return out
}

func toPascal(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "X"
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
	out = strings.ReplaceAll(out, "Id", "ID")
	out = strings.ReplaceAll(out, "Url", "URL")
	out = strings.ReplaceAll(out, "Http", "HTTP")
	out = strings.ReplaceAll(out, "Api", "API")
	out = strings.ReplaceAll(out, "Json", "JSON")
	out = strings.ReplaceAll(out, "Sse", "SSE")
	return out
}

func toLowerCamel(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
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
