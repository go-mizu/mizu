// Package s3 implements storage.Driver for S3-compatible object storage.
//
// DSN format:
//
//	s3://endpoint/bucket?region=us-east-1&force_path_style=true
//	s3://localhost:9000/mybucket?insecure=true
//	s3://mybucket?region=us-west-2
//
// When endpoint is omitted or matches AWS patterns, the driver uses AWS S3.
// For self-hosted S3-compatible storage (MinIO, SeaweedFS, etc.), specify
// the endpoint and typically enable force_path_style=true.
package s3

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-mizu/blueprints/localflare/pkg/storage"
)

func init() {
	storage.Register("s3", &driver{})
}

type driver struct{}

func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	cfg, err := parseDSN(dsn)
	if err != nil {
		return nil, err
	}

	awsCfg, err := buildAWSConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("s3: build aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.endpoint)
		}
		o.UsePathStyle = cfg.forcePathStyle
		// Disable response checksum validation for S3-compatible storage
		// (MinIO, SeaweedFS, RustFS, etc. may not return checksums)
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})

	return &store{
		client:        client,
		defaultBucket: cfg.bucket,
		region:        cfg.region,
		endpoint:      cfg.endpoint,
	}, nil
}

// dsnConfig holds parsed DSN configuration.
type dsnConfig struct {
	endpoint       string
	bucket         string
	region         string
	accessKey      string
	secretKey      string
	sessionToken   string
	forcePathStyle bool
	insecure       bool
	useAccelerate  bool
}

// parseDSN parses an S3 DSN string into configuration.
//
// Format examples:
//
//	s3://mybucket                                           # AWS S3, bucket only
//	s3://mybucket?region=us-west-2                          # AWS S3 with region
//	s3://localhost:9000/mybucket?insecure=true              # MinIO/custom endpoint
//	s3://accesskey:secretkey@localhost:9000/mybucket        # With credentials
//
// When host contains a port or doesn't look like a bucket name, it's treated as endpoint.
// Otherwise, host is treated as the bucket name (AWS S3 style).
func parseDSN(dsn string) (*dsnConfig, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("s3: empty dsn")
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("s3: parse dsn: %w", err)
	}

	if u.Scheme != "s3" {
		return nil, fmt.Errorf("s3: unexpected scheme %q", u.Scheme)
	}

	cfg := &dsnConfig{
		region:         "us-east-1",
		forcePathStyle: false,
		insecure:       false,
	}

	// Extract credentials if present
	if u.User != nil {
		cfg.accessKey = u.User.Username()
		cfg.secretKey, _ = u.User.Password()
	}

	// Parse query parameters
	q := u.Query()
	if r := q.Get("region"); r != "" {
		cfg.region = r
	}
	cfg.forcePathStyle = parseBool(q.Get("force_path_style"))
	cfg.insecure = parseBool(q.Get("insecure")) || parseBool(q.Get("disable_ssl"))
	cfg.sessionToken = q.Get("session_token")
	cfg.useAccelerate = parseBool(q.Get("use_accelerate"))

	// Parse host and path
	host := u.Host
	path := strings.Trim(u.Path, "/")

	// Determine if host is an endpoint or bucket name
	// It's an endpoint if:
	// - Contains a port (e.g., localhost:9000)
	// - Looks like an AWS endpoint (contains amazonaws.com, starts with s3.)
	// - Contains a dot and is not just a simple name
	// Otherwise, treat host as bucket name (AWS S3 shorthand: s3://mybucket)
	if host == "" {
		// No host at all - path must be bucket
		cfg.bucket = path
		cfg.endpoint = ""
	} else if strings.Contains(host, ":") {
		// Host has port - it's a custom endpoint
		scheme := "https"
		if cfg.insecure {
			scheme = "http"
		}
		cfg.endpoint = fmt.Sprintf("%s://%s", scheme, host)
		cfg.bucket = path
		cfg.forcePathStyle = true // Custom endpoints typically need path style
	} else if isAWSEndpoint(host) {
		// AWS endpoint pattern
		cfg.endpoint = ""
		cfg.bucket = path
	} else if strings.Contains(host, ".") {
		// Has dots but not AWS - treat as custom endpoint
		scheme := "https"
		if cfg.insecure {
			scheme = "http"
		}
		cfg.endpoint = fmt.Sprintf("%s://%s", scheme, host)
		cfg.bucket = path
	} else {
		// Simple name without port or dots - treat as bucket name (AWS S3 shorthand)
		// s3://mybucket -> bucket=mybucket, endpoint=AWS default
		cfg.bucket = host
		cfg.endpoint = ""
		// If there's also a path, that's an error or the path is ignored
		if path != "" {
			// host/path format - host is bucket, path could be prefix (ignore for now)
			cfg.bucket = host
		}
	}

	return cfg, nil
}

// isAWSEndpoint returns true if the host looks like an AWS S3 endpoint.
func isAWSEndpoint(host string) bool {
	host = strings.ToLower(host)
	return strings.HasSuffix(host, ".amazonaws.com") ||
		strings.HasPrefix(host, "s3.") ||
		strings.HasPrefix(host, "s3-")
}

// parseBool parses common boolean string values.
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
}

// buildAWSConfig creates an AWS configuration from DSN config.
func buildAWSConfig(ctx context.Context, cfg *dsnConfig) (aws.Config, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.region),
	}

	// Use static credentials if provided
	if cfg.accessKey != "" && cfg.secretKey != "" {
		creds := credentials.NewStaticCredentialsProvider(
			cfg.accessKey,
			cfg.secretKey,
			cfg.sessionToken,
		)
		opts = append(opts, config.WithCredentialsProvider(creds))
	}

	// For insecure connections, configure custom HTTP client with proper connection pooling
	if cfg.insecure {
		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig:     nil, // Allow insecure
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				MaxConnsPerHost:     100,
				IdleConnTimeout:     90 * time.Second,
			},
		}
		opts = append(opts, config.WithHTTPClient(httpClient))
	}

	return config.LoadDefaultConfig(ctx, opts...)
}
