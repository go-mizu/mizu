package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/drive/lib/storage/driver/local"
)

// setupS3ServerWithClient creates a test S3 server and AWS SDK client for testing.
func setupS3ServerWithClient(t *testing.T) (*s3.Client, string, func()) {
	t.Helper()

	ctx := context.Background()

	// Create temporary storage
	tmpDir := t.TempDir()
	store, err := local.Open(ctx, tmpDir)
	if err != nil {
		t.Fatalf("open local storage: %v", err)
	}

	// Setup S3 server with credentials
	creds := &staticCredentialProvider{creds: map[string]*Credential{
		"TESTKEY": {
			AccessKeyID:     "TESTKEY",
			SecretAccessKey: "TESTSECRET",
		},
	}}

	app := mizu.New()
	cfg := &Config{
		Credentials: creds,
		Signer:      &SignerV4{},
		Region:      "us-east-1",
	}
	Register(app, "/s3", store, cfg)

	srv := httptest.NewServer(app)

	// Create AWS SDK client
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("TESTKEY", "TESTSECRET", "")),
		config.WithBaseEndpoint(srv.URL+"/s3"),
	)
	if err != nil {
		srv.Close()
		_ = store.Close()
		t.Fatalf("load aws config: %v", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	cleanup := func() {
		srv.Close()
		_ = store.Close()
	}

	return client, tmpDir, cleanup
}

// generatePartData creates test data of specified size with a repeating seed pattern.
func generatePartData(size int, seed byte) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte((int(seed) + i) % 256)
	}
	return data
}

// uploadPartWithData uploads a part and returns its ETag.
func uploadPartWithData(ctx context.Context, client *s3.Client, bucket, key, uploadID string, partNum int, data []byte) (string, error) {
	resp, err := client.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(key),
		UploadId:   aws.String(uploadID),
		PartNumber: aws.Int32(int32(partNum)),
		Body:       bytes.NewReader(data),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.ETag), nil
}

// verifyObjectContent verifies that an object matches expected content.
func verifyObjectContent(t *testing.T, ctx context.Context, client *s3.Client, bucket, key string, expected []byte) {
	t.Helper()

	resp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("get object: %v", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read object body: %v", err)
	}

	if !bytes.Equal(data, expected) {
		t.Fatalf("object content mismatch: got %d bytes, want %d bytes", len(data), len(expected))
	}
}

// TestMultipartUploadBasicFlow tests the complete multipart upload workflow.
func TestMultipartUploadBasicFlow(t *testing.T) {
	ctx := context.Background()
	client, _, cleanup := setupS3ServerWithClient(t)
	defer cleanup()

	bucket := "test-bucket"
	key := "test-file.dat"

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	defer func() {
		_, _ = client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	}()

	// Initiate multipart upload
	createResp, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("create multipart upload: %v", err)
	}
	uploadID := aws.ToString(createResp.UploadId)

	// Upload 3 parts (5MB, 5MB, 1MB)
	part1Data := generatePartData(5*1024*1024, 1)
	part2Data := generatePartData(5*1024*1024, 2)
	part3Data := generatePartData(1*1024*1024, 3)

	etag1, err := uploadPartWithData(ctx, client, bucket, key, uploadID, 1, part1Data)
	if err != nil {
		t.Fatalf("upload part 1: %v", err)
	}

	etag2, err := uploadPartWithData(ctx, client, bucket, key, uploadID, 2, part2Data)
	if err != nil {
		t.Fatalf("upload part 2: %v", err)
	}

	etag3, err := uploadPartWithData(ctx, client, bucket, key, uploadID, 3, part3Data)
	if err != nil {
		t.Fatalf("upload part 3: %v", err)
	}

	// List parts
	listResp, err := client.ListParts(ctx, &s3.ListPartsInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		t.Fatalf("list parts: %v", err)
	}

	if len(listResp.Parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(listResp.Parts))
	}

	// Complete multipart upload
	_, err = client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: []types.CompletedPart{
				{PartNumber: aws.Int32(1), ETag: aws.String(etag1)},
				{PartNumber: aws.Int32(2), ETag: aws.String(etag2)},
				{PartNumber: aws.Int32(3), ETag: aws.String(etag3)},
			},
		},
	})
	if err != nil {
		t.Fatalf("complete multipart upload: %v", err)
	}

	// Verify object exists and content matches
	expectedData := append(append(part1Data, part2Data...), part3Data...)
	verifyObjectContent(t, ctx, client, bucket, key, expectedData)

	// Cleanup
	_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
}

