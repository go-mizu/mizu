package main

import (
	"context"
	"fmt"
	"strings"
)

// LabelPR puts the derived labels on a pull request.
//
// In strict mode a title the tool cannot read is an error, which is how the
// convention is enforced. Labels are applied first either way, so a
// contributor who has to fix the title still gets everything else the tool
// worked out.
func LabelPR(ctx context.Context, c *Client, number int, strict bool) error {
	pr, err := c.PullRequest(ctx, number)
	if err != nil {
		return err
	}
	paths, err := c.ChangedFiles(ctx, number)
	if err != nil {
		return err
	}

	// Resolve against the repository's real labels first. Adding a label that
	// does not exist creates it, so a bug in the classifier or a typo in a
	// Checklist trailer would quietly grow the label set, which is the one
	// thing .github/labels.yml exists to prevent.
	existing, err := c.Labels(ctx)
	if err != nil {
		return err
	}
	class := Classify(c.Repo, pr, paths)
	want, unknown := resolve(class.Labels(), existing)
	for _, u := range unknown {
		fmt.Printf("#%d: no label named %q on this repository, skipping it\n", number, u)
	}

	if add := Missing(pr, want); len(add) > 0 {
		if err := c.AddLabels(ctx, number, add); err != nil {
			return err
		}
		fmt.Printf("#%d labelled: %s\n", number, strings.Join(add, ", "))
	} else {
		fmt.Printf("#%d already carries every label this would add\n", number)
	}

	if len(class.Areas) == 0 {
		fmt.Printf("#%d touches no path this tool recognises, so it got no area label\n", number)
	}
	if class.Type != "" {
		return nil
	}

	msg := fmt.Sprintf("#%d has no type label because its title does not start with a recognised prefix.\n"+
		"Titles look like `feat(cache): add the memory driver`.\n"+
		"The prefixes are feat, fix, docs, perf, refactor, test, build, ci, chore, sec, and revert,\n"+
		"and an exclamation mark before the colon marks a breaking change.", number)
	if strict {
		return fmt.Errorf("%s", msg)
	}
	fmt.Println(msg)
	return nil
}

// OnMerge ticks the checklist items a merged pull request finished and says so
// on each tracking issue.
//
// The pull request is read from c and the tracking issue is updated through
// tracker, which is usually the same client. They differ when the work happens
// in one repository and the checklist lives in another, which is the case for
// the site: its items are tracked in go-mizu/mizu and merged in go-mizu/docs.
// The two clients can carry different tokens, which is how each one gets only
// the access it needs.
//
// Two properties matter more than they look. It is idempotent, so re-running a
// workflow does not tick anything twice or double-comment. And it fails loudly
// on an identifier that does not exist, because a silent no-op there means the
// checklist quietly stops tracking reality, which is the exact failure this
// tool exists to prevent.
func OnMerge(ctx context.Context, c, tracker *Client, number int) error {
	pr, err := c.PullRequest(ctx, number)
	if err != nil {
		return err
	}
	if !pr.Merged {
		fmt.Printf("#%d is closed but not merged, nothing to do\n", number)
		return nil
	}

	ids := ChecklistIDs(pr.Body)
	if len(ids) == 0 {
		fmt.Printf("#%d names no checklist item, nothing to do\n", number)
		return nil
	}

	for _, id := range ids {
		if err := tickOne(ctx, tracker, pr, id); err != nil {
			return err
		}
	}
	return nil
}

func tickOne(ctx context.Context, c *Client, pr *PullRequest, id string) error {
	ms := Item{ID: id}.Milestone()
	if ms == "" {
		return fmt.Errorf("#%d: checklist identifier %q has no milestone prefix, and the shape is M0-03", pr.Number, id)
	}
	label := "milestone/" + ms

	// Every tracking issue at once, then match here. Filtering server side
	// would mean the identifier's case had to match the label's exactly, and
	// `m0-03` in a description should still find M0. There are as many
	// tracking issues as there are milestones, so this is one request.
	all, err := c.IssuesByLabel(ctx, "tracking")
	if err != nil {
		return err
	}
	var issues []Issue
	for _, is := range all {
		for _, l := range is.Labels {
			if strings.EqualFold(l.Name, label) {
				issues = append(issues, is)
				break
			}
		}
	}

	switch len(issues) {
	case 1:
	case 0:
		return fmt.Errorf("#%d names %s but no open issue carries both `tracking` and `%s`", pr.Number, id, label)
	default:
		var ns []string
		for _, is := range issues {
			ns = append(ns, fmt.Sprintf("#%d", is.Number))
		}
		return fmt.Errorf("#%d names %s and %s all carry both `tracking` and `%s`, so there is no single tracking issue to update", pr.Number, id, strings.Join(ns, ", "), label)
	}
	issue := issues[0]

	body, item, changed, err := Tick(issue.Body, id)
	if err != nil {
		return fmt.Errorf("#%d: tracking issue #%d: %w", pr.Number, issue.Number, err)
	}
	if !changed {
		fmt.Printf("#%d: %s was already ticked on #%d, leaving it and the thread alone\n", pr.Number, id, issue.Number)
		return nil
	}
	if err := c.UpdateIssueBody(ctx, issue.Number, body); err != nil {
		return fmt.Errorf("tick %s on #%d: %w", id, issue.Number, err)
	}

	done, total := Progress(ParseChecklist(body))
	if err := c.Comment(ctx, issue.Number, mergeComment(pr, item, done, total)); err != nil {
		return fmt.Errorf("comment on #%d: %w", issue.Number, err)
	}
	fmt.Printf("#%d: ticked %s on #%d, now %d of %d\n", pr.Number, id, issue.Number, done, total)
	return nil
}

// mergeComment is the note left on a tracking issue. It says what landed, who
// landed it, and what is left, in that order, because somebody skimming the
// thread six months from now wants the third of those most.
func mergeComment(pr *PullRequest, item Item, done, total int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**`%s` is done.** %s\n\n", item.ID, item.Text)
	fmt.Fprintf(&b, "Landed in %s by @%s.\n\n", pr.HTMLURL, pr.User.Login)
	fmt.Fprintf(&b, "%s\n\n", progressBar(done, total))
	fmt.Fprintf(&b, "%d of %d items in this milestone are done.", done, total)
	if done == total {
		b.WriteString(" That is the whole checklist. What is left is the acceptance criteria below it, which are ticked by someone checking them rather than by a merge.")
	}
	return b.String()
}

// progressBar draws the checklist as twenty cells. A number is precise and a
// bar is legible, so the comment carries both.
func progressBar(done, total int) string {
	const width = 20
	if total == 0 {
		return ""
	}
	filled := done * width / total
	return "`" + strings.Repeat("#", filled) + strings.Repeat(".", width-filled) + "`"
}
