# WebDAV Transport Layer Specification

## Overview

The WebDAV transport layer exposes `storage.Storage` backends over the WebDAV protocol (RFC 4918). This enables any WebDAV-compatible client (file managers, IDEs, sync tools) to interact with the underlying storage as if it were a remote filesystem.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     WebDAV Clients                          │
│  (Finder, Windows Explorer, Cyberduck, rclone, etc.)       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   HTTP/HTTPS Server                         │
│                  (net/http or mizu)                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              WebDAV Transport Layer                         │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                    Server                            │   │
│  │  - Config (auth, options)                           │   │
│  │  - Handler (golang.org/x/net/webdav)               │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              StorageFileSystem                       │   │
│  │  - Implements webdav.FileSystem                     │   │
│  │  - Maps WebDAV paths to bucket/key                  │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                StorageFile                           │   │
│  │  - Implements webdav.File                           │   │
│  │  - Read/Write operations via storage.Bucket        │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   storage.Storage                           │
│  (local, s3, azureblob, sftp, memory, etc.)               │
└─────────────────────────────────────────────────────────────┘
```

## Path Mapping

WebDAV paths are mapped to storage buckets and objects:

| WebDAV Path | Storage Mapping |
|-------------|-----------------|
| `/` | List all buckets |
| `/<bucket>/` | Bucket root (list objects) |
| `/<bucket>/<key>` | Object within bucket |
| `/<bucket>/dir/` | Virtual directory (prefix) |
| `/<bucket>/dir/file.txt` | Object with key `dir/file.txt` |

### Single-Bucket Mode

When configured with a specific bucket, the path mapping changes:

| WebDAV Path | Storage Mapping |
|-------------|-----------------|
| `/` | Bucket root |
| `/<key>` | Object in bucket |
| `/dir/` | Virtual directory |
| `/dir/file.txt` | Object with key `dir/file.txt` |

## Supported WebDAV Methods

### RFC 4918 Required Methods

| Method | Description | Implementation |
|--------|-------------|----------------|
| `OPTIONS` | Discover supported methods | Returns WebDAV capability headers |
| `GET` | Download file | `bucket.Open()` |
| `HEAD` | Get file metadata | `bucket.Stat()` |
| `PUT` | Upload/update file | `bucket.Write()` |
| `DELETE` | Delete file/directory | `bucket.Delete()` with recursive option |
| `MKCOL` | Create collection (directory) | `storage.CreateBucket()` or no-op for virtual dirs |
| `COPY` | Copy resource | `bucket.Copy()` |
| `MOVE` | Move/rename resource | `bucket.Move()` |
| `PROPFIND` | Get properties | Returns file metadata |
| `PROPPATCH` | Set properties | Stored as object metadata if supported |

### Locking (RFC 4918)

| Method | Description | Implementation |
|--------|-------------|----------------|
| `LOCK` | Create lock | In-memory or persistent lock system |
| `UNLOCK` | Release lock | In-memory or persistent lock system |

## Configuration

```go
// Config controls WebDAV server behavior.
type Config struct {
    // Prefix is the URL path prefix (e.g., "/webdav").
    // Stripped from incoming paths before mapping to storage.
    Prefix string

    // Bucket restricts access to a single bucket.
    // If empty, all buckets are accessible.
    Bucket string

    // ReadOnly disables all write operations.
    ReadOnly bool

    // Auth configures authentication.
    Auth AuthConfig

    // LockSystem manages WebDAV locks.
    // If nil, an in-memory lock system is used.
    LockSystem webdav.LockSystem

    // Logger for request/error logging.
    Logger *slog.Logger

    // HideDotFiles hides files starting with '.' in listings.
    HideDotFiles bool

    // DefaultContentType for files without extension.
    DefaultContentType string

    // MaxUploadSize limits upload size (0 = unlimited).
    MaxUploadSize int64

    // DeadPropsStore enables storage of custom properties.
    // If nil, PROPPATCH operations are not supported.
    DeadPropsStore DeadPropsStore
}

