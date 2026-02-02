package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// InstallSpec represents a dependency install option parsed from metadata.
type InstallSpec struct {
	ID              string   `json:"id"`
	Kind            string   `json:"kind"`  // "brew", "node", "go", "uv", "download"
	Label           string   `json:"label"` // human-readable label
	Bins            []string `json:"bins"`  // binaries this installs
	OS              []string `json:"os"`    // platform filter
	Formula         string   `json:"formula"`         // brew formula
	Package         string   `json:"package"`         // node/uv package
	Module          string   `json:"module"`          // go module
	URL             string   `json:"url"`             // download URL
	Archive         string   `json:"archive"`         // archive type
	Extract         bool     `json:"extract"`         // extract after download
	StripComponents int      `json:"stripComponents"` // tar strip-components
	TargetDir       string   `json:"targetDir"`       // download target dir
}

// InstallResult is the outcome of an install attempt.
type InstallResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Stdout  string `json:"stdout"`
	Stderr  string `json:"stderr"`
	Code    *int   `json:"code"`
}

// ParseInstallSpecs extracts install specs from a skill's metadata JSON.
// It reads the InstallRaw field from the Skill (populated during metadata parsing)
// and parses it into []InstallSpec. If InstallRaw is nil/empty, returns nil.
func ParseInstallSpecs(s *Skill) []InstallSpec {
	if s == nil || len(s.InstallRaw) == 0 {
		return nil
	}
	return ParseInstallSpecsFromRaw(s.InstallRaw)
}

// ParseInstallSpecsFromRaw parses install specs from raw JSON.
func ParseInstallSpecsFromRaw(raw json.RawMessage) []InstallSpec {
	if len(raw) == 0 {
		return nil
	}

	var specs []InstallSpec
	if err := json.Unmarshal(raw, &specs); err != nil {
		return nil
	}
	if len(specs) == 0 {
		return nil
	}
	return specs
}

