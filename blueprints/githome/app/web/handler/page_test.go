package handler

import (
	"bytes"
	"html/template"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/assets"
	"github.com/go-mizu/blueprints/githome/feature/branches"
	"github.com/go-mizu/blueprints/githome/feature/comments"
	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/labels"
	"github.com/go-mizu/blueprints/githome/feature/milestones"
	"github.com/go-mizu/blueprints/githome/feature/notifications"
	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

var _ = template.Template{} // Silence unused import

// loadTemplates loads all templates for testing using the assets package.
func loadTemplates(t *testing.T) map[string]*template.Template {
	t.Helper()

	templates, err := assets.Templates()
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	return templates
}

// TestLoginTemplate tests the login template renders without errors.
func TestLoginTemplate(t *testing.T) {
	templates := loadTemplates(t)

	data := LoginData{
		Title:    "Sign in",
		ReturnTo: "/dashboard",
	}

	var buf bytes.Buffer
	err := templates["login"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Login template error: %v", err)
	}
}

// TestLoginTemplateWithError tests the login template with error message.
func TestLoginTemplateWithError(t *testing.T) {
	templates := loadTemplates(t)

	data := LoginData{
		Title: "Sign in",
		Error: "Invalid username or password",
	}

	var buf bytes.Buffer
	err := templates["login"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Login template with error: %v", err)
	}
}

// TestRegisterTemplate tests the register template renders without errors.
func TestRegisterTemplate(t *testing.T) {
	templates := loadTemplates(t)

	data := RegisterData{
		Title: "Create your account",
	}

	var buf bytes.Buffer
	err := templates["register"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Register template error: %v", err)
	}
}

// TestRegisterTemplateWithError tests the register template with error.
func TestRegisterTemplateWithError(t *testing.T) {
	templates := loadTemplates(t)

	data := RegisterData{
		Title: "Create your account",
		Error: "Username already taken",
	}

	var buf bytes.Buffer
	err := templates["register"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Register template with error: %v", err)
	}
}

// TestHomeTemplate tests the home/dashboard template.
func TestHomeTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:        1,
		Login:     "testuser",
		Name:      "Test User",
		Email:     "test@example.com",
		AvatarURL: "https://example.com/avatar.png",
	}

	repo := &repos.Repository{
		ID:              1,
		Name:            "test-repo",
		FullName:        "testuser/test-repo",
		Description:     "A test repository",
		Private:         false,
		DefaultBranch:   "main",
		StargazersCount: 10,
		ForksCount:      2,
		Language:        "Go",
		UpdatedAt:       time.Now(),
	}

	org := &orgs.OrgSimple{
		ID:        1,
		Login:     "test-org",
		AvatarURL: "https://example.com/org.png",
	}

	notif := &notifications.Notification{
		ID:     "1",
		Unread: true,
		Subject: &notifications.Subject{
			Title: "New issue",
			URL:   "/testuser/test-repo/issues/1",
			Type:  "Issue",
		},
		Repository: &notifications.Repository{
			ID:       1,
			FullName: "testuser/test-repo",
		},
		UpdatedAt: time.Now(),
	}

	data := HomeData{
		Title:         "Dashboard",
		User:          user,
		Repositories:  []*repos.Repository{repo},
		StarredRepos:  []*repos.Repository{repo},
		Organizations: []*orgs.OrgSimple{org},
		Notifications: []*notifications.Notification{notif},
		UnreadCount:   1,
		ActiveNav:     "home",
	}

	var buf bytes.Buffer
	err := templates["home"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Home template error: %v", err)
	}
}

// TestHomeTemplateEmpty tests the home template with no data.
func TestHomeTemplateEmpty(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:    1,
		Login: "testuser",
		Name:  "Test User",
	}

	data := HomeData{
		Title:         "Dashboard",
		User:          user,
		Repositories:  []*repos.Repository{},
		StarredRepos:  []*repos.Repository{},
		Organizations: []*orgs.OrgSimple{},
		Notifications: []*notifications.Notification{},
		UnreadCount:   0,
		ActiveNav:     "home",
	}

	var buf bytes.Buffer
	err := templates["home"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Home empty template error: %v", err)
	}
}

