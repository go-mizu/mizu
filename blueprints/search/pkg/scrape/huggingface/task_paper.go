package huggingface

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type PaperState struct {
	URL    string
	Status string
	Error  string
}

type PaperTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[PaperState, ResultMetric] = (*PaperTask)(nil)

func (t *PaperTask) Run(ctx context.Context, emit func(*PaperState)) (ResultMetric, error) {
	var m ResultMetric
	paperID, canonical, err := NormalizePaperRef(t.URL)
	if err != nil {
		m.Failed++
		emit(&PaperState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	emit(&PaperState{URL: canonical, Status: "fetching"})
	paper, code, err := t.Client.GetPaper(ctx, paperID)
	if err != nil {
		m.Failed++
		emit(&PaperState{URL: canonical, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(canonical, err.Error())
		}
		return m, nil
	}
	if code == 404 || paper == nil {
		m.Skipped++
		emit(&PaperState{URL: canonical, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(canonical, 404, EntityPaper)
		}
		return m, nil
	}
	if err := t.DB.UpsertPaper(*paper); err != nil {
		return m, fmt.Errorf("upsert paper: %w", err)
	}
	if t.StateDB != nil {
		_ = t.StateDB.Done(canonical, code, EntityPaper)
	}
	m.Fetched++
	emit(&PaperState{URL: canonical, Status: "done"})
	return m, nil
}
