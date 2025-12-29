package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/go-mizu/blueprints/githome/feature/branches"
	"github.com/go-mizu/blueprints/githome/feature/comments"
	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/labels"
	"github.com/go-mizu/blueprints/githome/feature/milestones"
	"github.com/go-mizu/blueprints/githome/feature/notifications"
	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/pulls"
	"github.com/go-mizu/blueprints/githome/feature/releases"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/stars"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/feature/watches"
)

// Breadcrumb represents a navigation breadcrumb.
type Breadcrumb struct {
	Label string
	URL   string
}

// NavItem represents a navigation menu item.
type NavItem struct {
	Label    string
	URL      string
	Icon     string
	Active   bool
	Count    int
	External bool
}

// RepoTab represents a tab on the repository page.
type RepoTab struct {
	Name   string
	URL    string
	Icon   string
	Count  int
	Active bool
}

// LoginData holds data for the login page.
type LoginData struct {
	Title    string
	Error    string
	ReturnTo string
}

// RegisterData holds data for the registration page.
type RegisterData struct {
	Title string
	Error string
}

// HomeData holds data for the authenticated dashboard.
type HomeData struct {
	Title         string
	User          *users.User
	Repositories  []*repos.Repository
	StarredRepos  []*repos.Repository
	Organizations []*orgs.OrgSimple
	Notifications []*notifications.Notification
	UnreadCount   int
	ActiveNav     string
}

// ExploreData holds data for the explore page.
type ExploreData struct {
	Title         string
	User          *users.User
	Repositories  []*repos.Repository
	TrendingRepos []*repos.Repository
	Query         string
	Language      string
	Sort          string
	Page          int
	TotalCount    int
	HasNext       bool
	UnreadCount   int
	ActiveNav     string
}

// UserProfileData holds data for user profile page.
type UserProfileData struct {
	Title            string
	User             *users.User
	ProfileUser      *users.User
	Repositories     []*repos.Repository
	PinnedRepos      []*repos.Repository
	Organizations    []*orgs.OrgSimple
	IsOwnProfile     bool
	IsFollowing      bool
	ContributionData string
	ActiveTab        string
	Breadcrumbs      []Breadcrumb
	UnreadCount      int
	ActiveNav        string
}

// RepoView wraps a repository with navigation context.
type RepoView struct {
	*repos.Repository
	Tabs       []RepoTab
	ActiveTab  string
	CanPush    bool
	CanAdmin   bool
	IsStarred  bool
	IsWatching bool
}

// TreeEntry represents a file/directory in repo.
type TreeEntry struct {
	Name              string
	Path              string
	Type              string // file, dir, submodule, symlink
	Size              int64
	SHA               string
	LastCommitSHA     string
	LastCommitMessage string
	LastCommitAuthor  string
	LastCommitDate    string // formatted as "2 days ago"
}

// CommitView represents a commit for display
type CommitView struct {
	SHA             string
	ShortSHA        string
	Message         string
	MessageTitle    string
	Author          string
	AuthorEmail     string
	Date            string
	TimeAgo         string
}

// RepoHomeData holds data for repository home page.
type RepoHomeData struct {
	Title         string
	User          *users.User
	Repo          *RepoView
	Readme        template.HTML
	Tree          []*TreeEntry
	Branches      []*branches.Branch
	Releases      []*releases.Release
	CurrentBranch string
	CurrentPath   string
	License       string
	Languages     map[string]int
	Contributors  []*repos.Contributor
	Breadcrumbs   []Breadcrumb
	UnreadCount   int
	ActiveNav     string
	LatestCommit  *CommitView
	CommitCount   int
}

// RepoCodeData holds data for file browser.
type RepoCodeData struct {
	Title         string
	User          *users.User
	Repo          *RepoView
	Tree          []*TreeEntry
	FileContent   string
	FilePath      string
	FileName      string
	IsFile        bool
	IsBinary      bool
	IsMarkdown    bool
	IsImage       bool
	MarkdownHTML  template.HTML
	Language      string
	LineCount     int
	FileSize      int64
	FileSizeHuman string
	CurrentBranch string
	CurrentPath   string
	Branches      []*branches.Branch
	Breadcrumbs   []Breadcrumb
	UnreadCount   int
	ActiveNav     string
}

// BlameLineView represents a line with blame info for display
type BlameLineView struct {
	LineNumber int
	Content    string
	CommitSHA  string
	ShortSHA   string
	Author     string
	TimeAgo    string
}

// RepoBlameData holds data for blame view.
type RepoBlameData struct {
	Title         string
	User          *users.User
	Repo          *RepoView
	Lines         []*BlameLineView
	FilePath      string
	FileName      string
	LineCount     int
	CurrentBranch string
	Branches      []*branches.Branch
	Breadcrumbs   []Breadcrumb
	UnreadCount   int
	ActiveNav     string
}

// LabelView wraps a label with contrast color.
type LabelView struct {
	*labels.Label
	TextColor string
}

// IssueView wraps an issue with display context.
type IssueView struct {
	*issues.Issue
	TimeAgo    string
	LabelViews []*LabelView
}

// RepoIssuesData holds data for issues list.
type RepoIssuesData struct {
	Title         string
	User          *users.User
	Repo          *RepoView
	Issues        []*IssueView
	OpenCount     int
	ClosedCount   int
	Labels        []*labels.Label
	Milestones    []*milestones.Milestone
	Assignees     []*users.SimpleUser
	CurrentState  string
	CurrentLabel  string
	CurrentSort   string
	Page          int
	HasNext       bool
	Breadcrumbs   []Breadcrumb
	UnreadCount   int
	ActiveNav     string
}

