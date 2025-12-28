# GitHome API Specification - Mizu Migration

## Overview

This specification details the migration of GitHome's REST API from raw `net/http` handlers to the go-mizu/mizu framework, ensuring 100% compatibility with the GitHub REST API v3.

## Current State Analysis

### Existing Architecture
- **Server**: Uses `http.ServeMux` directly in `app/web/server.go`
- **Handlers**: Located in `app/web/handler/api/*.go`, using `func(http.ResponseWriter, *http.Request)` signature
- **Response helpers**: `WriteJSON`, `WriteError`, `WriteNotFound`, etc. in `response.go`
- **Auth middleware**: `RequireAuth`, `OptionalAuth` wrapping `http.Handler`

### Feature Services
The application has well-structured service interfaces:
- `users.API` - User management (authentication, profiles, following)
- `repos.API` - Repository operations
- `issues.API` - Issue tracking
- `pulls.API` - Pull requests and reviews
- `orgs.API` - Organizations and members
- `teams.API` - Team management
- `labels.API` - Label management
- `milestones.API` - Milestone tracking
- `comments.API` - Issue and commit comments
- `releases.API` - Release management
- `stars.API` - Repository starring
- `watches.API` - Repository subscriptions
- `webhooks.API` - Webhook configuration
- `notifications.API` - Notification management
- `reactions.API` - Emoji reactions
- `collaborators.API` - Repository collaborators
- `branches.API` - Branch management and protection
- `commits.API` - Commit history and statuses
- `git.API` - Low-level git operations
- `search.API` - Search across entities
- `activities.API` - Activity events and feeds

## Target Architecture

### Mizu Framework Integration

#### Handler Signature
```go
// Old
func (h *Handler) Method(w http.ResponseWriter, r *http.Request)

// New
func (h *Handler) Method(c *mizu.Ctx) error
```

#### Key Mizu Context Methods
| Old Pattern | New Mizu Pattern |
|-------------|------------------|
| `r.PathValue("name")` | `c.Param("name")` |
| `r.URL.Query().Get("key")` | `c.Query("key")` |
| `json.NewDecoder(r.Body).Decode(&v)` | `c.BindJSON(&v, maxSize)` |
| `WriteJSON(w, code, v)` | `c.JSON(code, v)` |
| `WriteNoContent(w)` | `c.NoContent()` |
| `w.WriteHeader(code)` | `c.Status(code)` |
| `r.Context()` | `c.Context()` |
| `r.Header.Get("Authorization")` | `c.Request().Header.Get("Authorization")` |

#### Middleware Pattern
```go
// Old
func (h *AuthHandler) RequireAuth(next http.Handler) http.Handler

// New
func RequireAuth(users users.API) mizu.Middleware {
    return func(next mizu.Handler) mizu.Handler {
        return func(c *mizu.Ctx) error {
            // auth logic
            return next(c)
        }
    }
}
```

#### Router Configuration
```go
// Old
s.mux.HandleFunc("GET /users/{username}", userHandler.GetUser)
s.mux.Handle("GET /user", requireAuth(http.HandlerFunc(userHandler.GetAuthenticatedUser)))

// New
app.Get("/users/{username}", userHandler.GetUser)
app.With(requireAuth).Get("/user", userHandler.GetAuthenticatedUser)
```

## API Endpoints

### Authentication (Custom - not GitHub standard)
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| POST | `/login` | No | `AuthHandler.Login` |
| POST | `/register` | No | `AuthHandler.Register` |

### Users
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/user` | Required | `UserHandler.GetAuthenticatedUser` |
| PATCH | `/user` | Required | `UserHandler.UpdateAuthenticatedUser` |
| GET | `/users` | No | `UserHandler.ListUsers` |
| GET | `/users/{username}` | No | `UserHandler.GetUser` |
| GET | `/users/{username}/followers` | No | `UserHandler.ListFollowers` |
| GET | `/users/{username}/following` | No | `UserHandler.ListFollowing` |
| GET | `/users/{username}/following/{target_user}` | No | `UserHandler.CheckFollowing` |
| GET | `/user/followers` | Required | `UserHandler.ListAuthenticatedUserFollowers` |
| GET | `/user/following` | Required | `UserHandler.ListAuthenticatedUserFollowing` |
| GET | `/user/following/{username}` | Required | `UserHandler.CheckAuthenticatedUserFollowing` |
| PUT | `/user/following/{username}` | Required | `UserHandler.FollowUser` |
| DELETE | `/user/following/{username}` | Required | `UserHandler.UnfollowUser` |

### Organizations
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/organizations` | No | `OrgHandler.ListOrgs` |
| GET | `/orgs/{org}` | No | `OrgHandler.GetOrg` |
| PATCH | `/orgs/{org}` | Required | `OrgHandler.UpdateOrg` |
| GET | `/user/orgs` | Required | `OrgHandler.ListAuthenticatedUserOrgs` |
| GET | `/users/{username}/orgs` | No | `OrgHandler.ListUserOrgs` |
| GET | `/orgs/{org}/members` | No | `OrgHandler.ListOrgMembers` |
| GET | `/orgs/{org}/members/{username}` | No | `OrgHandler.CheckOrgMember` |
| DELETE | `/orgs/{org}/members/{username}` | Required | `OrgHandler.RemoveOrgMember` |
| GET | `/orgs/{org}/memberships/{username}` | Required | `OrgHandler.GetOrgMembership` |
| PUT | `/orgs/{org}/memberships/{username}` | Required | `OrgHandler.SetOrgMembership` |
| DELETE | `/orgs/{org}/memberships/{username}` | Required | `OrgHandler.RemoveOrgMembership` |
| GET | `/orgs/{org}/outside_collaborators` | Required | `OrgHandler.ListOutsideCollaborators` |
| GET | `/user/memberships/orgs/{org}` | Required | `OrgHandler.GetAuthenticatedUserOrgMembership` |
| PATCH | `/user/memberships/orgs/{org}` | Required | `OrgHandler.UpdateAuthenticatedUserOrgMembership` |

