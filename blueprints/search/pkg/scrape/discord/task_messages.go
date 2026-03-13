package discord

import (
	"context"
	"strconv"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

type MessagesState struct {
	ChannelID string
	Before    string
	Status    string
	Error     string
	Count     int
}

type MessagesMetric struct {
	Stored  int
	Skipped int
	Failed  int
	Pages   int
}

type MessagesTask struct {
	// URL is the full queue URL: discord://channels/{id}/messages[?before={id}]
	URL     string
	// GuildID is optional; used to tag stored messages.
	GuildID string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[MessagesState, MessagesMetric] = (*MessagesTask)(nil)

func (t *MessagesTask) Run(ctx context.Context, emit func(*MessagesState)) (MessagesMetric, error) {
	var m MessagesMetric

	ref, err := ParseRef(t.URL, EntityMessagePage)
	if err != nil {
		return m, err
	}

	if t.StateDB != nil && t.StateDB.IsVisited(ref.URL) {
		m.Skipped++
		emit(&MessagesState{ChannelID: ref.ChannelID, Before: ref.Before, Status: "skipped"})
		return m, nil
	}

	emit(&MessagesState{ChannelID: ref.ChannelID, Before: ref.Before, Status: "fetching"})

	msgs, code, err := t.Client.FetchMessages(ctx, ref.ChannelID, ref.Before)
	if err != nil {
		m.Failed++
		if t.StateDB != nil {
			t.StateDB.Fail(ref.URL, err.Error())
		}
		emit(&MessagesState{ChannelID: ref.ChannelID, Status: "failed", Error: err.Error()})
		return m, nil
	}
	if code == 403 || msgs == nil {
		// No access to channel — mark done so we don't retry
		if t.StateDB != nil {
			t.StateDB.Done(ref.URL, code, EntityMessagePage)
		}
		m.Skipped++
		emit(&MessagesState{ChannelID: ref.ChannelID, Status: "no_access"})
		return m, nil
	}

	// Discover oldest message ID for next-page cursor (numeric comparison for snowflakes)
	var oldestID string
	var oldestNum uint64
	seenUsers := make(map[string]struct{})

	for _, raw := range msgs {
		msg := ParseMessage(raw, ref.ChannelID, t.GuildID)
		if msg.MessageID == "" {
			continue
		}
		_ = t.DB.UpsertMessage(msg)
		m.Stored++

		// Track oldest message for pagination cursor — compare as uint64
		if n, err := strconv.ParseUint(msg.MessageID, 10, 64); err == nil {
			if oldestID == "" || n < oldestNum {
				oldestID = msg.MessageID
				oldestNum = n
			}
		}

		// Enqueue author for user profile fetch (deduplicated within this page)
		if msg.AuthorID != "" && t.StateDB != nil {
			if _, seen := seenUsers[msg.AuthorID]; !seen {
				seenUsers[msg.AuthorID] = struct{}{}
				// Store minimal user record from message author data
				if author, ok := raw["author"].(map[string]any); ok {
					u := ParseUser(author)
					if u.UserID != "" {
						_ = t.DB.UpsertUser(u)
						_ = t.StateDB.Enqueue(userQueueURL(u.UserID), EntityUser, 5)
					}
				}
			}
		}
	}

	// If we got a full page, enqueue the next page
	if len(msgs) == 100 && oldestID != "" && t.StateDB != nil {
		nextURL := messagePageQueueURL(ref.ChannelID, oldestID)
		_ = t.StateDB.Enqueue(nextURL, EntityMessagePage, 8)
	}

	if t.StateDB != nil {
		t.StateDB.Done(ref.URL, 200, EntityMessagePage)
	}
	m.Pages++
	emit(&MessagesState{ChannelID: ref.ChannelID, Before: ref.Before, Status: "done", Count: m.Stored})
	return m, nil
}
