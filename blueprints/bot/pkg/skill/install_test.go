package skill

import (
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ParseInstallSpecs
// ---------------------------------------------------------------------------

func TestParseInstallSpecs_Brew(t *testing.T) {
	raw := json.RawMessage(`[{"id":"brew-0","kind":"brew","formula":"gh","bins":["gh"],"label":"Install GitHub CLI (brew)"}]`)
	sk := &Skill{
		Name:       "github",
		InstallRaw: raw,
	}

	specs := ParseInstallSpecs(sk)
	if len(specs) != 1 {
		t.Fatalf("got %d specs; want 1", len(specs))
	}

	spec := specs[0]
	if spec.ID != "brew-0" {
		t.Errorf("ID = %q; want %q", spec.ID, "brew-0")
	}
	if spec.Kind != "brew" {
		t.Errorf("Kind = %q; want %q", spec.Kind, "brew")
	}
	if spec.Formula != "gh" {
		t.Errorf("Formula = %q; want %q", spec.Formula, "gh")
	}
	if spec.Label != "Install GitHub CLI (brew)" {
		t.Errorf("Label = %q; want %q", spec.Label, "Install GitHub CLI (brew)")
	}
	if len(spec.Bins) != 1 || spec.Bins[0] != "gh" {
		t.Errorf("Bins = %v; want [gh]", spec.Bins)
	}
}

func TestParseInstallSpecs_Multiple(t *testing.T) {
	raw := json.RawMessage(`[
		{"id":"brew-0","kind":"brew","formula":"ripgrep","bins":["rg"],"label":"Install ripgrep (brew)"},
		{"id":"node-0","kind":"node","package":"prettier","bins":["prettier"],"label":"Install prettier (npm)"}
	]`)
	sk := &Skill{
		Name:       "search",
		InstallRaw: raw,
	}

	specs := ParseInstallSpecs(sk)
	if len(specs) != 2 {
		t.Fatalf("got %d specs; want 2", len(specs))
	}

	// First spec: brew.
	if specs[0].ID != "brew-0" {
		t.Errorf("specs[0].ID = %q; want %q", specs[0].ID, "brew-0")
	}
	if specs[0].Kind != "brew" {
		t.Errorf("specs[0].Kind = %q; want %q", specs[0].Kind, "brew")
	}
	if specs[0].Formula != "ripgrep" {
		t.Errorf("specs[0].Formula = %q; want %q", specs[0].Formula, "ripgrep")
	}
	if len(specs[0].Bins) != 1 || specs[0].Bins[0] != "rg" {
		t.Errorf("specs[0].Bins = %v; want [rg]", specs[0].Bins)
	}

	// Second spec: node.
	if specs[1].ID != "node-0" {
		t.Errorf("specs[1].ID = %q; want %q", specs[1].ID, "node-0")
	}
	if specs[1].Kind != "node" {
		t.Errorf("specs[1].Kind = %q; want %q", specs[1].Kind, "node")
	}
	if specs[1].Package != "prettier" {
		t.Errorf("specs[1].Package = %q; want %q", specs[1].Package, "prettier")
	}
	if len(specs[1].Bins) != 1 || specs[1].Bins[0] != "prettier" {
		t.Errorf("specs[1].Bins = %v; want [prettier]", specs[1].Bins)
	}
}

func TestParseInstallSpecs_Empty(t *testing.T) {
	// Nil InstallRaw.
	sk := &Skill{Name: "empty"}
	specs := ParseInstallSpecs(sk)
	if specs != nil {
		t.Errorf("expected nil for nil InstallRaw; got %v", specs)
	}

	// Empty InstallRaw.
	sk2 := &Skill{Name: "empty2", InstallRaw: json.RawMessage(`[]`)}
	specs2 := ParseInstallSpecs(sk2)
	if specs2 != nil {
		t.Errorf("expected nil for empty array; got %v", specs2)
	}

	// Nil skill.
	specs3 := ParseInstallSpecs(nil)
	if specs3 != nil {
		t.Errorf("expected nil for nil skill; got %v", specs3)
	}
}

func TestParseInstallSpecs_FilterOS(t *testing.T) {
	// Create specs with OS filtering -- one matches current OS, one does not.
	currentOS := runtime.GOOS
	otherOS := "impossible_os_xyz"

	raw := json.RawMessage(`[
		{"id":"brew-0","kind":"brew","formula":"gh","bins":["gh"],"label":"Install gh (brew)","os":["` + currentOS + `"]},
		{"id":"apt-0","kind":"brew","formula":"gh","bins":["gh"],"label":"Install gh (apt)","os":["` + otherOS + `"]}
	]`)
	sk := &Skill{
		Name:       "github",
		InstallRaw: raw,
	}

	specs := ParseInstallSpecs(sk)
	if len(specs) != 2 {
		t.Fatalf("ParseInstallSpecs should return all specs (filtering is in ToInstallOpts); got %d", len(specs))
	}

	// ToInstallOpts should filter by OS.
	opts := ToInstallOpts(specs)
	if len(opts) != 1 {
		t.Fatalf("ToInstallOpts should return 1 opt (filtered by OS); got %d", len(opts))
	}
	if opts[0].ID != "brew-0" {
		t.Errorf("opts[0].ID = %q; want %q", opts[0].ID, "brew-0")
	}
}

func TestParseInstallSpecs_ViaMetadataParsing(t *testing.T) {
	// Test the full path: SKILL.md with metadata -> ParseInstallSpecs.
	input := `---
name: github
description: "Interact with GitHub"
metadata: {"openclaw":{"emoji":"E","requires":{"bins":["gh"]},"install":[{"id":"brew-0","kind":"brew","formula":"gh","bins":["gh"],"label":"Install GitHub CLI (brew)"}]}}
---
# GitHub
`
	sk, _, err := ParseSkillMD(input)
	if err != nil {
		t.Fatalf("ParseSkillMD error: %v", err)
	}

	if len(sk.InstallRaw) == 0 {
		t.Fatal("InstallRaw should be populated from metadata")
	}

	specs := ParseInstallSpecs(sk)
	if len(specs) != 1 {
		t.Fatalf("got %d specs; want 1", len(specs))
	}
	if specs[0].Kind != "brew" {
		t.Errorf("Kind = %q; want %q", specs[0].Kind, "brew")
	}
	if specs[0].Formula != "gh" {
		t.Errorf("Formula = %q; want %q", specs[0].Formula, "gh")
	}
}

// ---------------------------------------------------------------------------
// ToInstallOpts
// ---------------------------------------------------------------------------

func TestToInstallOpts(t *testing.T) {
	specs := []InstallSpec{
		{
			ID:      "brew-0",
			Kind:    "brew",
			Label:   "Install ripgrep (brew)",
			Bins:    []string{"rg"},
			Formula: "ripgrep",
		},
		{
			ID:      "node-0",
			Kind:    "node",
			Label:   "Install prettier (npm)",
			Bins:    []string{"prettier"},
			Package: "prettier",
		},
	}

	opts := ToInstallOpts(specs)
	if len(opts) != 2 {
		t.Fatalf("got %d opts; want 2", len(opts))
	}

	// Verify first opt.
	if opts[0].ID != "brew-0" {
		t.Errorf("opts[0].ID = %q; want %q", opts[0].ID, "brew-0")
	}
	if opts[0].Kind != "brew" {
		t.Errorf("opts[0].Kind = %q; want %q", opts[0].Kind, "brew")
	}
	if opts[0].Label != "Install ripgrep (brew)" {
		t.Errorf("opts[0].Label = %q; want %q", opts[0].Label, "Install ripgrep (brew)")
	}
	if len(opts[0].Bins) != 1 || opts[0].Bins[0] != "rg" {
		t.Errorf("opts[0].Bins = %v; want [rg]", opts[0].Bins)
	}

	// Verify second opt.
	if opts[1].ID != "node-0" {
		t.Errorf("opts[1].ID = %q; want %q", opts[1].ID, "node-0")
	}
	if opts[1].Kind != "node" {
		t.Errorf("opts[1].Kind = %q; want %q", opts[1].Kind, "node")
	}
}

func TestToInstallOpts_NilBinsBecomesEmptySlice(t *testing.T) {
	specs := []InstallSpec{
		{ID: "go-0", Kind: "go", Label: "Install tool (go)", Module: "example.com/tool@latest"},
	}

	opts := ToInstallOpts(specs)
	if len(opts) != 1 {
		t.Fatalf("got %d opts; want 1", len(opts))
	}
	// Bins should be non-nil empty slice (via nonNil).
	if opts[0].Bins == nil {
		t.Error("Bins should be non-nil empty slice, got nil")
	}
	if len(opts[0].Bins) != 0 {
		t.Errorf("Bins = %v; want empty", opts[0].Bins)
	}
}

func TestToInstallOpts_Empty(t *testing.T) {
	opts := ToInstallOpts(nil)
	if opts != nil {
		t.Errorf("expected nil for nil specs; got %v", opts)
	}

	opts2 := ToInstallOpts([]InstallSpec{})
	if opts2 != nil {
		t.Errorf("expected nil for empty specs; got %v", opts2)
	}
}

func TestToInstallOpts_FiltersOS(t *testing.T) {
	currentOS := runtime.GOOS
	otherOS := "impossible_os_xyz"

	specs := []InstallSpec{
		{ID: "match", Kind: "brew", Label: "Matching OS", Bins: []string{"tool"}, OS: []string{currentOS}},
		{ID: "nomatch", Kind: "brew", Label: "Non-matching OS", Bins: []string{"tool"}, OS: []string{otherOS}},
		{ID: "any", Kind: "brew", Label: "Any OS", Bins: []string{"tool"}}, // no OS filter
	}

	opts := ToInstallOpts(specs)
	if len(opts) != 2 {
		t.Fatalf("got %d opts; want 2 (match + any)", len(opts))
	}

	ids := map[string]bool{}
	for _, opt := range opts {
		ids[opt.ID] = true
	}
	if !ids["match"] {
		t.Error("expected 'match' opt to be included")
	}
	if !ids["any"] {
		t.Error("expected 'any' opt to be included")
	}
	if ids["nomatch"] {
		t.Error("expected 'nomatch' opt to be excluded")
	}
}

// ---------------------------------------------------------------------------
// InstallSkillDep - error cases
// ---------------------------------------------------------------------------

func TestInstallSkillDep_NotFound(t *testing.T) {
	skills := []*Skill{
		{Name: "alpha"},
		{Name: "beta"},
	}

	_, err := InstallSkillDep(skills, "nonexistent", "brew-0", 5000, DefaultInstallPrefs())
	if err == nil {
		t.Fatal("expected error for missing skill, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q; want to contain 'not found'", err.Error())
	}
}

func TestInstallSkillDep_BadInstallID(t *testing.T) {
	raw := json.RawMessage(`[{"id":"brew-0","kind":"brew","formula":"gh","bins":["gh"],"label":"Install gh"}]`)
	skills := []*Skill{
		{Name: "github", InstallRaw: raw},
	}

	_, err := InstallSkillDep(skills, "github", "nonexistent-id", 5000, DefaultInstallPrefs())
	if err == nil {
		t.Fatal("expected error for missing install ID, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q; want to contain 'not found'", err.Error())
	}
}

func TestInstallSkillDep_NoInstallSpecs(t *testing.T) {
	skills := []*Skill{
		{Name: "plain"}, // no InstallRaw
	}

	_, err := InstallSkillDep(skills, "plain", "brew-0", 5000, DefaultInstallPrefs())
	if err == nil {
		t.Fatal("expected error for skill with no install specs, got nil")
	}
	if !strings.Contains(err.Error(), "no install specs") {
		t.Errorf("error = %q; want to contain 'no install specs'", err.Error())
	}
}

func TestInstallSkillDep_DownloadMissingURL(t *testing.T) {
	raw := json.RawMessage(`[{"id":"dl-0","kind":"download","bins":["tool"],"label":"Download tool"}]`)
	skills := []*Skill{
		{Name: "tool", InstallRaw: raw},
	}

	_, err := InstallSkillDep(skills, "tool", "dl-0", 5000, DefaultInstallPrefs())
	if err == nil {
		t.Fatal("expected error for download with no URL, got nil")
	}
	if !strings.Contains(err.Error(), "missing URL") {
		t.Errorf("error = %q; want to contain 'missing URL'", err.Error())
	}
}

func TestInstallSkillDep_UnknownKind(t *testing.T) {
	raw := json.RawMessage(`[{"id":"x-0","kind":"mystery","label":"Mystery install"}]`)
	skills := []*Skill{
		{Name: "mystery", InstallRaw: raw},
	}

	_, err := InstallSkillDep(skills, "mystery", "x-0", 5000, DefaultInstallPrefs())
	if err == nil {
		t.Fatal("expected error for unknown kind, got nil")
	}
	if !strings.Contains(err.Error(), "unknown install kind") {
		t.Errorf("error = %q; want to contain 'unknown install kind'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// buildInstallCommand
// ---------------------------------------------------------------------------

func TestBuildInstallCommand_Brew(t *testing.T) {
	spec := &InstallSpec{Kind: "brew", Formula: "ripgrep"}
	args, err := buildInstallCommand(spec, DefaultInstallPrefs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"brew", "install", "ripgrep"}
	if len(args) != len(want) {
		t.Fatalf("args = %v; want %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("args[%d] = %q; want %q", i, args[i], want[i])
		}
	}
}

func TestBuildInstallCommand_Node(t *testing.T) {
	spec := &InstallSpec{Kind: "node", Package: "prettier"}
	args, err := buildInstallCommand(spec, DefaultInstallPrefs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"npm", "install", "-g", "prettier"}
	if len(args) != len(want) {
		t.Fatalf("args = %v; want %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("args[%d] = %q; want %q", i, args[i], want[i])
		}
	}
}

func TestBuildInstallCommand_Go(t *testing.T) {
	spec := &InstallSpec{Kind: "go", Module: "golang.org/x/tools/cmd/goimports@latest"}
	args, err := buildInstallCommand(spec, DefaultInstallPrefs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"go", "install", "golang.org/x/tools/cmd/goimports@latest"}
	if len(args) != len(want) {
		t.Fatalf("args = %v; want %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("args[%d] = %q; want %q", i, args[i], want[i])
		}
	}
}

func TestBuildInstallCommand_UV(t *testing.T) {
	spec := &InstallSpec{Kind: "uv", Package: "ruff"}
	args, err := buildInstallCommand(spec, DefaultInstallPrefs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"uv", "tool", "install", "ruff"}
	if len(args) != len(want) {
		t.Fatalf("args = %v; want %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("args[%d] = %q; want %q", i, args[i], want[i])
		}
	}
}

func TestBuildInstallCommand_BrewMissingFormula(t *testing.T) {
	spec := &InstallSpec{Kind: "brew"}
	_, err := buildInstallCommand(spec, DefaultInstallPrefs())
	if err == nil {
		t.Fatal("expected error for brew with no formula")
	}
}

func TestBuildInstallCommand_NodeMissingPackage(t *testing.T) {
	spec := &InstallSpec{Kind: "node"}
	_, err := buildInstallCommand(spec, DefaultInstallPrefs())
	if err == nil {
		t.Fatal("expected error for node with no package")
	}
}

func TestBuildInstallCommand_GoMissingModule(t *testing.T) {
	spec := &InstallSpec{Kind: "go"}
	_, err := buildInstallCommand(spec, DefaultInstallPrefs())
	if err == nil {
		t.Fatal("expected error for go with no module")
	}
}

func TestBuildInstallCommand_UVMissingPackage(t *testing.T) {
	spec := &InstallSpec{Kind: "uv"}
	_, err := buildInstallCommand(spec, DefaultInstallPrefs())
	if err == nil {
		t.Fatal("expected error for uv with no package")
	}
}

// ---------------------------------------------------------------------------
// ParseInstallSpecsFromRaw
// ---------------------------------------------------------------------------

func TestParseInstallSpecsFromRaw_ValidJSON(t *testing.T) {
	raw := json.RawMessage(`[{"id":"go-0","kind":"go","module":"example.com/tool@latest","bins":["tool"],"label":"Install tool"}]`)
	specs := ParseInstallSpecsFromRaw(raw)
	if len(specs) != 1 {
		t.Fatalf("got %d specs; want 1", len(specs))
	}
	if specs[0].Kind != "go" {
		t.Errorf("Kind = %q; want %q", specs[0].Kind, "go")
	}
	if specs[0].Module != "example.com/tool@latest" {
		t.Errorf("Module = %q; want %q", specs[0].Module, "example.com/tool@latest")
	}
}

func TestParseInstallSpecsFromRaw_InvalidJSON(t *testing.T) {
	raw := json.RawMessage(`not valid json`)
	specs := ParseInstallSpecsFromRaw(raw)
	if specs != nil {
		t.Errorf("expected nil for invalid JSON; got %v", specs)
	}
}

func TestParseInstallSpecsFromRaw_EmptyArray(t *testing.T) {
	raw := json.RawMessage(`[]`)
	specs := ParseInstallSpecsFromRaw(raw)
	if specs != nil {
		t.Errorf("expected nil for empty array; got %v", specs)
	}
}

func TestParseInstallSpecsFromRaw_Nil(t *testing.T) {
	specs := ParseInstallSpecsFromRaw(nil)
	if specs != nil {
		t.Errorf("expected nil for nil input; got %v", specs)
	}
}

// ---------------------------------------------------------------------------
// Node manager preferences
// ---------------------------------------------------------------------------

func TestBuildNodeCommand_NPM(t *testing.T) {
	args := buildNodeCommand("prettier", "npm")
	want := []string{"npm", "install", "-g", "prettier"}
	assertArgs(t, args, want)
}

func TestBuildNodeCommand_PNPM(t *testing.T) {
	args := buildNodeCommand("prettier", "pnpm")
	want := []string{"pnpm", "add", "-g", "prettier"}
	assertArgs(t, args, want)
}

func TestBuildNodeCommand_Yarn(t *testing.T) {
	args := buildNodeCommand("prettier", "yarn")
	want := []string{"yarn", "global", "add", "prettier"}
	assertArgs(t, args, want)
}

func TestBuildNodeCommand_Bun(t *testing.T) {
	args := buildNodeCommand("prettier", "bun")
	want := []string{"bun", "add", "-g", "prettier"}
	assertArgs(t, args, want)
}

func TestBuildNodeCommand_DefaultFallback(t *testing.T) {
	args := buildNodeCommand("prettier", "unknown")
	want := []string{"npm", "install", "-g", "prettier"}
	assertArgs(t, args, want)
}

func TestBuildInstallCommand_NodeWithPNPM(t *testing.T) {
	spec := &InstallSpec{Kind: "node", Package: "prettier"}
	prefs := InstallPrefs{NodeManager: "pnpm"}
	args, err := buildInstallCommand(spec, prefs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"pnpm", "add", "-g", "prettier"}
	assertArgs(t, args, want)
}

// ---------------------------------------------------------------------------
// Install preferences parsing
// ---------------------------------------------------------------------------

func TestParseInstallPrefs_Defaults(t *testing.T) {
	prefs := ParseInstallPrefs(map[string]any{})
	if !prefs.PreferBrew {
		t.Error("PreferBrew should default to true")
	}
	if prefs.NodeManager != "npm" {
		t.Errorf("NodeManager = %q; want %q", prefs.NodeManager, "npm")
	}
}

func TestParseInstallPrefs_FromConfig(t *testing.T) {
	cfg := map[string]any{
		"skills": map[string]any{
			"install": map[string]any{
				"preferBrew":  false,
				"nodeManager": "yarn",
			},
		},
	}
	prefs := ParseInstallPrefs(cfg)
	if prefs.PreferBrew {
		t.Error("PreferBrew should be false")
	}
	if prefs.NodeManager != "yarn" {
		t.Errorf("NodeManager = %q; want %q", prefs.NodeManager, "yarn")
	}
}

func TestParseInstallPrefs_InvalidNodeManager(t *testing.T) {
	cfg := map[string]any{
		"skills": map[string]any{
			"install": map[string]any{
				"nodeManager": "invalid",
			},
		},
	}
	prefs := ParseInstallPrefs(cfg)
	if prefs.NodeManager != "npm" {
		t.Errorf("NodeManager = %q; want %q (default)", prefs.NodeManager, "npm")
	}
}

// ---------------------------------------------------------------------------
// Download helpers
// ---------------------------------------------------------------------------

func TestFilenameFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/tool-v1.0.tar.gz", "tool-v1.0.tar.gz"},
		{"https://example.com/releases/download/v1.0/tool.zip", "tool.zip"},
		{"https://example.com/", "download"},
		{"https://example.com", "download"},
		{"not a url", "not a url"},
	}
	for _, tt := range tests {
		got := filenameFromURL(tt.url)
		if got != tt.want {
			t.Errorf("filenameFromURL(%q) = %q; want %q", tt.url, got, tt.want)
		}
	}
}

func TestDetectArchiveType(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"tool.tar.gz", "tar.gz"},
		{"tool.tgz", "tar.gz"},
		{"tool.tar.bz2", "tar.bz2"},
		{"tool.tbz2", "tar.bz2"},
		{"tool.zip", "zip"},
		{"tool.bin", ""},
		{"tool", ""},
	}
	for _, tt := range tests {
		got := detectArchiveType(tt.filename)
		if got != tt.want {
			t.Errorf("detectArchiveType(%q) = %q; want %q", tt.filename, got, tt.want)
		}
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	tests := []struct {
		input string
		want  string
	}{
		{"~/tools/foo", home + "/tools/foo"},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}
	for _, tt := range tests {
		got := expandHome(tt.input)
		if got != tt.want {
			t.Errorf("expandHome(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func assertArgs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("args = %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("args[%d] = %q; want %q", i, got[i], want[i])
		}
	}
}
