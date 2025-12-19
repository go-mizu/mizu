// contract/transport/rest/client.go
package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	contract "github.com/go-mizu/mizu/contract/v2"
)

// Client calls a REST API described by a contract.Service using HTTP bindings.
//
// This is a generic client (not per-resource typed wrappers). It is intended as:
//   - a runtime client for dynamic usage
//   - a building block for codegen
type Client struct {
	Svc *contract.Service

	BaseURL string
	Token   string // bearer token (optional)

	Headers map[string]string
	HTTP    *http.Client
}

// NewClient creates a client with sensible defaults.
func NewClient(svc *contract.Service) (*Client, error) {
	if svc == nil {
		return nil, errors.New("rest: nil service")
	}
	base := ""
	token := ""
	h := map[string]string{}

	if svc.Client != nil {
		base = strings.TrimRight(svc.Client.BaseURL, "/")
		if strings.EqualFold(svc.Client.Auth, "bearer") {
			// Token remains empty until user sets it.
			token = ""
		}
		for k, v := range svc.Client.Headers {
			h[k] = v
		}
	}

	return &Client{
		Svc:     svc,
		BaseURL: base,
		Token:   token,
		Headers: h,
		HTTP:    http.DefaultClient,
	}, nil
}

// Call invokes resource.method with input in and decodes JSON output into out.
// If the method has no output, out may be nil.
//
// For GET, input is encoded into query params and path params.
// For non-GET, input is encoded as JSON body and also used for path params.
func (c *Client) Call(ctx context.Context, resource string, method string, in any, out any) error {
	if c.Svc == nil {
		return errors.New("rest: nil service")
	}
	m := c.findMethod(resource, method)
	if m == nil || m.HTTP == nil {
		return fmt.Errorf("rest: method not found or missing http binding: %s.%s", resource, method)
	}

	httpMethod := strings.ToUpper(strings.TrimSpace(m.HTTP.Method))
	path := strings.TrimSpace(m.HTTP.Path)

	fullURL, err := c.buildURL(path, in, httpMethod == "GET")
	if err != nil {
		return err
	}

	var body io.Reader
	if httpMethod != "GET" && in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, httpMethod, fullURL, body)
	if err != nil {
		return err
	}

	// Headers
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("content-type") == "" && httpMethod != "GET" {
		req.Header.Set("content-type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("authorization", "Bearer "+c.Token)
	}

	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}

	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		// Try decode server error shape, fallback to text.
		var eb map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&eb)
		if msg, ok := eb["message"].(string); ok && msg != "" {
			return fmt.Errorf("rest: %s %s: %s", httpMethod, path, msg)
		}
		return fmt.Errorf("rest: %s %s: http %d", httpMethod, path, resp.StatusCode)
	}

	if out == nil {
		return nil
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	dec := json.NewDecoder(resp.Body)
	return dec.Decode(out)
}

func (c *Client) findMethod(resource, method string) *contract.Method {
	for _, r := range c.Svc.Resources {
		if r != nil && r.Name == resource {
			for _, m := range r.Methods {
				if m != nil && m.Name == method {
					return m
				}
			}
		}
	}
	return nil
}

func (c *Client) buildURL(pathTpl string, in any, isGET bool) (string, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		return "", errors.New("rest: empty BaseURL")
	}

	path, used, err := substitutePathParams(pathTpl, in)
	if err != nil {
		return "", err
	}

	u, err := url.Parse(base + path)
	if err != nil {
		return "", err
	}

	// For GET (and optionally for any verb), encode remaining fields as query params.
	// We skip params already used in the path.
	if in != nil {
		q := u.Query()
		addQueryFromStruct(q, in, used)
		u.RawQuery = q.Encode()
	}

	_ = isGET
	return u.String(), nil
}

func substitutePathParams(pathTpl string, in any) (string, map[string]bool, error) {
	used := map[string]bool{}

	if !strings.Contains(pathTpl, "{") {
		return pathTpl, used, nil
	}
	if in == nil {
		return "", nil, fmt.Errorf("rest: path has params but input is nil")
	}

	// Extract param names and replace from input struct by matching json tag or field name.
	out := pathTpl
	for {
		start := strings.Index(out, "{")
		if start < 0 {
			break
		}
		end := strings.Index(out[start:], "}")
		if end < 0 {
			return "", nil, fmt.Errorf("rest: invalid path template: %q", pathTpl)
		}
		end = start + end
		name := strings.TrimSpace(out[start+1 : end])
		if name == "" {
			return "", nil, fmt.Errorf("rest: empty path param in %q", pathTpl)
		}
		val, ok := getFieldStringByWireName(in, name)
		if !ok {
			return "", nil, fmt.Errorf("rest: missing path param %q in input", name)
		}
		used[name] = true
		out = out[:start] + url.PathEscape(val) + out[end+1:]
	}
	return out, used, nil
}

func addQueryFromStruct(q url.Values, in any, used map[string]bool) {
	v := reflect.ValueOf(in)
	if !v.IsValid() {
		return
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		wire := wireName(sf)
		if wire == "" || wire == "-" {
			continue
		}
		if used != nil && used[wire] {
			continue
		}

		fv := v.Field(i)
		if sf.Type.Kind() == reflect.Pointer {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
		}

		// Basic query encoding; complex fields can be JSON-stringified by the caller.
		switch fv.Kind() {
		case reflect.String:
			if fv.String() != "" {
				q.Set(wire, fv.String())
			}
		case reflect.Bool:
			q.Set(wire, strconvBool(fv.Bool()))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			q.Set(wire, fmt.Sprintf("%d", fv.Int()))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			q.Set(wire, fmt.Sprintf("%d", fv.Uint()))
		case reflect.Float32, reflect.Float64:
			q.Set(wire, fmt.Sprintf("%v", fv.Float()))
		default:
			// Skip complex types for query by default.
		}
	}
}

func getFieldStringByWireName(in any, name string) (string, bool) {
	v := reflect.ValueOf(in)
	if !v.IsValid() {
		return "", false
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return "", false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return "", false
	}
	t := v.Type()

	wireLower := strings.ToLower(name)
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		w := wireName(sf)
		if strings.ToLower(w) == wireLower || strings.ToLower(sf.Name) == wireLower {
			fv := v.Field(i)
			if sf.Type.Kind() == reflect.Pointer {
				if fv.IsNil() {
					return "", false
				}
				fv = fv.Elem()
			}
			return fmt.Sprint(fv.Interface()), true
		}
	}
	return "", false
}

func wireName(sf reflect.StructField) string {
	tag := sf.Tag.Get("json")
	if tag == "" {
		return sf.Name
	}
	name := strings.Split(tag, ",")[0]
	name = strings.TrimSpace(name)
	if name == "" {
		return sf.Name
	}
	return name
}

func strconvBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
