package console

import (
	"fmt"
	"strings"
)

// A TreeNode is one line of a tree and whatever hangs below it.
//
// The label is the whole line, already formatted. A node holds a string rather
// than columns because the things drawn as trees here, a route table, a
// directory, a dependency, all have one name per line and their own idea of
// what belongs next to it.
type TreeNode struct {
	Label    string     `json:"label"`
	Children []TreeNode `json:"children,omitempty"`
}

// Tree writes a node and its children to stdout, drawn with box characters.
//
//	app
//	├── cmd
//	│   └── main.go
//	└── go.mod
//
// In JSON mode it writes the node as an object instead, with label and children
// members, for the same reason [IO.Table] writes an array: a command that draws
// a tree supports --json without building the tree twice.
//
// It goes to stdout because a tree is the answer to what was asked, not the
// program talking about itself. That holds inside a [IO.Section] too, so a tree
// drawn under a title is not indented along with the title. Indenting it would
// mean the shape of the data depends on where in the command it was printed.
func (c *IO) Tree(root TreeNode) {
	if c.jsonMode {
		// A failed write to stdout is the caller's pipe closing, and there is
		// nothing this can do about it that the next write will not do again.
		_ = c.JSON(root)
		return
	}

	var b strings.Builder
	b.WriteString(root.Label)
	b.WriteByte('\n')
	// Sixteen levels of prefix up front. Past that it grows, and a tree that
	// deep has other problems.
	writeChildren(&b, root.Children, make([]byte, 0, 64))
	fmt.Fprint(c.out, b.String())
}

// writeChildren draws one level and recurses.
//
// The prefix carries the vertical bars belonging to the levels above, which is
// what makes a deep tree readable: every line already knows which of its
// ancestors have more siblings coming.
//
// It is a byte slice that grows and is reused rather than a string that is
// joined, so a tree of four hundred routes does not allocate a prefix per line.
// Appending to it is safe across siblings because each one writes its own tail
// at the same offset before recursing, and the slice this call holds keeps its
// own length whatever the levels below did.
func writeChildren(b *strings.Builder, nodes []TreeNode, prefix []byte) {
	const (
		branch = "├── "
		last   = "└── "
		bar    = "│   "
		gap    = "    "
	)
	for i, node := range nodes {
		head, tail := branch, bar
		if i == len(nodes)-1 {
			head, tail = last, gap
		}
		b.Write(prefix)
		b.WriteString(head)
		b.WriteString(node.Label)
		b.WriteByte('\n')
		if len(node.Children) > 0 {
			writeChildren(b, node.Children, append(prefix, tail...))
		}
	}
}
