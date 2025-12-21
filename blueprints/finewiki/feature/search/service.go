// feature/search/service.go
package search

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

func (s *Service) Search(ctx context.Context, q Query) ([]Result, error) {
	if s.store == nil {
		return nil, errors.New("search: nil store")
	}

	q.Text = normalizeQuery(q.Text)
	if q.Text == "" {
		return []Result{}, nil
	}

	if runeLen(q.Text) < 2 {
		return []Result{}, nil
	}

	if q.Limit < 0 {
		return nil, errors.New("search: negative limit")
	}
	if q.Limit == 0 {
		q.Limit = 20
	}
	if q.Limit > 200 {
		q.Limit = 200
	}

	q.WikiName = strings.TrimSpace(q.WikiName)
	q.InLanguage = strings.TrimSpace(q.InLanguage)

	return s.store.Search(ctx, q)
}

func normalizeQuery(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(s))

	space := false
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\r':
			if !space {
				b.WriteByte(' ')
				space = true
			}
		default:
			space = false
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func runeLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
