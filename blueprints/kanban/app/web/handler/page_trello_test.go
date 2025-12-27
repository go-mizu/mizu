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

// loadTrelloTemplates loads all Trello templates for testing using the assets package
func loadTrelloTemplates(t *testing.T) map[string]*template.Template {
	t.Helper()

	templates, err := assets.TemplatesForTheme("trello")
	if err != nil {
		t.Fatalf("Failed to load Trello templates: %v", err)
	}

	return templates
}

// TestTrelloLoginTemplate tests the Trello login template renders without errors
func TestTrelloLoginTemplate(t *testing.T) {
	templates := loadTrelloTemplates(t)

	data := TrelloLoginData{
		Title: "Log in to Trello",
	}

	var buf bytes.Buffer
	err := templates["trello-login"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Trello login template error: %v", err)
	}

	// Check that the output contains expected content
	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("Log in")) {
		t.Errorf("Expected 'Log in' text in output, got: %s", output)
	}
}

// TestTrelloLoginTemplateWithError tests the Trello login template with an error message
func TestTrelloLoginTemplateWithError(t *testing.T) {
	templates := loadTrelloTemplates(t)

	data := TrelloLoginData{
		Title: "Log in to Trello",
		Error: "Invalid email or password",
	}

	var buf bytes.Buffer
	err := templates["trello-login"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Trello login template with error error: %v", err)
	}
}

// TestTrelloRegisterTemplate tests the Trello register template renders without errors
func TestTrelloRegisterTemplate(t *testing.T) {
	templates := loadTrelloTemplates(t)

	data := TrelloRegisterData{
		Title: "Create a Trello Account",
	}

	var buf bytes.Buffer
	err := templates["trello-register"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Trello register template error: %v", err)
	}
}

// TestTrelloBoardsTemplate tests the Trello boards list template renders without errors
func TestTrelloBoardsTemplate(t *testing.T) {
	templates := loadTrelloTemplates(t)

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

	project := &projects.Project{
		ID:   "proj-1",
		Name: "Website Redesign",
		Key:  "WEB",
	}

	project2 := &projects.Project{
		ID:   "proj-2",
		Name: "Mobile App",
		Key:  "MOB",
	}

	data := TrelloBoardsData{
		Title:      "Test Workspace | Boards",
		User:       user,
		Workspace:  workspace,
		Workspaces: []*workspaces.Workspace{workspace},
		Starred:    []*projects.Project{},
		Recent:     []*projects.Project{project, project2},
		All:        []*projects.Project{project, project2},
		Teams:      []*teams.Team{team},
	}

	var buf bytes.Buffer
	err := templates["trello-boards"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Trello boards template error: %v", err)
	}
}

// TestTrelloBoardsEmptyTemplate tests the Trello boards template with no boards
func TestTrelloBoardsEmptyTemplate(t *testing.T) {
	templates := loadTrelloTemplates(t)

	user := &users.User{
		ID:          "user-1",
		DisplayName: "Test User",
		Email:       "test@example.com",
	}

	workspace := &workspaces.Workspace{
		ID:   "ws-1",
		Name: "New Workspace",
		Slug: "new-workspace",
	}

	team := &teams.Team{
		ID:   "team-1",
		Name: "Engineering",
	}

	data := TrelloBoardsData{
		Title:      "New Workspace | Boards",
		User:       user,
		Workspace:  workspace,
		Workspaces: []*workspaces.Workspace{workspace},
		Starred:    []*projects.Project{},
		Recent:     []*projects.Project{},
		All:        []*projects.Project{},
		Teams:      []*teams.Team{team},
	}

	var buf bytes.Buffer
	err := templates["trello-boards"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Trello boards empty template error: %v", err)
	}
}

