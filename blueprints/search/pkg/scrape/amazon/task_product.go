package amazon

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
)

// ProductState is the observable state for a ProductTask.
type ProductState struct {
	URL    string
	Status string
	Error  string
}

// ProductMetric is the final result of a ProductTask.
type ProductMetric struct {
	Fetched int
	Skipped int
	Failed  int
}

// ProductTask fetches and stores a single Amazon product page.
type ProductTask struct {
	URL      string
	Client   *Client
	DB       *DB
	StateDB  *State
	MaxPages int
}

var _ core.Task[ProductState, ProductMetric] = (*ProductTask)(nil)

func (t *ProductTask) Run(ctx context.Context, emit func(*ProductState)) (ProductMetric, error) {
	var m ProductMetric

	emit(&ProductState{URL: t.URL, Status: "fetching"})

	// 1. IsVisited check
	if t.StateDB != nil && t.StateDB.IsVisited(t.URL) {
		m.Skipped++
		emit(&ProductState{URL: t.URL, Status: "skipped"})
		return m, nil
	}

	// 2. Extract ASIN
	asin := ExtractASIN(t.URL)
	if asin == "" {
		m.Failed++
		emit(&ProductState{URL: t.URL, Status: "failed", Error: "cannot extract ASIN"})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, "cannot extract ASIN")
		}
		return m, nil
	}

	// 3. Fetch
	doc, code, err := t.Client.FetchHTML(ctx, t.URL)
	if err != nil {
		m.Failed++
		emit(&ProductState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}
	if code == 404 {
		m.Skipped++
		emit(&ProductState{URL: t.URL, Status: "not_found"})
		if t.StateDB != nil {
			t.StateDB.Done(t.URL, EntityProduct, code)
		}
		return m, nil
	}
	if doc == nil {
		m.Failed++
		msg := fmt.Sprintf("HTTP %d", code)
		emit(&ProductState{URL: t.URL, Status: "failed", Error: msg})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, msg)
		}
		return m, nil
	}

	emit(&ProductState{URL: t.URL, Status: "parsing"})

	// 4. Parse
	product, err := ParseProduct(doc, asin, t.URL)
	if err != nil {
		m.Failed++
		emit(&ProductState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	// 5. DB upsert
	if err := t.DB.UpsertProduct(*product); err != nil {
		m.Failed++
		emit(&ProductState{URL: t.URL, Status: "failed", Error: err.Error()})
		if t.StateDB != nil {
			t.StateDB.Fail(t.URL, err.Error())
		}
		return m, nil
	}

	// 6. Mark done and enqueue discovered links
	if t.StateDB != nil {
		t.StateDB.Done(t.URL, EntityProduct, code)
		t.enqueueLinks(product)
	}

	m.Fetched++
	emit(&ProductState{URL: t.URL, Status: "done"})
	return m, nil
}

func (t *ProductTask) enqueueLinks(p *Product) {
	var items []QueueItem

	// Seller page
	if p.SellerID != "" {
		items = append(items, QueueItem{
			URL:        BaseURL + "/sp?seller=" + p.SellerID,
			EntityType: EntitySeller,
			Priority:   10,
		})
	}

	// Brand/store page
	if p.BrandID != "" {
		items = append(items, QueueItem{
			URL:        BaseURL + "/stores/" + p.BrandID,
			EntityType: EntityBrand,
			Priority:   5,
		})
	}

	// Similar products
	for _, asin := range p.SimilarASINs {
		items = append(items, QueueItem{
			URL:        BaseURL + "/dp/" + asin,
			EntityType: EntityProduct,
			Priority:   10,
		})
	}

	// Variant products
	for _, asin := range p.VariantASINs {
		items = append(items, QueueItem{
			URL:        BaseURL + "/dp/" + asin,
			EntityType: EntityProduct,
			Priority:   10,
		})
	}

	// Reviews
	if p.ASIN != "" {
		items = append(items, QueueItem{
			URL:        BaseURL + "/product-reviews/" + p.ASIN,
			EntityType: EntityReview,
			Priority:   1,
		})
		// Q&A
		items = append(items, QueueItem{
			URL:        BaseURL + "/ask/" + p.ASIN,
			EntityType: EntityQA,
			Priority:   1,
		})
	}

	t.StateDB.EnqueueBatch(items)
}
