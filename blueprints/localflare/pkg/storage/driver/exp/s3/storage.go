package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-mizu/blueprints/localflare/pkg/storage"
)

// store implements storage.Storage for S3.
type store struct {
	client        *s3.Client
	defaultBucket string
	region        string
	endpoint      string
}

var _ storage.Storage = (*store)(nil)

func (s *store) Bucket(name string) storage.Bucket {
	if name == "" {
		name = s.defaultBucket
	}
	if name == "" {
		name = "default"
	}
	return &bucket{
		store: s,
		name:  name,
	}
}

func (s *store) Buckets(ctx context.Context, limit, offset int, opts storage.Options) (storage.BucketIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	resp, err := s.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, mapS3Error(err)
	}

	infos := make([]*storage.BucketInfo, 0, len(resp.Buckets))
	for _, b := range resp.Buckets {
		info := &storage.BucketInfo{
			Name:     aws.ToString(b.Name),
			Metadata: map[string]string{},
		}
		if b.CreationDate != nil {
			info.CreatedAt = *b.CreationDate
		}
		infos = append(infos, info)
	}

	// Apply pagination
	if offset < 0 {
		offset = 0
	}
	if offset > len(infos) {
		offset = len(infos)
	}
	infos = infos[offset:]

	if limit > 0 && limit < len(infos) {
		infos = infos[:limit]
	}

	return &bucketIter{buckets: infos}, nil
}

func (s *store) CreateBucket(ctx context.Context, name string, opts storage.Options) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("s3: bucket name is empty")
	}

	input := &s3.CreateBucketInput{
		Bucket: aws.String(name),
	}

	// For regions other than us-east-1, we need to specify LocationConstraint
	if s.region != "" && s.region != "us-east-1" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(s.region),
		}
	}

	_, err := s.client.CreateBucket(ctx, input)
	if err != nil {
		return nil, mapS3Error(err)
	}

	return &storage.BucketInfo{
		Name:      name,
		CreatedAt: time.Now(),
		Metadata:  map[string]string{},
	}, nil
}

func (s *store) DeleteBucket(ctx context.Context, name string, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("s3: bucket name is empty")
	}

	force := boolOpt(opts, "force")
	if force {
		// Delete all objects first
		if err := s.emptyBucket(ctx, name); err != nil {
			return err
		}
	}

	_, err := s.client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(name),
	})
	return mapS3Error(err)
}

// emptyBucket deletes all objects in a bucket.
func (s *store) emptyBucket(ctx context.Context, name string) error {
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(name),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return mapS3Error(err)
		}

		if len(page.Contents) == 0 {
			continue
		}

		objects := make([]types.ObjectIdentifier, 0, len(page.Contents))
		for _, obj := range page.Contents {
			objects = append(objects, types.ObjectIdentifier{
				Key: obj.Key,
			})
		}

		_, err = s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(name),
			Delete: &types.Delete{
				Objects: objects,
				Quiet:   aws.Bool(true),
			},
		})
		if err != nil {
			return mapS3Error(err)
		}
	}

	return nil
}

func (s *store) Features() storage.Features {
	return storage.Features{
		"move":              true,
		"server_side_copy":  true,
		"server_side_move":  false, // S3 doesn't have native move
		"directories":       true,
		"public_url":        true,
		"signed_url":        true,
		"multipart":         true,
		"hash:md5":          true,
		"hash:sha256":       true,
		"conditional_write": true,
	}
}

func (s *store) Close() error {
	return nil
}

// bucket implements storage.Bucket for S3.
type bucket struct {
	store *store
	name  string
}

var (
	_ storage.Bucket         = (*bucket)(nil)
	_ storage.HasMultipart   = (*bucket)(nil)
	_ storage.HasDirectories = (*bucket)(nil)
)

func (b *bucket) Name() string {
	return b.name
}

func (b *bucket) Info(ctx context.Context) (*storage.BucketInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Use HeadBucket to verify bucket exists
	_, err := b.store.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(b.name),
	})
	if err != nil {
		return nil, mapS3Error(err)
	}

	return &storage.BucketInfo{
		Name:     b.name,
		Metadata: map[string]string{},
	}, nil
}

func (b *bucket) Features() storage.Features {
	return b.store.Features()
}