// AuthConfig configures authentication.
type AuthConfig struct {
    // Type is the authentication method.
    // Supported: "none", "basic", "digest", "jwt"
    Type string

    // Realm for Basic/Digest authentication.
    Realm string

    // BasicAuth validates username/password for Basic auth.
    BasicAuth func(username, password string) bool

    // DigestAuth provides password/hash for Digest auth.
    DigestAuth func(username string) (password string, ok bool)

    // JWTSecret for JWT token validation.
    JWTSecret string

    // JWTClaims extracts claims from validated JWT.
    JWTClaims func(claims map[string]any) (*User, error)
}

// User represents an authenticated user.
type User struct {
    ID       string
    Username string
    Roles    []string
    Metadata map[string]string
}
```

## Properties

### Live Properties (Read-Only)

Standard WebDAV live properties automatically computed:

| Property | Description | Source |
|----------|-------------|--------|
| `DAV:creationdate` | Creation time | `Object.Created` |
| `DAV:getlastmodified` | Last modification time | `Object.Updated` |
| `DAV:getetag` | Entity tag | `Object.ETag` |
| `DAV:getcontentlength` | Content size | `Object.Size` |
| `DAV:getcontenttype` | MIME type | `Object.ContentType` |
| `DAV:resourcetype` | Resource type (collection/file) | `Object.IsDir` |
| `DAV:displayname` | Display name | Filename from key |
| `DAV:supportedlock` | Supported lock types | Lock configuration |
| `DAV:lockdiscovery` | Active locks | Lock system |

### Dead Properties

Custom properties can be stored if `DeadPropsStore` is configured:

```go
// DeadPropsStore persists custom WebDAV properties.
type DeadPropsStore interface {
    // Get retrieves properties for a resource.
    Get(ctx context.Context, path string) (map[xml.Name]Property, error)

    // Set stores properties for a resource.
    Set(ctx context.Context, path string, props []Property) error

    // Remove deletes properties for a resource.
    Remove(ctx context.Context, path string, names []xml.Name) error

    // Delete removes all properties when resource is deleted.
    Delete(ctx context.Context, path string) error
}
```

## API

### Server

```go
// New creates a WebDAV server backed by storage.Storage.
func New(store storage.Storage, cfg *Config) *Server

// Server handles WebDAV requests.
type Server struct {
    // contains filtered or unexported fields
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request)

// Handler returns the underlying webdav.Handler for advanced configuration.
func (s *Server) Handler() *webdav.Handler
```

### Registration with Mizu

```go
// Register mounts the WebDAV server under basePath using mizu.
func Register(app *mizu.App, basePath string, store storage.Storage, cfg *Config) *Server

// Example:
//   store, _ := storage.Open(ctx, "local:///data")
//   cfg := &webdav.Config{
//       Auth: webdav.AuthConfig{
//           Type: "basic",
//           Realm: "WebDAV",
//           BasicAuth: func(u, p string) bool { return u == "admin" && p == "secret" },
//       },
//   }
//   webdav.Register(app, "/dav", store, cfg)
```

### Standalone HTTP Server

```go
// ListenAndServe starts a standalone WebDAV server.
func (s *Server) ListenAndServe(addr string) error

// Example:
//   store, _ := storage.Open(ctx, "s3://my-bucket")
//   server := webdav.New(store, &webdav.Config{Bucket: "my-bucket"})
//   server.ListenAndServe(":8080")
```

## Implementation Details

### StorageFileSystem

Implements `webdav.FileSystem`:

```go
type StorageFileSystem struct {
    store    storage.Storage
    bucket   string // optional single-bucket mode
    readOnly bool
}

