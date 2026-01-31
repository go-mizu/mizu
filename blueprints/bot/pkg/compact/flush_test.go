package compact

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// DefaultFlushConfig
// ---------------------------------------------------------------------------

func TestDefaultFlushConfig_Enabled(t *testing.T) {
	cfg := DefaultFlushConfig()
	if !cfg.Enabled {
		t.Error("DefaultFlushConfig().Enabled should be true")
	}
}

func TestDefaultFlushConfig_ReserveTokensFloor(t *testing.T) {
	cfg := DefaultFlushConfig()
	if cfg.ReserveTokensFloor != 6000 {
		t.Errorf("ReserveTokensFloor = %d, want 6000", cfg.ReserveTokensFloor)
	}
}

func TestDefaultFlushConfig_SoftThresholdTokens(t *testing.T) {
	cfg := DefaultFlushConfig()
	if cfg.SoftThresholdTokens != 4000 {
		t.Errorf("SoftThresholdTokens = %d, want 4000", cfg.SoftThresholdTokens)
	}
}

func TestDefaultFlushConfig_PromptContainsMemoryFlush(t *testing.T) {
	cfg := DefaultFlushConfig()
	if !strings.Contains(cfg.Prompt, "memory flush") {
		t.Errorf("default Prompt should contain 'memory flush', got: %s", cfg.Prompt)
	}
}

func TestDefaultFlushConfig_NoSystemPrompt(t *testing.T) {
	cfg := DefaultFlushConfig()
	if cfg.SystemPrompt != "" {
		t.Errorf("default SystemPrompt should be empty, got: %s", cfg.SystemPrompt)
	}
}

// ---------------------------------------------------------------------------
// ShouldRunMemoryFlush
// ---------------------------------------------------------------------------

func TestShouldRunMemoryFlush_Disabled(t *testing.T) {
	cfg := FlushConfig{Enabled: false, ReserveTokensFloor: 6000, SoftThresholdTokens: 4000}
	if ShouldRunMemoryFlush(999999, 100000, cfg) {
		t.Error("disabled config should return false")
	}
}

func TestShouldRunMemoryFlush_ZeroContextWindow(t *testing.T) {
	cfg := DefaultFlushConfig()
	if ShouldRunMemoryFlush(50000, 0, cfg) {
		t.Error("zero context window should return false")
	}
}

func TestShouldRunMemoryFlush_BelowThreshold(t *testing.T) {
	cfg := DefaultFlushConfig()
	// threshold = 100000 - 6000 - 4000 = 90000
	// totalTokens = 89999 < 90000 → false
	if ShouldRunMemoryFlush(89999, 100000, cfg) {
		t.Error("totalTokens 89999 < threshold 90000 should return false")
	}
}

func TestShouldRunMemoryFlush_AtThreshold(t *testing.T) {
	cfg := DefaultFlushConfig()
	// threshold = 100000 - 6000 - 4000 = 90000
	// totalTokens = 90000 >= 90000 → true
	if !ShouldRunMemoryFlush(90000, 100000, cfg) {
		t.Error("totalTokens 90000 >= threshold 90000 should return true")
	}
}

func TestShouldRunMemoryFlush_AboveThreshold(t *testing.T) {
	cfg := DefaultFlushConfig()
	// threshold = 100000 - 6000 - 4000 = 90000
	// totalTokens = 95000 > 90000 → true
	if !ShouldRunMemoryFlush(95000, 100000, cfg) {
		t.Error("totalTokens 95000 > threshold 90000 should return true")
	}
}

func TestShouldRunMemoryFlush_ThresholdCalculation(t *testing.T) {
	cfg := FlushConfig{
		Enabled:             true,
		ReserveTokensFloor:  6000,
		SoftThresholdTokens: 4000,
	}
	// contextWindow=100000, reserve=6000, soft=4000 → threshold=90000
	// 89999 → false
	if ShouldRunMemoryFlush(89999, 100000, cfg) {
		t.Error("89999 < 90000 should be false")
	}
	// 90000 → true
	if !ShouldRunMemoryFlush(90000, 100000, cfg) {
		t.Error("90000 >= 90000 should be true")
	}
}

