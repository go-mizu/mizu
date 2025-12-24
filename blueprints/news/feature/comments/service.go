package comments

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/news/feature/users"
	"github.com/go-mizu/mizu/blueprints/news/feature/votes"
)

// Service implements the comments.API interface.
type Service struct {
	store      Store
	usersStore users.Store
	votesStore votes.Store
}

// NewService creates a new comments service.
func NewService(store Store, usersStore users.Store, votesStore votes.Store) *Service {
	return &Service{
		store:      store,
		usersStore: usersStore,
		votesStore: votesStore,
	}
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
