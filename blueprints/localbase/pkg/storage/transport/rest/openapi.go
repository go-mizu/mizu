// File: lib/storage/transport/rest/openapi.go
package rest

import (
	"net/http"

	"github.com/go-mizu/mizu/blueprints/localbase/pkg/openapi"
	"github.com/go-mizu/mizu"
)

// RegisterDocs registers OpenAPI documentation endpoints.
//
// This adds:
//   - GET {docsPath}          - OpenAPI JSON specification
//   - GET {docsPath}/         - API documentation UI (Scalar by default)
//
// Example:
//
//	rest.RegisterDocs(app, "/storage/v1", "/docs", "Storage API", "1.0.0")
//
// This would create:
//   - GET /docs      -> OpenAPI JSON
//   - GET /docs/     -> Interactive documentation UI
//
// The documentation UI can be selected using ?ui=scalar|redoc|swagger|rapidoc|stoplight
func RegisterDocs(app *mizu.App, basePath, docsPath, title, version string) {
	doc := NewDocument(title, version, basePath)

	// Serve OpenAPI JSON
	app.Get(docsPath, func(c *mizu.Ctx) error {
		c.Writer().Header().Set("Content-Type", "application/json")
		return c.JSON(http.StatusOK, doc)
	})

	// Serve documentation UI
	docsHandler, err := openapi.NewHandler(openapi.Config{
		SpecURL:   docsPath,
		DefaultUI: "scalar",
	})
	if err != nil {
		// If handler creation fails, serve a simple redirect to JSON
		app.Get(docsPath+"/", func(c *mizu.Ctx) error {
			http.Redirect(c.Writer(), c.Request(), docsPath, http.StatusTemporaryRedirect)
			return nil
		})
		return
	}

	app.Get(docsPath+"/", func(c *mizu.Ctx) error {
		docsHandler.ServeHTTP(c.Writer(), c.Request())
		return nil
	})
}

// NewDocument builds an OpenAPI 3.1 document that describes the Supabase-compatible
// Storage REST API for bucket and object management.
//
// Bucket endpoints:
//
//	POST   /bucket              - create bucket
//	GET    /bucket              - list buckets
//	GET    /bucket/{bucketId}   - get bucket details
//	PUT    /bucket/{bucketId}   - update bucket
//	DELETE /bucket/{bucketId}   - delete bucket
//	POST   /bucket/{bucketId}/empty - empty bucket
//
// Object endpoints:
//
//	POST   /object/{bucketName}/{path}       - upload object
//	GET    /object/{bucketName}/{path}       - download object
//	PUT    /object/{bucketName}/{path}       - update object
//	DELETE /object/{bucketName}/{path}       - delete object
//	DELETE /object/{bucketName}              - delete multiple objects
//	POST   /object/list/{bucketName}         - list objects
//	POST   /object/move                      - move object
//	POST   /object/copy                      - copy object
//	POST   /object/sign/{bucketName}/{path}  - create signed URL
//	POST   /object/sign/{bucketName}         - create multiple signed URLs
//	POST   /object/upload/sign/{bucketName}/{path} - create signed upload URL
//	GET    /object/public/{bucketName}/{path}      - get public object
//	GET    /object/authenticated/{bucketName}/{path} - get authenticated object
//	GET    /object/info/{bucketName}/{path}        - get object info
//	GET    /object/info/public/{bucketName}/{path} - get public object info
//	GET    /object/info/authenticated/{bucketName}/{path} - get authenticated object info
//
// TUS Resumable Upload endpoints (TUS 1.0.0):
//
//	OPTIONS /upload/resumable/                  - TUS capabilities discovery
//	POST    /upload/resumable/{bucketName}/{path} - create resumable upload
//	PATCH   /upload/resumable/{bucketName}/{path} - upload chunk
//	HEAD    /upload/resumable/{bucketName}/{path} - get upload status
//	DELETE  /upload/resumable/{bucketName}/{path} - cancel upload
//
// basePath is the URL prefix for storage endpoints, for example "/" or "/storage/v1".
func NewDocument(title, version, basePath string) *openapi.Document {
	if basePath == "" {
		basePath = "/"
	}
	if basePath != "/" && basePath[len(basePath)-1] == '/' {
		basePath = basePath[:len(basePath)-1]
	}

	doc := openapi.New()
	doc.Info.Title = title
	doc.Info.Version = version
	doc.Info.Description = "Supabase-compatible Storage REST API for managing buckets and objects. " +
		"This API provides a complete interface for object storage operations including uploads, downloads, " +
		"signed URLs, and access control."

	doc.Tags = []*openapi.Tag{
		{
			Name: "Buckets",
			Description: "Bucket management operations. Buckets are containers for storing objects. " +
				"Each bucket can be configured as public or private, with optional file size limits and MIME type restrictions.",
		},
		{
			Name: "Objects",
			Description: "Object storage operations. Objects are the actual files stored in buckets. " +
				"Supports upload, download, move, copy, and signed URL generation for secure temporary access.",
		},
	}

	// Register common schemas
	registerStorageSchemas(doc)

	// Register bucket operations
	addBucketOperations(doc, basePath)

	// Register object operations
	addObjectOperations(doc, basePath)

	return doc
}

