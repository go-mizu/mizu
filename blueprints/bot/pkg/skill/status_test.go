package skill

import (
	"runtime"
	"testing"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// ---------------------------------------------------------------------------
// BuildSkillStatus
// ---------------------------------------------------------------------------

func TestBuildSkillStatus_BasicReadySkill(t *testing.T) {
	s := &Skill{
		Name:          "greeter",
		Description:   "Greets people",
		Emoji:         "wave",
		Source:        "bundled",
		Dir:           "/skills/greeter",
		UserInvocable: true,
	}
	entry := BuildSkillStatus(s, nil, map[string]any{})

	if entry.Key != "greeter" {
		t.Errorf("Key = %q; want %q", entry.Key, "greeter")
	}
	if entry.Name != "greeter" {
		t.Errorf("Name = %q; want %q", entry.Name, "greeter")
	}
	if !entry.Eligible {
		t.Error("Eligible = false; want true for skill with no requirements")
	}
	if !entry.Enabled {
		t.Error("Enabled = false; want true for eligible skill")
	}
	if entry.Disabled {
		t.Error("Disabled = true; want false")
	}
	if entry.FilePath != "/skills/greeter/SKILL.md" {
		t.Errorf("FilePath = %q; want %q", entry.FilePath, "/skills/greeter/SKILL.md")
	}
}

func TestBuildSkillStatus_DisabledSkill(t *testing.T) {
	s := &Skill{
		Name:   "disabled-skill",
		Source: "bundled",
		Dir:    "/skills/disabled-skill",
	}
	skillCfg := map[string]any{"enabled": false}
	entry := BuildSkillStatus(s, nil, skillCfg)

	if !entry.Disabled {
		t.Error("Disabled = false; want true when skillCfg enabled=false")
	}
	if entry.Eligible {
		t.Error("Eligible = true; want false for disabled skill")
	}
	if entry.Enabled {
		t.Error("Enabled = true; want false for disabled skill")
	}
}

func TestBuildSkillStatus_MissingBinary(t *testing.T) {
	s := &Skill{
		Name:   "needs-bin",
		Source: "bundled",
		Dir:    "/skills/needs-bin",
		Requires: Requires{
			Binaries: []string{"nonexistent_binary_xyz_test_999"},
		},
	}
	entry := BuildSkillStatus(s, nil, map[string]any{})

	if entry.Eligible {
		t.Error("Eligible = true; want false when binary is missing")
	}
	if len(entry.Missing.Bins) != 1 || entry.Missing.Bins[0] != "nonexistent_binary_xyz_test_999" {
		t.Errorf("Missing.Bins = %v; want [nonexistent_binary_xyz_test_999]", entry.Missing.Bins)
	}
	// Requirements should still list the binary.
	if len(entry.Requirements.Bins) != 1 || entry.Requirements.Bins[0] != "nonexistent_binary_xyz_test_999" {
		t.Errorf("Requirements.Bins = %v; want [nonexistent_binary_xyz_test_999]", entry.Requirements.Bins)
	}
}

func TestBuildSkillStatus_MissingEnvWithOverride(t *testing.T) {
	s := &Skill{
		Name:   "needs-env",
		Source: "bundled",
		Dir:    "/skills/needs-env",
		Requires: Requires{
			Config: []string{"MY_SECRET_KEY_XYZ_TEST"},
		},
	}
	// The env var is NOT set in the real environment, but the skillCfg provides
	// an override, so it should not be missing.
	skillCfg := map[string]any{
		"env": map[string]any{
			"MY_SECRET_KEY_XYZ_TEST": "overridden-value",
		},
	}
	entry := BuildSkillStatus(s, nil, skillCfg)

	if len(entry.Missing.Env) != 0 {
		t.Errorf("Missing.Env = %v; want empty (env override should satisfy)", entry.Missing.Env)
	}
	if !entry.Eligible {
		t.Error("Eligible = false; want true (env override satisfies requirement)")
	}
}

func TestBuildSkillStatus_MissingEnvWithApiKeyFallback(t *testing.T) {
	s := &Skill{
		Name:       "needs-api",
		Source:     "bundled",
		Dir:        "/skills/needs-api",
		PrimaryEnv: "MY_API_KEY_XYZ_TEST",
		Requires: Requires{
			Config: []string{"MY_API_KEY_XYZ_TEST"},
		},
	}
	// The env var is NOT set, but skillCfg has an apiKey that should act as
	// fallback for the primaryEnv variable.
	skillCfg := map[string]any{
		"apiKey": "sk-fallback-key-123",
	}
	entry := BuildSkillStatus(s, nil, skillCfg)

	if len(entry.Missing.Env) != 0 {
		t.Errorf("Missing.Env = %v; want empty (apiKey fallback should satisfy primaryEnv)", entry.Missing.Env)
	}
	if !entry.Eligible {
		t.Error("Eligible = false; want true (apiKey fallback satisfies primaryEnv)")
	}
}

func TestBuildSkillStatus_SkillKeyOverride(t *testing.T) {
	s := &Skill{
		Name:     "my-skill",
		SkillKey: "original-key",
		Source:   "bundled",
		Dir:      "/skills/my-skill",
	}
	skillCfg := map[string]any{
		"skillKey": "overridden-key",
	}
	entry := BuildSkillStatus(s, nil, skillCfg)

	if entry.SkillKey != "overridden-key" {
		t.Errorf("SkillKey = %q; want %q", entry.SkillKey, "overridden-key")
	}
}

func TestBuildSkillStatus_SkillKeyDefaultsToName(t *testing.T) {
	s := &Skill{
		Name:   "my-skill",
		Source: "bundled",
		Dir:    "/skills/my-skill",
	}
	entry := BuildSkillStatus(s, nil, map[string]any{})

	if entry.SkillKey != "my-skill" {
		t.Errorf("SkillKey = %q; want %q (should default to Name)", entry.SkillKey, "my-skill")
	}
}

func TestBuildSkillStatus_AllArraysNonNil(t *testing.T) {
	s := &Skill{
		Name:   "empty-reqs",
		Source: "bundled",
		Dir:    "/skills/empty-reqs",
	}
	entry := BuildSkillStatus(s, nil, map[string]any{})

	// All slice fields must be non-nil (empty, not nil).
	assertNonNilSlice(t, "Requirements.Bins", entry.Requirements.Bins)
	assertNonNilSlice(t, "Requirements.AnyBins", entry.Requirements.AnyBins)
	assertNonNilSlice(t, "Requirements.Env", entry.Requirements.Env)
	assertNonNilSlice(t, "Requirements.Config", entry.Requirements.Config)
	assertNonNilSlice(t, "Requirements.OS", entry.Requirements.OS)
	assertNonNilSlice(t, "Missing.Bins", entry.Missing.Bins)
	assertNonNilSlice(t, "Missing.AnyBins", entry.Missing.AnyBins)
	assertNonNilSlice(t, "Missing.Env", entry.Missing.Env)
	assertNonNilSlice(t, "Missing.Config", entry.Missing.Config)
	assertNonNilSlice(t, "Missing.OS", entry.Missing.OS)

	if entry.Install == nil {
		t.Error("Install is nil; want non-nil empty slice")
	}
}

func TestBuildSkillStatus_AlwaysBypassesBinAndEnv(t *testing.T) {
	s := &Skill{
		Name:   "always-skill",
		Source: "bundled",
		Dir:    "/skills/always-skill",
		Always: true,
		Requires: Requires{
			Binaries: []string{"nonexistent_binary_xyz_always"},
			Config:   []string{"MISSING_ENV_VAR_XYZ_ALWAYS"},
		},
	}
	entry := BuildSkillStatus(s, nil, map[string]any{})

	if !entry.Always {
		t.Error("Always = false; want true")
	}
	// Always skills should skip bin and env checks.
	if len(entry.Missing.Bins) != 0 {
		t.Errorf("Missing.Bins = %v; want empty (always skill bypasses bin check)", entry.Missing.Bins)
	}
	if len(entry.Missing.Env) != 0 {
		t.Errorf("Missing.Env = %v; want empty (always skill bypasses env check)", entry.Missing.Env)
	}
	if !entry.Eligible {
		t.Error("Eligible = false; want true for always skill that passes OS check")
	}
}

func TestBuildSkillStatus_OSMismatch(t *testing.T) {
	// Pick an OS that is definitely not the current one.
	fakeOS := "impossible_os_xyz"
	if runtime.GOOS == fakeOS {
		t.Skip("somehow running on impossible_os_xyz")
	}

	s := &Skill{
		Name:   "os-restricted",
		Source: "bundled",
		Dir:    "/skills/os-restricted",
		Requires: Requires{
			OS: []string{fakeOS},
		},
	}
	entry := BuildSkillStatus(s, nil, map[string]any{})

	if entry.Eligible {
		t.Error("Eligible = true; want false when OS does not match")
	}
	if len(entry.Missing.OS) == 0 {
		t.Error("Missing.OS is empty; want non-empty when OS does not match")
	}
	if entry.Missing.OS[0] != fakeOS {
		t.Errorf("Missing.OS = %v; want [%s]", entry.Missing.OS, fakeOS)
	}
}

func TestBuildSkillStatus_OSMatch(t *testing.T) {
	s := &Skill{
		Name:   "os-match",
		Source: "bundled",
		Dir:    "/skills/os-match",
		Requires: Requires{
			OS: []string{runtime.GOOS},
		},
	}
	entry := BuildSkillStatus(s, nil, map[string]any{})

	if !entry.Eligible {
		t.Error("Eligible = false; want true when OS matches")
	}
	if len(entry.Missing.OS) != 0 {
		t.Errorf("Missing.OS = %v; want empty when OS matches", entry.Missing.OS)
	}
}

func TestBuildSkillStatus_AlwaysStillChecksOS(t *testing.T) {
	s := &Skill{
		Name:   "always-os-fail",
		Source: "bundled",
		Dir:    "/skills/always-os-fail",
		Always: true,
		Requires: Requires{
			OS: []string{"impossible_os_xyz"},
		},
	}
	entry := BuildSkillStatus(s, nil, map[string]any{})

	if entry.Eligible {
		t.Error("Eligible = true; want false (always=true still checks OS)")
	}
	if len(entry.Missing.OS) == 0 {
		t.Error("Missing.OS is empty; want non-empty for OS mismatch even with always=true")
	}
}

func TestBuildSkillStatus_MissingConfigPath(t *testing.T) {
	s := &Skill{
		Name:   "needs-config",
		Source: "bundled",
		Dir:    "/skills/needs-config",
		Requires: Requires{
			CfgPaths: []string{"channels.slack"},
		},
	}
	// Config does not contain channels.slack.
	cfg := map[string]any{}
	entry := BuildSkillStatus(s, cfg, map[string]any{})

	if entry.Eligible {
		t.Error("Eligible = true; want false when config path is missing")
	}
	if len(entry.Missing.Config) != 1 || entry.Missing.Config[0] != "channels.slack" {
		t.Errorf("Missing.Config = %v; want [channels.slack]", entry.Missing.Config)
	}
}

func TestBuildSkillStatus_ConfigPathPresent(t *testing.T) {
	s := &Skill{
		Name:   "has-config",
		Source: "bundled",
		Dir:    "/skills/has-config",
		Requires: Requires{
			CfgPaths: []string{"channels.slack"},
		},
	}
	cfg := map[string]any{
		"channels": map[string]any{
			"slack": true,
		},
	}
	entry := BuildSkillStatus(s, cfg, map[string]any{})

	if !entry.Eligible {
		t.Error("Eligible = false; want true when config path is present")
	}
	if len(entry.Missing.Config) != 0 {
		t.Errorf("Missing.Config = %v; want empty", entry.Missing.Config)
	}
}

func TestBuildSkillStatus_MissingEnvNoFallback(t *testing.T) {
	s := &Skill{
		Name:   "needs-env-nofb",
		Source: "bundled",
		Dir:    "/skills/needs-env-nofb",
		Requires: Requires{
			Config: []string{"TOTALLY_MISSING_ENV_VAR_XYZ_123"},
		},
	}
	entry := BuildSkillStatus(s, nil, map[string]any{})

	if entry.Eligible {
		t.Error("Eligible = true; want false when env var is missing without fallback")
	}
	if len(entry.Missing.Env) != 1 || entry.Missing.Env[0] != "TOTALLY_MISSING_ENV_VAR_XYZ_123" {
		t.Errorf("Missing.Env = %v; want [TOTALLY_MISSING_ENV_VAR_XYZ_123]", entry.Missing.Env)
	}
}

func TestBuildSkillStatus_MissingEnvWithRealEnv(t *testing.T) {
	key := "BUILD_SKILL_STATUS_TEST_ENV_SET"
	t.Setenv(key, "some-value")

	s := &Skill{
		Name:   "has-env",
		Source: "bundled",
		Dir:    "/skills/has-env",
		Requires: Requires{
			Config: []string{key},
		},
	}
	entry := BuildSkillStatus(s, nil, map[string]any{})

	if !entry.Eligible {
		t.Error("Eligible = false; want true when env var is set")
	}
	if len(entry.Missing.Env) != 0 {
		t.Errorf("Missing.Env = %v; want empty", entry.Missing.Env)
	}
}

func TestBuildSkillStatus_ExistingBinary(t *testing.T) {
	s := &Skill{
		Name:   "has-bin",
		Source: "bundled",
		Dir:    "/skills/has-bin",
		Requires: Requires{
			Binaries: []string{"ls"},
		},
	}
	entry := BuildSkillStatus(s, nil, map[string]any{})

	if !entry.Eligible {
		t.Error("Eligible = false; want true when binary exists")
	}
	if len(entry.Missing.Bins) != 0 {
		t.Errorf("Missing.Bins = %v; want empty", entry.Missing.Bins)
	}
}

// assertNonNilSlice fails if the given slice is nil.
func assertNonNilSlice(t *testing.T, name string, s []string) {
	t.Helper()
	if s == nil {
		t.Errorf("%s is nil; want non-nil empty slice", name)
	}
}

// ---------------------------------------------------------------------------
// isBundledSkillBlocked (bundled allowlist)
// ---------------------------------------------------------------------------

func TestIsBundledSkillBlocked_NoBundledSource(t *testing.T) {
	// Workspace and user skills are never blocked.
	for _, src := range []string{"workspace", "user"} {
		s := &Skill{Name: "test", Source: src}
		cfg := map[string]any{
			"skills": map[string]any{
				"allowBundled": []any{"other-skill"},
			},
		}
		if isBundledSkillBlocked(s, cfg) {
			t.Errorf("source=%q: should not be blocked (only bundled skills are blocked)", src)
		}
	}
}

func TestIsBundledSkillBlocked_NoAllowlistAllowed(t *testing.T) {
	// When no allowBundled is configured, all bundled skills are allowed.
	s := &Skill{Name: "my-skill", Source: "bundled"}
	cfg := map[string]any{}
	if isBundledSkillBlocked(s, cfg) {
		t.Error("should not be blocked when no allowBundled configured")
	}
}

func TestIsBundledSkillBlocked_EmptyAllowlistAllowed(t *testing.T) {
	s := &Skill{Name: "my-skill", Source: "bundled"}
	cfg := map[string]any{
		"skills": map[string]any{
			"allowBundled": []any{},
		},
	}
	if isBundledSkillBlocked(s, cfg) {
		t.Error("should not be blocked when allowBundled is empty")
	}
}

func TestIsBundledSkillBlocked_MatchByName(t *testing.T) {
	s := &Skill{Name: "allowed-skill", Source: "bundled"}
	cfg := map[string]any{
		"skills": map[string]any{
			"allowBundled": []any{"allowed-skill", "other"},
		},
	}
	if isBundledSkillBlocked(s, cfg) {
		t.Error("should not be blocked when name is in allowBundled")
	}
}

func TestIsBundledSkillBlocked_MatchBySkillKey(t *testing.T) {
	s := &Skill{Name: "my-skill", SkillKey: "custom-key", Source: "bundled"}
	cfg := map[string]any{
		"skills": map[string]any{
			"allowBundled": []any{"custom-key"},
		},
	}
	if isBundledSkillBlocked(s, cfg) {
		t.Error("should not be blocked when skillKey is in allowBundled")
	}
}

func TestIsBundledSkillBlocked_NotInAllowlist(t *testing.T) {
	s := &Skill{Name: "unlisted-skill", Source: "bundled"}
	cfg := map[string]any{
		"skills": map[string]any{
			"allowBundled": []any{"other-skill"},
		},
	}
	if !isBundledSkillBlocked(s, cfg) {
		t.Error("should be blocked when skill is not in allowBundled list")
	}
}

func TestBuildSkillStatus_BlockedByAllow(t *testing.T) {
	s := &Skill{
		Name:   "blocked-skill",
		Source: "bundled",
		Dir:    "/skills/blocked-skill",
	}
	cfg := map[string]any{
		"skills": map[string]any{
			"allowBundled": []any{"other-skill"},
		},
	}
	entry := BuildSkillStatus(s, cfg, map[string]any{})

	if !entry.BlockedByAllow {
		t.Error("BlockedByAllow = false; want true when skill not in allowBundled")
	}
	if entry.Eligible {
		t.Error("Eligible = true; want false when blocked by allowBundled")
	}
}

// Verify the entry type matches the expected types package struct.
func TestBuildSkillStatus_ReturnType(t *testing.T) {
	s := &Skill{Name: "type-check", Dir: "/tmp"}
	entry := BuildSkillStatus(s, nil, map[string]any{})
	// This is a compile-time check: entry must be types.SkillEntry.
	var _ types.SkillEntry = entry
	if entry.Key != "type-check" {
		t.Errorf("Key = %q; want %q", entry.Key, "type-check")
	}
}
