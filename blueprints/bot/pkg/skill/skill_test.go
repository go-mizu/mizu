package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ParseSkillMD
// ---------------------------------------------------------------------------

func TestParseSkillMD_FullFrontmatter(t *testing.T) {
	input := `---
name: weather
description: Get current weather and forecasts
emoji: "\U0001F324"
requires:
  binaries:
    - curl
  config:
    - WEATHER_API_KEY
---
# Weather Skill
Instructions...
`
	sk, body, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sk.Name != "weather" {
		t.Errorf("Name = %q; want %q", sk.Name, "weather")
	}
	if sk.Description != "Get current weather and forecasts" {
		t.Errorf("Description = %q; want %q", sk.Description, "Get current weather and forecasts")
	}
	if sk.Emoji != `"\U0001F324"` {
		t.Errorf("Emoji = %q; want %q", sk.Emoji, `"\U0001F324"`)
	}
	if len(sk.Requires.Binaries) != 1 || sk.Requires.Binaries[0] != "curl" {
		t.Errorf("Requires.Binaries = %v; want [curl]", sk.Requires.Binaries)
	}
	if len(sk.Requires.Config) != 1 || sk.Requires.Config[0] != "WEATHER_API_KEY" {
		t.Errorf("Requires.Config = %v; want [WEATHER_API_KEY]", sk.Requires.Config)
	}

	wantBody := "# Weather Skill\nInstructions...\n"
	if body != wantBody {
		t.Errorf("body = %q; want %q", body, wantBody)
	}
}

func TestParseSkillMD_ContentAfterDelimiter(t *testing.T) {
	input := "---\nname: test\n---\nThis is the body.\nSecond line."
	sk, body, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sk.Name != "test" {
		t.Errorf("Name = %q; want %q", sk.Name, "test")
	}
	if body != "This is the body.\nSecond line." {
		t.Errorf("body = %q; want %q", body, "This is the body.\nSecond line.")
	}
}

func TestParseSkillMD_NoFrontmatter(t *testing.T) {
	input := "Just plain content\nwith no frontmatter."
	sk, body, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sk.Name != "" {
		t.Errorf("Name = %q; want empty", sk.Name)
	}
	if sk.Description != "" {
		t.Errorf("Description = %q; want empty", sk.Description)
	}
	if sk.Emoji != "" {
		t.Errorf("Emoji = %q; want empty", sk.Emoji)
	}
	if body != input {
		t.Errorf("body = %q; want %q", body, input)
	}
}

func TestParseSkillMD_MissingClosingDelimiter(t *testing.T) {
	input := "---\nname: broken\nNo closing delimiter here."
	sk, body, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Without closing ---, the entire content is treated as body.
	if body != input {
		t.Errorf("body = %q; want %q (entire content)", body, input)
	}
	// Skill fields should remain empty since frontmatter was not parsed.
	if sk.Name != "" {
		t.Errorf("Name = %q; want empty (no closing delimiter)", sk.Name)
	}
}

func TestParseSkillMD_EmptyFrontmatter(t *testing.T) {
	input := "---\n---\nBody after empty frontmatter."
	sk, body, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sk.Name != "" {
		t.Errorf("Name = %q; want empty", sk.Name)
	}
	if sk.Description != "" {
		t.Errorf("Description = %q; want empty", sk.Description)
	}
	if body != "Body after empty frontmatter." {
		t.Errorf("body = %q; want %q", body, "Body after empty frontmatter.")
	}
}

func TestParseSkillMD_EmptyContent(t *testing.T) {
	input := ""
	sk, body, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sk.Name != "" || sk.Description != "" || sk.Emoji != "" {
		t.Errorf("expected empty skill fields, got Name=%q Desc=%q Emoji=%q", sk.Name, sk.Description, sk.Emoji)
	}
	if body != "" {
		t.Errorf("body = %q; want empty", body)
	}
}

func TestParseSkillMD_MultipleBinariesAndConfig(t *testing.T) {
	input := `---
name: deploy
requires:
  binaries:
    - docker
    - kubectl
    - helm
  config:
    - KUBECONFIG
    - DOCKER_REGISTRY
---
Deploy instructions.
`
	sk, body, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sk.Name != "deploy" {
		t.Errorf("Name = %q; want %q", sk.Name, "deploy")
	}
	wantBins := []string{"docker", "kubectl", "helm"}
	if len(sk.Requires.Binaries) != len(wantBins) {
		t.Fatalf("Requires.Binaries length = %d; want %d", len(sk.Requires.Binaries), len(wantBins))
	}
	for i, b := range wantBins {
		if sk.Requires.Binaries[i] != b {
			t.Errorf("Requires.Binaries[%d] = %q; want %q", i, sk.Requires.Binaries[i], b)
		}
	}
	wantCfg := []string{"KUBECONFIG", "DOCKER_REGISTRY"}
	if len(sk.Requires.Config) != len(wantCfg) {
		t.Fatalf("Requires.Config length = %d; want %d", len(sk.Requires.Config), len(wantCfg))
	}
	for i, c := range wantCfg {
		if sk.Requires.Config[i] != c {
			t.Errorf("Requires.Config[%d] = %q; want %q", i, sk.Requires.Config[i], c)
		}
	}
	if !strings.HasPrefix(body, "Deploy instructions.") {
		t.Errorf("body = %q; want prefix %q", body, "Deploy instructions.")
	}
}

func TestParseSkillMD_OnlyNameField(t *testing.T) {
	input := "---\nname: minimal\n---\nMinimal body."
	sk, body, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sk.Name != "minimal" {
		t.Errorf("Name = %q; want %q", sk.Name, "minimal")
	}
	if sk.Description != "" {
		t.Errorf("Description = %q; want empty", sk.Description)
	}
	if sk.Emoji != "" {
		t.Errorf("Emoji = %q; want empty", sk.Emoji)
	}
	if body != "Minimal body." {
		t.Errorf("body = %q; want %q", body, "Minimal body.")
	}
}

