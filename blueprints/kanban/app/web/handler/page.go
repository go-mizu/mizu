package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/activities"
	"github.com/go-mizu/blueprints/kanban/feature/assignees"
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

// CommentView wraps a comment with author info for template rendering.
type CommentView struct {
	*comments.Comment
	AuthorName string
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
	Title           string
	User            *users.User
	Workspace       *workspaces.Workspace
	Workspaces      []*workspaces.Workspace
	Teams           []*teams.Team
	Projects        []*projects.Project
	ActiveCycle     *cycles.Cycle
	Stats           Stats
	DefaultTeamID   string
	ActiveTeamID    string
	ActiveProjectID string
	ActiveNav       string
	Breadcrumbs     []Breadcrumb
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
	TeamMembers     []*users.User
	Cycles          []*cycles.Cycle
	ActiveTeamID    string
	ActiveProjectID string
	ActiveNav       string
	Breadcrumbs     []Breadcrumb
}

// IssueStatusGroup groups issues by status for the issues list view.
type IssueStatusGroup struct {
	Status string
	Issues []*IssueView
}

// IssuesData holds data for the issues list page.
type IssuesData struct {
	Title            string
	User             *users.User
	Workspace        *workspaces.Workspace
	Workspaces       []*workspaces.Workspace
	Teams            []*teams.Team
	Issues           []*IssueView
	IssuesByStatus   []*IssueStatusGroup
	Columns          []*columns.Column
	Projects         []*projects.Project
	TeamMembers      []*users.User
	Cycles           []*cycles.Cycle
	DefaultProjectID string
	ActiveTeamID     string
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
	Comments        []*CommentView
	Activities      []*activities.ActivityWithContext
	Cycles          []*cycles.Cycle
	Fields          []*fields.Field
	TeamMembers     []*users.User
	Assignees       []string // User IDs of assigned users
	ActiveTeamID    string
	ActiveProjectID string
	ActiveNav       string
	Breadcrumbs     []Breadcrumb
}

// CyclesData holds data for the cycles page.
type CyclesData struct {
	Title           string
	User            *users.User
	Workspace       *workspaces.Workspace
	Workspaces      []*workspaces.Workspace
	Teams           []*teams.Team
	Projects        []*projects.Project
	Cycles          []*cycles.Cycle
	Columns         []*columns.Column
	TeamMembers     []*users.User
	DefaultTeamID   string
	ActiveTeamID    string
	ActiveProjectID string
	ActiveNav       string
	Breadcrumbs     []Breadcrumb
}

// ActivitiesData holds data for the activities page.
type ActivitiesData struct {
	Title           string
	User            *users.User
	Workspace       *workspaces.Workspace
	Workspaces      []*workspaces.Workspace
	Teams           []*teams.Team
	Projects        []*projects.Project
	Activities      []*activities.ActivityWithContext
	Columns         []*columns.Column
	TeamMembers     []*users.User
	Cycles          []*cycles.Cycle
	ActiveTeamID    string
	ActiveProjectID string
	ActiveNav       string
	Breadcrumbs     []Breadcrumb
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
	Columns         []*columns.Column
	TeamMembers     []*users.User
	Cycles          []*cycles.Cycle
	ActiveTeamID    string
	ActiveProjectID string
	ActiveNav       string
	Breadcrumbs     []Breadcrumb
}

// WorkspaceSettingsData holds data for the workspace settings page.
type WorkspaceSettingsData struct {
	Title           string
	User            *users.User
	Workspace       *workspaces.Workspace
	Workspaces      []*workspaces.Workspace
	Teams           []*teams.Team
	Projects        []*projects.Project
	Members         []*WorkspaceMemberView
	ActiveTeamID    string
	ActiveProjectID string
	ActiveNav       string
	Breadcrumbs     []Breadcrumb
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
	ActiveTeamID    string
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
	ActiveTeamID    string
	ActiveProjectID string
	ActiveNav       string
	Breadcrumbs     []Breadcrumb
}

// InboxIssue wraps an issue with inbox-specific metadata.
type InboxIssue struct {
	*issues.Issue
	Project   *projects.Project
	Column    *columns.Column
	Assignees []*users.User
	TimeGroup string // "today", "yesterday", "this_week", "older"
}

// CalendarDay represents a single day in the calendar.
type CalendarDay struct {
	Date         time.Time
	Issues       []*CalendarIssue
	IsToday      bool
	IsWeekend    bool
	IsOtherMonth bool
}

// CalendarIssue wraps an issue with calendar-specific metadata.
type CalendarIssue struct {
	*issues.Issue
	Project      *projects.Project
	Column       *columns.Column
	DaysUntilDue int
	IsOverdue    bool
}

