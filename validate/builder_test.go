package validate

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// rules is what failed, as "field:rule", which is short enough to write an
// expectation in and specific enough to catch a rule recording the wrong name.
func rules(v *V) []string {
	var out []string
	for field, rs := range v.Errors().Fields() {
		for _, r := range rs {
			out = append(out, field+":"+r.Rule)
		}
	}
	return out
}

func wantRules(t *testing.T, v *V, want ...string) {
	t.Helper()
	if got := rules(v); !slices.Equal(got, want) {
		t.Errorf("failed %v, want %v", got, want)
	}
}

func TestAPassingChainSaysNothing(t *testing.T) {
	v := New()
	v.Field("email", "user@example.com").Required().Email()
	v.Field("title", "Hello").Required().Min(3).Max(100)

	if err := v.Err(); err != nil {
		t.Fatalf("Err = %v, want nil", err)
	}
	wantRules(t, v)
}

func TestRequired(t *testing.T) {
	v := New()
	v.Field("title", "").Required()
	v.Field("count", 0).Required()
	v.Field("tags", []string{}).Required()
	v.Field("agreed", false).Required()
	v.Field("ok", "x").Required()

	wantRules(t, v, "title:required", "count:required", "tags:required", "agreed:required")
}

// The chain stops at its first failure, so a blank field says it is required
// and does not also say it is not an email address.
func TestAChainStopsAtItsFirstFailure(t *testing.T) {
	v := New()
	v.Field("email", "").Required().Email().Min(5).Max(1).Between(1, 2).Size(9).That(false, "unique")

	wantRules(t, v, "email:required")
}

// Optional is the other way a chain stops, and it records nothing when it does.
func TestOptional(t *testing.T) {
	v := New()
	v.Field("website", "").Optional().URL()
	v.Field("nickname", "").Optional().Min(3)
	v.Field("homepage", "nope").Optional().URL()

	wantRules(t, v, "homepage:url")
}

// Optional and Required after a failure change nothing, since the chain has
// already stopped. Writing them in that order is odd and it still has to have
// one answer.
func TestOptionalAndRequiredAfterAFailureAreQuiet(t *testing.T) {
	v := New()
	v.Field("website", "nope").URL().Optional().URL()
	v.Field("email", "nope").Email().Required()
	v.Field("nickname", "").Optional().Required()

	wantRules(t, v, "website:url", "email:email")
}

func TestSizeRulesReadTheValuesType(t *testing.T) {
	cases := []struct {
		name    string
		run     func(*V)
		want    string
		subject string
	}{
		{"short string", func(v *V) { v.Field("f", "ab").Min(3) }, "f:min", subjString},
		{"long string", func(v *V) { v.Field("f", "abcd").Max(3) }, "f:max", subjString},
		{"small number", func(v *V) { v.Field("f", 2).Min(3) }, "f:min", subjNumeric},
		{"big number", func(v *V) { v.Field("f", 9).Max(3) }, "f:max", subjNumeric},
		{"short list", func(v *V) { v.Field("f", []int{1}).Min(2) }, "f:min", subjArray},
		{"long list", func(v *V) { v.Field("f", []int{1, 2, 3}).Max(2) }, "f:max", subjArray},
		{"brief", func(v *V) { v.Field("f", time.Second).Min(time.Minute) }, "f:min", subjDuration},
		{"wrong size", func(v *V) { v.Field("f", "abcd").Size(3) }, "f:size", subjString},
		{"under", func(v *V) { v.Field("f", 1).Between(2, 4) }, "f:between", subjNumeric},
		{"over", func(v *V) { v.Field("f", 9).Between(2, 4) }, "f:between", subjNumeric},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := New()
			c.run(v)
			wantRules(t, v, c.want)

			for _, rs := range v.Errors().Fields() {
				if rs[0].Subject != c.subject {
					t.Errorf("subject %q, want %q", rs[0].Subject, c.subject)
				}
			}
		})
	}
}

func TestSizeRulesThatPass(t *testing.T) {
	v := New()
	v.Field("a", "abc").Min(3).Max(3).Size(3).Between(1, 5)
	v.Field("b", 4).Between(4, 4)
	v.Field("c", time.Hour).Min(time.Minute).Max(time.Hour)

	wantRules(t, v)
}

// The bound keeps the type it was written with, because that is what the
// sentence prints.
func TestABoundKeepsItsType(t *testing.T) {
	v := New()
	v.Field("session", time.Second).Min(time.Minute)

	if got := v.Errors().First("session"); got != "Session must be at least 1m0s." {
		t.Errorf("First = %q", got)
	}
}

