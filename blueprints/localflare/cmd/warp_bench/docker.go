package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DockerCleanup handles Docker container and volume cleanup for benchmarks.
type DockerCleanup struct {
	composeDir string
	logger     func(format string, args ...any)
}

// NewDockerCleanup creates a new Docker cleanup handler.
func NewDockerCleanup(composeDir string) *DockerCleanup {
	return &DockerCleanup{
		composeDir: composeDir,
		logger: func(format string, args ...any) {
			fmt.Printf(format+"\n", args...)
		},
	}
}

// SetLogger sets the logger function.
func (d *DockerCleanup) SetLogger(logger func(format string, args ...any)) {
	d.logger = logger
}

// PreBenchmarkCleanup performs fast cleanup before a driver benchmark.
// This restarts the container to clear any in-memory caches and resets state.
func (d *DockerCleanup) PreBenchmarkCleanup(ctx context.Context, driver *DriverConfig) error {
	if driver.Container == "" {
		return nil
	}

	d.logger("  [PRE] Restarting %s container...", driver.Name)

	// Restart the container (fast - just restart, don't clear volumes)
	cmd := exec.CommandContext(ctx, "docker", "restart", driver.Container)
	if err := cmd.Run(); err != nil {
		// Try with docker compose if direct restart fails
		return d.restartWithCompose(ctx, driver)
	}

	// Wait for container to be healthy
	return d.waitForHealthy(ctx, driver, 60*time.Second)
}

// PostBenchmarkCleanup performs thorough cleanup after a driver benchmark.
// This clears the bucket data to prepare for the next driver.
func (d *DockerCleanup) PostBenchmarkCleanup(ctx context.Context, driver *DriverConfig) error {
	if driver.Container == "" {
		return nil
	}

	d.logger("  [POST] Cleaning up %s data...", driver.Name)

	// Use warp's bucket data clearing or aws s3 rm
	// First try to clear the bucket using aws cli
	if err := d.clearBucket(ctx, driver); err != nil {
		d.logger("  [POST] Bucket clear failed: %v", err)
	}

	return nil
}

// FullCleanup performs complete cleanup including volume reset.
// Use this before running a full benchmark suite.
func (d *DockerCleanup) FullCleanup(ctx context.Context, driver *DriverConfig) error {
	if driver.Container == "" {
		return nil
	}

	d.logger("  [FULL] Full cleanup for %s...", driver.Name)

	// Stop container
	stopCmd := exec.CommandContext(ctx, "docker", "stop", driver.Container)
	stopCmd.Run() // Ignore errors

	// Get volume name based on driver
	volumeName := d.getVolumeName(driver)
	if volumeName != "" {
		// Clear volume data using docker run
		clearCmd := exec.CommandContext(ctx, "docker", "run", "--rm",
			"-v", volumeName+":/data",
			"alpine", "sh", "-c", "rm -rf /data/* /data/.[!.]* 2>/dev/null || true")
		clearCmd.Run() // Ignore errors
	}

	// Start container
	startCmd := exec.CommandContext(ctx, "docker", "start", driver.Container)
	if err := startCmd.Run(); err != nil {
		return fmt.Errorf("failed to start %s: %w", driver.Container, err)
	}

	// Wait for healthy
	return d.waitForHealthy(ctx, driver, 60*time.Second)
}

// restartWithCompose uses docker compose to restart a service.
func (d *DockerCleanup) restartWithCompose(ctx context.Context, driver *DriverConfig) error {
	serviceName := d.getServiceName(driver)
	if serviceName == "" {
		return fmt.Errorf("unknown service for driver %s", driver.Name)
	}

	cmd := exec.CommandContext(ctx, "docker", "compose", "restart", serviceName)
	cmd.Dir = d.composeDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compose restart failed: %w", err)
	}

	return d.waitForHealthy(ctx, driver, 60*time.Second)
}

