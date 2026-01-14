package bench

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DockerStats holds container resource usage.
type DockerStats struct {
	ContainerName string  `json:"container_name"`
	MemoryUsage   string  `json:"memory_usage"`
	MemoryPercent float64 `json:"memory_percent"`
	CPUPercent    float64 `json:"cpu_percent"`
	DiskUsage     float64 `json:"disk_usage_mb,omitempty"`

	// Enhanced metrics
	MemoryUsageMB float64 `json:"memory_usage_mb,omitempty"`  // Parsed memory in MB
	MemoryLimitMB float64 `json:"memory_limit_mb,omitempty"`  // Container memory limit
	MemoryCacheMB float64 `json:"memory_cache_mb,omitempty"`  // Page cache (for disk drivers)
	MemoryRSSMB   float64 `json:"memory_rss_mb,omitempty"`    // Resident Set Size (actual app memory)
	BlockRead     string  `json:"block_read,omitempty"`       // Block I/O read
	BlockWrite    string  `json:"block_write,omitempty"`      // Block I/O write
	NetIO         string  `json:"net_io,omitempty"`           // Network I/O
	PIDs          int     `json:"pids,omitempty"`             // Number of processes
	VolumeSize    float64 `json:"volume_size_mb,omitempty"`   // Docker volume size in MB
	VolumeName    string  `json:"volume_name,omitempty"`      // Docker volume name
	ImageSize     float64 `json:"image_size_mb,omitempty"`    // Container image size
	ContainerSize float64 `json:"container_size_mb,omitempty"` // Container writable layer size
}

// DockerStatsCollector collects Docker container statistics.
type DockerStatsCollector struct {
	projectPrefix string
}

// NewDockerStatsCollector creates a new Docker stats collector.
func NewDockerStatsCollector(projectPrefix string) *DockerStatsCollector {
	if projectPrefix == "" {
		projectPrefix = "all-"
	}
	return &DockerStatsCollector{
		projectPrefix: projectPrefix,
	}
}

// GetStats retrieves stats for a container.
func (c *DockerStatsCollector) GetStats(ctx context.Context, containerName string) (*DockerStats, error) {
	stats := &DockerStats{ContainerName: containerName}

	// Try to get stats via docker stats command with more fields
	statsCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(statsCtx, "docker", "stats", "--no-stream", "--format",
		`{"memory_usage":"{{.MemUsage}}","memory_percent":"{{.MemPerc}}","cpu_percent":"{{.CPUPerc}}","block_io":"{{.BlockIO}}","net_io":"{{.NetIO}}","pids":"{{.PIDs}}"}`,
		containerName)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker stats: %w", err)
	}

	var parsed struct {
		MemoryUsage   string `json:"memory_usage"`
		MemoryPercent string `json:"memory_percent"`
		CPUPercent    string `json:"cpu_percent"`
		BlockIO       string `json:"block_io"`
		NetIO         string `json:"net_io"`
		PIDs          string `json:"pids"`
	}

	if err := json.Unmarshal(output, &parsed); err != nil {
		return nil, fmt.Errorf("parse stats: %w", err)
	}

	stats.MemoryUsage = parsed.MemoryUsage
	stats.MemoryPercent = parsePercent(parsed.MemoryPercent)
	stats.CPUPercent = parsePercent(parsed.CPUPercent)
	stats.NetIO = parsed.NetIO
	stats.PIDs, _ = strconv.Atoi(parsed.PIDs)

	// Parse memory usage and limit from "123MiB / 7.5GiB" format
	if parts := strings.Split(parsed.MemoryUsage, " / "); len(parts) == 2 {
		stats.MemoryUsageMB = parseSize(parts[0])
		stats.MemoryLimitMB = parseSize(parts[1])
	}

	// Parse block I/O from "1.5MB / 2.3MB" format
	if parts := strings.Split(parsed.BlockIO, " / "); len(parts) == 2 {
		stats.BlockRead = strings.TrimSpace(parts[0])
		stats.BlockWrite = strings.TrimSpace(parts[1])
	}

	// Get detailed memory breakdown (cache vs RSS)
	c.getMemoryBreakdown(ctx, containerName, stats)

	// Get volume information
	c.getVolumeInfo(ctx, containerName, stats)

	// Get container size info
	c.getContainerSize(ctx, containerName, stats)

	// Legacy disk usage (volume size)
	stats.DiskUsage = stats.VolumeSize

	return stats, nil
}

// getMemoryBreakdown retrieves cache and RSS memory from cgroup stats.
func (c *DockerStatsCollector) getMemoryBreakdown(ctx context.Context, containerName string, stats *DockerStats) {
	memCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Try to get memory stats from container inspect
	cmd := exec.CommandContext(memCtx, "docker", "exec", containerName,
		"cat", "/sys/fs/cgroup/memory.stat")

	output, err := cmd.Output()
	if err != nil {
		// Try cgroup v1 path
		cmd = exec.CommandContext(memCtx, "docker", "exec", containerName,
			"cat", "/sys/fs/cgroup/memory/memory.stat")
		output, err = cmd.Output()
		if err != nil {
			return
		}
	}

	// Parse memory.stat for cache and rss
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		val, _ := strconv.ParseFloat(parts[1], 64)
		valMB := val / (1024 * 1024)

		switch parts[0] {
		case "cache", "file":
			stats.MemoryCacheMB = valMB
		case "rss", "anon":
			stats.MemoryRSSMB = valMB
		}
	}
}

