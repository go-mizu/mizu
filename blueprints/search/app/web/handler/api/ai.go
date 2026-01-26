package api

import (
	"net/http"
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/feature/ai"
	"github.com/go-mizu/mizu/blueprints/search/feature/canvas"
	"github.com/go-mizu/mizu/blueprints/search/feature/session"
)

// AIHandler handles AI mode API endpoints.
type AIHandler struct {
	ai       *ai.Service
	session  *session.Service
	canvas   *canvas.Service
	registry *ai.ModelRegistry
}

// NewAIHandler creates a new AI handler.
func NewAIHandler(aiSvc *ai.Service, sessionSvc *session.Service, canvasSvc *canvas.Service, registry *ai.ModelRegistry) *AIHandler {
	return &AIHandler{
		ai:       aiSvc,
		session:  sessionSvc,
		canvas:   canvasSvc,
		registry: registry,
	}
}

// Register registers AI routes on a router group.
func (h *AIHandler) Register(r *mizu.Router) {
	// AI modes and query
	r.Get("/modes", h.GetModes)
	r.Get("/models", h.GetModels)
	r.Get("/models/{id}/health", h.GetModelHealth)
	r.Post("/query", h.Query)
	r.Post("/query/stream", h.QueryStream)
	r.Post("/deepsearch", h.DeepSearch)
	r.Post("/deepsearch/stream", h.DeepSearchStream)

	// Sessions
	r.Get("/sessions", h.ListSessions)
	r.Post("/sessions", h.CreateSession)
	r.Get("/sessions/{id}", h.GetSession)
	r.Delete("/sessions/{id}", h.DeleteSession)

	// Canvas
	r.Get("/canvas/{session_id}", h.GetCanvas)
	r.Put("/canvas/{session_id}", h.UpdateCanvas)
	r.Post("/canvas/{session_id}/blocks", h.AddBlock)
	r.Put("/canvas/{session_id}/blocks/{block_id}", h.UpdateBlock)
	r.Delete("/canvas/{session_id}/blocks/{block_id}", h.DeleteBlock)
	r.Post("/canvas/{session_id}/reorder", h.ReorderBlocks)
	r.Get("/canvas/{session_id}/export", h.ExportCanvas)
}

// GetModes returns available AI modes.
func (h *AIHandler) GetModes(c *mizu.Ctx) error {
	return c.JSON(http.StatusOK, map[string]any{
		"modes": ai.GetModes(),
	})
}

// GetModels returns available AI models.
func (h *AIHandler) GetModels(c *mizu.Ctx) error {
	if h.registry == nil {
		return c.JSON(http.StatusOK, []ai.ModelInfo{})
	}

	// Update availability before returning
	h.registry.UpdateAvailability(c.Context())

	return c.JSON(http.StatusOK, h.registry.ListModels())
}

// GetModelHealth checks if a specific model is available.
func (h *AIHandler) GetModelHealth(c *mizu.Ctx) error {
	id := c.Param("id")

	if h.registry == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "model not found"})
	}

	available := h.registry.CheckHealth(c.Context(), id)

	return c.JSON(http.StatusOK, map[string]any{
		"available": available,
	})
}

// QueryRequest represents an AI query request.
type QueryRequest struct {
	Text      string   `json:"text"`
	Mode      ai.Mode  `json:"mode"`
	ModelID   string   `json:"model_id,omitempty"`
	SessionID string   `json:"session_id,omitempty"`
	ImageURLs []string `json:"image_urls,omitempty"`
	AudioURL  string   `json:"audio_url,omitempty"`
}

// Query handles a non-streaming AI query.
func (h *AIHandler) Query(c *mizu.Ctx) error {
	var req QueryRequest
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Text == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "text is required"})
	}

	if req.Mode == "" {
		req.Mode = ai.ModeQuick
	}

	resp, err := h.ai.Process(c.Context(), ai.Query{
		Text:      req.Text,
		Mode:      req.Mode,
		SessionID: req.SessionID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, resp)
}

