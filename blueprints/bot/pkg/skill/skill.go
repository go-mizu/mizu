package skill

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const skillFileName = "SKILL.md"

// Skill represents a loaded skill from SKILL.md.
type Skill struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Emoji       string   `yaml:"emoji"`
	Source      string   // "bundled", "workspace", "user"
	Dir         string   // directory containing SKILL.md
	Content     string   // full SKILL.md content (after frontmatter)
	Ready       bool     // all requirements met
	Requires    Requires `yaml:"requires"`
}

// Requires declares what a skill needs to be eligible.
type Requires struct {
	Binaries []string `yaml:"binaries"`
	Config   []string `yaml:"config"`
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

// ParseSkillMD parses SKILL.md with YAML frontmatter (--- delimited).
// Returns the parsed Skill metadata, the body content after frontmatter, and any error.
// Frontmatter is parsed manually to avoid a YAML dependency -- it handles
// simple key: value pairs and the list syntax for requires.binaries and requires.config.
func ParseSkillMD(content string) (*Skill, string, error) {
	sk := &Skill{}

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
// Supports:
//
//	name: value
//	description: value
//	emoji: value
//	requires:
//	  binaries:
//	    - git
//	  config:
//	    - SOME_KEY
func parseFrontmatter(sk *Skill, fm string) {
	lines := strings.Split(fm, "\n")

	var section string    // current top-level key being parsed ("requires")
	var subsection string // current sub-key under requires ("binaries", "config")

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
			sk.Description = val
		case "emoji":
			sk.Emoji = val
		case "requires":
			section = "requires"
			subsection = ""
		default:
			// Reset section tracking for unknown top-level keys.
			section = ""
			subsection = ""
		}
	}
}

// CheckEligibility checks if a skill's requirements are met.
// Returns true when all required binaries are in PATH and all required
// config environment variables are set.
func CheckEligibility(s *Skill) bool {
	for _, bin := range s.Requires.Binaries {
		if _, err := exec.LookPath(bin); err != nil {
			return false
		}
	}
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

// LoadAllSkills loads skills from the workspace and any number of bundled directories.
// Workspace skills take precedence over bundled skills with the same name.
func LoadAllSkills(workspaceDir string, bundledDirs ...string) ([]*Skill, error) {
	seen := make(map[string]bool)
	var all []*Skill

	// Workspace skills have highest precedence.
	ws, err := LoadWorkspaceSkills(workspaceDir)
	if err != nil {
		return nil, err
	}
	for _, s := range ws {
		s.Source = "workspace"
		seen[s.Name] = true
		all = append(all, s)
	}

	// Bundled skills fill in the rest.
	for _, dir := range bundledDirs {
		bundled, err := loadSkillsFromDir(dir, "bundled")
		if err != nil {
			return nil, err
		}
		for _, s := range bundled {
			if seen[s.Name] {
				continue // workspace override wins
			}
			s.Source = "bundled"
			seen[s.Name] = true
			all = append(all, s)
		}
	}

	return all, nil
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
// Format per skill: "- <emoji> **<name>**: <description>"
// Only ready (eligible) skills are included.
func BuildSkillsPrompt(skills []*Skill) string {
	var b strings.Builder
	b.WriteString("# Available Skills\n\n")

	any := false
	for _, s := range skills {
		if !s.Ready {
			continue
		}
		any = true

		b.WriteString("- ")
		if s.Emoji != "" {
			b.WriteString(s.Emoji)
			b.WriteString(" ")
		}
		b.WriteString("**")
		b.WriteString(s.Name)
		b.WriteString("**")
		if s.Description != "" {
			b.WriteString(": ")
			b.WriteString(s.Description)
		}
		b.WriteString("\n")
	}

	if !any {
		b.WriteString("No skills available.\n")
	}

	return b.String()
}
