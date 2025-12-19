// Package contract: api_spec.go
package contract

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
)

// Codec decodes an api document into a Go value.
// contract is stdlib-only, so YAML support must be provided by another package.
type Codec interface {
	Decode(r io.Reader, v any) error
}

// JSONCodec decodes api.json (and can also decode JSON-formatted api.yaml, if desired).
// It is strict by default (unknown fields are rejected).
type JSONCodec struct {
	// Strict rejects unknown fields when true.
	// Default: true.
	Strict bool
}

func (c JSONCodec) Decode(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	if c.Strict || (c.Strict == false && c.Strict == (JSONCodec{}).Strict) {
		dec.DisallowUnknownFields()
	}
	return dec.Decode(v)
}

// Parse decodes a document using the provided codec and lints it.
// Parse returns an error if any lint issues have SeverityError.
func Parse(r io.Reader, c Codec) (*Service, error) {
	if c == nil {
		return nil, fmt.Errorf("contract: nil codec")
	}
	var s Service
	if err := c.Decode(r, &s); err != nil {
		return nil, err
	}
	issues := Lint(&s)
	if err := LintError(issues); err != nil {
		return nil, err
	}
	return &s, nil
}

func ParseBytes(b []byte, c Codec) (*Service, error)  { return Parse(bytes.NewReader(b), c) }
func ParseString(s string, c Codec) (*Service, error) { return Parse(strings.NewReader(s), c) }

func ParseFile(path string, c Codec) (*Service, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f, c)
}

type LintSeverity string

const (
	SeverityError LintSeverity = "error"
	SeverityWarn  LintSeverity = "warn"
	SeverityInfo  LintSeverity = "info"
)

type LintIssue struct {
	Severity LintSeverity
	Path     string
	Message  string
}

func (i LintIssue) Error() string {
	if i.Path == "" {
		return fmt.Sprintf("contract: %s: %s", i.Severity, i.Message)
	}
	return fmt.Sprintf("contract: %s: %s: %s", i.Severity, i.Path, i.Message)
}

// Lint enforces SDK-ready constraints and naming conventions.
// It is intentionally opinionated to keep generators deterministic.
func Lint(s *Service) []LintIssue {
	if s == nil {
		return []LintIssue{{Severity: SeverityError, Path: "", Message: "nil service"}}
	}

	var out []LintIssue
	add := func(sev LintSeverity, path, msg string, args ...any) {
		out = append(out, LintIssue{
			Severity: sev,
			Path:     path,
			Message:  fmt.Sprintf(msg, args...),
		})
	}

	// service.name
	if strings.TrimSpace(s.Name) == "" {
		add(SeverityError, "name", "is required")
	} else if !isPascalCase(s.Name) {
		add(SeverityError, "name", "must be PascalCase (got %q)", s.Name)
	}

	// client (optional, but if present must be valid)
	if s.Client != nil {
		lintClient(add, s.Client)
	}

	// types (index early, used by methods/fields)
	decl := buildTypeIndex(add, s.Types)

	// types detail
	for i, t := range s.Types {
		p := fmt.Sprintf("types[%d]", i)
		if t == nil {
			continue
		}
		if strings.TrimSpace(t.Name) == "" {
			continue
		}
		if !isPascalCase(t.Name) {
			add(SeverityError, p+".name", "must be PascalCase (got %q)", t.Name)
		}
		lintType(add, p, t, decl)
	}

	// resources
	if len(s.Resources) == 0 {
		add(SeverityError, "resources", "must not be empty")
	} else {
		seenRes := map[string]int{}
		for i, r := range s.Resources {
			p := fmt.Sprintf("resources[%d]", i)
			if r == nil {
				add(SeverityError, p, "is nil")
				continue
			}
			if strings.TrimSpace(r.Name) == "" {
				add(SeverityError, p+".name", "is required")
			} else {
				if !isSnakeCase(r.Name) {
					add(SeverityError, p+".name", "must be snake_case (got %q)", r.Name)
				}
				if j, ok := seenRes[r.Name]; ok {
					add(SeverityError, p+".name", "duplicates resources[%d].name %q", j, r.Name)
				} else {
					seenRes[r.Name] = i
				}
			}

			if len(r.Methods) == 0 {
				add(SeverityError, p+".methods", "must not be empty")
				continue
			}

			seenMeth := map[string]int{}
			for j, m := range r.Methods {
				mp := fmt.Sprintf("%s.methods[%d]", p, j)
				if m == nil {
					add(SeverityError, mp, "is nil")
					continue
				}
				if strings.TrimSpace(m.Name) == "" {
					add(SeverityError, mp+".name", "is required")
				} else {
					if !isSnakeCase(m.Name) {
						add(SeverityError, mp+".name", "must be snake_case (got %q)", m.Name)
					}
					if k, ok := seenMeth[m.Name]; ok {
						add(SeverityError, mp+".name", "duplicates %s.methods[%d].name %q", p, k, m.Name)
					} else {
						seenMeth[m.Name] = j
					}
				}

				lintTypeRef(add, mp+".input", m.Input, decl)
				lintTypeRef(add, mp+".output", m.Output, decl)

				if m.Stream != nil {
					lintStream(add, mp+".stream", m.Stream, decl)
				}
				if m.HTTP != nil {
					lintHTTP(add, mp+".http", m, decl)
				}
			}
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		// Stable ordering for tests and CLI output.
		if out[i].Severity != out[j].Severity {
			return out[i].Severity < out[j].Severity
		}
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Message < out[j].Message
	})

	return out
}

