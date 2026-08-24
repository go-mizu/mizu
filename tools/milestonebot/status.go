package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Status prints where every milestone is.
//
// It reads and never writes, which makes it the command to run when you want
// to know whether the checklists still parse. A tracking issue whose items the
// tool cannot see reports zero of zero here, months before anybody notices
// that merges stopped ticking anything.
func Status(ctx context.Context, c *Client) error {
	issues, err := c.IssuesByLabel(ctx, "tracking")
	if err != nil {
		return err
	}
	if len(issues) == 0 {
		return fmt.Errorf("no open issue carries the `tracking` label, so there is nothing to report")
	}
	sort.Slice(issues, func(i, j int) bool { return issues[i].Number < issues[j].Number })

	var totalDone, totalItems int
	for _, is := range issues {
		items := ParseChecklist(is.Body)
		done, total := Progress(items)
		totalDone += done
		totalItems += total

		fmt.Printf("%-14s %s %3d/%-3d  #%d %s\n",
			milestoneOf(is), progressBar(done, total), done, total, is.Number, is.Title)
		if total == 0 {
			fmt.Println("               no identified checklist items, so no merge can tick anything here")
		}
	}
	fmt.Printf("\n%d of %d items across %d milestones\n", totalDone, totalItems, len(issues))
	return nil
}

// milestoneOf reports the milestone a tracking issue belongs to, which is the
// suffix of its milestone label.
func milestoneOf(is Issue) string {
	for _, l := range is.Labels {
		if after, ok := strings.CutPrefix(l.Name, "milestone/"); ok {
			return after
		}
	}
	return "?"
}
