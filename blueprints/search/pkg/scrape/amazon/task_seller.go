package amazon

import (
	"context"
	"fmt"
	"net/url"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// SellerState is the observable state for a SellerTask.
type SellerState struct {
	URL    string
	Status string
	Error  string
}

// SellerMetric is the final result of a SellerTask.
type SellerMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

// SellerTask fetches and stores an Amazon third-party seller profile page.
type SellerTask struct {
	URL     string
	Client  *Client
	DB      *DB
	StateDB *State
}

var _ core.Task[SellerState, SellerMetric] = (*SellerTask)(nil)

func (t *SellerTask) Run(ctx context.Context, emit func(*SellerState)) (SellerMetric, error) {
	var m SellerMetric

	emit(&SellerState{URL: t.URL, Status: "fetching"})

	// 1. IsVisited check
	if t.StateDB != nil && t.StateDB.IsVisited(t.URL) {
		m.Skipped++
		emit(&SellerState{URL: t.URL, Status: "skipped"})
		return m, nil
	}

	// 2. Extract sellerID from URL (?seller= param)
	sellerID := extractSellerID(t.URL)
	if sellerID == "" {
		m.Failed++
		emit(&SellerState{URL: t.URL, Status: "failed", Error: "cannot extract seller ID"})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, "cannot extract seller ID")
		}
		return m, nil
	}

	// 3. Fetch
	doc, code, err := t.Client.FetchHTML(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&SellerState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 {
		m.Skipped++
		emit(&SellerState{URL: t.URL, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(t.URL, EntitySeller, code)
		}
		return m, nil
	}
	if doc == nil {
		m.Failed++
		msg := fmt.Sprintf("HTTP %d", code)
		emit(&SellerState{URL: t.URL, Status: "failed", Error: msg})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, msg)
		}
		return m, nil
	}

	emit(&SellerState{URL: t.URL, Status: "parsing"})

	// 4. Parse
	seller, err := ParseSeller(doc, sellerID, t.URL)
	if err != nil {
		m.Failed++
		emit(&SellerState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	// 5. DB upsert
	if err := t.DB.UpsertSeller(*seller); err != nil {
		m.Failed++
		emit(&SellerState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	// 6. Mark done
	if t.StateDB != nil {
		t.StateDB.Done(t.URL, EntitySeller, code)
	}

	m.Fetched++
	emit(&SellerState{URL: t.URL, Status: "done"})
	return m, nil
}

// extractSellerID parses the "seller" query parameter from an Amazon seller URL.
func extractSellerID(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Query().Get("seller")
}
