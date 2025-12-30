package uploads

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/cms/config"
	"github.com/go-mizu/blueprints/cms/pkg/ulid"
	"github.com/go-mizu/blueprints/cms/store/duckdb"
)

// UploadService implements the Service interface.
type UploadService struct {
	store     *duckdb.UploadsStore
	storage   Storage
	processor ImageProcessor
	config    *config.UploadConfig
}

// NewService creates a new upload service.
func NewService(store *duckdb.UploadsStore, storage Storage, processor ImageProcessor, cfg *config.UploadConfig) *UploadService {
	return &UploadService{
		store:     store,
		storage:   storage,
		processor: processor,
		config:    cfg,
	}
}

// Upload uploads a new file.
func (s *UploadService) Upload(ctx context.Context, collection string, file *UploadFile, opts *UploadOptions) (*Upload, error) {
	if opts == nil {
		opts = &UploadOptions{}
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	id := ulid.New()
	filename := id + ext

	// Read file content into buffer for processing
	var buf bytes.Buffer
	size, err := io.Copy(&buf, file.Reader)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Validate file size
	if s.config != nil && s.config.MaxSize > 0 && size > s.config.MaxSize {
		return nil, fmt.Errorf("file too large: %d > %d", size, s.config.MaxSize)
	}

	// Validate mime type
	if s.config != nil && len(s.config.MimeTypes) > 0 {
		allowed := false
		for _, mt := range s.config.MimeTypes {
			if mt == file.ContentType || mt == "*/*" {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("mime type not allowed: %s", file.ContentType)
		}
	}

	upload := &Upload{
		ID:               id,
		Filename:         filename,
		OriginalFilename: file.Filename,
		MimeType:         file.ContentType,
		Filesize:         size,
		Alt:              opts.Alt,
		Caption:          opts.Caption,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Process images
	isImage := s.processor != nil && s.processor.CanProcess(file.ContentType)
	if isImage {
		// Get dimensions
		width, height, err := s.processor.GetDimensions(bytes.NewReader(buf.Bytes()))
		if err == nil {
			upload.Width = &width
			upload.Height = &height
		}

		// Set focal point
		if opts.FocalX != 0 || opts.FocalY != 0 {
			upload.FocalX = &opts.FocalX
			upload.FocalY = &opts.FocalY
		} else {
			// Default to center
			centerX, centerY := 0.5, 0.5
			upload.FocalX = &centerX
			upload.FocalY = &centerY
		}

		// Generate image sizes if configured
		if s.config != nil && len(s.config.ImageSizes) > 0 {
			focalX, focalY := 0.5, 0.5
			if upload.FocalX != nil {
				focalX = *upload.FocalX
			}
			if upload.FocalY != nil {
				focalY = *upload.FocalY
			}

			sizes, err := s.processor.GenerateSizes(bytes.NewReader(buf.Bytes()), s.config.ImageSizes, focalX, focalY)
			if err != nil {
				return nil, fmt.Errorf("generate sizes: %w", err)
			}

			upload.Sizes = make(map[string]SizeInfo)
			for name, processed := range sizes {
				// Store resized image
				sizePath, err := s.storage.Store(ctx, processed.Filename, processed.Reader)
				processed.Reader.Close()
				if err != nil {
					return nil, fmt.Errorf("store size %s: %w", name, err)
				}

				upload.Sizes[name] = SizeInfo{
					Width:    processed.Width,
					Height:   processed.Height,
					Filename: processed.Filename,
					MimeType: processed.MimeType,
					Filesize: processed.Filesize,
					URL:      s.storage.GetURL(sizePath),
				}
			}
		}
	}

	// Store original file
	if !opts.DisableLocal {
		path, err := s.storage.Store(ctx, filename, bytes.NewReader(buf.Bytes()))
		if err != nil {
			return nil, fmt.Errorf("store file: %w", err)
		}
		upload.URL = s.storage.GetURL(path)
	}

	// Save to database
	dbUpload := &duckdb.Upload{
		ID:               upload.ID,
		Filename:         upload.Filename,
		OriginalFilename: upload.OriginalFilename,
		MimeType:         upload.MimeType,
		Filesize:         upload.Filesize,
		Width:            upload.Width,
		Height:           upload.Height,
		FocalX:           upload.FocalX,
		FocalY:           upload.FocalY,
		Alt:              upload.Alt,
		Caption:          upload.Caption,
	}

	if upload.Sizes != nil {
		dbUpload.Sizes = make(map[string]duckdb.ImageSizeInfo)
		for k, v := range upload.Sizes {
			dbUpload.Sizes[k] = duckdb.ImageSizeInfo{
				Width:    v.Width,
				Height:   v.Height,
				Filename: v.Filename,
				MimeType: v.MimeType,
				Filesize: v.Filesize,
			}
		}
	}

	if err := s.store.Create(ctx, dbUpload); err != nil {
		return nil, fmt.Errorf("save upload: %w", err)
	}

	return upload, nil
}

// GetByID retrieves an upload by ID.
func (s *UploadService) GetByID(ctx context.Context, id string) (*Upload, error) {
	dbUpload, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dbUpload == nil {
		return nil, nil
	}

	return s.toUpload(dbUpload), nil
}

// Update updates upload metadata.
func (s *UploadService) Update(ctx context.Context, id string, input *UpdateInput) (*Upload, error) {
	dbUpload, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dbUpload == nil {
		return nil, fmt.Errorf("upload not found")
	}

	if input.Alt != nil {
		dbUpload.Alt = *input.Alt
	}
	if input.Caption != nil {
		dbUpload.Caption = *input.Caption
	}

	needsRegenerate := false
	if input.FocalX != nil {
		dbUpload.FocalX = input.FocalX
		needsRegenerate = true
	}
	if input.FocalY != nil {
		dbUpload.FocalY = input.FocalY
		needsRegenerate = true
	}

	if err := s.store.Update(ctx, dbUpload); err != nil {
		return nil, fmt.Errorf("update upload: %w", err)
	}

	// Regenerate sizes if focal point changed
	if needsRegenerate && dbUpload.Sizes != nil && len(dbUpload.Sizes) > 0 {
		upload := s.toUpload(dbUpload)
		return s.RegenerateImageSizes(ctx, upload.ID)
	}

	return s.toUpload(dbUpload), nil
}

// Delete removes an upload and its files.
func (s *UploadService) Delete(ctx context.Context, id string) error {
	dbUpload, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if dbUpload == nil {
		return nil
	}

	// Delete main file
	if err := s.storage.Delete(ctx, dbUpload.Filename); err != nil {
		// Log but don't fail
	}

	// Delete size variants
	for _, size := range dbUpload.Sizes {
		if err := s.storage.Delete(ctx, size.Filename); err != nil {
			// Log but don't fail
		}
	}

	return s.store.Delete(ctx, id)
}

// Find finds uploads matching the options.
func (s *UploadService) Find(ctx context.Context, opts *FindOptions) (*FindResult, error) {
	if opts == nil {
		opts = &FindOptions{}
	}

	dbOpts := &duckdb.FindOptions{
		Where: opts.Where,
		Limit: opts.Limit,
		Page:  opts.Page,
	}

	if opts.Sort != "" {
		desc := strings.HasPrefix(opts.Sort, "-")
		field := strings.TrimPrefix(opts.Sort, "-")
		dbOpts.Sort = []duckdb.SortField{{Field: field, Desc: desc}}
	}

	result, err := s.store.Find(ctx, dbOpts)
	if err != nil {
		return nil, err
	}

	uploads := make([]*Upload, 0, len(result.Docs))
	for _, doc := range result.Docs {
		upload := &Upload{
			ID:        doc.ID,
			CreatedAt: doc.CreatedAt,
			UpdatedAt: doc.UpdatedAt,
		}
		if filename, ok := doc.Data["filename"].(string); ok {
			upload.Filename = filename
			upload.URL = s.storage.GetURL(filename)
		}
		if v, ok := doc.Data["originalFilename"].(string); ok {
			upload.OriginalFilename = v
		}
		if v, ok := doc.Data["mimeType"].(string); ok {
			upload.MimeType = v
		}
		if v, ok := doc.Data["filesize"].(int64); ok {
			upload.Filesize = v
		} else if v, ok := doc.Data["filesize"].(float64); ok {
			upload.Filesize = int64(v)
		}
		if v, ok := doc.Data["width"].(int); ok {
			upload.Width = &v
		}
		if v, ok := doc.Data["height"].(int); ok {
			upload.Height = &v
		}
		if v, ok := doc.Data["focalX"].(float64); ok {
			upload.FocalX = &v
		}
		if v, ok := doc.Data["focalY"].(float64); ok {
			upload.FocalY = &v
		}
		if v, ok := doc.Data["alt"].(string); ok {
			upload.Alt = v
		}
		if v, ok := doc.Data["caption"].(string); ok {
			upload.Caption = v
		}
		uploads = append(uploads, upload)
	}

	return &FindResult{
		Docs:          uploads,
		TotalDocs:     result.TotalDocs,
		Limit:         result.Limit,
		TotalPages:    result.TotalPages,
		Page:          result.Page,
		PagingCounter: result.PagingCounter,
		HasPrevPage:   result.HasPrevPage,
		HasNextPage:   result.HasNextPage,
		PrevPage:      result.PrevPage,
		NextPage:      result.NextPage,
	}, nil
}

// SetFocalPoint updates the focal point and regenerates image sizes.
func (s *UploadService) SetFocalPoint(ctx context.Context, id string, x, y float64) (*Upload, error) {
	return s.Update(ctx, id, &UpdateInput{
		FocalX: &x,
		FocalY: &y,
	})
}

// Crop crops an image and regenerates sizes.
func (s *UploadService) Crop(ctx context.Context, id string, crop *CropOptions) (*Upload, error) {
	if s.processor == nil {
		return nil, fmt.Errorf("image processing not available")
	}

	dbUpload, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dbUpload == nil {
		return nil, fmt.Errorf("upload not found")
	}

	if !s.processor.CanProcess(dbUpload.MimeType) {
		return nil, fmt.Errorf("cannot process this file type")
	}

	// Open original file
	reader, err := s.storage.Open(ctx, dbUpload.Filename)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer reader.Close()

	// Crop image
	cropped, err := s.processor.Crop(reader, crop.X, crop.Y, crop.Width, crop.Height)
	if err != nil {
		return nil, fmt.Errorf("crop image: %w", err)
	}
	defer cropped.Close()

	// Read cropped content
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, cropped); err != nil {
		return nil, fmt.Errorf("read cropped: %w", err)
	}

	// Store cropped image (replace original)
	if _, err := s.storage.Store(ctx, dbUpload.Filename, bytes.NewReader(buf.Bytes())); err != nil {
		return nil, fmt.Errorf("store cropped: %w", err)
	}

	// Update dimensions
	width, height := crop.Width, crop.Height
	dbUpload.Width = &width
	dbUpload.Height = &height
	dbUpload.Filesize = int64(buf.Len())

	if err := s.store.Update(ctx, dbUpload); err != nil {
		return nil, fmt.Errorf("update upload: %w", err)
	}

	// Regenerate sizes with new cropped image
	return s.RegenerateImageSizes(ctx, id)
}

// RegenerateImageSizes regenerates all image size variants.
func (s *UploadService) RegenerateImageSizes(ctx context.Context, id string) (*Upload, error) {
	if s.processor == nil || s.config == nil || len(s.config.ImageSizes) == 0 {
		return s.GetByID(ctx, id)
	}

	dbUpload, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dbUpload == nil {
		return nil, fmt.Errorf("upload not found")
	}

	if !s.processor.CanProcess(dbUpload.MimeType) {
		return s.toUpload(dbUpload), nil
	}

	// Delete existing sizes
	for _, size := range dbUpload.Sizes {
		s.storage.Delete(ctx, size.Filename)
	}

	// Open original file
	reader, err := s.storage.Open(ctx, dbUpload.Filename)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer reader.Close()

	focalX, focalY := 0.5, 0.5
	if dbUpload.FocalX != nil {
		focalX = *dbUpload.FocalX
	}
	if dbUpload.FocalY != nil {
		focalY = *dbUpload.FocalY
	}

	sizes, err := s.processor.GenerateSizes(reader, s.config.ImageSizes, focalX, focalY)
	if err != nil {
		return nil, fmt.Errorf("generate sizes: %w", err)
	}

	dbUpload.Sizes = make(map[string]duckdb.ImageSizeInfo)
	for name, processed := range sizes {
		path, err := s.storage.Store(ctx, processed.Filename, processed.Reader)
		processed.Reader.Close()
		if err != nil {
			return nil, fmt.Errorf("store size %s: %w", name, err)
		}

		dbUpload.Sizes[name] = duckdb.ImageSizeInfo{
			Width:    processed.Width,
			Height:   processed.Height,
			Filename: processed.Filename,
			MimeType: processed.MimeType,
			Filesize: processed.Filesize,
		}
		_ = path
	}

	if err := s.store.Update(ctx, dbUpload); err != nil {
		return nil, fmt.Errorf("update upload: %w", err)
	}

	return s.toUpload(dbUpload), nil
}

func (s *UploadService) toUpload(db *duckdb.Upload) *Upload {
	upload := &Upload{
		ID:               db.ID,
		Filename:         db.Filename,
		OriginalFilename: db.OriginalFilename,
		MimeType:         db.MimeType,
		Filesize:         db.Filesize,
		Width:            db.Width,
		Height:           db.Height,
		FocalX:           db.FocalX,
		FocalY:           db.FocalY,
		Alt:              db.Alt,
		Caption:          db.Caption,
		URL:              s.storage.GetURL(db.Filename),
		CreatedAt:        db.CreatedAt,
		UpdatedAt:        db.UpdatedAt,
	}

	if db.Sizes != nil {
		upload.Sizes = make(map[string]SizeInfo)
		for k, v := range db.Sizes {
			upload.Sizes[k] = SizeInfo{
				Width:    v.Width,
				Height:   v.Height,
				Filename: v.Filename,
				MimeType: v.MimeType,
				Filesize: v.Filesize,
				URL:      s.storage.GetURL(v.Filename),
			}
		}
	}

	return upload
}