// TestExploreTemplate tests the explore page template.
func TestExploreTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:    1,
		Login: "testuser",
	}

	repo := &repos.Repository{
		ID:              1,
		Name:            "popular-repo",
		FullName:        "someone/popular-repo",
		Description:     "A popular repository",
		StargazersCount: 1000,
		ForksCount:      200,
		Language:        "Python",
	}

	data := ExploreData{
		Title:         "Explore repositories",
		User:          user,
		Repositories:  []*repos.Repository{repo},
		TrendingRepos: []*repos.Repository{repo},
		Query:         "",
		Language:      "",
		Sort:          "stars",
		Page:          1,
		TotalCount:    100,
		HasNext:       true,
		ActiveNav:     "explore",
	}

	var buf bytes.Buffer
	err := templates["explore"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Explore template error: %v", err)
	}
}

// TestExploreTemplateLoggedOut tests explore page for logged out users.
func TestExploreTemplateLoggedOut(t *testing.T) {
	templates := loadTemplates(t)

	data := ExploreData{
		Title:        "Explore repositories",
		User:         nil,
		Repositories: []*repos.Repository{},
		Sort:         "stars",
		ActiveNav:    "explore",
	}

	var buf bytes.Buffer
	err := templates["explore"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Explore logged out template error: %v", err)
	}
}

// TestUserProfileTemplate tests the user profile template.
func TestUserProfileTemplate(t *testing.T) {
	templates := loadTemplates(t)

	currentUser := &users.User{
		ID:    1,
		Login: "currentuser",
	}

	profileUser := &users.User{
		ID:            2,
		Login:         "profileuser",
		Name:          "Profile User",
		Bio:           "Software developer",
		Email:         "profile@example.com",
		Location:      "San Francisco",
		Company:       "Tech Corp",
		Blog:          "https://blog.example.com",
		AvatarURL:     "https://example.com/avatar.png",
		Followers:     50,
		Following:     30,
		PublicRepos:   10,
		TwitterUsername: "profileuser",
		CreatedAt:     time.Now().AddDate(-1, 0, 0),
	}

	repo := &repos.Repository{
		ID:              1,
		Name:            "my-project",
		FullName:        "profileuser/my-project",
		Description:     "My awesome project",
		StargazersCount: 25,
		ForksCount:      5,
		Language:        "JavaScript",
	}

	org := &orgs.OrgSimple{
		ID:    1,
		Login: "cool-org",
	}

	data := UserProfileData{
		Title:            profileUser.Login,
		User:             currentUser,
		ProfileUser:      profileUser,
		Repositories:     []*repos.Repository{repo},
		PinnedRepos:      []*repos.Repository{repo},
		Organizations:    []*orgs.OrgSimple{org},
		IsOwnProfile:     false,
		IsFollowing:      true,
		ContributionData: "",
		ActiveTab:        "overview",
		ActiveNav:        "",
	}

	var buf bytes.Buffer
	err := templates["user_profile"].Execute(&buf, data)
	if err != nil {
		t.Errorf("User profile template error: %v", err)
	}
}

// TestUserProfileTemplateOwn tests viewing own profile.
func TestUserProfileTemplateOwn(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:        1,
		Login:     "myuser",
		Name:      "My User",
		AvatarURL: "https://example.com/avatar.png",
	}

	data := UserProfileData{
		Title:         user.Login,
		User:          user,
		ProfileUser:   user,
		Repositories:  []*repos.Repository{},
		IsOwnProfile:  true,
		IsFollowing:   false,
		ActiveTab:     "repositories",
		ActiveNav:     "",
	}

	var buf bytes.Buffer
	err := templates["user_profile"].Execute(&buf, data)
	if err != nil {
		t.Errorf("User profile own template error: %v", err)
	}
}

