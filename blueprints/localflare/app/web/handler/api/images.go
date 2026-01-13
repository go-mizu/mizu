package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"
)

// Images handles Cloudflare Images requests.
type Images struct{}

// NewImages creates a new Images handler.
func NewImages() *Images {
	return &Images{}
}

// CloudflareImage represents an image.
type CloudflareImage struct {
	ID       string            `json:"id"`
	Filename string            `json:"filename"`
	Uploaded string            `json:"uploaded"`
	Variants []string          `json:"variants"`
	Meta     map[string]int    `json:"meta"`
}

// ImageVariant represents an image variant.
type ImageVariant struct {
	ID                     string                 `json:"id"`
	Name                   string                 `json:"name"`
	Options                map[string]any         `json:"options"`
	NeverRequireSignedURLs bool                   `json:"never_require_signed_urls"`
}

// List lists all images.
func (h *Images) List(c *mizu.Ctx) error {
	now := time.Now()
	images := []CloudflareImage{
		{
			ID:       "img-" + ulid.Make().String()[:8],
			Filename: "hero-banner.jpg",
			Uploaded: now.Add(-1 * time.Hour).Format(time.RFC3339),
			Variants: []string{"public", "thumbnail"},
			Meta:     map[string]int{"width": 1920, "height": 1080},
		},
		{
			ID:       "img-" + ulid.Make().String()[:8],
			Filename: "logo.png",
			Uploaded: now.Add(-24 * time.Hour).Format(time.RFC3339),
			Variants: []string{"public", "thumbnail"},
			Meta:     map[string]int{"width": 512, "height": 512},
		},
		{
			ID:       "img-" + ulid.Make().String()[:8],
			Filename: "product-1.webp",
			Uploaded: now.Add(-48 * time.Hour).Format(time.RFC3339),
			Variants: []string{"public", "thumbnail", "product"},
			Meta:     map[string]int{"width": 800, "height": 800},
		},
		{
			ID:       "img-" + ulid.Make().String()[:8],
			Filename: "team-photo.jpg",
			Uploaded: now.Add(-72 * time.Hour).Format(time.RFC3339),
			Variants: []string{"public", "thumbnail"},
			Meta:     map[string]int{"width": 2400, "height": 1600},
		},
		{
			ID:       "img-" + ulid.Make().String()[:8],
			Filename: "banner-mobile.png",
			Uploaded: now.Add(-96 * time.Hour).Format(time.RFC3339),
			Variants: []string{"public", "thumbnail"},
			Meta:     map[string]int{"width": 750, "height": 1334},
		},
		{
			ID:       "img-" + ulid.Make().String()[:8],
			Filename: "icon-set.svg",
			Uploaded: now.Add(-120 * time.Hour).Format(time.RFC3339),
			Variants: []string{"public"},
			Meta:     map[string]int{"width": 100, "height": 100},
		},
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"images": images,
		},
	})
}

// Upload handles image upload.
func (h *Images) Upload(c *mizu.Ctx) error {
	image := CloudflareImage{
		ID:       "img-" + ulid.Make().String()[:8],
		Filename: "uploaded-image.jpg",
		Uploaded: time.Now().Format(time.RFC3339),
		Variants: []string{"public", "thumbnail"},
		Meta:     map[string]int{"width": 1024, "height": 768},
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  image,
	})
}

// Delete deletes an image.
func (h *Images) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// ListVariants lists image variants.
func (h *Images) ListVariants(c *mizu.Ctx) error {
	variants := []ImageVariant{
		{
			ID:   "public",
			Name: "public",
			Options: map[string]any{
				"fit":    "scale-down",
				"width":  1920,
				"height": 1080,
			},
			NeverRequireSignedURLs: true,
		},
		{
			ID:   "thumbnail",
			Name: "thumbnail",
			Options: map[string]any{
				"fit":    "cover",
				"width":  150,
				"height": 150,
			},
			NeverRequireSignedURLs: true,
		},
		{
			ID:   "product",
			Name: "product",
			Options: map[string]any{
				"fit":    "contain",
				"width":  600,
				"height": 600,
			},
			NeverRequireSignedURLs: false,
		},
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"variants": variants,
		},
	})
}