### Repositories
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/repositories` | No | `RepoHandler.ListPublicRepos` |
| GET | `/user/repos` | Required | `RepoHandler.ListAuthenticatedUserRepos` |
| POST | `/user/repos` | Required | `RepoHandler.CreateAuthenticatedUserRepo` |
| GET | `/users/{username}/repos` | No | `RepoHandler.ListUserRepos` |
| GET | `/orgs/{org}/repos` | No | `RepoHandler.ListOrgRepos` |
| POST | `/orgs/{org}/repos` | Required | `RepoHandler.CreateOrgRepo` |
| GET | `/repos/{owner}/{repo}` | Optional | `RepoHandler.GetRepo` |
| PATCH | `/repos/{owner}/{repo}` | Required | `RepoHandler.UpdateRepo` |
| DELETE | `/repos/{owner}/{repo}` | Required | `RepoHandler.DeleteRepo` |
| GET | `/repos/{owner}/{repo}/topics` | No | `RepoHandler.ListRepoTopics` |
| PUT | `/repos/{owner}/{repo}/topics` | Required | `RepoHandler.ReplaceRepoTopics` |
| GET | `/repos/{owner}/{repo}/languages` | No | `RepoHandler.ListRepoLanguages` |
| GET | `/repos/{owner}/{repo}/contributors` | No | `RepoHandler.ListRepoContributors` |
| GET | `/repos/{owner}/{repo}/tags` | No | `RepoHandler.ListRepoTags` |
| POST | `/repos/{owner}/{repo}/transfer` | Required | `RepoHandler.TransferRepo` |
| GET | `/repos/{owner}/{repo}/readme` | No | `RepoHandler.GetRepoReadme` |
| GET | `/repos/{owner}/{repo}/contents/{path...}` | No | `RepoHandler.GetRepoContent` |
| PUT | `/repos/{owner}/{repo}/contents/{path...}` | Required | `RepoHandler.CreateOrUpdateFileContent` |
| DELETE | `/repos/{owner}/{repo}/contents/{path...}` | Required | `RepoHandler.DeleteFileContent` |
| POST | `/repos/{owner}/{repo}/forks` | Required | `RepoHandler.ForkRepo` |
| GET | `/repos/{owner}/{repo}/forks` | No | `RepoHandler.ListForks` |

### Issues
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/issues` | Required | `IssueHandler.ListIssues` |
| GET | `/user/issues` | Required | `IssueHandler.ListAuthenticatedUserIssues` |
| GET | `/orgs/{org}/issues` | Required | `IssueHandler.ListOrgIssues` |
| GET | `/repos/{owner}/{repo}/issues` | No | `IssueHandler.ListRepoIssues` |
| GET | `/repos/{owner}/{repo}/issues/{issue_number}` | No | `IssueHandler.GetIssue` |
| POST | `/repos/{owner}/{repo}/issues` | Required | `IssueHandler.CreateIssue` |
| PATCH | `/repos/{owner}/{repo}/issues/{issue_number}` | Required | `IssueHandler.UpdateIssue` |
| PUT | `/repos/{owner}/{repo}/issues/{issue_number}/lock` | Required | `IssueHandler.LockIssue` |
| DELETE | `/repos/{owner}/{repo}/issues/{issue_number}/lock` | Required | `IssueHandler.UnlockIssue` |
| GET | `/repos/{owner}/{repo}/assignees` | No | `IssueHandler.ListIssueAssignees` |
| GET | `/repos/{owner}/{repo}/assignees/{assignee}` | No | `IssueHandler.CheckAssignee` |
| POST | `/repos/{owner}/{repo}/issues/{issue_number}/assignees` | Required | `IssueHandler.AddAssignees` |
| DELETE | `/repos/{owner}/{repo}/issues/{issue_number}/assignees` | Required | `IssueHandler.RemoveAssignees` |
| GET | `/repos/{owner}/{repo}/issues/{issue_number}/events` | No | `IssueHandler.ListIssueEvents` |
| GET | `/repos/{owner}/{repo}/issues/events/{event_id}` | No | `IssueHandler.GetIssueEvent` |
| GET | `/repos/{owner}/{repo}/issues/events` | No | `IssueHandler.ListRepoIssueEvents` |
| GET | `/repos/{owner}/{repo}/issues/{issue_number}/timeline` | No | `IssueHandler.ListIssueTimeline` |

