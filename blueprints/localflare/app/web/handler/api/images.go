package api

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"
)

// Images handles Cloudflare Images requests.
type Images struct {
	store   store.Store
	dataDir string
}

// NewImages creates a new Images handler.
func NewImages(st store.Store, dataDir string) *Images {
	return &Images{store: st, dataDir: filepath.Join(dataDir, "images")}
}

// CloudflareImageResponse represents an image response.
type CloudflareImageResponse struct {
	ID       string         `json:"id"`
	Filename string         `json:"filename"`
	Uploaded string         `json:"uploaded"`
	Variants []string       `json:"variants"`
	Meta     map[string]int `json:"meta"`
}

// ImageVariantResponse represents an image variant response.
type ImageVariantResponse struct {
	ID                     string         `json:"id"`
	Name                   string         `json:"name"`
	Options                map[string]any `json:"options"`
	NeverRequireSignedURLs bool           `json:"never_require_signed_urls"`
}

// List lists all images.
func (h *Images) List(c *mizu.Ctx) error {
	images, err := h.store.Images().ListImages(c.Request().Context(), 100, 0)
	if err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	var result []CloudflareImageResponse
	for _, img := range images {
		result = append(result, h.imageToResponse(img))
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"images": result,
		},
	})
}

// Upload handles image upload.
func (h *Images) Upload(c *mizu.Ctx) error {
	// Ensure data directory exists
	if err := os.MkdirAll(h.dataDir, 0755); err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	// Parse multipart form
	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return c.JSON(400, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": "No file provided"}},
		})
	}
	defer file.Close()

	uid := ulid.Make().String()[:12]
	ext := filepath.Ext(header.Filename)
	storageKey := uid + ext
	filePath := filepath.Join(h.dataDir, storageKey)

	// Save file to disk
	dst, err := os.Create(filePath)
	if err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}
	defer dst.Close()

	size, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(filePath)
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	// Determine format from extension
	format := ""
	switch ext {
	case ".jpg", ".jpeg":
		format = "jpeg"
	case ".png":
		format = "png"
	case ".webp":
		format = "webp"
	case ".gif":
		format = "gif"
	case ".svg":
		format = "svg"
	}

	image := &store.Image{
		ID:         "img-" + uid,
		Filename:   header.Filename,
		StorageKey: storageKey,
		Size:       size,
		Format:     format,
		UploadedAt: time.Now(),
	}

	if err := h.store.Images().CreateImage(c.Request().Context(), image); err != nil {
		os.Remove(filePath)
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  h.imageToResponse(image),
	})
}

// Delete deletes an image.
func (h *Images) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.store.Images().DeleteImage(c.Request().Context(), id); err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(404, map[string]any{
				"success": false,
				"errors":  []map[string]any{{"code": 1001, "message": "Image not found"}},
			})
		}
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// ListVariants lists image variants.
func (h *Images) ListVariants(c *mizu.Ctx) error {
	variants, err := h.store.Images().ListVariants(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	var result []ImageVariantResponse
	for _, v := range variants {
		result = append(result, h.variantToResponse(v))
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"variants": result,
		},
	})
}

// CreateVariant creates a new image variant.
func (h *Images) CreateVariant(c *mizu.Ctx) error {
	var input struct {
		Name                   string `json:"name"`
		Width                  int    `json:"width"`
		Height                 int    `json:"height"`
		Fit                    string `json:"fit"`
		Quality                int    `json:"quality"`
		NeverRequireSignedURLs bool   `json:"never_require_signed_urls"`
	}
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": "Invalid input"}},
		})
	}

	if input.Name == "" {
		return c.JSON(400, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": "Name is required"}},
		})
	}

	if input.Fit == "" {
		input.Fit = "scale-down"
	}
	if input.Quality == 0 {
		input.Quality = 85
	}

	variant := &store.ImageVariant{
		ID:                     input.Name,
		Name:                   input.Name,
		Width:                  input.Width,
		Height:                 input.Height,
		Fit:                    input.Fit,
		Quality:                input.Quality,
		NeverRequireSignedURLs: input.NeverRequireSignedURLs,
	}

	if err := h.store.Images().CreateVariant(c.Request().Context(), variant); err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  h.variantToResponse(variant),
	})
}

// DeleteVariant deletes an image variant.
func (h *Images) DeleteVariant(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.store.Images().DeleteVariant(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

func (h *Images) imageToResponse(img *store.Image) CloudflareImageResponse {
	// Get available variants
	variants := []string{"public", "thumbnail"}

	meta := map[string]int{
		"width":  img.Width,
		"height": img.Height,
	}

	return CloudflareImageResponse{
		ID:       img.ID,
		Filename: img.Filename,
		Uploaded: img.UploadedAt.Format(time.RFC3339),
		Variants: variants,
		Meta:     meta,
	}
}

func (h *Images) variantToResponse(v *store.ImageVariant) ImageVariantResponse {
	options := map[string]any{
		"fit": v.Fit,
	}
	if v.Width > 0 {
		options["width"] = v.Width
	}
	if v.Height > 0 {
		options["height"] = v.Height
	}
	if v.Quality > 0 {
		options["quality"] = v.Quality
	}

	return ImageVariantResponse{
		ID:                     v.ID,
		Name:                   v.Name,
		Options:                options,
		NeverRequireSignedURLs: v.NeverRequireSignedURLs,
	}
}
