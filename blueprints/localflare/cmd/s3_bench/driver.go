package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// DriverConfig holds the configuration for a single S3 driver.
type DriverConfig struct {
	Name      string
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	Enabled   bool
}

// DefaultDrivers returns all configured S3 drivers matching docker-compose.yaml.
func DefaultDrivers() []*DriverConfig {
	return []*DriverConfig{
		{
			Name:      "liteio",
			Endpoint:  "http://localhost:9200",
			AccessKey: "liteio",
			SecretKey: "liteio123",
			Bucket:    "test-bucket",
			Region:    "us-east-1",
			Enabled:   true,
		},
		{
			Name:      "minio",
			Endpoint:  "http://localhost:9000",
			AccessKey: "minioadmin",
			SecretKey: "minioadmin",
			Bucket:    "test-bucket",
			Region:    "us-east-1",
			Enabled:   true,
		},
		{
			Name:      "rustfs",
			Endpoint:  "http://localhost:9100",
			AccessKey: "rustfsadmin",
			SecretKey: "rustfsadmin",
			Bucket:    "test-bucket",
			Region:    "us-east-1",
			Enabled:   true,
		},
		{
			Name:      "liteio_mem",
			Endpoint:  "http://localhost:9201",
			AccessKey: "liteio",
			SecretKey: "liteio123",
			Bucket:    "test-bucket",
			Region:    "us-east-1",
			Enabled:   true,
		},
		{
			Name:      "seaweedfs",
			Endpoint:  "http://localhost:8333",
			AccessKey: "admin",
			SecretKey: "adminpassword",
			Bucket:    "test-bucket",
			Region:    "us-east-1",
			Enabled:   true,
		},
		{
			Name:      "localstack",
			Endpoint:  "http://localhost:4566",
			AccessKey: "test",
			SecretKey: "test",
			Bucket:    "test-bucket",
			Region:    "us-east-1",
			Enabled:   true,
		},
	}
}

// FilterDrivers returns only the drivers matching the given names.
// If names is empty, returns all enabled drivers.
func FilterDrivers(drivers []*DriverConfig, names []string) []*DriverConfig {
	if len(names) == 0 {
		var enabled []*DriverConfig
		for _, d := range drivers {
			if d.Enabled {
				enabled = append(enabled, d)
			}
		}
		return enabled
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}

	var filtered []*DriverConfig
	for _, d := range drivers {
		if nameSet[d.Name] && d.Enabled {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// S3Client wraps the AWS S3 client with driver info.
type S3Client struct {
	Client *s3.Client
	Driver *DriverConfig
}

// NewS3Client creates a new S3 client for the given driver.
func NewS3Client(ctx context.Context, driver *DriverConfig) (*S3Client, error) {
	// Create custom HTTP client with connection pooling
	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
		Timeout: 5 * time.Minute,
	}

	// Create credentials provider
	creds := credentials.NewStaticCredentialsProvider(
		driver.AccessKey,
		driver.SecretKey,
		"",
	)

	// Create AWS config
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(driver.Region),
		config.WithCredentialsProvider(creds),
		config.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Create S3 client with custom endpoint
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(driver.Endpoint)
		o.UsePathStyle = true // Required for most S3-compatible services
	})

	return &S3Client{
		Client: client,
		Driver: driver,
	}, nil
}

// CheckAvailable checks if the driver is available.
func (c *S3Client) CheckAvailable(ctx context.Context) error {
	// Try to head the bucket
	_, err := c.Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.Driver.Bucket),
	})
	if err != nil {
		// Try to create the bucket if it doesn't exist
		_, createErr := c.Client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(c.Driver.Bucket),
		})
		if createErr != nil {
			return fmt.Errorf("bucket not accessible: %w", err)
		}
	}
	return nil
}

// HostPort extracts host:port from the endpoint URL.
func (d *DriverConfig) HostPort() string {
	// Remove http:// or https:// prefix
	endpoint := d.Endpoint
	if len(endpoint) > 7 && endpoint[:7] == "http://" {
		return endpoint[7:]
	}
	if len(endpoint) > 8 && endpoint[:8] == "https://" {
		return endpoint[8:]
	}
	return endpoint
}

// CheckConnectivity checks TCP connectivity to the driver.
func (d *DriverConfig) CheckConnectivity() error {
	conn, err := net.DialTimeout("tcp", d.HostPort(), 5*time.Second)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}
