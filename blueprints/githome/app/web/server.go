package web

import (
	"net/http"

	"github.com/mizu-framework/mizu/blueprints/githome/app/web/handler/api"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/activities"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/branches"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/collaborators"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/comments"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/commits"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/git"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/issues"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/labels"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/milestones"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/notifications"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/orgs"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/pulls"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/reactions"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/releases"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/repos"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/search"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/stars"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/teams"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/users"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/watches"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/webhooks"
)

// Services contains all service dependencies
type Services struct {
	Users         users.API
	Orgs          orgs.API
	Repos         repos.API
	Issues        issues.API
	Pulls         pulls.API
	Labels        labels.API
	Milestones    milestones.API
	Comments      comments.API
	Teams         teams.API
	Releases      releases.API
	Stars         stars.API
	Watches       watches.API
	Webhooks      webhooks.API
	Notifications notifications.API
	Reactions     reactions.API
	Collaborators collaborators.API
	Branches      branches.API
	Commits       commits.API
	Git           git.API
	Search        search.API
	Activities    activities.API
}

// Server represents the HTTP server
type Server struct {
	mux      *http.ServeMux
	services *Services
}

// NewServer creates a new server with all routes configured
func NewServer(services *Services) *Server {
	s := &Server{
		mux:      http.NewServeMux(),
		services: services,
	}
	s.setupRoutes()
	return s
}

