// Package r2 provides R2 storage management.
package r2

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("bucket not found")
	ErrObjectNotFound = errors.New("object not found")
	ErrNameRequired = errors.New("name is required")
)

// Bucket represents an R2 bucket.
type Bucket struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Location  string    `json:"location"`
	CreatedAt time.Time `json:"created_at"`
}

// Object represents an R2 object.
type Object struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	ETag         string            `json:"etag"`
	ContentType  string            `json:"content_type"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	LastModified time.Time         `json:"last_modified"`
}

// CreateBucketIn contains input for creating a bucket.
type CreateBucketIn struct {
	Name     string `json:"name"`
	Location string `json:"location"`
}

// PutObjectIn contains input for putting an object.
type PutObjectIn struct {
	Key         string            `json:"key"`
	Data        []byte            `json:"-"`
	ContentType string            `json:"content_type"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ListObjectsOpts specifies options for listing objects.
type ListObjectsOpts struct {
	Prefix    string
	Delimiter string
	Limit     int
}

// API defines the R2 service contract.
type API interface {
	CreateBucket(ctx context.Context, in *CreateBucketIn) (*Bucket, error)
	GetBucket(ctx context.Context, id string) (*Bucket, error)
	ListBuckets(ctx context.Context) ([]*Bucket, error)
	DeleteBucket(ctx context.Context, id string) error
	PutObject(ctx context.Context, bucketID string, in *PutObjectIn) error
	GetObject(ctx context.Context, bucketID, key string) ([]byte, *Object, error)
	DeleteObject(ctx context.Context, bucketID, key string) error
	ListObjects(ctx context.Context, bucketID string, opts ListObjectsOpts) ([]*Object, error)
}

// Store defines the data access contract.
type Store interface {
	CreateBucket(ctx context.Context, bucket *Bucket) error
	GetBucket(ctx context.Context, id string) (*Bucket, error)
	ListBuckets(ctx context.Context) ([]*Bucket, error)
	DeleteBucket(ctx context.Context, id string) error
	PutObject(ctx context.Context, bucketID, key string, data []byte, metadata map[string]string) error
	GetObject(ctx context.Context, bucketID, key string) ([]byte, *Object, error)
	DeleteObject(ctx context.Context, bucketID, key string) error
	ListObjects(ctx context.Context, bucketID, prefix, delimiter string, limit int) ([]*Object, error)
}
