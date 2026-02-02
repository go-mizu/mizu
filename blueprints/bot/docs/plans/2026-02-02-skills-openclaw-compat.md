# Skills OpenClaw Compatibility Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Achieve 100% OpenClaw skills feature compatibility in OpenBot's gateway, backend, and frontend.

**Architecture:** Enhance the existing `pkg/skill` package with full status reporting, per-skill config (apiKey/env), and install support. Extend gateway RPC with `skills.bins`, `skills.install`, `skills.update` (full). Upgrade frontend SkillsPage with detail view, API key input, env editor, and install buttons.

**Tech Stack:** Go (backend), TypeScript/React (frontend), SQLite (config persistence via JSON file), WebSocket RPC

---

### Task 1: Enhance SkillEntry Type with Full Status Fields

**Files:**
- Modify: `types/types.go:229-241`
- Test: `pkg/skill/skill_test.go` (existing tests must still pass)

**Step 1: Update the SkillEntry struct**

Add fields to match OpenClaw's full SkillStatus. In `types/types.go`, replace the existing `SkillEntry`:

```go
// SkillEntry represents a skill's full status for the dashboard.
type SkillEntry struct {
	Key               string            `json:"key"`
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	Emoji             string            `json:"emoji"`
	Source            string            `json:"source"`
	FilePath          string            `json:"filePath,omitempty"`
	BaseDir           string            `json:"baseDir,omitempty"`
	SkillKey          string            `json:"skillKey"`
	PrimaryEnv        string            `json:"primaryEnv,omitempty"`
	Homepage          string            `json:"homepage,omitempty"`
	Always            bool              `json:"always"`
	Disabled          bool              `json:"disabled"`
	BlockedByAllow    bool              `json:"blockedByAllowlist"`
	Eligible          bool              `json:"eligible"`
	Enabled           bool              `json:"enabled"`
	UserInvocable     bool              `json:"userInvocable"`
	Requirements      SkillRequirements `json:"requirements"`
	Missing           SkillMissing      `json:"missing"`
	Install           []SkillInstallOpt `json:"install"`
}

// SkillRequirements lists what a skill needs.
type SkillRequirements struct {
	Bins    []string `json:"bins"`
	AnyBins []string `json:"anyBins"`
	Env     []string `json:"env"`
	Config  []string `json:"config"`
	OS      []string `json:"os"`
}

// SkillMissing lists what's not satisfied.
type SkillMissing struct {
	Bins    []string `json:"bins"`
	AnyBins []string `json:"anyBins"`
	Env     []string `json:"env"`
	Config  []string `json:"config"`
	OS      []string `json:"os"`
}

// SkillInstallOpt represents an install option for a missing dependency.
type SkillInstallOpt struct {
	ID    string   `json:"id"`
	Kind  string   `json:"kind"`
	Label string   `json:"label"`
	Bins  []string `json:"bins"`
}
```

**Step 2: Run existing tests to verify no regressions**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/bot && go test ./pkg/skill/... -v -count=1`
Expected: All existing tests PASS

**Step 3: Commit**

```bash
git add types/types.go
git commit -m "feat(skills): enhance SkillEntry with full OpenClaw-compatible status fields"
```

---

### Task 2: Add Skill Missing-Requirements Computation

**Files:**
- Modify: `pkg/skill/skill.go`
- Create: `pkg/skill/status.go`
- Test: `pkg/skill/status_test.go`

**Step 1: Create status.go with BuildSkillStatus function**

```go
package skill

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// BuildSkillStatus builds a full status entry for a single skill.
func BuildSkillStatus(s *Skill, cfg map[string]any, skillCfg map[string]any) types.SkillEntry {
	key := s.SkillKey
	if key == "" {
		key = s.Name
	}

	disabled := false
	if skillCfg != nil {
		if v, ok := skillCfg["enabled"]; ok {
			if b, ok := v.(bool); ok && !b {
				disabled = true
			}
		}
	}

	reqs := types.SkillRequirements{
		Bins:    nonNil(s.Requires.Binaries),
		AnyBins: nonNil(s.Requires.AnyBins),
		Env:     nonNil(s.Requires.Config),
		Config:  nonNil(s.Requires.CfgPaths),
		OS:      nonNil(s.Requires.OS),
	}

	missing := computeMissing(s, cfg, skillCfg)
	eligible := !disabled && s.Ready

	entry := types.SkillEntry{
		Key:           key,
		Name:          s.Name,
		Description:   s.Description,
		Emoji:         s.Emoji,
		Source:        s.Source,
		FilePath:      filepath.Join(s.Dir, "SKILL.md"),
		BaseDir:       s.Dir,
		SkillKey:      key,
		PrimaryEnv:    s.PrimaryEnv,
		Homepage:      s.Homepage,
		Always:        s.Always,
		Disabled:      disabled,
		Eligible:      eligible,
		Enabled:       !disabled,
		UserInvocable: s.UserInvocable,
		Requirements:  reqs,
		Missing:       missing,
		Install:       []types.SkillInstallOpt{},
	}

	return entry
}

