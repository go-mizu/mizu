// Package rest provides REST transport and OpenAPI 3.1 specification generation.
//
// This package handles HTTP requests following RESTful conventions and generates
// valid OpenAPI 3.1 documents from registered contract services.
//
// Usage:
//
//	svc, _ := contract.Register("todo", &TodoService{})
//	rest.MountWithSpec(mux, "/openapi.json", svc)
package rest

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/go-mizu/mizu/contract"
)

// Version is the OpenAPI specification version.
const Version = "3.1.0"

// Document represents an OpenAPI 3.1 document.
type Document struct {
	OpenAPI    string                `json:"openapi"`
	Info       *Info                 `json:"info"`
	Servers    []*Server             `json:"servers,omitempty"`
	Paths      map[string]*PathItem  `json:"paths,omitempty"`
	Components *Components           `json:"components,omitempty"`
	Tags       []*Tag                `json:"tags,omitempty"`
}

// Info provides metadata about the API.
type Info struct {
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty"`
	License        *License `json:"license,omitempty"`
	Version        string   `json:"version"`
}

// Contact information for the API.
type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// License information for the API.
type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// Server represents a server.
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// Tag adds metadata to operations.
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// PathItem describes operations available on a single path.
type PathItem struct {
	Summary     string     `json:"summary,omitempty"`
	Description string     `json:"description,omitempty"`
	Get         *Operation `json:"get,omitempty"`
	Put         *Operation `json:"put,omitempty"`
	Post        *Operation `json:"post,omitempty"`
	Delete      *Operation `json:"delete,omitempty"`
	Patch       *Operation `json:"patch,omitempty"`
	Options     *Operation `json:"options,omitempty"`
	Head        *Operation `json:"head,omitempty"`
	Trace       *Operation `json:"trace,omitempty"`
	Parameters  []*Param   `json:"parameters,omitempty"`
}

// SetOperation sets an operation for the given HTTP method.
func (p *PathItem) SetOperation(method string, op *Operation) {
	switch strings.ToLower(method) {
	case "get":
		p.Get = op
	case "put":
		p.Put = op
	case "post":
		p.Post = op
	case "delete":
		p.Delete = op
	case "patch":
		p.Patch = op
	case "options":
		p.Options = op
	case "head":
		p.Head = op
	case "trace":
		p.Trace = op
	}
}

// Operation describes a single API operation on a path.
type Operation struct {
	OperationID string              `json:"operationId,omitempty"`
	Summary     string              `json:"summary,omitempty"`
	Description string              `json:"description,omitempty"`
	Tags        []string            `json:"tags,omitempty"`
	Deprecated  bool                `json:"deprecated,omitempty"`
	Parameters  []*Param            `json:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]*Resp    `json:"responses"`
	Security    []map[string][]string `json:"security,omitempty"`
}

// Param describes a single operation parameter.
type Param struct {
	Name        string     `json:"name"`
	In          string     `json:"in"` // query, header, path, cookie
	Description string     `json:"description,omitempty"`
	Required    bool       `json:"required,omitempty"`
	Deprecated  bool       `json:"deprecated,omitempty"`
	Schema      *SchemaRef `json:"schema,omitempty"`
}

// RequestBody describes a request body.
type RequestBody struct {
	Description string                `json:"description,omitempty"`
	Required    bool                  `json:"required,omitempty"`
	Content     map[string]*MediaType `json:"content"`
}

// MediaType describes a media type.
type MediaType struct {
	Schema *SchemaRef `json:"schema,omitempty"`
}

// Resp describes a single response.
type Resp struct {
	Description string                `json:"description"`
	Headers     map[string]*Header    `json:"headers,omitempty"`
	Content     map[string]*MediaType `json:"content,omitempty"`
}

// Header describes a header.
type Header struct {
	Description string     `json:"description,omitempty"`
	Required    bool       `json:"required,omitempty"`
	Schema      *SchemaRef `json:"schema,omitempty"`
}

// Components holds reusable objects.
type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty"`
	Responses       map[string]*Resp           `json:"responses,omitempty"`
	Parameters      map[string]*Param          `json:"parameters,omitempty"`
	RequestBodies   map[string]*RequestBody    `json:"requestBodies,omitempty"`
	Headers         map[string]*Header         `json:"headers,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty"`
}

// SecurityScheme describes a security scheme.
type SecurityScheme struct {
	Type   string `json:"type"`
	Scheme string `json:"scheme,omitempty"`
	Name   string `json:"name,omitempty"`
	In     string `json:"in,omitempty"`
}

