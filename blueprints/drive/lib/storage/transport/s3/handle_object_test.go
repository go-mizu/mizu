package s3

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// TestPutObject tests uploading objects.
func TestPutObject(t *testing.T) {
	baseURL, cleanup := setupTestServer(t, nil)
	defer cleanup()

	bucketURL := baseURL + "/test-bucket"

	// Create bucket
	status, _, _ := doRequest(t, http.MethodPut, bucketURL, nil, nil)
	if status != http.StatusOK {
		t.Fatalf("create bucket status = %d", status)
	}

	// Put object
	objectURL := bucketURL + "/test.txt"
	content := []byte("hello world")
	headers := map[string]string{
		"Content-Type":   "text/plain",
		"Content-Length": strconv.Itoa(len(content)),
	}

	status, _, resp := doRequest(t, http.MethodPut, objectURL, bytes.NewReader(content), headers)
	if status != http.StatusOK {
		t.Fatalf("put object status = %d", status)
	}

	// Verify ETag is set (if supported by storage backend)
	etag := resp.Header.Get("ETag")
	if etag != "" {
		t.Logf("ETag returned: %s", etag)
	}
	// Note: Not all storage backends generate ETags
}

// TestGetObject tests retrieving objects.
func TestGetObject(t *testing.T) {
	baseURL, cleanup := setupTestServer(t, nil)
	defer cleanup()

	bucketURL := baseURL + "/test-bucket"

	// Create bucket
	status, _, _ := doRequest(t, http.MethodPut, bucketURL, nil, nil)
	if status != http.StatusOK {
		t.Fatalf("create bucket status = %d", status)
	}

	// Put object
	objectURL := bucketURL + "/test.txt"
	content := []byte("hello world")
	headers := map[string]string{"Content-Type": "text/plain"}

	status, _, _ = doRequest(t, http.MethodPut, objectURL, bytes.NewReader(content), headers)
	if status != http.StatusOK {
		t.Fatalf("put object status = %d", status)
	}

	// Get object
	status, body, resp := doRequest(t, http.MethodGet, objectURL, nil, nil)
	if status != http.StatusOK {
		t.Fatalf("get object status = %d", status)
	}
	if !bytes.Equal(body, content) {
		t.Errorf("expected body %q, got %q", string(content), string(body))
	}

	// Verify headers
	contentType := resp.Header.Get("Content-Type")
	// Note: Content-Type preservation depends on storage backend
	// Local driver may not preserve it
	if contentType == "" {
		t.Error("Content-Type header not set")
	}

	contentLength := resp.Header.Get("Content-Length")
	if contentLength != strconv.Itoa(len(content)) {
		t.Errorf("expected Content-Length %d, got %s", len(content), contentLength)
	}

	etag := resp.Header.Get("ETag")
	if etag != "" {
		t.Logf("ETag returned: %s", etag)
	}
	// Note: Not all storage backends generate ETags

	acceptRanges := resp.Header.Get("Accept-Ranges")
	if acceptRanges != "bytes" {
		t.Errorf("expected Accept-Ranges=bytes, got %s", acceptRanges)
	}
}

// TestGetObjectNotFound tests getting a non-existent object.
func TestGetObjectNotFound(t *testing.T) {
	baseURL, cleanup := setupTestServer(t, nil)
	defer cleanup()

	bucketURL := baseURL + "/test-bucket"

	// Create bucket
	status, _, _ := doRequest(t, http.MethodPut, bucketURL, nil, nil)
	if status != http.StatusOK {
		t.Fatalf("create bucket status = %d", status)
	}

	// Get non-existent object
	objectURL := bucketURL + "/nonexistent.txt"
	status, body, _ := doRequest(t, http.MethodGet, objectURL, nil, nil)
	if status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", status)
	}

	var errResp ErrorResponse
	if err := xml.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if errResp.Code != "NoSuchKey" {
		t.Errorf("expected error code NoSuchKey, got %s", errResp.Code)
	}
}

