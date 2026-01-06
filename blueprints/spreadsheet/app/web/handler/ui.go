package handler

import (
	"html/template"
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/spreadsheet/feature/users"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
)

// UI handles UI endpoints.
type UI struct {
	tmpl      *template.Template
	users     users.API
	workbooks workbooks.API
}

// NewUI creates a new UI handler.
func NewUI(tmpl *template.Template, users users.API, workbooks workbooks.API) *UI {
	return &UI{
		tmpl:      tmpl,
		users:     users,
		workbooks: workbooks,
	}
}

// Login renders the login page.
func (h *UI) Login(c *mizu.Ctx) error {
	return h.tmpl.ExecuteTemplate(c.Writer(), "login.html", nil)
}

// Register renders the registration page.
func (h *UI) Register(c *mizu.Ctx) error {
	return h.tmpl.ExecuteTemplate(c.Writer(), "register.html", nil)
}

// AppRedirect redirects to the first workbook or creates a new one.
func (h *UI) AppRedirect(c *mizu.Ctx) error {
	// For now, just redirect to create a new workbook
	http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
	return nil
}

// Spreadsheet renders the main spreadsheet application.
func (h *UI) Spreadsheet(c *mizu.Ctx) error {
	workbookID := c.Param("workbookID")
	sheetID := c.Param("sheetID")

	data := map[string]any{
		"WorkbookID": workbookID,
		"SheetID":    sheetID,
	}

	return h.tmpl.ExecuteTemplate(c.Writer(), "spreadsheet.html", data)
}
