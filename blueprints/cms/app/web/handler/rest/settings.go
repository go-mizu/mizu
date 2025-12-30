package rest

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/cms/feature/options"
)

// Settings handles REST API requests for settings.
type Settings struct {
	options   options.API
	getUserID func(*mizu.Ctx) string
}

// NewSettings creates a new settings handler.
func NewSettings(o options.API, getUserID func(*mizu.Ctx) string) *Settings {
	return &Settings{options: o, getUserID: getUserID}
}

// Get retrieves all settings.
func (h *Settings) Get(c *mizu.Ctx) error {
	settings, err := h.options.GetSettings(c.Request().Context())
	if err != nil {
		return InternalError(c, "Error retrieving settings")
	}

	return Success(c, h.formatSettings(settings))
}

// Update updates settings.
func (h *Settings) Update(c *mizu.Ctx) error {
	var in options.UpdateSettingsIn
	if err := c.BindJSON(&in, 0); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	settings, err := h.options.UpdateSettings(c.Request().Context(), in)
	if err != nil {
		return InternalError(c, "Error updating settings")
	}

	return Success(c, h.formatSettings(settings))
}

func (h *Settings) formatSettings(s *options.Settings) map[string]interface{} {
	return map[string]interface{}{
		"title":                  s.Title,
		"description":            s.Description,
		"url":                    s.URL,
		"email":                  s.Email,
		"timezone":               s.Timezone,
		"date_format":            s.DateFormat,
		"time_format":            s.TimeFormat,
		"start_of_week":          s.StartOfWeek,
		"language":               s.Language,
		"use_smilies":            s.UseSmilies,
		"default_category":       s.DefaultCategory,
		"default_post_format":    s.DefaultPostFormat,
		"posts_per_page":         s.PostsPerPage,
		"show_on_front":          s.ShowOnFront,
		"page_on_front":          s.PageOnFront,
		"page_for_posts":         s.PageForPosts,
		"default_ping_status":    s.DefaultPingStatus,
		"default_comment_status": s.DefaultCommentStatus,
		"site_icon":              s.SiteIcon,
		"site_logo":              s.SiteLogo,
	}
}
