// Package types contains shared data types for the search blueprint.
package types

import "time"

// WidgetType represents the type of search widget.
type WidgetType string

const (
	WidgetInlineImages     WidgetType = "inline_images"
	WidgetInlineVideos     WidgetType = "inline_videos"
	WidgetInlineNews       WidgetType = "inline_news"
	WidgetInlineDiscussions WidgetType = "inline_discussions"
	WidgetInterestingFinds WidgetType = "interesting_finds"
	WidgetListicles        WidgetType = "listicles"
	WidgetInlineMaps       WidgetType = "inline_maps"
	WidgetPublicRecords    WidgetType = "public_records"
	WidgetPodcasts         WidgetType = "podcasts"
	WidgetQuickPeek        WidgetType = "quick_peek"
	WidgetSummaryBox       WidgetType = "summary_box"
	WidgetCheatSheet       WidgetType = "cheat_sheet"
	WidgetBlastFromPast    WidgetType = "blast_from_past"
	WidgetCode             WidgetType = "code"
	WidgetRelatedSearches  WidgetType = "related_searches"
	WidgetWikipedia        WidgetType = "wikipedia"
)

// AllWidgetTypes returns all available widget types.
func AllWidgetTypes() []WidgetType {
	return []WidgetType{
		WidgetInlineImages,
		WidgetInlineVideos,
		WidgetInlineNews,
		WidgetInlineDiscussions,
		WidgetInterestingFinds,
		WidgetListicles,
		WidgetInlineMaps,
		WidgetPublicRecords,
		WidgetPodcasts,
		WidgetQuickPeek,
		WidgetSummaryBox,
		WidgetCheatSheet,
		WidgetBlastFromPast,
		WidgetCode,
		WidgetRelatedSearches,
		WidgetWikipedia,
	}
}

// Widget represents a search result widget.
type Widget struct {
	Type     WidgetType  `json:"type"`
	Title    string      `json:"title,omitempty"`
	Position int         `json:"position"` // Position in results (0 = top, -1 = sidebar)
	Content  any `json:"content"`
}

// WidgetSetting represents user widget preferences.
type WidgetSetting struct {
	ID         int64      `json:"id"`
	UserID     string     `json:"user_id"`
	WidgetType WidgetType `json:"widget_type"`
	Enabled    bool       `json:"enabled"`
	Position   int        `json:"position"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CheatSheet represents a programming cheat sheet.
type CheatSheet struct {
	Language string         `json:"language"`
	Title    string         `json:"title"`
	Sections []CheatSection `json:"sections"`
}

// CheatSection represents a section of a cheat sheet.
type CheatSection struct {
	Name  string      `json:"name"`
	Items []CheatItem `json:"items"`
}

// CheatItem represents a single cheat sheet entry.
type CheatItem struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// QuickPeek represents a site preview card.
type QuickPeek struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Screenshot  string `json:"screenshot,omitempty"`
	LastUpdated string `json:"last_updated,omitempty"`
	Domain      string `json:"domain"`
	Favicon     string `json:"favicon,omitempty"`
}

// InterestingFind represents a small web/blog result.
type InterestingFind struct {
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Snippet   string    `json:"snippet"`
	Source    string    `json:"source"` // "blog", "forum", "discussion"
	Domain    string    `json:"domain"`
	Published time.Time `json:"published,omitempty"`
}

// BlastFromPast represents a historical article.
type BlastFromPast struct {
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Snippet   string    `json:"snippet"`
	Domain    string    `json:"domain"`
	Published time.Time `json:"published"`
	YearsAgo  int       `json:"years_ago"`
}

// CodeSnippet represents a code snippet widget content.
type CodeSnippet struct {
	Language    string `json:"language"`
	Code        string `json:"code"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source,omitempty"`
	SourceURL   string `json:"source_url,omitempty"`
}

// Discussion represents a forum/discussion result.
type Discussion struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Snippet     string    `json:"snippet"`
	Source      string    `json:"source"` // "reddit", "hackernews", "forum"
	Author      string    `json:"author,omitempty"`
	Replies     int       `json:"replies,omitempty"`
	Published   time.Time `json:"published,omitempty"`
}

// Podcast represents a podcast episode.
type Podcast struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ShowName    string    `json:"show_name"`
	Duration    int       `json:"duration_seconds,omitempty"`
	Published   time.Time `json:"published,omitempty"`
	AudioURL    string    `json:"audio_url,omitempty"`
	ImageURL    string    `json:"image_url,omitempty"`
}
