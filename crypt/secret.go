package crypt

import (
	"crypto/subtle"
	"fmt"
	"io"
	"log/slog"
)

// Redacted is what a hidden value prints as.
//
// It is the text [github.com/go-mizu/mizu/log] writes when it masks an
// attribute by name, so a value that is hidden because of its type and one that
// is hidden because of its key look the same in a log.
//
// It is a fixed string rather than a run of asterisks the length of the value,
// because the length of a secret is information about the secret.
const Redacted = "[redacted]"

// Secret is a string that does not print.
//
// It is a password, an API token, a webhook signing key: something that arrives
// as text, is compared or handed to another service, and has no business in a
// log line, an error message or a crash dump. Making it a type rather than
// remembering to mask it means the masking travels with the value, through
// every struct it is a field of and every handler it reaches.
//
//	type Config struct {
//		DatabaseURL string
//		StripeKey   crypt.Secret
//	}
//
//	fmt.Println(cfg)                 // {postgres://... [redacted]}
//	slog.Info("starting", "cfg", cfg) // the same
//	http.Post(url, cfg.StripeKey.Reveal(), body)
//
// An empty Secret prints as nothing, so a value that was never configured
// reads as missing rather than as present and hidden.
//
// A Secret is a string underneath and converting one back is a conversion
// anybody can write. That is on purpose: this is a guard against the value
// leaking by accident, which is where secrets actually leak, and not a claim
// about what the process can be made to do.
type Secret string

// Reveal returns the value.
//
// This is the only method that does, so a search for Reveal finds every place
// the secret is used for what it is for.
func (s Secret) Reveal() string { return string(s) }

// String returns [Redacted], or nothing at all when the secret is empty.
func (s Secret) String() string {
	if s == "" {
		return ""
	}
	return Redacted
}

// Equal is whether two secrets are the same, in constant time.
//
// Comparing with == takes a time that depends on how many leading bytes match,
// which is enough to recover a value one byte at a time when the caller is
// allowed to keep asking. Use this for anything that arrives from outside, such
// as a webhook signature or an API token from a header.
//
// The length of a secret is not hidden by any of this, since two strings of
// different lengths are never equal.
func (s Secret) Equal(other Secret) bool {
	return subtle.ConstantTimeCompare([]byte(s), []byte(other)) == 1
}

// LogValue is the redacted form, so a secret logged as an attribute is masked
// in every handler, including the ones in the standard library.
func (s Secret) LogValue() slog.Value { return slog.StringValue(s.String()) }

// Format writes the redacted form for every verb, including the ones that print
// the fields of a struct, which is what this type exists to stop.
func (s Secret) Format(f fmt.State, verb rune) { format(f, verb, s.String()) }

// MarshalText writes the redacted form. JSON goes through it too, since a
// [encoding.TextMarshaler] needs nothing else.
//
// A secret survives a round trip through text only as far as the program that
// wrote it. Marshalling a configuration to show somebody does not put the
// secret in it.
func (s Secret) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

// UnmarshalText reads the value, so a Secret works as a field in a
// configuration struct.
func (s *Secret) UnmarshalText(b []byte) error {
	*s = Secret(b)
	return nil
}

// format writes a redacted value under whatever verb was asked for. A verb that
// would print the value prints this instead, and %q still comes out as a quoted
// Go string so that output holding one stays parseable.
func format(f fmt.State, verb rune, s string) {
	if verb == 'q' {
		fmt.Fprintf(f, "%q", s)
		return
	}
	io.WriteString(f, s)
}