func TestParseSkillMD_EmptyConfigList(t *testing.T) {
	// config: [] style should result in empty Config slice.
	input := "---\nname: noconfig\nrequires:\n  binaries:\n    - git\n  config:\n---\nBody."
	sk, _, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sk.Requires.Binaries) != 1 || sk.Requires.Binaries[0] != "git" {
		t.Errorf("Requires.Binaries = %v; want [git]", sk.Requires.Binaries)
	}
	if len(sk.Requires.Config) != 0 {
		t.Errorf("Requires.Config = %v; want empty", sk.Requires.Config)
	}
}

// ---------------------------------------------------------------------------
// LoadSkill
// ---------------------------------------------------------------------------

func TestLoadSkill_FullSkillMD(t *testing.T) {
	dir := t.TempDir()
	content := `---
name: greeter
description: Greets people
emoji: "\U0001F44B"
---
Hello, welcome!
`
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	sk, err := LoadSkill(dir)
	if err != nil {
		t.Fatalf("LoadSkill error: %v", err)
	}

	if sk.Name != "greeter" {
		t.Errorf("Name = %q; want %q", sk.Name, "greeter")
	}
	if sk.Description != "Greets people" {
		t.Errorf("Description = %q; want %q", sk.Description, "Greets people")
	}
	if sk.Emoji != `"\U0001F44B"` {
		t.Errorf("Emoji = %q; want %q", sk.Emoji, `"\U0001F44B"`)
	}
	if sk.Dir != dir {
		t.Errorf("Dir = %q; want %q", sk.Dir, dir)
	}
	wantBody := "Hello, welcome!\n"
	if sk.Content != wantBody {
		t.Errorf("Content = %q; want %q", sk.Content, wantBody)
	}
}

