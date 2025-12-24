package posts

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/social/feature/accounts"
	"github.com/go-mizu/blueprints/social/feature/interactions"
	"github.com/go-mizu/blueprints/social/pkg/ulid"
)

const (
	maxContentLength = 500
)

var (
	hashtagRegex = regexp.MustCompile(`#(\w+)`)
	mentionRegex = regexp.MustCompile(`@(\w+)`)
)

// Service implements the posts API.
type Service struct {
	store        Store
	accounts     accounts.API
	interactions interactions.Store
}

// NewService creates a new posts service.
func NewService(store Store, accountsSvc accounts.API, interactionsStore interactions.Store) *Service {
	return &Service{
		store:        store,
		accounts:     accountsSvc,
		interactions: interactionsStore,
	}
}

// Create creates a new post.
func (s *Service) Create(ctx context.Context, accountID string, in *CreateIn) (*Post, error) {
	content := strings.TrimSpace(in.Content)
	if content == "" && len(in.MediaIDs) == 0 {
		return nil, ErrEmpty
	}
	if len(content) > maxContentLength {
		return nil, ErrTooLong
	}

	visibility := in.Visibility
	if visibility == "" {
		visibility = VisibilityPublic
	}

	// Determine thread ID
	threadID := ""
	if in.ReplyToID != "" {
		parent, err := s.store.GetByID(ctx, in.ReplyToID)
		if err != nil {
			return nil, err
		}
		if parent.ThreadID != "" {
			threadID = parent.ThreadID
		} else {
			threadID = parent.ID
		}
	}

	now := time.Now()
	post := &Post{
		ID:             ulid.New(),
		AccountID:      accountID,
		Content:        content,
		ContentWarning: in.ContentWarning,
		Visibility:     visibility,
		ReplyToID:      in.ReplyToID,
		ThreadID:       threadID,
		QuoteOfID:      in.QuoteOfID,
		Language:       in.Language,
		Sensitive:      in.Sensitive,
		CreatedAt:      now,
	}

	if err := s.store.Insert(ctx, post); err != nil {
		return nil, err
	}

	// Update reply count on parent
	if in.ReplyToID != "" {
		_ = s.store.IncrementRepliesCount(ctx, in.ReplyToID)
	}

	// Update quotes count
	if in.QuoteOfID != "" {
		_ = s.store.IncrementQuotesCount(ctx, in.QuoteOfID)
	}

	// Extract and link hashtags
	hashtags := extractHashtags(content)
	for _, tag := range hashtags {
		hashtagID, err := s.store.UpsertHashtag(ctx, tag)
		if err == nil {
			_ = s.store.LinkPostHashtag(ctx, post.ID, hashtagID)
		}
	}
	post.Hashtags = hashtags

	// Extract and link mentions
	mentions := extractMentions(content)
	for _, username := range mentions {
		acc, err := s.accounts.GetByUsername(ctx, username)
		if err == nil {
			_ = s.store.InsertMention(ctx, post.ID, acc.ID)
		}
	}
	post.Mentions = mentions

	return post, nil
}

// GetByID retrieves a post by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Post, error) {
	return s.store.GetByID(ctx, id)
}

// GetByIDs retrieves multiple posts by IDs.
func (s *Service) GetByIDs(ctx context.Context, ids []string) ([]*Post, error) {
	return s.store.GetByIDs(ctx, ids)
}

// Update updates a post.
func (s *Service) Update(ctx context.Context, accountID, postID string, in *UpdateIn) (*Post, error) {
	post, err := s.store.GetByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	if post.AccountID != accountID {
		return nil, ErrUnauthorized
	}

	// Save edit history
	_ = s.store.InsertEditHistory(ctx, postID, post.Content, post.ContentWarning, post.Sensitive)

	if err := s.store.Update(ctx, postID, in); err != nil {
		return nil, err
	}

	return s.store.GetByID(ctx, postID)
}

