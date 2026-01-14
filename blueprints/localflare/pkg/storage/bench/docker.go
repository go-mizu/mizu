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

	// Try to get stats via docker stats command
	statsCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(statsCtx, "docker", "stats", "--no-stream", "--format",
		`{"memory_usage":"{{.MemUsage}}","memory_percent":"{{.MemPerc}}","cpu_percent":"{{.CPUPerc}}"}`,
		containerName)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker stats: %w", err)
	}

	var parsed struct {
		MemoryUsage   string `json:"memory_usage"`
		MemoryPercent string `json:"memory_percent"`
		CPUPercent    string `json:"cpu_percent"`
	}

	if err := json.Unmarshal(output, &parsed); err != nil {
		return nil, fmt.Errorf("parse stats: %w", err)
	}

	stats.MemoryUsage = parsed.MemoryUsage
	stats.MemoryPercent = parsePercent(parsed.MemoryPercent)
	stats.CPUPercent = parsePercent(parsed.CPUPercent)

	// Try to get disk usage
	stats.DiskUsage = c.getDiskUsage(ctx, containerName)

	return stats, nil
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
