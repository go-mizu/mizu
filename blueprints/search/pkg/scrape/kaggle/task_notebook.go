package kaggle

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type NotebookState struct {
	URL    string
	Status string
	Error  string
}

type NotebookMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

type NotebookTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[NotebookState, NotebookMetric] = (*NotebookTask)(nil)

func (t *NotebookTask) Run(ctx context.Context, emit func(*NotebookState)) (NotebookMetric, error) {
	var m NotebookMetric
	t.URL = NormalizeNotebookURL(t.URL)
	emit(&NotebookState{URL: t.URL, Status: "fetching"})
	ref := ExtractNotebookRef(t.URL)
	if ref == "" {
		m.Failed++
		msg := "cannot extract notebook ref"
		emit(&NotebookState{URL: t.URL, Status: "failed", Error: msg})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, msg)
		}
		return m, nil
	}
	if t.StateDB != nil && t.StateDB.IsVisited(t.URL) {
		m.Skipped++
		_ = t.StateDB.Done(t.URL, EntityNotebook, 200)
		emit(&NotebookState{URL: t.URL, Status: "skipped"})
		return m, nil
	}
	meta, code, err := t.Client.FetchHTMLMeta(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&NotebookState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 || meta == nil {
		m.Skipped++
		if t.StateDB != nil {
			_ = t.StateDB.Done(t.URL, EntityNotebook, 404)
		}
		return m, nil
	}
	raw, _ := json.Marshal(meta)
	owner, slug, _ := strings.Cut(ref, "/")
	item := Notebook{
		Ref:         ref,
		OwnerRef:    owner,
		Slug:        slug,
		Title:       strings.TrimSpace(firstNonEmpty(meta.OGTitle, meta.Title)),
		Description: firstNonEmpty(meta.Description, meta.OGDesc),
		URL:         t.URL,
		ImageURL:    meta.OGImage,
		RawMetaJSON: string(raw),
		FetchedAt:   time.Now(),
	}
	if err := t.DB.UpsertNotebook(item); err != nil {
		m.Failed++
		emit(&NotebookState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			_ = t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if t.StateDB != nil {
		if owner != "" {
			_ = t.StateDB.Enqueue(NormalizeProfileURL(owner), EntityProfile, 1)
		}
		_ = t.StateDB.Done(t.URL, EntityNotebook, 200)
	}
	m.Fetched++
	emit(&NotebookState{URL: t.URL, Status: "done"})
	return m, nil
}

func FetchNotebook(ctx context.Context, client *Client, db *DB, stateDB *State, raw string) error {
	task := &NotebookTask{URL: raw, Client: client, DB: db, StateDB: stateDB}
	m, err := task.Run(ctx, func(*NotebookState) {})
	if err != nil {
		return err
	}
	if m.Failed > 0 {
		return fmt.Errorf("failed to fetch notebook")
	}
	return nil
}
