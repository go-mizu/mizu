package main

import (
	"strings"
	"testing"
)

func TestTypeFromTitle(t *testing.T) {
	for _, tc := range []struct {
		title    string
		want     string
		breaking bool
	}{
		{"feat(cache): add Flexible", "type/feature", false},
		{"fix: router drops the trailing slash", "type/bug", false},
		{"feat(errs)!: close the Kind taxonomy", "type/feature", true},
		{"docs: write the cache page", "type/docs", false},
		{"ci: pin the runner image", "type/build", false},
		{"sec: rotate the signing key", "type/security", false},
		{"Add a thing", "", false},
		{"feat:no space after the colon", "", false},
	} {
		got, breaking := TypeFromTitle(tc.title)
		if got != tc.want || breaking != tc.breaking {
			t.Errorf("TypeFromTitle(%q) = %q, %v; want %q, %v", tc.title, got, breaking, tc.want, tc.breaking)
		}
	}
}

func TestAreasFromPaths(t *testing.T) {
	for _, tc := range []struct {
		name  string
		repo  string
		paths []string
		want  string
	}{
		{"seams", "go-mizu/mizu", []string{"errs/kind.go", "errs/kind_test.go"}, "area/seams"},
		{"the standard library extensions are seams too", "go-mizu/mizu", []string{"conc/group.go", "try/try.go"}, "area/seams"},
		{"two areas", "go-mizu/mizu", []string{"cache/cache.go", "db/db.go"}, "area/async area/db"},
		{"codegen beats cli", "go-mizu/mizu", []string{"cmd/mizu/gen/rpc.go"}, "area/codegen"},
		{"cli", "go-mizu/mizu", []string{"cmd/mizu/new.go"}, "area/cli"},
		{"workflows", "go-mizu/mizu", []string{".github/workflows/ci.yml"}, "area/ci"},
		{"the bot is repository tooling", "go-mizu/mizu", []string{"tools/milestonebot/main.go"}, "area/ci"},
		{"release scripting", "go-mizu/mizu", []string{"scripts/tag-release.sh"}, "area/ci"},
		{"the import graph assertions are test tooling", "go-mizu/mizu", []string{"archtest/archtest.go"}, "area/testing"},
		{"top level router file", "go-mizu/mizu", []string{"router.go"}, "area/router"},
		{"unknown path gets nothing", "go-mizu/mizu", []string{"NOTICE"}, ""},

		// The same paths mean different things in the three repositories, which
		// is why each has its own rule set.
		{"a page in the site", "go-mizu/docs", []string{"content/docs/cache.mdx"}, "area/content"},
		{"the site application", "go-mizu/docs", []string{"src/layouts/Doc.astro"}, "area/site"},
		{"the ingest pipeline", "go-mizu/docs", []string{"artefacts/api-index.json"}, "area/artefacts"},
		{"content is not a site path in the toolkit", "go-mizu/mizu", []string{"content/docs/cache.mdx"}, ""},
		{"a design token", "go-mizu/shizuku", []string{"css/tokens/color.css"}, "area/tokens"},
		{"a component beats the token rule by order", "go-mizu/shizuku", []string{"css/button.css"}, "area/components"},
		{"an unknown repository falls back to the toolkit", "someone/fork", []string{"errs/kind.go"}, "area/seams"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.Join(AreasFromPaths(rulesFor(tc.repo), tc.paths), " ")
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSizeLabel(t *testing.T) {
	for _, tc := range []struct {
		add, del int
		want     string
	}{
		{5, 0, "size/xs"},
		{50, 0, "size/s"},
		{0, 400, "size/m"},   // deleting 400 lines reads like adding 200
		{300, 100, "size/m"}, // 350
		{900, 0, "size/l"},
		{2000, 0, "size/xl"},
	} {
		if got := SizeLabel(tc.add, tc.del); got != tc.want {
			t.Errorf("SizeLabel(%d, %d) = %q, want %q", tc.add, tc.del, got, tc.want)
		}
	}
}

func TestMilestoneLabel(t *testing.T) {
	for _, tc := range []struct {
		name string
		ids  []string
		want string
	}{
		{"one", []string{"M0-03"}, "milestone/M0"},
		{"same milestone twice", []string{"M3-01", "M3-02"}, "milestone/M3"},
		{"two milestones is a split, not a label", []string{"M0-03", "M1-02"}, ""},
		{"the prefix is spelled through unchanged", []string{"site-04"}, "milestone/site"},
		{"none", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := MilestoneLabel(tc.ids); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestClassify(t *testing.T) {
	pr := &PullRequest{
		Title:     "feat(cache): add the memory driver",
		Body:      "Adds TinyLFU admission.\n\nChecklist: M6-03\n",
		Additions: 220,
		Deletions: 40,
		Labels:    []Label{{Name: "type/feature"}},
	}
	got := Classify("go-mizu/mizu", pr, []string{"cache/memory.go", "cache/memory_test.go"})
	want := "type/feature area/async size/m milestone/M6"
	if joined := strings.Join(got.Labels(), " "); joined != want {
		t.Errorf("labels = %q, want %q", joined, want)
	}
	// The type label is already on the pull request, so only the news is sent.
	if joined := strings.Join(Missing(pr, got.Labels()), " "); joined != "area/async size/m milestone/M6" {
		t.Errorf("missing = %q", joined)
	}
}

func TestResolveNeverInventsALabel(t *testing.T) {
	existing := []Label{{Name: "milestone/M0"}, {Name: "type/bug"}}
	found, unknown := resolve([]string{"milestone/m0", "type/bug", "area/nope"}, existing)

	// Case is normalised to the repository's spelling, so the API is asked for
	// a name it already has rather than one it would create.
	if strings.Join(found, " ") != "milestone/M0 type/bug" {
		t.Errorf("found = %v", found)
	}
	if strings.Join(unknown, " ") != "area/nope" {
		t.Errorf("unknown = %v", unknown)
	}
}

func TestClassifyMarksBreaking(t *testing.T) {
	pr := &PullRequest{Title: "refactor(errs)!: rename Kind", Additions: 10}
	if !Classify("go-mizu/mizu", pr, nil).Breaking {
		t.Error("the exclamation mark in the title should mark it breaking")
	}
	pr = &PullRequest{Title: "refactor(errs): rename Kind", Body: "BREAKING CHANGE: Kind is now closed.", Additions: 10}
	if !Classify("go-mizu/mizu", pr, nil).Breaking {
		t.Error("the footer should mark it breaking")
	}
}
