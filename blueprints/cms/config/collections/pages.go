package collections

import (
	"github.com/go-mizu/blueprints/cms/config"
)

// Pages is the default pages collection.
var Pages = config.CollectionConfig{
	Slug: "pages",
	Labels: config.Labels{
		Singular: "Page",
		Plural:   "Pages",
	},
	Fields: []config.Field{
		{
			Type:      config.FieldTypeText,
			Name:      "title",
			Label:     "Title",
			Required:  true,
			Localized: true,
		},
		{
			Type:     config.FieldTypeText,
			Name:     "slug",
			Label:    "Slug",
			Required: true,
			Unique:   true,
			Index:    true,
			Admin: &config.FieldAdmin{
				Description: "URL-friendly identifier for the page",
			},
		},
		{
			Type:      config.FieldTypeRichText,
			Name:      "content",
			Label:     "Content",
			Localized: true,
		},
		{
			Type:       config.FieldTypeRelationship,
			Name:       "parent",
			Label:      "Parent Page",
			RelationTo: []string{"pages"},
			Admin: &config.FieldAdmin{
				Description: "Select a parent page for hierarchical organization",
			},
		},
		{
			Type:       config.FieldTypeUpload,
			Name:       "featuredImage",
			Label:      "Featured Image",
			RelationTo: []string{"media"},
		},
		{
			Type:  config.FieldTypeGroup,
			Name:  "meta",
			Label: "SEO",
			Admin: &config.FieldAdmin{
				Description: "Search engine optimization settings",
			},
			Fields: []config.Field{
				{
					Type:  config.FieldTypeText,
					Name:  "title",
					Label: "Meta Title",
					Admin: &config.FieldAdmin{
						Description: "Title for search engines (defaults to page title)",
					},
				},
				{
					Type:  config.FieldTypeTextarea,
					Name:  "description",
					Label: "Meta Description",
					Admin: &config.FieldAdmin{
						Description: "Description for search engines",
					},
				},
				{
					Type:       config.FieldTypeUpload,
					Name:       "image",
					Label:      "OG Image",
					RelationTo: []string{"media"},
					Admin: &config.FieldAdmin{
						Description: "Image for social media sharing",
					},
				},
			},
		},
		{
			Type:  config.FieldTypeSelect,
			Name:  "status",
			Label: "Status",
			Options: []config.SelectOption{
				{Label: "Draft", Value: "draft"},
				{Label: "Published", Value: "published"},
			},
			DefaultValue: "draft",
			Admin: &config.FieldAdmin{
				Position: "sidebar",
			},
		},
		{
			Type:  config.FieldTypeDate,
			Name:  "publishedAt",
			Label: "Publish Date",
			Admin: &config.FieldAdmin{
				Position:    "sidebar",
				Description: "Schedule publication for a future date",
			},
		},
	},
	Versions: &config.VersionsConfig{
		Drafts:    true,
		MaxPerDoc: 10,
	},
	Timestamps: true,
	Admin: &config.CollectionAdmin{
		UseAsTitle:     "title",
		DefaultColumns: []string{"title", "slug", "status", "updatedAt"},
		ListSearchableFields: []string{"title", "slug"},
	},
}

