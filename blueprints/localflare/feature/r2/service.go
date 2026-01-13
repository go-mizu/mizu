package r2

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"
)

// Service implements the R2 API.
type Service struct {
	store Store
}

// NewService creates a new R2 service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreateBucket creates a new bucket.
func (s *Service) CreateBucket(ctx context.Context, in *CreateBucketIn) (*Bucket, error) {
	if in.Name == "" {
		return nil, ErrNameRequired
	}

	location := in.Location
	if location == "" {
		location = "auto"
	}

	bucket := &Bucket{
		ID:        ulid.Make().String(),
		Name:      in.Name,
		Location:  location,
		CreatedAt: time.Now(),
	}

	if err := s.store.CreateBucket(ctx, bucket); err != nil {
		return nil, err
	}

	return bucket, nil
}

// GetBucket retrieves a bucket by ID.
func (s *Service) GetBucket(ctx context.Context, id string) (*Bucket, error) {
	bucket, err := s.store.GetBucket(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return bucket, nil
}

// ListBuckets lists all buckets.
func (s *Service) ListBuckets(ctx context.Context) ([]*Bucket, error) {
	return s.store.ListBuckets(ctx)
}

// DeleteBucket deletes a bucket.
func (s *Service) DeleteBucket(ctx context.Context, id string) error {
	return s.store.DeleteBucket(ctx, id)
}

// PutObject stores an object.
func (s *Service) PutObject(ctx context.Context, bucketID string, in *PutObjectIn) error {
	metadata := in.Metadata
	if metadata == nil {
		metadata = make(map[string]string)
	}
	if in.ContentType != "" {
		metadata["content-type"] = in.ContentType
	}

	return s.store.PutObject(ctx, bucketID, in.Key, in.Data, metadata)
}

// GetObject retrieves an object.
func (s *Service) GetObject(ctx context.Context, bucketID, key string) ([]byte, *Object, error) {
	data, obj, err := s.store.GetObject(ctx, bucketID, key)
	if err != nil {
		return nil, nil, ErrObjectNotFound
	}
	return data, obj, nil
}

// DeleteObject deletes an object.
func (s *Service) DeleteObject(ctx context.Context, bucketID, key string) error {
	return s.store.DeleteObject(ctx, bucketID, key)
}

// ListObjects lists objects in a bucket.
func (s *Service) ListObjects(ctx context.Context, bucketID string, opts ListObjectsOpts) ([]*Object, error) {
	if opts.Limit <= 0 {
		opts.Limit = 1000
	}
	return s.store.ListObjects(ctx, bucketID, opts.Prefix, opts.Delimiter, opts.Limit)
}
