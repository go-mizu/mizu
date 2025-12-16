package contract

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu/openapi"
)

// GenerateOpenAPI creates an OpenAPI 3.1 document from services.
func GenerateOpenAPI(services ...*Service) *openapi.Document {
	doc := openapi.New()
	doc.Info = &openapi.Info{
		Title:   "API",
		Version: "1.0.0",
	}

	// Combine service info
	if len(services) == 1 {
		svc := services[0]
		doc.Info.Title = svc.Name + " API"
		if svc.Description != "" {
			doc.Info.Description = svc.Description
		}
		if svc.Version != "" {
			doc.Info.Version = svc.Version
		}
	}

	// Add schemas and paths for each service
	for _, svc := range services {
		addServiceToOpenAPI(doc, svc)
	}

	return doc
}

func addServiceToOpenAPI(doc *openapi.Document, svc *Service) {
	// Add schemas
	for _, schema := range svc.Types.Schemas() {
		s := convertJSONSchemaToOpenAPI(schema.JSON)
		doc.AddSchema(schema.ID, s)
	}

	basePath := "/" + pluralize(svc.Name)

	for _, m := range svc.Methods {
		// Determine HTTP method and path
		httpMethod := strings.ToLower(m.HTTPMethod)
		if httpMethod == "" {
			httpMethod = strings.ToLower(restVerb(m.Name))
		}

		path := m.HTTPPath
		if path == "" {
			path = basePath
			if needsID(m) {
				path = basePath + "/{id}"
			}
		}

		op := &openapi.Operation{
			OperationID: m.FullName,
			Summary:     m.Summary,
			Description: m.Description,
			Tags:        m.Tags,
			Deprecated:  m.Deprecated,
			Responses:   make(openapi.Responses),
		}

		// Add parameters for path variables
		if strings.Contains(path, "{id}") {
			op.Parameters = append(op.Parameters, &openapi.Parameter{
				Name:        "id",
				In:          "path",
				Required:    true,
				Description: "Resource ID",
				Schema:      &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}},
			})
		}

		// Add request body
		if m.Input != nil {
			op.RequestBody = &openapi.RequestBody{
				Required: true,
				Content: map[string]*openapi.MediaType{
					"application/json": {
						Schema: &openapi.SchemaRef{
							Ref: "#/components/schemas/" + m.Input.ID,
						},
					},
				},
			}
		}

		// Add responses
		successResp := &openapi.Response{
			Description: "Successful response",
		}
		if m.Output != nil {
			successResp.Content = map[string]*openapi.MediaType{
				"application/json": {
					Schema: &openapi.SchemaRef{
						Ref: "#/components/schemas/" + m.Output.ID,
					},
				},
			}
		}
		op.Responses["200"] = successResp

		// Add error responses
		op.Responses["400"] = &openapi.Response{
			Description: "Invalid request",
			Content: map[string]*openapi.MediaType{
				"application/json": {
					Schema: &openapi.SchemaRef{
						Schema: errorResponseSchema(),
					},
				},
			},
		}
		op.Responses["500"] = &openapi.Response{
			Description: "Internal server error",
			Content: map[string]*openapi.MediaType{
				"application/json": {
					Schema: &openapi.SchemaRef{
						Schema: errorResponseSchema(),
					},
				},
			},
		}

		doc.AddPathOperation(path, httpMethod, op)
	}
}

