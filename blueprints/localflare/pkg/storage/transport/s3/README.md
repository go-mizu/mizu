# S3-Compatible Transport Layer

This package provides an S3-compatible HTTP API on top of Open-Lake's storage abstraction layer. It implements the AWS S3 REST API specification, allowing any S3-compatible client (AWS SDK, CLI tools, GUI applications) to interact with any storage backend supported by Open-Lake.

## Overview

The S3 transport exposes storage backends (local, cloud providers, etc.) through a standards-compliant S3 API. This enables:

- **Universal Client Support**: Use AWS CLI, AWS SDKs (Go, Python, JavaScript, etc.), or GUI tools like Cyberduck
- **Storage Flexibility**: Access any Open-Lake storage driver (local, S3, Azure Blob, Box, Hugging Face, SFTP) through a unified S3 interface
- **Authentication**: Optional AWS Signature Version 4 authentication with configurable credentials
- **Standard Features**: Multipart uploads, range requests, metadata, presigned URLs

## Architecture

```
┌─────────────────┐
│  S3 Clients     │  (AWS SDK, CLI, curl, etc.)
└────────┬────────┘
         │ HTTP/S3 Protocol
         ▼
┌─────────────────┐
│  S3 Transport   │  (this package)
│  - Server       │
│  - Auth (SigV4) │
│  - Handlers     │
└────────┬────────┘
         │ storage.Storage interface
         ▼
┌─────────────────┐
│ Storage Drivers │  (local, s3, azureblob, etc.)
└─────────────────┘
```

### Key Components

- **server.go**: Core server setup, routing, request parsing, authentication
- **handle_bucket.go**: Bucket-level operations (list, create, delete, location)
- **handle_object.go**: Object operations (get, put, delete, head, copy)
- **handle_multipart.go**: Multipart upload operations
- **Signature V4**: AWS authentication support with configurable credential providers

## S3 API Compatibility

### Bucket Operations

| Operation | Method | Endpoint | Status |
|-----------|--------|----------|--------|
| ListBuckets | GET | / | ✅ Supported |
| CreateBucket | PUT | /{bucket} | ✅ Supported |
| DeleteBucket | DELETE | /{bucket} | ✅ Supported |
| HeadBucket | HEAD | /{bucket} | ✅ Supported |
| GetBucketLocation | GET | /{bucket}?location | ✅ Supported |
| ListObjectsV2 | GET | /{bucket}?list-type=2 | ✅ Supported |

### Object Operations

| Operation | Method | Endpoint | Status |
|-----------|--------|----------|--------|
| PutObject | PUT | /{bucket}/{key} | ✅ Supported |
| GetObject | GET | /{bucket}/{key} | ✅ Supported |
| HeadObject | HEAD | /{bucket}/{key} | ✅ Supported |
| DeleteObject | DELETE | /{bucket}/{key} | ✅ Supported |
| CopyObject | PUT | /{bucket}/{key} (with x-amz-copy-source) | ✅ Supported |

### Multipart Upload Operations

| Operation | Method | Endpoint | Status |
|-----------|--------|----------|--------|
| CreateMultipartUpload | POST | /{bucket}/{key}?uploads | ✅ Supported |
| UploadPart | PUT | /{bucket}/{key}?partNumber=N&uploadId=ID | ✅ Supported |
| CompleteMultipartUpload | POST | /{bucket}/{key}?uploadId=ID | ✅ Supported |
| AbortMultipartUpload | DELETE | /{bucket}/{key}?uploadId=ID | ✅ Supported |
| ListParts | GET | /{bucket}/{key}?uploadId=ID | ✅ Supported |
| ListMultipartUploads | GET | /{bucket}?uploads | ⚠️ Returns empty list |

### Advanced Features

- **Range Requests**: Full support for byte-range requests (`Range: bytes=start-end`)
- **Presigned URLs**: Generate and validate presigned GET/PUT URLs
- **Custom Metadata**: x-amz-meta-* headers preserved
- **Content-Type**: Automatic detection and preservation
- **ETags**: MD5-based ETags for object integrity
- **Pagination**: Continuation tokens for large listings

## Usage

### Basic Server Setup

```go
package main

import (
    "context"
    "net/http"

    "github.com/go-mizu/mizu"
    "open-lake.dev/lib/storage/driver/local"
    "open-lake.dev/lib/storage/transport/s3"
)

func main() {
    ctx := context.Background()

    // Open storage backend
    store, _ := local.Open(ctx, "/var/data")

    // Create web app
    app := mizu.New()

    // Register S3 API at /s3 endpoint
    s3.Register(app, "/s3", store, nil)

    // Start server
    http.ListenAndServe(":8080", app)
}
```

### With Authentication

