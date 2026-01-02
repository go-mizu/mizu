package questions

import (
	"context"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/qa/feature/tags"
	"github.com/go-mizu/mizu/blueprints/qa/pkg/markdown"
	"github.com/go-mizu/mizu/blueprints/qa/pkg/ulid"
)

// Service implements the questions API.
type Service struct {
	store    Store
	accounts accounts.API
	tags     tags.API
}

// NewService creates a new questions service.
func NewService(store Store, accounts accounts.API, tags tags.API) *Service {
	return &Service{store: store, accounts: accounts, tags: tags}
}

// Create creates a new question.
func (s *Service) Create(ctx context.Context, authorID string, in CreateIn) (*Question, error) {
	if len(in.Title) < TitleMinLen || len(in.Title) > TitleMaxLen {
		return nil, ErrInvalidTitle
	}
	if len(in.Body) == 0 || len(in.Body) > BodyMaxLen {
		return nil, ErrInvalidBody
	}

	html, _ := markdown.RenderSafe(in.Body)

	now := time.Now()
	question := &Question{
		ID:        ulid.New(),
		AuthorID:  authorID,
		Title:     in.Title,
		Body:      in.Body,
		BodyHTML:  html,
		Score:     0,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Create(ctx, question); err != nil {
		return nil, err
	}

	if len(in.Tags) > 0 {
		tagNames := normalizeTags(in.Tags)
		_ = s.tags.UpsertBatch(ctx, tagNames)
		_ = s.store.SetTags(ctx, question.ID, tagNames)
		for _, tag := range tagNames {
			_ = s.tags.IncrementQuestionCount(ctx, tag, 1)
		}
	}

	question.Tags, _ = s.store.GetTags(ctx, question.ID)
	question.Author, _ = s.accounts.GetByID(ctx, authorID)

	return question, nil
}

// GetByID retrieves a question.
func (s *Service) GetByID(ctx context.Context, id string) (*Question, error) {
	question, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	question.Tags, _ = s.store.GetTags(ctx, question.ID)
	question.Author, _ = s.accounts.GetByID(ctx, question.AuthorID)
	return question, nil
}

// GetByIDs retrieves questions by IDs.
func (s *Service) GetByIDs(ctx context.Context, ids []string) (map[string]*Question, error) {
	return s.store.GetByIDs(ctx, ids)
}

// Update updates a question.
func (s *Service) Update(ctx context.Context, id string, in UpdateIn) (*Question, error) {
	question, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if question.IsClosed {
		return nil, ErrClosed
	}

	if in.Title != nil {
		if len(*in.Title) < TitleMinLen || len(*in.Title) > TitleMaxLen {
			return nil, ErrInvalidTitle
		}
		question.Title = *in.Title
	}
	if in.Body != nil {
		if len(*in.Body) == 0 || len(*in.Body) > BodyMaxLen {
			return nil, ErrInvalidBody
		}
		question.Body = *in.Body
		question.BodyHTML, _ = markdown.RenderSafe(*in.Body)
	}
	if len(in.Tags) > 0 {
		tagNames := normalizeTags(in.Tags)
		_ = s.tags.UpsertBatch(ctx, tagNames)
		_ = s.store.SetTags(ctx, question.ID, tagNames)
	}

	question.UpdatedAt = time.Now()
	if err := s.store.Update(ctx, question); err != nil {
		return nil, err
	}

	question.Tags, _ = s.store.GetTags(ctx, question.ID)
	question.Author, _ = s.accounts.GetByID(ctx, question.AuthorID)
	return question, nil
}

func normalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" {
			continue
		}
		out = append(out, tag)
	}
	return out
}

// Delete deletes a question.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// IncrementViews increments view count.
func (s *Service) IncrementViews(ctx context.Context, id string) error {
	return s.store.IncrementViews(ctx, id)
}

// List lists questions.
func (s *Service) List(ctx context.Context, opts ListOpts) ([]*Question, error) {
	return s.store.List(ctx, opts)
}

// ListByTag lists questions for a tag.
func (s *Service) ListByTag(ctx context.Context, tag string, opts ListOpts) ([]*Question, error) {
	return s.store.ListByTag(ctx, tag, opts)
}

// ListByAuthor lists questions by author.
func (s *Service) ListByAuthor(ctx context.Context, authorID string, opts ListOpts) ([]*Question, error) {
	return s.store.ListByAuthor(ctx, authorID, opts)
}

// Search searches questions.
func (s *Service) Search(ctx context.Context, query string, limit int) ([]*Question, error) {
	return s.store.Search(ctx, query, limit)
}

// SetAcceptedAnswer sets accepted answer.
func (s *Service) SetAcceptedAnswer(ctx context.Context, id string, answerID string) error {
	return s.store.SetAcceptedAnswer(ctx, id, answerID)
}

// Close closes a question.
func (s *Service) Close(ctx context.Context, id string, reason string) error {
	return s.store.SetClosed(ctx, id, true, reason)
}

// Reopen reopens a question.
func (s *Service) Reopen(ctx context.Context, id string) error {
	return s.store.SetClosed(ctx, id, false, "")
}

// UpdateStats updates question counters.
func (s *Service) UpdateStats(ctx context.Context, id string, answerDelta, commentDelta, favoriteDelta int64) error {
	return s.store.UpdateStats(ctx, id, answerDelta, commentDelta, favoriteDelta)
}

// UpdateScore updates question score.
func (s *Service) UpdateScore(ctx context.Context, id string, delta int64) error {
	return s.store.UpdateScore(ctx, id, delta)
}
