package handler

import (
	"html/template"
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/columns"
	"github.com/go-mizu/blueprints/kanban/feature/comments"
	"github.com/go-mizu/blueprints/kanban/feature/cycles"
	"github.com/go-mizu/blueprints/kanban/feature/fields"
	"github.com/go-mizu/blueprints/kanban/feature/issues"
	"github.com/go-mizu/blueprints/kanban/feature/projects"
	"github.com/go-mizu/blueprints/kanban/feature/teams"
	"github.com/go-mizu/blueprints/kanban/feature/users"
	"github.com/go-mizu/blueprints/kanban/feature/workspaces"
)

// Breadcrumb represents a navigation breadcrumb.
type Breadcrumb struct {
	Label string
	URL   string
}

// Stats holds dashboard statistics.
type Stats struct {
	OpenIssues int
	InProgress int
	Completed  int
}

// BoardColumn wraps a column with its issues for template rendering.
type BoardColumn struct {
	*columns.Column
	Issues []*issues.Issue
}

// IssueView wraps an issue with its related data for template rendering.
type IssueView struct {
	*issues.Issue
	Column    *columns.Column
	Assignees []*users.User
	Priority  string
}

// MemberView wraps a team member with user data for template rendering.
type MemberView struct {
	*teams.Member
	User *users.User
}

// WorkspaceMemberView wraps a workspace member with user data for template rendering.
type WorkspaceMemberView struct {
	Member *workspaces.Member
	User   *users.User
}

// ColumnView wraps a column with its issue count for template rendering.
type ColumnView struct {
	*columns.Column
	IssueCount int
}

// LoginData holds data for the login page.
type LoginData struct {
	Title      string
	Subtitle   string
	Error      string
	FooterText string
}

// RegisterData holds data for the registration page.
type RegisterData struct {
	Title      string
	Subtitle   string
	Error      string
	FooterText string
}

// HomeData holds data for the home/dashboard page.
type HomeData struct {
	Title         string
	User          *users.User
	Workspace     *workspaces.Workspace
	Workspaces    []*workspaces.Workspace
	Teams         []*teams.Team
	Projects      []*projects.Project
	ActiveCycle   *cycles.Cycle
	Stats         Stats
	DefaultTeamID string
	ActiveNav     string
	Breadcrumbs   []Breadcrumb
}

// BoardData holds data for the kanban board page.
type BoardData struct {
	Title           string
	User            *users.User
	Workspace       *workspaces.Workspace
	Workspaces      []*workspaces.Workspace
	Teams           []*teams.Team
	Projects        []*projects.Project
	Project         *projects.Project
	Columns         []*BoardColumn
	ActiveProjectID string
	ActiveNav       string
	Breadcrumbs     []Breadcrumb
}

// IssuesData holds data for the issues list page.
type IssuesData struct {
	Title            string
	User             *users.User
	Workspace        *workspaces.Workspace
	Workspaces       []*workspaces.Workspace
	Teams            []*teams.Team
	Issues           []*IssueView
	Columns          []*columns.Column
	Projects         []*projects.Project
	DefaultProjectID string
	ActiveProjectID  string
	TotalCount       int
	ActiveNav        string
	Breadcrumbs      []Breadcrumb
}

// IssueData holds data for the issue detail page.
type IssueData struct {
	Title           string
	User            *users.User
	Workspace       *workspaces.Workspace
	Workspaces      []*workspaces.Workspace
	Teams           []*teams.Team
	Projects        []*projects.Project
	Issue           *issues.Issue
	Project         *projects.Project
	Columns         []*columns.Column
	Comments        []*comments.Comment
	Cycles          []*cycles.Cycle
	Fields          []*fields.Field
	TeamMembers     []*users.User
	ActiveProjectID string
	ActiveNav       string
	Breadcrumbs     []Breadcrumb
}

// CyclesData holds data for the cycles page.
type CyclesData struct {
	Title         string
	User          *users.User
	Workspace     *workspaces.Workspace
	Workspaces    []*workspaces.Workspace
	Teams         []*teams.Team
	Projects      []*projects.Project
	Cycles        []*cycles.Cycle
	DefaultTeamID string
	ActiveNav     string
	Breadcrumbs   []Breadcrumb
}