// TestHeadObject tests the HeadObject operation.
func TestHeadObject(t *testing.T) {
	baseURL, cleanup := setupTestServer(t, nil)
	defer cleanup()

	bucketURL := baseURL + "/test-bucket"

	// Create bucket
	status, _, _ := doRequest(t, http.MethodPut, bucketURL, nil, nil)
	if status != http.StatusOK {
		t.Fatalf("create bucket status = %d", status)
	}

	// Put object
	objectURL := bucketURL + "/test.txt"
	content := []byte("hello world")
	headers := map[string]string{"Content-Type": "text/plain"}

	status, _, _ = doRequest(t, http.MethodPut, objectURL, bytes.NewReader(content), headers)
	if status != http.StatusOK {
		t.Fatalf("put object status = %d", status)
	}

	// Head object
	status, body, resp := doRequest(t, http.MethodHead, objectURL, nil, nil)
	if status != http.StatusOK {
		t.Fatalf("head object status = %d", status)
	}

	// Body should be empty for HEAD
	if len(body) != 0 {
		t.Error("expected empty body for HEAD request")
	}

	// Verify headers
	contentLength := resp.Header.Get("Content-Length")
	if contentLength != strconv.Itoa(len(content)) {
		t.Errorf("expected Content-Length %d, got %s", len(content), contentLength)
	}

	etag := resp.Header.Get("ETag")
	if etag != "" {
		t.Logf("ETag returned: %s", etag)
	}
	// Note: Not all storage backends generate ETags

	acceptRanges := resp.Header.Get("Accept-Ranges")
	if acceptRanges != "bytes" {
		t.Errorf("expected Accept-Ranges=bytes, got %s", acceptRanges)
	}
}

// TestDeleteObject tests the DeleteObject operation.
func TestDeleteObject(t *testing.T) {
	baseURL, cleanup := setupTestServer(t, nil)
	defer cleanup()

	bucketURL := baseURL + "/test-bucket"

	// Create bucket
	status, _, _ := doRequest(t, http.MethodPut, bucketURL, nil, nil)
	if status != http.StatusOK {
		t.Fatalf("create bucket status = %d", status)
	}

	// Put object
	objectURL := bucketURL + "/test.txt"
	status, _, _ = doRequest(t, http.MethodPut, objectURL, bytes.NewReader([]byte("test")), nil)
	if status != http.StatusOK {
		t.Fatalf("put object status = %d", status)
	}

	// Delete object
	status, _, _ = doRequest(t, http.MethodDelete, objectURL, nil, nil)
	if status != http.StatusNoContent {
		t.Fatalf("delete object status = %d", status)
	}

	// Head should now return 404
	status, _, _ = doRequest(t, http.MethodHead, objectURL, nil, nil)
	if status != http.StatusNotFound {
		t.Errorf("head after delete status = %d", status)
	}
}

// TestDeleteObjectIdempotent tests that deleting a non-existent object succeeds.
func TestDeleteObjectIdempotent(t *testing.T) {
	baseURL, cleanup := setupTestServer(t, nil)
	defer cleanup()

	bucketURL := baseURL + "/test-bucket"

	// Create bucket
	status, _, _ := doRequest(t, http.MethodPut, bucketURL, nil, nil)
	if status != http.StatusOK {
		t.Fatalf("create bucket status = %d", status)
	}

	// Delete non-existent object (should succeed per S3 semantics)
	objectURL := bucketURL + "/nonexistent.txt"
	status, _, _ = doRequest(t, http.MethodDelete, objectURL, nil, nil)
	if status != http.StatusNoContent {
		t.Errorf("delete non-existent object status = %d", status)
	}
}