// registerStorageSchemas registers shared schemas for the Storage API.
func registerStorageSchemas(doc *openapi.Document) {
	// Error schema
	doc.AddSchema("Error", &openapi.Schema{
		Type:  "object",
		Title: "Error",
		Description: "Error response following Supabase Storage API format. " +
			"Contains status code, error type, and detailed message.",
		Properties: map[string]*openapi.SchemaRef{
			"statusCode": {
				Schema: &openapi.Schema{
					Type:        "integer",
					Description: "HTTP status code for the error.",
				},
			},
			"error": {
				Schema: &openapi.Schema{
					Type:        "string",
					Description: "Error type or category (e.g. 'Bad Request', 'Not Found').",
				},
			},
			"message": {
				Schema: &openapi.Schema{
					Type:        "string",
					Description: "Human readable error message with details.",
				},
			},
		},
		Required: []string{"statusCode", "error", "message"},
	})

	// Bucket schema
	doc.AddSchema("Bucket", &openapi.Schema{
		Type:  "object",
		Title: "Bucket",
		Description: "Represents a storage bucket. Buckets are containers that hold objects and define " +
			"access policies, size limits, and allowed file types.",
		Properties: map[string]*openapi.SchemaRef{
			"id": {
				Schema: &openapi.Schema{
					Type:        "string",
					Description: "Unique identifier for the bucket.",
				},
			},
			"name": {
				Schema: &openapi.Schema{
					Type:        "string",
					Description: "Bucket name. Must be unique within the storage instance.",
				},
			},
			"owner": {
				Schema: &openapi.Schema{
					Type:        "string",
					Description: "Owner identifier for the bucket.",
					Nullable:    true,
				},
			},
			"public": {
				Schema: &openapi.Schema{
					Type:        "boolean",
					Description: "Whether the bucket is publicly accessible without authentication.",
				},
			},
			"type": {
				Schema: &openapi.Schema{
					Type:        "string",
					Description: "Bucket type: STANDARD or ANALYTICS.",
					Enum:        []any{"STANDARD", "ANALYTICS"},
					Nullable:    true,
				},
			},
			"file_size_limit": {
				Schema: &openapi.Schema{
					Type:        "integer",
					Format:      "int64",
					Description: "Maximum file size in bytes allowed for objects in this bucket.",
					Nullable:    true,
				},
			},
			"allowed_mime_types": {
				Schema: &openapi.Schema{
					Type:        "array",
					Description: "List of allowed MIME types for objects in this bucket.",
					Items: &openapi.SchemaRef{
						Schema: &openapi.Schema{Type: "string"},
					},
					Nullable: true,
				},
			},
			"created_at": {
				Schema: &openapi.Schema{
					Type:        "string",
					Format:      "date-time",
					Description: "Timestamp when the bucket was created.",
				},
			},
			"updated_at": {
				Schema: &openapi.Schema{
					Type:        "string",
					Format:      "date-time",
					Description: "Timestamp when the bucket was last updated.",
				},
			},
		},
		Required: []string{"id", "name", "public", "created_at", "updated_at"},
	})

	// Object/File metadata schema
	doc.AddSchema("ObjectMetadata", &openapi.Schema{
		Type:  "object",
		Title: "ObjectMetadata",
		Description: "Metadata about a stored object/file. Includes name, timestamps, owner, " +
			"and custom metadata key-value pairs.",
		Properties: map[string]*openapi.SchemaRef{
			"id": {
				Schema: &openapi.Schema{
					Type:        "string",
					Description: "Object identifier (usually the key/path).",
					Nullable:    true,
				},
			},
			"name": {
				Schema: &openapi.Schema{
					Type:        "string",
					Description: "Object name (file name without path).",
				},
			},
			"bucket_id": {
				Schema: &openapi.Schema{
					Type:        "string",
					Description: "Bucket containing this object.",
					Nullable:    true,
				},
			},
			"owner": {
				Schema: &openapi.Schema{
					Type:        "string",
					Description: "Owner identifier for the object.",
					Nullable:    true,
				},
			},
			"created_at": {
				Schema: &openapi.Schema{
					Type:        "string",
					Format:      "date-time",
					Description: "Timestamp when the object was created.",
					Nullable:    true,
				},
			},
			"updated_at": {
				Schema: &openapi.Schema{
					Type:        "string",
					Format:      "date-time",
					Description: "Timestamp when the object was last updated.",
					Nullable:    true,
				},
			},
			"last_accessed_at": {
				Schema: &openapi.Schema{
					Type:        "string",
					Format:      "date-time",
					Description: "Timestamp when the object was last accessed.",
					Nullable:    true,
				},
			},
			"metadata": {
				Schema: &openapi.Schema{
					Type:        "object",
					Description: "Custom metadata key-value pairs.",
					AdditionalProperties: &openapi.SchemaRef{
						Schema: &openapi.Schema{Type: "string"},
					},
					Nullable: true,
				},
			},
		},
		Required: []string{"name"},
	})

	// Message response schema
	doc.AddSchema("Message", &openapi.Schema{
		Type:        "object",
		Title:       "Message",
		Description: "Generic message response for operations that don't return specific data.",
		Properties: map[string]*openapi.SchemaRef{
			"message": {
				Schema: &openapi.Schema{
					Type:        "string",
					Description: "Success or status message.",
				},
			},
		},
		Required: []string{"message"},
	})
}

