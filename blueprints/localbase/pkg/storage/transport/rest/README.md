# Supabase-Compatible Storage REST API

This package provides a Supabase-compatible Storage REST API transport layer for the Open-Lake storage abstraction.

## Overview

The REST transport implements the [Supabase Storage API](https://supabase.com/docs/reference/storage) specification, allowing clients built for Supabase Storage to work seamlessly with Open-Lake storage backends.

## Features

- **Supabase Storage API Compatible**: Full compatibility with Supabase Storage client SDKs
- **JWT Authentication**: Secure API access with JWT tokens
- **Multiple Storage Backends**: Works with any Open-Lake storage driver (local, S3, Azure, etc.)
- **TUS Resumable Uploads**: Support for chunked, resumable uploads following TUS 1.0.0 protocol
- **OpenAPI Documentation**: Auto-generated API documentation

## API Endpoints

### Bucket Operations

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/bucket` | Create a new bucket |
| GET | `/bucket` | List all buckets |
| GET | `/bucket/{bucketId}` | Get bucket details |
| PUT | `/bucket/{bucketId}` | Update bucket properties |
| DELETE | `/bucket/{bucketId}` | Delete a bucket |
| POST | `/bucket/{bucketId}/empty` | Empty a bucket |

### Object Operations

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/object/{bucketName}/{path}` | Upload an object |
| GET | `/object/{bucketName}/{path}` | Download an object |
| PUT | `/object/{bucketName}/{path}` | Update an object |
| DELETE | `/object/{bucketName}/{path}` | Delete an object |
| DELETE | `/object/{bucketName}` | Delete multiple objects |
| POST | `/object/list/{bucketName}` | List objects in a bucket |
| POST | `/object/move` | Move an object |
| POST | `/object/copy` | Copy an object |
| POST | `/object/sign/{bucketName}/{path}` | Create signed URL |
| POST | `/object/sign/{bucketName}` | Create multiple signed URLs |
| POST | `/object/upload/sign/{bucketName}/{path}` | Create signed upload URL |
| GET | `/object/public/{bucketName}/{path}` | Download public object |
| GET | `/object/authenticated/{bucketName}/{path}` | Download authenticated object |
| GET | `/object/info/{bucketName}/{path}` | Get object info |

### TUS Resumable Upload Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| OPTIONS | `/upload/resumable/` | TUS capabilities discovery |
| POST | `/upload/resumable/{bucketName}/{path}` | Create resumable upload |
| PATCH | `/upload/resumable/{bucketName}/{path}` | Upload chunk |
| HEAD | `/upload/resumable/{bucketName}/{path}` | Get upload status |
| DELETE | `/upload/resumable/{bucketName}/{path}` | Cancel upload |

## Usage

### Basic Registration

```go
import (
    "github.com/go-mizu/mizu"
    "open-lake.dev/lib/storage"
    "open-lake.dev/lib/storage/transport/rest"
)

// Open storage
store, _ := storage.Open(ctx, "local:///var/data")

// Create web server
app := mizu.New()

// Register REST API without authentication
rest.Register(app, "/storage/v1", store)
```

### With JWT Authentication

```go
// Configure authentication
authConfig := rest.AuthConfig{
    JWTSecret:            "your-jwt-secret-minimum-32-characters",
    AllowAnonymousPublic: true, // Allow unauthenticated access to public buckets
}

// Register REST API with authentication
rest.RegisterWithAuth(app, "/storage/v1", store, authConfig)

// Optionally add API documentation
rest.RegisterDocs(app, "/storage/v1", "/docs", "Storage API", "1.0.0")
```

## Authentication

The API supports JWT-based authentication compatible with Supabase Auth tokens.

### Token Format

Tokens should be provided in the `Authorization` header:

```
Authorization: Bearer <jwt-token>
```

Or via the `apikey` header (for Supabase client compatibility):

```
apikey: <jwt-token>
```

### JWT Claims

The following JWT claims are recognized:

| Claim | Description |
|-------|-------------|
| `sub` | User ID |
| `role` | User role (e.g., `authenticated`, `service_role`) |
| `aud` | Audience |
| `iss` | Issuer |
| `exp` | Expiration timestamp |
| `iat` | Issued at timestamp |

### Roles

- `authenticated`: Standard authenticated user
- `service_role`: Service account with elevated privileges
- `anon`: Anonymous/unauthenticated user

## Error Responses

All errors follow the Supabase Storage API format:

```json
{
    "statusCode": 404,
    "error": "Not Found",
    "message": "bucket not found"
}
```

## TUS Resumable Uploads

The API supports the [TUS 1.0.0 protocol](https://tus.io/protocols/resumable-upload.html) for resumable uploads.

### Supported Extensions

- `creation`: Create new uploads
- `termination`: Cancel uploads

### Example TUS Upload Flow

```bash
# 1. Create upload
curl -X POST "http://localhost:9000/storage/v1/upload/resumable/bucket/file.txt" \
    -H "Tus-Resumable: 1.0.0" \
    -H "Upload-Length: 1024" \
    -H "Upload-Metadata: contentType YXBwbGljYXRpb24vb2N0ZXQtc3RyZWFt"

# 2. Upload chunks
curl -X PATCH "http://localhost:9000/storage/v1/upload/resumable/bucket/file.txt" \
    -H "Tus-Resumable: 1.0.0" \
    -H "Upload-Offset: 0" \
    -H "Content-Type: application/offset+octet-stream" \
    --data-binary @chunk1.bin

# 3. Check status
curl -I "http://localhost:9000/storage/v1/upload/resumable/bucket/file.txt" \
    -H "Tus-Resumable: 1.0.0"
```

## API Documentation

When `RegisterDocs` is called, the following documentation endpoints are available:

- `GET /docs` - OpenAPI JSON specification
- `GET /docs/` - Interactive documentation UI (Scalar by default)

UI options via query parameter: `?ui=scalar|redoc|swagger|rapidoc|stoplight`

## Configuration

### AuthConfig Options

| Field | Type | Description |
|-------|------|-------------|
| `JWTSecret` | string | Secret key for JWT verification (min 32 chars recommended) |
| `AllowAnonymousPublic` | bool | Allow unauthenticated access to public bucket objects |

## Testing

Run the tests with:

```bash
go test ./lib/storage/transport/rest/...
```

Tests are organized into:

- `auth_test.go` - Authentication and authorization tests
- `handle_bucket_test.go` - Bucket CRUD operations
- `handle_object_test.go` - Object CRUD operations
- `handle_upload_test.go` - TUS resumable upload tests
- `jwt_test.go` - JWT token creation and verification
- `rest_test.go` - Integration tests

## License

See the main project LICENSE file.
