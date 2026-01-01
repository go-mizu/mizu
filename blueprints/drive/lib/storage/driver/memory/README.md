# Memory Storage Driver

The `memdriver` package provides an in-memory implementation of the storage interface. It's designed for testing, development, and scenarios where temporary storage is needed.

## Features

- Full implementation of `storage.Storage` and `storage.Bucket` interfaces
- Multipart upload support (`storage.HasMultipart`)
- Directory operations (`storage.HasDirectories`)
- Thread-safe operations with proper locking
- No persistence - data is lost when the process exits

## Installation

The driver automatically registers itself on import:

```go
import (
    "open-lake.dev/lib/storage"
    _ "open-lake.dev/lib/storage/driver/memory"
)
```

## DSN Format

```
mem://[name][?bucket=default_bucket]
```

Examples:
- `mem://` - Basic memory storage
- `mem://test` - Named instance (name currently reserved for future use)
- `mem://?bucket=data` - With default bucket name

## Usage

### Basic Operations

```go
import (
    "context"
    "strings"
    "open-lake.dev/lib/storage"
    _ "open-lake.dev/lib/storage/driver/memory"
)

func main() {
    ctx := context.Background()

    // Open memory storage
    st, err := storage.Open(ctx, "mem://")
    if err != nil {
        panic(err)
    }
    defer st.Close()

    // Create a bucket
    _, err = st.CreateBucket(ctx, "test-bucket", nil)

    // Get bucket handle
    bucket := st.Bucket("test-bucket")

    // Write object
    _, err = bucket.Write(ctx, "key.txt", strings.NewReader("content"), 7, "text/plain", nil)

    // Read object
    rc, obj, err := bucket.Open(ctx, "key.txt", 0, 0, nil)
    // ...
}
```

### Multipart Uploads

```go
mp := bucket.(storage.HasMultipart)

// Start upload
mu, _ := mp.InitMultipart(ctx, "large.bin", "application/octet-stream", nil)

// Upload parts
part1, _ := mp.UploadPart(ctx, mu, 1, reader1, partSize, nil)
part2, _ := mp.UploadPart(ctx, mu, 2, reader2, partSize, nil)

// Complete
obj, _ := mp.CompleteMultipart(ctx, mu, []*storage.PartInfo{part1, part2}, nil)
```

### Directory Operations

```go
dirs := bucket.(storage.HasDirectories)

// Get directory handle
dir := dirs.Directory("path/to/dir")

// List contents
iter, _ := dir.List(ctx, 100, 0, nil)

// Move directory
newDir, _ := dir.Move(ctx, "new/path", nil)

// Delete directory
dir.Delete(ctx, storage.Options{"recursive": true})
```

## Testing

The memory driver is ideal for unit tests:

```go
func TestMyFeature(t *testing.T) {
    ctx := context.Background()

    // Create isolated storage for this test
    st, _ := storage.Open(ctx, "mem://")
    defer st.Close()

    // Run tests against st
    bucket := st.Bucket("test")
    // ...
}
```

### Factory Pattern

For conformance testing:

```go
func memoryFactory(t *testing.T) (storage.Storage, func()) {
    ctx := context.Background()
    st, err := storage.Open(ctx, "mem://")
    if err != nil {
        t.Fatalf("Open: %v", err)
    }
    return st, func() { st.Close() }
}

func TestConformance(t *testing.T) {
    storage_test.ConformanceSuite(t, memoryFactory)
}
```

## Supported Features

```go
features := st.Features()
// Returns:
// {
//     "move":             true,
//     "server_side_move": true,
//     "server_side_copy": true,
//     "directories":      true,
//     "multipart":        true,
//     "hash:md5":         false,
//     "watch":            false,
//     "public_url":       false,
//     "signed_url":       false,
// }
```

## Limitations

1. **No Persistence**: Data exists only in memory and is lost on process exit
2. **Memory Usage**: Large files consume equivalent RAM
3. **No SignedURL**: Returns `ErrUnsupported` for `SignedURL()`
4. **No CopyPart**: Multipart `CopyPart()` returns `ErrUnsupported`
5. **Single Process**: No sharing between processes (name parameter is reserved)

## Thread Safety

All operations are thread-safe:
- Store-level operations use `sync.RWMutex`
- Bucket-level operations use separate `sync.RWMutex`
- Multipart operations use dedicated `sync.RWMutex`

Concurrent reads and writes to different keys are safe. Concurrent writes to the same key will result in last-write-wins behavior.

## Implementation Details

### Object Storage

Objects are stored in a map with the key as the map key:

```go
type entry struct {
    obj  storage.Object  // Metadata
    data []byte          // Content
}

type bucket struct {
    obj map[string]*entry
}
```

### Multipart Upload

Multipart uploads are tracked separately until completion:

```go
type multipartUpload struct {
    mu          *storage.MultipartUpload
    contentType string
    metadata    map[string]string
    parts       map[int]*partData
}
```

Parts are stored in memory and concatenated on completion. The final object replaces any existing object with the same key.
