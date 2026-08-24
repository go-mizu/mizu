package crypt

import (
	"strings"
	"testing"
)

// FuzzParseKey holds the one rule the parser has to keep: whatever it accepts,
// it accepts as the key that was written, and whatever it rejects, it rejects
// without a key and without the key in the error.
func FuzzParseKey(f *testing.F) {
	f.Add("")
	f.Add("mizu1:")
	f.Add(GenerateKey().Reveal())
	f.Add(GenerateKey().Reveal() + "\n")
	f.Add("mizu1:[redacted]")
	f.Add("mizu2:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")

	f.Fuzz(func(t *testing.T, s string) {
		key, err := ParseKey(s)
		if err != nil {
			if !key.IsZero() {
				t.Fatalf("%q failed with a key", s)
			}
			if body, ok := strings.CutPrefix(strings.TrimSpace(s), keyPrefix); ok && len(body) > 8 {
				if strings.Contains(err.Error(), body) {
					t.Fatalf("the error holds the text it rejected: %v", err)
				}
			}
			return
		}

		if got := key.Reveal(); got != strings.TrimSpace(s) {
			t.Fatalf("%q parsed and came back as %q", s, got)
		}

		again, err := ParseKey(key.Reveal())
		if err != nil {
			t.Fatalf("a key this package wrote does not parse: %v", err)
		}
		if !again.Equal(key) {
			t.Fatalf("%q did not survive a second round trip", s)
		}
		if len(key.ID()) != 16 {
			t.Fatalf("%q parsed to a key with the id %q", s, key.ID())
		}
	})
}

// FuzzSecret checks the property the type exists for: nothing printed holds the
// value, whatever the value is.
func FuzzSecret(f *testing.F) {
	f.Add("")
	f.Add("hunter2")
	f.Add("[redacted]")
	f.Add("a b\tc\n")
	f.Add("\x00\xff")

	f.Fuzz(func(t *testing.T, s string) {
		secret := Secret(s)

		if got := secret.Reveal(); got != s {
			t.Fatalf("Reveal gave %q, want %q", got, s)
		}
		if !secret.Equal(Secret(s)) {
			t.Fatal("a secret does not equal itself")
		}

		text, err := secret.MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		for _, printed := range []string{secret.String(), string(text), secret.LogValue().String()} {
			// A value that is part of the mask, such as a single letter or the
			// mask itself, survives printing and gives nothing away. Everything
			// else has to be gone.
			if s != "" && !strings.Contains(Redacted, s) && strings.Contains(printed, s) {
				t.Fatalf("the secret %q leaked as %q", s, printed)
			}
			if s != "" && printed != Redacted {
				t.Fatalf("a secret printed as %q", printed)
			}
			if s == "" && printed != "" {
				t.Fatalf("an empty secret printed as %q", printed)
			}
		}
	})
}

// FuzzPick checks that a draw is the length that was asked for and holds
// nothing that is not in the alphabet, for any alphabet.
func FuzzPick(f *testing.F) {
	f.Add(0, "a")
	f.Add(1, Alphanumeric)
	f.Add(20, Digits10)
	f.Add(7, "\x00\xff")

	f.Fuzz(func(t *testing.T, n int, alphabet string) {
		if n < 0 || n > 1000 || len(alphabet) == 0 || len(alphabet) > 256 {
			t.Skip()
		}

		got := Pick(n, alphabet)
		if len(got) != n {
			t.Fatalf("Pick(%d, %q) is %d characters", n, alphabet, len(got))
		}
		for i := range len(got) {
			if !strings.Contains(alphabet, got[i:i+1]) {
				t.Fatalf("Pick(%d, %q) drew %q", n, alphabet, got[i])
			}
		}
	})
}