// Handler returns the HTTP handler
func (s *Server) Handler() http.Handler {
	return s.mux
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Create handlers
	authHandler := api.NewAuthHandler(s.services.Users)
	userHandler := api.NewUserHandler(s.services.Users)
	orgHandler := api.NewOrgHandler(s.services.Orgs)
	repoHandler := api.NewRepoHandler(s.services.Repos)
	issueHandler := api.NewIssueHandler(s.services.Issues, s.services.Repos)
	pullHandler := api.NewPullHandler(s.services.Pulls, s.services.Repos)
	labelHandler := api.NewLabelHandler(s.services.Labels, s.services.Repos)
	milestoneHandler := api.NewMilestoneHandler(s.services.Milestones, s.services.Repos)
	commentHandler := api.NewCommentHandler(s.services.Comments, s.services.Repos)
	teamHandler := api.NewTeamHandler(s.services.Teams)
	releaseHandler := api.NewReleaseHandler(s.services.Releases, s.services.Repos)
	starHandler := api.NewStarHandler(s.services.Stars, s.services.Repos)
	watchHandler := api.NewWatchHandler(s.services.Watches, s.services.Repos)
	webhookHandler := api.NewWebhookHandler(s.services.Webhooks, s.services.Repos)
	notificationHandler := api.NewNotificationHandler(s.services.Notifications, s.services.Repos)
	reactionHandler := api.NewReactionHandler(s.services.Reactions, s.services.Repos)
	collaboratorHandler := api.NewCollaboratorHandler(s.services.Collaborators, s.services.Repos)
	branchHandler := api.NewBranchHandler(s.services.Branches, s.services.Repos)
	commitHandler := api.NewCommitHandler(s.services.Commits, s.services.Repos)
	gitHandler := api.NewGitHandler(s.services.Git, s.services.Repos)
	searchHandler := api.NewSearchHandler(s.services.Search)
	activityHandler := api.NewActivityHandler(s.services.Activities, s.services.Repos)

	// Helper to wrap with auth middleware
	requireAuth := authHandler.RequireAuth
	optionalAuth := authHandler.OptionalAuth

	// ==========================================================================
	// Authentication
	// ==========================================================================
	s.mux.HandleFunc("POST /login", authHandler.Login)
	s.mux.HandleFunc("POST /register", authHandler.Register)

	// ==========================================================================
	// Users
	// ==========================================================================
	s.mux.Handle("GET /user", requireAuth(http.HandlerFunc(userHandler.GetAuthenticatedUser)))
	s.mux.Handle("PATCH /user", requireAuth(http.HandlerFunc(userHandler.UpdateAuthenticatedUser)))
	s.mux.HandleFunc("GET /users", userHandler.ListUsers)
	s.mux.HandleFunc("GET /users/{username}", userHandler.GetUser)
	s.mux.HandleFunc("GET /users/{username}/followers", userHandler.ListFollowers)
	s.mux.HandleFunc("GET /users/{username}/following", userHandler.ListFollowing)
	s.mux.HandleFunc("GET /users/{username}/following/{target_user}", userHandler.CheckFollowing)
	s.mux.Handle("GET /user/followers", requireAuth(http.HandlerFunc(userHandler.ListAuthenticatedUserFollowers)))
	s.mux.Handle("GET /user/following", requireAuth(http.HandlerFunc(userHandler.ListAuthenticatedUserFollowing)))
	s.mux.Handle("GET /user/following/{username}", requireAuth(http.HandlerFunc(userHandler.CheckAuthenticatedUserFollowing)))
	s.mux.Handle("PUT /user/following/{username}", requireAuth(http.HandlerFunc(userHandler.FollowUser)))
	s.mux.Handle("DELETE /user/following/{username}", requireAuth(http.HandlerFunc(userHandler.UnfollowUser)))

	// ==========================================================================
	// Organizations
	// ==========================================================================
	s.mux.HandleFunc("GET /organizations", orgHandler.ListOrgs)
	s.mux.HandleFunc("GET /orgs/{org}", orgHandler.GetOrg)
	s.mux.Handle("PATCH /orgs/{org}", requireAuth(http.HandlerFunc(orgHandler.UpdateOrg)))
	s.mux.Handle("GET /user/orgs", requireAuth(http.HandlerFunc(orgHandler.ListAuthenticatedUserOrgs)))
	s.mux.HandleFunc("GET /users/{username}/orgs", orgHandler.ListUserOrgs)
	s.mux.HandleFunc("GET /orgs/{org}/members", orgHandler.ListOrgMembers)
	s.mux.HandleFunc("GET /orgs/{org}/members/{username}", orgHandler.CheckOrgMember)
	s.mux.Handle("DELETE /orgs/{org}/members/{username}", requireAuth(http.HandlerFunc(orgHandler.RemoveOrgMember)))
	s.mux.Handle("GET /orgs/{org}/memberships/{username}", requireAuth(http.HandlerFunc(orgHandler.GetOrgMembership)))
	s.mux.Handle("PUT /orgs/{org}/memberships/{username}", requireAuth(http.HandlerFunc(orgHandler.SetOrgMembership)))
	s.mux.Handle("DELETE /orgs/{org}/memberships/{username}", requireAuth(http.HandlerFunc(orgHandler.RemoveOrgMembership)))
	s.mux.Handle("GET /orgs/{org}/outside_collaborators", requireAuth(http.HandlerFunc(orgHandler.ListOutsideCollaborators)))
	s.mux.Handle("GET /user/memberships/orgs/{org}", requireAuth(http.HandlerFunc(orgHandler.GetAuthenticatedUserOrgMembership)))
	s.mux.Handle("PATCH /user/memberships/orgs/{org}", requireAuth(http.HandlerFunc(orgHandler.UpdateAuthenticatedUserOrgMembership)))

	// ==========================================================================
	// Repositories
	// ==========================================================================
	s.mux.HandleFunc("GET /repositories", repoHandler.ListPublicRepos)
	s.mux.Handle("GET /user/repos", requireAuth(http.HandlerFunc(repoHandler.ListAuthenticatedUserRepos)))
	s.mux.Handle("POST /user/repos", requireAuth(http.HandlerFunc(repoHandler.CreateAuthenticatedUserRepo)))
	s.mux.HandleFunc("GET /users/{username}/repos", repoHandler.ListUserRepos)
	s.mux.HandleFunc("GET /orgs/{org}/repos", repoHandler.ListOrgRepos)
	s.mux.Handle("POST /orgs/{org}/repos", requireAuth(http.HandlerFunc(repoHandler.CreateOrgRepo)))
	s.mux.Handle("GET /repos/{owner}/{repo}", optionalAuth(http.HandlerFunc(repoHandler.GetRepo)))
	s.mux.Handle("PATCH /repos/{owner}/{repo}", requireAuth(http.HandlerFunc(repoHandler.UpdateRepo)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}", requireAuth(http.HandlerFunc(repoHandler.DeleteRepo)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/topics", repoHandler.ListRepoTopics)
	s.mux.Handle("PUT /repos/{owner}/{repo}/topics", requireAuth(http.HandlerFunc(repoHandler.ReplaceRepoTopics)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/languages", repoHandler.ListRepoLanguages)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/contributors", repoHandler.ListRepoContributors)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/tags", repoHandler.ListRepoTags)
	s.mux.Handle("POST /repos/{owner}/{repo}/transfer", requireAuth(http.HandlerFunc(repoHandler.TransferRepo)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/readme", repoHandler.GetRepoReadme)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/contents/{path...}", repoHandler.GetRepoContent)
	s.mux.Handle("PUT /repos/{owner}/{repo}/contents/{path...}", requireAuth(http.HandlerFunc(repoHandler.CreateOrUpdateFileContent)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/contents/{path...}", requireAuth(http.HandlerFunc(repoHandler.DeleteFileContent)))
	s.mux.Handle("POST /repos/{owner}/{repo}/forks", requireAuth(http.HandlerFunc(repoHandler.ForkRepo)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/forks", repoHandler.ListForks)

	// ==========================================================================
	// Issues
	// ==========================================================================
	s.mux.Handle("GET /issues", requireAuth(http.HandlerFunc(issueHandler.ListIssues)))
	s.mux.Handle("GET /user/issues", requireAuth(http.HandlerFunc(issueHandler.ListAuthenticatedUserIssues)))
	s.mux.Handle("GET /orgs/{org}/issues", requireAuth(http.HandlerFunc(issueHandler.ListOrgIssues)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/issues", issueHandler.ListRepoIssues)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/issues/{issue_number}", issueHandler.GetIssue)
	s.mux.Handle("POST /repos/{owner}/{repo}/issues", requireAuth(http.HandlerFunc(issueHandler.CreateIssue)))
	s.mux.Handle("PATCH /repos/{owner}/{repo}/issues/{issue_number}", requireAuth(http.HandlerFunc(issueHandler.UpdateIssue)))
	s.mux.Handle("PUT /repos/{owner}/{repo}/issues/{issue_number}/lock", requireAuth(http.HandlerFunc(issueHandler.LockIssue)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/issues/{issue_number}/lock", requireAuth(http.HandlerFunc(issueHandler.UnlockIssue)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/assignees", issueHandler.ListIssueAssignees)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/assignees/{assignee}", issueHandler.CheckAssignee)
	s.mux.Handle("POST /repos/{owner}/{repo}/issues/{issue_number}/assignees", requireAuth(http.HandlerFunc(issueHandler.AddAssignees)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/issues/{issue_number}/assignees", requireAuth(http.HandlerFunc(issueHandler.RemoveAssignees)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/issues/{issue_number}/events", issueHandler.ListIssueEvents)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/issues/events/{event_id}", issueHandler.GetIssueEvent)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/issues/events", issueHandler.ListRepoIssueEvents)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/issues/{issue_number}/timeline", issueHandler.ListIssueTimeline)

	// ==========================================================================
	// Pull Requests
	// ==========================================================================
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/pulls", pullHandler.ListPulls)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/pulls/{pull_number}", pullHandler.GetPull)
	s.mux.Handle("POST /repos/{owner}/{repo}/pulls", requireAuth(http.HandlerFunc(pullHandler.CreatePull)))
	s.mux.Handle("PATCH /repos/{owner}/{repo}/pulls/{pull_number}", requireAuth(http.HandlerFunc(pullHandler.UpdatePull)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/pulls/{pull_number}/commits", pullHandler.ListPullCommits)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/pulls/{pull_number}/files", pullHandler.ListPullFiles)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/pulls/{pull_number}/merge", pullHandler.CheckPullMerged)
	s.mux.Handle("PUT /repos/{owner}/{repo}/pulls/{pull_number}/merge", requireAuth(http.HandlerFunc(pullHandler.MergePull)))
	s.mux.Handle("PUT /repos/{owner}/{repo}/pulls/{pull_number}/update-branch", requireAuth(http.HandlerFunc(pullHandler.UpdatePullBranch)))

	// Pull Request Reviews
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/pulls/{pull_number}/reviews", pullHandler.ListPullReviews)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}", pullHandler.GetPullReview)
	s.mux.Handle("POST /repos/{owner}/{repo}/pulls/{pull_number}/reviews", requireAuth(http.HandlerFunc(pullHandler.CreatePullReview)))
	s.mux.Handle("PUT /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}", requireAuth(http.HandlerFunc(pullHandler.UpdatePullReview)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}", requireAuth(http.HandlerFunc(pullHandler.DeletePullReview)))
	s.mux.Handle("POST /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/events", requireAuth(http.HandlerFunc(pullHandler.SubmitPullReview)))
	s.mux.Handle("PUT /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/dismissals", requireAuth(http.HandlerFunc(pullHandler.DismissPullReview)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/comments", pullHandler.ListReviewComments)

	// Pull Request Review Comments
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/pulls/{pull_number}/comments", pullHandler.ListPullReviewComments)
	s.mux.Handle("POST /repos/{owner}/{repo}/pulls/{pull_number}/comments", requireAuth(http.HandlerFunc(pullHandler.CreatePullReviewComment)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/pulls/comments/{comment_id}", pullHandler.GetPullReviewComment)
	s.mux.Handle("PATCH /repos/{owner}/{repo}/pulls/comments/{comment_id}", requireAuth(http.HandlerFunc(pullHandler.UpdatePullReviewComment)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/pulls/comments/{comment_id}", requireAuth(http.HandlerFunc(pullHandler.DeletePullReviewComment)))

	// Requested Reviewers
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers", pullHandler.ListRequestedReviewers)
	s.mux.Handle("POST /repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers", requireAuth(http.HandlerFunc(pullHandler.RequestReviewers)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers", requireAuth(http.HandlerFunc(pullHandler.RemoveRequestedReviewers)))

	// ==========================================================================
	// Labels
	// ==========================================================================
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/labels", labelHandler.ListRepoLabels)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/labels/{name}", labelHandler.GetLabel)
	s.mux.Handle("POST /repos/{owner}/{repo}/labels", requireAuth(http.HandlerFunc(labelHandler.CreateLabel)))
	s.mux.Handle("PATCH /repos/{owner}/{repo}/labels/{name}", requireAuth(http.HandlerFunc(labelHandler.UpdateLabel)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/labels/{name}", requireAuth(http.HandlerFunc(labelHandler.DeleteLabel)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/issues/{issue_number}/labels", labelHandler.ListIssueLabels)
	s.mux.Handle("POST /repos/{owner}/{repo}/issues/{issue_number}/labels", requireAuth(http.HandlerFunc(labelHandler.AddIssueLabels)))
	s.mux.Handle("PUT /repos/{owner}/{repo}/issues/{issue_number}/labels", requireAuth(http.HandlerFunc(labelHandler.SetIssueLabels)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/issues/{issue_number}/labels", requireAuth(http.HandlerFunc(labelHandler.RemoveAllIssueLabels)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/issues/{issue_number}/labels/{name}", requireAuth(http.HandlerFunc(labelHandler.RemoveIssueLabel)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/milestones/{milestone_number}/labels", labelHandler.ListLabelsForMilestone)

	// ==========================================================================
	// Milestones
	// ==========================================================================
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/milestones", milestoneHandler.ListMilestones)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/milestones/{milestone_number}", milestoneHandler.GetMilestone)
	s.mux.Handle("POST /repos/{owner}/{repo}/milestones", requireAuth(http.HandlerFunc(milestoneHandler.CreateMilestone)))
	s.mux.Handle("PATCH /repos/{owner}/{repo}/milestones/{milestone_number}", requireAuth(http.HandlerFunc(milestoneHandler.UpdateMilestone)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/milestones/{milestone_number}", requireAuth(http.HandlerFunc(milestoneHandler.DeleteMilestone)))

	// ==========================================================================
	// Comments (Issue & Commit)
	// ==========================================================================
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/issues/{issue_number}/comments", commentHandler.ListIssueComments)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/issues/comments/{comment_id}", commentHandler.GetIssueComment)
	s.mux.Handle("POST /repos/{owner}/{repo}/issues/{issue_number}/comments", requireAuth(http.HandlerFunc(commentHandler.CreateIssueComment)))
	s.mux.Handle("PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}", requireAuth(http.HandlerFunc(commentHandler.UpdateIssueComment)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/issues/comments/{comment_id}", requireAuth(http.HandlerFunc(commentHandler.DeleteIssueComment)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/issues/comments", commentHandler.ListRepoComments)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/commits/{commit_sha}/comments", commentHandler.ListCommitComments)
	s.mux.Handle("POST /repos/{owner}/{repo}/commits/{commit_sha}/comments", requireAuth(http.HandlerFunc(commentHandler.CreateCommitComment)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/comments/{comment_id}", commentHandler.GetCommitComment)
	s.mux.Handle("PATCH /repos/{owner}/{repo}/comments/{comment_id}", requireAuth(http.HandlerFunc(commentHandler.UpdateCommitComment)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/comments/{comment_id}", requireAuth(http.HandlerFunc(commentHandler.DeleteCommitComment)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/comments", commentHandler.ListRepoCommitComments)

	// ==========================================================================
	// Teams
	// ==========================================================================
	s.mux.HandleFunc("GET /orgs/{org}/teams", teamHandler.ListOrgTeams)
	s.mux.HandleFunc("GET /orgs/{org}/teams/{team_slug}", teamHandler.GetOrgTeam)
	s.mux.Handle("POST /orgs/{org}/teams", requireAuth(http.HandlerFunc(teamHandler.CreateTeam)))
	s.mux.Handle("PATCH /orgs/{org}/teams/{team_slug}", requireAuth(http.HandlerFunc(teamHandler.UpdateTeam)))
	s.mux.Handle("DELETE /orgs/{org}/teams/{team_slug}", requireAuth(http.HandlerFunc(teamHandler.DeleteTeam)))
	s.mux.HandleFunc("GET /orgs/{org}/teams/{team_slug}/members", teamHandler.ListTeamMembers)
	s.mux.Handle("GET /orgs/{org}/teams/{team_slug}/memberships/{username}", requireAuth(http.HandlerFunc(teamHandler.GetTeamMembership)))
	s.mux.Handle("PUT /orgs/{org}/teams/{team_slug}/memberships/{username}", requireAuth(http.HandlerFunc(teamHandler.AddTeamMember)))
	s.mux.Handle("DELETE /orgs/{org}/teams/{team_slug}/memberships/{username}", requireAuth(http.HandlerFunc(teamHandler.RemoveTeamMember)))
	s.mux.HandleFunc("GET /orgs/{org}/teams/{team_slug}/repos", teamHandler.ListTeamRepos)
	s.mux.HandleFunc("GET /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}", teamHandler.CheckTeamRepoPermission)
	s.mux.Handle("PUT /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}", requireAuth(http.HandlerFunc(teamHandler.AddTeamRepo)))
	s.mux.Handle("DELETE /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}", requireAuth(http.HandlerFunc(teamHandler.RemoveTeamRepo)))
	s.mux.HandleFunc("GET /orgs/{org}/teams/{team_slug}/teams", teamHandler.ListChildTeams)
	s.mux.Handle("GET /user/teams", requireAuth(http.HandlerFunc(teamHandler.ListAuthenticatedUserTeams)))

	// ==========================================================================
	// Releases
	// ==========================================================================
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/releases", releaseHandler.ListReleases)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/releases/{release_id}", releaseHandler.GetRelease)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/releases/latest", releaseHandler.GetLatestRelease)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/releases/tags/{tag}", releaseHandler.GetReleaseByTag)
	s.mux.Handle("POST /repos/{owner}/{repo}/releases", requireAuth(http.HandlerFunc(releaseHandler.CreateRelease)))
	s.mux.Handle("PATCH /repos/{owner}/{repo}/releases/{release_id}", requireAuth(http.HandlerFunc(releaseHandler.UpdateRelease)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/releases/{release_id}", requireAuth(http.HandlerFunc(releaseHandler.DeleteRelease)))
	s.mux.Handle("POST /repos/{owner}/{repo}/releases/generate-notes", requireAuth(http.HandlerFunc(releaseHandler.GenerateReleaseNotes)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/releases/{release_id}/assets", releaseHandler.ListReleaseAssets)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/releases/assets/{asset_id}", releaseHandler.GetReleaseAsset)
	s.mux.Handle("PATCH /repos/{owner}/{repo}/releases/assets/{asset_id}", requireAuth(http.HandlerFunc(releaseHandler.UpdateReleaseAsset)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/releases/assets/{asset_id}", requireAuth(http.HandlerFunc(releaseHandler.DeleteReleaseAsset)))
	s.mux.Handle("POST /repos/{owner}/{repo}/releases/{release_id}/assets", requireAuth(http.HandlerFunc(releaseHandler.UploadReleaseAsset)))

	// ==========================================================================
	// Stars
	// ==========================================================================
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/stargazers", starHandler.ListStargazers)
	s.mux.HandleFunc("GET /users/{username}/starred", starHandler.ListStarredRepos)
	s.mux.Handle("GET /user/starred", requireAuth(http.HandlerFunc(starHandler.ListAuthenticatedUserStarredRepos)))
	s.mux.Handle("GET /user/starred/{owner}/{repo}", requireAuth(http.HandlerFunc(starHandler.CheckRepoStarred)))
	s.mux.Handle("PUT /user/starred/{owner}/{repo}", requireAuth(http.HandlerFunc(starHandler.StarRepo)))
	s.mux.Handle("DELETE /user/starred/{owner}/{repo}", requireAuth(http.HandlerFunc(starHandler.UnstarRepo)))

	// ==========================================================================
	// Watches (Subscriptions)
	// ==========================================================================
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/subscribers", watchHandler.ListWatchers)
	s.mux.Handle("GET /repos/{owner}/{repo}/subscription", requireAuth(http.HandlerFunc(watchHandler.GetSubscription)))
	s.mux.Handle("PUT /repos/{owner}/{repo}/subscription", requireAuth(http.HandlerFunc(watchHandler.SetSubscription)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/subscription", requireAuth(http.HandlerFunc(watchHandler.DeleteSubscription)))
	s.mux.HandleFunc("GET /users/{username}/subscriptions", watchHandler.ListWatchedRepos)
	s.mux.Handle("GET /user/subscriptions", requireAuth(http.HandlerFunc(watchHandler.ListAuthenticatedUserWatchedRepos)))

	// ==========================================================================
	// Webhooks
	// ==========================================================================
	s.mux.Handle("GET /repos/{owner}/{repo}/hooks", requireAuth(http.HandlerFunc(webhookHandler.ListRepoWebhooks)))
	s.mux.Handle("GET /repos/{owner}/{repo}/hooks/{hook_id}", requireAuth(http.HandlerFunc(webhookHandler.GetRepoWebhook)))
	s.mux.Handle("POST /repos/{owner}/{repo}/hooks", requireAuth(http.HandlerFunc(webhookHandler.CreateRepoWebhook)))
	s.mux.Handle("PATCH /repos/{owner}/{repo}/hooks/{hook_id}", requireAuth(http.HandlerFunc(webhookHandler.UpdateRepoWebhook)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/hooks/{hook_id}", requireAuth(http.HandlerFunc(webhookHandler.DeleteRepoWebhook)))
	s.mux.Handle("POST /repos/{owner}/{repo}/hooks/{hook_id}/pings", requireAuth(http.HandlerFunc(webhookHandler.PingRepoWebhook)))
	s.mux.Handle("POST /repos/{owner}/{repo}/hooks/{hook_id}/tests", requireAuth(http.HandlerFunc(webhookHandler.TestRepoWebhook)))
	s.mux.Handle("GET /repos/{owner}/{repo}/hooks/{hook_id}/deliveries", requireAuth(http.HandlerFunc(webhookHandler.ListWebhookDeliveries)))
	s.mux.Handle("GET /repos/{owner}/{repo}/hooks/{hook_id}/deliveries/{delivery_id}", requireAuth(http.HandlerFunc(webhookHandler.GetWebhookDelivery)))
	s.mux.Handle("POST /repos/{owner}/{repo}/hooks/{hook_id}/deliveries/{delivery_id}/attempts", requireAuth(http.HandlerFunc(webhookHandler.RedeliverWebhook)))
	s.mux.Handle("GET /orgs/{org}/hooks", requireAuth(http.HandlerFunc(webhookHandler.ListOrgWebhooks)))
	s.mux.Handle("GET /orgs/{org}/hooks/{hook_id}", requireAuth(http.HandlerFunc(webhookHandler.GetOrgWebhook)))
	s.mux.Handle("POST /orgs/{org}/hooks", requireAuth(http.HandlerFunc(webhookHandler.CreateOrgWebhook)))
	s.mux.Handle("PATCH /orgs/{org}/hooks/{hook_id}", requireAuth(http.HandlerFunc(webhookHandler.UpdateOrgWebhook)))
	s.mux.Handle("DELETE /orgs/{org}/hooks/{hook_id}", requireAuth(http.HandlerFunc(webhookHandler.DeleteOrgWebhook)))
	s.mux.Handle("POST /orgs/{org}/hooks/{hook_id}/pings", requireAuth(http.HandlerFunc(webhookHandler.PingOrgWebhook)))

	// ==========================================================================
	// Notifications
	// ==========================================================================
	s.mux.Handle("GET /notifications", requireAuth(http.HandlerFunc(notificationHandler.ListNotifications)))
	s.mux.Handle("PUT /notifications", requireAuth(http.HandlerFunc(notificationHandler.MarkAllAsRead)))
	s.mux.Handle("GET /notifications/threads/{thread_id}", requireAuth(http.HandlerFunc(notificationHandler.GetThread)))
	s.mux.Handle("PATCH /notifications/threads/{thread_id}", requireAuth(http.HandlerFunc(notificationHandler.MarkThreadAsRead)))
	s.mux.Handle("DELETE /notifications/threads/{thread_id}", requireAuth(http.HandlerFunc(notificationHandler.MarkThreadAsDone)))
	s.mux.Handle("GET /notifications/threads/{thread_id}/subscription", requireAuth(http.HandlerFunc(notificationHandler.GetThreadSubscription)))
	s.mux.Handle("PUT /notifications/threads/{thread_id}/subscription", requireAuth(http.HandlerFunc(notificationHandler.SetThreadSubscription)))
	s.mux.Handle("DELETE /notifications/threads/{thread_id}/subscription", requireAuth(http.HandlerFunc(notificationHandler.DeleteThreadSubscription)))
	s.mux.Handle("GET /repos/{owner}/{repo}/notifications", requireAuth(http.HandlerFunc(notificationHandler.ListRepoNotifications)))
	s.mux.Handle("PUT /repos/{owner}/{repo}/notifications", requireAuth(http.HandlerFunc(notificationHandler.MarkRepoNotificationsAsRead)))

	// ==========================================================================
	// Reactions
	// ==========================================================================
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/issues/{issue_number}/reactions", reactionHandler.ListIssueReactions)
	s.mux.Handle("POST /repos/{owner}/{repo}/issues/{issue_number}/reactions", requireAuth(http.HandlerFunc(reactionHandler.CreateIssueReaction)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/issues/{issue_number}/reactions/{reaction_id}", requireAuth(http.HandlerFunc(reactionHandler.DeleteIssueReaction)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/issues/comments/{comment_id}/reactions", reactionHandler.ListIssueCommentReactions)
	s.mux.Handle("POST /repos/{owner}/{repo}/issues/comments/{comment_id}/reactions", requireAuth(http.HandlerFunc(reactionHandler.CreateIssueCommentReaction)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/issues/comments/{comment_id}/reactions/{reaction_id}", requireAuth(http.HandlerFunc(reactionHandler.DeleteIssueCommentReaction)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions", reactionHandler.ListPullReviewCommentReactions)
	s.mux.Handle("POST /repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions", requireAuth(http.HandlerFunc(reactionHandler.CreatePullReviewCommentReaction)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions/{reaction_id}", requireAuth(http.HandlerFunc(reactionHandler.DeletePullReviewCommentReaction)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/comments/{comment_id}/reactions", reactionHandler.ListCommitCommentReactions)
	s.mux.Handle("POST /repos/{owner}/{repo}/comments/{comment_id}/reactions", requireAuth(http.HandlerFunc(reactionHandler.CreateCommitCommentReaction)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/comments/{comment_id}/reactions/{reaction_id}", requireAuth(http.HandlerFunc(reactionHandler.DeleteCommitCommentReaction)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/releases/{release_id}/reactions", reactionHandler.ListReleaseReactions)
	s.mux.Handle("POST /repos/{owner}/{repo}/releases/{release_id}/reactions", requireAuth(http.HandlerFunc(reactionHandler.CreateReleaseReaction)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/releases/{release_id}/reactions/{reaction_id}", requireAuth(http.HandlerFunc(reactionHandler.DeleteReleaseReaction)))

	// ==========================================================================
	// Collaborators
	// ==========================================================================
	s.mux.Handle("GET /repos/{owner}/{repo}/collaborators", requireAuth(http.HandlerFunc(collaboratorHandler.ListCollaborators)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/collaborators/{username}", collaboratorHandler.CheckCollaborator)
	s.mux.Handle("PUT /repos/{owner}/{repo}/collaborators/{username}", requireAuth(http.HandlerFunc(collaboratorHandler.AddCollaborator)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/collaborators/{username}", requireAuth(http.HandlerFunc(collaboratorHandler.RemoveCollaborator)))
	s.mux.Handle("GET /repos/{owner}/{repo}/collaborators/{username}/permission", requireAuth(http.HandlerFunc(collaboratorHandler.GetCollaboratorPermission)))
	s.mux.Handle("GET /repos/{owner}/{repo}/invitations", requireAuth(http.HandlerFunc(collaboratorHandler.ListInvitations)))
	s.mux.Handle("PATCH /repos/{owner}/{repo}/invitations/{invitation_id}", requireAuth(http.HandlerFunc(collaboratorHandler.UpdateInvitation)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/invitations/{invitation_id}", requireAuth(http.HandlerFunc(collaboratorHandler.DeleteInvitation)))
	s.mux.Handle("GET /user/repository_invitations", requireAuth(http.HandlerFunc(collaboratorHandler.ListUserInvitations)))
	s.mux.Handle("PATCH /user/repository_invitations/{invitation_id}", requireAuth(http.HandlerFunc(collaboratorHandler.AcceptInvitation)))
	s.mux.Handle("DELETE /user/repository_invitations/{invitation_id}", requireAuth(http.HandlerFunc(collaboratorHandler.DeclineInvitation)))

	// ==========================================================================
	// Branches
	// ==========================================================================
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/branches", branchHandler.ListBranches)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/branches/{branch}", branchHandler.GetBranch)
	s.mux.Handle("POST /repos/{owner}/{repo}/branches/{branch}/rename", requireAuth(http.HandlerFunc(branchHandler.RenameBranch)))
	s.mux.Handle("POST /repos/{owner}/{repo}/merge-upstream", requireAuth(http.HandlerFunc(branchHandler.SyncFork)))
	s.mux.Handle("GET /repos/{owner}/{repo}/branches/{branch}/protection", requireAuth(http.HandlerFunc(branchHandler.GetBranchProtection)))
	s.mux.Handle("PUT /repos/{owner}/{repo}/branches/{branch}/protection", requireAuth(http.HandlerFunc(branchHandler.UpdateBranchProtection)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/branches/{branch}/protection", requireAuth(http.HandlerFunc(branchHandler.DeleteBranchProtection)))
	s.mux.Handle("GET /repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks", requireAuth(http.HandlerFunc(branchHandler.GetRequiredStatusChecks)))
	s.mux.Handle("PATCH /repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks", requireAuth(http.HandlerFunc(branchHandler.UpdateRequiredStatusChecks)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks", requireAuth(http.HandlerFunc(branchHandler.DeleteRequiredStatusChecks)))
	s.mux.Handle("GET /repos/{owner}/{repo}/branches/{branch}/protection/required_pull_request_reviews", requireAuth(http.HandlerFunc(branchHandler.GetRequiredPullRequestReviews)))
	s.mux.Handle("PATCH /repos/{owner}/{repo}/branches/{branch}/protection/required_pull_request_reviews", requireAuth(http.HandlerFunc(branchHandler.UpdateRequiredPullRequestReviews)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/branches/{branch}/protection/required_pull_request_reviews", requireAuth(http.HandlerFunc(branchHandler.DeleteRequiredPullRequestReviews)))
	s.mux.Handle("GET /repos/{owner}/{repo}/branches/{branch}/protection/enforce_admins", requireAuth(http.HandlerFunc(branchHandler.GetAdminEnforcement)))
	s.mux.Handle("POST /repos/{owner}/{repo}/branches/{branch}/protection/enforce_admins", requireAuth(http.HandlerFunc(branchHandler.SetAdminEnforcement)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/branches/{branch}/protection/enforce_admins", requireAuth(http.HandlerFunc(branchHandler.DeleteAdminEnforcement)))

	// ==========================================================================
	// Commits
	// ==========================================================================
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/commits", commitHandler.ListCommits)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/commits/{ref}", commitHandler.GetCommit)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/compare/{basehead}", commitHandler.CompareCommits)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/commits/{commit_sha}/branches-where-head", commitHandler.ListBranchesForHead)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/commits/{commit_sha}/pulls", commitHandler.ListPullsForCommit)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/commits/{ref}/status", commitHandler.GetCombinedStatus)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/commits/{ref}/statuses", commitHandler.ListStatuses)
	s.mux.Handle("POST /repos/{owner}/{repo}/statuses/{sha}", requireAuth(http.HandlerFunc(commitHandler.CreateStatus)))

	// ==========================================================================
	// Git Data (Low-level)
	// ==========================================================================
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/git/blobs/{file_sha}", gitHandler.GetBlob)
	s.mux.Handle("POST /repos/{owner}/{repo}/git/blobs", requireAuth(http.HandlerFunc(gitHandler.CreateBlob)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/git/commits/{commit_sha}", gitHandler.GetGitCommit)
	s.mux.Handle("POST /repos/{owner}/{repo}/git/commits", requireAuth(http.HandlerFunc(gitHandler.CreateGitCommit)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/git/ref/{ref...}", gitHandler.GetRef)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/git/matching-refs/{ref...}", gitHandler.ListMatchingRefs)
	s.mux.Handle("POST /repos/{owner}/{repo}/git/refs", requireAuth(http.HandlerFunc(gitHandler.CreateRef)))
	s.mux.Handle("PATCH /repos/{owner}/{repo}/git/refs/{ref...}", requireAuth(http.HandlerFunc(gitHandler.UpdateRef)))
	s.mux.Handle("DELETE /repos/{owner}/{repo}/git/refs/{ref...}", requireAuth(http.HandlerFunc(gitHandler.DeleteRef)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/git/trees/{tree_sha}", gitHandler.GetTree)
	s.mux.Handle("POST /repos/{owner}/{repo}/git/trees", requireAuth(http.HandlerFunc(gitHandler.CreateTree)))
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/git/tags/{tag_sha}", gitHandler.GetTag)
	s.mux.Handle("POST /repos/{owner}/{repo}/git/tags", requireAuth(http.HandlerFunc(gitHandler.CreateTag)))

	// ==========================================================================
	// Search
	// ==========================================================================
	s.mux.HandleFunc("GET /search/repositories", searchHandler.SearchRepositories)
	s.mux.HandleFunc("GET /search/code", searchHandler.SearchCode)
	s.mux.HandleFunc("GET /search/commits", searchHandler.SearchCommits)
	s.mux.HandleFunc("GET /search/issues", searchHandler.SearchIssues)
	s.mux.HandleFunc("GET /search/users", searchHandler.SearchUsers)
	s.mux.HandleFunc("GET /search/labels", searchHandler.SearchLabels)
	s.mux.HandleFunc("GET /search/topics", searchHandler.SearchTopics)

	// ==========================================================================
	// Activity (Events & Feeds)
	// ==========================================================================
	s.mux.HandleFunc("GET /events", activityHandler.ListPublicEvents)
	s.mux.HandleFunc("GET /repos/{owner}/{repo}/events", activityHandler.ListRepoEvents)
	s.mux.HandleFunc("GET /networks/{owner}/{repo}/events", activityHandler.ListRepoNetworkEvents)
	s.mux.HandleFunc("GET /orgs/{org}/events", activityHandler.ListOrgEvents)
	s.mux.HandleFunc("GET /users/{username}/received_events", activityHandler.ListUserReceivedEvents)
	s.mux.HandleFunc("GET /users/{username}/received_events/public", activityHandler.ListUserReceivedPublicEvents)
	s.mux.HandleFunc("GET /users/{username}/events", activityHandler.ListUserEvents)
	s.mux.HandleFunc("GET /users/{username}/events/public", activityHandler.ListUserPublicEvents)
	s.mux.Handle("GET /users/{username}/events/orgs/{org}", requireAuth(http.HandlerFunc(activityHandler.ListUserOrgEvents)))
	s.mux.Handle("GET /feeds", optionalAuth(http.HandlerFunc(activityHandler.ListFeeds)))
}