// QueryStream handles a streaming AI query via SSE.
func (h *AIHandler) QueryStream(c *mizu.Ctx) error {
	var req QueryRequest
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Text == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "text is required"})
	}

	if req.Mode == "" {
		req.Mode = ai.ModeQuick
	}

	stream, err := h.ai.ProcessStream(c.Context(), ai.Query{
		Text:      req.Text,
		Mode:      req.Mode,
		SessionID: req.SessionID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Convert ai.StreamEvent to any channel for SSE
	ch := make(chan any, 100)
	go func() {
		defer close(ch)
		for event := range stream {
			ch <- event
		}
	}()

	return c.SSE(ch)
}

// ListSessions returns a list of sessions.
func (h *AIHandler) ListSessions(c *mizu.Ctx) error {
	limit := queryInt(c, "limit", 20)
	offset := queryInt(c, "offset", 0)

	sessions, total, err := h.session.List(c.Context(), limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"sessions": sessions,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// CreateSessionRequest represents a create session request.
type CreateSessionRequest struct {
	Title string `json:"title"`
}

// CreateSession creates a new session.
func (h *AIHandler) CreateSession(c *mizu.Ctx) error {
	var req CreateSessionRequest
	_ = c.BindJSON(&req, 0)

	sess, err := h.session.Create(c.Context(), req.Title)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"session": sess,
	})
}

// GetSession returns a session with messages.
func (h *AIHandler) GetSession(c *mizu.Ctx) error {
	id := c.Param("id")

	sess, err := h.session.Get(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "session not found"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"session": sess,
	})
}

// DeleteSession deletes a session.
func (h *AIHandler) DeleteSession(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.session.Delete(c.Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
	})
}

// GetCanvas returns the canvas for a session.
func (h *AIHandler) GetCanvas(c *mizu.Ctx) error {
	sessionID := c.Param("session_id")

	canv, err := h.canvas.GetBySessionID(c.Context(), sessionID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "canvas not found"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"canvas": canv,
	})
}

// UpdateCanvasRequest represents an update canvas request.
type UpdateCanvasRequest struct {
	Title string `json:"title"`
}

// UpdateCanvas updates a canvas.
func (h *AIHandler) UpdateCanvas(c *mizu.Ctx) error {
	sessionID := c.Param("session_id")

	var req UpdateCanvasRequest
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	canv, err := h.canvas.GetBySessionID(c.Context(), sessionID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "canvas not found"})
	}

	canv.Title = req.Title
	if err := h.canvas.Update(c.Context(), canv); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"canvas": canv,
	})
}

// AddBlockRequest represents an add block request.
type AddBlockRequest struct {
	Type    canvas.BlockType `json:"type"`
	Content string           `json:"content"`
	Order   int              `json:"order"`
}

// AddBlock adds a block to a canvas.
func (h *AIHandler) AddBlock(c *mizu.Ctx) error {
	sessionID := c.Param("session_id")

	var req AddBlockRequest
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	canv, err := h.canvas.GetBySessionID(c.Context(), sessionID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "canvas not found"})
	}

	block, err := h.canvas.AddBlock(c.Context(), canv.ID, req.Type, req.Content, req.Order)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"block": block,
	})
}

// UpdateBlockRequest represents an update block request.
type UpdateBlockRequest struct {
	Type    canvas.BlockType `json:"type"`
	Content string           `json:"content"`
	Order   int              `json:"order"`
}

// UpdateBlock updates a block.
func (h *AIHandler) UpdateBlock(c *mizu.Ctx) error {
	blockID := c.Param("block_id")

	var req UpdateBlockRequest
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	block := &canvas.Block{
		ID:      blockID,
		Type:    req.Type,
		Content: req.Content,
		Order:   req.Order,
	}

	if err := h.canvas.UpdateBlock(c.Context(), block); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"block": block,
	})
}

// DeleteBlock deletes a block.
func (h *AIHandler) DeleteBlock(c *mizu.Ctx) error {
	blockID := c.Param("block_id")

	if err := h.canvas.DeleteBlock(c.Context(), blockID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
	})
}

