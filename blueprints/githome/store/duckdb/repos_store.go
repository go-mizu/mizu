package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/repos"
)

// ReposStore handles repository data access.
type ReposStore struct {
	db *sql.DB
}

// NewReposStore creates a new repos store.
func NewReposStore(db *sql.DB) *ReposStore {
	return &ReposStore{db: db}
}

func (s *ReposStore) Create(ctx context.Context, r *repos.Repository) error {
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO repositories (node_id, name, full_name, owner_id, owner_type, private, description,
			fork, parent_id, homepage, language, forks_count, stargazers_count, watchers_count, size,
			default_branch, open_issues_count, is_template, has_issues, has_projects, has_wiki, has_pages,
			has_downloads, has_discussions, archived, disabled, visibility, pushed_at,
			allow_rebase_merge, allow_squash_merge, allow_merge_commit, allow_auto_merge,
			delete_branch_on_merge, allow_forking, web_commit_signoff_required,
			license_key, license_name, license_spdx_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38, $39, $40)
		RETURNING id
	`, "", r.Name, r.FullName, r.OwnerID, r.OwnerType, r.Private, r.Description,
		r.Fork, nullInt64(0), r.Homepage, r.Language, r.ForksCount, r.StargazersCount, r.WatchersCount,
		r.Size, r.DefaultBranch, r.OpenIssuesCount, r.IsTemplate, r.HasIssues, r.HasProjects,
		r.HasWiki, r.HasPages, r.HasDownloads, r.HasDiscussions, r.Archived, r.Disabled, r.Visibility,
		nullTime(r.PushedAt), r.AllowRebaseMerge, r.AllowSquashMerge, r.AllowMergeCommit,
		r.AllowAutoMerge, r.DeleteBranchOnMerge, r.AllowForking, r.WebCommitSignoffRequired,
		"", "", "", r.CreatedAt, r.UpdatedAt,
	).Scan(&r.ID)
	if err != nil {
		return err
	}

	r.NodeID = generateNodeID("R", r.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE repositories SET node_id = $1 WHERE id = $2`, r.NodeID, r.ID)
	return err
}

func (s *ReposStore) GetByID(ctx context.Context, id int64) (*repos.Repository, error) {
	r := &repos.Repository{}
	var pushedAt sql.NullTime
	var parentID sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, name, full_name, owner_id, owner_type, private, description,
			fork, parent_id, homepage, language, forks_count, stargazers_count, watchers_count, size,
			default_branch, open_issues_count, is_template, has_issues, has_projects, has_wiki, has_pages,
			has_downloads, has_discussions, archived, disabled, visibility, pushed_at,
			allow_rebase_merge, allow_squash_merge, allow_merge_commit, allow_auto_merge,
			delete_branch_on_merge, allow_forking, web_commit_signoff_required, created_at, updated_at
		FROM repositories WHERE id = $1
	`, id).Scan(&r.ID, &r.NodeID, &r.Name, &r.FullName, &r.OwnerID, &r.OwnerType, &r.Private,
		&r.Description, &r.Fork, &parentID, &r.Homepage, &r.Language, &r.ForksCount, &r.StargazersCount,
		&r.WatchersCount, &r.Size, &r.DefaultBranch, &r.OpenIssuesCount, &r.IsTemplate, &r.HasIssues,
		&r.HasProjects, &r.HasWiki, &r.HasPages, &r.HasDownloads, &r.HasDiscussions, &r.Archived,
		&r.Disabled, &r.Visibility, &pushedAt, &r.AllowRebaseMerge, &r.AllowSquashMerge,
		&r.AllowMergeCommit, &r.AllowAutoMerge, &r.DeleteBranchOnMerge, &r.AllowForking,
		&r.WebCommitSignoffRequired, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if pushedAt.Valid {
		r.PushedAt = &pushedAt.Time
	}
	r.Forks = r.ForksCount
	r.Watchers = r.WatchersCount
	r.OpenIssues = r.OpenIssuesCount
	return r, err
}

func (s *ReposStore) GetByOwnerAndName(ctx context.Context, ownerID int64, name string) (*repos.Repository, error) {
	r := &repos.Repository{}
	var pushedAt sql.NullTime
	var parentID sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, name, full_name, owner_id, owner_type, private, description,
			fork, parent_id, homepage, language, forks_count, stargazers_count, watchers_count, size,
			default_branch, open_issues_count, is_template, has_issues, has_projects, has_wiki, has_pages,
			has_downloads, has_discussions, archived, disabled, visibility, pushed_at,
			allow_rebase_merge, allow_squash_merge, allow_merge_commit, allow_auto_merge,
			delete_branch_on_merge, allow_forking, web_commit_signoff_required, created_at, updated_at
		FROM repositories WHERE owner_id = $1 AND name = $2
	`, ownerID, name).Scan(&r.ID, &r.NodeID, &r.Name, &r.FullName, &r.OwnerID, &r.OwnerType, &r.Private,
		&r.Description, &r.Fork, &parentID, &r.Homepage, &r.Language, &r.ForksCount, &r.StargazersCount,
		&r.WatchersCount, &r.Size, &r.DefaultBranch, &r.OpenIssuesCount, &r.IsTemplate, &r.HasIssues,
		&r.HasProjects, &r.HasWiki, &r.HasPages, &r.HasDownloads, &r.HasDiscussions, &r.Archived,
		&r.Disabled, &r.Visibility, &pushedAt, &r.AllowRebaseMerge, &r.AllowSquashMerge,
		&r.AllowMergeCommit, &r.AllowAutoMerge, &r.DeleteBranchOnMerge, &r.AllowForking,
		&r.WebCommitSignoffRequired, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if pushedAt.Valid {
		r.PushedAt = &pushedAt.Time
	}
	r.Forks = r.ForksCount
	r.Watchers = r.WatchersCount
	r.OpenIssues = r.OpenIssuesCount
	return r, err
}