func convertJSONSchemaToOpenAPI(json map[string]any) *openapi.Schema {
	s := &openapi.Schema{}

	if t, ok := json["type"].(string); ok {
		s.Type = t
	}
	if f, ok := json["format"].(string); ok {
		s.Format = f
	}
	if d, ok := json["description"].(string); ok {
		s.Description = d
	}
	if n, ok := json["nullable"].(bool); ok {
		s.Nullable = n
	}
	if enum, ok := json["enum"].([]any); ok {
		s.Enum = enum
	}
	if def, ok := json["default"]; ok {
		s.Default = def
	}

	// Numeric constraints
	if min, ok := json["minimum"].(float64); ok {
		s.Minimum = &min
	}
	if max, ok := json["maximum"].(float64); ok {
		s.Maximum = &max
	}

	// String constraints
	if minLen, ok := json["minLength"].(int); ok {
		s.MinLength = &minLen
	}
	if maxLen, ok := json["maxLength"].(int); ok {
		s.MaxLength = &maxLen
	}
	if pattern, ok := json["pattern"].(string); ok {
		s.Pattern = pattern
	}

	// Array constraints
	if minItems, ok := json["minItems"].(int); ok {
		s.MinItems = &minItems
	}
	if maxItems, ok := json["maxItems"].(int); ok {
		s.MaxItems = &maxItems
	}
	if unique, ok := json["uniqueItems"].(bool); ok {
		s.UniqueItems = unique
	}

	// Array items
	if items, ok := json["items"].(map[string]any); ok {
		s.Items = &openapi.SchemaRef{
			Schema: convertJSONSchemaToOpenAPI(items),
		}
	}

	// Object properties
	if props, ok := json["properties"].(map[string]any); ok {
		s.Properties = make(map[string]*openapi.SchemaRef)
		for k, v := range props {
			if propMap, ok := v.(map[string]any); ok {
				s.Properties[k] = &openapi.SchemaRef{
					Schema: convertJSONSchemaToOpenAPI(propMap),
				}
			}
		}
	}

	// Required fields
	if req, ok := json["required"].([]string); ok {
		s.Required = req
	}
	if req, ok := json["required"].([]any); ok {
		for _, r := range req {
			if str, ok := r.(string); ok {
				s.Required = append(s.Required, str)
			}
		}
	}

	// Additional properties
	if addProps, ok := json["additionalProperties"].(map[string]any); ok {
		s.AdditionalProperties = &openapi.SchemaRef{
			Schema: convertJSONSchemaToOpenAPI(addProps),
		}
	}

	return s
}

func errorResponseSchema() *openapi.Schema {
	return &openapi.Schema{
		Type: "object",
		Properties: map[string]*openapi.SchemaRef{
			"code": {
				Schema: &openapi.Schema{
					Type:        "string",
					Description: "Error code",
				},
			},
			"message": {
				Schema: &openapi.Schema{
					Type:        "string",
					Description: "Error message",
				},
			},
			"details": {
				Schema: &openapi.Schema{
					Type:        "object",
					Description: "Additional error details",
				},
			},
		},
		Required: []string{"code", "message"},
	}
}

// ServeOpenAPIJSON serves the OpenAPI document as JSON.
func ServeOpenAPIJSON(mux *http.ServeMux, path string, services ...*Service) {
	if path == "" {
		path = "/openapi.json"
	}

	doc := GenerateOpenAPI(services...)

	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = doc.WriteJSON(w)
	})
}

// OpenAPIHandler returns an http.Handler that serves the OpenAPI document.
func OpenAPIHandler(services ...*Service) http.Handler {
	doc := GenerateOpenAPI(services...)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = doc.WriteJSON(w)
	})
}

// OpenAPIWithDocsHandler returns handlers for both OpenAPI spec and docs UI.
func OpenAPIWithDocsHandler(services ...*Service) (specHandler, docsHandler http.Handler) {
	doc := GenerateOpenAPI(services...)

	specHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = doc.WriteJSON(w)
	})

	// Create docs handler using the openapi package
	docsH, err := openapi.NewHandler(openapi.Config{
		SpecURL:   "/openapi.json",
		DefaultUI: "scalar",
	})
	if err != nil {
		// Fallback to simple redirect
		docsHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/openapi.json", http.StatusFound)
		})
	} else {
		docsHandler = docsH
	}

	return specHandler, docsHandler
}

// GenerateOpenAPIJSON generates OpenAPI JSON bytes.
func GenerateOpenAPIJSON(services ...*Service) ([]byte, error) {
	doc := GenerateOpenAPI(services...)
	return json.MarshalIndent(doc, "", "  ")
}
