package huggingface

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type CollectionState struct {
	URL        string
	Status     string
	Error      string
	ItemsFound int
}

type CollectionTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[CollectionState, ResultMetric] = (*CollectionTask)(nil)

func (t *CollectionTask) Run(ctx context.Context, emit func(*CollectionState)) (ResultMetric, error) {
	var m ResultMetric
	slug, canonical, err := NormalizeCollectionRef(t.URL)
	if err != nil {
		m.Failed++
		emit(&CollectionState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	emit(&CollectionState{URL: canonical, Status: "fetching"})
	collection, items, code, err := t.Client.GetCollection(ctx, slug)
	if err != nil {
		m.Failed++
		emit(&CollectionState{URL: canonical, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(canonical, err.Error())
		}
		return m, nil
	}
	if code == 404 || collection == nil {
		m.Skipped++
		emit(&CollectionState{URL: canonical, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(canonical, 404, EntityCollection)
		}
		return m, nil
	}
	if err := t.DB.UpsertCollection(*collection); err != nil {
		return m, fmt.Errorf("upsert collection: %w", err)
	}
	if err := t.DB.ReplaceCollectionItems(collection.Slug, items); err != nil {
		return m, fmt.Errorf("replace collection items: %w", err)
	}
	if t.StateDB != nil {
		for _, item := range items {
			switch item.ItemType {
			case EntityModel:
				_ = t.StateDB.Enqueue(canonicalRepoURL(EntityModel, item.ItemID), EntityModel, 2)
			case EntityDataset:
				_ = t.StateDB.Enqueue(canonicalRepoURL(EntityDataset, item.ItemID), EntityDataset, 2)
			case EntitySpace:
				_ = t.StateDB.Enqueue(canonicalRepoURL(EntitySpace, item.ItemID), EntitySpace, 2)
			case EntityPaper:
				_ = t.StateDB.Enqueue(canonicalPaperURL(item.ItemID), EntityPaper, 2)
			}
		}
		_ = t.StateDB.Done(canonical, code, EntityCollection)
	}
	m.Fetched++
	emit(&CollectionState{URL: canonical, Status: "done", ItemsFound: len(items)})
	return m, nil
}
