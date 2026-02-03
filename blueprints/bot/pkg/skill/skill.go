package skill

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const skillFileName = "SKILL.md"

// Skill represents a loaded skill from SKILL.md.
type Skill struct {
	Name                   string   // skill identifier
	Description            string   // trigger text for AI
	Emoji                  string   // visual indicator
	Homepage               string   // documentation URL
	Source                 string   // "bundled", "workspace", "user"
	Dir                    string   // directory containing SKILL.md
	Content                string   // full SKILL.md content (after frontmatter)
	Ready                  bool     // all requirements met
	Always                 bool     // always included in prompt (not on-demand)
	UserInvocable          bool     // can user /command invoke (default true)
	DisableModelInvocation bool     // prevent AI auto-trigger (default false)
	PrimaryEnv             string            // main env var for API key
	SkillKey               string            // override key for config lookup
	Requires               Requires          // dependency requirements
	InstallRaw             json.RawMessage   // raw install JSON from metadata
}

// Requires declares what a skill needs to be eligible.
type Requires struct {
	Binaries []string // AND: all must exist in PATH
	AnyBins  []string // OR: at least one must exist
	Config   []string // env vars (AND: all must be non-empty)
	OS       []string // platform filter (e.g. ["darwin", "linux"])
	CfgPaths []string // JSON config path requirements
}

// LoadSkill loads a skill from a directory containing a SKILL.md file.
func LoadSkill(dir string) (*Skill, error) {
	p := filepath.Join(dir, skillFileName)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", p, err)
	}

	sk, body, err := ParseSkillMD(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", p, err)
	}

	sk.Dir = dir
	sk.Content = body
	sk.Ready = CheckEligibility(sk)

	// Default name from directory name if not specified in frontmatter.
	if sk.Name == "" {
		sk.Name = filepath.Base(dir)
	}

	return sk, nil
}

// openclawMetadata mirrors the JSON structure inside metadata: {"openclaw":{...}}.
type openclawMetadata struct {
	Emoji      string `json:"emoji"`
	Homepage   string `json:"homepage"`
	SkillKey   string `json:"skillKey"`
	PrimaryEnv string `json:"primaryEnv"`
	Always     *bool  `json:"always"`
	OS         any    `json:"os"` // string or []string
	Requires   *struct {
		Bins    any `json:"bins"`    // string or []string
		AnyBins any `json:"anyBins"` // string or []string
		Env     any `json:"env"`     // string or []string
		Config  any `json:"config"`  // string or []string
	} `json:"requires"`
	Install json.RawMessage `json:"install"` // preserved but not parsed
}

// resolveStringList normalises a JSON value that may be a single string or
// an array of strings into a []string.
func resolveStringList(v any) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case string:
		parts := strings.Split(val, ",")
		var out []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	case []any:
		var out []string
		for _, item := range val {
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return out
	}
	return nil
}

// parseMetadataJSON parses the OpenClaw metadata JSON blob and merges
// its fields into the Skill. Fields from metadata override simple YAML
// equivalents when present.
func parseMetadataJSON(sk *Skill, raw string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return
	}

	// The metadata value is JSON: {"openclaw": {...}} or legacy keys.
	var outer map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &outer); err != nil {
		return
	}

	// Try "openclaw" key, then legacy keys.
	keys := []string{"openclaw", "clawdbot", "moltbot"}
	var inner json.RawMessage
	for _, k := range keys {
		if v, ok := outer[k]; ok {
			inner = v
			break
		}
	}
	if inner == nil {
		return
	}

	var meta openclawMetadata
	if err := json.Unmarshal(inner, &meta); err != nil {
		return
	}

	// Merge fields - metadata overrides YAML equivalents when non-empty.
	if meta.Emoji != "" {
		sk.Emoji = meta.Emoji
	}
	if meta.Homepage != "" {
		sk.Homepage = meta.Homepage
	}
	if meta.SkillKey != "" {
		sk.SkillKey = meta.SkillKey
	}
	if meta.PrimaryEnv != "" {
		sk.PrimaryEnv = meta.PrimaryEnv
	}
	if meta.Always != nil {
		sk.Always = *meta.Always
	}

	// OS list.
	if osList := resolveStringList(meta.OS); len(osList) > 0 {
		sk.Requires.OS = osList
	}

	// Requirements.
	if meta.Requires != nil {
		if bins := resolveStringList(meta.Requires.Bins); len(bins) > 0 {
			sk.Requires.Binaries = bins
		}
		if anyBins := resolveStringList(meta.Requires.AnyBins); len(anyBins) > 0 {
			sk.Requires.AnyBins = anyBins
		}
		if env := resolveStringList(meta.Requires.Env); len(env) > 0 {
			sk.Requires.Config = env
		}
		if cfgPaths := resolveStringList(meta.Requires.Config); len(cfgPaths) > 0 {
			sk.Requires.CfgPaths = cfgPaths
		}
	}

	// Preserve raw install JSON for later parsing by ParseInstallSpecs.
	if len(meta.Install) > 0 {
		sk.InstallRaw = meta.Install
	}
}

