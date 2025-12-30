package posts

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/cms/pkg/ulid"
	"github.com/go-mizu/mizu/blueprints/cms/store/duckdb"
)

var slugRegex = regexp.MustCompile(`[^a-z0-9-]+`)

// Service implements the posts API.
type Service struct {
	posts            *duckdb.PostsStore
	postmeta         *duckdb.PostmetaStore
	termRelationships *duckdb.TermRelationshipsStore
	termTaxonomy     *duckdb.TermTaxonomyStore
	options          *duckdb.OptionsStore
}

// NewService creates a new posts service.
func NewService(
	posts *duckdb.PostsStore,
	postmeta *duckdb.PostmetaStore,
	termRelationships *duckdb.TermRelationshipsStore,
	termTaxonomy *duckdb.TermTaxonomyStore,
	options *duckdb.OptionsStore,
) *Service {
	return &Service{
		posts:            posts,
		postmeta:         postmeta,
		termRelationships: termRelationships,
		termTaxonomy:     termTaxonomy,
		options:          options,
	}
}

// Create creates a new post.
func (s *Service) Create(ctx context.Context, in CreateIn) (*Post, error) {
	now := time.Now()
	postDate := now
	if in.Date != nil {
		postDate = *in.Date
	}

	postType := in.Type
	if postType == "" {
		postType = TypePost
	}

	status := in.Status
	if status == "" {
		status = StatusDraft
	}

	slug := in.Slug
	if slug == "" {
		slug = s.generateSlug(in.Title)
	}

	// Ensure unique slug
	slug, _ = s.ensureUniqueSlug(ctx, slug, postType, "")

	id := ulid.New()
	guid := "/" + postType + "/" + slug + "/"

	post := &duckdb.Post{
		ID:             id,
		PostAuthor:     in.Author,
		PostDate:       postDate,
		PostDateGmt:    postDate.UTC(),
		PostContent:    in.Content,
		PostTitle:      in.Title,
		PostExcerpt:    in.Excerpt,
		PostStatus:     status,
		CommentStatus:  s.defaultIfEmpty(in.CommentStatus, "open"),
		PingStatus:     s.defaultIfEmpty(in.PingStatus, "open"),
		PostPassword:   in.Password,
		PostName:       slug,
		PostModified:   now,
		PostModifiedGmt: now.UTC(),
		PostParent:     in.Parent,
		Guid:           guid,
		MenuOrder:      in.MenuOrder,
		PostType:       postType,
	}

	if err := s.posts.Create(ctx, post); err != nil {
		return nil, err
	}

	// Set featured media
	if in.FeaturedMedia != "" {
		_ = s.SetMeta(ctx, id, "_thumbnail_id", in.FeaturedMedia)
	}

	// Set format
	if in.Format != "" && in.Format != "standard" {
		_ = s.SetMeta(ctx, id, "_format", in.Format)
	}

	// Set template
	if in.Template != "" {
		_ = s.SetMeta(ctx, id, "_wp_page_template", in.Template)
	}

	// Set sticky
	if in.Sticky {
		_ = s.SetSticky(ctx, id, true)
	}

	// Set categories and tags
	if len(in.Categories) > 0 {
		_ = s.SetTerms(ctx, id, "category", in.Categories)
	}
	if len(in.Tags) > 0 {
		_ = s.SetTerms(ctx, id, "post_tag", in.Tags)
	}

	// Set meta
	for key, value := range in.Meta {
		if v, ok := value.(string); ok {
			_ = s.SetMeta(ctx, id, key, v)
		} else {
			jsonVal, _ := json.Marshal(value)
			_ = s.SetMeta(ctx, id, key, string(jsonVal))
		}
	}

	return s.toPost(ctx, post)
}

// GetByID retrieves a post by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Post, error) {
	post, err := s.posts.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, ErrNotFound
	}
	return s.toPost(ctx, post)
}

// GetBySlug retrieves a post by slug and type.
func (s *Service) GetBySlug(ctx context.Context, slug, postType string) (*Post, error) {
	post, err := s.posts.GetBySlug(ctx, slug, postType)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, ErrNotFound
	}
	return s.toPost(ctx, post)
}