// CommentView wraps a comment with author info.
type CommentView struct {
	*comments.IssueComment
	Author   *users.SimpleUser
	TimeAgo  string
	IsEdited bool
}

// TimelineEvent represents an issue timeline event.
type TimelineEvent struct {
	ID      int64
	Type    string
	Actor   *users.SimpleUser
	TimeAgo string
	Data    map[string]interface{}
}

// IssueDetailData holds data for single issue view.
type IssueDetailData struct {
	Title       string
	User        *users.User
	Repo        *RepoView
	Issue       *IssueView
	Comments    []*CommentView
	Timeline    []*TimelineEvent
	Labels      []*labels.Label
	Milestones  []*milestones.Milestone
	Assignees   []*users.SimpleUser
	CanEdit     bool
	CanClose    bool
	Breadcrumbs []Breadcrumb
	UnreadCount int
	ActiveNav   string
}

// NewIssueData holds data for create issue form.
type NewIssueData struct {
	Title       string
	User        *users.User
	Repo        *RepoView
	Labels      []*labels.Label
	Milestones  []*milestones.Milestone
	Assignees   []*users.SimpleUser
	Error       string
	Breadcrumbs []Breadcrumb
	UnreadCount int
	ActiveNav   string
}

// PullView wraps a PR with display context.
type PullView struct {
	*pulls.PullRequest
	TimeAgo     string
	LabelViews  []*LabelView
	StatusIcon  string
	StatusColor string
}

// RepoPullsData holds data for PR list.
type RepoPullsData struct {
	Title        string
	User         *users.User
	Repo         *RepoView
	PullRequests []*PullView
	OpenCount    int
	ClosedCount  int
	Labels       []*labels.Label
	CurrentState string
	CurrentSort  string
	Page         int
	HasNext      bool
	Breadcrumbs  []Breadcrumb
	UnreadCount  int
	ActiveNav    string
}

// CheckStatus represents a CI check.
type CheckStatus struct {
	Name        string
	State       string
	TargetURL   string
	Description string
}

// PullDetailData holds data for single PR view.
type PullDetailData struct {
	Title          string
	User           *users.User
	Repo           *RepoView
	Pull           *PullView
	Commits        []*pulls.Commit
	Files          []*pulls.PRFile
	Reviews        []*pulls.Review
	Comments       []*CommentView
	Timeline       []*TimelineEvent
	CanMerge       bool
	MergeableState string
	Checks         []*CheckStatus
	Breadcrumbs    []Breadcrumb
	UnreadCount    int
	ActiveNav      string
}

// NewRepoData holds data for create repository form.
type NewRepoData struct {
	Title              string
	User               *users.User
	Organizations      []*orgs.OrgSimple
	Licenses           []License
	GitignoreTemplates []string
	Error              string
	UnreadCount        int
	ActiveNav          string
}

// License represents a license template.
type License struct {
	Key  string
	Name string
}

// CollaboratorView wraps a collaborator.
type CollaboratorView struct {
	*users.SimpleUser
	Permission string
}

// RepoSettingsData holds data for repository settings.
type RepoSettingsData struct {
	Title         string
	User          *users.User
	Repo          *RepoView
	Collaborators []*CollaboratorView
	Section       string
	Error         string
	Success       string
	Breadcrumbs   []Breadcrumb
	UnreadCount   int
	ActiveNav     string
}

// NotificationView wraps a notification.
type NotificationView struct {
	*notifications.Notification
	TimeAgo   string
	Icon      string
	TypeLabel string
}

// NotificationsData holds data for notifications page.
type NotificationsData struct {
	Title         string
	User          *users.User
	Notifications []*NotificationView
	UnreadCount   int
	Filter        string
	ActiveNav     string
}

// Page handles page rendering.
type Page struct {
	templates     map[string]*template.Template
	users         users.API
	repos         repos.API
	issues        issues.API
	pulls         pulls.API
	comments      comments.API
	orgs          orgs.API
	notifications notifications.API
	stars         stars.API
	watches       watches.API
	branches      branches.API
	releases      releases.API
	labels        labels.API
	milestones    milestones.API
	getUserID     func(*mizu.Ctx) int64
}

// NewPage creates a new Page handler.
func NewPage(
	templates map[string]*template.Template,
	usersAPI users.API,
	reposAPI repos.API,
	issuesAPI issues.API,
	pullsAPI pulls.API,
	commentsAPI comments.API,
	orgsAPI orgs.API,
	notificationsAPI notifications.API,
	starsAPI stars.API,
	watchesAPI watches.API,
	branchesAPI branches.API,
	releasesAPI releases.API,
	labelsAPI labels.API,
	milestonesAPI milestones.API,
	getUserID func(*mizu.Ctx) int64,
) *Page {
	return &Page{
		templates:     templates,
		users:         usersAPI,
		repos:         reposAPI,
		issues:        issuesAPI,
		pulls:         pullsAPI,
		comments:      commentsAPI,
		orgs:          orgsAPI,
		notifications: notificationsAPI,
		stars:         starsAPI,
		watches:       watchesAPI,
		branches:      branchesAPI,
		releases:      releasesAPI,
		labels:        labelsAPI,
		milestones:    milestonesAPI,
		getUserID:     getUserID,
	}
}

