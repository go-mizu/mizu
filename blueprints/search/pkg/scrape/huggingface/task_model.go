package huggingface

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type ModelState struct {
	URL    string
	Status string
	Error  string
}

type ModelTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[ModelState, ResultMetric] = (*ModelTask)(nil)

func (t *ModelTask) Run(ctx context.Context, emit func(*ModelState)) (ResultMetric, error) {
	var m ResultMetric
	repoID, canonical, err := NormalizeModelRef(t.URL)
	if err != nil {
		m.Failed++
		emit(&ModelState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	emit(&ModelState{URL: canonical, Status: "fetching"})
	model, files, links, code, err := t.Client.GetModel(ctx, repoID)
	if err != nil {
		m.Failed++
		emit(&ModelState{URL: canonical, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(canonical, err.Error())
		}
		return m, nil
	}
	if code == 404 || model == nil {
		m.Skipped++
		emit(&ModelState{URL: canonical, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(canonical, 404, EntityModel)
		}
		return m, nil
	}
	if err := t.DB.UpsertModel(*model); err != nil {
		return m, fmt.Errorf("upsert model: %w", err)
	}
	if err := t.DB.ReplaceRepoFiles(EntityModel, model.RepoID, files); err != nil {
		return m, fmt.Errorf("replace model files: %w", err)
	}
	if err := t.DB.ReplaceSourceLinks(EntityModel, model.RepoID, links); err != nil {
		return m, fmt.Errorf("replace model links: %w", err)
	}
	if t.StateDB != nil {
		for _, link := range links {
			if link.DstType == EntitySpace {
				_ = t.StateDB.Enqueue(canonicalRepoURL(EntitySpace, link.DstID), EntitySpace, 2)
			}
		}
		_ = t.StateDB.Done(canonical, code, EntityModel)
	}
	m.Fetched++
	emit(&ModelState{URL: canonical, Status: "done"})
	return m, nil
}
