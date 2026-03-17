package kaggle

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
		`CREATE TABLE IF NOT EXISTS datasets (
			id                      BIGINT,
			ref                     VARCHAR PRIMARY KEY,
			owner_ref               VARCHAR,
			owner_name              VARCHAR,
			creator_name            VARCHAR,
			creator_url             VARCHAR,
			title                   VARCHAR,
			subtitle                VARCHAR,
			description             VARCHAR,
			url                     VARCHAR,
			license_name            VARCHAR,
			thumbnail_image_url     VARCHAR,
			download_count          BIGINT,
			view_count              BIGINT,
			vote_count              BIGINT,
			kernel_count            BIGINT,
			topic_count             BIGINT,
			current_version_number  INTEGER,
			usability_rating        DOUBLE,
			total_bytes             BIGINT,
			is_private              BOOLEAN,
			is_featured             BOOLEAN,
			last_updated            TIMESTAMP,
			tags_json               VARCHAR,
			versions_json           VARCHAR,
			raw_json                VARCHAR,
			fetched_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS dataset_files (
			dataset_ref    VARCHAR,
			name           VARCHAR,
			total_bytes    BIGINT,
			creation_date  VARCHAR,
			PRIMARY KEY (dataset_ref, name)
		)`,
		`CREATE TABLE IF NOT EXISTS models (
			id                BIGINT,
			ref               VARCHAR PRIMARY KEY,
			owner_ref         VARCHAR,
			title             VARCHAR,
			subtitle          VARCHAR,
			description       VARCHAR,
			author            VARCHAR,
			author_image_url  VARCHAR,
			url               VARCHAR,
			vote_count        BIGINT,
			update_time       TIMESTAMP,
			is_private        BOOLEAN,
			tags_json         VARCHAR,
			raw_json          VARCHAR,
			fetched_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS model_instances (
			model_ref                  VARCHAR,
			instance_id                BIGINT,
			slug                       VARCHAR,
			framework                  VARCHAR,
			fine_tunable               BOOLEAN,
			overview                   VARCHAR,
			usage                      VARCHAR,
			download_url               VARCHAR,
			version_id                 BIGINT,
			version_number             INTEGER,
			url                        VARCHAR,
			license_name               VARCHAR,
			model_instance_type        VARCHAR,
			external_base_model_url    VARCHAR,
			total_uncompressed_bytes   BIGINT,
			raw_json                   VARCHAR,
			PRIMARY KEY (model_ref, instance_id)
		)`,
		`CREATE TABLE IF NOT EXISTS competitions (
			slug           VARCHAR PRIMARY KEY,
			title          VARCHAR,
			description    VARCHAR,
			url            VARCHAR,
			image_url      VARCHAR,
			raw_meta_json  VARCHAR,
			fetched_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS notebooks (
			ref            VARCHAR PRIMARY KEY,
			owner_ref      VARCHAR,
			slug           VARCHAR,
			title          VARCHAR,
			description    VARCHAR,
			url            VARCHAR,
			image_url      VARCHAR,
			raw_meta_json  VARCHAR,
			fetched_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS profiles (
			handle         VARCHAR PRIMARY KEY,
			display_name   VARCHAR,
			bio            VARCHAR,
			url            VARCHAR,
			image_url      VARCHAR,
			raw_meta_json  VARCHAR,
			fetched_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, stmt := range stmts {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}

func (d *DB) UpsertDataset(item Dataset) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT OR REPLACE INTO datasets (
		id, ref, owner_ref, owner_name, creator_name, creator_url, title, subtitle,
		description, url, license_name, thumbnail_image_url, download_count, view_count,
		vote_count, kernel_count, topic_count, current_version_number, usability_rating,
		total_bytes, is_private, is_featured, last_updated, tags_json, versions_json,
		raw_json, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		item.ID, item.Ref, nullStr(item.OwnerRef), nullStr(item.OwnerName), nullStr(item.CreatorName),
		nullStr(item.CreatorURL), nullStr(item.Title), nullStr(item.Subtitle), nullStr(item.Description),
		nullStr(item.URL), nullStr(item.LicenseName), nullStr(item.ThumbnailImageURL), item.DownloadCount,
		item.ViewCount, item.VoteCount, item.KernelCount, item.TopicCount, item.CurrentVersionNumber,
		nullFloat(item.UsabilityRating), item.TotalBytes, item.IsPrivate, item.IsFeatured, nullTime(item.LastUpdated),
		encodeJSON(item.Tags), nullStr(item.VersionsJSON), nullStr(item.RawJSON), item.FetchedAt,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM dataset_files WHERE dataset_ref = ?`, item.Ref); err != nil {
		return err
	}
	for _, f := range item.Files {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO dataset_files (
			dataset_ref, name, total_bytes, creation_date
		) VALUES (?,?,?,?)`, item.Ref, f.Name, f.TotalBytes, nullStr(f.CreationDate)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *DB) UpsertModel(item Model) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT OR REPLACE INTO models (
		id, ref, owner_ref, title, subtitle, description, author, author_image_url,
		url, vote_count, update_time, is_private, tags_json, raw_json, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		item.ID, item.Ref, nullStr(item.OwnerRef), nullStr(item.Title), nullStr(item.Subtitle),
		nullStr(item.Description), nullStr(item.Author), nullStr(item.AuthorImageURL), nullStr(item.URL),
		item.VoteCount, nullTime(item.UpdateTime), item.IsPrivate, encodeJSON(item.Tags), nullStr(item.RawJSON),
		item.FetchedAt,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM model_instances WHERE model_ref = ?`, item.Ref); err != nil {
		return err
	}
	for _, inst := range item.Instances {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO model_instances (
			model_ref, instance_id, slug, framework, fine_tunable, overview, usage,
			download_url, version_id, version_number, url, license_name,
			model_instance_type, external_base_model_url, total_uncompressed_bytes, raw_json
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			item.Ref, inst.InstanceID, nullStr(inst.Slug), nullStr(inst.Framework), inst.FineTunable,
			nullStr(inst.Overview), nullStr(inst.Usage), nullStr(inst.DownloadURL), inst.VersionID,
			inst.VersionNumber, nullStr(inst.URL), nullStr(inst.LicenseName), nullStr(inst.ModelInstanceType),
			nullStr(inst.ExternalBaseModelURL), inst.TotalUncompressedBytes, nullStr(inst.RawJSON),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *DB) UpsertCompetition(item Competition) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO competitions (
		slug, title, description, url, image_url, raw_meta_json, fetched_at
	) VALUES (?,?,?,?,?,?,?)`,
		item.Slug, nullStr(item.Title), nullStr(item.Description), nullStr(item.URL), nullStr(item.ImageURL),
		nullStr(item.RawMetaJSON), item.FetchedAt,
	)
	return err
}

func (d *DB) UpsertNotebook(item Notebook) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO notebooks (
		ref, owner_ref, slug, title, description, url, image_url, raw_meta_json, fetched_at
	) VALUES (?,?,?,?,?,?,?,?,?)`,
		item.Ref, nullStr(item.OwnerRef), nullStr(item.Slug), nullStr(item.Title), nullStr(item.Description),
		nullStr(item.URL), nullStr(item.ImageURL), nullStr(item.RawMetaJSON), item.FetchedAt,
	)
	return err
}

