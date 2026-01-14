// Package openapi provides a minimal but expressive OpenAPI 3.1 model.
//
// It is designed to be generic enough for any HTTP API, while staying
// small and idiomatic. Your data API builder can sit on top of this
// package to generate OpenAPI docs for /data/rest, /data/rpc, etc.
package openapi

import (
	"encoding/json"
	"io"
)

// Document is the root of an OpenAPI 3.1 document.
type Document struct {
	OpenAPI      string                `json:"openapi,omitempty"` // typically "3.1.0"
	Info         *Info                 `json:"info,omitempty"`
	Servers      []*Server             `json:"servers,omitempty"`
	Paths        Paths                 `json:"paths,omitempty"`
	Components   *Components           `json:"components,omitempty"`
	Security     []SecurityRequirement `json:"security,omitempty"`
	Tags         []*Tag                `json:"tags,omitempty"`
	ExternalDocs *ExternalDocs         `json:"externalDocs,omitempty"`
}

// Info provides metadata about the API.
type Info struct {
	Title          string   `json:"title,omitempty"`
	Summary        string   `json:"summary,omitempty"`
	Description    string   `json:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty"`
	License        *License `json:"license,omitempty"`
	Version        string   `json:"version,omitempty"`
}

// Contact information for the exposed API.
type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// License information for the exposed API.
type License struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

// Server represents a server that hosts the API.
type Server struct {
	URL         string                     `json:"url,omitempty"`
	Description string                     `json:"description,omitempty"`
	Variables   map[string]*ServerVariable `json:"variables,omitempty"`
}

// ServerVariable for URL template substitution.
type ServerVariable struct {
	Enum        []string `json:"enum,omitempty"`
	Default     string   `json:"default,omitempty"`
	Description string   `json:"description,omitempty"`
}

// Paths is a map of path templates to path items.
type Paths map[string]*PathItem

// PathItem describes the operations available on a single path.
type PathItem struct {
	Summary     string     `json:"summary,omitempty"`
	Description string     `json:"description,omitempty"`
	Trace       *Operation `json:"trace,omitempty"`
	Head        *Operation `json:"head,omitempty"`
	Options     *Operation `json:"options,omitempty"`
	Get         *Operation `json:"get,omitempty"`
	Put         *Operation `json:"put,omitempty"`
	Post        *Operation `json:"post,omitempty"`
	Delete      *Operation `json:"delete,omitempty"`
	Patch       *Operation `json:"patch,omitempty"`
}

// Operation describes a single API operation on a path.
type Operation struct {
	OperationID  string                `json:"operationId,omitempty"`
	Summary      string                `json:"summary,omitempty"`
	Description  string                `json:"description,omitempty"`
	Tags         []string              `json:"tags,omitempty"`
	Parameters   []*Parameter          `json:"parameters,omitempty"`
	RequestBody  *RequestBody          `json:"requestBody,omitempty"`
	Responses    Responses             `json:"responses,omitempty"`
	Callbacks    map[string]*Callback  `json:"callbacks,omitempty"`
	Deprecated   bool                  `json:"deprecated,omitempty"`
	Security     []SecurityRequirement `json:"security,omitempty"`
	ExternalDocs *ExternalDocs         `json:"externalDocs,omitempty"`
}

// Parameter describes a single operation parameter.
type Parameter struct {
	Name            string     `json:"name,omitempty"`
	In              string     `json:"in,omitempty"` // "query", "header", "path", "cookie"
	Description     string     `json:"description,omitempty"`
	Required        bool       `json:"required,omitempty"`
	Deprecated      bool       `json:"deprecated,omitempty"`
	AllowEmptyValue bool       `json:"allowEmptyValue,omitempty"`
	Schema          *SchemaRef `json:"schema,omitempty"`
}

// RequestBody describes the request body.
type RequestBody struct {
	Description string                `json:"description,omitempty"`
	Content     map[string]*MediaType `json:"content,omitempty"`
	Required    bool                  `json:"required,omitempty"`
}

