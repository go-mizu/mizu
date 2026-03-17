package huggingface

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type DiscoverState struct {
	EntityType string
	Page       int
	Discovered int
	Enqueued   int
	NextURL    string
	Status     string
	Error      string
}

type DiscoverMetric struct {
	Discovered int
	Enqueued   int
	Pages      int
}

type DiscoverTask struct {
	Config   Config
	Client   *Client
	StateDB  *State
	Priority int
}

var _ core.Task[DiscoverState, DiscoverMetric] = (*DiscoverTask)(nil)

func (t *DiscoverTask) Run(ctx context.Context, emit func(*DiscoverState)) (DiscoverMetric, error) {
	var metric DiscoverMetric
	pageSize := t.Config.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	for _, entityType := range expandTypes(t.Config.Types) {
		next := ""
		for page := 1; ; page++ {
			if ctx.Err() != nil {
				return metric, ctx.Err()
			}
			if t.Config.MaxPages > 0 && page > t.Config.MaxPages {
				break
			}
			emit(&DiscoverState{EntityType: entityType, Page: page, Status: "fetching", NextURL: next})
			items, nextURL, code, err := t.Client.ListPage(ctx, entityType, next, pageSize)
			if err != nil {
				emit(&DiscoverState{EntityType: entityType, Page: page, Status: "failed", Error: err.Error()})
				return metric, err
			}
			if code == 404 {
				break
			}
			queueItems := make([]QueueItem, 0, len(items))
			discovered := 0
			for _, item := range items {
				url := discoverCanonicalURL(entityType, item)
				if url == "" {
					continue
				}
				queueItems = append(queueItems, QueueItem{URL: url, EntityType: entityType, Priority: t.Priority})
				discovered++
			}
			if err := t.StateDB.EnqueueBatch(queueItems); err != nil {
				return metric, fmt.Errorf("enqueue discover batch: %w", err)
			}
			metric.Discovered += discovered
			metric.Enqueued += len(queueItems)
			metric.Pages++
			emit(&DiscoverState{
				EntityType: entityType,
				Page:       page,
				Discovered: discovered,
				Enqueued:   len(queueItems),
				NextURL:    nextURL,
				Status:     "done",
			})
			if nextURL == "" || len(items) == 0 {
				break
			}
			next = nextURL
			time.Sleep(10 * time.Millisecond)
		}
	}
	return metric, nil
}

func discoverCanonicalURL(entityType string, item map[string]any) string {
	switch entityType {
	case EntityModel:
		id := firstString(item["id"], item["modelId"])
		if id == "" {
			return ""
		}
		return canonicalRepoURL(EntityModel, id)
	case EntityDataset:
		id := stringValue(item["id"])
		if id == "" {
			return ""
		}
		return canonicalRepoURL(EntityDataset, id)
	case EntitySpace:
		id := stringValue(item["id"])
		if id == "" {
			return ""
		}
		return canonicalRepoURL(EntitySpace, id)
	case EntityCollection:
		slug := stringValue(item["slug"])
		if slug == "" {
			return ""
		}
		return canonicalCollectionURL(slug)
	case EntityPaper:
		id := stringValue(item["id"])
		if id == "" {
			return ""
		}
		return canonicalPaperURL(id)
	default:
		return ""
	}
}

func expandTypes(types []string) []string {
	if len(types) == 0 {
		return DefaultConfig().Types
	}
	seen := map[string]bool{}
	var out []string
	for _, raw := range types {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(strings.ToLower(part))
			switch part {
			case "models":
				part = EntityModel
			case "datasets":
				part = EntityDataset
			case "spaces":
				part = EntitySpace
			case "collections":
				part = EntityCollection
			case "papers":
				part = EntityPaper
			}
			if part == "" || seen[part] {
				continue
			}
			seen[part] = true
			out = append(out, part)
		}
	}
	return out
}
