package huggingface

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type SpaceState struct {
	URL    string
	Status string
	Error  string
}

type SpaceTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[SpaceState, ResultMetric] = (*SpaceTask)(nil)

func (t *SpaceTask) Run(ctx context.Context, emit func(*SpaceState)) (ResultMetric, error) {
	var m ResultMetric
	repoID, canonical, err := NormalizeSpaceRef(t.URL)
	if err != nil {
		m.Failed++
		emit(&SpaceState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	emit(&SpaceState{URL: canonical, Status: "fetching"})
	space, files, links, code, err := t.Client.GetSpace(ctx, repoID)
	if err != nil {
		m.Failed++
		emit(&SpaceState{URL: canonical, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(canonical, err.Error())
		}
		return m, nil
	}
	if code == 404 || space == nil {
		m.Skipped++
		emit(&SpaceState{URL: canonical, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(canonical, 404, EntitySpace)
		}
		return m, nil
	}
	if err := t.DB.UpsertSpace(*space); err != nil {
		return m, fmt.Errorf("upsert space: %w", err)
	}
	if err := t.DB.ReplaceRepoFiles(EntitySpace, space.RepoID, files); err != nil {
		return m, fmt.Errorf("replace space files: %w", err)
	}
	if err := t.DB.ReplaceSourceLinks(EntitySpace, space.RepoID, links); err != nil {
		return m, fmt.Errorf("replace space links: %w", err)
	}
	if t.StateDB != nil {
		for _, link := range links {
			switch link.DstType {
			case EntityModel, EntityDataset:
				_ = t.StateDB.Enqueue(canonicalRepoURL(link.DstType, link.DstID), link.DstType, 2)
			}
		}
		_ = t.StateDB.Done(canonical, code, EntitySpace)
	}
	m.Fetched++
	emit(&SpaceState{URL: canonical, Status: "done"})
	return m, nil
}