func (d *DB) UpsertProfile(item Profile) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO profiles (
		handle, display_name, bio, url, image_url, raw_meta_json, fetched_at
	) VALUES (?,?,?,?,?,?,?)`,
		item.Handle, nullStr(item.DisplayName), nullStr(item.Bio), nullStr(item.URL), nullStr(item.ImageURL),
		nullStr(item.RawMetaJSON), item.FetchedAt,
	)
	return err
}

func (d *DB) GetStats() (DBStats, error) {
	var s DBStats
	d.db.QueryRow(`SELECT COUNT(*) FROM datasets`).Scan(&s.Datasets)
	d.db.QueryRow(`SELECT COUNT(*) FROM models`).Scan(&s.Models)
	d.db.QueryRow(`SELECT COUNT(*) FROM competitions`).Scan(&s.Competitions)
	d.db.QueryRow(`SELECT COUNT(*) FROM notebooks`).Scan(&s.Notebooks)
	d.db.QueryRow(`SELECT COUNT(*) FROM profiles`).Scan(&s.Profiles)
	if fi, err := os.Stat(d.path); err == nil {
		s.DBSize = fi.Size()
	}
	return s, nil
}

func (d *DB) RecentDatasets(limit int) ([]Dataset, error) {
	rows, err := d.db.Query(`SELECT ref, title, owner_ref, url FROM datasets ORDER BY fetched_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Dataset
	for rows.Next() {
		var item Dataset
		var title, ownerRef, rawURL sql.NullString
		if err := rows.Scan(&item.Ref, &title, &ownerRef, &rawURL); err != nil {
			return items, err
		}
		item.Title = title.String
		item.OwnerRef = ownerRef.String
		item.URL = rawURL.String
		items = append(items, item)
	}
	return items, rows.Err()
}

func (d *DB) Close() error { return d.db.Close() }
func (d *DB) Path() string { return d.path }

func encodeJSON(v any) any {
	switch x := v.(type) {
	case []Tag:
		if len(x) == 0 {
			return nil
		}
	case []DatasetFile:
		if len(x) == 0 {
			return nil
		}
	case []ModelInstance:
		if len(x) == 0 {
			return nil
		}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return string(b)
}

func nullStr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func nullFloat(f float64) any {
	if f == 0 {
		return nil
	}
	return f
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