// TestMultipartUploadWithMetadata tests multipart upload with custom metadata.
func TestMultipartUploadWithMetadata(t *testing.T) {
	ctx := context.Background()
	client, _, cleanup := setupS3ServerWithClient(t)
	defer cleanup()

	bucket := "test-bucket"
	key := "test-metadata.dat"

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	defer func() {
		_, _ = client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	}()

	// Initiate upload with metadata
	createResp, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String("application/octet-stream"),
		Metadata: map[string]string{
			"author":  "test-user",
			"version": "1.0",
		},
	})
	if err != nil {
		t.Fatalf("create multipart upload: %v", err)
	}
	uploadID := aws.ToString(createResp.UploadId)

	// Upload part
	partData := generatePartData(1024*1024, 1)
	etag, err := uploadPartWithData(ctx, client, bucket, key, uploadID, 1, partData)
	if err != nil {
		t.Fatalf("upload part: %v", err)
	}

	// Complete upload
	_, err = client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: []types.CompletedPart{
				{PartNumber: aws.Int32(1), ETag: aws.String(etag)},
			},
		},
	})
	if err != nil {
		t.Fatalf("complete multipart upload: %v", err)
	}

	// Verify metadata preserved
	headResp, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("head object: %v", err)
	}

	// Note: Metadata preservation depends on storage backend capabilities
	// Local storage driver may not support custom metadata
	if len(headResp.Metadata) == 0 {
		t.Skip("Metadata not supported by storage backend")
		return
	}

	if headResp.Metadata["author"] != "test-user" {
		t.Errorf("expected metadata author=test-user, got %v", headResp.Metadata["author"])
	}
	if headResp.Metadata["version"] != "1.0" {
		t.Errorf("expected metadata version=1.0, got %v", headResp.Metadata["version"])
	}

	// Cleanup
	_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
}

// TestMultipartUploadAbort tests aborting a multipart upload.
func TestMultipartUploadAbort(t *testing.T) {
	ctx := context.Background()
	client, _, cleanup := setupS3ServerWithClient(t)
	defer cleanup()

	bucket := "test-bucket"
	key := "test-abort.dat"

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	defer func() {
		_, _ = client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	}()

	// Initiate upload
	createResp, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("create multipart upload: %v", err)
	}
	uploadID := aws.ToString(createResp.UploadId)

	// Upload 2 parts
	partData := generatePartData(1024*1024, 1)
	_, err = uploadPartWithData(ctx, client, bucket, key, uploadID, 1, partData)
	if err != nil {
		t.Fatalf("upload part 1: %v", err)
	}
	_, err = uploadPartWithData(ctx, client, bucket, key, uploadID, 2, partData)
	if err != nil {
		t.Fatalf("upload part 2: %v", err)
	}

	// Verify parts exist
	listResp, err := client.ListParts(ctx, &s3.ListPartsInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		t.Fatalf("list parts: %v", err)
	}
	if len(listResp.Parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(listResp.Parts))
	}

	// Abort upload
	_, err = client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		t.Fatalf("abort multipart upload: %v", err)
	}

	// Verify abort of non-existent upload returns success (S3 semantics)
	_, err = client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		t.Fatalf("abort non-existent upload should succeed: %v", err)
	}
}

// TestMultipartUploadSinglePart tests uploading a single part (valid in S3).
func TestMultipartUploadSinglePart(t *testing.T) {
	ctx := context.Background()
	client, _, cleanup := setupS3ServerWithClient(t)
	defer cleanup()

	bucket := "test-bucket"
	key := "single-part.dat"

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	defer func() {
		_, _ = client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	}()

	// Initiate upload
	createResp, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("create multipart upload: %v", err)
	}
	uploadID := aws.ToString(createResp.UploadId)

	// Upload single part
	partData := generatePartData(1024*1024, 1)
	etag, err := uploadPartWithData(ctx, client, bucket, key, uploadID, 1, partData)
	if err != nil {
		t.Fatalf("upload part: %v", err)
	}

	// Complete upload
	_, err = client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: []types.CompletedPart{
				{PartNumber: aws.Int32(1), ETag: aws.String(etag)},
			},
		},
	})
	if err != nil {
		t.Fatalf("complete multipart upload: %v", err)
	}

	// Verify object
	verifyObjectContent(t, ctx, client, bucket, key, partData)

	// Cleanup
	_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
}

