package comments

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/news/feature/stories"
	"github.com/go-mizu/mizu/blueprints/news/feature/users"
	"github.com/go-mizu/mizu/blueprints/news/feature/votes"
	"github.com/go-mizu/mizu/blueprints/news/pkg/markdown"
	"github.com/go-mizu/mizu/blueprints/news/pkg/ulid"
)

// Service implements the comments.API interface.
type Service struct {
	store        Store
	storiesStore stories.Store
	usersStore   users.Store
	votesStore   votes.Store
}

// NewService creates a new comments service.
func NewService(store Store, storiesStore stories.Store, usersStore users.Store, votesStore votes.Store) *Service {
	return &Service{
		store:        store,
		storiesStore: storiesStore,
		usersStore:   usersStore,
		votesStore:   votesStore,
	}
}

// Create creates a new comment.
func (s *Service) Create(ctx context.Context, authorID string, in CreateIn) (*Comment, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	// Verify story exists
	if _, err := s.storiesStore.GetByID(ctx, in.StoryID); err != nil {
		return nil, err
	}

	// Determine depth and path
	var depth int
	var path string
	commentID := ulid.New()

	if in.ParentID != "" {
		parent, err := s.store.GetByID(ctx, in.ParentID)
		if err != nil {
			return nil, err
		}

		if parent.Depth >= MaxDepth {
			return nil, ErrTooDeep
		}

		depth = parent.Depth + 1
		path = fmt.Sprintf("%s/%s", parent.Path, commentID)

		// Increment parent's child count
		_ = s.store.IncrementChildCount(ctx, in.ParentID, 1)
	} else {
		depth = 0
		path = commentID
	}

	comment := &Comment{
		ID:        commentID,
		StoryID:   in.StoryID,
		ParentID:  in.ParentID,
		AuthorID:  authorID,
		Text:      in.Text,
		TextHTML:  markdown.Render(in.Text),
		Score:     1,
		Depth:     depth,
		Path:      path,
		CreatedAt: time.Now(),
	}

	if err := s.store.Create(ctx, comment); err != nil {
		return nil, err
	}

	// Increment story's comment count
	_ = s.storiesStore.IncrementCommentCount(ctx, in.StoryID, 1)

	return comment, nil
}

// GetByID retrieves a comment by ID.
func (s *Service) GetByID(ctx context.Context, id string, viewerID string) (*Comment, error) {
	comment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Populate author
	if author, err := s.usersStore.GetByID(ctx, comment.AuthorID); err == nil {
		comment.Author = author
	}

	// Populate user vote
	if viewerID != "" {
		if vote, _ := s.votesStore.GetByUserAndTarget(ctx, viewerID, votes.TargetComment, id); vote != nil {
			comment.UserVote = vote.Value
		}
	}

	return comment, nil
}

// Update updates a comment.
func (s *Service) Update(ctx context.Context, id string, authorID string, text string) (*Comment, error) {
	comment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if comment.AuthorID != authorID {
		return nil, ErrNotFound
	}

	comment.Text = text
	comment.TextHTML = markdown.Render(text)

	if err := s.store.Update(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

// Delete deletes a comment.
func (s *Service) Delete(ctx context.Context, id string, authorID string) error {
	comment, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if comment.AuthorID != authorID {
		return ErrNotFound
	}

	// Decrement story's comment count
	_ = s.storiesStore.IncrementCommentCount(ctx, comment.StoryID, -1)

	// Decrement parent's child count
	if comment.ParentID != "" {
		_ = s.store.IncrementChildCount(ctx, comment.ParentID, -1)
	}

	return s.store.Delete(ctx, id)
}

// Vote adds an upvote to a comment.
func (s *Service) Vote(ctx context.Context, commentID, userID string, value int) error {
	// Check if already voted
	if existing, _ := s.votesStore.GetByUserAndTarget(ctx, userID, votes.TargetComment, commentID); existing != nil {
		return votes.ErrAlreadyVoted
	}

	vote := &votes.Vote{
		ID:         ulid.New(),
		UserID:     userID,
		TargetType: votes.TargetComment,
		TargetID:   commentID,
		Value:      1, // Only upvotes
		CreatedAt:  time.Now(),
	}

	if err := s.votesStore.Create(ctx, vote); err != nil {
		return err
	}

	// Update comment score
	return s.store.UpdateScore(ctx, commentID, 1)
}

// Unvote removes a vote from a comment.
func (s *Service) Unvote(ctx context.Context, commentID, userID string) error {
	if err := s.votesStore.Delete(ctx, userID, votes.TargetComment, commentID); err != nil {
		return err
	}

	return s.store.UpdateScore(ctx, commentID, -1)
}

// ListByStory lists all comments for a story as a tree.
func (s *Service) ListByStory(ctx context.Context, storyID string, viewerID string) ([]*Comment, error) {
	comments, err := s.store.ListByStory(ctx, storyID)
	if err != nil {
		return nil, err
	}

	if len(comments) == 0 {
		return comments, nil
	}

	// Populate authors and votes
	authorIDs := make([]string, 0, len(comments))
	commentIDs := make([]string, 0, len(comments))
	for _, c := range comments {
		authorIDs = append(authorIDs, c.AuthorID)
		commentIDs = append(commentIDs, c.ID)
	}

	authors, _ := s.usersStore.GetByIDs(ctx, authorIDs)

	var userVotes map[string]*votes.Vote
	if viewerID != "" {
		userVotes, _ = s.votesStore.GetByUserAndTargets(ctx, viewerID, votes.TargetComment, commentIDs)
	}

	for _, c := range comments {
		if author, ok := authors[c.AuthorID]; ok {
			c.Author = author
		}
		if vote, ok := userVotes[c.ID]; ok {
			c.UserVote = vote.Value
		}
	}

	// Build tree structure
	return buildCommentTree(comments), nil
}

// ListByAuthor lists comments by author.
func (s *Service) ListByAuthor(ctx context.Context, authorID string, limit, offset int, viewerID string) ([]*Comment, error) {
	if limit <= 0 {
		limit = 30
	}

	comments, err := s.store.ListByAuthor(ctx, authorID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Populate user votes
	if viewerID != "" && len(comments) > 0 {
		commentIDs := make([]string, 0, len(comments))
		for _, c := range comments {
			commentIDs = append(commentIDs, c.ID)
		}

		userVotes, _ := s.votesStore.GetByUserAndTargets(ctx, viewerID, votes.TargetComment, commentIDs)
		for _, c := range comments {
			if vote, ok := userVotes[c.ID]; ok {
				c.UserVote = vote.Value
			}
		}
	}

	return comments, nil
}

// UpdateScore updates a comment's score.
func (s *Service) UpdateScore(ctx context.Context, id string, delta int64) error {
	return s.store.UpdateScore(ctx, id, delta)
}

// buildCommentTree builds a tree structure from flat comments.
func buildCommentTree(comments []*Comment) []*Comment {
	if len(comments) == 0 {
		return nil
	}

	// Create a map for quick lookup
	byID := make(map[string]*Comment)
	for _, c := range comments {
		byID[c.ID] = c
	}

	// Build tree
	var roots []*Comment
	for _, c := range comments {
		if c.ParentID == "" {
			roots = append(roots, c)
		} else if parent, ok := byID[c.ParentID]; ok {
			parent.Children = append(parent.Children, c)
		}
	}

	return roots
}
