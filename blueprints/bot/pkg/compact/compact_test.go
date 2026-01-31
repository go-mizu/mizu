package compact

import (
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

func TestConstants(t *testing.T) {
	if BaseChunkRatio != 0.4 {
		t.Errorf("BaseChunkRatio = %v, want 0.4", BaseChunkRatio)
	}
	if MinChunkRatio != 0.15 {
		t.Errorf("MinChunkRatio = %v, want 0.15", MinChunkRatio)
	}
	if SafetyMargin != 1.2 {
		t.Errorf("SafetyMargin = %v, want 1.2", SafetyMargin)
	}
	if DefaultReserveTokensFloor != 6000 {
		t.Errorf("DefaultReserveTokensFloor = %v, want 6000", DefaultReserveTokensFloor)
	}
	if DefaultSoftThreshold != 4000 {
		t.Errorf("DefaultSoftThreshold = %v, want 4000", DefaultSoftThreshold)
	}
}

// ---------------------------------------------------------------------------
// EstimateTokens
// ---------------------------------------------------------------------------

func TestEstimateTokens_Empty(t *testing.T) {
	got := EstimateTokens("")
	if got != 0 {
		t.Errorf("EstimateTokens('') = %d, want 0", got)
	}
}

func TestEstimateTokens_Short(t *testing.T) {
	// "hi" is 2 chars → (2+3)/4 = 1
	got := EstimateTokens("hi")
	if got != 1 {
		t.Errorf("EstimateTokens('hi') = %d, want 1", got)
	}
}

func TestEstimateTokens_Hello(t *testing.T) {
	// "hello" is 5 chars → (5+3)/4 = 2
	got := EstimateTokens("hello")
	if got != 2 {
		t.Errorf("EstimateTokens('hello') = %d, want 2", got)
	}
}

func TestEstimateTokens_400Chars(t *testing.T) {
	text := strings.Repeat("a", 400)
	// (400+3)/4 = 100
	got := EstimateTokens(text)
	if got != 100 {
		t.Errorf("EstimateTokens(400 chars) = %d, want 100", got)
	}
}

// ---------------------------------------------------------------------------
// EstimateMessagesTokens
// ---------------------------------------------------------------------------

func TestEstimateMessagesTokens_Empty(t *testing.T) {
	got := EstimateMessagesTokens(nil)
	if got != 0 {
		t.Errorf("EstimateMessagesTokens(nil) = %d, want 0", got)
	}
}

func TestEstimateMessagesTokens_Single(t *testing.T) {
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: strings.Repeat("a", 400)},
	}
	// 4 (overhead) + 100 (content) = 104
	got := EstimateMessagesTokens(msgs)
	if got != 104 {
		t.Errorf("EstimateMessagesTokens = %d, want 104", got)
	}
}

func TestEstimateMessagesTokens_Multiple(t *testing.T) {
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: "hi"},      // 4 + 1 = 5
		{Role: types.RoleAssistant, Content: "hey"}, // 4 + 1 = 5
	}
	got := EstimateMessagesTokens(msgs)
	if got != 10 {
		t.Errorf("EstimateMessagesTokens = %d, want 10", got)
	}
}

func TestEstimateMessagesTokens_RoleOverhead(t *testing.T) {
	// Even an empty content message adds 4 tokens for the role overhead.
	// (0+3)/4 = 0 content tokens + 4 overhead = 4
	msgs := []types.LLMMsg{
		{Role: types.RoleSystem, Content: ""},
	}
	got := EstimateMessagesTokens(msgs)
	if got != 4 {
		t.Errorf("EstimateMessagesTokens (empty content) = %d, want 4", got)
	}
}

// ---------------------------------------------------------------------------
// ComputeAdaptiveChunkRatio
// ---------------------------------------------------------------------------

func TestComputeAdaptiveChunkRatio_EmptyMessages(t *testing.T) {
	got := ComputeAdaptiveChunkRatio(nil, 100000)
	if got != BaseChunkRatio {
		t.Errorf("empty msgs = %v, want %v", got, BaseChunkRatio)
	}
}

