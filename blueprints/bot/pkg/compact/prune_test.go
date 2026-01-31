package compact

import (
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// ---------------------------------------------------------------------------
// DefaultPruneConfig
// ---------------------------------------------------------------------------

func TestDefaultPruneConfig(t *testing.T) {
	cfg := DefaultPruneConfig()

	tests := []struct {
		name string
		got  int
		want int
	}{
		{"SoftTrimMaxChars", cfg.SoftTrimMaxChars, 4000},
		{"SoftTrimHeadChars", cfg.SoftTrimHeadChars, 1500},
		{"SoftTrimTailChars", cfg.SoftTrimTailChars, 1500},
		{"KeepLastAssistants", cfg.KeepLastAssistants, 3},
		{"CacheTTLSeconds", cfg.CacheTTLSeconds, 600},
		{"MinPrunableChars", cfg.MinPrunableChars, 50000},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
		}
	}

	if !cfg.HardClearEnabled {
		t.Error("HardClearEnabled should be true")
	}

	if cfg.HardClearPlaceholder != "[Old tool result content cleared]" {
		t.Errorf("HardClearPlaceholder = %q, want '[Old tool result content cleared]'", cfg.HardClearPlaceholder)
	}
}

// ---------------------------------------------------------------------------
// SoftTrim
// ---------------------------------------------------------------------------

func TestSoftTrim_ShortText(t *testing.T) {
	cfg := DefaultPruneConfig()
	text := "short text"
	got := SoftTrim(text, cfg)
	if got != text {
		t.Errorf("short text should be returned unchanged, got: %q", got)
	}
}

func TestSoftTrim_ExactlyAtLimit(t *testing.T) {
	cfg := DefaultPruneConfig()
	text := strings.Repeat("a", 4000) // exactly SoftTrimMaxChars
	got := SoftTrim(text, cfg)
	if got != text {
		t.Error("text at exactly SoftTrimMaxChars should be returned unchanged")
	}
}

func TestSoftTrim_LongText(t *testing.T) {
	cfg := DefaultPruneConfig()
	text := strings.Repeat("a", 10000) // well over 4000
	got := SoftTrim(text, cfg)

	if got == text {
		t.Error("long text should be trimmed")
	}

	// Should contain head (1500 a's) + "\n...\n" + tail (1500 a's)
	expectedLen := 1500 + len("\n...\n") + 1500
	if len(got) != expectedLen {
		t.Errorf("trimmed len = %d, want %d", len(got), expectedLen)
	}

	if !strings.Contains(got, "\n...\n") {
		t.Error("trimmed text should contain ellipsis separator")
	}

	head := got[:1500]
	if head != strings.Repeat("a", 1500) {
		t.Error("head portion should be the first 1500 chars")
	}

	tail := got[len(got)-1500:]
	if tail != strings.Repeat("a", 1500) {
		t.Error("tail portion should be the last 1500 chars")
	}
}

func TestSoftTrim_HeadPlusTailExceedsLength(t *testing.T) {
	// Text is > SoftTrimMaxChars but head+tail >= len(text).
	// Use custom config: maxChars=100, head=80, tail=80 -> head+tail=160.
	customCfg := PruneConfig{
		SoftTrimMaxChars:  100,
		SoftTrimHeadChars: 80,
		SoftTrimTailChars: 80,
	}
	text := strings.Repeat("b", 150) // > 100 but head+tail (160) >= 150
	got := SoftTrim(text, customCfg)
	if got != text {
		t.Error("when head+tail >= text length, text should be returned unchanged")
	}
}

func TestSoftTrim_DistinctContent(t *testing.T) {
	cfg := DefaultPruneConfig()
	// Build a text with distinct head and tail to verify correct slicing.
	head := strings.Repeat("H", 2000)
	middle := strings.Repeat("M", 6000)
	tail := strings.Repeat("T", 2000)
	text := head + middle + tail // 10000 chars total

	got := SoftTrim(text, cfg)

	// Head portion: first 1500 chars → all "H"
	gotHead := got[:1500]
	if gotHead != strings.Repeat("H", 1500) {
		t.Error("head should be first 1500 chars of original")
	}

	// Tail portion: last 1500 chars → all "T"
	gotTail := got[len(got)-1500:]
	if gotTail != strings.Repeat("T", 1500) {
		t.Error("tail should be last 1500 chars of original")
	}
}

// ---------------------------------------------------------------------------
// HardClear
// ---------------------------------------------------------------------------

func TestHardClear_DefaultPlaceholder(t *testing.T) {
	cfg := DefaultPruneConfig()
	got := HardClear(cfg)
	if got != "[Old tool result content cleared]" {
		t.Errorf("HardClear = %q, want '[Old tool result content cleared]'", got)
	}
}

func TestHardClear_CustomPlaceholder(t *testing.T) {
	cfg := PruneConfig{HardClearPlaceholder: "[CLEARED]"}
	got := HardClear(cfg)
	if got != "[CLEARED]" {
		t.Errorf("HardClear = %q, want '[CLEARED]'", got)
	}
}

