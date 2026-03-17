package huggingface

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type DatasetState struct {
	URL    string
	Status string
	Error  string
}

type DatasetTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[DatasetState, ResultMetric] = (*DatasetTask)(nil)

func (t *DatasetTask) Run(ctx context.Context, emit func(*DatasetState)) (ResultMetric, error) {
	var m ResultMetric
	repoID, canonical, err := NormalizeDatasetRef(t.URL)
	if err != nil {
		m.Failed++
		emit(&DatasetState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	emit(&DatasetState{URL: canonical, Status: "fetching"})
	ds, files, code, err := t.Client.GetDataset(ctx, repoID)
	if err != nil {
		m.Failed++
		emit(&DatasetState{URL: canonical, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(canonical, err.Error())
		}
		return m, nil
	}
	if code == 404 || ds == nil {
		m.Skipped++
		emit(&DatasetState{URL: canonical, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(canonical, 404, EntityDataset)
		}
		return m, nil
	}
	if err := t.DB.UpsertDataset(*ds); err != nil {
		return m, fmt.Errorf("upsert dataset: %w", err)
	}
	if err := t.DB.ReplaceRepoFiles(EntityDataset, ds.RepoID, files); err != nil {
		return m, fmt.Errorf("replace dataset files: %w", err)
	}
	if t.StateDB != nil {
		_ = t.StateDB.Done(canonical, code, EntityDataset)
	}
	m.Fetched++
	emit(&DatasetState{URL: canonical, Status: "done"})
	return m, nil
}