// CalendarData holds data for the calendar page.
type CalendarData struct {
	Title            string
	User             *users.User
	Workspace        *workspaces.Workspace
	Workspaces       []*workspaces.Workspace
	Teams            []*teams.Team
	Projects         []*projects.Project
	Year             int
	Month            time.Month
	MonthName        string
	Days             [][]CalendarDay
	PrevMonth        string
	NextMonth        string
	Today            time.Time
	ActiveView       string
	Columns          []*columns.Column
	TeamMembers      []*users.User
	Cycles           []*cycles.Cycle
	DefaultProjectID string
	ActiveTeamID     string
	ActiveProjectID  string
	ActiveNav        string
	Breadcrumbs      []Breadcrumb
}

// GanttIssue wraps an issue with Gantt-specific metadata.
type GanttIssue struct {
	*issues.Issue
	Project          *projects.Project
	Column           *columns.Column
	LeftOffset       float64
	Width            float64
	Row              int
	EffectiveStart   time.Time
	EffectiveEnd     time.Time
	HasExplicitDates bool
}

// GanttHeaderDate represents a date marker in the timeline header.
type GanttHeaderDate struct {
	Date      time.Time
	Label     string
	Offset    float64
	IsToday   bool
	IsWeekend bool
}

// GanttGroup represents a group of issues.
type GanttGroup struct {
	ID     string
	Name   string
	Issues []*GanttIssue
}

// GanttData holds data for the Gantt chart page.
type GanttData struct {
	Title            string
	User             *users.User
	Workspace        *workspaces.Workspace
	Workspaces       []*workspaces.Workspace
	Teams            []*teams.Team
	Projects         []*projects.Project
	Issues           []*GanttIssue
	TimelineStart    time.Time
	TimelineEnd      time.Time
	TimelineDays     int
	TodayOffset      float64
	Scale            string
	HeaderDates      []GanttHeaderDate
	GroupBy          string
	Groups           []*GanttGroup
	ActiveView       string
	Columns          []*columns.Column
	TeamMembers      []*users.User
	Cycles           []*cycles.Cycle
	DefaultProjectID string
	ActiveTeamID     string
	ActiveProjectID  string
	ActiveNav        string
	Breadcrumbs      []Breadcrumb
}

// IssueGroup groups issues by time.
type IssueGroup struct {
	Label  string // "Today", "Yesterday", "This Week", "Older"
	Key    string // "today", "yesterday", "this_week", "older"
	Issues []*InboxIssue
}

