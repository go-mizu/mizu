# Spec 0382: Storage REST API (Supabase Storage & S3 Compatible) Testing Plan

## Document Info

| Field | Value |
|-------|-------|
| Spec ID | 0382 |
| Version | 1.0 |
| Date | 2025-01-16 |
| Status | **In Progress** |
| Priority | Critical |
| Estimated Tests | 200+ |
| Supabase Storage Version | Latest |
| Supabase Local Port | 54421 |
| Localbase Port | 54321 |

## Overview

This document outlines a comprehensive testing plan for the Localbase Storage REST API to achieve 100% compatibility with Supabase's Storage implementation. Testing will be performed against both Supabase Local and Localbase to verify identical behavior for inputs, outputs, and error codes.

### Testing Philosophy

- **No mocks**: All tests run against real storage backends
- **Side-by-side comparison**: Every request runs against both Supabase and Localbase
- **Comprehensive coverage**: Every endpoint, edge case, and error condition
- **Regression prevention**: Tests ensure compatibility is maintained over time
- **Response accuracy**: Response bodies, headers, and error codes must match exactly

### Compatibility Target

| Aspect | Target |
|--------|--------|
| HTTP Status Codes | 100% match |
| Error Response Format | 100% match |
| Response Headers | 100% match |
| Response Body Structure | 100% match |
| S3 API Compatibility | 100% match |

## Reference Documentation

