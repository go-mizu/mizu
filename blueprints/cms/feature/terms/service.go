package terms

import (
	"context"
	"regexp"
	"strings"

	"github.com/go-mizu/mizu/blueprints/cms/pkg/ulid"
	"github.com/go-mizu/mizu/blueprints/cms/store/duckdb"
)

var slugRegex = regexp.MustCompile(`[^a-z0-9-]+`)

// Service implements the terms API.
type Service struct {
	terms        *duckdb.TermsStore
	termTaxonomy *duckdb.TermTaxonomyStore
	termmeta     *duckdb.TermmetaStore
}

// NewService creates a new terms service.
func NewService(terms *duckdb.TermsStore, termTaxonomy *duckdb.TermTaxonomyStore, termmeta *duckdb.TermmetaStore) *Service {
	return &Service{
		terms:        terms,
		termTaxonomy: termTaxonomy,
		termmeta:     termmeta,
	}
}

// Create creates a new term.
func (s *Service) Create(ctx context.Context, in CreateIn) (*Term, error) {
	slug := in.Slug
	if slug == "" {
		slug = s.generateSlug(in.Name)
	}

	// Ensure unique slug within taxonomy
	slug, _ = s.ensureUniqueSlug(ctx, slug, in.Taxonomy, "")

	termID := ulid.New()
	term := &duckdb.Term{
		TermID: termID,
		Name:   in.Name,
		Slug:   slug,
	}

	if err := s.terms.Create(ctx, term); err != nil {
		return nil, err
	}

	ttID := ulid.New()
	tt := &duckdb.TermTaxonomy{
		TermTaxonomyID: ttID,
		TermID:         termID,
		Taxonomy:       in.Taxonomy,
		Description:    in.Description,
		Parent:         in.Parent,
		Count:          0,
	}

	if err := s.termTaxonomy.Create(ctx, tt); err != nil {
		// Rollback term creation
		_ = s.terms.Delete(ctx, termID)
		return nil, err
	}

	return s.toTerm(term, tt), nil
}

// GetByID retrieves a term by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Term, error) {
	term, err := s.terms.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if term == nil {
		return nil, ErrNotFound
	}

	// Get term taxonomy (we don't know the taxonomy, so we need to find it)
	// For now, get the first one associated with this term
	results, _, err := s.termTaxonomy.ListWithTaxonomy(ctx, duckdb.TermListOpts{
		Include: []string{id},
		Limit:   1,
	})
	if err != nil || len(results) == 0 {
		return nil, ErrNotFound
	}

	return s.toTerm(term, &results[0].TermTaxonomy), nil
}

// GetBySlug retrieves a term by slug and taxonomy.
func (s *Service) GetBySlug(ctx context.Context, slug, taxonomy string) (*Term, error) {
	term, err := s.terms.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if term == nil {
		return nil, ErrNotFound
	}

	tt, err := s.termTaxonomy.GetByTermAndTaxonomy(ctx, term.TermID, taxonomy)
	if err != nil || tt == nil {
		return nil, ErrNotFound
	}

	return s.toTerm(term, tt), nil
}

// Update updates a term.
func (s *Service) Update(ctx context.Context, id string, in UpdateIn) (*Term, error) {
	term, err := s.terms.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if term == nil {
		return nil, ErrNotFound
	}

	// Get term taxonomy
	results, _, err := s.termTaxonomy.ListWithTaxonomy(ctx, duckdb.TermListOpts{
		Include: []string{id},
		Limit:   1,
	})
	if err != nil || len(results) == 0 {
		return nil, ErrNotFound
	}
	tt := &results[0].TermTaxonomy

	if in.Name != nil {
		term.Name = *in.Name
	}

	if in.Slug != nil {
		slug, _ := s.ensureUniqueSlug(ctx, *in.Slug, tt.Taxonomy, id)
		term.Slug = slug
	}

	if in.Description != nil {
		tt.Description = *in.Description
	}

	if in.Parent != nil {
		tt.Parent = *in.Parent
	}

	if err := s.terms.Update(ctx, term); err != nil {
		return nil, err
	}

	if err := s.termTaxonomy.Update(ctx, tt); err != nil {
		return nil, err
	}

	return s.toTerm(term, tt), nil
}

// Delete deletes a term.
func (s *Service) Delete(ctx context.Context, id string, force bool) error {
	term, err := s.terms.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if term == nil {
		return ErrNotFound
	}

	// Get term taxonomy
	results, _, err := s.termTaxonomy.ListWithTaxonomy(ctx, duckdb.TermListOpts{
		Include: []string{id},
		Limit:   1,
	})
	if err == nil && len(results) > 0 {
		// Delete term taxonomy first
		_ = s.termTaxonomy.Delete(ctx, results[0].TermTaxonomyID)
	}

	// Delete term meta
	// Note: would need to implement DeleteAllForTerm

	// Delete term
	return s.terms.Delete(ctx, id)
}

