package globals

import (
	"github.com/go-mizu/blueprints/cms/config"
)

// SiteSettings is the global site settings.
var SiteSettings = config.GlobalConfig{
	Slug:  "site-settings",
	Label: "Site Settings",
	Fields: []config.Field{
		{
			Type:     config.FieldTypeText,
			Name:     "siteName",
			Label:    "Site Name",
			Required: true,
		},
		{
			Type:  config.FieldTypeTextarea,
			Name:  "siteDescription",
			Label: "Site Description",
		},
		{
			Type:       config.FieldTypeUpload,
			Name:       "logo",
			Label:      "Logo",
			RelationTo: []string{"media"},
		},
		{
			Type:       config.FieldTypeUpload,
			Name:       "favicon",
			Label:      "Favicon",
			RelationTo: []string{"media"},
		},
		{
			Type:  config.FieldTypeGroup,
			Name:  "social",
			Label: "Social Links",
			Fields: []config.Field{
				{
					Type:  config.FieldTypeText,
					Name:  "twitter",
					Label: "Twitter",
					Admin: &config.FieldAdmin{
						Placeholder: "https://twitter.com/username",
					},
				},
				{
					Type:  config.FieldTypeText,
					Name:  "facebook",
					Label: "Facebook",
					Admin: &config.FieldAdmin{
						Placeholder: "https://facebook.com/page",
					},
				},
				{
					Type:  config.FieldTypeText,
					Name:  "instagram",
					Label: "Instagram",
					Admin: &config.FieldAdmin{
						Placeholder: "https://instagram.com/username",
					},
				},
				{
					Type:  config.FieldTypeText,
					Name:  "linkedin",
					Label: "LinkedIn",
					Admin: &config.FieldAdmin{
						Placeholder: "https://linkedin.com/company/name",
					},
				},
				{
					Type:  config.FieldTypeText,
					Name:  "github",
					Label: "GitHub",
					Admin: &config.FieldAdmin{
						Placeholder: "https://github.com/username",
					},
				},
				{
					Type:  config.FieldTypeText,
					Name:  "youtube",
					Label: "YouTube",
					Admin: &config.FieldAdmin{
						Placeholder: "https://youtube.com/channel/id",
					},
				},
			},
		},
		{
			Type:  config.FieldTypeGroup,
			Name:  "contact",
			Label: "Contact Information",
			Fields: []config.Field{
				{
					Type:  config.FieldTypeEmail,
					Name:  "email",
					Label: "Email",
				},
				{
					Type:  config.FieldTypeText,
					Name:  "phone",
					Label: "Phone",
				},
				{
					Type:  config.FieldTypeTextarea,
					Name:  "address",
					Label: "Address",
				},
			},
		},
		{
			Type:  config.FieldTypeGroup,
			Name:  "seo",
			Label: "Default SEO",
			Fields: []config.Field{
				{
					Type:  config.FieldTypeText,
					Name:  "titleSuffix",
					Label: "Title Suffix",
					Admin: &config.FieldAdmin{
						Description: "Appended to page titles (e.g., ' | My Site')",
					},
				},
				{
					Type:  config.FieldTypeTextarea,
					Name:  "defaultDescription",
					Label: "Default Meta Description",
				},
				{
					Type:       config.FieldTypeUpload,
					Name:       "defaultOGImage",
					Label:      "Default OG Image",
					RelationTo: []string{"media"},
				},
			},
		},
		{
			Type:  config.FieldTypeCode,
			Name:  "headerScripts",
			Label: "Header Scripts",
			Admin: &config.FieldAdmin{
				Description: "Scripts to include in the <head> section",
			},
		},
		{
			Type:  config.FieldTypeCode,
			Name:  "footerScripts",
			Label: "Footer Scripts",
			Admin: &config.FieldAdmin{
				Description: "Scripts to include before </body>",
			},
		},
	},
	Admin: &config.GlobalAdmin{
		Group:       "Settings",
		Description: "Configure site-wide settings",
	},
}

// Navigation is the global navigation configuration.
var Navigation = config.GlobalConfig{
	Slug:  "navigation",
	Label: "Navigation",
	Fields: []config.Field{
		{
			Type:  config.FieldTypeArray,
			Name:  "header",
			Label: "Header Navigation",
			Fields: []config.Field{
				{
					Type:     config.FieldTypeText,
					Name:     "label",
					Label:    "Label",
					Required: true,
				},
				{
					Type:  config.FieldTypeSelect,
					Name:  "type",
					Label: "Link Type",
					Options: []config.SelectOption{
						{Label: "Internal Page", Value: "page"},
						{Label: "Custom URL", Value: "custom"},
					},
					DefaultValue: "page",
				},
				{
					Type:       config.FieldTypeRelationship,
					Name:       "page",
					Label:      "Page",
					RelationTo: []string{"pages"},
				},
				{
					Type:  config.FieldTypeText,
					Name:  "url",
					Label: "Custom URL",
				},
				{
					Type:  config.FieldTypeCheckbox,
					Name:  "newTab",
					Label: "Open in New Tab",
				},
				{
					Type:  config.FieldTypeArray,
					Name:  "children",
					Label: "Submenu",
					Fields: []config.Field{
						{
							Type:     config.FieldTypeText,
							Name:     "label",
							Label:    "Label",
							Required: true,
						},
						{
							Type:  config.FieldTypeSelect,
							Name:  "type",
							Label: "Link Type",
							Options: []config.SelectOption{
								{Label: "Internal Page", Value: "page"},
								{Label: "Custom URL", Value: "custom"},
							},
							DefaultValue: "page",
						},
						{
							Type:       config.FieldTypeRelationship,
							Name:       "page",
							Label:      "Page",
							RelationTo: []string{"pages"},
						},
						{
							Type:  config.FieldTypeText,
							Name:  "url",
							Label: "Custom URL",
						},
						{
							Type:  config.FieldTypeCheckbox,
							Name:  "newTab",
							Label: "Open in New Tab",
						},
					},
				},
			},
		},
		{
			Type:  config.FieldTypeArray,
			Name:  "footer",
			Label: "Footer Navigation",
			Fields: []config.Field{
				{
					Type:     config.FieldTypeText,
					Name:     "label",
					Label:    "Label",
					Required: true,
				},
				{
					Type:  config.FieldTypeSelect,
					Name:  "type",
					Label: "Link Type",
					Options: []config.SelectOption{
						{Label: "Internal Page", Value: "page"},
						{Label: "Custom URL", Value: "custom"},
					},
					DefaultValue: "page",
				},
				{
					Type:       config.FieldTypeRelationship,
					Name:       "page",
					Label:      "Page",
					RelationTo: []string{"pages"},
				},
				{
					Type:  config.FieldTypeText,
					Name:  "url",
					Label: "Custom URL",
				},
			},
		},
	},
	Admin: &config.GlobalAdmin{
		Group:       "Settings",
		Description: "Configure site navigation menus",
	},
}