// Schema represents a JSON Schema.
type Schema struct {
	Type                 string             `json:"type,omitempty"`
	Format               string             `json:"format,omitempty"`
	Description          string             `json:"description,omitempty"`
	Default              any                `json:"default,omitempty"`
	Nullable             bool               `json:"nullable,omitempty"`
	ReadOnly             bool               `json:"readOnly,omitempty"`
	WriteOnly            bool               `json:"writeOnly,omitempty"`
	Deprecated           bool               `json:"deprecated,omitempty"`
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

// UnmarshalJSON implements json.Unmarshaler.
func (s *SchemaRef) UnmarshalJSON(data []byte) error {
	var ref struct {
		Ref string `json:"$ref"`
	}
	if err := json.Unmarshal(data, &ref); err == nil && ref.Ref != "" {
		s.Ref = ref.Ref
		return nil
	}
	s.Schema = &Schema{}
	return json.Unmarshal(data, s.Schema)
}

// NewDocument creates a new OpenAPI document.
func NewDocument() *Document {
	return &Document{
		OpenAPI: Version,
		Info: &Info{
			Title:   "API",
			Version: "1.0.0",
		},
		Paths: make(map[string]*PathItem),
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}
}

// Generate creates an OpenAPI document from services.
func Generate(services ...*contract.Service) *Document {
	doc := NewDocument()

	// Use first service for info if single service
	if len(services) == 1 {
		svc := services[0]
		doc.Info.Title = svc.Name + " API"
		if svc.Description != "" {
			doc.Info.Description = svc.Description
		}
		if svc.Version != "" {
			doc.Info.Version = svc.Version
		}
		if len(svc.Tags) > 0 {
			for _, tag := range svc.Tags {
				doc.Tags = append(doc.Tags, &Tag{Name: tag})
			}
		}
	}

	for _, svc := range services {
		addService(doc, svc)
	}

	return doc
}

func addService(doc *Document, svc *contract.Service) {
	// Add schemas
	for _, t := range svc.Types.All() {
		schema := svc.Types.Schema(t.ID)
		if schema != nil {
			doc.Components.Schemas[t.ID] = convertSchema(schema)
		}
	}

	basePath := "/" + pluralize(svc.Name)

	for _, m := range svc.Methods {
		addMethod(doc, svc, m, basePath)
	}
}

func addMethod(doc *Document, svc *contract.Service, m *contract.Method, basePath string) {
	// Determine HTTP method and path
	httpMethod := m.HTTPMethod
	if httpMethod == "" {
		httpMethod = restVerb(m.Name)
	}

	path := m.HTTPPath
	if path == "" {
		path = basePath
		if needsID(m) {
			path = basePath + "/{id}"
		}
	}

	// Get or create path item
	pathItem, ok := doc.Paths[path]
	if !ok {
		pathItem = &PathItem{}
		doc.Paths[path] = pathItem
	}

	// Create operation
	op := &Operation{
		OperationID: m.FullName,
		Summary:     m.Summary,
		Description: m.Description,
		Tags:        m.Tags,
		Deprecated:  m.Deprecated,
		Responses:   make(map[string]*Resp),
	}

	// Add path parameters
	if strings.Contains(path, "{id}") {
		op.Parameters = append(op.Parameters, &Param{
			Name:        "id",
			In:          "path",
			Required:    true,
			Description: "Resource ID",
			Schema:      &SchemaRef{Schema: &Schema{Type: "string"}},
		})
	}

	// Add request body
	if m.Input != nil {
		op.RequestBody = &RequestBody{
			Required: true,
			Content: map[string]*MediaType{
				"application/json": {
					Schema: &SchemaRef{
						Ref: "#/components/schemas/" + m.Input.ID,
					},
				},
			},
		}
	}

	// Add success response
	successResp := &Resp{
		Description: "Successful response",
	}
	if m.Output != nil {
		successResp.Content = map[string]*MediaType{
			"application/json": {
				Schema: &SchemaRef{
					Ref: "#/components/schemas/" + m.Output.ID,
				},
			},
		}
	}
	op.Responses["200"] = successResp

	// Add standard error responses
	op.Responses["400"] = &Resp{
		Description: "Bad request",
		Content: map[string]*MediaType{
			"application/json": {
				Schema: &SchemaRef{Schema: errorSchema()},
			},
		},
	}
	op.Responses["500"] = &Resp{
		Description: "Internal server error",
		Content: map[string]*MediaType{
			"application/json": {
				Schema: &SchemaRef{Schema: errorSchema()},
			},
		},
	}

	pathItem.SetOperation(httpMethod, op)
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

func errorSchema() *Schema {
	return &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"code": {
				Type:        "string",
				Description: "Error code",
			},
			"message": {
				Type:        "string",
				Description: "Error message",
			},
			"details": {
				Type:        "object",
				Description: "Additional error details",
			},
		},
		Required: []string{"code", "message"},
	}
}

// WriteJSON writes the document as JSON.
func (d *Document) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(d)
}

// MarshalJSON returns the JSON representation.
func (d *Document) MarshalJSON() ([]byte, error) {
	type alias Document
	return json.Marshal((*alias)(d))
}

// MarshalIndent returns formatted JSON.
func (d *Document) MarshalIndent() ([]byte, error) {
	return json.MarshalIndent(d, "", "  ")
}

