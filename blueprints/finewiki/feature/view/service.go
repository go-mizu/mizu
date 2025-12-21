// feature/view/service.go
package view

import (
	"context"
	"errors"
	"strings"
)

type Service struct {
	store Store
}

func New(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ByID(ctx context.Context, id string) (*Page, error) {
	if s.store == nil {
		return nil, errors.New("view: nil store")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("view: empty id")
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) ByTitle(ctx context.Context, wikiname, title string) (*Page, error) {
	if s.store == nil {
		return nil, errors.New("view: nil store")
	}
	wikiname = strings.TrimSpace(wikiname)
	title = strings.TrimSpace(title)
	if wikiname == "" {
		return nil, errors.New("view: empty wikiname")
	}
	if title == "" {
		return nil, errors.New("view: empty title")
	}
	return s.store.GetByTitle(ctx, wikiname, title)
}