// TeamData holds data for the team page.
type TeamData struct {
	Title           string
	User            *users.User
	Workspace       *workspaces.Workspace
	Workspaces      []*workspaces.Workspace
	Teams           []*teams.Team
	Team            *teams.Team
	Members         []*MemberView
	Projects        []*projects.Project
	ActiveTeamID    string
	ActiveProjectID string
	ActiveNav       string
	Breadcrumbs     []Breadcrumb
}

// WorkspaceSettingsData holds data for the workspace settings page.
type WorkspaceSettingsData struct {
	Title       string
	User        *users.User
	Workspace   *workspaces.Workspace
	Workspaces  []*workspaces.Workspace
	Teams       []*teams.Team
	Projects    []*projects.Project
	Members     []*WorkspaceMemberView
	ActiveNav   string
	Breadcrumbs []Breadcrumb
}

// ProjectSettingsData holds data for the project settings page.
type ProjectSettingsData struct {
	Title           string
	User            *users.User
	Workspace       *workspaces.Workspace
	Workspaces      []*workspaces.Workspace
	Teams           []*teams.Team
	Projects        []*projects.Project
	Project         *projects.Project
	Columns         []*ColumnView
	ActiveProjectID string
	ActiveNav       string
	Breadcrumbs     []Breadcrumb
}

// ProjectFieldsData holds data for the project fields page.
type ProjectFieldsData struct {
	Title           string
	User            *users.User
	Workspace       *workspaces.Workspace
	Workspaces      []*workspaces.Workspace
	Teams           []*teams.Team
	Projects        []*projects.Project
	Project         *projects.Project
	Fields          []*fields.Field
	ActiveProjectID string
	ActiveNav       string
	Breadcrumbs     []Breadcrumb
}

// Page handles page rendering.
type Page struct {
	templates  map[string]*template.Template
	users      users.API
	workspaces workspaces.API
	teams      teams.API
	projects   projects.API
	columns    columns.API
	issues     issues.API
	cycles     cycles.API
	comments   comments.API
	fields     fields.API
	getUserID  func(*mizu.Ctx) string
}

// NewPage creates a new Page handler.
func NewPage(
	templates map[string]*template.Template,
	users users.API,
	workspaces workspaces.API,
	teams teams.API,
	projects projects.API,
	columns columns.API,
	issues issues.API,
	cycles cycles.API,
	comments comments.API,
	fields fields.API,
	getUserID func(*mizu.Ctx) string,
) *Page {
	return &Page{
		templates:  templates,
		users:      users,
		workspaces: workspaces,
		teams:      teams,
		projects:   projects,
		columns:    columns,
		issues:     issues,
		cycles:     cycles,
		comments:   comments,
		fields:     fields,
		getUserID:  getUserID,
	}
}

func render[T any](h *Page, c *mizu.Ctx, name string, data T) error {
	tmpl, ok := h.templates[name]
	if !ok {
		return c.Text(http.StatusInternalServerError, "Template not found: "+name)
	}

	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.Execute(c.Writer(), data)
}

// Login renders the login page.
func (h *Page) Login(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID != "" {
		http.Redirect(c.Writer(), c.Request(), "/app", http.StatusFound)
		return nil
	}
	return render(h, c, "login", LoginData{
		Title:    "Sign In",
		Subtitle: "Sign in to your account",
	})
}

// Register renders the registration page.
func (h *Page) Register(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID != "" {
		http.Redirect(c.Writer(), c.Request(), "/app", http.StatusFound)
		return nil
	}
	return render(h, c, "register", RegisterData{
		Title:    "Create Account",
		Subtitle: "Create a new account",
	})
}