func TestHardClear_EmptyPlaceholderUsesDefault(t *testing.T) {
	cfg := PruneConfig{HardClearPlaceholder: ""}
	got := HardClear(cfg)
	if got != "[Old tool result content cleared]" {
		t.Errorf("empty placeholder should use default, got: %q", got)
	}
}

// ---------------------------------------------------------------------------
// PruneMessages
// ---------------------------------------------------------------------------

func TestPruneMessages_BelowMinPrunableChars(t *testing.T) {
	cfg := DefaultPruneConfig()
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: "short message"},
		{Role: types.RoleAssistant, Content: "short reply"},
	}
	result := PruneMessages(msgs, 100, 100000, cfg)
	if len(result) != 2 {
		t.Fatalf("below MinPrunableChars: got %d messages, want 2", len(result))
	}
	if result[0].Content != "short message" || result[1].Content != "short reply" {
		t.Error("messages should be unchanged when below MinPrunableChars")
	}
}

func TestPruneMessages_Empty(t *testing.T) {
	cfg := DefaultPruneConfig()
	result := PruneMessages(nil, 0, 100000, cfg)
	if len(result) != 0 {
		t.Errorf("empty input: got %d messages, want 0", len(result))
	}
}

func TestPruneMessages_ProtectsSystemMessages(t *testing.T) {
	cfg := DefaultPruneConfig()

	systemContent := strings.Repeat("S", 6000) // > SoftTrimMaxChars
	longContent := strings.Repeat("X", 6000)

	// Need total chars > MinPrunableChars (50000). system(6000) + 8*6000 = 54000.
	msgs := []types.LLMMsg{
		{Role: types.RoleSystem, Content: systemContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleAssistant, Content: "a1"},
		{Role: types.RoleAssistant, Content: "a2"},
		{Role: types.RoleAssistant, Content: "a3"},
	}
	// Total chars > 50000 so pruning activates.

	result := PruneMessages(msgs, 50000, 100000, cfg)

	// System message must remain unchanged.
	if result[0].Role != types.RoleSystem {
		t.Error("system message should remain at index 0")
	}
	if result[0].Content != systemContent {
		t.Error("system message content should be unchanged (protected)")
	}
}

func TestPruneMessages_ProtectsLastAssistants(t *testing.T) {
	cfg := DefaultPruneConfig()

	longContent := strings.Repeat("Z", 6000)
	a1 := "assistant reply one"
	a2 := "assistant reply two"
	a3 := "assistant reply three"

	// Need total chars > MinPrunableChars (50000). 9 * 6000 = 54000.
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleAssistant, Content: a1},
		{Role: types.RoleAssistant, Content: a2},
		{Role: types.RoleAssistant, Content: a3},
	}
	// Total > 50000 chars.

	result := PruneMessages(msgs, 50000, 100000, cfg)

	// Last 3 assistant messages should be untouched.
	assistantContents := make([]string, 0)
	for _, m := range result {
		if m.Role == types.RoleAssistant {
			assistantContents = append(assistantContents, m.Content)
		}
	}

	found := map[string]bool{}
	for _, c := range assistantContents {
		found[c] = true
	}
	for _, want := range []string{a1, a2, a3} {
		if !found[want] {
			t.Errorf("protected assistant content %q should be preserved", want)
		}
	}
}

func TestPruneMessages_Phase1SoftTrim(t *testing.T) {
	cfg := DefaultPruneConfig()

	longContent := strings.Repeat("L", 6000) // > SoftTrimMaxChars

	// Need total chars > MinPrunableChars (50000). 9 * 6000 = 54000. Plus assistant msgs.
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleAssistant, Content: "done1"},
		{Role: types.RoleAssistant, Content: "done2"},
		{Role: types.RoleAssistant, Content: "done3"},
	}

	// Use a very large context window so soft trim alone is sufficient.
	result := PruneMessages(msgs, 50000, 1000000, cfg)

	// Non-protected user messages with long content should be soft trimmed.
	for _, m := range result {
		if m.Role == types.RoleUser && strings.Contains(m.Content, "\n...\n") {
			return // At least one message was soft trimmed.
		}
	}
	t.Error("expected at least one user message to be soft trimmed")
}

func TestPruneMessages_Phase2HardClear(t *testing.T) {
	cfg := DefaultPruneConfig()

	longContent := strings.Repeat("C", 6000)

	// Need total chars > MinPrunableChars (50000). 9 * 6000 = 54000.
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleAssistant, Content: "final1"},
		{Role: types.RoleAssistant, Content: "final2"},
		{Role: types.RoleAssistant, Content: "final3"},
	}

	// After soft trim, each user msg ~ 3005 chars ~ 756 tokens.
	// 9 user * 756 + 3 assistant * ~5 = ~6819 tokens total.
	// Budget = contextWindow - 6000. Need budget < 6819.
	// contextWindow = 8000 -> budget = 2000. 6819 > 2000 -> hard clear triggered.
	result := PruneMessages(msgs, 50000, 8000, cfg)

	placeholder := cfg.HardClearPlaceholder
	foundHardCleared := false
	for _, m := range result {
		if m.Content == placeholder {
			foundHardCleared = true
			break
		}
	}

	if !foundHardCleared {
		t.Error("expected at least one message to be hard cleared")
	}
}