// InboxData holds data for the inbox page.
type InboxData struct {
	Title            string
	User             *users.User
	Workspace        *workspaces.Workspace
	Workspaces       []*workspaces.Workspace
	Teams            []*teams.Team
	Projects         []*projects.Project
	DefaultProject   *projects.Project
	DefaultProjectID string
	DefaultTeamID    string

	// Inbox-specific
	IssueGroups   []*IssueGroup
	ActiveTab     string // "assigned", "created", "all"
	AssignedCount int
	CreatedCount  int

	// For create issue form
	Columns     []*columns.Column
	TeamMembers []*users.User
	Cycles      []*cycles.Cycle

	// Standard fields
	ActiveTeamID    string
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
	activities activities.API
	assignees  assignees.API
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
	activities activities.API,
	assigneesAPI assignees.API,
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
		activities: activities,
		assignees:  assigneesAPI,
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

// AppRedirect redirects /app to the first workspace's inbox.
func (h *Page) AppRedirect(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()

	// Get user's workspaces
	workspaceList, _ := h.workspaces.ListByUser(ctx, userID)
	if len(workspaceList) > 0 {
		// Redirect to first workspace's inbox
		http.Redirect(c.Writer(), c.Request(), fmt.Sprintf("/w/%s/inbox", workspaceList[0].Slug), http.StatusFound)
		return nil
	}

	// No workspaces, redirect to login
	http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
	return nil
}

// Inbox renders the inbox page with issue list and create form.
func (h *Page) Inbox(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	tab := c.Query("tab")
	if tab == "" {
		tab = "assigned"
	}
	workspaceSlug := c.Param("workspace")

	user, _ := h.users.GetByID(ctx, userID)
	workspace, _ := h.workspaces.GetBySlug(ctx, workspaceSlug)
	workspaceList, _ := h.workspaces.ListByUser(ctx, userID)

	// Get teams and projects
	var teamList []*teams.Team
	var projectList []*projects.Project
	var defaultProjectID string
	var defaultTeamID string
	var defaultProject *projects.Project

	if workspace != nil {
		teamList, _ = h.teams.ListByWorkspace(ctx, workspace.ID)
		if len(teamList) > 0 {
			defaultTeamID = teamList[0].ID
			projectList, _ = h.projects.ListByTeam(ctx, teamList[0].ID)
			if len(projectList) > 0 {
				defaultProjectID = projectList[0].ID
				defaultProject = projectList[0]
			}
		}
	}

	// Pre-fetch all columns and projects for lookup
	columnMap := make(map[string]*columns.Column)
	projectMap := make(map[string]*projects.Project)
	for _, project := range projectList {
		projectMap[project.ID] = project
		cols, _ := h.columns.ListByProject(ctx, project.ID)
		for _, col := range cols {
			columnMap[col.ID] = col
		}
	}

	// Fetch all issues from workspace
	var allIssues []*issues.Issue
	for _, project := range projectList {
		issueList, _ := h.issues.ListByProject(ctx, project.ID)
		allIssues = append(allIssues, issueList...)
	}

	// Filter issues based on tab
	var filteredIssues []*issues.Issue
	var assignedCount, createdCount int

	for _, issue := range allIssues {
		isCreator := issue.CreatorID == userID
		// TODO: Check assignees when assignee API is available
		isAssigned := isCreator // Fallback: show creator's issues as assigned

		if isAssigned {
			assignedCount++
		}
		if isCreator {
			createdCount++
		}

		switch tab {
		case "assigned":
			if isAssigned {
				filteredIssues = append(filteredIssues, issue)
			}
		case "created":
			if isCreator {
				filteredIssues = append(filteredIssues, issue)
			}
		case "all":
			filteredIssues = append(filteredIssues, issue)
		}
	}

	// Group issues by time
	issueGroups := groupIssuesByTime(filteredIssues, columnMap, projectMap)

	// Get columns for default project (for create form)
	var columnList []*columns.Column
	if defaultProjectID != "" {
		columnList, _ = h.columns.ListByProject(ctx, defaultProjectID)
	}

	// Get cycles for default team
	var cycleList []*cycles.Cycle
	if defaultTeamID != "" {
		cycleList, _ = h.cycles.ListByTeam(ctx, defaultTeamID)
	}

	// Get team members
	var teamMembers []*users.User
	if defaultTeamID != "" {
		members, _ := h.teams.ListMembers(ctx, defaultTeamID)
		if len(members) > 0 {
			userIDs := make([]string, len(members))
			for i, m := range members {
				userIDs[i] = m.UserID
			}
			teamMembers, _ = h.users.GetByIDs(ctx, userIDs)
		}
	}

	return render(h, c, "inbox", InboxData{
		Title:            "Inbox",
		User:             user,
		Workspace:        workspace,
		Workspaces:       workspaceList,
		Teams:            teamList,
		Projects:         projectList,
		DefaultProject:   defaultProject,
		DefaultProjectID: defaultProjectID,
		DefaultTeamID:    defaultTeamID,
		IssueGroups:      issueGroups,
		ActiveTab:        tab,
		AssignedCount:    assignedCount,
		CreatedCount:     createdCount,
		Columns:          columnList,
		TeamMembers:      teamMembers,
		Cycles:           cycleList,
		ActiveNav:        "inbox",
	})
}

// groupIssuesByTime groups issues into time-based sections.
func groupIssuesByTime(issueList []*issues.Issue, columnMap map[string]*columns.Column, projectMap map[string]*projects.Project) []*IssueGroup {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)
	weekAgo := today.AddDate(0, 0, -7)

	groups := map[string]*IssueGroup{
		"today":     {Label: "Today", Key: "today", Issues: []*InboxIssue{}},
		"yesterday": {Label: "Yesterday", Key: "yesterday", Issues: []*InboxIssue{}},
		"this_week": {Label: "This Week", Key: "this_week", Issues: []*InboxIssue{}},
		"older":     {Label: "Older", Key: "older", Issues: []*InboxIssue{}},
	}

	for _, issue := range issueList {
		var groupKey string
		if issue.UpdatedAt.After(today) || issue.UpdatedAt.Equal(today) {
			groupKey = "today"
		} else if issue.UpdatedAt.After(yesterday) || issue.UpdatedAt.Equal(yesterday) {
			groupKey = "yesterday"
		} else if issue.UpdatedAt.After(weekAgo) {
			groupKey = "this_week"
		} else {
			groupKey = "older"
		}

		inboxIssue := &InboxIssue{
			Issue:     issue,
			Column:    columnMap[issue.ColumnID],
			Project:   projectMap[issue.ProjectID],
			TimeGroup: groupKey,
		}
		groups[groupKey].Issues = append(groups[groupKey].Issues, inboxIssue)
	}

	// Return in order
	return []*IssueGroup{
		groups["today"],
		groups["yesterday"],
		groups["this_week"],
		groups["older"],
	}
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

	// Get team members and cycles for the create issue modal
	var teamMembers []*users.User
	var cycleList []*cycles.Cycle
	if project != nil {
		members, _ := h.teams.ListMembers(ctx, project.TeamID)
		cycleList, _ = h.cycles.ListByTeam(ctx, project.TeamID)

		if len(members) > 0 {
			userIDs := make([]string, len(members))
			for i, m := range members {
				userIDs[i] = m.UserID
			}
			teamMembers, _ = h.users.GetByIDs(ctx, userIDs)
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
		TeamMembers:     teamMembers,
		Cycles:          cycleList,
		ActiveProjectID: projectID,
		ActiveNav:       "issues",
		Breadcrumbs: []Breadcrumb{
			{Label: project.Name, URL: ""},
		},
	})
}

