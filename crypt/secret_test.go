package crypt

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestSecret(t *testing.T) {
	s := Secret("t0ps3cr3t-not-a-real-token")

	if got := s.Reveal(); got != "t0ps3cr3t-not-a-real-token" {
		t.Errorf("Reveal gave %q", got)
	}
	if got := s.String(); got != Redacted {
		t.Errorf("a secret prints as %q, want %q", got, Redacted)
	}
	if got := string(s); got != s.Reveal() {
		t.Errorf("a conversion gave %q", got)
	}
}

// TestSecretEmpty is the case worth being careful about. A value that was never
// configured should read as missing, not as present and hidden.
func TestSecretEmpty(t *testing.T) {
	var s Secret

	if got := s.String(); got != "" {
		t.Errorf("an empty secret prints as %q, want nothing", got)
	}
	if got := fmt.Sprintf("%v", s); got != "" {
		t.Errorf("an empty secret formats as %q, want nothing", got)
	}
	if got := mustJSON(t, s); got != `""` {
		t.Errorf("an empty secret marshals as %s", got)
	}
}

// TestSecretHides is the whole reason this is a type rather than a string.
func TestSecretHides(t *testing.T) {
	const value = "hunter2-and-then-some"
	s := Secret(value)

	held := struct {
		User   string
		Passwd Secret
	}{"ada", s}

	printed := []string{
		s.String(),
		fmt.Sprint(s),
		fmt.Sprintf("%v", s),
		fmt.Sprintf("%s", s),
		fmt.Sprintf("%q", s),
		fmt.Sprintf("%x", s),
		fmt.Sprintf("%#v", s),
		fmt.Sprintf("%+v", held),
		fmt.Sprint(&s),
		mustJSON(t, s),
		mustJSON(t, held),
		s.LogValue().String(),
	}
	for _, got := range printed {
		if strings.Contains(got, value) {
			t.Errorf("the secret leaked: %s", got)
		}
		if !strings.Contains(got, Redacted) {
			t.Errorf("nothing was redacted in %s", got)
		}
	}
}

// TestSecretQuoted checks that output holding a redacted value is still
// parseable, since %q on a struct field is how a lot of logging is written.
func TestSecretQuoted(t *testing.T) {
	got := fmt.Sprintf("%q", Secret("x"))
	if want := `"` + Redacted + `"`; got != want {
		t.Errorf("%%q gave %s, want %s", got, want)
	}
}

func TestSecretEqual(t *testing.T) {
	s := Secret("signature")

	if !s.Equal(Secret("signature")) {
		t.Error("a secret does not equal itself")
	}
	if s.Equal(Secret("signatur")) {
		t.Error("a shorter secret is equal")
	}
	if s.Equal(Secret("signaturf")) {
		t.Error("a different secret is equal")
	}
	if !Secret("").Equal("") {
		t.Error("two empty secrets are not equal")
	}
}

func TestSecretLogValue(t *testing.T) {
	s := Secret("token")
	if got, want := s.LogValue().Kind(), slog.KindString; got != want {
		t.Errorf("a secret logs as %v, want %v", got, want)
	}
	if got := s.LogValue().String(); got != Redacted {
		t.Errorf("a secret logs as %q", got)
	}
}

func TestSecretMarshalText(t *testing.T) {
	s := Secret("token")

	b, err := s.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != Redacted {
		t.Errorf("a secret marshals as %q", b)
	}

	// Text goes in as the real thing, which is what a configuration holds.
	var got Secret
	if err := got.UnmarshalText([]byte("token")); err != nil {
		t.Fatal(err)
	}
	if got != s {
		t.Errorf("a secret unmarshalled as %q", got.Reveal())
	}
}

func TestSecretJSONRoundTrip(t *testing.T) {
	type config struct {
		URL string `json:"url"`
		Key Secret `json:"key"`
	}

	var got config
	if err := json.Unmarshal([]byte(`{"url":"https://api","key":"t0ps3cr3t"}`), &got); err != nil {
		t.Fatal(err)
	}
	if got.Key.Reveal() != "t0ps3cr3t" {
		t.Errorf("the key unmarshalled as %q", got.Key.Reveal())
	}

	// Out again it is redacted, which is what makes dumping a configuration
	// safe and reloading that dump a mistake somebody notices.
	if out := mustJSON(t, got); !strings.Contains(out, Redacted) || strings.Contains(out, "t0ps3cr3t") {
		t.Errorf("the configuration marshalled as %s", out)
	}
}
