package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/cms/pkg/ulid"
)

// UploadsStore handles media upload operations.
type UploadsStore struct {
	db *sql.DB
}

// NewUploadsStore creates a new UploadsStore.
func NewUploadsStore(db *sql.DB) *UploadsStore {
	return &UploadsStore{db: db}
}

// Upload represents an uploaded file.
type Upload struct {
	ID               string
	Filename         string
	OriginalFilename string
	MimeType         string
	Filesize         int64
	Width            *int
	Height           *int
	FocalX           *float64
	FocalY           *float64
	Alt              string
	Caption          string
	Sizes            map[string]ImageSizeInfo
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ImageSizeInfo holds information about a resized image.
type ImageSizeInfo struct {
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Filename string `json:"filename"`
	MimeType string `json:"mimeType"`
	Filesize int64  `json:"filesize"`
}

// Create creates a new upload record.
func (s *UploadsStore) Create(ctx context.Context, upload *Upload) error {
	upload.ID = ulid.New()
	upload.CreatedAt = time.Now()
	upload.UpdatedAt = upload.CreatedAt

	sizesJSON, err := json.Marshal(upload.Sizes)
	if err != nil {
		return fmt.Errorf("marshal sizes: %w", err)
	}

	query := `INSERT INTO media (id, filename, original_filename, mime_type, filesize, width, height, focal_x, focal_y, alt, caption, sizes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = s.db.ExecContext(ctx, query,
		upload.ID, upload.Filename, upload.OriginalFilename, upload.MimeType, upload.Filesize,
		upload.Width, upload.Height, upload.FocalX, upload.FocalY, upload.Alt, upload.Caption,
		string(sizesJSON), upload.CreatedAt, upload.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create upload: %w", err)
	}

	return nil
}

// GetByID retrieves an upload by ID.
func (s *UploadsStore) GetByID(ctx context.Context, id string) (*Upload, error) {
	query := `SELECT id, filename, original_filename, mime_type, filesize, width, height, focal_x, focal_y, alt, caption, sizes, created_at, updated_at
		FROM media WHERE id = ?`

	var upload Upload
	var sizesJSON sql.NullString
	var width, height sql.NullInt64
	var focalX, focalY sql.NullFloat64
	var alt, caption sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&upload.ID, &upload.Filename, &upload.OriginalFilename, &upload.MimeType, &upload.Filesize,
		&width, &height, &focalX, &focalY, &alt, &caption, &sizesJSON,
		&upload.CreatedAt, &upload.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get upload: %w", err)
	}

	if width.Valid {
		w := int(width.Int64)
		upload.Width = &w
	}
	if height.Valid {
		h := int(height.Int64)
		upload.Height = &h
	}
	if focalX.Valid {
		upload.FocalX = &focalX.Float64
	}
	if focalY.Valid {
		upload.FocalY = &focalY.Float64
	}
	if alt.Valid {
		upload.Alt = alt.String
	}
	if caption.Valid {
		upload.Caption = caption.String
	}
	if sizesJSON.Valid {
		if err := json.Unmarshal([]byte(sizesJSON.String), &upload.Sizes); err != nil {
			upload.Sizes = make(map[string]ImageSizeInfo)
		}
	}

	return &upload, nil
}

// Update updates an upload record.
func (s *UploadsStore) Update(ctx context.Context, upload *Upload) error {
	upload.UpdatedAt = time.Now()

	sizesJSON, err := json.Marshal(upload.Sizes)
	if err != nil {
		return fmt.Errorf("marshal sizes: %w", err)
	}

	query := `UPDATE media SET filename = ?, original_filename = ?, mime_type = ?, filesize = ?,
		width = ?, height = ?, focal_x = ?, focal_y = ?, alt = ?, caption = ?, sizes = ?, updated_at = ?
		WHERE id = ?`

	_, err = s.db.ExecContext(ctx, query,
		upload.Filename, upload.OriginalFilename, upload.MimeType, upload.Filesize,
		upload.Width, upload.Height, upload.FocalX, upload.FocalY, upload.Alt, upload.Caption,
		string(sizesJSON), upload.UpdatedAt, upload.ID,
	)
	if err != nil {
		return fmt.Errorf("update upload: %w", err)
	}

	return nil
}

// Delete removes an upload record.
func (s *UploadsStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM media WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete upload: %w", err)
	}
	return nil
}

// Find finds uploads matching the options.
func (s *UploadsStore) Find(ctx context.Context, opts *FindOptions) (*FindResult, error) {
	if opts == nil {
		opts = &FindOptions{}
	}

	if opts.Limit <= 0 {
		opts.Limit = 10
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}

	whereClause, whereArgs := buildWhereClause(opts.Where)
	orderClause := buildOrderClause(opts.Sort)

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM media %s", whereClause)
	var totalDocs int
	if err := s.db.QueryRowContext(ctx, countQuery, whereArgs...).Scan(&totalDocs); err != nil {
		return nil, fmt.Errorf("count uploads: %w", err)
	}

	totalPages := (totalDocs + opts.Limit - 1) / opts.Limit
	offset := (opts.Page - 1) * opts.Limit

	query := fmt.Sprintf(
		"SELECT id, filename, original_filename, mime_type, filesize, width, height, focal_x, focal_y, alt, caption, sizes, created_at, updated_at FROM media %s %s LIMIT ? OFFSET ?",
		whereClause, orderClause,
	)
	args := append(whereArgs, opts.Limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find uploads: %w", err)
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var upload Upload
		var sizesJSON sql.NullString
		var width, height sql.NullInt64
		var focalX, focalY sql.NullFloat64
		var alt, caption sql.NullString

		if err := rows.Scan(
			&upload.ID, &upload.Filename, &upload.OriginalFilename, &upload.MimeType, &upload.Filesize,
			&width, &height, &focalX, &focalY, &alt, &caption, &sizesJSON,
			&upload.CreatedAt, &upload.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan upload: %w", err)
		}

		data := map[string]any{
			"filename":         upload.Filename,
			"originalFilename": upload.OriginalFilename,
			"mimeType":         upload.MimeType,
			"filesize":         upload.Filesize,
		}
		if width.Valid {
			data["width"] = int(width.Int64)
		}
		if height.Valid {
			data["height"] = int(height.Int64)
		}
		if focalX.Valid {
			data["focalX"] = focalX.Float64
		}
		if focalY.Valid {
			data["focalY"] = focalY.Float64
		}
		if alt.Valid {
			data["alt"] = alt.String
		}
		if caption.Valid {
			data["caption"] = caption.String
		}
		if sizesJSON.Valid {
			var sizes map[string]ImageSizeInfo
			if err := json.Unmarshal([]byte(sizesJSON.String), &sizes); err == nil {
				data["sizes"] = sizes
			}
		}

		docs = append(docs, Document{
			ID:        upload.ID,
			Data:      data,
			CreatedAt: upload.CreatedAt,
			UpdatedAt: upload.UpdatedAt,
		})
	}

	result := &FindResult{
		Docs:          docs,
		TotalDocs:     totalDocs,
		Limit:         opts.Limit,
		TotalPages:    totalPages,
		Page:          opts.Page,
		PagingCounter: offset + 1,
		HasPrevPage:   opts.Page > 1,
		HasNextPage:   opts.Page < totalPages,
	}

	if result.HasPrevPage {
		prev := opts.Page - 1
		result.PrevPage = &prev
	}
	if result.HasNextPage {
		next := opts.Page + 1
		result.NextPage = &next
	}

	return result, nil
}