// ReorderBlocksRequest represents a reorder blocks request.
type ReorderBlocksRequest struct {
	BlockIDs []string `json:"block_ids"`
}

// ReorderBlocks reorders blocks in a canvas.
func (h *AIHandler) ReorderBlocks(c *mizu.Ctx) error {
	sessionID := c.Param("session_id")

	var req ReorderBlocksRequest
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	canv, err := h.canvas.GetBySessionID(c.Context(), sessionID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "canvas not found"})
	}

	if err := h.canvas.ReorderBlocks(c.Context(), canv.ID, req.BlockIDs); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
	})
}

// ExportCanvas exports a canvas to the specified format.
func (h *AIHandler) ExportCanvas(c *mizu.Ctx) error {
	sessionID := c.Param("session_id")
	format := c.Query("format")
	if format == "" {
		format = "markdown"
	}

	canv, err := h.canvas.GetBySessionID(c.Context(), sessionID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "canvas not found"})
	}

	exportFormat := canvas.ExportFormat(format)
	data, contentType, err := h.canvas.Export(c.Context(), canv.ID, exportFormat)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	filename := canv.Title + "." + format
	c.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

	return c.Bytes(http.StatusOK, data, contentType)
}

// queryInt parses an integer query parameter with a default value.
func queryInt(c *mizu.Ctx, key string, defaultVal int) int {
	s := c.Query(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// DeepSearch handles a non-streaming deep search request.
func (h *AIHandler) DeepSearch(c *mizu.Ctx) error {
	var req QueryRequest
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Text == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "text is required"})
	}

	resp, err := h.ai.ProcessDeepSearch(c.Context(), ai.Query{
		Text:      req.Text,
		Mode:      ai.ModeDeepSearch,
		SessionID: req.SessionID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, resp)
}

// DeepSearchStream handles a streaming deep search request via SSE.
func (h *AIHandler) DeepSearchStream(c *mizu.Ctx) error {
	var req QueryRequest
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Text == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "text is required"})
	}

	stream, err := h.ai.ProcessDeepSearchStream(c.Context(), ai.Query{
		Text:      req.Text,
		Mode:      ai.ModeDeepSearch,
		SessionID: req.SessionID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Convert DeepSearchStreamEvent to unified StreamEvent format for frontend
	ch := make(chan any, 100)
	go func() {
		defer close(ch)

		// Send start event
		ch <- ai.StreamEvent{Type: "start"}

		citationIndex := 0
		for event := range stream {
			switch event.Type {
			case "progress":
				// Convert progress to thinking event
				if event.Progress != nil {
					ch <- ai.StreamEvent{
						Type:     "thinking",
						Thinking: event.Progress.Message,
					}
				}
			case "source":
				// Convert source to citation event
				if event.Source != nil {
					citationIndex++
					ch <- ai.StreamEvent{
						Type: "citation",
						Citation: &session.Citation{
							Index: citationIndex,
							URL:   event.Source.URL,
							Title: event.Source.Title,
						},
					}
				}
			case "citation":
				if event.Citation != nil {
					ch <- ai.StreamEvent{
						Type:     "citation",
						Citation: event.Citation,
					}
				}
			case "section":
				// Convert section to thinking + token events
				if event.Section != nil {
					ch <- ai.StreamEvent{
						Type:     "thinking",
						Thinking: "Writing section: " + event.Section.Title,
					}
					// Stream section content as tokens
					for _, c := range event.Section.Content {
						ch <- ai.StreamEvent{Type: "token", Content: string(c)}
					}
					ch <- ai.StreamEvent{Type: "token", Content: "\n\n"}
				}
			case "token":
				ch <- ai.StreamEvent{Type: "token", Content: event.Token}
			case "done":
				// Send done event with response
				ch <- ai.StreamEvent{
					Type: "done",
					Response: &ai.StreamResponse{
						SessionID: event.SessionID,
						FollowUps: event.FollowUps,
						Mode:      ai.ModeDeepSearch,
					},
				}
			case "error":
				ch <- ai.StreamEvent{Type: "error", Error: event.Error}
			}
		}
	}()

	return c.SSE(ch)
}
