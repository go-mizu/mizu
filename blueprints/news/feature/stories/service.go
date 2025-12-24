package stories

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/news/feature/users"
	"github.com/go-mizu/mizu/blueprints/news/feature/votes"
)

// Service implements the stories.API interface.
type Service struct {
	store      Store
	usersStore users.Store
	votesStore votes.Store
}

// NewService creates a new stories service.
func NewService(store Store, usersStore users.Store, votesStore votes.Store) *Service {
	return &Service{
		store:      store,
		usersStore: usersStore,
		votesStore: votesStore,
	}
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