// ToInstallOpts converts install specs to the dashboard-facing format,
// filtering by the current platform.
func ToInstallOpts(specs []InstallSpec) []types.SkillInstallOpt {
	if len(specs) == 0 {
		return nil
	}

	currentOS := runtime.GOOS
	var opts []types.SkillInstallOpt

	for _, spec := range specs {
		// Filter by OS if specified.
		if len(spec.OS) > 0 {
			match := false
			for _, o := range spec.OS {
				if o == currentOS {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		opts = append(opts, types.SkillInstallOpt{
			ID:    spec.ID,
			Kind:  spec.Kind,
			Label: spec.Label,
			Bins:  nonNil(spec.Bins),
		})
	}

	return opts
}

// InstallSkillDep installs a dependency for a skill.
// It finds the skill by name from the provided skills slice, locates the
// install spec by ID, then runs the appropriate command.
// prefs controls install preferences (node manager, brew preference).
func InstallSkillDep(skills []*Skill, skillName, installID string, timeoutMs int, prefs InstallPrefs) (*InstallResult, error) {
	// Find the skill.
	var target *Skill
	for _, s := range skills {
		if s.Name == skillName {
			target = s
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("skill %q not found", skillName)
	}

	// Parse install specs for this skill.
	specs := ParseInstallSpecs(target)
	if len(specs) == 0 {
		return nil, fmt.Errorf("skill %q has no install specs", skillName)
	}

	// Find the spec by ID.
	var spec *InstallSpec
	for i := range specs {
		if specs[i].ID == installID {
			spec = &specs[i]
			break
		}
	}
	if spec == nil {
		return nil, fmt.Errorf("install spec %q not found for skill %q", installID, skillName)
	}

	// Set up timeout.
	if timeoutMs <= 0 {
		if spec.Kind == "download" {
			timeoutMs = 300000 // 5 minute default for downloads
		} else {
			timeoutMs = 120000 // 2 minute default
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	// Handle download kind separately.
	if spec.Kind == "download" {
		skillKey := target.SkillKey
		if skillKey == "" {
			skillKey = target.Name
		}
		return installDownload(ctx, spec, skillKey)
	}

	// Build the command based on kind.
	args, err := buildInstallCommand(spec, prefs)
	if err != nil {
		return nil, err
	}

	// Run the command.
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	stdout, err := cmd.Output()

	result := &InstallResult{}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			result.Code = &code
			result.Stderr = string(exitErr.Stderr)
			result.OK = false
			result.Message = fmt.Sprintf("command exited with code %d", code)
		} else if ctx.Err() == context.DeadlineExceeded {
			result.OK = false
			result.Message = "command timed out"
		} else {
			result.OK = false
			result.Message = err.Error()
		}
		result.Stdout = string(stdout)
		return result, nil
	}

	result.OK = true
	result.Message = "installed successfully"
	result.Stdout = string(stdout)
	code := 0
	result.Code = &code
	return result, nil
}

// InstallPrefs holds install preference configuration from skills.install config.
type InstallPrefs struct {
	PreferBrew  bool   // prefer brew over other package managers
	NodeManager string // "npm", "pnpm", "yarn", "bun"
}

// DefaultInstallPrefs returns default install preferences (matches OpenClaw defaults).
func DefaultInstallPrefs() InstallPrefs {
	return InstallPrefs{
		PreferBrew:  true,
		NodeManager: "npm",
	}
}

// ParseInstallPrefs extracts install preferences from config map.
// Reads from cfg["skills"]["install"].
func ParseInstallPrefs(cfg map[string]any) InstallPrefs {
	prefs := DefaultInstallPrefs()
	skills, ok := cfg["skills"].(map[string]any)
	if !ok {
		return prefs
	}
	install, ok := skills["install"].(map[string]any)
	if !ok {
		return prefs
	}

	if v, ok := install["preferBrew"].(bool); ok {
		prefs.PreferBrew = v
	}
	if v, ok := install["nodeManager"].(string); ok {
		nm := strings.ToLower(strings.TrimSpace(v))
		switch nm {
		case "npm", "pnpm", "yarn", "bun":
			prefs.NodeManager = nm
		}
	}
	return prefs
}

// buildInstallCommand returns the command args for installing a dependency.
// The download kind is handled separately in installDownload.
func buildInstallCommand(spec *InstallSpec, prefs InstallPrefs) ([]string, error) {
	switch spec.Kind {
	case "brew":
		if spec.Formula == "" {
			return nil, fmt.Errorf("brew install spec missing formula")
		}
		return []string{"brew", "install", spec.Formula}, nil
	case "node":
		if spec.Package == "" {
			return nil, fmt.Errorf("node install spec missing package")
		}
		return buildNodeCommand(spec.Package, prefs.NodeManager), nil
	case "go":
		if spec.Module == "" {
			return nil, fmt.Errorf("go install spec missing module")
		}
		return []string{"go", "install", spec.Module}, nil
	case "uv":
		if spec.Package == "" {
			return nil, fmt.Errorf("uv install spec missing package")
		}
		return []string{"uv", "tool", "install", spec.Package}, nil
	case "download":
		// Download is handled specially, not via a single command.
		return nil, fmt.Errorf("download kind uses installDownload")
	default:
		return nil, fmt.Errorf("unknown install kind %q", spec.Kind)
	}
}

// buildNodeCommand returns the install args for the configured node package manager.
func buildNodeCommand(pkg, nodeManager string) []string {
	switch nodeManager {
	case "pnpm":
		return []string{"pnpm", "add", "-g", pkg}
	case "yarn":
		return []string{"yarn", "global", "add", pkg}
	case "bun":
		return []string{"bun", "add", "-g", pkg}
	default:
		return []string{"npm", "install", "-g", pkg}
	}
}

// installDownload handles the download install kind.
// It fetches a URL, optionally detects/extracts archives, and places files
// in the target directory. Matches OpenClaw's download install behavior.
func installDownload(ctx context.Context, spec *InstallSpec, skillKey string) (*InstallResult, error) {
	if spec.URL == "" {
		return nil, fmt.Errorf("download install spec missing URL")
	}

	// Determine target directory.
	targetDir := spec.TargetDir
	if targetDir == "" {
		home, _ := os.UserHomeDir()
		targetDir = filepath.Join(home, ".openbot", "tools", skillKey)
	}
	targetDir = expandHome(targetDir)

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return &InstallResult{
			OK:      false,
			Message: fmt.Sprintf("create target dir: %v", err),
		}, nil
	}

	// Determine filename from URL.
	filename := filenameFromURL(spec.URL)

	// Download the file.
	downloadPath := filepath.Join(targetDir, filename)
	bytesWritten, err := downloadFile(ctx, spec.URL, downloadPath)
	if err != nil {
		return &InstallResult{
			OK:      false,
			Message: fmt.Sprintf("download failed: %v", err),
		}, nil
	}

	stdout := fmt.Sprintf("downloaded=%d bytes", bytesWritten)

	// Determine if we should extract.
	archiveType := spec.Archive
	if archiveType == "" {
		archiveType = detectArchiveType(filename)
	}

	shouldExtract := spec.Extract
	if !shouldExtract && archiveType == "" {
		// No archive, no extraction.
		return &InstallResult{
			OK:      true,
			Message: fmt.Sprintf("Downloaded to %s", downloadPath),
			Stdout:  stdout,
		}, nil
	}

	if archiveType == "" {
		return &InstallResult{
			OK:      true,
			Message: fmt.Sprintf("Downloaded to %s", downloadPath),
			Stdout:  stdout,
		}, nil
	}

	// Extract the archive.
	extractResult, err := extractArchive(ctx, downloadPath, targetDir, archiveType, spec.StripComponents)
	if err != nil {
		return &InstallResult{
			OK:      false,
			Message: fmt.Sprintf("extraction failed: %v", err),
			Stdout:  stdout,
		}, nil
	}

	// Remove the archive after successful extraction.
	os.Remove(downloadPath)

	code := 0
	return &InstallResult{
		OK:      true,
		Message: fmt.Sprintf("Downloaded and extracted to %s", targetDir),
		Stdout:  stdout + "\n" + extractResult,
		Code:    &code,
	}, nil
}

// downloadFile fetches a URL and writes it to disk.
func downloadFile(ctx context.Context, rawURL, dest string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	f, err := os.Create(dest)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	n, err := io.Copy(f, resp.Body)
	return n, err
}

// filenameFromURL extracts the filename from a URL path.
func filenameFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		parts := strings.Split(rawURL, "/")
		if last := parts[len(parts)-1]; last != "" {
			return last
		}
		return "download"
	}
	base := filepath.Base(u.Path)
	if base == "" || base == "." || base == "/" {
		return "download"
	}
	return base
}

// detectArchiveType infers archive type from filename extension.
func detectArchiveType(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
		return "tar.gz"
	case strings.HasSuffix(lower, ".tar.bz2") || strings.HasSuffix(lower, ".tbz2"):
		return "tar.bz2"
	case strings.HasSuffix(lower, ".zip"):
		return "zip"
	default:
		return ""
	}
}

// extractArchive extracts an archive to the target directory.
func extractArchive(ctx context.Context, archivePath, targetDir, archiveType string, stripComponents int) (string, error) {
	var args []string

	switch archiveType {
	case "tar.gz", "tar.bz2":
		args = []string{"tar", "xf", archivePath, "-C", targetDir}
		if stripComponents > 0 {
			args = append(args, fmt.Sprintf("--strip-components=%d", stripComponents))
		}
	case "zip":
		args = []string{"unzip", "-q", archivePath, "-d", targetDir}
	default:
		return "", fmt.Errorf("unsupported archive type: %s", archiveType)
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s: %w", args[0], err)
	}
	return string(out), nil
}

// expandHome expands ~ to the user's home directory.
func expandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}
