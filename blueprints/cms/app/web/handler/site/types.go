// Package site provides the public-facing frontend handlers.
package site

import (
	"time"

	"github.com/go-mizu/blueprints/cms/feature/categories"
	"github.com/go-mizu/blueprints/cms/feature/media"
	"github.com/go-mizu/blueprints/cms/feature/pages"
	"github.com/go-mizu/blueprints/cms/feature/tags"
	"github.com/go-mizu/blueprints/cms/feature/users"
)

// SiteContext holds site-level information.
type SiteContext struct {
	Name        string
	Tagline     string
	Description string
	URL         string
	Language    string
	Timezone    string
	Logo        *media.Media
	Icon        *media.Media
}

// ThemeContext holds theme configuration.
type ThemeContext struct {
	Name       string
	Slug       string
	Version    string
	Config     map[string]interface{}
	Colors     map[string]string
	DarkColors map[string]string
	Fonts      map[string]string
	Features   map[string]bool
}

// RequestContext holds request information.
type RequestContext struct {
	URL       string
	Path      string
	Query     map[string]string
	IsHome    bool
	IsSingle  bool
	IsPage    bool
	IsArchive bool
	IsSearch  bool
	Is404     bool
}

// Pagination holds pagination state.
type Pagination struct {
	CurrentPage int
	TotalPages  int
	Total       int
	PerPage     int
	BaseURL     string
	PrevURL     string
	NextURL     string
	Pages       []int
}

// MenuContext holds menu data.
type MenuContext struct {
	Items []*MenuItem
}

// MenuItem represents a navigation item.
type MenuItem struct {
	Title    string
	URL      string
	Target   string
	CSSClass string
	Active   bool
	Children []*MenuItem
}

// PostView represents a post for the frontend.
type PostView struct {
	ID            string
	Title         string
	Slug          string
	Excerpt       string
	Content       string
	FeaturedImage *media.Media
	Author        *users.User
	Categories    []*categories.Category
	Tags          []*tags.Tag
	PublishedAt   *time.Time
	UpdatedAt     time.Time
	ReadingTime   int
	WordCount     int
	AllowComments bool
	IsFeatured    bool
	IsSticky      bool
}

// PageView represents a page for the frontend.
type PageView struct {
	ID            string
	Title         string
	Slug          string
	Excerpt       string
	Content       string
	FeaturedImage *media.Media
	Author        *users.User
	Template      string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CommentView represents a comment for the frontend.
type CommentView struct {
	ID         string
	Content    string
	Author     *users.User
	AuthorName string
	ParentID   string
	CreatedAt  time.Time
	Replies    []*CommentView
}

// BaseData holds common template data.
type BaseData struct {
	Site       SiteContext
	Theme      ThemeContext
	Request    RequestContext
	Menus      map[string]*MenuContext
	User       *users.User
	Categories []*categories.Category
	Tags       []*tags.Tag
	RecentPosts []*PostView
}

// HomeData holds homepage data.
type HomeData struct {
	BaseData
	Posts      []*PostView
	Featured   []*PostView
	Pagination Pagination
}

// PostData holds single post data.
type PostData struct {
	BaseData
	Post     *PostView
	Author   *users.User
	PrevPost *PostView
	NextPost *PostView
	Related  []*PostView
	Comments []*CommentView
}

// PageData holds single page data.
type PageData struct {
	BaseData
	Page       *PageView
	ChildPages []*pages.Page
}

// ArchiveData holds archive page data.
type ArchiveData struct {
	BaseData
	Posts      []*PostView
	Pagination Pagination
	Category   *categories.Category
	Tag        *tags.Tag
	AuthorData *users.User
}

// SearchData holds search results data.
type SearchData struct {
	BaseData
	Query      string
	Posts      []*PostView
	Pages      []*PageView
	Total      int
	Pagination Pagination
}

// ErrorData holds error page data.
type ErrorData struct {
	BaseData
	ErrorCode    int
	ErrorMessage string
}