```go
type staticCreds struct {
    creds map[string]*s3.Credential
}

func (c *staticCreds) Lookup(accessKeyID string) (*s3.Credential, error) {
    if cred, ok := c.creds[accessKeyID]; ok {
        return cred, nil
    }
    return nil, errors.New("access denied")
}

func main() {
    // ... setup storage ...

    cfg := &s3.Config{
        Region: "us-east-1",
        Credentials: &staticCreds{
            creds: map[string]*s3.Credential{
                "AKIAIOSFODNN7EXAMPLE": {
                    AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
                    SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
                },
            },
        },
        Signer: &s3.SignerV4{},
    }

    s3.Register(app, "/s3", store, cfg)
}
```

### Configuration Options

```go
type Config struct {
    // Region for Signature V4 scope (default: "us-east-1")
    Region string

    // Endpoint for Location headers (default: derived from request)
    Endpoint string

    // MaxObjectSize limits single-part uploads (0 = unlimited)
    MaxObjectSize int64

    // Clock for timestamps and signature validation (default: time.Now)
    Clock func() time.Time

    // Credentials provider for authentication (nil = no auth)
    Credentials CredentialProvider

    // Signer validates AWS Signature V4 (nil = no validation)
    Signer Signer

    // AllowedSkew for signature time validation (default: 15min)
    AllowedSkew time.Duration

    // Service name in signing scope (default: "s3")
    Service string
}
```

## Client Examples

### AWS CLI

```bash
# Configure endpoint
export AWS_ENDPOINT_URL=http://localhost:8080/s3

# Or use --endpoint-url flag
aws s3 ls --endpoint-url http://localhost:8080/s3

# Create bucket
aws s3 mb s3://my-bucket

# Upload file
aws s3 cp file.txt s3://my-bucket/

# Download file
aws s3 cp s3://my-bucket/file.txt downloaded.txt

# List objects
aws s3 ls s3://my-bucket/

# Sync directory
aws s3 sync ./local-dir s3://my-bucket/prefix/
```

### Go SDK

```go
package main

import (
    "context"
    "bytes"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
    ctx := context.Background()

    cfg, _ := config.LoadDefaultConfig(ctx,
        config.WithRegion("us-east-1"),
        config.WithCredentialsProvider(
            credentials.NewStaticCredentialsProvider("KEY", "SECRET", ""),
        ),
        config.WithBaseEndpoint("http://localhost:8080/s3"),
    )

    client := s3.NewFromConfig(cfg, func(o *s3.Options) {
        o.UsePathStyle = true // Required for non-AWS endpoints
    })

    // Create bucket
    client.CreateBucket(ctx, &s3.CreateBucketInput{
        Bucket: aws.String("my-bucket"),
    })

    // Upload object
    client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String("my-bucket"),
        Key:    aws.String("file.txt"),
        Body:   bytes.NewReader([]byte("hello")),
    })

    // Download object
    resp, _ := client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String("my-bucket"),
        Key:    aws.String("file.txt"),
    })
    defer resp.Body.Close()
}
```

### Python SDK (boto3)

```python
import boto3

s3 = boto3.client(
    's3',
    endpoint_url='http://localhost:8080/s3',
    aws_access_key_id='KEY',
    aws_secret_access_key='SECRET',
    region_name='us-east-1'
)

# Create bucket
s3.create_bucket(Bucket='my-bucket')

# Upload file
s3.upload_file('local.txt', 'my-bucket', 'remote.txt')

# Download file
s3.download_file('my-bucket', 'remote.txt', 'downloaded.txt')

# List objects
response = s3.list_objects_v2(Bucket='my-bucket')
for obj in response.get('Contents', []):
    print(obj['Key'])
```

### curl

```bash
# List buckets (no auth)
curl http://localhost:8080/s3/

# Create bucket
curl -X PUT http://localhost:8080/s3/my-bucket

# Upload object
curl -X PUT http://localhost:8080/s3/my-bucket/file.txt \
  -H "Content-Type: text/plain" \
  -d "hello world"

# Download object
curl http://localhost:8080/s3/my-bucket/file.txt

# Range request
curl http://localhost:8080/s3/my-bucket/file.txt \
  -H "Range: bytes=0-4"

# Delete object
curl -X DELETE http://localhost:8080/s3/my-bucket/file.txt
```

## Multipart Uploads

For large files (>5GB) or parallel uploads, use multipart uploads:

### AWS CLI

```bash
# Multipart upload automatically used for large files
aws s3 cp large-file.bin s3://my-bucket/

# Force multipart with threshold
aws configure set default.s3.multipart_threshold 5MB
```

### Go SDK

```go
// Initiate multipart upload
createResp, _ := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
    Bucket: aws.String("my-bucket"),
    Key:    aws.String("large-file.bin"),
})

uploadID := aws.ToString(createResp.UploadId)

// Upload parts
parts := []types.CompletedPart{}
for i := 1; i <= 3; i++ {
    partResp, _ := client.UploadPart(ctx, &s3.UploadPartInput{
        Bucket:     aws.String("my-bucket"),
        Key:        aws.String("large-file.bin"),
        UploadId:   aws.String(uploadID),
        PartNumber: aws.Int32(int32(i)),
        Body:       bytes.NewReader(partData),
    })

    parts = append(parts, types.CompletedPart{
        PartNumber: aws.Int32(int32(i)),
        ETag:       partResp.ETag,
    })
}

// Complete multipart upload
client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
    Bucket:   aws.String("my-bucket"),
    Key:      aws.String("large-file.bin"),
    UploadId: aws.String(uploadID),
    MultipartUpload: &types.CompletedMultipartUpload{
        Parts: parts,
    },
})
```

