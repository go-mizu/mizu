package config

import (
	"errors"
	"log/slog"
	"math"
	"math/big"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/toml"
)

// fromText is a value the way every layer except a file has it.
func fromText(s string) Value {
	return Value{Source: Source{From: FromEnv, Name: "TEST"}, Text: s}
}

// fromFile is a value the way a file has it, keeping the type it was written
// with. The body is the right hand side of x = ..., so fromFile("1") is an
// integer and fromFile(`"1"`) is a string.
func fromFile(t *testing.T, body string) Value {
	t.Helper()
	doc, err := toml.Parse("test.toml", []byte("x = "+body+"\n"))
	if err != nil {
		t.Fatalf("parsing %s: %v", body, err)
	}
	return Value{Source: Source{From: FromFile, Name: "test.toml:1:5"}, TOML: doc.Get("x")}
}

// parses checks that a parser reads a value into what it should.
func parses[T comparable](t *testing.T, parse Parse[T], v Value, want T) {
	t.Helper()
	var got T
	if err := parse(&got, v); err != nil {
		t.Fatalf("reading %s: %v", v.Display(), err)
	}
	if got != want {
		t.Errorf("%s read as %v, want %v", v.Display(), got, want)
	}
}

// refuses checks that a parser rejects a value, and that it says why in a way
// that mentions the part of the message asked for.
func refuses[T any](t *testing.T, parse Parse[T], v Value, contains string) {
	t.Helper()
	var got T
	err := parse(&got, v)
	if err == nil {
		t.Fatalf("%s was accepted, want an error", v.Display())
	}
	if !strings.Contains(err.Error(), contains) {
		t.Errorf("error is %q, want it to mention %q", err, contains)
	}
}

func TestString(t *testing.T) {
	type name string

	parses(t, String[string], fromText("hello"), "hello")
	parses(t, String[string], fromText(""), "")
	parses(t, String[string], fromFile(t, `"hello"`), "hello")
	parses(t, String[name], fromText("hello"), name("hello"))

	// A file keeps its types, so a number is a number and not the text of one.
	refuses(t, String[string], fromFile(t, "1"), "want a string, got integer")
	refuses(t, String[string], fromFile(t, "true"), "want a string")
}

func TestBool(t *testing.T) {
	type flag bool

	parses(t, Bool[bool], fromFile(t, "true"), true)
	parses(t, Bool[bool], fromFile(t, "false"), false)
	parses(t, Bool[flag], fromText("1"), flag(true))

	for _, s := range []string{"true", "TRUE", "True", "t", "1"} {
		parses(t, Bool[bool], fromText(s), true)
	}
	for _, s := range []string{"false", "FALSE", "f", "0"} {
		parses(t, Bool[bool], fromText(s), false)
	}

	refuses(t, Bool[bool], fromText("yes"), `want a boolean, got "yes"`)
	refuses(t, Bool[bool], fromText(""), "want a boolean")
	refuses(t, Bool[bool], fromFile(t, `"true"`), "want a boolean, got string")
}

func TestInt(t *testing.T) {
	type port int32

	parses(t, Int[int], fromFile(t, "25"), 25)
	parses(t, Int[int], fromFile(t, "-25"), -25)
	parses(t, Int[int], fromText("25"), 25)
	parses(t, Int[port], fromText("8080"), port(8080))

	// Text goes through strconv, which understands the bases Go does.
	parses(t, Int[int], fromText("0x10"), 16)
	parses(t, Int[int], fromText("0b101"), 5)
	parses(t, Int[int], fromText("1_000"), 1000)

	parses(t, Int[int8], fromFile(t, "127"), 127)
	parses(t, Int[int8], fromFile(t, "-128"), -128)
	parses(t, Int[int64], fromFile(t, "9223372036854775807"), math.MaxInt64)

	refuses(t, Int[int8], fromFile(t, "128"), "128 does not fit in 8 bits")
	refuses(t, Int[int8], fromFile(t, "-129"), "does not fit in 8 bits")
	refuses(t, Int[int16], fromText("40000"), "does not fit in 16 bits")
	refuses(t, Int[int], fromText("nine"), `want an integer, got "nine"`)
	refuses(t, Int[int], fromFile(t, "1.5"), "want an integer, got float")
}

