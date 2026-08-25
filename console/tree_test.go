package console

import (
	"strings"
	"testing"
)

// project is the shape most trees have: something nested next to something
// that is not, so both the bar and the gap under a branch get drawn.
var project = TreeNode{
	Label: "app",
	Children: []TreeNode{
		{Label: "cmd", Children: []TreeNode{
			{Label: "mizu", Children: []TreeNode{{Label: "main.go"}}},
		}},
		{Label: "go.mod"},
	},
}

func TestTree(t *testing.T) {
	s := newStreams(t, Options{})

	s.io.Tree(project)

	want := "app\n" +
		"├── cmd\n" +
		"│   └── mizu\n" +
		"│       └── main.go\n" +
		"└── go.mod\n"
	if got := s.out.String(); got != want {
		t.Errorf("stdout has\n%s\nwant\n%s", got, want)
	}
}

// TestTreeReusesThePrefix is the test for the buffer the prefix is built in.
// The line under a2 has to say "│       ", and the four bytes in the middle of
// it are where the branch under a1 wrote "│   " a moment earlier.
func TestTreeReusesThePrefix(t *testing.T) {
	s := newStreams(t, Options{})

	s.io.Tree(TreeNode{Label: "root", Children: []TreeNode{
		{Label: "a", Children: []TreeNode{
			{Label: "a1", Children: []TreeNode{{Label: "deep"}}},
			{Label: "a2", Children: []TreeNode{{Label: "also"}}},
		}},
		{Label: "b"},
	}})

	want := "root\n" +
		"├── a\n" +
		"│   ├── a1\n" +
		"│   │   └── deep\n" +
		"│   └── a2\n" +
		"│       └── also\n" +
		"└── b\n"
	if got := s.out.String(); got != want {
		t.Errorf("stdout has\n%s\nwant\n%s", got, want)
	}
}

// TestDeepTree goes past the sixteen levels the prefix buffer is sized for, so
// the growth path is the one being drawn here.
func TestDeepTree(t *testing.T) {
	const depth = 20

	node := TreeNode{Label: "leaf"}
	for range depth {
		node = TreeNode{Label: "level", Children: []TreeNode{node}}
	}

	s := newStreams(t, Options{})
	s.io.Tree(node)

	lines := strings.Split(strings.TrimSuffix(s.out.String(), "\n"), "\n")
	if len(lines) != depth+1 {
		t.Fatalf("drew %d lines, want %d", len(lines), depth+1)
	}
	// Every node is an only child, so each level is indented by one more gap
	// than the one above and the last line is the deepest.
	want := strings.Repeat("    ", depth-1) + "└── leaf"
	if got := lines[depth]; got != want {
		t.Errorf("the deepest line is %q, want %q", got, want)
	}
}

func TestTreeGoesToStdout(t *testing.T) {
	s := newStreams(t, Options{})

	s.io.Tree(project)

	if got := s.err.String(); got != "" {
		t.Errorf("stderr has %q, want the tree to have gone to stdout", got)
	}
}

func TestTreeWithNothingUnderIt(t *testing.T) {
	s := newStreams(t, Options{})

	s.io.Tree(TreeNode{Label: "app"})

	if got := s.out.String(); got != "app\n" {
		t.Errorf("stdout has %q, want %q", got, "app\n")
	}
}

func TestTreeInJSONMode(t *testing.T) {
	s := newStreams(t, Options{JSON: true})

	s.io.Tree(TreeNode{Label: "app", Children: []TreeNode{{Label: "go.mod"}}})

	want := `{
  "label": "app",
  "children": [
    {
      "label": "go.mod"
    }
  ]
}
`
	if got := s.out.String(); got != want {
		t.Errorf("stdout has\n%s\nwant\n%s", got, want)
	}
}

// TestTreeInJSONModeLeavesOutEmptyChildren keeps the leaves from carrying a
// member that says nothing, which is what a jq expression walking the tree
// would otherwise have to test for.
func TestTreeInJSONModeLeavesOutEmptyChildren(t *testing.T) {
	s := newStreams(t, Options{JSON: true})

	s.io.Tree(TreeNode{Label: "app"})

	want := "{\n  \"label\": \"app\"\n}\n"
	if got := s.out.String(); got != want {
		t.Errorf("stdout has\n%s\nwant\n%s", got, want)
	}
}

// TestTreeInASectionIsNotIndented is the same rule as [TestSectionLeavesDataAlone],
// written again here because a tree looks like decoration and is not.
func TestTreeInASectionIsNotIndented(t *testing.T) {
	s := newStreams(t, Options{})

	s.io.Section("Layout").Tree(TreeNode{Label: "app", Children: []TreeNode{{Label: "go.mod"}}})

	want := "app\n└── go.mod\n"
	if got := s.out.String(); got != want {
		t.Errorf("stdout has\n%s\nwant\n%s", got, want)
	}
	if got := s.err.String(); got != "Layout\n" {
		t.Errorf("stderr has %q, want just the title", got)
	}
}