func (s *ReposStore) GetByFullName(ctx context.Context, owner, name string) (*repos.Repository, error) {
	fullName := fmt.Sprintf("%s/%s", owner, name)
	r := &repos.Repository{}
	var pushedAt sql.NullTime
	var parentID sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, name, full_name, owner_id, owner_type, private, description,
			fork, parent_id, homepage, language, forks_count, stargazers_count, watchers_count, size,
			default_branch, open_issues_count, is_template, has_issues, has_projects, has_wiki, has_pages,
			has_downloads, has_discussions, archived, disabled, visibility, pushed_at,
			allow_rebase_merge, allow_squash_merge, allow_merge_commit, allow_auto_merge,
			delete_branch_on_merge, allow_forking, web_commit_signoff_required, created_at, updated_at
		FROM repositories WHERE full_name = $1
	`, fullName).Scan(&r.ID, &r.NodeID, &r.Name, &r.FullName, &r.OwnerID, &r.OwnerType, &r.Private,
		&r.Description, &r.Fork, &parentID, &r.Homepage, &r.Language, &r.ForksCount, &r.StargazersCount,
		&r.WatchersCount, &r.Size, &r.DefaultBranch, &r.OpenIssuesCount, &r.IsTemplate, &r.HasIssues,
		&r.HasProjects, &r.HasWiki, &r.HasPages, &r.HasDownloads, &r.HasDiscussions, &r.Archived,
		&r.Disabled, &r.Visibility, &pushedAt, &r.AllowRebaseMerge, &r.AllowSquashMerge,
		&r.AllowMergeCommit, &r.AllowAutoMerge, &r.DeleteBranchOnMerge, &r.AllowForking,
		&r.WebCommitSignoffRequired, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if pushedAt.Valid {
		r.PushedAt = &pushedAt.Time
	}
	r.Forks = r.ForksCount
	r.Watchers = r.WatchersCount
	r.OpenIssues = r.OpenIssuesCount
	return r, err
}

func (s *ReposStore) Update(ctx context.Context, id int64, in *repos.UpdateIn) error {
	// If name is being changed, we also need to update full_name
	// First get current repo to find owner
	if in.Name != nil {
		var ownerType, currentFullName string
		var ownerID int64
		err := s.db.QueryRowContext(ctx, `SELECT owner_id, owner_type, full_name FROM repositories WHERE id = $1`, id).Scan(&ownerID, &ownerType, &currentFullName)
		if err != nil {
			return err
		}
		// Extract owner login from current full_name
		parts := strings.SplitN(currentFullName, "/", 2)
		if len(parts) == 2 {
			newFullName := parts[0] + "/" + *in.Name
			_, err = s.db.ExecContext(ctx, `UPDATE repositories SET name = $2, full_name = $3, updated_at = $4 WHERE id = $1`,
				id, *in.Name, newFullName, time.Now())
			if err != nil {
				return err
			}
		}
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE repositories SET
			description = COALESCE($2, description),
			homepage = COALESCE($3, homepage),
			private = COALESCE($4, private),
			visibility = COALESCE($5, visibility),
			has_issues = COALESCE($6, has_issues),
			has_projects = COALESCE($7, has_projects),
			has_wiki = COALESCE($8, has_wiki),
			has_discussions = COALESCE($9, has_discussions),
			is_template = COALESCE($10, is_template),
			default_branch = COALESCE($11, default_branch),
			allow_squash_merge = COALESCE($12, allow_squash_merge),
			allow_merge_commit = COALESCE($13, allow_merge_commit),
			allow_rebase_merge = COALESCE($14, allow_rebase_merge),
			allow_auto_merge = COALESCE($15, allow_auto_merge),
			delete_branch_on_merge = COALESCE($16, delete_branch_on_merge),
			allow_forking = COALESCE($17, allow_forking),
			archived = COALESCE($18, archived),
			web_commit_signoff_required = COALESCE($19, web_commit_signoff_required),
			updated_at = $20
		WHERE id = $1
	`, id, nullStringPtr(in.Description), nullStringPtr(in.Homepage),
		nullBoolPtr(in.Private), nullStringPtr(in.Visibility), nullBoolPtr(in.HasIssues),
		nullBoolPtr(in.HasProjects), nullBoolPtr(in.HasWiki), nullBoolPtr(in.HasDiscussions),
		nullBoolPtr(in.IsTemplate), nullStringPtr(in.DefaultBranch), nullBoolPtr(in.AllowSquashMerge),
		nullBoolPtr(in.AllowMergeCommit), nullBoolPtr(in.AllowRebaseMerge), nullBoolPtr(in.AllowAutoMerge),
		nullBoolPtr(in.DeleteBranchOnMerge), nullBoolPtr(in.AllowForking), nullBoolPtr(in.Archived),
		nullBoolPtr(in.WebCommitSignoffRequired), time.Now())
	return err
}