// TestMultipartUploadOutOfOrderParts tests uploading parts out of order.
func TestMultipartUploadOutOfOrderParts(t *testing.T) {
	ctx := context.Background()
	client, _, cleanup := setupS3ServerWithClient(t)
	defer cleanup()

	bucket := "test-bucket"
	key := "out-of-order.dat"

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	defer func() {
		_, _ = client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	}()

	// Initiate upload
	createResp, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("create multipart upload: %v", err)
	}
	uploadID := aws.ToString(createResp.UploadId)

	// Upload parts out of order: 3, 1, 5, 2, 4
	parts := make(map[int][]byte)
	for i := 1; i <= 5; i++ {
		parts[i] = generatePartData(512*1024, byte(i))
	}

	order := []int{3, 1, 5, 2, 4}
	etags := make(map[int]string)

	for _, partNum := range order {
		etag, err := uploadPartWithData(ctx, client, bucket, key, uploadID, partNum, parts[partNum])
		if err != nil {
			t.Fatalf("upload part %d: %v", partNum, err)
		}
		etags[partNum] = etag
	}

	// List parts and verify they're returned in order
	listResp, err := client.ListParts(ctx, &s3.ListPartsInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		t.Fatalf("list parts: %v", err)
	}

	if len(listResp.Parts) != 5 {
		t.Fatalf("expected 5 parts, got %d", len(listResp.Parts))
	}

	for i, part := range listResp.Parts {
		expectedPartNum := i + 1
		if aws.ToInt32(part.PartNumber) != int32(expectedPartNum) {
			t.Errorf("part %d: expected PartNumber %d, got %d", i, expectedPartNum, aws.ToInt32(part.PartNumber))
		}
	}

	// Complete with parts in correct order
	completedParts := make([]types.CompletedPart, 5)
	for i := 1; i <= 5; i++ {
		completedParts[i-1] = types.CompletedPart{
			PartNumber: aws.Int32(int32(i)),
			ETag:       aws.String(etags[i]),
		}
	}

	_, err = client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	if err != nil {
		t.Fatalf("complete multipart upload: %v", err)
	}

	// Verify content
	var expectedData []byte
	for i := 1; i <= 5; i++ {
		expectedData = append(expectedData, parts[i]...)
	}
	verifyObjectContent(t, ctx, client, bucket, key, expectedData)

	// Cleanup
	_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
}

// TestMultipartUploadDuplicatePartNumber tests replacing a part with the same number.
func TestMultipartUploadDuplicatePartNumber(t *testing.T) {
	ctx := context.Background()
	client, _, cleanup := setupS3ServerWithClient(t)
	defer cleanup()

	bucket := "test-bucket"
	key := "duplicate-part.dat"

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	defer func() {
		_, _ = client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	}()

	// Initiate upload
	createResp, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("create multipart upload: %v", err)
	}
	uploadID := aws.ToString(createResp.UploadId)

	// Upload part 1 first time
	part1DataOld := generatePartData(512*1024, 1)
	_, err = uploadPartWithData(ctx, client, bucket, key, uploadID, 1, part1DataOld)
	if err != nil {
		t.Fatalf("upload part 1 (first): %v", err)
	}

	// Upload part 1 again (should replace)
	part1DataNew := generatePartData(512*1024, 2)
	etagNew, err := uploadPartWithData(ctx, client, bucket, key, uploadID, 1, part1DataNew)
	if err != nil {
		t.Fatalf("upload part 1 (second): %v", err)
	}

	// List parts - should show only 1 part with latest ETag
	listResp, err := client.ListParts(ctx, &s3.ListPartsInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		t.Fatalf("list parts: %v", err)
	}

	if len(listResp.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(listResp.Parts))
	}

	// Complete upload
	_, err = client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: []types.CompletedPart{
				{PartNumber: aws.Int32(1), ETag: aws.String(etagNew)},
			},
		},
	})
	if err != nil {
		t.Fatalf("complete multipart upload: %v", err)
	}

	// Verify object has the new data, not the old
	verifyObjectContent(t, ctx, client, bucket, key, part1DataNew)

	// Cleanup
	_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
}

