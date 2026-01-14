# SFTP Transport Layer Specification

## Overview

The SFTP transport layer exposes `storage.Storage` backends over the SSH File Transfer Protocol (SFTP), enabling standard SFTP clients (sftp, FileZilla, WinSCP, Cyberduck, etc.) to access any storage backend supported by the Open-Lake storage abstraction.

This transport acts as an SFTP server that translates SFTP protocol operations into the corresponding `storage.Storage` API calls.

## Architecture

```
┌────────────────────────────────────────────────────────────┐
│                      SFTP Clients                          │
│  (sftp CLI, FileZilla, WinSCP, Cyberduck, sshfs, etc.)    │
└────────────────────────────────────────────────────────────┘
                              │
                              │ SSH/SFTP Protocol
                              ▼
┌────────────────────────────────────────────────────────────┐
│                    SFTP Transport Layer                    │
│                                                            │
│  ┌──────────────┐  ┌────────────────┐  ┌───────────────┐  │
│  │  SSH Server  │  │ SFTP Handlers  │  │ Auth Provider │  │
│  │  (listener)  │──│  (subsystem)   │──│  (callbacks)  │  │
│  └──────────────┘  └────────────────┘  └───────────────┘  │
│                              │                             │
│                              ▼                             │
│                    ┌─────────────────┐                     │
│                    │ Path Translator │                     │
│                    │ /bucket/key → B │                     │
│                    └─────────────────┘                     │
└────────────────────────────────────────────────────────────┘
                              │
                              │ storage.Storage API
                              ▼
┌────────────────────────────────────────────────────────────┐
│                    storage.Storage                         │
│           (local, s3, memory, azureblob, etc.)            │
└────────────────────────────────────────────────────────────┘
```

## Directory/Path Mapping

SFTP clients see a virtual filesystem where:

```
/                          → Storage root (list buckets)
/<bucket>/                 → Bucket root (list objects in bucket)
/<bucket>/<key>            → Object (file)
/<bucket>/dir1/dir2/file   → Nested object key: "dir1/dir2/file"
```

### Path Translation Rules

1. **Root directory (`/`)**: Lists all buckets as subdirectories
2. **First level (`/<bucket>`)**: Maps to `storage.Bucket(name)`
3. **Nested paths (`/<bucket>/path/to/file`)**: Maps to object key `path/to/file`
4. **Trailing slashes**: Normalized away; directories are virtual based on key prefixes

### Example Mappings

| SFTP Path | Storage Operation |
|-----------|------------------|
| `ls /` | `storage.Buckets()` |
| `mkdir /mybucket` | `storage.CreateBucket("mybucket")` |
| `rmdir /mybucket` | `storage.DeleteBucket("mybucket")` |
| `ls /mybucket` | `bucket.List("")` |
| `put file /mybucket/data.txt` | `bucket.Write("data.txt", ...)` |
| `get /mybucket/data.txt` | `bucket.Open("data.txt", ...)` |
| `rm /mybucket/data.txt` | `bucket.Delete("data.txt")` |
| `ls /mybucket/folder/` | `bucket.List("folder/")` |
| `put file /mybucket/a/b/c.txt` | `bucket.Write("a/b/c.txt", ...)` |

## SFTP Operations Mapping

### File Operations

| SFTP Operation | Storage API | Notes |
|----------------|-------------|-------|
| `SSH_FXP_OPEN` (read) | `Bucket.Open()` | Supports offset/length for range reads |
| `SSH_FXP_OPEN` (write) | `Bucket.Write()` | Creates temp buffer, writes on close |
| `SSH_FXP_CLOSE` | Finalize write | Commits buffered writes to storage |
| `SSH_FXP_READ` | io.Reader from Open | Sequential or offset reads |
| `SSH_FXP_WRITE` | Buffer writes | Committed on close |
| `SSH_FXP_REMOVE` | `Bucket.Delete()` | Deletes single object |
| `SSH_FXP_RENAME` | `Bucket.Move()` | Within same bucket |
| `SSH_FXP_STAT` | `Bucket.Stat()` | Returns file attributes |
| `SSH_FXP_LSTAT` | `Bucket.Stat()` | Same as STAT (no symlinks) |
| `SSH_FXP_FSTAT` | Cached attributes | From open handle |
| `SSH_FXP_SETSTAT` | Limited support | Only mtime if backend supports |

### Directory Operations