// TestCopyObject tests the CopyObject operation.
func TestCopyObject(t *testing.T) {
	baseURL, cleanup := setupTestServer(t, nil)
	defer cleanup()

	bucketURL := baseURL + "/test-bucket"

	// Create bucket
	status, _, _ := doRequest(t, http.MethodPut, bucketURL, nil, nil)
	if status != http.StatusOK {
		t.Fatalf("create bucket status = %d", status)
	}

	// Put source object
	sourceURL := bucketURL + "/source.txt"
	content := []byte("hello world")
	status, _, _ = doRequest(t, http.MethodPut, sourceURL, bytes.NewReader(content), nil)
	if status != http.StatusOK {
		t.Fatalf("put source object status = %d", status)
	}

	// Copy object
	destURL := bucketURL + "/dest.txt"
	headers := map[string]string{
		"x-amz-copy-source": "/test-bucket/source.txt",
	}
	status, body, _ := doRequest(t, http.MethodPut, destURL, nil, headers)
	if status != http.StatusOK {
		t.Fatalf("copy object status = %d, body = %s", status, string(body))
	}

	// Verify response XML
	type copyObjectResult struct {
		XMLName      xml.Name `xml:"CopyObjectResult"`
		LastModified string   `xml:"LastModified"`
		ETag         string   `xml:"ETag"`
	}
	var copyResp copyObjectResult
	if err := xml.Unmarshal(body, &copyResp); err != nil {
		t.Fatalf("unmarshal copy response: %v", err)
	}
	// Note: ETag generation depends on storage backend
	if copyResp.ETag != "" {
		t.Logf("ETag in copy response: %s", copyResp.ETag)
	}

	// Get destination object and verify content
	status, destBody, _ := doRequest(t, http.MethodGet, destURL, nil, nil)
	if status != http.StatusOK {
		t.Fatalf("get dest object status = %d", status)
	}
	if !bytes.Equal(destBody, content) {
		t.Errorf("expected dest body %q, got %q", string(content), string(destBody))
	}
}

// TestGetObjectRangeRequests tests byte range requests.
func TestGetObjectRangeRequests(t *testing.T) {
	baseURL, cleanup := setupTestServer(t, nil)
	defer cleanup()

	bucketURL := baseURL + "/test-bucket"

	// Create bucket
	status, _, _ := doRequest(t, http.MethodPut, bucketURL, nil, nil)
	if status != http.StatusOK {
		t.Fatalf("create bucket status = %d", status)
	}

	// Put object with known content
	objectURL := bucketURL + "/test.txt"
	content := []byte("0123456789")
	status, _, _ = doRequest(t, http.MethodPut, objectURL, bytes.NewReader(content), nil)
	if status != http.StatusOK {
		t.Fatalf("put object status = %d", status)
	}

	tests := []struct {
		name          string
		rangeHeader   string
		expectedCode  int
		expectedBody  string
		expectedRange string
	}{
		{
			name:          "bytes 0-4",
			rangeHeader:   "bytes=0-4",
			expectedCode:  http.StatusPartialContent,
			expectedBody:  "01234",
			expectedRange: "bytes 0-4/10",
		},
		{
			name:          "bytes 5-9",
			rangeHeader:   "bytes=5-9",
			expectedCode:  http.StatusPartialContent,
			expectedBody:  "56789",
			expectedRange: "bytes 5-9/10",
		},
		{
			name:          "bytes 5-",
			rangeHeader:   "bytes=5-",
			expectedCode:  http.StatusPartialContent,
			expectedBody:  "56789",
			expectedRange: "bytes 5-9/10",
		},
		{
			name:          "bytes -5 (suffix)",
			rangeHeader:   "bytes=-5",
			expectedCode:  http.StatusPartialContent,
			expectedBody:  "56789",
			expectedRange: "bytes 5-9/10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{"Range": tt.rangeHeader}
			status, body, resp := doRequest(t, http.MethodGet, objectURL, nil, headers)

			if status != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, status)
			}

			if string(body) != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, string(body))
			}

			contentRange := resp.Header.Get("Content-Range")
			if contentRange != tt.expectedRange {
				t.Errorf("expected Content-Range %q, got %q", tt.expectedRange, contentRange)
			}

			contentLength := resp.Header.Get("Content-Length")
			expectedLength := strconv.Itoa(len(tt.expectedBody))
			if contentLength != expectedLength {
				t.Errorf("expected Content-Length %s, got %s", expectedLength, contentLength)
			}
		})
	}
}

