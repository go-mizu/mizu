package handler

import (
	"bytes"
	"html/template"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/kanban/assets"
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

var _ = template.Template{} // Silence unused import

// loadTemplates loads all templates for testing using the assets package
func loadTemplates(t *testing.T) map[string]*template.Template {
	t.Helper()

	templates, err := assets.Templates()
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	return templates
}

// TestLoginTemplate tests the login template renders without errors
func TestLoginTemplate(t *testing.T) {
	templates := loadTemplates(t)

	data := LoginData{
		Title:    "Sign In",
		Subtitle: "Sign in to your account",
	}

	var buf bytes.Buffer
	err := templates["login"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Login template error: %v", err)
	}
}

// TestRegisterTemplate tests the register template renders without errors
func TestRegisterTemplate(t *testing.T) {
	templates := loadTemplates(t)

	data := RegisterData{
		Title:    "Create Account",
		Subtitle: "Create a new account",
	}

	var buf bytes.Buffer
	err := templates["register"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Register template error: %v", err)
	}
}

// TestHomeTemplate tests the home template renders without errors
func TestHomeTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:          "user-1",
		DisplayName: "Test User",
		Email:       "test@example.com",
	}

	workspace := &workspaces.Workspace{
		ID:   "ws-1",
		Name: "Test Workspace",
		Slug: "test-workspace",
	}

	data := HomeData{
		Title:         "Dashboard",
		User:          user,
		Workspace:     workspace,
		Workspaces:    []*workspaces.Workspace{workspace},
		Teams:         []*teams.Team{},
		Projects:      []*projects.Project{},
		Stats:         Stats{OpenIssues: 0, InProgress: 0, Completed: 0},
		DefaultTeamID: "",
		ActiveNav:     "home",
		Breadcrumbs:   []Breadcrumb{},
	}

	var buf bytes.Buffer
	err := templates["home"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Home template error: %v", err)
	}
}

// TestBoardTemplate tests the board template renders without errors
func TestBoardTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:          "user-1",
		DisplayName: "Test User",
	}

	workspace := &workspaces.Workspace{
		ID:   "ws-1",
		Name: "Test Workspace",
		Slug: "test-workspace",
	}

	project := &projects.Project{
		ID:   "proj-1",
		Name: "Test Project",
		Key:  "TEST",
	}

	col := &columns.Column{
		ID:   "col-1",
		Name: "To Do",
	}

	issue := &issues.Issue{
		ID:       "issue-1",
		Key:      "TEST-1",
		Title:    "Test Issue",
		ColumnID: "col-1",
	}

	boardColumns := []*BoardColumn{
		{
			Column: col,
			Issues: []*issues.Issue{issue},
		},
	}

	data := BoardData{
		Title:           "Test Project",
		User:            user,
		Workspace:       workspace,
		Workspaces:      []*workspaces.Workspace{workspace},
		Teams:           []*teams.Team{},
		Projects:        []*projects.Project{project},
		Project:         project,
		Columns:         boardColumns,
		ActiveProjectID: "proj-1",
		ActiveNav:       "issues",
		Breadcrumbs:     []Breadcrumb{{Label: "Test Project", URL: ""}},
	}

	var buf bytes.Buffer
	err := templates["board"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Board template error: %v", err)
	}
}