// Posts is a blog posts collection.
var Posts = config.CollectionConfig{
	Slug: "posts",
	Labels: config.Labels{
		Singular: "Post",
		Plural:   "Posts",
	},
	Fields: []config.Field{
		{
			Type:      config.FieldTypeText,
			Name:      "title",
			Label:     "Title",
			Required:  true,
			Localized: true,
		},
		{
			Type:     config.FieldTypeText,
			Name:     "slug",
			Label:    "Slug",
			Required: true,
			Unique:   true,
			Index:    true,
		},
		{
			Type:      config.FieldTypeRichText,
			Name:      "content",
			Label:     "Content",
			Localized: true,
		},
		{
			Type:      config.FieldTypeTextarea,
			Name:      "excerpt",
			Label:     "Excerpt",
			Localized: true,
			Admin: &config.FieldAdmin{
				Description: "Short summary of the post",
			},
		},
		{
			Type:       config.FieldTypeRelationship,
			Name:       "author",
			Label:      "Author",
			RelationTo: []string{"users"},
			Required:   true,
		},
		{
			Type:       config.FieldTypeRelationship,
			Name:       "categories",
			Label:      "Categories",
			RelationTo: []string{"categories"},
			HasMany:    true,
		},
		{
			Type:       config.FieldTypeRelationship,
			Name:       "tags",
			Label:      "Tags",
			RelationTo: []string{"tags"},
			HasMany:    true,
		},
		{
			Type:       config.FieldTypeUpload,
			Name:       "featuredImage",
			Label:      "Featured Image",
			RelationTo: []string{"media"},
		},
		{
			Type:  config.FieldTypeGroup,
			Name:  "meta",
			Label: "SEO",
			Fields: []config.Field{
				{
					Type:  config.FieldTypeText,
					Name:  "title",
					Label: "Meta Title",
				},
				{
					Type:  config.FieldTypeTextarea,
					Name:  "description",
					Label: "Meta Description",
				},
			},
		},
		{
			Type:  config.FieldTypeSelect,
			Name:  "status",
			Label: "Status",
			Options: []config.SelectOption{
				{Label: "Draft", Value: "draft"},
				{Label: "Published", Value: "published"},
			},
			DefaultValue: "draft",
			Admin: &config.FieldAdmin{
				Position: "sidebar",
			},
		},
		{
			Type:  config.FieldTypeDate,
			Name:  "publishedAt",
			Label: "Publish Date",
			Admin: &config.FieldAdmin{
				Position: "sidebar",
			},
		},
	},
	Versions: &config.VersionsConfig{
		Drafts:    true,
		MaxPerDoc: 10,
	},
	Timestamps: true,
	Admin: &config.CollectionAdmin{
		UseAsTitle:     "title",
		DefaultColumns: []string{"title", "author", "status", "publishedAt"},
		ListSearchableFields: []string{"title", "slug", "excerpt"},
	},
}

// Categories is a taxonomy collection for posts.
var Categories = config.CollectionConfig{
	Slug: "categories",
	Labels: config.Labels{
		Singular: "Category",
		Plural:   "Categories",
	},
	Fields: []config.Field{
		{
			Type:      config.FieldTypeText,
			Name:      "name",
			Label:     "Name",
			Required:  true,
			Localized: true,
		},
		{
			Type:     config.FieldTypeText,
			Name:     "slug",
			Label:    "Slug",
			Required: true,
			Unique:   true,
			Index:    true,
		},
		{
			Type:      config.FieldTypeTextarea,
			Name:      "description",
			Label:     "Description",
			Localized: true,
		},
		{
			Type:       config.FieldTypeRelationship,
			Name:       "parent",
			Label:      "Parent Category",
			RelationTo: []string{"categories"},
		},
	},
	Timestamps: true,
	Admin: &config.CollectionAdmin{
		UseAsTitle:     "name",
		DefaultColumns: []string{"name", "slug", "parent"},
	},
}

// Tags is a taxonomy collection for posts.
var Tags = config.CollectionConfig{
	Slug: "tags",
	Labels: config.Labels{
		Singular: "Tag",
		Plural:   "Tags",
	},
	Fields: []config.Field{
		{
			Type:      config.FieldTypeText,
			Name:      "name",
			Label:     "Name",
			Required:  true,
			Localized: true,
		},
		{
			Type:     config.FieldTypeText,
			Name:     "slug",
			Label:    "Slug",
			Required: true,
			Unique:   true,
			Index:    true,
		},
	},
	Timestamps: true,
	Admin: &config.CollectionAdmin{
		UseAsTitle: "name",
	},
}
