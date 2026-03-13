package huggingface

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
)

type DB struct {
	db   *sql.DB
	path string
}

func OpenDB(path string) (*DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("open duckdb %s: %w", path, err)
	}
	d := &DB{db: db, path: path}
	if err := d.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return d, nil
}

func (d *DB) initSchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS models (
			repo_id                 VARCHAR PRIMARY KEY,
			author                  VARCHAR,
			sha                     VARCHAR,
			created_at              TIMESTAMP,
			last_modified           TIMESTAMP,
			private                 BOOLEAN,
			gated                   BOOLEAN,
			disabled                BOOLEAN,
			likes                   BIGINT,
			downloads               BIGINT,
			trending_score          BIGINT,
			pipeline_tag            VARCHAR,
			library_name            VARCHAR,
			tags_json               VARCHAR,
			card_data_json          VARCHAR,
			config_json             VARCHAR,
			transformers_info_json  VARCHAR,
			widget_data_json        VARCHAR,
			spaces_json             VARCHAR,
			raw_json                VARCHAR,
			fetched_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS datasets (
			repo_id         VARCHAR PRIMARY KEY,
			author          VARCHAR,
			sha             VARCHAR,
			created_at      TIMESTAMP,
			last_modified   TIMESTAMP,
			private         BOOLEAN,
			gated           BOOLEAN,
			disabled        BOOLEAN,
			likes           BIGINT,
			downloads       BIGINT,
			trending_score  BIGINT,
			description     VARCHAR,
			tags_json       VARCHAR,
			card_data_json  VARCHAR,
			raw_json        VARCHAR,
			fetched_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS spaces (
			repo_id         VARCHAR PRIMARY KEY,
			author          VARCHAR,
			sha             VARCHAR,
			created_at      TIMESTAMP,
			last_modified   TIMESTAMP,
			private         BOOLEAN,
			disabled        BOOLEAN,
			likes           BIGINT,
			sdk             VARCHAR,
			subdomain       VARCHAR,
			tags_json       VARCHAR,
			runtime_json    VARCHAR,
			card_data_json  VARCHAR,
			raw_json        VARCHAR,
			fetched_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS collections (
			slug          VARCHAR PRIMARY KEY,
			namespace     VARCHAR,
			title         VARCHAR,
			description   VARCHAR,
			owner_json    VARCHAR,
			theme         VARCHAR,
			upvotes       BIGINT,
			private       BOOLEAN,
			gating        BOOLEAN,
			last_updated  TIMESTAMP,
			items_json    VARCHAR,
			raw_json      VARCHAR,
			fetched_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS papers (
			paper_id        VARCHAR PRIMARY KEY,
			title           VARCHAR,
			summary         VARCHAR,
			ai_summary      VARCHAR,
			published_at    TIMESTAMP,
			upvotes         BIGINT,
			authors_json    VARCHAR,
			github_repo     VARCHAR,
			project_page    VARCHAR,
			thumbnail_url   VARCHAR,
			raw_json        VARCHAR,
			fetched_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS repo_files (
			entity_type   VARCHAR NOT NULL,
			repo_id       VARCHAR NOT NULL,
			path          VARCHAR NOT NULL,
			size          BIGINT,
			lfs_json      VARCHAR,
			PRIMARY KEY (entity_type, repo_id, path)
		)`,
		`CREATE TABLE IF NOT EXISTS repo_links (
			src_type  VARCHAR NOT NULL,
			src_id    VARCHAR NOT NULL,
			rel       VARCHAR NOT NULL,
			dst_type  VARCHAR NOT NULL,
			dst_id    VARCHAR NOT NULL,
			PRIMARY KEY (src_type, src_id, rel, dst_type, dst_id)
		)`,
		`CREATE TABLE IF NOT EXISTS collection_items (
			collection_slug  VARCHAR NOT NULL,
			item_id          VARCHAR NOT NULL,
			item_type        VARCHAR NOT NULL,
			position         INTEGER,
			author           VARCHAR,
			repo_type        VARCHAR,
			raw_json         VARCHAR,
			PRIMARY KEY (collection_slug, item_id, item_type)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}

func (d *DB) UpsertModel(m Model) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO models (
		repo_id, author, sha, created_at, last_modified, private, gated, disabled,
		likes, downloads, trending_score, pipeline_tag, library_name, tags_json,
		card_data_json, config_json, transformers_info_json, widget_data_json, spaces_json, raw_json, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		m.RepoID, nullStr(m.Author), nullStr(m.SHA), nullTime(m.CreatedAt), nullTime(m.LastModified), m.Private, m.Gated, m.Disabled,
		m.Likes, m.Downloads, m.TrendingScore, nullStr(m.PipelineTag), nullStr(m.LibraryName), nullStr(m.TagsJSON),
		nullStr(m.CardDataJSON), nullStr(m.ConfigJSON), nullStr(m.TransformersInfoJSON), nullStr(m.WidgetDataJSON), nullStr(m.SpacesJSON), nullStr(m.RawJSON), nullTime(m.FetchedAt),
	)
	return err
}

func (d *DB) UpsertDataset(ds Dataset) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO datasets (
		repo_id, author, sha, created_at, last_modified, private, gated, disabled,
		likes, downloads, trending_score, description, tags_json, card_data_json, raw_json, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		ds.RepoID, nullStr(ds.Author), nullStr(ds.SHA), nullTime(ds.CreatedAt), nullTime(ds.LastModified), ds.Private, ds.Gated, ds.Disabled,
		ds.Likes, ds.Downloads, ds.TrendingScore, nullStr(ds.Description), nullStr(ds.TagsJSON), nullStr(ds.CardDataJSON), nullStr(ds.RawJSON), nullTime(ds.FetchedAt),
	)
	return err
}

func (d *DB) UpsertSpace(s Space) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO spaces (
		repo_id, author, sha, created_at, last_modified, private, disabled,
		likes, sdk, subdomain, tags_json, runtime_json, card_data_json, raw_json, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		s.RepoID, nullStr(s.Author), nullStr(s.SHA), nullTime(s.CreatedAt), nullTime(s.LastModified), s.Private, s.Disabled,
		s.Likes, nullStr(s.SDK), nullStr(s.Subdomain), nullStr(s.TagsJSON), nullStr(s.RuntimeJSON), nullStr(s.CardDataJSON), nullStr(s.RawJSON), nullTime(s.FetchedAt),
	)
	return err
}

func (d *DB) UpsertCollection(c Collection) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO collections (
		slug, namespace, title, description, owner_json, theme, upvotes, private,
		gating, last_updated, items_json, raw_json, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		c.Slug, nullStr(c.Namespace), nullStr(c.Title), nullStr(c.Description), nullStr(c.OwnerJSON), nullStr(c.Theme), c.Upvotes, c.Private,
		c.Gating, nullTime(c.LastUpdated), nullStr(c.ItemsJSON), nullStr(c.RawJSON), nullTime(c.FetchedAt),
	)
	return err
}

func (d *DB) UpsertPaper(p Paper) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO papers (
		paper_id, title, summary, ai_summary, published_at, upvotes,
		authors_json, github_repo, project_page, thumbnail_url, raw_json, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.PaperID, nullStr(p.Title), nullStr(p.Summary), nullStr(p.AISummary), nullTime(p.PublishedAt), p.Upvotes,
		nullStr(p.AuthorsJSON), nullStr(p.GitHubRepo), nullStr(p.ProjectPage), nullStr(p.ThumbnailURL), nullStr(p.RawJSON), nullTime(p.FetchedAt),
	)
	return err
}

