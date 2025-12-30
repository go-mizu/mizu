// Package uploads provides file upload handling with image processing.
package uploads

import (
	"context"
	"io"
	"time"

	"github.com/go-mizu/blueprints/cms/config"
)

// Upload represents an uploaded file with metadata.
type Upload struct {
	ID               string                 `json:"id"`
	Filename         string                 `json:"filename"`
	OriginalFilename string                 `json:"originalFilename"`
	MimeType         string                 `json:"mimeType"`
	Filesize         int64                  `json:"filesize"`
	Width            *int                   `json:"width,omitempty"`
	Height           *int                   `json:"height,omitempty"`
	FocalX           *float64               `json:"focalX,omitempty"`
	FocalY           *float64               `json:"focalY,omitempty"`
	Alt              string                 `json:"alt,omitempty"`
	Caption          string                 `json:"caption,omitempty"`
	Sizes            map[string]SizeInfo    `json:"sizes,omitempty"`
	URL              string                 `json:"url"`
	CreatedAt        time.Time              `json:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt"`
}

// SizeInfo holds information about a resized image.
type SizeInfo struct {
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Filename string `json:"filename"`
	MimeType string `json:"mimeType"`
	Filesize int64  `json:"filesize"`
	URL      string `json:"url"`
}

// UploadFile represents a file being uploaded.
type UploadFile struct {
	Filename    string
	ContentType string
	Size        int64
	Reader      io.Reader
}

// UploadOptions holds options for upload operations.
type UploadOptions struct {
	Alt          string
	Caption      string
	FocalX       float64
	FocalY       float64
	DisableLocal bool // Don't write to local storage
}

// UpdateInput holds data for updating upload metadata.
type UpdateInput struct {
	Alt     *string  `json:"alt,omitempty"`
	Caption *string  `json:"caption,omitempty"`
	FocalX  *float64 `json:"focalX,omitempty"`
	FocalY  *float64 `json:"focalY,omitempty"`
}

// CropOptions holds options for cropping an image.
type CropOptions struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// FindOptions holds options for finding uploads.
type FindOptions struct {
	Where map[string]any
	Sort  string
	Limit int
	Page  int
}

// FindResult holds the result of a find operation.
type FindResult struct {
	Docs          []*Upload `json:"docs"`
	TotalDocs     int       `json:"totalDocs"`
	Limit         int       `json:"limit"`
	TotalPages    int       `json:"totalPages"`
	Page          int       `json:"page"`
	PagingCounter int       `json:"pagingCounter"`
	HasPrevPage   bool      `json:"hasPrevPage"`
	HasNextPage   bool      `json:"hasNextPage"`
	PrevPage      *int      `json:"prevPage"`
	NextPage      *int      `json:"nextPage"`
}

// Service defines the upload service interface.
type Service interface {
	// Upload operations
	Upload(ctx context.Context, collection string, file *UploadFile, opts *UploadOptions) (*Upload, error)
	GetByID(ctx context.Context, id string) (*Upload, error)
	Update(ctx context.Context, id string, input *UpdateInput) (*Upload, error)
	Delete(ctx context.Context, id string) error
	Find(ctx context.Context, opts *FindOptions) (*FindResult, error)

	// Image operations
	SetFocalPoint(ctx context.Context, id string, x, y float64) (*Upload, error)
	Crop(ctx context.Context, id string, crop *CropOptions) (*Upload, error)
	RegenerateImageSizes(ctx context.Context, id string) (*Upload, error)
}

// ImageProcessor defines the image processing interface.
type ImageProcessor interface {
	// GetDimensions returns the dimensions of an image.
	GetDimensions(reader io.Reader) (width, height int, err error)

	// Resize resizes an image maintaining aspect ratio.
	Resize(src io.Reader, maxWidth, maxHeight int) (io.ReadCloser, int, int, error)

	// ResizeWithFocalPoint resizes with focal point consideration.
	ResizeWithFocalPoint(src io.Reader, maxWidth, maxHeight int, focalX, focalY float64) (io.ReadCloser, int, int, error)

	// Crop crops an image to specified dimensions.
	Crop(src io.Reader, x, y, width, height int) (io.ReadCloser, error)

	// GenerateSizes generates all configured image sizes.
	GenerateSizes(src io.Reader, sizes []config.ImageSize, focalX, focalY float64) (map[string]*ProcessedImage, error)

	// CanProcess returns true if the mime type can be processed.
	CanProcess(mimeType string) bool
}

// ProcessedImage represents a processed image variant.
type ProcessedImage struct {
	Reader   io.ReadCloser
	Width    int
	Height   int
	Filename string
	MimeType string
	Filesize int64
}

// Storage defines the file storage interface.
type Storage interface {
	// Store saves a file and returns its path.
	Store(ctx context.Context, filename string, reader io.Reader) (string, error)

	// Delete removes a file.
	Delete(ctx context.Context, path string) error

	// GetURL returns the URL for a file.
	GetURL(path string) string

	// Open opens a file for reading.
	Open(ctx context.Context, path string) (io.ReadCloser, error)
}
