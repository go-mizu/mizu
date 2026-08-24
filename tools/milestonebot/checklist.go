package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Item is one checkbox in a tracking issue.
//
// The identifier is what makes this work. Matching on the item's text would
// break the moment somebody fixed a typo in the tracking issue, and matching
// on position would break the moment somebody inserted a line, so every item
// carries a short identifier that is assigned once and then left alone.
type Item struct {
	ID   string // for example M0-03
	Text string // the prose after the identifier
	Done bool
	Line int // zero-based index into the body's lines, for a precise error
}

// Milestone returns the part of the identifier before the dash, which is the
// milestone the item belongs to.
func (it Item) Milestone() string {
	id, _, _ := strings.Cut(it.ID, "-")
	return id
}

// itemLine matches a checklist line and nothing else. The identifier is in
// backticks so it reads as a token rather than as part of the sentence, and
// the leading whitespace is captured so a nested list round-trips unchanged.
var itemLine = regexp.MustCompile("^(\\s*)- \\[([ xX])\\] `([A-Za-z][A-Za-z0-9]*-[0-9]+)`\\s*(.*)$")

// ParseChecklist returns every identified checklist item in an issue body, in
// the order they appear. A checkbox without an identifier is not an item: the
// tracking issues use plain checkboxes for acceptance criteria, which are
// ticked by a human reading the criterion rather than by a merge.
func ParseChecklist(body string) []Item {
	lines := strings.Split(body, "\n")
	var items []Item
	for i, line := range lines {
		m := itemLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		items = append(items, Item{
			ID:   m[3],
			Text: strings.TrimSpace(m[4]),
			Done: m[2] == "x" || m[2] == "X",
			Line: i,
		})
	}
	return items
}

// Find returns the item with the given identifier.
func Find(items []Item, id string) (Item, bool) {
	for _, it := range items {
		if strings.EqualFold(it.ID, id) {
			return it, true
		}
	}
	return Item{}, false
}

// Progress counts the ticked items and the total.
func Progress(items []Item) (done, total int) {
	for _, it := range items {
		if it.Done {
			done++
		}
	}
	return done, len(items)
}

// Tick marks one item done and returns the rewritten body.
//
// Ticking an item that is already ticked is not an error and not a rewrite:
// two pull requests can legitimately finish parts of the same item, and the
// second one should still comment on the thread rather than fail the workflow.
// The returned bool says whether the body changed, so the caller can skip the
// write.
func Tick(body, id string) (string, Item, bool, error) {
	items := ParseChecklist(body)
	it, ok := Find(items, id)
	if !ok {
		return body, Item{}, false, fmt.Errorf("no checklist item %q in this issue, and the identifiers are assigned once and never renumbered, so check the identifier rather than the position", id)
	}
	if it.Done {
		return body, it, false, nil
	}

	lines := strings.Split(body, "\n")
	m := itemLine.FindStringSubmatch(lines[it.Line])
	lines[it.Line] = fmt.Sprintf("%s- [x] `%s` %s", m[1], m[3], strings.TrimSpace(m[4]))
	return strings.Join(lines, "\n"), it, true, nil
}

// trailer matches a `Key: value` line at the start of a line, which is the
// shape git already uses for Signed-off-by and Co-authored-by.
func trailer(body, key string) []string {
	re := regexp.MustCompile(`(?mi)^` + regexp.QuoteMeta(key) + `:[ \t]*(.+)$`)
	var out []string
	for _, m := range re.FindAllStringSubmatch(body, -1) {
		for _, f := range strings.Split(m[1], ",") {
			f = strings.TrimSpace(strings.Trim(strings.TrimSpace(f), "`"))
			if f != "" && !strings.EqualFold(f, "none") {
				out = append(out, f)
			}
		}
	}
	return out
}

// ChecklistIDs returns the identifiers a pull request description claims to
// finish. A description with no Checklist trailer returns nothing, which is
// the common case and not a problem.
func ChecklistIDs(prBody string) []string {
	// Comments in the pull request template explain the trailer, and a
	// contributor who leaves the template untouched should not accidentally
	// tick an example. Strip HTML comments before looking.
	body := regexp.MustCompile(`(?s)<!--.*?-->`).ReplaceAllString(prBody, "")

	seen := map[string]bool{}
	var ids []string
	for _, id := range trailer(body, "Checklist") {
		up := strings.ToUpper(id)
		if !seen[up] {
			seen[up] = true
			ids = append(ids, id)
		}
	}
	return ids
}
