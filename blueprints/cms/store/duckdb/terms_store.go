package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Term represents a WordPress term (category, tag, custom taxonomy term).
type Term struct {
	TermID    string
	Name      string
	Slug      string
	TermGroup int64
}

// TermTaxonomy represents the relationship between a term and a taxonomy.
type TermTaxonomy struct {
	TermTaxonomyID string
	TermID         string
	Taxonomy       string
	Description    string
	Parent         string
	Count          int64
}

// TermWithTaxonomy combines Term and TermTaxonomy for convenience.
type TermWithTaxonomy struct {
	Term
	TermTaxonomy
}

// TermRelationship represents a relationship between an object (post) and a term.
type TermRelationship struct {
	ObjectID       string
	TermTaxonomyID string
	TermOrder      int
}

// TermsStore handles term persistence.
type TermsStore struct {
	db *sql.DB
}

// NewTermsStore creates a new terms store.
func NewTermsStore(db *sql.DB) *TermsStore {
	return &TermsStore{db: db}
}

// Create creates a new term.
func (s *TermsStore) Create(ctx context.Context, t *Term) error {
	query := `INSERT INTO wp_terms (term_id, name, slug, term_group) VALUES ($1, $2, $3, $4)`
	_, err := s.db.ExecContext(ctx, query, t.TermID, t.Name, t.Slug, t.TermGroup)
	return err
}

