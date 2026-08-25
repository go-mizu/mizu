package console

import (
	"net/netip"
	"testing"
	"time"
)

func TestString(t *testing.T) {
	var s string
	v := String(&s)

	if err := v.Set("  spaces kept  "); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if s != "  spaces kept  " {
		t.Errorf("set %q, want the text as it stands", s)
	}
	if v.String() != s {
		t.Errorf("String is %q, want %q", v.String(), s)
	}
}

// TestStringOfANamedType is the reason these take a type parameter. A command
// holding a Level or an Env should not have to hold a string next to it.
func TestStringOfANamedType(t *testing.T) {
	type env string

	var e env
	if err := String(&e).Set("staging"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if e != "staging" {
		t.Errorf("set %q, want staging", e)
	}
}

func TestInt(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want int
	}{
		{"42", 42},
		{"-7", -7},
		{"0x2a", 42},
		{"0b101010", 42},
		{"1_000", 1000},
	} {
		var n int
		if err := Int(&n).Set(tt.in); err != nil {
			t.Errorf("Set(%q): %v", tt.in, err)
			continue
		}
		if n != tt.want {
			t.Errorf("Set(%q) gave %d, want %d", tt.in, n, tt.want)
		}
	}
}

func TestIntRejects(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want string
	}{
		{"nine", `"nine" is not a number`},
		{"", `"" is not a number`},
		{"1.5", `"1.5" is not a number`},
		{"99999999999999999999", "99999999999999999999 is out of range"},
	} {
		var n int
		err := Int(&n).Set(tt.in)
		if err == nil {
			t.Errorf("Set(%q) was accepted as %d", tt.in, n)
			continue
		}
		if err.Error() != tt.want {
			t.Errorf("Set(%q) says %q, want %q", tt.in, err, tt.want)
		}
	}
}

// TestIntDoesNotWrapAround is the bug this would have if the width were left
// to the conversion: 300 into an int8 is 44, and a command that took it would
// do something nobody asked for.
func TestIntDoesNotWrapAround(t *testing.T) {
	var n int8
	err := Int(&n).Set("300")
	if err == nil {
		t.Fatalf("300 went into an int8 as %d", n)
	}
	if want := "300 does not fit"; err.Error() != want {
		t.Errorf("says %q, want %q", err, want)
	}
	if n != 0 {
		t.Errorf("left %d behind, want the value untouched", n)
	}
}