func TestComputeAdaptiveChunkRatio_ZeroContextWindow(t *testing.T) {
	msgs := []types.LLMMsg{{Role: types.RoleUser, Content: "hello"}}
	got := ComputeAdaptiveChunkRatio(msgs, 0)
	if got != BaseChunkRatio {
		t.Errorf("zero context window = %v, want %v", got, BaseChunkRatio)
	}
}

func TestComputeAdaptiveChunkRatio_NegativeContextWindow(t *testing.T) {
	msgs := []types.LLMMsg{{Role: types.RoleUser, Content: "hello"}}
	got := ComputeAdaptiveChunkRatio(msgs, -1)
	if got != BaseChunkRatio {
		t.Errorf("negative context window = %v, want %v", got, BaseChunkRatio)
	}
}

func TestComputeAdaptiveChunkRatio_SmallMessages(t *testing.T) {
	// Context window = 100000, threshold = 10000.
	// Each msg: 4 + EstimateTokens("hi") = 4+1 = 5 tokens.
	// Average = 5, well below 10000 threshold.
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: "hi"},
		{Role: types.RoleAssistant, Content: "ok"},
	}
	got := ComputeAdaptiveChunkRatio(msgs, 100000)
	if got != BaseChunkRatio {
		t.Errorf("small messages = %v, want %v", got, BaseChunkRatio)
	}
}

func TestComputeAdaptiveChunkRatio_LargeMessages(t *testing.T) {
	// Context window = 1000, threshold = 100 tokens.
	// Single msg with 800 chars = 4 + (800+3)/4 = 4+200 = 204 tokens avg.
	// avgTokens (204) > threshold (100) → ratio = BaseChunkRatio * (100/204)
	// = 0.4 * 0.4902 ≈ 0.1961
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: strings.Repeat("x", 800)},
	}
	got := ComputeAdaptiveChunkRatio(msgs, 1000)
	if got >= BaseChunkRatio {
		t.Errorf("large messages ratio %v should be less than BaseChunkRatio %v", got, BaseChunkRatio)
	}
	if got < MinChunkRatio {
		t.Errorf("large messages ratio %v should be >= MinChunkRatio %v", got, MinChunkRatio)
	}
}

func TestComputeAdaptiveChunkRatio_VeryLargeMessages(t *testing.T) {
	// Context window = 100, threshold = 10 tokens.
	// Single msg with 4000 chars = 4 + (4000+3)/4 = 4+1000 = 1004 tokens avg.
	// ratio = 0.4 * (10/1004) ≈ 0.004 → capped at MinChunkRatio (0.15).
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: strings.Repeat("x", 4000)},
	}
	got := ComputeAdaptiveChunkRatio(msgs, 100)
	if got != MinChunkRatio {
		t.Errorf("very large messages = %v, want MinChunkRatio %v", got, MinChunkRatio)
	}
}

// ---------------------------------------------------------------------------
// IsOversizedForSummary
// ---------------------------------------------------------------------------

func TestIsOversizedForSummary_Small(t *testing.T) {
	// 100 chars → (100+3)/4 = 25 tokens. contextWindow=100, 50% = 50.
	// 25 < 50 → false.
	msg := types.LLMMsg{Role: types.RoleUser, Content: strings.Repeat("a", 100)}
	if IsOversizedForSummary(msg, 100) {
		t.Error("small message should not be oversized")
	}
}

func TestIsOversizedForSummary_Large(t *testing.T) {
	// 400 chars → 100 tokens. contextWindow=100, 50% = 50.
	// 100 > 50 → true.
	msg := types.LLMMsg{Role: types.RoleUser, Content: strings.Repeat("a", 400)}
	if !IsOversizedForSummary(msg, 100) {
		t.Error("large message should be oversized")
	}
}

