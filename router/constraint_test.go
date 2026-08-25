package router

import (
	"strings"
	"testing"
)

func TestBuiltinConstraints(t *testing.T) {
	cases := []struct {
		name string
		yes  []string
		no   []string
	}{{
		name: "int",
		yes:  []string{"0", "7", "-7", "+7", "9223372036854775807", "-9223372036854775808", "007"},
		no:   []string{"", "-", "+", "7.0", "7a", " 7", "9223372036854775808", "99999999999999999999"},
	}, {
		name: "uint",
		yes:  []string{"0", "42", "18446744073709551615", "0000000000000000000", "000000000000000000000042"},
		no:   []string{"", "-1", "+1", "18446744073709551616", "99999999999999999999", "4e3"},
	}, {
		name: "uuid",
		yes: []string{
			"018f6f7d-0000-7000-8000-000000000000",
			"F81D4FAE-7DEC-11D0-A765-00A0C91E6BF6",
		},
		no: []string{
			"",
			"018f6f7d000070008000000000000000",
			"{018f6f7d-0000-7000-8000-000000000000}",
			"018f6f7d-0000-7000-8000-00000000000",
			"018f6f7d-0000-7000-8000-00000000000g",
			"018f6f7d.0000-7000-8000-000000000000",
		},
	}, {
		name: "ulid",
		yes:  []string{"01ARZ3NDEKTSV4RRFFQ69G5FAV", "01arz3ndektsv4rrffq69g5fav", "7ZZZZZZZZZZZZZZZZZZZZZZZZZ"},
		no: []string{
			"",
			"01ARZ3NDEKTSV4RRFFQ69G5FA",
			"81ARZ3NDEKTSV4RRFFQ69G5FAV", // first character above 7 overflows 128 bits
			"01ARZ3NDEKTSV4RRFFQ69G5FAI", // I, L, O and U are not in the alphabet
			"01ARZ3NDEKTSV4RRFFQ69G5FAU",
			"01ARZ3NDEKTSV4RRFFQ69G5FA-",
		},
	}, {
		name: "slug",
		yes:  []string{"a", "hello-world", "go-1-27", "2026"},
		no:   []string{"", "-a", "a-", "a--b", "Hello", "hello_world", "héllo"},
	}, {
		name: "alpha",
		yes:  []string{"a", "Go", "ABC"},
		no:   []string{"", "a1", "a-b", "a_b", "é"},
	}, {
		name: "word",
		yes:  []string{"a", "go_1_27", "A1"},
		no:   []string{"", "a-b", "a.b", "a b"},
	}, {
		name: "date",
		yes:  []string{"2026-08-25", "2024-02-29", "2000-02-29", "0001-01-01", "2026-12-31", "2026-04-30"},
		no: []string{
			"",
			"2026-8-25",
			"2026-02-30",
			"2025-02-29", // not a leap year
			"1900-02-29", // a century that is not a leap year
			"2026-04-31", // April has thirty
			"2026-13-01",
			"2026-00-10",
			"2026-01-00",
			"2026/08/25",
			"2026-08/25",
			"20a6-08-25",
			"2026-0a-25",
			"2026-08-2a",
			"2026-08-25T00:00:00Z",
		},
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			check, ok := builtin[c.name]
			if !ok {
				t.Fatalf("no built-in constraint named %q", c.name)
			}
			for _, s := range c.yes {
				if !check(s) {
					t.Errorf("%s turned down %q", c.name, s)
				}
			}
			for _, s := range c.no {
				if check(s) {
					t.Errorf("%s accepted %q", c.name, s)
				}
			}
		})
	}
}

// Every built-in constraint runs once per request that reaches its wildcard, so
// none of them may allocate, including on the segments they turn down.
func TestConstraintsDoNotAllocate(t *testing.T) {
	segments := []string{
		"", "42", "-42", "99999999999999999999", "018f6f7d-0000-7000-8000-000000000000",
		"01ARZ3NDEKTSV4RRFFQ69G5FAV", "hello-world", "2026-08-25", strings.Repeat("x", 64),
	}
	for name, check := range builtin {
		got := testing.AllocsPerRun(100, func() {
			for _, s := range segments {
				check(s)
			}
		})
		if got != 0 {
			t.Errorf("%s allocated %v times per run", name, got)
		}
	}
}

func TestRegexp(t *testing.T) {
	even := Regexp(`[0-9]*[02468]`)
	for _, s := range []string{"0", "12", "1234"} {
		if !even(s) {
			t.Errorf("even turned down %q", s)
		}
	}
	// Anchored at both ends, so a match in the middle is not a match.
	for _, s := range []string{"", "13", "12a", "a12", "1\n2"} {
		if even(s) {
			t.Errorf("even accepted %q", s)
		}
	}
}

func TestRegexpPanicsOnABadExpression(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Regexp took an expression that does not compile")
		}
	}()
	Regexp("(")
}

func TestConstraintNames(t *testing.T) {
	want := "alpha, date, int, slug, uint, ulid, uuid, word"
	if got := strings.Join(builtin.names(), ", "); got != want {
		t.Errorf("names() = %q, want %q", got, want)
	}
}