// Update updates a post.
func (s *Service) Update(ctx context.Context, id string, in UpdateIn) (*Post, error) {
	post, err := s.posts.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, ErrNotFound
	}

	now := time.Now()

	if in.Title != nil {
		post.PostTitle = *in.Title
	}
	if in.Content != nil {
		post.PostContent = *in.Content
	}
	if in.Excerpt != nil {
		post.PostExcerpt = *in.Excerpt
	}
	if in.Status != nil {
		post.PostStatus = *in.Status
	}
	if in.Slug != nil {
		slug, _ := s.ensureUniqueSlug(ctx, *in.Slug, post.PostType, id)
		post.PostName = slug
	}
	if in.Author != nil {
		post.PostAuthor = *in.Author
	}
	if in.CommentStatus != nil {
		post.CommentStatus = *in.CommentStatus
	}
	if in.PingStatus != nil {
		post.PingStatus = *in.PingStatus
	}
	if in.Date != nil {
		post.PostDate = *in.Date
		post.PostDateGmt = in.Date.UTC()
	}
	if in.Password != nil {
		post.PostPassword = *in.Password
	}
	if in.Parent != nil {
		post.PostParent = *in.Parent
	}
	if in.MenuOrder != nil {
		post.MenuOrder = *in.MenuOrder
	}

	post.PostModified = now
	post.PostModifiedGmt = now.UTC()

	if err := s.posts.Update(ctx, post); err != nil {
		return nil, err
	}

	// Update featured media
	if in.FeaturedMedia != nil {
		if *in.FeaturedMedia != "" {
			_ = s.SetMeta(ctx, id, "_thumbnail_id", *in.FeaturedMedia)
		} else {
			_ = s.DeleteMeta(ctx, id, "_thumbnail_id")
		}
	}

	// Update format
	if in.Format != nil {
		if *in.Format != "" && *in.Format != "standard" {
			_ = s.SetMeta(ctx, id, "_format", *in.Format)
		} else {
			_ = s.DeleteMeta(ctx, id, "_format")
		}
	}

	// Update template
	if in.Template != nil {
		if *in.Template != "" {
			_ = s.SetMeta(ctx, id, "_wp_page_template", *in.Template)
		} else {
			_ = s.DeleteMeta(ctx, id, "_wp_page_template")
		}
	}

	// Update sticky
	if in.Sticky != nil {
		_ = s.SetSticky(ctx, id, *in.Sticky)
	}

	// Update terms
	if len(in.Categories) > 0 {
		_ = s.SetTerms(ctx, id, "category", in.Categories)
	}
	if len(in.Tags) > 0 {
		_ = s.SetTerms(ctx, id, "post_tag", in.Tags)
	}

	// Update meta
	for key, value := range in.Meta {
		if v, ok := value.(string); ok {
			_ = s.SetMeta(ctx, id, key, v)
		} else {
			jsonVal, _ := json.Marshal(value)
			_ = s.SetMeta(ctx, id, key, string(jsonVal))
		}
	}

	return s.toPost(ctx, post)
}

// Delete deletes a post.
func (s *Service) Delete(ctx context.Context, id string, force bool) error {
	post, err := s.posts.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if post == nil {
		return ErrNotFound
	}

	if !force && post.PostStatus != StatusTrash {
		// Move to trash
		_, err = s.Trash(ctx, id)
		return err
	}

	// Delete meta
	_ = s.postmeta.DeleteAllForPost(ctx, id)

	// Delete term relationships
	_ = s.termRelationships.DeleteByObject(ctx, id)

	// Delete revisions
	revisions, _ := s.posts.GetRevisions(ctx, id)
	for _, rev := range revisions {
		_ = s.posts.Delete(ctx, rev.ID)
	}

	// Delete post
	return s.posts.Delete(ctx, id)
}

// Trash moves a post to trash.
func (s *Service) Trash(ctx context.Context, id string) (*Post, error) {
	post, err := s.posts.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, ErrNotFound
	}

	// Store original status
	_ = s.SetMeta(ctx, id, "_wp_trash_meta_status", post.PostStatus)
	_ = s.SetMeta(ctx, id, "_wp_trash_meta_time", time.Now().Format(time.RFC3339))

	post.PostStatus = StatusTrash
	if err := s.posts.Update(ctx, post); err != nil {
		return nil, err
	}

	return s.toPost(ctx, post)
}

