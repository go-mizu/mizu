package options

import (
	"context"
	"strconv"

	"github.com/go-mizu/mizu/blueprints/cms/store/duckdb"
)

// Service implements the options API.
type Service struct {
	options *duckdb.OptionsStore
}

// NewService creates a new options service.
func NewService(options *duckdb.OptionsStore) *Service {
	return &Service{options: options}
}

// Get retrieves an option value.
func (s *Service) Get(ctx context.Context, name string) (string, error) {
	return s.options.Get(ctx, name)
}

// GetWithDefault retrieves an option value with a default.
func (s *Service) GetWithDefault(ctx context.Context, name, defaultValue string) (string, error) {
	return s.options.GetWithDefault(ctx, name, defaultValue)
}

// Set sets an option value.
func (s *Service) Set(ctx context.Context, name, value string) error {
	return s.options.Set(ctx, name, value, true)
}

// Delete deletes an option.
func (s *Service) Delete(ctx context.Context, name string) error {
	return s.options.Delete(ctx, name)
}

// GetMultiple retrieves multiple options.
func (s *Service) GetMultiple(ctx context.Context, names []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, name := range names {
		value, err := s.options.Get(ctx, name)
		if err != nil {
			continue
		}
		result[name] = value
	}
	return result, nil
}

// SetMultiple sets multiple options.
func (s *Service) SetMultiple(ctx context.Context, options map[string]string) error {
	for name, value := range options {
		if err := s.Set(ctx, name, value); err != nil {
			return err
		}
	}
	return nil
}

// GetSettings retrieves all settings in REST API format.
func (s *Service) GetSettings(ctx context.Context) (*Settings, error) {
	title, _ := s.GetWithDefault(ctx, OptionBlogName, "My Site")
	description, _ := s.GetWithDefault(ctx, OptionBlogDescription, "Just another Mizu site")
	url, _ := s.GetWithDefault(ctx, OptionSiteURL, "http://localhost:8080")
	email, _ := s.GetWithDefault(ctx, OptionAdminEmail, "admin@example.com")
	timezone, _ := s.GetWithDefault(ctx, OptionTimezone, "UTC")
	dateFormat, _ := s.GetWithDefault(ctx, OptionDateFormat, "F j, Y")
	timeFormat, _ := s.GetWithDefault(ctx, OptionTimeFormat, "g:i a")
	startOfWeekStr, _ := s.GetWithDefault(ctx, OptionStartOfWeek, "1")
	language, _ := s.GetWithDefault(ctx, OptionLanguage, "en_US")
	defaultCategory, _ := s.GetWithDefault(ctx, OptionDefaultCategory, "1")
	defaultPostFormat, _ := s.GetWithDefault(ctx, OptionDefaultPostFormat, "standard")
	postsPerPageStr, _ := s.GetWithDefault(ctx, OptionPostsPerPage, "10")
	showOnFront, _ := s.GetWithDefault(ctx, OptionShowOnFront, "posts")
	pageOnFront, _ := s.GetWithDefault(ctx, OptionPageOnFront, "0")
	pageForPosts, _ := s.GetWithDefault(ctx, OptionPageForPosts, "0")
	defaultPingStatus, _ := s.GetWithDefault(ctx, OptionDefaultPingStatus, "open")
	defaultCommentStatus, _ := s.GetWithDefault(ctx, OptionDefaultCommentStatus, "open")

	startOfWeek, _ := strconv.Atoi(startOfWeekStr)
	postsPerPage, _ := strconv.Atoi(postsPerPageStr)

	return &Settings{
		Title:                title,
		Description:          description,
		URL:                  url,
		Email:                email,
		Timezone:             timezone,
		DateFormat:           dateFormat,
		TimeFormat:           timeFormat,
		StartOfWeek:          startOfWeek,
		Language:             language,
		DefaultCategory:      defaultCategory,
		DefaultPostFormat:    defaultPostFormat,
		PostsPerPage:         postsPerPage,
		ShowOnFront:          showOnFront,
		PageOnFront:          pageOnFront,
		PageForPosts:         pageForPosts,
		DefaultPingStatus:    defaultPingStatus,
		DefaultCommentStatus: defaultCommentStatus,
	}, nil
}

