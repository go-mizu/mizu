package handler

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/go-mizu/blueprints/githome/feature/branches"
	"github.com/go-mizu/blueprints/githome/feature/comments"
	"github.com/go-mizu/blueprints/githome/feature/commits"
	"github.com/go-mizu/blueprints/githome/feature/git"
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

// UserView represents a user for commit display
type UserView struct {
	Login     string
	Name      string
	Email     string
	AvatarURL string
	HTMLURL   string
}

// CommitViewItem represents a commit item in the list
type CommitViewItem struct {
	SHA          string
	ShortSHA     string
	Message      string
	MessageTitle string
	MessageBody  string
	Author       *UserView
	Committer    *UserView
	AuthorDate   time.Time
	TimeAgo      string
	IsSameAuthor bool
	TreeURL      string
	CommitURL    string
}

// CommitGroup represents commits grouped by date
type CommitGroup struct {
	Date    string
	Commits []*CommitViewItem
}

// RepoCommitsData holds data for commits list page
type RepoCommitsData struct {
	Title         string
	User          *users.User
	Repo          *RepoView
	CommitGroups  []*CommitGroup
	CurrentBranch string
	Branches      []*branches.Branch
	Page          int
	HasNext       bool
	HasPrev       bool
	Breadcrumbs   []Breadcrumb
	UnreadCount   int
	ActiveNav     string
}

// ParentCommit represents a parent commit reference
type ParentCommit struct {
	SHA      string
	ShortSHA string
	HTMLURL  string
}

// CommitViewDetail represents full commit details for single commit view
type CommitViewDetail struct {
	SHA          string
	ShortSHA     string
	Message      string
	MessageTitle string
	MessageBody  string
	Author       *UserView
	Committer    *UserView
	AuthorDate   time.Time
	TimeAgo      string
	IsSameAuthor bool
	Parents      []*ParentCommit
	TreeSHA      string
	TreeURL      string
	Verified     bool
	HTMLURL      string
}

// DiffLine represents a single line in a diff
type DiffLine struct {
	Type       string // context, addition, deletion, hunk
	OldLineNum int
	NewLineNum int
	Content    string
}

// FileChangeView represents a changed file in commit view
type FileChangeView struct {
	SHA              string
	Filename         string
	PreviousFilename string
	Status           string // added, removed, modified, renamed
	Additions        int
	Deletions        int
	Changes          int
	BlobURL          string
	RawURL           string
	Patch            string
	DiffLines        []*DiffLine
	IsBinary         bool
	TooLarge         bool
}

// StatsView represents commit statistics
type StatsView struct {
	FilesChanged int
	Additions    int
	Deletions    int
	Total        int
}

// CommitDetailData holds data for single commit view
type CommitDetailData struct {
	Title       string
	User        *users.User
	Repo        *RepoView
	Commit      *CommitViewDetail
	Files       []*FileChangeView
	Stats       *StatsView
	Branches    []*branches.Branch
	Tags        []string
	Breadcrumbs []Breadcrumb
	UnreadCount int
	ActiveNav   string
}

// ReleaseView represents a release for template rendering.
type ReleaseView struct {
	ID      int64
	TagName string
	Name    string
	TimeAgo string
}

// RepoHomeData holds data for repository home page.
type RepoHomeData struct {
	Title            string
	User             *users.User
	Repo             *RepoView
	Readme           template.HTML
	Tree             []*TreeEntry
	Branches         []*branches.Branch
	Releases         []*releases.Release
	CurrentBranch    string
	CurrentPath      string
	License          string
	Languages        map[string]int
	Contributors     []*repos.Contributor
	Breadcrumbs      []Breadcrumb
	UnreadCount      int
	ActiveNav        string
	LatestCommit     *CommitView
	CommitCount      int
	Host             string
	TagCount         int
	ReleaseCount     int
	ContributorCount int
	LatestRelease    *ReleaseView
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
	ParentPath    string // URL to go up one directory
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
	Language      string
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
	BodyHTML   template.HTML
}

// RepoIssuesData holds data for issues list.
type RepoIssuesData struct {
	Title            string
	User             *users.User
	Repo             *RepoView
	Issues           []*IssueView
	OpenCount        int
	ClosedCount      int
	Labels           []*labels.Label
	Milestones       []*milestones.Milestone
	Assignees        []*users.SimpleUser
	CurrentState     string
	CurrentLabel     string
	CurrentMilestone string
	CurrentSort      string
	Query            string
	Page             int
	TotalPages       int
	HasNext          bool
	HasPrev          bool
	Breadcrumbs      []Breadcrumb
	UnreadCount      int
	ActiveNav        string
}

// CommentView wraps a comment with author info.
type CommentView struct {
	*comments.IssueComment
	Author   *users.SimpleUser
	TimeAgo  string
	IsEdited bool
	BodyHTML template.HTML
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
	Title        string
	User         *users.User
	Repo         *RepoView
	Issue        *IssueView
	Comments     []*CommentView
	Timeline     []*TimelineEvent
	Labels       []*labels.Label
	Milestones   []*milestones.Milestone
	Assignees    []*users.SimpleUser
	Participants []*users.SimpleUser
	CanEdit      bool
	CanClose     bool
	Breadcrumbs  []Breadcrumb
	UnreadCount  int
	ActiveNav    string
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
	BodyHTML    template.HTML
}

// RepoPullsData holds data for PR list.
type RepoPullsData struct {
	Title            string
	User             *users.User
	Repo             *RepoView
	PullRequests     []*PullView
	OpenCount        int
	ClosedCount      int
	Labels           []*labels.Label
	Milestones       []*milestones.Milestone
	Assignees        []*users.SimpleUser
	CurrentState     string
	CurrentLabel     string
	CurrentMilestone string
	CurrentSort      string
	Page             int
	TotalPages       int
	HasNext          bool
	HasPrev          bool
	Breadcrumbs      []Breadcrumb
	UnreadCount      int
	ActiveNav        string
}