// Restore restores a post from trash.
func (s *Service) Restore(ctx context.Context, id string) (*Post, error) {
	post, err := s.posts.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, ErrNotFound
	}

	if post.PostStatus != StatusTrash {
		return s.toPost(ctx, post)
	}

	// Restore original status
	originalStatus, _ := s.GetMeta(ctx, id, "_wp_trash_meta_status")
	if originalStatus == "" {
		originalStatus = StatusDraft
	}

	post.PostStatus = originalStatus
	if err := s.posts.Update(ctx, post); err != nil {
		return nil, err
	}

	// Clean up trash meta
	_ = s.DeleteMeta(ctx, id, "_wp_trash_meta_status")
	_ = s.DeleteMeta(ctx, id, "_wp_trash_meta_time")

	return s.toPost(ctx, post)
}

// List lists posts.
func (s *Service) List(ctx context.Context, opts ListOpts) ([]*Post, int, error) {
	postType := opts.Type
	if postType == "" {
		postType = TypePost
	}

	storeOpts := duckdb.PostListOpts{
		Page:      opts.Page,
		PerPage:   opts.PerPage,
		Search:    opts.Search,
		After:     opts.After,
		Before:    opts.Before,
		ModifiedAfter: opts.ModifiedAfter,
		ModifiedBefore: opts.ModifiedBefore,
		Author:    opts.Author,
		Include:   opts.Include,
		Exclude:   opts.Exclude,
		Offset:    opts.Offset,
		Order:     opts.Order,
		OrderBy:   opts.OrderBy,
		Slug:      opts.Slug,
		Status:    opts.Status,
		PostType:  []string{postType},
		Parent:    opts.Parent,
	}

	if len(storeOpts.Status) == 0 {
		storeOpts.Status = []string{StatusPublish}
	}

	if storeOpts.PerPage == 0 {
		storeOpts.PerPage = 10
	}

	posts, total, err := s.posts.List(ctx, storeOpts)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*Post, 0, len(posts))
	for _, p := range posts {
		post, err := s.toPost(ctx, p)
		if err != nil {
			continue
		}
		result = append(result, post)
	}

	return result, total, nil
}

// Count returns the number of posts.
func (s *Service) Count(ctx context.Context, postType, status string) (int, error) {
	return s.posts.Count(ctx, postType, status)
}

// CreateRevision creates a revision of a post.
func (s *Service) CreateRevision(ctx context.Context, postID string) (*Revision, error) {
	post, err := s.posts.GetByID(ctx, postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, ErrNotFound
	}

	now := time.Now()
	revision := &duckdb.Post{
		ID:             ulid.New(),
		PostAuthor:     post.PostAuthor,
		PostDate:       now,
		PostDateGmt:    now.UTC(),
		PostContent:    post.PostContent,
		PostTitle:      post.PostTitle,
		PostExcerpt:    post.PostExcerpt,
		PostStatus:     StatusInherit,
		PostName:       postID + "-revision-v1",
		PostModified:   now,
		PostModifiedGmt: now.UTC(),
		PostParent:     postID,
		PostType:       TypeRevision,
	}

	if err := s.posts.Create(ctx, revision); err != nil {
		return nil, err
	}

	return s.toRevision(revision), nil
}

// GetRevisions gets all revisions of a post.
func (s *Service) GetRevisions(ctx context.Context, postID string) ([]*Revision, error) {
	revisions, err := s.posts.GetRevisions(ctx, postID)
	if err != nil {
		return nil, err
	}

	result := make([]*Revision, 0, len(revisions))
	for _, r := range revisions {
		if strings.Contains(r.PostName, "-autosave") {
			continue
		}
		result = append(result, s.toRevision(r))
	}

	return result, nil
}

// GetRevision gets a specific revision.
func (s *Service) GetRevision(ctx context.Context, postID, revisionID string) (*Revision, error) {
	revision, err := s.posts.GetByID(ctx, revisionID)
	if err != nil {
		return nil, err
	}
	if revision == nil || revision.PostParent != postID || revision.PostType != TypeRevision {
		return nil, ErrNotFound
	}
	return s.toRevision(revision), nil
}

// DeleteRevision deletes a revision.
func (s *Service) DeleteRevision(ctx context.Context, postID, revisionID string) error {
	revision, err := s.posts.GetByID(ctx, revisionID)
	if err != nil {
		return err
	}
	if revision == nil || revision.PostParent != postID || revision.PostType != TypeRevision {
		return ErrNotFound
	}
	return s.posts.Delete(ctx, revisionID)
}

