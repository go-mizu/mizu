package kaggle

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

type ModelMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

type ModelTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[ModelState, ModelMetric] = (*ModelTask)(nil)

func (t *ModelTask) Run(ctx context.Context, emit func(*ModelState)) (ModelMetric, error) {
	var m ModelMetric
	t.URL = NormalizeModelURL(t.URL)
	emit(&ModelState{URL: t.URL, Status: "fetching"})
	ref := ExtractModelRef(t.URL)
	if ref == "" {
		m.Failed++
		msg := "cannot extract model ref"
		emit(&ModelState{URL: t.URL, Status: "failed", Error: msg})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, msg)
		}
		return m, nil
	}
	if t.StateDB != nil && t.StateDB.IsVisited(t.URL) {
		m.Skipped++
		_ = t.StateDB.Done(t.URL, EntityModel, 200)
		emit(&ModelState{URL: t.URL, Status: "skipped"})
		return m, nil
	}
	item, err := t.Client.FindModel(ctx, ref)
	if err != nil {
		m.Failed++
		emit(&ModelState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if err := t.DB.UpsertModel(*item); err != nil {
		m.Failed++
		emit(&ModelState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if t.StateDB != nil {
		if item.OwnerRef != "" {
			_ = t.StateDB.Enqueue(NormalizeProfileURL(item.OwnerRef), EntityProfile, 1)
		}
		_ = t.StateDB.Done(t.URL, EntityModel, 200)
	}
	m.Fetched++
	emit(&ModelState{URL: t.URL, Status: "done"})
	return m, nil
}

func FetchModel(ctx context.Context, client *Client, db *DB, stateDB *State, raw string) error {
	task := &ModelTask{URL: raw, Client: client, DB: db, StateDB: stateDB}
	m, err := task.Run(ctx, func(*ModelState) {})
	if err != nil {
		return err
	}
	if m.Failed > 0 {
		return fmt.Errorf("failed to fetch model")
	}
	return nil
}