// TestRepoHomeTemplate tests the repository home page template.
func TestRepoHomeTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:    1,
		Login: "testuser",
	}

	repo := &repos.Repository{
		ID:              1,
		Name:            "my-repo",
		FullName:        "testuser/my-repo",
		Description:     "A great repository",
		Private:         false,
		DefaultBranch:   "main",
		StargazersCount: 50,
		ForksCount:      10,
		OpenIssuesCount: 5,
		HasWiki:         true,
		HasIssues:       true,
		Language:        "Go",
		Owner: &users.SimpleUser{
			ID:    1,
			Login: "testuser",
		},
	}

	repoView := &RepoView{
		Repository: repo,
		Tabs: []RepoTab{
			{Name: "Code", URL: "/testuser/my-repo", Icon: "code", Active: true},
			{Name: "Issues", URL: "/testuser/my-repo/issues", Icon: "issue-opened", Count: 5},
			{Name: "Pull requests", URL: "/testuser/my-repo/pulls", Icon: "git-pull-request"},
			{Name: "Settings", URL: "/testuser/my-repo/settings", Icon: "gear"},
		},
		ActiveTab:  "code",
		CanPush:    true,
		CanAdmin:   true,
		IsStarred:  false,
		IsWatching: true,
	}

	branch := &branches.Branch{
		Name:      "main",
		Protected: true,
	}

	data := RepoHomeData{
		Title:         repo.FullName,
		User:          user,
		Repo:          repoView,
		Readme:        template.HTML("<h1>My Repo</h1><p>Welcome to my repository!</p>"),
		Tree:          []*TreeEntry{},
		Branches:      []*branches.Branch{branch},
		CurrentBranch: "main",
		Languages:     map[string]int{"Go": 80, "Makefile": 20},
		ActiveNav:     "",
	}

	var buf bytes.Buffer
	err := templates["repo_home"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Repo home template error: %v", err)
	}
}

// TestRepoHomeTemplateWithTree tests repo home with file tree.
func TestRepoHomeTemplateWithTree(t *testing.T) {
	templates := loadTemplates(t)

	repo := &repos.Repository{
		ID:            1,
		Name:          "my-repo",
		FullName:      "testuser/my-repo",
		DefaultBranch: "main",
		Owner: &users.SimpleUser{
			ID:    1,
			Login: "testuser",
		},
	}

	repoView := &RepoView{
		Repository: repo,
		Tabs: []RepoTab{
			{Name: "Code", URL: "/testuser/my-repo", Active: true},
		},
		ActiveTab: "code",
	}

	tree := []*TreeEntry{
		{Name: "README.md", Path: "README.md", Type: "file", Size: 1024},
		{Name: "src", Path: "src", Type: "dir"},
		{Name: "go.mod", Path: "go.mod", Type: "file", Size: 256},
	}

	data := RepoHomeData{
		Title:         repo.FullName,
		User:          nil,
		Repo:          repoView,
		Tree:          tree,
		CurrentBranch: "main",
		ActiveNav:     "",
	}

	var buf bytes.Buffer
	err := templates["repo_home"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Repo home with tree template error: %v", err)
	}
}

