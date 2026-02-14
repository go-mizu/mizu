package api

import (
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/book/store"
)

type StatsHandler struct{ st store.Store }

func NewStatsHandler(st store.Store) *StatsHandler { return &StatsHandler{st: st} }

func (h *StatsHandler) Overall(c *mizu.Ctx) error {
	stats, err := h.st.Stats().GetOverallStats(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, stats)
}

func (h *StatsHandler) ByYear(c *mizu.Ctx) error {
	year, _ := strconv.Atoi(c.Param("year"))
	stats, err := h.st.Stats().GetStats(c.Context(), year)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, stats)
}
