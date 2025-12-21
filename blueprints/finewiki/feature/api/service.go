package api

import (
	"context"
	"errors"
	"strings"

	"github.com/go-mizu/blueprints/finewiki/feature/search"
	"github.com/go-mizu/blueprints/finewiki/feature/view"
)

// Ensure Service implements WikiAPI at compile time.
var _ WikiAPI = (*Service)(nil)

// Service implements the WikiAPI contract.
type Service struct {
	view   view.API
	search search.API
}

// New creates a new API service wrapping view and search services.
func New(viewAPI view.API, searchAPI search.API) *Service {
	return &Service{
		view:   viewAPI,
		search: searchAPI,
	}
}

// GetPage retrieves a wiki page by ID or by (wikiname, title).
// The response includes parsed infoboxes and human-readable date formatting.
func (s *Service) GetPage(ctx context.Context, in *GetPageIn) (*view.Page, error) {
	if in == nil {
		return nil, errors.New("missing input")
	}

	id := strings.TrimSpace(in.ID)
	wikiname := strings.TrimSpace(in.WikiName)
	title := strings.TrimSpace(in.Title)

	var p *view.Page
	var err error

	switch {
	case id != "":
		p, err = s.view.ByID(ctx, id)
	case wikiname != "" && title != "":
		p, err = s.view.ByTitle(ctx, wikiname, title)
	default:
		return nil, errors.New("provide either 'id' or both 'wikiname' and 'title'")
	}

	if err != nil {
		return nil, err
	}

	// Enhance the page with parsed data
	_ = p.ParseInfoboxes()
	p.FormatDates()

	return p, nil
}

// Search searches for pages by title prefix with optional filters.
func (s *Service) Search(ctx context.Context, in *SearchIn) (*SearchOut, error) {
	if in == nil {
		return nil, errors.New("missing input")
	}

	results, err := s.search.Search(ctx, search.Query{
		Text:       in.Q,
		WikiName:   in.WikiName,
		InLanguage: in.InLanguage,
		Limit:      in.Limit,
		EnableFTS:  in.EnableFTS,
	})
	if err != nil {
		return nil, err
	}

	return &SearchOut{
		Results: results,
		Count:   len(results),
	}, nil
}