// TestIssuesTemplate tests the issues list template renders without errors
func TestIssuesTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:          "user-1",
		DisplayName: "Test User",
	}

	workspace := &workspaces.Workspace{
		ID:   "ws-1",
		Name: "Test Workspace",
		Slug: "test-workspace",
	}

	col := &columns.Column{
		ID:   "col-1",
		Name: "To Do",
	}

	issue := &issues.Issue{
		ID:        "issue-1",
		Key:       "TEST-1",
		Title:     "Test Issue",
		ColumnID:  "col-1",
		CreatorID: "user-1",
		UpdatedAt: time.Now(),
	}

	issueViews := []*IssueView{
		{
			Issue:    issue,
			Column:   col,
			Priority: "",
		},
	}

	project := &projects.Project{
		ID:   "proj-1",
		Name: "Test Project",
		Key:  "TEST",
	}

	data := IssuesData{
		Title:            "Issues",
		User:             user,
		Workspace:        workspace,
		Workspaces:       []*workspaces.Workspace{workspace},
		Teams:            []*teams.Team{},
		Issues:           issueViews,
		Columns:          []*columns.Column{col},
		Projects:         []*projects.Project{project},
		DefaultProjectID: "proj-1",
		ActiveProjectID:  "",
		TotalCount:       1,
		ActiveNav:        "issues",
		Breadcrumbs:      []Breadcrumb{},
	}

	var buf bytes.Buffer
	err := templates["issues"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Issues template error: %v", err)
	}
}

// TestIssueDetailTemplate tests the issue detail template renders without errors
func TestIssueDetailTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:          "user-1",
		DisplayName: "Test User",
	}

	workspace := &workspaces.Workspace{
		ID:   "ws-1",
		Name: "Test Workspace",
		Slug: "test-workspace",
	}

	project := &projects.Project{
		ID:   "proj-1",
		Name: "Test Project",
		Key:  "TEST",
	}

	col := &columns.Column{
		ID:   "col-1",
		Name: "To Do",
	}

	issue := &issues.Issue{
		ID:        "issue-1",
		Key:       "TEST-1",
		Title:     "Test Issue",
		ColumnID:  "col-1",
		CreatorID: "user-1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	data := IssueData{
		Title:           "TEST-1",
		User:            user,
		Workspace:       workspace,
		Workspaces:      []*workspaces.Workspace{workspace},
		Teams:           []*teams.Team{},
		Projects:        []*projects.Project{project},
		Issue:           issue,
		Project:         project,
		Columns:         []*columns.Column{col},
		Comments:        []*comments.Comment{},
		Cycles:          []*cycles.Cycle{},
		Fields:          []*fields.Field{},
		TeamMembers:     []*users.User{},
		ActiveProjectID: "proj-1",
		ActiveNav:       "issues",
		Breadcrumbs:     []Breadcrumb{{Label: "Issues", URL: "/issues"}, {Label: "TEST-1", URL: ""}},
	}

	var buf bytes.Buffer
	err := templates["issue"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Issue detail template error: %v", err)
	}
}

// TestCyclesTemplate tests the cycles template renders without errors
func TestCyclesTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:          "user-1",
		DisplayName: "Test User",
	}

	workspace := &workspaces.Workspace{
		ID:   "ws-1",
		Name: "Test Workspace",
		Slug: "test-workspace",
	}

	cycle := &cycles.Cycle{
		ID:        "cycle-1",
		Name:      "Sprint 1",
		Status:    "planning",
		StartDate: time.Now(),
		EndDate:   time.Now().Add(14 * 24 * time.Hour),
	}

	data := CyclesData{
		Title:         "Cycles",
		User:          user,
		Workspace:     workspace,
		Workspaces:    []*workspaces.Workspace{workspace},
		Teams:         []*teams.Team{},
		Projects:      []*projects.Project{},
		Cycles:        []*cycles.Cycle{cycle},
		DefaultTeamID: "team-1",
		ActiveNav:     "cycles",
		Breadcrumbs:   []Breadcrumb{},
	}

	var buf bytes.Buffer
	err := templates["cycles"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Cycles template error: %v", err)
	}
}

// TestCyclesEmptyTemplate tests the cycles template with no cycles
func TestCyclesEmptyTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:          "user-1",
		DisplayName: "Test User",
	}

	workspace := &workspaces.Workspace{
		ID:   "ws-1",
		Name: "Test Workspace",
		Slug: "test-workspace",
	}

	data := CyclesData{
		Title:         "Cycles",
		User:          user,
		Workspace:     workspace,
		Workspaces:    []*workspaces.Workspace{workspace},
		Teams:         []*teams.Team{},
		Projects:      []*projects.Project{},
		Cycles:        []*cycles.Cycle{},
		DefaultTeamID: "team-1",
		ActiveNav:     "cycles",
		Breadcrumbs:   []Breadcrumb{},
	}

	var buf bytes.Buffer
	err := templates["cycles"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Cycles empty template error: %v", err)
	}
}