func TestUint(t *testing.T) {
	type count uint

	parses(t, Uint[uint], fromFile(t, "25"), 25)
	parses(t, Uint[uint], fromText("25"), 25)
	parses(t, Uint[count], fromText("0"), count(0))
	parses(t, Uint[uint8], fromFile(t, "255"), 255)
	parses(t, Uint[uint16], fromText("0xffff"), 65535)

	// The largest uint64 is not an int64, so it has to come as text. A file
	// cannot hold it either, because TOML integers are signed.
	parses(t, Uint[uint64], fromText("18446744073709551615"), math.MaxUint64)

	refuses(t, Uint[uint], fromFile(t, "-1"), "want a number that is not negative, got -1")
	refuses(t, Uint[uint], fromText("-1"), "want a number that is not negative")
	refuses(t, Uint[uint8], fromFile(t, "256"), "does not fit in 8 bits")
	refuses(t, Uint[uint8], fromText("256"), "does not fit in 8 bits")
	refuses(t, Uint[uint], fromFile(t, `"25"`), "want an integer, got string")
}

func TestFloat(t *testing.T) {
	type ratio float64

	parses(t, Float[float64], fromFile(t, "1.5"), 1.5)
	parses(t, Float[float64], fromFile(t, "1"), 1) // an integer is a number too
	parses(t, Float[float64], fromText("1.5"), 1.5)
	parses(t, Float[float32], fromText("1.5"), 1.5)
	parses(t, Float[ratio], fromText("0.25"), ratio(0.25))
	parses(t, Float[float64], fromFile(t, "inf"), math.Inf(1))

	refuses(t, Float[float32], fromText("1e40"), "too large a number to hold")
	refuses(t, Float[float64], fromText("1e400"), "too large a number to hold")
	refuses(t, Float[float64], fromText("half"), `want a number, got "half"`)
	refuses(t, Float[float64], fromFile(t, "true"), "want a number, got boolean")
}

func TestDuration(t *testing.T) {
	parses(t, Duration[time.Duration], fromText("30s"), 30*time.Second)
	parses(t, Duration[time.Duration], fromText("2h45m"), 2*time.Hour+45*time.Minute)
	parses(t, Duration[time.Duration], fromFile(t, `"150ms"`), 150*time.Millisecond)

	refuses(t, Duration[time.Duration], fromText("30"), "want a length of time")
	refuses(t, Duration[time.Duration], fromFile(t, "30"), "want a string, got integer")
}

func TestTime(t *testing.T) {
	want := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	var got time.Time
	if err := Time(&got, fromText("2026-01-02T03:04:05Z")); err != nil {
		t.Fatal(err)
	}
	if !got.Equal(want) {
		t.Errorf("text read as %v, want %v", got, want)
	}

	// A file may write it as a date and time of its own, or as a string.
	for _, body := range []string{"2026-01-02T03:04:05Z", `"2026-01-02T03:04:05Z"`} {
		got = time.Time{}
		if err := Time(&got, fromFile(t, body)); err != nil {
			t.Fatal(err)
		}
		if !got.Equal(want) {
			t.Errorf("%s read as %v, want %v", body, got, want)
		}
	}

	// A date on its own is midnight on that date.
	got = time.Time{}
	if err := Time(&got, fromFile(t, "2026-01-02")); err != nil {
		t.Fatal(err)
	}
	if y, m, d := got.Date(); y != 2026 || m != time.January || d != 2 {
		t.Errorf("a bare date read as %v", got)
	}

	refuses(t, Time, fromText("yesterday"), "want a date and time")
	refuses(t, Time, fromFile(t, "1"), "want a date and time, got integer")
}

func TestBytes(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"standard", "aGVsbG8=", "hello"},
		{"standard without padding", "aGVsbG8", "hello"},
		{"with a prefix", "base64:aGVsbG8=", "hello"},
		{"url safe", "-_8=", "\xfb\xff"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []byte
			if err := Bytes(&got, fromText(tt.in)); err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("read as %q, want %q", got, tt.want)
			}
		})
	}

	refuses(t, Bytes, fromText("not base64!"), "want base64")
	refuses(t, Bytes, fromFile(t, "1"), "want a string")
}