### Pull Requests
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/repos/{owner}/{repo}/pulls` | No | `PullHandler.ListPulls` |
| GET | `/repos/{owner}/{repo}/pulls/{pull_number}` | No | `PullHandler.GetPull` |
| POST | `/repos/{owner}/{repo}/pulls` | Required | `PullHandler.CreatePull` |
| PATCH | `/repos/{owner}/{repo}/pulls/{pull_number}` | Required | `PullHandler.UpdatePull` |
| GET | `/repos/{owner}/{repo}/pulls/{pull_number}/commits` | No | `PullHandler.ListPullCommits` |
| GET | `/repos/{owner}/{repo}/pulls/{pull_number}/files` | No | `PullHandler.ListPullFiles` |
| GET | `/repos/{owner}/{repo}/pulls/{pull_number}/merge` | No | `PullHandler.CheckPullMerged` |
| PUT | `/repos/{owner}/{repo}/pulls/{pull_number}/merge` | Required | `PullHandler.MergePull` |
| PUT | `/repos/{owner}/{repo}/pulls/{pull_number}/update-branch` | Required | `PullHandler.UpdatePullBranch` |

### Pull Request Reviews
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/repos/{owner}/{repo}/pulls/{pull_number}/reviews` | No | `PullHandler.ListPullReviews` |
| GET | `/repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}` | No | `PullHandler.GetPullReview` |
| POST | `/repos/{owner}/{repo}/pulls/{pull_number}/reviews` | Required | `PullHandler.CreatePullReview` |
| PUT | `/repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}` | Required | `PullHandler.UpdatePullReview` |
| DELETE | `/repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}` | Required | `PullHandler.DeletePullReview` |
| POST | `/repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/events` | Required | `PullHandler.SubmitPullReview` |
| PUT | `/repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/dismissals` | Required | `PullHandler.DismissPullReview` |
| GET | `/repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/comments` | No | `PullHandler.ListReviewComments` |

### Pull Request Comments
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/repos/{owner}/{repo}/pulls/{pull_number}/comments` | No | `PullHandler.ListPullReviewComments` |
| POST | `/repos/{owner}/{repo}/pulls/{pull_number}/comments` | Required | `PullHandler.CreatePullReviewComment` |
| GET | `/repos/{owner}/{repo}/pulls/comments/{comment_id}` | No | `PullHandler.GetPullReviewComment` |
| PATCH | `/repos/{owner}/{repo}/pulls/comments/{comment_id}` | Required | `PullHandler.UpdatePullReviewComment` |
| DELETE | `/repos/{owner}/{repo}/pulls/comments/{comment_id}` | Required | `PullHandler.DeletePullReviewComment` |

### Requested Reviewers
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers` | No | `PullHandler.ListRequestedReviewers` |
| POST | `/repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers` | Required | `PullHandler.RequestReviewers` |
| DELETE | `/repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers` | Required | `PullHandler.RemoveRequestedReviewers` |

### Labels
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/repos/{owner}/{repo}/labels` | No | `LabelHandler.ListRepoLabels` |
| GET | `/repos/{owner}/{repo}/labels/{name}` | No | `LabelHandler.GetLabel` |
| POST | `/repos/{owner}/{repo}/labels` | Required | `LabelHandler.CreateLabel` |
| PATCH | `/repos/{owner}/{repo}/labels/{name}` | Required | `LabelHandler.UpdateLabel` |
| DELETE | `/repos/{owner}/{repo}/labels/{name}` | Required | `LabelHandler.DeleteLabel` |
| GET | `/repos/{owner}/{repo}/issues/{issue_number}/labels` | No | `LabelHandler.ListIssueLabels` |
| POST | `/repos/{owner}/{repo}/issues/{issue_number}/labels` | Required | `LabelHandler.AddIssueLabels` |
| PUT | `/repos/{owner}/{repo}/issues/{issue_number}/labels` | Required | `LabelHandler.SetIssueLabels` |
| DELETE | `/repos/{owner}/{repo}/issues/{issue_number}/labels` | Required | `LabelHandler.RemoveAllIssueLabels` |
| DELETE | `/repos/{owner}/{repo}/issues/{issue_number}/labels/{name}` | Required | `LabelHandler.RemoveIssueLabel` |
| GET | `/repos/{owner}/{repo}/milestones/{milestone_number}/labels` | No | `LabelHandler.ListLabelsForMilestone` |

### Milestones
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/repos/{owner}/{repo}/milestones` | No | `MilestoneHandler.ListMilestones` |
| GET | `/repos/{owner}/{repo}/milestones/{milestone_number}` | No | `MilestoneHandler.GetMilestone` |
| POST | `/repos/{owner}/{repo}/milestones` | Required | `MilestoneHandler.CreateMilestone` |
| PATCH | `/repos/{owner}/{repo}/milestones/{milestone_number}` | Required | `MilestoneHandler.UpdateMilestone` |
| DELETE | `/repos/{owner}/{repo}/milestones/{milestone_number}` | Required | `MilestoneHandler.DeleteMilestone` |

