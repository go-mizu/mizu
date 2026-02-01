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
