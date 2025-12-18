// contract/transport/rest/server.go
package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	contract "github.com/go-mizu/mizu/contract/v2"
)

// Server exposes a contract.Invoker over HTTP using method HTTP bindings.
//
// It is intentionally small:
//   - It does not depend on any third-party router.
//   - It uses the contract descriptor as the routing table.
//   - It fills inputs from:
//       - Path params (from "{name}" segments)
//       - Query params (for GET)
//       - JSON body (for non-GET when present)
//   - It returns JSON for outputs and standard error JSON for failures.
type Server struct {
	inv contract.Invoker
	svc *contract.Service

	mux   *http.ServeMux
	routes []route
}

type route struct {
	httpMethod   string
	pathTemplate string
	segments     []string
	paramNames   []string

	resource string
	method   string

	hasInput  bool
	hasOutput bool
}

// NewServer constructs a REST server for the given invoker.
// It builds an internal route table from svc.Resources[*].Methods[*].HTTP.
func NewServer(inv contract.Invoker) (*Server, error) {
	if inv == nil {
		return nil, errors.New("rest: nil invoker")
	}
	svc := inv.Descriptor()
	if svc == nil {
		return nil, errors.New("rest: nil descriptor")
	}

	s := &Server{
		inv: inv,
		svc: svc,
		mux: http.NewServeMux(),
	}
	if err := s.buildRoutes(); err != nil {
		return nil, err
	}

	// One handler for all routes, simple linear match.
	s.mux.HandleFunc("/", s.handle)

	return s, nil
}

// Handler returns the underlying http.Handler.
func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) buildRoutes() error {
	for _, res := range s.svc.Resources {
		if res == nil {
			continue
		}
		for _, m := range res.Methods {
			if m == nil || m.HTTP == nil {
				continue
			}
			hm := strings.ToUpper(strings.TrimSpace(m.HTTP.Method))
			p := strings.TrimSpace(m.HTTP.Path)
			if hm == "" || p == "" || !strings.HasPrefix(p, "/") {
				return fmt.Errorf("rest: invalid http binding for %s.%s", res.Name, m.Name)
			}
			seg, params, err := parsePathTemplate(p)
			if err != nil {
				return fmt.Errorf("rest: %s.%s: %v", res.Name, m.Name, err)
			}
			s.routes = append(s.routes, route{
				httpMethod:   hm,
				pathTemplate: p,
				segments:     seg,
				paramNames:   params,
				resource:     res.Name,
				method:       m.Name,
				hasInput:     m.Input != "",
				hasOutput:    m.Output != "",
			})
		}
	}
	if len(s.routes) == 0 {
		return errors.New("rest: no http routes in descriptor")
	}
	return nil
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	rt, pathParams := s.matchRoute(r.Method, r.URL.Path)
	if rt == nil {
		http.NotFound(w, r)
		return
	}

	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	var in any
	if rt.hasInput {
		var err error
		in, err = s.inv.NewInput(rt.resource, rt.method)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "new_input_failed", err.Error())
			return
		}
		if in == nil {
			writeError(w, http.StatusInternalServerError, "new_input_failed", "invoker returned nil input")
			return
		}

		// Fill from path params first.
		if err := fillStructFromStrings(in, pathParams); err != nil {
			writeError(w, http.StatusBadRequest, "bad_path_params", err.Error())
			return
		}

		// Fill from query for GET (and for any verb, query can be used as extra input).
		if err := fillStructFromStrings(in, r.URL.Query()); err != nil {
			writeError(w, http.StatusBadRequest, "bad_query", err.Error())
			return
		}

		// If request has a body, merge JSON into the struct (non-GET typical).
		if r.Body != nil && r.ContentLength != 0 {
			defer r.Body.Close()
			dec := json.NewDecoder(r.Body)
			dec.DisallowUnknownFields()

			if err := dec.Decode(in); err != nil {
				// If content-type is not JSON, Decode may fail; keep message simple.
				writeError(w, http.StatusBadRequest, "bad_json", err.Error())
				return
			}
		}
	}

	out, err := s.inv.Call(ctx, rt.resource, rt.method, in)
	if err != nil {
		// You can extend this later with richer error mapping.
		writeError(w, http.StatusBadRequest, "call_failed", err.Error())
		return
	}

	if !rt.hasOutput {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) matchRoute(method, path string) (*route, map[string][]string) {
	method = strings.ToUpper(method)
	for i := range s.routes {
		rt := &s.routes[i]
		if rt.httpMethod != method {
			continue
		}
		ok, params := matchPath(rt.segments, path)
		if ok {
			return rt, params
		}
	}
	return nil, nil
}