func (d *DB) ReplaceRepoFiles(entityType, repoID string, files []RepoFile) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM repo_files WHERE entity_type = ? AND repo_id = ?`, entityType, repoID); err != nil {
		return err
	}
	for _, file := range files {
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO repo_files (entity_type, repo_id, path, size, lfs_json) VALUES (?,?,?,?,?)`,
			file.EntityType, file.RepoID, file.Path, nullInt64(file.Size), nullStr(file.LFSJSON),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *DB) ReplaceSourceLinks(srcType, srcID string, links []RepoLink) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM repo_links WHERE src_type = ? AND src_id = ?`, srcType, srcID); err != nil {
		return err
	}
	for _, link := range links {
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO repo_links (src_type, src_id, rel, dst_type, dst_id) VALUES (?,?,?,?,?)`,
			link.SrcType, link.SrcID, link.Rel, link.DstType, link.DstID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *DB) ReplaceCollectionItems(collectionSlug string, items []CollectionItem) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM collection_items WHERE collection_slug = ?`, collectionSlug); err != nil {
		return err
	}
	for _, item := range items {
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO collection_items (collection_slug, item_id, item_type, position, author, repo_type, raw_json) VALUES (?,?,?,?,?,?,?)`,
			item.CollectionSlug, item.ItemID, item.ItemType, item.Position, nullStr(item.Author), nullStr(item.RepoType), nullStr(item.RawJSON),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *DB) GetStats() (DBStats, error) {
	var s DBStats
	d.db.QueryRow(`SELECT COUNT(*) FROM models`).Scan(&s.Models)
	d.db.QueryRow(`SELECT COUNT(*) FROM datasets`).Scan(&s.Datasets)
	d.db.QueryRow(`SELECT COUNT(*) FROM spaces`).Scan(&s.Spaces)
	d.db.QueryRow(`SELECT COUNT(*) FROM collections`).Scan(&s.Collections)
	d.db.QueryRow(`SELECT COUNT(*) FROM papers`).Scan(&s.Papers)
	d.db.QueryRow(`SELECT COUNT(*) FROM repo_files`).Scan(&s.RepoFiles)
	d.db.QueryRow(`SELECT COUNT(*) FROM repo_links`).Scan(&s.RepoLinks)
	if fi, err := os.Stat(d.path); err == nil {
		s.DBSize = fi.Size()
	}
	return s, nil
}

func (d *DB) RecentModels(limit int) ([]Model, error) {
	rows, err := d.db.Query(`SELECT repo_id, author, pipeline_tag FROM models ORDER BY fetched_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Model
	for rows.Next() {
		var m Model
		var author, tag sql.NullString
		if err := rows.Scan(&m.RepoID, &author, &tag); err != nil {
			return out, err
		}
		m.Author = author.String
		m.PipelineTag = tag.String
		out = append(out, m)
	}
	return out, rows.Err()
}

func (d *DB) Close() error { return d.db.Close() }
func (d *DB) Path() string { return d.path }

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullTime(t interface{ IsZero() bool }) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func nullInt64(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}
