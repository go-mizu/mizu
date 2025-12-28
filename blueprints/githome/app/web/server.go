package web

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/app/web/handler/api"
	"github.com/go-mizu/blueprints/githome/feature/activities"
	"github.com/go-mizu/blueprints/githome/feature/branches"
	"github.com/go-mizu/blueprints/githome/feature/collaborators"
	"github.com/go-mizu/blueprints/githome/feature/comments"
	"github.com/go-mizu/blueprints/githome/feature/commits"
	"github.com/go-mizu/blueprints/githome/feature/git"
	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/labels"
	"github.com/go-mizu/blueprints/githome/feature/milestones"
	"github.com/go-mizu/blueprints/githome/feature/notifications"
	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/pulls"
	"github.com/go-mizu/blueprints/githome/feature/reactions"
	"github.com/go-mizu/blueprints/githome/feature/releases"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/search"
	"github.com/go-mizu/blueprints/githome/feature/stars"
	"github.com/go-mizu/blueprints/githome/feature/teams"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/feature/watches"
	"github.com/go-mizu/blueprints/githome/feature/webhooks"
	"github.com/go-mizu/mizu"
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
	app      *mizu.App
	services *Services
}

// NewServer creates a new server with all routes configured
func NewServer(services *Services) *Server {
	s := &Server{
		app:      mizu.New(),
		services: services,
	}
	s.setupRoutes()
	return s
}

// Handler returns the HTTP handler
func (s *Server) Handler() http.Handler {
	return s.app
}