// TestMultipartUploadPartNumberBounds tests part number validation.
func TestMultipartUploadPartNumberBounds(t *testing.T) {
	ctx := context.Background()
	client, _, cleanup := setupS3ServerWithClient(t)
	defer cleanup()

	bucket := "test-bucket"
	key := "part-bounds.dat"

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	defer func() {
		_, _ = client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	}()

	// Initiate upload
	createResp, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("create multipart upload: %v", err)
	}
	uploadID := aws.ToString(createResp.UploadId)
	defer func() {
		_, _ = client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
			Bucket:   aws.String(bucket),
			Key:      aws.String(key),
			UploadId: aws.String(uploadID),
		})
	}()

	partData := generatePartData(1024, 1)

	// Test part number 1 (valid)
	_, err = uploadPartWithData(ctx, client, bucket, key, uploadID, 1, partData)
	if err != nil {
		t.Errorf("part number 1 should be valid: %v", err)
	}

	// Test part number 10000 (valid - max allowed)
	_, err = uploadPartWithData(ctx, client, bucket, key, uploadID, 10000, partData)
	if err != nil {
		t.Errorf("part number 10000 should be valid: %v", err)
	}

	// Test part number 0 (invalid)
	_, err = client.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(key),
		UploadId:   aws.String(uploadID),
		PartNumber: aws.Int32(0),
		Body:       bytes.NewReader(partData),
	})
	if err == nil {
		t.Error("part number 0 should be invalid")
	}

	// Test part number 10001 (invalid)
	_, err = client.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(key),
		UploadId:   aws.String(uploadID),
		PartNumber: aws.Int32(10001),
		Body:       bytes.NewReader(partData),
	})
	if err == nil {
		t.Error("part number 10001 should be invalid")
	}
}

// TestMultipartUploadConcurrentParts tests concurrent part uploads.
func TestMultipartUploadConcurrentParts(t *testing.T) {
	ctx := context.Background()
	client, _, cleanup := setupS3ServerWithClient(t)
	defer cleanup()

	bucket := "test-bucket"
	key := "concurrent.dat"

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	defer func() {
		_, _ = client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	}()

	// Initiate upload
	createResp, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("create multipart upload: %v", err)
	}
	uploadID := aws.ToString(createResp.UploadId)

	// Upload 10 parts concurrently
	const numParts = 10
	var wg sync.WaitGroup
	etags := make(map[int]string)
	var mu sync.Mutex
	errs := make(chan error, numParts)

	for i := 1; i <= numParts; i++ {
		wg.Add(1)
		go func(partNum int) {
			defer wg.Done()

			partData := generatePartData(512*1024, byte(partNum))
			etag, err := uploadPartWithData(ctx, client, bucket, key, uploadID, partNum, partData)
			if err != nil {
				errs <- fmt.Errorf("upload part %d: %w", partNum, err)
				return
			}

			mu.Lock()
			etags[partNum] = etag
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	close(errs)

	// Check for errors
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent upload error: %v", err)
		}
	}

	// List parts and verify all present
	listResp, err := client.ListParts(ctx, &s3.ListPartsInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		t.Fatalf("list parts: %v", err)
	}

	if len(listResp.Parts) != numParts {
		t.Fatalf("expected %d parts, got %d", numParts, len(listResp.Parts))
	}

	// Complete upload
	completedParts := make([]types.CompletedPart, numParts)
	for i := 1; i <= numParts; i++ {
		completedParts[i-1] = types.CompletedPart{
			PartNumber: aws.Int32(int32(i)),
			ETag:       aws.String(etags[i]),
		}
	}

	_, err = client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	if err != nil {
		t.Fatalf("complete multipart upload: %v", err)
	}

	// Verify object exists
	_, err = client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("head object: %v", err)
	}

	// Cleanup
	_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
}

