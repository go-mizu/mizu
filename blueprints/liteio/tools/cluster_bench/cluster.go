// Package main provides cluster lifecycle management for benchmarking
// MinIO, RustFS, SeaweedFS, and Herd in true multi-node S3 cluster mode.
//
// Architecture (fair comparison — all systems use proper clustering):
//   - MinIO:     4-node distributed (EC:4) + HAProxy round-robin   (5 procs)
//   - RustFS:    4-node distributed (EC:4) + HAProxy round-robin   (5 procs)
//   - SeaweedFS: 1 master + 3 volumes + 1 filer/S3 gateway         (5 procs)
//   - Herd:      4 TCP node servers + 1 S3 gateway (rendezvous)     (5 procs)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const baseDir = "/tmp/cluster-bench"

// Cluster manages a multi-node storage cluster behind HAProxy.
type Cluster struct {
	Name         string
	S3Port       int // HAProxy frontend port
	AccessKey    string
	SecretKey    string
	HealthURL    string
	DataDir      string
	procs        []*exec.Cmd
	backendAddrs []string // direct backend addresses (host:port)
	endpointMode string   // "roundrobin" or "rendezvous"
}

// BackendAddrs returns the direct backend addresses for client-side LB.
func (c *Cluster) BackendAddrs() []string { return c.backendAddrs }

// EndpointMode returns the endpoint routing mode ("roundrobin" or "rendezvous").
func (c *Cluster) EndpointMode() string { return c.endpointMode }

func (c *Cluster) S3Endpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%d", c.S3Port)
}

func (c *Cluster) DSN() string {
	return fmt.Sprintf("s3://%s:%s@127.0.0.1:%d/test-bucket?insecure=true&force_path_style=true&unsigned_payload=true",
		c.AccessKey, c.SecretKey, c.S3Port)
}

// DirectDSN returns a DSN with multi-endpoint parameters for client-side LB
// (bypasses HAProxy). The S3 driver's MultiEndpointTransport distributes
// requests directly to backend nodes.
func (c *Cluster) DirectDSN() string {
	if len(c.backendAddrs) == 0 {
		return c.DSN()
	}
	// Use first backend as base endpoint; MultiEndpointTransport overrides host per request.
	return fmt.Sprintf("s3://%s:%s@%s/test-bucket?insecure=true&force_path_style=true&unsigned_payload=true&endpoints=%s&endpoint_mode=%s",
		c.AccessKey, c.SecretKey, c.backendAddrs[0],
		strings.Join(c.backendAddrs, ","), c.endpointMode)
}

func (c *Cluster) Stop() {
	for _, p := range c.procs {
		if p.Process != nil {
			p.Process.Signal(syscall.SIGTERM)
		}
	}
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

// ---------------------------------------------------------------------------
// HAProxy management
// ---------------------------------------------------------------------------

// writeHAProxyConfig writes an HAProxy config for load-balancing S3 backends.
func writeHAProxyConfig(cfgPath string, frontendPort int, backends []haproxyBackend, balanceMode string) error {
	var sb strings.Builder
	sb.WriteString("global\n")
	sb.WriteString("    maxconn 50000\n")
	sb.WriteString("    log stdout format raw local0\n\n")
	sb.WriteString("defaults\n")
	sb.WriteString("    mode http\n")
	sb.WriteString("    timeout connect 5s\n")
	sb.WriteString("    timeout client 300s\n")
	sb.WriteString("    timeout server 60s\n")
	sb.WriteString("    option forwardfor\n")
	sb.WriteString("    option dontlognull\n")
	sb.WriteString("    retries 3\n\n")
	sb.WriteString("frontend s3_front\n")
	sb.WriteString(fmt.Sprintf("    bind *:%d\n", frontendPort))
	sb.WriteString("    default_backend s3_back\n\n")
	sb.WriteString("backend s3_back\n")
	sb.WriteString(fmt.Sprintf("    balance %s\n", balanceMode))
	if balanceMode == "uri" {
		sb.WriteString("    hash-type consistent\n")
	}
	for _, b := range backends {
		sb.WriteString(fmt.Sprintf("    server %s 127.0.0.1:%d check inter 2s rise 1 fall 2\n", b.name, b.port))
	}
	return os.WriteFile(cfgPath, []byte(sb.String()), 0o644)
}

type haproxyBackend struct {
	name string
	port int
}

// startHAProxy starts an HAProxy process with the given config.
func startHAProxy(cfgPath, logPrefix string) (*exec.Cmd, error) {
	return startProcess("haproxy", []string{"-f", cfgPath}, nil, logPrefix)
}

// waitForPort waits until a TCP port is accepting connections.
func waitForPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("port %d not ready after %v", port, timeout)
}

