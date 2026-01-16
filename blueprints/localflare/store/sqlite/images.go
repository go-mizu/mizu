package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/go-mizu/blueprints/localflare/store"
)

// ImagesStoreImpl implements store.ImagesStore.
type ImagesStoreImpl struct {
	db      *sql.DB
	dataDir string
}

// CreateImage creates a new image entry.
func (s *ImagesStoreImpl) CreateImage(ctx context.Context, image *store.Image) error {
	// Ensure data directory exists
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return err
	}

	meta, _ := json.Marshal(image.Meta)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO images (id, filename, storage_key, size, width, height, format, meta, uploaded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		image.ID, image.Filename, image.StorageKey, image.Size, image.Width, image.Height,
		image.Format, string(meta), image.UploadedAt)
	return err
}

// GetImage retrieves an image by ID.
func (s *ImagesStoreImpl) GetImage(ctx context.Context, id string) (*store.Image, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, filename, storage_key, size, width, height, format, meta, uploaded_at
		FROM images WHERE id = ?`, id)
	return s.scanImage(row)
}

// ListImages lists all images with pagination.
func (s *ImagesStoreImpl) ListImages(ctx context.Context, limit, offset int) ([]*store.Image, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, filename, storage_key, size, width, height, format, meta, uploaded_at
		FROM images ORDER BY uploaded_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []*store.Image
	for rows.Next() {
		image, err := s.scanImage(rows)
		if err != nil {
			return nil, err
		}
		images = append(images, image)
	}
	return images, rows.Err()
}

// DeleteImage deletes an image.
func (s *ImagesStoreImpl) DeleteImage(ctx context.Context, id string) error {
	// First get the image to find its storage key
	image, err := s.GetImage(ctx, id)
	if err == nil && image.StorageKey != "" {
		// Delete the file from disk
		filePath := filepath.Join(s.dataDir, image.StorageKey)
		os.Remove(filePath)
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM images WHERE id = ?`, id)
	return err
}

// CreateVariant creates a new image variant.
func (s *ImagesStoreImpl) CreateVariant(ctx context.Context, variant *store.ImageVariant) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO image_variants (id, name, width, height, fit, quality, format, never_require_signed_urls)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		variant.ID, variant.Name, variant.Width, variant.Height, variant.Fit, variant.Quality,
		variant.Format, variant.NeverRequireSignedURLs)
	return err
}

// GetVariant retrieves a variant by ID.
func (s *ImagesStoreImpl) GetVariant(ctx context.Context, id string) (*store.ImageVariant, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, width, height, fit, quality, format, never_require_signed_urls
		FROM image_variants WHERE id = ?`, id)
	return s.scanVariant(row)
}

// ListVariants lists all image variants.
func (s *ImagesStoreImpl) ListVariants(ctx context.Context) ([]*store.ImageVariant, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, width, height, fit, quality, format, never_require_signed_urls
		FROM image_variants ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var variants []*store.ImageVariant
	for rows.Next() {
		variant, err := s.scanVariant(rows)
		if err != nil {
			return nil, err
		}
		variants = append(variants, variant)
	}
	return variants, rows.Err()
}

// UpdateVariant updates an image variant.
func (s *ImagesStoreImpl) UpdateVariant(ctx context.Context, variant *store.ImageVariant) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE image_variants SET name = ?, width = ?, height = ?, fit = ?, quality = ?, format = ?, never_require_signed_urls = ? WHERE id = ?`,
		variant.Name, variant.Width, variant.Height, variant.Fit, variant.Quality, variant.Format,
		variant.NeverRequireSignedURLs, variant.ID)
	return err
}

// DeleteVariant deletes an image variant.
func (s *ImagesStoreImpl) DeleteVariant(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM image_variants WHERE id = ?`, id)
	return err
}

func (s *ImagesStoreImpl) scanImage(row scanner) (*store.Image, error) {
	var image store.Image
	var meta, format sql.NullString
	if err := row.Scan(&image.ID, &image.Filename, &image.StorageKey, &image.Size,
		&image.Width, &image.Height, &format, &meta, &image.UploadedAt); err != nil {
		return nil, err
	}
	image.Format = format.String
	if meta.Valid && meta.String != "" {
		json.Unmarshal([]byte(meta.String), &image.Meta)
	}
	return &image, nil
}

func (s *ImagesStoreImpl) scanVariant(row scanner) (*store.ImageVariant, error) {
	var variant store.ImageVariant
	var width, height, quality sql.NullInt64
	var format sql.NullString
	if err := row.Scan(&variant.ID, &variant.Name, &width, &height, &variant.Fit,
		&quality, &format, &variant.NeverRequireSignedURLs); err != nil {
		return nil, err
	}
	variant.Width = int(width.Int64)
	variant.Height = int(height.Int64)
	variant.Quality = int(quality.Int64)
	variant.Format = format.String
	return &variant, nil
}