// CreateAutosave creates an autosave of a post.
func (s *Service) CreateAutosave(ctx context.Context, postID string, in UpdateIn) (*Revision, error) {
	post, err := s.posts.GetByID(ctx, postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, ErrNotFound
	}

	now := time.Now()

	// Check for existing autosave
	revisions, _ := s.posts.GetRevisions(ctx, postID)
	var autosave *duckdb.Post
	for _, r := range revisions {
		if strings.Contains(r.PostName, "-autosave") {
			autosave = r
			break
		}
	}

	title := post.PostTitle
	if in.Title != nil {
		title = *in.Title
	}
	content := post.PostContent
	if in.Content != nil {
		content = *in.Content
	}
	excerpt := post.PostExcerpt
	if in.Excerpt != nil {
		excerpt = *in.Excerpt
	}

	if autosave != nil {
		autosave.PostTitle = title
		autosave.PostContent = content
		autosave.PostExcerpt = excerpt
		autosave.PostModified = now
		autosave.PostModifiedGmt = now.UTC()
		if err := s.posts.Update(ctx, autosave); err != nil {
			return nil, err
		}
		return s.toRevision(autosave), nil
	}

	autosave = &duckdb.Post{
		ID:             ulid.New(),
		PostAuthor:     post.PostAuthor,
		PostDate:       now,
		PostDateGmt:    now.UTC(),
		PostContent:    content,
		PostTitle:      title,
		PostExcerpt:    excerpt,
		PostStatus:     StatusInherit,
		PostName:       postID + "-autosave-v1",
		PostModified:   now,
		PostModifiedGmt: now.UTC(),
		PostParent:     postID,
		PostType:       TypeRevision,
	}

	if err := s.posts.Create(ctx, autosave); err != nil {
		return nil, err
	}

	return s.toRevision(autosave), nil
}

// GetAutosaves gets all autosaves of a post.
func (s *Service) GetAutosaves(ctx context.Context, postID string) ([]*Revision, error) {
	revisions, err := s.posts.GetRevisions(ctx, postID)
	if err != nil {
		return nil, err
	}

	result := make([]*Revision, 0)
	for _, r := range revisions {
		if strings.Contains(r.PostName, "-autosave") {
			result = append(result, s.toRevision(r))
		}
	}

	return result, nil
}

// GetMeta retrieves a post meta value.
func (s *Service) GetMeta(ctx context.Context, postID, key string) (string, error) {
	return s.postmeta.Get(ctx, postID, key)
}

// SetMeta sets a post meta value.
func (s *Service) SetMeta(ctx context.Context, postID, key, value string) error {
	existing, _ := s.postmeta.Get(ctx, postID, key)
	if existing != "" {
		return s.postmeta.Update(ctx, postID, key, value)
	}
	return s.postmeta.Create(ctx, &duckdb.Postmeta{
		MetaID:    ulid.New(),
		PostID:    postID,
		MetaKey:   key,
		MetaValue: value,
	})
}

// DeleteMeta deletes a post meta value.
func (s *Service) DeleteMeta(ctx context.Context, postID, key string) error {
	return s.postmeta.Delete(ctx, postID, key)
}

// GetAllMeta retrieves all meta for a post.
func (s *Service) GetAllMeta(ctx context.Context, postID string) (map[string]string, error) {
	return s.postmeta.GetAll(ctx, postID)
}

// SetTerms sets the terms for a post in a taxonomy.
func (s *Service) SetTerms(ctx context.Context, postID string, taxonomy string, termIDs []string) error {
	// Get term taxonomy IDs for these terms
	for _, termID := range termIDs {
		tt, err := s.termTaxonomy.GetByTermAndTaxonomy(ctx, termID, taxonomy)
		if err != nil || tt == nil {
			continue
		}

		rel := &duckdb.TermRelationship{
			ObjectID:       postID,
			TermTaxonomyID: tt.TermTaxonomyID,
		}
		if err := s.termRelationships.Create(ctx, rel); err != nil {
			continue
		}

		// Increment count
		_ = s.termTaxonomy.IncrementCount(ctx, tt.TermTaxonomyID, 1)
	}
	return nil
}

// GetTerms gets the term IDs for a post in a taxonomy.
func (s *Service) GetTerms(ctx context.Context, postID string, taxonomy string) ([]string, error) {
	terms, err := s.termRelationships.GetTermsForObject(ctx, postID, taxonomy)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(terms))
	for _, t := range terms {
		ids = append(ids, t.Term.TermID)
	}
	return ids, nil
}