// getVolumeInfo retrieves volume name and size.
func (c *DockerStatsCollector) getVolumeInfo(ctx context.Context, containerName string, stats *DockerStats) {
	volCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get volume name from container inspect
	cmd := exec.CommandContext(volCtx, "docker", "inspect", "--format",
		`{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{end}}{{end}}`,
		containerName)

	output, err := cmd.Output()
	if err != nil {
		return
	}

	volumeName := strings.TrimSpace(string(output))
	if volumeName == "" {
		return
	}
	stats.VolumeName = volumeName

	// Get volume size using docker system df
	dfCtx, dfCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dfCancel()

	dfCmd := exec.CommandContext(dfCtx, "docker", "system", "df", "-v", "--format", "json")
	dfOutput, err := dfCmd.Output()
	if err != nil {
		// Fallback to table format
		stats.VolumeSize = c.getDiskUsage(ctx, containerName)
		return
	}

	// Docker system df -v --format json returns JSONL (one object per category)
	for _, line := range strings.Split(string(dfOutput), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var item struct {
			Volumes []struct {
				Name string `json:"Name"`
				Size string `json:"Size"`
			} `json:"Volumes"`
		}
		if err := json.Unmarshal([]byte(line), &item); err == nil && len(item.Volumes) > 0 {
			for _, vol := range item.Volumes {
				if strings.Contains(vol.Name, volumeName) || vol.Name == volumeName {
					stats.VolumeSize = parseSize(vol.Size)
					return
				}
			}
		}
	}

	// Fallback
	stats.VolumeSize = c.getDiskUsage(ctx, containerName)
}

// getContainerSize retrieves container and image size.
func (c *DockerStatsCollector) getContainerSize(ctx context.Context, containerName string, stats *DockerStats) {
	sizeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get container size (writable layer) and image size
	cmd := exec.CommandContext(sizeCtx, "docker", "inspect", "--format",
		`{{.SizeRw}} {{.SizeRootFs}}`,
		containerName)

	output, err := cmd.Output()
	if err != nil {
		return
	}

	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) >= 2 {
		if sizeRw, err := strconv.ParseFloat(parts[0], 64); err == nil && sizeRw > 0 {
			stats.ContainerSize = sizeRw / (1024 * 1024)
		}
		if sizeRoot, err := strconv.ParseFloat(parts[1], 64); err == nil && sizeRoot > 0 {
			stats.ImageSize = sizeRoot / (1024 * 1024)
		}
	}
}

// GetAllStats retrieves stats for all configured drivers.
func (c *DockerStatsCollector) GetAllStats(ctx context.Context, drivers []DriverConfig) map[string]*DockerStats {
	results := make(map[string]*DockerStats)

	for _, d := range drivers {
		if d.Container == "" {
			continue
		}
		stats, err := c.GetStats(ctx, d.Container)
		if err == nil {
			results[d.Name] = stats
		}
	}

	return results
}

func (c *DockerStatsCollector) getDiskUsage(ctx context.Context, containerName string) float64 {
	// Try to get volume size
	volCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get the volume name from container inspect
	cmd := exec.CommandContext(volCtx, "docker", "inspect", "--format",
		`{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{end}}{{end}}`,
		containerName)

	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	volumeName := strings.TrimSpace(string(output))
	if volumeName == "" {
		return 0
	}

	// Get volume size using docker system df
	dfCtx, dfCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dfCancel()

	dfCmd := exec.CommandContext(dfCtx, "docker", "system", "df", "-v", "--format",
		`{{range .Volumes}}{{.Name}}\t{{.Size}}{{"\n"}}{{end}}`)

	dfOutput, err := dfCmd.Output()
	if err != nil {
		return 0
	}

	// Parse output to find our volume
	for _, line := range strings.Split(string(dfOutput), "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 && strings.Contains(parts[0], volumeName) {
			return parseSize(parts[1])
		}
	}

	return 0
}

// parsePercent parses percentage strings like "2.5%" or "12.34%".
func parsePercent(s string) float64 {
	s = strings.TrimSuffix(strings.TrimSpace(s), "%")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// parseSize parses size strings like "123.4MiB", "7.656GiB", "500kB".
func parseSize(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "0B" || s == "N/A" {
		return 0
	}

	// Extract number and unit
	re := regexp.MustCompile(`^([\d.]+)\s*([A-Za-z]+)$`)
	match := re.FindStringSubmatch(s)
	if match == nil {
		return 0
	}

	num, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0
	}

	unit := strings.ToUpper(match[2])
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

	if mult, ok := multipliers[unit]; ok {
		return num * mult
	}
	return 0
}

// IsDockerAvailable checks if Docker is available.
func IsDockerAvailable(ctx context.Context) bool {
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(checkCtx, "docker", "info")
	return cmd.Run() == nil
}
