package toml

import (
	"math"
	"testing"
	"time"
)

func TestParseStrings(t *testing.T) {
	tests := []struct{ doc, want string }{
		{`a = "hello"`, "hello"},
		{`a = ""`, ""},
		{`a = "he said \"no\""`, `he said "no"`},
		{`a = "c:\\temp"`, `c:\temp`},
		{`a = "\b\t\n\f\r"`, "\b\t\n\f\r"},
		{`a = "\u00e9"`, "é"},
		{`a = "\U0001F600"`, "\U0001F600"},
		{`a = "mizu 水"`, "mizu 水"},
		{`a = 'c:\temp'`, `c:\temp`},
		{`a = 'he said "no"'`, `he said "no"`},
		{`a = ''`, ""},
		{`a = """one"""`, "one"},
		{"a = \"\"\"\none\n\"\"\"", "one\n"},
		{"a = \"\"\"\r\none\r\n\"\"\"", "one\r\n"},
		{`a = """he said "no""""`, `he said "no"`},
		{`a = """two quotes: "" done"""`, `two quotes: "" done`},
		{"a = \"\"\"one \\\n     two\"\"\"", "one two"},
		{"a = \"\"\"one \\\r\n     two\"\"\"", "one two"},
		{"a = \"\"\"one\\\n\\\n  two\"\"\"", "onetwo"},
		{`a = '''one'''`, "one"},
		{"a = '''\none\n'''", "one\n"},
		{`a = '''it's fine'''`, "it's fine"},
		{`a = '''quote: '' done'''`, "quote: '' done"},
		{`a = '''no escapes \n'''`, `no escapes \n`},
		{"a = \"\"\"\ntabs\there\n\"\"\"", "tabs\there\n"},
	}
	for _, tt := range tests {
		check(t, parse(t, tt.doc), []string{"a"}, tt.want)
	}
}

func TestParseIntegers(t *testing.T) {
	tests := []struct {
		doc  string
		want int64
	}{
		{"a = 0", 0},
		{"a = 7", 7},
		{"a = +7", 7},
		{"a = -7", -7},
		{"a = -0", 0},
		{"a = 1_000_000", 1000000},
		{"a = 9223372036854775807", math.MaxInt64},
		{"a = -9223372036854775808", math.MinInt64},
		{"a = 0xdead_beef", 0xdeadbeef},
		{"a = 0xDEADBEEF", 0xdeadbeef},
		{"a = 0o755", 0o755},
		{"a = 0b1010_1010", 0b10101010},
		{"a = 0x0", 0},
	}
	for _, tt := range tests {
		check(t, parse(t, tt.doc), []string{"a"}, tt.want)
	}
}

func TestParseFloats(t *testing.T) {
	tests := []struct {
		doc  string
		want float64
	}{
		{"a = 1.0", 1},
		{"a = 3.1415", 3.1415},
		{"a = -0.01", -0.01},
		{"a = 5e+22", 5e22},
		{"a = 1e06", 1e6},
		{"a = -2E-2", -2e-2},
		{"a = 6.626e-34", 6.626e-34},
		{"a = 224_617.445_991_228", 224617.445991228},
		{"a = 0.0", 0},
		{"a = inf", math.Inf(1)},
		{"a = +inf", math.Inf(1)},
		{"a = -inf", math.Inf(-1)},
		{"a = nan", math.NaN()},
		{"a = +nan", math.NaN()},
		{"a = -nan", math.NaN()},
	}
	for _, tt := range tests {
		check(t, parse(t, tt.doc), []string{"a"}, tt.want)
	}
}

func TestParseBooleans(t *testing.T) {
	check(t, parse(t, "a = true"), []string{"a"}, true)
	check(t, parse(t, "a = false"), []string{"a"}, false)
}

func TestParseDatesAndTimes(t *testing.T) {
	tests := []struct {
		doc  string
		kind Kind
		want time.Time
	}{
		{"a = 1979-05-27T07:32:00Z", KindOffsetDateTime, date("1979-05-27T07:32:00Z")},
		{"a = 1979-05-27t07:32:00z", KindOffsetDateTime, date("1979-05-27T07:32:00Z")},
		{"a = 1979-05-27 07:32:00Z", KindOffsetDateTime, date("1979-05-27T07:32:00Z")},
		{"a = 1979-05-27T00:32:00-07:00", KindOffsetDateTime, date("1979-05-27T00:32:00-07:00")},
		{"a = 1979-05-27T00:32:00.999999-07:00", KindOffsetDateTime, date("1979-05-27T00:32:00.999999-07:00")},
		{"a = 1979-05-27T07:32:00", KindLocalDateTime, date("1979-05-27T07:32:00Z")},
		{"a = 1979-05-27 07:32:00.999", KindLocalDateTime, date("1979-05-27T07:32:00.999Z")},
		{"a = 1979-05-27", KindLocalDate, date("1979-05-27T00:00:00Z")},
		{"a = 07:32:00", KindLocalTime, date("0000-01-01T07:32:00Z")},
		{"a = 00:32:00.999999", KindLocalTime, date("0000-01-01T00:32:00.999999Z")},
	}
	for _, tt := range tests {
		tab := parse(t, tt.doc)
		v := tab.Get("a")
		if v.Kind != tt.kind {
			t.Errorf("%s is a %s, want a %s", tt.doc, v.Kind, tt.kind)
			continue
		}
		check(t, tab, []string{"a"}, tt.want)
	}
}
