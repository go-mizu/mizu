package gateway

import (
	"os"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/config"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/skill"
)

// applySkillEnvOverrides sets environment variables defined by skill configurations.
// For each ready skill it checks the per-skill config (from openbot.json) for:
//   - "env" map: sets each key=value as a process env var
//   - "apiKey" + skill.PrimaryEnv: sets primaryEnv = apiKey
//
// Returns a cleanup function that restores all original env values.
// This matches OpenClaw's applySkillEnvOverrides (dist/agents/skills/env-overrides.js).
func applySkillEnvOverrides(skills []*skill.Skill, workspaceDir string) func() {
	if len(skills) == 0 {
		return func() {}
	}

	// Load raw config to resolve per-skill configs.
	rawCfg, err := config.LoadRawConfig(config.DefaultConfigPath())
	if err != nil {
		return func() {}
	}

	type envRestore struct {
		key    string
		value  string
		exists bool
	}
	var restores []envRestore

	save := func(key string) {
		old, exists := os.LookupEnv(key)
		restores = append(restores, envRestore{key: key, value: old, exists: exists})
	}

	for _, s := range skills {
		if !s.Ready {
			continue
		}

		skillKey := s.SkillKey
		if skillKey == "" {
			skillKey = s.Name
		}

		skillCfg := config.ResolveSkillConfig(rawCfg, skillKey)
		if skillCfg == nil {
			continue
		}

		// Apply env map overrides.
		if raw, ok := skillCfg["env"]; ok {
			if envMap, ok := raw.(map[string]any); ok {
				for k, v := range envMap {
					if sv, ok := v.(string); ok && sv != "" {
						save(k)
						os.Setenv(k, sv)
					}
				}
			}
		}

		// Apply apiKey → primaryEnv.
		if s.PrimaryEnv != "" {
			if apiKey, ok := skillCfg["apiKey"]; ok {
				if sv, ok := apiKey.(string); ok && sv != "" {
					save(s.PrimaryEnv)
					os.Setenv(s.PrimaryEnv, sv)
				}
			}
		}
	}

	return func() {
		for i := len(restores) - 1; i >= 0; i-- {
			r := restores[i]
			if r.exists {
				os.Setenv(r.key, r.value)
			} else {
				os.Unsetenv(r.key)
			}
		}
	}
}
