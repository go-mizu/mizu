package handler

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
)

// Board handles board endpoints.
type Board struct {
	boards       boards.API
	getAccountID func(*mizu.Ctx) string
}

// NewBoard creates a new board handler.
func NewBoard(boards boards.API, getAccountID func(*mizu.Ctx) string) *Board {
	return &Board{boards: boards, getAccountID: getAccountID}
}

// List lists boards.
func (h *Board) List(c *mizu.Ctx) error {
	opts := boards.ListOpts{
		Limit: 25,
	}

	boardList, err := h.boards.List(c.Request().Context(), opts)
	if err != nil {
		return InternalError(c)
	}

	// Enrich with viewer state
	viewerID := h.getAccountID(c)
	if viewerID != "" {
		_ = h.boards.EnrichBoards(c.Request().Context(), boardList, viewerID)
	}

	return Success(c, boardList)
}

// Create creates a board.
func (h *Board) Create(c *mizu.Ctx) error {
	var in boards.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	board, err := h.boards.Create(c.Request().Context(), accountID, in)
	if err != nil {
		switch err {
		case boards.ErrNameTaken:
			return Conflict(c, "Board name already taken")
		case boards.ErrInvalidName:
			return BadRequest(c, "Invalid board name")
		default:
			return InternalError(c)
		}
	}

	return Created(c, board)
}

// Get gets a board by name.
func (h *Board) Get(c *mizu.Ctx) error {
	name := c.Param("name")

	board, err := h.boards.GetByName(c.Request().Context(), name)
	if err != nil {
		if err == boards.ErrNotFound {
			return NotFound(c, "Board")
		}
		return InternalError(c)
	}

	// Enrich with viewer state
	viewerID := h.getAccountID(c)
	if viewerID != "" {
		_ = h.boards.EnrichBoard(c.Request().Context(), board, viewerID)
	}

	return Success(c, board)
}

// Update updates a board.
func (h *Board) Update(c *mizu.Ctx) error {
	name := c.Param("name")

	board, err := h.boards.GetByName(c.Request().Context(), name)
	if err != nil {
		if err == boards.ErrNotFound {
			return NotFound(c, "Board")
		}
		return InternalError(c)
	}

	var in boards.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	updated, err := h.boards.Update(c.Request().Context(), board.ID, in)
	if err != nil {
		if err == boards.ErrBoardArchived {
			return BadRequest(c, "Board is archived")
		}
		return InternalError(c)
	}

	return Success(c, updated)
}

// Join joins a board.
func (h *Board) Join(c *mizu.Ctx) error {
	name := c.Param("name")
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	board, err := h.boards.GetByName(c.Request().Context(), name)
	if err != nil {
		if err == boards.ErrNotFound {
			return NotFound(c, "Board")
		}
		return InternalError(c)
	}

	if err := h.boards.Join(c.Request().Context(), board.ID, accountID); err != nil {
		switch err {
		case boards.ErrAlreadyMember:
			return Success(c, map[string]any{"message": "Already a member"})
		case boards.ErrBoardArchived:
			return BadRequest(c, "Board is archived")
		default:
			return InternalError(c)
		}
	}

	return Success(c, map[string]any{"message": "Joined"})
}

// Leave leaves a board.
func (h *Board) Leave(c *mizu.Ctx) error {
	name := c.Param("name")
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	board, err := h.boards.GetByName(c.Request().Context(), name)
	if err != nil {
		if err == boards.ErrNotFound {
			return NotFound(c, "Board")
		}
		return InternalError(c)
	}

	if err := h.boards.Leave(c.Request().Context(), board.ID, accountID); err != nil {
		return InternalError(c)
	}

	return Success(c, map[string]any{"message": "Left"})
}

// ListModerators lists board moderators.
func (h *Board) ListModerators(c *mizu.Ctx) error {
	name := c.Param("name")

	board, err := h.boards.GetByName(c.Request().Context(), name)
	if err != nil {
		if err == boards.ErrNotFound {
			return NotFound(c, "Board")
		}
		return InternalError(c)
	}

	mods, err := h.boards.ListModerators(c.Request().Context(), board.ID)
	if err != nil {
		return InternalError(c)
	}

	return Success(c, mods)
}
