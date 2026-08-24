package main

import (
	"regexp"
	"sort"
	"strings"
)

// Classification is the set of labels the tool believes a pull request should
// carry. It is a suggestion in the sense that a maintainer can add more, and a
// rule in the sense that the tool applies it without asking.
type Classification struct {
	Type      string   // exactly one, or empty if the title has no prefix
	Areas     []string // zero or more, sorted
	Size      string   // exactly one
	Milestone string   // one, or empty if the description names no checklist item
	Breaking  bool
}

// Labels flattens a classification into the label names to add.
func (c Classification) Labels() []string {
	var out []string
	if c.Type != "" {
		out = append(out, c.Type)
	}
	out = append(out, c.Areas...)
	if c.Size != "" {
		out = append(out, c.Size)
	}
	if c.Milestone != "" {
		out = append(out, c.Milestone)
	}
	if c.Breaking {
		out = append(out, "breaking")
	}
	return out
}

// conventional matches the commit-message prefix the repository requires in a
// pull request title, for example `feat(cache): add Flexible`. The exclamation
// mark before the colon is the conventional-commits marker for a breaking
// change.
var conventional = regexp.MustCompile(`^([a-z]+)(\([^)]*\))?(!)?:\s`)

// typeByPrefix maps a conventional-commit prefix to a type label. Prefixes not
// in this table are rejected by the title check in CI, so an unknown prefix
// here means the two have drifted and the pull request gets no type label
// rather than a wrong one.
var typeByPrefix = map[string]string{
	"feat":     "type/feature",
	"fix":      "type/bug",
	"docs":     "type/docs",
	"perf":     "type/perf",
	"refactor": "type/refactor",
	"test":     "type/test",
	"build":    "type/build",
	"ci":       "type/build",
	"chore":    "type/build",
	"sec":      "type/security",
	"revert":   "type/bug",
}

// TypeFromTitle reads the conventional-commit prefix. The second return value
// reports whether the title marked itself as breaking.
func TypeFromTitle(title string) (label string, breaking bool) {
	m := conventional.FindStringSubmatch(strings.TrimSpace(title))
	if m == nil {
		return "", false
	}
	return typeByPrefix[m[1]], m[3] == "!"
}

// areaRules maps a path prefix to an area label. The list is ordered from
// most specific to least, and the first prefix that matches wins, so
// `cmd/mizu/gen/` reaches codegen before `cmd/` reaches the CLI.
//
// A path that matches nothing gets no area label. That is deliberate: a
// missing label is a question for the reviewer, and a wrong one is a lie in
// the search index.
var areaRules = []struct{ prefix, label string }{
	{".github/", "area/ci"},
	{"tools/", "area/ci"},
	{"cmd/mizu/gen/", "area/codegen"},
	{"cmd/", "area/cli"},
	{"gen/", "area/codegen"},

	{"errs/", "area/seams"},
	{"ctxdata/", "area/seams"},
	{"clock/", "area/seams"},
	{"log/", "area/seams"},
	{"otelx/", "area/observability"},
	{"codec/", "area/seams"},

	{"web/", "area/router"},
	{"router", "area/router"},
	{"httpc/", "area/router"},
	{"rpc/", "area/rpc"},

	{"db/", "area/db"},
	{"query/", "area/db"},
	{"orm/", "area/orm"},
	{"migrate/", "area/orm"},

	{"view/", "area/views"},
	{"vite/", "area/views"},
	{"islands/", "area/views"},
	{"ssr/", "area/views"},
	{"hx/", "area/views"},
	{"inertia/", "area/views"},

	{"session/", "area/auth"},
	{"auth/", "area/auth"},
	{"gate/", "area/auth"},
	{"csrf/", "area/auth"},
	{"crypt/", "area/auth"},
	{"hash/", "area/auth"},

	{"cache/", "area/async"},
	{"lock/", "area/async"},
	{"queue/", "area/async"},
	{"schedule/", "area/async"},
	{"event/", "area/async"},

	{"mail/", "area/messaging"},
	{"notify/", "area/messaging"},
	{"storage/", "area/messaging"},
	{"broadcast/", "area/messaging"},

	{"telescope/", "area/observability"},
	{"pulse/", "area/observability"},
	{"health/", "area/observability"},

	{"mizutest/", "area/testing"},
	{"blueprints/", "area/cli"},
}

