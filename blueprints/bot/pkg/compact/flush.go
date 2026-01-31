package compact

import "fmt"

// FlushConfig configures the memory flush trigger that runs before compaction.
// When the conversation approaches the context window limit, a flush prompt
// is injected to give the agent a chance to persist important memories.
type FlushConfig struct {
	// Enabled controls whether memory flush is active.
	Enabled bool

	// ReserveTokensFloor is the minimum tokens reserved for system prompt
	// and response generation. Defaults to DefaultReserveTokensFloor.
	ReserveTokensFloor int

	// SoftThresholdTokens is the additional headroom before triggering flush.
	// Defaults to DefaultSoftThreshold.
	SoftThresholdTokens int

	// SystemPrompt is an optional system-level instruction prepended to the flush prompt.
	SystemPrompt string

	// Prompt is the user-facing flush prompt sent to the agent.
	Prompt string
}

const defaultFlushPrompt = "Pre-compaction memory flush. Store durable memories now (use memory/YYYY-MM-DD.md). If nothing to store, reply NO_REPLY."

// DefaultFlushConfig returns a FlushConfig with sensible defaults.
func DefaultFlushConfig() FlushConfig {
	return FlushConfig{
		Enabled:             true,
		ReserveTokensFloor:  DefaultReserveTokensFloor,
		SoftThresholdTokens: DefaultSoftThreshold,
		Prompt:              defaultFlushPrompt,
	}
}

// ShouldRunMemoryFlush checks if a memory flush should trigger based on token usage.
// It triggers when the total tokens used have consumed enough of the context window
// that only the reserve floor and soft threshold remain.
//
// Specifically: totalTokens >= (contextWindow - reserve - softThreshold)
func ShouldRunMemoryFlush(totalTokens, contextWindow int, cfg FlushConfig) bool {
	if !cfg.Enabled {
		return false
	}
	if contextWindow <= 0 {
		return false
	}

	reserve := cfg.ReserveTokensFloor
	if reserve <= 0 {
		reserve = DefaultReserveTokensFloor
	}

	soft := cfg.SoftThresholdTokens
	if soft <= 0 {
		soft = DefaultSoftThreshold
	}

	threshold := contextWindow - reserve - soft
	if threshold <= 0 {
		// Context window too small; always flush.
		return true
	}

	return totalTokens >= threshold
}

// BuildFlushPrompt returns the assembled memory flush prompt for the agent.
// If a system prompt is configured, it is prepended on its own line.
func BuildFlushPrompt(cfg FlushConfig) string {
	prompt := cfg.Prompt
	if prompt == "" {
		prompt = defaultFlushPrompt
	}

	if cfg.SystemPrompt != "" {
		return fmt.Sprintf("%s\n%s", cfg.SystemPrompt, prompt)
	}

	return prompt
}