// Home renders the dashboard/home page.
func (h *Page) Home(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	user, _ := h.users.GetByID(ctx, userID)

	// Get user's workspaces
	workspaceList, _ := h.workspaces.ListByUser(ctx, userID)

	// Get first workspace for dashboard
	var workspace *workspaces.Workspace
	var teamList []*teams.Team
	var projectList []*projects.Project
	var activeCycle *cycles.Cycle
	var defaultTeamID string
	stats := Stats{}

	if len(workspaceList) > 0 {
		workspace = workspaceList[0]

		// Get teams in workspace
		teamList, _ = h.teams.ListByWorkspace(ctx, workspace.ID)

		if len(teamList) > 0 {
			defaultTeamID = teamList[0].ID
			// Get projects from first team
			projectList, _ = h.projects.ListByTeam(ctx, teamList[0].ID)

			// Get active cycle
			activeCycle, _ = h.cycles.GetActive(ctx, teamList[0].ID)

			// Pre-fetch all columns for all projects (avoids N+1 in inner loop)
			columnMap := make(map[string]*columns.Column)
			for _, project := range projectList {
				cols, _ := h.columns.ListByProject(ctx, project.ID)
				for _, col := range cols {
					columnMap[col.ID] = col
				}
			}

			// Calculate stats from projects (now uses in-memory lookup)
			for _, project := range projectList {
				issueList, _ := h.issues.ListByProject(ctx, project.ID)
				for _, issue := range issueList {
					col := columnMap[issue.ColumnID]
					if col != nil {
						switch col.Name {
						case "Done", "Completed":
							stats.Completed++
						case "In Progress", "Doing":
							stats.InProgress++
						default:
							stats.OpenIssues++
						}
					}
				}
			}
		}
	}

	return render(h, c, "home", HomeData{
		Title:         "Dashboard",
		User:          user,
		Workspace:     workspace,
		Workspaces:    workspaceList,
		Teams:         teamList,
		Projects:      projectList,
		ActiveCycle:   activeCycle,
		Stats:         stats,
		DefaultTeamID: defaultTeamID,
		ActiveNav:     "home",
	})
}

// Board renders the kanban board page.
func (h *Page) Board(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	projectID := c.Param("projectID")
	workspaceSlug := c.Param("workspace")

	user, _ := h.users.GetByID(ctx, userID)
	project, err := h.projects.GetByID(ctx, projectID)
	if err != nil {
		http.Redirect(c.Writer(), c.Request(), "/app", http.StatusFound)
		return nil
	}

	// Get workspace
	workspace, _ := h.workspaces.GetBySlug(ctx, workspaceSlug)
	workspaceList, _ := h.workspaces.ListByUser(ctx, userID)

	// Get teams and projects for sidebar
	var teamList []*teams.Team
	var projectList []*projects.Project
	if workspace != nil {
		teamList, _ = h.teams.ListByWorkspace(ctx, workspace.ID)
		projectList, _ = h.projects.ListByTeam(ctx, project.TeamID)
	}

	// Get columns and issues in parallel (2 queries instead of N+1)
	columnList, _ := h.columns.ListByProject(ctx, projectID)
	allIssues, _ := h.issues.ListByProject(ctx, projectID)

	// Group issues by column ID
	issuesByColumn := make(map[string][]*issues.Issue)
	for _, issue := range allIssues {
		issuesByColumn[issue.ColumnID] = append(issuesByColumn[issue.ColumnID], issue)
	}

	// Build board columns with grouped issues
	boardColumns := make([]*BoardColumn, len(columnList))
	for i, col := range columnList {
		boardColumns[i] = &BoardColumn{
			Column: col,
			Issues: issuesByColumn[col.ID],
		}
	}

	return render(h, c, "board", BoardData{
		Title:           project.Name,
		User:            user,
		Workspace:       workspace,
		Workspaces:      workspaceList,
		Teams:           teamList,
		Projects:        projectList,
		Project:         project,
		Columns:         boardColumns,
		ActiveProjectID: projectID,
		ActiveNav:       "issues",
		Breadcrumbs: []Breadcrumb{
			{Label: project.Name, URL: ""},
		},
	})
}

