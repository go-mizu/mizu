package handler

import (
	"bytes"
	"html/template"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/assets"
	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
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
		Title: "Sign In",
	}

	var buf bytes.Buffer
	err := templates["login"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Login template error: %v", err)
	}

	// Verify essential HTML content
	output := buf.String()
	if !containsAll(output, "Sign in to GitHome", "Email address", "Password", "Sign in") {
		t.Error("Login template missing expected content")
	}
}

// TestLoginTemplateWithError tests the login template with an error message
func TestLoginTemplateWithError(t *testing.T) {
	templates := loadTemplates(t)

	data := LoginData{
		Title: "Sign In",
		Error: "Invalid credentials",
	}

	var buf bytes.Buffer
	err := templates["login"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Login template error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "Invalid credentials") {
		t.Error("Login template should display error message")
	}
}

// TestRegisterTemplate tests the register template renders without errors
func TestRegisterTemplate(t *testing.T) {
	templates := loadTemplates(t)

	data := RegisterData{
		Title: "Sign Up",
	}

	var buf bytes.Buffer
	err := templates["register"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Register template error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "Create your account", "Username", "Email", "Password", "Create account") {
		t.Error("Register template missing expected content")
	}
}

// TestHomeTemplateUnauthenticated tests the home template for unauthenticated users
func TestHomeTemplateUnauthenticated(t *testing.T) {
	templates := loadTemplates(t)

	data := HomeData{
		Title:        "Home",
		User:         nil,
		Repositories: []*repos.Repository{},
	}

	var buf bytes.Buffer
	err := templates["home"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Home template error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "Build and ship software", "Sign up for free") {
		t.Error("Unauthenticated home should show landing page")
	}
}

// TestHomeTemplateAuthenticated tests the home template for authenticated users
func TestHomeTemplateAuthenticated(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:       "user-1",
		Username: "testuser",
		Email:    "test@example.com",
	}

	repo := &repos.Repository{
		ID:        "repo-1",
		Name:      "test-repo",
		OwnerID:   user.ID,
		OwnerName: user.Username,
	}

	data := HomeData{
		Title:        "Home",
		User:         user,
		Repositories: []*repos.Repository{repo},
	}

	var buf bytes.Buffer
	err := templates["home"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Home template error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "Repositories", "test-repo") {
		t.Error("Authenticated home should show dashboard with repositories")
	}
}

// TestExploreTemplate tests the explore template renders without errors
func TestExploreTemplate(t *testing.T) {
	templates := loadTemplates(t)

	repo := &repos.Repository{
		ID:          "repo-1",
		Name:        "awesome-project",
		OwnerID:     "user-1",
		OwnerName:   "developer",
		Description: "An awesome project",
		StarCount:   42,
	}

	data := ExploreData{
		Title:        "Explore",
		User:         nil,
		Repositories: []*repos.Repository{repo},
		Query:        "",
	}

	var buf bytes.Buffer
	err := templates["explore"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Explore template error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "Explore repositories", "awesome-project", "developer") {
		t.Error("Explore template missing expected content")
	}
}

// TestExploreTemplateEmpty tests the explore template with no repositories
func TestExploreTemplateEmpty(t *testing.T) {
	templates := loadTemplates(t)

	data := ExploreData{
		Title:        "Explore",
		User:         nil,
		Repositories: []*repos.Repository{},
		Query:        "",
	}

	var buf bytes.Buffer
	err := templates["explore"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Explore template error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "No repositories found") {
		t.Error("Explore template should show empty state")
	}
}

// TestNewRepoTemplate tests the new repository template renders without errors
func TestNewRepoTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:       "user-1",
		Username: "testuser",
	}

	data := NewRepoData{
		Title: "Create a new repository",
		User:  user,
	}

	var buf bytes.Buffer
	err := templates["new_repo"].Execute(&buf, data)
	if err != nil {
		t.Errorf("NewRepo template error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "Create a new repository", "testuser", "Public", "Private") {
		t.Error("NewRepo template missing expected content")
	}
}

