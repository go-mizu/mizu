package github

import (
	"time"

	"github.com/go-mizu/blueprints/githome/feature/comments"
	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/labels"
	"github.com/go-mizu/blueprints/githome/feature/milestones"
	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/pulls"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// mapRepository maps a GitHub repository to a GitHome repository.
func mapRepository(gh *ghRepository, ownerID int64, ownerType string, isPublic bool) *repos.Repository {
	visibility := "public"
	if gh.Private || !isPublic {
		visibility = "private"
	}

	return &repos.Repository{
		Name:            gh.Name,
		FullName:        gh.FullName,
		OwnerID:         ownerID,
		OwnerType:       ownerType,
		Private:         gh.Private || !isPublic,
		Visibility:      visibility,
		Description:     gh.Description,
		Fork:            gh.Fork,
		DefaultBranch:   gh.DefaultBranch,
		HasIssues:       gh.HasIssues,
		HasProjects:     gh.HasProjects,
		HasWiki:         gh.HasWiki,
		HasDownloads:    gh.HasDownloads,
		ForksCount:      gh.ForksCount,
		StargazersCount: gh.StargazersCount,
		WatchersCount:   gh.WatchersCount,
		Size:            gh.Size,
		OpenIssuesCount: gh.OpenIssuesCount,
		AllowSquashMerge:  true,
		AllowMergeCommit:  true,
		AllowRebaseMerge:  true,
		AllowForking:      true,
		CreatedAt:       gh.CreatedAt,
		UpdatedAt:       gh.UpdatedAt,
		PushedAt:        &gh.PushedAt,
	}
}

// mapUser maps a GitHub user to a GitHome user.
func mapUser(gh *ghUser) *users.User {
	if gh == nil {
		return nil
	}
	return &users.User{
		Login:     gh.Login,
		Name:      gh.Name,
		Email:     gh.Email,
		AvatarURL: gh.AvatarURL,
		HTMLURL:   gh.HTMLURL,
		Type:      gh.Type,
		SiteAdmin: gh.SiteAdmin,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// mapOrganization maps a GitHub user (organization) to a GitHome organization.
func mapOrganization(gh *ghUser) *orgs.Organization {
	if gh == nil {
		return nil
	}
	return &orgs.Organization{
		Login:                       gh.Login,
		Name:                        gh.Name,
		Email:                       gh.Email,
		AvatarURL:                   gh.AvatarURL,
		Type:                        "Organization",
		HasOrganizationProjects:     true,
		HasRepositoryProjects:       true,
		MembersCanCreateRepositories: true,
		MembersCanCreatePublicRepositories: true,
		MembersCanCreatePrivateRepositories: true,
		DefaultRepositoryPermission: "read",
		CreatedAt:                   time.Now(),
		UpdatedAt:                   time.Now(),
	}
}

// mapIssue maps a GitHub issue to a GitHome issue.
func mapIssue(gh *ghIssue, repoID int64, creatorID int64) *issues.Issue {
	issue := &issues.Issue{
		RepoID:           repoID,
		Number:           gh.Number,
		Title:            gh.Title,
		Body:             gh.Body,
		State:            gh.State,
		StateReason:      gh.StateReason,
		CreatorID:        creatorID,
		Locked:           gh.Locked,
		ActiveLockReason: gh.ActiveLockReason,
		Comments:         gh.Comments,
		ClosedAt:         gh.ClosedAt,
		CreatedAt:        gh.CreatedAt,
		UpdatedAt:        gh.UpdatedAt,
	}
	return issue
}

// mapPullRequest maps a GitHub pull request to a GitHome pull request.
func mapPullRequest(gh *ghPullRequest, repoID int64, creatorID int64) *pulls.PullRequest {
	pr := &pulls.PullRequest{
		RepoID:           repoID,
		Number:           gh.Number,
		Title:            gh.Title,
		Body:             gh.Body,
		State:            gh.State,
		CreatorID:        creatorID,
		Locked:           gh.Locked,
		ActiveLockReason: gh.ActiveLockReason,
		Draft:            gh.Draft,
		Merged:           gh.Merged,
		Mergeable:        gh.Mergeable,
		MergeableState:   gh.MergeableState,
		MergedAt:         gh.MergedAt,
		MergeCommitSHA:   gh.MergeCommitSHA,
		Comments:         gh.Comments,
		ReviewComments:   gh.ReviewComments,
		Commits:          gh.Commits,
		Additions:        gh.Additions,
		Deletions:        gh.Deletions,
		ChangedFiles:     gh.ChangedFiles,
		ClosedAt:         gh.ClosedAt,
		CreatedAt:        gh.CreatedAt,
		UpdatedAt:        gh.UpdatedAt,
	}

	// Map head/base branches
	if gh.Head != nil {
		pr.Head = &pulls.PRBranch{
			Label: gh.Head.Label,
			Ref:   gh.Head.Ref,
			SHA:   gh.Head.SHA,
		}
	}
	if gh.Base != nil {
		pr.Base = &pulls.PRBranch{
			Label: gh.Base.Label,
			Ref:   gh.Base.Ref,
			SHA:   gh.Base.SHA,
		}
	}

	return pr
}

// mapLabel maps a GitHub label to a GitHome label.
func mapLabel(gh *ghLabel, repoID int64) *labels.Label {
	return &labels.Label{
		RepoID:      repoID,
		Name:        gh.Name,
		Description: gh.Description,
		Color:       gh.Color,
		Default:     gh.Default,
	}
}

// mapMilestone maps a GitHub milestone to a GitHome milestone.
func mapMilestone(gh *ghMilestone, repoID int64, creatorID int64) *milestones.Milestone {
	return &milestones.Milestone{
		RepoID:       repoID,
		Number:       gh.Number,
		Title:        gh.Title,
		Description:  gh.Description,
		State:        gh.State,
		CreatorID:    creatorID,
		OpenIssues:   gh.OpenIssues,
		ClosedIssues: gh.ClosedIssues,
		ClosedAt:     gh.ClosedAt,
		DueOn:        gh.DueOn,
		CreatedAt:    gh.CreatedAt,
		UpdatedAt:    gh.UpdatedAt,
	}
}

// mapIssueComment maps a GitHub comment to a GitHome issue comment.
func mapIssueComment(gh *ghComment, issueID, repoID, creatorID int64) *comments.IssueComment {
	return &comments.IssueComment{
		IssueID:   issueID,
		RepoID:    repoID,
		CreatorID: creatorID,
		Body:      gh.Body,
		CreatedAt: gh.CreatedAt,
		UpdatedAt: gh.UpdatedAt,
	}
}

// mapReviewComment maps a GitHub PR review comment to a GitHome review comment.
func mapReviewComment(gh *ghReviewComment, prID int64, creatorID int64) *pulls.ReviewComment {
	rc := &pulls.ReviewComment{
		PRID:                prID,
		UserID:              creatorID,
		Body:                gh.Body,
		DiffHunk:            gh.DiffHunk,
		Path:                gh.Path,
		CommitID:            gh.CommitID,
		OriginalCommitID:    gh.OriginalCommitID,
		InReplyToID:         gh.InReplyToID,
		Side:                gh.Side,
		StartSide:           gh.StartSide,
		CreatedAt:           gh.CreatedAt,
		UpdatedAt:           gh.UpdatedAt,
	}

	if gh.PullRequestReviewID != 0 {
		reviewID := gh.PullRequestReviewID
		rc.ReviewID = &reviewID
	}
	if gh.Position != nil {
		rc.Position = *gh.Position
	}
	if gh.OriginalPosition != nil {
		rc.OriginalPosition = *gh.OriginalPosition
	}
	if gh.Line != nil {
		rc.Line = *gh.Line
	}
	if gh.OriginalLine != nil {
		rc.OriginalLine = *gh.OriginalLine
	}
	if gh.StartLine != nil {
		rc.StartLine = *gh.StartLine
	}
	if gh.OriginalStartLine != nil {
		rc.OriginalStartLine = *gh.OriginalStartLine
	}

	return rc
}
