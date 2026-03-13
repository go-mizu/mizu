package kaggle

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type CompetitionState struct {
	URL    string
	Status string
	Error  string
}

type CompetitionMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

type CompetitionTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[CompetitionState, CompetitionMetric] = (*CompetitionTask)(nil)

func (t *CompetitionTask) Run(ctx context.Context, emit func(*CompetitionState)) (CompetitionMetric, error) {
	var m CompetitionMetric
	t.URL = NormalizeCompetitionURL(t.URL)
	emit(&CompetitionState{URL: t.URL, Status: "fetching"})
	slug := ExtractCompetitionSlug(t.URL)
	if slug == "" {
		m.Failed++
		msg := "cannot extract competition slug"
		emit(&CompetitionState{URL: t.URL, Status: "failed", Error: msg})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, msg)
		}
		return m, nil
	}
	if t.StateDB != nil && t.StateDB.IsVisited(t.URL) {
		m.Skipped++
		_ = t.StateDB.Done(t.URL, EntityCompetition, 200)
		emit(&CompetitionState{URL: t.URL, Status: "skipped"})
		return m, nil
	}
	meta, code, err := t.Client.FetchHTMLMeta(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&CompetitionState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 || meta == nil {
		m.Skipped++
		if t.StateDB != nil {
			_ = t.StateDB.Done(t.URL, EntityCompetition, 404)
		}
		return m, nil
	}
	raw, _ := json.Marshal(meta)
	item := Competition{
		Slug:        slug,
		Title:       strings.TrimSpace(firstNonEmpty(meta.OGTitle, meta.Title)),
		Description: firstNonEmpty(meta.Description, meta.OGDesc),
		URL:         t.URL,
		ImageURL:    meta.OGImage,
		RawMetaJSON: string(raw),
		FetchedAt:   time.Now(),
	}
	if err := t.DB.UpsertCompetition(item); err != nil {
		m.Failed++
		emit(&CompetitionState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if t.StateDB != nil {
		_ = t.StateDB.Done(t.URL, EntityCompetition, 200)
	}
	m.Fetched++
	emit(&CompetitionState{URL: t.URL, Status: "done"})
	return m, nil
}

func FetchCompetition(ctx context.Context, client *Client, db *DB, stateDB *State, raw string) error {
	task := &CompetitionTask{URL: raw, Client: client, DB: db, StateDB: stateDB}
	m, err := task.Run(ctx, func(*CompetitionState) {})
	if err != nil {
		return err
	}
	if m.Failed > 0 {
		return fmt.Errorf("failed to fetch competition")
	}
	return nil
}
