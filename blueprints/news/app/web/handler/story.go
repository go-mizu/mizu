package handler

import (
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/news/feature/stories"
)

// Story handles story endpoints.
type Story struct {
	stories   *stories.Service
	getUserID func(*mizu.Ctx) string
}

// NewStory creates a new story handler.
func NewStory(stories *stories.Service, getUserID func(*mizu.Ctx) string) *Story {
	return &Story{
		stories:   stories,
		getUserID: getUserID,
	}
}

// CreateInput is the input for creating a story.
type CreateInput struct {
	Title string   `json:"title"`
	URL   string   `json:"url,omitempty"`
	Text  string   `json:"text,omitempty"`
	Tags  []string `json:"tags,omitempty"`
}

// List lists stories.
func (h *Story) List(c *mizu.Ctx) error {
	sort := c.Query("sort")
	if sort == "" {
		sort = "hot"
	}
	tag := c.Query("tag")
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit == 0 {
		limit = 30
	}
	offset, _ := strconv.Atoi(c.Query("offset"))

	if limit > 100 {
		limit = 100
	}

	userID := h.getUserID(c)

	list, err := h.stories.List(c.Request().Context(), stories.ListIn{
		Sort:   sort,
		Tag:    tag,
		Limit:  limit,
		Offset: offset,
	}, userID)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, list)
}

// Get gets a story by ID.
func (h *Story) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	story, err := h.stories.GetByID(c.Request().Context(), id, userID)
	if err != nil {
		if err == stories.ErrNotFound {
			return NotFound(c, "story")
		}
		return InternalError(c, err)
	}

	return Success(c, story)
}

// Create creates a new story.
func (h *Story) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c)
	}

	var in CreateInput
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid input")
	}

	story, err := h.stories.Create(c.Request().Context(), userID, stories.CreateIn{
		Title: in.Title,
		URL:   in.URL,
		Text:  in.Text,
		Tags:  in.Tags,
	})
	if err != nil {
		switch err {
		case stories.ErrInvalidTitle:
			return BadRequest(c, "title is required (3-150 characters)")
		case stories.ErrInvalidURL:
			return BadRequest(c, "invalid URL")
		case stories.ErrDuplicateURL:
			return Conflict(c, "URL already submitted")
		default:
			return InternalError(c, err)
		}
	}

	return Created(c, story)
}