// TestTrelloBoardTemplate tests the Trello board (kanban) template renders without errors
func TestTrelloBoardTemplate(t *testing.T) {
	templates := loadTrelloTemplates(t)

	user := &users.User{
		ID:          "user-1",
		DisplayName: "John Doe",
		Email:       "john@example.com",
	}

	user2 := &users.User{
		ID:          "user-2",
		DisplayName: "Jane Smith",
		Email:       "jane@example.com",
	}

	workspace := &workspaces.Workspace{
		ID:   "ws-1",
		Name: "Acme Corp",
		Slug: "acme-corp",
	}

	team := &teams.Team{
		ID:   "team-1",
		Name: "Engineering",
	}

	project := &projects.Project{
		ID:     "proj-1",
		TeamID: "team-1",
		Name:   "Website Redesign",
		Key:    "WEB",
	}

	// Create lists with cards
	col1 := &columns.Column{
		ID:        "col-1",
		ProjectID: "proj-1",
		Name:      "To Do",
		Position:  0,
	}

	col2 := &columns.Column{
		ID:        "col-2",
		ProjectID: "proj-1",
		Name:      "In Progress",
		Position:  1,
	}

	col3 := &columns.Column{
		ID:        "col-3",
		ProjectID: "proj-1",
		Name:      "Done",
		Position:  2,
	}

	dueDate := time.Now().Add(2 * 24 * time.Hour)
	overdueDueDate := time.Now().Add(-24 * time.Hour)

	card1 := &TrelloCard{
		Issue: &issues.Issue{
			ID:        "issue-1",
			ProjectID: "proj-1",
			ColumnID:  "col-1",
			Key:       "WEB-1",
			Title:     "Design new homepage",
			Priority:  2,
		},
		Labels: []*TrelloLabel{
			{ID: "green", Name: "Design", Color: "#61bd4f"},
			{ID: "blue", Name: "Frontend", Color: "#0079bf"},
		},
		Members:      []*users.User{user},
		HasDueDate:   true,
		IsDueSoon:    true,
		CommentCount: 3,
	}
	card1.Issue.DueDate = &dueDate

	card2 := &TrelloCard{
		Issue: &issues.Issue{
			ID:        "issue-2",
			ProjectID: "proj-1",
			ColumnID:  "col-1",
			Key:       "WEB-2",
			Title:     "Implement responsive navigation",
		},
		Labels:  []*TrelloLabel{},
		Members: []*users.User{},
	}

	card3 := &TrelloCard{
		Issue: &issues.Issue{
			ID:        "issue-3",
			ProjectID: "proj-1",
			ColumnID:  "col-2",
			Key:       "WEB-3",
			Title:     "Setup CI/CD pipeline",
			Priority:  1,
		},
		Labels: []*TrelloLabel{
			{ID: "red", Name: "Urgent", Color: "#eb5a46"},
		},
		Members:      []*users.User{user2},
		HasDueDate:   true,
		IsOverdue:    true,
		CommentCount: 1,
	}
	card3.Issue.DueDate = &overdueDueDate

	card4 := &TrelloCard{
		Issue: &issues.Issue{
			ID:        "issue-4",
			ProjectID: "proj-1",
			ColumnID:  "col-3",
			Key:       "WEB-4",
			Title:     "Create project repository",
		},
		Labels:  []*TrelloLabel{},
		Members: []*users.User{user, user2},
	}

	lists := []*TrelloList{
		{Column: col1, Cards: []*TrelloCard{card1, card2}},
		{Column: col2, Cards: []*TrelloCard{card3}},
		{Column: col3, Cards: []*TrelloCard{card4}},
	}

	labels := []*TrelloLabel{
		{ID: "green", Name: "", Color: "#61bd4f"},
		{ID: "yellow", Name: "", Color: "#f2d600"},
		{ID: "orange", Name: "", Color: "#ff9f1a"},
		{ID: "red", Name: "", Color: "#eb5a46"},
		{ID: "purple", Name: "", Color: "#c377e0"},
		{ID: "blue", Name: "", Color: "#0079bf"},
	}

	data := TrelloBoardData{
		Title:      "Website Redesign | Trello",
		User:       user,
		Workspace:  workspace,
		Workspaces: []*workspaces.Workspace{workspace},
		Board:      project,
		Lists:      lists,
		Members:    []*users.User{user, user2},
		Labels:     labels,
		Team:       team,
		AllTeams:   []*teams.Team{team},
		Projects:   []*projects.Project{project},
	}

	var buf bytes.Buffer
	err := templates["trello-board"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Trello board template error: %v", err)
	}
}

// TestTrelloBoardEmptyTemplate tests the Trello board template with no lists or cards
func TestTrelloBoardEmptyTemplate(t *testing.T) {
	templates := loadTrelloTemplates(t)

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
		ID:     "proj-1",
		TeamID: "team-1",
		Name:   "New Project",
		Key:    "NEW",
	}

	data := TrelloBoardData{
		Title:      "New Project | Trello",
		User:       user,
		Workspace:  workspace,
		Workspaces: []*workspaces.Workspace{workspace},
		Board:      project,
		Lists:      []*TrelloList{},
		Members:    []*users.User{user},
		Labels:     []*TrelloLabel{},
		AllTeams:   []*teams.Team{},
		Projects:   []*projects.Project{project},
	}

	var buf bytes.Buffer
	err := templates["trello-board"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Trello board empty template error: %v", err)
	}
}

