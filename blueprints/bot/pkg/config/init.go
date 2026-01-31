package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// EnsureConfig ensures ~/.openbot exists with a valid config.
// If openbotDir doesn't exist, it clones from openclawDir.
// If openbotDir already has openbot.json, it does nothing.
func EnsureConfig(openbotDir, openclawDir string) error {
	cfgPath := filepath.Join(openbotDir, "openbot.json")

	// If config already exists, nothing to do.
	if _, err := os.Stat(cfgPath); err == nil {
		return nil
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

	// Create data directory.
	if err := os.MkdirAll(filepath.Join(openbotDir, "data"), 0o700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
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
