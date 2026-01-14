package s3

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-mizu/blueprints/localflare/pkg/storage"
)

// bucketIter implements storage.BucketIter.
type bucketIter struct {
	buckets []*storage.BucketInfo
	idx     int
}

func (it *bucketIter) Next() (*storage.BucketInfo, error) {
	if it.idx >= len(it.buckets) {
		return nil, nil
	}
	b := it.buckets[it.idx]
	it.idx++
	return b, nil
}

func (it *bucketIter) Close() error {
	it.buckets = nil
	return nil
}

// objectIter implements storage.ObjectIter for S3 with pagination.
type objectIter struct {
	client    *s3.Client
	bucket    string
	input     *s3.ListObjectsV2Input
	limit     int
	offset    int
	dirsOnly  bool
	filesOnly bool

	// Internal state
	objects           []*storage.Object
	idx               int
	continuationToken *string
	done              bool
	skipped           int
	returned          int
}

func (it *objectIter) Next() (*storage.Object, error) {
	// Check if we've returned enough
	if it.limit > 0 && it.returned >= it.limit {
		return nil, nil
	}

	// Try to return from current buffer
	for it.idx < len(it.objects) {
		obj := it.objects[it.idx]
		it.idx++

		// Apply filters
		if it.dirsOnly && !obj.IsDir {
			continue
		}
		if it.filesOnly && obj.IsDir {
			continue
		}

		// Apply offset
		if it.skipped < it.offset {
			it.skipped++
			continue
		}

		it.returned++
		return obj, nil
	}

	// If done, no more results
	if it.done {
		return nil, nil
	}

	// Fetch next page
	if err := it.fetchNextPage(); err != nil {
		return nil, err
	}

	// Try again after fetching
	return it.Next()
}

func (it *objectIter) fetchNextPage() error {
	if it.continuationToken != nil {
		it.input.ContinuationToken = it.continuationToken
	}

	resp, err := it.client.ListObjectsV2(context.Background(), it.input)
	if err != nil {
		return mapS3Error(err)
	}

	it.objects = make([]*storage.Object, 0, len(resp.Contents)+len(resp.CommonPrefixes))
	it.idx = 0

	// Add common prefixes as directories
	for _, prefix := range resp.CommonPrefixes {
		if prefix.Prefix == nil {
			continue
		}
		key := strings.TrimSuffix(*prefix.Prefix, "/")
		it.objects = append(it.objects, &storage.Object{
			Bucket:  it.bucket,
			Key:     key,
			IsDir:   true,
			Updated: time.Now(),
		})
	}

	// Add objects
	for _, obj := range resp.Contents {
		if obj.Key == nil {
			continue
		}

		o := &storage.Object{
			Bucket: it.bucket,
			Key:    *obj.Key,
			Size:   aws.ToInt64(obj.Size),
			IsDir:  strings.HasSuffix(*obj.Key, "/"),
		}

		if obj.ETag != nil {
			o.ETag = strings.Trim(*obj.ETag, "\"")
			o.Hash = storage.Hashes{"etag": o.ETag}
		}

		if obj.LastModified != nil {
			o.Updated = *obj.LastModified
			o.Created = *obj.LastModified
		}

		it.objects = append(it.objects, o)
	}

	// Check if there are more pages
	if resp.IsTruncated != nil && *resp.IsTruncated && resp.NextContinuationToken != nil {
		it.continuationToken = resp.NextContinuationToken
	} else {
		it.done = true
	}

	return nil
}

func (it *objectIter) Close() error {
	it.objects = nil
	it.done = true
	return nil
}

// directory implements storage.Directory for S3.
type directory struct {
	bucket *bucket
	path   string
}

var _ storage.Directory = (*directory)(nil)

func (d *directory) Bucket() storage.Bucket {
	return d.bucket
}

func (d *directory) Path() string {
	return d.path
}