// TestRepoIssuesTemplate tests the issues list template.
func TestRepoIssuesTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:    1,
		Login: "testuser",
	}

	repo := &repos.Repository{
		ID:              1,
		Name:            "my-repo",
		FullName:        "testuser/my-repo",
		OpenIssuesCount: 5,
		Owner: &users.SimpleUser{
			ID:    1,
			Login: "testuser",
		},
	}

	repoView := &RepoView{
		Repository: repo,
		Tabs: []RepoTab{
			{Name: "Code", URL: "/testuser/my-repo"},
			{Name: "Issues", URL: "/testuser/my-repo/issues", Count: 5, Active: true},
		},
		ActiveTab: "issues",
	}

	issue := &issues.Issue{
		ID:        1,
		Number:    1,
		Title:     "Bug: Something is broken",
		Body:      "Please fix this",
		State:     "open",
		User: &users.SimpleUser{
			ID:    1,
			Login: "testuser",
		},
		CreatorID: 1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	issueView := &IssueView{
		Issue:   issue,
		TimeAgo: "2 hours ago",
	}

	label := &labels.Label{
		ID:    1,
		Name:  "bug",
		Color: "d73a4a",
	}

	milestone := &milestones.Milestone{
		ID:     1,
		Number: 1,
		Title:  "v1.0",
		State:  "open",
	}

	data := RepoIssuesData{
		Title:        "Issues",
		User:         user,
		Repo:         repoView,
		Issues:       []*IssueView{issueView},
		OpenCount:    5,
		ClosedCount:  3,
		Labels:       []*labels.Label{label},
		Milestones:   []*milestones.Milestone{milestone},
		CurrentState: "open",
		CurrentSort:  "created",
		Page:         1,
		HasNext:      false,
		ActiveNav:    "",
	}

	var buf bytes.Buffer
	err := templates["repo_issues"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Repo issues template error: %v", err)
	}
}

// TestRepoIssuesTemplateEmpty tests empty issues list.
func TestRepoIssuesTemplateEmpty(t *testing.T) {
	templates := loadTemplates(t)

	repo := &repos.Repository{
		ID:       1,
		Name:     "my-repo",
		FullName: "testuser/my-repo",
		Owner: &users.SimpleUser{
			ID:    1,
			Login: "testuser",
		},
	}

	repoView := &RepoView{
		Repository: repo,
		Tabs: []RepoTab{
			{Name: "Issues", URL: "/testuser/my-repo/issues", Active: true},
		},
		ActiveTab: "issues",
	}

	data := RepoIssuesData{
		Title:        "Issues",
		User:         nil,
		Repo:         repoView,
		Issues:       []*IssueView{},
		OpenCount:    0,
		ClosedCount:  0,
		Labels:       []*labels.Label{},
		Milestones:   []*milestones.Milestone{},
		CurrentState: "open",
		ActiveNav:    "",
	}

	var buf bytes.Buffer
	err := templates["repo_issues"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Repo issues empty template error: %v", err)
	}
}

// TestIssueViewTemplate tests the single issue view template.
func TestIssueViewTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:    1,
		Login: "testuser",
	}

	repo := &repos.Repository{
		ID:       1,
		Name:     "my-repo",
		FullName: "testuser/my-repo",
		Owner: &users.SimpleUser{
			ID:    1,
			Login: "testuser",
		},
	}

	repoView := &RepoView{
		Repository: repo,
		Tabs: []RepoTab{
			{Name: "Issues", URL: "/testuser/my-repo/issues", Active: true},
		},
		ActiveTab: "issues",
		CanPush:   true,
		CanAdmin:  true,
	}

	issue := &issues.Issue{
		ID:        1,
		Number:    42,
		Title:     "Feature request: Add dark mode",
		Body:      "Please add a dark mode option",
		State:     "open",
		User: &users.SimpleUser{
			ID:    2,
			Login: "featurerequest",
		},
		CreatorID: 2,
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}

	issueView := &IssueView{
		Issue:   issue,
		TimeAgo: "1 day ago",
	}

	comment := &comments.IssueComment{
		ID:        1,
		IssueID:   1,
		Body:      "Great idea! +1",
		CreatorID: 3,
		CreatedAt: time.Now().Add(-12 * time.Hour),
		UpdatedAt: time.Now().Add(-12 * time.Hour),
	}

	commentView := &CommentView{
		IssueComment: comment,
		Author: &users.SimpleUser{
			ID:        3,
			Login:     "commenter",
			AvatarURL: "https://example.com/commenter.png",
		},
		TimeAgo: "12 hours ago",
	}

	data := IssueDetailData{
		Title:      "Feature request: Add dark mode #42",
		User:       user,
		Repo:       repoView,
		Issue:      issueView,
		Comments:   []*CommentView{commentView},
		Timeline:   []*TimelineEvent{},
		Labels:     []*labels.Label{},
		Milestones: []*milestones.Milestone{},
		CanEdit:    true,
		CanClose:   true,
		Breadcrumbs: []Breadcrumb{
			{Label: "testuser/my-repo", URL: "/testuser/my-repo"},
			{Label: "Issues", URL: "/testuser/my-repo/issues"},
			{Label: "#42", URL: ""},
		},
		ActiveNav: "",
	}

	var buf bytes.Buffer
	err := templates["issue_view"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Issue view template error: %v", err)
	}
}

