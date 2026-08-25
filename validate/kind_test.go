package validate

import (
	"testing"
	"time"
)

type word string

func TestIsEmpty(t *testing.T) {
	s := "x"
	blank := ""

	cases := []struct {
		value any
		want  bool
	}{
		{nil, true},
		{"", true},
		{"x", false},
		{word(""), true},
		{word("x"), false},
		{0, true},
		{1, false},
		{0.0, true},
		{1.5, false},
		{false, true},
		{true, false},
		{time.Time{}, true},
		{time.Duration(0), true},
		{time.Second, false},
		{(*string)(nil), true},
		{&blank, true},
		{&s, false},
		{[]string(nil), true},
		{[]string{}, true},
		{[]string{"a"}, false},
		{map[string]int{}, true},
		{map[string]int{"a": 1}, false},
		{[0]int{}, true},
		{[1]int{0}, false},
		{struct{ N int }{}, true},
		{struct{ N int }{1}, false},
	}

	for _, c := range cases {
		if got := isEmpty(c.value); got != c.want {
			t.Errorf("isEmpty(%#v) = %v, want %v", c.value, got, c.want)
		}
	}
}

func TestSizeOf(t *testing.T) {
	n := 7

	cases := []struct {
		value   any
		subject string
		size    float64
	}{
		{"hello", subjString, 5},
		{"héllo", subjString, 5},
		{"a👍b", subjString, 3},
		{word("hey"), subjString, 3},
		{7, subjNumeric, 7},
		{int8(-3), subjNumeric, -3},
		{uint(9), subjNumeric, 9},
		{uint8(9), subjNumeric, 9},
		{2.5, subjNumeric, 2.5},
		{float32(1.5), subjNumeric, 1.5},
		{&n, subjNumeric, 7},
		{[]string{"a", "b"}, subjArray, 2},
		{map[string]int{"a": 1}, subjArray, 1},
		{[3]int{}, subjArray, 3},
		{time.Hour, subjDuration, float64(time.Hour)},
	}

	for _, c := range cases {
		subject, size := sizeOf(c.value)
		if subject != c.subject || size != c.size {
			t.Errorf("sizeOf(%#v) = %q, %v, want %q, %v", c.value, subject, size, c.subject, c.size)
		}
	}
}

// A size rule on something with no size is a mistake in the program, so it says
// so where the mistake is rather than telling somebody their input was wrong.
func TestSizeOfPanicsOnSomethingWithNoSize(t *testing.T) {
	for _, value := range []any{nil, struct{}{}, true, (*int)(nil)} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("sizeOf(%#v) did not panic", value)
				}
			}()
			sizeOf(value)
		}()
	}
}

func TestNumber(t *testing.T) {
	n := 3

	cases := []struct {
		bound any
		want  float64
	}{
		{3, 3},
		{int64(3), 3},
		{uint(3), 3},
		{uint16(3), 3},
		{2.5, 2.5},
		{float32(2.5), 2.5},
		{time.Hour, float64(time.Hour)},
		{&n, 3},
	}

	for _, c := range cases {
		if got := number(c.bound); got != c.want {
			t.Errorf("number(%#v) = %v, want %v", c.bound, got, c.want)
		}
	}
}

func TestNumberPanicsOnABoundThatIsNotANumber(t *testing.T) {
	for _, bound := range []any{nil, "three", []int{3}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("number(%#v) did not panic", bound)
				}
			}()
			number(bound)
		}()
	}
}

func TestText(t *testing.T) {
	s := "hello"

	cases := []struct {
		value  any
		want   string
		isText bool
	}{
		{"hello", "hello", true},
		{word("hello"), "hello", true},
		{&s, "hello", true},
		{(*string)(nil), "", true},
		{nil, "", true},
		{7, "", false},
		{[]string{"hello"}, "", false},
	}

	for _, c := range cases {
		got, isText := text(c.value)
		if got != c.want || isText != c.isText {
			t.Errorf("text(%#v) = %q, %v, want %q, %v", c.value, got, isText, c.want, c.isText)
		}
	}
}

// A pointer to a pointer is still the value at the end of it, and a nil
// anywhere along the way is nothing.
func TestIndirectFollowsThePointersAllTheWayDown(t *testing.T) {
	s := "x"
	p := &s
	pp := &p

	if got := indirect(pp); got != "x" {
		t.Errorf("indirect(**string) = %#v, want %q", got, "x")
	}
	if got := indirect((**string)(nil)); got != nil {
		t.Errorf("indirect(nil **string) = %#v, want nil", got)
	}
	if got := indirect(7); got != 7 {
		t.Errorf("indirect(7) = %#v, want 7", got)
	}
}