// Delete deletes a post.
func (s *Service) Delete(ctx context.Context, accountID, postID string) error {
	post, err := s.store.GetByID(ctx, postID)
	if err != nil {
		return err
	}

	if post.AccountID != accountID {
		return ErrUnauthorized
	}

	// Decrement reply count on parent
	if post.ReplyToID != "" {
		_ = s.store.DecrementRepliesCount(ctx, post.ReplyToID)
	}

	return s.store.Delete(ctx, postID)
}

// GetContext retrieves ancestors and descendants of a post.
func (s *Service) GetContext(ctx context.Context, id string) (*Context, error) {
	ancestors, err := s.store.GetAncestors(ctx, id)
	if err != nil {
		return nil, err
	}

	descendants, err := s.store.GetDescendants(ctx, id, 100)
	if err != nil {
		return nil, err
	}

	return &Context{
		Ancestors:   ancestors,
		Descendants: descendants,
	}, nil
}

// List returns posts matching the given options.
func (s *Service) List(ctx context.Context, opts ListOpts) ([]*Post, error) {
	return s.store.List(ctx, opts)
}

// GetReplies returns replies to a post.
func (s *Service) GetReplies(ctx context.Context, postID string, limit, offset int) ([]*Post, error) {
	return s.store.GetReplies(ctx, postID, limit, offset)
}

// PopulateAccount populates the account field for a post.
func (s *Service) PopulateAccount(ctx context.Context, p *Post) error {
	if p.AccountID == "" {
		return nil
	}

	acc, err := s.accounts.GetByID(ctx, p.AccountID)
	if err != nil {
		return err
	}

	p.Account = acc
	return nil
}

// PopulateAccounts populates accounts for multiple posts.
func (s *Service) PopulateAccounts(ctx context.Context, posts []*Post) error {
	if len(posts) == 0 {
		return nil
	}

	// Collect unique account IDs
	idSet := make(map[string]bool)
	for _, p := range posts {
		idSet[p.AccountID] = true
	}

	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	accs, err := s.accounts.GetByIDs(ctx, ids)
	if err != nil {
		return err
	}

	accountMap := make(map[string]*accounts.Account)
	for _, acc := range accs {
		accountMap[acc.ID] = acc
	}

	for _, p := range posts {
		p.Account = accountMap[p.AccountID]
	}

	return nil
}

// PopulateViewerState populates liked/reposted/bookmarked for a post.
func (s *Service) PopulateViewerState(ctx context.Context, p *Post, viewerID string) error {
	if viewerID == "" || s.interactions == nil {
		return nil
	}

	state, err := s.interactions.GetPostState(ctx, viewerID, p.ID)
	if err != nil {
		return err
	}

	p.Liked = state.Liked
	p.Reposted = state.Reposted
	p.Bookmarked = state.Bookmarked

	return nil
}

// PopulateViewerStates populates viewer states for multiple posts.
func (s *Service) PopulateViewerStates(ctx context.Context, posts []*Post, viewerID string) error {
	if viewerID == "" || s.interactions == nil || len(posts) == 0 {
		return nil
	}

	postIDs := make([]string, len(posts))
	for i, p := range posts {
		postIDs[i] = p.ID
	}

	states, err := s.interactions.GetPostStates(ctx, viewerID, postIDs)
	if err != nil {
		return err
	}

	for _, p := range posts {
		if state, ok := states[p.ID]; ok {
			p.Liked = state.Liked
			p.Reposted = state.Reposted
			p.Bookmarked = state.Bookmarked
		}
	}

	return nil
}

func extractHashtags(content string) []string {
	matches := hashtagRegex.FindAllStringSubmatch(content, -1)
	tags := make([]string, 0, len(matches))
	seen := make(map[string]bool)
	for _, match := range matches {
		tag := strings.ToLower(match[1])
		if !seen[tag] {
			tags = append(tags, tag)
			seen[tag] = true
		}
	}
	return tags
}

func extractMentions(content string) []string {
	matches := mentionRegex.FindAllStringSubmatch(content, -1)
	mentions := make([]string, 0, len(matches))
	seen := make(map[string]bool)
	for _, match := range matches {
		username := strings.ToLower(match[1])
		if !seen[username] {
			mentions = append(mentions, username)
			seen[username] = true
		}
	}
	return mentions
}