func (s *ReposStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM repositories WHERE id = $1`, id)
	return err
}

func (s *ReposStore) List(ctx context.Context, opts *repos.ListOpts) ([]*repos.Repository, error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	query := `
		SELECT id, node_id, name, full_name, owner_id, owner_type, private, description,
			fork, homepage, language, forks_count, stargazers_count, watchers_count, size,
			default_branch, open_issues_count, is_template, has_issues, has_projects, has_wiki, has_pages,
			has_downloads, has_discussions, archived, disabled, visibility, pushed_at,
			allow_rebase_merge, allow_squash_merge, allow_merge_commit, allow_auto_merge,
			delete_branch_on_merge, allow_forking, web_commit_signoff_required, created_at, updated_at
		FROM repositories ORDER BY created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRepos(rows)
}

func (s *ReposStore) ListByOwner(ctx context.Context, ownerID int64, opts *repos.ListOpts) ([]*repos.Repository, error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	query := `
		SELECT id, node_id, name, full_name, owner_id, owner_type, private, description,
			fork, homepage, language, forks_count, stargazers_count, watchers_count, size,
			default_branch, open_issues_count, is_template, has_issues, has_projects, has_wiki, has_pages,
			has_downloads, has_discussions, archived, disabled, visibility, pushed_at,
			allow_rebase_merge, allow_squash_merge, allow_merge_commit, allow_auto_merge,
			delete_branch_on_merge, allow_forking, web_commit_signoff_required, created_at, updated_at
		FROM repositories WHERE owner_id = $1 ORDER BY created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRepos(rows)
}

func (s *ReposStore) ListForks(ctx context.Context, repoID int64, opts *repos.ListOpts) ([]*repos.Repository, error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	query := `
		SELECT id, node_id, name, full_name, owner_id, owner_type, private, description,
			fork, homepage, language, forks_count, stargazers_count, watchers_count, size,
			default_branch, open_issues_count, is_template, has_issues, has_projects, has_wiki, has_pages,
			has_downloads, has_discussions, archived, disabled, visibility, pushed_at,
			allow_rebase_merge, allow_squash_merge, allow_merge_commit, allow_auto_merge,
			delete_branch_on_merge, allow_forking, web_commit_signoff_required, created_at, updated_at
		FROM repositories WHERE parent_id = $1 ORDER BY created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRepos(rows)
}