### Comments
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/repos/{owner}/{repo}/issues/{issue_number}/comments` | No | `CommentHandler.ListIssueComments` |
| GET | `/repos/{owner}/{repo}/issues/comments/{comment_id}` | No | `CommentHandler.GetIssueComment` |
| POST | `/repos/{owner}/{repo}/issues/{issue_number}/comments` | Required | `CommentHandler.CreateIssueComment` |
| PATCH | `/repos/{owner}/{repo}/issues/comments/{comment_id}` | Required | `CommentHandler.UpdateIssueComment` |
| DELETE | `/repos/{owner}/{repo}/issues/comments/{comment_id}` | Required | `CommentHandler.DeleteIssueComment` |
| GET | `/repos/{owner}/{repo}/issues/comments` | No | `CommentHandler.ListRepoComments` |
| GET | `/repos/{owner}/{repo}/commits/{commit_sha}/comments` | No | `CommentHandler.ListCommitComments` |
| POST | `/repos/{owner}/{repo}/commits/{commit_sha}/comments` | Required | `CommentHandler.CreateCommitComment` |
| GET | `/repos/{owner}/{repo}/comments/{comment_id}` | No | `CommentHandler.GetCommitComment` |
| PATCH | `/repos/{owner}/{repo}/comments/{comment_id}` | Required | `CommentHandler.UpdateCommitComment` |
| DELETE | `/repos/{owner}/{repo}/comments/{comment_id}` | Required | `CommentHandler.DeleteCommitComment` |
| GET | `/repos/{owner}/{repo}/comments` | No | `CommentHandler.ListRepoCommitComments` |

### Teams
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/orgs/{org}/teams` | No | `TeamHandler.ListOrgTeams` |
| GET | `/orgs/{org}/teams/{team_slug}` | No | `TeamHandler.GetOrgTeam` |
| POST | `/orgs/{org}/teams` | Required | `TeamHandler.CreateTeam` |
| PATCH | `/orgs/{org}/teams/{team_slug}` | Required | `TeamHandler.UpdateTeam` |
| DELETE | `/orgs/{org}/teams/{team_slug}` | Required | `TeamHandler.DeleteTeam` |
| GET | `/orgs/{org}/teams/{team_slug}/members` | No | `TeamHandler.ListTeamMembers` |
| GET | `/orgs/{org}/teams/{team_slug}/memberships/{username}` | Required | `TeamHandler.GetTeamMembership` |
| PUT | `/orgs/{org}/teams/{team_slug}/memberships/{username}` | Required | `TeamHandler.AddTeamMember` |
| DELETE | `/orgs/{org}/teams/{team_slug}/memberships/{username}` | Required | `TeamHandler.RemoveTeamMember` |
| GET | `/orgs/{org}/teams/{team_slug}/repos` | No | `TeamHandler.ListTeamRepos` |
| GET | `/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}` | No | `TeamHandler.CheckTeamRepoPermission` |
| PUT | `/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}` | Required | `TeamHandler.AddTeamRepo` |
| DELETE | `/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}` | Required | `TeamHandler.RemoveTeamRepo` |
| GET | `/orgs/{org}/teams/{team_slug}/teams` | No | `TeamHandler.ListChildTeams` |
| GET | `/user/teams` | Required | `TeamHandler.ListAuthenticatedUserTeams` |

### Releases
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/repos/{owner}/{repo}/releases` | No | `ReleaseHandler.ListReleases` |
| GET | `/repos/{owner}/{repo}/releases/{release_id}` | No | `ReleaseHandler.GetRelease` |
| GET | `/repos/{owner}/{repo}/releases/latest` | No | `ReleaseHandler.GetLatestRelease` |
| GET | `/repos/{owner}/{repo}/releases/tags/{tag}` | No | `ReleaseHandler.GetReleaseByTag` |
| POST | `/repos/{owner}/{repo}/releases` | Required | `ReleaseHandler.CreateRelease` |
| PATCH | `/repos/{owner}/{repo}/releases/{release_id}` | Required | `ReleaseHandler.UpdateRelease` |
| DELETE | `/repos/{owner}/{repo}/releases/{release_id}` | Required | `ReleaseHandler.DeleteRelease` |
| POST | `/repos/{owner}/{repo}/releases/generate-notes` | Required | `ReleaseHandler.GenerateReleaseNotes` |
| GET | `/repos/{owner}/{repo}/releases/{release_id}/assets` | No | `ReleaseHandler.ListReleaseAssets` |
| GET | `/repos/{owner}/{repo}/releases/assets/{asset_id}` | No | `ReleaseHandler.GetReleaseAsset` |
| PATCH | `/repos/{owner}/{repo}/releases/assets/{asset_id}` | Required | `ReleaseHandler.UpdateReleaseAsset` |
| DELETE | `/repos/{owner}/{repo}/releases/assets/{asset_id}` | Required | `ReleaseHandler.DeleteReleaseAsset` |
| POST | `/repos/{owner}/{repo}/releases/{release_id}/assets` | Required | `ReleaseHandler.UploadReleaseAsset` |

### Stars
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/repos/{owner}/{repo}/stargazers` | No | `StarHandler.ListStargazers` |
| GET | `/users/{username}/starred` | No | `StarHandler.ListStarredRepos` |
| GET | `/user/starred` | Required | `StarHandler.ListAuthenticatedUserStarredRepos` |
| GET | `/user/starred/{owner}/{repo}` | Required | `StarHandler.CheckRepoStarred` |
| PUT | `/user/starred/{owner}/{repo}` | Required | `StarHandler.StarRepo` |
| DELETE | `/user/starred/{owner}/{repo}` | Required | `StarHandler.UnstarRepo` |