// Responses is a map of HTTP status code to response.
type Responses map[string]*Response

// Response describes a single response from an API operation.
type Response struct {
	Description string                `json:"description,omitempty"`
	Headers     map[string]*Header    `json:"headers,omitempty"`
	Content     map[string]*MediaType `json:"content,omitempty"`
	Links       map[string]*Link      `json:"links,omitempty"`
}

// Header describes a header object.
type Header struct {
	Description string     `json:"description,omitempty"`
	Required    bool       `json:"required,omitempty"`
	Deprecated  bool       `json:"deprecated,omitempty"`
	Schema      *SchemaRef `json:"schema,omitempty"`
}

// MediaType describes the media type and schema for request or response.
type MediaType struct {
	Schema   *SchemaRef          `json:"schema,omitempty"`
	Example  any                 `json:"example,omitempty"`
	Examples map[string]*Example `json:"examples,omitempty"`
}

// Example describes an example value.
type Example struct {
	Summary       string `json:"summary,omitempty"`
	Description   string `json:"description,omitempty"`
	Value         any    `json:"value,omitempty"`
	ExternalValue string `json:"externalValue,omitempty"`
}

// Link describes a possible design time link for a response.
type Link struct {
	OperationRef string         `json:"operationRef,omitempty"`
	OperationID  string         `json:"operationId,omitempty"`
	Parameters   map[string]any `json:"parameters,omitempty"`
	RequestBody  any            `json:"requestBody,omitempty"`
	Description  string         `json:"description,omitempty"`
}

// Callback map of expression to PathItem.
type Callback map[string]*PathItem

// Components holds various reusable objects.
type Components struct {
	Schemas         map[string]*SchemaRef      `json:"schemas,omitempty"`
	Responses       map[string]*Response       `json:"responses,omitempty"`
	Parameters      map[string]*Parameter      `json:"parameters,omitempty"`
	Examples        map[string]*Example        `json:"examples,omitempty"`
	RequestBodies   map[string]*RequestBody    `json:"requestBodies,omitempty"`
	Headers         map[string]*Header         `json:"headers,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty"`
	Links           map[string]*Link           `json:"links,omitempty"`
	Callbacks       map[string]*Callback       `json:"callbacks,omitempty"`
}

// SchemaRef is either a direct Schema or a reference to a named schema.
type SchemaRef struct {
	Ref    string  `json:"$ref,omitempty"`
	Schema *Schema `json:"-,omitempty"`
}

// MarshalJSON customizes marshalling so that either $ref or the Schema is emitted.
func (r *SchemaRef) MarshalJSON() ([]byte, error) {
	if r == nil {
		return []byte("null"), nil
	}
	if r.Ref != "" {
		type refOnly SchemaRef
		return json.Marshal(&struct {
			*refOnly
		}{
			refOnly: (*refOnly)(r),
		})
	}
	return json.Marshal(r.Schema)
}