// Issues renders the issues list page.
func (h *Page) Issues(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	workspaceSlug := c.Param("workspace")

	user, _ := h.users.GetByID(ctx, userID)
	workspace, _ := h.workspaces.GetBySlug(ctx, workspaceSlug)
	workspaceList, _ := h.workspaces.ListByUser(ctx, userID)

	// Get all issues from all projects in workspace
	var issueViews []*IssueView
	var columnList []*columns.Column
	var projectList []*projects.Project
	var teamList []*teams.Team
	columnMap := make(map[string]*columns.Column)
	var defaultProjectID string

	if workspace != nil {
		// Phase 1: Collect all projects from all teams
		teamList, _ = h.teams.ListByWorkspace(ctx, workspace.ID)
		for _, team := range teamList {
			teamProjects, _ := h.projects.ListByTeam(ctx, team.ID)
			projectList = append(projectList, teamProjects...)
		}

		if len(projectList) > 0 {
			defaultProjectID = projectList[0].ID
		}

		// Phase 2: Pre-fetch all columns for all projects (batch load)
		for _, project := range projectList {
			cols, _ := h.columns.ListByProject(ctx, project.ID)
			for _, col := range cols {
				columnMap[col.ID] = col
			}
			if len(columnList) == 0 {
				columnList = cols
			}
		}

		// Phase 3: Fetch all issues and build views using pre-cached columns
		for _, project := range projectList {
			issueList, _ := h.issues.ListByProject(ctx, project.ID)
			for _, issue := range issueList {
				iv := &IssueView{
					Issue:    issue,
					Column:   columnMap[issue.ColumnID],
					Priority: "",
				}
				issueViews = append(issueViews, iv)
			}
		}
	}

	return render(h, c, "issues", IssuesData{
		Title:            "Issues",
		User:             user,
		Workspace:        workspace,
		Workspaces:       workspaceList,
		Teams:            teamList,
		Issues:           issueViews,
		Columns:          columnList,
		Projects:         projectList,
		DefaultProjectID: defaultProjectID,
		TotalCount:       len(issueViews),
		ActiveNav:        "issues",
	})
}

// Issue renders the issue detail page.
func (h *Page) Issue(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	issueKey := c.Param("key")
	workspaceSlug := c.Param("workspace")

	user, _ := h.users.GetByID(ctx, userID)
	workspace, _ := h.workspaces.GetBySlug(ctx, workspaceSlug)
	workspaceList, _ := h.workspaces.ListByUser(ctx, userID)

	issue, err := h.issues.GetByKey(ctx, issueKey)
	if err != nil {
		http.Redirect(c.Writer(), c.Request(), "/app", http.StatusFound)
		return nil
	}

	project, _ := h.projects.GetByID(ctx, issue.ProjectID)
	columnList, _ := h.columns.ListByProject(ctx, issue.ProjectID)
	commentList, _ := h.comments.ListByIssue(ctx, issue.ID)

	// Get custom fields for project
	fieldList, _ := h.fields.ListByProject(ctx, issue.ProjectID)

	// Get team for cycles and members
	var cycleList []*cycles.Cycle
	var teamMembers []*users.User
	var teamList []*teams.Team
	var projectList []*projects.Project
	if project != nil {
		cycleList, _ = h.cycles.ListByTeam(ctx, project.TeamID)
		members, _ := h.teams.ListMembers(ctx, project.TeamID)
		teamList, _ = h.teams.ListByWorkspace(ctx, workspace.ID)
		projectList, _ = h.projects.ListByTeam(ctx, project.TeamID)

		// Batch load all users at once (1 query instead of N)
		if len(members) > 0 {
			userIDs := make([]string, len(members))
			for i, m := range members {
				userIDs[i] = m.UserID
			}
			teamMembers, _ = h.users.GetByIDs(ctx, userIDs)
		}
	}

	return render(h, c, "issue", IssueData{
		Title:           issue.Key,
		User:            user,
		Workspace:       workspace,
		Workspaces:      workspaceList,
		Teams:           teamList,
		Projects:        projectList,
		Issue:           issue,
		Project:         project,
		Columns:         columnList,
		Comments:        commentList,
		Cycles:          cycleList,
		Fields:          fieldList,
		TeamMembers:     teamMembers,
		ActiveProjectID: issue.ProjectID,
		ActiveNav:       "issues",
		Breadcrumbs: []Breadcrumb{
			{Label: "Issues", URL: "/" + workspaceSlug + "/issues"},
			{Label: issue.Key, URL: ""},
		},
	})
}