### Official Supabase Storage Documentation
- [Supabase Storage Guide](https://supabase.com/docs/guides/storage)
- [Supabase Storage Buckets](https://supabase.com/docs/guides/storage/buckets/fundamentals)
- [Supabase Storage S3 Compatibility](https://supabase.com/docs/guides/storage/s3/compatibility)
- [Supabase Storage JavaScript API](https://supabase.com/docs/reference/javascript/storage-from-list)
- [Supabase Storage OpenAPI](https://supabase.github.io/storage/)

### S3 API Documentation
- [AWS S3 API Reference](https://docs.aws.amazon.com/AmazonS3/latest/API/Welcome.html)
- [AWS Signature V4](https://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-authenticating-requests.html)

## Test Environment Setup

### Supabase Local Configuration
```
Storage API: http://127.0.0.1:54421/storage/v1
S3 Endpoint: http://127.0.0.1:54421/storage/v1/s3
Database: postgresql://postgres:postgres@127.0.0.1:54322/postgres
API Key: sb_publishable_ACJWlzQHlZjBrEguHvfOxg_3BJgxAaH
```

### Localbase Configuration
```
Storage API: http://localhost:54321/storage/v1
S3 Endpoint: http://localhost:54321/s3
Database: postgresql://localbase:localbase@localhost:5432/localbase
API Key: sb_publishable_ACJWlzQHlZjBrEguHvfOxg_3BJgxAaH
```

---

## 1. Bucket Operations

### 1.1 Create Bucket (POST /bucket)

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| BUCKET-001 | Create public bucket | `POST /bucket` with `{"name": "test-public", "public": true}` | 200 OK, `{"name": "test-public"}` |
| BUCKET-002 | Create private bucket | `POST /bucket` with `{"name": "test-private", "public": false}` | 200 OK, `{"name": "test-private"}` |
| BUCKET-003 | Create bucket with file size limit | `POST /bucket` with `{"name": "limited", "file_size_limit": 1048576}` | 200 OK |
| BUCKET-004 | Create bucket with allowed MIME types | `POST /bucket` with `{"name": "images", "allowed_mime_types": ["image/png", "image/jpeg"]}` | 200 OK |
| BUCKET-005 | Create bucket with empty name | `POST /bucket` with `{"name": ""}` | 400 Bad Request |
| BUCKET-006 | Create duplicate bucket | Create same bucket twice | 409 Conflict |
| BUCKET-007 | Create bucket with invalid name | `POST /bucket` with `{"name": "invalid name!"}` | 400 Bad Request |

### 1.2 List Buckets (GET /bucket)

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| BUCKET-010 | List all buckets | `GET /bucket` | 200 OK, Array of buckets |
| BUCKET-011 | List with limit | `GET /bucket?limit=5` | 200 OK, Max 5 buckets |
| BUCKET-012 | List with offset | `GET /bucket?offset=2` | 200 OK, Skip first 2 |
| BUCKET-013 | List empty (no buckets) | `GET /bucket` after deleting all | 200 OK, `[]` |

### 1.3 Get Bucket (GET /bucket/:bucketId)

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| BUCKET-020 | Get existing bucket | `GET /bucket/test-bucket` | 200 OK, Bucket details |
| BUCKET-021 | Get non-existent bucket | `GET /bucket/nonexistent` | 404 Not Found |
| BUCKET-022 | Verify bucket properties | `GET /bucket/test-bucket` | Returns id, name, public, created_at |

### 1.4 Update Bucket (PUT /bucket/:bucketId)

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| BUCKET-030 | Update bucket to public | `PUT /bucket/test-bucket` with `{"public": true}` | 200 OK |
| BUCKET-031 | Update bucket to private | `PUT /bucket/test-bucket` with `{"public": false}` | 200 OK |
| BUCKET-032 | Update file size limit | `PUT /bucket/test-bucket` with `{"file_size_limit": 5242880}` | 200 OK |
| BUCKET-033 | Update allowed MIME types | `PUT /bucket/test-bucket` with `{"allowed_mime_types": ["image/*"]}` | 200 OK |
| BUCKET-034 | Update non-existent bucket | `PUT /bucket/nonexistent` | 404 Not Found |

### 1.5 Delete Bucket (DELETE /bucket/:bucketId)

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| BUCKET-040 | Delete empty bucket | `DELETE /bucket/empty-bucket` | 200 OK |
| BUCKET-041 | Delete non-empty bucket | `DELETE /bucket/bucket-with-files` | 409 Conflict |
| BUCKET-042 | Delete non-existent bucket | `DELETE /bucket/nonexistent` | 404 Not Found |

### 1.6 Empty Bucket (POST /bucket/:bucketId/empty)

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| BUCKET-050 | Empty bucket with files | `POST /bucket/test-bucket/empty` | 200 OK |
| BUCKET-051 | Empty already empty bucket | `POST /bucket/empty-bucket/empty` | 200 OK |
| BUCKET-052 | Empty non-existent bucket | `POST /bucket/nonexistent/empty` | 404 Not Found |

---

## 2. Object Upload Operations

### 2.1 Upload Object (POST /object/:bucketName/*path)

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| UPLOAD-001 | Upload small file | `POST /object/bucket/file.txt` with content | 200 OK, `{"Id": "...", "Key": "bucket/file.txt"}` |
| UPLOAD-002 | Upload large file | `POST /object/bucket/large.bin` (5MB) | 200 OK |
| UPLOAD-003 | Upload with Content-Type | Header: `Content-Type: image/png` | 200 OK, Content-Type stored |
| UPLOAD-004 | Upload to nested path | `POST /object/bucket/folder/subfolder/file.txt` | 200 OK |
| UPLOAD-005 | Upload duplicate (no upsert) | Upload same path without x-upsert | 409 Conflict |
| UPLOAD-006 | Upload with upsert | Header: `x-upsert: true` | 200 OK, File replaced |
| UPLOAD-007 | Upload to non-existent bucket | `POST /object/nonexistent/file.txt` | 404 Not Found |
| UPLOAD-008 | Upload empty file | Empty body | 200 OK |
| UPLOAD-009 | Upload with special characters in path | `POST /object/bucket/file%20name.txt` | 200 OK |

### 2.2 Update Object (PUT /object/:bucketName/*path)

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| UPLOAD-010 | Update existing file | `PUT /object/bucket/file.txt` | 200 OK |
| UPLOAD-011 | Update non-existent file | `PUT /object/bucket/nonexistent.txt` | 404 Not Found |
| UPLOAD-012 | Update with different content type | New Content-Type header | 200 OK |

### 2.3 TUS Resumable Upload

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| TUS-001 | OPTIONS discovery | `OPTIONS /upload/resumable/` | 200 OK with TUS headers |
| TUS-002 | Create resumable upload | `POST /upload/resumable/bucket/file.txt` | 201 Created with Location |
| TUS-003 | Upload chunk | `PATCH /upload/resumable/bucket/file.txt` | 204 No Content |
| TUS-004 | Get upload status | `HEAD /upload/resumable/bucket/file.txt` | 200 OK with Upload-Offset |
| TUS-005 | Cancel upload | `DELETE /upload/resumable/bucket/file.txt` | 204 No Content |
| TUS-006 | Resume interrupted upload | PATCH with Upload-Offset | 204 No Content |

---

## 3. Object Download Operations

### 3.1 Download Object (GET /object/:bucketName/*path)

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| DOWNLOAD-001 | Download existing file | `GET /object/bucket/file.txt` | 200 OK with content |
| DOWNLOAD-002 | Download non-existent file | `GET /object/bucket/nonexistent.txt` | 404 Not Found |
| DOWNLOAD-003 | Download with correct Content-Type | `GET /object/bucket/image.png` | Content-Type: image/png |
| DOWNLOAD-004 | Download with Range header | `Range: bytes=0-99` | 206 Partial Content |
| DOWNLOAD-005 | Download with ?download param | `GET /object/bucket/file.txt?download=filename.txt` | Content-Disposition: attachment |
| DOWNLOAD-006 | Download from nested path | `GET /object/bucket/folder/file.txt` | 200 OK |

### 3.2 Public Object Access (GET /object/public/:bucketName/*path)

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| PUBLIC-001 | Access public bucket file | `GET /object/public/public-bucket/file.txt` | 200 OK |
| PUBLIC-002 | Access private bucket file | `GET /object/public/private-bucket/file.txt` | 403 Forbidden |
| PUBLIC-003 | Access non-existent file | `GET /object/public/bucket/nonexistent.txt` | 404 Not Found |

### 3.3 Authenticated Object Access (GET /object/authenticated/:bucketName/*path)

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| AUTH-001 | Access with valid token | `GET /object/authenticated/bucket/file.txt` | 200 OK |
| AUTH-002 | Access without token | No Authorization header | 401 Unauthorized |
| AUTH-003 | Access with expired token | Expired JWT | 401 Unauthorized |

---

## 4. Object List Operations

### 4.1 List Objects (POST /object/list/:bucketName)

| Test Case | Description | Request Body | Expected Response |
|-----------|-------------|--------------|-------------------|
| LIST-001 | List root level | `{"prefix": ""}` | 200 OK, Array of objects |
| LIST-002 | List with prefix | `{"prefix": "folder/"}` | Only objects in folder |
| LIST-003 | List with limit | `{"prefix": "", "limit": 10}` | Max 10 objects |
| LIST-004 | List with offset | `{"prefix": "", "offset": 5}` | Skip first 5 |
| LIST-005 | List empty bucket | `{"prefix": ""}` on empty bucket | 200 OK, `[]` |
| LIST-006 | List with search | `{"prefix": "", "search": "test"}` | Filtered results |
| LIST-007 | List non-existent bucket | `POST /object/list/nonexistent` | 404 Not Found |
| LIST-008 | List with sort | `{"prefix": "", "sortBy": {"column": "name", "order": "asc"}}` | Sorted results |

---

## 5. Object Move/Copy Operations

### 5.1 Move Object (POST /object/move)

| Test Case | Description | Request Body | Expected Response |
|-----------|-------------|--------------|-------------------|
| MOVE-001 | Move within bucket | `{"bucketId": "bucket", "sourceKey": "a.txt", "destinationKey": "b.txt"}` | 200 OK |
| MOVE-002 | Move to different bucket | `{"bucketId": "src", "sourceKey": "file.txt", "destinationBucket": "dst", "destinationKey": "file.txt"}` | 200 OK |
| MOVE-003 | Move to nested path | `{"bucketId": "bucket", "sourceKey": "file.txt", "destinationKey": "folder/file.txt"}` | 200 OK |
| MOVE-004 | Move non-existent file | Source doesn't exist | 404 Not Found |
| MOVE-005 | Move to existing path | Destination exists | 409 Conflict or 200 OK |

### 5.2 Copy Object (POST /object/copy)

| Test Case | Description | Request Body | Expected Response |
|-----------|-------------|--------------|-------------------|
| COPY-001 | Copy within bucket | `{"bucketId": "bucket", "sourceKey": "a.txt", "destinationKey": "b.txt"}` | 200 OK with Key and Id |
| COPY-002 | Copy to different bucket | `{"bucketId": "src", "sourceKey": "file.txt", "destinationBucket": "dst", "destinationKey": "file.txt"}` | 200 OK |
| COPY-003 | Copy with metadata | `{"bucketId": "bucket", "sourceKey": "file.txt", "destinationKey": "copy.txt", "metadata": {"key": "value"}}` | 200 OK |
| COPY-004 | Copy non-existent file | Source doesn't exist | 404 Not Found |

---

## 6. Object Delete Operations

### 6.1 Delete Single Object (DELETE /object/:bucketName/*path)

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| DELETE-001 | Delete existing file | `DELETE /object/bucket/file.txt` | 200 OK |
| DELETE-002 | Delete non-existent file | `DELETE /object/bucket/nonexistent.txt` | 404 Not Found or 200 OK |
| DELETE-003 | Delete from nested path | `DELETE /object/bucket/folder/file.txt` | 200 OK |

### 6.2 Delete Multiple Objects (DELETE /object/:bucketName)

| Test Case | Description | Request Body | Expected Response |
|-----------|-------------|--------------|-------------------|
| DELETE-010 | Delete multiple files | `{"prefixes": ["file1.txt", "file2.txt"]}` | 200 OK, Array of deleted |
| DELETE-011 | Delete with mixed existence | Some files exist, some don't | 200 OK, Partial success |
| DELETE-012 | Delete empty prefixes array | `{"prefixes": []}` | 400 Bad Request |

---

## 7. Object Info Operations

### 7.1 Get Object Info (GET /object/info/:bucketName/*path)

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| INFO-001 | Get existing file info | `GET /object/info/bucket/file.txt` | 200 OK with metadata |
| INFO-002 | Get non-existent file info | `GET /object/info/bucket/nonexistent.txt` | 404 Not Found |
| INFO-003 | Verify info properties | GET existing file | Returns id, name, bucket_id, created_at, updated_at, metadata |

### 7.2 Public Object Info (GET /object/info/public/:bucketName/*path)

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| INFO-010 | Public bucket file info | `GET /object/info/public/public-bucket/file.txt` | 200 OK |
| INFO-011 | Private bucket file info | `GET /object/info/public/private-bucket/file.txt` | 403 Forbidden |

---

## 8. Signed URL Operations

### 8.1 Create Signed URL (POST /object/sign/:bucketName/*path)

| Test Case | Description | Request Body | Expected Response |
|-----------|-------------|--------------|-------------------|
| SIGN-001 | Create signed URL | `{"expiresIn": 3600}` | 200 OK, `{"signedURL": "..."}` |
| SIGN-002 | Create with short expiry | `{"expiresIn": 60}` | 200 OK |
| SIGN-003 | Invalid expiresIn | `{"expiresIn": 0}` | 400 Bad Request |
| SIGN-004 | Negative expiresIn | `{"expiresIn": -1}` | 400 Bad Request |
| SIGN-005 | Non-existent file | File doesn't exist | 200 OK (URL still generated) or 404 |
| SIGN-006 | Create with transform | `{"expiresIn": 3600, "transform": {"width": 100, "height": 100}}` | 200 OK |

### 8.2 Create Multiple Signed URLs (POST /object/sign/:bucketName)

| Test Case | Description | Request Body | Expected Response |
|-----------|-------------|--------------|-------------------|
| SIGN-010 | Multiple URLs | `{"expiresIn": 3600, "paths": ["file1.txt", "file2.txt"]}` | 200 OK, Array of URLs |
| SIGN-011 | Mixed existence | Some paths exist, some don't | 200 OK with errors for missing |
| SIGN-012 | Empty paths array | `{"expiresIn": 3600, "paths": []}` | 400 Bad Request |

### 8.3 Create Signed Upload URL (POST /object/upload/sign/:bucketName/*path)

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| SIGN-020 | Create upload URL | `POST /object/upload/sign/bucket/newfile.txt` | 200 OK, `{"url": "...", "token": "..."}` |
| SIGN-021 | Upload to signed URL | PUT to signed URL | 200 OK |

---

## 9. S3 API Compatibility

### 9.1 S3 Bucket Operations

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| S3-BUCKET-001 | List buckets | `GET /s3/` | 200 OK, XML ListAllMyBucketsResult |
| S3-BUCKET-002 | Create bucket | `PUT /s3/bucket-name` | 200 OK |
| S3-BUCKET-003 | Head bucket | `HEAD /s3/bucket-name` | 200 OK |
| S3-BUCKET-004 | Delete bucket | `DELETE /s3/bucket-name` | 204 No Content |
| S3-BUCKET-005 | Get bucket location | `GET /s3/bucket-name?location` | 200 OK, XML LocationConstraint |

### 9.2 S3 Object Operations

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| S3-OBJ-001 | Put object | `PUT /s3/bucket/key` | 200 OK with ETag |
| S3-OBJ-002 | Get object | `GET /s3/bucket/key` | 200 OK with content |
| S3-OBJ-003 | Head object | `HEAD /s3/bucket/key` | 200 OK with metadata headers |
| S3-OBJ-004 | Delete object | `DELETE /s3/bucket/key` | 204 No Content |
| S3-OBJ-005 | Copy object | `PUT /s3/bucket/key` with `x-amz-copy-source` | 200 OK |
| S3-OBJ-006 | List objects v2 | `GET /s3/bucket?list-type=2` | 200 OK, XML ListBucketResult |
| S3-OBJ-007 | List with prefix | `GET /s3/bucket?prefix=folder/` | Filtered results |
| S3-OBJ-008 | List with delimiter | `GET /s3/bucket?delimiter=/` | Common prefixes |

### 9.3 S3 Multipart Upload

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| S3-MP-001 | Create multipart upload | `POST /s3/bucket/key?uploads` | 200 OK, XML InitiateMultipartUploadResult |
| S3-MP-002 | Upload part | `PUT /s3/bucket/key?partNumber=1&uploadId=...` | 200 OK with ETag |
| S3-MP-003 | List parts | `GET /s3/bucket/key?uploadId=...` | 200 OK, XML ListPartsResult |
| S3-MP-004 | Complete multipart | `POST /s3/bucket/key?uploadId=...` with XML | 200 OK, XML CompleteMultipartUploadResult |
| S3-MP-005 | Abort multipart | `DELETE /s3/bucket/key?uploadId=...` | 204 No Content |
| S3-MP-006 | List multipart uploads | `GET /s3/bucket?uploads` | 200 OK, XML ListMultipartUploadsResult |

### 9.4 S3 Batch Delete

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| S3-BATCH-001 | Delete multiple objects | `POST /s3/bucket?delete` with XML | 200 OK, XML DeleteResult |
| S3-BATCH-002 | Delete with quiet mode | `<Quiet>true</Quiet>` in request | Only errors in response |

### 9.5 S3 Signature V4

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| S3-SIG-001 | Valid signature header | AWS4-HMAC-SHA256 Authorization | 200 OK |
| S3-SIG-002 | Valid presigned URL | Query params with X-Amz-Signature | 200 OK |
| S3-SIG-003 | Expired presigned URL | Past X-Amz-Date + X-Amz-Expires | 403 Access Denied |
| S3-SIG-004 | Invalid signature | Wrong signature value | 403 Access Denied |
| S3-SIG-005 | Missing signature | No auth | 403 Access Denied |

---

## 10. Error Handling

### 10.1 Error Response Format (Supabase Storage)

```json
{
  "statusCode": 400,
  "error": "Bad Request",
  "message": "bucket name is required"
}
```

| Test Case | Field | Description |
|-----------|-------|-------------|
| ERR-001 | `statusCode` | HTTP status code as integer |
| ERR-002 | `error` | Human-readable error type |
| ERR-003 | `message` | Detailed error message |

### 10.2 S3 Error Response Format

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Error>
  <Code>NoSuchBucket</Code>
  <Message>The specified bucket does not exist</Message>
  <Resource>/bucket-name</Resource>
  <RequestId>...</RequestId>
</Error>
```

### 10.3 Error Code Mapping

| Scenario | REST API Status | S3 Error Code |
|----------|-----------------|---------------|
| Bucket not found | 404 | NoSuchBucket |
| Object not found | 404 | NoSuchKey |
| Bucket exists | 409 | BucketAlreadyExists |
| Bucket not empty | 409 | BucketNotEmpty |
| Access denied | 403 | AccessDenied |
| Invalid request | 400 | InvalidRequest |
| Entity too large | 413 | EntityTooLarge |

---

## 11. Authentication Tests

### 11.1 API Key Authentication

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| AUTH-010 | Valid API key | `apikey` header with valid key | 200 OK |
| AUTH-011 | Invalid API key | `apikey` header with invalid key | 401 Unauthorized |
| AUTH-012 | Missing API key | No `apikey` header | 401 Unauthorized |
| AUTH-013 | Bearer token | `Authorization: Bearer <token>` | 200 OK |

### 11.2 JWT Authentication

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| JWT-001 | Valid JWT | Valid Bearer token | 200 OK |
| JWT-002 | Expired JWT | Expired Bearer token | 401 Unauthorized |
| JWT-003 | Invalid JWT signature | Tampered token | 401 Unauthorized |
| JWT-004 | JWT claims validation | role, sub claims | Access based on RLS |

---

## 12. Edge Cases

### 12.1 Path Edge Cases

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| EDGE-001 | Path with spaces | `/object/bucket/file%20name.txt` | 200 OK |
| EDGE-002 | Path with unicode | `/object/bucket/文件.txt` | 200 OK |
| EDGE-003 | Path with dots | `/object/bucket/../file.txt` | Normalized or 400 |
| EDGE-004 | Very long path | 1000+ character path | 200 OK or 400 |
| EDGE-005 | Path with special chars | `/object/bucket/file!@#$%.txt` | 200 OK |
| EDGE-006 | Empty path segment | `/object/bucket//file.txt` | Normalized |

### 12.2 Content Edge Cases

| Test Case | Description | Request | Expected Response |
|-----------|-------------|---------|-------------------|
| EDGE-010 | Empty file | 0 bytes | 200 OK |
| EDGE-011 | Binary content | Non-UTF8 bytes | 200 OK |
| EDGE-012 | Large metadata | Large custom metadata | 200 OK or 400 |
| EDGE-013 | Various MIME types | Different Content-Types | Correctly stored |

### 12.3 Concurrent Operations

| Test Case | Description | Expected Behavior |
|-----------|-------------|-------------------|
| EDGE-020 | Concurrent uploads same file | Last write wins |
| EDGE-021 | Upload during download | Consistent read |
| EDGE-022 | Delete during download | Graceful handling |

---

## 13. Test Data Schema

### 13.1 Test Buckets

```go
// Test buckets to create
testBuckets := []struct {
    Name     string
    Public   bool
    SizeLimit int64
}{
    {"test-public", true, 0},
    {"test-private", false, 0},
    {"test-limited", false, 10 * 1024 * 1024}, // 10MB limit
    {"test-images", false, 0}, // with MIME type restriction
}
```

### 13.2 Test Files

```go
// Test files to upload
testFiles := []struct {
    Bucket      string
    Path        string
    Content     []byte
    ContentType string
}{
    {"test-public", "hello.txt", []byte("Hello, World!"), "text/plain"},
    {"test-public", "folder/nested.txt", []byte("Nested file"), "text/plain"},
    {"test-public", "image.png", pngBytes, "image/png"},
    {"test-private", "secret.txt", []byte("Secret data"), "text/plain"},
}
```

---

## 14. Test Execution

### 14.1 Prerequisites

```bash
# 1. Start Supabase Local
cd /path/to/project
supabase start

# 2. Start Localbase
go run ./blueprints/localbase/cmd/localbase

# 3. Verify both are running
curl -s http://127.0.0.1:54421/storage/v1/bucket -H "apikey: $SUPABASE_API_KEY" | head -c 100
curl -s http://localhost:54321/storage/v1/bucket -H "apikey: test-api-key" | head -c 100
```

### 14.2 Run Tests

```bash
# Run all storage comparison tests
go test -v ./blueprints/localbase/pkg/storage/... -run TestStorage

# Run specific test categories
go test -v ./blueprints/localbase/pkg/storage/... -run TestBucket
go test -v ./blueprints/localbase/pkg/storage/... -run TestUpload
go test -v ./blueprints/localbase/pkg/storage/... -run TestDownload
go test -v ./blueprints/localbase/pkg/storage/... -run TestS3

# Run with verbose output
go test -v ./blueprints/localbase/pkg/storage/... -args -verbose
```

---

## 15. Success Criteria

### 15.1 Pass Rates

| Category | Required |
|----------|----------|
| Bucket Operations | 100% |
| Upload Operations | 100% |
| Download Operations | 100% |
| List Operations | 100% |
| Move/Copy Operations | 100% |
| Delete Operations | 100% |
| Signed URLs | 100% |
| S3 Bucket Operations | 100% |
| S3 Object Operations | 100% |
| S3 Multipart | 100% |
| Error Handling | 100% |

### 15.2 Definition of Done

1. All test categories pass at 100%
2. Error response format matches exactly
3. HTTP status codes match for all scenarios
4. Response headers match (Content-Type, ETag, etc.)
5. S3 XML responses match schema
6. Presigned URLs work identically
7. No security vulnerabilities

---

## 16. Implementation Status

### Current Implementation Status

**Last Updated: 2026-01-16**

#### REST API Tests (pkg/storage/transport/rest)

| Category | Tests | Status |
|----------|-------|--------|
| Authentication Middleware | 6 | Complete |
| Bucket Name Validation | 4 | Complete |
| Object Content Types | 8 | Complete |
| Object Path Edge Cases | 7 | Complete |
| Range Request Handling | 6 | Complete |
| Batch Delete Operations | 2 | Complete |
| List Objects (Sort/Search) | 7 | Complete |
| Empty File Upload | 1 | Complete |
| TUS Capabilities | 1 | Complete |
| TUS Create Upload | 6 | Complete |
| TUS Patch Upload | 6 | Complete |
| TUS Complete/Head/Delete | 5 | Complete |
| TUS Upsert & Multi-chunk | 2 | Complete |
| TUS Metadata Parsing | 5 | Complete |
| JWT Token Operations | 15 | Complete |
| Signed URLs with Auth | 6 | Complete |
| Bucket/Object CRUD | 2 | Complete |
| Error Response Format | 4 | Complete |
| Public Bucket Access | 2 | Complete |
| Concurrent Operations | 1 | Complete |

**Total REST Unit Tests: 63 (All Passing)**

#### Key Implementation Notes

1. **Signed URL Support**: Implemented server-level signed URL generation using HMAC-SHA256 tokens. Signed URLs work independently of storage backend capabilities.

2. **Render Endpoint**: Added `/object/render/:bucketName/*path` endpoint that validates signed URL tokens and serves files without requiring authentication headers.

3. **Range Header Support**: Full HTTP Range header support including:
   - `bytes=start-end` (exact range)
   - `bytes=start-` (from position to end)
   - `bytes=-N` (last N bytes/suffix range)
   - Returns 206 Partial Content with proper Content-Range headers

4. **TUS Protocol**: Complete TUS 1.0.0 protocol implementation for resumable uploads.

5. **Error Format**: All errors follow Supabase Storage API format with `statusCode`, `error`, and `message` fields.

#### S3 API Tests (pkg/storage/transport/s3)

| Category | Tests | Status |
|----------|-------|--------|
| Bucket Operations | 10 | Complete |
| Object Operations | 14 | Complete |
| Multipart Upload | 8 | Complete |
| Authentication (SigV4) | 6 | Complete |

**Total S3 Tests: 38 (All Passing)**

### Integration Test Requirements

Integration tests (`supabase_compat_test.go`) compare responses between Supabase Storage and Localbase to verify 100% API compatibility. These tests require both servers running simultaneously.

#### Prerequisites

1. **Supabase Local** - Running on default ports
2. **Localbase Server** - Running on port 54321
3. **Environment Variables** (optional - defaults to local development values)

#### Environment Configuration

```bash
# Supabase endpoints (defaults)
export SUPABASE_STORAGE_URL="http://127.0.0.1:54421/storage/v1"
export SUPABASE_S3_URL="http://127.0.0.1:54421/storage/v1/s3"
export SUPABASE_SERVICE_ROLE_KEY="<your-supabase-service-role-key>"

# Localbase endpoints (defaults)
export LOCALBASE_STORAGE_URL="http://localhost:54321/storage/v1"
export LOCALBASE_S3_URL="http://localhost:54321/s3"
export LOCALBASE_API_KEY="<your-localbase-api-key>"
```

#### Running Integration Tests

```bash
# 1. Start Supabase Local
supabase start

# 2. Start Localbase (in separate terminal)
go run ./blueprints/localbase/cmd/localbase

# 3. Run integration tests
go test -tags=integration ./pkg/storage/transport/rest/...

# 4. Run with verbose output
go test -tags=integration -v ./pkg/storage/transport/rest/...
```

#### Integration Test Coverage

The integration tests cover:
- Bucket CRUD operations (create, list, get, delete)
- Object upload with various content types
- Object download with Range header support
- Object list with prefix/limit/offset
- Object move and copy operations
- Object delete (single and batch)
- Object info retrieval
- Signed URL creation
- Public bucket access patterns
- Edge cases (special characters, empty files, etc.)
- Error response format consistency

### Test Command Reference

```bash
# Run all REST transport tests
go test ./pkg/storage/transport/rest/...

# Run with verbose output
go test -v ./pkg/storage/transport/rest/...

# Run specific test category
go test -v ./pkg/storage/transport/rest/... -run "TestBucket"
go test -v ./pkg/storage/transport/rest/... -run "TestSignedURL"
go test -v ./pkg/storage/transport/rest/... -run "TestTUS"

# Run S3 transport tests
go test ./pkg/storage/transport/s3/...

# Run integration tests (requires running servers)
go test -tags=integration ./pkg/storage/transport/rest/...
```

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-01-16 | Claude Code | Initial draft |
| 1.1 | 2026-01-16 | Claude Code | Added implementation status, signed URL support, render endpoint |
| 1.2 | 2026-01-16 | Claude Code | Updated test counts (63 REST, 38 S3), detailed integration test documentation |
