package handler

import (
	"html/template"
	"net/http"
	"sort"
	"time"

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

// TrelloLoginData holds data for the Trello login page.
type TrelloLoginData struct {
	Title string
	Error string
}

// TrelloRegisterData holds data for the Trello register page.
type TrelloRegisterData struct {
	Title string
	Error string
}

// TrelloBoardsData holds data for the boards list page.
type TrelloBoardsData struct {
	Title      string
	User       *users.User
	Workspace  *workspaces.Workspace
	Workspaces []*workspaces.Workspace
	Starred    []*projects.Project
	Recent     []*projects.Project
	All        []*projects.Project
	Teams      []*teams.Team
}

// TrelloList wraps a column with its cards for the board view.
type TrelloList struct {
	*columns.Column
	Cards []*TrelloCard
}

// TrelloCard wraps an issue with Trello-specific display data.
type TrelloCard struct {
	*issues.Issue
	Labels       []*TrelloLabel
	Members      []*users.User
	HasDueDate   bool
	IsOverdue    bool
	IsDueSoon    bool
	CommentCount int
}

// TrelloLabel represents a color label on a card.
type TrelloLabel struct {
	ID    string
	Name  string
	Color string
}

// TrelloBoardData holds data for the board view.
type TrelloBoardData struct {
	Title      string
	User       *users.User
	Workspace  *workspaces.Workspace
	Workspaces []*workspaces.Workspace
	Board      *projects.Project
	Lists      []*TrelloList
	Members    []*users.User
	Labels     []*TrelloLabel
	Team       *teams.Team
	AllTeams   []*teams.Team
	Projects   []*projects.Project
}

// TrelloCardData holds data for the card detail view.
type TrelloCardData struct {
	Title       string
	User        *users.User
	Workspace   *workspaces.Workspace
	Workspaces  []*workspaces.Workspace
	Board       *projects.Project
	Card        *issues.Issue
	List        *columns.Column
	Lists       []*columns.Column
	Labels      []*TrelloLabel
	AllLabels   []*TrelloLabel
	Members     []*users.User
	AllMembers  []*users.User
	Comments    []*comments.Comment
	Cycles      []*cycles.Cycle
	Fields      []*fields.Field
	CommentList []*TrelloComment
}

// TrelloComment wraps a comment with user data.
type TrelloComment struct {
	*comments.Comment
	User *users.User
}

// PageTrello handles Trello-themed page rendering.
type PageTrello struct {
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

// NewPageTrello creates a new PageTrello handler.
func NewPageTrello(
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
) *PageTrello {
	return &PageTrello{
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

func (h *PageTrello) render(c *mizu.Ctx, name string, data any) error {
	tmpl, ok := h.templates[name]
	if !ok {
		return c.Text(http.StatusInternalServerError, "Template not found: "+name)
	}

	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.Execute(c.Writer(), data)
}

// Login renders the Trello-style login page.
func (h *PageTrello) Login(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID != "" {
		http.Redirect(c.Writer(), c.Request(), "/t/", http.StatusFound)
		return nil
	}
	return h.render(c, "trello-login", TrelloLoginData{
		Title: "Log in to Kanban",
	})
}

// Register renders the Trello-style registration page.
func (h *PageTrello) Register(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID != "" {
		http.Redirect(c.Writer(), c.Request(), "/t/", http.StatusFound)
		return nil
	}
	return h.render(c, "trello-register", TrelloRegisterData{
		Title: "Create a Kanban Account",
	})
}

// Home redirects to the first workspace's boards.
func (h *PageTrello) Home(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/t/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	workspaceList, _ := h.workspaces.ListByUser(ctx, userID)
	if len(workspaceList) > 0 {
		http.Redirect(c.Writer(), c.Request(), "/t/"+workspaceList[0].Slug, http.StatusFound)
		return nil
	}

	http.Redirect(c.Writer(), c.Request(), "/t/login", http.StatusFound)
	return nil
}

// Boards renders the boards list for a workspace.
func (h *PageTrello) Boards(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/t/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	workspaceSlug := c.Param("workspace")

	user, _ := h.users.GetByID(ctx, userID)
	workspace, err := h.workspaces.GetBySlug(ctx, workspaceSlug)
	if err != nil {
		http.Redirect(c.Writer(), c.Request(), "/t/", http.StatusFound)
		return nil
	}

	workspaceList, _ := h.workspaces.ListByUser(ctx, userID)
	teamList, _ := h.teams.ListByWorkspace(ctx, workspace.ID)

	// Collect all projects from all teams
	var allProjects []*projects.Project
	for _, team := range teamList {
		projectList, _ := h.projects.ListByTeam(ctx, team.ID)
		allProjects = append(allProjects, projectList...)
	}

	// Sort by issue count (most active first) for recent
	sort.Slice(allProjects, func(i, j int) bool {
		return allProjects[i].IssueCount > allProjects[j].IssueCount
	})

	// Get recent (top 4 by activity)
	var recent []*projects.Project
	if len(allProjects) > 4 {
		recent = allProjects[:4]
	} else {
		recent = allProjects
	}

	return h.render(c, "trello-boards", TrelloBoardsData{
		Title:      workspace.Name + " | Boards",
		User:       user,
		Workspace:  workspace,
		Workspaces: workspaceList,
		Recent:     recent,
		All:        allProjects,
		Teams:      teamList,
	})
}

// Board renders the Trello-style kanban board.
func (h *PageTrello) Board(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/t/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	workspaceSlug := c.Param("workspace")
	boardID := c.Param("boardID")

	user, _ := h.users.GetByID(ctx, userID)
	workspace, err := h.workspaces.GetBySlug(ctx, workspaceSlug)
	if err != nil {
		http.Redirect(c.Writer(), c.Request(), "/t/", http.StatusFound)
		return nil
	}

	board, err := h.projects.GetByID(ctx, boardID)
	if err != nil {
		http.Redirect(c.Writer(), c.Request(), "/t/"+workspaceSlug, http.StatusFound)
		return nil
	}

	workspaceList, _ := h.workspaces.ListByUser(ctx, userID)
	teamList, _ := h.teams.ListByWorkspace(ctx, workspace.ID)
	projectList, _ := h.projects.ListByTeam(ctx, board.TeamID)

	// Get the team for this board
	team, _ := h.teams.GetByID(ctx, board.TeamID)

	// Get columns and issues
	columnList, _ := h.columns.ListByProject(ctx, boardID)
	allIssues, _ := h.issues.ListByProject(ctx, boardID)

	// Get team members
	var teamMembers []*users.User
	if team != nil {
		members, _ := h.teams.ListMembers(ctx, team.ID)
		if len(members) > 0 {
			userIDs := make([]string, len(members))
			for i, m := range members {
				userIDs[i] = m.UserID
			}
			teamMembers, _ = h.users.GetByIDs(ctx, userIDs)
		}
	}

	// Group issues by column
	issuesByColumn := make(map[string][]*issues.Issue)
	for _, issue := range allIssues {
		issuesByColumn[issue.ColumnID] = append(issuesByColumn[issue.ColumnID], issue)
	}

	// Build TrelloLists
	now := time.Now()
	lists := make([]*TrelloList, len(columnList))
	for i, col := range columnList {
		cards := make([]*TrelloCard, 0, len(issuesByColumn[col.ID]))
		for _, issue := range issuesByColumn[col.ID] {
			card := &TrelloCard{
				Issue:      issue,
				Labels:     [](*TrelloLabel){},
				Members:    [](*users.User){},
				HasDueDate: issue.DueDate != nil,
			}
			if issue.DueDate != nil {
				card.IsOverdue = issue.DueDate.Before(now)
				card.IsDueSoon = !card.IsOverdue && issue.DueDate.Before(now.Add(24*time.Hour))
			}
			cards = append(cards, card)
		}
		lists[i] = &TrelloList{
			Column: col,
			Cards:  cards,
		}
	}

	// Default labels
	defaultLabels := []*TrelloLabel{
		{ID: "green", Name: "", Color: "#61bd4f"},
		{ID: "yellow", Name: "", Color: "#f2d600"},
		{ID: "orange", Name: "", Color: "#ff9f1a"},
		{ID: "red", Name: "", Color: "#eb5a46"},
		{ID: "purple", Name: "", Color: "#c377e0"},
		{ID: "blue", Name: "", Color: "#0079bf"},
	}

	return h.render(c, "trello-board", TrelloBoardData{
		Title:      board.Name + " | Kanban",
		User:       user,
		Workspace:  workspace,
		Workspaces: workspaceList,
		Board:      board,
		Lists:      lists,
		Members:    teamMembers,
		Labels:     defaultLabels,
		Team:       team,
		AllTeams:   teamList,
		Projects:   projectList,
	})
}

// Card renders the card detail view.
func (h *PageTrello) Card(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/t/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	workspaceSlug := c.Param("workspace")
	cardKey := c.Param("cardKey")

	user, _ := h.users.GetByID(ctx, userID)
	workspace, err := h.workspaces.GetBySlug(ctx, workspaceSlug)
	if err != nil {
		http.Redirect(c.Writer(), c.Request(), "/t/", http.StatusFound)
		return nil
	}

	card, err := h.issues.GetByKey(ctx, cardKey)
	if err != nil {
		http.Redirect(c.Writer(), c.Request(), "/t/"+workspaceSlug, http.StatusFound)
		return nil
	}

	workspaceList, _ := h.workspaces.ListByUser(ctx, userID)
	board, _ := h.projects.GetByID(ctx, card.ProjectID)
	columnList, _ := h.columns.ListByProject(ctx, card.ProjectID)
	commentList, _ := h.comments.ListByIssue(ctx, card.ID)
	fieldList, _ := h.fields.ListByProject(ctx, card.ProjectID)

	// Find the current list
	var currentList *columns.Column
	for _, col := range columnList {
		if col.ID == card.ColumnID {
			currentList = col
			break
		}
	}

	// Get team members
	var teamMembers []*users.User
	if board != nil {
		members, _ := h.teams.ListMembers(ctx, board.TeamID)
		if len(members) > 0 {
			userIDs := make([]string, len(members))
			for i, m := range members {
				userIDs[i] = m.UserID
			}
			teamMembers, _ = h.users.GetByIDs(ctx, userIDs)
		}
	}

	// Build user map for comments
	userMap := make(map[string]*users.User)
	for _, u := range teamMembers {
		userMap[u.ID] = u
	}

	// Build TrelloComments
	trelloComments := make([]*TrelloComment, len(commentList))
	for i, cmt := range commentList {
		trelloComments[i] = &TrelloComment{
			Comment: cmt,
			User:    userMap[cmt.AuthorID],
		}
	}

	// Get cycles for this team
	var cycleList []*cycles.Cycle
	if board != nil {
		cycleList, _ = h.cycles.ListByTeam(ctx, board.TeamID)
	}

	// Default labels
	defaultLabels := []*TrelloLabel{
		{ID: "green", Name: "Green", Color: "#61bd4f"},
		{ID: "yellow", Name: "Yellow", Color: "#f2d600"},
		{ID: "orange", Name: "Orange", Color: "#ff9f1a"},
		{ID: "red", Name: "Red", Color: "#eb5a46"},
		{ID: "purple", Name: "Purple", Color: "#c377e0"},
		{ID: "blue", Name: "Blue", Color: "#0079bf"},
	}

	return h.render(c, "trello-card", TrelloCardData{
		Title:       card.Key + " | " + card.Title,
		User:        user,
		Workspace:   workspace,
		Workspaces:  workspaceList,
		Board:       board,
		Card:        card,
		List:        currentList,
		Lists:       columnList,
		AllLabels:   defaultLabels,
		AllMembers:  teamMembers,
		Comments:    commentList,
		Cycles:      cycleList,
		Fields:      fieldList,
		CommentList: trelloComments,
	})
}