// AreasFromPaths derives the area labels for a set of changed files.
func AreasFromPaths(paths []string) []string {
	seen := map[string]bool{}
	for _, p := range paths {
		p = strings.TrimPrefix(p, "./")
		for _, r := range areaRules {
			if strings.HasPrefix(p, r.prefix) {
				seen[r.label] = true
				break
			}
		}
		// A package that ends in `test` is a test helper wherever it lives,
		// and a conformance suite is the deliverable rather than a detail.
		if strings.Contains(p, "test/") && strings.Contains(p, "conformance") {
			seen["area/testing"] = true
		}
	}
	out := make([]string, 0, len(seen))
	for a := range seen {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

// SizeLabel buckets a pull request by how much there is to read. Deletions
// count half, because deleting 400 lines is genuinely less work to review than
// adding them and pretending otherwise pushes people towards leaving dead code
// in place.
func SizeLabel(additions, deletions int) string {
	n := additions + deletions/2
	switch {
	case n < 20:
		return "size/xs"
	case n < 100:
		return "size/s"
	case n < 400:
		return "size/m"
	case n < 1000:
		return "size/l"
	default:
		return "size/xl"
	}
}

// MilestoneLabel returns the milestone label implied by the checklist items a
// pull request claims. Items from two different milestones in one pull request
// return nothing, because that pull request wants splitting and a label saying
// otherwise would hide it.
//
// The prefix goes through unchanged, so `M0-03` gives `milestone/M0` and
// `site-04` gives `milestone/site`. That is the rule: an identifier's prefix is
// spelled the same way as the milestone label. Getting it wrong produces a
// label that does not exist, and LabelPR drops those rather than creating
// them, so a typo costs a missing label rather than a new one nobody meant.
func MilestoneLabel(ids []string) string {
	var ms string
	for _, id := range ids {
		m, _, ok := strings.Cut(id, "-")
		if !ok {
			continue
		}
		if ms == "" {
			ms = m
		} else if !strings.EqualFold(ms, m) {
			return ""
		}
	}
	if ms == "" {
		return ""
	}
	return "milestone/" + ms
}

// Classify puts the four rules together.
func Classify(pr *PullRequest, paths []string) Classification {
	typ, breaking := TypeFromTitle(pr.Title)
	ids := ChecklistIDs(pr.Body)
	return Classification{
		Type:      typ,
		Areas:     AreasFromPaths(paths),
		Size:      SizeLabel(pr.Additions, pr.Deletions),
		Milestone: MilestoneLabel(ids),
		Breaking:  breaking || strings.Contains(pr.Body, "BREAKING CHANGE:"),
	}
}

// resolve maps wanted label names onto the repository's real ones, matching
// without regard to case so that `milestone/m0` finds `milestone/M0`. Names
// with no match come back separately rather than being passed through, because
// GitHub creates a label it has not seen before instead of rejecting it.
func resolve(want []string, existing []Label) (found, unknown []string) {
	byLower := make(map[string]string, len(existing))
	for _, l := range existing {
		byLower[strings.ToLower(l.Name)] = l.Name
	}
	for _, w := range want {
		if name, ok := byLower[strings.ToLower(w)]; ok {
			found = append(found, name)
		} else {
			unknown = append(unknown, w)
		}
	}
	return found, unknown
}

// Missing returns the labels from a classification that a pull request does
// not already carry, so the tool sends one request with only the news in it.
func Missing(pr *PullRequest, want []string) []string {
	have := map[string]bool{}
	for _, l := range pr.Labels {
		have[l.Name] = true
	}
	var out []string
	for _, w := range want {
		if !have[w] {
			out = append(out, w)
		}
	}
	return out
}
