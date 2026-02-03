package bot

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/tools"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

const maxToolIterations = tools.MaxToolIterations

// runToolLoop delegates to the shared tool loop in pkg/tools.
func runToolLoop(
	ctx context.Context,
	provider llm.ToolProvider,
	registry *tools.Registry,
	req *types.LLMToolRequest,
) (*types.LLMToolResponse, error) {
	return tools.RunToolLoop(ctx, provider, registry, req)
}
