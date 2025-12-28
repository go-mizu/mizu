package duckdb

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/go-mizu/blueprints/githome/feature/search"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// SearchStore handles search data access.
type SearchStore struct {
	db *sql.DB
}

// NewSearchStore creates a new search store.
func NewSearchStore(db *sql.DB) *SearchStore {
	return &SearchStore{db: db}
}

func (s *SearchStore) SearchCode(ctx context.Context, query string, opts *search.SearchCodeOpts) (*search.Result[search.CodeResult], error) {
	// Code search would require a full-text index on file contents
	// For now, return empty results
	return &search.Result[search.CodeResult]{
		TotalCount:        0,
		IncompleteResults: false,
		Items:             []search.CodeResult{},
	}, nil
}

func (s *SearchStore) SearchCommits(ctx context.Context, query string, opts *search.SearchOpts) (*search.Result[search.CommitResult], error) {
	// Commit search would require git integration
	// For now, return empty results
	return &search.Result[search.CommitResult]{
		TotalCount:        0,
		IncompleteResults: false,
		Items:             []search.CommitResult{},
	}, nil
}

func (s *SearchStore) SearchIssues(ctx context.Context, query string, opts *search.SearchIssuesOpts) (*search.Result[search.Issue], error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	// Simple LIKE-based search on title and body
	searchPattern := "%" + strings.ToLower(query) + "%"

	countQuery := `
		SELECT COUNT(*) FROM issues
		WHERE LOWER(title) LIKE $1 OR LOWER(body) LIKE $1`
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, searchPattern).Scan(&totalCount); err != nil {
		return nil, err
	}

	sqlQuery := `
		SELECT i.id, i.node_id, i.number, i.state, i.title, i.body, i.comments, i.created_at, i.updated_at, i.closed_at,
			u.id, u.node_id, u.login, u.name, u.avatar_url, u.type
		FROM issues i
		LEFT JOIN users u ON u.id = i.creator_id
		WHERE LOWER(i.title) LIKE $1 OR LOWER(i.body) LIKE $1
		ORDER BY i.created_at DESC`
	sqlQuery = applyPagination(sqlQuery, page, perPage)

	rows, err := s.db.QueryContext(ctx, sqlQuery, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []search.Issue
	for rows.Next() {
		issue := search.Issue{
			User:   &users.SimpleUser{},
			Labels: []*search.Label{},
		}
		var closedAt sql.NullTime
		if err := rows.Scan(&issue.ID, &issue.NodeID, &issue.Number, &issue.State, &issue.Title, &issue.Body,
			&issue.Comments, &issue.CreatedAt, &issue.UpdatedAt, &closedAt,
			&issue.User.ID, &issue.User.NodeID, &issue.User.Login, &issue.User.Name, &issue.User.AvatarURL, &issue.User.Type); err != nil {
			return nil, err
		}
		if closedAt.Valid {
			issue.ClosedAt = &closedAt.Time
		}
		issue.Score = 1.0
		items = append(items, issue)
	}

	return &search.Result[search.Issue]{
		TotalCount:        totalCount,
		IncompleteResults: false,
		Items:             items,
	}, nil
}

func (s *SearchStore) SearchLabels(ctx context.Context, repoID int64, query string, opts *search.SearchOpts) (*search.Result[search.Label], error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	searchPattern := "%" + strings.ToLower(query) + "%"

	countQuery := `
		SELECT COUNT(*) FROM labels
		WHERE repo_id = $1 AND (LOWER(name) LIKE $2 OR LOWER(description) LIKE $2)`
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, repoID, searchPattern).Scan(&totalCount); err != nil {
		return nil, err
	}

	sqlQuery := `
		SELECT id, node_id, name, description, color, is_default
		FROM labels
		WHERE repo_id = $1 AND (LOWER(name) LIKE $2 OR LOWER(description) LIKE $2)
		ORDER BY name`
	sqlQuery = applyPagination(sqlQuery, page, perPage)

	rows, err := s.db.QueryContext(ctx, sqlQuery, repoID, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []search.Label
	for rows.Next() {
		label := search.Label{}
		if err := rows.Scan(&label.ID, &label.NodeID, &label.Name, &label.Description, &label.Color, &label.Default); err != nil {
			return nil, err
		}
		items = append(items, label)
	}

	return &search.Result[search.Label]{
		TotalCount:        totalCount,
		IncompleteResults: false,
		Items:             items,
	}, nil
}

