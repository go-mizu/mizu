package gateway

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/config"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/memory"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/skill"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/workspace"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// contextBuilder assembles the full system prompt for an LLM request by
// combining the agent's base prompt, workspace bootstrap files, skills,
// memory search results, and OpenClaw-compatible sections.
type contextBuilder struct {
	memReg *memoryRegistry
}

func newContextBuilder(memReg *memoryRegistry) *contextBuilder {
	return &contextBuilder{memReg: memReg}
}

// BuildResult is the output of building a system prompt.
// It contains the final prompt string plus metadata for reporting.
type BuildResult struct {
	Prompt        string
	InjectedFiles []types.InjectedFileStats // workspace files that were injected
	ContextChars  int                        // chars from project context section
	SkillNames    []string                   // names of included skills
	HasSOUL       bool                       // whether SOUL.md was present
}

// buildSystemPrompt constructs the enriched system prompt for an agent.
// It produces sections in OpenClaw order:
//  1. Identity (agent base prompt)
//  2. Project Context (workspace bootstrap files)
//  3. Memory Recall
//  4. User Identity
//  5. Messaging (when message tool available)
//  6. Skills
//  7. Heartbeats
//  8. CLI Quick Reference
//  9. Tool Availability
//  10. Date & Time
//  11. Runtime
func (cb *contextBuilder) buildSystemPrompt(ctx context.Context, params *SystemPromptParams) *BuildResult {
	var sections []string
	result := &BuildResult{}

	// 1. Identity (agent base prompt or default).
	sections = append(sections, buildIdentitySection(params.Agent))

	// 2. Project Context (workspace bootstrap files).
	if params.WorkspaceDir != "" {
		wsSection, injected, hasSoul := cb.buildWorkspaceSection(params.WorkspaceDir, params.Origin)
		if wsSection != "" {
			// Add SOUL.md guidance if present.
			if hasSoul {
				wsSection += "\nIf SOUL.md is present, embody its persona and tone. " +
					"Avoid stiff, generic replies; follow its guidance unless " +
					"higher-priority instructions override it.\n"
				result.HasSOUL = true
			}
			result.ContextChars = len(wsSection)
			result.InjectedFiles = injected
			sections = append(sections, wsSection)
		}
	}

	// 3. Memory Recall (MEMORY.md + daily logs + search).
	if params.MemoryPrompt != "" {
		sections = append(sections, "## Memory Recall\n"+params.MemoryPrompt)
	} else if params.WorkspaceDir != "" {
		if memSection := cb.buildMemoryRecallSection(ctx, params); memSection != "" {
			sections = append(sections, memSection)
		}
	}

	// 4. User Identity.
	if ui := buildUserIdentitySection(params); ui != "" {
		sections = append(sections, ui)
	}

	// 5. Messaging (when message tool is available).
	if hasMessageTool(params.ToolNames) {
		sections = append(sections, buildMessagingSection(params))
	}

	// 6. Skills.
	if params.SkillsPrompt != "" {
		sections = append(sections, buildSkillsGuidanceSection(params))
	}

	// 7. Group Chat Context.
	if params.GroupContext != "" {
		sections = append(sections, "## Group Chat Context\n"+params.GroupContext)
	}

	// 8. Heartbeats.
	if params.HeartbeatPrompt != "" {
		sections = append(sections, "## Heartbeats\n"+params.HeartbeatPrompt)
	}

	// 9. CLI Quick Reference.
	sections = append(sections, buildCLIReferenceSection())

	// 10. Tool Availability.
	if len(params.ToolNames) > 0 {
		sections = append(sections, buildToolAvailabilitySection(params.ToolNames))
	}

	// 11. Date & Time.
	sections = append(sections, buildDateTimeSection(params.UserTimezone))

	// 12. Runtime.
	sections = append(sections, buildRuntimeSection(params))

	result.Prompt = strings.Join(sections, "\n\n")
	return result
}

