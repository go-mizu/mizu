package collections

import (
	"github.com/go-mizu/blueprints/cms/config"
)

// Media is the default media collection for file uploads.
var Media = config.CollectionConfig{
	Slug: "media",
	Labels: config.Labels{
		Singular: "Media",
		Plural:   "Media",
	},
	Upload: &config.CollectionUploadConfig{
		StaticDir:  "./uploads",
		StaticURL:  "/uploads",
		MimeTypes:  []string{"image/*", "video/*", "audio/*", "application/pdf"},
		MaxSize:    10 * 1024 * 1024, // 10MB
		ImageSizes: []config.ImageSize{
			{Name: "thumbnail", Width: 150, Height: 150, Crop: true},
			{Name: "card", Width: 640, Height: 480, Crop: false},
			{Name: "feature", Width: 1200, Height: 0, Crop: false},
		},
		FocalPoint: true,
		BulkUpload: true,
	},
	Fields: []config.Field{
		{
			Type:  config.FieldTypeText,
			Name:  "alt",
			Label: "Alt Text",
			Admin: &config.FieldAdmin{
				Description: "Describe the image for accessibility and SEO",
			},
		},
		{
			Type:  config.FieldTypeTextarea,
			Name:  "caption",
			Label: "Caption",
		},
	},
	Timestamps: true,
	Admin: &config.CollectionAdmin{
		UseAsTitle:     "filename",
		DefaultColumns: []string{"filename", "mimeType", "filesize", "createdAt"},
	},
}