// ---------------------------------------------------------------------------
// Process management
// ---------------------------------------------------------------------------

func startProcess(name string, args []string, env []string, logPrefix string) (*exec.Cmd, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), env...)

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

	go func() {
		cmd.Wait()
		logFile.Close()
	}()

	return cmd, nil
}

// ---------------------------------------------------------------------------
// MinIO: 4 SNMD instances behind HAProxy
// ---------------------------------------------------------------------------

// NewMinIOCluster creates a 4-node MinIO distributed cluster behind HAProxy.
// 4 nodes × 2 drives = 8 drives, erasure coded (EC:4).
// Each process receives the full endpoint list; MinIO routes internally.
// HAProxy on :9000 round-robins across nodes.
func NewMinIOCluster() (*Cluster, error) {
	c := &Cluster{
		Name:         "minio_cluster",
		S3Port:       9000,
		AccessKey:    "minioadmin",
		SecretKey:    "minioadmin",
		HealthURL:    "http://127.0.0.1:9000/minio/health/live",
		DataDir:      filepath.Join(baseDir, "minio"),
		endpointMode: "roundrobin",
	}

	// Pre-create all data directories (node-specific paths avoid filesystem collisions).
	for i := 0; i < 4; i++ {
		for d := 1; d <= 2; d++ {
			dir := filepath.Join(c.DataDir, fmt.Sprintf("node%d", i), fmt.Sprintf("vol%d", d))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, err
			}
		}
	}

	env := []string{
		"MINIO_ROOT_USER=minioadmin",
		"MINIO_ROOT_PASSWORD=minioadmin",
		"MINIO_CI_CD=true",
	}

	// Build per-node endpoints (each node has node-specific path to avoid
	// filesystem collisions on single machine). Expansion {1...2} for volumes.
	var endpoints []string
	for j := 0; j < 4; j++ {
		ep := fmt.Sprintf("http://127.0.0.1:%d%s/node%d/vol{1...2}", 9050+j, c.DataDir, j)
		endpoints = append(endpoints, ep)
	}

	var backends []haproxyBackend
	for i := 0; i < 4; i++ {
		port := 9050 + i
		args := []string{
			"server",
			"--address", fmt.Sprintf(":%d", port),
			"--console-address", fmt.Sprintf(":%d", 9060+i),
		}
		args = append(args, endpoints...)

		proc, err := startProcess("minio", args, env, fmt.Sprintf("minio-node%d", i))
		if err != nil {
			c.Stop()
			return nil, err
		}
		c.procs = append(c.procs, proc)
		backends = append(backends, haproxyBackend{fmt.Sprintf("minio%d", i), port})
		c.backendAddrs = append(c.backendAddrs, fmt.Sprintf("127.0.0.1:%d", port))
	}

	// Wait for all MinIO instances to be ready.
	for _, b := range backends {
		if err := waitForPort(b.port, 30*time.Second); err != nil {
			c.Stop()
			return nil, fmt.Errorf("minio node %s: %w", b.name, err)
		}
	}

	// Write HAProxy config and start it.
	cfgPath := filepath.Join(c.DataDir, "haproxy.cfg")
	if err := writeHAProxyConfig(cfgPath, c.S3Port, backends, "roundrobin"); err != nil {
		c.Stop()
		return nil, err
	}
	proc, err := startHAProxy(cfgPath, "minio-haproxy")
	if err != nil {
		c.Stop()
		return nil, err
	}
	c.procs = append(c.procs, proc)

	return c, nil
}

// ---------------------------------------------------------------------------
// RustFS: 4 SNMD instances behind HAProxy
// ---------------------------------------------------------------------------

