package s3

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-mizu/blueprints/localflare/pkg/storage"
)

// InitMultipart starts a multipart upload for the given key.
func (b *bucket) InitMultipart(ctx context.Context, key string, contentType string, opts storage.Options) (*storage.MultipartUpload, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key = cleanKey(key)
	if key == "" {
		return nil, fmt.Errorf("s3: empty key")
	}

	input := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(b.name),
		Key:    aws.String(key),
	}

	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}

	// Handle metadata
	meta := map[string]string{}
	if m, ok := opts["metadata"].(map[string]string); ok && len(m) > 0 {
		meta = m
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

	resp, err := b.store.client.CreateMultipartUpload(ctx, input)
	if err != nil {
		return nil, mapS3Error(err)
	}

	return &storage.MultipartUpload{
		Bucket:   b.name,
		Key:      key,
		UploadID: aws.ToString(resp.UploadId),
		Metadata: meta,
	}, nil
}

// UploadPart uploads a single part for an existing multipart upload.
func (b *bucket) UploadPart(ctx context.Context, mu *storage.MultipartUpload, number int, src io.Reader, size int64, opts storage.Options) (*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if number <= 0 || number > 10000 {
		return nil, fmt.Errorf("s3: part number %d out of range (1-10000)", number)
	}

	input := &s3.UploadPartInput{
		Bucket:     aws.String(mu.Bucket),
		Key:        aws.String(mu.Key),
		UploadId:   aws.String(mu.UploadID),
		PartNumber: aws.Int32(int32(number)),
		Body:       src,
	}

	if size >= 0 {
		input.ContentLength = aws.Int64(size)
	}

	resp, err := b.store.client.UploadPart(ctx, input)
	if err != nil {
		return nil, mapS3Error(err)
	}

	part := &storage.PartInfo{
		Number: number,
		Size:   size,
	}

	if resp.ETag != nil {
		part.ETag = strings.Trim(*resp.ETag, "\"")
	}

	return part, nil
}

// CopyPart uploads a single part by copying a range from an existing source object.
func (b *bucket) CopyPart(ctx context.Context, mu *storage.MultipartUpload, number int, opts storage.Options) (*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if number <= 0 || number > 10000 {
		return nil, fmt.Errorf("s3: part number %d out of range (1-10000)", number)
	}

	// Get source parameters from options
	srcBucket, _ := opts["source_bucket"].(string)
	srcKey, _ := opts["source_key"].(string)
	srcOffset, _ := opts["source_offset"].(int64)
	srcLength, _ := opts["source_length"].(int64)

	if srcBucket == "" {
		srcBucket = mu.Bucket
	}
	if srcKey == "" {
		return nil, fmt.Errorf("s3: source_key is required for CopyPart")
	}

	copySource := fmt.Sprintf("%s/%s", srcBucket, srcKey)

	input := &s3.UploadPartCopyInput{
		Bucket:     aws.String(mu.Bucket),
		Key:        aws.String(mu.Key),
		UploadId:   aws.String(mu.UploadID),
		PartNumber: aws.Int32(int32(number)),
		CopySource: aws.String(copySource),
	}

	// Handle range if specified
	if srcOffset >= 0 || srcLength > 0 {
		var rangeStr string
		if srcLength > 0 {
			rangeStr = fmt.Sprintf("bytes=%d-%d", srcOffset, srcOffset+srcLength-1)
		} else {
			rangeStr = fmt.Sprintf("bytes=%d-", srcOffset)
		}
		input.CopySourceRange = aws.String(rangeStr)
	}

	resp, err := b.store.client.UploadPartCopy(ctx, input)
	if err != nil {
		return nil, mapS3Error(err)
	}

	part := &storage.PartInfo{
		Number: number,
	}

	if resp.CopyPartResult != nil {
		if resp.CopyPartResult.ETag != nil {
			part.ETag = strings.Trim(*resp.CopyPartResult.ETag, "\"")
		}
		if resp.CopyPartResult.LastModified != nil {
			part.LastModified = resp.CopyPartResult.LastModified
		}
	}

	return part, nil
}