func TestIsOversizedForSummary_ZeroContextWindow(t *testing.T) {
	msg := types.LLMMsg{Role: types.RoleUser, Content: strings.Repeat("a", 400)}
	if IsOversizedForSummary(msg, 0) {
		t.Error("zero context window should return false")
	}
}

func TestIsOversizedForSummary_ExactlyHalf(t *testing.T) {
	// 200 chars → (200+3)/4 = 50 tokens. contextWindow=100, 50% = 50.
	// 50 > 50 is false (not strictly greater).
	msg := types.LLMMsg{Role: types.RoleUser, Content: strings.Repeat("a", 200)}
	if IsOversizedForSummary(msg, 100) {
		t.Error("message at exactly 50%% should not be oversized (not strictly greater)")
	}
}

// ---------------------------------------------------------------------------
// SplitMessagesByTokenShare
// ---------------------------------------------------------------------------

func TestSplitMessagesByTokenShare_TwoParts(t *testing.T) {
	msgs := make([]types.LLMMsg, 4)
	for i := range msgs {
		msgs[i] = types.LLMMsg{Role: types.RoleUser, Content: strings.Repeat("a", 100)}
	}

	parts := SplitMessagesByTokenShare(msgs, 2)
	if len(parts) != 2 {
		t.Fatalf("got %d parts, want 2", len(parts))
	}

	total := 0
	for _, p := range parts {
		total += len(p)
	}
	if total != 4 {
		t.Errorf("total messages across parts = %d, want 4", total)
	}
}

func TestSplitMessagesByTokenShare_MorePartsThanMessages(t *testing.T) {
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: "a"},
		{Role: types.RoleUser, Content: "b"},
	}

	parts := SplitMessagesByTokenShare(msgs, 10)
	if len(parts) != 2 {
		t.Fatalf("got %d parts, want 2 (one per message)", len(parts))
	}
	for i, p := range parts {
		if len(p) != 1 {
			t.Errorf("part[%d] has %d messages, want 1", i, len(p))
		}
	}
}

func TestSplitMessagesByTokenShare_ZeroParts(t *testing.T) {
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: "a"},
	}
	parts := SplitMessagesByTokenShare(msgs, 0)
	if len(parts) != 1 {
		t.Fatalf("zero parts: got %d groups, want 1", len(parts))
	}
}

func TestSplitMessagesByTokenShare_NegativeParts(t *testing.T) {
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: "a"},
	}
	parts := SplitMessagesByTokenShare(msgs, -5)
	if len(parts) != 1 {
		t.Fatalf("negative parts: got %d groups, want 1", len(parts))
	}
}

func TestSplitMessagesByTokenShare_Empty(t *testing.T) {
	parts := SplitMessagesByTokenShare(nil, 3)
	if parts != nil {
		t.Errorf("empty messages: got %v, want nil", parts)
	}
}

// ---------------------------------------------------------------------------
// ChunkMessagesByMaxTokens
// ---------------------------------------------------------------------------

func TestChunkMessagesByMaxTokens_AllFitInOne(t *testing.T) {
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: "hi"},
		{Role: types.RoleAssistant, Content: "ok"},
	}
	// Each msg: 4 + 1 = 5 tokens. Total 10. Budget 100.
	chunks := ChunkMessagesByMaxTokens(msgs, 100)
	if len(chunks) != 1 {
		t.Fatalf("got %d chunks, want 1", len(chunks))
	}
	if len(chunks[0]) != 2 {
		t.Errorf("chunk[0] has %d messages, want 2", len(chunks[0]))
	}
}