// Cycles renders the cycles page.
func (h *Page) Cycles(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	workspaceSlug := c.Param("workspace")

	user, _ := h.users.GetByID(ctx, userID)
	workspace, _ := h.workspaces.GetBySlug(ctx, workspaceSlug)
	workspaceList, _ := h.workspaces.ListByUser(ctx, userID)

	var cycleList []*cycles.Cycle
	var teamList []*teams.Team
	var projectList []*projects.Project
	var defaultTeamID string

	if workspace != nil {
		teamList, _ = h.teams.ListByWorkspace(ctx, workspace.ID)
		if len(teamList) > 0 {
			defaultTeamID = teamList[0].ID
			cycleList, _ = h.cycles.ListByTeam(ctx, teamList[0].ID)
			projectList, _ = h.projects.ListByTeam(ctx, teamList[0].ID)
		}
	}

	return render(h, c, "cycles", CyclesData{
		Title:         "Cycles",
		User:          user,
		Workspace:     workspace,
		Workspaces:    workspaceList,
		Teams:         teamList,
		Projects:      projectList,
		Cycles:        cycleList,
		DefaultTeamID: defaultTeamID,
		ActiveNav:     "cycles",
	})
}

// Team renders the team page.
func (h *Page) Team(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	teamID := c.Param("teamID")
	workspaceSlug := c.Param("workspace")

	user, _ := h.users.GetByID(ctx, userID)
	workspace, _ := h.workspaces.GetBySlug(ctx, workspaceSlug)
	workspaceList, _ := h.workspaces.ListByUser(ctx, userID)

	team, err := h.teams.GetByID(ctx, teamID)
	if err != nil {
		http.Redirect(c.Writer(), c.Request(), "/app", http.StatusFound)
		return nil
	}

	// Get teams for sidebar
	var teamList []*teams.Team
	if workspace != nil {
		teamList, _ = h.teams.ListByWorkspace(ctx, workspace.ID)
	}

	members, _ := h.teams.ListMembers(ctx, teamID)
	projectList, _ := h.projects.ListByTeam(ctx, teamID)

	// Batch load all users at once (1 query instead of N)
	userMap := make(map[string]*users.User)
	if len(members) > 0 {
		userIDs := make([]string, len(members))
		for i, m := range members {
			userIDs[i] = m.UserID
		}
		userList, _ := h.users.GetByIDs(ctx, userIDs)
		for _, u := range userList {
			userMap[u.ID] = u
		}
	}

	// Build MemberViews with pre-fetched user data
	memberViews := make([]*MemberView, len(members))
	for i, m := range members {
		memberViews[i] = &MemberView{
			Member: m,
			User:   userMap[m.UserID],
		}
	}

	return render(h, c, "team", TeamData{
		Title:        team.Name,
		User:         user,
		Workspace:    workspace,
		Workspaces:   workspaceList,
		Teams:        teamList,
		Team:         team,
		Members:      memberViews,
		Projects:     projectList,
		ActiveTeamID: teamID,
		ActiveNav:    "teams",
		Breadcrumbs: []Breadcrumb{
			{Label: "Teams", URL: "/" + workspaceSlug + "/teams"},
			{Label: team.Name, URL: ""},
		},
	})
}

// WorkspaceSettings renders the workspace settings page.
func (h *Page) WorkspaceSettings(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	workspaceSlug := c.Param("workspace")

	user, _ := h.users.GetByID(ctx, userID)
	workspace, err := h.workspaces.GetBySlug(ctx, workspaceSlug)
	if err != nil {
		http.Redirect(c.Writer(), c.Request(), "/app", http.StatusFound)
		return nil
	}

	workspaceList, _ := h.workspaces.ListByUser(ctx, userID)
	teamList, _ := h.teams.ListByWorkspace(ctx, workspace.ID)
	projectList, _ := h.projects.ListByTeam(ctx, teamList[0].ID)

	// Get workspace members
	members, _ := h.workspaces.ListMembers(ctx, workspace.ID)

	// Batch load all users
	userMap := make(map[string]*users.User)
	if len(members) > 0 {
		userIDs := make([]string, len(members))
		for i, m := range members {
			userIDs[i] = m.UserID
		}
		userList, _ := h.users.GetByIDs(ctx, userIDs)
		for _, u := range userList {
			userMap[u.ID] = u
		}
	}

	// Build MemberViews
	memberViews := make([]*WorkspaceMemberView, len(members))
	for i, m := range members {
		memberViews[i] = &WorkspaceMemberView{
			Member: m,
			User:   userMap[m.UserID],
		}
	}

	return render(h, c, "workspace-settings", WorkspaceSettingsData{
		Title:      "Workspace Settings",
		User:       user,
		Workspace:  workspace,
		Workspaces: workspaceList,
		Teams:      teamList,
		Projects:   projectList,
		Members:    memberViews,
		ActiveNav:  "settings",
	})
}