// TestIssuesEmptyTemplate tests the issues template with no issues
func TestIssuesEmptyTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:          "user-1",
		DisplayName: "Test User",
	}

	workspace := &workspaces.Workspace{
		ID:   "ws-1",
		Name: "Test Workspace",
		Slug: "test-workspace",
	}

	data := IssuesData{
		Title:            "Issues",
		User:             user,
		Workspace:        workspace,
		Workspaces:       []*workspaces.Workspace{workspace},
		Teams:            []*teams.Team{},
		Issues:           []*IssueView{},
		Columns:          []*columns.Column{},
		Projects:         []*projects.Project{},
		DefaultProjectID: "",
		ActiveProjectID:  "",
		TotalCount:       0,
		ActiveNav:        "issues",
		Breadcrumbs:      []Breadcrumb{},
	}

	var buf bytes.Buffer
	err := templates["issues"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Issues empty template error: %v", err)
	}
}

// TestTeamTemplate tests the team template renders without errors
func TestTeamTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:          "user-1",
		DisplayName: "Test User",
		Email:       "test@example.com",
	}

	workspace := &workspaces.Workspace{
		ID:   "ws-1",
		Name: "Test Workspace",
		Slug: "test-workspace",
	}

	team := &teams.Team{
		ID:   "team-1",
		Name: "Engineering",
	}

	memberUser := &users.User{
		ID:          "user-2",
		DisplayName: "Team Member",
		Email:       "member@example.com",
	}

	member := &teams.Member{
		TeamID:   "team-1",
		UserID:   "user-2",
		Role:     "member",
		JoinedAt: time.Now(),
	}

	memberViews := []*MemberView{
		{
			Member: member,
			User:   memberUser,
		},
	}

	project := &projects.Project{
		ID:   "proj-1",
		Name: "Test Project",
		Key:  "TEST",
	}

	data := TeamData{
		Title:           team.Name,
		User:            user,
		Workspace:       workspace,
		Workspaces:      []*workspaces.Workspace{workspace},
		Teams:           []*teams.Team{team},
		Team:            team,
		Members:         memberViews,
		Projects:        []*projects.Project{project},
		ActiveTeamID:    "team-1",
		ActiveProjectID: "",
		ActiveNav:       "teams",
		Breadcrumbs:     []Breadcrumb{{Label: "Teams", URL: "/teams"}, {Label: team.Name, URL: ""}},
	}

	var buf bytes.Buffer
	err := templates["team"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Team template error: %v", err)
	}
}

// TestTeamEmptyTemplate tests the team template with no members
func TestTeamEmptyTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:          "user-1",
		DisplayName: "Test User",
	}

	workspace := &workspaces.Workspace{
		ID:   "ws-1",
		Name: "Test Workspace",
		Slug: "test-workspace",
	}

	team := &teams.Team{
		ID:   "team-1",
		Name: "Engineering",
	}

	data := TeamData{
		Title:           team.Name,
		User:            user,
		Workspace:       workspace,
		Workspaces:      []*workspaces.Workspace{workspace},
		Teams:           []*teams.Team{team},
		Team:            team,
		Members:         []*MemberView{},
		Projects:        []*projects.Project{},
		ActiveTeamID:    "team-1",
		ActiveProjectID: "",
		ActiveNav:       "teams",
		Breadcrumbs:     []Breadcrumb{{Label: "Teams", URL: "/teams"}, {Label: team.Name, URL: ""}},
	}

	var buf bytes.Buffer
	err := templates["team"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Team empty template error: %v", err)
	}
}
