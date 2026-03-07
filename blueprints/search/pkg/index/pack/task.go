package pack

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// Task runs the filesystem markdown->engine indexing pipeline.
type Task struct {
	Engine   Engine
	Config   PipelineConfig
	Progress ProgressFunc
}

var _ core.Task[PipelineStats, *PipelineStats] = (*Task)(nil)

func (t *Task) Run(ctx context.Context, emit func(*PipelineStats)) (*PipelineStats, error) {
	if emit == nil {
		return RunPipeline(ctx, t.Engine, t.Config, t.Progress)
	}
	return RunPipeline(ctx, t.Engine, t.Config, func(s *PipelineStats) {
		emit(s)
		if t.Progress != nil {
			t.Progress(s)
		}
	})
}

// ChannelTask runs the channel->engine pipeline.
type ChannelTask struct {
	Engine    Engine
	DocCh     <-chan Document
	Total     int64
	BatchSize int
	Progress  PackProgressFunc
}

var _ core.Task[PipelineStats, *PipelineStats] = (*ChannelTask)(nil)

func (t *ChannelTask) Run(ctx context.Context, emit func(*PipelineStats)) (*PipelineStats, error) {
	stats, err := RunPipelineFromChannel(ctx, t.Engine, t.DocCh, t.Total, t.BatchSize, t.Progress)
	if emit != nil && stats != nil {
		emit(stats)
	}
	return stats, err
}
