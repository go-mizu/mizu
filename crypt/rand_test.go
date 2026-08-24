package crypt

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBytes(t *testing.T) {
	for _, n := range []int{0, 1, 16, 32, 1000} {
		b := Bytes(n)
		if len(b) != n {
			t.Errorf("Bytes(%d) gave %d bytes", n, len(b))
		}
	}

	// Two draws of 32 bytes matching would be the end of everything else in
	// this package, so it is worth one line to notice.
	if string(Bytes(32)) == string(Bytes(32)) {
		t.Error("two draws of 32 bytes are the same")
	}

	// A zero byte is a fine thing to draw, but a slice of nothing but zeroes
	// means the source did not write.
	if allZero(Bytes(64)) {
		t.Error("64 random bytes are all zero")
	}
}

func TestString(t *testing.T) {
	for _, n := range []int{0, 1, 20, 100} {
		s := String(n)
		if len(s) != n {
			t.Errorf("String(%d) is %d characters", n, len(s))
		}
		if rest := strings.Trim(s, Alphanumeric); rest != "" {
			t.Errorf("String(%d) is %q, which is not alphanumeric", n, s)
		}
	}
	if String(20) == String(20) {
		t.Error("two strings are the same")
	}
}

func TestToken(t *testing.T) {
	tok := Token(32)
	if len(tok) != 43 {
		t.Errorf("Token(32) is %d characters, want 43", len(tok))
	}
	if strings.ContainsAny(tok, "+/=") {
		t.Errorf("Token(32) is not URL safe: %q", tok)
	}

	b, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil {
		t.Fatalf("Token(32) does not decode: %v", err)
	}
	if len(b) != 32 {
		t.Errorf("Token(32) carries %d bytes", len(b))
	}
	if Token(32) == Token(32) {
		t.Error("two tokens are the same")
	}
}

func TestDigits(t *testing.T) {
	code := Digits(6)
	if len(code) != 6 {
		t.Errorf("Digits(6) is %q", code)
	}
	if rest := strings.Trim(code, Digits10); rest != "" {
		t.Errorf("Digits(6) is %q, which is not digits", code)
	}

	// Leading zeroes are kept. A code that came back as a number would
	// sometimes be five characters, which is the sort of thing that gets found
	// in production rather than here.
	seen := false
	for range 500 {
		if strings.HasPrefix(Digits(6), "0") {
			seen = true
			break
		}
	}
	if !seen {
		t.Error("500 codes and none of them started with a zero")
	}
}

func TestPassword(t *testing.T) {
	for _, n := range []int{3, 4, 8, 16, 24, 64} {
		p := Password(n)
		if len(p) != n {
			t.Errorf("Password(%d) is %d characters", n, len(p))
		}
		if rest := strings.Trim(p, Letters+Digits10+Symbols); rest != "" {
			t.Errorf("Password(%d) is %q, which has %q in it", n, p, rest)
		}
	}
	if Password(16) == Password(16) {
		t.Error("two passwords are the same")
	}
}

// TestPasswordCoversEveryClass is the promise in the doc comment. The short
// lengths are the ones worth repeating, since those are where a draw misses a
// class often enough for a loop that gave up to be caught.
func TestPasswordCoversEveryClass(t *testing.T) {
	for _, n := range []int{3, 4, 6} {
		for range 300 {
			p := Password(n)
			for name, class := range map[string]string{
				"letter": Letters,
				"digit":  Digits10,
				"symbol": Symbols,
			} {
				if !strings.ContainsAny(p, class) {
					t.Fatalf("Password(%d) gave %q, which has no %s in it", n, p, name)
				}
			}
		}
	}
}

// TestPasswordUsesTheWholeAlphabet checks that redrawing does not quietly
// narrow what comes out, which is what a loop written to stop early would do.
func TestPasswordUsesTheWholeAlphabet(t *testing.T) {
	const alphabet = Letters + Digits10 + Symbols

	seen := map[byte]bool{}
	for range 400 {
		for _, c := range []byte(Password(24)) {
			seen[c] = true
		}
	}
	for _, c := range []byte(alphabet) {
		if !seen[c] {
			t.Errorf("%q never came up", c)
		}
	}
}