func TestNetwork(t *testing.T) {
	parses(t, Addr, fromText("10.0.0.1"), netip.MustParseAddr("10.0.0.1"))
	parses(t, Addr, fromFile(t, `"::1"`), netip.MustParseAddr("::1"))
	refuses(t, Addr, fromText("10.0.0.256"), "want an IP address")
	refuses(t, Addr, fromFile(t, "1"), "want a string, got integer")

	parses(t, Prefix, fromText("10.0.0.0/8"), netip.MustParsePrefix("10.0.0.0/8"))
	refuses(t, Prefix, fromText("10.0.0.0"), "want a network")
	refuses(t, Prefix, fromFile(t, "1"), "want a string, got integer")

	parses(t, AddrPort, fromText("127.0.0.1:8080"), netip.MustParseAddrPort("127.0.0.1:8080"))
	refuses(t, AddrPort, fromText("127.0.0.1"), "want an address and a port")
	refuses(t, AddrPort, fromFile(t, "1"), "want a string, got integer")
}

func TestLevel(t *testing.T) {
	parses(t, Level, fromText("debug"), slog.LevelDebug)
	parses(t, Level, fromText("INFO"), slog.LevelInfo)
	parses(t, Level, fromFile(t, `"warn"`), slog.LevelWarn)
	parses(t, Level, fromText("debug+2"), slog.LevelDebug+2)

	refuses(t, Level, fromText("loud"), "want a level such as debug")
	refuses(t, Level, fromFile(t, "1"), "want a string")
}

func TestText(t *testing.T) {
	// The type argument is worked out from the destination, which is the whole
	// point of the second parameter being tied to the first.
	var got big.Int
	if err := Text(&got, fromText("123456789012345678901234567890")); err != nil {
		t.Fatal(err)
	}
	if got.String() != "123456789012345678901234567890" {
		t.Errorf("read as %v", &got)
	}

	var when time.Time
	if err := Text(&when, fromFile(t, `"2026-01-02T03:04:05Z"`)); err != nil {
		t.Fatal(err)
	}
	if !when.Equal(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Errorf("read as %v", when)
	}

	refuses(t, Text[big.Int, *big.Int], fromText("twelve"), "twelve")
	refuses(t, Text[big.Int, *big.Int], fromFile(t, "1"), "want a string")
}

// mode is a type that reads the whole of whatever it is given, which is what
// Parser is for.
type mode struct {
	name string
	n    int
}

func (m *mode) ParseConfig(v Value) error {
	if v.TOML != nil && v.TOML.Kind == toml.KindTable {
		name := v.TOML.Table.Get("name")
		if name == nil {
			return errors.New("a mode needs a name")
		}
		m.name = name.Str
		if n := v.TOML.Table.Get("n"); n != nil {
			m.n = int(n.Int)
		}
		return nil
	}
	m.name = v.Display()
	return nil
}

func TestConfig(t *testing.T) {
	var got mode
	if err := Config(&got, fromFile(t, `{name = "fast", n = 3}`)); err != nil {
		t.Fatal(err)
	}
	if got.name != "fast" || got.n != 3 {
		t.Errorf("read as %+v, want fast and 3", got)
	}

	got = mode{}
	if err := Config(&got, fromText("slow")); err != nil {
		t.Fatal(err)
	}
	if got.name != "slow" {
		t.Errorf("read as %+v, want slow", got)
	}

	refuses(t, Config[mode, *mode], fromFile(t, "{n = 1}"), "a mode needs a name")
}

