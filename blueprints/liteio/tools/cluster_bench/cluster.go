// Package main provides cluster lifecycle management for benchmarking
// MinIO, RustFS, SeaweedFS, and Herd in 3-node S3 cluster mode.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const baseDir = "/tmp/cluster-bench"

// Cluster manages a 3-node storage cluster.
type Cluster struct {
	Name      string
	S3Port    int
	AccessKey string
	SecretKey string
	HealthURL string
	DataDir   string
	procs     []*exec.Cmd
}

func (c *Cluster) S3Endpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%d", c.S3Port)
}

func (c *Cluster) DSN() string {
	return fmt.Sprintf("s3://%s:%s@127.0.0.1:%d/test-bucket?insecure=true&force_path_style=true&unsigned_payload=true",
		c.AccessKey, c.SecretKey, c.S3Port)
}

func (c *Cluster) Stop() {
	for _, p := range c.procs {
		if p.Process != nil {
			p.Process.Signal(syscall.SIGTERM)
		}
	}
	// Wait briefly for graceful shutdown
	time.Sleep(2 * time.Second)
	for _, p := range c.procs {
		if p.Process != nil {
			p.Process.Kill()
		}
	}
}

func (c *Cluster) Cleanup() error {
	return os.RemoveAll(c.DataDir)
}

func (c *Cluster) WaitReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		req, _ := http.NewRequestWithContext(reqCtx, "GET", c.HealthURL, nil)
		resp, err := http.DefaultClient.Do(req)
		cancel()
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("%s: not ready after %v", c.Name, timeout)
}

// startProcess starts a background process with given args and env.
func startProcess(name string, args []string, env []string, logPrefix string) (*exec.Cmd, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), env...)

	// Pipe stdout/stderr to files for debugging
	logDir := filepath.Join(baseDir, "logs")
	os.MkdirAll(logDir, 0o755)

	logFile, err := os.Create(filepath.Join(logDir, logPrefix+".log"))
	if err != nil {
		return nil, err
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return nil, fmt.Errorf("start %s: %w", name, err)
	}

	// Don't wait — let it run in background
	go func() {
		cmd.Wait()
		logFile.Close()
	}()

	return cmd, nil
}

// NewMinIOCluster creates a MinIO erasure-coded cluster with 6 drives.
// Uses SNMD (Single Node Multiple Drive) mode since all drives are on localhost.
// MINIO_CI_CD=true bypasses macOS APFS drive-uniqueness check.
func NewMinIOCluster() (*Cluster, error) {
	c := &Cluster{
		Name:      "minio_cluster",
		S3Port:    9000,
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		HealthURL: "http://127.0.0.1:9000/minio/health/live",
		DataDir:   filepath.Join(baseDir, "minio"),
	}

	// Create 6 drive directories (erasure set of 6)
	drives := make([]string, 6)
	for i := 0; i < 6; i++ {
		drives[i] = filepath.Join(c.DataDir, fmt.Sprintf("vol%d", i+1))
		if err := os.MkdirAll(drives[i], 0o755); err != nil {
			return nil, err
		}
	}

	env := []string{
		"MINIO_ROOT_USER=minioadmin",
		"MINIO_ROOT_PASSWORD=minioadmin",
		"MINIO_CI_CD=true", // Bypass drive uniqueness check on macOS APFS
	}

	args := []string{
		"server",
		"--address", ":9000",
		"--console-address", ":9010",
	}
	args = append(args, drives...)

	proc, err := startProcess("minio", args, env, "minio")
	if err != nil {
		return nil, err
	}
	c.procs = append(c.procs, proc)

	return c, nil
}

// NewRustFSCluster creates a RustFS server with 6 local drives.
// RustFS uses positional args for volume directories.
func NewRustFSCluster() (*Cluster, error) {
	c := &Cluster{
		Name:      "rustfs_cluster",
		S3Port:    9100,
		AccessKey: "rustfsadmin",
		SecretKey: "rustfsadmin",
		HealthURL: "http://127.0.0.1:9100/minio/health/live",
		DataDir:   filepath.Join(baseDir, "rustfs"),
	}

	// Create 6 drive directories
	drives := make([]string, 6)
	for i := 0; i < 6; i++ {
		drives[i] = filepath.Join(c.DataDir, fmt.Sprintf("vol%d", i+1))
		if err := os.MkdirAll(drives[i], 0o755); err != nil {
			return nil, err
		}
	}

	args := []string{
		"--address", ":9100",
		"--console-address", ":9110",
		"--access-key", "rustfsadmin",
		"--secret-key", "rustfsadmin",
	}
	args = append(args, drives...)

	proc, err := startProcess("rustfs", args, nil, "rustfs")
	if err != nil {
		return nil, err
	}
	c.procs = append(c.procs, proc)

	return c, nil
}

