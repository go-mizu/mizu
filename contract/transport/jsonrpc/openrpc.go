package jsonrpc

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu/contract"
)

// OpenRPCVersion is the OpenRPC specification version.
const OpenRPCVersion = "1.3.0"

// OpenRPCDocument represents an OpenRPC 1.3 document.
type OpenRPCDocument struct {
	OpenRPC    string              `json:"openrpc"`
	Info       *OpenRPCInfo        `json:"info"`
	Servers    []*OpenRPCServer    `json:"servers,omitempty"`
	Methods    []*OpenRPCMethod    `json:"methods"`
	Components *OpenRPCComponents  `json:"components,omitempty"`
}

// OpenRPCInfo provides metadata about the API.
type OpenRPCInfo struct {
	Title          string          `json:"title"`
	Description    string          `json:"description,omitempty"`
	TermsOfService string          `json:"termsOfService,omitempty"`
	Contact        *OpenRPCContact `json:"contact,omitempty"`
	License        *OpenRPCLicense `json:"license,omitempty"`
	Version        string          `json:"version"`
}

// OpenRPCContact provides contact information.
type OpenRPCContact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// OpenRPCLicense provides license information.
type OpenRPCLicense struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// OpenRPCServer describes a server.
type OpenRPCServer struct {
	Name        string `json:"name,omitempty"`
	URL         string `json:"url"`
	Summary     string `json:"summary,omitempty"`
	Description string `json:"description,omitempty"`
}

// OpenRPCMethod describes a JSON-RPC method.
type OpenRPCMethod struct {
	Name        string                  `json:"name"`
	Summary     string                  `json:"summary,omitempty"`
	Description string                  `json:"description,omitempty"`
	Tags        []*OpenRPCTag           `json:"tags,omitempty"`
	Params      []*OpenRPCContentDesc   `json:"params"`
	Result      *OpenRPCContentDesc     `json:"result,omitempty"`
	Deprecated  bool                    `json:"deprecated,omitempty"`
	Errors      []*OpenRPCErrorRef      `json:"errors,omitempty"`
	Examples    []*OpenRPCExamplePairing `json:"examples,omitempty"`
}

// OpenRPCTag adds metadata to methods.
type OpenRPCTag struct {
	Name        string `json:"name"`
	Summary     string `json:"summary,omitempty"`
	Description string `json:"description,omitempty"`
}

// OpenRPCContentDesc describes a parameter or result.
type OpenRPCContentDesc struct {
	Name        string     `json:"name"`
	Summary     string     `json:"summary,omitempty"`
	Description string     `json:"description,omitempty"`
	Required    bool       `json:"required,omitempty"`
	Schema      *SchemaRef `json:"schema"`
}

// OpenRPCErrorRef references an error definition.
type OpenRPCErrorRef struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// OpenRPCExamplePairing groups request and result examples.
type OpenRPCExamplePairing struct {
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Summary     string         `json:"summary,omitempty"`
	Params      []*OpenRPCExample `json:"params,omitempty"`
	Result      *OpenRPCExample   `json:"result,omitempty"`
}