// render is a generic template renderer.
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
	if userID != 0 {
		http.Redirect(c.Writer(), c.Request(), "/", http.StatusFound)
		return nil
	}
	return render(h, c, "login", LoginData{
		Title:    "Sign in",
		ReturnTo: c.Query("return_to"),
	})
}

// Register renders the registration page.
func (h *Page) Register(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID != 0 {
		http.Redirect(c.Writer(), c.Request(), "/", http.StatusFound)
		return nil
	}
	return render(h, c, "register", RegisterData{
		Title: "Create your account",
	})
}

// Home renders the dashboard.
func (h *Page) Home(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == 0 {
		return h.Landing(c)
	}

	ctx := c.Request().Context()
	user, _ := h.users.GetByID(ctx, userID)

	repoList, _ := h.repos.ListForAuthenticatedUser(ctx, userID, &repos.ListOpts{PerPage: 10})

	var starredRepos []*repos.Repository
	if user != nil {
		starRepos, _ := h.stars.ListForUser(ctx, user.Login, &stars.ListOpts{PerPage: 5})
		// Convert stars.Repository to repos.Repository
		for _, sr := range starRepos {
			starredRepos = append(starredRepos, &repos.Repository{
				ID:              sr.ID,
				Name:            sr.Name,
				FullName:        sr.FullName,
				Description:     sr.Description,
				Private:         sr.Private,
				HTMLURL:         sr.HTMLURL,
			})
		}
	}

	notifs, _ := h.notifications.List(ctx, userID, &notifications.ListOpts{PerPage: 10})
	var unreadCount int
	for _, n := range notifs {
		if n.Unread {
			unreadCount++
		}
	}

	var orgList []*orgs.OrgSimple
	if user != nil {
		orgList, _ = h.orgs.ListForUser(ctx, user.Login, &orgs.ListOpts{PerPage: 10})
	}

	return render(h, c, "home", HomeData{
		Title:         "Dashboard",
		User:          user,
		Repositories:  repoList,
		StarredRepos:  starredRepos,
		Organizations: orgList,
		Notifications: notifs,
		UnreadCount:   unreadCount,
		ActiveNav:     "home",
	})
}

// Landing renders the landing page for logged out users.
func (h *Page) Landing(c *mizu.Ctx) error {
	return render(h, c, "explore", ExploreData{
		Title:     "GitHome",
		ActiveNav: "explore",
	})
}

// Explore renders the explore page.
func (h *Page) Explore(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	userID := h.getUserID(c)

	var user *users.User
	if userID != 0 {
		user, _ = h.users.GetByID(ctx, userID)
	}

	query := c.Query("q")
	language := c.Query("language")
	sort := c.Query("sort")
	if sort == "" {
		sort = "stars"
	}

	repoList, _ := h.repos.ListForUser(ctx, "", &repos.ListOpts{
		PerPage:   30,
		Sort:      sort,
		Direction: "desc",
	})

	return render(h, c, "explore", ExploreData{
		Title:        "Explore repositories",
		User:         user,
		Repositories: repoList,
		Query:        query,
		Language:     language,
		Sort:         sort,
		Page:         1,
		ActiveNav:    "explore",
	})
}

// UserProfile renders the user profile page.
func (h *Page) UserProfile(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	username := c.Param("username")
	userID := h.getUserID(c)

	profileUser, err := h.users.GetByLogin(ctx, username)
	if err != nil {
		return c.Text(http.StatusNotFound, "User not found")
	}

	var user *users.User
	var isFollowing bool
	if userID != 0 {
		user, _ = h.users.GetByID(ctx, userID)
		isFollowing, _ = h.users.IsFollowing(ctx, user.Login, profileUser.Login)
	}

	repoList, _ := h.repos.ListForUser(ctx, username, &repos.ListOpts{PerPage: 20, Sort: "updated"})
	orgList, _ := h.orgs.ListForUser(ctx, username, &orgs.ListOpts{PerPage: 10})

	return render(h, c, "user_profile", UserProfileData{
		Title:        profileUser.Login,
		User:         user,
		ProfileUser:  profileUser,
		Repositories: repoList,
		Organizations: orgList,
		IsOwnProfile: userID == profileUser.ID,
		IsFollowing:  isFollowing,
		ActiveTab:    "overview",
		ActiveNav:    "",
	})
}

