package handler

import (
	"html/template"
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/issues"
	"github.com/go-mizu/blueprints/kanban/feature/labels"
	"github.com/go-mizu/blueprints/kanban/feature/notifications"
	"github.com/go-mizu/blueprints/kanban/feature/projects"
	"github.com/go-mizu/blueprints/kanban/feature/sprints"
	"github.com/go-mizu/blueprints/kanban/feature/users"
	"github.com/go-mizu/blueprints/kanban/feature/workspaces"
)

// Page handles page rendering.
type Page struct {
	tmpl          *template.Template
	users         users.API
	workspaces    workspaces.API
	projects      projects.API
	issues        issues.API
	labels        labels.API
	sprints       sprints.API
	notifications notifications.API
	optionalAuth  func(*mizu.Ctx) *users.User
	dev           bool
}

// NewPage creates a new page handler.
func NewPage(
	tmpl *template.Template,
	users users.API,
	workspaces workspaces.API,
	projects projects.API,
	issues issues.API,
	labels labels.API,
	sprints sprints.API,
	notifications notifications.API,
	optionalAuth func(*mizu.Ctx) *users.User,
	dev bool,
) *Page {
	return &Page{
		tmpl:          tmpl,
		users:         users,
		workspaces:    workspaces,
		projects:      projects,
		issues:        issues,
		labels:        labels,
		sprints:       sprints,
		notifications: notifications,
		optionalAuth:  optionalAuth,
		dev:           dev,
	}
}

func (p *Page) render(c *mizu.Ctx, name string, data map[string]any) error {
	if data == nil {
		data = make(map[string]any)
	}
	data["Dev"] = p.dev

	c.Header().Set("Content-Type", "text/html; charset=utf-8")
	return p.tmpl.ExecuteTemplate(c.Writer(), name, data)
}

// Home renders the home page.
func (p *Page) Home(c *mizu.Ctx) error {
	user := p.optionalAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	// Get user's workspaces
	wsList, _ := p.workspaces.ListByUser(c.Context(), user.ID)

	// If user has workspaces, redirect to first one
	if len(wsList) > 0 {
		return c.Redirect(http.StatusFound, "/"+wsList[0].Slug)
	}

	return p.render(c, "pages/home.html", map[string]any{
		"User":       user,
		"Workspaces": wsList,
	})
}

// Login renders the login page.
func (p *Page) Login(c *mizu.Ctx) error {
	user := p.optionalAuth(c)
	if user != nil {
		return c.Redirect(http.StatusFound, "/")
	}
	return p.render(c, "pages/login.html", nil)
}

// Register renders the registration page.
func (p *Page) Register(c *mizu.Ctx) error {
	user := p.optionalAuth(c)
	if user != nil {
		return c.Redirect(http.StatusFound, "/")
	}
	return p.render(c, "pages/register.html", nil)
}

// Workspace renders the workspace dashboard.
func (p *Page) Workspace(c *mizu.Ctx) error {
	user := p.optionalAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	slug := c.Param("workspace")
	ws, err := p.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.Text(http.StatusNotFound, "Workspace not found")
	}

	projectList, _ := p.projects.ListByWorkspace(c.Context(), ws.ID)
	wsList, _ := p.workspaces.ListByUser(c.Context(), user.ID)

	return p.render(c, "pages/workspace.html", map[string]any{
		"User":       user,
		"Workspace":  ws,
		"Workspaces": wsList,
		"Projects":   projectList,
	})
}

// Projects renders the projects list.
func (p *Page) Projects(c *mizu.Ctx) error {
	user := p.optionalAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	slug := c.Param("workspace")
	ws, err := p.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.Text(http.StatusNotFound, "Workspace not found")
	}

	projectList, _ := p.projects.ListByWorkspace(c.Context(), ws.ID)
	wsList, _ := p.workspaces.ListByUser(c.Context(), user.ID)

	return p.render(c, "pages/projects.html", map[string]any{
		"User":       user,
		"Workspace":  ws,
		"Workspaces": wsList,
		"Projects":   projectList,
	})
}

// Board renders the kanban board.
func (p *Page) Board(c *mizu.Ctx) error {
	user := p.optionalAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	slug := c.Param("workspace")
	key := c.Param("key")

	ws, err := p.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.Text(http.StatusNotFound, "Workspace not found")
	}

	project, err := p.projects.GetByKey(c.Context(), ws.ID, key)
	if err != nil {
		return c.Text(http.StatusNotFound, "Project not found")
	}

	issuesByStatus, _ := p.issues.ListByStatus(c.Context(), project.ID)
	labelList, _ := p.labels.ListByProject(c.Context(), project.ID)
	wsList, _ := p.workspaces.ListByUser(c.Context(), user.ID)

	return p.render(c, "pages/board.html", map[string]any{
		"User":           user,
		"Workspace":      ws,
		"Workspaces":     wsList,
		"Project":        project,
		"IssuesByStatus": issuesByStatus,
		"Labels":         labelList,
		"Statuses":       issues.Statuses(),
	})
}

// List renders the list view.
func (p *Page) List(c *mizu.Ctx) error {
	user := p.optionalAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	slug := c.Param("workspace")
	key := c.Param("key")

	ws, err := p.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.Text(http.StatusNotFound, "Workspace not found")
	}

	project, err := p.projects.GetByKey(c.Context(), ws.ID, key)
	if err != nil {
		return c.Text(http.StatusNotFound, "Project not found")
	}

	issueList, _ := p.issues.ListByProject(c.Context(), project.ID, nil)
	labelList, _ := p.labels.ListByProject(c.Context(), project.ID)
	wsList, _ := p.workspaces.ListByUser(c.Context(), user.ID)

	return p.render(c, "pages/list.html", map[string]any{
		"User":       user,
		"Workspace":  ws,
		"Workspaces": wsList,
		"Project":    project,
		"Issues":     issueList,
		"Labels":     labelList,
	})
}