## Error Handling

The server returns standard S3 error responses:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Error>
  <Code>NoSuchBucket</Code>
  <Message>The specified bucket does not exist</Message>
</Error>
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| AccessDenied | 403 | Authentication failed or insufficient permissions |
| NoSuchBucket | 404 | Bucket does not exist |
| NoSuchKey | 404 | Object does not exist |
| BucketNotEmpty | 409 | Cannot delete non-empty bucket |
| EntityTooLarge | 413 | Object exceeds MaxObjectSize |
| InvalidRequest | 400 | Malformed request |
| MethodNotAllowed | 405 | HTTP method not supported for this resource |
| NotImplemented | 501 | Feature not implemented |
| InternalError | 500 | Server-side error |

## Testing

The package includes comprehensive tests covering:

- **handle_bucket_test.go**: Bucket operations (create, delete, list, location)
- **handle_object_test.go**: Object operations (put, get, head, delete, copy, ranges)
- **handle_multipart_test.go**: Multipart uploads (create, upload parts, complete, abort)
- **server_test.go**: Authentication, error handling, end-to-end scenarios

Run tests:

```bash
# Run all tests
go test ./lib/storage/transport/s3

# Run with verbose output
go test -v ./lib/storage/transport/s3

# Run specific test
go test -run TestMultipartUpload ./lib/storage/transport/s3

# Run with race detection
go test -race ./lib/storage/transport/s3

# Check coverage
go test -cover ./lib/storage/transport/s3
```

## Security Considerations

### Authentication

- Always enable authentication in production via `Credentials` and `Signer` config
- Use HTTPS/TLS to protect credentials in transit
- Rotate access keys regularly
- Implement rate limiting at the HTTP layer

### Signature V4

The implementation validates:
- Request timestamp (with configurable skew tolerance)
- Canonical request format
- HMAC-SHA256 signature
- Credential scope (region, service, date)

### Input Validation

- Object keys are validated for path traversal
- Content-Length is enforced
- MaxObjectSize limits upload size
- Part numbers validated (1-10000)

## Performance

### Optimizations

- Streaming transfers (no buffering in memory)
- Range request support for partial downloads
- Efficient pagination with continuation tokens
- Backend storage driver optimizations apply

### Benchmarks

```bash
# Run benchmarks
go test -bench=. ./lib/storage/transport/s3

# With memory profiling
go test -bench=. -memprofile=mem.out ./lib/storage/transport/s3
```

## Limitations

### Current Limitations

- **ListMultipartUploads**: Returns empty list (backend limitation)
- **ACLs**: Not implemented (access control via authentication only)
- **Versioning**: Not supported
- **Lifecycle Policies**: Not supported
- **Bucket Policies**: Not implemented
- **Server-Side Encryption**: Headers accepted but not enforced
- **Object Locking**: Not supported
- **Replication**: Not supported

### Backend Requirements

For full functionality, storage backends must implement:

- `storage.Storage`: Bucket and object operations
- `storage.HasMultipart`: Multipart upload support

## Troubleshooting

### Common Issues

**Authentication Errors**

```
Error: AccessDenied: Access Denied
```

- Verify credentials match server configuration
- Check clock sync (signature validation is time-sensitive)
- Ensure region matches server configuration

**404 Errors**

```
Error: NoSuchBucket: The specified bucket does not exist
```

- Verify bucket name spelling
- Check bucket was created successfully
- Ensure using path-style URLs (`UsePathStyle: true` in AWS SDK)

**Connection Refused**

- Check server is running
- Verify correct endpoint URL
- Check firewall rules

### Debug Logging

Enable debug logging in your application:

```go
app := mizu.New()
app.SetLogger(customLogger) // Set custom logger with debug level
```

## Contributing

When contributing to the S3 transport:

1. Ensure all tests pass: `go test ./lib/storage/transport/s3`
2. Add tests for new features
3. Update this README for API changes
4. Follow S3 API specifications: https://docs.aws.amazon.com/s3/
5. Test with AWS SDK clients for compatibility

## References

- [AWS S3 API Reference](https://docs.aws.amazon.com/AmazonS3/latest/API/)
- [AWS Signature Version 4](https://docs.aws.amazon.com/general/latest/gr/signature-version-4.html)
- [Open-Lake Storage Package](../../storage/)
- [Mizu Web Framework](https://github.com/go-mizu/mizu)

## License

See the main Open-Lake repository for license information.
