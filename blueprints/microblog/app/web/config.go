// Package web provides the HTTP server and handlers.
package web

import (
	"os"
	"path/filepath"
)

// Config holds server configuration.
type Config struct {
	Addr    string // HTTP listen address
	DataDir string // Data directory
	Dev     bool   // Development mode
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	return Config{
		Addr:    ":8080",
		DataDir: filepath.Join(homeDir, "data", "blueprint", "microblog"),
		Dev:     false,
	}
}
