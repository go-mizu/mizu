// Package sdkgo generates typed Go SDK clients from contract.Service.
package sdkgo

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"unicode"

	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/sdk"
)

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
		Parse(goTemplate)
	if err != nil {
		return nil, fmt.Errorf("sdkgo: parse template: %w", err)
	}

	var out bytes.Buffer
	if err := tpl.Execute(&out, m); err != nil {
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

	HasTime      bool
	HasStreaming bool

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

	m.HasStreaming = hasStreaming(svc)
	m.HasTime = hasTime(svc)

	// Types in stable order
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

	// Resources
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

	m.Imports = computeImports(m.HasStreaming, m.HasTime)
	return m, nil
}

func computeImports(hasStreaming, hasTime bool) []string {
	imports := []string{
		"bytes",
		"context",
		"encoding/json",
		"fmt",
		"io",
		"net/http",
		"strings",
	}
	if hasStreaming {
		imports = append(imports, "bufio")
	}
	if hasTime {
		imports = append(imports, "time")
	}
	sort.Strings(imports)
	return imports
}

func hasStreaming(svc *contract.Service) bool {
	for _, r := range svc.Resources {
		if r == nil {
			continue
		}
		for _, m := range r.Methods {
			if m != nil && m.Stream != nil {
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

	// Defensive support for inline collection refs.
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

const goTemplate = `// Code generated by sdkgo. DO NOT EDIT.

package {{.Package}}

import (
{{- range .Imports}}
	{{quote .}}
{{- end}}
)

{{- if .Service.Description}}
// {{.Service.Sanitized}} {{.Service.Description}}
{{- end}}

// Client is the {{if .Service.Sanitized}}{{.Service.Sanitized}}{{else}}API{{end}} API client.
type Client struct {
{{- range .Resources}}
	{{.GoName}} *{{.GoName}}Resource
{{- end}}

	baseURL string
	token   string
	auth    string
	headers map[string]string
	http    *http.Client
}

// Option configures the client.
type Option func(*Client)

// NewClient creates a new client.
func NewClient(token string, opts ...Option) *Client {
	c := &Client{
		baseURL: {{quote .Defaults.BaseURL}},
		token:   token,
		auth:    {{if .Defaults.Auth}}{{quote .Defaults.Auth}}{{else}}"bearer"{{end}},
		headers: make(map[string]string),
		http:    http.DefaultClient,
	}

{{- if .Defaults.Headers}}
{{- range .Defaults.Headers}}
	c.headers[{{quote .K}}] = {{quote .V}}
{{- end}}
{{- end}}

	for _, opt := range opts {
		opt(c)
	}

{{- range .Resources}}
	c.{{.GoName}} = &{{.GoName}}Resource{client: c}
{{- end}}

	return c
}

// WithBaseURL sets a custom base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(url, "/")
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// WithHeader adds a custom header to all requests.
func WithHeader(key, value string) Option {
	return func(c *Client) { c.headers[key] = value }
}

// WithAuth sets the auth hint ("bearer", "basic", "none").
func WithAuth(auth string) Option {
	return func(c *Client) { c.auth = auth }
}

{{- if .Types}}
// Types
{{- range .Types}}

{{- if .Description}}
// {{.Name}} {{.Description}}
{{- end}}
{{- if eq .Kind "struct"}}
type {{.Name}} struct {
{{- range .Fields}}
	{{.GoName}} {{.GoType}} ` + "`" + `json:"{{.Tag}}"` + "`" + `{{- if .Const}} // always {{quote .Const}}{{else if .Enum}} // one of: {{join .Enum ", "}}{{else if .Description}} // {{.Description}}{{end}}
{{- end}}
}
{{- end}}

{{- if eq .Kind "slice"}}
type {{.Name}} []{{.Elem}}
{{- end}}

{{- if eq .Kind "map"}}
type {{.Name}} map[string]{{.Elem}}
{{- end}}

{{- if eq .Kind "union"}}
{{- if .Tag}}
// {{.Name}} is a discriminated union (tag: {{quote .Tag}}).
{{- else}}
// {{.Name}} is a discriminated union.
{{- end}}
type {{.Name}} struct {
{{- range .Variants}}
	{{.FieldName}} *{{.Type}} ` + "`" + `json:"-"` + "`" + `
{{- end}}
}

func (u *{{.Name}}) MarshalJSON() ([]byte, error) {
{{- range .Variants}}
	if u.{{.FieldName}} != nil {
		return json.Marshal(u.{{.FieldName}})
	}
{{- end}}
	return []byte("null"), nil
}

func (u *{{.Name}}) UnmarshalJSON(data []byte) error {
{{- if .Tag}}
	var disc struct {
		{{goName .Tag}} string ` + "`" + `json:"{{.Tag}}"` + "`" + `
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return err
	}
	switch disc.{{goName .Tag}} {
{{- range .Variants}}
	case {{quote .Value}}:
		u.{{.FieldName}} = new({{.Type}})
		return json.Unmarshal(data, u.{{.FieldName}})
{{- end}}
	}
	return fmt.Errorf("unknown {{.Name}} {{.Tag}}: %q", disc.{{goName .Tag}})
{{- else}}
	return fmt.Errorf("union {{.Name}} missing discriminator tag")
{{- end}}
}
{{- end}}

{{- end}}
{{- end}}

// Resources
{{- range .Resources}}

{{- if .Description}}
// {{.GoName}}Resource {{.Description}}
{{- else}}
// {{.GoName}}Resource handles {{.Name}} operations.
{{- end}}
type {{.GoName}}Resource struct {
	client *Client
}

{{- $res := .}}
{{- range .Methods}}

{{- if .Description}}
// {{.GoName}} {{.Description}}
{{- end}}

{{- if .IsStreaming}}
{{- if .StreamIsSSE}}
func (r *{{$res.GoName}}Resource) {{.GoName}}(ctx context.Context{{if .HasInput}}, in *{{.InputType}}{{end}}) *EventStream[{{.StreamItemType}}] {
	parse := func(data []byte) ({{.StreamItemType}}, error) {
		var v {{.StreamItemType}}
		err := json.Unmarshal(data, &v)
		return v, err
	}

	s := &EventStream[{{.StreamItemType}}]{parse: parse}
	u := r.client.baseURL + {{quote .HTTPPath}}

	var body io.Reader
{{- if .HasInput}}
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			s.err = err
			return s
		}
		body = bytes.NewReader(b)
	}
{{- end}}

	req, err := http.NewRequestWithContext(ctx, {{quote .HTTPMethod}}, u, body)
	if err != nil {
		s.err = err
		return s
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	applyAuth(req, r.client.auth, r.client.token)
	for k, v := range r.client.headers {
		req.Header.Set(k, v)
	}

	resp, err := r.client.http.Do(req)
	if err != nil {
		s.err = err
		return s
	}

	if resp.StatusCode >= 400 {
		s.err = decodeError(resp)
		_ = resp.Body.Close()
		return s
	}

	s.resp = resp
	s.scanner = bufio.NewScanner(resp.Body)
	s.scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	return s
}
{{- else}}
func (r *{{$res.GoName}}Resource) {{.GoName}}(ctx context.Context{{if .HasInput}}, in *{{.InputType}}{{end}}) error {
	return fmt.Errorf("stream mode %q is not supported by this Go SDK generator", {{quote .StreamMode}})
}
{{- end}}
{{- else}}

{{- if and (not .HasInput) (not .HasOutput)}}
func (r *{{$res.GoName}}Resource) {{.GoName}}(ctx context.Context) error {
	return r.client.do(ctx, {{quote .HTTPMethod}}, {{quote .HTTPPath}}, nil, nil)
}
{{- end}}

{{- if and (not .HasInput) (.HasOutput)}}
func (r *{{$res.GoName}}Resource) {{.GoName}}(ctx context.Context) (*{{.OutputType}}, error) {
	var out {{.OutputType}}
	err := r.client.do(ctx, {{quote .HTTPMethod}}, {{quote .HTTPPath}}, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
{{- end}}

{{- if and (.HasInput) (not .HasOutput)}}
func (r *{{$res.GoName}}Resource) {{.GoName}}(ctx context.Context, in *{{.InputType}}) error {
	return r.client.do(ctx, {{quote .HTTPMethod}}, {{quote .HTTPPath}}, in, nil)
}
{{- end}}

{{- if and (.HasInput) (.HasOutput)}}
func (r *{{$res.GoName}}Resource) {{.GoName}}(ctx context.Context, in *{{.InputType}}) (*{{.OutputType}}, error) {
	var out {{.OutputType}}
	err := r.client.do(ctx, {{quote .HTTPMethod}}, {{quote .HTTPPath}}, in, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
{{- end}}

{{- end}}

{{- end}}
{{- end}}

{{- if .HasStreaming}}
// Streaming

// EventStream reads server-sent events.
type EventStream[T any] struct {
	resp    *http.Response
	scanner *bufio.Scanner
	parse   func([]byte) (T, error)

	current T
	err     error

	buf strings.Builder
}

// Next advances to the next event. Returns false when done or on error.
func (s *EventStream[T]) Next() bool {
	if s.err != nil || s.scanner == nil {
		return false
	}

	s.buf.Reset()

	for s.scanner.Scan() {
		line := s.scanner.Text()

		if line == "" {
			data := strings.TrimSpace(s.buf.String())
			if data == "" {
				continue
			}
			if data == "[DONE]" {
				return false
			}
			s.current, s.err = s.parse([]byte(data))
			if s.err != nil {
				return false
			}
			return true
		}

		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			if strings.HasPrefix(data, " ") {
				data = strings.TrimPrefix(data, " ")
			}
			if s.buf.Len() > 0 {
				s.buf.WriteByte('\n')
			}
			s.buf.WriteString(data)
			continue
		}
	}

	if err := s.scanner.Err(); err != nil {
		s.err = err
		return false
	}

	data := strings.TrimSpace(s.buf.String())
	if data == "" || data == "[DONE]" {
		return false
	}

	s.current, s.err = s.parse([]byte(data))
	if s.err != nil {
		return false
	}
	return true
}

// Event returns the current event. Call after Next returns true.
func (s *EventStream[T]) Event() T { return s.current }

// Err returns any error that occurred during streaming.
func (s *EventStream[T]) Err() error { return s.err }

// Close closes the underlying connection.
func (s *EventStream[T]) Close() error {
	if s.resp != nil && s.resp.Body != nil {
		return s.resp.Body.Close()
	}
	return nil
}
{{- end}}

// Errors and transport

// Error represents an API error response.
type Error struct {
	StatusCode int    ` + "`" + `json:"-"` + "`" + `
	Code       string ` + "`" + `json:"code,omitempty"` + "`" + `
	Message    string ` + "`" + `json:"message"` + "`" + `
}

func (e *Error) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Message
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	u := c.baseURL + path

	var body io.Reader
	if method != "GET" && in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	applyAuth(req, c.auth, c.token)

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return decodeError(resp)
	}

	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func applyAuth(req *http.Request, auth, token string) {
	if token == "" {
		return
	}
	a := strings.ToLower(strings.TrimSpace(auth))
	if a == "" {
		a = "bearer"
	}
	switch a {
	case "none":
		return
	case "basic":
		req.Header.Set("Authorization", "Basic "+token)
	default:
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

func decodeError(resp *http.Response) error {
	var e Error
	e.StatusCode = resp.StatusCode
	if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
		e.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return &e
	}
	if e.Message == "" {
		e.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return &e
}

var _ = strings.TrimSpace
`
