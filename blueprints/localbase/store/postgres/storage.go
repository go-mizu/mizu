package postgres

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StorageStore implements store.StorageStore using PostgreSQL.
type StorageStore struct {
	pool *pgxpool.Pool
}

// CreateBucket creates a new storage bucket.
func (s *StorageStore) CreateBucket(ctx context.Context, bucket *store.Bucket) error {
	sql := `
	INSERT INTO storage.buckets (id, name, public, file_size_limit, allowed_mime_types)
	VALUES ($1, $2, $3, $4, $5)
	`

	_, err := s.pool.Exec(ctx, sql,
		bucket.ID,
		bucket.Name,
		bucket.Public,
		bucket.FileSizeLimit,
		bucket.AllowedMimeTypes,
	)

	return err
}

// GetBucket retrieves a bucket by ID.
func (s *StorageStore) GetBucket(ctx context.Context, id string) (*store.Bucket, error) {
	sql := `
	SELECT id, name, public, file_size_limit, allowed_mime_types, created_at, updated_at
	FROM storage.buckets
	WHERE id = $1
	`

	bucket := &store.Bucket{}

	err := s.pool.QueryRow(ctx, sql, id).Scan(
		&bucket.ID,
		&bucket.Name,
		&bucket.Public,
		&bucket.FileSizeLimit,
		&bucket.AllowedMimeTypes,
		&bucket.CreatedAt,
		&bucket.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("bucket not found")
	}
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

// GetBucketByName retrieves a bucket by name.
func (s *StorageStore) GetBucketByName(ctx context.Context, name string) (*store.Bucket, error) {
	sql := `
	SELECT id, name, public, file_size_limit, allowed_mime_types, created_at, updated_at
	FROM storage.buckets
	WHERE name = $1
	`

	bucket := &store.Bucket{}

	err := s.pool.QueryRow(ctx, sql, name).Scan(
		&bucket.ID,
		&bucket.Name,
		&bucket.Public,
		&bucket.FileSizeLimit,
		&bucket.AllowedMimeTypes,
		&bucket.CreatedAt,
		&bucket.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("bucket not found")
	}
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

// ListBuckets lists all buckets.
func (s *StorageStore) ListBuckets(ctx context.Context) ([]*store.Bucket, error) {
	sql := `
	SELECT id, name, public, file_size_limit, allowed_mime_types, created_at, updated_at
	FROM storage.buckets
	ORDER BY name
	`

	rows, err := s.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buckets []*store.Bucket
	for rows.Next() {
		bucket := &store.Bucket{}

		err := rows.Scan(
			&bucket.ID,
			&bucket.Name,
			&bucket.Public,
			&bucket.FileSizeLimit,
			&bucket.AllowedMimeTypes,
			&bucket.CreatedAt,
			&bucket.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		buckets = append(buckets, bucket)
	}

	return buckets, nil
}

// UpdateBucket updates a bucket.
func (s *StorageStore) UpdateBucket(ctx context.Context, bucket *store.Bucket) error {
	sql := `
	UPDATE storage.buckets
	SET name = $2, public = $3, file_size_limit = $4, allowed_mime_types = $5, updated_at = NOW()
	WHERE id = $1
	`

	_, err := s.pool.Exec(ctx, sql,
		bucket.ID,
		bucket.Name,
		bucket.Public,
		bucket.FileSizeLimit,
		bucket.AllowedMimeTypes,
	)

	return err
}

// DeleteBucket deletes a bucket.
func (s *StorageStore) DeleteBucket(ctx context.Context, id string) error {
	sql := `DELETE FROM storage.buckets WHERE id = $1`
	_, err := s.pool.Exec(ctx, sql, id)
	return err
}

// CreateObject creates a new storage object.
func (s *StorageStore) CreateObject(ctx context.Context, obj *store.Object) error {
	sql := `
	INSERT INTO storage.objects (id, bucket_id, name, owner, version, metadata, content_type, size)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := s.pool.Exec(ctx, sql,
		obj.ID,
		obj.BucketID,
		obj.Name,
		nullIfEmpty(obj.Owner),
		nullIfEmpty(obj.Version),
		obj.Metadata,
		nullIfEmpty(obj.ContentType),
		obj.Size,
	)

	return err
}

// GetObject retrieves an object by bucket ID and name.
func (s *StorageStore) GetObject(ctx context.Context, bucketID, name string) (*store.Object, error) {
	sql := `
	SELECT id, bucket_id, name, owner, path_tokens, version, metadata, content_type, size,
		created_at, updated_at, last_accessed_at
	FROM storage.objects
	WHERE bucket_id = $1 AND name = $2
	`

	obj := &store.Object{}
	var owner, version, contentType *string

	err := s.pool.QueryRow(ctx, sql, bucketID, name).Scan(
		&obj.ID,
		&obj.BucketID,
		&obj.Name,
		&owner,
		&obj.PathTokens,
		&version,
		&obj.Metadata,
		&contentType,
		&obj.Size,
		&obj.CreatedAt,
		&obj.UpdatedAt,
		&obj.LastAccessedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("object not found")
	}
	if err != nil {
		return nil, err
	}

	if owner != nil {
		obj.Owner = *owner
	}
	if version != nil {
		obj.Version = *version
	}
	if contentType != nil {
		obj.ContentType = *contentType
	}

	return obj, nil
}

// ListObjects lists objects in a bucket with optional prefix filter.
func (s *StorageStore) ListObjects(ctx context.Context, bucketID, prefix string, limit, offset int) ([]*store.Object, error) {
	sql := `
	SELECT id, bucket_id, name, owner, path_tokens, version, metadata, content_type, size,
		created_at, updated_at, last_accessed_at
	FROM storage.objects
	WHERE bucket_id = $1 AND ($2 = '' OR name LIKE $2 || '%')
	ORDER BY name
	LIMIT $3 OFFSET $4
	`

	rows, err := s.pool.Query(ctx, sql, bucketID, prefix, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var objects []*store.Object
	for rows.Next() {
		obj := &store.Object{}
		var owner, version, contentType *string

		err := rows.Scan(
			&obj.ID,
			&obj.BucketID,
			&obj.Name,
			&owner,
			&obj.PathTokens,
			&version,
			&obj.Metadata,
			&contentType,
			&obj.Size,
			&obj.CreatedAt,
			&obj.UpdatedAt,
			&obj.LastAccessedAt,
		)
		if err != nil {
			return nil, err
		}

		if owner != nil {
			obj.Owner = *owner
		}
		if version != nil {
			obj.Version = *version
		}
		if contentType != nil {
			obj.ContentType = *contentType
		}

		objects = append(objects, obj)
	}

	return objects, nil
}

// UpdateObject updates an object.
func (s *StorageStore) UpdateObject(ctx context.Context, obj *store.Object) error {
	sql := `
	UPDATE storage.objects
	SET metadata = $3, content_type = $4, size = $5, updated_at = NOW()
	WHERE bucket_id = $1 AND name = $2
	`

	_, err := s.pool.Exec(ctx, sql,
		obj.BucketID,
		obj.Name,
		obj.Metadata,
		nullIfEmpty(obj.ContentType),
		obj.Size,
	)

	return err
}

// DeleteObject deletes an object.
func (s *StorageStore) DeleteObject(ctx context.Context, bucketID, name string) error {
	sql := `DELETE FROM storage.objects WHERE bucket_id = $1 AND name = $2`
	_, err := s.pool.Exec(ctx, sql, bucketID, name)
	return err
}

// MoveObject moves/renames an object.
func (s *StorageStore) MoveObject(ctx context.Context, bucketID, srcName, dstName string) error {
	sql := `
	UPDATE storage.objects
	SET name = $3, updated_at = NOW()
	WHERE bucket_id = $1 AND name = $2
	`

	_, err := s.pool.Exec(ctx, sql, bucketID, srcName, dstName)
	return err
}

// CopyObject copies an object to a new location.
func (s *StorageStore) CopyObject(ctx context.Context, srcBucketID, srcName, dstBucketID, dstName string) error {
	sql := `
	INSERT INTO storage.objects (id, bucket_id, name, owner, version, metadata, content_type, size)
	SELECT gen_random_uuid(), $3, $4, owner, version, metadata, content_type, size
	FROM storage.objects
	WHERE bucket_id = $1 AND name = $2
	`

	_, err := s.pool.Exec(ctx, sql, srcBucketID, srcName, dstBucketID, dstName)
	return err
}

// UpdateObjectSize updates only the size of an object.
func (s *StorageStore) UpdateObjectSize(ctx context.Context, objectID string, size int64) error {
	sql := `UPDATE storage.objects SET size = $2, updated_at = NOW() WHERE id = $1`
	_, err := s.pool.Exec(ctx, sql, objectID, size)
	return err
}
