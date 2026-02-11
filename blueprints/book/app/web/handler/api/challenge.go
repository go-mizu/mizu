package api

import (
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/book/store"
	"github.com/go-mizu/mizu/blueprints/book/types"
)

type ChallengeHandler struct{ st store.Store }

func NewChallengeHandler(st store.Store) *ChallengeHandler { return &ChallengeHandler{st: st} }

func (h *ChallengeHandler) Get(c *mizu.Ctx) error {
	year, _ := strconv.Atoi(c.Param("year"))
	ch, err := h.st.Challenge().Get(c.Context(), year)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if ch == nil {
		return c.JSON(404, map[string]string{"error": "no challenge set"})
	}
	progress, _ := h.st.Challenge().GetProgress(c.Context(), year)
	ch.Progress = progress
	return c.JSON(200, ch)
}

func (h *ChallengeHandler) Set(c *mizu.Ctx) error {
	var ch types.ReadingChallenge
	if err := c.BindJSON(&ch, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON"})
	}
	if err := h.st.Challenge().Set(c.Context(), &ch); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, ch)
}