// ProjectSettings renders the project settings page.
func (h *Page) ProjectSettings(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	projectID := c.Param("projectID")
	workspaceSlug := c.Param("workspace")

	user, _ := h.users.GetByID(ctx, userID)
	project, err := h.projects.GetByID(ctx, projectID)
	if err != nil {
		http.Redirect(c.Writer(), c.Request(), "/app", http.StatusFound)
		return nil
	}

	workspace, _ := h.workspaces.GetBySlug(ctx, workspaceSlug)
	workspaceList, _ := h.workspaces.ListByUser(ctx, userID)
	teamList, _ := h.teams.ListByWorkspace(ctx, workspace.ID)
	projectList, _ := h.projects.ListByTeam(ctx, project.TeamID)

	// Get columns with issue counts
	columnList, _ := h.columns.ListByProject(ctx, projectID)
	allIssues, _ := h.issues.ListByProject(ctx, projectID)

	// Count issues per column
	issueCountByColumn := make(map[string]int)
	for _, issue := range allIssues {
		issueCountByColumn[issue.ColumnID]++
	}

	columnViews := make([]*ColumnView, len(columnList))
	for i, col := range columnList {
		columnViews[i] = &ColumnView{
			Column:     col,
			IssueCount: issueCountByColumn[col.ID],
		}
	}

	return render(h, c, "project-settings", ProjectSettingsData{
		Title:           "Project Settings",
		User:            user,
		Workspace:       workspace,
		Workspaces:      workspaceList,
		Teams:           teamList,
		Projects:        projectList,
		Project:         project,
		Columns:         columnViews,
		ActiveProjectID: projectID,
		ActiveNav:       "settings",
		Breadcrumbs: []Breadcrumb{
			{Label: project.Name, URL: "/w/" + workspaceSlug + "/board/" + projectID},
			{Label: "Settings", URL: ""},
		},
	})
}

// ProjectFields renders the project fields management page.
func (h *Page) ProjectFields(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	projectID := c.Param("projectID")
	workspaceSlug := c.Param("workspace")

	user, _ := h.users.GetByID(ctx, userID)
	project, err := h.projects.GetByID(ctx, projectID)
	if err != nil {
		http.Redirect(c.Writer(), c.Request(), "/app", http.StatusFound)
		return nil
	}

	workspace, _ := h.workspaces.GetBySlug(ctx, workspaceSlug)
	workspaceList, _ := h.workspaces.ListByUser(ctx, userID)
	teamList, _ := h.teams.ListByWorkspace(ctx, workspace.ID)
	projectList, _ := h.projects.ListByTeam(ctx, project.TeamID)

	// Get custom fields
	fieldList, _ := h.fields.ListByProject(ctx, projectID)

	return render(h, c, "project-fields", ProjectFieldsData{
		Title:           "Custom Fields",
		User:            user,
		Workspace:       workspace,
		Workspaces:      workspaceList,
		Teams:           teamList,
		Projects:        projectList,
		Project:         project,
		Fields:          fieldList,
		ActiveProjectID: projectID,
		ActiveNav:       "settings",
		Breadcrumbs: []Breadcrumb{
			{Label: project.Name, URL: "/w/" + workspaceSlug + "/board/" + projectID},
			{Label: "Fields", URL: ""},
		},
	})
}