// TestNewIssueTemplate tests the create issue form template.
func TestNewIssueTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:    1,
		Login: "testuser",
	}

	repo := &repos.Repository{
		ID:       1,
		Name:     "my-repo",
		FullName: "testuser/my-repo",
		Owner: &users.SimpleUser{
			ID:    1,
			Login: "testuser",
		},
	}

	repoView := &RepoView{
		Repository: repo,
		Tabs: []RepoTab{
			{Name: "Issues", URL: "/testuser/my-repo/issues", Active: true},
		},
		ActiveTab: "issues",
	}

	label := &labels.Label{
		ID:    1,
		Name:  "enhancement",
		Color: "a2eeef",
	}

	milestone := &milestones.Milestone{
		ID:     1,
		Number: 1,
		Title:  "v2.0",
	}

	data := NewIssueData{
		Title:      "New Issue",
		User:       user,
		Repo:       repoView,
		Labels:     []*labels.Label{label},
		Milestones: []*milestones.Milestone{milestone},
		Error:      "",
		Breadcrumbs: []Breadcrumb{
			{Label: "testuser/my-repo", URL: "/testuser/my-repo"},
			{Label: "Issues", URL: "/testuser/my-repo/issues"},
			{Label: "New", URL: ""},
		},
		ActiveNav: "",
	}

	var buf bytes.Buffer
	err := templates["new_issue"].Execute(&buf, data)
	if err != nil {
		t.Errorf("New issue template error: %v", err)
	}
}

// TestNewRepoTemplate tests the create repository form template.
func TestNewRepoTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:    1,
		Login: "testuser",
	}

	org := &orgs.OrgSimple{
		ID:    1,
		Login: "my-org",
	}

	data := NewRepoData{
		Title:         "Create a new repository",
		User:          user,
		Organizations: []*orgs.OrgSimple{org},
		Licenses: []License{
			{Key: "mit", Name: "MIT License"},
			{Key: "apache-2.0", Name: "Apache License 2.0"},
		},
		GitignoreTemplates: []string{"Go", "Python", "Node"},
		Error:              "",
		ActiveNav:          "new",
	}

	var buf bytes.Buffer
	err := templates["new_repo"].Execute(&buf, data)
	if err != nil {
		t.Errorf("New repo template error: %v", err)
	}
}

// TestRepoSettingsTemplate tests the repository settings template.
func TestRepoSettingsTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:    1,
		Login: "testuser",
	}

	repo := &repos.Repository{
		ID:            1,
		Name:          "my-repo",
		FullName:      "testuser/my-repo",
		Private:       false,
		DefaultBranch: "main",
		HasIssues:     true,
		HasWiki:       true,
		HasProjects:   false,
		Owner: &users.SimpleUser{
			ID:    1,
			Login: "testuser",
		},
	}

	repoView := &RepoView{
		Repository: repo,
		Tabs: []RepoTab{
			{Name: "Settings", URL: "/testuser/my-repo/settings", Active: true},
		},
		ActiveTab: "settings",
		CanAdmin:  true,
	}

	collaborator := &CollaboratorView{
		SimpleUser: &users.SimpleUser{
			ID:        2,
			Login:     "collaborator",
			AvatarURL: "https://example.com/collab.png",
		},
		Permission: "write",
	}

	data := RepoSettingsData{
		Title:         "Settings",
		User:          user,
		Repo:          repoView,
		Collaborators: []*CollaboratorView{collaborator},
		Section:       "general",
		Error:         "",
		Success:       "",
		Breadcrumbs: []Breadcrumb{
			{Label: "testuser/my-repo", URL: "/testuser/my-repo"},
			{Label: "Settings", URL: ""},
		},
		ActiveNav: "",
	}

	var buf bytes.Buffer
	err := templates["repo_settings"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Repo settings template error: %v", err)
	}
}