func computeMissing(s *Skill, cfg map[string]any, skillCfg map[string]any) types.SkillMissing {
	m := types.SkillMissing{
		Bins:    []string{},
		AnyBins: []string{},
		Env:     []string{},
		Config:  []string{},
		OS:      []string{},
	}

	if s.Always {
		return m
	}

	// Missing bins
	for _, bin := range s.Requires.Binaries {
		if _, err := exec.LookPath(bin); err != nil {
			m.Bins = append(m.Bins, bin)
		}
	}

	// Missing anyBins
	if len(s.Requires.AnyBins) > 0 {
		found := false
		for _, bin := range s.Requires.AnyBins {
			if _, err := exec.LookPath(bin); err == nil {
				found = true
				break
			}
		}
		if !found {
			m.AnyBins = append(m.AnyBins, s.Requires.AnyBins...)
		}
	}

	// Missing env
	for _, envName := range s.Requires.Config {
		if os.Getenv(envName) != "" {
			continue
		}
		// Check skill config env overrides
		if skillCfg != nil {
			if envMap, ok := skillCfg["env"].(map[string]any); ok {
				if v, ok := envMap[envName]; ok {
					if str, ok := v.(string); ok && str != "" {
						continue
					}
				}
			}
			// Check apiKey as primaryEnv fallback
			if s.PrimaryEnv == envName {
				if apiKey, ok := skillCfg["apiKey"].(string); ok && apiKey != "" {
					continue
				}
			}
		}
		m.Env = append(m.Env, envName)
	}

	// Missing config paths
	for _, path := range s.Requires.CfgPaths {
		if !ConfigPathTruthy(cfg, path) {
			m.Config = append(m.Config, path)
		}
	}

	// Missing OS
	if len(s.Requires.OS) > 0 {
		currentOS := runtime.GOOS
		found := false
		for _, o := range s.Requires.OS {
			if o == currentOS {
				found = true
				break
			}
		}
		if !found {
			m.OS = append(m.OS, s.Requires.OS...)
		}
	}

	return m
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
```

**Step 2: Write tests in status_test.go**

```go
package skill

import (
	"testing"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

func TestBuildSkillStatus_BasicReady(t *testing.T) {
	sk := &Skill{Name: "test", Description: "Test skill", Ready: true, UserInvocable: true}
	entry := BuildSkillStatus(sk, nil, nil)
	if !entry.Eligible {
		t.Error("expected eligible")
	}
	if entry.Disabled {
		t.Error("expected not disabled")
	}
	if entry.Key != "test" {
		t.Errorf("Key = %q; want %q", entry.Key, "test")
	}
}

func TestBuildSkillStatus_Disabled(t *testing.T) {
	sk := &Skill{Name: "test", Ready: true}
	skillCfg := map[string]any{"enabled": false}
	entry := BuildSkillStatus(sk, nil, skillCfg)
	if entry.Eligible {
		t.Error("expected not eligible when disabled")
	}
	if !entry.Disabled {
		t.Error("expected disabled")
	}
}

func TestBuildSkillStatus_MissingBins(t *testing.T) {
	sk := &Skill{
		Name:     "test",
		Ready:    false,
		Requires: Requires{Binaries: []string{"nonexistent_xyz_123"}},
	}
	entry := BuildSkillStatus(sk, nil, nil)
	if len(entry.Missing.Bins) != 1 {
		t.Errorf("Missing.Bins = %v; want 1 entry", entry.Missing.Bins)
	}
}

func TestBuildSkillStatus_MissingEnvWithOverride(t *testing.T) {
	sk := &Skill{
		Name:     "test",
		Ready:    false,
		Requires: Requires{Config: []string{"MY_API_KEY"}},
	}
	skillCfg := map[string]any{
		"env": map[string]any{"MY_API_KEY": "secret"},
	}
	entry := BuildSkillStatus(sk, nil, skillCfg)
	if len(entry.Missing.Env) != 0 {
		t.Errorf("Missing.Env = %v; want empty (env override provided)", entry.Missing.Env)
	}
}

func TestBuildSkillStatus_SkillKeyOverride(t *testing.T) {
	sk := &Skill{Name: "test", SkillKey: "custom-key"}
	entry := BuildSkillStatus(sk, nil, nil)
	if entry.SkillKey != "custom-key" {
		t.Errorf("SkillKey = %q; want %q", entry.SkillKey, "custom-key")
	}
}

func TestBuildSkillStatus_NonNilArrays(t *testing.T) {
	sk := &Skill{Name: "test"}
	entry := BuildSkillStatus(sk, nil, nil)
	// All arrays should be non-nil (empty, not null in JSON)
	if entry.Requirements.Bins == nil {
		t.Error("Requirements.Bins should not be nil")
	}
	if entry.Missing.Bins == nil {
		t.Error("Missing.Bins should not be nil")
	}
	if entry.Install == nil {
		t.Error("Install should not be nil")
	}
}
```

**Step 3: Run tests**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/bot && go test ./pkg/skill/... -v -count=1`

**Step 4: Commit**

```bash
git add pkg/skill/status.go pkg/skill/status_test.go
git commit -m "feat(skills): add BuildSkillStatus with full missing-requirements computation"
```

---

### Task 3: Add Per-Skill Config Support

**Files:**
- Modify: `pkg/config/config.go`
- Test: `pkg/config/config_test.go`

**Step 1: Add skill config resolution helpers to config package**

Add functions to load/save per-skill config entries:

```go
// ResolveSkillConfig returns the per-skill config for a given skill key.
func ResolveSkillConfig(cfg map[string]any, skillKey string) map[string]any {
	skills, _ := cfg["skills"].(map[string]any)
	if skills == nil {
		return nil
	}
	entries, _ := skills["entries"].(map[string]any)
	if entries == nil {
		return nil
	}
	entry, _ := entries[skillKey].(map[string]any)
	return entry
}

// UpdateSkillConfig updates per-skill config (enabled, apiKey, env).
func UpdateSkillConfig(cfg map[string]any, skillKey string, enabled *bool, apiKey *string, env map[string]string) {
	skills, _ := cfg["skills"].(map[string]any)
	if skills == nil {
		skills = make(map[string]any)
		cfg["skills"] = skills
	}
	entries, _ := skills["entries"].(map[string]any)
	if entries == nil {
		entries = make(map[string]any)
		skills["entries"] = entries
	}
	entry, _ := entries[skillKey].(map[string]any)
	if entry == nil {
		entry = make(map[string]any)
		entries[skillKey] = entry
	}

	if enabled != nil {
		entry["enabled"] = *enabled
	}
	if apiKey != nil {
		trimmed := strings.TrimSpace(*apiKey)
		if trimmed != "" {
			entry["apiKey"] = trimmed
		} else {
			delete(entry, "apiKey")
		}
	}
	if env != nil {
		envMap, _ := entry["env"].(map[string]any)
		if envMap == nil {
			envMap = make(map[string]any)
		}
		for k, v := range env {
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			if k == "" {
				continue
			}
			if v == "" {
				delete(envMap, k)
			} else {
				envMap[k] = v
			}
		}
		entry["env"] = envMap
	}
}
```

**Step 2: Write tests**

**Step 3: Run tests, commit**

---

### Task 4: Enhance Gateway RPC - Full skills.status

**Files:**
- Modify: `app/web/rpc/rpc.go:456-527`

**Step 1: Rewrite registerSkillMethods with full status**

Replace the `registerSkillMethods` function to return full skill status matching OpenClaw schema, load per-skill config, and compute missing requirements.

**Step 2: Run the server and test via dashboard**

**Step 3: Commit**

---

### Task 5: Add skills.update RPC (apiKey + env)

**Files:**
- Modify: `app/web/rpc/rpc.go`

**Step 1: Replace skills.toggle with skills.update**

Support `{skillKey, enabled?, apiKey?, env?}` params matching OpenClaw's `skills.update` schema.

**Step 2: Test via dashboard**

**Step 3: Commit**

---

### Task 6: Add skills.bins RPC

**Files:**
- Modify: `app/web/rpc/rpc.go`

**Step 1: Add skills.bins handler**

Collect all required binaries across all loaded skills and return as sorted array.

**Step 2: Commit**

---

### Task 7: Add skills.install RPC

**Files:**
- Create: `pkg/skill/install.go`
- Create: `pkg/skill/install_test.go`
- Modify: `app/web/rpc/rpc.go`

**Step 1: Create install.go with InstallSkill function**

Support brew, node (npm), go, uv, and download install kinds. Parse install specs from metadata. Execute commands with timeout.

**Step 2: Write tests**

**Step 3: Register skills.install RPC handler**

**Step 4: Commit**

---

### Task 8: Enhance Frontend SkillsPage

**Files:**
- Modify: `app/frontend/src/pages/SkillsPage.tsx`

**Step 1: Update Skill interface with full fields**

**Step 2: Add skill detail expand/collapse view**

Show requirements, missing items, homepage link, install options.

**Step 3: Add per-skill API key input**

Input field that calls `skills.update` with `{skillKey, apiKey}`.

**Step 4: Add per-skill env vars editor**

Key-value editor that calls `skills.update` with `{skillKey, env}`.

**Step 5: Add install buttons for missing dependencies**

Button per install option that calls `skills.install`.

**Step 6: Add status badges (ready/disabled/missing/blocked)**

Color-coded badges matching OpenClaw's CLI format.

**Step 7: Commit**

---

### Task 9: Integration Tests

**Files:**
- Create: `test/compat/skills_rpc_test.go`

**Step 1: Test skills.status returns full schema**

**Step 2: Test skills.update with apiKey and env**

**Step 3: Test skills.bins returns all required binaries**

**Step 4: Commit**

---

### Task 10: Run Full Test Suite

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/bot && go test ./... -v -count=1`

Verify all tests pass including existing compat tests.