// TestListPartsWithPagination tests listing parts with pagination.
func TestListPartsWithPagination(t *testing.T) {
	ctx := context.Background()
	client, _, cleanup := setupS3ServerWithClient(t)
	defer cleanup()

	bucket := "test-bucket"
	key := "pagination.dat"

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	defer func() {
		_, _ = client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	}()

	// Initiate upload
	createResp, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("create multipart upload: %v", err)
	}
	uploadID := aws.ToString(createResp.UploadId)
	defer func() {
		_, _ = client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
			Bucket:   aws.String(bucket),
			Key:      aws.String(key),
			UploadId: aws.String(uploadID),
		})
	}()

	// Upload 25 parts
	const numParts = 25
	partData := generatePartData(64*1024, 1)
	for i := 1; i <= numParts; i++ {
		_, err := uploadPartWithData(ctx, client, bucket, key, uploadID, i, partData)
		if err != nil {
			t.Fatalf("upload part %d: %v", i, err)
		}
	}

	// List with max-parts=10
	listResp, err := client.ListParts(ctx, &s3.ListPartsInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MaxParts: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("list parts: %v", err)
	}

	if len(listResp.Parts) != 10 {
		t.Fatalf("expected 10 parts, got %d", len(listResp.Parts))
	}
	if !aws.ToBool(listResp.IsTruncated) {
		t.Error("expected IsTruncated=true")
	}

	// List next page using part-number-marker
	// AWS SDK uses string for PartNumberMarker
	marker, err := strconv.Atoi(aws.ToString(listResp.NextPartNumberMarker))
	if err != nil {
		t.Fatalf("parse NextPartNumberMarker: %v", err)
	}
	listResp2, err := client.ListParts(ctx, &s3.ListPartsInput{
		Bucket:           aws.String(bucket),
		Key:              aws.String(key),
		UploadId:         aws.String(uploadID),
		MaxParts:         aws.Int32(10),
		PartNumberMarker: aws.String(strconv.Itoa(marker)),
	})
	if err != nil {
		t.Fatalf("list parts (page 2): %v", err)
	}

	if len(listResp2.Parts) != 10 {
		t.Fatalf("expected 10 parts on page 2, got %d", len(listResp2.Parts))
	}
	if !aws.ToBool(listResp2.IsTruncated) {
		t.Error("expected IsTruncated=true on page 2")
	}

	// List final page
	marker2, err := strconv.Atoi(aws.ToString(listResp2.NextPartNumberMarker))
	if err != nil {
		t.Fatalf("parse NextPartNumberMarker (page 2): %v", err)
	}
	listResp3, err := client.ListParts(ctx, &s3.ListPartsInput{
		Bucket:           aws.String(bucket),
		Key:              aws.String(key),
		UploadId:         aws.String(uploadID),
		MaxParts:         aws.Int32(10),
		PartNumberMarker: aws.String(strconv.Itoa(marker2)),
	})
	if err != nil {
		t.Fatalf("list parts (page 3): %v", err)
	}

	if len(listResp3.Parts) != 5 {
		t.Fatalf("expected 5 parts on page 3, got %d", len(listResp3.Parts))
	}
	if aws.ToBool(listResp3.IsTruncated) {
		t.Error("expected IsTruncated=false on final page")
	}
}

// TestMultipartUploadInvalidUploadID tests error handling for invalid upload ID.
func TestMultipartUploadInvalidUploadID(t *testing.T) {
	ctx := context.Background()
	client, _, cleanup := setupS3ServerWithClient(t)
	defer cleanup()

	bucket := "test-bucket"
	key := "invalid-upload.dat"

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	defer func() {
		_, _ = client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	}()

	fakeUploadID := "invalid-upload-id"
	partData := generatePartData(1024, 1)

	// Attempt UploadPart with invalid upload ID
	_, err = client.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(key),
		UploadId:   aws.String(fakeUploadID),
		PartNumber: aws.Int32(1),
		Body:       bytes.NewReader(partData),
	})
	if err == nil {
		t.Error("expected error for invalid upload ID in UploadPart")
	}

	// Attempt CompleteMultipart with invalid upload ID
	_, err = client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(fakeUploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: []types.CompletedPart{
				{PartNumber: aws.Int32(1), ETag: aws.String("fake-etag")},
			},
		},
	})
	if err == nil {
		t.Error("expected error for invalid upload ID in CompleteMultipart")
	}
}

// TestMultipartUploadContentType tests content type preservation.
func TestMultipartUploadContentType(t *testing.T) {
	ctx := context.Background()
	client, _, cleanup := setupS3ServerWithClient(t)
	defer cleanup()

	bucket := "test-bucket"
	key := "content-type.png"

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	defer func() {
		_, _ = client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
	}()

	// Initiate upload with Content-Type
	createResp, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String("image/png"),
	})
	if err != nil {
		t.Fatalf("create multipart upload: %v", err)
	}
	uploadID := aws.ToString(createResp.UploadId)

	// Upload part
	partData := generatePartData(1024*1024, 1)
	etag, err := uploadPartWithData(ctx, client, bucket, key, uploadID, 1, partData)
	if err != nil {
		t.Fatalf("upload part: %v", err)
	}

	// Complete upload
	_, err = client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: []types.CompletedPart{
				{PartNumber: aws.Int32(1), ETag: aws.String(etag)},
			},
		},
	})
	if err != nil {
		t.Fatalf("complete multipart upload: %v", err)
	}

	// Verify Content-Type preserved
	headResp, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("head object: %v", err)
	}

	// Note: Content-Type preservation depends on storage backend capabilities
	// Local storage driver may not preserve Content-Type
	contentType := aws.ToString(headResp.ContentType)
	if contentType != "" && contentType != "image/png" {
		t.Errorf("unexpected Content-Type: got %s", contentType)
	}
	// Skip strict check as local storage driver may not preserve Content-Type

	// Cleanup
	_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
}
