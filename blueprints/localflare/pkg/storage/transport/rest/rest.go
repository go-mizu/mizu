// File: lib/storage/transport/rest/rest.go
package rest

import (
	"errors"
	"net/http"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
	"github.com/go-mizu/mizu"
)

// Server exposes Supabase Storage compatible REST endpoints on top of storage.Storage.
//
// Bucket endpoints:
//
//	POST   /bucket              - create bucket
//	GET    /bucket              - list buckets
//	GET    /bucket/:bucketId    - get bucket details
//	PUT    /bucket/:bucketId    - update bucket
//	DELETE /bucket/:bucketId    - delete bucket
//	POST   /bucket/:bucketId/empty - empty bucket
//
// Object endpoints:
//
//	POST   /object/:bucketName/*path       - upload object
//	GET    /object/:bucketName/*path       - download object
//	PUT    /object/:bucketName/*path       - update object
//	DELETE /object/:bucketName/*path       - delete object
//	DELETE /object/:bucketName             - delete multiple objects
//	POST   /object/list/:bucketName        - list objects in bucket
//	POST   /object/move                    - move object
//	POST   /object/copy                    - copy object
//	GET    /object/public/:bucketName/*path    - get public object
//	GET    /object/info/:bucketName/*path      - get object info
//	POST   /object/sign/:bucketName/*path      - create signed URL
//	POST   /object/sign/:bucketName            - create multiple signed URLs
//	POST   /object/upload/sign/:bucketName/*path - create signed upload URL
//
// TUS resumable upload endpoints (TUS protocol 1.0.0):
//
//	OPTIONS /upload/resumable/                       - TUS capabilities discovery
//	OPTIONS /upload/resumable/:bucketName/*path      - TUS capabilities discovery
//	POST    /upload/resumable/:bucketName/*path      - create resumable upload
//	PATCH   /upload/resumable/:bucketName/*path      - upload chunk
//	HEAD    /upload/resumable/:bucketName/*path      - get upload status
//	DELETE  /upload/resumable/:bucketName/*path      - cancel upload
type Server struct {
	store      storage.Storage
	authConfig *AuthConfig
}

// New creates a new Server.
func New(store storage.Storage) *Server {
	return &Server{
		store:      store,
		authConfig: nil, // No auth by default
	}
}

// NewWithAuth creates a new Server with JWT authentication enabled.
func NewWithAuth(store storage.Storage, authConfig AuthConfig) *Server {
	return &Server{
		store:      store,
		authConfig: &authConfig,
	}
}

// Register wires Storage REST routes into a Mizu app under the given base path.
//
// Example:
//
//	store, _ := local.Open(ctx, "/data")
//	rest.Register(app, "/storage/v1", store)
//
// The basePath must not have a trailing slash.
func Register(app *mizu.App, basePath string, store storage.Storage) {
	s := New(store)
	registerRoutes(app, basePath, s)
}

// RegisterWithAuth wires Storage REST routes with JWT authentication.
//
// Example:
//
//	store, _ := local.Open(ctx, "/data")
//	authConfig := rest.AuthConfig{
//		JWTSecret: "your-jwt-secret",
//		AllowAnonymousPublic: true,
//	}
//	rest.RegisterWithAuth(app, "/storage/v1", store, authConfig)
//
// The basePath must not have a trailing slash.
func RegisterWithAuth(app *mizu.App, basePath string, store storage.Storage, authConfig AuthConfig) {
	s := NewWithAuth(store, authConfig)
	registerRoutes(app, basePath, s)
}