// List lists terms.
func (s *Service) List(ctx context.Context, opts ListOpts) ([]*Term, int, error) {
	storeOpts := duckdb.TermListOpts{
		Taxonomy:  opts.Taxonomy,
		Include:   opts.Include,
		Exclude:   opts.Exclude,
		Parent:    opts.Parent,
		HideEmpty: opts.HideEmpty,
		Search:    opts.Search,
		Slug:      opts.Slug,
		OrderBy:   opts.OrderBy,
		Order:     opts.Order,
		Page:      opts.Page,
		PerPage:   opts.PerPage,
	}

	if storeOpts.PerPage == 0 {
		storeOpts.PerPage = 10
	}

	results, total, err := s.termTaxonomy.ListWithTaxonomy(ctx, storeOpts)
	if err != nil {
		return nil, 0, err
	}

	terms := make([]*Term, 0, len(results))
	for _, r := range results {
		terms = append(terms, s.toTerm(&r.Term, &r.TermTaxonomy))
	}

	return terms, total, nil
}

// GetMeta retrieves a term meta value.
func (s *Service) GetMeta(ctx context.Context, termID, key string) (string, error) {
	return s.termmeta.Get(ctx, termID, key)
}

// SetMeta sets a term meta value.
func (s *Service) SetMeta(ctx context.Context, termID, key, value string) error {
	existing, _ := s.termmeta.Get(ctx, termID, key)
	if existing != "" {
		return s.termmeta.Update(ctx, termID, key, value)
	}
	return s.termmeta.Create(ctx, &duckdb.Termmeta{
		MetaID:    ulid.New(),
		TermID:    termID,
		MetaKey:   key,
		MetaValue: value,
	})
}

// DeleteMeta deletes a term meta value.
func (s *Service) DeleteMeta(ctx context.Context, termID, key string) error {
	return s.termmeta.Delete(ctx, termID, key)
}

// GetTaxonomy returns info about a taxonomy.
func (s *Service) GetTaxonomy(ctx context.Context, taxonomy string) (*TaxonomyInfo, error) {
	taxonomies := s.getTaxonomies()
	for _, t := range taxonomies {
		if t.Slug == taxonomy {
			return t, nil
		}
	}
	return nil, ErrNotFound
}

// GetTaxonomies returns all registered taxonomies.
func (s *Service) GetTaxonomies(ctx context.Context) ([]*TaxonomyInfo, error) {
	return s.getTaxonomies(), nil
}

// GetChildren returns all child terms for a parent.
func (s *Service) GetChildren(ctx context.Context, parentID, taxonomy string) ([]*Term, error) {
	parent := parentID
	results, _, err := s.termTaxonomy.ListWithTaxonomy(ctx, duckdb.TermListOpts{
		Taxonomy: taxonomy,
		Parent:   &parent,
		Limit:    100,
	})
	if err != nil {
		return nil, err
	}

	terms := make([]*Term, 0, len(results))
	for _, r := range results {
		terms = append(terms, s.toTerm(&r.Term, &r.TermTaxonomy))
	}

	return terms, nil
}

// GetAncestors returns all ancestors for a term.
func (s *Service) GetAncestors(ctx context.Context, termID string) ([]*Term, error) {
	var ancestors []*Term
	currentID := termID

	for i := 0; i < 10; i++ { // Max depth
		term, err := s.GetByID(ctx, currentID)
		if err != nil || term == nil || term.Parent == "" {
			break
		}

		parent, err := s.GetByID(ctx, term.Parent)
		if err != nil || parent == nil {
			break
		}

		ancestors = append(ancestors, parent)
		currentID = parent.ID
	}

	return ancestors, nil
}

// Helper methods

func (s *Service) generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = slugRegex.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "term"
	}
	return slug
}

func (s *Service) ensureUniqueSlug(ctx context.Context, slug, taxonomy, excludeID string) (string, error) {
	baseSlug := slug
	counter := 1

	for {
		existing, _ := s.terms.GetBySlug(ctx, slug)
		if existing == nil || existing.TermID == excludeID {
			return slug, nil
		}

		// Check if it's in the same taxonomy
		tt, _ := s.termTaxonomy.GetByTermAndTaxonomy(ctx, existing.TermID, taxonomy)
		if tt == nil {
			return slug, nil
		}

		counter++
		slug = baseSlug + "-" + string(rune('0'+counter))
	}
}

func (s *Service) getTaxonomies() []*TaxonomyInfo {
	return []*TaxonomyInfo{
		{
			Name:         "Categories",
			Slug:         "category",
			Description:  "Post categories",
			Hierarchical: true,
			ShowCloud:    false,
			Types:        []string{"post"},
			RestBase:     "categories",
		},
		{
			Name:         "Tags",
			Slug:         "post_tag",
			Description:  "Post tags",
			Hierarchical: false,
			ShowCloud:    true,
			Types:        []string{"post"},
			RestBase:     "tags",
		},
		{
			Name:         "Navigation Menus",
			Slug:         "nav_menu",
			Description:  "Navigation menus",
			Hierarchical: false,
			ShowCloud:    false,
			Types:        []string{"nav_menu_item"},
			RestBase:     "menus",
		},
	}
}

func (s *Service) toTerm(t *duckdb.Term, tt *duckdb.TermTaxonomy) *Term {
	return &Term{
		ID:          t.TermID,
		Name:        t.Name,
		Slug:        t.Slug,
		Taxonomy:    tt.Taxonomy,
		Description: tt.Description,
		Parent:      tt.Parent,
		Count:       tt.Count,
	}
}