func (fs *StorageFileSystem) Mkdir(ctx context.Context, name string, perm os.FileMode) error
func (fs *StorageFileSystem) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error)
func (fs *StorageFileSystem) RemoveAll(ctx context.Context, name string) error
func (fs *StorageFileSystem) Rename(ctx context.Context, oldName, newName string) error
func (fs *StorageFileSystem) Stat(ctx context.Context, name string) (os.FileInfo, error)
```

#### Path Resolution

```go
// parsePath converts WebDAV path to bucket and key.
// Returns (bucket, key, error).
//
// Examples (multi-bucket mode):
//   "/"           -> ("", "", nil)           // root listing
//   "/bucket"     -> ("bucket", "", nil)     // bucket listing
//   "/bucket/key" -> ("bucket", "key", nil)  // object
//
// Examples (single-bucket mode, bucket="data"):
//   "/"         -> ("data", "", nil)         // bucket listing
//   "/key"      -> ("data", "key", nil)      // object
//   "/dir/file" -> ("data", "dir/file", nil) // nested object
func (fs *StorageFileSystem) parsePath(name string) (bucket, key string, err error)
```

### StorageFile

Implements `webdav.File` (which extends `http.File` and `io.Writer`):

```go
type StorageFile struct {
    fs          *StorageFileSystem
    bucket      string
    key         string
    isDir       bool

    // For reading
    reader      io.ReadCloser
    offset      int64

    // For writing
    buffer      *bytes.Buffer
    tempFile    *os.File
    contentType string
}

// http.File interface
func (f *StorageFile) Read(p []byte) (n int, err error)
func (f *StorageFile) Seek(offset int64, whence int) (int64, error)
func (f *StorageFile) Readdir(count int) ([]os.FileInfo, error)
func (f *StorageFile) Stat() (os.FileInfo, error)
func (f *StorageFile) Close() error

// io.Writer interface (for PUT operations)
func (f *StorageFile) Write(p []byte) (n int, err error)
```

### StorageFileInfo

Implements `os.FileInfo`:

```go
type StorageFileInfo struct {
    name    string
    size    int64
    mode    os.FileMode
    modTime time.Time
    isDir   bool

    // For optional interfaces
    contentType string
    etag        string
}

func (fi *StorageFileInfo) Name() string
func (fi *StorageFileInfo) Size() int64
func (fi *StorageFileInfo) Mode() os.FileMode
func (fi *StorageFileInfo) ModTime() time.Time
func (fi *StorageFileInfo) IsDir() bool
func (fi *StorageFileInfo) Sys() interface{}

// Optional: ContentTyper interface
func (fi *StorageFileInfo) ContentType(ctx context.Context) (string, error)

// Optional: ETager interface
func (fi *StorageFileInfo) ETag(ctx context.Context) (string, error)
```

## Error Handling

Storage errors are mapped to appropriate WebDAV/HTTP status codes:

| Storage Error | HTTP Status | WebDAV Status |
|---------------|-------------|---------------|
| `ErrNotExist` | 404 Not Found | - |
| `ErrExist` | 409 Conflict | - |
| `ErrPermission` | 403 Forbidden | - |
| `ErrUnsupported` | 501 Not Implemented | - |
| Lock conflict | 423 Locked | `DAV:lock-token-submitted` |
| Insufficient storage | 507 Insufficient Storage | - |

## Authentication

### Basic Authentication

```go
cfg := &webdav.Config{
    Auth: webdav.AuthConfig{
        Type:  "basic",
        Realm: "WebDAV Server",
        BasicAuth: func(username, password string) bool {
            // Validate credentials
            return username == "user" && password == "pass"
        },
    },
}
```

### JWT Authentication

```go
cfg := &webdav.Config{
    Auth: webdav.AuthConfig{
        Type:      "jwt",
        JWTSecret: "your-secret-key",
        JWTClaims: func(claims map[string]any) (*webdav.User, error) {
            // Extract user from claims
            return &webdav.User{
                ID:       claims["sub"].(string),
                Username: claims["name"].(string),
            }, nil
        },
    },
}
```

## Limitations

### Virtual Directories

Object storage systems (S3, Azure Blob) don't have true directories. The WebDAV transport handles this by:

1. **MKCOL on bucket root**: Creates a real bucket via `storage.CreateBucket()`
2. **MKCOL on path within bucket**: No-op (directories are virtual)
3. **DELETE on directory**: Recursively deletes all objects with the prefix
4. **Listing**: Groups objects by common prefix to show virtual directories

### Large File Handling

For uploads:
- Small files (<`WriteBufferSize`): Buffered in memory
- Large files (>`WriteBufferSize`): Spooled to temp file

For downloads:
- Streamed directly from storage without buffering

### Locking

Default in-memory lock system limitations:
- Locks are lost on server restart
- Not distributed (single server only)

For production with multiple servers, implement a custom `LockSystem`:
- Redis-based
- Database-backed
- etcd-based

## Usage Examples

### Basic Usage

```go
package main