// waitForHealthy waits for a container to become healthy.
func (d *DockerCleanup) waitForHealthy(ctx context.Context, driver *DriverConfig, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		cmd := exec.CommandContext(ctx, "docker", "inspect",
			"--format={{.State.Health.Status}}", driver.Container)
		output, err := cmd.Output()
		if err == nil {
			status := strings.TrimSpace(string(output))
			if status == "healthy" {
				d.logger("  [OK] %s is healthy", driver.Name)
				return nil
			}
		}

		// Also check if container is running (for containers without health checks)
		runningCmd := exec.CommandContext(ctx, "docker", "inspect",
			"--format={{.State.Running}}", driver.Container)
		runningOutput, _ := runningCmd.Output()
		if strings.TrimSpace(string(runningOutput)) == "true" {
			// Give it a bit more time even if no health check
			time.Sleep(2 * time.Second)
			d.logger("  [OK] %s is running", driver.Name)
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}

	return fmt.Errorf("timeout waiting for %s to be healthy", driver.Container)
}

// clearBucket clears all objects from the test bucket.
func (d *DockerCleanup) clearBucket(ctx context.Context, driver *DriverConfig) error {
	// Use aws cli to clear the bucket
	cmd := exec.CommandContext(ctx, "aws", "s3", "rm",
		"s3://"+driver.Bucket, "--recursive",
		"--endpoint-url", "http://"+driver.Endpoint)
	cmd.Env = append(cmd.Environ(),
		"AWS_ACCESS_KEY_ID="+driver.AccessKey,
		"AWS_SECRET_ACCESS_KEY="+driver.SecretKey,
		"AWS_DEFAULT_REGION=us-east-1",
	)

	// Run with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		return fmt.Errorf("bucket clear timed out")
	case <-ctx.Done():
		cmd.Process.Kill()
		return ctx.Err()
	}
}

// getVolumeName returns the Docker volume name for a driver.
func (d *DockerCleanup) getVolumeName(driver *DriverConfig) string {
	switch driver.Name {
	case "minio":
		return "all_minio_data"
	case "rustfs":
		return "all_rustfs_data"
	case "seaweedfs":
		return "all_seaweedfs_master" // SeaweedFS has multiple volumes
	case "localstack":
		return "all_localstack_data"
	case "liteio":
		return "all_liteio_data"
	case "liteio_mem":
		return "" // Memory-based, no volume
	default:
		return ""
	}
}

// getServiceName returns the docker compose service name for a driver.
func (d *DockerCleanup) getServiceName(driver *DriverConfig) string {
	switch driver.Name {
	case "minio":
		return "minio"
	case "rustfs":
		return "rustfs"
	case "seaweedfs":
		return "seaweedfs-s3"
	case "localstack":
		return "localstack"
	case "liteio":
		return "liteio"
	case "liteio_mem":
		return "liteio_mem"
	default:
		return ""
	}
}

// RecreateContainer fully recreates a container with fresh volumes.
func (d *DockerCleanup) RecreateContainer(ctx context.Context, driver *DriverConfig) error {
	serviceName := d.getServiceName(driver)
	if serviceName == "" {
		return fmt.Errorf("unknown service for driver %s", driver.Name)
	}

	d.logger("  [RECREATE] Recreating %s with fresh volumes...", driver.Name)

	// Stop and remove container
	stopCmd := exec.CommandContext(ctx, "docker", "compose", "stop", serviceName)
	stopCmd.Dir = d.composeDir
	stopCmd.Run()

	rmCmd := exec.CommandContext(ctx, "docker", "compose", "rm", "-f", serviceName)
	rmCmd.Dir = d.composeDir
	rmCmd.Run()

	// Remove and recreate volume
	volumeName := d.getVolumeName(driver)
	if volumeName != "" {
		exec.CommandContext(ctx, "docker", "volume", "rm", "-f", volumeName).Run()
	}

	// Start fresh
	upCmd := exec.CommandContext(ctx, "docker", "compose", "up", "-d", serviceName)
	upCmd.Dir = d.composeDir
	if err := upCmd.Run(); err != nil {
		return fmt.Errorf("compose up failed: %w", err)
	}

	// Also start init container if exists
	initService := serviceName + "-init"
	initCmd := exec.CommandContext(ctx, "docker", "compose", "up", "-d", initService)
	initCmd.Dir = d.composeDir
	initCmd.Run() // Ignore errors - init might not exist

	return d.waitForHealthy(ctx, driver, 90*time.Second)
}
