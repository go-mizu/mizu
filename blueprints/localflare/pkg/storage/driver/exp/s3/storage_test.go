package s3

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestParseDSN(t *testing.T) {
	tests := []struct {
		name     string
		dsn      string
		wantErr  bool
		check    func(*testing.T, *dsnConfig)
	}{
		{
			name: "AWS S3 with bucket only",
			dsn:  "s3://mybucket",
			check: func(t *testing.T, cfg *dsnConfig) {
				if cfg.bucket != "mybucket" {
					t.Errorf("bucket = %q, want %q", cfg.bucket, "mybucket")
				}
				if cfg.endpoint != "" {
					t.Errorf("endpoint = %q, want empty", cfg.endpoint)
				}
				if cfg.region != "us-east-1" {
					t.Errorf("region = %q, want %q", cfg.region, "us-east-1")
				}
			},
		},
		{
			name: "AWS S3 with region",
			dsn:  "s3://mybucket?region=eu-west-1",
			check: func(t *testing.T, cfg *dsnConfig) {
				if cfg.bucket != "mybucket" {
					t.Errorf("bucket = %q, want %q", cfg.bucket, "mybucket")
				}
				if cfg.region != "eu-west-1" {
					t.Errorf("region = %q, want %q", cfg.region, "eu-west-1")
				}
			},
		},
		{
			name: "MinIO with path style",
			dsn:  "s3://localhost:9000/mybucket?force_path_style=true&insecure=true",
			check: func(t *testing.T, cfg *dsnConfig) {
				if cfg.bucket != "mybucket" {
					t.Errorf("bucket = %q, want %q", cfg.bucket, "mybucket")
				}
				if cfg.endpoint != "http://localhost:9000" {
					t.Errorf("endpoint = %q, want %q", cfg.endpoint, "http://localhost:9000")
				}
				if !cfg.forcePathStyle {
					t.Error("forcePathStyle = false, want true")
				}
				if !cfg.insecure {
					t.Error("insecure = false, want true")
				}
			},
		},
		{
			name: "Custom endpoint with credentials",
			dsn:  "s3://accesskey:secretkey@storage.example.com:9000/data?region=us-west-2",
			check: func(t *testing.T, cfg *dsnConfig) {
				if cfg.accessKey != "accesskey" {
					t.Errorf("accessKey = %q, want %q", cfg.accessKey, "accesskey")
				}
				if cfg.secretKey != "secretkey" {
					t.Errorf("secretKey = %q, want %q", cfg.secretKey, "secretkey")
				}
				if cfg.bucket != "data" {
					t.Errorf("bucket = %q, want %q", cfg.bucket, "data")
				}
				if cfg.region != "us-west-2" {
					t.Errorf("region = %q, want %q", cfg.region, "us-west-2")
				}
			},
		},
		{
			name: "AWS endpoint patterns",
			dsn:  "s3://s3.us-west-2.amazonaws.com/mybucket",
			check: func(t *testing.T, cfg *dsnConfig) {
				// Should treat as AWS, not custom endpoint
				if cfg.endpoint != "" {
					t.Errorf("endpoint = %q, want empty for AWS", cfg.endpoint)
				}
			},
		},
		{
			name:    "Empty DSN",
			dsn:     "",
			wantErr: true,
		},
		{
			name:    "Wrong scheme",
			dsn:     "http://localhost:9000/bucket",
			wantErr: true,
		},
		{
			name: "Session token",
			dsn:  "s3://mybucket?session_token=abc123",
			check: func(t *testing.T, cfg *dsnConfig) {
				if cfg.sessionToken != "abc123" {
					t.Errorf("sessionToken = %q, want %q", cfg.sessionToken, "abc123")
				}
			},
		},
		{
			name: "Disable SSL alias",
			dsn:  "s3://localhost:9000/bucket?disable_ssl=true",
			check: func(t *testing.T, cfg *dsnConfig) {
				if !cfg.insecure {
					t.Error("insecure = false, want true (via disable_ssl)")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseDSN(tt.dsn)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"1", true},
		{"yes", true},
		{"on", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"off", false},
		{"", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseBool(tt.input)
			if got != tt.want {
				t.Errorf("parseBool(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCleanKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"/leading/slash", "leading/slash"},
		{"trailing/slash/", "trailing/slash"},
		{"  spaces  ", "spaces"},
		{"back\\slashes", "back/slashes"},
		{"path/./to/../file", "path/file"},
		{"", ""},
		{".", ""},
		{"/", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cleanKey(tt.input)
			if got != tt.want {
				t.Errorf("cleanKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsAWSEndpoint(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"s3.amazonaws.com", true},
		{"s3.us-west-2.amazonaws.com", true},
		{"s3-us-west-2.amazonaws.com", true},
		{"bucket.s3.amazonaws.com", true},
		{"localhost:9000", false},
		{"minio.example.com", false},
		{"storage.example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := isAWSEndpoint(tt.host)
			if got != tt.want {
				t.Errorf("isAWSEndpoint(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

// Integration tests - require S3-compatible storage running

func skipIfNoS3(t *testing.T) string {
	endpoint := os.Getenv("S3_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("S3_TEST_ENDPOINT not set, skipping integration test")
	}
	return endpoint
}

func getTestDSN(t *testing.T) string {
	endpoint := skipIfNoS3(t)
	accessKey := os.Getenv("S3_TEST_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "minioadmin"
	}
	secretKey := os.Getenv("S3_TEST_SECRET_KEY")
	if secretKey == "" {
		secretKey = "minioadmin"
	}
	bucket := os.Getenv("S3_TEST_BUCKET")
	if bucket == "" {
		bucket = "test-bucket"
	}
	region := os.Getenv("S3_TEST_REGION")
	if region == "" {
		region = "us-east-1"
	}
	insecure := os.Getenv("S3_TEST_INSECURE")
	if insecure == "" {
		insecure = "true"
	}

	return "s3://" + accessKey + ":" + secretKey + "@" + endpoint + "/" + bucket + "?region=" + region + "&force_path_style=true&insecure=" + insecure
}

func TestIntegration_Open(t *testing.T) {
	dsn := getTestDSN(t)

	d := &driver{}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	// Check features
	features := st.Features()
	if !features["multipart"] {
		t.Error("expected multipart feature")
	}
	if !features["signed_url"] {
		t.Error("expected signed_url feature")
	}
}

func TestIntegration_BucketOperations(t *testing.T) {
	dsn := getTestDSN(t)

	d := &driver{}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	st, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	// Get default bucket
	b := st.Bucket("")
	if b == nil {
		t.Fatal("Bucket returned nil")
	}

	// Verify bucket info
	_, err = b.Info(ctx)
	if err != nil {
		t.Logf("Bucket info error (may not exist yet): %v", err)
	}
}
