package discord

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type GuildState struct {
	GuildID      string
	Name         string
	Status       string
	Error        string
	ChannelsSeen int
}

type GuildMetric struct {
	Fetched  int
	Skipped  int
	Failed   int
	Channels int
}

type GuildTask struct {
	ID      string // raw guild ID or URL
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[GuildState, GuildMetric] = (*GuildTask)(nil)

func (t *GuildTask) Run(ctx context.Context, emit func(*GuildState)) (GuildMetric, error) {
	var m GuildMetric

	ref, err := ParseRef(t.ID, EntityGuild)
	if err != nil {
		return m, err
	}

	if t.StateDB != nil && t.StateDB.IsVisited(ref.URL) {
		_ = t.StateDB.Done(ref.URL, 200, EntityGuild)
		m.Skipped++
		emit(&GuildState{GuildID: ref.ID, Status: "skipped"})
		return m, nil
	}

	emit(&GuildState{GuildID: ref.ID, Status: "fetching"})

	raw, code, err := t.Client.FetchGuild(ctx, ref.ID)
	if err != nil {
		m.Failed++
		if t.StateDB != nil {
			t.StateDB.Fail(ref.URL, err.Error())
		}
		emit(&GuildState{GuildID: ref.ID, Status: "failed", Error: err.Error()})
		return m, nil
	}
	if code == 404 || raw == nil {
		m.Skipped++
		if t.StateDB != nil {
			t.StateDB.Done(ref.URL, 404, EntityGuild)
		}
		emit(&GuildState{GuildID: ref.ID, Status: "not_found"})
		return m, nil
	}

	guild := ParseGuild(raw)
	if guild.GuildID == "" {
		guild.GuildID = ref.ID
	}
	guild.FetchedAt = time.Now()

	if err := t.DB.UpsertGuild(guild); err != nil {
		m.Failed++
		return m, err
	}

	// Fetch channels
	channelCount := 0
	rawChannels, _, err := t.Client.FetchGuildChannels(ctx, ref.ID)
	if err == nil {
		for _, rc := range rawChannels {
			ch := ParseChannel(rc, ref.ID)
			if ch.ChannelID == "" {
				continue
			}
			if !isTextChannel(ch.ChannelType) {
				continue
			}
			_ = t.DB.UpsertChannel(ch)
			if t.StateDB != nil {
				t.StateDB.Enqueue(channelQueueURL(ch.ChannelID), EntityChannel, 12)
			}
			channelCount++
		}
	}

	if t.StateDB != nil {
		t.StateDB.Done(ref.URL, 200, EntityGuild)
	}
	m.Fetched++
	m.Channels = channelCount
	emit(&GuildState{GuildID: ref.ID, Name: guild.Name, Status: "done", ChannelsSeen: channelCount})
	return m, nil
}
