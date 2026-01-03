package handler

import (
	"html/template"
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/databases"
	"github.com/go-mizu/blueprints/workspace/feature/favorites"
	"github.com/go-mizu/blueprints/workspace/feature/members"
	"github.com/go-mizu/blueprints/workspace/feature/pages"
	"github.com/go-mizu/blueprints/workspace/feature/users"
	"github.com/go-mizu/blueprints/workspace/feature/views"
	"github.com/go-mizu/blueprints/workspace/feature/workspaces"
)

// UI handles UI page rendering.
type UI struct {
	templates  map[string]*template.Template
	users      users.API
	workspaces workspaces.API
	members    members.API
	pages      pages.API
	blocks     blocks.API
	databases  databases.API
	views      views.API
	favorites  favorites.API
	getUserID  func(c *mizu.Ctx) string
}

// NewUI creates a new UI handler.
func NewUI(
	templates map[string]*template.Template,
	users users.API,
	workspaces workspaces.API,
	members members.API,
	pages pages.API,
	blocks blocks.API,
	databases databases.API,
	views views.API,
	favorites favorites.API,
	getUserID func(c *mizu.Ctx) string,
) *UI {
	return &UI{
		templates:  templates,
		users:      users,
		workspaces: workspaces,
		members:    members,
		pages:      pages,
		blocks:     blocks,
		databases:  databases,
		views:      views,
		favorites:  favorites,
		getUserID:  getUserID,
	}
}

// Login renders the login page.
func (h *UI) Login(c *mizu.Ctx) error {
	return h.render(c, "login", nil)
}

// Register renders the register page.
func (h *UI) Register(c *mizu.Ctx) error {
	return h.render(c, "register", nil)
}

// AppRedirect redirects to the first workspace.
func (h *UI) AppRedirect(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	wsList, err := h.workspaces.ListByUser(c.Request().Context(), userID)
	if err != nil || len(wsList) == 0 {
		// Create default workspace
		ws, err := h.workspaces.Create(c.Request().Context(), userID, &workspaces.CreateIn{
			Name: "My Workspace",
			Slug: "my-workspace",
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		h.members.Add(c.Request().Context(), ws.ID, userID, members.RoleOwner, userID)
		http.Redirect(c.Writer(), c.Request(), "/w/"+ws.Slug, http.StatusFound)
		return nil
	}

	http.Redirect(c.Writer(), c.Request(), "/w/"+wsList[0].Slug, http.StatusFound)
	return nil
}

// Workspace renders the workspace page.
func (h *UI) Workspace(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	workspaceSlug := c.Param("workspace")

	ws, err := h.workspaces.GetBySlug(c.Request().Context(), workspaceSlug)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "workspace not found"})
	}

	user, _ := h.users.GetByID(c.Request().Context(), userID)
	pagesList, _ := h.pages.ListByWorkspace(c.Request().Context(), ws.ID, pages.ListOpts{})
	favs, _ := h.favorites.List(c.Request().Context(), userID, ws.ID)
	wsList, _ := h.workspaces.ListByUser(c.Request().Context(), userID)

	data := map[string]interface{}{
		"User":       user,
		"Workspace":  ws,
		"Workspaces": wsList,
		"Pages":      pagesList,
		"Favorites":  favs,
	}

	return h.render(c, "workspace", data)
}

// Page renders a page.
func (h *UI) Page(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	workspaceSlug := c.Param("workspace")
	pageID := c.Param("pageID")

	ws, err := h.workspaces.GetBySlug(c.Request().Context(), workspaceSlug)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "workspace not found"})
	}

	page, err := h.pages.GetByID(c.Request().Context(), pageID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "page not found"})
	}

	blocksList, _ := h.blocks.GetByPage(c.Request().Context(), pageID)
	breadcrumb, _ := h.pages.GetBreadcrumb(c.Request().Context(), pageID)
	user, _ := h.users.GetByID(c.Request().Context(), userID)
	pagesList, _ := h.pages.ListByWorkspace(c.Request().Context(), ws.ID, pages.ListOpts{})
	favs, _ := h.favorites.List(c.Request().Context(), userID, ws.ID)
	wsList, _ := h.workspaces.ListByUser(c.Request().Context(), userID)

	data := map[string]interface{}{
		"User":       user,
		"Workspace":  ws,
		"Workspaces": wsList,
		"Pages":      pagesList,
		"Favorites":  favs,
		"Page":       page,
		"Blocks":     blocksList,
		"Breadcrumb": breadcrumb,
	}

	return h.render(c, "page", data)
}

// Database renders a database view.
func (h *UI) Database(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	workspaceSlug := c.Param("workspace")
	databaseID := c.Param("databaseID")

	ws, err := h.workspaces.GetBySlug(c.Request().Context(), workspaceSlug)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "workspace not found"})
	}

	db, err := h.databases.GetByID(c.Request().Context(), databaseID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "database not found"})
	}

	viewsList, _ := h.views.ListByDatabase(c.Request().Context(), databaseID)
	user, _ := h.users.GetByID(c.Request().Context(), userID)
	pagesList, _ := h.pages.ListByWorkspace(c.Request().Context(), ws.ID, pages.ListOpts{})
	favs, _ := h.favorites.List(c.Request().Context(), userID, ws.ID)
	wsList, _ := h.workspaces.ListByUser(c.Request().Context(), userID)

	data := map[string]interface{}{
		"User":       user,
		"Workspace":  ws,
		"Workspaces": wsList,
		"Pages":      pagesList,
		"Favorites":  favs,
		"Database":   db,
		"Views":      viewsList,
	}

	return h.render(c, "database", data)
}

// Search renders the search page.
func (h *UI) Search(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	workspaceSlug := c.Param("workspace")

	ws, err := h.workspaces.GetBySlug(c.Request().Context(), workspaceSlug)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "workspace not found"})
	}

	user, _ := h.users.GetByID(c.Request().Context(), userID)
	pagesList, _ := h.pages.ListByWorkspace(c.Request().Context(), ws.ID, pages.ListOpts{})
	wsList, _ := h.workspaces.ListByUser(c.Request().Context(), userID)

	data := map[string]interface{}{
		"User":       user,
		"Workspace":  ws,
		"Workspaces": wsList,
		"Pages":      pagesList,
		"Query":      c.Query("q"),
	}

	return h.render(c, "search", data)
}

// Settings renders the workspace settings page.
func (h *UI) Settings(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	workspaceSlug := c.Param("workspace")

	ws, err := h.workspaces.GetBySlug(c.Request().Context(), workspaceSlug)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "workspace not found"})
	}

	user, _ := h.users.GetByID(c.Request().Context(), userID)
	membersList, _ := h.members.List(c.Request().Context(), ws.ID)
	wsList, _ := h.workspaces.ListByUser(c.Request().Context(), userID)

	data := map[string]interface{}{
		"User":       user,
		"Workspace":  ws,
		"Workspaces": wsList,
		"Members":    membersList,
	}

	return h.render(c, "settings", data)
}

func (h *UI) render(c *mizu.Ctx, name string, data interface{}) error {
	tmpl, ok := h.templates[name]
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "template not found"})
	}

	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.Execute(c.Writer(), data)
}
