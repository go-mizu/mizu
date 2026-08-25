package console

import (
	"strings"
	"testing"
)

var (
	headers = []string{"Name", "Email", "Posts"}
	rows    = [][]string{
		{"Ada Lovelace", "ada@example.com", "17"},
		{"Bo", "bo@example.com", "3"},
	}
)

func TestTable(t *testing.T) {
	s := newStreams(t, Options{})

	s.io.Table(headers, rows)

	want := strings.Join([]string{
		"Name          Email            Posts",
		"Ada Lovelace  ada@example.com  17",
		"Bo            bo@example.com   3",
		"",
	}, "\n")
	if got := s.out.String(); got != want {
		t.Errorf("Table wrote\n%s\nwant\n%s", got, want)
	}
}

// TestTableHasNoTrailingSpaces is what makes a table copyable. A line padded
// to the width of its last column brings invisible spaces with it.
func TestTableHasNoTrailingSpaces(t *testing.T) {
	s := newStreams(t, Options{})

	s.io.Table([]string{"Name", "Note"}, [][]string{{"Ada", "short"}, {"Bo", "a much longer note"}})

	for _, line := range strings.Split(strings.TrimSuffix(s.out.String(), "\n"), "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("the line %q ends in a space", line)
		}
	}
}

func TestTableAlignsRight(t *testing.T) {
	s := newStreams(t, Options{})

	s.io.Table(headers, rows, AlignRight(2))

	want := strings.Join([]string{
		"Name          Email            Posts",
		"Ada Lovelace  ada@example.com     17",
		"Bo            bo@example.com       3",
		"",
	}, "\n")
	if got := s.out.String(); got != want {
		t.Errorf("Table wrote\n%s\nwant\n%s", got, want)
	}
}

func TestTableCountsRunesNotBytes(t *testing.T) {
	s := newStreams(t, Options{})

	s.io.Table([]string{"Name", "Note"}, [][]string{{"Amélie", "yes"}, {"Bo", "no"}})

	want := strings.Join([]string{
		"Name    Note",
		"Amélie  yes",
		"Bo      no",
		"",
	}, "\n")
	if got := s.out.String(); got != want {
		t.Errorf("Table wrote\n%s\nwant\n%s", got, want)
	}
}

// TestTableShapesTheRows covers a row that does not match the headers. It is a
// bug in the command, and printing four columns of nine is a worse way to
// report it than the table coming out straight.
func TestTableShapesTheRows(t *testing.T) {
	s := newStreams(t, Options{})

	s.io.Table([]string{"A", "B"}, [][]string{{"1"}, {"1", "2", "3"}})

	want := "A  B\n1  \n1  2\n"
	if got := s.out.String(); got != want {
		t.Errorf("Table wrote %q, want %q", got, want)
	}
}

func TestTableWithNoRowsWritesNothing(t *testing.T) {
	s := newStreams(t, Options{})

	s.io.Table(headers, nil)

	if got := s.out.String(); got != "" {
		t.Errorf("an empty table wrote %q", got)
	}
}

// TestTableIsJSONInJSONMode is what makes --json free for a command that
// prints a list. The list is built once and rendered twice.
func TestTableIsJSONInJSONMode(t *testing.T) {
	s := newStreams(t, Options{JSON: true})

	s.io.Table(headers, rows)

	want := strings.Join([]string{
		`[`,
		`  {"name": "Ada Lovelace", "email": "ada@example.com", "posts": "17"},`,
		`  {"name": "Bo", "email": "bo@example.com", "posts": "3"}`,
		`]`,
		``,
	}, "\n")
	if got := s.out.String(); got != want {
		t.Errorf("Table wrote\n%s\nwant\n%s", got, want)
	}
}

// TestEmptyTableIsAnEmptyArray is the case a jq expression cares about. No
// output at all is a parse error at the other end.
func TestEmptyTableIsAnEmptyArray(t *testing.T) {
	s := newStreams(t, Options{JSON: true})

	s.io.Table(headers, nil)

	if got := s.out.String(); got != "[]\n" {
		t.Errorf("an empty table wrote %q, want an empty array", got)
	}
}

func TestTableJSONEscapes(t *testing.T) {
	s := newStreams(t, Options{JSON: true})

	s.io.Table([]string{"Note"}, [][]string{{`he said "no"` + "\n"}})

	if got := s.out.String(); !strings.Contains(got, `"he said \"no\"\n"`) {
		t.Errorf("Table wrote %q, want the quotes and the newline escaped", got)
	}
}

func TestColumnKey(t *testing.T) {
	tests := map[string]string{
		"Name":        "name",
		"Last seen":   "last_seen",
		"Exit-code":   "exit_code",
		"  Padded  ":  "padded",
		"ALREADYLOUD": "alreadyloud",
	}
	for in, want := range tests {
		if got := columnKey(in); got != want {
			t.Errorf("columnKey(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestTableHeadersAreBold is the one piece of decoration the table has, and it
// is the one that survives being copied into a message, because it disappears
// when colour is off.
func TestTableHeadersAreBold(t *testing.T) {
	s := newStreams(t, Options{Color: ColorAlways})

	s.io.Table([]string{"Name"}, [][]string{{"Ada"}})

	if got := s.out.String(); got != "\x1b[1mName\x1b[0m\nAda\n" {
		t.Errorf("Table wrote %q", got)
	}
}

func TestTableIsPlainWithColourOff(t *testing.T) {
	s := newStreams(t, Options{Color: ColorNever})

	s.io.Table([]string{"Name"}, [][]string{{"Ada"}})

	if got := s.out.String(); strings.Contains(got, "\x1b") {
		t.Errorf("Table wrote %q, want no escapes", got)
	}
}
