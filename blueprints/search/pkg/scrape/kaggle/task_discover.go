package kaggle

import (
	"context"
	"fmt"
	"slices"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type DiscoverState struct {
	Status           string
	Page             int
	DatasetsFound    int
	ModelsFound      int
	ProfilesEnqueued int
	Error            string
}

type DiscoverMetric struct {
	DatasetsFound    int
	ModelsFound      int
	ProfilesEnqueued int
	Pages            int
}

type DiscoverTask struct {
	Config  Config
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[DiscoverState, DiscoverMetric] = (*DiscoverTask)(nil)

func (t *DiscoverTask) Run(ctx context.Context, emit func(*DiscoverState)) (DiscoverMetric, error) {
	var m DiscoverMetric
	types := t.Config.Types
	if len(types) == 0 {
		types = []string{EntityDataset, EntityModel}
	}
	maxPages := t.Config.MaxPages
	if maxPages <= 0 {
		maxPages = DefaultMaxPages
	}

	if slices.Contains(types, EntityDataset) {
		for page := 1; page <= maxPages; page++ {
			items, count, err := t.Client.ListDatasets(ctx, page, "")
			if err != nil {
				emit(&DiscoverState{Status: "failed", Page: page, Error: err.Error()})
				return m, nil
			}
			if count == 0 {
				break
			}
			queue := make([]QueueItem, 0, len(items)*2)
			for _, item := range items {
				_ = t.DB.UpsertDataset(item)
				queue = append(queue, QueueItem{URL: NormalizeDatasetURL(item.Ref), EntityType: EntityDataset, Priority: 5})
				if handle := firstNonEmpty(item.OwnerRef, item.CreatorURL); handle != "" {
					queue = append(queue, QueueItem{URL: NormalizeProfileURL(handle), EntityType: EntityProfile, Priority: 1})
					m.ProfilesEnqueued++
				}
				m.DatasetsFound++
			}
			_ = t.StateDB.EnqueueBatch(queue)
			m.Pages++
			emit(&DiscoverState{
				Status:           "datasets",
				Page:             page,
				DatasetsFound:    m.DatasetsFound,
				ModelsFound:      m.ModelsFound,
				ProfilesEnqueued: m.ProfilesEnqueued,
			})
		}
	}

	if slices.Contains(types, EntityModel) {
		nextToken := ""
		for page := 1; page <= maxPages; page++ {
			items, next, err := t.Client.ListModels(ctx, "", nextToken)
			if err != nil {
				emit(&DiscoverState{Status: "failed", Page: page, Error: err.Error()})
				return m, nil
			}
			if len(items) == 0 {
				break
			}
			queue := make([]QueueItem, 0, len(items)*2)
			for _, item := range items {
				_ = t.DB.UpsertModel(item)
				queue = append(queue, QueueItem{URL: NormalizeModelURL(item.Ref), EntityType: EntityModel, Priority: 5})
				if item.OwnerRef != "" {
					queue = append(queue, QueueItem{URL: NormalizeProfileURL(item.OwnerRef), EntityType: EntityProfile, Priority: 1})
					m.ProfilesEnqueued++
				}
				m.ModelsFound++
			}
			_ = t.StateDB.EnqueueBatch(queue)
			m.Pages++
			emit(&DiscoverState{
				Status:           "models",
				Page:             page,
				DatasetsFound:    m.DatasetsFound,
				ModelsFound:      m.ModelsFound,
				ProfilesEnqueued: m.ProfilesEnqueued,
			})
			if next == "" {
				break
			}
			nextToken = next
		}
	}

	emit(&DiscoverState{
		Status:           "done",
		Page:             m.Pages,
		DatasetsFound:    m.DatasetsFound,
		ModelsFound:      m.ModelsFound,
		ProfilesEnqueued: m.ProfilesEnqueued,
	})
	return m, nil
}

func ValidateDiscoverTypes(types []string) error {
	for _, t := range types {
		switch t {
		case EntityDataset, EntityModel:
		default:
			return fmt.Errorf("unsupported discover type: %s", t)
		}
	}
	return nil
}