// TestTrelloCardTemplate tests the Trello card detail template renders without errors
func TestTrelloCardTemplate(t *testing.T) {
	templates := loadTrelloTemplates(t)

	user := &users.User{
		ID:          "user-1",
		DisplayName: "John Doe",
		Email:       "john@example.com",
	}

	user2 := &users.User{
		ID:          "user-2",
		DisplayName: "Jane Smith",
		Email:       "jane@example.com",
	}

	workspace := &workspaces.Workspace{
		ID:   "ws-1",
		Name: "Acme Corp",
		Slug: "acme-corp",
	}

	project := &projects.Project{
		ID:     "proj-1",
		TeamID: "team-1",
		Name:   "Website Redesign",
		Key:    "WEB",
	}

	col := &columns.Column{
		ID:        "col-1",
		ProjectID: "proj-1",
		Name:      "In Progress",
		Position:  1,
	}

	dueDate := time.Now().Add(5 * 24 * time.Hour)
	card := &issues.Issue{
		ID:          "issue-1",
		ProjectID:   "proj-1",
		ColumnID:    "col-1",
		Key:         "WEB-1",
		Title:       "Design new homepage",
		Description: "We need to redesign the homepage to be more modern and user-friendly.\n\n## Requirements\n- Mobile responsive\n- Fast loading\n- Accessible",
		Priority:    2,
		CreatorID:   "user-1",
		DueDate:     &dueDate,
		CreatedAt:   time.Now().Add(-7 * 24 * time.Hour),
		UpdatedAt:   time.Now().Add(-1 * time.Hour),
	}

	comment1 := &comments.Comment{
		ID:        "comment-1",
		IssueID:   "issue-1",
		AuthorID:  "user-2",
		Content:   "I've started working on the mockups. Will share them soon!",
		CreatedAt: time.Now().Add(-2 * 24 * time.Hour),
	}

	comment2 := &comments.Comment{
		ID:        "comment-2",
		IssueID:   "issue-1",
		AuthorID:  "user-1",
		Content:   "Great! Looking forward to seeing them.",
		CreatedAt: time.Now().Add(-1 * 24 * time.Hour),
	}

	trelloComments := []*TrelloComment{
		{Comment: comment1, User: user2},
		{Comment: comment2, User: user},
	}

	labels := []*TrelloLabel{
		{ID: "green", Name: "Design", Color: "#61bd4f"},
		{ID: "blue", Name: "Frontend", Color: "#0079bf"},
	}

	allLabels := []*TrelloLabel{
		{ID: "green", Name: "Green", Color: "#61bd4f"},
		{ID: "yellow", Name: "Yellow", Color: "#f2d600"},
		{ID: "orange", Name: "Orange", Color: "#ff9f1a"},
		{ID: "red", Name: "Red", Color: "#eb5a46"},
		{ID: "purple", Name: "Purple", Color: "#c377e0"},
		{ID: "blue", Name: "Blue", Color: "#0079bf"},
	}

	cycle := &cycles.Cycle{
		ID:        "cycle-1",
		TeamID:    "team-1",
		Name:      "Sprint 5",
		Status:    "active",
		StartDate: time.Now().Add(-7 * 24 * time.Hour),
		EndDate:   time.Now().Add(7 * 24 * time.Hour),
	}

	data := TrelloCardData{
		Title:       "WEB-1 | Design new homepage",
		User:        user,
		Workspace:   workspace,
		Workspaces:  []*workspaces.Workspace{workspace},
		Board:       project,
		Card:        card,
		List:        col,
		Lists:       []*columns.Column{col},
		Labels:      labels,
		AllLabels:   allLabels,
		Members:     []*users.User{user},
		AllMembers:  []*users.User{user, user2},
		Comments:    []*comments.Comment{comment1, comment2},
		Cycles:      []*cycles.Cycle{cycle},
		Fields:      []*fields.Field{},
		CommentList: trelloComments,
	}

	var buf bytes.Buffer
	err := templates["trello-card"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Trello card template error: %v", err)
	}
}

