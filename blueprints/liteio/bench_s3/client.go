package bench_s3

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// NewS3Client creates an S3 client configured for the given endpoint.
// Uses optimized HTTP transport for benchmark accuracy.
func NewS3Client(ep Endpoint) *s3.Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   200,
		IdleConnTimeout:       90 * time.Second,
		DisableCompression:    true,
		ExpectContinueTimeout: 0, // Disable expect-100-continue for PUT perf
		ForceAttemptHTTP2:     false,
		ResponseHeaderTimeout: 30 * time.Second,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}

	endpointURL := fmt.Sprintf("http://%s", ep.Host)

	client := s3.New(s3.Options{
		BaseEndpoint: &endpointURL,
		Region:       "us-east-1",
		Credentials:  credentials.NewStaticCredentialsProvider(ep.AccessKey, ep.SecretKey, ""),
		UsePathStyle: true,
		HTTPClient:   httpClient,
		RetryMaxAttempts: 0, // No retries — measure raw latency
	})

	return client
}

// CheckEndpoint verifies an S3 endpoint is reachable by listing buckets.
func CheckEndpoint(ctx context.Context, ep Endpoint) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client := NewS3Client(ep)
	_, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	return err == nil
}

// EnsureBucket creates the bucket if it doesn't exist.
func EnsureBucket(ctx context.Context, client *s3.Client, bucket string) error {
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		// Ignore "already exists" errors
		_, listErr := client.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(bucket),
		})
		if listErr != nil {
			return fmt.Errorf("create bucket %s: %w", bucket, err)
		}
	}
	return nil
}