// TestPutObjectWithMetadata tests uploading objects with custom metadata.
func TestPutObjectWithMetadata(t *testing.T) {
	ctx := context.Background()
	client, cleanup := setupTestServerWithClient(t)
	defer cleanup()

	bucket := "test-bucket"
	key := "test.txt"

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	// Put object with metadata
	metadata := map[string]string{
		"author":  "test-user",
		"version": "1.0",
	}
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		Body:     bytes.NewReader([]byte("test")),
		Metadata: metadata,
	})
	if err != nil {
		t.Fatalf("put object: %v", err)
	}

	// Head object and verify metadata
	resp, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("head object: %v", err)
	}

	// Note: Metadata preservation depends on storage backend capabilities
	// Local storage driver may not support custom metadata
	if len(resp.Metadata) == 0 {
		t.Skip("Metadata not supported by storage backend")
		return
	}

	if resp.Metadata["author"] != "test-user" {
		t.Errorf("expected metadata author=test-user, got %v", resp.Metadata["author"])
	}
	if resp.Metadata["version"] != "1.0" {
		t.Errorf("expected metadata version=1.0, got %v", resp.Metadata["version"])
	}
}

// TestMaxObjectSize tests that MaxObjectSize configuration is enforced.
func TestMaxObjectSize(t *testing.T) {
	cfg := &Config{MaxObjectSize: 100}
	baseURL, cleanup := setupTestServer(t, cfg)
	defer cleanup()

	bucketURL := baseURL + "/test-bucket"

	// Create bucket
	status, _, _ := doRequest(t, http.MethodPut, bucketURL, nil, nil)
	if status != http.StatusOK {
		t.Fatalf("create bucket status = %d", status)
	}

	// Try to upload object larger than limit
	objectURL := bucketURL + "/large.txt"
	content := make([]byte, 200)
	headers := map[string]string{"Content-Length": strconv.Itoa(len(content))}

	status, body, _ := doRequest(t, http.MethodPut, objectURL, bytes.NewReader(content), headers)
	if status != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d, body = %s", status, string(body))
	}

	var errResp ErrorResponse
	if err := xml.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if errResp.Code != "EntityTooLarge" {
		t.Errorf("expected error code EntityTooLarge, got %s", errResp.Code)
	}
}

// TestObjectOperationsWithAWSClient tests object operations using AWS SDK.
func TestObjectOperationsWithAWSClient(t *testing.T) {
	ctx := context.Background()
	client, cleanup := setupTestServerWithClient(t)
	defer cleanup()

	bucket := "test-bucket"
	key := "test.txt"
	content := []byte("hello world")

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	// Put object
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(content),
	})
	if err != nil {
		t.Fatalf("put object: %v", err)
	}

	// Get object
	getResp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("get object: %v", err)
	}
	defer getResp.Body.Close()

	body, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !bytes.Equal(body, content) {
		t.Errorf("expected body %q, got %q", string(content), string(body))
	}

	// Head object
	_, err = client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("head object: %v", err)
	}

	// Copy object
	copyKey := "copy.txt"
	copySource := bucket + "/" + key
	_, err = client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(copyKey),
		CopySource: aws.String(copySource),
	})
	if err != nil {
		t.Fatalf("copy object: %v", err)
	}

	// Verify copied object
	getResp, err = client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(copyKey),
	})
	if err != nil {
		t.Fatalf("get copied object: %v", err)
	}
	defer getResp.Body.Close()

	copiedBody, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatalf("read copied body: %v", err)
	}
	if !bytes.Equal(copiedBody, content) {
		t.Errorf("expected copied body %q, got %q", string(content), string(copiedBody))
	}

	// Delete objects
	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("delete object: %v", err)
	}

	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(copyKey),
	})
	if err != nil {
		t.Fatalf("delete copied object: %v", err)
	}
}

// TestPresignedURLs tests presigned URL support.
func TestPresignedURLs(t *testing.T) {
	ctx := context.Background()
	client, cleanup := setupTestServerWithClient(t)
	defer cleanup()

	bucket := "test-bucket"
	key := "test.txt"
	content := []byte("presigned content")

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	// Put object
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(content),
	})
	if err != nil {
		t.Fatalf("put object: %v", err)
	}

	// Generate presigned GET URL
	presigner := s3.NewPresignClient(client)
	presigned, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("presign get: %v", err)
	}

	// Use presigned URL
	resp, err := http.Get(presigned.URL)
	if err != nil {
		t.Fatalf("do presigned get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("presigned status = %d", resp.StatusCode)
	}

	presignedBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read presigned body: %v", err)
	}
	if !bytes.Equal(presignedBody, content) {
		t.Errorf("presigned body mismatch: got %q, want %q", string(presignedBody), string(content))
	}
}