// ParseSkillMD parses SKILL.md with YAML frontmatter (--- delimited).
// Returns the parsed Skill metadata, the body content after frontmatter, and any error.
// Frontmatter is parsed manually to avoid a YAML dependency -- it handles
// simple key: value pairs, list syntax, and OpenClaw's metadata JSON blob.
func ParseSkillMD(content string) (*Skill, string, error) {
	sk := &Skill{
		UserInvocable: true, // default per OpenClaw spec
	}

	// No frontmatter: entire content is body.
	if !strings.HasPrefix(content, "---") {
		return sk, content, nil
	}

	// Find closing ---.
	rest := content[3:]
	// Trim the newline right after opening ---.
	rest = strings.TrimLeft(rest, "\r\n")

	idx := strings.Index(rest, "---")
	if idx < 0 {
		return sk, content, nil // no closing delimiter, treat all as body
	}

	frontmatter := rest[:idx]
	body := strings.TrimLeft(rest[idx+3:], "\r\n")

	// Parse simple YAML key: value frontmatter.
	parseFrontmatter(sk, frontmatter)

	return sk, body, nil
}

// parseFrontmatter handles the simple YAML subset used in SKILL.md frontmatter.
// Supports simple key: value pairs, nested requires lists, and OpenClaw's
// metadata JSON blob, homepage, user-invocable, and disable-model-invocation.
func parseFrontmatter(sk *Skill, fm string) {
	lines := strings.Split(fm, "\n")

	var section string    // current top-level key being parsed ("requires")
	var subsection string // current sub-key under requires ("binaries", "config")
	var metadataRaw string

	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r")

		// Skip blank lines.
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Detect indentation level.
		trimmed := strings.TrimLeft(line, " \t")
		indent := len(line) - len(trimmed)

		// List item (- value) under a subsection.
		if strings.HasPrefix(trimmed, "- ") && section == "requires" && subsection != "" {
			val := strings.TrimSpace(trimmed[2:])
			switch subsection {
			case "binaries":
				sk.Requires.Binaries = append(sk.Requires.Binaries, val)
			case "config":
				sk.Requires.Config = append(sk.Requires.Config, val)
			case "os":
				sk.Requires.OS = append(sk.Requires.OS, val)
			}
			continue
		}

		// key: value pair.
		colonIdx := strings.Index(trimmed, ":")
		if colonIdx < 0 {
			continue
		}

		key := strings.TrimSpace(trimmed[:colonIdx])
		val := strings.TrimSpace(trimmed[colonIdx+1:])

		// Subsection under requires (indent > 0, section == "requires").
		if indent > 0 && section == "requires" {
			if val == "" {
				// This is a subsection header like "binaries:" or "config:".
				subsection = key
			}
			continue
		}

		// Top-level keys.
		switch key {
		case "name":
			sk.Name = val
		case "description":
			// Handle quoted descriptions (common in OpenClaw SKILL.md).
			sk.Description = unquoteValue(val)
		case "emoji":
			sk.Emoji = val
		case "homepage":
			sk.Homepage = val
		case "always":
			sk.Always = val == "true" || val == "yes"
		case "user-invocable":
			sk.UserInvocable = val != "false" && val != "no"
		case "disable-model-invocation":
			sk.DisableModelInvocation = val == "true" || val == "yes"
		case "metadata":
			metadataRaw = val
		case "requires":
			section = "requires"
			subsection = ""
		default:
			// Reset section tracking for unknown top-level keys.
			section = ""
			subsection = ""
		}
	}

	// Parse OpenClaw metadata JSON blob (overrides simple YAML fields).
	if metadataRaw != "" {
		parseMetadataJSON(sk, metadataRaw)
	}
}

