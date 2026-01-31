package compact

import (
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

const (
	// BaseChunkRatio is the default fraction of context window used per compaction chunk.
	BaseChunkRatio = 0.4

	// MinChunkRatio is the minimum chunk ratio, even for very large messages.
	MinChunkRatio = 0.15

	// SafetyMargin multiplier applied to token estimates to avoid edge-case overflows.
	SafetyMargin = 1.2

	// DefaultReserveTokensFloor is the minimum tokens reserved for the system prompt
	// and response generation.
	DefaultReserveTokensFloor = 6000

	// DefaultSoftThreshold is the token headroom before triggering compaction.
	DefaultSoftThreshold = 4000
)

// EstimateTokens estimates token count for text using the ~4 characters per token heuristic.
func EstimateTokens(text string) int {
	return (len(text) + 3) / 4
}

// EstimateMessagesTokens estimates the total token count across all messages,
// accounting for both role labels and content.
func EstimateMessagesTokens(msgs []types.LLMMsg) int {
	total := 0
	for _, m := range msgs {
		// Each message has overhead for the role label (~4 tokens) plus content tokens.
		total += 4 + EstimateTokens(m.Content)
	}
	return total
}

// ComputeAdaptiveChunkRatio computes a chunk ratio based on the average message size
// relative to the context window. If the average message exceeds 10% of the context
// window, the ratio is reduced proportionally down to MinChunkRatio.
func ComputeAdaptiveChunkRatio(msgs []types.LLMMsg, contextWindow int) float64 {
	if len(msgs) == 0 || contextWindow <= 0 {
		return BaseChunkRatio
	}

	totalTokens := EstimateMessagesTokens(msgs)
	avgTokens := float64(totalTokens) / float64(len(msgs))
	threshold := float64(contextWindow) * 0.10

	if avgTokens <= threshold {
		return BaseChunkRatio
	}

	// Scale down linearly: as avgTokens grows beyond threshold,
	// ratio shrinks from BaseChunkRatio toward MinChunkRatio.
	scale := threshold / avgTokens
	ratio := BaseChunkRatio * scale
	if ratio < MinChunkRatio {
		ratio = MinChunkRatio
	}
	return ratio
}

// IsOversizedForSummary checks if a single message is too large to summarize
// meaningfully, defined as exceeding 50% of the context window.
func IsOversizedForSummary(msg types.LLMMsg, contextWindow int) bool {
	if contextWindow <= 0 {
		return false
	}
	tokens := EstimateTokens(msg.Content)
	return float64(tokens) > float64(contextWindow)*0.5
}

// SplitMessagesByTokenShare splits messages into N roughly equal groups by token count.
// Each group aims to hold approximately totalTokens/parts tokens.
func SplitMessagesByTokenShare(msgs []types.LLMMsg, parts int) [][]types.LLMMsg {
	if parts <= 0 {
		parts = 1
	}
	if len(msgs) == 0 {
		return nil
	}
	if parts >= len(msgs) {
		// One message per group.
		result := make([][]types.LLMMsg, 0, len(msgs))
		for _, m := range msgs {
			result = append(result, []types.LLMMsg{m})
		}
		return result
	}

	totalTokens := EstimateMessagesTokens(msgs)
	targetPerPart := totalTokens / parts
	if targetPerPart <= 0 {
		targetPerPart = 1
	}

	result := make([][]types.LLMMsg, 0, parts)
	var current []types.LLMMsg
	currentTokens := 0

	for _, m := range msgs {
		msgTokens := 4 + EstimateTokens(m.Content)
		current = append(current, m)
		currentTokens += msgTokens

		// Start a new group when we've accumulated enough tokens,
		// but ensure we don't create more groups than requested.
		if currentTokens >= targetPerPart && len(result) < parts-1 {
			result = append(result, current)
			current = nil
			currentTokens = 0
		}
	}

	// Append any remaining messages to the last group.
	if len(current) > 0 {
		result = append(result, current)
	}

	return result
}

// ChunkMessagesByMaxTokens creates chunks where each chunk contains at most
// maxTokens worth of messages. Individual messages that exceed maxTokens
// are placed alone in their own chunk.
func ChunkMessagesByMaxTokens(msgs []types.LLMMsg, maxTokens int) [][]types.LLMMsg {
	if maxTokens <= 0 {
		maxTokens = 1
	}
	if len(msgs) == 0 {
		return nil
	}

	result := make([][]types.LLMMsg, 0)
	var current []types.LLMMsg
	currentTokens := 0

	for _, m := range msgs {
		msgTokens := 4 + EstimateTokens(m.Content)

		// If adding this message would exceed the limit and we already have messages
		// in the current chunk, finalize the current chunk first.
		if currentTokens+msgTokens > maxTokens && len(current) > 0 {
			result = append(result, current)
			current = nil
			currentTokens = 0
		}

		current = append(current, m)
		currentTokens += msgTokens

		// If a single message exceeds maxTokens, it sits alone in its chunk.
		if msgTokens > maxTokens {
			result = append(result, current)
			current = nil
			currentTokens = 0
		}
	}

	if len(current) > 0 {
		result = append(result, current)
	}

	return result
}

// PruneResult is the result of history pruning, containing the kept messages
// and statistics about what was dropped.
type PruneResult struct {
	// Messages is the pruned message list that fits within budget.
	Messages []types.LLMMsg

	// DroppedMessages contains messages that were removed to fit the budget.
	DroppedMessages []types.LLMMsg

	// DroppedChunks is the number of contiguous message groups dropped.
	DroppedChunks int

	// DroppedTokens is the estimated token count of all dropped messages.
	DroppedTokens int

	// KeptTokens is the estimated token count of retained messages.
	KeptTokens int

	// BudgetTokens is the token budget that was targeted.
	BudgetTokens int
}

// PruneHistoryForContextShare prunes old messages from the front of the history
// to fit within a token budget defined by maxContextTokens * maxHistoryShare.
// Messages are dropped from oldest first until the remaining messages fit.
func PruneHistoryForContextShare(msgs []types.LLMMsg, maxContextTokens int, maxHistoryShare float64) PruneResult {
	if maxHistoryShare <= 0 {
		maxHistoryShare = 1.0
	}
	if maxHistoryShare > 1.0 {
		maxHistoryShare = 1.0
	}

	budgetTokens := int(float64(maxContextTokens) * maxHistoryShare)
	totalTokens := EstimateMessagesTokens(msgs)

	// Apply safety margin to the estimate.
	adjustedTotal := int(float64(totalTokens) * SafetyMargin)

	// If already within budget, return everything.
	if adjustedTotal <= budgetTokens {
		return PruneResult{
			Messages:     msgs,
			BudgetTokens: budgetTokens,
			KeptTokens:   totalTokens,
		}
	}

	// Drop messages from the front (oldest) until we fit.
	dropped := make([]types.LLMMsg, 0)
	droppedTokens := 0
	remaining := make([]types.LLMMsg, len(msgs))
	copy(remaining, msgs)

	for len(remaining) > 0 {
		remainingTokens := EstimateMessagesTokens(remaining)
		adjustedRemaining := int(float64(remainingTokens) * SafetyMargin)
		if adjustedRemaining <= budgetTokens {
			break
		}

		// Drop the oldest message.
		msg := remaining[0]
		remaining = remaining[1:]
		dropped = append(dropped, msg)
		droppedTokens += 4 + EstimateTokens(msg.Content)
	}

	keptTokens := EstimateMessagesTokens(remaining)

	return PruneResult{
		Messages:        remaining,
		DroppedMessages: dropped,
		DroppedChunks:   1, // All dropped messages form one contiguous block from the front.
		DroppedTokens:   droppedTokens,
		KeptTokens:      keptTokens,
		BudgetTokens:    budgetTokens,
	}
}
