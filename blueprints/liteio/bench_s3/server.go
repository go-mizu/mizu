package bench_s3

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// LocalServer defines a local S3 server to start for benchmarking.
type LocalServer struct {
	Endpoint  Endpoint
	Binary    string   // absolute path to binary
	Args      []string // command-line arguments
	Env       []string // extra environment variables
	DataDir   string   // temp data directory (will be cleaned up)
	NeedBuild bool     // true = needs go build, false = external binary
}

// ServerProcess is a running server process.
type ServerProcess struct {
	server  LocalServer
	cmds    []*exec.Cmd // may have multiple processes (e.g., SeaweedFS)
	stopped bool
}

// LocalServerConfigs returns all local server configurations.
// Binary paths are resolved dynamically; unavailable binaries are skipped.
func LocalServerConfigs() []LocalServer {
	servers := []LocalServer{
		// MinIO — single-node local
		{
			Endpoint: Endpoint{
				Name: "minio", Host: "localhost:19000",
				AccessKey: "minioadmin", SecretKey: "minioadmin",
			},
			Env: []string{
				"MINIO_ROOT_USER=minioadmin",
				"MINIO_ROOT_PASSWORD=minioadmin",
				"MINIO_BROWSER=off",
			},
		},
		// RustFS — single-node local
		{
			Endpoint: Endpoint{
				Name: "rustfs", Host: "localhost:19100",
				AccessKey: "rustfsadmin", SecretKey: "rustfsadmin",
			},
			Env: []string{
				"RUSTFS_ACCESS_KEY=rustfsadmin",
				"RUSTFS_SECRET_KEY=rustfsadmin",
			},
		},
		// SeaweedFS — all-in-one local (weed server)
		{
			Endpoint: Endpoint{
				Name: "seaweedfs", Host: "localhost:18333",
				AccessKey: "admin", SecretKey: "adminpassword",
			},
		},
		// LiteIO with local driver
		{
			Endpoint: Endpoint{
				Name: "liteio_local", Host: "localhost:19200",
				AccessKey: "bench", SecretKey: "bench123",
			},
			NeedBuild: true,
		},
		// LiteIO with herd driver
		{
			Endpoint: Endpoint{
				Name: "liteio_herd", Host: "localhost:19230",
				AccessKey: "bench", SecretKey: "bench123",
			},
			NeedBuild: true,
		},
	}
	return servers
}

// ResolveBinaries finds external binaries and builds Go binaries.
// Returns which servers are available and sets their Binary/Args/DataDir fields.
func ResolveBinaries(servers []LocalServer, log func(string, ...any)) ([]LocalServer, func()) {
	var available []LocalServer
	var cleanupFns []func()

	// Check for external binaries
	minioBin, _ := exec.LookPath("minio")
	rustfsBin, _ := exec.LookPath("rustfs")
	weedBin, _ := exec.LookPath("weed")

	// Build Go binaries if needed
	var liteioBin, herdBin string
	needGoBuild := false
	for _, s := range servers {
		if s.NeedBuild {
			needGoBuild = true
			break
		}
	}

	if needGoBuild {
		log("Building Go binaries...")
		var err error
		liteioBin, herdBin, err = buildBinaries(log)
		if err != nil {
			log("  Warning: Go build failed: %v", err)
		} else {
			cleanupFns = append(cleanupFns, func() {
				log("  Removing built binaries...")
				os.Remove(liteioBin)
				os.Remove(herdBin)
			})
		}
	}

	for _, srv := range servers {
		switch srv.Endpoint.Name {
		case "minio":
			if minioBin == "" {
				log("  %s: binary not found, skipping", srv.Endpoint.Name)
				continue
			}
			srv.Binary = minioBin
			dir := makeTempDir(srv.Endpoint.Name)
			srv.DataDir = dir
			srv.Args = []string{"server", dir, "--address", ":19000", "--console-address", ":19001"}
			cleanupFns = append(cleanupFns, rmDir(dir, srv.Endpoint.Name, log))

		case "rustfs":
			if rustfsBin == "" {
				log("  %s: binary not found, skipping", srv.Endpoint.Name)
				continue
			}
			srv.Binary = rustfsBin
			dir := makeTempDir(srv.Endpoint.Name)
			srv.DataDir = dir
			// rustfs takes volumes as positional args (no "server" subcommand)
			srv.Args = []string{dir, "--address", ":19100", "--console-address", ":19101"}
			cleanupFns = append(cleanupFns, rmDir(dir, srv.Endpoint.Name, log))

		case "seaweedfs":
			if weedBin == "" {
				log("  %s: binary not found, skipping", srv.Endpoint.Name)
				continue
			}
			srv.Binary = weedBin
			dir := makeTempDir(srv.Endpoint.Name)
			srv.DataDir = dir

			// Create s3 config file for auth
			s3Conf := filepath.Join(dir, "s3.json")
			os.WriteFile(s3Conf, []byte(`{"identities":[{"name":"admin","credentials":[{"accessKey":"admin","secretKey":"adminpassword"}],"actions":["*"]}]}`), 0o644)

			// weed server runs master + volume + filer + s3 all-in-one
			srv.Args = []string{
				"server",
				"-dir", dir,
				"-s3", "-s3.port", "18333",
				"-s3.config", s3Conf,
				"-master.port", "19333",
				"-volume.port", "18080",
				"-filer.port", "18888",
				"-ip", "127.0.0.1",
			}
			cleanupFns = append(cleanupFns, rmDir(dir, srv.Endpoint.Name, log))

		case "liteio_local":
			if liteioBin == "" {
				log("  %s: binary not available, skipping", srv.Endpoint.Name)
				continue
			}
			srv.Binary = liteioBin
			dir := makeTempDir(srv.Endpoint.Name)
			srv.DataDir = dir
			srv.Args = []string{
				"--port", "19200",
				"--data-dir", dir,
				"--access-key", srv.Endpoint.AccessKey,
				"--secret-key", srv.Endpoint.SecretKey,
				"--no-log",
			}
			srv.Env = []string{"LITEIO_NO_FSYNC=true"}
			cleanupFns = append(cleanupFns, rmDir(dir, srv.Endpoint.Name, log))

		case "liteio_herd":
			if herdBin == "" {
				log("  %s: binary not available, skipping", srv.Endpoint.Name)
				continue
			}
			srv.Binary = herdBin
			dir := makeTempDir(srv.Endpoint.Name)
			srv.DataDir = dir
			srv.Args = []string{
				"-listen", ":19230",
				"-data-dir", dir,
				"-access-key", srv.Endpoint.AccessKey,
				"-secret-key", srv.Endpoint.SecretKey,
				"-no-log",
				"-stripes", "16",
				"-sync", "none",
				"-inline-kb", "8",
				"-prealloc", "1024",
				"-bufsize", "8388608",
			}
			cleanupFns = append(cleanupFns, rmDir(dir, srv.Endpoint.Name, log))

		default:
			continue
		}

		log("  %s: ready (%s)", srv.Endpoint.Name, srv.Binary)
		available = append(available, srv)
	}

	cleanup := func() {
		for _, fn := range cleanupFns {
			fn()
		}
	}

	return available, cleanup
}

