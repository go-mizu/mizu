package fineweb

import (
	"os"
	"path/filepath"
)

// DriverConfig is the common configuration for all drivers.
type DriverConfig struct {
	// DataDir is the base directory for index storage.
	DataDir string

	// Language is the language code (e.g., "vie_Latn").
	Language string

	// Options contains driver-specific configuration.
	Options map[string]any
}

// DefaultDriverConfig returns default driver configuration.
func DefaultDriverConfig() DriverConfig {
	home, _ := os.UserHomeDir()
	return DriverConfig{
		DataDir:  filepath.Join(home, "data", "blueprints", "search", "fineweb-2"),
		Language: "",
		Options:  make(map[string]any),
	}
}

// GetString returns a string option or the default value.
func (c DriverConfig) GetString(key, defaultVal string) string {
	if c.Options == nil {
		return defaultVal
	}
	if v, ok := c.Options[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

// GetInt returns an int option or the default value.
func (c DriverConfig) GetInt(key string, defaultVal int) int {
	if c.Options == nil {
		return defaultVal
	}
	if v, ok := c.Options[key]; ok {
		switch i := v.(type) {
		case int:
			return i
		case int64:
			return int(i)
		case float64:
			return int(i)
		}
	}
	return defaultVal
}

// GetInt64 returns an int64 option or the default value.
func (c DriverConfig) GetInt64(key string, defaultVal int64) int64 {
	if c.Options == nil {
		return defaultVal
	}
	if v, ok := c.Options[key]; ok {
		switch i := v.(type) {
		case int64:
			return i
		case int:
			return int64(i)
		case float64:
			return int64(i)
		}
	}
	return defaultVal
}

// GetFloat64 returns a float64 option or the default value.
func (c DriverConfig) GetFloat64(key string, defaultVal float64) float64 {
	if c.Options == nil {
		return defaultVal
	}
	if v, ok := c.Options[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return defaultVal
}

// GetBool returns a bool option or the default value.
func (c DriverConfig) GetBool(key string, defaultVal bool) bool {
	if c.Options == nil {
		return defaultVal
	}
	if v, ok := c.Options[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultVal
}

// With returns a copy of the config with the given option set.
func (c DriverConfig) With(key string, value any) DriverConfig {
	newOpts := make(map[string]any, len(c.Options)+1)
	for k, v := range c.Options {
		newOpts[k] = v
	}
	newOpts[key] = value
	return DriverConfig{
		DataDir:  c.DataDir,
		Language: c.Language,
		Options:  newOpts,
	}
}

// WithDataDir returns a copy with the given data directory.
func (c DriverConfig) WithDataDir(dir string) DriverConfig {
	return DriverConfig{
		DataDir:  dir,
		Language: c.Language,
		Options:  c.Options,
	}
}

// WithLanguage returns a copy with the given language.
func (c DriverConfig) WithLanguage(lang string) DriverConfig {
	return DriverConfig{
		DataDir:  c.DataDir,
		Language: lang,
		Options:  c.Options,
	}
}