func (s *ReposStore) GetTopics(ctx context.Context, repoID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT topic FROM repo_topics WHERE repo_id = $1`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topics []string
	for rows.Next() {
		var topic string
		if err := rows.Scan(&topic); err != nil {
			return nil, err
		}
		topics = append(topics, topic)
	}
	return topics, rows.Err()
}

func (s *ReposStore) SetTopics(ctx context.Context, repoID int64, topics []string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM repo_topics WHERE repo_id = $1`, repoID)
	if err != nil {
		return err
	}

	for _, topic := range topics {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO repo_topics (repo_id, topic) VALUES ($1, $2)
		`, repoID, topic)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ReposStore) GetLanguages(ctx context.Context, repoID int64) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT language, bytes FROM repo_languages WHERE repo_id = $1
	`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	languages := make(map[string]int)
	for rows.Next() {
		var lang string
		var bytes int
		if err := rows.Scan(&lang, &bytes); err != nil {
			return nil, err
		}
		languages[lang] = bytes
	}
	return languages, rows.Err()
}

func (s *ReposStore) SetLanguages(ctx context.Context, repoID int64, languages map[string]int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM repo_languages WHERE repo_id = $1`, repoID)
	if err != nil {
		return err
	}

	for lang, bytes := range languages {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO repo_languages (repo_id, language, bytes) VALUES ($1, $2, $3)
		`, repoID, lang, bytes)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ReposStore) IncrementOpenIssues(ctx context.Context, repoID int64, delta int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE repositories SET open_issues_count = open_issues_count + $2, updated_at = $3 WHERE id = $1
	`, repoID, delta, time.Now())
	return err
}

func (s *ReposStore) IncrementStargazers(ctx context.Context, repoID int64, delta int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE repositories SET stargazers_count = stargazers_count + $2, updated_at = $3 WHERE id = $1
	`, repoID, delta, time.Now())
	return err
}

func (s *ReposStore) IncrementWatchers(ctx context.Context, repoID int64, delta int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE repositories SET watchers_count = watchers_count + $2, updated_at = $3 WHERE id = $1
	`, repoID, delta, time.Now())
	return err
}

func (s *ReposStore) IncrementForks(ctx context.Context, repoID int64, delta int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE repositories SET forks_count = forks_count + $2, updated_at = $3 WHERE id = $1
	`, repoID, delta, time.Now())
	return err
}

// Helper function

func scanRepos(rows *sql.Rows) ([]*repos.Repository, error) {
	var list []*repos.Repository
	for rows.Next() {
		r := &repos.Repository{}
		var pushedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.NodeID, &r.Name, &r.FullName, &r.OwnerID, &r.OwnerType, &r.Private,
			&r.Description, &r.Fork, &r.Homepage, &r.Language, &r.ForksCount, &r.StargazersCount,
			&r.WatchersCount, &r.Size, &r.DefaultBranch, &r.OpenIssuesCount, &r.IsTemplate, &r.HasIssues,
			&r.HasProjects, &r.HasWiki, &r.HasPages, &r.HasDownloads, &r.HasDiscussions, &r.Archived,
			&r.Disabled, &r.Visibility, &pushedAt, &r.AllowRebaseMerge, &r.AllowSquashMerge,
			&r.AllowMergeCommit, &r.AllowAutoMerge, &r.DeleteBranchOnMerge, &r.AllowForking,
			&r.WebCommitSignoffRequired, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		if pushedAt.Valid {
			r.PushedAt = &pushedAt.Time
		}
		r.Forks = r.ForksCount
		r.Watchers = r.WatchersCount
		r.OpenIssues = r.OpenIssuesCount
		list = append(list, r)
	}
	return list, rows.Err()
}