// registerRoutes registers all storage routes with optional authentication.
func registerRoutes(app *mizu.App, basePath string, s *Server) {
	// Helper to optionally apply auth middleware
	withAuth := func(handler mizu.Handler) mizu.Handler {
		if s.authConfig != nil {
			return AuthMiddleware(*s.authConfig)(handler)
		}
		return handler
	}

	// Helper for endpoints that always require auth (bucket management)
	requireAuth := func(handler mizu.Handler) mizu.Handler {
		if s.authConfig != nil {
			return RequireAuth(*s.authConfig)(handler)
		}
		return handler
	}

	// Bucket endpoints - always require authentication
	app.Post(basePath+"/bucket", requireAuth(s.handleCreateBucket))
	app.Get(basePath+"/bucket", requireAuth(s.handleListBuckets))
	app.Get(basePath+"/bucket/{bucketId}", requireAuth(s.handleGetBucket))
	app.Put(basePath+"/bucket/{bucketId}", requireAuth(s.handleUpdateBucket))
	app.Delete(basePath+"/bucket/{bucketId}", requireAuth(s.handleDeleteBucket))
	app.Post(basePath+"/bucket/{bucketId}/empty", requireAuth(s.handleEmptyBucket))

	// Object endpoints - support public bucket access
	app.Post(basePath+"/object/list/{bucketName}", withAuth(s.handleListObjects))
	app.Post(basePath+"/object/move", withAuth(s.handleMoveObject))
	app.Post(basePath+"/object/copy", withAuth(s.handleCopyObject))
	app.Post(basePath+"/object/sign/{bucketName}/{path...}", withAuth(s.handleCreateSignedURL))
	app.Post(basePath+"/object/sign/{bucketName}", withAuth(s.handleCreateSignedURLs))
	app.Post(basePath+"/object/upload/sign/{bucketName}/{path...}", withAuth(s.handleCreateUploadSignedURL))

	// Object CRUD with wildcard path
	app.Post(basePath+"/object/{bucketName}/{path...}", withAuth(s.handleUploadObject))
	app.Get(basePath+"/object/{bucketName}/{path...}", withAuth(s.handleDownloadObject))
	app.Put(basePath+"/object/{bucketName}/{path...}", withAuth(s.handleUpdateObject))
	app.Delete(basePath+"/object/{bucketName}/{path...}", withAuth(s.handleDeleteObject))
	app.Delete(basePath+"/object/{bucketName}", withAuth(s.handleDeleteObjects))

	// Public endpoints - no auth required
	app.Get(basePath+"/object/public/{bucketName}/{path...}", s.handlePublicObject)
	app.Get(basePath+"/object/info/public/{bucketName}/{path...}", s.handlePublicObjectInfo)

	// Authenticated endpoints - always require auth
	app.Get(basePath+"/object/authenticated/{bucketName}/{path...}", requireAuth(s.handleAuthenticatedObject))
	app.Get(basePath+"/object/info/authenticated/{bucketName}/{path...}", requireAuth(s.handleAuthenticatedObjectInfo))

	// Info endpoint - support public bucket access
	app.Get(basePath+"/object/info/{bucketName}/{path...}", withAuth(s.handleObjectInfo))

	// TUS resumable upload endpoints
	app.Handle("OPTIONS", basePath+"/upload/resumable/", s.handleTUSOptions)
	app.Handle("OPTIONS", basePath+"/upload/resumable/{bucketName}/{path...}", s.handleTUSOptions)
	app.Post(basePath+"/upload/resumable/{bucketName}/{path...}", withAuth(s.handleTUSCreate))
	app.Patch(basePath+"/upload/resumable/{bucketName}/{path...}", withAuth(s.handleTUSPatch))
	app.Head(basePath+"/upload/resumable/{bucketName}/{path...}", withAuth(s.handleTUSHead))
	app.Delete(basePath+"/upload/resumable/{bucketName}/{path...}", withAuth(s.handleTUSDelete))
}

// errorPayload matches the Supabase Storage API error response shape:
//
//	{
//	  "statusCode": "400",
//	  "error": "Bad Request",
//	  "message": "..."
//	}
type errorPayload struct {
	StatusCode int    `json:"statusCode"`
	Error      string `json:"error"`
	Message    string `json:"message"`
}

// writeError writes a JSON error response using the Supabase Storage compatible shape.
func writeError(c *mizu.Ctx, status int, err error) error {
	if err == nil {
		status = http.StatusInternalServerError
		err = errors.New("unknown error")
	}

	body := errorPayload{
		StatusCode: status,
		Error:      errorNameFromStatus(status),
		Message:    err.Error(),
	}

	return c.JSON(status, body)
}

// errorNameFromStatus maps HTTP status codes to error names.
func errorNameFromStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "Bad Request"
	case http.StatusUnauthorized:
		return "Unauthorized"
	case http.StatusForbidden:
		return "Forbidden"
	case http.StatusNotFound:
		return "Not Found"
	case http.StatusConflict:
		return "Conflict"
	case http.StatusUnprocessableEntity:
		return "Unprocessable Entity"
	case http.StatusNotImplemented:
		return "Not Implemented"
	case http.StatusInternalServerError:
		return "Internal Server Error"
	default:
		return "Error"
	}
}

// mapStorageError maps storage package errors to appropriate HTTP status codes.
func mapStorageError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	switch {
	case errors.Is(err, storage.ErrNotExist):
		return http.StatusNotFound
	case errors.Is(err, storage.ErrExist):
		return http.StatusConflict
	case errors.Is(err, storage.ErrPermission):
		return http.StatusForbidden
	case errors.Is(err, storage.ErrUnsupported):
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}