// unquoteValue strips surrounding double quotes from a YAML value.
func unquoteValue(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// CheckEligibility checks if a skill's requirements are met.
// Returns true when all required binaries are in PATH, all required
// config environment variables are set, the OS matches (if specified),
// and anyBins has at least one match. Skills with always=true bypass
// binary/config checks but still respect OS filtering.
func CheckEligibility(s *Skill) bool {
	// Check OS requirement first (always applies, even for "always" skills).
	if len(s.Requires.OS) > 0 {
		currentOS := runtime.GOOS
		osMatch := false
		for _, o := range s.Requires.OS {
			if o == currentOS {
				osMatch = true
				break
			}
		}
		if !osMatch {
			return false
		}
	}

	// Skills with always=true skip binary/config checks.
	if s.Always {
		return true
	}

	// AND: all required binaries must exist.
	for _, bin := range s.Requires.Binaries {
		if _, err := exec.LookPath(bin); err != nil {
			return false
		}
	}

	// OR: at least one of anyBins must exist.
	if len(s.Requires.AnyBins) > 0 {
		found := false
		for _, bin := range s.Requires.AnyBins {
			if _, err := exec.LookPath(bin); err == nil {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// AND: all config env vars must be non-empty.
	for _, key := range s.Requires.Config {
		if os.Getenv(key) == "" {
			return false
		}
	}

	return true
}

// LoadWorkspaceSkills loads all skills from the workspace/skills/ directory.
// Each subdirectory containing a SKILL.md is loaded as a workspace skill.
func LoadWorkspaceSkills(workspaceDir string) ([]*Skill, error) {
	skillsDir := filepath.Join(workspaceDir, "skills")
	return loadSkillsFromDir(skillsDir, "workspace")
}

// LoadUserSkills loads skills from the user's ~/.openbot/skills/ directory.
// These are user-installed skills that take precedence over bundled but not workspace.
func LoadUserSkills() ([]*Skill, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil // non-fatal
	}
	userSkillsDir := filepath.Join(home, ".openbot", "skills")
	return loadSkillsFromDir(userSkillsDir, "user")
}

// LoadAllSkills loads skills from workspace, user-installed, and bundled directories.
// Precedence: workspace > user-installed > bundled.
// For extra directory support, use LoadAllSkillsWithExtras.
func LoadAllSkills(workspaceDir string, bundledDirs ...string) ([]*Skill, error) {
	return LoadAllSkillsWithExtras(workspaceDir, nil, bundledDirs...)
}

// LoadAllSkillsWithExtras loads skills from workspace, user-installed, extra directories,
// and bundled directories. Precedence: workspace > user > extra > bundled.
// extraDirs supports the OpenClaw skills.load.extraDirs config.
func LoadAllSkillsWithExtras(workspaceDir string, extraDirs []string, bundledDirs ...string) ([]*Skill, error) {
	seen := make(map[string]bool)
	var all []*Skill

	// 1. Workspace skills have highest precedence.
	ws, err := LoadWorkspaceSkills(workspaceDir)
	if err != nil {
		return nil, err
	}
	for _, s := range ws {
		s.Source = "workspace"
		seen[s.Name] = true
		all = append(all, s)
	}

	// 2. User-installed skills (from ~/.openbot/skills/).
	userSkills, err := LoadUserSkills()
	if err != nil {
		return nil, err
	}
	for _, s := range userSkills {
		if seen[s.Name] {
			continue // workspace override wins
		}
		s.Source = "user"
		seen[s.Name] = true
		all = append(all, s)
	}

	// 3. Extra directories from config (skills.load.extraDirs).
	for _, dir := range extraDirs {
		extra, err := loadSkillsFromDir(dir, "extra")
		if err != nil {
			continue // non-fatal for extra dirs
		}
		for _, s := range extra {
			if seen[s.Name] {
				continue // workspace/user override wins
			}
			s.Source = "extra"
			seen[s.Name] = true
			all = append(all, s)
		}
	}

	// 4. Bundled skills fill in the rest.
	for _, dir := range bundledDirs {
		bundled, err := loadSkillsFromDir(dir, "bundled")
		if err != nil {
			return nil, err
		}
		for _, s := range bundled {
			if seen[s.Name] {
				continue // workspace/user/extra override wins
			}
			s.Source = "bundled"
			seen[s.Name] = true
			all = append(all, s)
		}
	}

	return all, nil
}

// ParseExtraDirs extracts extra skill directories from config.
// Reads from cfg["skills"]["load"]["extraDirs"] as a string array.
// Expands ~ in paths to the user's home directory.
func ParseExtraDirs(cfg map[string]any) []string {
	skills, ok := cfg["skills"].(map[string]any)
	if !ok {
		return nil
	}
	load, ok := skills["load"].(map[string]any)
	if !ok {
		return nil
	}
	raw, ok := load["extraDirs"]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	var dirs []string
	for _, v := range arr {
		if sv, ok := v.(string); ok && sv != "" {
			dirs = append(dirs, expandHome(sv))
		}
	}
	return dirs
}

// loadSkillsFromDir loads all skills from subdirectories of the given path.
// Each subdirectory must contain a SKILL.md file to be recognized.
func loadSkillsFromDir(dir, source string) ([]*Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // missing directory is fine
		}
		return nil, fmt.Errorf("read skills dir %s: %w", dir, err)
	}

	var skills []*Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subdir := filepath.Join(dir, entry.Name())
		// Only load if SKILL.md exists.
		if _, err := os.Stat(filepath.Join(subdir, skillFileName)); err != nil {
			continue
		}
		sk, err := LoadSkill(subdir)
		if err != nil {
			continue // skip broken skills silently
		}
		sk.Source = source
		skills = append(skills, sk)
	}

	return skills, nil
}

