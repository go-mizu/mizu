package gateway

import (
	"path/filepath"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/skill"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// generatePromptReport creates a SystemPromptReport from the build result.
func generatePromptReport(result *BuildResult, params *SystemPromptParams) *types.SystemPromptReport {
	promptChars := len(result.Prompt)
	contextChars := result.ContextChars
	nonContextChars := promptChars - contextChars

	var skillNames []string
	for _, name := range result.SkillNames {
		skillNames = append(skillNames, name)
	}
	if skillNames == nil {
		skillNames = []string{}
	}

	toolNames := params.ToolNames
	if toolNames == nil {
		toolNames = []string{}
	}

	return &types.SystemPromptReport{
		Source:            "run",
		GeneratedAt:       time.Now().UnixMilli(),
		SessionID:         params.SessionID,
		Provider:          "anthropic",
		Model:             params.Agent.Model,
		WorkspaceDir:      params.WorkspaceDir,
		BootstrapMaxChars: 20000,
		SystemPrompt: types.SystemPromptStats{
			Chars:               promptChars,
			ProjectContextChars: contextChars,
			NonProjectChars:     nonContextChars,
		},
		InjectedFiles: result.InjectedFiles,
		Skills: types.SkillsReportStats{
			Count: len(skillNames),
			Chars: len(params.SkillsPrompt),
			Names: skillNames,
		},
		Tools: types.ToolsReportStats{
			Count: len(toolNames),
			Names: toolNames,
		},
	}
}

// buildSkillsSnapshot creates a SkillsSnapshot from loaded skills.
func buildSkillsSnapshot(skills []*skill.Skill, skillsPrompt string) *types.SkillsSnapshot {
	items := make([]types.SkillsSnapshotItem, 0, len(skills))
	for _, s := range skills {
		items = append(items, types.SkillsSnapshotItem{
			Name:        s.Name,
			Description: s.Description,
			Location:    filepath.Join(s.Dir, "SKILL.md"),
			Ready:       s.Ready,
			Source:      s.Source,
		})
	}
	return &types.SkillsSnapshot{
		Prompt:  skillsPrompt,
		Skills:  items,
		Version: 0,
	}
}