// NewRustFSCluster creates a 4-node RustFS distributed cluster behind HAProxy.
// RustFS is a MinIO fork; same distributed endpoint syntax.
// 4 nodes × 2 drives = 8 drives, erasure coded.
func NewRustFSCluster() (*Cluster, error) {
	c := &Cluster{
		Name:         "rustfs_cluster",
		S3Port:       9100,
		AccessKey:    "rustfsadmin",
		SecretKey:    "rustfsadmin",
		HealthURL:    "http://127.0.0.1:9100/minio/health/live",
		DataDir:      filepath.Join(baseDir, "rustfs"),
		endpointMode: "roundrobin",
	}

	// Pre-create node-specific data directories.
	for i := 0; i < 4; i++ {
		for d := 1; d <= 2; d++ {
			dir := filepath.Join(c.DataDir, fmt.Sprintf("node%d", i), fmt.Sprintf("vol%d", d))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, err
			}
		}
	}

	// Build per-node endpoints (same syntax as MinIO distributed mode).
	var endpoints []string
	for j := 0; j < 4; j++ {
		ep := fmt.Sprintf("http://127.0.0.1:%d%s/node%d/vol{1...2}", 9150+j, c.DataDir, j)
		endpoints = append(endpoints, ep)
	}

	env := []string{
		"MINIO_ROOT_USER=rustfsadmin",
		"MINIO_ROOT_PASSWORD=rustfsadmin",
		"MINIO_CI_CD=true",
	}

	var backends []haproxyBackend
	for i := 0; i < 4; i++ {
		port := 9150 + i
		args := []string{
			"server",
			"--address", fmt.Sprintf(":%d", port),
			"--console-address", fmt.Sprintf(":%d", 9160+i),
		}
		args = append(args, endpoints...)

		proc, err := startProcess("rustfs", args, env, fmt.Sprintf("rustfs-node%d", i))
		if err != nil {
			c.Stop()
			return nil, err
		}
		c.procs = append(c.procs, proc)
		backends = append(backends, haproxyBackend{fmt.Sprintf("rustfs%d", i), port})
		c.backendAddrs = append(c.backendAddrs, fmt.Sprintf("127.0.0.1:%d", port))
	}

	for _, b := range backends {
		if err := waitForPort(b.port, 30*time.Second); err != nil {
			c.Stop()
			return nil, fmt.Errorf("rustfs node %s: %w", b.name, err)
		}
	}

	cfgPath := filepath.Join(c.DataDir, "haproxy.cfg")
	if err := writeHAProxyConfig(cfgPath, c.S3Port, backends, "roundrobin"); err != nil {
		c.Stop()
		return nil, err
	}
	proc, err := startHAProxy(cfgPath, "rustfs-haproxy")
	if err != nil {
		c.Stop()
		return nil, err
	}
	c.procs = append(c.procs, proc)

	return c, nil
}

// ---------------------------------------------------------------------------
// SeaweedFS: 1 master + 3 volumes + 4 filer/S3 behind HAProxy
// ---------------------------------------------------------------------------

// NewSeaweedFSCluster creates a SeaweedFS cluster:
// 1 master, 3 volume servers, 1 filer+S3 gateway (no HAProxy).
// This is the standard SeaweedFS architecture: filer is the native S3 gateway.
// 5 processes total: 1 master + 3 volumes + 1 filer (fair comparison with others).
func NewSeaweedFSCluster() (*Cluster, error) {
	c := &Cluster{
		Name:         "seaweedfs_cluster",
		S3Port:       8333,
		AccessKey:    "admin",
		SecretKey:    "adminpassword",
		HealthURL:    "http://127.0.0.1:9333/cluster/status",
		DataDir:      filepath.Join(baseDir, "seaweedfs"),
		endpointMode: "roundrobin",
	}

	// Create all data dirs.
	for _, sub := range []string{"master", "vol1", "vol2", "vol3", "filer"} {
		if err := os.MkdirAll(filepath.Join(c.DataDir, sub), 0o755); err != nil {
			return nil, err
		}
	}

	// Write S3 config.
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

	// Master needs time for Raft leader election before gRPC port opens.
	if err := waitForPort(9333, 30*time.Second); err != nil {
		c.Stop()
		return nil, fmt.Errorf("seaweedfs master: %w", err)
	}
	time.Sleep(3 * time.Second) // Let gRPC port (19333) start after Raft election.

	// 2. Volume servers (3 data nodes)
	for i := 1; i <= 3; i++ {
		port := 8079 + i
		grpcPort := 18079 + i
		proc, err := startProcess("weed", []string{
			"volume",
			"-mserver=127.0.0.1:9333",
			"-ip.bind=0.0.0.0",
			fmt.Sprintf("-dir=%s/vol%d", c.DataDir, i),
			fmt.Sprintf("-port=%d", port),
			fmt.Sprintf("-port.grpc=%d", grpcPort),
			fmt.Sprintf("-rack=rack%d", i),
			"-max=0",
		}, nil, fmt.Sprintf("seaweedfs-vol%d", i))
		if err != nil {
			c.Stop()
			return nil, err
		}
		c.procs = append(c.procs, proc)
	}

	// Wait for volumes to register.
	for i := 1; i <= 3; i++ {
		if err := waitForPort(8079+i, 30*time.Second); err != nil {
			c.Stop()
			return nil, fmt.Errorf("seaweedfs volume %d: %w", i, err)
		}
	}
	time.Sleep(2 * time.Second) // Extra wait for volume registration with master.

	// 3. Single filer + S3 gateway (native entry point, no HAProxy needed).
	proc, err = startProcess("weed", []string{
		"filer",
		"-master=127.0.0.1:9333",
		"-ip.bind=0.0.0.0",
		"-port=8880",
		"-port.grpc=18880",
		fmt.Sprintf("-defaultStoreDir=%s/filer", c.DataDir),
		"-s3",
		fmt.Sprintf("-s3.port=%d", c.S3Port),
		"-s3.config=" + s3ConfigPath,
	}, nil, "seaweedfs-filer")
	if err != nil {
		c.Stop()
		return nil, err
	}
	c.procs = append(c.procs, proc)
	c.backendAddrs = append(c.backendAddrs, fmt.Sprintf("127.0.0.1:%d", c.S3Port))

	// Wait for S3 port.
	if err := waitForPort(c.S3Port, 60*time.Second); err != nil {
		c.Stop()
		return nil, fmt.Errorf("seaweedfs S3 filer: %w", err)
	}

	return c, nil
}