func (b *bucket) Write(ctx context.Context, key string, src io.Reader, size int64, contentType string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key = cleanKey(key)
	if key == "" {
		return nil, fmt.Errorf("s3: empty key")
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(b.name),
		Key:    aws.String(key),
		Body:   src,
	}

	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}

	if size >= 0 {
		input.ContentLength = aws.Int64(size)
	}

	// Handle metadata
	if m, ok := opts["metadata"].(map[string]string); ok && len(m) > 0 {
		input.Metadata = m
	}

	// Handle cache control
	if cc, ok := opts["cache_control"].(string); ok && cc != "" {
		input.CacheControl = aws.String(cc)
	}

	// Handle content disposition
	if cd, ok := opts["content_disposition"].(string); ok && cd != "" {
		input.ContentDisposition = aws.String(cd)
	}

	// Handle content encoding
	if ce, ok := opts["content_encoding"].(string); ok && ce != "" {
		input.ContentEncoding = aws.String(ce)
	}

	resp, err := b.store.client.PutObject(ctx, input)
	if err != nil {
		return nil, mapS3Error(err)
	}

	// Get object info after upload
	obj := &storage.Object{
		Bucket:      b.name,
		Key:         key,
		ContentType: contentType,
		Updated:     time.Now(),
		Created:     time.Now(),
	}

	if resp.ETag != nil {
		obj.ETag = strings.Trim(*resp.ETag, "\"")
		obj.Hash = storage.Hashes{"etag": obj.ETag}
	}

	if size >= 0 {
		obj.Size = size
	}

	if m, ok := opts["metadata"].(map[string]string); ok {
		obj.Metadata = m
	}

	return obj, nil
}

func (b *bucket) Open(ctx context.Context, key string, offset, length int64, opts storage.Options) (io.ReadCloser, *storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	key = cleanKey(key)
	if key == "" {
		return nil, nil, fmt.Errorf("s3: empty key")
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(b.name),
		Key:    aws.String(key),
	}

	// Handle range requests
	if offset > 0 || length > 0 {
		var rangeStr string
		if length > 0 {
			rangeStr = fmt.Sprintf("bytes=%d-%d", offset, offset+length-1)
		} else if length < 0 {
			// From offset to end
			rangeStr = fmt.Sprintf("bytes=%d-", offset)
		} else {
			// length == 0, offset > 0: from offset to end
			rangeStr = fmt.Sprintf("bytes=%d-", offset)
		}
		input.Range = aws.String(rangeStr)
	}

	// Handle version
	if v, ok := opts["version"].(string); ok && v != "" {
		input.VersionId = aws.String(v)
	}

	resp, err := b.store.client.GetObject(ctx, input)
	if err != nil {
		return nil, nil, mapS3Error(err)
	}

	obj := &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        aws.ToInt64(resp.ContentLength),
		ContentType: aws.ToString(resp.ContentType),
		Metadata:    resp.Metadata,
	}

	if resp.ETag != nil {
		obj.ETag = strings.Trim(*resp.ETag, "\"")
		obj.Hash = storage.Hashes{"etag": obj.ETag}
	}

	if resp.LastModified != nil {
		obj.Updated = *resp.LastModified
		obj.Created = *resp.LastModified
	}

	if resp.VersionId != nil {
		obj.Version = *resp.VersionId
	}

	return resp.Body, obj, nil
}

func (b *bucket) Stat(ctx context.Context, key string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key = cleanKey(key)
	if key == "" {
		return nil, fmt.Errorf("s3: empty key")
	}

	// Check if this is a directory request
	if strings.HasSuffix(key, "/") {
		return b.statDirectory(ctx, key)
	}

	input := &s3.HeadObjectInput{
		Bucket: aws.String(b.name),
		Key:    aws.String(key),
	}

	if v, ok := opts["version"].(string); ok && v != "" {
		input.VersionId = aws.String(v)
	}

	resp, err := b.store.client.HeadObject(ctx, input)
	if err != nil {
		return nil, mapS3Error(err)
	}

	obj := &storage.Object{
		Bucket:      b.name,
		Key:         key,
		Size:        aws.ToInt64(resp.ContentLength),
		ContentType: aws.ToString(resp.ContentType),
		Metadata:    resp.Metadata,
	}

	if resp.ETag != nil {
		obj.ETag = strings.Trim(*resp.ETag, "\"")
		obj.Hash = storage.Hashes{"etag": obj.ETag}
	}

	if resp.LastModified != nil {
		obj.Updated = *resp.LastModified
		obj.Created = *resp.LastModified
	}

	if resp.VersionId != nil {
		obj.Version = *resp.VersionId
	}

	return obj, nil
}