func makeTempDir(name string) string {
	dir, err := os.MkdirTemp("", fmt.Sprintf("bench_s3_%s_*", name))
	if err != nil {
		// Fallback to deterministic path
		dir = filepath.Join(os.TempDir(), fmt.Sprintf("bench_s3_%s", name))
		os.MkdirAll(dir, 0o755)
	}
	return dir
}

func rmDir(dir, name string, log func(string, ...any)) func() {
	return func() {
		if err := os.RemoveAll(dir); err != nil {
			log("  Warning: cleanup %s (%s) failed: %v", name, dir, err)
		} else {
			log("  Removed %s data: %s", name, dir)
		}
	}
}

func buildBinaries(log func(string, ...any)) (string, string, error) {
	tmpDir := os.TempDir()
	liteioBin := filepath.Join(tmpDir, "bench_s3_liteio")
	herdBin := filepath.Join(tmpDir, "bench_s3_herd")

	log("  Building liteio...")
	cmd := exec.Command("go", "build", "-o", liteioBin, "./cmd/liteio/")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("build liteio: %w", err)
	}

	log("  Building herd...")
	cmd = exec.Command("go", "build", "-o", herdBin, "./cmd/herd/")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("build herd: %w", err)
	}

	return liteioBin, herdBin, nil
}

// StartServer starts a local S3 server process and waits for it to become healthy.
func StartServer(ctx context.Context, srv LocalServer, log func(string, ...any)) (*ServerProcess, error) {
	log("  Starting %s: %s %s", srv.Endpoint.Name, filepath.Base(srv.Binary), strings.Join(srv.Args, " "))
	if srv.DataDir != "" {
		log("  Data dir: %s", srv.DataDir)
	}

	cmd := exec.CommandContext(ctx, srv.Binary, srv.Args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Set env
	if len(srv.Env) > 0 {
		cmd.Env = append(os.Environ(), srv.Env...)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", srv.Endpoint.Name, err)
	}

	proc := &ServerProcess{
		server: srv,
		cmds:   []*exec.Cmd{cmd},
	}

	// Wait for healthy
	timeout := 15 * time.Second
	if srv.Endpoint.Name == "seaweedfs" {
		timeout = 30 * time.Second // SeaweedFS needs more startup time
	}

	if err := waitHealthy(srv.Endpoint, timeout); err != nil {
		proc.Stop(log)
		return nil, fmt.Errorf("%s not healthy: %w", srv.Endpoint.Name, err)
	}

	log("  %s: healthy on %s (pid=%d)", srv.Endpoint.Name, srv.Endpoint.Host, cmd.Process.Pid)
	return proc, nil
}

// Stop kills the server process(es). Data dir cleanup is handled by ResolveBinaries cleanup.
func (p *ServerProcess) Stop(log func(string, ...any)) {
	if p.stopped {
		return
	}
	p.stopped = true

	for _, cmd := range p.cmds {
		if cmd.Process != nil {
			log("  Killing %s (pid=%d)...", p.server.Endpoint.Name, cmd.Process.Pid)
			cmd.Process.Kill()
			cmd.Wait()
		}
	}
}

// waitHealthy polls the S3 endpoint until it responds or timeout.
func waitHealthy(ep Endpoint, timeout time.Duration) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Try a simple GET — any response means the server is up
		resp, err := client.Get(fmt.Sprintf("http://%s/", ep.Host))
		if err == nil {
			resp.Body.Close()
			return nil
		}

		// Also try LiteIO-specific health endpoint
		resp, err = client.Get(fmt.Sprintf("http://%s/healthz/ready", ep.Host))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}

		time.Sleep(250 * time.Millisecond)
	}

	return fmt.Errorf("timeout after %v", timeout)
}