// Issues renders the issues page with different view modes.
func (h *Page) Issues(c *mizu.Ctx) error {
	view := c.Query("view")
	switch view {
	case "calendar":
		return h.IssuesCalendar(c)
	case "gantt":
		return h.IssuesGantt(c)
	default:
		return h.IssuesList(c)
	}
}

// IssuesList renders the issues list view.
func (h *Page) IssuesList(c *mizu.Ctx) error {
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

	// Group issues by status (column name)
	issuesByStatus := groupIssuesByStatus(issueViews, columnList)

	// Get team members and cycles for the create issue modal
	var teamMembers []*users.User
	var cycleList []*cycles.Cycle
	if len(teamList) > 0 {
		members, _ := h.teams.ListMembers(ctx, teamList[0].ID)
		cycleList, _ = h.cycles.ListByTeam(ctx, teamList[0].ID)

		if len(members) > 0 {
			userIDs := make([]string, len(members))
			for i, m := range members {
				userIDs[i] = m.UserID
			}
			teamMembers, _ = h.users.GetByIDs(ctx, userIDs)
		}
	}

	return render(h, c, "issues", IssuesData{
		Title:            "Issues",
		User:             user,
		Workspace:        workspace,
		Workspaces:       workspaceList,
		Teams:            teamList,
		Issues:           issueViews,
		IssuesByStatus:   issuesByStatus,
		Columns:          columnList,
		Projects:         projectList,
		TeamMembers:      teamMembers,
		Cycles:           cycleList,
		DefaultProjectID: defaultProjectID,
		TotalCount:       len(issueViews),
		ActiveNav:        "issues",
	})
}

// groupIssuesByStatus groups issues by their column/status.
func groupIssuesByStatus(issueViews []*IssueView, columnList []*columns.Column) []*IssueStatusGroup {
	// Create a map to hold issues by status
	statusMap := make(map[string][]*IssueView)
	statusOrder := []string{}

	// First, establish order from column list
	for _, col := range columnList {
		if _, exists := statusMap[col.Name]; !exists {
			statusMap[col.Name] = []*IssueView{}
			statusOrder = append(statusOrder, col.Name)
		}
	}

	// Group issues
	for _, iv := range issueViews {
		status := "Unknown"
		if iv.Column != nil {
			status = iv.Column.Name
		}
		if _, exists := statusMap[status]; !exists {
			statusMap[status] = []*IssueView{}
			statusOrder = append(statusOrder, status)
		}
		statusMap[status] = append(statusMap[status], iv)
	}

	// Build result in order
	result := make([]*IssueStatusGroup, 0, len(statusOrder))
	for _, status := range statusOrder {
		if issues, ok := statusMap[status]; ok && len(issues) > 0 {
			result = append(result, &IssueStatusGroup{
				Status: status,
				Issues: issues,
			})
		}
	}

	return result
}