// LintError returns a joined error containing only error-severity issues.
func LintError(issues []LintIssue) error {
	var errs []error
	for _, it := range issues {
		if it.Severity == SeverityError {
			errs = append(errs, it)
		}
	}
	return errors.Join(errs...)
}

func lintClient(add func(LintSeverity, string, string, ...any), c *Client) {
	if strings.TrimSpace(c.BaseURL) != "" {
		u, err := url.Parse(c.BaseURL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			add(SeverityError, "client.base_url", "must be an absolute URL (got %q)", c.BaseURL)
		}
	}

	if c.Headers != nil {
		for k := range c.Headers {
			if strings.TrimSpace(k) == "" {
				add(SeverityError, "client.headers", "contains an empty key")
			}
		}
	}

	// Auth is a hint; keep it loose, but warn if it is not a stable token.
	if strings.TrimSpace(c.Auth) != "" {
		a := strings.ToLower(strings.TrimSpace(c.Auth))
		if !isSnakeCase(a) {
			add(SeverityWarn, "client.auth", "should be a lower_snake_case token (got %q)", c.Auth)
		}
	}
}

func lintHTTP(add func(LintSeverity, string, string, ...any), path string, m *Method, decl map[string]*Type) {
	if strings.TrimSpace(m.HTTP.Method) == "" {
		add(SeverityError, path+".method", "is required")
	} else {
		up := strings.ToUpper(strings.TrimSpace(m.HTTP.Method))
		switch up {
		case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		default:
			add(SeverityError, path+".method", "must be a valid HTTP method (got %q)", m.HTTP.Method)
		}
		if m.HTTP.Method != up {
			add(SeverityWarn, path+".method", "should be uppercase (got %q)", m.HTTP.Method)
		}
	}

	if strings.TrimSpace(m.HTTP.Path) == "" {
		add(SeverityError, path+".path", "is required")
		return
	}
	if !strings.HasPrefix(m.HTTP.Path, "/") {
		add(SeverityError, path+".path", "must start with '/' (got %q)", m.HTTP.Path)
	}

	// Path params: {param} must be snake_case and must exist in the input struct.
	params := extractPathParams(m.HTTP.Path)
	for _, p := range params {
		if !isSnakeCase(p) {
			add(SeverityError, path+".path", "path param {%s} must be snake_case", p)
		}
	}

	if len(params) > 0 {
		inName := strings.TrimSpace(string(m.Input))
		if inName == "" {
			add(SeverityError, path, "path has params %v but method.input is empty", params)
			return
		}
		inType, ok := decl[inName]
		if !ok {
			add(SeverityError, path, "method.input %q must refer to a declared struct type when using path params", inName)
			return
		}
		if inType.Kind != KindStruct {
			add(SeverityError, path, "method.input %q must be kind=struct when using path params", inName)
			return
		}
		fieldSet := map[string]bool{}
		for _, f := range inType.Fields {
			fieldSet[f.Name] = true
		}
		for _, p := range params {
			if !fieldSet[p] {
				add(SeverityError, path, "path param {%s} must exist as a field in %q", p, inName)
			}
		}
	}
}