// CheckStatus represents a CI check.
type CheckStatus struct {
	Name        string
	State       string
	TargetURL   string
	Description string
}

// PullCommitView extends pull commit with display fields for PR commits tab.
type PullCommitView struct {
	*pulls.Commit
	MessageTitle string // First line of commit message
	MessageBody  string // Rest of commit message
	IsTruncated  bool   // Message > 72 chars
	DateShort    string // "Sep 27"
}

// PullCommitGroup represents commits grouped by date for PR commits tab.
type PullCommitGroup struct {
	Date    string            // "Sep 27, 2025"
	DateKey string            // "2025-09-27" for sorting
	Commits []*PullCommitView // Commits in this group
}

// PullDetailData holds data for single PR view.
type PullDetailData struct {
	Title          string
	User           *users.User
	Repo           *RepoView
	Pull           *PullView
	Commits        []*pulls.Commit
	CommitGroups   []*PullCommitGroup
	Files          []*pulls.PRFile
	FileViews      []*FileChangeView
	Reviews        []*pulls.Review
	Comments       []*CommentView
	Timeline       []*TimelineEvent
	Labels         []*labels.Label
	Milestones     []*milestones.Milestone
	Assignees      []*users.SimpleUser
	Participants   []*users.SimpleUser
	CanEdit        bool
	CanClose       bool
	CanMerge       bool
	MergeableState string
	Checks         []*CheckStatus
	Breadcrumbs    []Breadcrumb
	UnreadCount    int
	ActiveNav      string
	ActiveTab      string // "conversation", "commits", or "files"
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
	commits       commits.API
	orgs          orgs.API
	notifications notifications.API
	stars         stars.API
	watches       watches.API
	branches      branches.API
	releases      releases.API
	labels        labels.API
	milestones    milestones.API
	git           git.API
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
	commitsAPI commits.API,
	orgsAPI orgs.API,
	notificationsAPI notifications.API,
	starsAPI stars.API,
	watchesAPI watches.API,
	branchesAPI branches.API,
	releasesAPI releases.API,
	labelsAPI labels.API,
	milestonesAPI milestones.API,
	gitAPI git.API,
	getUserID func(*mizu.Ctx) int64,
) *Page {
	return &Page{
		templates:     templates,
		users:         usersAPI,
		repos:         reposAPI,
		issues:        issuesAPI,
		pulls:         pullsAPI,
		comments:      commentsAPI,
		commits:       commitsAPI,
		orgs:          orgsAPI,
		notifications: notificationsAPI,
		stars:         starsAPI,
		watches:       watchesAPI,
		branches:      branchesAPI,
		releases:      releasesAPI,
		labels:        labelsAPI,
		milestones:    milestonesAPI,
		git:           gitAPI,
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

	// Get tag count
	var tagCount int
	if tags, err := h.git.ListTags(ctx, owner, repoName, nil); err == nil {
		tagCount = len(tags)
	}

	// Get contributors
	contributors, _ := h.repos.ListContributors(ctx, owner, repoName, nil)
	contributorCount := len(contributors)

	// Get latest release
	var latestRelease *ReleaseView
	var releaseCount int
	if releaseList, err := h.releases.List(ctx, owner, repoName, nil); err == nil {
		releaseCount = len(releaseList)
		if len(releaseList) > 0 {
			r := releaseList[0]
			latestRelease = &ReleaseView{
				ID:      r.ID,
				TagName: r.TagName,
				Name:    r.Name,
				TimeAgo: formatTimeAgo(r.CreatedAt),
			}
		}
	}

	// Get host from request
	host := c.Request().Host

	return render(h, c, "repo_home", RepoHomeData{
		Title:            repo.FullName,
		User:             user,
		Repo:             repoView,
		Readme:           readmeHTML,
		Tree:             tree,
		Branches:         branchList,
		CurrentBranch:    currentBranch,
		Languages:        languages,
		ActiveNav:        "",
		LatestCommit:     latestCommit,
		CommitCount:      commitCount,
		Host:             host,
		TagCount:         tagCount,
		ReleaseCount:     releaseCount,
		ContributorCount: contributorCount,
		LatestRelease:    latestRelease,
		Contributors:     contributors,
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

	query := c.Query("q")
	labelFilter := c.Query("label")
	milestoneFilter := c.Query("milestone")
	sortParam := c.Query("sort")

	// Map sort param to ListOpts sort/direction
	sort := "created"
	direction := "desc"
	switch sortParam {
	case "created-asc":
		sort = "created"
		direction = "asc"
	case "comments-desc":
		sort = "comments"
		direction = "desc"
	case "updated-desc":
		sort = "updated"
		direction = "desc"
	default:
		sortParam = "created-desc"
	}

	page := parseInt(c.Query("page"))
	if page < 1 {
		page = 1
	}
	perPage := 30

	issueList, _ := h.issues.ListForRepo(ctx, owner, repoName, &issues.ListOpts{
		State:     state,
		Labels:    labelFilter,
		Milestone: milestoneFilter,
		Sort:      sort,
		Direction: direction,
		Page:      page,
		PerPage:   perPage + 1, // Request one extra to check if there's a next page
	})

	// Determine if there are more results
	hasNext := len(issueList) > perPage
	if hasNext {
		issueList = issueList[:perPage] // Trim to the requested page size
	}
	hasPrev := page > 1

	// Get accurate counts for open and closed issues
	openCount, _ := h.issues.CountByState(ctx, owner, repoName, "open")
	closedCount, _ := h.issues.CountByState(ctx, owner, repoName, "closed")

	// Calculate total pages based on current state
	totalCount := openCount
	if state == "closed" {
		totalCount = closedCount
	}
	totalPages := (totalCount + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
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

	// Get list of assignable users (collaborators + owner)
	assignees, _ := h.issues.ListAssignees(ctx, owner, repoName)

	var user *users.User
	if userID != 0 {
		user, _ = h.users.GetByID(ctx, userID)
	}

	return render(h, c, "repo_issues", RepoIssuesData{
		Title:            "Issues",
		User:             user,
		Repo:             repoView,
		Issues:           issueViews,
		OpenCount:        openCount,
		ClosedCount:      closedCount,
		Labels:           labelList,
		Milestones:       milestoneList,
		Assignees:        assignees,
		CurrentState:     state,
		CurrentLabel:     labelFilter,
		CurrentMilestone: milestoneFilter,
		CurrentSort:      sortParam,
		Query:            query,
		Page:             page,
		TotalPages:       totalPages,
		HasNext:          hasNext,
		HasPrev:          hasPrev,
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

	// Render issue body markdown
	var issueBodyHTML template.HTML
	if issue.Body != "" {
		issueBodyHTML = template.HTML(renderMarkdown([]byte(issue.Body)))
	}

	issueView := &IssueView{
		Issue:    issue,
		TimeAgo:  timeAgo(issue.CreatedAt),
		BodyHTML: issueBodyHTML,
	}

	commentList, err := h.comments.ListForIssue(ctx, owner, repoName, number, &comments.ListOpts{PerPage: 100})
	if err != nil {
		// Log error but continue - comments are optional
		_ = err
	}
	commentViews := make([]*CommentView, len(commentList))
	for i, comment := range commentList {
		// Render comment body markdown
		var commentBodyHTML template.HTML
		if comment.Body != "" {
			commentBodyHTML = template.HTML(renderMarkdown([]byte(comment.Body)))
		}
		commentViews[i] = &CommentView{
			IssueComment: comment,
			TimeAgo:      formatTimeAgo(comment.CreatedAt),
			BodyHTML:     commentBodyHTML,
		}
	}

	// Fetch unique participants via SQL (includes all commenters, not just first page)
	commenters, _ := h.comments.ListUniqueCommentersForIssue(ctx, owner, repoName, number)
	participantMap := make(map[string]*users.SimpleUser)
	if issue.User != nil {
		participantMap[issue.User.Login] = issue.User
	}
	for _, u := range commenters {
		if u != nil && u.Login != "" {
			if _, exists := participantMap[u.Login]; !exists {
				participantMap[u.Login] = u
			}
		}
	}
	participants := make([]*users.SimpleUser, 0, len(participantMap))
	for _, p := range participantMap {
		participants = append(participants, p)
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
		Title:        fmt.Sprintf("%s #%d", issue.Title, issue.Number),
		User:         user,
		Repo:         repoView,
		Issue:        issueView,
		Comments:     commentViews,
		Labels:       labelList,
		Milestones:   milestoneList,
		Participants: participants,
		CanEdit:      canEdit,
		CanClose:     canClose,
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

// RepoPulls renders the pull requests list page.
func (h *Page) RepoPulls(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	ctx := c.Request().Context()
	userID := h.getUserID(c)

	repo, err := h.repos.Get(ctx, owner, repoName)
	if err != nil {
		return c.Text(http.StatusNotFound, "Repository not found")
	}

	repoView := h.buildRepoView(ctx, repo, userID, "pulls")

	state := c.Query("state")
	if state == "" {
		state = "open"
	}

	labelFilter := c.Query("label")
	sortParam := c.Query("sort")

	// Map sort param to ListOpts sort/direction
	sort := "created"
	direction := "desc"
	switch sortParam {
	case "created-asc":
		sort = "created"
		direction = "asc"
	case "updated-desc":
		sort = "updated"
		direction = "desc"
	default:
		sortParam = "created-desc"
	}

	page := parseInt(c.Query("page"))
	if page < 1 {
		page = 1
	}
	perPage := 30

	pullList, _ := h.pulls.List(ctx, owner, repoName, &pulls.ListOpts{
		State:     state,
		Sort:      sort,
		Direction: direction,
		Page:      page,
		PerPage:   perPage + 1,
	})

	// Determine if there are more results
	hasNext := len(pullList) > perPage
	if hasNext {
		pullList = pullList[:perPage]
	}
	hasPrev := page > 1

	// Count open and closed PRs
	openCount := 0
	closedCount := 0
	allPRs, _ := h.pulls.List(ctx, owner, repoName, &pulls.ListOpts{State: "all", PerPage: 1000})
	for _, pr := range allPRs {
		if pr.State == "open" {
			openCount++
		} else {
			closedCount++
		}
	}

	// Calculate total pages
	totalCount := openCount
	if state == "closed" {
		totalCount = closedCount
	}
	totalPages := (totalCount + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	pullViews := make([]*PullView, len(pullList))
	for i, pr := range pullList {
		statusIcon := "git-pull-request"
		statusColor := "color-fg-success"
		if pr.Draft {
			statusIcon = "git-pull-request-draft"
			statusColor = "color-fg-muted"
		} else if pr.Merged {
			statusIcon = "git-merge"
			statusColor = "color-fg-done"
		} else if pr.State == "closed" {
			statusIcon = "git-pull-request-closed"
			statusColor = "color-fg-danger"
		}

		pullViews[i] = &PullView{
			PullRequest: pr,
			TimeAgo:     timeAgo(pr.CreatedAt),
			StatusIcon:  statusIcon,
			StatusColor: statusColor,
		}
	}

	labelList, _ := h.labels.List(ctx, owner, repoName, nil)
	milestoneList, _ := h.milestones.List(ctx, owner, repoName, nil)

	var user *users.User
	if userID != 0 {
		user, _ = h.users.GetByID(ctx, userID)
	}

	return render(h, c, "repo_pulls", RepoPullsData{
		Title:        "Pull Requests",
		User:         user,
		Repo:         repoView,
		PullRequests: pullViews,
		OpenCount:    openCount,
		ClosedCount:  closedCount,
		Labels:       labelList,
		Milestones:   milestoneList,
		CurrentState: state,
		CurrentLabel: labelFilter,
		CurrentSort:  sortParam,
		Page:         page,
		TotalPages:   totalPages,
		HasNext:      hasNext,
		HasPrev:      hasPrev,
		Breadcrumbs: []Breadcrumb{
			{Label: repo.FullName, URL: "/" + repo.FullName},
			{Label: "Pull requests", URL: ""},
		},
		ActiveNav: "",
	})
}

// PullDetail renders the single pull request view.
func (h *Page) PullDetail(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	number := parseInt(c.Param("number"))

	ctx := c.Request().Context()
	userID := h.getUserID(c)

	repo, err := h.repos.Get(ctx, owner, repoName)
	if err != nil {
		return c.Text(http.StatusNotFound, "Repository not found")
	}

	pr, err := h.pulls.Get(ctx, owner, repoName, number)
	if err != nil {
		return c.Text(http.StatusNotFound, "Pull request not found")
	}

	repoView := h.buildRepoView(ctx, repo, userID, "pulls")

	// Render PR body markdown
	var prBodyHTML template.HTML
	if pr.Body != "" {
		prBodyHTML = template.HTML(renderMarkdown([]byte(pr.Body)))
	}

	// Determine status icon and color
	statusIcon := "git-pull-request"
	statusColor := "color-fg-success"
	if pr.Draft {
		statusIcon = "git-pull-request-draft"
		statusColor = "color-fg-muted"
	} else if pr.Merged {
		statusIcon = "git-merge"
		statusColor = "color-fg-done"
	} else if pr.State == "closed" {
		statusIcon = "git-pull-request-closed"
		statusColor = "color-fg-danger"
	}

	pullView := &PullView{
		PullRequest: pr,
		TimeAgo:     timeAgo(pr.CreatedAt),
		StatusIcon:  statusIcon,
		StatusColor: statusColor,
		BodyHTML:    prBodyHTML,
	}

	// Get commits
	commitList, _ := h.pulls.ListCommits(ctx, owner, repoName, number, nil)

	// Get files changed
	fileList, _ := h.pulls.ListFiles(ctx, owner, repoName, number, nil)

	// Convert files to FileChangeView for diff display
	var fileViews []*FileChangeView
	for _, f := range fileList {
		diffLines := parsePatch(f.Patch)
		isBinary := f.Patch == "" && f.Status == "modified" && f.Additions == 0 && f.Deletions == 0
		tooLarge := len(f.Patch) > 100000

		fileViews = append(fileViews, &FileChangeView{
			SHA:       f.SHA,
			Filename:  f.Filename,
			Status:    f.Status,
			Additions: f.Additions,
			Deletions: f.Deletions,
			Changes:   f.Changes,
			BlobURL:   f.BlobURL,
			RawURL:    f.RawURL,
			Patch:     f.Patch,
			DiffLines: diffLines,
			IsBinary:  isBinary,
			TooLarge:  tooLarge,
		})
	}

	// Get reviews
	reviewList, _ := h.pulls.ListReviews(ctx, owner, repoName, number, nil)

	// Get comments (PR comments are stored with issue_id = pr.ID)
	commentList, _ := h.comments.ListForPR(ctx, owner, repoName, pr.ID, &comments.ListOpts{PerPage: 100})
	commentViews := make([]*CommentView, len(commentList))

	// Build unique participants list (PR author + unique commenters)
	participantMap := make(map[string]*users.SimpleUser)
	if pr.User != nil {
		participantMap[pr.User.Login] = pr.User
	}

	for i, comment := range commentList {
		var commentBodyHTML template.HTML
		if comment.Body != "" {
			commentBodyHTML = template.HTML(renderMarkdown([]byte(comment.Body)))
		}
		commentViews[i] = &CommentView{
			IssueComment: comment,
			TimeAgo:      formatTimeAgo(comment.CreatedAt),
			BodyHTML:     commentBodyHTML,
		}
		if comment.User != nil && comment.User.Login != "" {
			if _, exists := participantMap[comment.User.Login]; !exists {
				participantMap[comment.User.Login] = comment.User
			}
		}
	}

	participants := make([]*users.SimpleUser, 0, len(participantMap))
	for _, p := range participantMap {
		participants = append(participants, p)
	}

	labelList, _ := h.labels.List(ctx, owner, repoName, nil)
	milestoneList, _ := h.milestones.List(ctx, owner, repoName, nil)

	var user *users.User
	canEdit := false
	canClose := false
	canMerge := false
	if userID != 0 {
		user, _ = h.users.GetByID(ctx, userID)
		canEdit = userID == pr.CreatorID || repoView.CanAdmin
		canClose = canEdit || repoView.CanPush
		canMerge = repoView.CanPush && pr.State == "open" && !pr.Merged
	}

	return render(h, c, "pull_view", PullDetailData{
		Title:          fmt.Sprintf("%s #%d", pr.Title, pr.Number),
		User:           user,
		Repo:           repoView,
		Pull:           pullView,
		Commits:        commitList,
		Files:          fileList,
		FileViews:      fileViews,
		Reviews:        reviewList,
		Comments:       commentViews,
		Labels:         labelList,
		Milestones:     milestoneList,
		Participants:   participants,
		CanEdit:        canEdit,
		CanClose:       canClose,
		CanMerge:       canMerge,
		MergeableState: pr.MergeableState,
		Breadcrumbs: []Breadcrumb{
			{Label: repo.FullName, URL: "/" + repo.FullName},
			{Label: "Pull requests", URL: "/" + repo.FullName + "/pulls"},
			{Label: fmt.Sprintf("#%d", pr.Number), URL: ""},
		},
		ActiveNav: "",
		ActiveTab: "conversation",
	})
}

// PullCommits renders the PR commits tab.
func (h *Page) PullCommits(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	number := parseInt(c.Param("number"))

	ctx := c.Request().Context()
	userID := h.getUserID(c)

	repo, err := h.repos.Get(ctx, owner, repoName)
	if err != nil {
		return c.Text(http.StatusNotFound, "Repository not found")
	}

	pr, err := h.pulls.Get(ctx, owner, repoName, number)
	if err != nil {
		return c.Text(http.StatusNotFound, "Pull request not found")
	}

	repoView := h.buildRepoView(ctx, repo, userID, "pulls")

	// Render PR body markdown
	var prBodyHTML template.HTML
	if pr.Body != "" {
		prBodyHTML = template.HTML(renderMarkdown([]byte(pr.Body)))
	}

	// Determine status
	statusIcon := "git-pull-request"
	statusColor := "color-fg-success"
	if pr.Draft {
		statusIcon = "git-pull-request-draft"
		statusColor = "color-fg-muted"
	} else if pr.Merged {
		statusIcon = "git-merge"
		statusColor = "color-fg-done"
	} else if pr.State == "closed" {
		statusIcon = "git-pull-request-closed"
		statusColor = "color-fg-danger"
	}

	pullView := &PullView{
		PullRequest: pr,
		TimeAgo:     timeAgo(pr.CreatedAt),
		StatusIcon:  statusIcon,
		StatusColor: statusColor,
		BodyHTML:    prBodyHTML,
	}

	// Get commits and group by date
	commitList, _ := h.pulls.ListCommits(ctx, owner, repoName, number, nil)
	commitGroups := groupCommitsByDateForPR(commitList)

	var user *users.User
	if userID != 0 {
		user, _ = h.users.GetByID(ctx, userID)
	}

	return render(h, c, "pull_commits", PullDetailData{
		Title:        fmt.Sprintf("Commits · %s #%d", pr.Title, pr.Number),
		User:         user,
		Repo:         repoView,
		Pull:         pullView,
		Commits:      commitList,
		CommitGroups: commitGroups,
		Breadcrumbs: []Breadcrumb{
			{Label: repo.FullName, URL: "/" + repo.FullName},
			{Label: "Pull requests", URL: "/" + repo.FullName + "/pulls"},
			{Label: fmt.Sprintf("#%d", pr.Number), URL: "/" + repo.FullName + "/pulls/" + fmt.Sprint(pr.Number)},
			{Label: "Commits", URL: ""},
		},
		ActiveNav: "",
		ActiveTab: "commits",
	})
}

// PullFiles renders the PR files changed tab.
func (h *Page) PullFiles(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	number := parseInt(c.Param("number"))

	ctx := c.Request().Context()
	userID := h.getUserID(c)

	repo, err := h.repos.Get(ctx, owner, repoName)
	if err != nil {
		return c.Text(http.StatusNotFound, "Repository not found")
	}

	pr, err := h.pulls.Get(ctx, owner, repoName, number)
	if err != nil {
		return c.Text(http.StatusNotFound, "Pull request not found")
	}

	repoView := h.buildRepoView(ctx, repo, userID, "pulls")

	// Render PR body markdown
	var prBodyHTML template.HTML
	if pr.Body != "" {
		prBodyHTML = template.HTML(renderMarkdown([]byte(pr.Body)))
	}

	// Determine status
	statusIcon := "git-pull-request"
	statusColor := "color-fg-success"
	if pr.Draft {
		statusIcon = "git-pull-request-draft"
		statusColor = "color-fg-muted"
	} else if pr.Merged {
		statusIcon = "git-merge"
		statusColor = "color-fg-done"
	} else if pr.State == "closed" {
		statusIcon = "git-pull-request-closed"
		statusColor = "color-fg-danger"
	}

	pullView := &PullView{
		PullRequest: pr,
		TimeAgo:     timeAgo(pr.CreatedAt),
		StatusIcon:  statusIcon,
		StatusColor: statusColor,
		BodyHTML:    prBodyHTML,
	}

	// Get commits for "All commits" dropdown
	commitList, _ := h.pulls.ListCommits(ctx, owner, repoName, number, nil)

	// Get files changed
	fileList, _ := h.pulls.ListFiles(ctx, owner, repoName, number, nil)

	// Convert files to FileChangeView for diff display
	var fileViews []*FileChangeView
	for _, f := range fileList {
		diffLines := parsePatch(f.Patch)
		isBinary := f.Patch == "" && f.Status == "modified" && f.Additions == 0 && f.Deletions == 0
		tooLarge := len(f.Patch) > 100000

		fileViews = append(fileViews, &FileChangeView{
			SHA:       f.SHA,
			Filename:  f.Filename,
			Status:    f.Status,
			Additions: f.Additions,
			Deletions: f.Deletions,
			Changes:   f.Changes,
			BlobURL:   f.BlobURL,
			RawURL:    f.RawURL,
			Patch:     f.Patch,
			DiffLines: diffLines,
			IsBinary:  isBinary,
			TooLarge:  tooLarge,
		})
	}

	var user *users.User
	if userID != 0 {
		user, _ = h.users.GetByID(ctx, userID)
	}

	return render(h, c, "pull_files", PullDetailData{
		Title:     fmt.Sprintf("Files changed · %s #%d", pr.Title, pr.Number),
		User:      user,
		Repo:      repoView,
		Pull:      pullView,
		Commits:   commitList,
		Files:     fileList,
		FileViews: fileViews,
		Breadcrumbs: []Breadcrumb{
			{Label: repo.FullName, URL: "/" + repo.FullName},
			{Label: "Pull requests", URL: "/" + repo.FullName + "/pulls"},
			{Label: fmt.Sprintf("#%d", pr.Number), URL: "/" + repo.FullName + "/pulls/" + fmt.Sprint(pr.Number)},
			{Label: "Files changed", URL: ""},
		},
		ActiveNav: "",
		ActiveTab: "files",
	})
}

// groupCommitsByDateForPR groups commits by their date for PR commits tab.
func groupCommitsByDateForPR(commits []*pulls.Commit) []*PullCommitGroup {
	if len(commits) == 0 {
		return nil
	}

	groups := make(map[string]*PullCommitGroup)
	var order []string

	for _, c := range commits {
		if c.Commit == nil || c.Commit.Author == nil {
			continue
		}

		date := c.Commit.Author.Date.Format("Jan 2, 2006")
		key := c.Commit.Author.Date.Format("2006-01-02")

		if _, ok := groups[key]; !ok {
			groups[key] = &PullCommitGroup{Date: date, DateKey: key}
			order = append(order, key)
		}

		// Extract first line of commit message
		msg := c.Commit.Message
		msgTitle := msg
		msgBody := ""
		if idx := strings.Index(msg, "\n"); idx != -1 {
			msgTitle = msg[:idx]
			msgBody = strings.TrimSpace(msg[idx+1:])
		}

		groups[key].Commits = append(groups[key].Commits, &PullCommitView{
			Commit:       c,
			MessageTitle: msgTitle,
			MessageBody:  msgBody,
			IsTruncated:  len(msgTitle) > 72,
			DateShort:    c.Commit.Author.Date.Format("Jan 2"),
		})
	}

	// Sort by date (newest first)
	sort.Sort(sort.Reverse(sort.StringSlice(order)))

	result := make([]*PullCommitGroup, len(order))
	for i, k := range order {
		result[i] = groups[k]
	}
	return result
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

	// Build breadcrumbs and parent path
	breadcrumbs := []Breadcrumb{
		{Label: repo.FullName, URL: "/" + repo.FullName},
	}
	var parentPath string
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
		// Calculate parent path
		if len(parts) == 1 {
			// At top level of subdirectory, parent is repo root
			parentPath = "/" + repo.FullName
		} else {
			// Parent is one level up
			parentPath = fmt.Sprintf("/%s/tree/%s/%s", repo.FullName, ref, strings.Join(parts[:len(parts)-1], "/"))
		}
	}

	return render(h, c, "repo_code", RepoCodeData{
		Title:         repo.FullName,
		User:          user,
		Repo:          repoView,
		Tree:          tree,
		CurrentBranch: ref,
		CurrentPath:   path,
		ParentPath:    parentPath,
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

	// Detect language from file extension
	language := detectLanguage(path)

	return render(h, c, "repo_blame", RepoBlameData{
		Title:         "Blame: " + fileName + " - " + repo.FullName,
		User:          user,
		Repo:          repoView,
		Lines:         lines,
		FilePath:      path,
		FileName:      fileName,
		LineCount:     len(lines),
		Language:      language,
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
	return formatTimeAgo(t)
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
			html.WithUnsafe(), // Allow raw HTML like <pre> tags from GitHub issues
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
	days := int(diff.Hours() / 24)

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
	} else if days == 1 {
		return "yesterday"
	} else if days < 7 {
		return fmt.Sprintf("%d days ago", days)
	} else if days < 14 {
		return "last week"
	} else if days < 30 {
		weeks := days / 7
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if days < 60 {
		return "last month"
	} else if days < 365 {
		months := days / 30
		return fmt.Sprintf("%d months ago", months)
	} else if days < 730 {
		return "last year"
	}
	years := days / 365
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

// RepoCommits renders the commits list page
func (h *Page) RepoCommits(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	ref := c.Param("ref")

	ctx := c.Request().Context()
	userID := h.getUserID(c)

	repo, err := h.repos.Get(ctx, owner, repoName)
	if err != nil {
		return c.Text(http.StatusNotFound, "Repository not found")
	}

	// Default to main branch if ref is empty
	if ref == "" {
		ref = repo.DefaultBranch
		if ref == "" {
			ref = "master"
		}
	}

	repoView := h.buildRepoView(ctx, repo, userID, "code")
	branchList, _ := h.branches.List(ctx, owner, repoName, nil)

	var user *users.User
	if userID != 0 {
		user, _ = h.users.GetByID(ctx, userID)
	}

	// Parse page number
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
		if page < 1 {
			page = 1
		}
	}

	perPage := 35

	// Get commits from the service
	commitList, err := h.commits.List(ctx, owner, repoName, &commits.ListOpts{
		SHA:     ref,
		Page:    page,
		PerPage: perPage + 1, // Get one extra to check for next page
	})
	if err != nil {
		return c.Text(http.StatusInternalServerError, "Failed to load commits")
	}

	// Check if there are more pages
	hasNext := len(commitList) > perPage
	if hasNext {
		commitList = commitList[:perPage]
	}
	hasPrev := page > 1

	// Group commits by date
	groups := make(map[string][]*CommitViewItem)
	dateOrder := []string{}

	for _, commit := range commitList {
		// Get author date for grouping
		var authorDate time.Time
		if commit.Commit != nil && commit.Commit.Author != nil {
			authorDate = commit.Commit.Author.Date
		}

		dateKey := authorDate.Format("Jan 2, 2006")
		if _, exists := groups[dateKey]; !exists {
			dateOrder = append(dateOrder, dateKey)
		}

		// Build author view
		authorView := &UserView{}
		if commit.Author != nil {
			authorView.Login = commit.Author.Login
			authorView.AvatarURL = commit.Author.AvatarURL
			authorView.HTMLURL = commit.Author.HTMLURL
		}
		if commit.Commit != nil && commit.Commit.Author != nil {
			authorView.Name = commit.Commit.Author.Name
			authorView.Email = commit.Commit.Author.Email
		}
		// Ensure avatar URL is set
		authorView.AvatarURL = ensureAvatarURL(authorView.AvatarURL, authorView.Email, authorView.Login)

		// Build committer view
		committerView := &UserView{}
		if commit.Committer != nil {
			committerView.Login = commit.Committer.Login
			committerView.AvatarURL = commit.Committer.AvatarURL
			committerView.HTMLURL = commit.Committer.HTMLURL
		}
		if commit.Commit != nil && commit.Commit.Committer != nil {
			committerView.Name = commit.Commit.Committer.Name
			committerView.Email = commit.Commit.Committer.Email
		}
		// Ensure avatar URL is set
		committerView.AvatarURL = ensureAvatarURL(committerView.AvatarURL, committerView.Email, committerView.Login)

		// Check if same author
		isSame := authorView.Email == committerView.Email

		message := ""
		if commit.Commit != nil {
			message = commit.Commit.Message
		}

		item := &CommitViewItem{
			SHA:          commit.SHA,
			ShortSHA:     shortSHA(commit.SHA),
			Message:      message,
			MessageTitle: firstLine(message),
			Author:       authorView,
			Committer:    committerView,
			AuthorDate:   authorDate,
			TimeAgo:      formatTimeAgo(authorDate),
			IsSameAuthor: isSame,
			TreeURL:      fmt.Sprintf("/%s/tree/%s", repo.FullName, commit.SHA),
			CommitURL:    fmt.Sprintf("/%s/commit/%s", repo.FullName, commit.SHA),
		}

		groups[dateKey] = append(groups[dateKey], item)
	}

	// Build ordered commit groups
	var commitGroups []*CommitGroup
	for _, date := range dateOrder {
		commitGroups = append(commitGroups, &CommitGroup{
			Date:    date,
			Commits: groups[date],
		})
	}

	return render(h, c, "repo_commits", RepoCommitsData{
		Title:         "Commits - " + repo.FullName,
		User:          user,
		Repo:          repoView,
		CommitGroups:  commitGroups,
		CurrentBranch: ref,
		Branches:      branchList,
		Page:          page,
		HasNext:       hasNext,
		HasPrev:       hasPrev,
		Breadcrumbs: []Breadcrumb{
			{Label: repo.FullName, URL: "/" + repo.FullName},
			{Label: "Commits", URL: ""},
		},
		ActiveNav: "",
	})
}

// CommitDetail renders a single commit view
func (h *Page) CommitDetail(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	sha := c.Param("sha")

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

	// Get the commit with files
	commit, err := h.commits.Get(ctx, owner, repoName, sha)
	if err != nil {
		// Fallback: try to get from PR commits (seeded repos)
		prCommit, prErr := h.pulls.GetCommitBySHA(ctx, owner, repoName, sha)
		if prErr != nil || prCommit == nil {
			return c.Text(http.StatusNotFound, "Commit not found")
		}
		// Convert pulls.Commit to commits.Commit
		commit = &commits.Commit{
			SHA:     prCommit.SHA,
			NodeID:  prCommit.NodeID,
			URL:     prCommit.URL,
			HTMLURL: prCommit.HTMLURL,
		}
		if prCommit.Commit != nil {
			commit.Commit = &commits.CommitData{
				Message: prCommit.Commit.Message,
			}
			if prCommit.Commit.Author != nil {
				commit.Commit.Author = &commits.CommitAuthor{
					Name:  prCommit.Commit.Author.Name,
					Email: prCommit.Commit.Author.Email,
					Date:  prCommit.Commit.Author.Date,
				}
			}
			if prCommit.Commit.Committer != nil {
				commit.Commit.Committer = &commits.CommitAuthor{
					Name:  prCommit.Commit.Committer.Name,
					Email: prCommit.Commit.Committer.Email,
					Date:  prCommit.Commit.Committer.Date,
				}
			}
			if prCommit.Commit.Tree != nil {
				commit.Commit.Tree = &commits.TreeRef{
					SHA: prCommit.Commit.Tree.SHA,
					URL: prCommit.Commit.Tree.URL,
				}
			}
		}
		if prCommit.Author != nil {
			commit.Author = prCommit.Author
		}
		if prCommit.Committer != nil {
			commit.Committer = prCommit.Committer
		}
		for _, p := range prCommit.Parents {
			commit.Parents = append(commit.Parents, &commits.CommitRef{
				SHA: p.SHA,
				URL: p.URL,
			})
		}
		// Get files from PR for this commit
		prFiles, _ := h.pulls.ListFilesByCommitSHA(ctx, owner, repoName, sha)
		if len(prFiles) > 0 {
			commit.Files = make([]*commits.CommitFile, len(prFiles))
			for i, pf := range prFiles {
				commit.Files[i] = &commits.CommitFile{
					SHA:              pf.SHA,
					Filename:         pf.Filename,
					Status:           pf.Status,
					Additions:        pf.Additions,
					Deletions:        pf.Deletions,
					Changes:          pf.Changes,
					BlobURL:          pf.BlobURL,
					RawURL:           pf.RawURL,
					ContentsURL:      pf.ContentsURL,
					Patch:            pf.Patch,
					PreviousFilename: pf.PreviousFilename,
				}
			}
		}
	}

	// Build author view
	authorView := &UserView{}
	if commit.Author != nil {
		authorView.Login = commit.Author.Login
		authorView.AvatarURL = commit.Author.AvatarURL
		authorView.HTMLURL = commit.Author.HTMLURL
	}
	if commit.Commit != nil && commit.Commit.Author != nil {
		authorView.Name = commit.Commit.Author.Name
		authorView.Email = commit.Commit.Author.Email
	}
	// Ensure avatar URL is set
	authorView.AvatarURL = ensureAvatarURL(authorView.AvatarURL, authorView.Email, authorView.Login)

	// Build committer view
	committerView := &UserView{}
	if commit.Committer != nil {
		committerView.Login = commit.Committer.Login
		committerView.AvatarURL = commit.Committer.AvatarURL
		committerView.HTMLURL = commit.Committer.HTMLURL
	}
	if commit.Commit != nil && commit.Commit.Committer != nil {
		committerView.Name = commit.Commit.Committer.Name
		committerView.Email = commit.Commit.Committer.Email
	}
	// Ensure avatar URL is set
	committerView.AvatarURL = ensureAvatarURL(committerView.AvatarURL, committerView.Email, committerView.Login)

	isSame := authorView.Email == committerView.Email

	message := ""
	messageTitle := ""
	messageBody := ""
	var authorDate time.Time
	var treeSHA string

	if commit.Commit != nil {
		message = commit.Commit.Message
		messageTitle = firstLine(message)
		if idx := strings.Index(message, "\n"); idx >= 0 {
			messageBody = strings.TrimSpace(message[idx+1:])
		}
		if commit.Commit.Author != nil {
			authorDate = commit.Commit.Author.Date
		}
		if commit.Commit.Tree != nil {
			treeSHA = commit.Commit.Tree.SHA
		}
	}

	// Build parent commits
	var parents []*ParentCommit
	for _, p := range commit.Parents {
		parents = append(parents, &ParentCommit{
			SHA:      p.SHA,
			ShortSHA: shortSHA(p.SHA),
			HTMLURL:  fmt.Sprintf("/%s/commit/%s", repo.FullName, p.SHA),
		})
	}

	commitDetail := &CommitViewDetail{
		SHA:          commit.SHA,
		ShortSHA:     shortSHA(commit.SHA),
		Message:      message,
		MessageTitle: messageTitle,
		MessageBody:  messageBody,
		Author:       authorView,
		Committer:    committerView,
		AuthorDate:   authorDate,
		TimeAgo:      formatTimeAgo(authorDate),
		IsSameAuthor: isSame,
		Parents:      parents,
		TreeSHA:      treeSHA,
		TreeURL:      fmt.Sprintf("/%s/tree/%s", repo.FullName, commit.SHA),
		HTMLURL:      fmt.Sprintf("/%s/commit/%s", repo.FullName, commit.SHA),
	}

	// Build file changes
	var files []*FileChangeView
	var totalAdditions, totalDeletions int

	for _, f := range commit.Files {
		// Parse patch into diff lines
		diffLines := parsePatch(f.Patch)

		// Check if binary or too large
		isBinary := f.Patch == "" && f.Status == "modified" && f.Additions == 0 && f.Deletions == 0
		tooLarge := len(f.Patch) > 100000 // 100KB limit

		file := &FileChangeView{
			SHA:              f.SHA,
			Filename:         f.Filename,
			PreviousFilename: f.PreviousFilename,
			Status:           f.Status,
			Additions:        f.Additions,
			Deletions:        f.Deletions,
			Changes:          f.Changes,
			BlobURL:          f.BlobURL,
			RawURL:           f.RawURL,
			Patch:            f.Patch,
			DiffLines:        diffLines,
			IsBinary:         isBinary,
			TooLarge:         tooLarge,
		}
		files = append(files, file)

		totalAdditions += f.Additions
		totalDeletions += f.Deletions
	}

	stats := &StatsView{
		FilesChanged: len(files),
		Additions:    totalAdditions,
		Deletions:    totalDeletions,
		Total:        totalAdditions + totalDeletions,
	}

	return render(h, c, "commit_detail", CommitDetailData{
		Title:   fmt.Sprintf("Commit %s - %s", shortSHA(commit.SHA), repo.FullName),
		User:    user,
		Repo:    repoView,
		Commit:  commitDetail,
		Files:   files,
		Stats:   stats,
		Branches: branchList,
		Breadcrumbs: []Breadcrumb{
			{Label: repo.FullName, URL: "/" + repo.FullName},
			{Label: "Commits", URL: "/" + repo.FullName + "/commits"},
			{Label: shortSHA(commit.SHA), URL: ""},
		},
		ActiveNav: "",
	})
}

// parsePatch parses a unified diff patch into DiffLines
func parsePatch(patch string) []*DiffLine {
	if patch == "" {
		return nil
	}

	var lines []*DiffLine
	patchLines := strings.Split(patch, "\n")

	var oldLine, newLine int

	for _, line := range patchLines {
		if line == "" {
			continue
		}

		dl := &DiffLine{}

		if strings.HasPrefix(line, "@@") {
			// Parse hunk header: @@ -oldStart,oldCount +newStart,newCount @@
			dl.Type = "hunk"
			dl.Content = line

			// Parse line numbers from hunk header
			var oldStart, oldCount, newStart, newCount int
			fmt.Sscanf(line, "@@ -%d,%d +%d,%d @@", &oldStart, &oldCount, &newStart, &newCount)
			if oldStart == 0 {
				fmt.Sscanf(line, "@@ -%d +%d,%d @@", &oldStart, &newStart, &newCount)
			}
			if oldStart == 0 {
				fmt.Sscanf(line, "@@ -%d,%d +%d @@", &oldStart, &oldCount, &newStart)
			}
			if oldStart == 0 {
				fmt.Sscanf(line, "@@ -%d +%d @@", &oldStart, &newStart)
			}
			oldLine = oldStart
			newLine = newStart
		} else if strings.HasPrefix(line, "+") {
			dl.Type = "addition"
			dl.NewLineNum = newLine
			dl.Content = line[1:]
			newLine++
		} else if strings.HasPrefix(line, "-") {
			dl.Type = "deletion"
			dl.OldLineNum = oldLine
			dl.Content = line[1:]
			oldLine++
		} else if strings.HasPrefix(line, " ") {
			dl.Type = "context"
			dl.OldLineNum = oldLine
			dl.NewLineNum = newLine
			dl.Content = line[1:]
			oldLine++
			newLine++
		} else {
			// No prefix, treat as context
			dl.Type = "context"
			dl.OldLineNum = oldLine
			dl.NewLineNum = newLine
			dl.Content = line
			oldLine++
			newLine++
		}

		lines = append(lines, dl)
	}

	return lines
}

// restOfMessage returns everything after the first line
func restOfMessage(s string) string {
	if idx := strings.Index(s, "\n"); idx >= 0 {
		return strings.TrimSpace(s[idx+1:])
	}
	return ""
}

// gravatarURL generates a Gravatar URL from an email address.
func gravatarURL(email string) string {
	email = strings.TrimSpace(strings.ToLower(email))
	hash := md5.Sum([]byte(email))
	return fmt.Sprintf("https://www.gravatar.com/avatar/%s?d=identicon&s=40", hex.EncodeToString(hash[:]))
}

// ensureAvatarURL returns the provided URL if not empty, otherwise generates a Gravatar URL.
func ensureAvatarURL(avatarURL, email, login string) string {
	if avatarURL != "" {
		return avatarURL
	}
	// If we have a GitHub login, use GitHub avatar
	if login != "" {
		return fmt.Sprintf("https://avatars.githubusercontent.com/%s?s=40", login)
	}
	// Fall back to Gravatar
	if email != "" {
		return gravatarURL(email)
	}
	// Default avatar
	return "https://avatars.githubusercontent.com/u/0?s=40"
}
