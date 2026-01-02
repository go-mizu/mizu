package handler

import (
	"strings"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/qa/feature/questions"
	"github.com/go-mizu/mizu/blueprints/qa/feature/tags"
)

// Question handles question endpoints.
type Question struct {
	questions    questions.API
	tags         tags.API
	getAccountID func(*mizu.Ctx) string
}

// NewQuestion creates a new question handler.
func NewQuestion(questions questions.API, tags tags.API, getAccountID func(*mizu.Ctx) string) *Question {
	return &Question{questions: questions, tags: tags, getAccountID: getAccountID}
}

// Create creates a question.
func (h *Question) Create(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	var in questions.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}
	for i, tag := range in.Tags {
		in.Tags[i] = strings.ToLower(strings.TrimSpace(tag))
	}

	question, err := h.questions.Create(c.Request().Context(), accountID, in)
	if err != nil {
		return BadRequest(c, err.Error())
	}

	return Created(c, question)
}

// Get gets a question by ID.
func (h *Question) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	question, err := h.questions.GetByID(c.Request().Context(), id)
	if err != nil {
		return NotFound(c, "Question")
	}
	return Success(c, question)
}
