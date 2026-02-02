package config

import "strings"

// ResolveSkillConfig returns the per-skill config entry from
// cfg["skills"]["entries"][skillKey], or nil if not found.
func ResolveSkillConfig(cfg map[string]any, skillKey string) map[string]any {
	skills, ok := cfg["skills"].(map[string]any)
	if !ok {
		return nil
	}
	entries, ok := skills["entries"].(map[string]any)
	if !ok {
		return nil
	}
	entry, ok := entries[skillKey].(map[string]any)
	if !ok {
		return nil
	}
	return entry
}

// UpdateSkillConfig updates per-skill config in the map. It creates the
// nested structure (skills.entries.<key>) if missing.
//
//   - enabled: sets cfg.skills.entries.<key>.enabled
//   - apiKey: sets cfg.skills.entries.<key>.apiKey (removes the key if empty after trim)
//   - env: merges into cfg.skills.entries.<key>.env (removes keys whose values are empty)
func UpdateSkillConfig(cfg map[string]any, skillKey string, enabled *bool, apiKey *string, env map[string]string) {
	// Ensure skills map exists.
	skills, ok := cfg["skills"].(map[string]any)
	if !ok {
		skills = make(map[string]any)
		cfg["skills"] = skills
	}

	// Ensure entries map exists.
	entries, ok := skills["entries"].(map[string]any)
	if !ok {
		entries = make(map[string]any)
		skills["entries"] = entries
	}

	// Ensure skill entry exists.
	entry, ok := entries[skillKey].(map[string]any)
	if !ok {
		entry = make(map[string]any)
		entries[skillKey] = entry
	}

	// Set enabled if provided.
	if enabled != nil {
		entry["enabled"] = *enabled
	}

	// Set or remove apiKey if provided.
	if apiKey != nil {
		trimmed := strings.TrimSpace(*apiKey)
		if trimmed == "" {
			delete(entry, "apiKey")
		} else {
			entry["apiKey"] = trimmed
		}
	}

	// Merge env if provided.
	if env != nil {
		existing, ok := entry["env"].(map[string]any)
		if !ok {
			existing = make(map[string]any)
			entry["env"] = existing
		}
		for k, v := range env {
			if v == "" {
				delete(existing, k)
			} else {
				existing[k] = v
			}
		}
	}
}
