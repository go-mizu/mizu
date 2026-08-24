package crypt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs"
)

func TestGenerateKey(t *testing.T) {
	a, b := GenerateKey(), GenerateKey()
	if a == b {
		t.Fatal("two generated keys are the same")
	}
	if a.IsZero() || b.IsZero() {
		t.Fatal("a generated key is the zero key")
	}

	// The text form is the prefix and 32 bytes in base64url, which is 43
	// characters with nothing to pad.
	text := a.Reveal()
	if !strings.HasPrefix(text, "mizu1:") {
		t.Errorf("a key reads as %q", text)
	}
	if body := strings.TrimPrefix(text, "mizu1:"); len(body) != 43 {
		t.Errorf("a key is %d characters after the prefix, want 43", len(body))
	}
	if strings.ContainsAny(text, "+/=") {
		t.Errorf("a key is not URL safe: %q", text)
	}
}

func TestParseKey(t *testing.T) {
	want := GenerateKey()

	got, err := ParseKey(want.Reveal())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(want) {
		t.Error("a key did not survive the round trip")
	}

	// A key usually arrives from a file or an environment variable, and those
	// end in a newline more often than not.
	got, err = ParseKey("  " + want.Reveal() + "\n")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(want) {
		t.Error("a key with whitespace around it did not parse to the same key")
	}
}

func TestParseKeyFailures(t *testing.T) {
	short := base64.RawURLEncoding.EncodeToString(make([]byte, 16))
	long := base64.RawURLEncoding.EncodeToString(make([]byte, 48))

	cases := map[string]string{
		"empty":         "",
		"no prefix":     short,
		"other version": "mizu2:" + short,
		"prefix only":   "mizu1:",
		"not base64":    "mizu1:not base64 at all!",
		"standard b64":  "mizu1:" + base64.StdEncoding.EncodeToString(make([]byte, 32)),
		"too short":     "mizu1:" + short,
		"too long":      "mizu1:" + long,
	}
	for name, text := range cases {
		k, err := ParseKey(text)
		if err == nil {
			t.Errorf("%s: %q parsed", name, text)
			continue
		}
		if errs.KindOf(err) != errs.Invalid {
			t.Errorf("%s: the error is %v, want invalid", name, errs.KindOf(err))
		}
		if !k.IsZero() {
			t.Errorf("%s: a key came back with the error", name)
		}
	}
}

// TestParseKeyErrorKeepsQuiet is the reason the decoder's own error is not
// passed on: it quotes the text it stopped in, and the text is the key.
func TestParseKeyErrorKeepsQuiet(t *testing.T) {
	key := GenerateKey()
	body := strings.TrimPrefix(key.Reveal(), "mizu1:")

	_, err := ParseKey("mizu1:" + body[:20] + "!" + body[21:])
	if err == nil {
		t.Fatal("a key with a bad character parsed")
	}
	if strings.Contains(err.Error(), body[:20]) {
		t.Errorf("the error holds part of the key: %v", err)
	}
}

func TestMustParseKey(t *testing.T) {
	want := GenerateKey()
	if got := MustParseKey(want.Reveal()); !got.Equal(want) {
		t.Error("a key did not survive MustParseKey")
	}

	defer func() {
		if recover() == nil {
			t.Error("MustParseKey returned for a key that does not parse")
		}
	}()
	MustParseKey("nonsense")
}

func TestKeyID(t *testing.T) {
	a, b := GenerateKey(), GenerateKey()

	if len(a.ID()) != 16 {
		t.Errorf("an id is %q, want 16 hex characters", a.ID())
	}
	if a.ID() != a.ID() {
		t.Error("the same key gave two ids")
	}
	if a.ID() == b.ID() {
		t.Error("two keys share an id")
	}
	if strings.Contains(a.Reveal(), a.ID()) {
		t.Error("the id is part of the key")
	}

	// The id is the first eight bytes of SHA-256 over the label and the key, so
	// it is fixed. A change here would orphan every ciphertext already written.
	// The value below is the first eight bytes of:
	//
	//	{ printf 'mizu1 key id'; head -c 32 /dev/zero; } | shasum -a 256
	fixed := MustParseKey("mizu1:" + base64.RawURLEncoding.EncodeToString(make([]byte, 32)))
	if got, want := fixed.ID(), "2d0845ee2746f8e8"; got != want {
		t.Errorf("the id of the all zero key is %q, want %q", got, want)
	}
}