// UpdateSettings updates settings.
func (s *Service) UpdateSettings(ctx context.Context, in UpdateSettingsIn) (*Settings, error) {
	if in.Title != nil {
		_ = s.Set(ctx, OptionBlogName, *in.Title)
	}
	if in.Description != nil {
		_ = s.Set(ctx, OptionBlogDescription, *in.Description)
	}
	if in.URL != nil {
		_ = s.Set(ctx, OptionSiteURL, *in.URL)
		_ = s.Set(ctx, OptionHome, *in.URL)
	}
	if in.Email != nil {
		_ = s.Set(ctx, OptionAdminEmail, *in.Email)
	}
	if in.Timezone != nil {
		_ = s.Set(ctx, OptionTimezone, *in.Timezone)
	}
	if in.DateFormat != nil {
		_ = s.Set(ctx, OptionDateFormat, *in.DateFormat)
	}
	if in.TimeFormat != nil {
		_ = s.Set(ctx, OptionTimeFormat, *in.TimeFormat)
	}
	if in.StartOfWeek != nil {
		_ = s.Set(ctx, OptionStartOfWeek, strconv.Itoa(*in.StartOfWeek))
	}
	if in.Language != nil {
		_ = s.Set(ctx, OptionLanguage, *in.Language)
	}
	if in.DefaultCategory != nil {
		_ = s.Set(ctx, OptionDefaultCategory, *in.DefaultCategory)
	}
	if in.DefaultPostFormat != nil {
		_ = s.Set(ctx, OptionDefaultPostFormat, *in.DefaultPostFormat)
	}
	if in.PostsPerPage != nil {
		_ = s.Set(ctx, OptionPostsPerPage, strconv.Itoa(*in.PostsPerPage))
	}
	if in.ShowOnFront != nil {
		_ = s.Set(ctx, OptionShowOnFront, *in.ShowOnFront)
	}
	if in.PageOnFront != nil {
		_ = s.Set(ctx, OptionPageOnFront, *in.PageOnFront)
	}
	if in.PageForPosts != nil {
		_ = s.Set(ctx, OptionPageForPosts, *in.PageForPosts)
	}
	if in.DefaultPingStatus != nil {
		_ = s.Set(ctx, OptionDefaultPingStatus, *in.DefaultPingStatus)
	}
	if in.DefaultCommentStatus != nil {
		_ = s.Set(ctx, OptionDefaultCommentStatus, *in.DefaultCommentStatus)
	}

	return s.GetSettings(ctx)
}

// GetAutoloaded retrieves all autoloaded options.
func (s *Service) GetAutoloaded(ctx context.Context) (map[string]string, error) {
	return s.options.GetAutoloaded(ctx)
}

// InitDefaults initializes default options.
func (s *Service) InitDefaults(ctx context.Context, siteURL, siteTitle, adminEmail string) error {
	defaults := map[string]string{
		OptionSiteURL:          siteURL,
		OptionHome:             siteURL,
		OptionBlogName:         siteTitle,
		OptionBlogDescription:  "Just another Mizu site",
		OptionAdminEmail:       adminEmail,
		OptionUsersCanRegister: "0",
		OptionDefaultRole:      "subscriber",
		OptionTimezone:         "UTC",
		OptionDateFormat:       "F j, Y",
		OptionTimeFormat:       "g:i a",
		OptionStartOfWeek:      "1",
		OptionLanguage:         "en_US",

		OptionPostsPerPage: "10",
		OptionShowOnFront:  "posts",
		OptionPageOnFront:  "0",
		OptionPageForPosts: "0",
		OptionBlogPublic:   "1",

		OptionDefaultCategory:   "1",
		OptionDefaultPostFormat: "standard",

		OptionDefaultCommentStatus: "open",
		OptionDefaultPingStatus:    "open",
		OptionCommentModeration:    "0",
		OptionCommentRegistration:  "0",
		OptionCloseCommentsForOld:  "0",
		OptionThreadComments:       "1",
		OptionThreadCommentsDepth:  "5",
		OptionCommentsPerPage:      "50",
		OptionDefaultCommentsPage:  "newest",
		OptionCommentOrder:         "asc",

		OptionPermalinks:   "/%year%/%monthnum%/%postname%/",
		OptionCategoryBase: "",
		OptionTagBase:      "",

		OptionThumbnailSizeW:      "150",
		OptionThumbnailSizeH:      "150",
		OptionMediumSizeW:         "300",
		OptionMediumSizeH:         "300",
		OptionLargeSizeW:          "1024",
		OptionLargeSizeH:          "1024",
		OptionUploadsUseYearMonth: "1",

		OptionActiveTheme: "flavor",
		OptionStylesheet:  "flavor",
	}

	for name, value := range defaults {
		// Only set if not already exists
		existing, _ := s.options.Get(ctx, name)
		if existing == "" {
			if err := s.options.Set(ctx, name, value, true); err != nil {
				return err
			}
		}
	}

	return nil
}
