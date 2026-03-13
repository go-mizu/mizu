package kaggle

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type ProfileState struct {
	URL    string
	Status string
	Error  string
}

type ProfileMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

type ProfileTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[ProfileState, ProfileMetric] = (*ProfileTask)(nil)

func (t *ProfileTask) Run(ctx context.Context, emit func(*ProfileState)) (ProfileMetric, error) {
	var m ProfileMetric
	t.URL = NormalizeProfileURL(t.URL)
	emit(&ProfileState{URL: t.URL, Status: "fetching"})
	handle := ExtractProfileHandle(t.URL)
	if handle == "" {
		m.Failed++
		msg := "cannot extract profile handle"
		emit(&ProfileState{URL: t.URL, Status: "failed", Error: msg})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, msg)
		}
		return m, nil
	}
	if t.StateDB != nil && t.StateDB.IsVisited(t.URL) {
		m.Skipped++
		_ = t.StateDB.Done(t.URL, EntityProfile, 200)
		emit(&ProfileState{URL: t.URL, Status: "skipped"})
		return m, nil
	}
	meta, code, err := t.Client.FetchHTMLMeta(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&ProfileState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 || meta == nil {
		m.Skipped++
		if t.StateDB != nil {
			_ = t.StateDB.Done(t.URL, EntityProfile, 404)
		}
		return m, nil
	}
	raw, _ := json.Marshal(meta)
	item := Profile{
		Handle:      handle,
		DisplayName: strings.TrimSpace(strings.TrimSuffix(firstNonEmpty(meta.OGTitle, meta.Title), " | Kaggle")),
		Bio:         firstNonEmpty(meta.Description, meta.OGDesc),
		URL:         t.URL,
		ImageURL:    meta.OGImage,
		RawMetaJSON: string(raw),
		FetchedAt:   time.Now(),
	}
	if err := t.DB.UpsertProfile(item); err != nil {
		m.Failed++
		emit(&ProfileState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if t.StateDB != nil {
		_ = t.StateDB.Done(t.URL, EntityProfile, 200)
	}
	m.Fetched++
	emit(&ProfileState{URL: t.URL, Status: "done"})
	return m, nil
}

func FetchProfile(ctx context.Context, client *Client, db *DB, stateDB *State, raw string) error {
	task := &ProfileTask{URL: raw, Client: client, DB: db, StateDB: stateDB}
	m, err := task.Run(ctx, func(*ProfileState) {})
	if err != nil {
		return err
	}
	if m.Failed > 0 {
		return fmt.Errorf("failed to fetch profile")
	}
	return nil
}