// TestUserProfileTemplate tests the user profile template renders without errors
func TestUserProfileTemplate(t *testing.T) {
	templates := loadTemplates(t)

	profileUser := &users.User{
		ID:       "user-1",
		Username: "developer",
		FullName: "Developer Name",
		Bio:      "A passionate developer",
	}

	repo := &repos.Repository{
		ID:        "repo-1",
		Name:      "my-project",
		OwnerID:   profileUser.ID,
		OwnerName: profileUser.Username,
		StarCount: 10,
	}

	data := UserProfileData{
		Title:        profileUser.Username,
		User:         nil,
		Profile:      profileUser,
		Repositories: []*repos.Repository{repo},
		IsOwner:      false,
	}

	var buf bytes.Buffer
	err := templates["user_profile"].Execute(&buf, data)
	if err != nil {
		t.Errorf("UserProfile template error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "developer", "my-project", "Repositories") {
		t.Error("UserProfile template missing expected content")
	}
}

// TestUserProfileTemplateOwner tests the user profile template for the profile owner
func TestUserProfileTemplateOwner(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:       "user-1",
		Username: "developer",
	}

	data := UserProfileData{
		Title:        user.Username,
		User:         user,
		Profile:      user,
		Repositories: []*repos.Repository{},
		IsOwner:      true,
	}

	var buf bytes.Buffer
	err := templates["user_profile"].Execute(&buf, data)
	if err != nil {
		t.Errorf("UserProfile template error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "Edit profile", "Create repository") {
		t.Error("Owner profile should show edit and create options")
	}
}

// TestRepoHomeTemplate tests the repository home template renders without errors
func TestRepoHomeTemplate(t *testing.T) {
	templates := loadTemplates(t)

	owner := &users.User{
		ID:       "user-1",
		Username: "owner",
	}

	repo := &repos.Repository{
		ID:           "repo-1",
		Name:         "awesome-repo",
		OwnerID:      owner.ID,
		OwnerName:    owner.Username,
		Description:  "An awesome repository",
		StarCount:    100,
		ForkCount:    25,
		WatcherCount: 10,
	}

	data := RepoHomeData{
		Title:      repo.Name,
		User:       nil,
		Owner:      owner,
		Repository: repo,
		IsStarred:  false,
		CanEdit:    false,
	}

	var buf bytes.Buffer
	err := templates["repo_home"].Execute(&buf, data)
	if err != nil {
		t.Errorf("RepoHome template error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "owner", "awesome-repo", "Code", "Issues", "Star") {
		t.Error("RepoHome template missing expected content")
	}
}

// TestRepoHomeTemplateStarred tests the repository home when starred
func TestRepoHomeTemplateStarred(t *testing.T) {
	templates := loadTemplates(t)

	owner := &users.User{
		ID:       "user-1",
		Username: "owner",
	}

	user := &users.User{
		ID:       "user-2",
		Username: "viewer",
	}

	repo := &repos.Repository{
		ID:        "repo-1",
		Name:      "awesome-repo",
		OwnerID:   owner.ID,
		OwnerName: owner.Username,
		StarCount: 100,
	}

	data := RepoHomeData{
		Title:      repo.Name,
		User:       user,
		Owner:      owner,
		Repository: repo,
		IsStarred:  true,
		CanEdit:    false,
	}

	var buf bytes.Buffer
	err := templates["repo_home"].Execute(&buf, data)
	if err != nil {
		t.Errorf("RepoHome template error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "Starred") {
		t.Error("RepoHome should show Starred when user has starred")
	}
}

// TestRepoIssuesTemplate tests the repository issues template renders without errors
func TestRepoIssuesTemplate(t *testing.T) {
	templates := loadTemplates(t)

	owner := &users.User{
		ID:       "user-1",
		Username: "owner",
	}

	repo := &repos.Repository{
		ID:        "repo-1",
		Name:      "test-repo",
		OwnerID:   owner.ID,
		OwnerName: owner.Username,
	}

	issue := &issues.Issue{
		ID:        "issue-1",
		Number:    1,
		Title:     "First Issue",
		State:     "open",
		AuthorID:  owner.ID,
		CreatedAt: time.Now(),
	}

	data := RepoIssuesData{
		Title:      "Issues",
		User:       nil,
		Owner:      owner,
		Repository: repo,
		Issues:     []*issues.Issue{issue},
		Total:      1,
		State:      "open",
	}

	var buf bytes.Buffer
	err := templates["repo_issues"].Execute(&buf, data)
	if err != nil {
		t.Errorf("RepoIssues template error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "First Issue", "#1", "Open") {
		t.Error("RepoIssues template missing expected content")
	}
}

// TestRepoIssuesTemplateEmpty tests the repository issues template with no issues
func TestRepoIssuesTemplateEmpty(t *testing.T) {
	templates := loadTemplates(t)

	owner := &users.User{
		ID:       "user-1",
		Username: "owner",
	}

	repo := &repos.Repository{
		ID:        "repo-1",
		Name:      "test-repo",
		OwnerID:   owner.ID,
		OwnerName: owner.Username,
	}

	data := RepoIssuesData{
		Title:      "Issues",
		User:       nil,
		Owner:      owner,
		Repository: repo,
		Issues:     []*issues.Issue{},
		Total:      0,
		State:      "open",
	}

	var buf bytes.Buffer
	err := templates["repo_issues"].Execute(&buf, data)
	if err != nil {
		t.Errorf("RepoIssues template error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "There aren't any open issues") {
		t.Error("RepoIssues should show empty state")
	}
}

// TestIssueViewTemplate tests the issue view template renders without errors
func TestIssueViewTemplate(t *testing.T) {
	templates := loadTemplates(t)

	owner := &users.User{
		ID:       "user-1",
		Username: "owner",
	}

	author := &users.User{
		ID:       "user-2",
		Username: "author",
	}

	repo := &repos.Repository{
		ID:        "repo-1",
		Name:      "test-repo",
		OwnerID:   owner.ID,
		OwnerName: owner.Username,
	}

	issue := &issues.Issue{
		ID:        "issue-1",
		Number:    42,
		Title:     "Bug Report",
		Body:      "There is a bug in the code",
		State:     "open",
		AuthorID:  author.ID,
		CreatedAt: time.Now(),
	}

	comment := &issues.Comment{
		ID:        "comment-1",
		TargetID:  issue.ID,
		UserID:    owner.ID,
		Body:      "Thanks for reporting!",
		CreatedAt: time.Now(),
	}

	data := IssueViewData{
		Title:      issue.Title,
		User:       nil,
		Owner:      owner,
		Repository: repo,
		Issue:      issue,
		Author:     author,
		Comments:   []*issues.Comment{comment},
		CanEdit:    false,
	}

	var buf bytes.Buffer
	err := templates["issue_view"].Execute(&buf, data)
	if err != nil {
		t.Errorf("IssueView template error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "Bug Report", "#42", "Open", "There is a bug", "Thanks for reporting") {
		t.Error("IssueView template missing expected content")
	}
}

// TestNewIssueTemplate tests the new issue template renders without errors
func TestNewIssueTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:       "user-1",
		Username: "creator",
	}

	owner := &users.User{
		ID:       "user-2",
		Username: "owner",
	}

	repo := &repos.Repository{
		ID:        "repo-1",
		Name:      "test-repo",
		OwnerID:   owner.ID,
		OwnerName: owner.Username,
	}

	data := NewIssueData{
		Title:      "New Issue",
		User:       user,
		Owner:      owner,
		Repository: repo,
	}

	var buf bytes.Buffer
	err := templates["new_issue"].Execute(&buf, data)
	if err != nil {
		t.Errorf("NewIssue template error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "New issue", "Title", "Submit new issue") {
		t.Error("NewIssue template missing expected content")
	}
}

// TestRepoSettingsTemplate tests the repository settings template renders without errors
func TestRepoSettingsTemplate(t *testing.T) {
	templates := loadTemplates(t)

	owner := &users.User{
		ID:       "user-1",
		Username: "owner",
	}

	repo := &repos.Repository{
		ID:          "repo-1",
		Name:        "test-repo",
		OwnerID:     owner.ID,
		OwnerName:   owner.Username,
		Description: "A test repository",
		IsPrivate:   false,
	}

	collab := &repos.Collaborator{
		UserID:     "user-2",
		RepoID:     repo.ID,
		Permission: "write",
	}

	data := RepoSettingsData{
		Title:         "Settings",
		User:          owner,
		Owner:         owner,
		Repository:    repo,
		Collaborators: []*repos.Collaborator{collab},
	}

	var buf bytes.Buffer
	err := templates["repo_settings"].Execute(&buf, data)
	if err != nil {
		t.Errorf("RepoSettings template error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "Settings", "General", "Repository name", "Collaborators", "Danger Zone") {
		t.Error("RepoSettings template missing expected content")
	}
}

// TestRepoSettingsTemplateNoCollaborators tests the settings template with no collaborators
func TestRepoSettingsTemplateNoCollaborators(t *testing.T) {
	templates := loadTemplates(t)

	owner := &users.User{
		ID:       "user-1",
		Username: "owner",
	}

	repo := &repos.Repository{
		ID:        "repo-1",
		Name:      "test-repo",
		OwnerID:   owner.ID,
		OwnerName: owner.Username,
	}

	data := RepoSettingsData{
		Title:         "Settings",
		User:          owner,
		Owner:         owner,
		Repository:    repo,
		Collaborators: []*repos.Collaborator{},
	}

	var buf bytes.Buffer
	err := templates["repo_settings"].Execute(&buf, data)
	if err != nil {
		t.Errorf("RepoSettings template error: %v", err)
	}

	output := buf.String()
	if !containsAll(output, "No collaborators yet") {
		t.Error("RepoSettings should show no collaborators message")
	}
}

// containsAll checks if the output contains all the specified substrings
func containsAll(output string, substrings ...string) bool {
	for _, s := range substrings {
		if !contains(output, s) {
			return false
		}
	}
	return true
}

// contains checks if output contains substring
func contains(output, substring string) bool {
	return bytes.Contains([]byte(output), []byte(substring))
}