// App returns the mizu application
func (s *Server) App() *mizu.App {
	return s.app
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

	// Auth middleware
	requireAuth := api.RequireAuth(s.services.Users)
	optionalAuth := api.OptionalAuth(s.services.Users)

	r := s.app.Router
	auth := r.With(requireAuth)
	optAuth := r.With(optionalAuth)

	// ==========================================================================
	// Authentication
	// ==========================================================================
	r.Post("/login", authHandler.Login)
	r.Post("/register", authHandler.Register)

	// ==========================================================================
	// Users
	// ==========================================================================
	auth.Get("/user", userHandler.GetAuthenticatedUser)
	auth.Patch("/user", userHandler.UpdateAuthenticatedUser)
	r.Get("/users", userHandler.ListUsers)
	r.Get("/users/{username}", userHandler.GetUser)
	r.Get("/users/{username}/followers", userHandler.ListFollowers)
	r.Get("/users/{username}/following", userHandler.ListFollowing)
	r.Get("/users/{username}/following/{target_user}", userHandler.CheckFollowing)
	auth.Get("/user/followers", userHandler.ListAuthenticatedUserFollowers)
	auth.Get("/user/following", userHandler.ListAuthenticatedUserFollowing)
	auth.Get("/user/following/{username}", userHandler.CheckAuthenticatedUserFollowing)
	auth.Put("/user/following/{username}", userHandler.FollowUser)
	auth.Delete("/user/following/{username}", userHandler.UnfollowUser)

	// ==========================================================================
	// Organizations
	// ==========================================================================
	r.Get("/organizations", orgHandler.ListOrgs)
	r.Get("/orgs/{org}", orgHandler.GetOrg)
	auth.Patch("/orgs/{org}", orgHandler.UpdateOrg)
	auth.Get("/user/orgs", orgHandler.ListAuthenticatedUserOrgs)
	r.Get("/users/{username}/orgs", orgHandler.ListUserOrgs)
	r.Get("/orgs/{org}/members", orgHandler.ListOrgMembers)
	r.Get("/orgs/{org}/members/{username}", orgHandler.CheckOrgMember)
	auth.Delete("/orgs/{org}/members/{username}", orgHandler.RemoveOrgMember)
	auth.Get("/orgs/{org}/memberships/{username}", orgHandler.GetOrgMembership)
	auth.Put("/orgs/{org}/memberships/{username}", orgHandler.SetOrgMembership)
	auth.Delete("/orgs/{org}/memberships/{username}", orgHandler.RemoveOrgMembership)
	auth.Get("/orgs/{org}/outside_collaborators", orgHandler.ListOutsideCollaborators)
	r.Get("/orgs/{org}/public_members", orgHandler.ListPublicOrgMembers)
	r.Get("/orgs/{org}/public_members/{username}", orgHandler.CheckPublicOrgMember)
	auth.Put("/orgs/{org}/public_members/{username}", orgHandler.PublicizeMembership)
	auth.Delete("/orgs/{org}/public_members/{username}", orgHandler.ConcealMembership)
	auth.Get("/user/memberships/orgs/{org}", orgHandler.GetAuthenticatedUserOrgMembership)
	auth.Patch("/user/memberships/orgs/{org}", orgHandler.UpdateAuthenticatedUserOrgMembership)

	// ==========================================================================
	// Repositories
	// ==========================================================================
	r.Get("/repositories", repoHandler.ListPublicRepos)
	auth.Get("/user/repos", repoHandler.ListAuthenticatedUserRepos)
	auth.Post("/user/repos", repoHandler.CreateAuthenticatedUserRepo)
	r.Get("/users/{username}/repos", repoHandler.ListUserRepos)
	r.Get("/orgs/{org}/repos", repoHandler.ListOrgRepos)
	auth.Post("/orgs/{org}/repos", repoHandler.CreateOrgRepo)
	optAuth.Get("/repos/{owner}/{repo}", repoHandler.GetRepo)
	auth.Patch("/repos/{owner}/{repo}", repoHandler.UpdateRepo)
	auth.Delete("/repos/{owner}/{repo}", repoHandler.DeleteRepo)
	r.Get("/repos/{owner}/{repo}/topics", repoHandler.ListRepoTopics)
	auth.Put("/repos/{owner}/{repo}/topics", repoHandler.ReplaceRepoTopics)
	r.Get("/repos/{owner}/{repo}/languages", repoHandler.ListRepoLanguages)
	r.Get("/repos/{owner}/{repo}/contributors", repoHandler.ListRepoContributors)
	r.Get("/repos/{owner}/{repo}/tags", repoHandler.ListRepoTags)
	auth.Post("/repos/{owner}/{repo}/transfer", repoHandler.TransferRepo)
	r.Get("/repos/{owner}/{repo}/readme", repoHandler.GetRepoReadme)
	r.Get("/repos/{owner}/{repo}/contents/{path...}", repoHandler.GetRepoContent)
	auth.Put("/repos/{owner}/{repo}/contents/{path...}", repoHandler.CreateOrUpdateFileContent)
	auth.Delete("/repos/{owner}/{repo}/contents/{path...}", repoHandler.DeleteFileContent)
	auth.Post("/repos/{owner}/{repo}/forks", repoHandler.ForkRepo)
	r.Get("/repos/{owner}/{repo}/forks", repoHandler.ListForks)

	// ==========================================================================
	// Issues
	// ==========================================================================
	auth.Get("/issues", issueHandler.ListIssues)
	auth.Get("/user/issues", issueHandler.ListAuthenticatedUserIssues)
	auth.Get("/orgs/{org}/issues", issueHandler.ListOrgIssues)
	r.Get("/repos/{owner}/{repo}/issues", issueHandler.ListRepoIssues)
	r.Get("/repos/{owner}/{repo}/issues/{issue_number}", issueHandler.GetIssue)
	auth.Post("/repos/{owner}/{repo}/issues", issueHandler.CreateIssue)
	auth.Patch("/repos/{owner}/{repo}/issues/{issue_number}", issueHandler.UpdateIssue)
	auth.Put("/repos/{owner}/{repo}/issues/{issue_number}/lock", issueHandler.LockIssue)
	auth.Delete("/repos/{owner}/{repo}/issues/{issue_number}/lock", issueHandler.UnlockIssue)
	r.Get("/repos/{owner}/{repo}/assignees", issueHandler.ListIssueAssignees)
	r.Get("/repos/{owner}/{repo}/assignees/{assignee}", issueHandler.CheckAssignee)
	auth.Post("/repos/{owner}/{repo}/issues/{issue_number}/assignees", issueHandler.AddAssignees)
	auth.Delete("/repos/{owner}/{repo}/issues/{issue_number}/assignees", issueHandler.RemoveAssignees)
	r.Get("/repos/{owner}/{repo}/issues/{issue_number}/events", issueHandler.ListIssueEvents)
	r.Get("/repos/{owner}/{repo}/issues/events/{event_id}", issueHandler.GetIssueEvent)
	r.Get("/repos/{owner}/{repo}/issues/events", issueHandler.ListRepoIssueEvents)
	r.Get("/repos/{owner}/{repo}/issues/{issue_number}/timeline", issueHandler.ListIssueTimeline)

	// ==========================================================================
	// Pull Requests
	// ==========================================================================
	r.Get("/repos/{owner}/{repo}/pulls", pullHandler.ListPulls)
	r.Get("/repos/{owner}/{repo}/pulls/{pull_number}", pullHandler.GetPull)
	auth.Post("/repos/{owner}/{repo}/pulls", pullHandler.CreatePull)
	auth.Patch("/repos/{owner}/{repo}/pulls/{pull_number}", pullHandler.UpdatePull)
	r.Get("/repos/{owner}/{repo}/pulls/{pull_number}/commits", pullHandler.ListPullCommits)
	r.Get("/repos/{owner}/{repo}/pulls/{pull_number}/files", pullHandler.ListPullFiles)
	r.Get("/repos/{owner}/{repo}/pulls/{pull_number}/merge", pullHandler.CheckPullMerged)
	auth.Put("/repos/{owner}/{repo}/pulls/{pull_number}/merge", pullHandler.MergePull)
	auth.Put("/repos/{owner}/{repo}/pulls/{pull_number}/update-branch", pullHandler.UpdatePullBranch)

	// Pull Request Reviews
	r.Get("/repos/{owner}/{repo}/pulls/{pull_number}/reviews", pullHandler.ListPullReviews)
	r.Get("/repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}", pullHandler.GetPullReview)
	auth.Post("/repos/{owner}/{repo}/pulls/{pull_number}/reviews", pullHandler.CreatePullReview)
	auth.Put("/repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}", pullHandler.UpdatePullReview)
	auth.Delete("/repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}", pullHandler.DeletePullReview)
	auth.Post("/repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/events", pullHandler.SubmitPullReview)
	auth.Put("/repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/dismissals", pullHandler.DismissPullReview)
	r.Get("/repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/comments", pullHandler.ListReviewComments)

	// Pull Request Review Comments
	r.Get("/repos/{owner}/{repo}/pulls/{pull_number}/comments", pullHandler.ListPullReviewComments)
	auth.Post("/repos/{owner}/{repo}/pulls/{pull_number}/comments", pullHandler.CreatePullReviewComment)
	r.Get("/repos/{owner}/{repo}/pulls/comments/{comment_id}", pullHandler.GetPullReviewComment)
	auth.Patch("/repos/{owner}/{repo}/pulls/comments/{comment_id}", pullHandler.UpdatePullReviewComment)
	auth.Delete("/repos/{owner}/{repo}/pulls/comments/{comment_id}", pullHandler.DeletePullReviewComment)

	// Requested Reviewers
	r.Get("/repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers", pullHandler.ListRequestedReviewers)
	auth.Post("/repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers", pullHandler.RequestReviewers)
	auth.Delete("/repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers", pullHandler.RemoveRequestedReviewers)

	// ==========================================================================
	// Labels
	// ==========================================================================
	r.Get("/repos/{owner}/{repo}/labels", labelHandler.ListRepoLabels)
	r.Get("/repos/{owner}/{repo}/labels/{name}", labelHandler.GetLabel)
	auth.Post("/repos/{owner}/{repo}/labels", labelHandler.CreateLabel)
	auth.Patch("/repos/{owner}/{repo}/labels/{name}", labelHandler.UpdateLabel)
	auth.Delete("/repos/{owner}/{repo}/labels/{name}", labelHandler.DeleteLabel)
	r.Get("/repos/{owner}/{repo}/issues/{issue_number}/labels", labelHandler.ListIssueLabels)
	auth.Post("/repos/{owner}/{repo}/issues/{issue_number}/labels", labelHandler.AddIssueLabels)
	auth.Put("/repos/{owner}/{repo}/issues/{issue_number}/labels", labelHandler.SetIssueLabels)
	auth.Delete("/repos/{owner}/{repo}/issues/{issue_number}/labels", labelHandler.RemoveAllIssueLabels)
	auth.Delete("/repos/{owner}/{repo}/issues/{issue_number}/labels/{name}", labelHandler.RemoveIssueLabel)
	r.Get("/repos/{owner}/{repo}/milestones/{milestone_number}/labels", labelHandler.ListLabelsForMilestone)

	// ==========================================================================
	// Milestones
	// ==========================================================================
	r.Get("/repos/{owner}/{repo}/milestones", milestoneHandler.ListMilestones)
	r.Get("/repos/{owner}/{repo}/milestones/{milestone_number}", milestoneHandler.GetMilestone)
	auth.Post("/repos/{owner}/{repo}/milestones", milestoneHandler.CreateMilestone)
	auth.Patch("/repos/{owner}/{repo}/milestones/{milestone_number}", milestoneHandler.UpdateMilestone)
	auth.Delete("/repos/{owner}/{repo}/milestones/{milestone_number}", milestoneHandler.DeleteMilestone)

	// ==========================================================================
	// Comments (Issue & Commit)
	// ==========================================================================
	r.Get("/repos/{owner}/{repo}/issues/{issue_number}/comments", commentHandler.ListIssueComments)
	r.Get("/repos/{owner}/{repo}/issues/comments/{comment_id}", commentHandler.GetIssueComment)
	auth.Post("/repos/{owner}/{repo}/issues/{issue_number}/comments", commentHandler.CreateIssueComment)
	auth.Patch("/repos/{owner}/{repo}/issues/comments/{comment_id}", commentHandler.UpdateIssueComment)
	auth.Delete("/repos/{owner}/{repo}/issues/comments/{comment_id}", commentHandler.DeleteIssueComment)
	r.Get("/repos/{owner}/{repo}/issues/comments", commentHandler.ListRepoComments)
	r.Get("/repos/{owner}/{repo}/commits/{commit_sha}/comments", commentHandler.ListCommitComments)
	auth.Post("/repos/{owner}/{repo}/commits/{commit_sha}/comments", commentHandler.CreateCommitComment)
	r.Get("/repos/{owner}/{repo}/comments/{comment_id}", commentHandler.GetCommitComment)
	auth.Patch("/repos/{owner}/{repo}/comments/{comment_id}", commentHandler.UpdateCommitComment)
	auth.Delete("/repos/{owner}/{repo}/comments/{comment_id}", commentHandler.DeleteCommitComment)
	r.Get("/repos/{owner}/{repo}/comments", commentHandler.ListRepoCommitComments)

	// ==========================================================================
	// Teams
	// ==========================================================================
	r.Get("/orgs/{org}/teams", teamHandler.ListOrgTeams)
	r.Get("/orgs/{org}/teams/{team_slug}", teamHandler.GetOrgTeam)
	auth.Post("/orgs/{org}/teams", teamHandler.CreateTeam)
	auth.Patch("/orgs/{org}/teams/{team_slug}", teamHandler.UpdateTeam)
	auth.Delete("/orgs/{org}/teams/{team_slug}", teamHandler.DeleteTeam)
	r.Get("/orgs/{org}/teams/{team_slug}/members", teamHandler.ListTeamMembers)
	auth.Get("/orgs/{org}/teams/{team_slug}/memberships/{username}", teamHandler.GetTeamMembership)
	auth.Put("/orgs/{org}/teams/{team_slug}/memberships/{username}", teamHandler.AddTeamMember)
	auth.Delete("/orgs/{org}/teams/{team_slug}/memberships/{username}", teamHandler.RemoveTeamMember)
	r.Get("/orgs/{org}/teams/{team_slug}/repos", teamHandler.ListTeamRepos)
	r.Get("/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}", teamHandler.CheckTeamRepoPermission)
	auth.Put("/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}", teamHandler.AddTeamRepo)
	auth.Delete("/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}", teamHandler.RemoveTeamRepo)
	r.Get("/orgs/{org}/teams/{team_slug}/teams", teamHandler.ListChildTeams)
	auth.Get("/user/teams", teamHandler.ListAuthenticatedUserTeams)

	// ==========================================================================
	// Releases
	// ==========================================================================
	r.Get("/repos/{owner}/{repo}/releases", releaseHandler.ListReleases)
	r.Get("/repos/{owner}/{repo}/releases/{release_id}", releaseHandler.GetRelease)
	r.Get("/repos/{owner}/{repo}/releases/latest", releaseHandler.GetLatestRelease)
	r.Get("/repos/{owner}/{repo}/releases/tags/{tag}", releaseHandler.GetReleaseByTag)
	auth.Post("/repos/{owner}/{repo}/releases", releaseHandler.CreateRelease)
	auth.Patch("/repos/{owner}/{repo}/releases/{release_id}", releaseHandler.UpdateRelease)
	auth.Delete("/repos/{owner}/{repo}/releases/{release_id}", releaseHandler.DeleteRelease)
	auth.Post("/repos/{owner}/{repo}/releases/generate-notes", releaseHandler.GenerateReleaseNotes)
	r.Get("/repos/{owner}/{repo}/releases/{release_id}/assets", releaseHandler.ListReleaseAssets)
	r.Get("/repos/{owner}/{repo}/releases/assets/{asset_id}", releaseHandler.GetReleaseAsset)
	auth.Patch("/repos/{owner}/{repo}/releases/assets/{asset_id}", releaseHandler.UpdateReleaseAsset)
	auth.Delete("/repos/{owner}/{repo}/releases/assets/{asset_id}", releaseHandler.DeleteReleaseAsset)
	auth.Post("/repos/{owner}/{repo}/releases/{release_id}/assets", releaseHandler.UploadReleaseAsset)
	r.Get("/repos/{owner}/{repo}/releases/assets/{asset_id}/download", releaseHandler.DownloadReleaseAsset)

	// ==========================================================================
	// Stars
	// ==========================================================================
	r.Get("/repos/{owner}/{repo}/stargazers", starHandler.ListStargazers)
	r.Get("/users/{username}/starred", starHandler.ListStarredRepos)
	auth.Get("/user/starred", starHandler.ListAuthenticatedUserStarredRepos)
	auth.Get("/user/starred/{owner}/{repo}", starHandler.CheckRepoStarred)
	auth.Put("/user/starred/{owner}/{repo}", starHandler.StarRepo)
	auth.Delete("/user/starred/{owner}/{repo}", starHandler.UnstarRepo)

	// ==========================================================================
	// Watches (Subscriptions)
	// ==========================================================================
	r.Get("/repos/{owner}/{repo}/subscribers", watchHandler.ListWatchers)
	auth.Get("/repos/{owner}/{repo}/subscription", watchHandler.GetSubscription)
	auth.Put("/repos/{owner}/{repo}/subscription", watchHandler.SetSubscription)
	auth.Delete("/repos/{owner}/{repo}/subscription", watchHandler.DeleteSubscription)
	r.Get("/users/{username}/subscriptions", watchHandler.ListWatchedRepos)
	auth.Get("/user/subscriptions", watchHandler.ListAuthenticatedUserWatchedRepos)

	// ==========================================================================
	// Webhooks
	// ==========================================================================
	auth.Get("/repos/{owner}/{repo}/hooks", webhookHandler.ListRepoWebhooks)
	auth.Get("/repos/{owner}/{repo}/hooks/{hook_id}", webhookHandler.GetRepoWebhook)
	auth.Post("/repos/{owner}/{repo}/hooks", webhookHandler.CreateRepoWebhook)
	auth.Patch("/repos/{owner}/{repo}/hooks/{hook_id}", webhookHandler.UpdateRepoWebhook)
	auth.Delete("/repos/{owner}/{repo}/hooks/{hook_id}", webhookHandler.DeleteRepoWebhook)
	auth.Post("/repos/{owner}/{repo}/hooks/{hook_id}/pings", webhookHandler.PingRepoWebhook)
	auth.Post("/repos/{owner}/{repo}/hooks/{hook_id}/tests", webhookHandler.TestRepoWebhook)
	auth.Get("/repos/{owner}/{repo}/hooks/{hook_id}/deliveries", webhookHandler.ListWebhookDeliveries)
	auth.Get("/repos/{owner}/{repo}/hooks/{hook_id}/deliveries/{delivery_id}", webhookHandler.GetWebhookDelivery)
	auth.Post("/repos/{owner}/{repo}/hooks/{hook_id}/deliveries/{delivery_id}/attempts", webhookHandler.RedeliverWebhook)
	auth.Get("/orgs/{org}/hooks", webhookHandler.ListOrgWebhooks)
	auth.Get("/orgs/{org}/hooks/{hook_id}", webhookHandler.GetOrgWebhook)
	auth.Post("/orgs/{org}/hooks", webhookHandler.CreateOrgWebhook)
	auth.Patch("/orgs/{org}/hooks/{hook_id}", webhookHandler.UpdateOrgWebhook)
	auth.Delete("/orgs/{org}/hooks/{hook_id}", webhookHandler.DeleteOrgWebhook)
	auth.Post("/orgs/{org}/hooks/{hook_id}/pings", webhookHandler.PingOrgWebhook)
	auth.Get("/orgs/{org}/hooks/{hook_id}/deliveries", webhookHandler.ListOrgWebhookDeliveries)
	auth.Get("/orgs/{org}/hooks/{hook_id}/deliveries/{delivery_id}", webhookHandler.GetOrgWebhookDelivery)
	auth.Post("/orgs/{org}/hooks/{hook_id}/deliveries/{delivery_id}/attempts", webhookHandler.RedeliverOrgWebhook)

	// ==========================================================================
	// Notifications
	// ==========================================================================
	auth.Get("/notifications", notificationHandler.ListNotifications)
	auth.Put("/notifications", notificationHandler.MarkAllAsRead)
	auth.Get("/notifications/threads/{thread_id}", notificationHandler.GetThread)
	auth.Patch("/notifications/threads/{thread_id}", notificationHandler.MarkThreadAsRead)
	auth.Delete("/notifications/threads/{thread_id}", notificationHandler.MarkThreadAsDone)
	auth.Get("/notifications/threads/{thread_id}/subscription", notificationHandler.GetThreadSubscription)
	auth.Put("/notifications/threads/{thread_id}/subscription", notificationHandler.SetThreadSubscription)
	auth.Delete("/notifications/threads/{thread_id}/subscription", notificationHandler.DeleteThreadSubscription)
	auth.Get("/repos/{owner}/{repo}/notifications", notificationHandler.ListRepoNotifications)
	auth.Put("/repos/{owner}/{repo}/notifications", notificationHandler.MarkRepoNotificationsAsRead)

	// ==========================================================================
	// Reactions
	// ==========================================================================
	r.Get("/repos/{owner}/{repo}/issues/{issue_number}/reactions", reactionHandler.ListIssueReactions)
	auth.Post("/repos/{owner}/{repo}/issues/{issue_number}/reactions", reactionHandler.CreateIssueReaction)
	auth.Delete("/repos/{owner}/{repo}/issues/{issue_number}/reactions/{reaction_id}", reactionHandler.DeleteIssueReaction)
	r.Get("/repos/{owner}/{repo}/issues/comments/{comment_id}/reactions", reactionHandler.ListIssueCommentReactions)
	auth.Post("/repos/{owner}/{repo}/issues/comments/{comment_id}/reactions", reactionHandler.CreateIssueCommentReaction)
	auth.Delete("/repos/{owner}/{repo}/issues/comments/{comment_id}/reactions/{reaction_id}", reactionHandler.DeleteIssueCommentReaction)
	r.Get("/repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions", reactionHandler.ListPullReviewCommentReactions)
	auth.Post("/repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions", reactionHandler.CreatePullReviewCommentReaction)
	auth.Delete("/repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions/{reaction_id}", reactionHandler.DeletePullReviewCommentReaction)
	r.Get("/repos/{owner}/{repo}/comments/{comment_id}/reactions", reactionHandler.ListCommitCommentReactions)
	auth.Post("/repos/{owner}/{repo}/comments/{comment_id}/reactions", reactionHandler.CreateCommitCommentReaction)
	auth.Delete("/repos/{owner}/{repo}/comments/{comment_id}/reactions/{reaction_id}", reactionHandler.DeleteCommitCommentReaction)
	r.Get("/repos/{owner}/{repo}/releases/{release_id}/reactions", reactionHandler.ListReleaseReactions)
	auth.Post("/repos/{owner}/{repo}/releases/{release_id}/reactions", reactionHandler.CreateReleaseReaction)
	auth.Delete("/repos/{owner}/{repo}/releases/{release_id}/reactions/{reaction_id}", reactionHandler.DeleteReleaseReaction)

	// ==========================================================================
	// Collaborators
	// ==========================================================================
	auth.Get("/repos/{owner}/{repo}/collaborators", collaboratorHandler.ListCollaborators)
	r.Get("/repos/{owner}/{repo}/collaborators/{username}", collaboratorHandler.CheckCollaborator)
	auth.Put("/repos/{owner}/{repo}/collaborators/{username}", collaboratorHandler.AddCollaborator)
	auth.Delete("/repos/{owner}/{repo}/collaborators/{username}", collaboratorHandler.RemoveCollaborator)
	auth.Get("/repos/{owner}/{repo}/collaborators/{username}/permission", collaboratorHandler.GetCollaboratorPermission)
	auth.Get("/repos/{owner}/{repo}/invitations", collaboratorHandler.ListInvitations)
	auth.Patch("/repos/{owner}/{repo}/invitations/{invitation_id}", collaboratorHandler.UpdateInvitation)
	auth.Delete("/repos/{owner}/{repo}/invitations/{invitation_id}", collaboratorHandler.DeleteInvitation)
	auth.Get("/user/repository_invitations", collaboratorHandler.ListUserInvitations)
	auth.Patch("/user/repository_invitations/{invitation_id}", collaboratorHandler.AcceptInvitation)
	auth.Delete("/user/repository_invitations/{invitation_id}", collaboratorHandler.DeclineInvitation)

	// ==========================================================================
	// Branches
	// ==========================================================================
	r.Get("/repos/{owner}/{repo}/branches", branchHandler.ListBranches)
	r.Get("/repos/{owner}/{repo}/branches/{branch}", branchHandler.GetBranch)
	auth.Post("/repos/{owner}/{repo}/branches/{branch}/rename", branchHandler.RenameBranch)
	auth.Get("/repos/{owner}/{repo}/branches/{branch}/protection", branchHandler.GetBranchProtection)
	auth.Put("/repos/{owner}/{repo}/branches/{branch}/protection", branchHandler.UpdateBranchProtection)
	auth.Delete("/repos/{owner}/{repo}/branches/{branch}/protection", branchHandler.DeleteBranchProtection)
	auth.Get("/repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks", branchHandler.GetRequiredStatusChecks)
	auth.Patch("/repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks", branchHandler.UpdateRequiredStatusChecks)
	auth.Delete("/repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks", branchHandler.DeleteRequiredStatusChecks)
	auth.Get("/repos/{owner}/{repo}/branches/{branch}/protection/required_signatures", branchHandler.GetRequiredSignatures)
	auth.Post("/repos/{owner}/{repo}/branches/{branch}/protection/required_signatures", branchHandler.CreateRequiredSignatures)
	auth.Delete("/repos/{owner}/{repo}/branches/{branch}/protection/required_signatures", branchHandler.DeleteRequiredSignatures)

	// ==========================================================================
	// Commits
	// ==========================================================================
	r.Get("/repos/{owner}/{repo}/commits", commitHandler.ListCommits)
	r.Get("/repos/{owner}/{repo}/commits/{ref}", commitHandler.GetCommit)
	r.Get("/repos/{owner}/{repo}/compare/{basehead}", commitHandler.CompareCommits)
	r.Get("/repos/{owner}/{repo}/commits/{commit_sha}/branches-where-head", commitHandler.ListBranchesForHead)
	r.Get("/repos/{owner}/{repo}/commits/{commit_sha}/pulls", commitHandler.ListPullsForCommit)
	r.Get("/repos/{owner}/{repo}/commits/{ref}/status", commitHandler.GetCombinedStatus)
	r.Get("/repos/{owner}/{repo}/commits/{ref}/statuses", commitHandler.ListStatuses)
	auth.Post("/repos/{owner}/{repo}/statuses/{sha}", commitHandler.CreateStatus)

	// ==========================================================================
	// Git Data (Low-level)
	// ==========================================================================
	r.Get("/repos/{owner}/{repo}/git/blobs/{file_sha}", gitHandler.GetBlob)
	auth.Post("/repos/{owner}/{repo}/git/blobs", gitHandler.CreateBlob)
	r.Get("/repos/{owner}/{repo}/git/commits/{commit_sha}", gitHandler.GetGitCommit)
	auth.Post("/repos/{owner}/{repo}/git/commits", gitHandler.CreateGitCommit)
	r.Get("/repos/{owner}/{repo}/git/ref/{ref...}", gitHandler.GetRef)
	r.Get("/repos/{owner}/{repo}/git/matching-refs/{ref...}", gitHandler.ListMatchingRefs)
	auth.Post("/repos/{owner}/{repo}/git/refs", gitHandler.CreateRef)
	auth.Patch("/repos/{owner}/{repo}/git/refs/{ref...}", gitHandler.UpdateRef)
	auth.Delete("/repos/{owner}/{repo}/git/refs/{ref...}", gitHandler.DeleteRef)
	r.Get("/repos/{owner}/{repo}/git/trees/{tree_sha}", gitHandler.GetTree)
	auth.Post("/repos/{owner}/{repo}/git/trees", gitHandler.CreateTree)
	r.Get("/repos/{owner}/{repo}/git/tags/{tag_sha}", gitHandler.GetTag)
	auth.Post("/repos/{owner}/{repo}/git/tags", gitHandler.CreateTag)
	r.Get("/repos/{owner}/{repo}/git/tags", gitHandler.ListTags)

	// ==========================================================================
	// Search
	// ==========================================================================
	r.Get("/search/repositories", searchHandler.SearchRepositories)
	r.Get("/search/code", searchHandler.SearchCode)
	r.Get("/search/commits", searchHandler.SearchCommits)
	r.Get("/search/issues", searchHandler.SearchIssues)
	r.Get("/search/users", searchHandler.SearchUsers)
	r.Get("/search/labels", searchHandler.SearchLabels)
	r.Get("/search/topics", searchHandler.SearchTopics)

	// ==========================================================================
	// Activity (Events & Feeds)
	// ==========================================================================
	r.Get("/events", activityHandler.ListPublicEvents)
	r.Get("/repos/{owner}/{repo}/events", activityHandler.ListRepoEvents)
	r.Get("/networks/{owner}/{repo}/events", activityHandler.ListRepoNetworkEvents)
	r.Get("/orgs/{org}/events", activityHandler.ListOrgEvents)
	r.Get("/users/{username}/received_events", activityHandler.ListUserReceivedEvents)
	r.Get("/users/{username}/received_events/public", activityHandler.ListUserReceivedPublicEvents)
	r.Get("/users/{username}/events", activityHandler.ListUserEvents)
	r.Get("/users/{username}/events/public", activityHandler.ListUserPublicEvents)
	auth.Get("/users/{username}/events/orgs/{org}", activityHandler.ListUserOrgEvents)
	optAuth.Get("/feeds", activityHandler.ListFeeds)
}