// OpenRPCExample represents an example value.
type OpenRPCExample struct {
	Name        string `json:"name,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Description string `json:"description,omitempty"`
	Value       any    `json:"value"`
}

// OpenRPCComponents holds reusable objects.
type OpenRPCComponents struct {
	ContentDescriptors map[string]*OpenRPCContentDesc `json:"contentDescriptors,omitempty"`
	Schemas            map[string]*Schema             `json:"schemas,omitempty"`
	Errors             map[string]*OpenRPCErrorRef    `json:"errors,omitempty"`
	Tags               map[string]*OpenRPCTag         `json:"tags,omitempty"`
}

// Schema represents a JSON Schema.
type Schema struct {
	Type                 string             `json:"type,omitempty"`
	Format               string             `json:"format,omitempty"`
	Description          string             `json:"description,omitempty"`
	Default              any                `json:"default,omitempty"`
	Nullable             bool               `json:"nullable,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty"`
	Enum                 []any              `json:"enum,omitempty"`
	Minimum              *float64           `json:"minimum,omitempty"`
	Maximum              *float64           `json:"maximum,omitempty"`
	MinLength            *int               `json:"minLength,omitempty"`
	MaxLength            *int               `json:"maxLength,omitempty"`
	Pattern              string             `json:"pattern,omitempty"`
	MinItems             *int               `json:"minItems,omitempty"`
	MaxItems             *int               `json:"maxItems,omitempty"`
	UniqueItems          bool               `json:"uniqueItems,omitempty"`
}

// SchemaRef is either a schema or a $ref.
type SchemaRef struct {
	Ref    string  `json:"-"`
	Schema *Schema `json:"-"`
}

// MarshalJSON implements json.Marshaler.
func (s *SchemaRef) MarshalJSON() ([]byte, error) {
	if s.Ref != "" {
		return json.Marshal(map[string]string{"$ref": s.Ref})
	}
	return json.Marshal(s.Schema)
}

// GenerateOpenRPC creates an OpenRPC document from a service.
func GenerateOpenRPC(svc *contract.Service) *OpenRPCDocument {
	doc := &OpenRPCDocument{
		OpenRPC: OpenRPCVersion,
		Info: &OpenRPCInfo{
			Title:   svc.Name + " API",
			Version: "1.0.0",
		},
		Methods: make([]*OpenRPCMethod, 0, len(svc.Methods)),
		Components: &OpenRPCComponents{
			Schemas: make(map[string]*Schema),
		},
	}

	if svc.Description != "" {
		doc.Info.Description = svc.Description
	}
	if svc.Version != "" {
		doc.Info.Version = svc.Version
	}

	// Add schemas from types registry
	for _, schema := range svc.Types.Schemas() {
		doc.Components.Schemas[schema.ID] = convertSchema(schema.JSON)
	}

	// Add methods
	for _, m := range svc.Methods {
		doc.Methods = append(doc.Methods, convertMethod(svc, m))
	}

	return doc
}

func convertMethod(svc *contract.Service, m *contract.Method) *OpenRPCMethod {
	method := &OpenRPCMethod{
		Name:        svc.Name + "." + m.Name,
		Summary:     m.Summary,
		Description: m.Description,
		Deprecated:  m.Deprecated,
		Params:      make([]*OpenRPCContentDesc, 0),
	}

	// Add tags
	for _, tag := range m.Tags {
		method.Tags = append(method.Tags, &OpenRPCTag{Name: tag})
	}

	// Add params (single object param for named params)
	if m.Input != nil {
		method.Params = append(method.Params, &OpenRPCContentDesc{
			Name:     "params",
			Required: true,
			Schema:   &SchemaRef{Ref: "#/components/schemas/" + m.Input.ID},
		})
	}

	// Add result
	if m.Output != nil {
		method.Result = &OpenRPCContentDesc{
			Name:   "result",
			Schema: &SchemaRef{Ref: "#/components/schemas/" + m.Output.ID},
		}
	}

	// Add standard JSON-RPC errors
	method.Errors = standardOpenRPCErrors()

	return method
}

func standardOpenRPCErrors() []*OpenRPCErrorRef {
	return []*OpenRPCErrorRef{
		{Code: CodeParseError, Message: "Parse error"},
		{Code: CodeInvalidRequest, Message: "Invalid Request"},
		{Code: CodeMethodNotFound, Message: "Method not found"},
		{Code: CodeInvalidParams, Message: "Invalid params"},
		{Code: CodeInternalError, Message: "Internal error"},
	}
}

func convertSchema(m map[string]any) *Schema {
	s := &Schema{}

	if t, ok := m["type"].(string); ok {
		s.Type = t
	}
	if f, ok := m["format"].(string); ok {
		s.Format = f
	}
	if d, ok := m["description"].(string); ok {
		s.Description = d
	}
	if n, ok := m["nullable"].(bool); ok {
		s.Nullable = n
	}
	if enum, ok := m["enum"].([]any); ok {
		s.Enum = enum
	}
	if def, ok := m["default"]; ok {
		s.Default = def
	}

	// Numeric constraints
	if min, ok := m["minimum"].(float64); ok {
		s.Minimum = &min
	}
	if max, ok := m["maximum"].(float64); ok {
		s.Maximum = &max
	}

	// String constraints
	if minLen, ok := m["minLength"].(int); ok {
		s.MinLength = &minLen
	}
	if maxLen, ok := m["maxLength"].(int); ok {
		s.MaxLength = &maxLen
	}
	if pattern, ok := m["pattern"].(string); ok {
		s.Pattern = pattern
	}

	// Array constraints
	if minItems, ok := m["minItems"].(int); ok {
		s.MinItems = &minItems
	}
	if maxItems, ok := m["maxItems"].(int); ok {
		s.MaxItems = &maxItems
	}
	if unique, ok := m["uniqueItems"].(bool); ok {
		s.UniqueItems = unique
	}

	// Array items
	if items, ok := m["items"].(map[string]any); ok {
		s.Items = convertSchema(items)
	}

	// Object properties
	if props, ok := m["properties"].(map[string]any); ok {
		s.Properties = make(map[string]*Schema)
		for k, v := range props {
			if propMap, ok := v.(map[string]any); ok {
				s.Properties[k] = convertSchema(propMap)
			}
		}
	}

	// Required fields
	if req, ok := m["required"].([]string); ok {
		s.Required = req
	}
	if req, ok := m["required"].([]any); ok {
		for _, r := range req {
			if str, ok := r.(string); ok {
				s.Required = append(s.Required, str)
			}
		}
	}

	// Additional properties
	if addProps, ok := m["additionalProperties"].(map[string]any); ok {
		s.AdditionalProperties = convertSchema(addProps)
	}

	return s
}

// WriteJSON writes the document as JSON.
func (d *OpenRPCDocument) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(d)
}

// MarshalIndent returns formatted JSON.
func (d *OpenRPCDocument) MarshalIndent() ([]byte, error) {
	return json.MarshalIndent(d, "", "  ")
}

// OpenRPCHandler serves OpenRPC documents over HTTP.
type OpenRPCHandler struct {
	document *OpenRPCDocument
	json     []byte
}

// NewOpenRPCHandler creates an OpenRPC handler.
func NewOpenRPCHandler(svc *contract.Service) (*OpenRPCHandler, error) {
	doc := GenerateOpenRPC(svc)
	data, err := doc.MarshalIndent()
	if err != nil {
		return nil, err
	}
	return &OpenRPCHandler{document: doc, json: data}, nil
}

// Name returns the transport name.
func (h *OpenRPCHandler) Name() string {
	return "openrpc"
}

// ServeHTTP serves the OpenRPC document.
func (h *OpenRPCHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(h.json)
}

// Document returns the underlying OpenRPC document.
func (h *OpenRPCHandler) Document() *OpenRPCDocument {
	return h.document
}

// JSON returns the cached JSON bytes.
func (h *OpenRPCHandler) JSON() []byte {
	return h.json
}

// MountOpenRPC registers the OpenRPC handler at the given path.
func MountOpenRPC(mux *http.ServeMux, path string, svc *contract.Service) error {
	if mux == nil {
		return nil
	}
	if path == "" {
		path = "/openrpc.json"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	h, err := NewOpenRPCHandler(svc)
	if err != nil {
		return err
	}

	mux.Handle(path, h)
	return nil
}

// MountWithOpenRPC registers both JSON-RPC handler and OpenRPC spec.
func MountWithOpenRPC(mux *http.ServeMux, rpcPath, specPath string, svc *contract.Service, opts ...Option) error {
	Mount(mux, rpcPath, svc, opts...)
	return MountOpenRPC(mux, specPath, svc)
}
