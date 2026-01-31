package bot

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/skill"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/workspace"
)

// PromptBuilder constructs OpenClaw-compatible system prompts.
type PromptBuilder struct {
	workspaceDir string
}

// NewPromptBuilder creates a prompt builder for the given workspace.
func NewPromptBuilder(workspaceDir string) *PromptBuilder {
	return &PromptBuilder{workspaceDir: workspaceDir}
}

// Build constructs the full system prompt for the given origin and query.
func (pb *PromptBuilder) Build(origin, query string) string {
	var sections []string

	// 1. Identity section.
	sections = append(sections, pb.identitySection())

	// 2. Workspace bootstrap files.
	if pb.workspaceDir != "" {
		if ws := pb.workspaceSection(origin); ws != "" {
			sections = append(sections, ws)
		}
	}

	// 3. Skills.
	if pb.workspaceDir != "" {
		if sk := pb.skillsSection(); sk != "" {
			sections = append(sections, sk)
		}
	}

	// 4. Runtime info.
	sections = append(sections, pb.runtimeSection())

	return strings.Join(sections, "\n\n")
}

// identitySection returns the base identity prompt matching OpenClaw's style.
func (pb *PromptBuilder) identitySection() string {
	return `You are a personal assistant running inside OpenBot.
You help your human through Telegram.
Be genuinely helpful, not performatively helpful.
Have opinions. Be resourceful before asking.`
}

// workspaceSection loads and formats bootstrap files.
func (pb *PromptBuilder) workspaceSection(origin string) string {
	files, err := workspace.LoadBootstrapFiles(pb.workspaceDir)
	if err != nil {
		return ""
	}

	var filtered []workspace.BootstrapFile
	if origin == "subagent" {
		filtered = workspace.FilterForSubagent(files)
	} else {
		filtered = workspace.FilterForMain(files)
	}

	if len(filtered) == 0 {
		return ""
	}

	return workspace.BuildContextPrompt(filtered)
}

// skillsSection loads skills and formats the prompt.
func (pb *PromptBuilder) skillsSection() string {
	skills, err := skill.LoadAllSkills(pb.workspaceDir)
	if err != nil || len(skills) == 0 {
		return ""
	}

	hasReady := false
	for _, s := range skills {
		if s.Ready {
			hasReady = true
			break
		}
	}
	if !hasReady {
		return ""
	}

	return skill.BuildSkillsPrompt(skills)
}

// runtimeSection returns current date/time and runtime info.
func (pb *PromptBuilder) runtimeSection() string {
	now := time.Now()
	return fmt.Sprintf(`## Current Date & Time
%s

## Runtime
host=%s os=%s arch=%s`,
		now.Format("Monday, January 2, 2006 3:04 PM MST"),
		"openbot",
		runtime.GOOS,
		runtime.GOARCH,
	)
}