func lintStream(add func(LintSeverity, string, string, ...any), path string, st *MethodStream, decl map[string]*Type) {
	if strings.TrimSpace(st.Mode) != "" {
		switch strings.ToLower(strings.TrimSpace(st.Mode)) {
		case "sse", "ws", "grpc", "async":
		default:
			add(SeverityError, path+".mode", "must be one of sse|ws|grpc|async (got %q)", st.Mode)
		}
	}
	if strings.TrimSpace(string(st.Item)) == "" {
		add(SeverityError, path+".item", "is required")
	} else {
		lintTypeRef(add, path+".item", st.Item, decl)
	}
	if strings.TrimSpace(string(st.Done)) != "" {
		lintTypeRef(add, path+".done", st.Done, decl)
	}
	if strings.TrimSpace(string(st.Error)) != "" {
		lintTypeRef(add, path+".error", st.Error, decl)
	}
	if strings.TrimSpace(string(st.InputItem)) != "" {
		lintTypeRef(add, path+".input_item", st.InputItem, decl)
	}
}

func lintType(add func(LintSeverity, string, string, ...any), path string, t *Type, decl map[string]*Type) {
	switch t.Kind {
	case KindStruct:
		seen := map[string]int{}
		for i := range t.Fields {
			f := t.Fields[i]
			fp := fmt.Sprintf("%s.fields[%d]", path, i)

			if strings.TrimSpace(f.Name) == "" {
				add(SeverityError, fp+".name", "is required")
			} else {
				if !isSnakeCase(f.Name) {
					add(SeverityError, fp+".name", "must be snake_case (got %q)", f.Name)
				}
				if j, ok := seen[f.Name]; ok {
					add(SeverityError, fp+".name", "duplicates %s.fields[%d].name %q", path, j, f.Name)
				} else {
					seen[f.Name] = i
				}
			}

			if strings.TrimSpace(string(f.Type)) == "" {
				add(SeverityError, fp+".type", "is required")
			} else {
				lintTypeRef(add, fp+".type", f.Type, decl)
			}

			if len(f.Enum) > 0 && f.Const != "" {
				add(SeverityError, fp, "enum and const are mutually exclusive")
			}
		}

	case KindSlice, KindMap:
		if strings.TrimSpace(string(t.Elem)) == "" {
			add(SeverityError, path+".elem", "is required for kind %q", t.Kind)
		} else {
			lintTypeRef(add, path+".elem", t.Elem, decl)
		}
		if len(t.Fields) > 0 {
			add(SeverityError, path+".fields", "must be empty for kind %q", t.Kind)
		}
		if strings.TrimSpace(t.Tag) != "" || len(t.Variants) > 0 {
			add(SeverityError, path, "tag/variants must be empty for kind %q", t.Kind)
		}

	case KindUnion:
		if strings.TrimSpace(t.Tag) == "" {
			add(SeverityError, path+".tag", "is required for kind %q", t.Kind)
		} else if !isSnakeCase(t.Tag) {
			add(SeverityError, path+".tag", "must be snake_case (got %q)", t.Tag)
		}

		if len(t.Variants) == 0 {
			add(SeverityError, path+".variants", "must not be empty for kind %q", t.Kind)
		} else {
			seen := map[string]int{}
			for i := range t.Variants {
				v := t.Variants[i]
				vp := fmt.Sprintf("%s.variants[%d]", path, i)

				if strings.TrimSpace(v.Value) == "" {
					add(SeverityError, vp+".value", "is required")
				} else {
					if j, ok := seen[v.Value]; ok {
						add(SeverityError, vp+".value", "duplicates %s.variants[%d].value %q", path, j, v.Value)
					} else {
						seen[v.Value] = i
					}
				}

				if strings.TrimSpace(string(v.Type)) == "" {
					add(SeverityError, vp+".type", "is required")
					continue
				}

				// Union variants must refer to declared struct types (SDK simplicity).
				name := string(v.Type)
				dt, ok := decl[name]
				if !ok {
					add(SeverityError, vp+".type", "must refer to a declared type (got %q)", name)
				} else if dt.Kind != KindStruct {
					add(SeverityError, vp+".type", "must refer to a struct type (got %q kind %q)", name, dt.Kind)
				}
			}
		}

		if strings.TrimSpace(string(t.Elem)) != "" {
			add(SeverityError, path+".elem", "must be empty for kind %q", t.Kind)
		}
		if len(t.Fields) > 0 {
			add(SeverityError, path+".fields", "must be empty for kind %q", t.Kind)
		}

	default:
		add(SeverityError, path+".kind", "invalid kind %q", t.Kind)
	}
}