### Watches (Subscriptions)
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/repos/{owner}/{repo}/subscribers` | No | `WatchHandler.ListWatchers` |
| GET | `/repos/{owner}/{repo}/subscription` | Required | `WatchHandler.GetSubscription` |
| PUT | `/repos/{owner}/{repo}/subscription` | Required | `WatchHandler.SetSubscription` |
| DELETE | `/repos/{owner}/{repo}/subscription` | Required | `WatchHandler.DeleteSubscription` |
| GET | `/users/{username}/subscriptions` | No | `WatchHandler.ListWatchedRepos` |
| GET | `/user/subscriptions` | Required | `WatchHandler.ListAuthenticatedUserWatchedRepos` |

### Webhooks
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/repos/{owner}/{repo}/hooks` | Required | `WebhookHandler.ListRepoWebhooks` |
| GET | `/repos/{owner}/{repo}/hooks/{hook_id}` | Required | `WebhookHandler.GetRepoWebhook` |
| POST | `/repos/{owner}/{repo}/hooks` | Required | `WebhookHandler.CreateRepoWebhook` |
| PATCH | `/repos/{owner}/{repo}/hooks/{hook_id}` | Required | `WebhookHandler.UpdateRepoWebhook` |
| DELETE | `/repos/{owner}/{repo}/hooks/{hook_id}` | Required | `WebhookHandler.DeleteRepoWebhook` |
| POST | `/repos/{owner}/{repo}/hooks/{hook_id}/pings` | Required | `WebhookHandler.PingRepoWebhook` |
| POST | `/repos/{owner}/{repo}/hooks/{hook_id}/tests` | Required | `WebhookHandler.TestRepoWebhook` |
| GET | `/repos/{owner}/{repo}/hooks/{hook_id}/deliveries` | Required | `WebhookHandler.ListWebhookDeliveries` |
| GET | `/repos/{owner}/{repo}/hooks/{hook_id}/deliveries/{delivery_id}` | Required | `WebhookHandler.GetWebhookDelivery` |
| POST | `/repos/{owner}/{repo}/hooks/{hook_id}/deliveries/{delivery_id}/attempts` | Required | `WebhookHandler.RedeliverWebhook` |
| GET | `/orgs/{org}/hooks` | Required | `WebhookHandler.ListOrgWebhooks` |
| GET | `/orgs/{org}/hooks/{hook_id}` | Required | `WebhookHandler.GetOrgWebhook` |
| POST | `/orgs/{org}/hooks` | Required | `WebhookHandler.CreateOrgWebhook` |
| PATCH | `/orgs/{org}/hooks/{hook_id}` | Required | `WebhookHandler.UpdateOrgWebhook` |
| DELETE | `/orgs/{org}/hooks/{hook_id}` | Required | `WebhookHandler.DeleteOrgWebhook` |
| POST | `/orgs/{org}/hooks/{hook_id}/pings` | Required | `WebhookHandler.PingOrgWebhook` |

