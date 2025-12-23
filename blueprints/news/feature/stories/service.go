package stories

import (
	"context"
	"time"

	"github.com/go-mizu/mizu/blueprints/news/feature/tags"
	"github.com/go-mizu/mizu/blueprints/news/feature/users"
	"github.com/go-mizu/mizu/blueprints/news/feature/votes"
	"github.com/go-mizu/mizu/blueprints/news/pkg/markdown"
	"github.com/go-mizu/mizu/blueprints/news/pkg/ulid"
)

// Service implements the stories.API interface.
type Service struct {
	store      Store
	usersStore users.Store
	votesStore votes.Store
	tagsStore  tags.Store
}

// NewService creates a new stories service.
func NewService(store Store, usersStore users.Store, votesStore votes.Store, tagsStore tags.Store) *Service {
	return &Service{
		store:      store,
		usersStore: usersStore,
		votesStore: votesStore,
		tagsStore:  tagsStore,
	}
}

// Create creates a new story.
func (s *Service) Create(ctx context.Context, authorID string, in CreateIn) (*Story, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	// Check for duplicate URL
	if in.URL != "" {
		if existing, _ := s.store.GetByURL(ctx, in.URL); existing != nil {
			return nil, ErrDuplicateURL
		}
	}

	// Process tags
	var tagIDs []string
	if len(in.Tags) > 0 {
		existingTags, err := s.tagsStore.GetByNames(ctx, in.Tags)
		if err != nil {
			return nil, err
		}
		for _, tag := range existingTags {
			tagIDs = append(tagIDs, tag.ID)
		}
	}

	story := &Story{
		ID:        ulid.New(),
		AuthorID:  authorID,
		Title:     in.Title,
		URL:       in.URL,
		Domain:    ExtractDomain(in.URL),
		Text:      in.Text,
		Score:     1,
		CreatedAt: time.Now(),
	}

	if story.Text != "" {
		story.TextHTML = markdown.Render(story.Text)
	}

	if err := s.store.Create(ctx, story, tagIDs); err != nil {
		return nil, err
	}

	// Increment tag counts
	for _, tagID := range tagIDs {
		_ = s.tagsStore.IncrementCount(ctx, tagID, 1)
	}

	return story, nil
}

// GetByID retrieves a story by ID.
func (s *Service) GetByID(ctx context.Context, id string, viewerID string) (*Story, error) {
	story, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Populate author
	if author, err := s.usersStore.GetByID(ctx, story.AuthorID); err == nil {
		story.Author = author
	}

	// Populate tags
	if tags, err := s.store.GetTagsForStory(ctx, id); err == nil {
		story.Tags = tags
	}

	// Populate user vote
	if viewerID != "" {
		if vote, _ := s.votesStore.GetByUserAndTarget(ctx, viewerID, votes.TargetStory, id); vote != nil {
			story.UserVote = vote.Value
		}
	}

	return story, nil
}

// Update updates a story.
func (s *Service) Update(ctx context.Context, id string, authorID string, in CreateIn) (*Story, error) {
	story, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if story.AuthorID != authorID {
		return nil, ErrNotFound
	}

	if err := in.Validate(); err != nil {
		return nil, err
	}

	story.Title = in.Title
	story.Text = in.Text
	if story.Text != "" {
		story.TextHTML = markdown.Render(story.Text)
	}

	if err := s.store.Update(ctx, story); err != nil {
		return nil, err
	}

	return story, nil
}

// Delete deletes a story.
func (s *Service) Delete(ctx context.Context, id string, authorID string) error {
	story, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if story.AuthorID != authorID {
		return ErrNotFound
	}

	return s.store.Delete(ctx, id)
}

// Vote adds an upvote to a story.
func (s *Service) Vote(ctx context.Context, storyID, userID string, value int) error {
	// Check if already voted
	if existing, _ := s.votesStore.GetByUserAndTarget(ctx, userID, votes.TargetStory, storyID); existing != nil {
		return votes.ErrAlreadyVoted
	}

	vote := &votes.Vote{
		ID:         ulid.New(),
		UserID:     userID,
		TargetType: votes.TargetStory,
		TargetID:   storyID,
		Value:      1, // Only upvotes
		CreatedAt:  time.Now(),
	}

	if err := s.votesStore.Create(ctx, vote); err != nil {
		return err
	}

	// Update story score
	return s.store.UpdateScore(ctx, storyID, 1)
}

// Unvote removes a vote from a story.
func (s *Service) Unvote(ctx context.Context, storyID, userID string) error {
	if err := s.votesStore.Delete(ctx, userID, votes.TargetStory, storyID); err != nil {
		return err
	}

	return s.store.UpdateScore(ctx, storyID, -1)
}

// List lists stories.
func (s *Service) List(ctx context.Context, in ListIn, viewerID string) ([]*Story, error) {
	if in.Limit <= 0 {
		in.Limit = 30
	}

	stories, err := s.store.List(ctx, in)
	if err != nil {
		return nil, err
	}

	return s.populateStories(ctx, stories, viewerID)
}

// ListByAuthor lists stories by author.
func (s *Service) ListByAuthor(ctx context.Context, authorID string, limit, offset int, viewerID string) ([]*Story, error) {
	if limit <= 0 {
		limit = 30
	}

	stories, err := s.store.ListByAuthor(ctx, authorID, limit, offset)
	if err != nil {
		return nil, err
	}

	return s.populateStories(ctx, stories, viewerID)
}

// UpdateScore updates a story's score.
func (s *Service) UpdateScore(ctx context.Context, id string, delta int64) error {
	return s.store.UpdateScore(ctx, id, delta)
}

// RecalculateHotScores recalculates hot scores for all stories.
func (s *Service) RecalculateHotScores(ctx context.Context) error {
	return s.store.RecalculateHotScores(ctx)
}

func (s *Service) populateStories(ctx context.Context, stories []*Story, viewerID string) ([]*Story, error) {
	if len(stories) == 0 {
		return stories, nil
	}

	// Collect IDs
	authorIDs := make([]string, 0, len(stories))
	storyIDs := make([]string, 0, len(stories))
	for _, story := range stories {
		authorIDs = append(authorIDs, story.AuthorID)
		storyIDs = append(storyIDs, story.ID)
	}

	// Fetch authors
	authors, _ := s.usersStore.GetByIDs(ctx, authorIDs)

	// Fetch tags
	storyTags, _ := s.store.GetTagsForStories(ctx, storyIDs)

	// Fetch user votes
	var userVotes map[string]*votes.Vote
	if viewerID != "" {
		userVotes, _ = s.votesStore.GetByUserAndTargets(ctx, viewerID, votes.TargetStory, storyIDs)
	}

	// Populate
	for _, story := range stories {
		if author, ok := authors[story.AuthorID]; ok {
			story.Author = author
		}
		if tags, ok := storyTags[story.ID]; ok {
			story.Tags = tags
		}
		if vote, ok := userVotes[story.ID]; ok {
			story.UserVote = vote.Value
		}
	}

	return stories, nil
}