func TestKeyEqual(t *testing.T) {
	a := GenerateKey()
	b := MustParseKey(a.Reveal())

	if !a.Equal(b) || !b.Equal(a) {
		t.Error("a key does not equal itself")
	}
	if a.Equal(GenerateKey()) {
		t.Error("two different keys are equal")
	}

	var zero Key
	if !zero.IsZero() {
		t.Error("the zero key is not zero")
	}
	if a.IsZero() {
		t.Error("a generated key is zero")
	}
}

// TestKeyHides is the whole reason this is a type rather than a [32]byte. Every
// way a value gets printed has to come out redacted, including the ones nobody
// meant to call.
func TestKeyHides(t *testing.T) {
	key := GenerateKey()
	secret := strings.TrimPrefix(key.Reveal(), "mizu1:")

	held := struct {
		Name string
		Key  Key
	}{"session", key}

	printed := []string{
		key.String(),
		fmt.Sprint(key),
		fmt.Sprintf("%v", key),
		fmt.Sprintf("%s", key),
		fmt.Sprintf("%q", key),
		fmt.Sprintf("%x", key),
		fmt.Sprintf("%#v", key),
		fmt.Sprintf("%+v", held),
		fmt.Sprint(&key),
		mustJSON(t, key),
		mustJSON(t, held),
		key.LogValue().String(),
	}
	for _, got := range printed {
		if strings.Contains(got, secret) {
			t.Errorf("the key leaked: %s", got)
		}
		if !strings.Contains(got, Redacted) {
			t.Errorf("nothing was redacted in %s", got)
		}
	}
}

func TestKeyLogValue(t *testing.T) {
	key := GenerateKey()
	if got, want := key.LogValue().Kind(), slog.KindString; got != want {
		t.Errorf("a key logs as %v, want %v", got, want)
	}
	if got := key.LogValue().String(); got != key.String() {
		t.Errorf("a key logs as %q, want %q", got, key.String())
	}
}

func TestKeyMarshalText(t *testing.T) {
	key := GenerateKey()

	b, err := key.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != key.String() {
		t.Errorf("a key marshals as %q, want %q", b, key.String())
	}

	// Text goes in as the real thing, which is what a configuration file holds.
	var got Key
	if err := got.UnmarshalText([]byte(key.Reveal())); err != nil {
		t.Fatal(err)
	}
	if !got.Equal(key) {
		t.Error("a key did not survive UnmarshalText")
	}

	if err := got.UnmarshalText([]byte("nonsense")); err == nil {
		t.Error("a key that does not parse was unmarshalled")
	}

	// And a redacted key does not parse, so a configuration written from a
	// marshalled struct fails at startup rather than encrypting with a key
	// nobody has.
	if err := got.UnmarshalText([]byte(key.String())); err == nil {
		t.Error("the redacted form parsed as a key")
	}
}

func TestKeyJSONRoundTrip(t *testing.T) {
	key := GenerateKey()

	b, err := json.Marshal(map[string]Key{"key": key})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), Redacted) {
		t.Errorf("JSON holds %s", b)
	}

	raw, err := json.Marshal(map[string]string{"key": key.Reveal()})
	if err != nil {
		t.Fatal(err)
	}

	into := map[string]Key{}
	if err := json.Unmarshal(raw, &into); err != nil {
		t.Fatal(err)
	}
	if !into["key"].Equal(key) {
		t.Error("a key did not survive JSON")
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