func TestChunkMessagesByMaxTokens_OversizedSingle(t *testing.T) {
	// Single huge message exceeds maxTokens → gets its own chunk.
	big := types.LLMMsg{Role: types.RoleUser, Content: strings.Repeat("x", 4000)}
	small := types.LLMMsg{Role: types.RoleUser, Content: "hi"}

	chunks := ChunkMessagesByMaxTokens([]types.LLMMsg{small, big, small}, 50)
	// small fits in a chunk. big exceeds 50 → own chunk. Second small → own chunk.
	if len(chunks) < 2 {
		t.Fatalf("got %d chunks, want at least 2", len(chunks))
	}

	// Verify the big message is alone in its chunk.
	foundBigAlone := false
	for _, c := range chunks {
		if len(c) == 1 && len(c[0].Content) == 4000 {
			foundBigAlone = true
		}
	}
	if !foundBigAlone {
		t.Error("oversized message should be alone in its own chunk")
	}
}

func TestChunkMessagesByMaxTokens_GroupsByBudget(t *testing.T) {
	// 4 messages, each ~29 tokens (4 + 25). Budget = 60 → ~2 per chunk.
	msgs := make([]types.LLMMsg, 4)
	for i := range msgs {
		msgs[i] = types.LLMMsg{Role: types.RoleUser, Content: strings.Repeat("a", 100)}
	}

	chunks := ChunkMessagesByMaxTokens(msgs, 60)
	if len(chunks) < 2 {
		t.Fatalf("got %d chunks, want >= 2", len(chunks))
	}

	total := 0
	for _, c := range chunks {
		total += len(c)
	}
	if total != 4 {
		t.Errorf("total messages = %d, want 4", total)
	}
}

func TestChunkMessagesByMaxTokens_Empty(t *testing.T) {
	chunks := ChunkMessagesByMaxTokens(nil, 100)
	if chunks != nil {
		t.Errorf("empty messages: got %v, want nil", chunks)
	}
}

func TestChunkMessagesByMaxTokens_ZeroBudget(t *testing.T) {
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: "hi"},
	}
	// maxTokens <= 0 → defaults to 1; each msg alone in its chunk.
	chunks := ChunkMessagesByMaxTokens(msgs, 0)
	if len(chunks) != 1 {
		t.Fatalf("zero budget: got %d chunks, want 1", len(chunks))
	}
}

// ---------------------------------------------------------------------------
// PruneHistoryForContextShare
// ---------------------------------------------------------------------------

func TestPruneHistoryForContextShare_WithinBudget(t *testing.T) {
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: "hi"},
		{Role: types.RoleAssistant, Content: "hello"},
	}
	// Total: (4+1) + (4+2) = 11. Adjusted: 11*1.2 = 13.
	// Budget: 100000 * 1.0 = 100000. 13 <= 100000 → all kept.
	result := PruneHistoryForContextShare(msgs, 100000, 1.0)
	if len(result.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(result.Messages))
	}
	if len(result.DroppedMessages) != 0 {
		t.Errorf("expected 0 dropped, got %d", len(result.DroppedMessages))
	}
	if result.DroppedTokens != 0 {
		t.Errorf("expected 0 dropped tokens, got %d", result.DroppedTokens)
	}
	if result.KeptTokens != 11 {
		t.Errorf("expected KeptTokens 11, got %d", result.KeptTokens)
	}
	if result.BudgetTokens != 100000 {
		t.Errorf("expected BudgetTokens 100000, got %d", result.BudgetTokens)
	}
}

func TestPruneHistoryForContextShare_OverBudget(t *testing.T) {
	// Create messages that exceed a small budget.
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: strings.Repeat("a", 400)},   // 4+100=104
		{Role: types.RoleUser, Content: strings.Repeat("b", 400)},   // 4+100=104
		{Role: types.RoleUser, Content: strings.Repeat("c", 400)},   // 4+100=104
		{Role: types.RoleAssistant, Content: strings.Repeat("d", 40)}, // 4+10=14
	}
	// Total tokens: 104+104+104+14 = 326.
	// Adjusted: 326*1.2 = 391.
	// Budget: 200 * 1.0 = 200. 391 > 200 → need to drop from front.
	result := PruneHistoryForContextShare(msgs, 200, 1.0)

	if len(result.DroppedMessages) == 0 {
		t.Error("expected some messages to be dropped")
	}
	if len(result.Messages) >= 4 {
		t.Error("expected fewer messages to be kept")
	}

	// The oldest messages should be dropped first.
	for _, dm := range result.DroppedMessages {
		if dm.Content == msgs[3].Content {
			t.Error("newest message should not be in dropped list")
		}
	}
}