// ListParts lists already uploaded parts for a multipart upload.
func (b *bucket) ListParts(ctx context.Context, mu *storage.MultipartUpload, limit, offset int, opts storage.Options) ([]*storage.PartInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	input := &s3.ListPartsInput{
		Bucket:   aws.String(mu.Bucket),
		Key:      aws.String(mu.Key),
		UploadId: aws.String(mu.UploadID),
	}

	if limit > 0 && limit <= 1000 {
		input.MaxParts = aws.Int32(int32(limit + offset))
	}

	resp, err := b.store.client.ListParts(ctx, input)
	if err != nil {
		return nil, mapS3Error(err)
	}

	parts := make([]*storage.PartInfo, 0, len(resp.Parts))
	for _, p := range resp.Parts {
		part := &storage.PartInfo{
			Number: int(aws.ToInt32(p.PartNumber)),
			Size:   aws.ToInt64(p.Size),
		}
		if p.ETag != nil {
			part.ETag = strings.Trim(*p.ETag, "\"")
		}
		if p.LastModified != nil {
			part.LastModified = p.LastModified
		}
		parts = append(parts, part)
	}

	// Apply pagination
	if offset < 0 {
		offset = 0
	}
	if offset > len(parts) {
		offset = len(parts)
	}
	parts = parts[offset:]

	if limit > 0 && limit < len(parts) {
		parts = parts[:limit]
	}

	return parts, nil
}

// CompleteMultipart completes a multipart upload and assembles the final object.
func (b *bucket) CompleteMultipart(ctx context.Context, mu *storage.MultipartUpload, parts []*storage.PartInfo, opts storage.Options) (*storage.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if len(parts) == 0 {
		return nil, fmt.Errorf("s3: no parts to complete")
	}

	// Sort parts by number
	sortedParts := make([]*storage.PartInfo, len(parts))
	copy(sortedParts, parts)
	sort.Slice(sortedParts, func(i, j int) bool {
		return sortedParts[i].Number < sortedParts[j].Number
	})

	// Build completed parts list
	completedParts := make([]types.CompletedPart, 0, len(sortedParts))
	var totalSize int64
	for _, p := range sortedParts {
		cp := types.CompletedPart{
			PartNumber: aws.Int32(int32(p.Number)),
		}
		if p.ETag != "" {
			cp.ETag = aws.String(p.ETag)
		}
		completedParts = append(completedParts, cp)
		totalSize += p.Size
	}

	input := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(mu.Bucket),
		Key:      aws.String(mu.Key),
		UploadId: aws.String(mu.UploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	}

	resp, err := b.store.client.CompleteMultipartUpload(ctx, input)
	if err != nil {
		return nil, mapS3Error(err)
	}

	obj := &storage.Object{
		Bucket:   b.name,
		Key:      mu.Key,
		Size:     totalSize,
		Metadata: mu.Metadata,
	}

	if resp.ETag != nil {
		obj.ETag = strings.Trim(*resp.ETag, "\"")
		obj.Hash = storage.Hashes{"etag": obj.ETag}
	}

	if resp.VersionId != nil {
		obj.Version = *resp.VersionId
	}

	return obj, nil
}

// AbortMultipart aborts the multipart upload and discards all parts.
func (b *bucket) AbortMultipart(ctx context.Context, mu *storage.MultipartUpload, opts storage.Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	_, err := b.store.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(mu.Bucket),
		Key:      aws.String(mu.Key),
		UploadId: aws.String(mu.UploadID),
	})
	if err != nil {
		// Treat "not found" as success for abort
		if mapS3Error(err) == storage.ErrNotExist {
			return nil
		}
		return mapS3Error(err)
	}

	return nil
}
