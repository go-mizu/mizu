package handler

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/qa/feature/answers"
	"github.com/go-mizu/mizu/blueprints/qa/feature/questions"
)

// Answer handles answer endpoints.
type Answer struct {
	answers     answers.API
	questions   questions.API
	getAccountID func(*mizu.Ctx) string
}

// NewAnswer creates a new answer handler.
func NewAnswer(answers answers.API, questions questions.API, getAccountID func(*mizu.Ctx) string) *Answer {
	return &Answer{answers: answers, questions: questions, getAccountID: getAccountID}
}

// Create creates an answer for a question.
func (h *Answer) Create(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	questionID := c.Param("id")

	var in answers.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}
	in.QuestionID = questionID

	answer, err := h.answers.Create(c.Request().Context(), accountID, in)
	if err != nil {
		return BadRequest(c, err.Error())
	}
	return Created(c, answer)
}