// GetByID retrieves a term by ID.
func (s *TermsStore) GetByID(ctx context.Context, id string) (*Term, error) {
	query := `SELECT term_id, name, slug, term_group FROM wp_terms WHERE term_id = $1`
	t := &Term{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(&t.TermID, &t.Name, &t.Slug, &t.TermGroup)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

// GetBySlug retrieves a term by slug.
func (s *TermsStore) GetBySlug(ctx context.Context, slug string) (*Term, error) {
	query := `SELECT term_id, name, slug, term_group FROM wp_terms WHERE slug = $1`
	t := &Term{}
	err := s.db.QueryRowContext(ctx, query, slug).Scan(&t.TermID, &t.Name, &t.Slug, &t.TermGroup)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

// Update updates a term.
func (s *TermsStore) Update(ctx context.Context, t *Term) error {
	query := `UPDATE wp_terms SET name = $2, slug = $3, term_group = $4 WHERE term_id = $1`
	_, err := s.db.ExecContext(ctx, query, t.TermID, t.Name, t.Slug, t.TermGroup)
	return err
}

// Delete deletes a term.
func (s *TermsStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM wp_terms WHERE term_id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// TermTaxonomyStore handles term taxonomy persistence.
type TermTaxonomyStore struct {
	db *sql.DB
}

// NewTermTaxonomyStore creates a new term taxonomy store.
func NewTermTaxonomyStore(db *sql.DB) *TermTaxonomyStore {
	return &TermTaxonomyStore{db: db}
}

// Create creates a new term taxonomy.
func (s *TermTaxonomyStore) Create(ctx context.Context, tt *TermTaxonomy) error {
	query := `INSERT INTO wp_term_taxonomy (term_taxonomy_id, term_id, taxonomy, description, parent, count)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := s.db.ExecContext(ctx, query, tt.TermTaxonomyID, tt.TermID, tt.Taxonomy, tt.Description, tt.Parent, tt.Count)
	return err
}

// GetByID retrieves a term taxonomy by ID.
func (s *TermTaxonomyStore) GetByID(ctx context.Context, id string) (*TermTaxonomy, error) {
	query := `SELECT term_taxonomy_id, term_id, taxonomy, description, parent, count
		FROM wp_term_taxonomy WHERE term_taxonomy_id = $1`
	tt := &TermTaxonomy{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&tt.TermTaxonomyID, &tt.TermID, &tt.Taxonomy, &tt.Description, &tt.Parent, &tt.Count,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return tt, err
}

// GetByTermAndTaxonomy retrieves a term taxonomy by term ID and taxonomy.
func (s *TermTaxonomyStore) GetByTermAndTaxonomy(ctx context.Context, termID, taxonomy string) (*TermTaxonomy, error) {
	query := `SELECT term_taxonomy_id, term_id, taxonomy, description, parent, count
		FROM wp_term_taxonomy WHERE term_id = $1 AND taxonomy = $2`
	tt := &TermTaxonomy{}
	err := s.db.QueryRowContext(ctx, query, termID, taxonomy).Scan(
		&tt.TermTaxonomyID, &tt.TermID, &tt.Taxonomy, &tt.Description, &tt.Parent, &tt.Count,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return tt, err
}

// Update updates a term taxonomy.
func (s *TermTaxonomyStore) Update(ctx context.Context, tt *TermTaxonomy) error {
	query := `UPDATE wp_term_taxonomy SET term_id = $2, taxonomy = $3, description = $4, parent = $5, count = $6
		WHERE term_taxonomy_id = $1`
	_, err := s.db.ExecContext(ctx, query, tt.TermTaxonomyID, tt.TermID, tt.Taxonomy, tt.Description, tt.Parent, tt.Count)
	return err
}

// Delete deletes a term taxonomy.
func (s *TermTaxonomyStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM wp_term_taxonomy WHERE term_taxonomy_id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// IncrementCount increments the count for a term taxonomy.
func (s *TermTaxonomyStore) IncrementCount(ctx context.Context, id string, delta int) error {
	query := `UPDATE wp_term_taxonomy SET count = count + $2 WHERE term_taxonomy_id = $1`
	_, err := s.db.ExecContext(ctx, query, id, delta)
	return err
}

// TermListOpts contains options for listing terms.
type TermListOpts struct {
	Taxonomy    string
	Include     []string
	Exclude     []string
	Parent      *string
	HideEmpty   bool
	Search      string
	Slug        []string
	OrderBy     string
	Order       string
	Limit       int
	Offset      int
	Page        int
	PerPage     int
}

// ListWithTaxonomy lists terms with their taxonomy info.
func (s *TermTaxonomyStore) ListWithTaxonomy(ctx context.Context, opts TermListOpts) ([]*TermWithTaxonomy, int, error) {
	var args []interface{}
	var where []string
	argNum := 1

	// Base query joins terms and term_taxonomy
	baseQuery := `SELECT t.term_id, t.name, t.slug, t.term_group,
		tt.term_taxonomy_id, tt.term_id, tt.taxonomy, tt.description, tt.parent, tt.count
		FROM wp_terms t
		JOIN wp_term_taxonomy tt ON t.term_id = tt.term_id`
	countQuery := `SELECT COUNT(*) FROM wp_terms t JOIN wp_term_taxonomy tt ON t.term_id = tt.term_id`

	if opts.Taxonomy != "" {
		where = append(where, fmt.Sprintf("tt.taxonomy = $%d", argNum))
		args = append(args, opts.Taxonomy)
		argNum++
	}

	if opts.Parent != nil {
		where = append(where, fmt.Sprintf("tt.parent = $%d", argNum))
		args = append(args, *opts.Parent)
		argNum++
	}

	if opts.HideEmpty {
		where = append(where, "tt.count > 0")
	}

	if opts.Search != "" {
		where = append(where, fmt.Sprintf("LOWER(t.name) LIKE $%d", argNum))
		args = append(args, "%"+strings.ToLower(opts.Search)+"%")
		argNum++
	}

	if len(opts.Include) > 0 {
		placeholders := make([]string, len(opts.Include))
		for i, id := range opts.Include {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, id)
			argNum++
		}
		where = append(where, "t.term_id IN ("+strings.Join(placeholders, ",")+")")
	}

	if len(opts.Exclude) > 0 {
		placeholders := make([]string, len(opts.Exclude))
		for i, id := range opts.Exclude {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, id)
			argNum++
		}
		where = append(where, "t.term_id NOT IN ("+strings.Join(placeholders, ",")+")")
	}

	if len(opts.Slug) > 0 {
		placeholders := make([]string, len(opts.Slug))
		for i, slug := range opts.Slug {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, slug)
			argNum++
		}
		where = append(where, "t.slug IN ("+strings.Join(placeholders, ",")+")")
	}

	if len(where) > 0 {
		whereClause := " WHERE " + strings.Join(where, " AND ")
		baseQuery += whereClause
		countQuery += whereClause
	}

	// Order
	orderBy := "t.name"
	if opts.OrderBy != "" {
		switch opts.OrderBy {
		case "count":
			orderBy = "tt.count"
		case "slug":
			orderBy = "t.slug"
		case "id":
			orderBy = "t.term_id"
		default:
			orderBy = "t.name"
		}
	}
	order := "ASC"
	if opts.Order != "" {
		order = strings.ToUpper(opts.Order)
	}
	baseQuery += fmt.Sprintf(" ORDER BY %s %s", orderBy, order)

	// Pagination
	limit := opts.Limit
	if opts.PerPage > 0 {
		limit = opts.PerPage
	}
	if limit == 0 {
		limit = 10
	}

	offset := opts.Offset
	if opts.Page > 0 {
		offset = (opts.Page - 1) * limit
	}

	baseQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	// Execute count query
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Execute main query
	rows, err := s.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var terms []*TermWithTaxonomy
	for rows.Next() {
		t := &TermWithTaxonomy{}
		if err := rows.Scan(
			&t.Term.TermID, &t.Name, &t.Slug, &t.TermGroup,
			&t.TermTaxonomyID, &t.TermTaxonomy.TermID, &t.Taxonomy, &t.Description, &t.Parent, &t.Count,
		); err != nil {
			return nil, 0, err
		}
		terms = append(terms, t)
	}

	return terms, total, rows.Err()
}

// TermRelationshipsStore handles term relationships.
type TermRelationshipsStore struct {
	db *sql.DB
}

// NewTermRelationshipsStore creates a new term relationships store.
func NewTermRelationshipsStore(db *sql.DB) *TermRelationshipsStore {
	return &TermRelationshipsStore{db: db}
}

// Create creates a new term relationship.
func (s *TermRelationshipsStore) Create(ctx context.Context, tr *TermRelationship) error {
	query := `INSERT INTO wp_term_relationships (object_id, term_taxonomy_id, term_order)
		VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`
	_, err := s.db.ExecContext(ctx, query, tr.ObjectID, tr.TermTaxonomyID, tr.TermOrder)
	return err
}

// Delete deletes a term relationship.
func (s *TermRelationshipsStore) Delete(ctx context.Context, objectID, termTaxonomyID string) error {
	query := `DELETE FROM wp_term_relationships WHERE object_id = $1 AND term_taxonomy_id = $2`
	_, err := s.db.ExecContext(ctx, query, objectID, termTaxonomyID)
	return err
}

// DeleteByObject deletes all term relationships for an object.
func (s *TermRelationshipsStore) DeleteByObject(ctx context.Context, objectID string) error {
	query := `DELETE FROM wp_term_relationships WHERE object_id = $1`
	_, err := s.db.ExecContext(ctx, query, objectID)
	return err
}

// GetByObject retrieves all term taxonomy IDs for an object.
func (s *TermRelationshipsStore) GetByObject(ctx context.Context, objectID string) ([]string, error) {
	query := `SELECT term_taxonomy_id FROM wp_term_relationships WHERE object_id = $1 ORDER BY term_order`
	rows, err := s.db.QueryContext(ctx, query, objectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetObjectsByTerm retrieves all object IDs for a term taxonomy.
func (s *TermRelationshipsStore) GetObjectsByTerm(ctx context.Context, termTaxonomyID string) ([]string, error) {
	query := `SELECT object_id FROM wp_term_relationships WHERE term_taxonomy_id = $1`
	rows, err := s.db.QueryContext(ctx, query, termTaxonomyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetTermsForObject retrieves all terms with taxonomy for an object.
func (s *TermRelationshipsStore) GetTermsForObject(ctx context.Context, objectID, taxonomy string) ([]*TermWithTaxonomy, error) {
	query := `SELECT t.term_id, t.name, t.slug, t.term_group,
		tt.term_taxonomy_id, tt.term_id, tt.taxonomy, tt.description, tt.parent, tt.count
		FROM wp_terms t
		JOIN wp_term_taxonomy tt ON t.term_id = tt.term_id
		JOIN wp_term_relationships tr ON tt.term_taxonomy_id = tr.term_taxonomy_id
		WHERE tr.object_id = $1 AND tt.taxonomy = $2
		ORDER BY tr.term_order`
	rows, err := s.db.QueryContext(ctx, query, objectID, taxonomy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var terms []*TermWithTaxonomy
	for rows.Next() {
		t := &TermWithTaxonomy{}
		if err := rows.Scan(
			&t.Term.TermID, &t.Name, &t.Slug, &t.TermGroup,
			&t.TermTaxonomyID, &t.TermTaxonomy.TermID, &t.Taxonomy, &t.Description, &t.Parent, &t.Count,
		); err != nil {
			return nil, err
		}
		terms = append(terms, t)
	}
	return terms, rows.Err()
}