func (s *SearchStore) SearchRepositories(ctx context.Context, query string, opts *search.SearchReposOpts) (*search.Result[search.Repository], error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	searchPattern := "%" + strings.ToLower(query) + "%"

	countQuery := `
		SELECT COUNT(*) FROM repositories
		WHERE LOWER(name) LIKE $1 OR LOWER(full_name) LIKE $1 OR LOWER(description) LIKE $1`
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, searchPattern).Scan(&totalCount); err != nil {
		return nil, err
	}

	sqlQuery := `
		SELECT r.id, r.node_id, r.name, r.full_name, r.private, r.description, r.fork, r.language,
			r.forks_count, r.stargazers_count, r.watchers_count, r.default_branch, r.open_issues_count,
			r.created_at, r.updated_at, r.pushed_at,
			u.id, u.node_id, u.login, u.avatar_url, u.type
		FROM repositories r
		LEFT JOIN users u ON u.id = r.owner_id AND r.owner_type = 'User'
		WHERE LOWER(r.name) LIKE $1 OR LOWER(r.full_name) LIKE $1 OR LOWER(r.description) LIKE $1
		ORDER BY r.stargazers_count DESC`
	sqlQuery = applyPagination(sqlQuery, page, perPage)

	rows, err := s.db.QueryContext(ctx, sqlQuery, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []search.Repository
	for rows.Next() {
		repo := search.Repository{
			Owner:  &users.SimpleUser{},
			Topics: []string{},
		}
		var pushedAt sql.NullTime
		var ownerID sql.NullInt64
		var ownerNodeID, ownerLogin, ownerAvatarURL, ownerType sql.NullString
		if err := rows.Scan(&repo.ID, &repo.NodeID, &repo.Name, &repo.FullName, &repo.Private, &repo.Description,
			&repo.Fork, &repo.Language, &repo.ForksCount, &repo.StargazersCount, &repo.WatchersCount,
			&repo.DefaultBranch, &repo.OpenIssuesCount, &repo.CreatedAt, &repo.UpdatedAt, &pushedAt,
			&ownerID, &ownerNodeID, &ownerLogin, &ownerAvatarURL, &ownerType); err != nil {
			return nil, err
		}
		if pushedAt.Valid {
			repo.PushedAt = &pushedAt.Time
		}
		if ownerID.Valid {
			repo.Owner.ID = ownerID.Int64
			repo.Owner.NodeID = ownerNodeID.String
			repo.Owner.Login = ownerLogin.String
			repo.Owner.AvatarURL = ownerAvatarURL.String
			repo.Owner.Type = ownerType.String
		}
		repo.Score = 1.0
		items = append(items, repo)
	}

	return &search.Result[search.Repository]{
		TotalCount:        totalCount,
		IncompleteResults: false,
		Items:             items,
	}, nil
}

func (s *SearchStore) SearchTopics(ctx context.Context, query string, opts *search.SearchOpts) (*search.Result[search.Topic], error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	searchPattern := "%" + strings.ToLower(query) + "%"

	// Topics are stored in repo_topics table
	countQuery := `
		SELECT COUNT(DISTINCT topic) FROM repo_topics
		WHERE LOWER(topic) LIKE $1`
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, searchPattern).Scan(&totalCount); err != nil {
		return nil, err
	}

	sqlQuery := `
		SELECT DISTINCT topic FROM repo_topics
		WHERE LOWER(topic) LIKE $1
		ORDER BY topic`
	sqlQuery = applyPagination(sqlQuery, page, perPage)

	rows, err := s.db.QueryContext(ctx, sqlQuery, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []search.Topic
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		items = append(items, search.Topic{
			Name:        name,
			DisplayName: name,
			Score:       1.0,
		})
	}

	return &search.Result[search.Topic]{
		TotalCount:        totalCount,
		IncompleteResults: false,
		Items:             items,
	}, nil
}

func (s *SearchStore) SearchUsers(ctx context.Context, query string, opts *search.SearchUsersOpts) (*search.Result[search.User], error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	searchPattern := "%" + strings.ToLower(query) + "%"

	countQuery := `
		SELECT COUNT(*) FROM users
		WHERE LOWER(login) LIKE $1 OR LOWER(name) LIKE $1`
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, searchPattern).Scan(&totalCount); err != nil {
		return nil, err
	}

	sqlQuery := `
		SELECT id, node_id, login, avatar_url, type, site_admin
		FROM users
		WHERE LOWER(login) LIKE $1 OR LOWER(name) LIKE $1
		ORDER BY login`
	sqlQuery = applyPagination(sqlQuery, page, perPage)

	rows, err := s.db.QueryContext(ctx, sqlQuery, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []search.User
	for rows.Next() {
		user := search.User{}
		if err := rows.Scan(&user.ID, &user.NodeID, &user.Login, &user.AvatarURL, &user.Type, &user.SiteAdmin); err != nil {
			return nil, err
		}
		user.URL = fmt.Sprintf("/api/v3/users/%s", user.Login)
		user.HTMLURL = fmt.Sprintf("/%s", user.Login)
		user.Score = 1.0
		items = append(items, user)
	}

	return &search.Result[search.User]{
		TotalCount:        totalCount,
		IncompleteResults: false,
		Items:             items,
	}, nil
}

// generateNodeID creates a GitHub-compatible node ID.
func generateSearchNodeID(prefix string, id int64) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s_%d", prefix, id)))
}