// IssuesCalendar renders the calendar view.
func (h *Page) IssuesCalendar(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	workspaceSlug := c.Param("workspace")

	// Parse month parameter or use current
	monthStr := c.Query("month")
	var year int
	var month time.Month
	if monthStr != "" {
		t, err := time.Parse("2006-01", monthStr)
		if err == nil {
			year, month = t.Year(), t.Month()
		}
	}
	if year == 0 {
		now := time.Now()
		year, month = now.Year(), now.Month()
	}

	user, _ := h.users.GetByID(ctx, userID)
	workspace, _ := h.workspaces.GetBySlug(ctx, workspaceSlug)
	workspaceList, _ := h.workspaces.ListByUser(ctx, userID)

	var projectList []*projects.Project
	var teamList []*teams.Team
	var columnList []*columns.Column
	var cycleList []*cycles.Cycle
	var teamMembers []*users.User
	var allIssues []*issues.Issue
	columnMap := make(map[string]*columns.Column)
	projectMap := make(map[string]*projects.Project)
	var defaultProjectID string

	if workspace != nil {
		teamList, _ = h.teams.ListByWorkspace(ctx, workspace.ID)
		for _, team := range teamList {
			teamProjects, _ := h.projects.ListByTeam(ctx, team.ID)
			projectList = append(projectList, teamProjects...)
		}

		if len(projectList) > 0 {
			defaultProjectID = projectList[0].ID
		}

		for _, project := range projectList {
			projectMap[project.ID] = project
			cols, _ := h.columns.ListByProject(ctx, project.ID)
			for _, col := range cols {
				columnMap[col.ID] = col
			}
			if len(columnList) == 0 {
				columnList = cols
			}
			issueList, _ := h.issues.ListByProject(ctx, project.ID)
			allIssues = append(allIssues, issueList...)
		}

		// Get cycles and team members from first team
		if len(teamList) > 0 {
			cycleList, _ = h.cycles.ListByTeam(ctx, teamList[0].ID)
			members, _ := h.teams.ListMembers(ctx, teamList[0].ID)
			if len(members) > 0 {
				userIDs := make([]string, len(members))
				for i, m := range members {
					userIDs[i] = m.UserID
				}
				teamMembers, _ = h.users.GetByIDs(ctx, userIDs)
			}
		}
	}

	// Build calendar grid
	days := buildCalendarGrid(year, month, allIssues, columnMap, projectMap)

	// Calculate prev/next months
	prevMonth := time.Date(year, month-1, 1, 0, 0, 0, 0, time.UTC)
	nextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)

	return render(h, c, "calendar", CalendarData{
		Title:            "Calendar",
		User:             user,
		Workspace:        workspace,
		Workspaces:       workspaceList,
		Teams:            teamList,
		Projects:         projectList,
		Year:             year,
		Month:            month,
		MonthName:        month.String(),
		Days:             days,
		PrevMonth:        prevMonth.Format("2006-01"),
		NextMonth:        nextMonth.Format("2006-01"),
		Today:            time.Now(),
		ActiveView:       "calendar",
		Columns:          columnList,
		TeamMembers:      teamMembers,
		Cycles:           cycleList,
		DefaultProjectID: defaultProjectID,
		ActiveNav:        "issues",
	})
}

