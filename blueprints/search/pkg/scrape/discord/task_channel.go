package discord

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type ChannelState struct {
	ChannelID string
	Name      string
	Status    string
	Error     string
}

type ChannelMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

type ChannelTask struct {
	ID      string // raw channel ID or URL
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[ChannelState, ChannelMetric] = (*ChannelTask)(nil)

func (t *ChannelTask) Run(ctx context.Context, emit func(*ChannelState)) (ChannelMetric, error) {
	var m ChannelMetric

	ref, err := ParseRef(t.ID, EntityChannel)
	if err != nil {
		return m, err
	}

	if t.StateDB != nil && t.StateDB.IsVisited(ref.URL) {
		_ = t.StateDB.Done(ref.URL, 200, EntityChannel)
		m.Skipped++
		emit(&ChannelState{ChannelID: ref.ID, Status: "skipped"})
		return m, nil
	}

	emit(&ChannelState{ChannelID: ref.ID, Status: "fetching"})

	raw, code, err := t.Client.FetchChannel(ctx, ref.ID)
	if err != nil {
		m.Failed++
		if t.StateDB != nil {
			t.StateDB.Fail(ref.URL, err.Error())
		}
		emit(&ChannelState{ChannelID: ref.ID, Status: "failed", Error: err.Error()})
		return m, nil
	}
	if code == 404 || raw == nil {
		m.Skipped++
		if t.StateDB != nil {
			t.StateDB.Done(ref.URL, 404, EntityChannel)
		}
		emit(&ChannelState{ChannelID: ref.ID, Status: "not_found"})
		return m, nil
	}

	ch := ParseChannel(raw, "")
	if ch.ChannelID == "" {
		ch.ChannelID = ref.ID
	}
	ch.FetchedAt = time.Now()

	if err := t.DB.UpsertChannel(ch); err != nil {
		m.Failed++
		return m, err
	}

	// Enqueue first message page (before="" = latest)
	if t.StateDB != nil {
		pageURL := messagePageQueueURL(ch.ChannelID, "")
		_ = t.StateDB.Enqueue(pageURL, EntityMessagePage, 8)
	}

	if t.StateDB != nil {
		t.StateDB.Done(ref.URL, 200, EntityChannel)
	}
	m.Fetched++
	emit(&ChannelState{ChannelID: ref.ID, Name: ch.Name, Status: "done"})
	return m, nil
}