// GetStickyPosts returns all sticky post IDs.
func (s *Service) GetStickyPosts(ctx context.Context) ([]string, error) {
	value, err := s.options.Get(ctx, "sticky_posts")
	if err != nil || value == "" {
		return []string{}, nil
	}

	// Parse JSON array
	var ids []string
	if err := json.Unmarshal([]byte(value), &ids); err != nil {
		return []string{}, nil
	}
	return ids, nil
}

// SetSticky sets whether a post is sticky.
func (s *Service) SetSticky(ctx context.Context, postID string, sticky bool) error {
	stickyPosts, _ := s.GetStickyPosts(ctx)

	// Check if already in list
	found := -1
	for i, id := range stickyPosts {
		if id == postID {
			found = i
			break
		}
	}

	if sticky && found == -1 {
		stickyPosts = append(stickyPosts, postID)
	} else if !sticky && found != -1 {
		stickyPosts = append(stickyPosts[:found], stickyPosts[found+1:]...)
	} else {
		return nil // No change needed
	}

	data, _ := json.Marshal(stickyPosts)
	return s.options.Set(ctx, "sticky_posts", string(data), true)
}

// Helper methods

func (s *Service) generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = slugRegex.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "post"
	}
	return slug
}

func (s *Service) ensureUniqueSlug(ctx context.Context, slug, postType, excludeID string) (string, error) {
	baseSlug := slug
	counter := 1

	for {
		existing, _ := s.posts.GetBySlug(ctx, slug, postType)
		if existing == nil || existing.ID == excludeID {
			return slug, nil
		}
		counter++
		slug = baseSlug + "-" + string(rune('0'+counter))
	}
}

func (s *Service) defaultIfEmpty(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func (s *Service) toPost(ctx context.Context, p *duckdb.Post) (*Post, error) {
	featuredMedia, _ := s.GetMeta(ctx, p.ID, "_thumbnail_id")
	format, _ := s.GetMeta(ctx, p.ID, "_format")
	if format == "" {
		format = "standard"
	}
	template, _ := s.GetMeta(ctx, p.ID, "_wp_page_template")

	categories, _ := s.GetTerms(ctx, p.ID, "category")
	tags, _ := s.GetTerms(ctx, p.ID, "post_tag")

	stickyPosts, _ := s.GetStickyPosts(ctx)
	isSticky := false
	for _, id := range stickyPosts {
		if id == p.ID {
			isSticky = true
			break
		}
	}

	return &Post{
		ID:            p.ID,
		Date:          p.PostDate,
		DateGmt:       p.PostDateGmt,
		GUID:          RenderedField{Rendered: p.Guid},
		Modified:      p.PostModified,
		ModifiedGmt:   p.PostModifiedGmt,
		Slug:          p.PostName,
		Status:        p.PostStatus,
		Type:          p.PostType,
		Title:         RenderedField{Raw: p.PostTitle, Rendered: p.PostTitle},
		Content:       RenderedProtected{Raw: p.PostContent, Rendered: p.PostContent, Protected: p.PostPassword != ""},
		Excerpt:       RenderedProtected{Raw: p.PostExcerpt, Rendered: p.PostExcerpt, Protected: p.PostPassword != ""},
		Author:        p.PostAuthor,
		FeaturedMedia: featuredMedia,
		CommentStatus: p.CommentStatus,
		PingStatus:    p.PingStatus,
		Sticky:        isSticky,
		Template:      template,
		Format:        format,
		Categories:    categories,
		Tags:          tags,
		Parent:        p.PostParent,
		MenuOrder:     p.MenuOrder,
		Password:      p.PostPassword,
	}, nil
}

func (s *Service) toRevision(r *duckdb.Post) *Revision {
	return &Revision{
		ID:      r.ID,
		Author:  r.PostAuthor,
		Date:    r.PostDate,
		DateGmt: r.PostDateGmt,
		Parent:  r.PostParent,
		Title:   RenderedField{Raw: r.PostTitle, Rendered: r.PostTitle},
		Content: RenderedField{Raw: r.PostContent, Rendered: r.PostContent},
		Excerpt: RenderedField{Raw: r.PostExcerpt, Rendered: r.PostExcerpt},
	}
}