func parsePathTemplate(tpl string) (segments []string, params []string, err error) {
	if tpl == "" || tpl[0] != '/' {
		return nil, nil, fmt.Errorf("path must start with '/'")
	}
	parts := strings.Split(strings.Trim(tpl, "/"), "/")
	for _, p := range parts {
		if p == "" {
			continue
		}
		segments = append(segments, p)
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			name := strings.TrimSuffix(strings.TrimPrefix(p, "{"), "}")
			name = strings.TrimSpace(name)
			if name == "" {
				return nil, nil, fmt.Errorf("empty path param in %q", tpl)
			}
			params = append(params, name)
		}
	}
	return segments, params, nil
}

func matchPath(segments []string, path string) (bool, map[string][]string) {
	path = strings.Trim(path, "/")
	if path == "" && len(segments) == 0 {
		return true, map[string][]string{}
	}
	parts := strings.Split(path, "/")
	if len(parts) != len(segments) {
		return false, nil
	}

	params := make(map[string][]string)
	for i := range segments {
		t := segments[i]
		p := parts[i]
		if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
			name := strings.TrimSuffix(strings.TrimPrefix(t, "{"), "}")
			params[name] = []string{p}
			continue
		}
		if t != p {
			return false, nil
		}
	}
	return true, params
}

// fillStructFromStrings sets exported struct fields from a map of string slices.
// It matches by json tag name (preferred) or by field name (case-insensitive).
func fillStructFromStrings(dst any, values map[string][]string) error {
	if len(values) == 0 {
		return nil
	}
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("input must be non-nil pointer")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("input must point to a struct")
	}

	t := v.Type()
	for key, vs := range values {
		if len(vs) == 0 {
			continue
		}
		raw := vs[0]

		fi, ok := findFieldIndexByWireName(t, key)
		if !ok {
			// Ignore unknown keys by default for query/path params.
			continue
		}
		fv := v.Field(fi)
		if !fv.CanSet() {
			continue
		}
		if err := setFromString(fv, raw); err != nil {
			return fmt.Errorf("%s: %v", key, err)
		}
	}
	return nil
}

func findFieldIndexByWireName(t reflect.Type, wire string) (int, bool) {
	wire = strings.TrimSpace(wire)
	if wire == "" {
		return 0, false
	}
	wireLower := strings.ToLower(wire)

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		if sf.Anonymous {
			continue
		}

		tag := sf.Tag.Get("json")
		if tag != "" {
			name := strings.Split(tag, ",")[0]
			name = strings.TrimSpace(name)
			if name == "-" {
				continue
			}
			if strings.ToLower(name) == wireLower {
				return i, true
			}
		}

		if strings.ToLower(sf.Name) == wireLower {
			return i, true
		}
	}
	return 0, false
}

func setFromString(fv reflect.Value, raw string) error {
	// Support pointers to scalars by allocating.
	if fv.Kind() == reflect.Pointer {
		if fv.IsNil() {
			fv.Set(reflect.New(fv.Type().Elem()))
		}
		return setFromString(fv.Elem(), raw)
	}

	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)
		return nil
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		fv.SetBool(b)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// int64 might be used for IDs
		n, err := strconv.ParseInt(raw, 10, fv.Type().Bits())
		if err != nil {
			return err
		}
		fv.SetInt(n)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		n, err := strconv.ParseUint(raw, 10, fv.Type().Bits())
		if err != nil {
			return err
		}
		fv.SetUint(n)
		return nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, fv.Type().Bits())
		if err != nil {
			return err
		}
		fv.SetFloat(f)
		return nil
	}

	// If it is a struct, slice, map etc, accept JSON in the param.
	// This is a useful escape hatch for advanced filters.
	b := []byte(raw)
	ptr := reflect.New(fv.Type())
	if err := json.Unmarshal(b, ptr.Interface()); err != nil {
		return fmt.Errorf("unsupported field type %s (and JSON decode failed): %v", fv.Type(), err)
	}
	fv.Set(ptr.Elem())
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("content-type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

type errBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code string, msg string) {
	writeJSON(w, status, errBody{Error: code, Message: msg})
}