// TestSymbolsAreSafeToWriteDown pins the alphabet against the reason the doc
// comment gives for it. A quote or a backslash in a generated password breaks
// somewhere between here and the file it ends up in.
func TestSymbolsAreSafeToWriteDown(t *testing.T) {
	if bad := strings.ContainsAny(Symbols, "\"'`\\"); bad {
		t.Errorf("Symbols is %q, which has a quote or a backslash in it", Symbols)
	}
	for _, c := range []byte(Symbols) {
		if c <= ' ' || c >= 0x7f {
			t.Errorf("Symbols has %q in it, which is not printable ascii", c)
		}
	}
	if len(Symbols) < 8 {
		t.Errorf("Symbols is %d characters, which is not much to choose from", len(Symbols))
	}
}

// TestPickIsUniform is the reason for the rejection loop.
//
// Folding a random byte into the alphabet with a remainder biases whatever does
// not fit in a whole run of 256. The bias is small for a 62 character alphabet
// and hard to see in a test, so this uses one where it is not: 129 characters,
// where folding gives the first 127 two byte values each and the last two one
// each, making the tail half as likely as it should be.
func TestPickIsUniform(t *testing.T) {
	const draws = 20000
	alphabet := strings.Repeat("a", 127) + "xy"

	counts := map[byte]int{}
	for _, c := range []byte(Pick(draws, alphabet)) {
		counts[c]++
	}

	share := float64(counts['x']+counts['y']) / draws
	if want := 2.0 / 129.0; share < want-0.005 || share > want+0.005 {
		t.Errorf("the last two characters took %.4f of the draws, want about %.4f", share, want)
	}
}

// TestPickCoversTheAlphabet is the other half of it: every character of a real
// alphabet comes up, so nothing is quietly unreachable.
func TestPickCoversTheAlphabet(t *testing.T) {
	const draws = 20000

	counts := map[byte]int{}
	for _, c := range []byte(Pick(draws, Alphanumeric)) {
		counts[c]++
	}
	for _, c := range []byte(Alphanumeric) {
		if counts[c] == 0 {
			t.Errorf("%q was never drawn in %d", c, draws)
		}
	}
	if len(counts) != len(Alphanumeric) {
		t.Errorf("%d characters came out of an alphabet of %d", len(counts), len(Alphanumeric))
	}
}

func TestPickAlphabetSizes(t *testing.T) {
	// One character has nothing to reject, and 256 has nothing to reject
	// either. Both are the edges of the arithmetic.
	if got := Pick(4, "a"); got != "aaaa" {
		t.Errorf("Pick from one character gave %q", got)
	}

	var full []byte
	for i := range 256 {
		full = append(full, byte(i))
	}
	if got := Pick(1000, string(full)); len(got) != 1000 {
		t.Errorf("Pick from 256 characters gave %d", len(got))
	}
	if got := Pick(0, "abc"); got != "" {
		t.Errorf("Pick of nothing gave %q", got)
	}
}

func TestPickPanics(t *testing.T) {
	cases := map[string]func(){
		"negative length":  func() { Pick(-1, "abc") },
		"empty alphabet":   func() { Pick(4, "") },
		"alphabet too big": func() { Pick(4, strings.Repeat("a", 257)) },
		"negative bytes":   func() { Bytes(-1) },
		"zero bound":       func() { Intn(0) },
		"negative bound":   func() { Intn(-3) },
		"empty choice":     func() { Choice([]int{}) },
		"short password":   func() { Password(2) },
	}
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("it returned")
				}
			}()
			call()
		})
	}
}

func TestIntn(t *testing.T) {
	if got := Intn(1); got != 0 {
		t.Errorf("Intn(1) gave %d", got)
	}

	const draws = 5000
	counts := make([]int, 7)
	for range draws {
		n := Intn(len(counts))
		if n < 0 || n >= len(counts) {
			t.Fatalf("Intn(%d) gave %d", len(counts), n)
		}
		counts[n]++
	}
	for i, c := range counts {
		if share := float64(c) / draws; share < 0.10 || share > 0.19 {
			t.Errorf("%d came up %.3f of the time, want about %.3f", i, share, 1.0/7.0)
		}
	}
}

func TestChoice(t *testing.T) {
	type level string
	levels := []level{"debug", "info", "warn"}

	seen := map[level]bool{}
	for range 200 {
		seen[Choice(levels)] = true
	}
	if len(seen) != len(levels) {
		t.Errorf("200 choices from %v gave %v", levels, seen)
	}

	if got := Choice([]int{9}); got != 9 {
		t.Errorf("a choice of one gave %d", got)
	}
}

func allZero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}