// statDirectory checks if a directory (prefix) exists.
func (b *bucket) statDirectory(ctx context.Context, prefix string) (*storage.Object, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(b.name),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(1),
	}

	resp, err := b.store.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, mapS3Error(err)
	}

	if len(resp.Contents) == 0 && len(resp.CommonPrefixes) == 0 {
		return nil, storage.ErrNotExist
	}

	return &storage.Object{
		Bucket:  b.name,
		Key:     strings.TrimSuffix(prefix, "/"),
		IsDir:   true,
		Updated: time.Now(),
	}, nil
}

func (b *bucket) Delete(ctx context.Context, key string, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	key = cleanKey(key)
	if key == "" {
		return fmt.Errorf("s3: empty key")
	}

	recursive := boolOpt(opts, "recursive")
	if recursive && strings.HasSuffix(key, "/") {
		return b.deleteRecursive(ctx, key)
	}

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(b.name),
		Key:    aws.String(key),
	}

	if v, ok := opts["version"].(string); ok && v != "" {
		input.VersionId = aws.String(v)
	}

	_, err := b.store.client.DeleteObject(ctx, input)
	return mapS3Error(err)
}

// deleteRecursive deletes all objects with the given prefix.
func (b *bucket) deleteRecursive(ctx context.Context, prefix string) error {
	paginator := s3.NewListObjectsV2Paginator(b.store.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(b.name),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return mapS3Error(err)
		}

		if len(page.Contents) == 0 {
			continue
		}

		objects := make([]types.ObjectIdentifier, 0, len(page.Contents))
		for _, obj := range page.Contents {
			objects = append(objects, types.ObjectIdentifier{
				Key: obj.Key,
			})
		}

		_, err = b.store.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(b.name),
			Delete: &types.Delete{
				Objects: objects,
				Quiet:   aws.Bool(true),
			},
		})
		if err != nil {
			return mapS3Error(err)
		}
	}

	return nil
}

func (b *bucket) Copy(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	dstKey = cleanKey(dstKey)
	srcKey = cleanKey(srcKey)

	if dstKey == "" {
		return nil, fmt.Errorf("s3: empty destination key")
	}
	if srcKey == "" {
		return nil, fmt.Errorf("s3: empty source key")
	}

	if srcBucket == "" {
		srcBucket = b.name
	}

	copySource := fmt.Sprintf("%s/%s", srcBucket, srcKey)

	input := &s3.CopyObjectInput{
		Bucket:     aws.String(b.name),
		Key:        aws.String(dstKey),
		CopySource: aws.String(copySource),
	}

	// Handle metadata
	if m, ok := opts["metadata"].(map[string]string); ok && len(m) > 0 {
		input.Metadata = m
		input.MetadataDirective = types.MetadataDirectiveReplace
	}

	resp, err := b.store.client.CopyObject(ctx, input)
	if err != nil {
		return nil, mapS3Error(err)
	}

	obj := &storage.Object{
		Bucket: b.name,
		Key:    dstKey,
	}

	if resp.CopyObjectResult != nil {
		if resp.CopyObjectResult.ETag != nil {
			obj.ETag = strings.Trim(*resp.CopyObjectResult.ETag, "\"")
			obj.Hash = storage.Hashes{"etag": obj.ETag}
		}
		if resp.CopyObjectResult.LastModified != nil {
			obj.Updated = *resp.CopyObjectResult.LastModified
			obj.Created = *resp.CopyObjectResult.LastModified
		}
	}

	return obj, nil
}

func (b *bucket) Move(ctx context.Context, dstKey string, srcBucket, srcKey string, opts storage.Options) (*storage.Object, error) {
	// S3 doesn't have native move, so copy then delete
	obj, err := b.Copy(ctx, dstKey, srcBucket, srcKey, opts)
	if err != nil {
		return nil, err
	}

	// Delete source
	if srcBucket == "" {
		srcBucket = b.name
	}

	_, err = b.store.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(srcBucket),
		Key:    aws.String(srcKey),
	})
	if err != nil {
		return nil, mapS3Error(err)
	}

	return obj, nil
}