func TestLoadSkill_NameDefaultsToDirName(t *testing.T) {
	dir := t.TempDir()
	// Create a subdirectory with a specific name.
	subdir := filepath.Join(dir, "my-cool-skill")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// SKILL.md without a name field.
	content := "---\ndescription: A cool skill\n---\nSome instructions."
	if err := os.WriteFile(filepath.Join(subdir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	sk, err := LoadSkill(subdir)
	if err != nil {
		t.Fatalf("LoadSkill error: %v", err)
	}

	if sk.Name != "my-cool-skill" {
		t.Errorf("Name = %q; want %q (directory basename)", sk.Name, "my-cool-skill")
	}
	if sk.Description != "A cool skill" {
		t.Errorf("Description = %q; want %q", sk.Description, "A cool skill")
	}
}

func TestLoadSkill_ReadySetsFromEligibility(t *testing.T) {
	dir := t.TempDir()
	// No requirements: should be ready.
	content := "---\nname: simple\n---\nBody."
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	sk, err := LoadSkill(dir)
	if err != nil {
		t.Fatalf("LoadSkill error: %v", err)
	}

	if !sk.Ready {
		t.Errorf("Ready = false; want true (no requirements)")
	}
}

func TestLoadSkill_ReadyFalseWhenBinaryMissing(t *testing.T) {
	dir := t.TempDir()
	content := "---\nname: broken\nrequires:\n  binaries:\n    - nonexistent_binary_xyz_abc_123\n---\nBody."
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	sk, err := LoadSkill(dir)
	if err != nil {
		t.Fatalf("LoadSkill error: %v", err)
	}

	if sk.Ready {
		t.Errorf("Ready = true; want false (binary missing)")
	}
}

func TestLoadSkill_MissingSKILLMD(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadSkill(dir)
	if err == nil {
		t.Fatal("expected error for missing SKILL.md, got nil")
	}
}

// ---------------------------------------------------------------------------
// CheckEligibility
// ---------------------------------------------------------------------------

func TestCheckEligibility_NoRequirements(t *testing.T) {
	sk := &Skill{}
	if !CheckEligibility(sk) {
		t.Error("expected true for no requirements")
	}
}

func TestCheckEligibility_ExistingBinary(t *testing.T) {
	sk := &Skill{
		Requires: Requires{Binaries: []string{"ls"}},
	}
	if !CheckEligibility(sk) {
		t.Error("expected true for existing binary 'ls'")
	}
}

func TestCheckEligibility_MultiplExistingBinaries(t *testing.T) {
	sk := &Skill{
		Requires: Requires{Binaries: []string{"ls", "echo"}},
	}
	if !CheckEligibility(sk) {
		t.Error("expected true for existing binaries 'ls' and 'echo'")
	}
}

func TestCheckEligibility_MissingBinary(t *testing.T) {
	sk := &Skill{
		Requires: Requires{Binaries: []string{"nonexistent_binary_xyz"}},
	}
	if CheckEligibility(sk) {
		t.Error("expected false for nonexistent binary")
	}
}

func TestCheckEligibility_OneGoodOneBadBinary(t *testing.T) {
	sk := &Skill{
		Requires: Requires{Binaries: []string{"ls", "nonexistent_binary_xyz"}},
	}
	if CheckEligibility(sk) {
		t.Error("expected false when one binary is missing")
	}
}

func TestCheckEligibility_ConfigEnvVarSet(t *testing.T) {
	key := "SKILL_TEST_CFG_EXISTS_12345"
	t.Setenv(key, "some_value")

	sk := &Skill{
		Requires: Requires{Config: []string{key}},
	}
	if !CheckEligibility(sk) {
		t.Errorf("expected true when env var %s is set", key)
	}
}

func TestCheckEligibility_ConfigEnvVarNotSet(t *testing.T) {
	key := "SKILL_TEST_CFG_MISSING_99999"
	os.Unsetenv(key) // ensure it's unset

	sk := &Skill{
		Requires: Requires{Config: []string{key}},
	}
	if CheckEligibility(sk) {
		t.Errorf("expected false when env var %s is not set", key)
	}
}

func TestCheckEligibility_ConfigEnvVarEmpty(t *testing.T) {
	key := "SKILL_TEST_CFG_EMPTY_88888"
	t.Setenv(key, "")

	sk := &Skill{
		Requires: Requires{Config: []string{key}},
	}
	if CheckEligibility(sk) {
		t.Errorf("expected false when env var %s is empty string", key)
	}
}

func TestCheckEligibility_MixBinariesAndConfig_AllPass(t *testing.T) {
	key := "SKILL_TEST_MIX_OK_77777"
	t.Setenv(key, "value")

	sk := &Skill{
		Requires: Requires{
			Binaries: []string{"ls"},
			Config:   []string{key},
		},
	}
	if !CheckEligibility(sk) {
		t.Error("expected true when binary exists and config is set")
	}
}

func TestCheckEligibility_MixBinariesAndConfig_BinaryFails(t *testing.T) {
	key := "SKILL_TEST_MIX_BINFAIL_66666"
	t.Setenv(key, "value")

	sk := &Skill{
		Requires: Requires{
			Binaries: []string{"nonexistent_binary_xyz"},
			Config:   []string{key},
		},
	}
	if CheckEligibility(sk) {
		t.Error("expected false when binary is missing even though config is set")
	}
}

func TestCheckEligibility_MixBinariesAndConfig_ConfigFails(t *testing.T) {
	cfgKey := "SKILL_TEST_MIX_CFGFAIL_55555"
	os.Unsetenv(cfgKey)

	sk := &Skill{
		Requires: Requires{
			Binaries: []string{"ls"},
			Config:   []string{cfgKey},
		},
	}
	if CheckEligibility(sk) {
		t.Error("expected false when config var is missing even though binary exists")
	}
}

// ---------------------------------------------------------------------------
// LoadWorkspaceSkills
// ---------------------------------------------------------------------------

func TestLoadWorkspaceSkills_LoadsSubdirectories(t *testing.T) {
	ws := t.TempDir()
	skillsDir := filepath.Join(ws, "skills")

	// Create two valid skill subdirectories.
	for _, name := range []string{"alpha", "beta"} {
		dir := filepath.Join(skillsDir, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		md := "---\nname: " + name + "\ndescription: Skill " + name + "\n---\nBody for " + name + "."
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(md), 0644); err != nil {
			t.Fatalf("write SKILL.md: %v", err)
		}
	}

	skills, err := LoadWorkspaceSkills(ws)
	if err != nil {
		t.Fatalf("LoadWorkspaceSkills error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("got %d skills; want 2", len(skills))
	}

	names := map[string]bool{}
	for _, s := range skills {
		names[s.Name] = true
		if s.Source != "workspace" {
			t.Errorf("skill %q Source = %q; want %q", s.Name, s.Source, "workspace")
		}
	}
	if !names["alpha"] || !names["beta"] {
		t.Errorf("expected skills alpha and beta; got %v", names)
	}
}

func TestLoadWorkspaceSkills_SkipsDirsWithoutSKILLMD(t *testing.T) {
	ws := t.TempDir()
	skillsDir := filepath.Join(ws, "skills")

	// One valid skill.
	validDir := filepath.Join(skillsDir, "valid")
	if err := os.MkdirAll(validDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(validDir, "SKILL.md"), []byte("---\nname: valid\n---\nBody."), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// One directory without SKILL.md.
	noSkillDir := filepath.Join(skillsDir, "noskill")
	if err := os.MkdirAll(noSkillDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(noSkillDir, "README.md"), []byte("not a skill"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// One regular file (not a directory) in skills/.
	if err := os.WriteFile(filepath.Join(skillsDir, "stray-file.txt"), []byte("ignore me"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	skills, err := LoadWorkspaceSkills(ws)
	if err != nil {
		t.Fatalf("LoadWorkspaceSkills error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("got %d skills; want 1", len(skills))
	}
	if skills[0].Name != "valid" {
		t.Errorf("Name = %q; want %q", skills[0].Name, "valid")
	}
}

func TestLoadWorkspaceSkills_MissingSkillsDirReturnsNil(t *testing.T) {
	ws := t.TempDir()
	// Do not create a skills/ subdirectory.

	skills, err := LoadWorkspaceSkills(ws)
	if err != nil {
		t.Fatalf("expected nil error for missing skills dir, got: %v", err)
	}
	if skills != nil {
		t.Errorf("expected nil skills, got %v", skills)
	}
}

// ---------------------------------------------------------------------------
// LoadAllSkills
// ---------------------------------------------------------------------------

func TestLoadAllSkills_WorkspacePrecedenceOverBundled(t *testing.T) {
	ws := t.TempDir()
	bundledDir := t.TempDir()

	// Create a workspace skill named "deploy".
	wsSkillDir := filepath.Join(ws, "skills", "deploy")
	if err := os.MkdirAll(wsSkillDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsSkillDir, "SKILL.md"),
		[]byte("---\nname: deploy\ndescription: workspace deploy\n---\nWS body."), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Create a bundled skill with the same name "deploy" (should be overridden).
	bundledDeployDir := filepath.Join(bundledDir, "deploy")
	if err := os.MkdirAll(bundledDeployDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundledDeployDir, "SKILL.md"),
		[]byte("---\nname: deploy\ndescription: bundled deploy\n---\nBundled body."), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Create a bundled-only skill named "lint".
	bundledLintDir := filepath.Join(bundledDir, "lint")
	if err := os.MkdirAll(bundledLintDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundledLintDir, "SKILL.md"),
		[]byte("---\nname: lint\ndescription: bundled lint\n---\nLint body."), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	all, err := LoadAllSkills(ws, bundledDir)
	if err != nil {
		t.Fatalf("LoadAllSkills error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d skills; want 2", len(all))
	}

	byName := map[string]*Skill{}
	for _, s := range all {
		byName[s.Name] = s
	}

	// "deploy" should come from workspace.
	deploy, ok := byName["deploy"]
	if !ok {
		t.Fatal("missing deploy skill")
	}
	if deploy.Source != "workspace" {
		t.Errorf("deploy Source = %q; want %q", deploy.Source, "workspace")
	}
	if deploy.Description != "workspace deploy" {
		t.Errorf("deploy Description = %q; want %q", deploy.Description, "workspace deploy")
	}

	// "lint" should come from bundled.
	lint, ok := byName["lint"]
	if !ok {
		t.Fatal("missing lint skill")
	}
	if lint.Source != "bundled" {
		t.Errorf("lint Source = %q; want %q", lint.Source, "bundled")
	}
	if lint.Description != "bundled lint" {
		t.Errorf("lint Description = %q; want %q", lint.Description, "bundled lint")
	}
}

func TestLoadAllSkills_MultipleBundledDirs(t *testing.T) {
	ws := t.TempDir()
	bundledDir1 := t.TempDir()
	bundledDir2 := t.TempDir()

	// bundled dir 1 has "alpha".
	alphaDir := filepath.Join(bundledDir1, "alpha")
	if err := os.MkdirAll(alphaDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(alphaDir, "SKILL.md"),
		[]byte("---\nname: alpha\n---\nAlpha body."), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// bundled dir 2 has "beta" and a duplicate "alpha" (should be skipped).
	betaDir := filepath.Join(bundledDir2, "beta")
	if err := os.MkdirAll(betaDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(betaDir, "SKILL.md"),
		[]byte("---\nname: beta\n---\nBeta body."), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	alphaDir2 := filepath.Join(bundledDir2, "alpha")
	if err := os.MkdirAll(alphaDir2, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(alphaDir2, "SKILL.md"),
		[]byte("---\nname: alpha\ndescription: duplicate\n---\nDup body."), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	all, err := LoadAllSkills(ws, bundledDir1, bundledDir2)
	if err != nil {
		t.Fatalf("LoadAllSkills error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d skills; want 2 (alpha from dir1, beta from dir2)", len(all))
	}

	byName := map[string]*Skill{}
	for _, s := range all {
		byName[s.Name] = s
	}

	alpha, ok := byName["alpha"]
	if !ok {
		t.Fatal("missing alpha skill")
	}
	// alpha should be from dir1 (first bundled dir), not the duplicate from dir2.
	if alpha.Description == "duplicate" {
		t.Error("alpha was loaded from second bundled dir; should have been from first")
	}
	if alpha.Source != "bundled" {
		t.Errorf("alpha Source = %q; want %q", alpha.Source, "bundled")
	}

	if _, ok := byName["beta"]; !ok {
		t.Fatal("missing beta skill")
	}
}

func TestLoadAllSkills_NoWorkspaceNoBundle(t *testing.T) {
	ws := t.TempDir()
	all, err := LoadAllSkills(ws)
	if err != nil {
		t.Fatalf("LoadAllSkills error: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("got %d skills; want 0", len(all))
	}
}

// createSkillDir creates a skill subdirectory with a SKILL.md file.
func createSkillDir(t *testing.T, parentDir, name, content string) {
	t.Helper()
	dir := filepath.Join(parentDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ParseExtraDirs
// ---------------------------------------------------------------------------

func TestParseExtraDirs_Empty(t *testing.T) {
	dirs := ParseExtraDirs(map[string]any{})
	if len(dirs) != 0 {
		t.Errorf("got %d dirs; want 0", len(dirs))
	}
}

func TestParseExtraDirs_NoLoadSection(t *testing.T) {
	cfg := map[string]any{
		"skills": map[string]any{},
	}
	dirs := ParseExtraDirs(cfg)
	if len(dirs) != 0 {
		t.Errorf("got %d dirs; want 0", len(dirs))
	}
}

func TestParseExtraDirs_ValidDirs(t *testing.T) {
	cfg := map[string]any{
		"skills": map[string]any{
			"load": map[string]any{
				"extraDirs": []any{"/opt/skills", "/usr/local/skills"},
			},
		},
	}
	dirs := ParseExtraDirs(cfg)
	if len(dirs) != 2 {
		t.Fatalf("got %d dirs; want 2", len(dirs))
	}
	if dirs[0] != "/opt/skills" || dirs[1] != "/usr/local/skills" {
		t.Errorf("dirs = %v; want [/opt/skills /usr/local/skills]", dirs)
	}
}

func TestParseExtraDirs_ExpandsTilde(t *testing.T) {
	cfg := map[string]any{
		"skills": map[string]any{
			"load": map[string]any{
				"extraDirs": []any{"~/my-skills"},
			},
		},
	}
	dirs := ParseExtraDirs(cfg)
	if len(dirs) != 1 {
		t.Fatalf("got %d dirs; want 1", len(dirs))
	}
	if strings.HasPrefix(dirs[0], "~") {
		t.Errorf("dirs[0] = %q; should have expanded ~", dirs[0])
	}
}

func TestParseExtraDirs_SkipsEmptyStrings(t *testing.T) {
	cfg := map[string]any{
		"skills": map[string]any{
			"load": map[string]any{
				"extraDirs": []any{"", "/valid/dir", ""},
			},
		},
	}
	dirs := ParseExtraDirs(cfg)
	if len(dirs) != 1 {
		t.Fatalf("got %d dirs; want 1 (empty strings skipped)", len(dirs))
	}
	if dirs[0] != "/valid/dir" {
		t.Errorf("dirs[0] = %q; want /valid/dir", dirs[0])
	}
}

// ---------------------------------------------------------------------------
// LoadAllSkillsWithExtras
// ---------------------------------------------------------------------------

func TestLoadAllSkillsWithExtras_ExtraDirsLoaded(t *testing.T) {
	ws := t.TempDir()
	extraDir := t.TempDir()
	bundledDir := t.TempDir()

	// Create a skill in extra dir.
	createSkillDir(t, extraDir, "extra-skill", "---\nname: extra-skill\ndescription: From extra dir\n---\nExtra skill")

	// Create a different skill in bundled dir.
	createSkillDir(t, bundledDir, "bundled-skill", "---\nname: bundled-skill\ndescription: From bundled\n---\nBundled skill")

	all, err := LoadAllSkillsWithExtras(ws, []string{extraDir}, bundledDir)
	if err != nil {
		t.Fatalf("LoadAllSkillsWithExtras error: %v", err)
	}

	if len(all) != 2 {
		t.Fatalf("got %d skills; want 2", len(all))
	}

	// Find the extra skill.
	var extraSkill *Skill
	for _, s := range all {
		if s.Name == "extra-skill" {
			extraSkill = s
		}
	}
	if extraSkill == nil {
		t.Fatal("extra-skill not found")
	}
	if extraSkill.Source != "extra" {
		t.Errorf("extra-skill.Source = %q; want %q", extraSkill.Source, "extra")
	}
}

func TestLoadAllSkillsWithExtras_ExtraPrecedenceOverBundled(t *testing.T) {
	ws := t.TempDir()
	extraDir := t.TempDir()
	bundledDir := t.TempDir()

	// Same skill name in both extra and bundled.
	createSkillDir(t, extraDir, "conflict", "---\nname: conflict\ndescription: Extra version\n---\nExtra version")
	createSkillDir(t, bundledDir, "conflict", "---\nname: conflict\ndescription: Bundled version\n---\nBundled version")

	all, err := LoadAllSkillsWithExtras(ws, []string{extraDir}, bundledDir)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(all) != 1 {
		t.Fatalf("got %d skills; want 1 (extra wins over bundled)", len(all))
	}
	if all[0].Source != "extra" {
		t.Errorf("Source = %q; want %q (extra has precedence over bundled)", all[0].Source, "extra")
	}
	if all[0].Description != "Extra version" {
		t.Errorf("Description = %q; want %q", all[0].Description, "Extra version")
	}
}

func TestLoadAllSkillsWithExtras_WorkspacePrecedenceOverExtra(t *testing.T) {
	ws := t.TempDir()
	extraDir := t.TempDir()

	// Same skill in workspace and extra.
	wsSkillsDir := filepath.Join(ws, "skills")
	os.MkdirAll(wsSkillsDir, 0o755)
	createSkillDir(t, wsSkillsDir, "my-skill", "---\nname: my-skill\ndescription: Workspace version\n---\nWS")
	createSkillDir(t, extraDir, "my-skill", "---\nname: my-skill\ndescription: Extra version\n---\nExtra")

	all, err := LoadAllSkillsWithExtras(ws, []string{extraDir})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(all) != 1 {
		t.Fatalf("got %d skills; want 1", len(all))
	}
	if all[0].Source != "workspace" {
		t.Errorf("Source = %q; want %q (workspace wins over extra)", all[0].Source, "workspace")
	}
}

func TestLoadAllSkillsWithExtras_NilExtraDirs(t *testing.T) {
	ws := t.TempDir()
	bundledDir := t.TempDir()
	createSkillDir(t, bundledDir, "test-skill", "---\nname: test-skill\ndescription: Test\n---\nTest")

	all, err := LoadAllSkillsWithExtras(ws, nil, bundledDir)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("got %d skills; want 1", len(all))
	}
}

// ---------------------------------------------------------------------------
// BuildSkillsPrompt
// ---------------------------------------------------------------------------

func TestBuildSkillsPrompt_ReadySkillsWithEmoji(t *testing.T) {
	skills := []*Skill{
		{Name: "weather", Description: "Get weather", Emoji: "E", Ready: true},
		{Name: "deploy", Description: "Deploy app", Emoji: "D", Ready: true},
	}
	got := BuildSkillsPrompt(skills)

	if !strings.Contains(got, "<available_skills>") {
		t.Errorf("prompt should contain <available_skills> tag; got:\n%s", got)
	}
	if !strings.Contains(got, "<name>weather</name>") {
		t.Errorf("missing weather skill in:\n%s", got)
	}
	if !strings.Contains(got, "<name>deploy</name>") {
		t.Errorf("missing deploy skill in:\n%s", got)
	}
}

func TestBuildSkillsPrompt_NoEmoji(t *testing.T) {
	skills := []*Skill{
		{Name: "plain", Description: "No emoji skill", Ready: true},
	}
	got := BuildSkillsPrompt(skills)

	if !strings.Contains(got, "<name>plain</name>") {
		t.Errorf("unexpected format for skill without emoji:\n%s", got)
	}
	if !strings.Contains(got, "<description>No emoji skill</description>") {
		t.Errorf("missing description:\n%s", got)
	}
}

func TestBuildSkillsPrompt_NoDescription(t *testing.T) {
	skills := []*Skill{
		{Name: "nodesc", Ready: true},
	}
	got := BuildSkillsPrompt(skills)

	if !strings.Contains(got, "<name>nodesc</name>") {
		t.Errorf("missing skill name:\n%s", got)
	}
	// Should NOT contain description tag when there is no description.
	if strings.Contains(got, "<description></description>") {
		t.Errorf("should not have empty description tag:\n%s", got)
	}
}

func TestBuildSkillsPrompt_SkipsNotReadySkills(t *testing.T) {
	skills := []*Skill{
		{Name: "ready", Description: "I am ready", Emoji: "R", Ready: true},
		{Name: "notready", Description: "I am not ready", Emoji: "N", Ready: false},
	}
	got := BuildSkillsPrompt(skills)

	if !strings.Contains(got, "<name>ready</name>") {
		t.Errorf("ready skill should be included in:\n%s", got)
	}
	if strings.Contains(got, "<name>notready</name>") {
		t.Errorf("not-ready skill should be excluded from:\n%s", got)
	}
}

func TestBuildSkillsPrompt_NoReadySkills(t *testing.T) {
	skills := []*Skill{
		{Name: "a", Ready: false},
		{Name: "b", Ready: false},
	}
	got := BuildSkillsPrompt(skills)

	if !strings.Contains(got, "<!-- No skills available -->") {
		t.Errorf("should contain no-skills comment; got:\n%s", got)
	}
}

func TestBuildSkillsPrompt_NilSlice(t *testing.T) {
	got := BuildSkillsPrompt(nil)

	if !strings.Contains(got, "<!-- No skills available -->") {
		t.Errorf("should contain no-skills comment; got:\n%s", got)
	}
}

func TestBuildSkillsPrompt_EmptySlice(t *testing.T) {
	got := BuildSkillsPrompt([]*Skill{})

	if !strings.Contains(got, "<!-- No skills available -->") {
		t.Errorf("should contain no-skills comment; got:\n%s", got)
	}
}

func TestBuildSkillsPrompt_MixedReadyAndNotReady(t *testing.T) {
	skills := []*Skill{
		{Name: "first", Description: "Desc 1", Emoji: "1", Ready: true},
		{Name: "second", Ready: false},
		{Name: "third", Description: "Desc 3", Ready: true},
		{Name: "fourth", Ready: false},
	}
	got := BuildSkillsPrompt(skills)

	if !strings.Contains(got, "<name>first</name>") {
		t.Errorf("missing first in:\n%s", got)
	}
	if !strings.Contains(got, "<name>third</name>") {
		t.Errorf("missing third in:\n%s", got)
	}
	if strings.Contains(got, "<name>second</name>") {
		t.Errorf("second (not ready) should not appear in:\n%s", got)
	}
	if strings.Contains(got, "<name>fourth</name>") {
		t.Errorf("fourth (not ready) should not appear in:\n%s", got)
	}
	if strings.Contains(got, "No skills available") {
		t.Errorf("should not say 'No skills available' when some are ready:\n%s", got)
	}
}

// ---------------------------------------------------------------------------
// OpenClaw Metadata JSON Parsing
// ---------------------------------------------------------------------------

func TestParseSkillMD_MetadataJSON_Weather(t *testing.T) {
	// Real OpenClaw weather SKILL.md frontmatter.
	input := `---
name: weather
description: Get current weather and forecasts (no API key required).
homepage: https://wttr.in/:help
metadata: {"openclaw":{"emoji":"🌤️","requires":{"bins":["curl"]}}}
---
# Weather

Two free services, no API keys needed.
`
	sk, body, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sk.Name != "weather" {
		t.Errorf("Name = %q; want %q", sk.Name, "weather")
	}
	if sk.Description != "Get current weather and forecasts (no API key required)." {
		t.Errorf("Description = %q", sk.Description)
	}
	if sk.Homepage != "https://wttr.in/:help" {
		t.Errorf("Homepage = %q; want %q", sk.Homepage, "https://wttr.in/:help")
	}
	if sk.Emoji != "🌤️" {
		t.Errorf("Emoji = %q; want %q", sk.Emoji, "🌤️")
	}
	if len(sk.Requires.Binaries) != 1 || sk.Requires.Binaries[0] != "curl" {
		t.Errorf("Requires.Binaries = %v; want [curl]", sk.Requires.Binaries)
	}
	if !strings.HasPrefix(body, "# Weather") {
		t.Errorf("body should start with '# Weather'; got %q", body[:30])
	}
}

func TestParseSkillMD_MetadataJSON_GitHub(t *testing.T) {
	// Real OpenClaw github SKILL.md frontmatter.
	input := `---
name: github
description: "Interact with GitHub using the ` + "`gh`" + ` CLI. Use ` + "`gh issue`" + `, ` + "`gh pr`" + `, ` + "`gh run`" + `, and ` + "`gh api`" + ` for issues, PRs, CI runs, and advanced queries."
metadata: {"openclaw":{"emoji":"🐙","requires":{"bins":["gh"]},"install":[{"id":"brew","kind":"brew","formula":"gh","bins":["gh"],"label":"Install GitHub CLI (brew)"}]}}
---
# GitHub Skill
`
	sk, _, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sk.Name != "github" {
		t.Errorf("Name = %q; want %q", sk.Name, "github")
	}
	if sk.Emoji != "🐙" {
		t.Errorf("Emoji = %q; want %q", sk.Emoji, "🐙")
	}
	if len(sk.Requires.Binaries) != 1 || sk.Requires.Binaries[0] != "gh" {
		t.Errorf("Requires.Binaries = %v; want [gh]", sk.Requires.Binaries)
	}
	// Description should have quotes stripped.
	if !strings.HasPrefix(sk.Description, "Interact with GitHub") {
		t.Errorf("Description = %q; should start with 'Interact with GitHub'", sk.Description)
	}
}

func TestParseSkillMD_MetadataJSON_CodingAgent_AnyBins(t *testing.T) {
	// Real OpenClaw coding-agent SKILL.md uses anyBins.
	input := `---
name: coding-agent
description: Run coding agents via background process.
metadata: {"openclaw":{"emoji":"🧩","requires":{"anyBins":["claude","codex","opencode","pi"]}}}
---
# Coding Agent
`
	sk, _, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sk.Name != "coding-agent" {
		t.Errorf("Name = %q; want %q", sk.Name, "coding-agent")
	}
	if sk.Emoji != "🧩" {
		t.Errorf("Emoji = %q; want %q", sk.Emoji, "🧩")
	}
	wantAnyBins := []string{"claude", "codex", "opencode", "pi"}
	if len(sk.Requires.AnyBins) != len(wantAnyBins) {
		t.Fatalf("Requires.AnyBins length = %d; want %d", len(sk.Requires.AnyBins), len(wantAnyBins))
	}
	for i, b := range wantAnyBins {
		if sk.Requires.AnyBins[i] != b {
			t.Errorf("Requires.AnyBins[%d] = %q; want %q", i, sk.Requires.AnyBins[i], b)
		}
	}
}

func TestParseSkillMD_MetadataJSON_Slack_ConfigPaths(t *testing.T) {
	// Real OpenClaw slack SKILL.md uses config path requirements.
	input := `---
name: slack
description: Slack integration.
metadata: {"openclaw":{"emoji":"💬","requires":{"config":["channels.slack"]}}}
---
# Slack
`
	sk, _, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sk.Name != "slack" {
		t.Errorf("Name = %q; want %q", sk.Name, "slack")
	}
	if sk.Emoji != "💬" {
		t.Errorf("Emoji = %q; want %q", sk.Emoji, "💬")
	}
	if len(sk.Requires.CfgPaths) != 1 || sk.Requires.CfgPaths[0] != "channels.slack" {
		t.Errorf("Requires.CfgPaths = %v; want [channels.slack]", sk.Requires.CfgPaths)
	}
}

func TestParseSkillMD_MetadataJSON_EnvVar(t *testing.T) {
	// Test skill with env requirement and primaryEnv.
	input := `---
name: test-api
description: Test API skill
metadata: {"openclaw":{"emoji":"♊","primaryEnv":"TEST_API_KEY","requires":{"env":["TEST_API_KEY"]}}}
---
# Test API
`
	sk, _, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sk.PrimaryEnv != "TEST_API_KEY" {
		t.Errorf("PrimaryEnv = %q; want %q", sk.PrimaryEnv, "TEST_API_KEY")
	}
	if len(sk.Requires.Config) != 1 || sk.Requires.Config[0] != "TEST_API_KEY" {
		t.Errorf("Requires.Config = %v; want [TEST_API_KEY]", sk.Requires.Config)
	}
}

func TestParseSkillMD_MetadataJSON_SessionLogs_Always(t *testing.T) {
	input := `---
name: session-logs
description: Session logging.
metadata: {"openclaw":{"always":true}}
---
# Session Logs
`
	sk, _, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !sk.Always {
		t.Error("Always = false; want true")
	}
}

func TestParseSkillMD_MetadataJSON_OS_Darwin(t *testing.T) {
	input := `---
name: apple-notes
description: Apple Notes via AppleScript
metadata: {"openclaw":{"emoji":"📝","os":["darwin"]}}
---
# Apple Notes
`
	sk, _, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sk.Requires.OS) != 1 || sk.Requires.OS[0] != "darwin" {
		t.Errorf("Requires.OS = %v; want [darwin]", sk.Requires.OS)
	}
}

func TestParseSkillMD_MetadataJSON_OverridesSimpleYAML(t *testing.T) {
	// Simple YAML emoji vs metadata emoji -- metadata should win.
	input := `---
name: test
emoji: old-emoji
metadata: {"openclaw":{"emoji":"new-emoji"}}
---
Body.
`
	sk, _, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sk.Emoji != "new-emoji" {
		t.Errorf("Emoji = %q; want %q (metadata should override YAML)", sk.Emoji, "new-emoji")
	}
}

func TestParseSkillMD_UserInvocable_Default(t *testing.T) {
	input := "---\nname: test\n---\nBody."
	sk, _, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !sk.UserInvocable {
		t.Error("UserInvocable should default to true")
	}
	if sk.DisableModelInvocation {
		t.Error("DisableModelInvocation should default to false")
	}
}

func TestParseSkillMD_UserInvocable_False(t *testing.T) {
	input := "---\nname: test\nuser-invocable: false\n---\nBody."
	sk, _, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sk.UserInvocable {
		t.Error("UserInvocable = true; want false")
	}
}

func TestParseSkillMD_DisableModelInvocation_True(t *testing.T) {
	input := "---\nname: test\ndisable-model-invocation: true\n---\nBody."
	sk, _, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !sk.DisableModelInvocation {
		t.Error("DisableModelInvocation = false; want true")
	}
}

func TestParseSkillMD_Homepage(t *testing.T) {
	input := "---\nname: test\nhomepage: https://example.com\n---\nBody."
	sk, _, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sk.Homepage != "https://example.com" {
		t.Errorf("Homepage = %q; want %q", sk.Homepage, "https://example.com")
	}
}

// ---------------------------------------------------------------------------
// CheckEligibility - AnyBins (OR logic)
// ---------------------------------------------------------------------------

func TestCheckEligibility_AnyBins_OneExists(t *testing.T) {
	sk := &Skill{
		Requires: Requires{AnyBins: []string{"ls", "nonexistent_xyz_123"}},
	}
	if !CheckEligibility(sk) {
		t.Error("expected true when at least one anyBin exists")
	}
}

func TestCheckEligibility_AnyBins_NoneExist(t *testing.T) {
	sk := &Skill{
		Requires: Requires{AnyBins: []string{"nonexistent_xyz_1", "nonexistent_xyz_2"}},
	}
	if CheckEligibility(sk) {
		t.Error("expected false when no anyBins exist")
	}
}

func TestCheckEligibility_AnyBins_Empty(t *testing.T) {
	sk := &Skill{
		Requires: Requires{AnyBins: []string{}},
	}
	if !CheckEligibility(sk) {
		t.Error("expected true when anyBins is empty")
	}
}

// ---------------------------------------------------------------------------
// CheckEligibility - Always bypass
// ---------------------------------------------------------------------------

func TestCheckEligibility_AlwaysBypassesBinaries(t *testing.T) {
	sk := &Skill{
		Always:   true,
		Requires: Requires{Binaries: []string{"nonexistent_binary_xyz"}},
	}
	if !CheckEligibility(sk) {
		t.Error("expected true: always=true should bypass binary check")
	}
}

func TestCheckEligibility_AlwaysBypassesConfig(t *testing.T) {
	sk := &Skill{
		Always:   true,
		Requires: Requires{Config: []string{"MISSING_CONFIG_VAR_XYZ"}},
	}
	if !CheckEligibility(sk) {
		t.Error("expected true: always=true should bypass config check")
	}
}

func TestCheckEligibility_AlwaysStillRespectsOS(t *testing.T) {
	sk := &Skill{
		Always:   true,
		Requires: Requires{OS: []string{"impossible_os_xyz"}},
	}
	if CheckEligibility(sk) {
		t.Error("expected false: always=true should still respect OS check")
	}
}

// ---------------------------------------------------------------------------
// BuildSkillsPrompt - DisableModelInvocation
// ---------------------------------------------------------------------------

func TestBuildSkillsPrompt_ExcludesDisableModelInvocation(t *testing.T) {
	skills := []*Skill{
		{Name: "visible", Description: "I am visible", Ready: true},
		{Name: "hidden", Description: "I am hidden", Ready: true, DisableModelInvocation: true},
	}
	got := BuildSkillsPrompt(skills)

	if !strings.Contains(got, "<name>visible</name>") {
		t.Errorf("visible skill should be included in:\n%s", got)
	}
	if strings.Contains(got, "<name>hidden</name>") {
		t.Errorf("hidden (DisableModelInvocation) skill should be excluded from:\n%s", got)
	}
}

// ---------------------------------------------------------------------------
// BundledSkillsDir
// ---------------------------------------------------------------------------

func TestBundledSkillsDir_EnvVar(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENBOT_BUNDLED_SKILLS_DIR", dir)

	got := BundledSkillsDir()
	if got != dir {
		t.Errorf("BundledSkillsDir() = %q; want %q", got, dir)
	}
}

func TestBundledSkillsDir_InvalidEnvVar(t *testing.T) {
	t.Setenv("OPENBOT_BUNDLED_SKILLS_DIR", "/nonexistent/path/xyz123")

	got := BundledSkillsDir()
	// Should fall through to binary sibling check, which likely also doesn't exist.
	// Just verify it doesn't return the invalid path.
	if got == "/nonexistent/path/xyz123" {
		t.Error("should not return invalid path from env var")
	}
}

// ---------------------------------------------------------------------------
// MatchSkillCommand
// ---------------------------------------------------------------------------

func TestMatchSkillCommand_MatchesReadyUserInvocable(t *testing.T) {
	skills := []*Skill{
		{Name: "weather", Ready: true, UserInvocable: true},
		{Name: "deploy", Ready: true, UserInvocable: true},
	}
	s, ok := MatchSkillCommand("/weather", skills)
	if !ok {
		t.Fatal("expected match for /weather")
	}
	if s.Name != "weather" {
		t.Errorf("Name = %q; want %q", s.Name, "weather")
	}
}

func TestMatchSkillCommand_NoMatchNotReady(t *testing.T) {
	skills := []*Skill{
		{Name: "weather", Ready: false, UserInvocable: true},
	}
	_, ok := MatchSkillCommand("/weather", skills)
	if ok {
		t.Error("should not match skill that is not ready")
	}
}

func TestMatchSkillCommand_NoMatchNotUserInvocable(t *testing.T) {
	skills := []*Skill{
		{Name: "internal", Ready: true, UserInvocable: false},
	}
	_, ok := MatchSkillCommand("/internal", skills)
	if ok {
		t.Error("should not match skill that is not user-invocable")
	}
}

func TestMatchSkillCommand_NoMatchUnknown(t *testing.T) {
	skills := []*Skill{
		{Name: "weather", Ready: true, UserInvocable: true},
	}
	_, ok := MatchSkillCommand("/unknown", skills)
	if ok {
		t.Error("should not match unknown command")
	}
}

func TestMatchSkillCommand_EmptyCommand(t *testing.T) {
	skills := []*Skill{
		{Name: "weather", Ready: true, UserInvocable: true},
	}
	_, ok := MatchSkillCommand("/", skills)
	if ok {
		t.Error("should not match empty command")
	}
}

func TestMatchSkillCommand_NilSkills(t *testing.T) {
	_, ok := MatchSkillCommand("/weather", nil)
	if ok {
		t.Error("should not match with nil skills")
	}
}

// ---------------------------------------------------------------------------
// SkillCommandList
// ---------------------------------------------------------------------------

func TestSkillCommandList_ReturnsReadyUserInvocable(t *testing.T) {
	skills := []*Skill{
		{Name: "weather", Description: "Get weather", Emoji: "W", Ready: true, UserInvocable: true},
		{Name: "deploy", Description: "Deploy app", Emoji: "D", Ready: true, UserInvocable: true},
		{Name: "internal", Description: "Internal only", Ready: true, UserInvocable: false},
		{Name: "broken", Description: "Not ready", Ready: false, UserInvocable: true},
	}
	cmds := SkillCommandList(skills)
	if len(cmds) != 2 {
		t.Fatalf("got %d commands; want 2 (only ready+user-invocable)", len(cmds))
	}
	if cmds[0].Name != "/weather" {
		t.Errorf("cmds[0].Name = %q; want %q", cmds[0].Name, "/weather")
	}
	if cmds[0].SkillName != "weather" {
		t.Errorf("cmds[0].SkillName = %q; want %q", cmds[0].SkillName, "weather")
	}
	if cmds[1].Name != "/deploy" {
		t.Errorf("cmds[1].Name = %q; want %q", cmds[1].Name, "/deploy")
	}
}

func TestSkillCommandList_EmptySkills(t *testing.T) {
	cmds := SkillCommandList(nil)
	if len(cmds) != 0 {
		t.Errorf("got %d commands; want 0 for nil skills", len(cmds))
	}
}