### Notifications
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/notifications` | Required | `NotificationHandler.ListNotifications` |
| PUT | `/notifications` | Required | `NotificationHandler.MarkAllAsRead` |
| GET | `/notifications/threads/{thread_id}` | Required | `NotificationHandler.GetThread` |
| PATCH | `/notifications/threads/{thread_id}` | Required | `NotificationHandler.MarkThreadAsRead` |
| DELETE | `/notifications/threads/{thread_id}` | Required | `NotificationHandler.MarkThreadAsDone` |
| GET | `/notifications/threads/{thread_id}/subscription` | Required | `NotificationHandler.GetThreadSubscription` |
| PUT | `/notifications/threads/{thread_id}/subscription` | Required | `NotificationHandler.SetThreadSubscription` |
| DELETE | `/notifications/threads/{thread_id}/subscription` | Required | `NotificationHandler.DeleteThreadSubscription` |
| GET | `/repos/{owner}/{repo}/notifications` | Required | `NotificationHandler.ListRepoNotifications` |
| PUT | `/repos/{owner}/{repo}/notifications` | Required | `NotificationHandler.MarkRepoNotificationsAsRead` |

### Reactions
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/repos/{owner}/{repo}/issues/{issue_number}/reactions` | No | `ReactionHandler.ListIssueReactions` |
| POST | `/repos/{owner}/{repo}/issues/{issue_number}/reactions` | Required | `ReactionHandler.CreateIssueReaction` |
| DELETE | `/repos/{owner}/{repo}/issues/{issue_number}/reactions/{reaction_id}` | Required | `ReactionHandler.DeleteIssueReaction` |
| GET | `/repos/{owner}/{repo}/issues/comments/{comment_id}/reactions` | No | `ReactionHandler.ListIssueCommentReactions` |
| POST | `/repos/{owner}/{repo}/issues/comments/{comment_id}/reactions` | Required | `ReactionHandler.CreateIssueCommentReaction` |
| DELETE | `/repos/{owner}/{repo}/issues/comments/{comment_id}/reactions/{reaction_id}` | Required | `ReactionHandler.DeleteIssueCommentReaction` |
| GET | `/repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions` | No | `ReactionHandler.ListPullReviewCommentReactions` |
| POST | `/repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions` | Required | `ReactionHandler.CreatePullReviewCommentReaction` |
| DELETE | `/repos/{owner}/{repo}/pulls/comments/{comment_id}/reactions/{reaction_id}` | Required | `ReactionHandler.DeletePullReviewCommentReaction` |
| GET | `/repos/{owner}/{repo}/comments/{comment_id}/reactions` | No | `ReactionHandler.ListCommitCommentReactions` |
| POST | `/repos/{owner}/{repo}/comments/{comment_id}/reactions` | Required | `ReactionHandler.CreateCommitCommentReaction` |
| DELETE | `/repos/{owner}/{repo}/comments/{comment_id}/reactions/{reaction_id}` | Required | `ReactionHandler.DeleteCommitCommentReaction` |
| GET | `/repos/{owner}/{repo}/releases/{release_id}/reactions` | No | `ReactionHandler.ListReleaseReactions` |
| POST | `/repos/{owner}/{repo}/releases/{release_id}/reactions` | Required | `ReactionHandler.CreateReleaseReaction` |
| DELETE | `/repos/{owner}/{repo}/releases/{release_id}/reactions/{reaction_id}` | Required | `ReactionHandler.DeleteReleaseReaction` |

### Collaborators
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/repos/{owner}/{repo}/collaborators` | Required | `CollaboratorHandler.ListCollaborators` |
| GET | `/repos/{owner}/{repo}/collaborators/{username}` | No | `CollaboratorHandler.CheckCollaborator` |
| PUT | `/repos/{owner}/{repo}/collaborators/{username}` | Required | `CollaboratorHandler.AddCollaborator` |
| DELETE | `/repos/{owner}/{repo}/collaborators/{username}` | Required | `CollaboratorHandler.RemoveCollaborator` |
| GET | `/repos/{owner}/{repo}/collaborators/{username}/permission` | Required | `CollaboratorHandler.GetCollaboratorPermission` |
| GET | `/repos/{owner}/{repo}/invitations` | Required | `CollaboratorHandler.ListInvitations` |
| PATCH | `/repos/{owner}/{repo}/invitations/{invitation_id}` | Required | `CollaboratorHandler.UpdateInvitation` |
| DELETE | `/repos/{owner}/{repo}/invitations/{invitation_id}` | Required | `CollaboratorHandler.DeleteInvitation` |
| GET | `/user/repository_invitations` | Required | `CollaboratorHandler.ListUserInvitations` |
| PATCH | `/user/repository_invitations/{invitation_id}` | Required | `CollaboratorHandler.AcceptInvitation` |
| DELETE | `/user/repository_invitations/{invitation_id}` | Required | `CollaboratorHandler.DeclineInvitation` |

### Branches
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/repos/{owner}/{repo}/branches` | No | `BranchHandler.ListBranches` |
| GET | `/repos/{owner}/{repo}/branches/{branch}` | No | `BranchHandler.GetBranch` |
| POST | `/repos/{owner}/{repo}/branches/{branch}/rename` | Required | `BranchHandler.RenameBranch` |
| POST | `/repos/{owner}/{repo}/merge-upstream` | Required | `BranchHandler.SyncFork` |
| GET | `/repos/{owner}/{repo}/branches/{branch}/protection` | Required | `BranchHandler.GetBranchProtection` |
| PUT | `/repos/{owner}/{repo}/branches/{branch}/protection` | Required | `BranchHandler.UpdateBranchProtection` |
| DELETE | `/repos/{owner}/{repo}/branches/{branch}/protection` | Required | `BranchHandler.DeleteBranchProtection` |
| GET | `/repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks` | Required | `BranchHandler.GetRequiredStatusChecks` |
| PATCH | `/repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks` | Required | `BranchHandler.UpdateRequiredStatusChecks` |
| DELETE | `/repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks` | Required | `BranchHandler.DeleteRequiredStatusChecks` |
| GET | `/repos/{owner}/{repo}/branches/{branch}/protection/required_pull_request_reviews` | Required | `BranchHandler.GetRequiredPullRequestReviews` |
| PATCH | `/repos/{owner}/{repo}/branches/{branch}/protection/required_pull_request_reviews` | Required | `BranchHandler.UpdateRequiredPullRequestReviews` |
| DELETE | `/repos/{owner}/{repo}/branches/{branch}/protection/required_pull_request_reviews` | Required | `BranchHandler.DeleteRequiredPullRequestReviews` |
| GET | `/repos/{owner}/{repo}/branches/{branch}/protection/enforce_admins` | Required | `BranchHandler.GetAdminEnforcement` |
| POST | `/repos/{owner}/{repo}/branches/{branch}/protection/enforce_admins` | Required | `BranchHandler.SetAdminEnforcement` |
| DELETE | `/repos/{owner}/{repo}/branches/{branch}/protection/enforce_admins` | Required | `BranchHandler.DeleteAdminEnforcement` |

