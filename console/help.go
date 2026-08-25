package console

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// Help writes the command list to stdout.
//
// It goes to stdout because it is what was asked for, the same as any other
// output. Help that arrives on stderr cannot be piped into a pager or grepped
// for the command somebody half remembers.
func (a *App) Help(c *IO) {
	var b strings.Builder
	if a.Desc != "" {
		b.WriteString(a.Desc)
		b.WriteString("\n\n")
	}
	fmt.Fprintf(&b, "Usage:\n  %s <command> [flags]\n", strings.TrimSpace(a.Name+" "))

	if rows := a.commandRows(); len(rows) > 0 {
		b.WriteString("\nCommands:\n")
		writeRows(&b, rows)
		fmt.Fprintf(&b, "\nRun %q for more about one.\n", a.help()+" <command>")
	}
	fmt.Fprint(c.out, b.String())
}

// usage writes what one command takes.
func (a *App) usage(c *IO, spec Spec) {
	var b strings.Builder
	if spec.Desc != "" {
		b.WriteString(spec.Desc)
		b.WriteString("\n\n")
	}
	if spec.Long != "" {
		b.WriteString(strings.TrimSpace(spec.Long))
		b.WriteString("\n\n")
	}

	b.WriteString("Usage:\n  ")
	b.WriteString(strings.TrimSpace(a.Name + " " + spec.Name))
	if slices.ContainsFunc(spec.Flags, func(f Flag) bool { return !f.Hidden }) {
		b.WriteString(" [flags]")
	}
	for _, arg := range spec.Args {
		b.WriteString(" " + argShape(arg))
	}
	b.WriteByte('\n')

	if rows := argRows(spec.Args); len(rows) > 0 {
		b.WriteString("\nArguments:\n")
		writeRows(&b, rows)
	}
	b.WriteString("\nFlags:\n")
	writeRows(&b, flagRows(spec))

	fmt.Fprint(c.out, b.String())
}

// A row is one line of a help list: what to type, and what it means.
type row struct{ left, right string }

// commandRows lists the commands, ungrouped ones first and then a group per
// colon prefix, with a blank line between groups.
//
// The groups are the names themselves rather than a field on the spec, so a
// command cannot end up filed somewhere its name does not suggest. It is also
// what makes db:seed and db:wipe sit together without either of them saying so.
func (a *App) commandRows() []row {
	// The group is worked out once per command rather than inside the
	// comparison, which asks for it a few thousand times for a list this long.
	listed := make([]listing, 0, len(a.cmds))
	for _, e := range a.cmds {
		if !e.spec.Hidden {
			listed = append(listed, listing{group: group(e.spec.Name), spec: e.spec})
		}
	}
	slices.SortFunc(listed, byGroupThenName)

	rows := make([]row, 0, len(listed))
	for i, l := range listed {
		if i > 0 && l.group != listed[i-1].group {
			rows = append(rows, row{})
		}
		rows = append(rows, row{l.spec.Name, l.spec.Desc})
	}
	return rows
}

type listing struct {
	group string
	spec  Spec
}

// byGroupThenName puts the commands without a group first, because new and
// doctor are what somebody reaching for help is usually after, and a hundred
// make: commands above them would bury the answer.
func byGroupThenName(x, y listing) int {
	if x.group != y.group {
		switch {
		case x.group == "":
			return -1
		case y.group == "":
			return 1
		}
		return cmp.Compare(x.group, y.group)
	}
	return cmp.Compare(x.spec.Name, y.spec.Name)
}

// group is the part of a name before its colon, and nothing for a name that has
// none.
func group(name string) string {
	prefix, _, _ := strings.Cut(name, ":")
	if prefix == name {
		return ""
	}
	return prefix
}

func argRows(args []Arg) []row {
	rows := make([]row, 0, len(args))
	for _, a := range args {
		right := a.Desc
		if a.Default != "" {
			right = join(right, "(default "+a.Default+")")
		}
		rows = append(rows, row{a.Name, right})
	}
	return rows
}

func flagRows(spec Spec) []row {
	rows := make([]row, 0, len(spec.Flags)+1)
	for _, f := range spec.Flags {
		if f.Hidden {
			continue
		}
		left := "    --" + f.Name
		if f.Short != 0 {
			left = "-" + string(f.Short) + ", --" + f.Name
		}
		if k := kind(f.Value); k != "" {
			left += " " + k
		}

		right := f.Desc
		switch {
		case f.Required:
			right = join(right, "(required)")
		case f.Default != "":
			right = join(right, "(default "+f.Default+")")
		}
		if f.Env != "" {
			right = join(right, "[$"+f.Env+"]")
		}
		rows = append(rows, row{left, right})
	}

	// --help is not a declared flag, it is handled before the parse, and it is
	// still the flag most worth knowing about. A command that declared its own
	// keeps it and this stays quiet.
	if find(spec.Flags, func(f Flag) bool { return f.Name == "help" }) < 0 {
		left := "    --help"
		if find(spec.Flags, func(f Flag) bool { return f.Short == 'h' }) < 0 {
			left = "-h, --help"
		}
		rows = append(rows, row{left, "Show what this command takes"})
	}
	return rows
}

// kind is what a flag takes, for the help line. A flag that takes nothing says
// nothing, because --dry-run bool would be four characters of noise on the line
// somebody is scanning.
func kind(v Value) string {
	if isBool(v) {
		return ""
	}
	if k, ok := v.(kinded); ok {
		return k.Kind()
	}
	return "value"
}

// argShape is how an argument appears in the usage line: angle brackets for one
// that has to be there, square brackets around an optional one, and three dots
// on the one that takes what is left.
func argShape(a Arg) string {
	s := "<" + a.Name + ">"
	if a.Rest {
		s += "..."
	}
	if !a.Required {
		s = "[" + s + "]"
	}
	return s
}

// join puts a note after a description, or on its own when there is none.
func join(desc, note string) string {
	if desc == "" {
		return note
	}
	return desc + " " + note
}

// writeRows writes a two column list, indented, with the right column lined up.
//
// A row with nothing in it is a blank line, which is how the command groups are
// separated without a second kind of row to carry it.
func writeRows(b *strings.Builder, rows []row) {
	width := 0
	for _, r := range rows {
		width = max(width, utf8.RuneCountInString(r.left))
	}
	for _, r := range rows {
		if r.left == "" && r.right == "" {
			b.WriteByte('\n')
			continue
		}
		b.WriteString("  ")
		b.WriteString(r.left)
		if r.right != "" {
			b.WriteString(strings.Repeat(" ", width-utf8.RuneCountInString(r.left)))
			b.WriteString("  ")
			b.WriteString(r.right)
		}
		b.WriteByte('\n')
	}
}