func (d *directory) Info(ctx context.Context) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(d.bucket.name),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(1),
	}

	resp, err := d.bucket.store.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, mapS3Error(err)
	}

	if len(resp.Contents) == 0 && len(resp.CommonPrefixes) == 0 {
		return nil, storage.ErrNotExist
	}

	var created, updated time.Time
	if len(resp.Contents) > 0 && resp.Contents[0].LastModified != nil {
		created = *resp.Contents[0].LastModified
		updated = *resp.Contents[0].LastModified
	} else {
		created = time.Now()
		updated = time.Now()
	}

	return &storage.Object{
		Bucket:   d.bucket.name,
		Key:      d.path,
		IsDir:    true,
		Created:  created,
		Updated:  updated,
		Metadata: map[string]string{},
	}, nil
}

func (d *directory) List(ctx context.Context, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(d.bucket.name),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"), // Non-recursive for directory listing
	}

	return &objectIter{
		client: d.bucket.store.client,
		bucket: d.bucket.name,
		input:  input,
		limit:  limit,
		offset: offset,
	}, nil
}

func (d *directory) Delete(ctx context.Context, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	recursive := boolOpt(opts, "recursive")

	prefix := d.path
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	if !recursive {
		// Non-recursive: only delete if directory is empty
		input := &s3.ListObjectsV2Input{
			Bucket:  aws.String(d.bucket.name),
			Prefix:  aws.String(prefix),
			MaxKeys: aws.Int32(2),
		}

		resp, err := d.bucket.store.client.ListObjectsV2(ctx, input)
		if err != nil {
			return mapS3Error(err)
		}

		if len(resp.Contents) == 0 {
			return storage.ErrNotExist
		}

		// Check if there are nested objects
		hasNested := false
		for _, obj := range resp.Contents {
			key := aws.ToString(obj.Key)
			rest := strings.TrimPrefix(key, prefix)
			if rest != "" && rest != "/" {
				hasNested = true
				break
			}
		}

		if hasNested {
			return storage.ErrPermission
		}

		// Delete the directory marker if exists
		_, err = d.bucket.store.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(d.bucket.name),
			Key:    aws.String(prefix),
		})
		return mapS3Error(err)
	}

	// Recursive delete
	return d.bucket.deleteRecursive(ctx, prefix)
}

func (d *directory) Move(ctx context.Context, dstPath string, opts storage.Options) (storage.Directory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	srcPrefix := d.path
	if srcPrefix != "" && !strings.HasSuffix(srcPrefix, "/") {
		srcPrefix += "/"
	}

	dstPrefix := strings.Trim(dstPath, "/")
	if dstPrefix != "" && !strings.HasSuffix(dstPrefix, "/") {
		dstPrefix += "/"
	}

	// List all objects with source prefix
	paginator := s3.NewListObjectsV2Paginator(d.bucket.store.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(d.bucket.name),
		Prefix: aws.String(srcPrefix),
	})

	found := false
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, mapS3Error(err)
		}

		for _, obj := range page.Contents {
			found = true
			srcKey := aws.ToString(obj.Key)
			rest := strings.TrimPrefix(srcKey, srcPrefix)
			dstKey := dstPrefix + rest

			// Copy to new location
			copySource := d.bucket.name + "/" + srcKey
			_, err := d.bucket.store.client.CopyObject(ctx, &s3.CopyObjectInput{
				Bucket:     aws.String(d.bucket.name),
				Key:        aws.String(dstKey),
				CopySource: aws.String(copySource),
			})
			if err != nil {
				return nil, mapS3Error(err)
			}

			// Delete original
			_, err = d.bucket.store.client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(d.bucket.name),
				Key:    aws.String(srcKey),
			})
			if err != nil {
				return nil, mapS3Error(err)
			}
		}
	}

	if !found {
		return nil, storage.ErrNotExist
	}

	return &directory{
		bucket: d.bucket,
		path:   strings.Trim(dstPath, "/"),
	}, nil
}