// Schema describes a JSON schema used in OpenAPI.
type Schema struct {
	Title                string                `json:"title,omitempty"`
	Description          string                `json:"description,omitempty"`
	Type                 string                `json:"type,omitempty"`
	Format               string                `json:"format,omitempty"`
	Nullable             bool                  `json:"nullable,omitempty"`
	Enum                 []any                 `json:"enum,omitempty"`
	Default              any                   `json:"default,omitempty"`
	Items                *SchemaRef            `json:"items,omitempty"`
	Properties           map[string]*SchemaRef `json:"properties,omitempty"`
	Required             []string              `json:"required,omitempty"`
	AllOf                []*SchemaRef          `json:"allOf,omitempty"`
	OneOf                []*SchemaRef          `json:"oneOf,omitempty"`
	AnyOf                []*SchemaRef          `json:"anyOf,omitempty"`
	Not                  *SchemaRef            `json:"not,omitempty"`
	AdditionalProperties *SchemaRef            `json:"additionalProperties,omitempty"`
	// Numeric constraints
	Minimum          *float64 `json:"minimum,omitempty"`
	Maximum          *float64 `json:"maximum,omitempty"`
	ExclusiveMinimum *float64 `json:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum *float64 `json:"exclusiveMaximum,omitempty"`
	MultipleOf       *float64 `json:"multipleOf,omitempty"`
	// String constraints
	MinLength *int   `json:"minLength,omitempty"`
	MaxLength *int   `json:"maxLength,omitempty"`
	Pattern   string `json:"pattern,omitempty"`
	// Array constraints
	MinItems    *int `json:"minItems,omitempty"`
	MaxItems    *int `json:"maxItems,omitempty"`
	UniqueItems bool `json:"uniqueItems,omitempty"`

	Ref string `json:"$ref,omitempty"`
}

// SecurityScheme describes an authentication scheme.
type SecurityScheme struct {
	Type             string      `json:"type,omitempty"` // "http", "apiKey", "oauth2", "openIdConnect"
	Description      string      `json:"description,omitempty"`
	Name             string      `json:"name,omitempty"`   // for apiKey
	In               string      `json:"in,omitempty"`     // for apiKey: "query", "header", "cookie"
	Scheme           string      `json:"scheme,omitempty"` // for http: "basic", "bearer"
	BearerFormat     string      `json:"bearerFormat,omitempty"`
	Flows            *OAuthFlows `json:"flows,omitempty"`
	OpenIDConnectURL string      `json:"openIdConnectUrl,omitempty"`
}

// OAuthFlows is a container for different OAuth2 flows.
type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty"`
}

// OAuthFlow describes an OAuth2 flow configuration.
type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes,omitempty"`
}

// SecurityRequirement lists required security schemes for an operation or document.
type SecurityRequirement map[string][]string

// Tag adds metadata to a single tag that can be used by operations.
type Tag struct {
	Name         string        `json:"name,omitempty"`
	Description  string        `json:"description,omitempty"`
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty"`
}

// ExternalDocs describes external documentation.
type ExternalDocs struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

// New returns a new Document with sensible defaults.
// You can override fields as needed.
func New() *Document {
	return &Document{
		OpenAPI: "3.1.0",
		Info:    &Info{},
		Paths:   make(Paths),
		Components: &Components{
			Schemas:         make(map[string]*SchemaRef),
			Responses:       make(map[string]*Response),
			Parameters:      make(map[string]*Parameter),
			Examples:        make(map[string]*Example),
			RequestBodies:   make(map[string]*RequestBody),
			Headers:         make(map[string]*Header),
			SecuritySchemes: make(map[string]*SecurityScheme),
			Links:           make(map[string]*Link),
			Callbacks:       make(map[string]*Callback),
		},
	}
}

// AddPathOperation is a small helper to attach an operation to a path and method.
//
// method should be a lowercase HTTP method string such as "get", "post", "put", etc.
func (d *Document) AddPathOperation(path, method string, op *Operation) {
	if d.Paths == nil {
		d.Paths = make(Paths)
	}
	item, ok := d.Paths[path]
	if !ok {
		item = &PathItem{}
		d.Paths[path] = item
	}
	switch method {
	case "get":
		item.Get = op
	case "post":
		item.Post = op
	case "put":
		item.Put = op
	case "delete":
		item.Delete = op
	case "patch":
		item.Patch = op
	case "head":
		item.Head = op
	case "options":
		item.Options = op
	case "trace":
		item.Trace = op
	default:
		// unknown method, ignore for now
	}
}

// AddSchema registers a named schema under components.schemas and returns a reference.
func (d *Document) AddSchema(name string, schema *Schema) *SchemaRef {
	if d.Components == nil {
		d.Components = &Components{}
	}
	if d.Components.Schemas == nil {
		d.Components.Schemas = make(map[string]*SchemaRef)
	}
	ref := &SchemaRef{
		Ref: "#/components/schemas/" + name,
	}
	d.Components.Schemas[name] = &SchemaRef{Schema: schema}
	return ref
}

// WriteJSON encodes the document as pretty printed JSON to the given writer.
func (d *Document) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(d)
}