// addBucketOperations adds all bucket-related operations to the OpenAPI document.
func addBucketOperations(doc *openapi.Document, basePath string) {
	errorRef := &openapi.SchemaRef{Ref: "#/components/schemas/Error"}
	bucketRef := &openapi.SchemaRef{Ref: "#/components/schemas/Bucket"}
	messageRef := &openapi.SchemaRef{Ref: "#/components/schemas/Message"}

	// POST /bucket - Create bucket
	doc.AddPathOperation(basePath+"/bucket", "post", &openapi.Operation{
		OperationID: "CreateBucket",
		Summary:     "Create a new bucket",
		Description: "Creates a new storage bucket with the specified configuration. Bucket names must be unique.",
		Tags:        []string{"Buckets"},
		RequestBody: &openapi.RequestBody{
			Description: "Bucket configuration",
			Required:    true,
			Content: map[string]*openapi.MediaType{
				"application/json": {
					Schema: &openapi.SchemaRef{
						Schema: &openapi.Schema{
							Type: "object",
							Properties: map[string]*openapi.SchemaRef{
								"name":               {Schema: &openapi.Schema{Type: "string", Description: "Bucket name (required)"}},
								"id":                 {Schema: &openapi.Schema{Type: "string", Description: "Bucket ID (optional, defaults to name)"}},
								"public":             {Schema: &openapi.Schema{Type: "boolean", Description: "Make bucket publicly accessible"}},
								"type":               {Schema: &openapi.Schema{Type: "string", Enum: []any{"STANDARD", "ANALYTICS"}, Description: "Bucket type"}},
								"file_size_limit":    {Schema: &openapi.Schema{Type: "integer", Format: "int64", Description: "Maximum file size in bytes"}},
								"allowed_mime_types": {Schema: &openapi.Schema{Type: "array", Items: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Allowed MIME types"}},
							},
							Required: []string{"name"},
						},
					},
				},
			},
		},
		Responses: openapi.Responses{
			"200": {
				Description: "Bucket created successfully",
				Content: map[string]*openapi.MediaType{
					"application/json": {
						Schema: &openapi.SchemaRef{
							Schema: &openapi.Schema{
								Type:       "object",
								Properties: map[string]*openapi.SchemaRef{"name": {Schema: &openapi.Schema{Type: "string"}}},
							},
						},
					},
				},
			},
			"400": {Description: "Invalid request", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"409": {Description: "Bucket already exists", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"500": {Description: "Server error", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
		},
	})

	// GET /bucket - List buckets
	doc.AddPathOperation(basePath+"/bucket", "get", &openapi.Operation{
		OperationID: "ListBuckets",
		Summary:     "List all buckets",
		Description: "Retrieves a list of all storage buckets with optional filtering and pagination.",
		Tags:        []string{"Buckets"},
		Parameters: []*openapi.Parameter{
			{Name: "limit", In: "query", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "integer"}}, Description: "Maximum number of buckets to return"},
			{Name: "offset", In: "query", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "integer"}}, Description: "Number of buckets to skip"},
			{Name: "sortColumn", In: "query", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Column to sort by"},
			{Name: "sortOrder", In: "query", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string", Enum: []any{"asc", "desc"}}}, Description: "Sort order"},
			{Name: "search", In: "query", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Search term"},
		},
		Responses: openapi.Responses{
			"200": {
				Description: "List of buckets",
				Content: map[string]*openapi.MediaType{
					"application/json": {Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "array", Items: bucketRef}}},
				},
			},
			"500": {Description: "Server error", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
		},
	})

	// GET /bucket/{bucketId} - Get bucket
	doc.AddPathOperation(basePath+"/bucket/{bucketId}", "get", &openapi.Operation{
		OperationID: "GetBucket",
		Summary:     "Get bucket details",
		Description: "Retrieves detailed information about a specific bucket.",
		Tags:        []string{"Buckets"},
		Parameters: []*openapi.Parameter{
			{Name: "bucketId", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Bucket ID"},
		},
		Responses: openapi.Responses{
			"200": {Description: "Bucket details", Content: map[string]*openapi.MediaType{"application/json": {Schema: bucketRef}}},
			"404": {Description: "Bucket not found", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"500": {Description: "Server error", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
		},
	})

	// PUT /bucket/{bucketId} - Update bucket
	doc.AddPathOperation(basePath+"/bucket/{bucketId}", "put", &openapi.Operation{
		OperationID: "UpdateBucket",
		Summary:     "Update bucket properties",
		Description: "Updates the configuration of an existing bucket.",
		Tags:        []string{"Buckets"},
		Parameters: []*openapi.Parameter{
			{Name: "bucketId", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Bucket ID"},
		},
		RequestBody: &openapi.RequestBody{
			Description: "Properties to update",
			Required:    true,
			Content: map[string]*openapi.MediaType{
				"application/json": {
					Schema: &openapi.SchemaRef{
						Schema: &openapi.Schema{
							Type: "object",
							Properties: map[string]*openapi.SchemaRef{
								"public":             {Schema: &openapi.Schema{Type: "boolean", Description: "Make bucket public or private"}},
								"file_size_limit":    {Schema: &openapi.Schema{Type: "integer", Format: "int64", Description: "Maximum file size"}},
								"allowed_mime_types": {Schema: &openapi.Schema{Type: "array", Items: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Allowed MIME types"}},
							},
						},
					},
				},
			},
		},
		Responses: openapi.Responses{
			"200": {Description: "Bucket updated", Content: map[string]*openapi.MediaType{"application/json": {Schema: messageRef}}},
			"404": {Description: "Bucket not found", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"500": {Description: "Server error", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
		},
	})

	// DELETE /bucket/{bucketId} - Delete bucket
	doc.AddPathOperation(basePath+"/bucket/{bucketId}", "delete", &openapi.Operation{
		OperationID: "DeleteBucket",
		Summary:     "Delete a bucket",
		Description: "Permanently deletes a bucket. The bucket must be empty unless force is used.",
		Tags:        []string{"Buckets"},
		Parameters: []*openapi.Parameter{
			{Name: "bucketId", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Bucket ID"},
		},
		Responses: openapi.Responses{
			"200": {Description: "Bucket deleted", Content: map[string]*openapi.MediaType{"application/json": {Schema: messageRef}}},
			"404": {Description: "Bucket not found", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"500": {Description: "Server error", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
		},
	})

	// POST /bucket/{bucketId}/empty - Empty bucket
	doc.AddPathOperation(basePath+"/bucket/{bucketId}/empty", "post", &openapi.Operation{
		OperationID: "EmptyBucket",
		Summary:     "Empty a bucket",
		Description: "Deletes all objects in a bucket without deleting the bucket itself.",
		Tags:        []string{"Buckets"},
		Parameters: []*openapi.Parameter{
			{Name: "bucketId", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Bucket ID"},
		},
		Responses: openapi.Responses{
			"200": {Description: "Bucket emptied", Content: map[string]*openapi.MediaType{"application/json": {Schema: messageRef}}},
			"404": {Description: "Bucket not found", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"500": {Description: "Server error", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
		},
	})
}

// addObjectOperations adds all object-related operations to the OpenAPI document.
func addObjectOperations(doc *openapi.Document, basePath string) {
	errorRef := &openapi.SchemaRef{Ref: "#/components/schemas/Error"}
	objectRef := &openapi.SchemaRef{Ref: "#/components/schemas/ObjectMetadata"}
	messageRef := &openapi.SchemaRef{Ref: "#/components/schemas/Message"}

	// Add Upload tag for TUS operations
	doc.Tags = append(doc.Tags, &openapi.Tag{
		Name: "Upload",
		Description: "TUS protocol resumable upload operations. Supports chunked uploads with pause/resume capability " +
			"following the TUS 1.0.0 specification.",
	})

	// POST /object/{bucketName}/{path} - Upload object
	doc.AddPathOperation(basePath+"/object/{bucketName}/{path}", "post", &openapi.Operation{
		OperationID: "UploadObject",
		Summary:     "Upload an object",
		Description: "Uploads a new object to the specified bucket and path. Use x-upsert header to overwrite existing objects.",
		Tags:        []string{"Objects"},
		Parameters: []*openapi.Parameter{
			{Name: "bucketName", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Bucket name"},
			{Name: "path", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Object path (can include folders)"},
			{Name: "x-upsert", In: "header", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string", Enum: []any{"true", "false"}}}, Description: "Allow overwriting existing objects"},
		},
		RequestBody: &openapi.RequestBody{
			Description: "File content",
			Required:    true,
			Content:     map[string]*openapi.MediaType{"application/octet-stream": {}},
		},
		Responses: openapi.Responses{
			"200": {
				Description: "Object uploaded",
				Content: map[string]*openapi.MediaType{
					"application/json": {
						Schema: &openapi.SchemaRef{
							Schema: &openapi.Schema{
								Type:       "object",
								Properties: map[string]*openapi.SchemaRef{"Id": {Schema: &openapi.Schema{Type: "string"}}, "Key": {Schema: &openapi.Schema{Type: "string"}}},
							},
						},
					},
				},
			},
			"409": {Description: "Object already exists", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"500": {Description: "Server error", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
		},
	})

	// GET /object/{bucketName}/{path} - Download object
	doc.AddPathOperation(basePath+"/object/{bucketName}/{path}", "get", &openapi.Operation{
		OperationID: "DownloadObject",
		Summary:     "Download an object",
		Description: "Downloads an object from storage. Supports range requests for partial content.",
		Tags:        []string{"Objects"},
		Parameters: []*openapi.Parameter{
			{Name: "bucketName", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Bucket name"},
			{Name: "path", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Object path"},
			{Name: "download", In: "query", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Trigger browser download"},
			{Name: "Range", In: "header", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Byte range (e.g. bytes=0-1023)"},
		},
		Responses: openapi.Responses{
			"200": {Description: "Object content", Content: map[string]*openapi.MediaType{"application/octet-stream": {}}},
			"206": {Description: "Partial content (range request)", Content: map[string]*openapi.MediaType{"application/octet-stream": {}}},
			"404": {Description: "Object not found", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"500": {Description: "Server error", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
		},
	})

	// POST /object/list/{bucketName} - List objects
	doc.AddPathOperation(basePath+"/object/list/{bucketName}", "post", &openapi.Operation{
		OperationID: "ListObjects",
		Summary:     "List objects in a bucket",
		Description: "Lists objects in a bucket with optional prefix filtering and pagination.",
		Tags:        []string{"Objects"},
		Parameters: []*openapi.Parameter{
			{Name: "bucketName", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Bucket name"},
		},
		RequestBody: &openapi.RequestBody{
			Description: "List options",
			Required:    true,
			Content: map[string]*openapi.MediaType{
				"application/json": {
					Schema: &openapi.SchemaRef{
						Schema: &openapi.Schema{
							Type: "object",
							Properties: map[string]*openapi.SchemaRef{
								"prefix": {Schema: &openapi.Schema{Type: "string", Description: "Prefix filter"}},
								"limit":  {Schema: &openapi.Schema{Type: "integer", Description: "Max results"}},
								"offset": {Schema: &openapi.Schema{Type: "integer", Description: "Skip count"}},
								"search": {Schema: &openapi.Schema{Type: "string", Description: "Search term"}},
							},
							Required: []string{"prefix"},
						},
					},
				},
			},
		},
		Responses: openapi.Responses{
			"200": {Description: "List of objects", Content: map[string]*openapi.MediaType{"application/json": {Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "array", Items: objectRef}}}}},
			"404": {Description: "Bucket not found", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"500": {Description: "Server error", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
		},
	})

	// POST /object/move - Move object
	doc.AddPathOperation(basePath+"/object/move", "post", &openapi.Operation{
		OperationID: "MoveObject",
		Summary:     "Move an object",
		Description: "Moves an object to a new location within the same or different bucket.",
		Tags:        []string{"Objects"},
		RequestBody: &openapi.RequestBody{
			Description: "Move parameters",
			Required:    true,
			Content: map[string]*openapi.MediaType{
				"application/json": {
					Schema: &openapi.SchemaRef{
						Schema: &openapi.Schema{
							Type: "object",
							Properties: map[string]*openapi.SchemaRef{
								"bucketId":          {Schema: &openapi.Schema{Type: "string"}},
								"sourceKey":         {Schema: &openapi.Schema{Type: "string"}},
								"destinationBucket": {Schema: &openapi.Schema{Type: "string"}},
								"destinationKey":    {Schema: &openapi.Schema{Type: "string"}},
							},
							Required: []string{"bucketId", "sourceKey", "destinationKey"},
						},
					},
				},
			},
		},
		Responses: openapi.Responses{
			"200": {Description: "Object moved", Content: map[string]*openapi.MediaType{"application/json": {Schema: messageRef}}},
			"404": {Description: "Object not found", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"500": {Description: "Server error", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
		},
	})

	// POST /object/copy - Copy object
	doc.AddPathOperation(basePath+"/object/copy", "post", &openapi.Operation{
		OperationID: "CopyObject",
		Summary:     "Copy an object",
		Description: "Creates a copy of an object in the same or different bucket.",
		Tags:        []string{"Objects"},
		RequestBody: &openapi.RequestBody{
			Description: "Copy parameters",
			Required:    true,
			Content: map[string]*openapi.MediaType{
				"application/json": {
					Schema: &openapi.SchemaRef{
						Schema: &openapi.Schema{
							Type: "object",
							Properties: map[string]*openapi.SchemaRef{
								"bucketId":          {Schema: &openapi.Schema{Type: "string"}},
								"sourceKey":         {Schema: &openapi.Schema{Type: "string"}},
								"destinationBucket": {Schema: &openapi.Schema{Type: "string"}},
								"destinationKey":    {Schema: &openapi.Schema{Type: "string"}},
								"metadata":          {Schema: &openapi.Schema{Type: "object"}},
							},
							Required: []string{"bucketId", "sourceKey", "destinationKey"},
						},
					},
				},
			},
		},
		Responses: openapi.Responses{
			"200": {
				Description: "Object copied",
				Content: map[string]*openapi.MediaType{
					"application/json": {
						Schema: &openapi.SchemaRef{
							Schema: &openapi.Schema{
								Type:       "object",
								Properties: map[string]*openapi.SchemaRef{"Id": {Schema: &openapi.Schema{Type: "string"}}, "Key": {Schema: &openapi.Schema{Type: "string"}}},
							},
						},
					},
				},
			},
			"404": {Description: "Source object not found", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"500": {Description: "Server error", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
		},
	})

	// POST /object/sign/{bucketName}/{path} - Create signed URL
	doc.AddPathOperation(basePath+"/object/sign/{bucketName}/{path}", "post", &openapi.Operation{
		OperationID: "CreateSignedURL",
		Summary:     "Create signed URL",
		Description: "Generates a temporary signed URL for secure object access without authentication.",
		Tags:        []string{"Objects"},
		Parameters: []*openapi.Parameter{
			{Name: "bucketName", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Bucket name"},
			{Name: "path", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Object path"},
		},
		RequestBody: &openapi.RequestBody{
			Description: "Signing options",
			Required:    true,
			Content: map[string]*openapi.MediaType{
				"application/json": {
					Schema: &openapi.SchemaRef{
						Schema: &openapi.Schema{
							Type:       "object",
							Properties: map[string]*openapi.SchemaRef{"expiresIn": {Schema: &openapi.Schema{Type: "integer", Description: "Expiry time in seconds"}}},
							Required:   []string{"expiresIn"},
						},
					},
				},
			},
		},
		Responses: openapi.Responses{
			"200": {
				Description: "Signed URL created",
				Content: map[string]*openapi.MediaType{
					"application/json": {
						Schema: &openapi.SchemaRef{
							Schema: &openapi.Schema{
								Type:       "object",
								Properties: map[string]*openapi.SchemaRef{"signedURL": {Schema: &openapi.Schema{Type: "string"}}},
							},
						},
					},
				},
			},
			"404": {Description: "Object not found", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"501": {Description: "Not implemented by storage driver", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
		},
	})

	// POST /object/upload/sign/{bucketName}/{path} - Create upload signed URL
	doc.AddPathOperation(basePath+"/object/upload/sign/{bucketName}/{path}", "post", &openapi.Operation{
		OperationID: "CreateUploadSignedURL",
		Summary:     "Create signed upload URL",
		Description: "Generates a temporary signed URL for uploading objects without authentication.",
		Tags:        []string{"Objects"},
		Parameters: []*openapi.Parameter{
			{Name: "bucketName", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Bucket name"},
			{Name: "path", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Object path"},
		},
		Responses: openapi.Responses{
			"200": {
				Description: "Signed upload URL created",
				Content: map[string]*openapi.MediaType{
					"application/json": {
						Schema: &openapi.SchemaRef{
							Schema: &openapi.Schema{
								Type:       "object",
								Properties: map[string]*openapi.SchemaRef{"url": {Schema: &openapi.Schema{Type: "string"}}, "token": {Schema: &openapi.Schema{Type: "string", Nullable: true}}},
							},
						},
					},
				},
			},
			"501": {Description: "Not implemented by storage driver", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
		},
	})

	// TUS Resumable Upload endpoints
	addTUSOperations(doc, basePath, errorRef)
}

// addTUSOperations adds TUS protocol resumable upload operations to the OpenAPI document.
func addTUSOperations(doc *openapi.Document, basePath string, errorRef *openapi.SchemaRef) {
	tusResumableHeader := &openapi.Parameter{
		Name:        "Tus-Resumable",
		In:          "header",
		Required:    true,
		Schema:      &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string", Enum: []any{"1.0.0"}}},
		Description: "TUS protocol version (must be 1.0.0)",
	}

	// OPTIONS /upload/resumable/ - TUS capabilities discovery
	doc.AddPathOperation(basePath+"/upload/resumable/", "options", &openapi.Operation{
		OperationID: "TUSOptions",
		Summary:     "TUS capabilities discovery",
		Description: "Returns TUS protocol capabilities including supported extensions and maximum upload size.",
		Tags:        []string{"Upload"},
		Responses: openapi.Responses{
			"200": {
				Description: "TUS server capabilities",
				Headers: map[string]*openapi.Header{
					"Tus-Resumable": {Description: "TUS protocol version", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}},
					"Tus-Version":   {Description: "Supported TUS versions", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}},
					"Tus-Extension": {Description: "Supported TUS extensions", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}},
					"Tus-Max-Size":  {Description: "Maximum upload size in bytes", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}},
				},
			},
		},
	})

	// POST /upload/resumable/{bucketName}/{path} - Create resumable upload
	doc.AddPathOperation(basePath+"/upload/resumable/{bucketName}/{path}", "post", &openapi.Operation{
		OperationID: "TUSCreateUpload",
		Summary:     "Create resumable upload",
		Description: "Initiates a new TUS resumable upload. Returns a Location header with the upload URL.",
		Tags:        []string{"Upload"},
		Parameters: []*openapi.Parameter{
			{Name: "bucketName", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Bucket name"},
			{Name: "path", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Object path"},
			tusResumableHeader,
			{Name: "Upload-Length", In: "header", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "integer", Format: "int64"}}, Description: "Total size of the upload in bytes"},
			{Name: "Upload-Defer-Length", In: "header", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string", Enum: []any{"1"}}}, Description: "Defer length specification"},
			{Name: "Upload-Metadata", In: "header", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Upload metadata (key base64value pairs, comma separated)"},
			{Name: "x-upsert", In: "header", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string", Enum: []any{"true", "false"}}}, Description: "Allow overwriting existing objects"},
		},
		Responses: openapi.Responses{
			"201": {
				Description: "Upload created",
				Headers: map[string]*openapi.Header{
					"Location":       {Description: "URL for uploading chunks", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}},
					"Tus-Resumable":  {Description: "TUS protocol version", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}},
					"Upload-Offset":  {Description: "Current upload offset (0 for new uploads)", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}},
					"Upload-Expires": {Description: "Upload expiration time", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}},
				},
			},
			"400": {Description: "Invalid request", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"404": {Description: "Bucket not found", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"409": {Description: "Object already exists", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"412": {Description: "Unsupported TUS version", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"413": {Description: "Upload exceeds maximum size", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
		},
	})

	// PATCH /upload/resumable/{bucketName}/{path} - Upload chunk
	doc.AddPathOperation(basePath+"/upload/resumable/{bucketName}/{path}", "patch", &openapi.Operation{
		OperationID: "TUSPatchUpload",
		Summary:     "Upload chunk",
		Description: "Uploads a chunk of data to an existing resumable upload. The chunk is appended at the current offset.",
		Tags:        []string{"Upload"},
		Parameters: []*openapi.Parameter{
			{Name: "bucketName", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Bucket name"},
			{Name: "path", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Object path"},
			tusResumableHeader,
			{Name: "Upload-Offset", In: "header", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "integer", Format: "int64"}}, Description: "Current offset to write at"},
			{Name: "Content-Type", In: "header", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string", Enum: []any{"application/offset+octet-stream"}}}, Description: "Must be application/offset+octet-stream"},
		},
		RequestBody: &openapi.RequestBody{
			Description: "Chunk data",
			Required:    true,
			Content:     map[string]*openapi.MediaType{"application/offset+octet-stream": {}},
		},
		Responses: openapi.Responses{
			"204": {
				Description: "Chunk uploaded successfully",
				Headers: map[string]*openapi.Header{
					"Tus-Resumable": {Description: "TUS protocol version", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}},
					"Upload-Offset": {Description: "New upload offset after this chunk", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}},
				},
			},
			"400": {Description: "Invalid request", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"404": {Description: "Upload not found", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"409": {Description: "Offset mismatch", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
			"415": {Description: "Unsupported media type", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
		},
	})

	// HEAD /upload/resumable/{bucketName}/{path} - Get upload status
	doc.AddPathOperation(basePath+"/upload/resumable/{bucketName}/{path}", "head", &openapi.Operation{
		OperationID: "TUSHeadUpload",
		Summary:     "Get upload status",
		Description: "Returns the current status of a resumable upload including the current offset.",
		Tags:        []string{"Upload"},
		Parameters: []*openapi.Parameter{
			{Name: "bucketName", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Bucket name"},
			{Name: "path", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Object path"},
			tusResumableHeader,
		},
		Responses: openapi.Responses{
			"200": {
				Description: "Upload status",
				Headers: map[string]*openapi.Header{
					"Tus-Resumable": {Description: "TUS protocol version", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}},
					"Upload-Offset": {Description: "Current upload offset", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}},
					"Upload-Length": {Description: "Total upload length", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}},
					"Cache-Control": {Description: "Cache control directive", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}},
				},
			},
			"404": {Description: "Upload not found", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
		},
	})

	// DELETE /upload/resumable/{bucketName}/{path} - Cancel upload
	doc.AddPathOperation(basePath+"/upload/resumable/{bucketName}/{path}", "delete", &openapi.Operation{
		OperationID: "TUSDeleteUpload",
		Summary:     "Cancel upload",
		Description: "Cancels a resumable upload and cleans up any uploaded data.",
		Tags:        []string{"Upload"},
		Parameters: []*openapi.Parameter{
			{Name: "bucketName", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Bucket name"},
			{Name: "path", In: "path", Required: true, Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}, Description: "Object path"},
			tusResumableHeader,
		},
		Responses: openapi.Responses{
			"204": {
				Description: "Upload cancelled",
				Headers: map[string]*openapi.Header{
					"Tus-Resumable": {Description: "TUS protocol version", Schema: &openapi.SchemaRef{Schema: &openapi.Schema{Type: "string"}}},
				},
			},
			"404": {Description: "Upload not found", Content: map[string]*openapi.MediaType{"application/json": {Schema: errorRef}}},
		},
	})
}
