package kaggle

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

type DatasetMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

type DatasetTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[DatasetState, DatasetMetric] = (*DatasetTask)(nil)

func (t *DatasetTask) Run(ctx context.Context, emit func(*DatasetState)) (DatasetMetric, error) {
	var m DatasetMetric
	t.URL = NormalizeDatasetURL(t.URL)
	emit(&DatasetState{URL: t.URL, Status: "fetching"})
	ref := ExtractDatasetRef(t.URL)
	if ref == "" {
		m.Failed++
		msg := "cannot extract dataset ref"
		emit(&DatasetState{URL: t.URL, Status: "failed", Error: msg})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, msg)
		}
		return m, nil
	}
	if t.StateDB != nil && t.StateDB.IsVisited(t.URL) {
		m.Skipped++
		_ = t.StateDB.Done(t.URL, EntityDataset, 200)
		emit(&DatasetState{URL: t.URL, Status: "skipped"})
		return m, nil
	}
	item, err := t.Client.ViewDataset(ctx, ref)
	if err != nil {
		m.Failed++
		emit(&DatasetState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if err := t.DB.UpsertDataset(*item); err != nil {
		m.Failed++
		emit(&DatasetState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if t.StateDB != nil {
		if handle := firstNonEmpty(item.OwnerRef, item.CreatorURL); handle != "" {
			_ = t.StateDB.Enqueue(NormalizeProfileURL(handle), EntityProfile, 1)
		}
		_ = t.StateDB.Done(t.URL, EntityDataset, 200)
	}
	m.Fetched++
	emit(&DatasetState{URL: t.URL, Status: "done"})
	return m, nil
}

func FetchDataset(ctx context.Context, client *Client, db *DB, stateDB *State, raw string) error {
	task := &DatasetTask{URL: raw, Client: client, DB: db, StateDB: stateDB}
	m, err := task.Run(ctx, func(*DatasetState) {})
	if err != nil {
		return err
	}
	if m.Failed > 0 {
		return fmt.Errorf("failed to fetch dataset")
	}
	return nil
}