func (b *bucket) List(ctx context.Context, prefix string, limit, offset int, opts storage.Options) (storage.ObjectIter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	prefix = cleanPrefix(prefix)

	recursive := true
	if v, ok := opts["recursive"].(bool); ok {
		recursive = v
	}

	dirsOnly := boolOpt(opts, "dirs_only")
	filesOnly := boolOpt(opts, "files_only")

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(b.name),
	}

	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	if !recursive {
		input.Delimiter = aws.String("/")
	}

	return &objectIter{
		client:    b.store.client,
		bucket:    b.name,
		input:     input,
		limit:     limit,
		offset:    offset,
		dirsOnly:  dirsOnly,
		filesOnly: filesOnly,
	}, nil
}

func (b *bucket) SignedURL(ctx context.Context, key string, method string, expires time.Duration, opts storage.Options) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	key = cleanKey(key)
	if key == "" {
		return "", fmt.Errorf("s3: empty key")
	}

	presignClient := s3.NewPresignClient(b.store.client)

	switch strings.ToUpper(method) {
	case "GET":
		input := &s3.GetObjectInput{
			Bucket: aws.String(b.name),
			Key:    aws.String(key),
		}
		resp, err := presignClient.PresignGetObject(ctx, input, s3.WithPresignExpires(expires))
		if err != nil {
			return "", mapS3Error(err)
		}
		return resp.URL, nil

	case "PUT":
		input := &s3.PutObjectInput{
			Bucket: aws.String(b.name),
			Key:    aws.String(key),
		}
		if ct, ok := opts["content_type"].(string); ok && ct != "" {
			input.ContentType = aws.String(ct)
		}
		resp, err := presignClient.PresignPutObject(ctx, input, s3.WithPresignExpires(expires))
		if err != nil {
			return "", mapS3Error(err)
		}
		return resp.URL, nil

	case "DELETE":
		input := &s3.DeleteObjectInput{
			Bucket: aws.String(b.name),
			Key:    aws.String(key),
		}
		resp, err := presignClient.PresignDeleteObject(ctx, input, s3.WithPresignExpires(expires))
		if err != nil {
			return "", mapS3Error(err)
		}
		return resp.URL, nil

	default:
		return "", fmt.Errorf("s3: unsupported method %q for signed URL", method)
	}
}

// Directory implements storage.HasDirectories
func (b *bucket) Directory(p string) storage.Directory {
	p = strings.Trim(p, "/")
	return &directory{
		bucket: b,
		path:   p,
	}
}

// Helpers

// cleanKey normalizes an object key.
func cleanKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.ReplaceAll(key, "\\", "/")
	key = path.Clean(key)
	key = strings.TrimPrefix(key, "/")
	if key == "." {
		return ""
	}
	return key
}

// cleanPrefix normalizes a prefix for listing.
func cleanPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	prefix = strings.ReplaceAll(prefix, "\\", "/")
	prefix = strings.TrimPrefix(prefix, "/")
	return prefix
}

// boolOpt extracts a boolean option.
func boolOpt(opts storage.Options, key string) bool {
	if opts == nil {
		return false
	}
	v, ok := opts[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

// mapS3Error maps S3 errors to storage errors.
func mapS3Error(err error) error {
	if err == nil {
		return nil
	}

	// Check for specific S3 error types
	var notFound *types.NotFound
	var noSuchKey *types.NoSuchKey
	var noSuchBucket *types.NoSuchBucket
	var bucketAlreadyExists *types.BucketAlreadyExists
	var bucketAlreadyOwned *types.BucketAlreadyOwnedByYou

	if errors.As(err, &notFound) {
		return storage.ErrNotExist
	}
	if errors.As(err, &noSuchKey) {
		return storage.ErrNotExist
	}
	if errors.As(err, &noSuchBucket) {
		return storage.ErrNotExist
	}
	if errors.As(err, &bucketAlreadyExists) {
		return storage.ErrExist
	}
	if errors.As(err, &bucketAlreadyOwned) {
		return storage.ErrExist
	}

	// Check error message for common patterns
	errStr := err.Error()
	if strings.Contains(errStr, "NoSuchKey") ||
		strings.Contains(errStr, "NotFound") ||
		strings.Contains(errStr, "404") {
		return storage.ErrNotExist
	}
	if strings.Contains(errStr, "AccessDenied") ||
		strings.Contains(errStr, "Forbidden") ||
		strings.Contains(errStr, "403") {
		return storage.ErrPermission
	}
	if strings.Contains(errStr, "BucketAlreadyExists") ||
		strings.Contains(errStr, "BucketAlreadyOwnedByYou") {
		return storage.ErrExist
	}
	if strings.Contains(errStr, "BucketNotEmpty") {
		return storage.ErrPermission
	}

	return fmt.Errorf("s3: %w", err)
}