func TestThat(t *testing.T) {
	v := New()
	v.Field("email", "user@example.com").That(true, "unique")
	v.Field("name", "taken").That(false, "unique")
	v.Field("age", 12).That(false, "gte", 18)

	wantRules(t, v, "name:unique", "age:gte")

	for field, rs := range v.Errors().Fields() {
		if field == "age" && !slices.Equal(rs[0].Params, []any{18}) {
			t.Errorf("age params %v, want [18]", rs[0].Params)
		}
	}
}

func TestWhen(t *testing.T) {
	v := New()
	v.When(false, func(v *V) { v.Field("vat", "").Required() })
	v.When(true, func(v *V) { v.Field("company", "").Required() })

	wantRules(t, v, "company:required")
}

// When hands back the builder so a run of conditions is one statement.
func TestWhenChains(t *testing.T) {
	v := New()
	got := v.When(false, func(*V) {}).When(false, func(*V) {})

	if got != v {
		t.Error("When did not hand back the same builder")
	}
}

func TestMsgs(t *testing.T) {
	v := New().Msgs(shout{})
	v.Field("title", "").Required()

	if got := v.Errors().First("title"); got != "TITLE IS REQUIRED." {
		t.Errorf("First = %q", got)
	}
}

type shout struct{}

func (shout) Message(field string, r RuleError) string {
	return strings.ToUpper(English.Message(field, r))
}

// Every format rule has a method, and the method records the name the tag
// spells, because the two have to agree once tags exist.
func TestEveryFormatHasAMethod(t *testing.T) {
	cases := []struct {
		rule  string
		run   func(*Check) *Check
		takes string
		not   string
	}{
		{"email", (*Check).Email, "user@example.com", "user@localhost"},
		{"url", (*Check).URL, "https://example.com", "ftp://example.com"},
		{"uri", (*Check).URI, "urn:isbn:1", "/path"},
		{"hostname", (*Check).Hostname, "example.com", "-nope"},
		{"ip", (*Check).IP, "192.0.2.1", "nope"},
		{"ipv4", (*Check).IPv4, "192.0.2.1", "2001:db8::1"},
		{"ipv6", (*Check).IPv6, "2001:db8::1", "192.0.2.1"},
		{"cidr", (*Check).CIDR, "10.0.0.0/8", "10.0.0.0"},
		{"mac", (*Check).MAC, "00:1b:44:11:3a:b7", "001b44113ab7"},
		{"port", (*Check).Port, "8080", "0"},
		{"uuid", (*Check).UUID, "f47ac10b-58cc-4372-a567-0e02b2c3d479", "nope"},
		{"ulid", (*Check).ULID, "01ARZ3NDEKTSV4RRFFQ69G5FAV", "nope"},
		{"e164", (*Check).E164, "+15551234567", "5551234567"},
	}

	if len(cases) != len(formats) {
		t.Fatalf("%d formats and %d methods, add the new one here", len(formats), len(cases))
	}

	for _, c := range cases {
		t.Run(c.rule, func(t *testing.T) {
			v := New()
			c.run(v.Field("good", c.takes))
			c.run(v.Field("bad", c.not))
			wantRules(t, v, "bad:"+c.rule)
		})
	}
}

// A format rule reads through a pointer, so a *string field checks the string
// and a nil one fails the way an empty one does.
func TestAFormatRuleReadsThroughAPointer(t *testing.T) {
	s := "user@example.com"

	v := New()
	v.Field("a", &s).Email()
	v.Field("b", (*string)(nil)).Email()

	wantRules(t, v, "b:email")
}

// A format rule on something that is not a string is a mistake in the program.
func TestAFormatRuleOnANumberPanics(t *testing.T) {
	defer func() {
		got, _ := recover().(string)
		if !strings.Contains(got, "email") {
			t.Errorf("recovered %v, want a panic naming the rule", got)
		}
	}()

	New().Field("age", 30).Email()
}

// Err is the same error OrNil builds, so a caller reads it the way it reads
// every other one.
func TestErr(t *testing.T) {
	v := New()
	if err := v.Err(); err != nil {
		t.Fatalf("Err with nothing failed = %v", err)
	}

	v.Field("title", "").Required()
	err := v.Err()
	if err == nil {
		t.Fatal("Err with a failure = nil")
	}
	if got := err.Error(); !strings.Contains(got, "One field failed validation.") {
		t.Errorf("Error = %q", got)
	}
}

// Two chains on one field both report, since a field can be wrong in more than
// one way when the rules were written apart.
func TestTwoChainsOnOneField(t *testing.T) {
	v := New()
	v.Field("email", "nope").Email()
	v.Field("email", "nope").Min(10)

	wantRules(t, v, "email:email", "email:min")
}