func TestShouldRunMemoryFlush_VerySmallContextWindow(t *testing.T) {
	// Context window smaller than reserve + soft → threshold <= 0 → always true.
	cfg := FlushConfig{
		Enabled:             true,
		ReserveTokensFloor:  6000,
		SoftThresholdTokens: 4000,
	}
	// contextWindow = 5000 → threshold = 5000 - 6000 - 4000 = -5000 ≤ 0 → true
	if !ShouldRunMemoryFlush(0, 5000, cfg) {
		t.Error("very small context window should always trigger flush")
	}
	if !ShouldRunMemoryFlush(1, 5000, cfg) {
		t.Error("very small context window should always trigger flush even with minimal tokens")
	}
}

func TestShouldRunMemoryFlush_DefaultsForZeroReserveSoft(t *testing.T) {
	// When reserve and soft are zero, defaults are used.
	cfg := FlushConfig{
		Enabled:             true,
		ReserveTokensFloor:  0,
		SoftThresholdTokens: 0,
	}
	// Should use defaults: reserve=6000, soft=4000 → threshold=100000-10000=90000
	if ShouldRunMemoryFlush(89999, 100000, cfg) {
		t.Error("should use defaults: 89999 < 90000")
	}
	if !ShouldRunMemoryFlush(90000, 100000, cfg) {
		t.Error("should use defaults: 90000 >= 90000")
	}
}

func TestShouldRunMemoryFlush_CustomReserveSoft(t *testing.T) {
	cfg := FlushConfig{
		Enabled:             true,
		ReserveTokensFloor:  10000,
		SoftThresholdTokens: 5000,
	}
	// threshold = 100000 - 10000 - 5000 = 85000
	if ShouldRunMemoryFlush(84999, 100000, cfg) {
		t.Error("84999 < 85000 should be false")
	}
	if !ShouldRunMemoryFlush(85000, 100000, cfg) {
		t.Error("85000 >= 85000 should be true")
	}
}

// ---------------------------------------------------------------------------
// BuildFlushPrompt
// ---------------------------------------------------------------------------

func TestBuildFlushPrompt_Default(t *testing.T) {
	cfg := DefaultFlushConfig()
	prompt := BuildFlushPrompt(cfg)
	if !strings.Contains(prompt, "Pre-compaction memory flush") {
		t.Errorf("default prompt should contain 'Pre-compaction memory flush', got: %s", prompt)
	}
}

func TestBuildFlushPrompt_WithSystemPrompt(t *testing.T) {
	cfg := DefaultFlushConfig()
	cfg.SystemPrompt = "You are a helpful assistant."

	prompt := BuildFlushPrompt(cfg)
	if !strings.HasPrefix(prompt, "You are a helpful assistant.\n") {
		t.Errorf("prompt should start with system prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "Pre-compaction memory flush") {
		t.Error("prompt should still contain the flush prompt after system prompt")
	}

	// Verify it's system + "\n" + flush prompt.
	parts := strings.SplitN(prompt, "\n", 2)
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts separated by newline, got %d", len(parts))
	}
	if parts[0] != "You are a helpful assistant." {
		t.Errorf("first part = %q, want system prompt", parts[0])
	}
}

func TestBuildFlushPrompt_CustomPrompt(t *testing.T) {
	cfg := FlushConfig{
		Prompt: "Custom flush now!",
	}
	prompt := BuildFlushPrompt(cfg)
	if prompt != "Custom flush now!" {
		t.Errorf("custom prompt = %q, want 'Custom flush now!'", prompt)
	}
}

func TestBuildFlushPrompt_EmptyPromptUsesDefault(t *testing.T) {
	cfg := FlushConfig{
		Prompt: "",
	}
	prompt := BuildFlushPrompt(cfg)
	if !strings.Contains(prompt, "Pre-compaction memory flush") {
		t.Errorf("empty prompt should fall back to default, got: %s", prompt)
	}
}

func TestBuildFlushPrompt_SystemPromptWithCustomPrompt(t *testing.T) {
	cfg := FlushConfig{
		SystemPrompt: "System instruction",
		Prompt:       "Custom flush",
	}
	prompt := BuildFlushPrompt(cfg)
	expected := "System instruction\nCustom flush"
	if prompt != expected {
		t.Errorf("prompt = %q, want %q", prompt, expected)
	}
}

func TestBuildFlushPrompt_SystemPromptWithEmptyPrompt(t *testing.T) {
	cfg := FlushConfig{
		SystemPrompt: "System instruction",
		Prompt:       "",
	}
	prompt := BuildFlushPrompt(cfg)
	if !strings.HasPrefix(prompt, "System instruction\n") {
		t.Errorf("should prepend system prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "Pre-compaction memory flush") {
		t.Error("empty Prompt should fall back to default after system prompt")
	}
}
