package compact

import (
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// PruneConfig configures context pruning behavior for message history.
type PruneConfig struct {
	// SoftTrimMaxChars is the maximum character length before a tool result
	// is eligible for soft trimming. Messages shorter than this are left intact.
	SoftTrimMaxChars int

	// SoftTrimHeadChars is the number of characters to keep from the beginning
	// of a soft-trimmed message.
	SoftTrimHeadChars int

	// SoftTrimTailChars is the number of characters to keep from the end
	// of a soft-trimmed message.
	SoftTrimTailChars int

	// HardClearEnabled controls whether old tool results are replaced entirely
	// with a placeholder after soft trimming is exhausted.
	HardClearEnabled bool

	// HardClearPlaceholder is the text that replaces hard-cleared content.
	HardClearPlaceholder string

	// KeepLastAssistants is the number of most recent assistant messages
	// that are protected from any pruning.
	KeepLastAssistants int

	// CacheTTLSeconds is the time-to-live for pruning cache entries.
	// Messages older than this are eligible for more aggressive pruning.
	CacheTTLSeconds int

	// MinPrunableChars is the minimum total character count across all messages
	// before pruning is activated. Below this threshold, no pruning occurs.
	MinPrunableChars int
}

// DefaultPruneConfig returns a PruneConfig with sensible defaults.
func DefaultPruneConfig() PruneConfig {
	return PruneConfig{
		SoftTrimMaxChars:     4000,
		SoftTrimHeadChars:    1500,
		SoftTrimTailChars:    1500,
		HardClearEnabled:     true,
		HardClearPlaceholder: "[Old tool result content cleared]",
		KeepLastAssistants:   3,
		CacheTTLSeconds:      600,
		MinPrunableChars:     50000,
	}
}

// SoftTrim truncates text that exceeds SoftTrimMaxChars by keeping the head
// and tail portions with an ellipsis separator in between.
// If the text is shorter than SoftTrimMaxChars, it is returned unchanged.
func SoftTrim(text string, cfg PruneConfig) string {
	maxChars := cfg.SoftTrimMaxChars
	if maxChars <= 0 {
		maxChars = 4000
	}

	if len(text) <= maxChars {
		return text
	}

	headChars := cfg.SoftTrimHeadChars
	if headChars <= 0 {
		headChars = 1500
	}

	tailChars := cfg.SoftTrimTailChars
	if tailChars <= 0 {
		tailChars = 1500
	}

	// Ensure head + tail doesn't exceed the text length.
	if headChars+tailChars >= len(text) {
		return text
	}

	head := text[:headChars]
	tail := text[len(text)-tailChars:]

	return head + "\n...\n" + tail
}

// HardClear returns the hard-clear placeholder string from the config.
func HardClear(cfg PruneConfig) string {
	placeholder := cfg.HardClearPlaceholder
	if placeholder == "" {
		placeholder = "[Old tool result content cleared]"
	}
	return placeholder
}

// PruneMessages applies soft trim and hard clear to the message history to reduce
// token usage. It protects the last N assistant messages from modification and
// only prunes tool-result content (messages with role "tool" or content that
// appears to be tool output in assistant messages are left to the caller).
//
// The pruning strategy is:
//  1. If total characters are below MinPrunableChars, return messages unchanged.
//  2. Identify protected messages (last KeepLastAssistants assistant messages).
//  3. For eligible non-protected messages, apply soft trim to long content.
//  4. If still over budget after soft trim, apply hard clear to oldest eligible messages.
func PruneMessages(msgs []types.LLMMsg, totalTokens, contextWindow int, cfg PruneConfig) []types.LLMMsg {
	if len(msgs) == 0 {
		return msgs
	}

	// Check if total content is below the pruning threshold.
	totalChars := 0
	for _, m := range msgs {
		totalChars += len(m.Content)
	}

	minPrunable := cfg.MinPrunableChars
	if minPrunable <= 0 {
		minPrunable = 50000
	}

	if totalChars < minPrunable {
		return msgs
	}

	// Build the protected set: indices of the last N assistant messages.
	protectedSet := make(map[int]bool)
	keepLast := cfg.KeepLastAssistants
	if keepLast <= 0 {
		keepLast = 3
	}

	assistantCount := 0
	for i := len(msgs) - 1; i >= 0 && assistantCount < keepLast; i-- {
		if msgs[i].Role == types.RoleAssistant {
			protectedSet[i] = true
			assistantCount++
		}
	}

	// Also protect system messages entirely.
	for i, m := range msgs {
		if m.Role == types.RoleSystem {
			protectedSet[i] = true
		}
	}

	// Create a copy of the messages to avoid mutating the input.
	result := make([]types.LLMMsg, len(msgs))
	copy(result, msgs)

	// Phase 1: Soft trim eligible messages from oldest to newest.
	for i := range result {
		if protectedSet[i] {
			continue
		}

		maxChars := cfg.SoftTrimMaxChars
		if maxChars <= 0 {
			maxChars = 4000
		}

		if len(result[i].Content) > maxChars {
			result[i] = types.LLMMsg{
				Role:    result[i].Role,
				Content: SoftTrim(result[i].Content, cfg),
			}
		}
	}

	// Check if soft trim was sufficient.
	currentTokens := EstimateMessagesTokens(result)
	budget := contextWindow - DefaultReserveTokensFloor
	if budget <= 0 {
		budget = contextWindow
	}

	if currentTokens <= budget {
		return result
	}

	// Phase 2: Hard clear eligible messages from oldest to newest.
	if !cfg.HardClearEnabled {
		return result
	}

	placeholder := HardClear(cfg)
	for i := range result {
		if protectedSet[i] {
			continue
		}

		// Only hard-clear messages that still have substantial content.
		if len(result[i].Content) > len(placeholder) {
			result[i] = types.LLMMsg{
				Role:    result[i].Role,
				Content: placeholder,
			}
		}

		// Re-check after each hard clear to stop as soon as we're within budget.
		currentTokens = EstimateMessagesTokens(result)
		if currentTokens <= budget {
			break
		}
	}

	return result
}