// buildWorkspaceSection loads bootstrap files and formats them for injection.
// Returns the formatted section, injected file stats, and whether SOUL.md was present.
func (cb *contextBuilder) buildWorkspaceSection(workspaceDir, origin string) (string, []types.InjectedFileStats, bool) {
	files, err := workspace.LoadBootstrapFiles(workspaceDir)
	if err != nil {
		return "", nil, false
	}

	// Filter based on session origin.
	var filtered []workspace.BootstrapFile
	if origin == "subagent" {
		filtered = workspace.FilterForSubagent(files)
	} else {
		filtered = workspace.FilterForMain(files)
	}

	if len(filtered) == 0 {
		return "", nil, false
	}

	// Track injected files and detect SOUL.md.
	var injected []types.InjectedFileStats
	hasSoul := false
	for _, f := range filtered {
		if f.Missing || f.Content == "" {
			continue
		}
		injected = append(injected, types.InjectedFileStats{
			Path:  f.Name,
			Chars: len(f.Content),
		})
		if f.Name == workspace.SoulFile {
			hasSoul = true
		}
	}

	return workspace.BuildContextPrompt(filtered), injected, hasSoul
}

// buildSkillsSection loads skills and returns the XML prompt and loaded skills.
// It respects skills.load.extraDirs from the config.
func (cb *contextBuilder) buildSkillsSection(workspaceDir string) (string, []*skill.Skill) {
	// Load extra dirs from config if available.
	var extraDirs []string
	if rawCfg, err := config.LoadRawConfig(config.DefaultConfigPath()); err == nil {
		extraDirs = skill.ParseExtraDirs(rawCfg)
	}
	skills, err := skill.LoadAllSkillsWithExtras(workspaceDir, extraDirs, skill.BundledSkillsDir())
	if err != nil || len(skills) == 0 {
		return "", nil
	}

	hasReady := false
	for _, s := range skills {
		if s.Ready {
			hasReady = true
			break
		}
	}
	if !hasReady {
		return "", nil
	}

	return skill.BuildSkillsPrompt(skills), skills
}

// buildMemorySection searches the memory index and formats relevant results.
func (cb *contextBuilder) buildMemorySection(ctx context.Context, workspaceDir, query string) string {
	if cb.memReg == nil {
		return ""
	}

	mgr, err := cb.memReg.get(workspaceDir)
	if err != nil || mgr == nil {
		return ""
	}

	results, err := mgr.Search(ctx, query, 0, 0)
	if err != nil || len(results) == 0 {
		return ""
	}

	return formatMemoryResults(results)
}

// buildMemoryRecallSection combines MEMORY.md, daily logs, and search results.
// Matches OpenClaw's memory recall behavior.
func (cb *contextBuilder) buildMemoryRecallSection(ctx context.Context, params *SystemPromptParams) string {
	var parts []string

	// 1. MEMORY.md (main sessions only — security: don't leak to group/subagent).
	if params.Origin != "group" && params.Origin != "subagent" {
		memoryMdPath := filepath.Join(params.WorkspaceDir, "MEMORY.md")
		if content, err := os.ReadFile(memoryMdPath); err == nil && len(content) > 0 {
			// Truncate at 4000 chars to avoid bloating the prompt.
			text := string(content)
			if len(text) > 4000 {
				text = text[:4000] + "\n... [truncated]"
			}
			parts = append(parts, "### Long-term Memory\n"+text)
		}
	}

	// 2. Daily logs (today + yesterday).
	now := time.Now()
	for _, offset := range []int{0, -1} {
		day := now.AddDate(0, 0, offset)
		logPath := filepath.Join(params.WorkspaceDir, "memory", day.Format("2006-01-02")+".md")
		if content, err := os.ReadFile(logPath); err == nil && len(content) > 0 {
			text := string(content)
			if len(text) > 2000 {
				text = text[:2000] + "\n... [truncated]"
			}
			parts = append(parts, fmt.Sprintf("### Daily Log (%s)\n%s", day.Format("2006-01-02"), text))
		}
	}

	// 3. Search results from the current query.
	if params.Query != "" {
		if searchSection := cb.buildMemorySection(ctx, params.WorkspaceDir, params.Query); searchSection != "" {
			parts = append(parts, searchSection)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return "## Memory Recall\n\n" + strings.Join(parts, "\n\n")
}

// formatMemoryResults formats search results for system prompt injection.
func formatMemoryResults(results []memory.SearchResult) string {
	var b strings.Builder
	b.WriteString("# Relevant Context\n\n")
	b.WriteString("The following snippets from the workspace may be relevant:\n\n")

	for i, r := range results {
		b.WriteString(fmt.Sprintf("## [%d] %s (lines %d-%d, score: %.2f)\n",
			i+1, r.Path, r.StartLine, r.EndLine, r.Score))
		b.WriteString("```\n")
		b.WriteString(r.Snippet)
		if !strings.HasSuffix(r.Snippet, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("```\n\n")
	}

	return b.String()
}