| SFTP Operation | Storage API | Notes |
|----------------|-------------|-------|
| `SSH_FXP_OPENDIR` | `Bucket.List()` | Opens directory iterator |
| `SSH_FXP_READDIR` | Iterator.Next() | Returns file entries |
| `SSH_FXP_MKDIR` | `Storage.CreateBucket()` or implicit | See note below |
| `SSH_FXP_RMDIR` | `Storage.DeleteBucket()` or `Bucket.Delete()` recursive | Depends on path depth |
| `SSH_FXP_REALPATH` | Path normalization | Resolves `.` and `..` |

**Note on mkdir**:
- `mkdir /bucket` creates a new bucket
- `mkdir /bucket/path/to/dir` is typically a no-op since object storage has virtual directories, but the server may create a zero-byte `.keep` marker if the backend supports it

### Unsupported Operations

| SFTP Operation | Reason |
|----------------|--------|
| `SSH_FXP_SYMLINK` | Object storage has no symlink concept |
| `SSH_FXP_READLINK` | No symlinks |
| `SSH_FXP_LINK` | No hard links |
| Extended attributes | Not mapped to storage metadata |

## Authentication

The SFTP transport supports multiple authentication methods:

### 1. Public Key Authentication (Recommended)

```go
type PublicKeyAuth struct {
    // AuthorizedKeys maps usernames to their authorized public keys
    AuthorizedKeys map[string][]ssh.PublicKey
}
```

### 2. Password Authentication

```go
type PasswordAuth struct {
    // Verify checks username/password combinations
    Verify func(username, password string) bool
}
```

### 3. Custom Authentication

```go
type AuthProvider interface {
    // Authenticate validates the SSH connection
    // Returns the authenticated username or error
    Authenticate(conn ssh.ConnMetadata, method string, payload []byte) (*AuthResult, error)
}

type AuthResult struct {
    Username string
    Metadata map[string]string // Passed to storage options
}
```

### Per-User Bucket Restrictions

```go
type Config struct {
    // BucketAccess controls which buckets a user can access
    // If nil, user has access to all buckets
    BucketAccess func(username string) []string

    // HomeBucket if set, chroots the user to this bucket
    HomeBucket func(username string) string
}
```

## Configuration

### Server Configuration

```go
type Config struct {
    // HostKeys are the server's private keys (required)
    HostKeys []ssh.Signer

    // Listen address (default: ":2022")
    Addr string

    // MaxConnections limits concurrent connections (0 = unlimited)
    MaxConnections int

    // IdleTimeout for inactive sessions (default: 10 minutes)
    IdleTimeout time.Duration

    // MaxPacketSize for SFTP packets (default: 32KB)
    MaxPacketSize uint32

    // ReadOnly disables all write operations
    ReadOnly bool

    // Auth configuration (required)
    Auth AuthConfig

    // Logger for debugging and auditing
    Logger *slog.Logger

    // Banner displayed before authentication
    Banner string

    // AllowedCiphers restricts SSH ciphers (nil = defaults)
    AllowedCiphers []string

    // AllowedMACs restricts MAC algorithms (nil = defaults)
    AllowedMACs []string

    // AllowedKeyExchanges restricts key exchange algorithms (nil = defaults)
    AllowedKeyExchanges []string
}
```

### AuthConfig

```go
type AuthConfig struct {
    // PublicKeyCallback for public key auth
    PublicKeyCallback func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error)

    // PasswordCallback for password auth (optional)
    PasswordCallback func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error)

    // NoClientAuth allows anonymous access (dangerous)
    NoClientAuth bool

    // MaxAuthTries limits authentication attempts (default: 6)
    MaxAuthTries int
}
```

## API

### Creating the Server

```go
import (
    "open-lake.dev/lib/storage"
    "open-lake.dev/lib/storage/transport/sftp"
)

// Create storage backend
store, _ := storage.Open(ctx, "local:///data")

// Load host key
hostKey, _ := ssh.ParsePrivateKey(keyBytes)

// Configure authentication
authCfg := sftp.AuthConfig{
    PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
        // Validate public key
        if isAuthorizedKey(conn.User(), key) {
            return &ssh.Permissions{}, nil
        }
        return nil, errors.New("unauthorized")
    },
}

// Create server
cfg := &sftp.Config{
    Addr:     ":2022",
    HostKeys: []ssh.Signer{hostKey},
    Auth:     authCfg,
    Logger:   slog.Default(),
}

server := sftp.New(store, cfg)

// Start serving
err := server.ListenAndServe()
```

