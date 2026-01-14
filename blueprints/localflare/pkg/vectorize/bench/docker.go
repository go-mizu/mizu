package bench

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// DockerStats holds container resource usage metrics.
type DockerStats struct {
	ContainerName string  `json:"container_name"`
	ContainerID   string  `json:"container_id"`
	MemoryUsageMB float64 `json:"memory_usage_mb"`
	MemoryLimitMB float64 `json:"memory_limit_mb"`
	MemoryPercent float64 `json:"memory_percent"`
	CPUPercent    float64 `json:"cpu_percent"`
	DiskUsageMB   float64 `json:"disk_usage_mb"`
	Available     bool    `json:"available"`
	Error         string  `json:"error,omitempty"`
}

// DriverContainerMap maps driver names to Docker container name patterns.
// Container names in docker-compose are prefixed with the project name.
var DriverContainerMap = map[string]string{
	"qdrant":        "qdrant",
	"milvus":        "milvus",
	"weaviate":      "weaviate",
	"chroma":        "chroma",
	"pgvector":      "pgvector",
	"pgvectorscale": "pgvectorscale",
	"redis":         "redis",
	"opensearch":    "opensearch",
	"elasticsearch": "elasticsearch",
	"vald":          "vald",
	"vespa":         "vespa",
	// Embedded drivers (no Docker container)
	"mem":     "",
	"chromem": "",
	"sqlite":  "",
	"lancedb": "",
	"duckdb":  "",
}

// DockerStatsCollector collects Docker container statistics.
type DockerStatsCollector struct {
	projectPrefix string
}

// NewDockerStatsCollector creates a new Docker stats collector.
// projectPrefix is the docker-compose project name prefix (e.g., "all-" for docker/all/).
func NewDockerStatsCollector(projectPrefix string) *DockerStatsCollector {
	return &DockerStatsCollector{
		projectPrefix: projectPrefix,
	}
}

// GetContainerName returns the full container name for a driver.
func (c *DockerStatsCollector) GetContainerName(driverName string) string {
	baseName, ok := DriverContainerMap[driverName]
	if !ok || baseName == "" {
		return "" // Embedded driver, no container
	}
	if c.projectPrefix != "" {
		return c.projectPrefix + baseName + "-1"
	}
	return baseName
}

// CollectStats collects memory, CPU, and disk usage for a container.
func (c *DockerStatsCollector) CollectStats(ctx context.Context, driverName string) *DockerStats {
	stats := &DockerStats{
		ContainerName: c.GetContainerName(driverName),
	}

	if stats.ContainerName == "" {
		// Embedded driver
		stats.Available = false
		stats.Error = "embedded driver (no container)"
		return stats
	}

	// Get container stats using docker stats --no-stream
	if err := c.collectContainerStats(ctx, stats); err != nil {
		stats.Available = false
		stats.Error = err.Error()
		return stats
	}

	// Get disk usage from Docker volumes
	if err := c.collectDiskUsage(ctx, driverName, stats); err != nil {
		// Disk usage error is not fatal
		if stats.Error == "" {
			stats.Error = "disk: " + err.Error()
		}
	}

	stats.Available = true
	return stats
}

