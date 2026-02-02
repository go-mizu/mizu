package skill

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// BuildSkillStatus creates a full SkillEntry from a Skill, its global config,
// and per-skill config. It computes missing requirements (bins, env, config
// paths, OS) and derives eligibility accordingly.
//
// skillCfg may contain:
//   - "enabled": bool  -- explicit disable
//   - "env": map[string]any -- env var overrides (e.g. {"MY_KEY": "val"})
//   - "apiKey": string  -- fallback for primaryEnv
//   - "skillKey": string -- override the skill key
func BuildSkillStatus(s *Skill, cfg map[string]any, skillCfg map[string]any) types.SkillEntry {
	key := s.Name
	skillKey := s.SkillKey
	if skillKey == "" {
		skillKey = s.Name
	}

	// Allow skillCfg to override the skillKey.
	if v, ok := skillCfg["skillKey"]; ok {
		if sk, ok := v.(string); ok && sk != "" {
			skillKey = sk
		}
	}

	entry := types.SkillEntry{
		Key:           key,
		Name:          s.Name,
		Description:   s.Description,
		Emoji:         s.Emoji,
		Source:        s.Source,
		FilePath:      filepath.Join(s.Dir, "SKILL.md"),
		BaseDir:       s.Dir,
		SkillKey:      skillKey,
		PrimaryEnv:    s.PrimaryEnv,
		Homepage:      s.Homepage,
		Always:        s.Always,
		UserInvocable: s.UserInvocable,
		Requirements: types.SkillRequirements{
			Bins:    nonNil(s.Requires.Binaries),
			AnyBins: nonNil(s.Requires.AnyBins),
			Env:     nonNil(s.Requires.Config),
			Config:  nonNil(s.Requires.CfgPaths),
			OS:      nonNil(s.Requires.OS),
		},
		Install: []types.SkillInstallOpt{},
	}

	// Check explicit disable via skillCfg.
	if v, ok := skillCfg["enabled"]; ok {
		if b, ok := v.(bool); ok && !b {
			entry.Disabled = true
		}
	}

	// Check bundled allowlist: if skills.allowBundled is set and this is a
	// bundled skill, it must appear in the allowlist.
	entry.BlockedByAllow = isBundledSkillBlocked(s, cfg)

	// Compute missing OS (always checked, even for always-skills).
	missingOS := computeMissingOS(s.Requires.OS)

	var missingBins []string
	var missingAnyBins []string
	var missingEnv []string

	if !s.Always {
		missingBins = computeMissingBins(s.Requires.Binaries)
		missingAnyBins = computeMissingAnyBins(s.Requires.AnyBins)
		missingEnv = computeMissingEnv(s.Requires.Config, s.PrimaryEnv, skillCfg)
	}

	// Missing config paths (always checked).
	missingConfig := computeMissingConfig(s.Requires.CfgPaths, cfg)

	entry.Missing = types.SkillMissing{
		Bins:    nonNil(missingBins),
		AnyBins: nonNil(missingAnyBins),
		Env:     nonNil(missingEnv),
		Config:  nonNil(missingConfig),
		OS:      nonNil(missingOS),
	}

	// Eligible when not disabled, not blocked by allowlist, and nothing is missing.
	entry.Eligible = !entry.Disabled &&
		!entry.BlockedByAllow &&
		len(entry.Missing.Bins) == 0 &&
		len(entry.Missing.AnyBins) == 0 &&
		len(entry.Missing.Env) == 0 &&
		len(entry.Missing.Config) == 0 &&
		len(entry.Missing.OS) == 0

	entry.Enabled = entry.Eligible

	return entry
}

// computeMissingBins returns binaries not found in PATH.
func computeMissingBins(bins []string) []string {
	var missing []string
	for _, bin := range bins {
		if _, err := exec.LookPath(bin); err != nil {
			missing = append(missing, bin)
		}
	}
	return missing
}

// computeMissingAnyBins returns the full list of anyBins only when none of
// them are found. If at least one is found, nothing is missing.
func computeMissingAnyBins(anyBins []string) []string {
	if len(anyBins) == 0 {
		return nil
	}
	for _, bin := range anyBins {
		if _, err := exec.LookPath(bin); err == nil {
			return nil
		}
	}
	// None found -- all are missing.
	return anyBins
}

// computeMissingEnv returns env var names that are not set. It considers:
//  1. Actual os.Getenv
//  2. skillCfg "env" map overrides (env var present in override counts as set)
//  3. For a var matching primaryEnv, skillCfg "apiKey" is a fallback
func computeMissingEnv(envVars []string, primaryEnv string, skillCfg map[string]any) []string {
	// Build override map from skillCfg["env"].
	envOverrides := map[string]string{}
	if raw, ok := skillCfg["env"]; ok {
		if m, ok := raw.(map[string]any); ok {
			for k, v := range m {
				if sv, ok := v.(string); ok {
					envOverrides[k] = sv
				}
			}
		}
	}

	// Extract apiKey fallback.
	apiKey := ""
	if v, ok := skillCfg["apiKey"]; ok {
		if sv, ok := v.(string); ok {
			apiKey = sv
		}
	}

	var missing []string
	for _, key := range envVars {
		// 1. Check real environment.
		if os.Getenv(key) != "" {
			continue
		}
		// 2. Check skillCfg env overrides.
		if v, ok := envOverrides[key]; ok && v != "" {
			continue
		}
		// 3. If this key is the primaryEnv, check apiKey fallback.
		if key == primaryEnv && primaryEnv != "" && apiKey != "" {
			continue
		}
		missing = append(missing, key)
	}
	return missing
}

// computeMissingConfig returns config paths that are not truthy in cfg.
func computeMissingConfig(cfgPaths []string, cfg map[string]any) []string {
	var missing []string
	for _, path := range cfgPaths {
		if !ConfigPathTruthy(cfg, path) {
			missing = append(missing, path)
		}
	}
	return missing
}

// computeMissingOS returns the required OS list if the current OS is not in it.
// Returns nil when the requirement is satisfied or empty.
func computeMissingOS(osList []string) []string {
	if len(osList) == 0 {
		return nil
	}
	current := runtime.GOOS
	for _, o := range osList {
		if o == current {
			return nil
		}
	}
	return osList
}

// isBundledSkillBlocked checks whether a bundled skill is blocked by the
// skills.allowBundled allowlist in cfg. Returns false when:
//   - The skill is not bundled (workspace/user skills are never blocked)
//   - No allowlist is configured (all bundled skills allowed)
//   - The skill's name or skillKey appears in the allowlist
func isBundledSkillBlocked(s *Skill, cfg map[string]any) bool {
	if s.Source != "bundled" {
		return false
	}
	skills, ok := cfg["skills"].(map[string]any)
	if !ok {
		return false
	}
	raw, ok := skills["allowBundled"]
	if !ok {
		return false
	}
	allowList, ok := raw.([]any)
	if !ok {
		return false
	}
	if len(allowList) == 0 {
		return false
	}
	key := s.SkillKey
	if key == "" {
		key = s.Name
	}
	for _, v := range allowList {
		if sv, ok := v.(string); ok {
			if sv == s.Name || sv == key {
				return false
			}
		}
	}
	return true
}

// nonNil ensures a slice is non-nil (returns empty []string{} instead of nil).
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
