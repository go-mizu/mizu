// File: lib/storage/storage.go
package storage

import (
	"context"
	"errors"
	"io"
	"time"
)

// Common package level errors for storage backends.
//
// Backends should translate their provider specific errors into these
// when appropriate so callers can switch on them.
var (
	// ErrNotExist indicates a requested bucket, object or directory does not exist.
	ErrNotExist = errors.New("storage: not exist")

	// ErrExist indicates a bucket or object already exists when trying to create.
	ErrExist = errors.New("storage: already exists")

	// ErrPermission indicates an operation is not allowed, for example deleting
	// a non empty bucket without "force".
	ErrPermission = errors.New("storage: permission denied")

	// ErrUnsupported indicates the backend does not support a given operation or option.
	ErrUnsupported = errors.New("storage: unsupported")
)

// Options carries provider specific parameters for operations.
//
// Keys and values are implementation defined. Higher layers may agree
// on common keys by convention, such as:
//
//	"metadata": map[string]string
//	"tags": map[string]string
//	"acl": string
//	"cache_control": string
//	"content_encoding": string
//	"content_disposition": string
//	"content_language": string
//	"expires": time.Time or string
//
// Encryption:
//
//	"sse_algorithm": string
//	"sse_kms_key_id": string
//
// Conditional writes and reads:
//
//	"if_match": string
//	"if_none_match": string
//	"if_unmodified_since": time.Time
//	"if_modified_since": time.Time
//
// Versioning:
//
//	"version": string
//
// Checksums:
//
//	"checksum": Hashes
//	"checksum_algorithms": []string
//
// Multipart copy:
//
//	"source_bucket": string
//	"source_key": string
//	"source_offset": int64
//	"source_length": int64
type Options map[string]any

// Features describes supported capabilities. Missing keys mean false.
//
// Common feature flags:
//
//	"move": true
//	"server_side_copy": true
//	"server_side_move": true
//	"directories": true
//	"public_url": true
//	"signed_url": true
//	"multipart": true
//	"conditional_write": true
//	"versioning": true
//	"watch": true
//
// Hash related feature flags:
//
//	"hash:md5": true
//	"hash:sha1": true
//	"hash:sha256": true
type Features map[string]bool

// Hashes stores checksum values or related identifiers.
//
// Common keys:
//
//	"etag"
//	"md5"
//	"sha1"
//	"sha256"
//	"crc32c"
type Hashes map[string]string

// Storage is the root of a storage backend.
//
// A single Storage typically corresponds to a project, account or
// connection string in the underlying provider.
type Storage interface {
	// Bucket returns a handle. No network IO.
	Bucket(name string) Bucket

	// Buckets enumerates buckets. limit <= 0 means provider default.
	// opts is provider specific.
	Buckets(ctx context.Context, limit, offset int, opts Options) (BucketIter, error)

	// CreateBucket creates a bucket. opts is provider specific.
	CreateBucket(ctx context.Context, name string, opts Options) (*BucketInfo, error)

	// DeleteBucket deletes a bucket. opts is provider specific.
	//
	// Common options:
	//   "force": bool // delete non empty bucket recursively if true
	DeleteBucket(ctx context.Context, name string, opts Options) error

	// Features reports capability flags.
	Features() Features

	// Close releases resources.
	Close() error
}