// collectContainerStats gets memory and CPU usage from docker stats.
func (c *DockerStatsCollector) collectContainerStats(ctx context.Context, stats *DockerStats) error {
	// Use docker stats with JSON format for parsing
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "stats", "--no-stream", "--format",
		`{"container":"{{.Name}}","id":"{{.ID}}","memory_usage":"{{.MemUsage}}","memory_percent":"{{.MemPerc}}","cpu_percent":"{{.CPUPerc}}"}`,
		stats.ContainerName)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("docker stats failed: %w", err)
	}

	// Parse JSON output
	var result struct {
		Container     string `json:"container"`
		ID            string `json:"id"`
		MemoryUsage   string `json:"memory_usage"`
		MemoryPercent string `json:"memory_percent"`
		CPUPercent    string `json:"cpu_percent"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("failed to parse docker stats: %w", err)
	}

	stats.ContainerID = result.ID

	// Parse memory usage (e.g., "123.4MiB / 7.656GiB")
	stats.MemoryUsageMB, stats.MemoryLimitMB = parseMemoryUsage(result.MemoryUsage)

	// Parse memory percent (e.g., "1.57%")
	stats.MemoryPercent = parsePercent(result.MemoryPercent)

	// Parse CPU percent (e.g., "0.15%")
	stats.CPUPercent = parsePercent(result.CPUPercent)

	return nil
}

// collectDiskUsage gets volume disk usage for a driver.
func (c *DockerStatsCollector) collectDiskUsage(ctx context.Context, driverName string, stats *DockerStats) error {
	// Get volume name pattern
	volumeName := c.getVolumeName(driverName)
	if volumeName == "" {
		return nil // No volume for this driver
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Use docker system df -v --format json to get volume sizes
	cmd := exec.CommandContext(ctx, "docker", "system", "df", "-v", "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		// Try alternative: inspect the volume directly
		return c.collectDiskUsageFromInspect(ctx, volumeName, stats)
	}

	// Parse the output line by line (docker system df outputs multiple JSON objects)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var dfOutput struct {
			Volumes []struct {
				Name  string `json:"Name"`
				Size  string `json:"Size"`
				Links int    `json:"Links"`
			} `json:"Volumes"`
		}

		if err := json.Unmarshal([]byte(line), &dfOutput); err != nil {
			continue
		}

		for _, vol := range dfOutput.Volumes {
			if strings.Contains(vol.Name, volumeName) {
				stats.DiskUsageMB = parseSizeToMB(vol.Size)
				return nil
			}
		}
	}

	// Fallback to inspect
	return c.collectDiskUsageFromInspect(ctx, volumeName, stats)
}

// collectDiskUsageFromInspect gets volume size using docker volume inspect.
func (c *DockerStatsCollector) collectDiskUsageFromInspect(ctx context.Context, volumeName string, stats *DockerStats) error {
	// First, list volumes to find the exact name
	cmd := exec.CommandContext(ctx, "docker", "volume", "ls", "--filter", fmt.Sprintf("name=%s", volumeName), "--format", "{{.Name}}")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	volumes := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(volumes) == 0 || volumes[0] == "" {
		return fmt.Errorf("volume %s not found", volumeName)
	}

	// Get the actual size using du on the volume mountpoint
	exactVolume := volumes[0]
	cmd = exec.CommandContext(ctx, "docker", "volume", "inspect", exactVolume, "--format", "{{.Mountpoint}}")
	output, err = cmd.Output()
	if err != nil {
		return err
	}

	mountpoint := strings.TrimSpace(string(output))
	if mountpoint == "" {
		return fmt.Errorf("could not get mountpoint for volume %s", exactVolume)
	}

	// Use du to get actual disk usage (requires root access to Docker volumes on some systems)
	// Fallback: estimate from docker system df
	cmd = exec.CommandContext(ctx, "docker", "system", "df", "-v")
	output, err = cmd.Output()
	if err != nil {
		return err
	}

	// Parse the text output to find the volume
	lines := strings.Split(string(output), "\n")
	inVolumes := false
	for _, line := range lines {
		if strings.HasPrefix(line, "VOLUME NAME") {
			inVolumes = true
			continue
		}
		if inVolumes && strings.Contains(line, volumeName) {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				stats.DiskUsageMB = parseSizeToMB(fields[2])
				return nil
			}
		}
	}

	return fmt.Errorf("could not determine disk usage for %s", volumeName)
}

// getVolumeName returns the volume name pattern for a driver.
func (c *DockerStatsCollector) getVolumeName(driverName string) string {
	volumeMap := map[string]string{
		"qdrant":        "qdrant_data",
		"milvus":        "milvus_data",
		"weaviate":      "weaviate_data",
		"chroma":        "chroma_data",
		"pgvector":      "pgvector_data",
		"pgvectorscale": "pgvectorscale_data",
		"redis":         "redis_data",
		"opensearch":    "opensearch_data",
		"elasticsearch": "elasticsearch_data",
		"vald":          "vald_data",
		"vespa":         "vespa_data",
	}
	return volumeMap[driverName]
}

// parseMemoryUsage parses memory strings like "123.4MiB / 7.656GiB".
func parseMemoryUsage(s string) (used, limit float64) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, 0
	}
	used = parseSizeToMB(strings.TrimSpace(parts[0]))
	limit = parseSizeToMB(strings.TrimSpace(parts[1]))
	return
}

// parseSizeToMB converts size strings to MB (e.g., "123.4MiB", "7.656GiB", "500kB").
func parseSizeToMB(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "N/A" || s == "0B" {
		return 0
	}

	// Handle different suffixes
	multipliers := map[string]float64{
		"B":   1.0 / (1024 * 1024),
		"KB":  1.0 / 1024,
		"KIB": 1.0 / 1024,
		"MB":  1.0,
		"MIB": 1.0,
		"GB":  1024,
		"GIB": 1024,
		"TB":  1024 * 1024,
		"TIB": 1024 * 1024,
	}

	s = strings.ToUpper(s)
	for suffix, mult := range multipliers {
		if strings.HasSuffix(s, suffix) {
			numStr := strings.TrimSuffix(s, suffix)
			numStr = strings.TrimSpace(numStr)
			if val, err := strconv.ParseFloat(numStr, 64); err == nil {
				return val * mult
			}
		}
	}

	return 0
}

// parsePercent parses percentage strings like "1.57%".
func parsePercent(s string) float64 {
	s = strings.TrimSuffix(strings.TrimSpace(s), "%")
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return val
	}
	return 0
}

// CollectAllStats collects Docker stats for all drivers.
func (c *DockerStatsCollector) CollectAllStats(ctx context.Context, drivers []string) map[string]*DockerStats {
	results := make(map[string]*DockerStats)
	for _, driver := range drivers {
		results[driver] = c.CollectStats(ctx, driver)
	}
	return results
}