// BuildSkillsPrompt builds the skills section for system prompt injection.
// Uses OpenClaw's XML <available_skills> format for compatibility.
// Only ready (eligible) skills that allow model invocation are included.
func BuildSkillsPrompt(skills []*Skill) string {
	var b strings.Builder
	b.WriteString("\n\nThe following skills provide specialized instructions for specific tasks.\n")
	b.WriteString("Use the read tool to load a skill's file when the task matches its description.\n\n")
	b.WriteString("<available_skills>\n")

	hasReady := false
	for _, s := range skills {
		if !s.Ready {
			continue
		}
		if s.DisableModelInvocation {
			continue
		}
		hasReady = true

		b.WriteString("  <skill>\n")
		b.WriteString(fmt.Sprintf("    <name>%s</name>\n", s.Name))
		if s.Description != "" {
			b.WriteString(fmt.Sprintf("    <description>%s</description>\n", s.Description))
		}
		skillPath := filepath.Join(s.Dir, skillFileName)
		b.WriteString(fmt.Sprintf("    <location>%s</location>\n", skillPath))
		b.WriteString("  </skill>\n")
	}

	if !hasReady {
		b.WriteString("  <!-- No skills available -->\n")
	}

	b.WriteString("</available_skills>\n")
	return b.String()
}

// BuildAlwaysSkillsPrompt returns the body content of all always-mode skills
// that are ready, concatenated for system prompt injection.
// In OpenClaw, always-skills have their content injected directly into the
// prompt (not just listed in <available_skills>).
func BuildAlwaysSkillsPrompt(skills []*Skill) string {
	var parts []string
	for _, s := range skills {
		if !s.Ready || !s.Always {
			continue
		}
		if s.Content != "" {
			parts = append(parts, fmt.Sprintf("### %s\n%s", s.Name, s.Content))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n\n")
}

// CheckEligibilityWithConfig checks eligibility including config path requirements.
// Standard eligibility is checked first, then config paths are verified against
// the loaded JSON config.
func CheckEligibilityWithConfig(s *Skill, cfg map[string]any) bool {
	if !CheckEligibility(s) {
		return false
	}
	for _, path := range s.Requires.CfgPaths {
		if !ConfigPathTruthy(cfg, path) {
			return false
		}
	}
	return true
}

// ConfigPathTruthy checks if a dot-separated config path resolves to a truthy value.
func ConfigPathTruthy(cfg map[string]any, path string) bool {
	parts := strings.Split(path, ".")
	var current any = cfg
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return false
		}
		current, ok = m[part]
		if !ok {
			return false
		}
	}
	switch v := current.(type) {
	case bool:
		return v
	case string:
		return v != ""
	case nil:
		return false
	default:
		return true
	}
}

// MatchSkillCommand checks if a slash command (e.g. "/weather") matches a
// user-invocable, ready skill. Returns the skill and true if matched.
// The cmd should include the "/" prefix (e.g. "/weather").
func MatchSkillCommand(cmd string, skills []*Skill) (*Skill, bool) {
	// Strip leading "/" to get the skill name.
	name := strings.TrimPrefix(cmd, "/")
	if name == "" {
		return nil, false
	}
	for _, s := range skills {
		if !s.Ready || !s.UserInvocable {
			continue
		}
		if s.Name == name {
			return s, true
		}
	}
	return nil, false
}

// SkillCommandList returns the available slash commands derived from
// user-invocable, ready skills. Each command is the skill name prefixed
// with "/" and includes the skill's description and emoji.
func SkillCommandList(skills []*Skill) []SkillCommand {
	var cmds []SkillCommand
	for _, s := range skills {
		if !s.Ready || !s.UserInvocable {
			continue
		}
		cmds = append(cmds, SkillCommand{
			Name:        "/" + s.Name,
			Description: s.Description,
			Emoji:       s.Emoji,
			SkillName:   s.Name,
		})
	}
	return cmds
}

// SkillCommand represents a slash command generated from a user-invocable skill.
type SkillCommand struct {
	Name        string `json:"name"`        // e.g. "/weather"
	Description string `json:"description"` // skill description
	Emoji       string `json:"emoji"`       // visual indicator
	SkillName   string `json:"skillName"`   // underlying skill name
}

// BundledSkillsDir returns the path to bundled skills.
// Checks in order:
// 1. OPENBOT_BUNDLED_SKILLS_DIR env var
// 2. "skills" directory sibling to the running binary
// 3. Empty string (no bundled skills)
func BundledSkillsDir() string {
	if dir := os.Getenv("OPENBOT_BUNDLED_SKILLS_DIR"); dir != "" {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}

	// Check sibling to binary.
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(exe), "skills")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}

	return ""
}
