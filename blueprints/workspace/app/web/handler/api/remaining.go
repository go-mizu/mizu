package api

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/comments"
	"github.com/go-mizu/blueprints/workspace/feature/databases"
	"github.com/go-mizu/blueprints/workspace/feature/favorites"
	"github.com/go-mizu/blueprints/workspace/feature/search"
	"github.com/go-mizu/blueprints/workspace/feature/sharing"
	"github.com/go-mizu/blueprints/workspace/feature/views"
)

// Block handles block endpoints.
type Block struct {
	blocks    blocks.API
	getUserID func(c *mizu.Ctx) string
}

func NewBlock(blocks blocks.API, getUserID func(c *mizu.Ctx) string) *Block {
	return &Block{blocks: blocks, getUserID: getUserID}
}

func (h *Block) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	var in blocks.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	in.CreatedBy = userID
	block, err := h.blocks.Create(c.Request().Context(), &in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, block)
}

func (h *Block) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)
	var in blocks.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	in.UpdatedBy = userID
	block, err := h.blocks.Update(c.Request().Context(), id, &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, block)
}

func (h *Block) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.blocks.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Block) Move(c *mizu.Ctx) error {
	id := c.Param("id")
	var in struct {
		ParentID string `json:"parent_id"`
		Position int    `json:"position"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if err := h.blocks.Move(c.Request().Context(), id, in.ParentID, in.Position); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "moved"})
}

// Database handles database endpoints.
type Database struct {
	databases databases.API
	getUserID func(c *mizu.Ctx) string
}

func NewDatabase(databases databases.API, getUserID func(c *mizu.Ctx) string) *Database {
	return &Database{databases: databases, getUserID: getUserID}
}

func (h *Database) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	var in databases.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	in.CreatedBy = userID
	db, err := h.databases.Create(c.Request().Context(), &in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, db)
}

func (h *Database) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	db, err := h.databases.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "database not found"})
	}
	return c.JSON(http.StatusOK, db)
}

func (h *Database) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	var in databases.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	db, err := h.databases.Update(c.Request().Context(), id, &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, db)
}

func (h *Database) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.databases.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Database) AddProperty(c *mizu.Ctx) error {
	dbID := c.Param("id")
	var prop databases.Property
	if err := c.BindJSON(&prop, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	db, err := h.databases.AddProperty(c.Request().Context(), dbID, prop)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, db)
}

func (h *Database) UpdateProperty(c *mizu.Ctx) error {
	dbID := c.Param("id")
	propID := c.Param("propID")
	var prop databases.Property
	if err := c.BindJSON(&prop, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if err := h.databases.UpdateProperty(c.Request().Context(), dbID, propID, prop); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Database) DeleteProperty(c *mizu.Ctx) error {
	dbID := c.Param("id")
	propID := c.Param("propID")
	if err := h.databases.DeleteProperty(c.Request().Context(), dbID, propID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// View handles view endpoints.
type View struct {
	views     views.API
	getUserID func(c *mizu.Ctx) string
}

func NewView(views views.API, getUserID func(c *mizu.Ctx) string) *View {
	return &View{views: views, getUserID: getUserID}
}

func (h *View) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	var in views.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// Allow database_id from path parameter (for POST /databases/{id}/views)
	if dbID := c.Param("id"); dbID != "" && in.DatabaseID == "" {
		in.DatabaseID = dbID
	}

	in.CreatedBy = userID
	view, err := h.views.Create(c.Request().Context(), &in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, view)
}

func (h *View) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	view, err := h.views.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "view not found"})
	}
	return c.JSON(http.StatusOK, view)
}

func (h *View) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	var in views.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	view, err := h.views.Update(c.Request().Context(), id, &in)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, view)
}

func (h *View) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.views.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *View) List(c *mizu.Ctx) error {
	dbID := c.Param("id")
	list, err := h.views.ListByDatabase(c.Request().Context(), dbID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"views": list})
}

func (h *View) Query(c *mizu.Ctx) error {
	viewID := c.Param("id")
	var in struct {
		Cursor string `json:"cursor"`
		Limit  int    `json:"limit"`
	}
	c.BindJSON(&in, 1<<20)
	result, err := h.views.Query(c.Request().Context(), viewID, in.Cursor, in.Limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, result)
}

// Comment handles comment endpoints.
type Comment struct {
	comments  comments.API
	getUserID func(c *mizu.Ctx) string
}

func NewComment(comments comments.API, getUserID func(c *mizu.Ctx) string) *Comment {
	return &Comment{comments: comments, getUserID: getUserID}
}

func (h *Comment) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	var in comments.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	in.AuthorID = userID
	comment, err := h.comments.Create(c.Request().Context(), &in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, comment)
}

func (h *Comment) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	var in struct {
		Content []blocks.RichText `json:"content"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	comment, err := h.comments.Update(c.Request().Context(), id, in.Content)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, comment)
}

