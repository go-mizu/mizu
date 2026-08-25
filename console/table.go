package console

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// A TableOption changes how [IO.Table] renders.
type TableOption func(*tableConfig)

type tableConfig struct {
	right map[int]bool
}

// AlignRight right-aligns the given columns, counting from zero.
//
// Numbers are the reason this exists. A column of counts that is left-aligned
// cannot be compared by eye, which is most of what a column of counts is for.
func AlignRight(columns ...int) TableOption {
	return func(cfg *tableConfig) {
		if cfg.right == nil {
			cfg.right = make(map[int]bool, len(columns))
		}
		for _, c := range columns {
			cfg.right[c] = true
		}
	}
}

// Table writes rows to stdout under headers.
//
// In JSON mode it writes an array of objects instead, one per row, keyed by
// the headers lowercased with spaces turned into underscores. That is what
// makes --json work on a command that prints a list without the command
// building the list twice.
//
// Every row should have one cell per header. A short row is padded with empty
// cells and a long one is cut, because a display bug is not worth ending a
// command over.
//
// No rows means no output, and an empty array in JSON mode. What that means is
// the command's to say, since "no users" and "no users matching that filter"
// are different sentences.
func (c *IO) Table(headers []string, rows [][]string, opts ...TableOption) {
	var cfg tableConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	rows = shape(headers, rows)
	if c.jsonMode {
		c.tableJSON(headers, rows)
		return
	}
	if len(rows) == 0 {
		return
	}

	widths := columnWidths(headers, rows)
	var b strings.Builder
	writeRow(&b, headers, widths, cfg, styleBold, c.colorOut)
	for _, row := range rows {
		writeRow(&b, row, widths, cfg, styleNone, c.colorOut)
	}
	fmt.Fprint(c.out, b.String())
}

// shape makes every row exactly as wide as the headers.
func shape(headers []string, rows [][]string) [][]string {
	out := make([][]string, len(rows))
	for i, row := range rows {
		if len(row) == len(headers) {
			out[i] = row
			continue
		}
		fixed := make([]string, len(headers))
		copy(fixed, row)
		out[i] = fixed
	}
	return out
}

// columnWidths measures each column, in runes.
//
// Runes rather than display cells. A wide character and a combining mark both
// make this wrong, and getting them right means the Unicode width tables, for
// a column that is one space out. If that becomes the thing somebody notices,
// it is worth revisiting then.
func columnWidths(headers []string, rows [][]string) []int {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = utf8.RuneCountInString(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			widths[i] = max(widths[i], utf8.RuneCountInString(cell))
		}
	}
	return widths
}

// writeRow writes one line. The last column is not padded, so a table copied
// out of a terminal has no trailing spaces in it.
func writeRow(b *strings.Builder, cells []string, widths []int, cfg tableConfig, s style, color bool) {
	for i, cell := range cells {
		if i > 0 {
			b.WriteString("  ")
		}
		pad := widths[i] - utf8.RuneCountInString(cell)
		if cfg.right[i] {
			b.WriteString(strings.Repeat(" ", pad))
			b.WriteString(s.wrap(cell, color))
			continue
		}
		b.WriteString(s.wrap(cell, color))
		if i < len(cells)-1 {
			b.WriteString(strings.Repeat(" ", pad))
		}
	}
	b.WriteByte('\n')
}

// tableJSON writes the same rows as an array of objects.
//
// It is written out by hand rather than through a map so that the members come
// out in column order. A map would sort them, and a table whose JSON reorders
// the columns is one more thing to explain.
func (c *IO) tableJSON(headers []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Fprintln(c.out, "[]")
		return
	}

	// The member names are the same on every row, so they are built once. A
	// table of four hundred routes is otherwise four hundred passes over the
	// same five headers.
	keys := make([]string, len(headers))
	for i, h := range headers {
		var k strings.Builder
		appendQuoted(&k, columnKey(h))
		keys[i] = k.String()
	}

	var b strings.Builder
	b.WriteString("[\n")
	for i, row := range rows {
		b.WriteString("  {")
		for j, cell := range row {
			if j > 0 {
				b.WriteString(", ")
			}
			b.WriteString(keys[j])
			b.WriteString(": ")
			appendQuoted(&b, cell)
		}
		b.WriteString("}")
		if i < len(rows)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("]\n")
	fmt.Fprint(c.out, b.String())
}

// columnKey turns a header into a member name. "Last seen" becomes "last_seen",
// which is what somebody writing a jq expression will guess.
func columnKey(header string) string {
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' {
			return '_'
		}
		return r
	}, strings.ToLower(strings.TrimSpace(header)))
}

// appendQuoted writes s to b as a JSON string.
//
// Nearly every cell in a table is a name, a path, a status or a number, with
// nothing in it that JSON has to escape, so the common case is a copy with two
// quotes around it. Anything else goes through the standard library, which is
// the part worth being sure about.
//
// HTML metacharacters are left alone, the same as [IO.JSON]. That is why the
// slow path builds an encoder rather than calling json.Marshal, which escapes
// them and has no way to be asked not to.
func appendQuoted(b *strings.Builder, s string) {
	if plain(s) {
		b.WriteByte('"')
		b.WriteString(s)
		b.WriteByte('"')
		return
	}

	var enc bytes.Buffer
	e := json.NewEncoder(&enc)
	e.SetEscapeHTML(false)
	// Encoding a string has no failure case. Invalid UTF-8 comes back as
	// replacement characters rather than as an error.
	e.Encode(s)
	b.WriteString(strings.TrimSuffix(enc.String(), "\n"))
}

// plain reports whether s can be written between two quotes unchanged.
//
// Printable ASCII other than the quote and the backslash is what JSON takes
// literally. Everything above it is valid in a JSON string too, and is left to
// the encoder anyway, because invalid UTF-8 in there needs replacing and this
// is not the place to be deciding that.
func plain(s string) bool {
	for i := range len(s) {
		if c := s[i]; c < 0x20 || c > 0x7e || c == '"' || c == '\\' {
			return false
		}
	}
	return true
}