func TestPruneHistoryForContextShare_SafetyMarginApplied(t *testing.T) {
	// Budget is just barely above raw total but below safety-adjusted total.
	// 2 messages: each "a"*40 → (40+3)/4=10 tokens. 4+10=14 per msg. Total=28.
	// Adjusted: 28*1.2 = 33.
	// Set budget = 30: 30 < 33 so it should drop the first message.
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: strings.Repeat("a", 40)},
		{Role: types.RoleUser, Content: strings.Repeat("b", 40)},
	}
	result := PruneHistoryForContextShare(msgs, 30, 1.0)

	if len(result.DroppedMessages) == 0 {
		t.Error("safety margin should cause pruning at budget=30 with adjusted total=33")
	}
}

func TestPruneHistoryForContextShare_DroppedAndKeptTokens(t *testing.T) {
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: strings.Repeat("a", 400)}, // 4+100=104
		{Role: types.RoleUser, Content: strings.Repeat("b", 400)}, // 4+100=104
		{Role: types.RoleUser, Content: "hi"},                     // 4+1=5
	}
	// Total: 104+104+5 = 213. Adjusted: 213*1.2 = 255.
	// Budget: 100 * 1.0 = 100. Will need to drop.
	result := PruneHistoryForContextShare(msgs, 100, 1.0)

	if result.DroppedTokens <= 0 {
		t.Error("expected positive DroppedTokens")
	}
	if result.KeptTokens <= 0 {
		t.Error("expected positive KeptTokens")
	}
	if result.KeptTokens != EstimateMessagesTokens(result.Messages) {
		t.Errorf("KeptTokens %d does not match EstimateMessagesTokens of kept messages %d",
			result.KeptTokens, EstimateMessagesTokens(result.Messages))
	}
}

func TestPruneHistoryForContextShare_MaxHistoryShare(t *testing.T) {
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: strings.Repeat("a", 400)}, // 104 tokens
	}
	// Budget: 200 * 0.5 = 100. Adjusted: 104*1.2 = 124. 124 > 100 → drop.
	result := PruneHistoryForContextShare(msgs, 200, 0.5)
	if result.BudgetTokens != 100 {
		t.Errorf("BudgetTokens = %d, want 100", result.BudgetTokens)
	}
}

func TestPruneHistoryForContextShare_ClampShare(t *testing.T) {
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: "hi"},
	}
	// maxHistoryShare > 1.0 → clamped to 1.0. Budget = 100000.
	result := PruneHistoryForContextShare(msgs, 100000, 2.0)
	if result.BudgetTokens != 100000 {
		t.Errorf("BudgetTokens = %d, want 100000 (share clamped)", result.BudgetTokens)
	}

	// maxHistoryShare <= 0 → treated as 1.0.
	result2 := PruneHistoryForContextShare(msgs, 100000, -1.0)
	if result2.BudgetTokens != 100000 {
		t.Errorf("BudgetTokens = %d, want 100000 (negative share → 1.0)", result2.BudgetTokens)
	}
}

func TestPruneHistoryForContextShare_DroppedChunks(t *testing.T) {
	// When messages are dropped, DroppedChunks should be 1 (one contiguous block from front).
	msgs := []types.LLMMsg{
		{Role: types.RoleUser, Content: strings.Repeat("a", 400)},
		{Role: types.RoleUser, Content: strings.Repeat("b", 400)},
		{Role: types.RoleUser, Content: "hi"},
	}
	result := PruneHistoryForContextShare(msgs, 100, 1.0)
	if len(result.DroppedMessages) > 0 && result.DroppedChunks != 1 {
		t.Errorf("DroppedChunks = %d, want 1", result.DroppedChunks)
	}
}
