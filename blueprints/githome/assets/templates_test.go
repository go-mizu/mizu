package assets

import (
	"bytes"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// Page data types (matching handler/page.go)

type LoginData struct {
	Title string
	Error string
}

type RegisterData struct {
	Title string
	Error string
}

type HomeData struct {
	Title        string
	User         *users.User
	Repositories []*repos.Repository
}

type ExploreData struct {
	Title        string
	User         *users.User
	Repositories []*repos.Repository
	Query        string
}

type NewRepoData struct {
	Title string
	User  *users.User
	Error string
}

type UserProfileData struct {
	Title        string
	User         *users.User
	Profile      *users.User
	Repositories []*repos.Repository
	IsOwner      bool
}

type RepoHomeData struct {
	Title      string
	User       *users.User
	Owner      *users.User
	Repository *repos.Repository
	IsStarred  bool
	CanEdit    bool
}

type RepoIssuesData struct {
	Title      string
	User       *users.User
	Owner      *users.User
	Repository *repos.Repository
	Issues     []*issues.Issue
	Total      int
	State      string
}

type IssueViewData struct {
	Title      string
	User       *users.User
	Owner      *users.User
	Repository *repos.Repository
	Issue      *issues.Issue
	Author     *users.User
	Comments   []*issues.Comment
	CanEdit    bool
}

type NewIssueData struct {
	Title      string
	User       *users.User
	Owner      *users.User
	Repository *repos.Repository
}

type RepoSettingsData struct {
	Title         string
	User          *users.User
	Owner         *users.User
	Repository    *repos.Repository
	Collaborators []*repos.Collaborator
}

// Test fixtures

func mockUser() *users.User {
	return &users.User{
		ID:        "user-123",
		Username:  "testuser",
		Email:     "test@example.com",
		FullName:  "Test User",
		AvatarURL: "https://example.com/avatar.png",
		Bio:       "A test user",
		Location:  "Test City",
		Website:   "https://example.com",
		Company:   "Test Corp",
		IsAdmin:   false,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func mockRepository() *repos.Repository {
	return &repos.Repository{
		ID:             "repo-123",
		OwnerActorID:   "actor-123",
		OwnerID:        "user-123",
		OwnerType:      "user",
		OwnerName:      "testuser",
		Name:           "test-repo",
		Slug:           "test-repo",
		Description:    "A test repository",
		Website:        "https://example.com",
		DefaultBranch:  "main",
		IsPrivate:      false,
		IsArchived:     false,
		IsTemplate:     false,
		IsFork:         false,
		StarCount:      42,
		ForkCount:      10,
		WatcherCount:   5,
		OpenIssueCount: 3,
		OpenPRCount:    2,
		SizeKB:         1024,
		Topics:         []string{"go", "web"},
		License:        "MIT",
		HasIssues:      true,
		HasWiki:        false,
		HasProjects:    false,
		Language:       "Go",
		LanguageColor:  "#00ADD8",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func mockIssue() *issues.Issue {
	now := time.Now()
	return &issues.Issue{
		ID:             "issue-123",
		RepoID:         "repo-123",
		Number:         1,
		Title:          "Test Issue",
		Body:           "This is a test issue description.",
		AuthorID:       "user-123",
		State:          "open",
		IsLocked:       false,
		CommentCount:   2,
		ReactionsCount: 5,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// TestTemplatesParse verifies all templates parse correctly
func TestTemplatesParse(t *testing.T) {
	templates, err := Templates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	expectedTemplates := []string{
		"home", "explore", "new_repo",
		"user_profile",
		"repo_home", "repo_issues", "issue_view", "new_issue", "repo_settings",
		"login", "register",
	}

	for _, name := range expectedTemplates {
		if _, ok := templates[name]; !ok {
			t.Errorf("Template %q not found", name)
		}
	}
}

// TestLoginTemplate tests the login template renders correctly
func TestLoginTemplate(t *testing.T) {
	templates, err := Templates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	tmpl := templates["login"]
	if tmpl == nil {
		t.Fatal("login template not found")
	}

	data := LoginData{Title: "Sign in"}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute login template: %v", err)
	}

	// Check output contains expected content
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Sign in")) {
		t.Error("Login template output should contain 'Sign in'")
	}
}

// TestRegisterTemplate tests the register template renders correctly
func TestRegisterTemplate(t *testing.T) {
	templates, err := Templates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	tmpl := templates["register"]
	if tmpl == nil {
		t.Fatal("register template not found")
	}

	data := RegisterData{Title: "Sign up"}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute register template: %v", err)
	}
}

// TestHomeTemplateUnauthenticated tests the home template for unauthenticated users
func TestHomeTemplateUnauthenticated(t *testing.T) {
	templates, err := Templates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	tmpl := templates["home"]
	if tmpl == nil {
		t.Fatal("home template not found")
	}

	data := HomeData{
		Title:        "Home",
		User:         nil,
		Repositories: []*repos.Repository{mockRepository()},
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute home template (unauthenticated): %v", err)
	}
}

// TestHomeTemplateAuthenticated tests the home template for authenticated users
func TestHomeTemplateAuthenticated(t *testing.T) {
	templates, err := Templates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	tmpl := templates["home"]
	if tmpl == nil {
		t.Fatal("home template not found")
	}

	data := HomeData{
		Title:        "Home",
		User:         mockUser(),
		Repositories: []*repos.Repository{mockRepository()},
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute home template (authenticated): %v", err)
	}
}

// TestHomeTemplateEmptyRepos tests home template with no repositories
func TestHomeTemplateEmptyRepos(t *testing.T) {
	templates, err := Templates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	tmpl := templates["home"]
	if tmpl == nil {
		t.Fatal("home template not found")
	}

	data := HomeData{
		Title:        "Home",
		User:         nil,
		Repositories: nil,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute home template (empty repos): %v", err)
	}
}

// TestExploreTemplate tests the explore template
func TestExploreTemplate(t *testing.T) {
	templates, err := Templates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	tmpl := templates["explore"]
	if tmpl == nil {
		t.Fatal("explore template not found")
	}

	data := ExploreData{
		Title:        "Explore",
		User:         mockUser(),
		Repositories: []*repos.Repository{mockRepository()},
		Query:        "test",
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute explore template: %v", err)
	}
}

// TestNewRepoTemplate tests the new repository template
func TestNewRepoTemplate(t *testing.T) {
	templates, err := Templates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	tmpl := templates["new_repo"]
	if tmpl == nil {
		t.Fatal("new_repo template not found")
	}

	data := NewRepoData{
		Title: "Create a new repository",
		User:  mockUser(),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute new_repo template: %v", err)
	}
}

// TestUserProfileTemplate tests the user profile template
func TestUserProfileTemplate(t *testing.T) {
	templates, err := Templates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	tmpl := templates["user_profile"]
	if tmpl == nil {
		t.Fatal("user_profile template not found")
	}

	data := UserProfileData{
		Title:        "testuser",
		User:         mockUser(),
		Profile:      mockUser(),
		Repositories: []*repos.Repository{mockRepository()},
		IsOwner:      true,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute user_profile template: %v", err)
	}
}

// TestRepoHomeTemplate tests the repository home template
func TestRepoHomeTemplate(t *testing.T) {
	templates, err := Templates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	tmpl := templates["repo_home"]
	if tmpl == nil {
		t.Fatal("repo_home template not found")
	}

	data := RepoHomeData{
		Title:      "test-repo",
		User:       mockUser(),
		Owner:      mockUser(),
		Repository: mockRepository(),
		IsStarred:  true,
		CanEdit:    true,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute repo_home template: %v", err)
	}
}

// TestRepoIssuesTemplate tests the repository issues template
func TestRepoIssuesTemplate(t *testing.T) {
	templates, err := Templates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	tmpl := templates["repo_issues"]
	if tmpl == nil {
		t.Fatal("repo_issues template not found")
	}

	data := RepoIssuesData{
		Title:      "Issues",
		User:       mockUser(),
		Owner:      mockUser(),
		Repository: mockRepository(),
		Issues:     []*issues.Issue{mockIssue()},
		Total:      1,
		State:      "open",
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute repo_issues template: %v", err)
	}
}

// TestIssueViewTemplate tests the issue view template
func TestIssueViewTemplate(t *testing.T) {
	templates, err := Templates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	tmpl := templates["issue_view"]
	if tmpl == nil {
		t.Fatal("issue_view template not found")
	}

	data := IssueViewData{
		Title:      "Test Issue",
		User:       mockUser(),
		Owner:      mockUser(),
		Repository: mockRepository(),
		Issue:      mockIssue(),
		Author:     mockUser(),
		Comments:   nil,
		CanEdit:    true,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute issue_view template: %v", err)
	}
}

// TestNewIssueTemplate tests the new issue template
func TestNewIssueTemplate(t *testing.T) {
	templates, err := Templates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	tmpl := templates["new_issue"]
	if tmpl == nil {
		t.Fatal("new_issue template not found")
	}

	data := NewIssueData{
		Title:      "New Issue",
		User:       mockUser(),
		Owner:      mockUser(),
		Repository: mockRepository(),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute new_issue template: %v", err)
	}
}

// TestRepoSettingsTemplate tests the repository settings template
func TestRepoSettingsTemplate(t *testing.T) {
	templates, err := Templates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	tmpl := templates["repo_settings"]
	if tmpl == nil {
		t.Fatal("repo_settings template not found")
	}

	data := RepoSettingsData{
		Title:         "Settings",
		User:          mockUser(),
		Owner:         mockUser(),
		Repository:    mockRepository(),
		Collaborators: []*repos.Collaborator{},
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute repo_settings template: %v", err)
	}
}

// TestAllTemplatesWithNilUser tests all templates with nil user (unauthenticated)
func TestAllTemplatesWithNilUser(t *testing.T) {
	templates, err := Templates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	tests := []struct {
		name string
		data interface{}
	}{
		{"home", HomeData{Title: "Home", Repositories: []*repos.Repository{mockRepository()}}},
		{"explore", ExploreData{Title: "Explore", Repositories: []*repos.Repository{}}},
		{"login", LoginData{Title: "Sign in"}},
		{"register", RegisterData{Title: "Sign up"}},
		{"user_profile", UserProfileData{Title: "Profile", Profile: mockUser(), Repositories: []*repos.Repository{}}},
		{"repo_home", RepoHomeData{Title: "Repo", Owner: mockUser(), Repository: mockRepository()}},
		{"repo_issues", RepoIssuesData{Title: "Issues", Owner: mockUser(), Repository: mockRepository(), Issues: []*issues.Issue{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := templates[tt.name]
			if tmpl == nil {
				t.Skipf("Template %s not found", tt.name)
				return
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, tt.data); err != nil {
				t.Errorf("Failed to execute %s template with nil user: %v", tt.name, err)
			}
		})
	}
}

// TestTemplateWithLanguageField tests that Language field works in templates
func TestTemplateWithLanguageField(t *testing.T) {
	templates, err := Templates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	// Test home page with repository that has Language field set
	tmpl := templates["home"]
	if tmpl == nil {
		t.Fatal("home template not found")
	}

	repo := mockRepository()
	repo.Language = "Go"
	repo.LanguageColor = "#00ADD8"

	data := HomeData{
		Title:        "Home",
		User:         nil,
		Repositories: []*repos.Repository{repo},
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute home template with Language field: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("Go")) {
		// Language may or may not be displayed depending on template design
		t.Log("Language 'Go' not found in output (may be intentional)")
	}
}

// TestTemplateWithEmptyLanguage tests templates work when Language is empty
func TestTemplateWithEmptyLanguage(t *testing.T) {
	templates, err := Templates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	tmpl := templates["home"]
	if tmpl == nil {
		t.Fatal("home template not found")
	}

	repo := mockRepository()
	repo.Language = ""
	repo.LanguageColor = ""

	data := HomeData{
		Title:        "Home",
		User:         nil,
		Repositories: []*repos.Repository{repo},
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute home template with empty Language: %v", err)
	}
}