// Backlog renders the backlog view.
func (p *Page) Backlog(c *mizu.Ctx) error {
	user := p.optionalAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	slug := c.Param("workspace")
	key := c.Param("key")

	ws, err := p.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.Text(http.StatusNotFound, "Workspace not found")
	}

	project, err := p.projects.GetByKey(c.Context(), ws.ID, key)
	if err != nil {
		return c.Text(http.StatusNotFound, "Project not found")
	}

	issueList, _ := p.issues.ListByProject(c.Context(), project.ID, &issues.Filter{Status: "backlog"})
	wsList, _ := p.workspaces.ListByUser(c.Context(), user.ID)

	return p.render(c, "pages/backlog.html", map[string]any{
		"User":       user,
		"Workspace":  ws,
		"Workspaces": wsList,
		"Project":    project,
		"Issues":     issueList,
	})
}

// Sprints renders the sprints view.
func (p *Page) Sprints(c *mizu.Ctx) error {
	user := p.optionalAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	slug := c.Param("workspace")
	key := c.Param("key")

	ws, err := p.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.Text(http.StatusNotFound, "Workspace not found")
	}

	project, err := p.projects.GetByKey(c.Context(), ws.ID, key)
	if err != nil {
		return c.Text(http.StatusNotFound, "Project not found")
	}

	sprintList, _ := p.sprints.ListByProject(c.Context(), project.ID)
	wsList, _ := p.workspaces.ListByUser(c.Context(), user.ID)

	return p.render(c, "pages/sprints.html", map[string]any{
		"User":       user,
		"Workspace":  ws,
		"Workspaces": wsList,
		"Project":    project,
		"Sprints":    sprintList,
	})
}

// Issue renders the issue detail page.
func (p *Page) Issue(c *mizu.Ctx) error {
	user := p.optionalAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	slug := c.Param("workspace")
	key := c.Param("key")

	ws, err := p.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.Text(http.StatusNotFound, "Workspace not found")
	}

	issue, err := p.issues.GetByKey(c.Context(), key)
	if err != nil {
		return c.Text(http.StatusNotFound, "Issue not found")
	}

	project, _ := p.projects.GetByID(c.Context(), issue.ProjectID)
	labelList, _ := p.labels.GetByIssue(c.Context(), issue.ID)
	wsList, _ := p.workspaces.ListByUser(c.Context(), user.ID)

	return p.render(c, "pages/issue.html", map[string]any{
		"User":       user,
		"Workspace":  ws,
		"Workspaces": wsList,
		"Project":    project,
		"Issue":      issue,
		"Labels":     labelList,
	})
}

// ProjectSettings renders the project settings page.
func (p *Page) ProjectSettings(c *mizu.Ctx) error {
	user := p.optionalAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	slug := c.Param("workspace")
	key := c.Param("key")

	ws, err := p.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.Text(http.StatusNotFound, "Workspace not found")
	}

	project, err := p.projects.GetByKey(c.Context(), ws.ID, key)
	if err != nil {
		return c.Text(http.StatusNotFound, "Project not found")
	}

	labelList, _ := p.labels.ListByProject(c.Context(), project.ID)
	wsList, _ := p.workspaces.ListByUser(c.Context(), user.ID)

	return p.render(c, "pages/project_settings.html", map[string]any{
		"User":       user,
		"Workspace":  ws,
		"Workspaces": wsList,
		"Project":    project,
		"Labels":     labelList,
	})
}

// WorkspaceSettings renders the workspace settings page.
func (p *Page) WorkspaceSettings(c *mizu.Ctx) error {
	user := p.optionalAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	slug := c.Param("workspace")

	ws, err := p.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.Text(http.StatusNotFound, "Workspace not found")
	}

	members, _ := p.workspaces.ListMembers(c.Context(), ws.ID)
	wsList, _ := p.workspaces.ListByUser(c.Context(), user.ID)

	return p.render(c, "pages/workspace_settings.html", map[string]any{
		"User":       user,
		"Workspace":  ws,
		"Workspaces": wsList,
		"Members":    members,
	})
}

// Members renders the workspace members page.
func (p *Page) Members(c *mizu.Ctx) error {
	user := p.optionalAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	slug := c.Param("workspace")

	ws, err := p.workspaces.GetBySlug(c.Context(), slug)
	if err != nil {
		return c.Text(http.StatusNotFound, "Workspace not found")
	}

	members, _ := p.workspaces.ListMembers(c.Context(), ws.ID)
	wsList, _ := p.workspaces.ListByUser(c.Context(), user.ID)

	return p.render(c, "pages/members.html", map[string]any{
		"User":       user,
		"Workspace":  ws,
		"Workspaces": wsList,
		"Members":    members,
	})
}

// Notifications renders the notifications page.
func (p *Page) Notifications(c *mizu.Ctx) error {
	user := p.optionalAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	notifList, _ := p.notifications.ListByUser(c.Context(), user.ID, 50)
	wsList, _ := p.workspaces.ListByUser(c.Context(), user.ID)

	return p.render(c, "pages/notifications.html", map[string]any{
		"User":          user,
		"Workspaces":    wsList,
		"Notifications": notifList,
	})
}

// Settings renders the user settings page.
func (p *Page) Settings(c *mizu.Ctx) error {
	user := p.optionalAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	wsList, _ := p.workspaces.ListByUser(c.Context(), user.ID)

	return p.render(c, "pages/settings.html", map[string]any{
		"User":       user,
		"Workspaces": wsList,
	})
}