// IssuesGantt renders the Gantt chart view.
func (h *Page) IssuesGantt(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	workspaceSlug := c.Param("workspace")

	scale := c.Query("scale")
	if scale == "" {
		scale = "week"
	}
	groupBy := c.Query("group")
	if groupBy == "" {
		groupBy = "none"
	}

	user, _ := h.users.GetByID(ctx, userID)
	workspace, _ := h.workspaces.GetBySlug(ctx, workspaceSlug)
	workspaceList, _ := h.workspaces.ListByUser(ctx, userID)

	var projectList []*projects.Project
	var teamList []*teams.Team
	var columnList []*columns.Column
	var cycleList []*cycles.Cycle
	var teamMembers []*users.User
	var allIssues []*issues.Issue
	columnMap := make(map[string]*columns.Column)
	projectMap := make(map[string]*projects.Project)
	var defaultProjectID string

	if workspace != nil {
		teamList, _ = h.teams.ListByWorkspace(ctx, workspace.ID)
		for _, team := range teamList {
			teamProjects, _ := h.projects.ListByTeam(ctx, team.ID)
			projectList = append(projectList, teamProjects...)
		}

		if len(projectList) > 0 {
			defaultProjectID = projectList[0].ID
		}

		for _, project := range projectList {
			projectMap[project.ID] = project
			cols, _ := h.columns.ListByProject(ctx, project.ID)
			for _, col := range cols {
				columnMap[col.ID] = col
			}
			if len(columnList) == 0 {
				columnList = cols
			}
			issueList, _ := h.issues.ListByProject(ctx, project.ID)
			allIssues = append(allIssues, issueList...)
		}

		// Get cycles and team members from first team
		if len(teamList) > 0 {
			cycleList, _ = h.cycles.ListByTeam(ctx, teamList[0].ID)
			members, _ := h.teams.ListMembers(ctx, teamList[0].ID)
			if len(members) > 0 {
				userIDs := make([]string, len(members))
				for i, m := range members {
					userIDs[i] = m.UserID
				}
				teamMembers, _ = h.users.GetByIDs(ctx, userIDs)
			}
		}
	}

	// Calculate timeline range (default: 4 weeks before and after today)
	now := time.Now()
	timelineStart := now.AddDate(0, 0, -28)
	timelineEnd := now.AddDate(0, 0, 28)
	timelineDays := int(timelineEnd.Sub(timelineStart).Hours() / 24)

	// Build Gantt issues
	ganttIssues := buildGanttIssues(allIssues, columnMap, projectMap, timelineStart, timelineEnd)

	// Build header dates
	headerDates := buildGanttHeaderDates(timelineStart, timelineEnd, scale)

	// Calculate today offset
	todayOffset := float64(now.Sub(timelineStart).Hours()/24) / float64(timelineDays) * 100

	// Group issues if needed
	groups := groupGanttIssues(ganttIssues, groupBy, projectMap, columnMap)

	return render(h, c, "gantt", GanttData{
		Title:            "Gantt Chart",
		User:             user,
		Workspace:        workspace,
		Workspaces:       workspaceList,
		Teams:            teamList,
		Projects:         projectList,
		Issues:           ganttIssues,
		TimelineStart:    timelineStart,
		TimelineEnd:      timelineEnd,
		TimelineDays:     timelineDays,
		TodayOffset:      todayOffset,
		Scale:            scale,
		HeaderDates:      headerDates,
		GroupBy:          groupBy,
		Groups:           groups,
		ActiveView:       "gantt",
		Columns:          columnList,
		TeamMembers:      teamMembers,
		Cycles:           cycleList,
		DefaultProjectID: defaultProjectID,
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
	activityList, _ := h.activities.ListByIssue(ctx, issue.ID)

	// Get assignees for this issue
	var assigneeIDs []string
	if h.assignees != nil {
		assigneeIDs, _ = h.assignees.List(ctx, issue.ID)
	}

	// Get custom fields for project
	fieldList, _ := h.fields.ListByProject(ctx, issue.ProjectID)

	// Get team for cycles and members
	var cycleList []*cycles.Cycle
	var teamMembers []*users.User
	var teamList []*teams.Team
	var projectList []*projects.Project
	userMap := make(map[string]*users.User)
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
			// Build user map for activity actor lookup
			for _, u := range teamMembers {
				userMap[u.ID] = u
			}
		}
	}

	// Enhance activities with actor names
	activityWithContext := make([]*activities.ActivityWithContext, len(activityList))
	for i, a := range activityList {
		actorName := ""
		if u, ok := userMap[a.ActorID]; ok {
			actorName = u.DisplayName
		}
		activityWithContext[i] = &activities.ActivityWithContext{
			Activity:  a,
			ActorName: actorName,
			IssueKey:  issue.Key,
		}
	}

	// Enhance comments with author names
	commentViews := make([]*CommentView, len(commentList))
	for i, c := range commentList {
		authorName := "Someone"
		if u, ok := userMap[c.AuthorID]; ok {
			authorName = u.DisplayName
		}
		commentViews[i] = &CommentView{
			Comment:    c,
			AuthorName: authorName,
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
		Comments:        commentViews,
		Activities:      activityWithContext,
		Cycles:          cycleList,
		Fields:          fieldList,
		TeamMembers:     teamMembers,
		Assignees:       assigneeIDs,
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
	var columnList []*columns.Column
	var teamMembers []*users.User
	var defaultTeamID string

	if workspace != nil {
		teamList, _ = h.teams.ListByWorkspace(ctx, workspace.ID)
		if len(teamList) > 0 {
			defaultTeamID = teamList[0].ID
			cycleList, _ = h.cycles.ListByTeam(ctx, teamList[0].ID)
			projectList, _ = h.projects.ListByTeam(ctx, teamList[0].ID)

			// Get columns from first project for global create modal
			if len(projectList) > 0 {
				columnList, _ = h.columns.ListByProject(ctx, projectList[0].ID)
			}

			// Get team members
			members, _ := h.teams.ListMembers(ctx, teamList[0].ID)
			if len(members) > 0 {
				userIDs := make([]string, len(members))
				for i, m := range members {
					userIDs[i] = m.UserID
				}
				teamMembers, _ = h.users.GetByIDs(ctx, userIDs)
			}
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
		Columns:       columnList,
		TeamMembers:   teamMembers,
		DefaultTeamID: defaultTeamID,
		ActiveNav:     "cycles",
	})
}

// Activities renders the activities page.
func (h *Page) Activities(c *mizu.Ctx) error {
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

	var activityList []*activities.ActivityWithContext
	var teamList []*teams.Team
	var projectList []*projects.Project
	var columnList []*columns.Column
	var cycleList []*cycles.Cycle
	var teamMembers []*users.User

	if workspace != nil {
		// Get activities for this workspace
		activityList, _ = h.activities.ListByWorkspace(ctx, workspace.ID, 100, 0)

		teamList, _ = h.teams.ListByWorkspace(ctx, workspace.ID)
		if len(teamList) > 0 {
			projectList, _ = h.projects.ListByTeam(ctx, teamList[0].ID)
			cycleList, _ = h.cycles.ListByTeam(ctx, teamList[0].ID)

			// Get columns from first project for global create modal
			if len(projectList) > 0 {
				columnList, _ = h.columns.ListByProject(ctx, projectList[0].ID)
			}

			// Get team members
			members, _ := h.teams.ListMembers(ctx, teamList[0].ID)
			if len(members) > 0 {
				userIDs := make([]string, len(members))
				for i, m := range members {
					userIDs[i] = m.UserID
				}
				teamMembers, _ = h.users.GetByIDs(ctx, userIDs)
			}
		}
	}

	return render(h, c, "activities", ActivitiesData{
		Title:       "Activities",
		User:        user,
		Workspace:   workspace,
		Workspaces:  workspaceList,
		Teams:       teamList,
		Projects:    projectList,
		Activities:  activityList,
		Columns:     columnList,
		TeamMembers: teamMembers,
		Cycles:      cycleList,
		ActiveNav:   "activities",
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
	cycleList, _ := h.cycles.ListByTeam(ctx, teamID)

	// Get columns from first project for global create modal
	var columnList []*columns.Column
	if len(projectList) > 0 {
		columnList, _ = h.columns.ListByProject(ctx, projectList[0].ID)
	}

	// Batch load all users at once (1 query instead of N)
	userMap := make(map[string]*users.User)
	var teamMembers []*users.User
	if len(members) > 0 {
		userIDs := make([]string, len(members))
		for i, m := range members {
			userIDs[i] = m.UserID
		}
		userList, _ := h.users.GetByIDs(ctx, userIDs)
		teamMembers = userList
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
		Columns:      columnList,
		TeamMembers:  teamMembers,
		Cycles:       cycleList,
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

// buildCalendarGrid creates a 6x7 grid of days for the calendar view.
func buildCalendarGrid(year int, month time.Month, allIssues []*issues.Issue, columnMap map[string]*columns.Column, projectMap map[string]*projects.Project) [][]CalendarDay {
	// Get first day of month and calculate grid start (Monday-based week)
	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	weekday := int(firstOfMonth.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday
	}
	gridStart := firstOfMonth.AddDate(0, 0, -(weekday - 1))

	// Index issues by due date for quick lookup
	issuesByDate := make(map[string][]*CalendarIssue)
	now := time.Now()
	for _, issue := range allIssues {
		if issue.DueDate != nil {
			key := issue.DueDate.Format("2006-01-02")
			daysUntil := int(issue.DueDate.Sub(now).Hours() / 24)
			issuesByDate[key] = append(issuesByDate[key], &CalendarIssue{
				Issue:        issue,
				Project:      projectMap[issue.ProjectID],
				Column:       columnMap[issue.ColumnID],
				DaysUntilDue: daysUntil,
				IsOverdue:    issue.DueDate.Before(now),
			})
		}
	}

	// Build 6 weeks of days
	weeks := make([][]CalendarDay, 6)
	today := time.Now().Truncate(24 * time.Hour)

	for w := 0; w < 6; w++ {
		weeks[w] = make([]CalendarDay, 7)
		for d := 0; d < 7; d++ {
			date := gridStart.AddDate(0, 0, w*7+d)
			dateKey := date.Format("2006-01-02")

			weeks[w][d] = CalendarDay{
				Date:         date,
				Issues:       issuesByDate[dateKey],
				IsToday:      date.Year() == today.Year() && date.YearDay() == today.YearDay(),
				IsWeekend:    d >= 5,
				IsOtherMonth: date.Month() != month,
			}
		}
	}

	return weeks
}

// buildGanttIssues creates GanttIssue wrappers with position calculations.
func buildGanttIssues(allIssues []*issues.Issue, columnMap map[string]*columns.Column, projectMap map[string]*projects.Project, timelineStart, timelineEnd time.Time) []*GanttIssue {
	totalDays := timelineEnd.Sub(timelineStart).Hours() / 24
	result := make([]*GanttIssue, 0, len(allIssues))

	for i, issue := range allIssues {
		gi := &GanttIssue{
			Issue:   issue,
			Project: projectMap[issue.ProjectID],
			Column:  columnMap[issue.ColumnID],
			Row:     i,
		}

		// Determine effective dates
		if issue.StartDate != nil {
			gi.EffectiveStart = *issue.StartDate
			gi.HasExplicitDates = true
		} else {
			gi.EffectiveStart = issue.CreatedAt
		}

		if issue.EndDate != nil {
			gi.EffectiveEnd = *issue.EndDate
			gi.HasExplicitDates = gi.HasExplicitDates && true
		} else if issue.DueDate != nil {
			gi.EffectiveEnd = *issue.DueDate
		} else {
			// Default to 7 days after start
			gi.EffectiveEnd = gi.EffectiveStart.AddDate(0, 0, 7)
		}

		// Calculate position
		gi.LeftOffset, gi.Width = calculateGanttPosition(gi.EffectiveStart, gi.EffectiveEnd, timelineStart, timelineEnd, totalDays)

		result = append(result, gi)
	}

	return result
}

// calculateGanttPosition computes the left offset and width for a Gantt bar.
func calculateGanttPosition(start, end, timelineStart, timelineEnd time.Time, totalDays float64) (leftOffset, width float64) {
	startDays := start.Sub(timelineStart).Hours() / 24
	endDays := end.Sub(timelineStart).Hours() / 24

	leftOffset = (startDays / totalDays) * 100
	width = ((endDays - startDays) / totalDays) * 100

	// Clamp to visible range
	if leftOffset < 0 {
		width += leftOffset
		leftOffset = 0
	}
	if leftOffset+width > 100 {
		width = 100 - leftOffset
	}

	// Minimum 1% width for visibility
	if width < 1 {
		width = 1
	}

	return leftOffset, width
}

// buildGanttHeaderDates creates date markers for the Gantt timeline header.
func buildGanttHeaderDates(timelineStart, timelineEnd time.Time, scale string) []GanttHeaderDate {
	totalDays := timelineEnd.Sub(timelineStart).Hours() / 24
	var dates []GanttHeaderDate
	today := time.Now().Truncate(24 * time.Hour)

	current := timelineStart
	for current.Before(timelineEnd) {
		offset := (current.Sub(timelineStart).Hours() / 24 / totalDays) * 100

		var label string
		switch scale {
		case "day":
			label = current.Format("Mon 2")
		case "week":
			_, week := current.ISOWeek()
			label = fmt.Sprintf("W%d", week)
		case "month":
			label = current.Format("Jan")
		default:
			label = current.Format("Mon 2")
		}

		dates = append(dates, GanttHeaderDate{
			Date:      current,
			Label:     label,
			Offset:    offset,
			IsToday:   current.Year() == today.Year() && current.YearDay() == today.YearDay(),
			IsWeekend: current.Weekday() == time.Saturday || current.Weekday() == time.Sunday,
		})

		// Move to next marker based on scale
		switch scale {
		case "day":
			current = current.AddDate(0, 0, 1)
		case "week":
			current = current.AddDate(0, 0, 7)
		case "month":
			current = current.AddDate(0, 1, 0)
		default:
			current = current.AddDate(0, 0, 1)
		}
	}

	return dates
}

// groupGanttIssues groups issues by the specified field.
func groupGanttIssues(ganttIssues []*GanttIssue, groupBy string, projectMap map[string]*projects.Project, columnMap map[string]*columns.Column) []*GanttGroup {
	if groupBy == "none" || groupBy == "" {
		return []*GanttGroup{{
			ID:     "",
			Name:   "",
			Issues: ganttIssues,
		}}
	}

	groupsMap := make(map[string]*GanttGroup)
	var order []string

	for _, gi := range ganttIssues {
		var groupID, groupName string

		switch groupBy {
		case "project":
			if gi.Project != nil {
				groupID = gi.Project.ID
				groupName = gi.Project.Name
			}
		case "column":
			if gi.Column != nil {
				groupID = gi.Column.ID
				groupName = gi.Column.Name
			}
		}

		if groupID == "" {
			groupID = "ungrouped"
			groupName = "Ungrouped"
		}

		if _, exists := groupsMap[groupID]; !exists {
			groupsMap[groupID] = &GanttGroup{
				ID:     groupID,
				Name:   groupName,
				Issues: []*GanttIssue{},
			}
			order = append(order, groupID)
		}
		groupsMap[groupID].Issues = append(groupsMap[groupID].Issues, gi)
	}

	result := make([]*GanttGroup, 0, len(order))
	for _, id := range order {
		result = append(result, groupsMap[id])
	}

	return result
}