func TestSlice(t *testing.T) {
	tests := []struct {
		name string
		v    Value
		want []string
	}{
		{"an array in a file", fromFile(t, `["a", "b"]`), []string{"a", "b"}},
		{"an empty array", fromFile(t, "[]"), []string{}},
		{"text", fromText("a,b"), []string{"a", "b"}},
		{"spaces are trimmed", fromText(" a , b "), []string{"a", "b"}},
		{"quotes hold a comma", fromText(`a,b,"c,d"`), []string{"a", "b", "c,d"}},
		{"one item", fromText("a"), []string{"a"}},
		{"nothing", fromText(""), nil},
		{"only spaces", fromText("   "), nil},
		{"an empty item stays", fromText("a,,b"), []string{"a", "", "b"}},
	}

	parse := Slice(String[string])
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			if err := parse(&got, tt.v); err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("read as %q, want %q", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("item %d is %q, want %q", i+1, got[i], tt.want[i])
				}
			}
		})
	}

	// An element parser that is not generic needs no type argument.
	var nets []netip.Prefix
	if err := Slice(Prefix)(&nets, fromText("10.0.0.0/8, 192.168.0.0/16")); err != nil {
		t.Fatal(err)
	}
	if len(nets) != 2 || nets[0].String() != "10.0.0.0/8" {
		t.Errorf("read as %v", nets)
	}

	// An element that will not read says which one it was.
	refuses(t, Slice(Int[int]), fromText("1,two,3"), "item 2:")
	refuses(t, Slice(Int[int]), fromFile(t, `[1, "two"]`), "item 2: want an integer, got string")
	refuses(t, Slice(String[string]), fromFile(t, `"a"`), "want a list, got string")
}

func TestMap(t *testing.T) {
	var got map[string]string
	v := fromFile(t, `{one = "1", two = "2"}`)
	if err := Map(String[string])(&got, v); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got["one"] != "1" || got["two"] != "2" {
		t.Errorf("read as %v", got)
	}

	var empty map[string]int
	if err := Map(Int[int])(&empty, fromFile(t, "{}")); err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Errorf("read as %v, want nothing in it", empty)
	}

	refuses(t, Map(Int[int]), fromFile(t, `{one = "1"}`), "one: want an integer, got string")
	refuses(t, Map(String[string]), fromFile(t, `"a"`), "want a table, got string")
	refuses(t, Map(String[string]), fromText("one=1"), "has to be written in a configuration file")
}

func TestBits(t *testing.T) {
	tests := []struct {
		name string
		got  int
		want int
	}{
		{"int8", bits(new(int8)), 8},
		{"int16", bits(new(int16)), 16},
		{"int32", bits(new(int32)), 32},
		{"int64", bits(new(int64)), 64},
		{"uint8", bits(new(uint8)), 8},
		{"uint16", bits(new(uint16)), 16},
		{"uint32", bits(new(uint32)), 32},
		{"uint64", bits(new(uint64)), 64},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s is %d bits, want %d", tt.name, tt.got, tt.want)
		}
	}

	// int and uint are whatever the machine says, and the two agree.
	if n := bits(new(int)); n != 32 && n != 64 {
		t.Errorf("int is %d bits", n)
	}
	if bits(new(int)) != bits(new(uint)) {
		t.Errorf("int is %d bits and uint is %d", bits(new(int)), bits(new(uint)))
	}
	if bits(new(uintptr)) != bits(new(uint)) {
		t.Errorf("uintptr is %d bits and uint is %d", bits(new(uintptr)), bits(new(uint)))
	}
}

func TestFits(t *testing.T) {
	tests := []struct {
		n    int64
		size int
		want bool
	}{
		{0, 8, true},
		{127, 8, true},
		{128, 8, false},
		{-128, 8, true},
		{-129, 8, false},
		{math.MaxInt64, 64, true},
		{math.MinInt64, 64, true},
	}
	for _, tt := range tests {
		if got := fits(tt.n, tt.size); got != tt.want {
			t.Errorf("fits(%d, %d) is %v, want %v", tt.n, tt.size, got, tt.want)
		}
	}

	unsigned := []struct {
		n    uint64
		size int
		want bool
	}{
		{0, 8, true},
		{255, 8, true},
		{256, 8, false},
		{math.MaxUint64, 64, true},
	}
	for _, tt := range unsigned {
		if got := fitsUnsigned(tt.n, tt.size); got != tt.want {
			t.Errorf("fitsUnsigned(%d, %d) is %v, want %v", tt.n, tt.size, got, tt.want)
		}
	}
}

func TestDecodeBase64(t *testing.T) {
	// The last encoding tried is the one whose error comes back, and it has to
	// be an error and not a panic on an empty input.
	if _, err := decodeBase64("~~~~"); err == nil {
		t.Error("nonsense was decoded")
	}
}