import (
    "log"
    "net/http"

    "open-lake.dev/lib/storage"
    "open-lake.dev/lib/storage/transport/webdav"
)

func main() {
    ctx := context.Background()

    // Open storage backend
    store, err := storage.Open(ctx, "local:///var/data")
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()

    // Create WebDAV server
    server := webdav.New(store, &webdav.Config{
        Prefix: "/webdav",
    })

    // Start HTTP server
    http.Handle("/webdav/", server)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### With Mizu Framework

```go
package main

import (
    "github.com/go-mizu/mizu"
    "open-lake.dev/lib/storage"
    "open-lake.dev/lib/storage/transport/webdav"
)

func main() {
    ctx := context.Background()

    store, _ := storage.Open(ctx, "s3://my-bucket?region=us-east-1")
    defer store.Close()

    app := mizu.New()

    // Mount WebDAV with authentication
    webdav.Register(app, "/dav", store, &webdav.Config{
        Auth: webdav.AuthConfig{
            Type:  "basic",
            Realm: "Storage",
            BasicAuth: validateUser,
        },
    })

    app.Listen(":8080")
}
```

### Single Bucket Mode

```go
// Expose only one bucket at the WebDAV root
server := webdav.New(store, &webdav.Config{
    Bucket: "public-files",
    ReadOnly: true, // Optional: make read-only
})
```

### With Custom Properties Store

```go
// Store custom properties in a database
propsStore := &DatabasePropsStore{db: db}

server := webdav.New(store, &webdav.Config{
    DeadPropsStore: propsStore,
})
```

## Testing with WebDAV Clients

### macOS Finder
```
Finder > Go > Connect to Server
URL: http://localhost:8080/webdav
```

### Windows Explorer
```
Map Network Drive
Folder: \\localhost@8080\webdav
```

### curl
```bash
# List directory
curl -X PROPFIND http://localhost:8080/webdav/

# Upload file
curl -T file.txt http://localhost:8080/webdav/bucket/file.txt

# Download file
curl http://localhost:8080/webdav/bucket/file.txt

# Delete file
curl -X DELETE http://localhost:8080/webdav/bucket/file.txt

# Create directory
curl -X MKCOL http://localhost:8080/webdav/new-bucket/
```

### cadaver (CLI WebDAV client)
```bash
cadaver http://localhost:8080/webdav/
> ls
> put file.txt
> get file.txt
> mkcol newdir
> delete file.txt
```

## Compliance

This implementation targets WebDAV Class 2 compliance (RFC 4918):

- [x] Class 1: Basic WebDAV methods (GET, PUT, DELETE, MKCOL, COPY, MOVE, PROPFIND, PROPPATCH)
- [x] Class 2: Locking support (LOCK, UNLOCK)
- [ ] Class 3: Not implemented (advanced features)

### Tested Clients

- macOS Finder
- Windows Explorer
- Cyberduck
- WinSCP
- rclone
- cadaver
- curl

## References

- [RFC 4918 - HTTP Extensions for Web Distributed Authoring and Versioning (WebDAV)](https://tools.ietf.org/html/rfc4918)
- [RFC 2518 - HTTP Extensions for Distributed Authoring (Original WebDAV)](https://tools.ietf.org/html/rfc2518)
- [golang.org/x/net/webdav](https://pkg.go.dev/golang.org/x/net/webdav)
