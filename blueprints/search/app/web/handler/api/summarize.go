package api

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/feature/summarize"
	"github.com/go-mizu/mizu/blueprints/search/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/types"
)

// SummarizeHandler handles summarization API requests.
type SummarizeHandler struct {
	service *summarize.Service
}

// NewSummarizeHandler creates a new summarize handler.
func NewSummarizeHandler(st store.Store, provider llm.Provider) *SummarizeHandler {
	return &SummarizeHandler{
		service: summarize.NewService(st.Summary(), provider),
	}
}

// Summarize handles POST /api/summarize requests.
func (h *SummarizeHandler) Summarize(c *mizu.Ctx) error {
	// Support both GET (url param) and POST (JSON body)
	var req types.SummarizeRequest

	if c.Request().Method == "GET" {
		req.URL = c.Query("url")
		req.Text = c.Query("text")
		if engine := c.Query("engine"); engine != "" {
			req.Engine = types.SummaryEngine(engine)
		}
		if summaryType := c.Query("summary_type"); summaryType != "" {
			req.SummaryType = types.SummaryType(summaryType)
		}
		req.TargetLanguage = c.Query("target_language")
		if cache := c.Query("cache"); cache == "false" {
			f := false
			req.Cache = &f
		}
	} else {
		if err := c.BindJSON(&req, 0); err != nil {
			return c.JSON(400, map[string]string{"error": "invalid request body"})
		}
	}

	if req.URL == "" && req.Text == "" {
		return c.JSON(400, map[string]string{"error": "either url or text must be provided"})
	}

	resp, err := h.service.Summarize(c.Context(), &req)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, resp)
}
