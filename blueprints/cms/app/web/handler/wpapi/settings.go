package wpapi

import (
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/settings"
)

// GetSettings handles GET /wp/v2/settings
func (h *Handler) GetSettings(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	allSettings, err := h.settings.GetAll(c.Context())
	if err != nil {
		return ErrorInternal(c, "rest_cannot_read", "Could not read settings")
	}

	// Convert to WordPress settings format
	wp := h.settingsToWP(allSettings)

	return OK(c, wp)
}

// UpdateSettings handles POST/PUT/PATCH /wp/v2/settings
func (h *Handler) UpdateSettings(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	var req map[string]any
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return ErrorBadRequest(c, "rest_invalid_json", "Invalid JSON body")
	}

	// Map WordPress setting keys to internal keys
	settingMap := map[string]string{
		"title":                 "site_title",
		"description":           "site_description",
		"url":                   "site_url",
		"email":                 "admin_email",
		"timezone":              "timezone",
		"date_format":           "date_format",
		"time_format":           "time_format",
		"start_of_week":         "start_of_week",
		"language":              "language",
		"posts_per_page":        "posts_per_page",
		"default_comment_status": "default_comment_status",
		"default_ping_status":   "default_ping_status",
	}

	for wpKey, value := range req {
		internalKey, ok := settingMap[wpKey]
		if !ok {
			internalKey = wpKey
		}

		// Convert value to string
		var strValue string
		switch v := value.(type) {
		case string:
			strValue = v
		case float64:
			strValue = strconv.FormatFloat(v, 'f', -1, 64)
		case bool:
			strValue = strconv.FormatBool(v)
		default:
			continue
		}

		in := &settings.SetIn{
			Key:      internalKey,
			Value:    strValue,
			IsPublic: boolPtr(isPublicSetting(internalKey)),
		}

		if _, err := h.settings.Set(c.Context(), in); err != nil {
			// Continue on error for individual settings
			continue
		}
	}

	// Return updated settings
	allSettings, err := h.settings.GetAll(c.Context())
	if err != nil {
		return ErrorInternal(c, "rest_cannot_read", "Could not read settings")
	}

	return OK(c, h.settingsToWP(allSettings))
}

// settingsToWP converts internal settings to WordPress format.
func (h *Handler) settingsToWP(settingsList []*settings.Setting) WPSettings {
	// Build a map for easy lookup
	settingsMap := make(map[string]string)
	for _, s := range settingsList {
		settingsMap[s.Key] = s.Value
	}

	// Get values with defaults
	getValue := func(key, defaultVal string) string {
		if v, ok := settingsMap[key]; ok && v != "" {
			return v
		}
		return defaultVal
	}

	getIntValue := func(key string, defaultVal int) int {
		if v, ok := settingsMap[key]; ok && v != "" {
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
		return defaultVal
	}

	getInt64Value := func(key string, defaultVal int64) int64 {
		if v, ok := settingsMap[key]; ok && v != "" {
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				return i
			}
		}
		return defaultVal
	}

	getBoolValue := func(key string, defaultVal bool) bool {
		if v, ok := settingsMap[key]; ok && v != "" {
			return v == "true" || v == "1"
		}
		return defaultVal
	}

	return WPSettings{
		Title:                getValue("site_title", "My CMS"),
		Description:          getValue("site_description", "Just another CMS site"),
		URL:                  getValue("site_url", h.baseURL),
		Email:                getValue("admin_email", ""),
		Timezone:             getValue("timezone", "UTC"),
		DateFormat:           getValue("date_format", "F j, Y"),
		TimeFormat:           getValue("time_format", "g:i a"),
		StartOfWeek:          getIntValue("start_of_week", 0),
		Language:             getValue("language", "en_US"),
		UseSmilies:           getBoolValue("use_smilies", true),
		DefaultCategory:      getInt64Value("default_category", 1),
		DefaultPostFormat:    getValue("default_post_format", "standard"),
		PostsPerPage:         getIntValue("posts_per_page", 10),
		ShowOnFront:          getValue("show_on_front", "posts"),
		PageOnFront:          getInt64Value("page_on_front", 0),
		PageForPosts:         getInt64Value("page_for_posts", 0),
		DefaultPingStatus:    getValue("default_ping_status", "closed"),
		DefaultCommentStatus: getValue("default_comment_status", "open"),
		SiteLogo:             getInt64Value("site_logo", 0),
		SiteIcon:             getInt64Value("site_icon", 0),
	}
}

// isPublicSetting returns true if a setting should be publicly accessible.
func isPublicSetting(key string) bool {
	publicSettings := map[string]bool{
		"site_title":       true,
		"site_description": true,
		"site_url":         true,
		"timezone":         true,
		"date_format":      true,
		"time_format":      true,
		"language":         true,
		"posts_per_page":   true,
		"show_on_front":    true,
	}
	return publicSettings[key]
}

// boolPtr returns a pointer to a bool.
func boolPtr(b bool) *bool {
	return &b
}