func TestPruneMessages_HardClearStopsWhenWithinBudget(t *testing.T) {
	cfg := DefaultPruneConfig()

	longContent := strings.Repeat("D", 6000)
	placeholder := cfg.HardClearPlaceholder

	// Need total chars > MinPrunableChars (50000). 9 * 6000 = 54000.
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: longContent},   // oldest, will be hard cleared first
		{Role: types.RoleUser, Content: longContent},   // second oldest
		{Role: types.RoleUser, Content: longContent},   // third
		{Role: types.RoleUser, Content: longContent},   // fourth
		{Role: types.RoleUser, Content: longContent},   // fifth
		{Role: types.RoleUser, Content: longContent},   // sixth
		{Role: types.RoleUser, Content: longContent},   // seventh
		{Role: types.RoleUser, Content: longContent},   // eighth
		{Role: types.RoleUser, Content: longContent},   // ninth
		{Role: types.RoleAssistant, Content: "reply1"}, // protected
		{Role: types.RoleAssistant, Content: "reply2"}, // protected
		{Role: types.RoleAssistant, Content: "reply3"}, // protected
	}

	// After soft trim, each user msg ~= 1500+5+1500 = 3005 chars ~ 755 tokens + 4 = 759 tokens per msg
	// 9 user msgs * 759 + 3 assistant msgs * ~5 = ~6846
	// Budget = contextWindow - 6000. Set contextWindow = 10000 -> budget = 4000.
	result := PruneMessages(msgs, 50000, 10000, cfg)

	hardClearedCount := 0
	nonClearedUserCount := 0
	for _, m := range result {
		if m.Role == types.RoleUser {
			if m.Content == placeholder {
				hardClearedCount++
			} else {
				nonClearedUserCount++
			}
		}
	}

	// Some should be hard cleared, but not necessarily all (stops when within budget).
	if hardClearedCount == 0 {
		t.Error("expected at least one hard cleared message")
	}
	// Total messages should remain the same (hard clear replaces content, not removes).
	if len(result) != len(msgs) {
		t.Errorf("message count changed: %d -> %d", len(msgs), len(result))
	}
}

func TestPruneMessages_HardClearDisabled(t *testing.T) {
	cfg := DefaultPruneConfig()
	cfg.HardClearEnabled = false

	longContent := strings.Repeat("E", 6000)

	// Need total chars > MinPrunableChars (50000). 9 * 6000 = 54000.
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleAssistant, Content: "end1"},
		{Role: types.RoleAssistant, Content: "end2"},
		{Role: types.RoleAssistant, Content: "end3"},
	}

	// Very small context window that would normally trigger hard clear.
	result := PruneMessages(msgs, 50000, 10000, cfg)

	placeholder := "[Old tool result content cleared]"
	for _, m := range result {
		if m.Content == placeholder {
			t.Error("HardClearEnabled=false should skip phase 2; found hard cleared message")
		}
	}
}

func TestPruneMessages_DoesNotMutateInput(t *testing.T) {
	cfg := DefaultPruneConfig()

	longContent := strings.Repeat("F", 6000)
	original := longContent

	// Need total chars > MinPrunableChars (50000). 9 * 6000 = 54000.
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleAssistant, Content: "x1"},
		{Role: types.RoleAssistant, Content: "x2"},
		{Role: types.RoleAssistant, Content: "x3"},
	}

	_ = PruneMessages(msgs, 50000, 100000, cfg)

	// Original messages should not be mutated.
	for i, m := range msgs {
		if m.Role == types.RoleUser && m.Content != original {
			t.Errorf("input msg[%d] was mutated", i)
		}
	}
}

func TestPruneMessages_OlderAssistantNotProtected(t *testing.T) {
	cfg := DefaultPruneConfig()

	longContent := strings.Repeat("G", 6000)

	// 4 assistant messages: only last 3 protected. The first one is eligible.
	// Need total chars > MinPrunableChars (50000). 9 * 6000 = 54000.
	msgs := []types.LLMMsg{
		{Role: types.RoleAssistant, Content: longContent}, // NOT protected (4th from end)
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleUser, Content: longContent},
		{Role: types.RoleAssistant, Content: "p1"},
		{Role: types.RoleAssistant, Content: "p2"},
		{Role: types.RoleAssistant, Content: "p3"},
	}

	result := PruneMessages(msgs, 50000, 1000000, cfg)

	// The first assistant message should have been soft trimmed since it's not protected.
	if result[0].Content == longContent {
		t.Error("older assistant message (not in last 3) should be eligible for soft trim")
	}
}