// ---------------------------------------------------------------------------
// Herd: 4 TCP node servers + 1 S3 gateway (no HAProxy)
// ---------------------------------------------------------------------------

// NewHerdCluster creates a Herd TCP cluster:
// 4 node servers (TCP binary protocol) + 1 S3 gateway connecting to peers.
// The S3 gateway uses rendezvous hashing to route keys to nodes.
// No HAProxy — the gateway IS the native entry point.
// 5 processes total (fair comparison with MinIO/RustFS/SeaweedFS).
func NewHerdCluster() (*Cluster, error) {
	c := &Cluster{
		Name:         "herd_cluster",
		S3Port:       9230,
		AccessKey:    "herd",
		SecretKey:    "herd123",
		HealthURL:    "http://127.0.0.1:9230/",
		DataDir:      filepath.Join(baseDir, "herd"),
		endpointMode: "rendezvous",
	}

	// Start 4 TCP node servers (data nodes).
	var peerAddrs []string
	for i := 0; i < 4; i++ {
		port := 9241 + i
		nodeDir := filepath.Join(c.DataDir, fmt.Sprintf("node%d", i))
		if err := os.MkdirAll(nodeDir, 0o755); err != nil {
			return nil, err
		}

		proc, err := startProcess("herd", []string{
			"-node",
			"-listen", fmt.Sprintf(":%d", port),
			"-data-dir", nodeDir,
			"-stripes", "16",
			"-sync", "batch",
			"-inline-kb", "8",
			"-prealloc", "1024",
			"-bufsize", "16777216",
		}, nil, fmt.Sprintf("herd-node%d", i))
		if err != nil {
			c.Stop()
			return nil, err
		}
		c.procs = append(c.procs, proc)
		peerAddrs = append(peerAddrs, fmt.Sprintf("127.0.0.1:%d", port))
		c.backendAddrs = append(c.backendAddrs, fmt.Sprintf("127.0.0.1:%d", port))
	}

	// Wait for all node servers to be ready.
	for i, addr := range peerAddrs {
		port := 9241 + i
		if err := waitForPort(port, 30*time.Second); err != nil {
			c.Stop()
			return nil, fmt.Errorf("herd node%d (%s): %w", i, addr, err)
		}
	}

	// Start S3 gateway connecting to the 4 TCP peers.
	proc, err := startProcess("herd", []string{
		"-listen", fmt.Sprintf(":%d", c.S3Port),
		"-peers", strings.Join(peerAddrs, ","),
		"-access-key", c.AccessKey,
		"-secret-key", c.SecretKey,
		"-no-log",
	}, nil, "herd-gateway")
	if err != nil {
		c.Stop()
		return nil, err
	}
	c.procs = append(c.procs, proc)

	// Wait for S3 gateway to be ready.
	if err := waitForPort(c.S3Port, 30*time.Second); err != nil {
		c.Stop()
		return nil, fmt.Errorf("herd gateway: %w", err)
	}

	return c, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// createBucket creates a test bucket via S3 PutBucket using the aws CLI.
func createBucket(endpoint, accessKey, secretKey, bucket string) error {
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
		if strings.Contains(string(out), "BucketAlreadyOwnedByYou") ||
			strings.Contains(string(out), "BucketAlreadyExists") {
			return nil
		}
		return fmt.Errorf("create bucket %s at %s: %s: %w", bucket, endpoint, string(out), err)
	}
	return nil
}

// checkHealth performs a quick S3 health check.
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