// NewSeaweedFSCluster creates a SeaweedFS cluster (1 master + 3 volumes + filer/S3).
func NewSeaweedFSCluster() (*Cluster, error) {
	c := &Cluster{
		Name:      "seaweedfs_cluster",
		S3Port:    8333,
		AccessKey: "admin",
		SecretKey: "adminpassword",
		HealthURL: "http://127.0.0.1:9333/cluster/status",
		DataDir:   filepath.Join(baseDir, "seaweedfs"),
	}

	// Create data dirs
	for _, sub := range []string{"master", "vol1", "vol2", "vol3"} {
		if err := os.MkdirAll(filepath.Join(c.DataDir, sub), 0o755); err != nil {
			return nil, err
		}
	}

	// Write S3 config
	s3Config := map[string]any{
		"identities": []map[string]any{
			{
				"name": "admin",
				"credentials": []map[string]string{
					{"accessKey": "admin", "secretKey": "adminpassword"},
				},
				"actions": []string{"Admin", "Read", "Write", "List", "Tagging"},
			},
		},
	}
	s3ConfigPath := filepath.Join(c.DataDir, "s3.json")
	f, err := os.Create(s3ConfigPath)
	if err != nil {
		return nil, err
	}
	json.NewEncoder(f).Encode(s3Config)
	f.Close()

	// 1. Master
	proc, err := startProcess("weed", []string{
		"master",
		"-mdir=" + filepath.Join(c.DataDir, "master"),
		"-port=9333",
		"-volumeSizeLimitMB=1024",
	}, nil, "seaweedfs-master")
	if err != nil {
		return nil, err
	}
	c.procs = append(c.procs, proc)

	// Wait for master to start
	time.Sleep(2 * time.Second)

	// 2. Volume servers (3 nodes)
	for i := 1; i <= 3; i++ {
		port := 8079 + i // 8080, 8081, 8082
		proc, err := startProcess("weed", []string{
			"volume",
			"-mserver=localhost:9333",
			fmt.Sprintf("-dir=%s/vol%d", c.DataDir, i),
			fmt.Sprintf("-port=%d", port),
			fmt.Sprintf("-rack=rack%d", i),
		}, nil, fmt.Sprintf("seaweedfs-vol%d", i))
		if err != nil {
			c.Stop()
			return nil, err
		}
		c.procs = append(c.procs, proc)
	}

	// Wait for volumes to register
	time.Sleep(2 * time.Second)

	// 3. Filer + S3 gateway
	proc, err = startProcess("weed", []string{
		"filer",
		"-master=localhost:9333",
		"-port=8888",
		"-s3",
		"-s3.port=8333",
		"-s3.config=" + s3ConfigPath,
	}, nil, "seaweedfs-filer")
	if err != nil {
		c.Stop()
		return nil, err
	}
	c.procs = append(c.procs, proc)

	return c, nil
}

// NewHerdCluster creates a 3-node embedded Herd cluster with S3 gateway.
func NewHerdCluster() (*Cluster, error) {
	c := &Cluster{
		Name:      "herd_cluster",
		S3Port:    9230,
		AccessKey: "herd",
		SecretKey: "herd123",
		HealthURL: "http://127.0.0.1:9230/",
		DataDir:   filepath.Join(baseDir, "herd"),
	}

	if err := os.MkdirAll(c.DataDir, 0o755); err != nil {
		return nil, err
	}

	// Herd embedded 3-node: single process, zero network overhead.
	// sync=batch: CRC enabled, all writes go to volume (durable), buffer ring for batching.
	// inline_kb=8: read cache in memory + volume write for durability.
	proc, err := startProcess("herd", []string{
		"-nodes", "3",
		"-listen", ":9230",
		"-data-dir", c.DataDir,
		"-stripes", "16",
		"-sync", "batch",
		"-inline-kb", "8",
		"-prealloc", "1024",
		"-bufsize", "16777216",
		"-access-key", "herd",
		"-secret-key", "herd123",
		"-no-log",
	}, nil, "herd")
	if err != nil {
		return nil, err
	}
	c.procs = append(c.procs, proc)

	return c, nil
}

// createBucket creates a test bucket via S3 PutBucket.
func createBucket(endpoint, accessKey, secretKey, bucket string) error {
	// Use a simple PUT request to create the bucket.
	// The S3 PutBucket is just PUT /{bucket} with auth.
	// For simplicity, we'll use the aws CLI or a direct HTTP call.
	// Since all our systems support unsigned payload or simple auth,
	// use the aws CLI if available, otherwise fall back to curl.

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "aws", "s3", "mb",
		fmt.Sprintf("s3://%s", bucket),
		"--endpoint-url", endpoint,
	)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", accessKey),
		fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", secretKey),
		"AWS_DEFAULT_REGION=us-east-1",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Bucket may already exist — that's OK
		if strings.Contains(string(out), "BucketAlreadyOwnedByYou") ||
			strings.Contains(string(out), "BucketAlreadyExists") {
			return nil
		}
		return fmt.Errorf("create bucket %s at %s: %s: %w", bucket, endpoint, string(out), err)
	}
	return nil
}

// checkHealth performs a quick S3 health check by listing buckets.
func checkHealth(endpoint, accessKey, secretKey string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", endpoint+"/", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("health check: HTTP %d", resp.StatusCode)
	}
	return nil
}