// RepoHome renders the repository home page.
func (h *Page) RepoHome(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	ctx := c.Request().Context()
	userID := h.getUserID(c)

	repo, err := h.repos.Get(ctx, owner, repoName)
	if err != nil {
		return c.Text(http.StatusNotFound, "Repository not found")
	}

	repoView := h.buildRepoView(ctx, repo, userID, "code")

	branchList, _ := h.branches.List(ctx, owner, repoName, nil)
	languages, _ := h.repos.ListLanguages(ctx, owner, repoName)

	var user *users.User
	if userID != 0 {
		user, _ = h.users.GetByID(ctx, userID)
	}

	// Get root tree entries - try default branch first, fallback to common branches
	var tree []*TreeEntry
	branchesToTry := []string{repo.DefaultBranch, "master", "main"}
	var usedBranch string
	for _, branch := range branchesToTry {
		if branch == "" {
			continue
		}
		// Use ListTreeEntriesWithCommits to get last commit info per file
		entries, err := h.repos.ListTreeEntriesWithCommits(ctx, owner, repoName, "", branch)
		if err == nil && entries != nil && len(entries) > 0 {
			tree = make([]*TreeEntry, len(entries))
			for i, e := range entries {
				tree[i] = &TreeEntry{
					Name:              e.Name,
					Path:              e.Path,
					Type:              e.Type,
					Size:              e.Size,
					SHA:               e.SHA,
					LastCommitSHA:     e.LastCommitSHA,
					LastCommitMessage: e.LastCommitMessage,
					LastCommitAuthor:  e.LastCommitAuthor,
					LastCommitDate:    formatTimeAgo(e.LastCommitDate),
				}
			}
			usedBranch = branch
			break
		}
	}
	// Update current branch if we found one
	currentBranch := repo.DefaultBranch
	if usedBranch != "" {
		currentBranch = usedBranch
	}

	// Get latest commit and commit count
	var latestCommit *CommitView
	var commitCount int
	if commit, err := h.repos.GetLatestCommit(ctx, owner, repoName, currentBranch); err == nil && commit != nil {
		latestCommit = &CommitView{
			SHA:          commit.SHA,
			ShortSHA:     shortSHA(commit.SHA),
			Message:      commit.Message,
			MessageTitle: firstLine(commit.Message),
			Author:       commit.Author.Name,
			AuthorEmail:  commit.Author.Email,
			Date:         commit.Author.Date.Format("Jan 2, 2006"),
			TimeAgo:      formatTimeAgo(commit.Author.Date),
		}
	}
	commitCount, _ = h.repos.GetCommitCount(ctx, owner, repoName, currentBranch)

	// Fetch and render README using the correct branch
	var readmeHTML template.HTML
	readme, _ := h.repos.GetReadme(ctx, owner, repoName, currentBranch)
	if readme != nil && readme.Content != "" {
		// Decode base64 content
		decodedContent := decodeBase64Content(readme.Content)
		// Render markdown to HTML
		rendered := renderMarkdown([]byte(decodedContent))
		readmeHTML = template.HTML(rendered)
	}

	return render(h, c, "repo_home", RepoHomeData{
		Title:         repo.FullName,
		User:          user,
		Repo:          repoView,
		Readme:        readmeHTML,
		Tree:          tree,
		Branches:      branchList,
		CurrentBranch: currentBranch,
		Languages:     languages,
		ActiveNav:     "",
		LatestCommit:  latestCommit,
		CommitCount:   commitCount,
	})
}

// RepoIssues renders the issues list page.
func (h *Page) RepoIssues(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	ctx := c.Request().Context()
	userID := h.getUserID(c)

	repo, err := h.repos.Get(ctx, owner, repoName)
	if err != nil {
		return c.Text(http.StatusNotFound, "Repository not found")
	}

	repoView := h.buildRepoView(ctx, repo, userID, "issues")

	state := c.Query("state")
	if state == "" {
		state = "open"
	}

	issueList, _ := h.issues.ListForRepo(ctx, owner, repoName, &issues.ListOpts{
		State:   state,
		PerPage: 30,
	})

	openCount := 0
	closedCount := 0
	if state == "open" {
		openCount = len(issueList)
	} else {
		closedCount = len(issueList)
	}

	issueViews := make([]*IssueView, len(issueList))
	for i, issue := range issueList {
		issueViews[i] = &IssueView{
			Issue:   issue,
			TimeAgo: timeAgo(issue.CreatedAt),
		}
	}

	labelList, _ := h.labels.List(ctx, owner, repoName, nil)
	milestoneList, _ := h.milestones.List(ctx, owner, repoName, nil)

	var user *users.User
	if userID != 0 {
		user, _ = h.users.GetByID(ctx, userID)
	}

	return render(h, c, "repo_issues", RepoIssuesData{
		Title:        "Issues",
		User:         user,
		Repo:         repoView,
		Issues:       issueViews,
		OpenCount:    openCount,
		ClosedCount:  closedCount,
		Labels:       labelList,
		Milestones:   milestoneList,
		CurrentState: state,
		Page:         1,
		Breadcrumbs: []Breadcrumb{
			{Label: repo.FullName, URL: "/" + repo.FullName},
			{Label: "Issues", URL: ""},
		},
		ActiveNav: "",
	})
}

// IssueView renders the single issue view.
func (h *Page) IssueDetail(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	number := parseInt(c.Param("number"))

	ctx := c.Request().Context()
	userID := h.getUserID(c)

	repo, err := h.repos.Get(ctx, owner, repoName)
	if err != nil {
		return c.Text(http.StatusNotFound, "Repository not found")
	}

	issue, err := h.issues.Get(ctx, owner, repoName, number)
	if err != nil {
		return c.Text(http.StatusNotFound, "Issue not found")
	}

	repoView := h.buildRepoView(ctx, repo, userID, "issues")

	issueView := &IssueView{
		Issue:   issue,
		TimeAgo: timeAgo(issue.CreatedAt),
	}

	commentList, _ := h.comments.ListForIssue(ctx, owner, repoName, number, nil)
	commentViews := make([]*CommentView, len(commentList))
	for i, comment := range commentList {
		commentViews[i] = &CommentView{
			IssueComment: comment,
			TimeAgo:      timeAgo(comment.CreatedAt),
		}
	}

	labelList, _ := h.labels.List(ctx, owner, repoName, nil)
	milestoneList, _ := h.milestones.List(ctx, owner, repoName, nil)

	var user *users.User
	canEdit := false
	canClose := false
	if userID != 0 {
		user, _ = h.users.GetByID(ctx, userID)
		canEdit = userID == issue.CreatorID || repoView.CanAdmin
		canClose = canEdit || repoView.CanPush
	}

	return render(h, c, "issue_view", IssueDetailData{
		Title:      fmt.Sprintf("%s #%d", issue.Title, issue.Number),
		User:       user,
		Repo:       repoView,
		Issue:      issueView,
		Comments:   commentViews,
		Labels:     labelList,
		Milestones: milestoneList,
		CanEdit:    canEdit,
		CanClose:   canClose,
		Breadcrumbs: []Breadcrumb{
			{Label: repo.FullName, URL: "/" + repo.FullName},
			{Label: "Issues", URL: "/" + repo.FullName + "/issues"},
			{Label: fmt.Sprintf("#%d", issue.Number), URL: ""},
		},
		ActiveNav: "",
	})
}

