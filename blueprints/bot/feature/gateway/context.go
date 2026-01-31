package gateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/memory"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/skill"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/workspace"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// contextBuilder assembles the full system prompt for an LLM request by
// combining the agent's base prompt, workspace bootstrap files, skills,
// and memory search results.
type contextBuilder struct {
	memReg *memoryRegistry
}

func newContextBuilder(memReg *memoryRegistry) *contextBuilder {
	return &contextBuilder{memReg: memReg}
}

// buildSystemPrompt constructs the enriched system prompt for an agent.
// It layers: base prompt → workspace context → skills → memory results.
func (cb *contextBuilder) buildSystemPrompt(ctx context.Context, agent *types.Agent, origin, query string) string {
	var sections []string

	// 1. Agent's base system prompt.
	if agent.SystemPrompt != "" {
		sections = append(sections, agent.SystemPrompt)
	}

	// 2. Workspace bootstrap files (only if agent has a workspace).
	if agent.Workspace != "" {
		if wsSection := cb.buildWorkspaceSection(agent.Workspace, origin); wsSection != "" {
			sections = append(sections, wsSection)
		}
	}

	// 3. Skills from workspace.
	if agent.Workspace != "" {
		if skillsSection := cb.buildSkillsSection(agent.Workspace); skillsSection != "" {
			sections = append(sections, skillsSection)
		}
	}

	// 4. Memory search results relevant to the current query.
	if agent.Workspace != "" && query != "" {
		if memSection := cb.buildMemorySection(ctx, agent.Workspace, query); memSection != "" {
			sections = append(sections, memSection)
		}
	}

	return strings.Join(sections, "\n\n")
}

// buildWorkspaceSection loads bootstrap files and formats them for injection.
func (cb *contextBuilder) buildWorkspaceSection(workspaceDir, origin string) string {
	files, err := workspace.LoadBootstrapFiles(workspaceDir)
	if err != nil {
		return ""
	}

	// Filter based on session origin.
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

// buildSkillsSection loads skills and formats the prompt section.
func (cb *contextBuilder) buildSkillsSection(workspaceDir string) string {
	skills, err := skill.LoadAllSkills(workspaceDir)
	if err != nil || len(skills) == 0 {
		return ""
	}

	// Only include if at least one skill is ready.
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

// buildMemorySection searches the memory index and formats relevant results.
func (cb *contextBuilder) buildMemorySection(ctx context.Context, workspaceDir, query string) string {
	if cb.memReg == nil {
		return ""
	}

	mgr, err := cb.memReg.get(workspaceDir)
	if err != nil || mgr == nil {
		return ""
	}

	results, err := mgr.Search(ctx, query, 0, 0) // use defaults from config
	if err != nil || len(results) == 0 {
		return ""
	}

	return formatMemoryResults(results)
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