// Bucket is a logical container.
//
// For S3 like providers this maps to a bucket. For filesystem based
// backends this maps to a directory under a root.
type Bucket interface {
	// Name returns the bucket name.
	Name() string

	// Info returns metadata about this bucket.
	Info(ctx context.Context) (*BucketInfo, error)

	// Features may differ by bucket.
	Features() Features

	// Write creates or replaces an object.
	//
	// size is expected length. If negative, provider may stream.
	//
	// Common options:
	//   "metadata": map[string]string
	//   "tags": map[string]string
	//   "cache_control": string
	//   "content_encoding": string
	//   "content_disposition": string
	//   "content_language": string
	//   "expires": time.Time or string
	//   "acl": string
	//   "upsert": bool
	//
	// Encryption:
	//   "sse_algorithm": string
	//   "sse_kms_key_id": string
	//
	// Conditional (if supported):
	//   "if_match": string
	//   "if_none_match": string
	//   "if_unmodified_since": time.Time
	//   "if_modified_since": time.Time
	//
	// Checksums:
	//   "checksum": Hashes
	//   "checksum_algorithms": []string
	Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts Options) (*Object, error)

	// Open reads an object (full or range).
	//
	// If offset and length are zero, full object is returned.
	// If length is negative, return from offset to end.
	//
	// Conditional and version options may be recognized:
	//   "if_match"
	//   "if_none_match"
	//   "version"
	Open(ctx context.Context, key string, offset, length int64, opts Options) (io.ReadCloser, *Object, error)

	// Stat returns metadata without content.
	//
	// opts may include:
	//   "version"
	//   "if_match"
	//   "if_none_match"
	Stat(ctx context.Context, key string, opts Options) (*Object, error)

	// Delete removes an object.
	//
	// opts may include:
	//   "version"
	//   "recursive"
	//
	// Conditional:
	//   "if_match"
	//   "if_none_match"
	Delete(ctx context.Context, key string, opts Options) error

	// Copy copies an object.
	//
	// opts may include:
	//   "metadata"
	//   "tags"
	//   "override_metadata"
	//   "if_match"
	//   "if_none_match"
	Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts Options) (*Object, error)

	// Move renames an object. May map to server side move or copy plus delete.
	Move(ctx context.Context, dstKey string, srcBucket, srcKey string, opts Options) (*Object, error)

	// List enumerates objects with optional prefix and pagination.
	//
	// prefix is a key prefix to filter results. Empty prefix lists all.
	//
	// Common options:
	//   "recursive": bool              // recurse into sub prefixes
	//   "include_metadata": bool       // provider may choose to fill Metadata
	//   "order_by": string             // provider specific
	//   "order_desc": bool
	//   "dirs_only": bool              // when supported and IsDir is used
	//   "files_only": bool             // when supported and IsDir is used
	List(ctx context.Context, prefix string, limit, offset int, opts Options) (ObjectIter, error)

	// URL returns a public or signed url.
	//
	// method is GET, PUT etc.
	// expires controls validity.
	//
	// opts may include:
	//   "headers": map[string]string
	//   "content_type": string
	//
	// Conditional headers may apply:
	//   "if_match"
	//   "if_none_match"
	SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts Options) (string, error)
}

// BucketInfo describes a bucket.
type BucketInfo struct {
	Name      string            `json:"name,omitempty"`
	CreatedAt time.Time         `json:"created_at,omitempty"`
	Public    bool              `json:"public,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Object describes stored data or a logical directory entry.
//
// When IsDir is false the Object describes a blob or file.
//
// When IsDir is true the Object describes a logical directory. In that case:
//
//   - Bucket, Key and IsDir are always meaningful.
//   - Created and Updated may be filled on a best effort basis.
//   - Size, ContentType, ETag, Version and Hash are backend specific and
//     should not be relied on by generic callers.
type Object struct {
	Bucket      string `json:"bucket,omitempty"`
	Key         string `json:"key,omitempty"`
	Size        int64  `json:"size,omitempty"`
	ContentType string `json:"content_type,omitempty"`

	ETag    string    `json:"etag,omitempty"`
	Version string    `json:"version,omitempty"`
	Created time.Time `json:"created,omitempty"`
	Updated time.Time `json:"updated,omitempty"`

	Hash     Hashes            `json:"hash,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`

	// IsDir indicates this Object represents a logical directory entry
	// rather than a blob. See comment above for field semantics.
	IsDir bool `json:"is_dir,omitempty"`
}

// BucketIter enumerates buckets.
//
// Next returns the next BucketInfo or nil when iteration is done.
// Implementations must be safe to call Close multiple times.
type BucketIter interface {
	Next() (*BucketInfo, error)
	Close() error
}

// ObjectIter enumerates objects.
//
// Next returns the next Object or nil when iteration is done.
// Implementations must be safe to call Close multiple times.
type ObjectIter interface {
	Next() (*Object, error)
	Close() error
}

// Directory is an optional structured view on top of a Bucket.
//
// It models a prefix or path within a bucket and exposes operations that
// feel closer to a filesystem directory. Backends that support this
// can implement HasDirectories.
type Directory interface {
	// Bucket returns the underlying bucket.
	Bucket() Bucket

	// Path returns the directory path, using slash separators and no
	// leading slash. Empty path usually means the bucket root.
	Path() string

	// Info returns metadata for this directory as an Object with IsDir=true.
	Info(ctx context.Context) (*Object, error)

	// List lists objects directly under this directory (non recursive).
	//
	// Common options mirror Bucket.List, but "recursive" is typically
	// ignored here and callers use Bucket.List for recursive traversal.
	List(ctx context.Context, limit, offset int, opts Options) (ObjectIter, error)

	// Delete removes this directory.
	//
	// Common options:
	//   "recursive": bool  // delete recursively if true
	Delete(ctx context.Context, opts Options) error

	// Move renames or moves this directory to dstPath within the same bucket.
	Move(ctx context.Context, dstPath string, opts Options) (Directory, error)
}

// HasDirectories indicates a bucket implements Directory operations.
//
// This is an optional extension. Callers can type assert:
//
//	if hd, ok := bucket.(HasDirectories); ok {
//	    dir := hd.Directory("some/path")
//	    ...
//	}
type HasDirectories interface {
	Directory(path string) Directory
}
