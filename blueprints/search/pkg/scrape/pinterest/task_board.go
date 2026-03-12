package pinterest

import (
	"context"
	"fmt"
	"strings"

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

	// Extract board ID from the HTML page
	boardID, err := t.Client.FetchBoardPage(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&BoardState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	emit(&BoardState{URL: t.URL, BoardID: boardID, Status: "fetching_pins"})

	// Derive source_url for API headers (path only: /username/board-slug/)
	sourceURL := t.URL
	if strings.HasPrefix(sourceURL, "https://www.pinterest.com") {
		sourceURL = sourceURL[len("https://www.pinterest.com"):]
	}
	if !strings.HasSuffix(sourceURL, "/") {
		sourceURL += "/"
	}

	// Upsert a minimal board record — will be enriched if UserTask runs later
	username, slug := ExtractBoardSlug(t.URL)
	board := Board{
		BoardID:  boardID,
		Slug:     slug,
		Username: username,
		URL:      t.URL,
	}
	t.DB.UpsertBoard(board)

	// Paginate through board feed
	var bookmark string
	var totalPins int

	for page := 1; ; page++ {
		if ctx.Err() != nil {
			break
		}

		pins, next, err := t.Client.FetchBoardPins(ctx, boardID, sourceURL, bookmark)
		if err != nil {
			// Non-fatal: store what we have
			fmt.Printf("\n  board page %d error: %v\n", page, err)
			break
		}

		for _, pin := range pins {
			if ctx.Err() != nil {
				break
			}
			if err := t.DB.UpsertPin(pin); err != nil {
				m.Failed++
				continue
			}
			m.Fetched++
			totalPins++
		}

		m.Pages++
		emit(&BoardState{URL: t.URL, BoardID: boardID, Status: "fetching_pins", PinsFound: totalPins})

		if t.MaxPins > 0 && totalPins >= t.MaxPins {
			break
		}
		if isEndBookmark(next) || len(pins) == 0 {
			break
		}
		bookmark = next
	}

	if t.StateDB != nil {
		t.StateDB.Done(t.URL, 200, EntityBoard)
	}

	emit(&BoardState{URL: t.URL, BoardID: boardID, Status: "done", PinsFound: totalPins})
	return m, nil
}