### Using with Existing SSH Server

```go
// For integration with existing SSH infrastructure
handler := sftp.NewHandler(store, nil)

// Use with ssh.ServerConn
sshConn, chans, reqs, _ := ssh.NewServerConn(netConn, sshConfig)
go ssh.DiscardRequests(reqs)

for newChannel := range chans {
    if newChannel.ChannelType() == "session" {
        channel, requests, _ := newChannel.Accept()
        go handler.HandleChannel(channel, requests, sshConn.User())
    }
}
```

### Graceful Shutdown

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := server.Shutdown(ctx); err != nil {
    log.Printf("Shutdown error: %v", err)
}
```

## File Attributes

SFTP file attributes are mapped as follows:

### From Storage to SFTP

| Object Field | SFTP Attribute | Notes |
|--------------|----------------|-------|
| `Size` | `SSH_FILEXFER_ATTR_SIZE` | File size in bytes |
| `Updated` | `SSH_FILEXFER_ATTR_ACMODTIME` | Access and mod time |
| `IsDir` | `SSH_FILEXFER_ATTR_PERMISSIONS` | Directory flag in mode |
| `ContentType` | (extension) | Available via stat extensions |
| N/A | `SSH_FILEXFER_ATTR_UIDGID` | Returns configured UID/GID |

### Permissions

Since object storage doesn't have POSIX permissions:

- **Files**: Mode `0644` (or `0444` in read-only mode)
- **Directories**: Mode `0755` (or `0555` in read-only mode)
- **UID/GID**: Configurable, defaults to the SSH connection user

## Error Mapping

| Storage Error | SFTP Error Code |
|---------------|-----------------|
| `ErrNotExist` | `SSH_FX_NO_SUCH_FILE` |
| `ErrExist` | `SSH_FX_FAILURE` with message |
| `ErrPermission` | `SSH_FX_PERMISSION_DENIED` |
| `ErrUnsupported` | `SSH_FX_OP_UNSUPPORTED` |
| Context canceled | `SSH_FX_CONNECTION_LOST` |
| Other errors | `SSH_FX_FAILURE` |

## Concurrency and Performance

### Connection Handling

- Each SSH connection runs in its own goroutine
- Multiple SFTP requests per session can be handled concurrently
- Session-level mutex protects handle map access

### Large File Handling

```go
type Config struct {
    // WriteBufferSize for upload buffering (default: 32MB)
    WriteBufferSize int64

    // WriteToDisk buffers large uploads to temp files
    WriteToDisk bool

    // TempDir for disk buffering (default: os.TempDir())
    TempDir string
}
```

### Streaming Reads

Reads are streamed directly from `Bucket.Open()` without full buffering.

## Security Considerations

### Recommended Configuration

1. **Use public key authentication** - Disable password auth in production
2. **Restrict ciphers** - Use only modern, secure ciphers
3. **Set idle timeout** - Prevent resource exhaustion
4. **Enable logging** - Audit all file operations
5. **Use TLS termination** - If proxying through load balancer

### Example Secure Config

```go
cfg := &sftp.Config{
    Addr:     ":2022",
    HostKeys: []ssh.Signer{ed25519Key},
    Auth: sftp.AuthConfig{
        PublicKeyCallback: validatePublicKey,
        MaxAuthTries:      3,
    },
    IdleTimeout: 5 * time.Minute,
    MaxConnections: 100,
    AllowedCiphers: []string{
        "chacha20-poly1305@openssh.com",
        "aes256-gcm@openssh.com",
        "aes128-gcm@openssh.com",
    },
    AllowedMACs: []string{
        "hmac-sha2-256-etm@openssh.com",
        "hmac-sha2-512-etm@openssh.com",
    },
    AllowedKeyExchanges: []string{
        "curve25519-sha256",
        "curve25519-sha256@libssh.org",
    },
}
```

## Limitations

1. **No symbolic links**: Object storage doesn't support symlinks
2. **No hard links**: Objects are independent entities
3. **No extended attributes**: Use metadata options if needed
4. **Virtual directories**: mkdir for nested paths may be a no-op
5. **Atomic operations**: Writes are not atomic during upload
6. **Append mode**: Most object stores don't support append
7. **Truncate**: Not supported for partial overwrites

## Testing

The package includes comprehensive tests:

```bash
# Run all tests
go test ./lib/storage/transport/sftp/...

# Run with verbose output
go test -v ./lib/storage/transport/sftp/...

