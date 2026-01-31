package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Bootstrap file names matching OpenClaw.
const (
	AgentsFile    = "AGENTS.md"
	SoulFile      = "SOUL.md"
	ToolsFile     = "TOOLS.md"
	IdentityFile  = "IDENTITY.md"
	UserFile      = "USER.md"
	HeartbeatFile = "HEARTBEAT.md"
	BootFile      = "BOOTSTRAP.md"
	MemoryFile    = "MEMORY.md"
)

// bootstrapOrder defines the OpenClaw load order.
var bootstrapOrder = []string{
	AgentsFile,
	SoulFile,
	ToolsFile,
	IdentityFile,
	UserFile,
	HeartbeatFile,
	BootFile,
	MemoryFile,
}

// subagentFiles are the only bootstrap files loaded for subagent sessions.
var subagentFiles = map[string]bool{
	AgentsFile: true,
	ToolsFile:  true,
}

// BootstrapFile represents a single workspace bootstrap file.
type BootstrapFile struct {
	Name    string
	Path    string
	Content string
	Missing bool
}

// LoadBootstrapFiles loads all workspace bootstrap files.
// Returns them in OpenClaw order: AGENTS, SOUL, TOOLS, IDENTITY, USER, HEARTBEAT, BOOTSTRAP, MEMORY.
func LoadBootstrapFiles(workspaceDir string) ([]BootstrapFile, error) {
	files := make([]BootstrapFile, 0, len(bootstrapOrder))

	for _, name := range bootstrapOrder {
		p := filepath.Join(workspaceDir, name)
		bf := BootstrapFile{
			Name: name,
			Path: p,
		}

		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				bf.Missing = true
				files = append(files, bf)
				continue
			}
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		bf.Content = string(data)
		files = append(files, bf)
	}

	return files, nil
}

// FilterForSubagent filters bootstrap files for subagent sessions (only AGENTS.md + TOOLS.md).
func FilterForSubagent(files []BootstrapFile) []BootstrapFile {
	out := make([]BootstrapFile, 0, 2)
	for _, f := range files {
		if subagentFiles[f.Name] && !f.Missing {
			out = append(out, f)
		}
	}
	return out
}

// FilterForMain returns all non-missing bootstrap files (for main/DM sessions).
func FilterForMain(files []BootstrapFile) []BootstrapFile {
	out := make([]BootstrapFile, 0, len(files))
	for _, f := range files {
		if !f.Missing {
			out = append(out, f)
		}
	}
	return out
}

// BuildContextPrompt builds the project context section for system prompt injection.
// Format: "# Project Context\n\n## AGENTS.md\n[content]\n\n## SOUL.md\n[content]..."
func BuildContextPrompt(files []BootstrapFile) string {
	var b strings.Builder
	b.WriteString("# Project Context\n")

	for _, f := range files {
		if f.Missing || f.Content == "" {
			continue
		}
		b.WriteString("\n## ")
		b.WriteString(f.Name)
		b.WriteString("\n")
		b.WriteString(f.Content)
		if !strings.HasSuffix(f.Content, "\n") {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// defaultTemplates maps file names to default content created by EnsureWorkspace.
var defaultTemplates = map[string]string{
	AgentsFile: `# AGENTS.md - Your Workspace

## Every Session

Before doing anything else:
1. Read SOUL.md - this is who you are
2. Read USER.md - this is who you're helping
3. Read memory/YYYY-MM-DD.md (today + yesterday) for recent context
4. If in MAIN SESSION (direct chat with your human): Also read MEMORY.md

Don't ask permission. Just do it.

## Memory

You wake up fresh each session. These files are your continuity:
- Daily notes: memory/YYYY-MM-DD.md
- Long-term: MEMORY.md
`,
	SoulFile: `# SOUL.md - Who You Are

Be genuinely helpful, not performatively helpful.
Have opinions. Be resourceful before asking.
Earn trust through competence.
`,
	UserFile: `# USER.md - About Your Human

(Edit this file with information about the user)
`,
	ToolsFile: `# TOOLS.md - Local Notes

Skills define how tools work. This file is for your specifics.
`,
}

// EnsureWorkspace creates the workspace directory and default template files if missing.
func EnsureWorkspace(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create workspace dir: %w", err)
	}

	for name, content := range defaultTemplates {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			continue // file already exists, don't overwrite
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}

	return EnsureMemoryDir(dir)
}

// EnsureMemoryDir creates the memory/ subdirectory inside the workspace.
func EnsureMemoryDir(workspaceDir string) error {
	memDir := filepath.Join(workspaceDir, "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		return fmt.Errorf("create memory dir: %w", err)
	}
	return nil
}
