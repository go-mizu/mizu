package config

import "testing"

func TestResolveSkillConfig_MissingSkills(t *testing.T) {
	cfg := map[string]any{}
	got := ResolveSkillConfig(cfg, "web-search")
	if got != nil {
		t.Errorf("expected nil for missing skills section, got %v", got)
	}
}

func TestResolveSkillConfig_MissingEntries(t *testing.T) {
	cfg := map[string]any{
		"skills": map[string]any{},
	}
	got := ResolveSkillConfig(cfg, "web-search")
	if got != nil {
		t.Errorf("expected nil for missing entries section, got %v", got)
	}
}

func TestResolveSkillConfig_MissingKey(t *testing.T) {
	cfg := map[string]any{
		"skills": map[string]any{
			"entries": map[string]any{
				"other-skill": map[string]any{"enabled": true},
			},
		},
	}
	got := ResolveSkillConfig(cfg, "web-search")
	if got != nil {
		t.Errorf("expected nil for missing skill key, got %v", got)
	}
}

func TestResolveSkillConfig_ReturnsEntry(t *testing.T) {
	cfg := map[string]any{
		"skills": map[string]any{
			"entries": map[string]any{
				"web-search": map[string]any{
					"enabled": true,
					"apiKey":  "sk-123",
				},
			},
		},
	}
	got := ResolveSkillConfig(cfg, "web-search")
	if got == nil {
		t.Fatal("expected non-nil entry")
	}
	if enabled, ok := got["enabled"].(bool); !ok || !enabled {
		t.Errorf("expected enabled=true, got %v", got["enabled"])
	}
	if apiKey, ok := got["apiKey"].(string); !ok || apiKey != "sk-123" {
		t.Errorf("expected apiKey=sk-123, got %v", got["apiKey"])
	}
}

func TestUpdateSkillConfig_CreatesNestedStructure(t *testing.T) {
	cfg := map[string]any{}
	enabled := true
	UpdateSkillConfig(cfg, "web-search", &enabled, nil, nil)

	// Verify the full nested structure was created.
	skills, ok := cfg["skills"].(map[string]any)
	if !ok {
		t.Fatal("expected skills map to be created")
	}
	entries, ok := skills["entries"].(map[string]any)
	if !ok {
		t.Fatal("expected entries map to be created")
	}
	entry, ok := entries["web-search"].(map[string]any)
	if !ok {
		t.Fatal("expected web-search entry to be created")
	}
	if v, ok := entry["enabled"].(bool); !ok || !v {
		t.Errorf("expected enabled=true, got %v", entry["enabled"])
	}
}

func TestUpdateSkillConfig_SetsEnabled(t *testing.T) {
	cfg := map[string]any{
		"skills": map[string]any{
			"entries": map[string]any{
				"web-search": map[string]any{
					"enabled": true,
				},
			},
		},
	}

	disabled := false
	UpdateSkillConfig(cfg, "web-search", &disabled, nil, nil)

	entry := ResolveSkillConfig(cfg, "web-search")
	if entry == nil {
		t.Fatal("expected entry to exist")
	}
	if v, ok := entry["enabled"].(bool); !ok || v {
		t.Errorf("expected enabled=false, got %v", entry["enabled"])
	}
}

func TestUpdateSkillConfig_SetsApiKey(t *testing.T) {
	cfg := map[string]any{}
	key := "sk-new-key"
	UpdateSkillConfig(cfg, "web-search", nil, &key, nil)

	entry := ResolveSkillConfig(cfg, "web-search")
	if entry == nil {
		t.Fatal("expected entry to exist")
	}
	if v, ok := entry["apiKey"].(string); !ok || v != "sk-new-key" {
		t.Errorf("expected apiKey=sk-new-key, got %v", entry["apiKey"])
	}
}

func TestUpdateSkillConfig_RemovesEmptyApiKey(t *testing.T) {
	cfg := map[string]any{
		"skills": map[string]any{
			"entries": map[string]any{
				"web-search": map[string]any{
					"apiKey": "old-key",
				},
			},
		},
	}

	empty := "   "
	UpdateSkillConfig(cfg, "web-search", nil, &empty, nil)

	entry := ResolveSkillConfig(cfg, "web-search")
	if entry == nil {
		t.Fatal("expected entry to exist")
	}
	if _, exists := entry["apiKey"]; exists {
		t.Errorf("expected apiKey to be removed, but it still exists: %v", entry["apiKey"])
	}
}

func TestUpdateSkillConfig_MergesEnv(t *testing.T) {
	cfg := map[string]any{
		"skills": map[string]any{
			"entries": map[string]any{
				"web-search": map[string]any{
					"env": map[string]any{
						"EXISTING_VAR": "keep-me",
						"REMOVE_ME":    "old-value",
					},
				},
			},
		},
	}

	env := map[string]string{
		"NEW_VAR":   "new-value",
		"REMOVE_ME": "",
	}
	UpdateSkillConfig(cfg, "web-search", nil, nil, env)

	entry := ResolveSkillConfig(cfg, "web-search")
	if entry == nil {
		t.Fatal("expected entry to exist")
	}
	envMap, ok := entry["env"].(map[string]any)
	if !ok {
		t.Fatal("expected env map to exist")
	}
	if v, ok := envMap["EXISTING_VAR"].(string); !ok || v != "keep-me" {
		t.Errorf("expected EXISTING_VAR=keep-me, got %v", envMap["EXISTING_VAR"])
	}
	if v, ok := envMap["NEW_VAR"].(string); !ok || v != "new-value" {
		t.Errorf("expected NEW_VAR=new-value, got %v", envMap["NEW_VAR"])
	}
	if _, exists := envMap["REMOVE_ME"]; exists {
		t.Errorf("expected REMOVE_ME to be removed, but it still exists: %v", envMap["REMOVE_ME"])
	}
}

func TestUpdateSkillConfig_CreatesEnvIfMissing(t *testing.T) {
	cfg := map[string]any{}
	env := map[string]string{
		"API_URL": "https://example.com",
	}
	UpdateSkillConfig(cfg, "web-search", nil, nil, env)

	entry := ResolveSkillConfig(cfg, "web-search")
	if entry == nil {
		t.Fatal("expected entry to exist")
	}
	envMap, ok := entry["env"].(map[string]any)
	if !ok {
		t.Fatal("expected env map to be created")
	}
	if v, ok := envMap["API_URL"].(string); !ok || v != "https://example.com" {
		t.Errorf("expected API_URL=https://example.com, got %v", envMap["API_URL"])
	}
}