# Run specific test
go test -run TestSFTPReadWrite ./lib/storage/transport/sftp/...
```

### Integration Testing

Tests use an in-memory storage backend by default. For integration testing with real SFTP clients:

```bash
# Start test server on port 2022
go test -run TestIntegration -sftp-listen=:2022 ./lib/storage/transport/sftp/...

# Connect with sftp client
sftp -P 2022 -i ~/.ssh/test_key testuser@localhost
```

## Examples

### Basic Server

```go
package main

import (
    "context"
    "log"
    "os"

    "golang.org/x/crypto/ssh"
    "open-lake.dev/lib/storage"
    "open-lake.dev/lib/storage/transport/sftp"
)

func main() {
    ctx := context.Background()

    // Open local storage
    store, err := storage.Open(ctx, "local:///var/data")
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()

    // Load host key
    keyBytes, _ := os.ReadFile("/etc/ssh/ssh_host_ed25519_key")
    hostKey, _ := ssh.ParsePrivateKey(keyBytes)

    // Load authorized keys
    authorizedKeys := loadAuthorizedKeys("/etc/sftp/authorized_keys")

    cfg := &sftp.Config{
        Addr:     ":2022",
        HostKeys: []ssh.Signer{hostKey},
        Auth: sftp.AuthConfig{
            PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
                if isAuthorized(conn.User(), key, authorizedKeys) {
                    return &ssh.Permissions{}, nil
                }
                return nil, fmt.Errorf("unauthorized key for %s", conn.User())
            },
        },
    }

    server := sftp.New(store, cfg)
    log.Printf("SFTP server listening on %s", cfg.Addr)

    if err := server.ListenAndServe(); err != nil {
        log.Fatal(err)
    }
}
```

### S3 Backend with User Isolation

```go
cfg := &sftp.Config{
    Addr:     ":2022",
    HostKeys: []ssh.Signer{hostKey},
    Auth: sftp.AuthConfig{
        PublicKeyCallback: validateKey,
    },
    // Each user sees only their bucket
    HomeBucket: func(username string) string {
        return "user-" + username
    },
}

store, _ := storage.Open(ctx, "s3://my-sftp-data?region=us-east-1")
server := sftp.New(store, cfg)
```

### Read-Only Access

```go
cfg := &sftp.Config{
    Addr:     ":2022",
    HostKeys: []ssh.Signer{hostKey},
    Auth: sftp.AuthConfig{
        PublicKeyCallback: validateKey,
    },
    ReadOnly: true, // Disable all write operations
}
```

## Wire Protocol Reference

For implementers, the SFTP protocol version 3 (RFC draft) is used:

- **SSH_FXP_INIT** (1): Client initialization
- **SSH_FXP_VERSION** (2): Server version response
- **SSH_FXP_OPEN** (3): Open file
- **SSH_FXP_CLOSE** (4): Close handle
- **SSH_FXP_READ** (5): Read file data
- **SSH_FXP_WRITE** (6): Write file data
- **SSH_FXP_LSTAT** (7): Stat without following links
- **SSH_FXP_FSTAT** (8): Stat open handle
- **SSH_FXP_SETSTAT** (9): Set file attributes
- **SSH_FXP_FSETSTAT** (10): Set open handle attributes
- **SSH_FXP_OPENDIR** (11): Open directory
- **SSH_FXP_READDIR** (12): Read directory entries
- **SSH_FXP_REMOVE** (13): Delete file
- **SSH_FXP_MKDIR** (14): Create directory
- **SSH_FXP_RMDIR** (15): Remove directory
- **SSH_FXP_REALPATH** (16): Canonicalize path
- **SSH_FXP_STAT** (17): Stat following links
- **SSH_FXP_RENAME** (18): Rename file
- **SSH_FXP_READLINK** (19): Read symbolic link (unsupported)
- **SSH_FXP_SYMLINK** (20): Create symbolic link (unsupported)

Response packets:
- **SSH_FXP_STATUS** (101): Status/error response
- **SSH_FXP_HANDLE** (102): File handle
- **SSH_FXP_DATA** (103): Read data
- **SSH_FXP_NAME** (104): Name entries
- **SSH_FXP_ATTRS** (105): File attributes

## Changelog

### v1.0.0

- Initial release
- Full SFTP protocol v3 support
- Public key and password authentication
- User isolation with HomeBucket
- Read-only mode support
- Comprehensive test suite