// TestRepoSettingsTemplateDanger tests the danger zone section.
func TestRepoSettingsTemplateDanger(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:    1,
		Login: "testuser",
	}

	repo := &repos.Repository{
		ID:       1,
		Name:     "my-repo",
		FullName: "testuser/my-repo",
		Private:  false,
		Owner: &users.SimpleUser{
			ID:    1,
			Login: "testuser",
		},
	}

	repoView := &RepoView{
		Repository: repo,
		Tabs: []RepoTab{
			{Name: "Settings", URL: "/testuser/my-repo/settings", Active: true},
		},
		ActiveTab: "settings",
		CanAdmin:  true,
	}

	data := RepoSettingsData{
		Title:         "Settings",
		User:          user,
		Repo:          repoView,
		Collaborators: []*CollaboratorView{},
		Section:       "danger",
		ActiveNav:     "",
	}

	var buf bytes.Buffer
	err := templates["repo_settings"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Repo settings danger template error: %v", err)
	}
}

// TestNotificationsTemplate tests the notifications page template.
func TestNotificationsTemplate(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:    1,
		Login: "testuser",
	}

	notif := &notifications.Notification{
		ID:     "1",
		Unread: true,
		Subject: &notifications.Subject{
			Title: "New comment on issue",
			URL:   "/testuser/my-repo/issues/1",
			Type:  "Issue",
		},
		Repository: &notifications.Repository{
			ID:       1,
			FullName: "testuser/my-repo",
		},
		Reason:    "subscribed",
		UpdatedAt: time.Now(),
	}

	notifView := &NotificationView{
		Notification: notif,
		TimeAgo:      "5 minutes ago",
		TypeLabel:    "Issue",
	}

	data := NotificationsData{
		Title:         "Notifications",
		User:          user,
		Notifications: []*NotificationView{notifView},
		UnreadCount:   1,
		Filter:        "unread",
		ActiveNav:     "notifications",
	}

	var buf bytes.Buffer
	err := templates["notifications"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Notifications template error: %v", err)
	}
}

// TestNotificationsTemplateEmpty tests empty notifications.
func TestNotificationsTemplateEmpty(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:    1,
		Login: "testuser",
	}

	data := NotificationsData{
		Title:         "Notifications",
		User:          user,
		Notifications: []*NotificationView{},
		UnreadCount:   0,
		Filter:        "unread",
		ActiveNav:     "notifications",
	}

	var buf bytes.Buffer
	err := templates["notifications"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Notifications empty template error: %v", err)
	}
}

// TestNotificationsTemplateAll tests all notifications filter.
func TestNotificationsTemplateAll(t *testing.T) {
	templates := loadTemplates(t)

	user := &users.User{
		ID:    1,
		Login: "testuser",
	}

	readNotif := &notifications.Notification{
		ID:     "1",
		Unread: false,
		Subject: &notifications.Subject{
			Title: "Old notification",
			URL:   "/testuser/my-repo/issues/1",
			Type:  "PullRequest",
		},
		Repository: &notifications.Repository{
			ID:       1,
			FullName: "testuser/my-repo",
		},
		UpdatedAt: time.Now().Add(-24 * time.Hour),
	}

	notifView := &NotificationView{
		Notification: readNotif,
		TimeAgo:      "1 day ago",
		TypeLabel:    "PullRequest",
	}

	data := NotificationsData{
		Title:         "Notifications",
		User:          user,
		Notifications: []*NotificationView{notifView},
		UnreadCount:   0,
		Filter:        "all",
		ActiveNav:     "notifications",
	}

	var buf bytes.Buffer
	err := templates["notifications"].Execute(&buf, data)
	if err != nil {
		t.Errorf("Notifications all template error: %v", err)
	}
}