func (h *Comment) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.comments.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Comment) List(c *mizu.Ctx) error {
	pageID := c.Param("id")
	list, err := h.comments.ListByPage(c.Request().Context(), pageID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, list)
}

func (h *Comment) Resolve(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.comments.Resolve(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "resolved"})
}

// Share handles sharing endpoints.
type Share struct {
	sharing   sharing.API
	getUserID func(c *mizu.Ctx) string
}

func NewShare(sharing sharing.API, getUserID func(c *mizu.Ctx) string) *Share {
	return &Share{sharing: sharing, getUserID: getUserID}
}

func (h *Share) Create(c *mizu.Ctx) error {
	pageID := c.Param("id")
	userID := h.getUserID(c)
	var in struct {
		Type       sharing.ShareType   `json:"type"`
		Permission sharing.Permission  `json:"permission"`
		UserID     string              `json:"user_id"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	var share *sharing.Share
	var err error
	if in.Type == sharing.ShareUser {
		share, err = h.sharing.ShareWithUser(c.Request().Context(), pageID, in.UserID, in.Permission, userID)
	} else if in.Type == sharing.ShareLink {
		share, err = h.sharing.CreateShareLink(c.Request().Context(), pageID, sharing.LinkOpts{Permission: in.Permission}, userID)
	} else {
		share, err = h.sharing.EnablePublic(c.Request().Context(), pageID, userID)
	}
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, share)
}

func (h *Share) List(c *mizu.Ctx) error {
	pageID := c.Param("id")
	list, err := h.sharing.ListByPage(c.Request().Context(), pageID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, list)
}

func (h *Share) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.sharing.RemoveUserShare(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// Favorite handles favorite endpoints.
type Favorite struct {
	favorites favorites.API
	getUserID func(c *mizu.Ctx) string
}

func NewFavorite(favorites favorites.API, getUserID func(c *mizu.Ctx) string) *Favorite {
	return &Favorite{favorites: favorites, getUserID: getUserID}
}

func (h *Favorite) Add(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	var in struct {
		PageID      string `json:"page_id"`
		WorkspaceID string `json:"workspace_id"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	fav, err := h.favorites.Add(c.Request().Context(), userID, in.PageID, in.WorkspaceID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, fav)
}

func (h *Favorite) Remove(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	pageID := c.Param("pageID")
	if err := h.favorites.Remove(c.Request().Context(), userID, pageID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "removed"})
}

func (h *Favorite) List(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	workspaceID := c.Param("id")
	list, err := h.favorites.List(c.Request().Context(), userID, workspaceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, list)
}

// Search handles search endpoints.
type Search struct {
	search    search.API
	getUserID func(c *mizu.Ctx) string
}

func NewSearch(search search.API, getUserID func(c *mizu.Ctx) string) *Search {
	return &Search{search: search, getUserID: getUserID}
}

func (h *Search) Search(c *mizu.Ctx) error {
	workspaceID := c.Param("id")
	query := c.Query("q")
	result, err := h.search.Search(c.Request().Context(), workspaceID, query, search.SearchOpts{Limit: 20})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, result)
}

func (h *Search) QuickSearch(c *mizu.Ctx) error {
	workspaceID := c.Param("id")
	query := c.Query("q")
	results, err := h.search.QuickSearch(c.Request().Context(), workspaceID, query, 10)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, results)
}

func (h *Search) Recent(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	workspaceID := c.Param("id")
	pages, err := h.search.GetRecent(c.Request().Context(), userID, workspaceID, 10)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, pages)
}