// NewIssue renders the create issue form.
func (h *Page) NewIssue(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	ctx := c.Request().Context()
	userID := h.getUserID(c)

	if userID == 0 {
		http.Redirect(c.Writer(), c.Request(), "/login?return_to="+c.Request().URL.Path, http.StatusFound)
		return nil
	}

	repo, err := h.repos.Get(ctx, owner, repoName)
	if err != nil {
		return c.Text(http.StatusNotFound, "Repository not found")
	}

	repoView := h.buildRepoView(ctx, repo, userID, "issues")
	user, _ := h.users.GetByID(ctx, userID)

	labelList, _ := h.labels.List(ctx, owner, repoName, nil)
	milestoneList, _ := h.milestones.List(ctx, owner, repoName, nil)

	return render(h, c, "new_issue", NewIssueData{
		Title:      "New Issue",
		User:       user,
		Repo:       repoView,
		Labels:     labelList,
		Milestones: milestoneList,
		Breadcrumbs: []Breadcrumb{
			{Label: repo.FullName, URL: "/" + repo.FullName},
			{Label: "Issues", URL: "/" + repo.FullName + "/issues"},
			{Label: "New", URL: ""},
		},
		ActiveNav: "",
	})
}

// NewRepo renders the create repository form.
func (h *Page) NewRepo(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	userID := h.getUserID(c)

	if userID == 0 {
		http.Redirect(c.Writer(), c.Request(), "/login?return_to=/new", http.StatusFound)
		return nil
	}

	user, _ := h.users.GetByID(ctx, userID)
	orgList, _ := h.orgs.ListForUser(ctx, user.Login, &orgs.ListOpts{PerPage: 50})

	return render(h, c, "new_repo", NewRepoData{
		Title:         "Create a new repository",
		User:          user,
		Organizations: orgList,
		Licenses: []License{
			{Key: "mit", Name: "MIT License"},
			{Key: "apache-2.0", Name: "Apache License 2.0"},
			{Key: "gpl-3.0", Name: "GNU GPLv3"},
			{Key: "bsd-3-clause", Name: "BSD 3-Clause"},
		},
		GitignoreTemplates: []string{"Go", "Python", "Node", "Java", "Ruby"},
		ActiveNav:          "new",
	})
}

