package posts

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-mizu/blueprints/cms/pkg/slug"
	"github.com/go-mizu/blueprints/cms/pkg/ulid"
)

var (
	ErrNotFound     = errors.New("post not found")
	ErrMissingTitle = errors.New("title is required")
)

// Service implements the posts API.
type Service struct {
	store Store
}

// NewService creates a new posts service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, authorID string, in *CreateIn) (*Post, error) {
	if in.Title == "" {
		return nil, ErrMissingTitle
	}

	now := time.Now()
	postSlug := in.Slug
	if postSlug == "" {
		postSlug = slug.Generate(in.Title)
	}

	contentFormat := in.ContentFormat
	if contentFormat == "" {
		contentFormat = "markdown"
	}

	status := in.Status
	if status == "" {
		status = "draft"
	}

	visibility := in.Visibility
	if visibility == "" {
		visibility = "public"
	}

	allowComments := true
	if in.AllowComments != nil {
		allowComments = *in.AllowComments
	}

	// Calculate word count and reading time
	wordCount := countWords(in.Content)
	readingTime := wordCount / 200 // ~200 words per minute
	if readingTime < 1 {
		readingTime = 1
	}

	post := &Post{
		ID:              ulid.New(),
		AuthorID:        authorID,
		Title:           in.Title,
		Slug:            postSlug,
		Excerpt:         in.Excerpt,
		Content:         in.Content,
		ContentFormat:   contentFormat,
		FeaturedImageID: in.FeaturedImageID,
		Status:          status,
		Visibility:      visibility,
		Meta:            in.Meta,
		ReadingTime:     readingTime,
		WordCount:       wordCount,
		AllowComments:   allowComments,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.store.Create(ctx, post); err != nil {
		return nil, err
	}

	// Set categories and tags
	if len(in.CategoryIDs) > 0 {
		_ = s.store.SetCategories(ctx, post.ID, in.CategoryIDs)
	}
	if len(in.TagIDs) > 0 {
		_ = s.store.SetTags(ctx, post.ID, in.TagIDs)
	}

	return post, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Post, error) {
	post, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, ErrNotFound
	}
	return post, nil
}

func (s *Service) GetBySlug(ctx context.Context, postSlug string) (*Post, error) {
	post, err := s.store.GetBySlug(ctx, postSlug)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, ErrNotFound
	}
	return post, nil
}

func (s *Service) List(ctx context.Context, in *ListIn) ([]*Post, int, error) {
	if in.Limit <= 0 {
		in.Limit = 20
	}
	if in.OrderBy == "" {
		in.OrderBy = "created_at"
	}
	if in.Order == "" {
		in.Order = "desc"
	}
	return s.store.List(ctx, in)
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Post, error) {
	// Update word count and reading time if content changed
	if in.Content != nil {
		wordCount := countWords(*in.Content)
		readingTime := wordCount / 200
		if readingTime < 1 {
			readingTime = 1
		}
		// Note: Could add word_count and reading_time to UpdateIn if needed
	}

	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}

	// Update categories and tags if provided
	if in.CategoryIDs != nil {
		_ = s.store.SetCategories(ctx, id, in.CategoryIDs)
	}
	if in.TagIDs != nil {
		_ = s.store.SetTags(ctx, id, in.TagIDs)
	}

	return s.store.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *Service) Publish(ctx context.Context, id string) (*Post, error) {
	now := time.Now()
	status := "published"
	if err := s.store.Update(ctx, id, &UpdateIn{
		Status:      &status,
		PublishedAt: &now,
	}); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) Unpublish(ctx context.Context, id string) (*Post, error) {
	status := "draft"
	if err := s.store.Update(ctx, id, &UpdateIn{
		Status: &status,
	}); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) GetCategoryIDs(ctx context.Context, postID string) ([]string, error) {
	return s.store.GetCategoryIDs(ctx, postID)
}

func (s *Service) GetTagIDs(ctx context.Context, postID string) ([]string, error) {
	return s.store.GetTagIDs(ctx, postID)
}

func (s *Service) SetCategories(ctx context.Context, postID string, categoryIDs []string) error {
	return s.store.SetCategories(ctx, postID, categoryIDs)
}

func (s *Service) SetTags(ctx context.Context, postID string, tagIDs []string) error {
	return s.store.SetTags(ctx, postID, tagIDs)
}

// countWords counts the number of words in a string.
func countWords(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	return utf8.RuneCountInString(s) / 5 // Approximate: 5 chars per word
}