// TestTrelloCardEmptyTemplate tests the Trello card template with minimal data
func TestTrelloCardEmptyTemplate(t *testing.T) {
	templates := loadTrelloTemplates(t)

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
		ID:     "proj-1",
		TeamID: "team-1",
		Name:   "Test Project",
		Key:    "TEST",
	}

	col := &columns.Column{
		ID:        "col-1",
		ProjectID: "proj-1",
		Name:      "To Do",
	}

	card := &issues.Issue{
		ID:        "issue-1",
		ProjectID: "proj-1",
		ColumnID:  "col-1",
		Key:       "TEST-1",
		Title:     "Simple task",
		CreatorID: "user-1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	data := TrelloCardData{
		Title:       "TEST-1 | Simple task",
		User:        user,
		Workspace:   workspace,
		Workspaces:  []*workspaces.Workspace{workspace},
		Board:       project,
		Card:        card,
		List:        col,
		Lists:       []*columns.Column{col},
		Labels:      []*TrelloLabel{},
		AllLabels:   []*TrelloLabel{},
		Members:     []*users.User{},
		AllMembers:  []*users.User{user},
		Comments:    []*comments.Comment{},
		Cycles:      []*cycles.Cycle{},
		Fields:      []*fields.Field{},
		CommentList: []*TrelloComment{},
	}

	var buf bytes.Buffer
	err := templates["trello-card"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Trello card empty template error: %v", err)
	}
}

// TestTrelloBoardWithManyLists tests the Trello board template with many lists
func TestTrelloBoardWithManyLists(t *testing.T) {
	templates := loadTrelloTemplates(t)

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
		ID:     "proj-1",
		TeamID: "team-1",
		Name:   "Sprint Board",
		Key:    "SPR",
	}

	// Create 6 lists with various cards
	listNames := []string{"Backlog", "To Do", "In Progress", "Review", "Testing", "Done"}
	var lists []*TrelloList

	for i, name := range listNames {
		col := &columns.Column{
			ID:        "col-" + string(rune('1'+i)),
			ProjectID: "proj-1",
			Name:      name,
			Position:  i,
		}

		cards := []*TrelloCard{}
		// Add some cards to each list
		for j := 0; j < 3; j++ {
			card := &TrelloCard{
				Issue: &issues.Issue{
					ID:       "issue-" + string(rune('1'+i)) + string(rune('1'+j)),
					ColumnID: col.ID,
					Key:      "SPR-" + string(rune('1'+i*3+j)),
					Title:    "Task " + string(rune('A'+i*3+j)),
				},
				Labels:  []*TrelloLabel{},
				Members: []*users.User{},
			}
			cards = append(cards, card)
		}

		lists = append(lists, &TrelloList{Column: col, Cards: cards})
	}

	data := TrelloBoardData{
		Title:      "Sprint Board | Trello",
		User:       user,
		Workspace:  workspace,
		Workspaces: []*workspaces.Workspace{workspace},
		Board:      project,
		Lists:      lists,
		Members:    []*users.User{user},
		Labels:     []*TrelloLabel{},
		AllTeams:   []*teams.Team{},
		Projects:   []*projects.Project{project},
	}

	var buf bytes.Buffer
	err := templates["trello-board"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Trello board with many lists template error: %v", err)
	}
}

// TestTrelloCardWithLongDescription tests the Trello card template with long description
func TestTrelloCardWithLongDescription(t *testing.T) {
	templates := loadTrelloTemplates(t)

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
		ID:     "proj-1",
		TeamID: "team-1",
		Name:   "Test Project",
		Key:    "TEST",
	}

	col := &columns.Column{
		ID:   "col-1",
		Name: "In Progress",
	}

	longDescription := `# Project Overview

This is a comprehensive task that requires careful planning and execution.

## Requirements

1. First, we need to analyze the current system
2. Then, design the new architecture
3. Implement the changes incrementally
4. Write comprehensive tests
5. Deploy to staging environment
6. Get stakeholder approval
7. Deploy to production

## Technical Details

- Use the new API v2 endpoints
- Ensure backwards compatibility
- Add proper error handling
- Include logging and monitoring

## Acceptance Criteria

- [ ] All tests passing
- [ ] Code review approved
- [ ] Documentation updated
- [ ] Performance benchmarks met`

	card := &issues.Issue{
		ID:          "issue-1",
		ProjectID:   "proj-1",
		ColumnID:    "col-1",
		Key:         "TEST-1",
		Title:       "Major refactoring task",
		Description: longDescription,
		CreatorID:   "user-1",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	data := TrelloCardData{
		Title:       "TEST-1 | Major refactoring task",
		User:        user,
		Workspace:   workspace,
		Workspaces:  []*workspaces.Workspace{workspace},
		Board:       project,
		Card:        card,
		List:        col,
		Lists:       []*columns.Column{col},
		Labels:      []*TrelloLabel{},
		AllLabels:   []*TrelloLabel{},
		Members:     []*users.User{},
		AllMembers:  []*users.User{user},
		Comments:    []*comments.Comment{},
		Cycles:      []*cycles.Cycle{},
		Fields:      []*fields.Field{},
		CommentList: []*TrelloComment{},
	}

	var buf bytes.Buffer
	err := templates["trello-card"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Trello card with long description template error: %v", err)
	}
}
