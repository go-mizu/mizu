package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EnsureConfig ensures ~/.openbot exists with a valid config.
// If openbotDir doesn't exist, it clones from openclawDir.
// If openbotDir already has openbot.json, it does nothing.
// Creates the full OpenClaw-compatible directory structure.
func EnsureConfig(openbotDir, openclawDir string) error {
	cfgPath := filepath.Join(openbotDir, "openbot.json")

	// If config already exists, ensure directory structure and return.
	if _, err := os.Stat(cfgPath); err == nil {
		return ensureDirectoryStructure(openbotDir)
	}

	// Check that source (openclaw) exists.
	srcCfgPath := filepath.Join(openclawDir, "openclaw.json")
	if _, err := os.Stat(srcCfgPath); err != nil {
		return fmt.Errorf("no existing config and no openclaw source at %s: %w", openclawDir, err)
	}

	// Create openbot directory.
	if err := os.MkdirAll(openbotDir, 0o700); err != nil {
		return fmt.Errorf("create openbot dir: %w", err)
	}

	// Copy and rewrite config.
	if err := cloneConfig(srcCfgPath, cfgPath, openbotDir); err != nil {
		return fmt.Errorf("clone config: %w", err)
	}

	// Copy workspace files.
	srcWs := filepath.Join(openclawDir, "workspace")
	dstWs := filepath.Join(openbotDir, "workspace")
	if err := copyDir(srcWs, dstWs); err != nil {
		return fmt.Errorf("clone workspace: %w", err)
	}

	// Create full directory structure.
	if err := ensureDirectoryStructure(openbotDir); err != nil {
		return fmt.Errorf("create directory structure: %w", err)
	}

	return nil
}

// ensureDirectoryStructure creates all OpenClaw-compatible subdirectories.
func ensureDirectoryStructure(baseDir string) error {
	dirs := []string{
		filepath.Join(baseDir, "agents", "main", "agent"),
		filepath.Join(baseDir, "agents", "main", "sessions"),
		filepath.Join(baseDir, "memory"),
		filepath.Join(baseDir, "identity"),
		filepath.Join(baseDir, "logs"),
		filepath.Join(baseDir, "telegram"),
		filepath.Join(baseDir, "cron"),
		filepath.Join(baseDir, "devices"),
		filepath.Join(baseDir, "credentials"),
		filepath.Join(baseDir, "canvas"),
		filepath.Join(baseDir, "workspace", "memory"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("create dir %s: %w", d, err)
		}
	}

	// Create default agent files if missing.
	modelsPath := filepath.Join(baseDir, "agents", "main", "agent", "models.json")
	if _, err := os.Stat(modelsPath); os.IsNotExist(err) {
		os.WriteFile(modelsPath, []byte("{}\n"), 0o600)
	}
	authPath := filepath.Join(baseDir, "agents", "main", "agent", "auth-profiles.json")
	if _, err := os.Stat(authPath); os.IsNotExist(err) {
		os.WriteFile(authPath, []byte("{}\n"), 0o600)
	}

	// Create default device/identity files if missing.
	devicePath := filepath.Join(baseDir, "identity", "device.json")
	if _, err := os.Stat(devicePath); os.IsNotExist(err) {
		os.WriteFile(devicePath, []byte("{}\n"), 0o600)
	}
	deviceAuthPath := filepath.Join(baseDir, "identity", "device-auth.json")
	if _, err := os.Stat(deviceAuthPath); os.IsNotExist(err) {
		os.WriteFile(deviceAuthPath, []byte("{}\n"), 0o600)
	}

	// Create default cron jobs file if missing.
	cronPath := filepath.Join(baseDir, "cron", "jobs.json")
	if _, err := os.Stat(cronPath); os.IsNotExist(err) {
		os.WriteFile(cronPath, []byte("{\"jobs\":[]}\n"), 0o600)
	}

	// Create default devices files if missing.
	pairedPath := filepath.Join(baseDir, "devices", "paired.json")
	if _, err := os.Stat(pairedPath); os.IsNotExist(err) {
		os.WriteFile(pairedPath, []byte("{}\n"), 0o600)
	}
	pendingPath := filepath.Join(baseDir, "devices", "pending.json")
	if _, err := os.Stat(pendingPath); os.IsNotExist(err) {
		os.WriteFile(pendingPath, []byte("{}\n"), 0o600)
	}

	// Create update-check.json if missing.
	updateCheckPath := filepath.Join(baseDir, "update-check.json")
	if _, err := os.Stat(updateCheckPath); os.IsNotExist(err) {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		content := fmt.Sprintf("{\"lastCheckedAt\":\"%s\"}\n", now)
		os.WriteFile(updateCheckPath, []byte(content), 0o600)
	}

	// Create canvas/index.html if missing.
	canvasPath := filepath.Join(baseDir, "canvas", "index.html")
	if _, err := os.Stat(canvasPath); os.IsNotExist(err) {
		html := "<!DOCTYPE html>\n<html><head><title>OpenBot Canvas</title></head>\n<body><h1>OpenBot Canvas</h1><p>Canvas UI placeholder.</p></body>\n</html>\n"
		os.WriteFile(canvasPath, []byte(html), 0o644)
	}

	// Create workspace/MEMORY.md if missing.
	memoryMdPath := filepath.Join(baseDir, "workspace", "MEMORY.md")
	if _, err := os.Stat(memoryMdPath); os.IsNotExist(err) {
		os.WriteFile(memoryMdPath, []byte("# Memory\n\nLong-term curated memories.\n"), 0o644)
	}

	return nil
}

// cloneConfig copies the openclaw config and rewrites workspace paths.
func cloneConfig(src, dst, openbotDir string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Parse, rewrite workspace path, and re-serialize.
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Rewrite agents.defaults.workspace to point to openbot workspace.
	if agents, ok := raw["agents"].(map[string]any); ok {
		if defaults, ok := agents["defaults"].(map[string]any); ok {
			if ws, ok := defaults["workspace"].(string); ok && ws != "" {
				if strings.Contains(ws, "openclaw") {
					defaults["workspace"] = filepath.Join(openbotDir, "workspace")
				}
			}
		}
	}

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(dst, out, 0o600)
}

// copyDir copies a directory tree, skipping .git.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // source doesn't exist, skip
		}
		return err
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("%s is not a directory", src)
	}

	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}

		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
