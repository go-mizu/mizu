package handler

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/qa/feature/questions"
	"github.com/go-mizu/mizu/blueprints/qa/feature/tags"
)

// Tag handles tag endpoints.
type Tag struct {
	tags      tags.API
	questions questions.API
}

// NewTag creates a new tag handler.
func NewTag(tags tags.API, questions questions.API) *Tag {
	return &Tag{tags: tags, questions: questions}
}

// List lists tags.
func (h *Tag) List(c *mizu.Ctx) error {
	list, err := h.tags.List(c.Request().Context(), tags.ListOpts{Limit: 100})
	if err != nil {
		return InternalError(c)
	}
	return Success(c, list)
}
