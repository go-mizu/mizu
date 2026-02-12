package api

import (
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/book/store"
	"github.com/go-mizu/mizu/blueprints/book/types"
)

type NoteHandler struct{ st store.Store }

func NewNoteHandler(st store.Store) *NoteHandler { return &NoteHandler{st: st} }

func (h *NoteHandler) Get(c *mizu.Ctx) error {
	bookID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	note, err := h.st.Note().Get(c.Context(), bookID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if note == nil {
		return c.JSON(200, &types.BookNote{BookID: bookID})
	}
	return c.JSON(200, note)
}

func (h *NoteHandler) Upsert(c *mizu.Ctx) error {
	bookID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var note types.BookNote
	if err := c.BindJSON(&note, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid JSON"})
	}
	note.BookID = bookID
	if err := h.st.Note().Upsert(c.Context(), &note); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, note)
}

func (h *NoteHandler) Delete(c *mizu.Ctx) error {
	bookID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.st.Note().Delete(c.Context(), bookID); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}
