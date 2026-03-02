package devnull

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func init() {
	index.Register("devnull", func() index.Engine { return &Engine{} })
}

type Engine struct{}

func (e *Engine) Name() string                                         { return "devnull" }
func (e *Engine) Open(_ context.Context, _ string) error               { return nil }
func (e *Engine) Close() error                                         { return nil }
func (e *Engine) Stats(_ context.Context) (index.EngineStats, error)   { return index.EngineStats{}, nil }
func (e *Engine) Index(_ context.Context, _ []index.Document) error    { return nil }
func (e *Engine) Search(_ context.Context, _ index.Query) (index.Results, error) {
	return index.Results{}, nil
}

var _ index.Engine = (*Engine)(nil)