// RepoSettings renders the repository settings page.
func (h *Page) RepoSettings(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	ctx := c.Request().Context()
	userID := h.getUserID(c)

	if userID == 0 {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	repo, err := h.repos.Get(ctx, owner, repoName)
	if err != nil {
		return c.Text(http.StatusNotFound, "Repository not found")
	}

	repoView := h.buildRepoView(ctx, repo, userID, "settings")
	if !repoView.CanAdmin {
		return c.Text(http.StatusForbidden, "Access denied")
	}

	user, _ := h.users.GetByID(ctx, userID)

	section := c.Query("section")
	if section == "" {
		section = "general"
	}

	return render(h, c, "repo_settings", RepoSettingsData{
		Title:   "Settings",
		User:    user,
		Repo:    repoView,
		Section: section,
		Breadcrumbs: []Breadcrumb{
			{Label: repo.FullName, URL: "/" + repo.FullName},
			{Label: "Settings", URL: ""},
		},
		ActiveNav: "",
	})
}

// Notifications renders the notifications page.
func (h *Page) Notifications(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	userID := h.getUserID(c)

	if userID == 0 {
		http.Redirect(c.Writer(), c.Request(), "/login?return_to=/notifications", http.StatusFound)
		return nil
	}

	user, _ := h.users.GetByID(ctx, userID)

	filter := c.Query("filter")
	if filter == "" {
		filter = "unread"
	}

	opts := &notifications.ListOpts{PerPage: 50}
	if filter != "all" {
		opts.All = false
	}

	notifList, _ := h.notifications.List(ctx, userID, opts)

	notifViews := make([]*NotificationView, len(notifList))
	var unreadCount int
	for i, n := range notifList {
		if n.Unread {
			unreadCount++
		}
		notifViews[i] = &NotificationView{
			Notification: n,
			TimeAgo:      timeAgo(n.UpdatedAt),
			TypeLabel:    n.Subject.Type,
		}
	}

	return render(h, c, "notifications", NotificationsData{
		Title:         "Notifications",
		User:          user,
		Notifications: notifViews,
		UnreadCount:   unreadCount,
		Filter:        filter,
		ActiveNav:     "notifications",
	})
}

// RepoTree renders the directory tree view.
func (h *Page) RepoTree(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	ref := c.Param("ref")
	path := c.Param("path")

	ctx := c.Request().Context()
	userID := h.getUserID(c)

	repo, err := h.repos.Get(ctx, owner, repoName)
	if err != nil {
		return c.Text(http.StatusNotFound, "Repository not found")
	}

	// If no ref, use default branch
	if ref == "" {
		ref = repo.DefaultBranch
	}

	// Get directory tree entries with last commit info
	var tree []*TreeEntry
	entries, err := h.repos.ListTreeEntriesWithCommits(ctx, owner, repoName, path, ref)
	if err != nil {
		return c.Text(http.StatusNotFound, "Path not found")
	}

	tree = make([]*TreeEntry, len(entries))
	for i, e := range entries {
		tree[i] = &TreeEntry{
			Name:              e.Name,
			Path:              e.Path,
			Type:              e.Type,
			Size:              e.Size,
			SHA:               e.SHA,
			LastCommitSHA:     e.LastCommitSHA,
			LastCommitMessage: e.LastCommitMessage,
			LastCommitAuthor:  e.LastCommitAuthor,
			LastCommitDate:    formatTimeAgo(e.LastCommitDate),
		}
	}

	repoView := h.buildRepoView(ctx, repo, userID, "code")
	branchList, _ := h.branches.List(ctx, owner, repoName, nil)

	var user *users.User
	if userID != 0 {
		user, _ = h.users.GetByID(ctx, userID)
	}

	// Build breadcrumbs
	breadcrumbs := []Breadcrumb{
		{Label: repo.FullName, URL: "/" + repo.FullName},
	}
	if path != "" {
		parts := splitPath(path)
		currentPath := ""
		for i, part := range parts {
			currentPath += part
			url := fmt.Sprintf("/%s/tree/%s/%s", repo.FullName, ref, currentPath)
			if i == len(parts)-1 {
				url = ""
			}
			breadcrumbs = append(breadcrumbs, Breadcrumb{Label: part, URL: url})
			currentPath += "/"
		}
	}

	return render(h, c, "repo_code", RepoCodeData{
		Title:         repo.FullName,
		User:          user,
		Repo:          repoView,
		Tree:          tree,
		CurrentBranch: ref,
		CurrentPath:   path,
		Branches:      branchList,
		Breadcrumbs:   breadcrumbs,
		ActiveNav:     "",
	})
}

// RepoBlob renders the file content view.
func (h *Page) RepoBlob(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	ref := c.Param("ref")
	path := c.Param("path")

	ctx := c.Request().Context()
	userID := h.getUserID(c)

	repo, err := h.repos.Get(ctx, owner, repoName)
	if err != nil {
		return c.Text(http.StatusNotFound, "Repository not found")
	}

	// Get file contents
	content, err := h.repos.GetContents(ctx, owner, repoName, path, ref)
	if err != nil {
		return c.Text(http.StatusNotFound, "File not found")
	}

	if content.Type != "file" {
		// Redirect to tree view if it's a directory
		return c.Redirect(http.StatusFound, fmt.Sprintf("/%s/tree/%s/%s", repo.FullName, ref, path))
	}

	repoView := h.buildRepoView(ctx, repo, userID, "code")
	branchList, _ := h.branches.List(ctx, owner, repoName, nil)

	var user *users.User
	if userID != 0 {
		user, _ = h.users.GetByID(ctx, userID)
	}

	// Decode base64 content
	decodedContent := decodeBase64Content(content.Content)

	// Detect language from file extension
	language := detectLanguage(path)

	// Check if this is an image file
	isImage := isImageFile(path)

	// Check if binary (check decoded content)
	isBinary := isBinaryContent(decodedContent)

	// Count lines (on decoded content)
	lineCount := countLines(decodedContent)

	// Check if markdown and render if so
	isMarkdown := language == "markdown"
	var markdownHTML template.HTML
	if isMarkdown && !isBinary {
		rendered := renderMarkdown([]byte(decodedContent))
		markdownHTML = template.HTML(rendered)
	}

	// Build breadcrumbs
	breadcrumbs := []Breadcrumb{
		{Label: repo.FullName, URL: "/" + repo.FullName},
	}
	parts := splitPath(path)
	currentPath := ""
	for i, part := range parts {
		currentPath += part
		url := fmt.Sprintf("/%s/tree/%s/%s", repo.FullName, ref, currentPath)
		if i == len(parts)-1 {
			url = "" // Current file, no link
		}
		breadcrumbs = append(breadcrumbs, Breadcrumb{Label: part, URL: url})
		currentPath += "/"
	}

	return render(h, c, "repo_blob", RepoCodeData{
		Title:         content.Name + " - " + repo.FullName,
		User:          user,
		Repo:          repoView,
		FileContent:   decodedContent,
		FilePath:      path,
		FileName:      content.Name,
		IsFile:        true,
		IsBinary:      isBinary,
		IsMarkdown:    isMarkdown,
		IsImage:       isImage,
		MarkdownHTML:  markdownHTML,
		Language:      language,
		LineCount:     lineCount,
		FileSize:      int64(content.Size),
		FileSizeHuman: humanizeBytes(int64(content.Size)),
		CurrentBranch: ref,
		Branches:      branchList,
		Breadcrumbs:   breadcrumbs,
		ActiveNav:     "",
	})
}

// RepoRaw serves raw file content.
func (h *Page) RepoRaw(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	ref := c.Param("ref")
	path := c.Param("path")

	ctx := c.Request().Context()

	// Get file contents
	content, err := h.repos.GetContents(ctx, owner, repoName, path, ref)
	if err != nil {
		return c.Text(http.StatusNotFound, "File not found")
	}

	if content.Type != "file" {
		return c.Text(http.StatusNotFound, "Not a file")
	}

	// Decode base64 content
	decodedContent := decodeBase64Content(content.Content)

	// Detect content type
	contentType := detectContentType(path, []byte(decodedContent))

	c.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", content.Name))
	c.Header().Set("X-Content-Type-Options", "nosniff")

	return c.Bytes(http.StatusOK, []byte(decodedContent), contentType)
}

// RepoBlame renders the blame view for a file.
func (h *Page) RepoBlame(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	ref := c.Param("ref")
	path := c.Param("path")

	ctx := c.Request().Context()
	userID := h.getUserID(c)

	repo, err := h.repos.Get(ctx, owner, repoName)
	if err != nil {
		return c.Text(http.StatusNotFound, "Repository not found")
	}

	repoView := h.buildRepoView(ctx, repo, userID, "code")
	branchList, _ := h.branches.List(ctx, owner, repoName, nil)

	var user *users.User
	if userID != 0 {
		user, _ = h.users.GetByID(ctx, userID)
	}

	// Get blame info
	blameResult, err := h.repos.GetBlame(ctx, owner, repoName, ref, path)
	if err != nil {
		return c.Text(http.StatusNotFound, "Could not generate blame for this file")
	}

	// Convert to view models
	lines := make([]*BlameLineView, len(blameResult.Lines))
	for i, line := range blameResult.Lines {
		lines[i] = &BlameLineView{
			LineNumber: line.LineNumber,
			Content:    line.Content,
			CommitSHA:  line.CommitSHA,
			ShortSHA:   shortSHA(line.CommitSHA),
			Author:     line.Author,
			TimeAgo:    formatTimeAgo(line.Date),
		}
	}

	// Build breadcrumbs
	fileName := filepath.Base(path)
	breadcrumbs := []Breadcrumb{
		{Label: repo.FullName, URL: "/" + repo.FullName},
	}
	parts := splitPath(path)
	currentPath := ""
	for i, part := range parts {
		currentPath += part
		url := fmt.Sprintf("/%s/tree/%s/%s", repo.FullName, ref, currentPath)
		if i == len(parts)-1 {
			url = "" // Current file, no link
		}
		breadcrumbs = append(breadcrumbs, Breadcrumb{Label: part, URL: url})
		currentPath += "/"
	}

	return render(h, c, "repo_blame", RepoBlameData{
		Title:         "Blame: " + fileName + " - " + repo.FullName,
		User:          user,
		Repo:          repoView,
		Lines:         lines,
		FilePath:      path,
		FileName:      fileName,
		LineCount:     len(lines),
		CurrentBranch: ref,
		Branches:      branchList,
		Breadcrumbs:   breadcrumbs,
		ActiveNav:     "",
	})
}

// detectContentType detects the MIME type from file extension and content
func detectContentType(path string, content []byte) string {
	ext := strings.ToLower(filepath.Ext(path))

	// Common text file types
	textTypes := map[string]string{
		".go":         "text/plain; charset=utf-8",
		".js":         "application/javascript; charset=utf-8",
		".ts":         "application/typescript; charset=utf-8",
		".jsx":        "text/jsx; charset=utf-8",
		".tsx":        "text/tsx; charset=utf-8",
		".py":         "text/x-python; charset=utf-8",
		".rb":         "text/x-ruby; charset=utf-8",
		".rs":         "text/x-rust; charset=utf-8",
		".java":       "text/x-java; charset=utf-8",
		".c":          "text/x-c; charset=utf-8",
		".h":          "text/x-c; charset=utf-8",
		".cpp":        "text/x-c++; charset=utf-8",
		".cc":         "text/x-c++; charset=utf-8",
		".hpp":        "text/x-c++; charset=utf-8",
		".cs":         "text/x-csharp; charset=utf-8",
		".php":        "text/x-php; charset=utf-8",
		".swift":      "text/x-swift; charset=utf-8",
		".kt":         "text/x-kotlin; charset=utf-8",
		".scala":      "text/x-scala; charset=utf-8",
		".html":       "text/html; charset=utf-8",
		".htm":        "text/html; charset=utf-8",
		".css":        "text/css; charset=utf-8",
		".scss":       "text/x-scss; charset=utf-8",
		".sass":       "text/x-sass; charset=utf-8",
		".json":       "application/json; charset=utf-8",
		".yaml":       "text/yaml; charset=utf-8",
		".yml":        "text/yaml; charset=utf-8",
		".xml":        "application/xml; charset=utf-8",
		".md":         "text/markdown; charset=utf-8",
		".markdown":   "text/markdown; charset=utf-8",
		".sql":        "text/x-sql; charset=utf-8",
		".sh":         "text/x-sh; charset=utf-8",
		".bash":       "text/x-sh; charset=utf-8",
		".dockerfile": "text/x-dockerfile; charset=utf-8",
		".txt":        "text/plain; charset=utf-8",
		".log":        "text/plain; charset=utf-8",
		".gitignore":  "text/plain; charset=utf-8",
		".env":        "text/plain; charset=utf-8",
		".mod":        "text/plain; charset=utf-8",
		".sum":        "text/plain; charset=utf-8",
		".toml":       "text/x-toml; charset=utf-8",
		".ini":        "text/plain; charset=utf-8",
		".cfg":        "text/plain; charset=utf-8",
		".conf":       "text/plain; charset=utf-8",
	}

	if contentType, ok := textTypes[ext]; ok {
		return contentType
	}

	// Image types
	imageTypes := map[string]string{
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",
		".webp": "image/webp",
	}

	if contentType, ok := imageTypes[ext]; ok {
		return contentType
	}

	// Default to octet-stream for unknown types
	return "application/octet-stream"
}

// splitPath splits a path into its components
func splitPath(path string) []string {
	if path == "" {
		return nil
	}
	var parts []string
	for _, p := range strings.Split(path, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// detectLanguage detects the programming language from file extension
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".jsx":
		return "javascript"
	case ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".rb":
		return "ruby"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".hpp":
		return "cpp"
	case ".cs":
		return "csharp"
	case ".php":
		return "php"
	case ".swift":
		return "swift"
	case ".kt":
		return "kotlin"
	case ".scala":
		return "scala"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	case ".scss", ".sass":
		return "scss"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".xml":
		return "xml"
	case ".md":
		return "markdown"
	case ".sql":
		return "sql"
	case ".sh", ".bash":
		return "bash"
	case ".dockerfile":
		return "dockerfile"
	default:
		return ""
	}
}

// countLines counts the number of lines in content
func countLines(content string) int {
	if content == "" {
		return 0
	}
	return strings.Count(content, "\n") + 1
}

// isBinaryContent checks if content appears to be binary
func isBinaryContent(content string) bool {
	// Check for null bytes or high ratio of non-printable characters
	for _, r := range content[:min(len(content), 8000)] {
		if r == 0 {
			return true
		}
	}
	return false
}

// min returns the smaller of two ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// buildRepoView creates a RepoView with tabs and permissions.
func (h *Page) buildRepoView(ctx context.Context, repo *repos.Repository, userID int64, activeTab string) *RepoView {
	view := &RepoView{
		Repository: repo,
		ActiveTab:  activeTab,
	}

	view.Tabs = []RepoTab{
		{Name: "Code", URL: fmt.Sprintf("/%s", repo.FullName), Icon: "code", Active: activeTab == "code"},
		{Name: "Issues", URL: fmt.Sprintf("/%s/issues", repo.FullName), Icon: "issue-opened", Count: repo.OpenIssuesCount, Active: activeTab == "issues"},
		{Name: "Pull requests", URL: fmt.Sprintf("/%s/pulls", repo.FullName), Icon: "git-pull-request", Active: activeTab == "pulls"},
	}

	if repo.HasWiki {
		view.Tabs = append(view.Tabs, RepoTab{Name: "Wiki", URL: fmt.Sprintf("/%s/wiki", repo.FullName), Icon: "book", Active: activeTab == "wiki"})
	}

	view.Tabs = append(view.Tabs, RepoTab{Name: "Settings", URL: fmt.Sprintf("/%s/settings", repo.FullName), Icon: "gear", Active: activeTab == "settings"})

	if userID != 0 {
		if repo.OwnerID == userID {
			view.CanPush = true
			view.CanAdmin = true
		}
		if repo.Owner != nil {
			view.IsStarred, _ = h.stars.IsStarred(ctx, userID, repo.Owner.Login, repo.Name)
			sub, err := h.watches.GetSubscription(ctx, userID, repo.Owner.Login, repo.Name)
			if err == nil && sub != nil {
				view.IsWatching = sub.Subscribed
			}
		}
	}

	return view
}

// timeAgo returns a human-readable time difference.
func timeAgo(t interface{}) string {
	// Simple implementation - can be enhanced
	return "recently"
}

// parseInt parses a string to int.
func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// renderMarkdown converts markdown content to HTML
func renderMarkdown(content []byte) string {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert(content, &buf); err != nil {
		return string(content)
	}
	return buf.String()
}

// decodeBase64Content decodes base64 content to string
func decodeBase64Content(content string) string {
	decoded, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return content
	}
	return string(decoded)
}

// humanizeBytes converts bytes to human-readable format
func humanizeBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatTimeAgo formats a time as a human-readable relative string
func formatTimeAgo(t interface{}) string {
	var when time.Time
	switch v := t.(type) {
	case time.Time:
		when = v
	default:
		return ""
	}

	if when.IsZero() {
		return ""
	}

	now := time.Now()
	diff := now.Sub(when)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 30*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if diff < 365*24*time.Hour {
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "last month"
		}
		return fmt.Sprintf("%d months ago", months)
	}
	years := int(diff.Hours() / 24 / 365)
	if years == 1 {
		return "last year"
	}
	return fmt.Sprintf("%d years ago", years)
}

// shortSHA returns the first 7 characters of a SHA
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// firstLine returns the first line of a string
func firstLine(s string) string {
	if idx := strings.Index(s, "\n"); idx >= 0 {
		return s[:idx]
	}
	return s
}

// isImageFile checks if a file path is an image file
func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	imageExtensions := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".gif":  true,
		".svg":  true,
		".webp": true,
		".ico":  true,
		".bmp":  true,
	}
	return imageExtensions[ext]
}
