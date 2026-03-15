package pinterest

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// BoardState is the observable state for a BoardTask.
type BoardState struct {
	URL       string
	BoardID   string
	Status    string
	Error     string
	PinsFound int
}

// BoardMetric is the final result of a BoardTask.
type BoardMetric struct {
	Fetched int
	Skipped int
	Failed  int
	Pages   int
}

// BoardTask fetches all pins from a Pinterest board and stores them in the DB.
type BoardTask struct {
	URL     string // full board URL: https://www.pinterest.com/user/board/
	MaxPins int    // 0 = unlimited
	Client  *Client
	DB      *DB
	StateDB *State // optional; marks visited
}

var _ core.Task[BoardState, BoardMetric] = (*BoardTask)(nil)

func (t *BoardTask) Run(ctx context.Context, emit func(*BoardState)) (BoardMetric, error) {
	var m BoardMetric

	if t.StateDB != nil && t.StateDB.IsVisited(t.URL) {
		m.Skipped++
		emit(&BoardState{URL: t.URL, Status: "skipped"})
		return m, nil
	}

	emit(&BoardState{URL: t.URL, Status: "fetching_board"})

	board, pins, err := t.Client.FetchBoardBootstrap(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&BoardState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	if err := t.DB.UpsertBoard(*board); err != nil {
		m.Failed++
		emit(&BoardState{URL: t.URL, BoardID: board.BoardID, Status: "failed", Error: err.Error()})
		return m, nil
	}

	emit(&BoardState{URL: t.URL, BoardID: board.BoardID, Status: "fetching_pins"})

	var totalPins int
	for _, pin := range pins {
		if ctx.Err() != nil {
			break
		}
		if t.MaxPins > 0 && totalPins >= t.MaxPins {
			break
		}
		if err := t.DB.UpsertPin(pin); err != nil {
			m.Failed++
			continue
		}
		m.Fetched++
		totalPins++
	}
	m.Pages = 1

	if t.StateDB != nil {
		t.StateDB.Done(t.URL, 200, EntityBoard)
	}

	emit(&BoardState{URL: t.URL, BoardID: board.BoardID, Status: "done", PinsFound: totalPins})
	return m, nil
}