### Commits
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/repos/{owner}/{repo}/commits` | No | `CommitHandler.ListCommits` |
| GET | `/repos/{owner}/{repo}/commits/{ref}` | No | `CommitHandler.GetCommit` |
| GET | `/repos/{owner}/{repo}/compare/{basehead}` | No | `CommitHandler.CompareCommits` |
| GET | `/repos/{owner}/{repo}/commits/{commit_sha}/branches-where-head` | No | `CommitHandler.ListBranchesForHead` |
| GET | `/repos/{owner}/{repo}/commits/{commit_sha}/pulls` | No | `CommitHandler.ListPullsForCommit` |
| GET | `/repos/{owner}/{repo}/commits/{ref}/status` | No | `CommitHandler.GetCombinedStatus` |
| GET | `/repos/{owner}/{repo}/commits/{ref}/statuses` | No | `CommitHandler.ListStatuses` |
| POST | `/repos/{owner}/{repo}/statuses/{sha}` | Required | `CommitHandler.CreateStatus` |

### Git Data (Low-level)
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/repos/{owner}/{repo}/git/blobs/{file_sha}` | No | `GitHandler.GetBlob` |
| POST | `/repos/{owner}/{repo}/git/blobs` | Required | `GitHandler.CreateBlob` |
| GET | `/repos/{owner}/{repo}/git/commits/{commit_sha}` | No | `GitHandler.GetGitCommit` |
| POST | `/repos/{owner}/{repo}/git/commits` | Required | `GitHandler.CreateGitCommit` |
| GET | `/repos/{owner}/{repo}/git/ref/{ref...}` | No | `GitHandler.GetRef` |
| GET | `/repos/{owner}/{repo}/git/matching-refs/{ref...}` | No | `GitHandler.ListMatchingRefs` |
| POST | `/repos/{owner}/{repo}/git/refs` | Required | `GitHandler.CreateRef` |
| PATCH | `/repos/{owner}/{repo}/git/refs/{ref...}` | Required | `GitHandler.UpdateRef` |
| DELETE | `/repos/{owner}/{repo}/git/refs/{ref...}` | Required | `GitHandler.DeleteRef` |
| GET | `/repos/{owner}/{repo}/git/trees/{tree_sha}` | No | `GitHandler.GetTree` |
| POST | `/repos/{owner}/{repo}/git/trees` | Required | `GitHandler.CreateTree` |
| GET | `/repos/{owner}/{repo}/git/tags/{tag_sha}` | No | `GitHandler.GetTag` |
| POST | `/repos/{owner}/{repo}/git/tags` | Required | `GitHandler.CreateTag` |

### Search
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/search/repositories` | No | `SearchHandler.SearchRepositories` |
| GET | `/search/code` | No | `SearchHandler.SearchCode` |
| GET | `/search/commits` | No | `SearchHandler.SearchCommits` |
| GET | `/search/issues` | No | `SearchHandler.SearchIssues` |
| GET | `/search/users` | No | `SearchHandler.SearchUsers` |
| GET | `/search/labels` | No | `SearchHandler.SearchLabels` |
| GET | `/search/topics` | No | `SearchHandler.SearchTopics` |

### Activity (Events & Feeds)
| Method | Path | Auth | Handler |
|--------|------|------|---------|
| GET | `/events` | No | `ActivityHandler.ListPublicEvents` |
| GET | `/repos/{owner}/{repo}/events` | No | `ActivityHandler.ListRepoEvents` |
| GET | `/networks/{owner}/{repo}/events` | No | `ActivityHandler.ListRepoNetworkEvents` |
| GET | `/orgs/{org}/events` | No | `ActivityHandler.ListOrgEvents` |
| GET | `/users/{username}/received_events` | No | `ActivityHandler.ListUserReceivedEvents` |
| GET | `/users/{username}/received_events/public` | No | `ActivityHandler.ListUserReceivedPublicEvents` |
| GET | `/users/{username}/events` | No | `ActivityHandler.ListUserEvents` |
| GET | `/users/{username}/events/public` | No | `ActivityHandler.ListUserPublicEvents` |
| GET | `/users/{username}/events/orgs/{org}` | Required | `ActivityHandler.ListUserOrgEvents` |
| GET | `/feeds` | Optional | `ActivityHandler.ListFeeds` |

## Implementation Details

### Response Helper Functions

Replace the current response helpers with Mizu context methods:

```go
// response.go helpers using mizu.Ctx

