package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/config"
)

// runConfigGet implements "openbot config get <path>"
func runConfigGet() error {
	if len(os.Args) < 4 {
		return fmt.Errorf("usage: openbot config get <path>")
	}
	path := os.Args[3]

	raw, err := config.LoadRawConfig(config.DefaultConfigPath())
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	val, ok := config.ConfigGet(path, raw)
	if !ok {
		return fmt.Errorf("key not found: %s", path)
	}

	// Print as JSON if the value is an object or array, plain string otherwise.
	switch val.(type) {
	case map[string]any, []any:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(val)
	default:
		fmt.Println(val)
	}
	return nil
}

// runConfigSet implements "openbot config set <path> <value>"
func runConfigSet() error {
	if len(os.Args) < 5 {
		return fmt.Errorf("usage: openbot config set <path> <value>")
	}
	path := os.Args[3]
	rawValue := os.Args[4]

	configPath := config.DefaultConfigPath()
	raw, err := config.LoadRawConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Try to parse the value as JSON first (handles objects, arrays, bools, numbers).
	var parsed any
	if err := json.Unmarshal([]byte(rawValue), &parsed); err != nil {
		// Fall back to treating it as a plain string.
		parsed = rawValue
	}

	config.ConfigSet(path, parsed, raw)

	if err := config.SaveRawConfig(configPath, raw); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("Set %s\n", path)
	return nil
}

// runConfigUnset implements "openbot config unset <path>"
func runConfigUnset() error {
	if len(os.Args) < 4 {
		return fmt.Errorf("usage: openbot config unset <path>")
	}
	path := os.Args[3]

	configPath := config.DefaultConfigPath()
	raw, err := config.LoadRawConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	config.ConfigUnset(path, raw)

	if err := config.SaveRawConfig(configPath, raw); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("Unset %s\n", path)
	return nil
}