func lintTypeRef(add func(LintSeverity, string, string, ...any), path string, tr TypeRef, decl map[string]*Type) {
	name := strings.TrimSpace(string(tr))
	if name == "" {
		return
	}
	if _, ok := decl[name]; ok {
		return
	}

	// If it looks like a user-defined type (PascalCase), require it to be declared.
	// This prevents silent typos that are disastrous for SDK generation.
	if looksPascalType(name) {
		add(SeverityError, path, "unknown type ref %q (did you forget to declare it in types?)", name)
		return
	}

	// Allow external primitives (string, int64, time.Time, json.RawMessage, etc),
	// but require token sanity.
	if !isTypeToken(name) {
		add(SeverityError, path, "invalid type ref %q", name)
	}
}

func buildTypeIndex(add func(LintSeverity, string, string, ...any), types []*Type) map[string]*Type {
	decl := make(map[string]*Type, len(types))
	for i, t := range types {
		p := fmt.Sprintf("types[%d]", i)
		if t == nil {
			add(SeverityError, p, "is nil")
			continue
		}
		if strings.TrimSpace(t.Name) == "" {
			add(SeverityError, p+".name", "is required")
			continue
		}
		if _, ok := decl[t.Name]; ok {
			add(SeverityError, p+".name", "duplicate type name %q", t.Name)
			continue
		}
		decl[t.Name] = t
	}
	return decl
}

var (
	reSnake    = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	rePascal   = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	reTypeName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_./\[\]]*$`)
	rePathVar  = regexp.MustCompile(`\{([A-Za-z0-9_]+)\}`)
)

func isSnakeCase(s string) bool  { return reSnake.MatchString(s) }
func isPascalCase(s string) bool { return rePascal.MatchString(s) }

func looksPascalType(s string) bool {
	// dotted and qualified names are treated as external-ish.
	if strings.ContainsAny(s, "./[]") {
		return false
	}
	return isPascalCase(s)
}

func isTypeToken(s string) bool { return reTypeName.MatchString(s) }

func extractPathParams(path string) []string {
	m := rePathVar.FindAllStringSubmatch(path, -1)
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	seen := map[string]bool{}
	for _, mm := range m {
		if len(mm) != 2 {
			continue
		}
		p := mm[1]
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}