func TestUint(t *testing.T) {
	var n uint16
	if err := Uint(&n).Set("65535"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if n != 65535 {
		t.Errorf("set %d, want 65535", n)
	}
	if err := Uint(&n).Set("65536"); err == nil || err.Error() != "65536 does not fit" {
		t.Errorf("65536 into a uint16 says %v", err)
	}
}

// TestUintSaysNegative because the alternative is telling somebody that -1 is
// not a number, which they will read twice and still not believe.
func TestUintSaysNegative(t *testing.T) {
	var n uint
	err := Uint(&n).Set("-1")
	if err == nil {
		t.Fatal("-1 was accepted")
	}
	if want := "-1 is negative"; err.Error() != want {
		t.Errorf("says %q, want %q", err, want)
	}

	if err := Uint(&n).Set("nine"); err == nil || err.Error() != `"nine" is not a number` {
		t.Errorf(`Set("nine") says %v`, err)
	}
}

func TestFloat(t *testing.T) {
	var f float64
	if err := Float(&f).Set("1.5"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if f != 1.5 {
		t.Errorf("set %v, want 1.5", f)
	}

	var small float32
	if err := Float(&small).Set("1e39"); err == nil {
		t.Errorf("1e39 went into a float32 as %v", small)
	}
	if err := Float(&small).Set("Inf"); err != nil {
		t.Errorf("Inf was rejected: %v", err)
	}
	if err := Float(&f).Set("half"); err == nil || err.Error() != `"half" is not a number` {
		t.Errorf(`Set("half") says %v`, err)
	}
}

// half is a type that reads itself from text and has no way to write itself
// back, which is most of them.
type half struct{ n int }

func (h *half) UnmarshalText(text []byte) error {
	n, err := parseInt(string(text))
	h.n = n / 2
	return err
}

func TestTextWithNothingToShow(t *testing.T) {
	var h half
	v := Text(&h)

	if err := v.Set("10"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if h.n != 5 {
		t.Errorf("got %d, want 5", h.n)
	}
	if want := "&{5}"; v.String() != want {
		t.Errorf("String is %q, want %q, which is the best that can be done for a type with no text form", v.String(), want)
	}
}

func TestBool(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want bool
	}{
		{"true", true},
		{"1", true},
		{"T", true},
		{"false", false},
		{"0", false},
	} {
		b := !tt.want
		if err := Bool(&b).Set(tt.in); err != nil {
			t.Errorf("Set(%q): %v", tt.in, err)
			continue
		}
		if b != tt.want {
			t.Errorf("Set(%q) gave %v, want %v", tt.in, b, tt.want)
		}
	}

	var b bool
	v := Bool(&b)
	if err := v.Set("yes"); err == nil || err.Error() != `"yes" is not true or false` {
		t.Errorf(`Set("yes") says %v`, err)
	}
	if v.String() != "false" {
		t.Errorf("String is %q, want false", v.String())
	}
	if !isBool(v) {
		t.Error("a bool value does not say it takes no argument")
	}
}

func TestCount(t *testing.T) {
	var n int
	v := Count(&n)

	c, ok := v.(counter)
	if !ok {
		t.Fatal("a count value does not count")
	}
	c.Count()
	c.Count()
	if n != 2 {
		t.Errorf("counted %d, want 2", n)
	}

	if err := v.Set("5"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if n != 5 {
		t.Errorf("set %d, want an explicit value to replace the count", n)
	}
	if v.String() != "5" {
		t.Errorf("String is %q, want 5", v.String())
	}
	if err := v.Set("lots"); err == nil {
		t.Error("lots was accepted")
	}
}

func TestDuration(t *testing.T) {
	var d time.Duration
	if err := Duration(&d).Set("1h30m"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if d != 90*time.Minute {
		t.Errorf("set %v, want 1h30m", d)
	}

	err := Duration(&d).Set("30")
	if err == nil {
		t.Fatal("a bare number was taken as a duration")
	}
	if want := `"30" is not a length of time, try 30s or 5m`; err.Error() != want {
		t.Errorf("says %q, want %q", err, want)
	}
}

func TestTime(t *testing.T) {
	var when time.Time
	v := Time(&when)

	if err := v.Set("2026-08-23T10:00:00Z"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if want := time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC); !when.Equal(want) {
		t.Errorf("set %v, want %v", when, want)
	}

	// A plain date, which is what a person types.
	if err := v.Set("2026-08-23"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if want := time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC); !when.Equal(want) {
		t.Errorf("set %v, want %v", when, want)
	}

	err := v.Set("yesterday")
	if err == nil {
		t.Fatal("yesterday was accepted")
	}
	if want := `"yesterday" is not a time, try ` + time.RFC3339; err.Error() != want {
		t.Errorf("says %q, want %q", err, want)
	}
}

func TestTimeWithALayout(t *testing.T) {
	var when time.Time
	if err := Time(&when, "2006/01/02").Set("2026/08/23"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if when.Year() != 2026 || when.Day() != 23 {
		t.Errorf("set %v", when)
	}
}

func TestText(t *testing.T) {
	var addr netip.Addr
	v := Text(&addr)

	if err := v.Set("192.0.2.1"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if addr.String() != "192.0.2.1" {
		t.Errorf("set %v", addr)
	}
	if v.String() != "192.0.2.1" {
		t.Errorf("String is %q", v.String())
	}
	if err := v.Set("not an address"); err == nil {
		t.Error("not an address was accepted")
	}
}

func TestSlice(t *testing.T) {
	var days []int
	v := Slice(&days, parseInt, ",")

	if err := v.Set("1,2"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := v.Set("3"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if len(days) != 3 || days[0] != 1 || days[2] != 3 {
		t.Errorf("collected %v, want 1 2 3", days)
	}
	if v.String() != "1,2,3" {
		t.Errorf("String is %q, want 1,2,3", v.String())
	}
	if err := v.Set("4,five"); err == nil {
		t.Error("five was accepted")
	}
}

// TestStringsWithoutASeparator is what a flag taking paths or SQL wants, since
// a comma is a character those are allowed to contain.
func TestStringsWithoutASeparator(t *testing.T) {
	var paths []string
	v := Strings(&paths, "")

	if err := v.Set("a,b"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if len(paths) != 1 || paths[0] != "a,b" {
		t.Errorf("collected %v, want the one string", paths)
	}
}

func TestKeyValues(t *testing.T) {
	var labels map[string]string
	v := KeyValues(&labels)

	for _, pair := range []string{"env=staging", "team=platform", "env=production"} {
		if err := v.Set(pair); err != nil {
			t.Fatalf("Set(%q): %v", pair, err)
		}
	}
	if len(labels) != 2 || labels["env"] != "production" {
		t.Errorf("collected %v, want the last env to win", labels)
	}
	if want := "env=production,team=platform"; v.String() != want {
		t.Errorf("String is %q, want %q, sorted so that it is the same every run", v.String(), want)
	}

	for _, bad := range []string{"env", "=staging"} {
		if err := v.Set(bad); err == nil {
			t.Errorf("Set(%q) was accepted", bad)
		}
	}
}

func TestKeyValuesWithAnEmptyValue(t *testing.T) {
	var labels map[string]string
	if err := KeyValues(&labels).Set("team="); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if v, ok := labels["team"]; !ok || v != "" {
		t.Errorf("collected %v, want an empty value to be a value", labels)
	}
}

func TestEnum(t *testing.T) {
	var mode string
	v := Enum(&mode, "auto", "always", "never")

	if err := v.Set("always"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if mode != "always" {
		t.Errorf("set %q", mode)
	}

	err := v.Set("sometimes")
	if err == nil {
		t.Fatal("sometimes was accepted")
	}
	if want := `"sometimes" is not one of auto, always, never`; err.Error() != want {
		t.Errorf("says %q, want %q", err, want)
	}
}

// parseInt is what a caller writes when it wants a slice of numbers, and is
// here so that two tests can share it.
func parseInt(s string) (int, error) {
	var n int
	err := Int(&n).Set(s)
	return n, err
}

// TestVar is the escape hatch: a type the package has never heard of, with the
// function that parses it.
func TestVar(t *testing.T) {
	var day time.Weekday
	v := Var(&day, func(s string) (time.Weekday, error) {
		var when time.Time
		if err := Time(&when).Set(s); err != nil {
			return 0, err
		}
		return when.Weekday(), nil
	})

	if err := v.Set("2026-08-23"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	want := time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC).Weekday()
	if day != want {
		t.Errorf("got %v, want %v", day, want)
	}
	if err := v.Set("yesterday"); err == nil {
		t.Error("yesterday was accepted")
	}
}

// TestStringOfANilPointer is the help path asking a value that nothing has been
// declared for yet. It should say nothing rather than panic.
func TestStringOfANilPointer(t *testing.T) {
	for name, v := range map[string]Value{
		"var":       value[int]{},
		"bool":      boolValue[bool]{},
		"count":     countValue{},
		"slice":     sliceValue[string]{},
		"keyvalues": mapValue{},
	} {
		if got := v.String(); got != "" {
			t.Errorf("%s says %q, want nothing", name, got)
		}
	}
}
