package command

import (
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/skill"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

func TestParse_SlashCommand(t *testing.T) {
	s := NewService()
	cmd, args, isCmd := s.Parse("/help")
	if !isCmd {
		t.Fatal("expected command")
	}
	if cmd != "/help" {
		t.Errorf("cmd = %q; want %q", cmd, "/help")
	}
	if args != "" {
		t.Errorf("args = %q; want empty", args)
	}
}

func TestParse_CommandWithArgs(t *testing.T) {
	s := NewService()
	cmd, args, isCmd := s.Parse("/model gpt-4")
	if !isCmd {
		t.Fatal("expected command")
	}
	if cmd != "/model" {
		t.Errorf("cmd = %q; want %q", cmd, "/model")
	}
	if args != "gpt-4" {
		t.Errorf("args = %q; want %q", args, "gpt-4")
	}
}

func TestParse_NotCommand(t *testing.T) {
	s := NewService()
	_, _, isCmd := s.Parse("Hello world")
	if isCmd {
		t.Error("should not be a command")
	}
}

func TestIsSkillCommand_MatchesSkill(t *testing.T) {
	s := NewService()
	s.SetSkills([]*skill.Skill{
		{Name: "weather", Ready: true, UserInvocable: true, Content: "Weather skill body"},
	})

	matched, ok := s.IsSkillCommand("/weather")
	if !ok {
		t.Fatal("expected skill command match")
	}
	if matched.Name != "weather" {
		t.Errorf("Name = %q; want %q", matched.Name, "weather")
	}
}

func TestIsSkillCommand_NoMatch(t *testing.T) {
	s := NewService()
	s.SetSkills([]*skill.Skill{
		{Name: "weather", Ready: true, UserInvocable: true},
	})

	_, ok := s.IsSkillCommand("/unknown")
	if ok {
		t.Error("should not match unknown command")
	}
}

func TestIsSkillCommand_NoSkills(t *testing.T) {
	s := NewService()
	_, ok := s.IsSkillCommand("/weather")
	if ok {
		t.Error("should not match with no skills")
	}
}

func TestExecute_HelpIncludesSkillCommands(t *testing.T) {
	s := NewService()
	s.SetSkills([]*skill.Skill{
		{Name: "weather", Description: "Get weather", Emoji: "W", Ready: true, UserInvocable: true},
		{Name: "deploy", Description: "Deploy app", Ready: true, UserInvocable: true},
	})

	agent := &types.Agent{Name: "test", Model: "claude"}
	result := s.Execute("/help", "", agent)

	if !strings.Contains(result, "Skill commands") {
		t.Error("help should include 'Skill commands' section")
	}
	if !strings.Contains(result, "/weather") {
		t.Error("help should include /weather skill command")
	}
	if !strings.Contains(result, "/deploy") {
		t.Error("help should include /deploy skill command")
	}
}

func TestExecute_HelpWithoutSkills(t *testing.T) {
	s := NewService()
	agent := &types.Agent{Name: "test", Model: "claude"}
	result := s.Execute("/help", "", agent)

	if strings.Contains(result, "Skill commands") {
		t.Error("help should not include 'Skill commands' when no skills available")
	}
}

func TestParse_MemoryCommand(t *testing.T) {
	s := NewService()
	cmd, args, isCmd := s.Parse("/memory search query")
	if !isCmd {
		t.Fatal("expected command")
	}
	if cmd != "/memory" {
		t.Errorf("cmd = %q; want %q", cmd, "/memory")
	}
	if args != "search query" {
		t.Errorf("args = %q; want %q", args, "search query")
	}
}

func TestExecute_MemoryNoArgs(t *testing.T) {
	s := NewService()
	agent := &types.Agent{Name: "test", Model: "claude"}
	result := s.Execute("/memory", "", agent)
	if !strings.Contains(result, "Usage: /memory") {
		t.Error("expected usage hint for /memory without args")
	}
}

func TestExecute_MemoryWithArgs(t *testing.T) {
	s := NewService()
	agent := &types.Agent{Name: "test", Model: "claude"}
	result := s.Execute("/memory", "test query", agent)
	if !strings.Contains(result, "__memory_search:test query") {
		t.Errorf("expected marker for gateway dispatch, got %q", result)
	}
}

func TestCommands_IncludesMemory(t *testing.T) {
	s := NewService()
	cmds := s.Commands()
	found := false
	for _, c := range cmds {
		if c.Name == "/memory" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Commands() should include /memory")
	}
}

func TestSkillCommands_ReturnsSkillList(t *testing.T) {
	s := NewService()
	s.SetSkills([]*skill.Skill{
		{Name: "weather", Description: "Get weather", Emoji: "W", Ready: true, UserInvocable: true},
		{Name: "internal", Description: "Internal", Ready: true, UserInvocable: false},
	})

	cmds := s.SkillCommands()
	if len(cmds) != 1 {
		t.Fatalf("got %d skill commands; want 1", len(cmds))
	}
	if cmds[0].Name != "/weather" {
		t.Errorf("Name = %q; want %q", cmds[0].Name, "/weather")
	}
}
