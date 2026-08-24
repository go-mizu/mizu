package toml

import (
	"fmt"
	"slices"
	"testing"
)

func TestKindString(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{KindString, "string"},
		{KindInt, "integer"},
		{KindFloat, "float"},
		{KindBool, "boolean"},
		{KindOffsetDateTime, "offset date-time"},
		{KindLocalDateTime, "local date-time"},
		{KindLocalDate, "local date"},
		{KindLocalTime, "local time"},
		{KindArray, "array"},
		{KindTable, "table"},
		{KindInvalid, "invalid"},
		{Kind(99), "invalid"},
	}
	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Errorf("Kind(%d) is %q, want %q", int(tt.kind), got, tt.want)
		}
	}
}

func TestPositionString(t *testing.T) {
	tests := []struct {
		pos  Position
		want string
	}{
		{Position{File: "config.toml", Line: 12, Col: 3}, "config.toml:12:3"},
		{Position{Line: 1, Col: 1}, "1:1"},
	}
	for _, tt := range tests {
		if got := tt.pos.String(); got != tt.want {
			t.Errorf("%#v is %q, want %q", tt.pos, got, tt.want)
		}
	}
}

func TestValueErrorf(t *testing.T) {
	tab := parse(t, "[db]\npool = \"ten\"\n")
	v := tab.Lookup("db", "pool")

	err := v.Errorf("db.pool: want an integer, got a %s", v.Kind)
	want := "test.toml:2:8: db.pool: want an integer, got a string"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err, want)
	}
}

func TestTableLookup(t *testing.T) {
	tab := parse(t, "[a.b]\nc = 1\nd = \"two\"\n")

	if v := tab.Lookup("a", "b", "c"); v == nil || v.Int != 1 {
		t.Errorf("a.b.c is %v, want 1", v)
	}
	for _, path := range [][]string{
		{"a", "b", "gone"},
		{"gone"},
		{"a", "b", "c", "deeper"}, // c is an integer, so there is no deeper
		{"a", "b", "d", "deeper"},
		{}, // no path names no value
	} {
		if v := tab.Lookup(path...); v != nil {
			t.Errorf("Lookup(%q) is %v, want nil", path, v)
		}
	}
}

func TestNilTable(t *testing.T) {
	var tab *Table
	if tab.Len() != 0 {
		t.Errorf("Len is %d, want 0", tab.Len())
	}
	if tab.Keys() != nil {
		t.Errorf("Keys is %v, want nil", tab.Keys())
	}
	if tab.Get("a") != nil {
		t.Errorf("Get is not nil")
	}
	if tab.Lookup("a", "b") != nil {
		t.Errorf("Lookup is not nil")
	}
	for range tab.All() {
		t.Errorf("All gave a key")
	}
}

func TestTableAllStops(t *testing.T) {
	tab := parse(t, "a = 1\nb = 2\nc = 3\n")
	var seen []string
	for k := range tab.All() {
		seen = append(seen, k)
		if len(seen) == 2 {
			break
		}
	}
	if want := []string{"a", "b"}; !slices.Equal(seen, want) {
		t.Errorf("got %v, want %v", seen, want)
	}
}

func TestTableKeysIsACopy(t *testing.T) {
	tab := parse(t, "a = 1\nb = 2\n")
	keys := tab.Keys()
	keys[0] = "changed"
	if got := tab.Keys()[0]; got != "a" {
		t.Errorf("the first key is now %q, so Keys handed out the table's own slice", got)
	}
}

func TestQuoteKey(t *testing.T) {
	tests := []struct{ in, want string }{
		{"name", "name"},
		{"bare_key", "bare_key"},
		{"bare-key", "bare-key"},
		{"1234", "1234"},
		{"", `""`},
		{"127.0.0.1", `"127.0.0.1"`},
		{"two words", `"two words"`},
		{"水", `"水"`},
	}
	for _, tt := range tests {
		if got := quoteKey(tt.in); got != tt.want {
			t.Errorf("quoteKey(%q) is %s, want %s", tt.in, got, tt.want)
		}
	}
}

func TestErrorMessage(t *testing.T) {
	err := &Error{Pos: Position{File: "a.toml", Line: 2, Col: 5}, Msg: "no"}
	if got, want := err.Error(), "a.toml:2:5: no"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func ExampleParse() {
	doc, err := Parse("config.toml", []byte(`
title = "mizu"

[server]
host = "localhost"
port = 8080
`))
	if err != nil {
		fmt.Println(err)
		return
	}
	for key, v := range doc.Lookup("server").Table.All() {
		fmt.Printf("%s = %v (%s at %s)\n", key, value(v), v.Kind, v.Pos)
	}
	// Output:
	// host = localhost (string at config.toml:5:8)
	// port = 8080 (integer at config.toml:6:8)
}

func ExampleValue_Errorf() {
	doc, err := Parse("config.toml", []byte("[server]\nport = \"8080\"\n"))
	if err != nil {
		fmt.Println(err)
		return
	}
	if v := doc.Lookup("server", "port"); v.Kind != KindInt {
		fmt.Println(v.Errorf("server.port: want an integer, got a %s", v.Kind))
	}
	// Output:
	// config.toml:2:8: server.port: want an integer, got a string
}
