// File: lib/storage/multipart.go
package storage

import (
	"context"
	"io"
	"time"
)

// MultipartUpload describes an in progress multipart upload.
//
// For S3 this corresponds to (Bucket, Key, UploadId).
type MultipartUpload struct {
	Bucket   string            `json:"bucket,omitempty"`
	Key      string            `json:"key,omitempty"`
	UploadID string            `json:"upload_id,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// PartInfo describes a single uploaded part.
type PartInfo struct {
	// Number is the 1 based part number.
	Number int `json:"number,omitempty"`

	// Size is the size of this part in bytes.
	Size int64 `json:"size,omitempty"`

	// ETag is a provider specific identifier. For S3 this is the ETag
	// returned when uploading the part.
	ETag string `json:"etag,omitempty"`

	// Hash holds optional checksums for this part when the provider
	// computes or accepts them.
	Hash Hashes `json:"hash,omitempty"`

	// LastModified is the provider-reported time when available.
	// Nil means the provider did not give it.
	LastModified *time.Time `json:"last_modified,omitempty"`
}

// HasMultipart indicates a bucket supports multipart upload operations.
//
// Backends that implement multipart should have their Bucket type implement
// this interface in addition to Bucket.
type HasMultipart interface {
	// InitMultipart starts a multipart upload for the given key.
	//
	// Common options (same as Bucket.Write):
	//   "metadata": map[string]string
	//   "tags": map[string]string
	//   "cache_control": string
	//   "content_encoding": string
	//   "content_disposition": string
	//   "content_language": string
	//   "expires": time.Time or string
	//   "acl": string
	//
	// Encryption:
	//   "sse_algorithm": string
	//   "sse_kms_key_id": string
	InitMultipart(ctx context.Context, key string, contentType string, opts Options) (*MultipartUpload, error)

	// UploadPart uploads a single part for an existing multipart upload.
	//
	// number must be in the provider allowed range, for S3 this is 1..10000.
	// size is the expected length. If negative, provider may stream but some
	// backends require a known size.
	//
	// Common options:
	//   "checksum": Hashes
	//   "checksum_algorithms": []string
	UploadPart(ctx context.Context, mu *MultipartUpload, number int, src io.Reader, size int64, opts Options) (*PartInfo, error)

	// CopyPart uploads a single part by copying a range from an existing
	// source object when the provider supports server side copy.
	//
	// Common options:
	//   "source_bucket": string  // defaults to mu.Bucket if empty
	//   "source_key": string
	//   "source_offset": int64   // start byte, inclusive
	//   "source_length": int64   // length in bytes, negative means to end
	//
	// Conditional copy may be supported via:
	//   "if_match"
	//   "if_none_match"
	CopyPart(ctx context.Context, mu *MultipartUpload, number int, opts Options) (*PartInfo, error)

	// ListParts lists already uploaded parts for a multipart upload.
	//
	// limit and offset are for client side pagination. Providers that expose
	// their own paging mechanisms may map those internally.
	ListParts(ctx context.Context, mu *MultipartUpload, limit, offset int, opts Options) ([]*PartInfo, error)

	// CompleteMultipart completes a multipart upload and assembles the
	// final object from the given parts.
	//
	// parts must contain all successfully uploaded parts that should be
	// included in the final object, in ascending Number order. Providers
	// that require exact ordering and no gaps can validate this.
	//
	// opts may include:
	//   "metadata": map[string]string       // to override or add metadata
	//   "tags": map[string]string
	//   "checksum": Hashes                  // checksum for whole object
	//   "checksum_algorithms": []string
	CompleteMultipart(ctx context.Context, mu *MultipartUpload, parts []*PartInfo, opts Options) (*Object, error)

	// AbortMultipart aborts the multipart upload and discards all parts.
	//
	// After aborting, the MultipartUpload is no longer valid.
	AbortMultipart(ctx context.Context, mu *MultipartUpload, opts Options) error
}