// JSON sends a JSON response
func JSON(c *mizu.Ctx, code int, v any) error {
    return c.JSON(code, v)
}

// Error sends an error response
func Error(c *mizu.Ctx, code int, message string) error {
    return c.JSON(code, &APIError{Message: message})
}

// NotFound sends a 404 response
func NotFound(c *mizu.Ctx, resource string) error {
    return c.JSON(http.StatusNotFound, &APIError{
        Message: fmt.Sprintf("%s not found", resource),
    })
}

// Unauthorized sends a 401 response
func Unauthorized(c *mizu.Ctx) error {
    return c.JSON(http.StatusUnauthorized, &APIError{
        Message: "Requires authentication",
    })
}

// NoContent sends a 204 response
func NoContent(c *mizu.Ctx) error {
    return c.NoContent()
}

// Created sends a 201 response with JSON body
func Created(c *mizu.Ctx, v any) error {
    return c.JSON(http.StatusCreated, v)
}
```

### Authentication Context

```go
// Context key for authenticated user
type ctxKey string
const UserContextKey ctxKey = "user"

// GetUser retrieves user from Mizu context
func GetUser(c *mizu.Ctx) *users.User {
    u, _ := c.Context().Value(UserContextKey).(*users.User)
    return u
}

// GetUserID retrieves user ID from context
func GetUserID(c *mizu.Ctx) int64 {
    if u := GetUser(c); u != nil {
        return u.ID
    }
    return 0
}
```

### Path Parameter Helpers

```go
// ParamInt extracts an integer path parameter
func ParamInt(c *mizu.Ctx, name string) (int, error) {
    return strconv.Atoi(c.Param(name))
}

// ParamInt64 extracts an int64 path parameter
func ParamInt64(c *mizu.Ctx, name string) (int64, error) {
    return strconv.ParseInt(c.Param(name), 10, 64)
}
```

### Pagination

```go
// PaginationParams holds pagination parameters
type PaginationParams struct {
    Page    int
    PerPage int
}

// GetPagination extracts pagination from query params
func GetPagination(c *mizu.Ctx) PaginationParams {
    p := PaginationParams{Page: 1, PerPage: 30}

    if page := c.Query("page"); page != "" {
        if n, err := strconv.Atoi(page); err == nil && n > 0 {
            p.Page = n
        }
    }

    if perPage := c.Query("per_page"); perPage != "" {
        if n, err := strconv.Atoi(perPage); err == nil && n > 0 && n <= 100 {
            p.PerPage = n
        }
    }

    return p
}
```

## File Structure

```
app/web/
├── server.go           # Mizu App setup and route registration
└── handler/
    └── api/
        ├── ctx.go      # Context helpers (GetUser, GetPagination, etc.)
        ├── errors.go   # Error types and response helpers
        ├── middleware.go # Auth middleware
        ├── user.go
        ├── org.go
        ├── repo.go
        ├── issue.go
        ├── pull.go
        ├── label.go
        ├── milestone.go
        ├── comment.go
        ├── team.go
        ├── release.go
        ├── star.go
        ├── watch.go
        ├── webhook.go
        ├── notification.go
        ├── reaction.go
        ├── collaborator.go
        ├── branch.go
        ├── commit.go
        ├── git.go
        ├── search.go
        └── activity.go
```

## Error Handling

Mizu handlers return errors which are handled by the central error handler:

```go
app.ErrorHandler(func(c *mizu.Ctx, err error) {
    // Log the error
    c.Logger().Error("handler error", slog.Any("error", err))

    // Map known errors to HTTP status codes
    switch {
    case errors.Is(err, users.ErrNotFound):
        _ = c.JSON(http.StatusNotFound, &APIError{Message: "Not found"})
    case errors.Is(err, users.ErrUnauthorized):
        _ = c.JSON(http.StatusUnauthorized, &APIError{Message: "Requires authentication"})
    default:
        _ = c.JSON(http.StatusInternalServerError, &APIError{Message: "Internal server error"})
    }
})
```

## Total Endpoint Count

| Category | Endpoints |
|----------|-----------|
| Authentication | 2 |
| Users | 12 |
| Organizations | 14 |
| Repositories | 20 |
| Issues | 17 |
| Pull Requests | 9 |
| PR Reviews | 8 |
| PR Comments | 5 |
| Requested Reviewers | 3 |
| Labels | 11 |
| Milestones | 5 |
| Comments | 11 |
| Teams | 15 |
| Releases | 14 |
| Stars | 6 |
| Watches | 6 |
| Webhooks | 16 |
| Notifications | 10 |
| Reactions | 15 |
| Collaborators | 11 |
| Branches | 15 |
| Commits | 8 |
| Git Data | 13 |
| Search | 7 |
| Activity | 10 |
| **Total** | **~258** |

## Implementation Priority

1. **Core Infrastructure**: Context helpers, error handling, middleware
2. **Authentication**: Login, register, auth middleware
3. **Users**: Profile endpoints (most commonly used)
4. **Repositories**: CRUD and content operations
5. **Issues & Pull Requests**: Core collaboration features
6. **Remaining endpoints**: Labels, milestones, teams, etc.
